<template>
  <div>
  <InfoMenu :title="selected?.username ?? system?.channel.name" :back-label="composerOpen ? undefined : selected ? 'Retour au salon' : 'Mes salons'" :back-only="!!selected" @back="back">
  <template #actions>
    <v-btn icon="mdi-email-edit-outline" color="white" variant="outlined" title="Écrire à…" aria-label="Écrire à…" @click="searchOpen = !searchOpen" />
    <v-autocomplete
      v-if="searchOpen"
      :model-value="null"
      :items="recipients"
      item-title="username"
      item-value="id"
      placeholder="Nom du destinataire"
      class="recipient-search"
      density="compact"
      variant="solo"
      autofocus
      auto-select-first
      hide-details
      @update:model-value="writeTo"
      @keydown.esc="searchOpen = false"
    />
  </template>
  </InfoMenu>
  <RocketPanel v-if="rocketLetter" :letter="rocketLetter" :recipient="rocketLetter.recipient ?? ''" :cancelling="cancelling" @cancel="cancelLetter" @arrived="onArrived" />
  <ChannelProfile v-if="system" v-model="channelOpen" :channel="system.channel" :planets="system.planets" :editable="canEditChannel" @select="pickFromChannel" @saved="onChannelSaved" @failed="notify" />
  <PlanetProfile v-if="selected" v-model="profileOpen" :planet="selected" :letters="messages" @write="writeSelected" @read="read" />
  <LetterComposer v-if="selected" v-model="composerOpen" :recipient="selected" :channel-id="channelId" @sent="onSent" @failed="notify" />
  <LetterViewer v-if="reading" v-model="readerOpen" :blocks="reading.blocks" :strokes="reading.strokes" :from="reading.from" :removable="canRemove" @remove="removeRocket" />
  <v-snackbar v-model="snackbar" :color="snackbarColor" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </div>
</template>

<style scoped>

.recipient-search {
  width: 260px;
  flex-shrink: 0;
}

@media (max-width: 599px) {
  .recipient-search {
    position: fixed;
    top: 72px;
    left: 8px;
    right: 8px;
    width: auto;
  }
}

</style>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const api = useChatApi()
const channelId = Number(useRoute().params.id)

// La scène plein écran (app.vue) lit cet état partagé
const { system, messages, selectedId: recipientId, profileOpen, channelOpen, clickedRocket, followedLetter, openedLetter } = useScene()
const me = ref<UserMe | null>(null)
const snackbar = ref(false)
const snackbarText = ref('')
const snackbarColor = ref('red')
const composerOpen = ref(false)
const searchOpen = ref(false)
const readerOpen = ref(false)
const reading = ref<{ blocks: LetterBlock[], strokes: Stroke[], from: string, letter: Message } | null>(null)

const selected = computed(() => system.value?.planets.find((p) => p.id === recipientId.value) ?? null)

const notify = (text: string, color = 'red') => {
  snackbarText.value = text
  snackbarColor.value = color
  snackbar.value = true
}

const formatDate = (iso: string) => (iso ? new Date(iso).toLocaleString() : '')

const load = async () => {
  try {
    ;[me.value, system.value, messages.value] = await Promise.all([
      me.value ?? api.getMe(),
      api.getSystem(channelId),
      api.getMessages(channelId),
    ])
  } catch (error) {
    notify(apiErrorMessage(error))
  }
}

const onSent = async (sent: Message) => {
  notify(sent.in_transit ? `Lettre partie, arrivée ${formatDate(sent.deliver_at)}` : 'Lettre envoyée', 'success')
  await load()
  followedLetter.value = letterKey(sent)
}

// Écrire à quelqu'un du salon en cherchant son nom
const recipients = computed(() => system.value?.planets.filter((p) => !p.me) ?? [])

const writeTo = (id: number | null) => {
  if (id == null) return
  recipientId.value = id
  composerOpen.value = true
  searchOpen.value = false
}

