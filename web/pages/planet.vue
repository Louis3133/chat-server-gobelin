<template>
  <div class="planet-editor">
    <header ref="headerEl" class="editor-header pixel-frame">
      <div class="header-row">
        <v-btn v-if="step === 1" title="Quitter" aria-label="Quitter" @click="navigateTo('/channels')"><img class="pixel-icon" src="/icons/actions/close.svg" alt=""></v-btn>
        <v-btn v-else title="Retour" aria-label="Retour" @click="step--"><img class="pixel-icon" src="/icons/actions/back.svg" alt=""></v-btn>
        <h1 class="text-subtitle">Personnalisation</h1>
        <span class="text-body">{{ step }}/{{ STEPS.length }}</span>
      </div>
      <div class="progress">
        <span v-for="n in STEPS.length" :key="n" :class="{ done: n <= step }" />
      </div>
    </header>

    <section ref="panelEl" class="editor-panel pixel-frame">
      <h2 class="text-subtitle">{{ STEPS[step - 1].question }}</h2>
      <p class="hint text-body">{{ STEPS[step - 1].hint }}</p>

      <div v-if="look" class="panel-body pixel-scroll">
        <template v-if="step === 1">
          <label class="name-field">
            <span v-if="profile.planet_name" class="name-label">Nom de la planète</span>
            <input v-model="profile.planet_name" placeholder="Nom de la planète" autocomplete="off">
          </label>
          <div class="field-help text-body">
            <span :class="{ invalid: !nameAllowed }">Lettres, chiffres et espaces</span>
            <span :class="{ invalid: nameTooLong }">{{ profile.planet_name.length }}/{{ NAME_MAX }}</span>
          </div>
        </template>

        <template v-else-if="step === 2">
          <div class="tabs" role="tablist">
            <button class="pixel-tab" :class="{ off: surfaceTab !== 'colors' }" type="button" role="tab" :aria-selected="surfaceTab === 'colors'" @click="surfaceTab = 'colors'">Couleurs</button>
            <button class="pixel-tab" :class="{ off: surfaceTab !== 'image' }" type="button" role="tab" :aria-selected="surfaceTab === 'image'" @click="surfaceTab = 'image'">Image</button>
          </div>

          <div v-if="surfaceTab === 'colors'" class="swatches">
            <button
              v-for="color in PRESET_COLORS"
              :key="color"
              class="swatch-btn"
              :class="{ selected: look.planet_color === color }"
              type="button"
              :title="color"
              :aria-label="`Couleur ${color}`"
              @click="look.planet_color = color"
            >
              <span class="swatch-fill" :style="{ background: color }" />
            </button>
            <label class="swatch-btn custom" :class="{ selected: customColor }" title="Autre couleur">
              <span v-if="customColor" class="swatch-fill" :style="{ background: look.planet_color }" />
              <span class="swatch-plus">+</span>
              <input v-model="look.planet_color" type="color" aria-label="Autre couleur">
            </label>
          </div>

          <div v-else class="image-tab">
            <button class="pixel-btn wide" type="button" @click="picker?.click()">{{ customizing?.texture ? "Changer d'image" : 'Importer une image' }}</button>
            <button v-if="customizing?.texture" class="pixel-btn wide off" type="button" @click="removeTexture">Retirer l'image</button>
          </div>
          <input ref="picker" type="file" accept="image/*" hidden @change="onFile">
        </template>

        <template v-else-if="step === 3">
          <p class="slider-label text-subtitle-medium">Taille de la planète</p>
          <PixelSlider v-model="look.planet_size" label="Taille de la planète" :min="0.4" :max="1" :step="0.01" />
        </template>

        <template v-else-if="step === 4">
          <div class="tabs three" role="tablist">
            <button
              v-for="option in ORNAMENTS"
              :key="option.value"
              class="pixel-tab"
              :class="{ off: ornament !== option.value }"
              type="button"
              role="tab"
              :aria-selected="ornament === option.value"
              @click="ornament = option.value"
            >
              {{ option.label }}
            </button>
          </div>
          <template v-if="look.planet_rings">
            <div class="color-row mb-2">
              <span>Couleur de l’anneau</span>
              <input v-model="look.planet_ring_color" type="color" aria-label="Couleur de l’anneau">
            </div>
            <p class="slider-label text-subtitle-medium">Taille de l’anneau</p>
            <PixelSlider v-model="look.planet_ring_size" label="Taille de l’anneau" :min="1.5" :max="4" :step="0.05" />
          </template>
        </template>

        <template v-else-if="step === 5">
          <p class="slider-label text-subtitle-medium">Vitesse de la planète</p>
          <PixelSlider v-model="look.planet_spin" label="Vitesse de la planète" :min="-0.1" :max="0.1" :step="0.001" />
          <p class="slider-label text-subtitle-medium">Rotation de la planète</p>
          <PixelSlider v-model="look.planet_tilt" label="Rotation de la planète" :min="-1.2" :max="1.2" :step="0.05" />
        </template>

        <template v-else>
          <ul class="recap">
            <li v-for="(line, index) in recap" :key="line.label">
              <button type="button" :title="`Modifier : ${line.label}`" @click="step = index + 1">
                <span class="recap-label">{{ line.label }}</span>
                <span class="recap-value">
                  <span v-if="line.color" class="swatch" :style="{ background: line.color }" />
                  {{ line.value }}
                </span>
                <img class="pixel-icon" src="/icons/actions/edit.svg" alt="">
              </button>
            </li>
          </ul>
        </template>
      </div>

      <div class="actions">
        <button v-if="step < STEPS.length" class="pixel-btn next" type="button" :disabled="!canContinue" @click="step++">Continuer</button>
        <button v-else class="pixel-btn next" type="button" :disabled="!canContinue || saving" @click="save">{{ saving ? 'Lancement…' : "C'est mon monde" }}</button>
      </div>
    </section>
    <v-snackbar v-model="snackbar" color="red" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const api = useChatApi()
