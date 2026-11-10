// Outils de réglage : actifs avec #debug dans l'URL, jusqu'au rechargement
export default defineNuxtPlugin(() => {
  const debug = useState('debug', () => location.hash === '#debug')
  window.addEventListener('hashchange', () => {
    if (location.hash === '#debug' && !debug.value) location.reload()
  })
})
