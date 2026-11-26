<template>
  <div class="app">
    <!-- La scène vit ici, derrière toutes les pages : elle ne se recharge pas à la navigation -->
    <SolarScene class="background" :system="system" :decor="decor" :channels="channels" :messages="messages" :selected-id="selectedId" :followed-letter="followedLetter" :opened-letter="openedLetter" :customizing="customizing" :focus-area="focusArea" :overview="!system" :me-id="meId" :npc="npc" :npc-open="npcOpen" :npc-letters="npcLetters" @select="pick" @open="openChannel" @sun="openSunPanel" @read="openedLetter = $event" @rocket="onRocket" @npc="toggleNpc" />
    <template v-if="route.path !== '/' && route.path !== '/planet' && !planetView">
      <v-btn v-if="xs" class="logout" icon="mdi-logout" color="white" variant="outlined" title="Déconnexion" aria-label="Déconnexion" @click="logout" />
      <v-btn v-else class="logout" color="white" variant="outlined" append-icon="mdi-logout" @click="logout">
        Déconnexion
      </v-btn>
    </template>
    <div class="ui-layer">
      <NuxtPage />
    </div>
    <ChannelPassword v-model="lockedChannel" @entered="goTo" />
    <PlanetProfile v-if="npc" v-model="npcOpen" :planet="npc" :letters="npcLetters" npc :editable="!!isAdmin" @read="readNpcLetter" @write="writeToNpc" @failed="notify" />
    <RocketPanel v-if="npcRocketLetter" :letter="npcRocketLetter" :recipient="npc?.planet_name || 'Zib Zib'" :cancellable="false" @arrived="onNpcArrived" />
    <LetterComposer v-if="npc" v-model="npcComposerOpen" :recipient="npc" :channel-id="0" @sent="onNpcSent" @failed="notify" />
    <LetterViewer v-if="npcReading" v-model="npcReaderOpen" :blocks="npcReading.blocks" :strokes="npcReading.strokes" :images="npcReading.images" :lines="npcReading.lines" :from="npcReading.letter.user_id === npc?.id ? npc?.planet_name || 'Zib Zib' : 'toi'" :removable="npcReading.letter.user_id === npc?.id && !npcReading.letter.rocket_removed" @remove="removeNpcRocket" />
    <v-snackbar v-model="snackbar" :color="snackbarColor" :timeout="3000" location="top">{{ snackbarText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
const { system, messages, selectedId, profileOpen, channelOpen, clickedRocket, decor, channels, followedLetter, openedLetter, customizing, focusArea, meId, npc, npcOpen, isAdmin, npcLetters, npcMailbox } = useScene()
const api = useChatApi()
const route = useRoute()
const { xs } = useDisplay()
const planetView = computed(() => !!system.value?.planets.some((p) => p.id === selectedId.value))

const lockedChannel = ref<Channel | null>(null)

const logout = async () => {
  api.clearToken()
  lockedChannel.value = null
  meId.value = null
  isAdmin.value = null
  npcOpen.value = false
  npcLetters.value = []
  system.value = null
  messages.value = []
  selectedId.value = null
  await navigateTo('/')
}

// Seules les planètes d'un vrai salon se sélectionnent, et pas la sienne
const pick = (id: number) => {
  if (!system.value?.planets.some((p) => p.id === id)) return
  selectedId.value = id
  profileOpen.value = true
  channelOpen.value = false
  clickedRocket.value = null
  npcOpen.value = false
}

// Zib Zib : sa fiche s'ouvre partout, les autres panneaux se ferment ; les admins peuvent la modifier
const toggleNpc = async (open: boolean) => {
  npcOpen.value = open
  if (!open) return
  selectedId.value = null
  profileOpen.value = channelOpen.value = false
  clickedRocket.value = null
  loadNpcLetters()
  if (isAdmin.value === null && api.getToken()) isAdmin.value = !!(await api.getMe().catch(() => null))?.is_admin
}

// Lettres de Zib Zib : chargées à chaque page et toutes les 30 s (la lettre d'anniversaire part à la première visite du jour)
const loadNpcLetters = async () => {
  if (!api.getToken()) return
  const [fresh, mailbox] = await Promise.all([api.getNpcLetters().catch(() => null), api.getNpcMailbox().catch(() => null)])
  if (fresh && JSON.stringify(fresh) !== JSON.stringify(npcLetters.value)) npcLetters.value = fresh
  if (mailbox) npcMailbox.value = mailbox.open
}

const NPC_POLL_MS = 30000
const npcPoll = setInterval(() => { if (!document.hidden) loadNpcLetters() }, NPC_POLL_MS)
onBeforeUnmount(() => clearInterval(npcPoll))

// Écrire à Zib Zib : la fiche laisse la place à l'éditeur, la caméra suit la fusée au départ
const npcComposerOpen = ref(false)
const snackbar = ref(false)
const snackbarText = ref('')
const snackbarColor = ref<string | undefined>()

const notify = (text: string, color = 'red') => {
  snackbarText.value = text
  snackbarColor.value = color
  snackbar.value = true
}

const writeToNpc = () => {
  npcOpen.value = false
  npcComposerOpen.value = true
}

const onNpcSent = async (sent: Message) => {
  notify(`Lettre partie, ${npc.value?.planet_name || 'Zib Zib'} la recevra dans 10 minutes`, 'success')
  await loadNpcLetters()
  followedLetter.value = letterKey(sent)
  clickedRocket.value = letterKey(sent)
}

// Sa fusée vers Zib Zib, cliquée ou tout juste partie : temps restant, sans annulation possible
const npcRocketLetter = computed(() =>
  npcLetters.value.find((m) => letterKey(m) === clickedRocket.value && m.user_id === meId.value && m.in_transit && !m.rocket_removed),
)

const onNpcArrived = () => {
  clickedRocket.value = null
  loadNpcLetters()
}

const npcReading = ref<(Letter & { letter: Message }) | null>(null)
const npcReaderOpen = ref(false)

const readNpcLetter = async (m: Message) => {
  if (!m.content) {
    await loadNpcLetters()
    m = npcLetters.value.find((x) => x.id === m.id) ?? m
  }
  const letter = parseLetter(m.content)
  if (!letter) return
  npcReading.value = { ...letter, letter: m }
  npcReaderOpen.value = true
}

// Fusée de Zib Zib ouverte dans la scène : le nez s'ouvre, puis la lettre s'affiche
const NOSE_OPEN_MS = 700
watch(openedLetter, async (key) => {
  const m = npcLetters.value.find((x) => letterKey(x) === key)
  if (!m) return
  await new Promise((resolve) => setTimeout(resolve, NOSE_OPEN_MS))
  if (openedLetter.value === key) await readNpcLetter(m)
  if (!npcReaderOpen.value && openedLetter.value === key) openedLetter.value = null
})

watch(npcReaderOpen, (open) => {
  if (!open && npcLetters.value.some((x) => letterKey(x) === openedLetter.value)) openedLetter.value = null
})

const removeNpcRocket = async () => {
  const m = npcReading.value?.letter
  if (!m) return
  try {
    await api.removeRocket(m.id)
    npcReaderOpen.value = false
    npcLetters.value = npcLetters.value.map((x) => (x.id === m.id ? { ...x, rocket_removed: true } : x))
    sound.play('delete')
  } catch {
  }
}

const onEscape = (event: KeyboardEvent) => { if (event.key === 'Escape') npcOpen.value = false }
watch(() => route.path, (path) => {
  npcOpen.value = false
  if (path === '/') isAdmin.value = null
  else loadNpcLetters()
})

// Clic sur le soleil du salon : retour à la vue du salon et fiche du salon
const openSunPanel = () => {
  selectedId.value = null
  profileOpen.value = false
  clickedRocket.value = null
  npcOpen.value = false
  channelOpen.value = true
}

// Clic sur une fusée : la fiche de la planète laisse la place à celle de la fusée
const onRocket = (key: string | null) => {
  clickedRocket.value = key
  if (key) profileOpen.value = channelOpen.value = npcOpen.value = false
}

// Salon rejoint : membres à jour dans la scène, puis ouverture du salon
const goTo = async (channel: Channel) => {
  channels.value = (await api.getChannels().catch(() => null)) ?? channels.value
  await navigateTo(`/channels/${channel.id}`)
}

// Clic sur un soleil : rejoindre le salon (sans effet si on y est déjà), mot de passe demandé s'il est protégé ; la page du salon affiche les autres erreurs
const openChannel = async (channel: Channel) => {
  if (!api.getToken()) return
  try {
    await api.enterChannel(channel.name)
  } catch (error: any) {
    if (error?.response?.status === 403) {
      lockedChannel.value = channel
      return
    }
  }
  await goTo(channel)
}

// Sons : musique dès l'arrivée, débloqués au premier geste ; clic sur un bouton, son à part s'il est désactivé
const sound = useSound()
const onPress = (event: PointerEvent) => {
  sound.unlock()
  const target = event.target as Element | null
  if (target?.closest?.('button:disabled, .v-btn--disabled, [aria-disabled="true"]')) sound.play('disabled')
  else if (target?.closest?.('button, .v-btn, .pixel-btn, .swatch-btn')) sound.play('click', 0.25)
}
const onKey = () => sound.unlock()
onMounted(() => {
  sound.startMusic()
  window.addEventListener('pointerdown', onPress, true)
  window.addEventListener('keydown', onKey, true)
  window.addEventListener('keydown', onEscape)
})
onBeforeUnmount(() => {
  window.removeEventListener('pointerdown', onPress, true)
  window.removeEventListener('keydown', onKey, true)
  window.removeEventListener('keydown', onEscape)
})

// Décor avant tout : routes publiques, une erreur ici ne bloque rien
onMounted(async () => {
  try {
    const users = (await api.getUsers()) ?? []
    decor.value = buildDecor(users)
    npc.value = npcOf(users)
  } catch {
    decor.value = buildDecor([])
  }
  try {
    channels.value = (await api.getChannels()) ?? []
  } catch {
    channels.value = []
  }
  loadNpcLetters()
})
</script>

<style>
html, body { margin: 0; overflow: hidden; background: #05060f; }
.app, .background { position: fixed; inset: 0; }
.background { z-index: 0; }
.v-btn.logout { position: fixed; top: 16px; right: 16px; z-index: 10; }
.ui-layer { position: fixed; inset: 0; z-index: 1; overflow-y: auto; pointer-events: none; }

.ui-layer .v-card, .ui-layer .v-btn, .ui-layer .v-chip, .ui-layer .v-list, .ui-layer .v-field, .ui-layer .v-tabs { pointer-events: auto; }
.ui-layer .v-card, .ui-layer .v-list { background: var(--Klaus) !important; backdrop-filter: blur(8px); color: #fff; }
.ui-layer h1, .ui-layer h2, .ui-layer p { text-shadow: 0 1px 6px #000; color: #fff; }
</style>
