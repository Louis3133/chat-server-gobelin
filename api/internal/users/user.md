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

Same information as /me. Un nouvel utilisateur n'est dans aucun salon : il choisit le sien dans la vue d'ensemble (`POST /channel/enter`). Better error handling will happen but for now ask the dev if it return a 500.

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

## Apparence de sa planète
PUT **/me/planet**

Authentification requise. Remplace l'apparence complète de sa propre planète (tous les champs sont obligatoires) ;
l'étape de personnalisation qui suit l'inscription (`web/pages/planet.vue`) s'en sert, et on peut y revenir ensuite.

```
{
  "planet_color": "#ff0000",       // #rrggbb
  "planet_size": 0.9,              // 0.4 à 1
  "planet_rings": true,
  "planet_tilt": 0.3,              // -1.2 à 1.2 (radians)
  "planet_ring_color": "#00ff00",  // #rrggbb
  "planet_ring_size": 3,           // 1.5 à 4 (multiple du rayon)
  "planet_spin": -0.02,            // -0.1 à 0.1 (radians par image)
  "planet_moon": false             // petite lune ; jamais avec planet_rings (400, et contrainte en base)
}
```

Réponse 200 : l'utilisateur mis à jour (mêmes champs que `GET /users`). 400 avec un message lisible si une valeur
sort des bornes (vérifiées par `Planet.Check` dans `users/models.go` et par les contraintes des migrations 00004 et 00005).
Ces champs sont aussi renvoyés par `GET /users`, `GET /me`, `POST /user`, et par `GET /channel/system` sous les noms
`color`, `size`, `rings`, `tilt`, `ring_color`, `ring_size`, `spin`, `moon`.

## Profil de sa planète
PUT **/me/profile**

Authentification requise. Remplace le profil affiché au clic sur sa planète (les trois champs ; une chaîne vide efface) :

```
{
  "planet_name": "Kepler-22b",   // 40 caractères au plus
  "pronouns": "elle",            // 30 caractères au plus
  "bio": "..."                   // 500 caractères au plus
}
```

Les espaces autour sont retirés. Réponse 200 : l'utilisateur mis à jour. 400 si un champ est trop long (`Profile.Clean`
dans `users/models.go`, et contraintes de la migration 00009). Ces champs sont aussi renvoyés par `GET /users`, `GET /me`
et `GET /channel/system` (donc publics, comme l'apparence).

## Anniversaire
PUT **/me/birthday** (authentification requise) : `{"month": 3, "day": 14}`, ou `null` pour l'effacer. Jour et mois
seulement ; 400 si la date n'existe pas (le 29 février est accepté). Réponse 200 : `{"birthday": {...}}`.
**Privé** : renvoyé uniquement par `GET /me` (`birthday`, `null` sans date), jamais par `GET /users`.

## Texture de sa planète
PUT **/me/planet/texture** (authentification requise) : le corps de la requête est l'image elle-même, JPEG ou PNG,
256 Ko et 1024×1024 au plus (le front l'a déjà réduite en JPEG 256×128). 413 si elle est trop lourde, 400 si ce n'est
pas une image JPEG/PNG ou si elle est trop grande. Réponse 200 : l'utilisateur mis à jour.

DELETE **/me/planet/texture** (authentification requise) : retire la texture, retour à la roche par défaut.

GET **/user/texture?user=ID** : **publique** (le décor de la page de connexion affiche toutes les planètes). Renvoie
l'image (404 sans texture). `planet_texture_version` (`texture_version` dans `/channel/system`) vaut 0 sans texture
et change à chaque envoi : le front l'ajoute à l'URL (`&v=N`), ce qui permet un cache navigateur d'un an.

## Zib Zib (PNJ)
Compte `zibzib` (`is_npc: true`), créé au démarrage de l'API avec le nom de planète « Zib Zib ». Son hash de mot de passe
est invalide : personne ne peut se connecter à sa place. Il n'est membre d'aucun salon (donc pas de lettres pour l'instant).
`GET /users` le renvoie avec `is_npc` ; le front l'affiche seul, hors des salons.

**Réservé aux admins** (403 sinon), mêmes corps et réponses que leurs équivalents `/me` :
PUT **/npc/planet**, PUT **/npc/profile**, PUT et DELETE **/npc/planet/texture**.

GET **/npc/birthday-letter** et PUT **/npc/birthday-letter** `{"text": "..."}` (admins, 600 caractères au plus, non vide) :
texte de la lettre d'anniversaire, `{pseudo}` remplacé par le nom d'utilisateur. Texte par défaut dans
`birthdays/models.go`, valeur modifiée dans la table `settings`.

GET **/npc/letters** (authentification requise) : les lettres échangées avec Zib Zib, dans les deux sens, au format des
lettres de salon (contenu d'une lettre de Zib Zib vide tant qu'elle est en route ; une réponse n'apparaît qu'à son départ). Si son anniversaire tombe aujourd'hui ou dans les 6 jours précédents (heure de
Paris ; le 29 février fêté le 28 les années non bissextiles) et qu'aucune lettre n'est partie cette année, l'appel en
envoie une : hors salon (`channel_id = 0`), en vol 1 minute. Une seule par personne et par an (table
`birthday_letters`). Sa fusée se retire avec `DELETE /channel/message/rocket`.

GET **/npc/mailbox** (authentification requise) : `{"open": true}` si l'on peut écrire à Zib Zib, c'est-à-dire si les
admins ont écrit sa réponse.

GET **/npc/reply-letter** et PUT **/npc/reply-letter** `{"text": "..."}` (admins, 600 caractères au plus) : texte de la
réponse de Zib Zib, `{pseudo}` remplacé par le nom d'utilisateur. Vide (par défaut) : la boîte aux lettres est fermée.
Valeur dans la table `settings` (clé `npc_reply`).

POST **/npc/letter** `{"content": "..."}` (authentification requise) : une lettre à Zib Zib, hors salon (`channel_id = 0`),
en vol 10 minutes. Sa réponse est enregistrée dans la même transaction, avec le texte du moment : elle part 10 minutes après
l'arrivée et vole 10 minutes (règles dans `mailbox/models.go`). 409 si la boîte est fermée, 400 si le contenu est vide ou
dépasse 100 000 caractères, 403 pour le PNJ. Une lettre à Zib Zib ne s'annule pas (`DELETE /channel/message` répond 404).
Réponse : la lettre envoyée.

## Delete User
DELETE **/user?user=ID**

**Réservé aux admins** (`ADMIN_USERNAMES`) : 403 sinon. Supprime l'utilisateur avec ses lettres envoyées et reçues, ses
adhésions et ses distances, dans une transaction ; les salons qu'il possédait gardent leur contenu (`owner_id` à `null`).
Son token devient invalide (401). 400 pour `internal-clock` (protégé : l'horloge écrit en son nom) et pour le PNJ, 404
s'il n'existe pas.
Réponse : 204 sans contenu.
