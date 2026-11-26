package birthdays

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"chat-gobelin-api/internal/users"
)

func day(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 12, 0, 0, 0, Paris) }

func TestDue(t *testing.T) {
	cases := []struct {
		name     string
		birthday users.Birthday
		now      time.Time
		year     int
		due      bool
	}{
		{"le jour même", users.Birthday{Month: 3, Day: 14}, day(2027, 3, 14), 2027, true},
		{"six jours après", users.Birthday{Month: 3, Day: 14}, day(2027, 3, 20), 2027, true},
		{"sept jours après", users.Birthday{Month: 3, Day: 14}, day(2027, 3, 21), 0, false},
		{"la veille", users.Birthday{Month: 3, Day: 14}, day(2027, 3, 13), 0, false},
		{"fin d'année", users.Birthday{Month: 12, Day: 30}, day(2027, 1, 2), 2026, true},
		{"29 février, année bissextile", users.Birthday{Month: 2, Day: 29}, day(2028, 2, 29), 2028, true},
		{"29 février fêté le 28", users.Birthday{Month: 2, Day: 29}, day(2027, 2, 28), 2027, true},
		{"28 février, année bissextile", users.Birthday{Month: 2, Day: 29}, day(2028, 2, 28), 0, false},
	}
	for _, c := range cases {
		year, due := Due(c.birthday, c.now)
		if year != c.year || due != c.due {
			t.Errorf("%s : Due = (%d, %v), attendu (%d, %v)", c.name, year, due, c.year, c.due)
		}
	}
}

func TestCheck(t *testing.T) {
	for _, b := range []users.Birthday{{Month: 2, Day: 29}, {Month: 12, Day: 31}, {Month: 1, Day: 1}} {
		if b.Check() != nil {
			t.Errorf("%v devrait être valide", b)
		}
	}
	for _, b := range []users.Birthday{{Month: 2, Day: 30}, {Month: 4, Day: 31}, {Month: 13, Day: 1}, {Month: 0, Day: 10}, {Month: 5, Day: 0}} {
		if b.Check() == nil {
			t.Errorf("%v devrait être refusé", b)
		}
	}
}

func TestLetter(t *testing.T) {
	var letter struct {
		V      int             `json:"v"`
		Blocks [][]interface{} `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(Letter(DefaultTemplate, "Louis")), &letter); err != nil {
		t.Fatal(err)
	}
	text := letter.Blocks[0][4].(string)
	if letter.V != 1 || !strings.HasPrefix(text, "Joyeux anniversaire Louis !") {
		t.Errorf("lettre inattendue : %q", text)
	}
	for _, line := range strings.Split(text, "\n") {
		if len([]rune(line)) > lineWidth {
			t.Errorf("ligne trop longue : %q", line)
		}
	}
}
