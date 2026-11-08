package handlers

import (
	"net/http"

	"chat-gobelin-api/internal/users"
)

// currentUser renvoie l'utilisateur authentifié, placé dans le contexte par le middleware d'auth.
// Sans lui (route déclarée publique par erreur), on répond 401 au lieu de paniquer sur une assertion de type.
func currentUser(writer http.ResponseWriter, request *http.Request) (*users.User, bool) {
	user, ok := request.Context().Value("user").(*users.User)
	if !ok || user == nil {
		http.Error(writer, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return user, true
}
