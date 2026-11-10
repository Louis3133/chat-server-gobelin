<template>
  <InfoMenu :title="selected?.username ?? system?.channel.name" :back-label="selected ? 'Retour au salon' : 'Mes salons'" @back="back">
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
  <v-card class="infos-panel px-4 py-8 bg-white">
    <h1 v-if="!selected" class="text-h5 mb-4">{{ system?.channel.name ?? 'Salon' }}</h1>
    <h1 v-if="selected" class="text-h5 mb-4">{{ selected.username }}</h1>

    <p v-if="!selected" class="text-caption mb-6">Clique sur une planète pour lui écrire et voir votre conversation. Les points jaunes sont des lettres en route.</p>

    <h2 class="text-h6 mb-2">Écrire</h2>
    <LetterComposer v-if="selected" v-model="composerOpen" :recipient="selected" :channel-id="channelId" @sent="onSent" @failed="notify" />
    <LetterViewer v-if="reading" v-model="readerOpen" :blocks="reading.blocks" :from="reading.from" />

    <div class="d-flex align-center mb-2">
      <h2 class="text-h6 flex-grow-1">Boîte de lettres</h2>
      <v-btn variant="text" icon="mdi-refresh" :loading="loading" @click="load" />
    </div>

    <v-btn @click="composerOpen = true" v-if="selected">Ecrire</v-btn>
    <p v-if="!loading && !shown.length">Aucun message pour l'instant.</p>
    <v-card v-for="(m, i) in shown" :key="i" class="mb-3 pa-3" :variant="m.in_transit ? 'tonal' : 'elevated'">
      <div class="text-caption">
        <strong>{{ m.user }}</strong>
        <template v-if="m.recipient_id"> → {{ m.recipient ?? 'moi' }}</template>
        <template v-else> → tout le salon</template>
        · envoyé {{ formatDate(m.creation_date) }}
      </div>
      <p v-if="m.in_transit && m.recipient_id === me?.id" class="font-italic">
        Une lettre est en route, arrivée {{ formatDate(m.deliver_at) }}.
      </p>
      <template v-else>
        <v-btn v-if="parseLetter(m.content)" class="mt-2" size="small" prepend-icon="mdi-email-open-outline" @click="read(m)">
          {{ m.user_id === me?.id ? 'Revoir ma lettre' : 'Lire la lettre' }}
        </v-btn>
        <p v-else style="white-space: pre-wrap">{{ m.content }}</p>
        <div v-if="m.in_transit" class="text-caption">En route, arrivée {{ formatDate(m.deliver_at) }}</div>
      </template>
    </v-card>


    <v-snackbar v-model="snackbar" :color="snackbarColor" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </v-card>
  </InfoMenu>
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

  .infos-panel {
    border-radius: 16px;
  }
}

.infos-panel {
  border-radius: 24px;
}

</style>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const api = useChatApi()
const channelId = Number(useRoute().params.id)

// La scène plein écran (app.vue) lit cet état partagé
const { system, messages, selectedId: recipientId } = useScene()
const me = ref<UserMe | null>(null)
const loading = ref(true)
const snackbar = ref(false)
const snackbarText = ref('')
const snackbarColor = ref('red')
const composerOpen = ref(false)
const searchOpen = ref(false)
const readerOpen = ref(false)
const reading = ref<{ blocks: LetterBlock[], from: string } | null>(null)

const selected = computed(() => system.value?.planets.find((p) => p.id === recipientId.value) ?? null)

// Avec une planète sélectionnée : seulement les lettres échangées avec elle
const shown = computed(() =>
  selected.value
    ? messages.value.filter((m) => m.user_id === selected.value!.id || m.recipient_id === selected.value!.id)
    : messages.value,
)

const notify = (text: string, color = 'red') => {
  snackbarText.value = text
  snackbarColor.value = color
  snackbar.value = true
}

const formatDate = (iso: string) => (iso ? new Date(iso).toLocaleString() : '')

const formatDelay = (seconds: number) => {
  if (seconds < 60) return `${Math.round(seconds)} s`
  if (seconds < 3600) return `${Math.round(seconds / 60)} min`
  return `${(seconds / 3600).toFixed(1)} h`
}

const load = async () => {
  loading.value = true
  try {
    ;[me.value, system.value, messages.value] = await Promise.all([
      me.value ?? api.getMe(),
      api.getSystem(channelId),
      api.getMessages(channelId),
    ])
  } catch (error) {
    notify(apiErrorMessage(error))
  } finally {
    loading.value = false
  }
}

const onSent = async (sent: Message) => {
  notify(sent.in_transit ? `Lettre partie, arrivée ${formatDate(sent.deliver_at)}` : 'Lettre envoyée', 'success')
  await load()
}

// Écrire à quelqu'un du salon en cherchant son nom
const recipients = computed(() => system.value?.planets.filter((p) => !p.me) ?? [])

const writeTo = (id: number | null) => {
  if (id == null) return
  recipientId.value = id
  composerOpen.value = true
  searchOpen.value = false
}

// Retour à l'étape précédente : de la planète au salon, du salon à la liste
const back = () => {
  if (!selected.value) return navigateTo('/channels')
  recipientId.value = null
  composerOpen.value = false
}

const read = (m: Message) => {
  reading.value = { blocks: parseLetter(m.content) ?? [], from: m.user }
  readerOpen.value = true
}

onMounted(load)

// En quittant le salon, la scène retrouve son décor
onBeforeUnmount(() => {
  system.value = null
  messages.value = []
  recipientId.value = null
})
</script>
