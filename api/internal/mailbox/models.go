package mailbox

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Lettres à Zib Zib : 10 minutes de vol, réponse 10 minutes après l'arrivée, 10 minutes de vol au retour
const (
	FlightTime      = 10 * time.Minute
	ReplyDelay      = 10 * time.Minute
	ReplyFlightTime = 10 * time.Minute
	MaxReply        = 600
	ReplyKey        = "npc_reply"
)

// CheckReply : vide ferme la boîte aux lettres
func CheckReply(text string) (string, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > MaxReply {
		return "", fmt.Errorf("text must be at most %d characters", MaxReply)
	}
	return text, nil
}
