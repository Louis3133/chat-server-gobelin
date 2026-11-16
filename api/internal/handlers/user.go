package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"chat-gobelin-api/internal/users"
	"chat-gobelin-api/internal/channels"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type userHandler struct {
	userRepo users.Repo
	channelRepo channels.Repo
	general channels.Channel
	admins map[string]bool // noms d'utilisateur autorisés à supprimer un utilisateur (ADMIN_USERNAMES)
}

func NewUserHandler(s users.Repo, c channels.Repo, channel channels.Channel, admins map[string]bool) (*userHandler, error) {
	return &userHandler{
		userRepo: s,
		channelRepo: c,
		general: channel,
		admins: admins,
	}, nil
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// Apparence de la planète, lue seulement à l'inscription (POST /user). Tous les champs sont facultatifs.
	PlanetColor *string  `json:"planet_color"`
	PlanetSize  *float64 `json:"planet_size"`
	PlanetRings bool     `json:"planet_rings"`
}

// planetFromRequest valide l'apparence demandée ; les valeurs absentes prennent celles par défaut (celles de la base).
func planetFromRequest(req loginRequest) (users.Planet, error) {
	planet := users.DefaultPlanet()
	planet.Rings = req.PlanetRings
	if req.PlanetColor != nil {
		planet.Color = strings.ToLower(*req.PlanetColor)
	}
	if req.PlanetSize != nil {
		planet.Size = *req.PlanetSize
	}
	return planet, planet.Check()
}

type UserPresentation struct {
	Username string `json:"username"`
	Channels []channels.Channel `json:"channels"`
	ID       int64  `json:"id"`
	users.Planet
	users.Profile
	IsAdmin     bool    `json:"is_admin"`
}

var JWTKey []byte // défini au démarrage depuis JWT_SECRET (cmd/api/api.go)

type Claims struct {
	Username string `json:"username"`
	ID int64 `json:"id"`
	jwt.StandardClaims
}

