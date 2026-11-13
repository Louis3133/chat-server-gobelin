<template>
  <div>
  <InfoMenu title="Mes salons" icon="/icons/actions/tools.svg">
  <template v-if="me?.is_admin" #default>
  <div class="salons pixel-frame pixel-scroll">
    <div class="salons__header">
      <h2 class="text-heading">Bonjour {{ me?.username }}</h2>
      <button class="salons__link" type="button" @click="logout">Déconnexion</button>
    </div>
    <NuxtLink class="pixel-btn salons__planet" to="/planet">
      Modifier ma planète
      <img class="pixel-icon" src="/icons/actions/earth.svg" alt="">
    </NuxtLink>

    <hr class="salons__separator">
    <h3 class="text-subtitle-medium">Mes salons</h3>
    <ul v-if="me?.channels.length" class="salons__list">
      <li v-for="c in me.channels" :key="c.id">
        <NuxtLink class="salon-item pixel-frame" :to="`/channels/${c.id}`">
          <span class="salon-item__text">
            <span class="text-body">{{ c.name }}</span>
            <span class="text-body salon-item__sub">{{ planetCount(c) }}</span>
          </span>
        </NuxtLink>
      </li>
    </ul>
    <p v-else class="text-body">Tu n'es dans aucun salon.</p>

    <hr class="salons__separator">
    <h3 class="text-subtitle-medium">Rejoindre un salon</h3>
    <form class="salons__row" @submit.prevent="join">
      <label class="field">
        <span v-if="toJoin" class="field-label">Salon</span>
        <select v-model="toJoin" :disabled="!joinable.length">
          <option :value="null" disabled>{{ joinable.length ? 'Salon' : 'Aucun salon à rejoindre' }}</option>
          <option v-for="name in joinable" :key="name" :value="name">{{ name }}</option>
        </select>
      </label>
      <button class="pixel-btn" type="submit" :disabled="!toJoin || busy">Rejoindre</button>
    </form>

    <h3 class="text-subtitle-medium">Créer un salon</h3>
    <form class="salons__row" @submit.prevent="create">
      <label class="field">
        <span v-if="newName" class="field-label">Nom du salon</span>
        <input v-model="newName" placeholder="Nom du salon" :readonly="busy" autocomplete="off" spellcheck="false">
      </label>
      <button class="pixel-btn" type="submit" :disabled="!newName.trim() || busy">Créer</button>
    </form>

    <template v-if="me">
      <hr class="salons__separator">
      <h3 class="text-subtitle-medium">Administration · Salons</h3>
      <ul class="salons__list">
        <li v-for="c in deletableChannels" :key="c.id" class="salon-item pixel-frame">
          <span class="salon-item__text">
            <span class="text-body">{{ c.name }}</span>
            <span class="text-body salon-item__sub">{{ planetCount(c) }}</span>
          </span>
          <button class="salon-item__delete" type="button" :disabled="busy" :title="`Supprimer ${c.name}`" :aria-label="`Supprimer ${c.name}`" @click="removeChannel(c)">
            <img class="pixel-icon" src="/icons/actions/close.svg" alt="">
          </button>
        </li>
      </ul>
      <h3 class="text-subtitle-medium">Administration · Utilisateurs</h3>
      <ul class="salons__list">
        <li v-for="u in deletableUsers" :key="u.id" class="salon-item pixel-frame">
          <PlanetAvatar :planet="planetOf(u.id)" />
          <span class="salon-item__text">
            <span class="text-body">{{ u.planet_name || `Planète de ${u.username}` }}</span>
            <span class="text-body salon-item__sub">{{ u.username }}</span>
          </span>
          <button class="salon-item__delete" type="button" :disabled="busy" :title="`Supprimer ${u.username}`" :aria-label="`Supprimer ${u.username}`" @click="removeUser(u)">
            <img class="pixel-icon" src="/icons/actions/close.svg" alt="">
          </button>
        </li>
      </ul>
    </template>
  </div>
  </template>
  </InfoMenu>
  <v-snackbar v-model="snackbar" color="red" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </div>
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

const planetCount = (c: Channel) => {
  const n = allChannels.value.find((x) => x.id === c.id)?.member_ids?.length ?? 0
  return `${n} ${n > 1 ? 'planètes' : 'planète'}`
}
const userPlanets = computed(() => buildDecor(allUsers.value).planets)
const planetOf = (id: number) => userPlanets.value.find((p) => p.id === id)

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

<style scoped>
.salons {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 100%;
  padding: var(--space-3);
  overflow-y: auto;
}

.salons h2,
.salons h3,
.salons p {
  margin: 0;
}

.salons__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  overflow-wrap: anywhere;
}

.salons__link {
  flex: none;
  color: var(--color-text);
  font-size: var(--body-size);
  text-decoration: underline;
  transition: color 0.15s;
}

.salons__link:hover {
  color: #d3fbb2;
}

.salons__planet {
  align-self: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  text-decoration: none;
}

.salons__separator {
  margin: var(--space-1) 0;
  border: none;
  border-top: var(--stroke-width) solid var(--color-text);
}

.salons__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.salon-item {
  --frame-width: 2px;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  min-height: 56px;
  padding: var(--space-1);
  background: none;
  color: var(--color-text);
  text-decoration: none;
}

a.salon-item:hover {
  background: rgba(255, 255, 255, 0.1);
}

.salon-item__text {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  overflow-wrap: anywhere;
}

.salon-item__sub {
  opacity: 0.8;
}

.salon-item__delete {
  flex: none;
  padding: var(--space-1);
  transition: opacity 0.15s;
}

.salon-item__delete:hover {
  background: rgba(255, 255, 255, 0.1);
}

.salon-item__delete:disabled {
  opacity: 0.5;
  cursor: default;
}

.salons__row {
  display: flex;
  align-items: stretch;
  gap: var(--space-2);
}

.salons__row .pixel-btn {
  flex: none;
  padding: var(--space-1) var(--space-2);
}

.field {
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
  min-height: 52px;
  padding: 4px var(--space-1);
  background: #fff;
  color: #000;
}

.field-label {
  font-size: 11px;
  opacity: 0.7;
}

.field input,
.field select {
  width: 100%;
  color: #000;
  font-size: var(--body-size);
  outline: none;
}

.field input::placeholder {
  color: #737373;
  opacity: 1;
}
</style>
