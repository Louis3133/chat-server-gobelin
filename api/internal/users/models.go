package users

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Bornes de l'apparence d'une planète (alignées sur les contraintes des migrations 00004 et 00005 et sur le front).
const (
	MinPlanetSize = 0.4
	MaxPlanetSize = 1.0
	MaxPlanetTilt = 1.2
	MinRingSize   = 1.5
	MaxRingSize   = 4.0
	MaxPlanetSpin = 0.1

	MaxPlanetName = 40
	MaxPronouns   = 30
	MaxBio        = 500
)

type User struct {
	ID           int64  `db:"id" json:"id"`
	Username     string `db:"username" json:"username"`
	PasswordHash []byte `db:"password_hash" json:"-"`
	Planet
	Profile
	TextureVersion int64 `db:"planet_texture_version" json:"planet_texture_version"` // 0 : pas de texture importée
}

// Planet est l'apparence de la planète : choisie après l'inscription, modifiable ensuite.
type Planet struct {
	Color     string  `db:"planet_color" json:"planet_color"` // "#rrggbb"
	Size      float64 `db:"planet_size" json:"planet_size"`
	Rings     bool    `db:"planet_rings" json:"planet_rings"`
	Tilt      float64 `db:"planet_tilt" json:"planet_tilt"`
	RingColor string  `db:"planet_ring_color" json:"planet_ring_color"`
	RingSize  float64 `db:"planet_ring_size" json:"planet_ring_size"`
	Spin      float64 `db:"planet_spin" json:"planet_spin"`
	Moon      bool    `db:"planet_moon" json:"planet_moon"` // à la place des anneaux, jamais les deux
}

// Profile est ce que la modale d'une planète affiche : son nom, les pronoms et la bio de la personne.
type Profile struct {
	PlanetName string `db:"planet_name" json:"planet_name"`
	Pronouns   string `db:"pronouns" json:"pronouns"`
	Bio        string `db:"bio" json:"bio"`
}

// Clean retire les espaces autour des champs puis vérifie leur longueur (alignée sur la migration 00009).
func (p *Profile) Clean() error {
	p.PlanetName = strings.TrimSpace(p.PlanetName)
	p.Pronouns = strings.TrimSpace(p.Pronouns)
	p.Bio = strings.TrimSpace(p.Bio)
	switch {
	case utf8.RuneCountInString(p.PlanetName) > MaxPlanetName:
		return fmt.Errorf("planet_name must be at most %d characters", MaxPlanetName)
	case utf8.RuneCountInString(p.Pronouns) > MaxPronouns:
		return fmt.Errorf("pronouns must be at most %d characters", MaxPronouns)
	case utf8.RuneCountInString(p.Bio) > MaxBio:
		return fmt.Errorf("bio must be at most %d characters", MaxBio)
	}
	return nil
}

// DefaultPlanet reprend les valeurs par défaut de la base : la planète grise d'un nouveau compte.
func DefaultPlanet() Planet {
	return Planet{Color: "#8d9db6", Size: 0.55, RingColor: "#d9c9a3", RingSize: 2, Spin: 0.01}
}

var colorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Check vérifie l'apparence avant de l'écrire en base, avec un message lisible plutôt qu'une erreur SQL.
func (p Planet) Check() error {
	switch {
	case !colorRe.MatchString(p.Color):
		return errors.New("planet_color must look like #rrggbb")
	case !(p.Size >= MinPlanetSize && p.Size <= MaxPlanetSize):
		return fmt.Errorf("planet_size must be between %v and %v", MinPlanetSize, MaxPlanetSize)
	case !(p.Tilt >= -MaxPlanetTilt && p.Tilt <= MaxPlanetTilt):
		return fmt.Errorf("planet_tilt must be between %v and %v", -MaxPlanetTilt, MaxPlanetTilt)
	case !colorRe.MatchString(p.RingColor):
		return errors.New("planet_ring_color must look like #rrggbb")
	case !(p.RingSize >= MinRingSize && p.RingSize <= MaxRingSize):
		return fmt.Errorf("planet_ring_size must be between %v and %v", MinRingSize, MaxRingSize)
	case !(p.Spin >= -MaxPlanetSpin && p.Spin <= MaxPlanetSpin):
		return fmt.Errorf("planet_spin must be between %v and %v", -MaxPlanetSpin, MaxPlanetSpin)
	case p.Rings && p.Moon:
		return errors.New("a planet has either rings or a moon, not both")
	}
	return nil
}
