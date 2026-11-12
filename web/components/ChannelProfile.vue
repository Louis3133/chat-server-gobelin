<template>
  <Teleport to="body">
    <div v-if="open" ref="panel" class="channel-profile scene-panel pixel-frame">
      <div>
        <h2 class="text-heading">{{ title }}</h2>
        <p class="text-subtitle-medium">{{ members.length }} {{ members.length > 1 ? 'planètes' : 'planète' }}</p>
      </div>

      <template v-if="editing">
        <label class="bio-field">
          <span v-if="draft" class="bio-field__label">Bio du salon</span>
          <textarea v-model="draft" placeholder="Bio du salon" rows="4" :readonly="saving" />
        </label>
        <div class="bio-help text-body">
          <span :class="{ invalid: tooLong }">{{ draft.length }}/{{ BIO_MAX }}</span>
        </div>
        <div class="channel-profile__edit-actions">
          <button class="pixel-btn" type="button" :disabled="saving" @click="editing = false">Annuler</button>
          <button class="pixel-btn" type="button" :disabled="saving || tooLong" @click="save">{{ saving ? '…' : 'Enregistrer' }}</button>
        </div>
      </template>
      <p v-else-if="channel.bio" class="channel-profile__bio text-body" :class="{ 'channel-profile__bio--full': shown }">{{ channel.bio }}</p>

      <div v-if="!editing" class="channel-profile__ctas">
        <button class="channel-profile__link" type="button" :aria-expanded="expanded" @click="toggle">
          {{ expanded ? 'Masquer le détail' : 'Voir le détail' }}
        </button>
        <button v-if="editable" class="channel-profile__link" type="button" @click="startEditing">Modifier la bio</button>
      </div>

      <div v-if="shown && !editing" ref="more" class="channel-profile__more">
        <hr class="channel-profile__separator">
        <h3 class="text-subtitle-medium">Les planètes</h3>
        <ul class="channel-profile__planets pixel-scroll">
          <li v-for="p in members" :key="p.id">
            <button class="planet-item pixel-frame" type="button" @click="emit('select', p.id)">
              <PlanetAvatar :planet="p" />
              <span class="planet-item__text">
                <span class="text-body">{{ p.planet_name || `Planète de ${p.username}` }}</span>
                <span class="text-body planet-item__username">{{ p.me ? `${p.username} (moi)` : p.username }}</span>
              </span>
            </button>
          </li>
        </ul>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const props = defineProps<{ channel: Channel, planets: Planet[], editable: boolean }>()
const emit = defineEmits<{ select: [id: number], saved: [channel: Channel], failed: [message: string] }>()
const open = defineModel<boolean>({ default: false })
const api = useChatApi()

const BIO_MAX = 500
const expanded = ref(false)
const shown = ref(false)
const panel = ref<HTMLElement>()
const more = ref<HTMLElement>()
let resize: Animation | undefined
const editing = ref(false)
const saving = ref(false)
const draft = ref('')
const tooLong = computed(() => draft.value.length > BIO_MAX)

const title = computed(() => {
  const name = props.channel.name
  return name === 'general' ? 'Général' : name.charAt(0).toUpperCase() + name.slice(1)
})

const members = computed(() => props.planets.filter((p) => p.username !== 'internal-clock'))

watch(open, (value) => {
  if (value) return
  resize?.cancel()
  expanded.value = shown.value = editing.value = false
})

// Ouverture et fermeture animées du détail, comme la fiche d'une planète
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

const startEditing = () => {
  draft.value = props.channel.bio ?? ''
  editing.value = true
}

const save = async () => {
  saving.value = true
  try {
    emit('saved', await api.updateChannelBio(props.channel.id, draft.value))
    editing.value = false
  } catch (error) {
    emit('failed', apiErrorMessage(error))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.channel-profile {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-height: 0;
  padding: var(--space-3);
}

.channel-profile > * {
  flex: none;
}

.channel-profile h2,
.channel-profile h3,
.channel-profile p {
  margin: 0;
}

.channel-profile h2 {
  margin-bottom: var(--space-1);
}

.channel-profile__bio {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  overflow: hidden;
  white-space: pre-line;
}

.channel-profile__bio--full {
  display: block;
}

.bio-field {
  display: flex;
  flex-direction: column;
  padding: 4px var(--space-1);
  background: #fff;
  color: #000;
}

.bio-field__label {
  font-size: 11px;
  opacity: 0.7;
}

.bio-field textarea {
  resize: none;
  color: #000;
  outline: none;
}

.bio-field textarea::placeholder {
  color: #737373;
  opacity: 1;
}

.bio-help {
  display: flex;
  justify-content: flex-end;
  margin-top: calc(var(--space-2) * -1);
  opacity: 0.8;
}

.bio-help .invalid {
  color: #ff8a80;
}

.channel-profile__edit-actions {
  display: flex;
  justify-content: center;
  gap: var(--space-2);
}

.channel-profile__edit-actions .pixel-btn {
  padding: var(--space-1) var(--space-2);
}

.channel-profile__edit-actions .pixel-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.channel-profile__more {
  display: flex;
  flex: 0 1 auto;
  flex-direction: column;
  gap: var(--space-2);
  min-height: 0;
}

.channel-profile__more > * {
  flex: none;
}

.channel-profile__separator {
  margin: var(--space-1) 0;
  border: none;
  border-top: var(--stroke-width) solid var(--color-text);
}

.channel-profile__planets {
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

.planet-item {
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

.planet-item:hover {
  background: rgba(255, 255, 255, 0.1);
}

.planet-item__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow-wrap: anywhere;
}

.planet-item__username {
  opacity: 0.8;
}

.channel-profile__ctas {
  display: flex;
  justify-content: center;
  gap: var(--space-3);
}

.channel-profile__link {
  color: var(--color-text);
  font-size: var(--body-size);
  text-decoration: underline;
  transition: color 0.15s;
}

.channel-profile__link:hover {
  color: #d3fbb2;
}
</style>
