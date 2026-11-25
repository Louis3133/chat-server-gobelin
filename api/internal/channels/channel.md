# Channel API

## List channels
GET **/channels** (alias : **/channel**)

**Public, sans token, par choix** : le système solaire sert de décor au formulaire de connexion. Renvoie tous les salons,
avec leur `owner_id` (l'admin qui l'a créé, `null` pour `general`), `locked` (protégé par un mot de passe ; le hash
n'est jamais renvoyé) et `member_ids` (ids des membres, pour peupler les
soleils des autres salons dans la scène ; absent si le salon est vide). Les noms se lisent dans `GET /users`.

## Create Channel
Post **/channel**

Create a new channel. **Réservé aux admins** (`ADMIN_USERNAMES`, voir `../letters/letters.md`) : 403 sinon, 400 si le nom
est vide. Le créateur devient `owner_id` du salon et en est membre. `password` est facultatif : s'il est fourni, le salon
est protégé (72 octets au plus, hash bcrypt).

Take a request of the form :
```
{
	"name":"ECNI",
	"password":"facultatif"
}
```

If successful, return:

```
{
  "name": "goblin-test",
  "id": 4,
  "owner_id": 1,
  "creation_date": "2026-11-04T09:00:00Z"
}
```

## Enter Channel
Post **/channel/enter**

Require authentication in the form of a Authorization header with a valid token.

Enter a new channel. It will add the channel to the list of the channel the user is in. Entrer deux fois dans le même
salon est sans effet (pas de doublon). 404 si le salon n'existe pas.

**Salon protégé** (`locked`) : le mot de passe n'est demandé qu'à l'entrée, une seule fois. Un membre ou un admin entre
sans. Sinon : 403 `password required` sans mot de passe, 403 `wrong password` s'il est faux.

Take a request of the form :
```
{
	"channel":"ECNI",
	"password":"seulement pour un salon protégé"
}
```

If successful, return the channel

## Bio du salon
PUT **/channel/bio?channel=ID**

Authentification requise. **Réservé aux admins (`ADMIN_USERNAMES`) et au propriétaire du salon** (`owner_id`) : 403 sinon,
404 si le salon n'existe pas. Les espaces autour sont retirés ; 500 caractères au plus (400 au-delà, contrainte de la
migration 00011). Une bio vide est permise.

```
{"bio": "Le salon avec tous les BG d'Ecni"}
```

Renvoie le salon mis à jour. La bio est aussi renvoyée par `GET /channels` et dans `channel` de `GET /channel/system` ;
le front l'affiche dans la modale ouverte au clic sur le soleil (`web/components/ChannelProfile.vue`).

## Mot de passe du salon
PUT **/channel/password?channel=ID**

Authentification requise. **Réservé aux admins et au propriétaire du salon** : 403 sinon, 404 si le salon n'existe pas.
Un mot de passe vide retire la protection ; les membres actuels restent membres. `general` (affiché « Libre ») reste
ouvert : 400 si on tente de le protéger.

```
{"password": "nouveau mot de passe"}
```

Renvoie le salon mis à jour (`locked`).

## Renommer un salon
PUT **/channel/name?channel=ID**

Authentification requise. **Réservé aux admins et au propriétaire du salon** : 403 sinon, 404 si le salon n'existe pas.
Espaces autour retirés, 30 caractères au plus, non vide (400). 409 si un autre salon porte déjà ce nom. `general` (affiché
« Libre ») ne se renomme pas et son nom est réservé (400) : l'API et la scène le retrouvent par son nom.

```
{"name": "ECNI 2028"}
```

Renvoie le salon mis à jour. Les membres, lettres et distances suivent (tout est lié à l'id du salon).

## Delete Channel
DELETE **/channel?channel=ID**

**Réservé aux admins** (`ADMIN_USERNAMES`) : 403 sinon. Supprime le salon avec toutes ses lettres, ses membres et ses
distances, dans une transaction. 400 pour `general` (le salon Libre, gardé en mémoire par l'API), 404 si le salon
n'existe pas. Réponse : 204 sans contenu.