const { decor, customizing, focusArea } = useScene()

const picker = ref<HTMLInputElement>()
const saving = ref(false)
const snackbar = ref(false)
const snackbarText = ref('')

// Les réglages modifient directement l'état partagé : la scène montre l'aperçu en direct
const look = computed(() => customizing.value?.look)
const profile = ref<Profile>({ planet_name: '', pronouns: '', bio: '' })

const STEPS = [
  { question: "Comment s'appelle ta planète\u00a0?", hint: "C'est le nom que verront les autres planètes du système" },
  { question: 'À quoi ressemble ta planète\u00a0?', hint: 'Choisis une couleur et/ou une image.' },
  { question: 'Quelle taille pour ta planète\u00a0?', hint: 'C’est la taille que verront toutes les autres planètes du système.' },
  { question: 'Une lune ou un anneau\u00a0?', hint: 'Ajoute-lui un compagnon, ou garde-la seule' },
  { question: 'À quelle vitesse tourne-t-elle\u00a0?', hint: 'Lentement ou vite, dans un sens ou dans l\'autre' },
  { question: 'Ta planète est prête', hint: 'Tu pourras modifier tout cela plus tard\u00a0!' },
]
const step = ref(1)

// Nom : obligatoire, lettres, chiffres et espaces
const NAME_MAX = 16
const nameAllowed = computed(() => /^[\p{L}\p{N} ]*$/u.test(profile.value.planet_name))
const nameTooLong = computed(() => profile.value.planet_name.length > NAME_MAX)
const nameValid = computed(() => nameAllowed.value && profile.value.planet_name.trim().length > 0 && !nameTooLong.value)
const canContinue = computed(() => !!look.value && (step.value !== 1 || nameValid.value))

// Surface : couleurs de base ou couleur libre (+), image importée teintée par la couleur
const PRESET_COLORS = ['#ffffff', '#8d9db6', '#3d5a80', '#5fa8d3', '#81b29a', '#f2cc8f', '#e07a5f', '#c1121f', '#9b5de5']
const surfaceTab = ref<'colors' | 'image'>('colors')
const customColor = computed(() => !!look.value && !PRESET_COLORS.includes(look.value.planet_color.toLowerCase()))

// Anneau ou lune, jamais les deux (l'API le refuse aussi)
const ORNAMENTS = [
  { value: 'none', label: 'Rien' },
  { value: 'moon', label: 'Lune' },
  { value: 'rings', label: 'Anneau' },
] as const
const ornament = computed({
  get: () => (look.value?.planet_rings ? 'rings' : look.value?.planet_moon ? 'moon' : 'none'),
  set: (value) => {
    if (!look.value) return
    look.value.planet_rings = value === 'rings'
    look.value.planet_moon = value === 'moon'
  },
})

// Récapitulatif
const recap = computed(() => {
  const l = look.value
  if (!l) return []
  const spin = Math.round(Math.abs(l.planet_spin) * 1000)
  return [
    { label: 'Nom', value: profile.value.planet_name.trim() || 'Sans nom' },
    { label: 'Apparence', value: customizing.value?.texture ? 'Image importée' : 'Couleur', color: l.planet_color },
    { label: 'Taille', value: `${Math.round(l.planet_size * 100)} %` },
    { label: 'Ornement', value: l.planet_moon ? 'Une lune' : l.planet_rings ? 'Un anneau' : 'Aucun', color: l.planet_rings ? l.planet_ring_color : undefined },
    { label: 'Rotation', value: spin ? `${spin} %, ${l.planet_spin > 0 ? 'sens direct' : 'sens inverse'}` : 'Immobile' },
  ]
})

