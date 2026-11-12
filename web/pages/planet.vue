<template>
  <div class="planet-editor">
    <v-card class="editor pa-6">
      <h1 class="text-h5 mb-2">Ta planète</h1>
      <p class="mb-4">Façonne-la à ton image. Tu pourras la retoucher plus tard.</p>

      <template v-if="look">
        <p class="step-title mb-4">Étape {{ step }} / {{ STEPS.length }} · {{ STEPS[step - 1] }}</p>

        <template v-if="step === 1">
          <div class="color-row mb-4">
            <span>Texture</span>
            <div class="d-flex ga-2">
              <v-btn size="small" variant="outlined" prepend-icon="mdi-image" @click="picker?.click()">Importer</v-btn>
              <v-btn v-if="customizing?.texture" size="small" variant="text" icon="mdi-close" title="Retirer l'image" aria-label="Retirer l'image" @click="removeTexture" />
            </div>
            <input ref="picker" type="file" accept="image/*" hidden @change="onFile">
          </div>
          <div class="color-row mb-2">
            <span>Couleur</span>
            <input v-model="look.planet_color" type="color" aria-label="Couleur de la planète">
          </div>
        </template>

        <template v-else-if="step === 2">
          <v-slider v-model="look.planet_size" label="Taille" :min="0.4" :max="1" :step="0.01" color="white" hide-details />
          <v-slider v-model="look.planet_spin" label="Rotation" :min="-0.1" :max="0.1" :step="0.001" color="white" hide-details />
          <v-slider v-model="look.planet_tilt" label="Inclinaison" :min="-1.2" :max="1.2" :step="0.05" color="white" hide-details />
        </template>

        <template v-else-if="step === 3">
          <v-btn-toggle v-model="ornament" class="mb-4" mandatory divided variant="outlined" density="comfortable">
            <v-btn value="none">Rien</v-btn>
            <v-btn value="rings">Anneaux</v-btn>
            <v-btn value="moon">Lune</v-btn>
          </v-btn-toggle>
          <template v-if="look.planet_rings">
            <div class="color-row mb-2">
              <span>Couleur des anneaux</span>
              <input v-model="look.planet_ring_color" type="color" aria-label="Couleur des anneaux">
            </div>
            <v-slider v-model="look.planet_ring_size" label="Taille des anneaux" :min="1.5" :max="4" :step="0.05" color="white" hide-details />
          </template>
        </template>

        <template v-else>
          <v-text-field v-model="profile.planet_name" label="Nom de ta planète" :counter="40" maxlength="40" />
          <v-text-field v-model="profile.pronouns" label="Tes pronoms" :counter="30" maxlength="30" />
          <v-textarea v-model="profile.bio" label="Ta bio" :counter="500" maxlength="500" rows="3" auto-grow />
        </template>
      </template>

      <div class="d-flex ga-2 mt-6">
        <v-btn v-if="step > 1" variant="text" @click="step--">Retour</v-btn>
        <v-spacer />
        <v-btn v-if="step < STEPS.length" color="primary" :disabled="!look" @click="step++">Suivant</v-btn>
        <v-btn v-else color="primary" :loading="saving" :disabled="!look" @click="save">C'est mon monde</v-btn>
      </div>
    </v-card>
    <v-snackbar v-model="snackbar" color="red" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const api = useChatApi()
const { decor, customizing } = useScene()

const picker = ref<HTMLInputElement>()
const saving = ref(false)
const snackbar = ref(false)
const snackbarText = ref('')

// Les réglages modifient directement l'état partagé : la scène montre l'aperçu en direct
const look = computed(() => customizing.value?.look)
const profile = ref<Profile>({ planet_name: '', pronouns: '', bio: '' })

const STEPS = ['Texture et couleur', 'Taille, rotation et inclinaison', 'Anneaux ou lune', 'Ton profil']
const step = ref(1)

// Anneaux ou lune, jamais les deux (l'API le refuse aussi)
const ornament = computed({
  get: () => (look.value?.planet_rings ? 'rings' : look.value?.planet_moon ? 'moon' : 'none'),
  set: (value) => {
    if (!look.value) return
    look.value.planet_rings = value === 'rings'
    look.value.planet_moon = value === 'moon'
  },
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
  } catch (error) {
    notify(error)
  }
})

onBeforeUnmount(() => {
  customizing.value = null
  if (preview) URL.revokeObjectURL(preview)
})
</script>

<style scoped>
.planet-editor {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  min-height: 100%;
  padding: 16px;
}

.editor {
  width: 360px;
  max-width: 100%;
}

.step-title {
  opacity: 0.7;
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

@media (max-width: 600px) {
  .planet-editor {
    align-items: flex-end;
  }
}
</style>
