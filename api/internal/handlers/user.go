package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
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

var planetColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// planetFromRequest valide l'apparence demandée ; les valeurs absentes prennent celles par défaut (celles de la base).
func planetFromRequest(req loginRequest) (users.Planet, error) {
	planet := users.Planet{Color: "#8d9db6", Size: 0.55, Rings: req.PlanetRings}
	if req.PlanetColor != nil {
		if !planetColorRe.MatchString(*req.PlanetColor) {
			return planet, errors.New("planet_color must look like #rrggbb")
		}
		planet.Color = strings.ToLower(*req.PlanetColor)
	}
	if req.PlanetSize != nil {
		if !(*req.PlanetSize >= users.MinPlanetSize && *req.PlanetSize <= users.MaxPlanetSize) {
			return planet, fmt.Errorf("planet_size must be between %v and %v", users.MinPlanetSize, users.MaxPlanetSize)
		}
		planet.Size = *req.PlanetSize
	}
	return planet, nil
}

type UserPresentation struct {
	Username string `json:"username"`
	Channels []channels.Channel `json:"channels"`
	ID       int64  `json:"id"`
	PlanetColor string  `json:"planet_color"`
	PlanetSize  float64 `json:"planet_size"`
	PlanetRings bool    `json:"planet_rings"`
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
		PlanetColor: user.PlanetColor,
		PlanetSize: user.PlanetSize,
		PlanetRings: user.PlanetRings,
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
