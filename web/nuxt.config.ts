// nuxt.config.ts
export default defineNuxtConfig({
  compatibilityDate: '2024-04-03',
  devtools: { enabled: true },
  css: ['@mdi/font/css/materialdesignicons.min.css', '~/assets/css/fonts.css'],
  ssr: false,
  modules: ['vuetify-nuxt-module'],
  runtimeConfig: {
    public: {
      // Surchargeable avec NUXT_PUBLIC_API_BASE
      apiBase: 'http://localhost:4000',
    },
  },
})
