<template>
  <Teleport to="body">
    <form v-if="open && settings" class="planet-profile planet-settings scene-panel pixel-frame" @submit.prevent="save">
      <div class="planet-settings__header">
        <h2 class="text-heading">{{ planet.username }}</h2>
        <button class="pixel-btn planet-settings__validate" type="submit" :disabled="saving" title="Enregistrer" aria-label="Enregistrer">
          <img class="pixel-icon" src="/icons/actions/validate.svg" alt="">
        </button>
      </div>
      <label class="planet-settings__row">
        <img class="pixel-icon" src="/icons/actions/earth.svg" alt="Nom de la planète">
        <input v-model="draft.planet_name" class="planet-settings__field text-body" placeholder="Nom de ta planète" maxlength="22" :readonly="saving" autocomplete="off">
      </label>
      <label class="planet-settings__row">
        <img class="pixel-icon" src="/icons/actions/pronoms.svg" alt="Pronoms">
        <input v-model="draft.pronouns" class="planet-settings__field text-body" placeholder="Tes pronoms" maxlength="30" :readonly="saving" autocomplete="off">
      </label>
      <label class="planet-settings__row planet-settings__row--bio">
        <img class="pixel-icon" src="/icons/actions/bio.svg" alt="Bio">
        <textarea v-model="draft.bio" class="planet-settings__field text-body pixel-scroll" placeholder="Ta bio" maxlength="500" :rows="xs ? 3 : 6" :readonly="saving" />
      </label>
      <NuxtLink class="pixel-btn planet-settings__customize" to="/planet">Personnaliser ma planète</NuxtLink>
    </form>

    <div v-else-if="open" ref="panel" class="planet-profile scene-panel pixel-frame">
      <h2 class="planet-profile__title text-heading">
        {{ planet.planet_name || `Planète de ${planet.username}` }}
        <img class="pixel-icon" src="/icons/actions/earth.svg" alt="">
      </h2>

      <ul class="planet-profile__infos">
        <li>
          <img class="pixel-icon" src="/icons/actions/pseudo.svg" alt="Pseudo">
          <span class="text-body">{{ planet.me ? `${planet.username} (moi)` : planet.username }}</span>
        </li>
        <li v-if="planet.pronouns">
          <img class="pixel-icon" src="/icons/actions/pronoms.svg" alt="Pronoms">
          <span class="text-body">{{ planet.pronouns }}</span>
        </li>
        <li v-if="planet.bio" class="planet-profile__bio-row">
          <img class="pixel-icon" src="/icons/actions/bio.svg" alt="Bio">
          <span class="planet-profile__bio text-body" :class="{ 'planet-profile__bio--full': shown }">{{ planet.bio }}</span>
        </li>
      </ul>

      <div v-if="planet.me" class="planet-profile__ctas">
        <button class="pixel-btn planet-profile__write" type="button" @click="openSettings">
          Mes réglages
          <img class="pixel-icon" src="/icons/actions/settings.svg" alt="">
        </button>
      </div>

      <div v-else class="planet-profile__ctas">
        <button class="pixel-btn planet-profile__write" type="button" @click="emit('write')">
          Envoyer une lettre
          <img class="pixel-icon" src="/icons/actions/send.svg" alt="">
        </button>
        <button class="planet-profile__detail" type="button" :aria-expanded="expanded" @click="toggle">
          {{ expanded ? 'Masquer le détail' : 'Voir le détail' }}
        </button>
      </div>

      <div v-if="shown && !planet.me" ref="more" class="planet-profile__more">
        <hr class="planet-profile__separator">
        <h3 class="text-subtitle-medium">Vos lettres</h3>
        <p v-if="!exchanged.length" class="text-body">Aucune lettre échangée pour l'instant.</p>
        <ul v-else class="planet-profile__letters pixel-scroll">
          <li v-for="m in exchanged" :key="m.id">
            <button class="letter-item pixel-frame" type="button" :disabled="!readable(m)" @click="emit('read', m)">
              <PlanetAvatar :planet="planetOf(m.user_id)" />
              <span class="letter-item__text">
                <span class="text-body">{{ m.user_id === planet.id ? `Lettre de ${planet.username}` : 'Ta lettre' }}</span>
                <span class="text-body letter-item__date">{{ status(m) }}</span>
              </span>
            </button>
          </li>
        </ul>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const props = defineProps<{ planet: Planet, letters: Message[] }>()
const emit = defineEmits<{ write: [], read: [letter: Message], failed: [message: string] }>()
const api = useChatApi()
const { xs } = useDisplay()

const open = defineModel<boolean>({ default: false })
const expanded = ref(false)
const shown = ref(false)
const panel = ref<HTMLElement>()
const more = ref<HTMLElement>()
const { system, decor } = useScene()
let resize: Animation | undefined

watch(open, (value) => {
  if (value) return
  resize?.cancel()
  expanded.value = shown.value = settings.value = false
})

// Réglages de sa propre planète : nom, pronoms et bio modifiables
const settings = ref(false)
const saving = ref(false)
const draft = reactive<Profile>({ planet_name: '', pronouns: '', bio: '' })

const openSettings = () => {
  draft.planet_name = props.planet.planet_name ?? ''
  draft.pronouns = props.planet.pronouns ?? ''
  draft.bio = props.planet.bio ?? ''
  settings.value = true
}

