package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"chat-gobelin-api/internal/channels"
	"chat-gobelin-api/internal/letters"
	"chat-gobelin-api/internal/messages"
)

const maxContentLength = 100000

type messageHandler struct {
	messageRepo messages.Repo
	channelRepo channels.Repo
	letters     *letters.Service
}

func NewMessageHandler(s messages.Repo, c channels.Repo, l *letters.Service) (*messageHandler, error) {
	return &messageHandler{
		messageRepo: s,
		channelRepo: c,
		letters:     l,
	}, nil
}

type createMessageRequest struct {
	Content     string `json:"content" binding:"required"`
	Channel     int64  `json:"channel" binding:"required"`
	RecipientID *int64 `json:"recipient_id"` // absent : message adressé à tout le salon ; présent : lettre
}

// memberChannel vérifie que le salon existe et que l'utilisateur en est membre.
// En cas d'échec, la réponse HTTP est déjà écrite et ok vaut false.
func (h messageHandler) memberChannel(writer http.ResponseWriter, request *http.Request, channelID int64, userID int64) (*channels.Channel, bool) {
	ctx := request.Context()

	channel, err := h.channelRepo.GetByID(ctx, channelID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(writer, "Channel not found", http.StatusNotFound)
		return nil, false
	}
	if err != nil {
		log.Printf("Error finding channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return nil, false
	}

	member, err := h.channelRepo.IsMember(ctx, channel.ID, userID)
	if err != nil {
		log.Printf("Error checking membership: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return nil, false
	}
	if !member {
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return nil, false
	}
	return channel, true
}

func channelIDFromQuery(writer http.ResponseWriter, request *http.Request) (int64, bool) {
	raw := request.URL.Query().Get("channel")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		log.Printf("Invalid channel parameter: %q", raw)
		http.Error(writer, "channel parameter required", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// GetMessages renvoie les messages que l'utilisateur peut voir : voir messages.Repo.GetForUser.
func (h messageHandler) GetMessages(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	channelID, ok := channelIDFromQuery(writer, request)
	if !ok {
		return
	}
	channel, ok := h.memberChannel(writer, request, channelID, user.ID)
	if !ok {
		return
	}

	list, err := h.messageRepo.GetForUser(request.Context(), *channel, *user)
	if err != nil {
		log.Printf("Error getting messages: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []messages.Message{} // un salon vide renvoie [] et non null
	}

	writeJSON(writer, list)
}

// CreateMessage envoie un message au salon (sans recipient_id) ou une lettre à une personne (avec).
// Une lettre n'arrive qu'à sa date de livraison, qui dépend de la distance entre les deux planètes.
func (h messageHandler) CreateMessage(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	var req createMessageRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %s", err)
		http.Error(writer, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Content == "" || len(req.Content) > maxContentLength {
		http.Error(writer, "content must be between 1 and 100000 characters", http.StatusBadRequest)
		return
	}

	channel, ok := h.memberChannel(writer, request, req.Channel, user.ID)
	if !ok {
		return
	}

	var (
		message *messages.Message
		err     error
	)
	if req.RecipientID == nil {
		message, err = h.messageRepo.Create(request.Context(), req.Content, *channel, *user)
	} else {
		message, err = h.letters.Send(request.Context(), channel.ID, *user, *req.RecipientID, req.Content)
	}
	switch {
	case errors.Is(err, letters.ErrSelfLetter):
		http.Error(writer, "cannot send a letter to yourself", http.StatusBadRequest)
		return
	case errors.Is(err, letters.ErrRecipientNotMember):
		http.Error(writer, "recipient is not in this channel", http.StatusBadRequest)
		return
	case err != nil:
		log.Printf("Error creating message: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(writer, message)
}

// System décrit le système solaire du salon du point de vue de l'utilisateur : le soleil, les planètes,
// sa distance à chacune et le temps qu'une lettre mettrait.
func (h messageHandler) System(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	channelID, ok := channelIDFromQuery(writer, request)
	if !ok {
		return
	}
	channel, ok := h.memberChannel(writer, request, channelID, user.ID)
	if !ok {
		return
	}

	system, err := h.letters.System(request.Context(), *channel, user.ID)
	if err != nil {
		log.Printf("Error building system: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(writer, system)
}

// CancelLetter : DELETE /channel/message?message=ID, l'expéditeur annule une lettre encore en route
func (h messageHandler) CancelLetter(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(request.URL.Query().Get("message"), 10, 64)
	if err != nil {
		http.Error(writer, "message parameter required", http.StatusBadRequest)
		return
	}
	err = h.letters.Cancel(request.Context(), *user, id)
	if errors.Is(err, letters.ErrNotCancellable) {
		http.Error(writer, "Letter not found, not yours, or already arrived", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error cancelling letter: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

// RemoveRocket : DELETE /channel/message/rocket?message=ID, réservé au destinataire d'une lettre arrivée
func (h messageHandler) RemoveRocket(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(request.URL.Query().Get("message"), 10, 64)
	if err != nil {
		http.Error(writer, "message parameter required", http.StatusBadRequest)
		return
	}
	err = h.messageRepo.RemoveRocket(request.Context(), id, *user)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(writer, "Letter not found, not yours, or still in transit", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error removing rocket: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func writeJSON(writer http.ResponseWriter, value interface{}) {
	result, err := json.Marshal(value)
	if err != nil {
		log.Printf("Error marshaling value: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}
