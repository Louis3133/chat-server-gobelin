<template>
  <InfoMenu title="Mes salons">
  <v-card class="py-8">
    <div class="d-flex align-center mb-4">
      <h1 class="text-h5 flex-grow-1">Bonjour {{ me?.username }}</h1>
      <v-btn variant="text" prepend-icon="mdi-logout" @click="logout">Déconnexion</v-btn>
    </div>

    <h2 class="text-h6 mb-2">Mes salons</h2>
    <v-list v-if="me?.channels.length" class="mb-6">
      <v-list-item v-for="c in me.channels" :key="c.id" :title="c.name" :to="`/channels/${c.id}`" />
    </v-list>
    <p v-else class="mb-6">Tu n'es dans aucun salon.</p>

    <h2 class="text-h6 mb-2">Rejoindre un salon</h2>
    <v-form class="d-flex ga-2 mb-6" @submit.prevent="join">
      <v-select v-model="toJoin" :items="joinable" label="Salon" hide-details />
      <v-btn :disabled="!toJoin" :loading="busy" type="submit" color="primary">Rejoindre</v-btn>
    </v-form>

    <h2 class="text-h6 mb-2">Créer un salon (admin)</h2>
    <v-form class="d-flex ga-2" @submit.prevent="create">
      <v-text-field v-model="newName" label="Nom du salon" hide-details />
      <v-btn :disabled="!newName" :loading="busy" type="submit" color="primary">Créer</v-btn>
    </v-form>

    <template v-if="me?.is_admin">
      <h2 class="text-h6 mt-6 mb-2">Administration</h2>
      <h3 class="text-subtitle-1">Salons</h3>
      <v-list density="compact" class="mb-4">
        <v-list-item v-for="c in deletableChannels" :key="c.id" :title="c.name">
          <template #append>
            <v-btn icon="mdi-delete" size="small" variant="text" color="red" :disabled="busy" @click="removeChannel(c)" />
          </template>
        </v-list-item>
      </v-list>
      <h3 class="text-subtitle-1">Utilisateurs</h3>
      <v-list density="compact">
        <v-list-item v-for="u in deletableUsers" :key="u.id" :title="u.username">
          <template #append>
            <v-btn icon="mdi-delete" size="small" variant="text" color="red" :disabled="busy" @click="removeUser(u)" />
          </template>
        </v-list-item>
      </v-list>
    </template>

    <v-snackbar v-model="snackbar" color="red" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </v-card>
  </InfoMenu>
</template>

<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

const api = useChatApi()
const scene = useScene()

const me = ref<UserMe | null>(null)
const allChannels = ref<Channel[]>([])
const allUsers = ref<PublicUser[]>([])
const toJoin = ref<string | null>(null)
const newName = ref('')
const busy = ref(false)
const snackbar = ref(false)
const snackbarText = ref('')

const joinable = computed(() => {
  const mine = new Set(me.value?.channels.map((c) => c.id))
  return allChannels.value.filter((c) => !mine.has(c.id)).map((c) => c.name)
})

const logout = () => {
  api.clearToken()
  navigateTo('/')
}

// Administration
const deletableChannels = computed(() => allChannels.value.filter((c) => c.name !== 'general'))
const deletableUsers = computed(() =>
  allUsers.value.filter((u) => u.username !== 'internal-clock' && u.id !== me.value?.id),
)

const fail = (error: unknown) => {
  snackbarText.value = apiErrorMessage(error)
  snackbar.value = true
}

const refresh = async () => {
  me.value = await api.getMe()
  allChannels.value = (await api.getChannels()) ?? []
  scene.channels.value = allChannels.value
  if (me.value.is_admin) allUsers.value = (await api.getUsers()) ?? []
  toJoin.value = null
}

const run = async (action: () => Promise<unknown>) => {
  busy.value = true
  try {
    await action()
    await refresh()
  } catch (error) {
    fail(error)
  } finally {
    busy.value = false
  }
}

const join = () => run(() => api.enterChannel(toJoin.value!))
const create = () => run(async () => {
  await api.createChannel(newName.value)
  newName.value = ''
})

const removeChannel = (c: Channel) => {
  if (!confirm(`Supprimer le salon « ${c.name} » et toutes ses lettres ?`)) return
  run(() => api.deleteChannel(c.id))
}
const removeUser = (u: PublicUser) => {
  if (!confirm(`Supprimer « ${u.username} » et toutes ses lettres ?`)) return
  run(() => api.deleteUser(u.id))
}

onMounted(() => refresh().catch(fail))
</script>
