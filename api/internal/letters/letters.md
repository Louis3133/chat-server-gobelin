# Lettres : les règles du jeu

Un salon est un système solaire. Les membres sont des planètes. On s'écrit des **lettres** : plus deux planètes sont
éloignées, plus la lettre met de temps à arriver. Plus on se répond, plus on se rapproche.

Le code est dans `rules.go` (les calculs, testés par `rules_test.go`) et `service.go` (envoi d'une lettre et vue
« système solaire », en base). Le service est le seul à écrire dans la table `relations`.

## Les règles
- **Distance** : une par couple et par salon, entre `LETTER_MIN_DISTANCE` (1) et `LETTER_MAX_DISTANCE` (10).
  Deux personnes qui ne se sont jamais répondu sont à 10.
- **Délai** : linéaire entre `LETTER_MIN_DELAY` (1 min, distance 1) et `LETTER_MAX_DELAY` (3 h, distance 10),
  puis **divisé par `LETTER_TIME_SCALE`**.
- **Réponse** : une lettre est une réponse si le dernier message échangé entre les deux (dans ce salon) venait de
  l'autre **et qu'il est déjà arrivé**. Une réponse retire `LETTER_APPROACH` (10 %) de la distance, sans descendre
  sous le minimum. Son propre trajet est calculé avec la distance déjà réduite.
- **Pas de rapprochement** : première lettre, plusieurs lettres de suite, ou réponse à une lettre pas encore arrivée.
- La distance ne **remonte pas** quand on cesse de s'écrire (pas encore implémenté).

Exemple (réel) : distance 10 → 3 h ; après une réponse, 9 → 2 h 40 ; après une autre, 8,1 → 2 h 22.

## Réglages (variables d'environnement de l'API)
| Variable | Défaut | Rôle |
|---|---|---|
| `LETTER_MIN_DISTANCE` | `1` | plus proche possible (plancher) |
| `LETTER_MAX_DISTANCE` | `10` | distance de départ et plafond |
| `LETTER_MIN_DELAY` | `1m` | délai à la distance minimale (durée Go : `90s`, `3h`) |
| `LETTER_MAX_DELAY` | `3h` | délai à la distance maximale |
| `LETTER_APPROACH` | `0.10` | part de distance retirée à chaque réponse (entre 0 et 1 exclus) |
| `LETTER_TIME_SCALE` | `1` | 1 = temps réel ; **3600 = mode démo** (une heure simulée dure une seconde) |
| `ADMIN_USERNAMES` | vide | noms autorisés à créer un salon, séparés par des virgules. Vide : personne |

Une valeur invalide empêche l'API de démarrer (mieux vaut échouer que tourner avec des règles que personne n'a voulues).
En local, les mettre dans `.env` à la racine (voir `.env.example`), puis `docker compose up -d api`.

## Confidentialité
- Une lettre en route n'est **jamais** envoyée à son destinataire avec son contenu : la requête SQL la vide
  (`content` = `""`, `in_transit` = `true`), il ne voit que l'expéditeur et la date d'arrivée (`deliver_at`).
- L'expéditeur voit ses propres lettres, en route ou non. Personne ne voit les lettres des autres.
- `GET /channel/system` ne renvoie que **les distances de celui qui demande**, pas celles des autres couples.
- Il faut être membre du salon pour y lire, y écrire, ou voir son système (sinon 403).

## Endpoints
Voir `../messages/message.md` (`POST` et `GET /channel/message`, `GET /channel/system`) et `../channels/channel.md`.
