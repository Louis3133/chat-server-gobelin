package letters

import (
	"math"
	"testing"
	"time"
)

func TestDelayIsLinearBetweenBounds(t *testing.T) {
	r := DefaultRules()
	cases := []struct {
		distance float64
		want     time.Duration
	}{
		{1, time.Minute},    // le plus proche : délai minimal
		{10, 3 * time.Hour}, // inconnus : délai maximal
		{5.5, time.Minute + (3*time.Hour-time.Minute)/2}, // à mi-chemin
		{0.2, time.Minute},  // sous le minimum : plafonné
		{99, 3 * time.Hour}, // au-dessus du maximum : plafonné
	}
	for _, c := range cases {
		if got := r.Delay(c.distance); got != c.want {
			t.Errorf("Delay(%v) = %v, attendu %v", c.distance, got, c.want)
		}
	}
}

func TestDelayIsDividedByTimeScale(t *testing.T) {
	r := DefaultRules()
	r.TimeScale = 3600 // démo : une heure simulée = une seconde
	if got := r.Delay(10); got != 3*time.Second {
		t.Errorf("Delay(10) en démo = %v, attendu 3s", got)
	}
}

func TestApproachedRemovesTenPercentWithAFloor(t *testing.T) {
	r := DefaultRules()
	if got := r.Approached(10); math.Abs(got-9) > 1e-9 {
		t.Errorf("Approached(10) = %v, attendu 9", got)
	}
	if got := r.Approached(1.05); got != 1 {
		t.Errorf("Approached(1.05) = %v, attendu le plancher 1", got)
	}
	d := r.InitialDistance()
	for i := 0; i < 200; i++ {
		d = r.Approached(d)
	}
	if d != r.MinDistance {
		t.Errorf("après 200 réponses la distance est %v, attendu le plancher %v", d, r.MinDistance)
	}
}

func TestValidateRejectsNonsense(t *testing.T) {
	bad := []func(*Rules){
		func(r *Rules) { r.MinDistance = 0 },
		func(r *Rules) { r.MaxDistance = r.MinDistance },
		func(r *Rules) { r.MaxDelay = r.MinDelay - 1 },
		func(r *Rules) { r.Approach = 0 },
		func(r *Rules) { r.Approach = 1 },
		func(r *Rules) { r.TimeScale = 0 },
	}
	for i, mutate := range bad {
		r := DefaultRules()
		mutate(&r)
		if r.Validate() == nil {
			t.Errorf("règle invalide n°%d acceptée", i)
		}
	}
	if err := DefaultRules().Validate(); err != nil {
		t.Errorf("les règles par défaut doivent être valides : %v", err)
	}
}

func TestRulesFromEnv(t *testing.T) {
	t.Setenv("LETTER_TIME_SCALE", "3600")
	t.Setenv("LETTER_MAX_DELAY", "90m")
	r, err := RulesFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if r.TimeScale != 3600 || r.MaxDelay != 90*time.Minute || r.MinDistance != 1 {
		t.Errorf("lecture de l'environnement incorrecte : %+v", r)
	}
	t.Setenv("LETTER_APPROACH", "beaucoup")
	if _, err := RulesFromEnv(); err == nil {
		t.Error("une valeur non numérique doit être refusée")
	}
}