// Depuis la modale de la planète : on la ferme et on ouvre l'éditeur
const writeSelected = () => {
  profileOpen.value = false
  composerOpen.value = true
}

// Fiche du salon : bio modifiable par les admins et le propriétaire, clic sur une planète de la liste
const canEditChannel = computed(() => !!me.value && (!!me.value.is_admin || system.value?.channel.owner_id === me.value.id))

const pickFromChannel = (id: number) => {
  channelOpen.value = false
  recipientId.value = id
  profileOpen.value = true
}

const onChannelSaved = (channel: Channel) => {
  if (system.value) system.value = { ...system.value, channel: { ...system.value.channel, bio: channel.bio } }
  notify('Bio du salon enregistrée', 'success')
}

// Retour à l'étape précédente : de la planète au salon, du salon à la liste
const back = () => {
  if (!selected.value) return navigateTo('/channels')
  recipientId.value = null
}

// Sans planète sélectionnée (retour ou clic sur le soleil), l'éditeur se referme
watch(recipientId, (id) => {
  if (id != null) return
  composerOpen.value = false
  profileOpen.value = false
})

const read = (m: Message) => {
  const letter = parseLetter(m.content)
  reading.value = { blocks: letter?.blocks ?? [], strokes: letter?.strokes ?? [], from: m.user, letter: m }
  readerOpen.value = true
}

// Fusée posée ouverte dans la scène : le nez s'ouvre, puis la lettre s'affiche ; la refermer referme le nez
const NOSE_OPEN_MS = 700
const letterFor = (key: string) => messages.value.find((m) => letterKey(m) === key)

watch(openedLetter, async (key) => {
  if (!key) return
  const wait = new Promise((resolve) => setTimeout(resolve, NOSE_OPEN_MS))
  if (!letterFor(key)?.content) {
    try {
      messages.value = await api.getMessages(channelId)
    } catch (error) {
      notify(apiErrorMessage(error))
    }
  }
  await wait
  const m = letterFor(key)
  if (m?.content && openedLetter.value === key) read(m)
  else openedLetter.value = null
})

watch(readerOpen, (open) => { if (!open) openedLetter.value = null })

// Fusée cliquée : son panneau seulement pour sa propre lettre encore en route
const cancelling = ref(false)
const rocketLetter = computed(() =>
  messages.value.find((m) => letterKey(m) === clickedRocket.value && m.user_id === me.value?.id && m.in_transit && !m.rocket_removed),
)

// Annuler : la fusée explose comme une fusée retirée, puis le salon est rechargé (distance éventuellement rendue)
const cancelLetter = async () => {
  const m = rocketLetter.value
  if (!m) return
  cancelling.value = true
  try {
    await api.cancelLetter(m.id)
    clickedRocket.value = null
    messages.value = messages.value.map((x) => (x.id === m.id ? { ...x, rocket_removed: true } : x))
    notify('Lettre annulée', 'success')
    await load()
  } catch (error) {
    notify(apiErrorMessage(error))
  } finally {
    cancelling.value = false
  }
}

const onArrived = () => {
  clickedRocket.value = null
  load()
}

// Retirer la fusée d'une lettre reçue : la lettre reste dans la liste, la fusée disparaît de la planète (pouf)
const canRemove = computed(() => {
  const m = reading.value?.letter
  return !!m && m.recipient_id === me.value?.id && !m.in_transit && !m.rocket_removed
})

const removeRocket = async () => {
  const m = reading.value?.letter
  if (!m) return
  try {
    await api.removeRocket(m.id)
    readerOpen.value = false
    messages.value = messages.value.map((x) => (x.id === m.id ? { ...x, rocket_removed: true } : x))
  } catch (error) {
    notify(apiErrorMessage(error))
  }
}

onMounted(load)

// En quittant le salon, la scène retrouve son décor
onBeforeUnmount(() => {
  system.value = null
  messages.value = []
  recipientId.value = null
  profileOpen.value = false
  channelOpen.value = false
  clickedRocket.value = null
  followedLetter.value = null
  openedLetter.value = null
})
</script>
