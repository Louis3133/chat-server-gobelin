package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"chat-gobelin-api/internal/users"
	"chat-gobelin-api/internal/channels"
)

type channelHandler struct {
	channelRepo channels.Repo
	userRepo users.Repo
	admins map[string]bool // noms d'utilisateur autorisés à créer et supprimer un salon (ADMIN_USERNAMES)
}

func NewChannelHandler(s channels.Repo, u users.Repo, admins map[string]bool) (*channelHandler, error) {
	return &channelHandler{
		channelRepo: s,
		userRepo: u,
		admins: admins,
	}, nil
}

type createChannelRequest struct {
	Name string `json:"name" binding:"required"`
}

type EnterChannelRequest struct {
	ChannelName string `json:"channel" binding:"required"`
}

func (h channelHandler) Create(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	decoder := json.NewDecoder(request.Body)

	var req createChannelRequest
	err := decoder.Decode(&req)

	if err != nil {
		log.Printf("Error decoding request: %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Seul un admin crée un salon (un soleil) : sans admin configuré, personne ne peut.
	if !h.admins[user.Username] {
		log.Printf("User %s tried to create a channel without being admin", user.Username)
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return
	}
	if req.Name == "" {
		http.Error(writer, "name required", http.StatusBadRequest)
		return
	}

	channel, err := h.channelRepo.Create(ctx, req.Name, &user.ID)
	if err != nil {
		log.Printf("Error creating channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.channelRepo.AddUserToChannel(ctx, *channel, *user)
	if err != nil {
		log.Printf("Error adding user to new channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(channel)
	if err != nil {
		log.Printf("Error marshaling value: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}

func (h channelHandler) EnterChannel(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	decoder := json.NewDecoder(request.Body)

	var req EnterChannelRequest
	err := decoder.Decode(&req)

	if err != nil {
		log.Printf("Error decoding request: %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	channel, err := h.channelRepo.GetByName(ctx, req.ChannelName)
	if err != nil {
		log.Printf("Error finding channel: %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.channelRepo.AddUserToChannel(ctx, *channel, *user)
	if err != nil {
		log.Printf("Error adding user to new channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(channel)
	if err != nil {
		log.Printf("Error marshaling value: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}

func (h channelHandler) GetAll(writer http.ResponseWriter, request *http.Request) {
  ctx := request.Context()

  // 1. Récupère tous les salons depuis la BDD via le repo
  channelList, err := h.channelRepo.GetAllChannels(ctx)
  if err != nil {
    log.Printf("Error retrieving all channels: %s", err.Error())
    writer.WriteHeader(http.StatusInternalServerError)
    return
  }

  // 2. Encode la liste en JSON
  result, err := json.Marshal(channelList)
  if err != nil {
    log.Printf("Error marshaling channels: %s", err.Error())
    writer.WriteHeader(http.StatusInternalServerError)
    return
  }

  // 3. Renvoie la réponse avec le statut 200 OK
  writer.Header().Set("Content-Type", "application/json")
  writer.WriteHeader(http.StatusOK)
  writer.Write(result)
}

type updateBioRequest struct {
	Bio string `json:"bio"`
}

// UpdateBio : PUT /channel/bio?channel=ID, réservé aux admins et au propriétaire du salon
func (h channelHandler) UpdateBio(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	channelID, ok := channelIDFromQuery(writer, request)
	if !ok {
		return
	}

	var req updateBioRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		http.Error(writer, "invalid body", http.StatusBadRequest)
		return
	}
	req.Bio = strings.TrimSpace(req.Bio)
	if utf8.RuneCountInString(req.Bio) > channels.MaxBio {
		http.Error(writer, fmt.Sprintf("bio must be at most %d characters", channels.MaxBio), http.StatusBadRequest)
		return
	}

	channel, err := h.channelRepo.GetByID(ctx, channelID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(writer, "Channel not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error finding channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	isOwner := channel.OwnerID != nil && *channel.OwnerID == user.ID
	if !h.admins[user.Username] && !isOwner {
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return
	}

	updated, err := h.channelRepo.UpdateBio(ctx, channelID, req.Bio)
	if err != nil {
		log.Printf("Error updating channel bio: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(updated)
	if err != nil {
		log.Printf("Error marshaling value: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}

// Delete : DELETE /channel?channel=ID, admins uniquement, general protégé
func (h channelHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	if !h.admins[user.Username] {
		log.Printf("User %s tried to delete a channel without being admin", user.Username)
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return
	}

	channelID, ok := channelIDFromQuery(writer, request)
	if !ok {
		return
	}

	channel, err := h.channelRepo.GetByID(ctx, channelID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(writer, "Channel not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error finding channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if channel.Name == "general" {
		http.Error(writer, "The general channel cannot be deleted", http.StatusBadRequest)
		return
	}

	err = h.channelRepo.Delete(ctx, channelID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(writer, "Channel not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error deleting channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("Channel %d deleted by %s", channelID, user.Username)
	writer.WriteHeader(http.StatusNoContent)
}
