package users

// Bornes de la taille d'une planète (alignées sur la contrainte de la migration 00004 et sur le front).
const (
	MinPlanetSize = 0.4
	MaxPlanetSize = 1.0
)

type User struct {
	ID           int64   `db:"id" json:"id"`
	Username     string  `db:"username" json:"username"`
	PasswordHash []byte  `db:"password_hash" json:"-"`
	PlanetColor  string  `db:"planet_color" json:"planet_color"` // "#rrggbb"
	PlanetSize   float64 `db:"planet_size" json:"planet_size"`
	PlanetRings  bool    `db:"planet_rings" json:"planet_rings"`
}

// Planet est l'apparence choisie à l'inscription.
type Planet struct {
	Color string
	Size  float64
	Rings bool
}
