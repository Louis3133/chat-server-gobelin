# Roadmap

Une étape = une session de travail courte. Cocher quand c'est terminé. Lire `docs/concept.md` avant de commencer.

## 0. Hygiène du dépôt
- [x] Ignorer `web/node_modules` et les dossiers générés (`web/.gitignore`)
- [x] Retirer le binaire `api/api` de l'index (dossier `api/` supprimé)
- [ ] Retirer le JWT de `memory_help.txt`
- [ ] Commit de nettoyage séparé du reste
- [ ] Formater le Go (`gofmt`) sur les fichiers modifiés

## 1. Sécurité de base de l'API
- [x] Corriger l'exemption d'auth par méthode (`publicRoutes` dans `api.go`) et répondre 401 si l'utilisateur du token n'existe plus
- [x] `GET /users` et `GET /channels` restent publics, par choix (décor du formulaire de connexion)
- [x] `JWT_SECRET` (obligatoire) et `DATABASE_URL` (optionnelle) en variables d'environnement
- [x] Plus de mots de passe, tokens ni corps de requête dans les logs
- [x] Vérifier l'appartenance au salon sur `GET` et `POST /channel/message`
- [x] Contrainte d'unicité sur `user_channels` (migration 00003, `POST /channel/enter` idempotent)

## 2. Modèle de données du concept
- [x] Décisions de `docs/concept.md` : valeurs par défaut adoptées (voir ce fichier), réglables sans toucher au code
- [x] Migration 00002 : `recipient_id` et `deliver_at` sur `messages`
- [x] Migration 00002 : `owner_id` sur les channels (admin)
- [x] Migration 00002 : table `relations` (distance par paire et par channel, `user_a < user_b`)
- [x] Adapter les repos Go (`messages`, `channels`) aux nouvelles colonnes

## 3. Logique de lettres (API)
- [x] Calcul du délai à partir de la distance, avec facteur d'échelle configurable (`internal/letters/rules.go`)
- [x] `POST /channel/message` avec destinataire et `deliver_at`
- [x] `GET /channel/message` filtré sur la livraison (contenu masqué tant que la lettre est en route)
- [x] Mise à jour de la distance à chaque réponse
- [x] Endpoint `GET /channel/system?channel=ID` pour la scène (distances de l'utilisateur uniquement)
- [x] Création de salon réservée aux admins (`ADMIN_USERNAMES`), propriétaire = `owner_id`
- [x] Mettre à jour les `*.md` de l'API (`letters.md`, `message.md`, `channel.md`)
- [ ] Remontée de la distance quand on cesse de s'écrire (hors MVP)
- [ ] Tests automatisés de `service.go` (SQL) : aujourd'hui seulement des essais manuels sur Postgres jetable
- [ ] Lettres de groupe : non (les messages de salon sans destinataire restent livrés tout de suite)

## 4. Front fonctionnel sans 3D
- [x] Supprimer `web/src/` et les restes de l'appli de virements
- [x] Unifier le token sur `useChatApi` (`chat_goblin_token`)
- [x] Pages : login/inscription, liste des channels, boîte de lettres, envoi d'une lettre
- [x] Corriger le `Dockerfile` web et le port pour Nuxt (Node 22, port 8080, volumes anonymes pour `node_modules` et `.nuxt`)
- [ ] Essai de bout en bout dans le navigateur (`docker compose up --build`) : seul `nuxt build` a été vérifié
- [x] Dépendances Vite/Vuex/axios/eslint retirées de `web/package.json`, `yarn.lock` supprimé : npm uniquement

## 5. Scène Three.js
- [x] Soleil, planètes et orbites à partir de `/system`
- [x] Sélection d'une planète et ouverture de la conversation
- [x] Animation des lettres en transit
- [x] Rapprochement visible des planètes (la planète glisse vers sa nouvelle place au rechargement ; pas encore de rafraîchissement automatique, cf. étape 6)
- [x] Scène plein écran en fond dès la page de connexion (décor des habitants publics, caméra libre, étoiles)
- [ ] Essai visuel dans le navigateur de la scène : seul `nuxt build` a été vérifié

## 5 bis. Personnalisation de la planète
- [x] Migration 00005 : inclinaison, couleur et taille des anneaux, vitesse de rotation (couleur, taille, anneaux existaient déjà)
- [x] `PUT /me/planet` (apparence complète, modifiable à tout moment), champs renvoyés par `/users`, `/me`, `/channel/system`
- [x] Page `/planet` après les dialogues d'accueil : caméra sur sa planète de `general`, aperçu en direct ; accessible ensuite depuis « Mes salons »
- [ ] Essai dans le navigateur : seuls `nuxt build` et l'API (Postgres jetable) ont été vérifiés
- [x] Page en 3 étapes (texture et couleur ; taille, rotation, inclinaison ; anneaux ou lune) ; lune proportionnelle à la planète, exclusive avec les anneaux (migration 00007)
- [x] Texture importée : réduite par le navigateur (JPEG 256×128), stockée dans Postgres (migration 00006, pas de service externe), servie par `GET /user/texture` (publique)
- [ ] Les autres navigateurs ne voient une nouvelle apparence qu'au rechargement (pas de temps réel, cf. étape 6)

## 5 ter. Lettres dans la scène
- [x] Caméra sur la fusée d'une lettre qu'on vient d'envoyer
- [x] Le destinataire clique sur une fusée posée : le nez s'ouvre et la lettre s'affiche
- [x] Retirer la fusée de sa planète (« pouf »), la lettre reste (migration 00008, `DELETE /channel/message/rocket`)
- [ ] Essai dans le navigateur (ouverture du nez, pouf) : seuls `nuxt build` et l'API (Postgres jetable) ont été vérifiés

## 5 quater. API partagée de l'atelier
- [x] `useChatApi` reconstruit sur le serveur commun (métadonnées du namespace `epistellaire`), même interface pour les pages
- [x] Règles des lettres, distances et visibilité portées côté front (`utils/letterRules.ts`, `utils/sharedApi.ts`)
- [x] Essai de bout en bout contre une copie locale du serveur de la prof (18 scénarios, pages sans erreur)
- [x] Variable Vercel `NUXT_PUBLIC_API_BASE`
- [x] Variable Vercel `NUXT_PUBLIC_ADMIN_USERNAMES`
- [x] Retirer `api/`, `db/` et leurs services de `docker-compose.yml`
- [ ] Arrêter Render et Neon
- [ ] Temps réel avec le WebSocket de l'API partagée (`/channel/socket`)

## 6. Finitions
- [ ] Temps réel (polling puis WebSocket si besoin)
- [ ] Textures, sons, polish visuel
- [ ] Mode démo avec temps accéléré
