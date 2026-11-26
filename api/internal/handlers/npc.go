package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"chat-gobelin-api/internal/birthdays"
	"chat-gobelin-api/internal/mailbox"
)

type npcHandler struct {
	birthdayRepo birthdays.Repo
	mailboxRepo  mailbox.Repo
	admins       map[string]bool
	npcID        int64
}

func NewNPCHandler(b birthdays.Repo, m mailbox.Repo, admins map[string]bool, npcID int64) (*npcHandler, error) {
	return &npcHandler{
		birthdayRepo: b,
		mailboxRepo:  m,
		admins:       admins,
		npcID:        npcID,
	}, nil
}

// Letters : GET /npc/letters, les lettres de Zib Zib ; la lettre d'anniversaire part à la première visite
func (h npcHandler) Letters(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	ctx := request.Context()
	if h.npcID == 0 || user.IsNPC {
		writeJSON(writer, []interface{}{})
		return
	}

	if birthday := user.Birthday(); birthday != nil {
		if year, due := birthdays.Due(*birthday, time.Now().In(birthdays.Paris)); due {
			template, err := h.birthdayRepo.Template(ctx)
			if err == nil {
				_, err = h.birthdayRepo.Deliver(ctx, h.npcID, user.ID, year, birthdays.Letter(template, user.Username))
			}
			if err != nil {
				log.Printf("Error sending birthday letter: %s", err.Error())
			}
		}
	}

	letters, err := h.mailboxRepo.LettersFor(ctx, h.npcID, user.ID)
	if err != nil {
		log.Printf("Error retrieving npc letters: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, letters)
}

type birthdayLetter struct {
	Text string `json:"text"`
}

func (h npcHandler) admin(writer http.ResponseWriter, request *http.Request) bool {
	user, ok := currentUser(writer, request)
	if !ok {
		return false
	}
	if !h.admins[user.Username] {
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return false
	}
	return true
}

// BirthdayLetter : GET /npc/birthday-letter, le texte de la lettre d'anniversaire (admins)
func (h npcHandler) BirthdayLetter(writer http.ResponseWriter, request *http.Request) {
	if !h.admin(writer, request) {
		return
	}
	text, err := h.birthdayRepo.Template(request.Context())
	if err != nil {
		log.Printf("Error retrieving birthday letter: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, birthdayLetter{Text: text})
}

// SetBirthdayLetter : PUT /npc/birthday-letter, {pseudo} est remplacé par le nom d'utilisateur (admins)
func (h npcHandler) SetBirthdayLetter(writer http.ResponseWriter, request *http.Request) {
	if !h.admin(writer, request) {
		return
	}
	var body birthdayLetter
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "invalid body", http.StatusBadRequest)
		return
	}
	text, err := birthdays.CheckTemplate(body.Text)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.birthdayRepo.SetTemplate(request.Context(), text); err != nil {
		log.Printf("Error saving birthday letter: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, birthdayLetter{Text: text})
}

type mailboxState struct {
	Open bool `json:"open"`
}

// Mailbox : GET /npc/mailbox, Zib Zib reçoit des lettres seulement si les admins ont écrit sa réponse
func (h npcHandler) Mailbox(writer http.ResponseWriter, request *http.Request) {
	if _, ok := currentUser(writer, request); !ok {
		return
	}
	reply, err := h.mailboxRepo.Reply(request.Context())
	if err != nil {
		log.Printf("Error retrieving npc reply: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, mailboxState{Open: h.npcID != 0 && reply != ""})
}

// ReplyLetter : GET /npc/reply-letter, le texte de la réponse de Zib Zib (admins)
func (h npcHandler) ReplyLetter(writer http.ResponseWriter, request *http.Request) {
	if !h.admin(writer, request) {
		return
	}
	text, err := h.mailboxRepo.Reply(request.Context())
	if err != nil {
		log.Printf("Error retrieving npc reply: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, birthdayLetter{Text: text})
}

// SetReplyLetter : PUT /npc/reply-letter, vide ferme la boîte aux lettres, {pseudo} est remplacé (admins)
func (h npcHandler) SetReplyLetter(writer http.ResponseWriter, request *http.Request) {
	if !h.admin(writer, request) {
		return
	}
	var body birthdayLetter
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "invalid body", http.StatusBadRequest)
		return
	}
	text, err := mailbox.CheckReply(body.Text)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.mailboxRepo.SetReply(request.Context(), text); err != nil {
		log.Printf("Error saving npc reply: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, birthdayLetter{Text: text})
}

type npcLetterRequest struct {
	Content string `json:"content"`
}

// SendLetter : POST /npc/letter, une lettre à Zib Zib, qui répond avec le texte des admins
func (h npcHandler) SendLetter(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	if h.npcID == 0 {
		http.Error(writer, "Zib Zib is unavailable", http.StatusServiceUnavailable)
		return
	}
	if user.IsNPC {
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return
	}
	var body npcLetterRequest
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Content == "" || len(body.Content) > maxContentLength {
		http.Error(writer, "content must be between 1 and 100000 characters", http.StatusBadRequest)
		return
	}
	ctx := request.Context()
	reply, err := h.mailboxRepo.Reply(ctx)
	if err != nil {
		log.Printf("Error retrieving npc reply: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if reply == "" {
		http.Error(writer, "Zib Zib does not accept letters yet", http.StatusConflict)
		return
	}
	letter, err := h.mailboxRepo.Send(ctx, h.npcID, user.ID, body.Content, birthdays.Letter(reply, user.Username))
	if err != nil {
		log.Printf("Error sending npc letter: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, letter)
}
