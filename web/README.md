# web

Front Nuxt 4 (`ssr: false`) + Vuetify. L'API est attendue sur `http://localhost:4000`, modifiable avec `NUXT_PUBLIC_API_BASE`.

- `pages/index.vue` : connexion / inscription
- `pages/channels/index.vue` : mes salons, rejoindre, créer (admin)
- `pages/channels/[id].vue` : boîte de lettres et envoi
- `composables/useChatApi.ts` : client de l'API (token `chat_goblin_token`)
- `middleware/auth.ts` : renvoie au login sans token

```
npm install
npm run dev     # http://localhost:3000
npm run build
```
