# Concept : messagerie système solaire

## Idée
- Un **channel** est un système solaire. Le channel (ex. « Ecni 2028 ») est le **soleil**, créé par un **admin**.
- Chaque **utilisateur** du channel est une **planète** en orbite.
- On s'envoie des **lettres** (messages à une personne précise).
- Une lettre n'arrive pas instantanément : plus la **distance** entre deux planètes est grande, plus le **délai de livraison** est long.
- Plus deux personnes échangent, plus leur distance **diminue** : leurs lettres arrivent plus vite.

## Règles métier (proposition, à valider en équipe)
1. Une lettre a un expéditeur, un destinataire unique et un channel.
2. `deliver_at = envoi + délai(distance)`. Tant que `deliver_at` n'est pas passé, le destinataire ne peut pas lire le contenu.
3. Le destinataire voit qu'une lettre est en route et son heure d'arrivée, mais pas son contenu. L'expéditeur voit ses lettres en transit.
4. La distance est stockée côté **serveur**, par paire d'utilisateurs et par channel. Le front ne fait que l'afficher.
5. Chaque échange diminue la distance, avec un **plancher** (jamais zéro). Sans échange pendant longtemps, elle remonte lentement, avec un **plafond**.
6. Le délai se calcule avec un **facteur d'échelle de temps** réglable (config serveur), pour pouvoir tester et faire des démos (secondes) tout en gardant la possibilité d'un vrai rythme (heures).

## Décisions (valeurs par défaut adoptées, réglables sans toucher au code)
- [x] Modèle de positions 3D : **simple** pour la scène. **Attention** : l'API stocke une distance **par couple** (c'est elle qui décide du délai) ; un rayon d'orbite par planète ne peut pas représenter toutes les distances par couple. La scène (étape 5) devra donc calculer une disposition approximative à partir des distances renvoyées par `/channel/system` (celles de l'utilisateur qui regarde).
- [x] Formule du délai : **linéaire**, 1 minute (distance 1) à 3 heures (distance 10).
- [x] Formule de rapprochement : **10 % par réponse**, plancher 1, plafond 10, distance de départ 10. Seules comptent les **réponses** (le dernier message du couple venait de l'autre et est arrivé).
- [ ] La distance **remonte** quand on cesse de s'écrire : pas dans le MVP.
- [x] Échelle de temps : `LETTER_TIME_SCALE` (1 = réel, 3600 = démo, une heure simulée = une seconde).
- [x] Qui crée un channel : **seulement les admins** désignés par `ADMIN_USERNAMES`.
- [ ] Un utilisateur peut être dans plusieurs channels : oui (le schéma le permet, la distance est par salon).
- [ ] Le channel `general` créé automatiquement est conservé (membres automatiques, sans propriétaire).
- [x] Lettres de groupe : **non** pour le MVP. Un message sans destinataire reste un message de salon, livré tout de suite.
- [ ] Temps réel : polling pour le MVP, WebSocket ensuite.

## Modèle de données cible (brouillon)
Existant : `users`, `channels`, `user_channels`, `messages` (voir `api/migrations/00001_*`).

**Fait** : migration `api/migrations/00002_letters_admin_relations` (testée en montée, descente et remontée sur Postgres 14.1). Choix retenus, par défaut, en attendant la décision de l'équipe :
- `messages.recipient_id` est **nullable** (les messages de salon existants et l'horloge restent valides) et `messages.deliver_at` vaut `NOW()` par défaut (livraison immédiate pour l'ancien code).
- Admin = `channels.owner_id`, nullable (`general` n'a pas de propriétaire). Une colonne de rôle reste possible plus tard.
- `relations` : une ligne par paire avec `user_a < user_b`, `distance > 0`, pas de distance de départ imposée par la base.
- Pas de colonnes d'orbite : modèle de positions « simple » non encore tranché.

Prévu à l'origine (brouillon) :
- `channels.owner_id` (admin du channel) ou une colonne `role` sur `user_channels`.
- `messages.recipient_id` (destinataire de la lettre) et `messages.deliver_at` (date de livraison).
- Table `relations` : `channel_id`, `user_a`, `user_b` (avec `user_a < user_b` pour l'unicité), `distance`, `last_interaction`, `exchange_count`.
- (Optionnel) `user_channels.orbit_radius` et `orbit_angle` si le modèle de positions simple est retenu.

## Impact sur l'API (brouillon)
- `POST /channel/message` : accepte `recipient_id`, calcule `deliver_at` à partir de la distance, met à jour la relation.
- `GET /channel/message` : ne renvoie le contenu que si `deliver_at <= now()` ; sinon renvoie les métadonnées (expéditeur, `deliver_at`).
- Nouveau `GET /channel/{id}/system` : soleil, planètes et distances pour alimenter la scène 3D.
- Rôle admin vérifié sur la création et la gestion des channels.

## Front (Three.js)
- Composant client unique pour la scène (Nuxt en `ssr: false`). TresJS envisageable pour rester dans l'écosystème Vue.
- Éléments : soleil, planètes (une par utilisateur), orbites, lettres en vol (animation du départ à `deliver_at`), caméra libre.
- Cliquer sur une planète ouvre la conversation avec cette personne.
- Prévoir un mode « version liste » sans 3D comme secours et pour les tests.

## Risques
- La géométrie : des distances par paires ne se placent pas toutes sur des orbites autour d'un soleil, d'où le choix du modèle simple.
- Temps de développement du 3D créatif : figer l'API avant.
- Démo : sans facteur d'échelle, l'attente des lettres rend les tests pénibles.
