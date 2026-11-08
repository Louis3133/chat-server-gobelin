// Pages protégées : sans token, retour au formulaire de connexion
export default defineNuxtRouteMiddleware(() => {
  if (!useChatApi().getToken()) {
    return navigateTo('/')
  }
})
