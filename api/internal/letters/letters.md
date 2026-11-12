# Lettres : les règles du jeu

Un salon est un système solaire. Les membres sont des planètes. On s'écrit des **lettres**. **Par défaut, toute lettre
met 1 minute à arriver et les planètes ne se rapprochent pas** (`LETTER_MAX_DELAY=1m`, `LETTER_APPROACH=0`). Les règles
ci-dessous (délai selon la distance, rapprochement) ne s'appliquent que si on les réactive par ces variables.

Le code est dans `rules.go` (les calculs, testés par `rules_test.go`) et `service.go` (envoi d'une lettre et vue
« système solaire », en base). Le service est le seul à écrire dans la table `relations`.

## Les règles
- **Distance** : une par couple et par salon, entre `LETTER_MIN_DISTANCE` (1) et `LETTER_MAX_DISTANCE` (10).
  Deux personnes qui ne se sont jamais répondu sont à 10.
- **Délai** : linéaire entre `LETTER_MIN_DELAY` (distance 1) et `LETTER_MAX_DELAY` (distance 10), tous deux à 1 min par défaut,
  puis **divisé par `LETTER_TIME_SCALE`**.
- **Réponse** : une lettre est une réponse si le dernier message échangé entre les deux (dans ce salon) venait de
  l'autre **et qu'il est déjà arrivé**. Une réponse retire `LETTER_APPROACH` (0 par défaut, donc rien) de la distance, sans descendre
  sous le minimum. Son propre trajet est calculé avec la distance déjà réduite.
- **Pas de rapprochement** : première lettre, plusieurs lettres de suite, ou réponse à une lettre pas encore arrivée.
- La distance ne **remonte pas** quand on cesse de s'écrire (pas encore implémenté).
- **Annulation** : l'expéditeur peut annuler une lettre en route (`Service.Cancel`). Si c'était une réponse, la distance
  d'avant est rendue : sans ça, envoyer puis annuler en boucle rapprocherait à l'infini.

Exemple (avec `LETTER_MAX_DELAY=3h` et `LETTER_APPROACH=0.10`) : distance 10 → 3 h ; après une réponse, 9 → 2 h 40 ; après une autre, 8,1 → 2 h 22.

## Réglages (variables d'environnement de l'API)
| Variable | Défaut | Rôle |
|---|---|---|
| `LETTER_MIN_DISTANCE` | `1` | plus proche possible (plancher) |
| `LETTER_MAX_DISTANCE` | `10` | distance de départ et plafond |
| `LETTER_MIN_DELAY` | `1m` | délai à la distance minimale (durée Go : `90s`, `3h`) |
| `LETTER_MAX_DELAY` | `1m` | délai à la distance maximale (`3h` pour un délai selon la distance) |
| `LETTER_APPROACH` | `0` | part de distance retirée à chaque réponse (0 = jamais, moins de 1 ; `0.10` = 10 %) |
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