const save = async () => {
  saving.value = true
  try {
    const user = await api.updateProfile({ ...draft })
    const profile = { planet_name: user.planet_name ?? '', pronouns: user.pronouns ?? '', bio: user.bio ?? '' }
    for (const sys of [system, decor]) {
      if (!sys.value) continue
      sys.value = { ...sys.value, planets: sys.value.planets.map((p) => (p.id === props.planet.id ? { ...p, ...profile } : p)) }
    }
    settings.value = false
  } catch (error) {
    emit('failed', apiErrorMessage(error))
  } finally {
    saving.value = false
  }
}

// Ouverture et fermeture animées de « Plus d'infos »
const heightWith = async (value: boolean) => {
  shown.value = value
  await nextTick()
  return panel.value?.offsetHeight ?? 0
}

const toggle = async () => {
  const target = !expanded.value
  expanded.value = target
  const el = panel.value
  if (!el || matchMedia('(prefers-reduced-motion: reduce)').matches) {
    shown.value = target
    return
  }
  const from = el.offsetHeight
  resize?.cancel()
  more.value?.getAnimations().forEach((a) => a.cancel())
  const to = await heightWith(target)
  if (!target) await heightWith(true)
  resize = el.animate([{ height: `${from}px` }, { height: `${to}px` }], { duration: 300, easing: 'cubic-bezier(0.2, 0.8, 0.2, 1)' })
  if (target) more.value?.animate([{ opacity: 0, transform: 'translateY(8px)' }, { opacity: 1, transform: 'none' }], { duration: 250, delay: 80, easing: 'ease-out', fill: 'backwards' })
  else more.value?.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 150, fill: 'forwards' })
  resize.onfinish = () => { shown.value = target }
}

// Seulement les lettres entre cette personne et moi, les plus récentes d'abord
const exchanged = computed(() => {
  const me = system.value?.planets.find((p) => p.me)?.id
  const them = props.planet.id
  return props.letters
    .filter((m) => (m.user_id === them && m.recipient_id === me) || (m.user_id === me && m.recipient_id === them))
    .sort((a, b) => b.creation_date.localeCompare(a.creation_date))
})

const readable = (m: Message) => !!parseLetter(m.content)

const planetOf = (id: number) => system.value?.planets.find((p) => p.id === id)

const formatDate = (iso: string) => new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })

const status = (m: Message) => {
  if (m.in_transit) return `En route, arrivée le ${formatDate(m.deliver_at)}`
  return m.user_id === props.planet.id ? `Reçue le ${formatDate(m.deliver_at)}` : `Envoyée le ${formatDate(m.creation_date)}`
}
</script>

<style scoped>
.planet-profile {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-height: 0;
  padding: var(--space-3);
}

.planet-profile > * {
  flex: none;
}

.planet-profile__title {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-1) !important;
  text-align: center;
  overflow-wrap: anywhere;
}

.planet-profile__infos {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}

.planet-profile__infos li {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  overflow-wrap: anywhere;
}

.planet-profile__bio-row {
  align-items: flex-start !important;
}

.planet-profile__ctas {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.planet-profile__write {
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  font-size: var(--subtitle-size);
}

.planet-profile__write .pixel-icon {
  width: 32px;
  height: 32px;
}

.planet-profile__detail {
  color: var(--color-text);
  font-size: var(--body-size);
  text-decoration: underline;
  transition: color 0.15s;
}

.planet-profile__detail:hover {
  color: #d3fbb2;
}

.planet-profile h2,
.planet-profile h3,
.planet-profile p {
  margin: 0;
}

.planet-profile__bio {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  overflow: hidden;
  white-space: pre-line;
}

.planet-profile__bio--full {
  display: block;
}

.planet-profile__more {
  display: flex;
  flex: 0 1 auto;
  flex-direction: column;
  gap: var(--space-2);
  min-height: 0;
}

.planet-profile__more > * {
  flex: none;
}

.planet-profile__separator {
  margin: var(--space-1) 0;
  border: none;
  border-top: var(--stroke-width) solid var(--color-text);
}

.planet-profile__letters {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  gap: var(--space-2);
  min-height: 0;
  margin: 0;
  padding: 0 var(--space-2) 0 0;
  overflow-y: auto;
  list-style: none;
}

.letter-item {
  --frame-width: 2px;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-1);
  background: none;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}

.letter-item:disabled {
  cursor: default;
  opacity: 0.6;
}

.letter-item:not(:disabled):hover {
  background: rgba(255, 255, 255, 0.1);
}

.letter-item__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.letter-item__date {
  opacity: 0.8;
}

.planet-settings {
  gap: var(--space-3);
}

.planet-settings__header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.planet-settings__header h2 {
  flex: 1;
  min-width: 0;
  text-align: center;
  overflow-wrap: anywhere;
}

.planet-settings__validate {
  flex: none;
  padding: var(--space-1);
}

.planet-settings__row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.planet-settings__row--bio {
  align-items: flex-start;
}

.planet-settings__row .pixel-icon {
  width: 32px;
  height: 32px;
}

.planet-settings__field {
  flex: 1;
  min-width: 0;
  padding: var(--space-1) var(--space-2);
  background: rgba(255, 255, 255, 0.15);
  color: var(--color-text);
  outline: none;
  resize: none;
}

.planet-settings__field::placeholder {
  color: var(--color-text);
  opacity: 0.5;
}

.planet-settings__field:focus {
  background: rgba(255, 255, 255, 0.22);
}

.planet-settings__customize {
  justify-content: center;
  width: 100% !important;
  margin-top: var(--space-2);
  padding: var(--space-2);
  font-size: var(--subtitle-size);
  text-decoration: none;
}

@media (max-width: 599px) {
  .planet-profile {
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .planet-settings {
    gap: var(--space-2);
  }

  .planet-profile__more {
    flex: none;
  }

  .planet-profile__letters {
    overflow-y: visible;
  }
}
</style>