func (h userHandler) Login(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)

	var req loginRequest
	err := decoder.Decode(&req)

	if err != nil {
		log.Printf("Error decoding request: %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.GetByUserName(request.Context(), req.Username)
	if err != nil {
		log.Printf("User retrieval issue: %s", err)
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(req.Password))
	if err != nil {
		log.Printf("Wrong Password: %s", err)
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(3 * 24 * 60 * time.Minute)
	claims := &Claims{
		Username: user.Username,
		ID: user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JWTKey)
	if err != nil {
		log.Printf("Error generating JWT token %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(tokenString))

	log.Printf("User %s succesfully logged in", req.Username)
}

func (h userHandler) Me(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	channels, err := h.channelRepo.GetByUser(ctx, *user)
	if err != nil {
		log.Printf("Error retrieving channels: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	presentation := UserPresentation{
		Username: user.Username,
		ID: user.ID,
		Channels: channels,
		Planet: user.Planet,
		Profile: user.Profile,
		IsAdmin: h.admins[user.Username],
	}
	result, err := json.Marshal(presentation)
	if err != nil {
		log.Printf("Error marshaling value: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}

// UpdatePlanet : PUT /me/planet, l'apparence complète de sa propre planète
func (h userHandler) UpdatePlanet(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	var planet users.Planet
	if err := json.NewDecoder(request.Body).Decode(&planet); err != nil {
		http.Error(writer, "invalid body", http.StatusBadRequest)
		return
	}
	planet.Color = strings.ToLower(planet.Color)
	planet.RingColor = strings.ToLower(planet.RingColor)
	if err := planet.Check(); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := h.userRepo.UpdatePlanet(request.Context(), user.ID, planet)
	if err != nil {
		log.Printf("Error updating planet: %s", err.Error())
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

// UpdateProfile : PUT /me/profile, le nom de sa planète, ses pronoms et sa bio
func (h userHandler) UpdateProfile(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	var profile users.Profile
	if err := json.NewDecoder(request.Body).Decode(&profile); err != nil {
		http.Error(writer, "invalid body", http.StatusBadRequest)
		return
	}
	if err := profile.Clean(); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	updated, err := h.userRepo.UpdateProfile(request.Context(), user.ID, profile)
	if err != nil {
		log.Printf("Error updating profile: %s", err.Error())
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

// Texture importée : petite image JPEG ou PNG, déjà réduite par le navigateur
const (
	maxTextureBytes = 256 << 10
	maxTextureSide  = 1024
)

// UploadTexture : PUT /me/planet/texture, le corps est l'image
func (h userHandler) UploadTexture(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}

	data, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, maxTextureBytes))
	if err != nil {
		http.Error(writer, fmt.Sprintf("texture must be at most %d KB", maxTextureBytes>>10), http.StatusRequestEntityTooLarge)
		return
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		http.Error(writer, "texture must be a JPEG or PNG image", http.StatusBadRequest)
		return
	}
	if config.Width > maxTextureSide || config.Height > maxTextureSide {
		http.Error(writer, fmt.Sprintf("texture must be at most %dx%d", maxTextureSide, maxTextureSide), http.StatusBadRequest)
		return
	}
	h.saveTexture(writer, request, user.ID, data)
}

// DeleteTexture : DELETE /me/planet/texture, retour à la roche par défaut
func (h userHandler) DeleteTexture(writer http.ResponseWriter, request *http.Request) {
	user, ok := currentUser(writer, request)
	if !ok {
		return
	}
	h.saveTexture(writer, request, user.ID, nil)
}

func (h userHandler) saveTexture(writer http.ResponseWriter, request *http.Request, id int64, data []byte) {
	updated, err := h.userRepo.SetTexture(request.Context(), id, data)
	if err != nil {
		log.Printf("Error saving texture: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(writer, updated)
}

// Texture : GET /user/texture?user=ID, publique comme le décor qui l'affiche ; l'URL porte la version, d'où le cache long
func (h userHandler) Texture(writer http.ResponseWriter, request *http.Request) {
	userID, err := strconv.ParseInt(request.URL.Query().Get("user"), 10, 64)
	if err != nil {
		http.Error(writer, "user parameter required", http.StatusBadRequest)
		return
	}
	data, err := h.userRepo.GetTexture(request.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && data == nil) {
		http.Error(writer, "Texture not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error retrieving texture: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", http.DetectContentType(data))
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	writer.Write(data)
}

func (h userHandler) Create(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	decoder := json.NewDecoder(request.Body)

	var req loginRequest
	err := decoder.Decode(&req)

	if err != nil {
		log.Printf("Error decoding request: %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	planet, err := planetFromRequest(req)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	//minimum cost for now
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 4)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := h.userRepo.Create(ctx, req.Username, hash, planet)
	if err != nil {
		log.Printf("Error creating user: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = h.channelRepo.AddUserToChannel(ctx, h.general, *user)
	if err != nil {
		log.Printf("Error adding user to general channel: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshaling value: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write(result)
}

func (h userHandler) GetAll(writer http.ResponseWriter, request *http.Request) {
  ctx := request.Context()

  userList, err := h.userRepo.GetAll(ctx)
  if err != nil {
    log.Printf("Error retrieving all users: %s", err.Error())
    writer.WriteHeader(http.StatusInternalServerError)
    return
  }

  result, err := json.Marshal(userList)
  if err != nil {
    log.Printf("Error marshaling users: %s", err.Error())
    writer.WriteHeader(http.StatusInternalServerError)
    return
  }

  writer.Header().Set("Content-Type", "application/json")
  writer.WriteHeader(http.StatusOK)
  writer.Write(result)
}


// Delete : DELETE /user?user=ID, admins uniquement, internal-clock protégé
func (h userHandler) Delete(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	admin, ok := currentUser(writer, request)
	if !ok {
		return
	}
	if !h.admins[admin.Username] {
		log.Printf("User %s tried to delete a user without being admin", admin.Username)
		http.Error(writer, "Forbidden", http.StatusForbidden)
		return
	}

	raw := request.URL.Query().Get("user")
	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		http.Error(writer, "user parameter required", http.StatusBadRequest)
		return
	}

	target, err := h.userRepo.GetByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(writer, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error finding user: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	if target.Username == "internal-clock" {
		http.Error(writer, "The internal clock user cannot be deleted", http.StatusBadRequest)
		return
	}

	err = h.userRepo.Delete(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(writer, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error deleting user: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("User %d deleted by %s", userID, admin.Username)
	writer.WriteHeader(http.StatusNoContent)
}
