package birthdays

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	_ "time/tzdata"

	"chat-gobelin-api/internal/users"
)

// Règles de la lettre d'anniversaire de Zib Zib
const (
	GraceDays   = 7
	FlightTime  = time.Minute
	MaxTemplate = 600
	TemplateKey = "birthday_letter"

	lineWidth = 36
	fontSize  = 34
	top       = 120
	left      = 90
)

const DefaultTemplate = `Joyeux anniversaire {pseudo} !

Aujourd'hui, ta planète a bouclé un tour de plus autour de son soleil. Toute la galaxie se joint à moi pour te souhaiter une année pleine de belles lettres et de rencontres lumineuses.

Zib Zib`

var Paris, _ = time.LoadLocation("Europe/Paris")

// Due : l'année de l'anniversaire à fêter, s'il tombe aujourd'hui ou dans les jours qui précèdent
func Due(b users.Birthday, now time.Time) (int, bool) {
	for d := 0; d < GraceDays; d++ {
		day := now.AddDate(0, 0, -d)
		if b.FallsOn(day) {
			return day.Year(), true
		}
	}
	return 0, false
}

func CheckTemplate(text string) (string, error) {
	text = strings.TrimSpace(text)
	switch {
	case text == "":
		return "", errors.New("text is required")
	case utf8.RuneCountInString(text) > MaxTemplate:
		return "", fmt.Errorf("text must be at most %d characters", MaxTemplate)
	}
	return text, nil
}

// Letter : le texte, pseudo inséré, au format des lettres du front (v1, un bloc de texte)
func Letter(template, username string) string {
	text := wrap(strings.ReplaceAll(template, "{pseudo}", username))
	content, _ := json.Marshal(map[string]interface{}{
		"v":       1,
		"blocks":  [][]interface{}{{top, left, 0, fontSize, text, "#ffffff"}},
		"strokes": []interface{}{},
		"images":  []interface{}{},
	})
	return string(content)
}

func wrap(text string) string {
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			if line != "" && utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) > lineWidth {
				out = append(out, line)
				line = word
			} else if line == "" {
				line = word
			} else {
				line += " " + word
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
