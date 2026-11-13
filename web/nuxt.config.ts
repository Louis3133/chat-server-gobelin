// nuxt.config.ts
export default defineNuxtConfig({
  compatibilityDate: '2024-04-03',
  devtools: { enabled: false },
  css: ['@mdi/font/css/materialdesignicons.min.css', '~/assets/css/fonts.css', '~/assets/css/typography.css', '~/assets/css/ui-kit.css'],
  ssr: false,
  modules: ['vuetify-nuxt-module'],
  runtimeConfig: {
    public: {
      // API partagée de l'atelier (NUXT_PUBLIC_API_BASE), données d'Epistellaire sous NUXT_PUBLIC_API_NAMESPACE
      apiBase: 'https://gobelins-chat-api.leomouraire.com:4000',
      apiNamespace: 'epistellaire',
      adminUsernames: '',
      // Règles des lettres (NUXT_PUBLIC_LETTER_*)
      letterMinDistance: 1,
      letterMaxDistance: 10,
      letterMinDelay: '1m',
      letterMaxDelay: '1m',
      letterApproach: 0,
      letterTimeScale: 1,
    },
  },
})
