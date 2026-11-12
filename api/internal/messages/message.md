# Message API
Toutes ces routes exigent un token (`Authorization: Bearer ...`) et que l'utilisateur soit **membre du salon**
(sinon 403 ; salon inconnu : 404). Règles des lettres : `../letters/letters.md`.

## Send a message or a letter
POST **/channel/message**

Sans `recipient_id`, le message est adressé à tout le salon et arrive tout de suite. Avec `recipient_id`, c'est une
**lettre** pour une personne du salon : elle n'arrive qu'à `deliver_at`, selon la distance entre les deux planètes.

```
{
  "content": "salut !",
  "channel": 4,
  "recipient_id": 2
}
```

Réponse (200) :

```
{
  "content": "salut !",
  "user_id": 1,
  "user": "isabelle",
  "recipient_id": 2,
  "recipient": null,
  "creation_date": "2026-11-04T09:00:00.1Z",
  "deliver_at": "2026-11-04T12:00:00.1Z",
  "in_transit": true
}
```

Erreurs : 400 (corps invalide, contenu vide ou de plus de 100 000 caractères, lettre à soi-même, destinataire hors du salon),
403 (pas membre du salon), 404 (salon inconnu).

## Get messages
GET **/channel/message?channel=4**

Les 100 derniers messages visibles par l'utilisateur, du plus ancien au plus récent, **classés par date de livraison** :
- les messages adressés à tout le salon (`recipient_id` null) ;
- les lettres qu'il a envoyées, avec leur contenu, même si elles sont encore en route ;
- les lettres qui lui sont adressées : arrivées avec leur contenu ; **en route sans contenu** (`content` vide,
  `in_transit` vrai) pour qu'il voie qu'une lettre arrive, et quand.

Les lettres entre d'autres personnes n'apparaissent jamais.

```
[
  {
    "content": "",
    "user_id": 2,
    "user": "sanae",
    "recipient_id": 1,
    "recipient": "isabelle",
    "creation_date": "2026-11-04T09:00:00Z",
    "deliver_at": "2026-11-04T11:40:00Z",
    "in_transit": true
  }
]
```

## Get the solar system
GET **/channel/system?channel=4**

Le salon vu par l'utilisateur : le soleil (le salon, avec son `owner_id`), les planètes (les membres), **sa** distance à
chacune et le temps qu'une lettre mettrait maintenant. Les distances des autres couples ne sont jamais renvoyées.

```
{
  "channel": { "name": "ecni", "id": 4, "owner_id": 1, "creation_date": "..." },
  "planets": [
    { "id": 1, "username": "isabelle", "me": true, "distance": null, "delay_seconds": null, "exchanges": 0,
      "planet_name": "Kepler-22b", "pronouns": "elle", "bio": "..." },
    { "id": 2, "username": "sanae", "me": false, "distance": 9, "delay_seconds": 9600, "exchanges": 1 }
  ],
  "min_distance": 1,
  "max_distance": 10,
  "time_scale": 1
}
```

`delay_seconds` est un temps réel (déjà divisé par `time_scale`). `exchanges` compte les réponses échangées avec cette personne. Chaque planète porte aussi son apparence et le profil de la personne (`planet_name`, `pronouns`, `bio`, cf. `PUT /me/profile`).

## Websocket

To be implemented

## Annuler une lettre en route
DELETE **/channel/message?message=ID** (authentification requise)

L'expéditeur annule sa lettre tant qu'elle n'est pas arrivée : elle est supprimée (le destinataire ne la voit plus), et
si c'était une réponse, le couple retrouve la distance d'avant ce rapprochement (`approached_from`, migration 00010).
Réponse 204. 404 si la lettre n'existe pas, n'est pas la sienne ou est déjà arrivée.

## Retirer la fusée d'une lettre reçue
DELETE **/channel/message/rocket?message=ID** (authentification requise)

Le destinataire retire la fusée posée sur sa planète, pour faire de la place : la lettre reste lisible dans
`GET /channel/message`, avec `"rocket_removed": true`, et la scène ne l'affiche plus (ni chez lui ni chez l'expéditeur).
Réponse 204. 404 si la lettre n'existe pas, ne lui est pas adressée ou est encore en route. Les messages renvoient
aussi leur `id`, qui sert à désigner la lettre ici.
