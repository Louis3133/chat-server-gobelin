# User API
The API exposes two endpoints:

## Login
POST **/login**, which is expecting a POST request with a JSON payload of the form:

Require a payload of the form :
```
{"username": "Alice", "password": "password123"}
```

If the credentials are valid, it responds with 200 OK, returning a JWT token in the response body which is valid for 60 minutes.

## Create User
Post **/user**

Create a new user

Take a request of the form :
```
{
	"username":"isabelle",
	"password":"reisen"
}
```

If successful, return:

```
{
  "username": "isabelle",
  "channels": [
    {
      "name": "general",
      "id": 5,
      "creation_date": "2026-11-02T09:43:22.190494Z"
    }
  ],
  "id": 1
}
```

Same information as /me. Users are automatically added to the general channels. Better error handling will happen but for now ask the dev if it return a 500.

## My informations
GET **/me**

Require authentication in the form of a Authorization header with a valid token.

Return relevant informations about the currently logged in user.

```
{
  "username": "isabelle",
  "channels": [
    {
      "name": "goblin-tests",
      "id": 4,
      "creation_date": "2026-11-02T08:20:38.75406Z"
    },
    {
      "name": "general",
      "id": 5,
      "creation_date": "2026-11-02T09:43:22.190494Z"
    }
  ],
  "id": 1
}
```
username is the name of the logged in user
id is the internal ID of the logged in user
channel is the list of channel the user is in

`/me` renvoie aussi `is_admin` (le nom fait partie de `ADMIN_USERNAMES`) : le front s'en sert pour afficher l'administration.

## Delete User
DELETE **/user?user=ID**

**Réservé aux admins** (`ADMIN_USERNAMES`) : 403 sinon. Supprime l'utilisateur avec ses lettres envoyées et reçues, ses
adhésions et ses distances, dans une transaction ; les salons qu'il possédait gardent leur contenu (`owner_id` à `null`).
Son token devient invalide (401). 400 pour `internal-clock` (protégé : l'horloge écrit en son nom), 404 s'il n'existe pas.
Réponse : 204 sans contenu.
