// Package letters porte les règles du jeu : par défaut, une lettre met 1 minute à arriver et les planètes ne se
// rapprochent pas. Délai selon la distance et rapprochement restent réglables par variables d'environnement.
package letters

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Rules regroupe les réglages. Les distances n'ont pas d'unité : seule leur échelle compte.
type Rules struct {
	MinDistance float64       // plus proche possible, atteint à force d'échanges
	MaxDistance float64       // distance de départ de deux inconnus, aussi le plafond
	MinDelay    time.Duration // délai d'une lettre à la distance minimale (temps « simulé »)
	MaxDelay    time.Duration // délai d'une lettre à la distance maximale (temps « simulé »)
	Approach    float64       // part de distance retirée à chaque réponse (0.10 = 10 %, 0 = jamais)
	TimeScale   float64       // 1 = temps réel ; 3600 = une heure simulée dure une seconde (démo)
}

func DefaultRules() Rules {
	return Rules{
		MinDistance: 1,
		MaxDistance: 10,
		MinDelay:    time.Minute,
		MaxDelay:    time.Minute,
		Approach:    0,
		TimeScale:   1,
	}
}

// RulesFromEnv lit LETTER_MIN_DISTANCE, LETTER_MAX_DISTANCE, LETTER_MIN_DELAY, LETTER_MAX_DELAY (durées Go :
// "90s", "3h"), LETTER_APPROACH et LETTER_TIME_SCALE. Une valeur absente garde le défaut, une valeur invalide
// est une erreur (mieux vaut ne pas démarrer que tourner avec des règles que personne n'a voulues).
func RulesFromEnv() (Rules, error) {
	r := DefaultRules()
	var err error
	if r.MinDistance, err = envFloat("LETTER_MIN_DISTANCE", r.MinDistance); err != nil {
		return r, err
	}
	if r.MaxDistance, err = envFloat("LETTER_MAX_DISTANCE", r.MaxDistance); err != nil {
		return r, err
	}
	if r.MinDelay, err = envDuration("LETTER_MIN_DELAY", r.MinDelay); err != nil {
		return r, err
	}
	if r.MaxDelay, err = envDuration("LETTER_MAX_DELAY", r.MaxDelay); err != nil {
		return r, err
	}
	if r.Approach, err = envFloat("LETTER_APPROACH", r.Approach); err != nil {
		return r, err
	}
	if r.TimeScale, err = envFloat("LETTER_TIME_SCALE", r.TimeScale); err != nil {
		return r, err
	}
	return r, r.Validate()
}

func (r Rules) Validate() error {
	switch {
	case r.MinDistance <= 0:
		return fmt.Errorf("letters: la distance minimale doit être > 0 (reçu %v)", r.MinDistance)
	case r.MaxDistance <= r.MinDistance:
		return fmt.Errorf("letters: la distance maximale (%v) doit dépasser la minimale (%v)", r.MaxDistance, r.MinDistance)
	case r.MinDelay < 0 || r.MaxDelay < r.MinDelay:
		return fmt.Errorf("letters: délais incohérents (min %v, max %v)", r.MinDelay, r.MaxDelay)
	case r.Approach < 0 || r.Approach >= 1:
		return fmt.Errorf("letters: le rapprochement doit être entre 0 inclus et 1 exclus (reçu %v)", r.Approach)
	case r.TimeScale <= 0:
		return fmt.Errorf("letters: le facteur de temps doit être > 0 (reçu %v)", r.TimeScale)
	}
	return nil
}

// InitialDistance : deux personnes qui ne se sont jamais écrit sont aussi loin que possible.
func (r Rules) InitialDistance() float64 { return r.MaxDistance }

// Delay est le temps réel que met une lettre pour parcourir cette distance : interpolation linéaire entre
// MinDelay et MaxDelay, puis division par TimeScale.
func (r Rules) Delay(distance float64) time.Duration {
	t := (distance - r.MinDistance) / (r.MaxDistance - r.MinDistance)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	simulated := float64(r.MinDelay) + t*float64(r.MaxDelay-r.MinDelay)
	return time.Duration(simulated / r.TimeScale)
}

// Approached est la distance après une réponse : Approach en moins, sans jamais descendre sous MinDistance.
func (r Rules) Approached(distance float64) float64 {
	next := distance * (1 - r.Approach)
	if next < r.MinDistance {
		return r.MinDistance
	}
	return next
}

func envFloat(name string, def float64) (float64, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def, fmt.Errorf("letters: %s=%q n'est pas un nombre", name, raw)
	}
	return v, nil
}

func envDuration(name string, def time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return def, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return def, fmt.Errorf("letters: %s=%q n'est pas une durée (exemples : 90s, 3h)", name, raw)
	}
	return v, nil
}