const notify = (error: unknown) => {
  snackbarText.value = typeof error === 'string' ? error : apiErrorMessage(error)
  snackbar.value = true
}

// Texture : réduite ici (256×128, le format d'une carte de planète) puis envoyée seulement à la validation
const TEXTURE_WIDTH = 256
const TEXTURE_HEIGHT = 128
let pending: Blob | null = null
let removed = false
let preview: string | null = null

const shrink = async (file: File) => {
  const image = await createImageBitmap(file)
  const canvas = document.createElement('canvas')
  canvas.width = TEXTURE_WIDTH
  canvas.height = TEXTURE_HEIGHT
  const scale = Math.max(TEXTURE_WIDTH / image.width, TEXTURE_HEIGHT / image.height)
  const w = image.width * scale
  const h = image.height * scale
  canvas.getContext('2d')!.drawImage(image, (TEXTURE_WIDTH - w) / 2, (TEXTURE_HEIGHT - h) / 2, w, h)
  return new Promise<Blob>((resolve, reject) =>
    canvas.toBlob((blob) => (blob ? resolve(blob) : reject(new Error())), 'image/jpeg', 0.9))
}

const showTexture = (url: string | null) => {
  if (preview) URL.revokeObjectURL(preview)
  preview = url?.startsWith('blob:') ? url : null
  customizing.value!.texture = url
}

// L'image remplace la couleur : on passe au blanc pour la voir telle quelle (la couleur la teinte ensuite)
const onFile = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !customizing.value) return
  try {
    pending = await shrink(file)
  } catch {
    return notify('Image illisible')
  }
  removed = false
  showTexture(URL.createObjectURL(pending))
  customizing.value.look.planet_color = '#ffffff'
}

const removeTexture = () => {
  pending = null
  removed = true
  showTexture(null)
}

// Valeurs par défaut de l'API : la planète grise d'un nouveau compte
const lookOf = (me: UserMe): PlanetLook => ({
  planet_color: me.planet_color ?? '#8d9db6',
  planet_size: me.planet_size ?? 0.55,
  planet_rings: me.planet_rings ?? false,
  planet_tilt: me.planet_tilt ?? 0,
  planet_ring_color: me.planet_ring_color ?? '#d9c9a3',
  planet_ring_size: me.planet_ring_size ?? 2,
  planet_spin: me.planet_spin ?? 0.01,
  planet_moon: me.planet_moon ?? false,
})

const save = async () => {
  if (!customizing.value) return
  saving.value = true
  try {
    if (pending) await api.uploadTexture(pending)
    else if (removed) await api.deleteTexture()
    pending = null
    removed = false
    await api.updatePlanet(customizing.value.look)
    await api.updateProfile(profile.value)
    await navigateTo('/channels')
  } catch (error) {
    notify(error)
  } finally {
    saving.value = false
  }
}

// Décor rechargé : un compte tout neuf n'y figure pas encore
onMounted(async () => {
  try {
    const [me, users] = await Promise.all([api.getMe(), api.getUsers()])
    decor.value = buildDecor(users ?? [])
    profile.value = { planet_name: me.planet_name ?? '', pronouns: me.pronouns ?? '', bio: me.bio ?? '' }
    const version = me.planet_texture_version ?? 0
    customizing.value = { id: me.id, look: lookOf(me), texture: version ? api.textureUrl(me.id, version) : null }
    if (version) surfaceTab.value = 'image'
  } catch (error) {
    notify(error)
  }
})

// Espace libre entre l'en-tête et le panneau, suivi quand ils changent de taille
const headerEl = ref<HTMLElement>()
const panelEl = ref<HTMLElement>()
const measure = () => {
  if (!headerEl.value || !panelEl.value) return
  focusArea.value = { top: headerEl.value.getBoundingClientRect().bottom, bottom: panelEl.value.getBoundingClientRect().top }
}
const observer = new ResizeObserver(measure)
onMounted(() => {
  if (headerEl.value) observer.observe(headerEl.value)
  if (panelEl.value) observer.observe(panelEl.value)
  window.addEventListener('resize', measure)
  measure()
})

onBeforeUnmount(() => {
  observer.disconnect()
  window.removeEventListener('resize', measure)
  focusArea.value = null
  customizing.value = null
  if (preview) URL.revokeObjectURL(preview)
})
</script>

<style scoped>
.planet-editor {
  position: fixed;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: flex-end;
  gap: var(--space-2);
  padding: var(--space-2);
}

