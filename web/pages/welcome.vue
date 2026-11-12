<template>
  <v-dialog :model-value="true" width="420" persistent>
    <div class="bubble-wrap">
      <div class="speaker-wrap">
        <div class="zibzib">
          <img v-show="!mouthOpen" src="/zibzib1.svg" alt="">
          <img v-show="mouthOpen" src="/zibzib2.svg" alt="">
        </div>
        <span class="speaker pixel-btn">Zib zib</span>
      </div>
      <div class="bubble pixel-frame">
        <h1 class="text-subtitle">Bienvenue, voyageur.</h1>
        <div class="texts">
          <p v-for="(text, i) in BUBBLES" :key="i" :class="{ current: i === index }" class="text-body">
            <template v-if="i === index">{{ text.slice(0, typed) }}<span class="typing-rest">{{ text.slice(typed) }}</span></template>
            <template v-else>{{ text }}</template>
          </p>
        </div>
        <button class="pixel-btn next" type="button" @click="next">Continuer</button>
      </div>
    </div>
  </v-dialog>
</template>

<style scoped>
.zibzib {
  width: 100px;
  height: 100px;
  margin-bottom: -6px;
}

.zibzib img {
  display: block;
  width: 100%;
  height: 100%;
}

/* Étiquette de 54px de haut, à cheval sur le cadre */
.bubble-wrap {
  --speaker-half: 27px;
  position: relative;
  margin-top: calc(94px + var(--speaker-half));
  user-select: none;
  -webkit-user-select: none;
}

.speaker-wrap {
  position: absolute;
  left: var(--space-3);
  bottom: calc(100% - var(--speaker-half));
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.speaker {
  position: relative;
  padding: 12px 44px;
  font-size: 20px;
  line-height: 24px;
  cursor: default;
}

.bubble {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: calc(var(--speaker-half) + var(--space-2)) var(--space-3) var(--space-3);
  background: #232323;
  color: #fff;
}

.bubble h1 {
  text-align: center;
  color: #fff;
}

.texts {
  display: grid;
}

.texts p {
  grid-area: 1 / 1;
  visibility: hidden;
}

.texts p.current {
  visibility: visible;
}

.typing-rest {
  visibility: hidden;
}

.next {
  margin: auto auto 0;
}
</style>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const BUBBLES = [
  "Quelque part, au milieu de l'immensité, un nouvel astre vient de naître. Le vôtre.",
  "Vous voici aux portes d'un univers où chaque planète abrite une âme, chaque orbite raconte un lien et chaque lettre transporte un fragment de celui qui l'écrit.",
  'Choisissez votre monde, façonnez-le à votre image et laissez-vous guider par la gravité de vos rencontres.',
  'Au fil des correspondances, les distances se réduisent, les trajectoires se croisent et de nouvelles constellations prennent vie.',
  'Mais prenez garde : les plus beaux voyages ne sont pas toujours les plus rapides.',
  "Certaines lettres méritent que l'on prenne le temps de les écrire, de les confier aux étoiles et d'attendre qu'elles trouvent leur chemin.",
  'Votre univers vous attend. À vos plumes, voyageurs.',
]

const index = ref(0)
const bubble = computed(() => BUBBLES[index.value])

// Apparition lettre par lettre, bulle par bulle
const charsPerMs = 0.04
const fastCharsPerMs = 0.12
const progress = ref(0)
const typed = computed(() => Math.min(Math.round(progress.value), bubble.value.length))

// Zib Zib parle : sa bouche s'ouvre et se ferme tant que le texte s'écrit
const MOUTH_MS = 150
const mouthOpen = ref(false)

const sound = useSound()
let frame = 0
let last = 0
let fast = false
const type = (time: number) => {
  if (last) progress.value += (time - last) * (fast ? fastCharsPerMs : charsPerMs)
  last = time
  if (progress.value < bubble.value.length) {
    mouthOpen.value = Math.floor(time / MOUTH_MS) % 2 === 1
    frame = requestAnimationFrame(type)
  } else {
    mouthOpen.value = false
    sound.stop('zibzib')
  }
}

const startTyping = () => {
  cancelAnimationFrame(frame)
  progress.value = 0
  last = 0
  sound.loop('zibzib')
  frame = requestAnimationFrame(type)
}

// Continuer : finit la bulle en cours, sinon passe à la suivante
const next = () => {
  if (typed.value < bubble.value.length) {
    progress.value = bubble.value.length
    return
  }
  if (index.value === BUBBLES.length - 1) return navigateTo('/planet')
  index.value++
  startTyping()
}

// Accélération tant qu'on maintient espace ou le clic
const onKeyDown = (e: KeyboardEvent) => { if (e.code === 'Space') fast = true }
const onKeyUp = (e: KeyboardEvent) => { if (e.code === 'Space') fast = false }
const onPointerDown = () => { fast = true }
const onPointerUp = () => { fast = false }

onMounted(() => {
  startTyping()
  window.addEventListener('keydown', onKeyDown)
  window.addEventListener('keyup', onKeyUp)
  window.addEventListener('pointerdown', onPointerDown)
  window.addEventListener('pointerup', onPointerUp)
})
onBeforeUnmount(() => {
  sound.stop('zibzib')
  cancelAnimationFrame(frame)
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('keyup', onKeyUp)
  window.removeEventListener('pointerdown', onPointerDown)
  window.removeEventListener('pointerup', onPointerUp)
})
</script>
