<template>
  <v-dialog class="dialog-content"  :model-value="true" max-width="420" persistent>
    <img class="zibzib" src="/zibzib.svg" alt="">
    <div class="intro-button-text">
<div class="intro pixel-frame">
      <h1 class="text-h5">Bienvenue, voyageur.</h1>
      <div class="intro-content">
      <p>{{ text.slice(0, typed) }}<span class="typing-rest">{{ text.slice(typed) }}</span></p>
      <p>{{ signature.slice(0, signatureTyped) }}<span class="typing-rest">{{ signature.slice(signatureTyped) }}</span></p>
      </div>
    </div>
    </div>
    <button class="btn-intro pixel-btn" type="button" @click="navigateTo('/planet')">Continuer</button>
  </v-dialog>
</template>

<style scoped>

.intro-button-text {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.btn-intro {
  color: white;
  justify-content: center;
  display: flex;
  margin-inline: auto;
  margin-top: 32px;
}

.intro {
  padding: var(--space-3);
}

.intro-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
  color: white;
}

.zibzib {
  width: 120px;
  height: 120px;
  margin-left: 20px;
}

.typing-rest {
  visibility: hidden;
}

.text-h5 {
  padding: unset;
  margin-bottom: 12px;
  color: white;
}

</style>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const text = "Quelque part, au milieu de l'immensité, un nouvel astre vient de naître. Le vôtre. Vous voici aux portes d'un univers où chaque planète abrite une âme, chaque orbite raconte un lien et chaque lettre transporte un fragment de celui qui l'écrit. Choisissez votre monde, façonnez-le à votre image et laissez-vous guider par la gravité de vos rencontres. Au fil des correspondances, les distances se réduisent, les trajectoires se croisent et de nouvelles constellations prennent vie. Mais prenez garde : les plus beaux voyages ne sont pas toujours les plus rapides. Certaines lettres méritent que l'on prenne le temps de les écrire, de les confier aux étoiles et d'attendre qu'elles trouvent leur chemin. Votre univers vous attend. À vos plumes, voyageurs."
const signature = '-Zib Zib'

// Apparition lettre par lettre
const charsPerMs = 0.04
const fastCharsPerMs = 0.12
const progress = ref(0)
const typed = computed(() => Math.min(Math.round(progress.value), text.length))
const signatureTyped = computed(() => Math.max(0, Math.round(progress.value) - text.length))

let frame = 0
let last = 0
let fast = false
const type = (time: number) => {
  if (last) progress.value += (time - last) * (fast ? fastCharsPerMs : charsPerMs)
  last = time
  if (progress.value < text.length + signature.length) frame = requestAnimationFrame(type)
}

// Accélération tant qu'on maintient espace ou le clic
const onKeyDown = (e: KeyboardEvent) => { if (e.code === 'Space') fast = true }
const onKeyUp = (e: KeyboardEvent) => { if (e.code === 'Space') fast = false }
const onPointerDown = () => { fast = true }
const onPointerUp = () => { fast = false }

onMounted(() => {
  frame = requestAnimationFrame(type)
  window.addEventListener('keydown', onKeyDown)
  window.addEventListener('keyup', onKeyUp)
  window.addEventListener('pointerdown', onPointerDown)
  window.addEventListener('pointerup', onPointerUp)
})
onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('keyup', onKeyUp)
  window.removeEventListener('pointerdown', onPointerDown)
  window.removeEventListener('pointerup', onPointerUp)
})
</script>