.editor-header,
.editor-panel {
  width: 400px;
  max-width: 100%;
  padding: var(--space-2);
  pointer-events: auto;
}

.header-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.header-row h1 {
  flex: 1;
  text-align: center;
}

.progress {
  display: flex;
  gap: var(--space-1);
  margin-top: var(--space-2);
}

.progress span {
  flex: 1;
  height: 4px;
  background: rgba(255, 255, 255, 0.35);
}

.progress span.done {
  background: var(--Zib);
}

.editor-panel {
  display: flex;
  flex-direction: column;
  max-height: 65vh;
}

.panel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.editor-panel h2 {
  text-align: center;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid rgba(255, 255, 255, 0.25);
}

.hint {
  margin: var(--space-2) 0;
}

.name-field {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 52px;
  padding: 4px var(--space-1);
  background: #fff;
  color: #000;
}

.name-field input::placeholder {
  color: #737373;
  opacity: 1;
}

.name-label {
  font-size: 11px;
  opacity: 0.7;
}

.name-field input,
.name-field textarea {
  resize: none;
  color: #000;
  outline: none;
}

.field-help {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-1);
  opacity: 0.8;
}

.field-help .invalid {
  color: #ff8a80;
  opacity: 1;
}

.color-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.color-row input {
  width: 48px;
  height: 32px;
  padding: 0;
  border: 1px solid rgba(255, 255, 255, 0.4);
  background: none;
  cursor: pointer;
}

.slider-label {
  margin-bottom: var(--space-1);
}

.pixel-slider + .slider-label {
  margin-top: var(--space-3);
}

.tabs {
  display: grid;
  grid-auto-columns: 1fr;
  grid-auto-flow: column;
  gap: var(--space-4);
  margin-bottom: var(--space-3);
}

.tabs.three {
  gap: var(--space-2);
}

.tabs .pixel-tab,
.image-tab .wide {
  width: 100% !important;
  padding: var(--space-1) var(--space-2);
  font-size: var(--subtitle-size);
}

.pixel-btn.off {
  border-image: linear-gradient(#fff, #fff) 1;
  background: #232323 !important;
}

.swatches {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: var(--space-2);
}

.swatch-btn {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  aspect-ratio: 1;
  color: #fff;
  cursor: pointer;
}

.swatch-btn::before,
.swatch-fill {
  content: '';
  position: absolute;
  mask-size: 100% 100%;
  -webkit-mask-size: 100% 100%;
}

.swatch-btn::before {
  inset: 0;
  z-index: 1;
  background: currentColor;
  mask-image: url('/pixel-ring.svg');
  -webkit-mask-image: url('/pixel-ring.svg');
}

.swatch-fill {
  inset: 18.75%;
  mask-image: url('/pixel-disc.svg');
  -webkit-mask-image: url('/pixel-disc.svg');
}

.swatch-btn.selected {
  color: #b5e48c;
}

.swatch-btn.selected .swatch-fill {
  inset: 12.5%;
}

.swatch-plus {
  position: relative;
  z-index: 1;
  font-size: 24px;
  line-height: 1;
  text-shadow: 0 0 3px #000;
}

.swatch-btn input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.image-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.recap {
  list-style: none;
  padding: 0;
  margin-bottom: var(--space-2);
}

.recap button {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  width: 100%;
  padding: var(--space-1) 0;
  color: inherit;
  text-align: left;
}

.recap-label {
  flex: none;
  width: 90px;
  opacity: 0.7;
}

.recap-value {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 6px;
}

.swatch {
  width: 14px;
  height: 14px;
  border: 1px solid rgba(255, 255, 255, 0.4);
}

.actions {
  display: flex;
  gap: var(--space-1);
  margin-top: auto;
  padding-top: var(--space-3);
}

.actions .next {
  flex: 1;
  white-space: nowrap;
  padding: var(--space-2);
  font-size: var(--subtitle-size);
}

.actions button:disabled {
  opacity: 0.5;
  cursor: default;
}

@media (min-width: 600px) {
  .planet-editor {
    align-items: center;
  }

  .editor-header,
  .editor-panel {
    width: 560px;
  }

  .editor-panel {
    padding: var(--space-3);
    max-height: 55vh;
  }

  .swatches {
    grid-template-columns: repeat(5, 40px);
    justify-content: center;
  }

  .swatch-plus {
    font-size: 18px;
  }
}

@media (max-width: 599px) {
  .planet-editor {
    align-items: stretch;
    padding: var(--space-1);
  }

  .editor-header,
  .editor-panel {
    width: 100%;
  }

  .editor-panel {
    min-height: 42vh;
  }
}
</style>
