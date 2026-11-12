<template>
  <Teleport to="body">
    <div v-if="open" ref="panel" class="planet-profile scene-panel pixel-frame">
      <div class="planet-profile__header">
        <div class="planet-profile__titles">
          <h2 class="text-heading">{{ planet.planet_name || `Planète de ${planet.username}` }}</h2>
          <p class="text-subtitle-medium">
            {{ planet.username }}<template v-if="planet.pronouns"> ({{ planet.pronouns }})</template>
          </p>
        </div>
        <div class="planet-profile__actions">
          <v-btn icon="mdi-information-outline" :title="expanded ? 'Moins d\'infos' : 'Plus d\'infos'" :aria-label="expanded ? 'Moins d\'infos' : 'Plus d\'infos'" :aria-expanded="expanded" @click="toggle" />
          <v-btn v-if="!planet.me" title="Écrire une lettre" aria-label="Écrire une lettre" @click="emit('write')"><img class="pixel-icon" src="/icons/actions/send.svg" alt=""></v-btn>
        </div>
      </div>
      <p v-if="planet.bio" class="planet-profile__bio text-body" :class="{ 'planet-profile__bio--full': shown }">{{ planet.bio }}</p>

      <div v-if="shown && !planet.me" ref="more" class="planet-profile__more">
        <hr class="planet-profile__separator">
        <h3 class="text-subtitle-medium">Vos lettres</h3>
        <p v-if="!exchanged.length" class="text-body">Aucune lettre échangée pour l'instant.</p>
        <ul v-else class="planet-profile__letters pixel-scroll">
          <li v-for="m in exchanged" :key="m.id">
            <button class="letter-item pixel-frame" type="button" :disabled="!readable(m)" @click="emit('read', m)">
              <span class="letter-item__planet" :style="{ background: colorOf(m.user_id) }" />
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
const emit = defineEmits<{ write: [], read: [letter: Message] }>()

const open = defineModel<boolean>({ default: false })
const expanded = ref(false)
const shown = ref(false)
const panel = ref<HTMLElement>()
const more = ref<HTMLElement>()
const { system } = useScene()
let resize: Animation | undefined

watch(open, (value) => {
  if (value) return
  resize?.cancel()
  expanded.value = shown.value = false
})

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

const colorOf = (id: number) => system.value?.planets.find((p) => p.id === id)?.color ?? '#d9d9d9'

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

.planet-profile__header {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
}

.planet-profile__titles {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
  overflow-wrap: anywhere;
}

.planet-profile h2,
.planet-profile h3,
.planet-profile p {
  margin: 0;
}

.planet-profile__actions {
  display: flex;
  gap: var(--space-1);
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

.letter-item__planet {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 50%;
}

.letter-item__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.letter-item__date {
  opacity: 0.8;
}
</style>
