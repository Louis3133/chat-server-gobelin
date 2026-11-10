<template>
  <div class="app">
    <!-- La scène vit ici, derrière toutes les pages : elle ne se recharge pas à la navigation -->
    <SolarScene class="background" :system="system" :decor="decor" :channels="channels" :messages="messages" :selected-id="selectedId" :overview="!system" @select="pick" @open="openChannel" />
    <template v-if="route.path !== '/'">
      <v-btn v-if="xs" class="logout" icon="mdi-logout" color="white" variant="outlined" title="Déconnexion" aria-label="Déconnexion" @click="logout" />
      <v-btn v-else class="logout" color="white" variant="outlined" prepend-icon="mdi-logout" @click="logout">
        Déconnexion
      </v-btn>
    </template>
    <div class="ui-layer">
      <NuxtPage />
    </div>
  </div>
</template>

<script setup lang="ts">
const { system, messages, selectedId, decor, channels } = useScene()
const api = useChatApi()
const route = useRoute()
const { xs } = useDisplay()

const logout = async () => {
  api.clearToken()
  system.value = null
  messages.value = []
  selectedId.value = null
  await navigateTo('/')
}

// Seules les planètes d'un vrai salon se sélectionnent, et pas la sienne
const pick = (id: number) => {
  if (system.value?.planets.some((p) => p.id === id)) selectedId.value = id
}

// Clic sur un soleil : rejoindre le salon (sans effet si on y est déjà), puis l'ouvrir ; la page du salon affiche l'erreur s'il y en a une
const openChannel = async (channel: Channel) => {
  if (!api.getToken()) return
  await api.enterChannel(channel.name).catch(() => {})
  await navigateTo(`/channels/${channel.id}`)
}

// Décor avant tout : routes publiques, une erreur ici ne bloque rien
onMounted(async () => {
  try {
    decor.value = buildDecor((await api.getUsers()) ?? [])
  } catch {
    decor.value = buildDecor([])
  }
  try {
    channels.value = (await api.getChannels()) ?? []
  } catch {
    channels.value = []
  }
})
</script>

<style>
html, body { margin: 0; overflow: hidden; }
.app, .background { position: fixed; inset: 0; }
.background { z-index: 0; }
.logout { position: fixed; top: 16px; right: 16px; z-index: 10; background: rgba(15, 18, 35, 0.78) !important; }
.ui-layer { position: fixed; inset: 0; z-index: 1; overflow-y: auto; pointer-events: none; }

.ui-layer .v-card, .ui-layer .v-btn, .ui-layer .v-chip, .ui-layer .v-list, .ui-layer .v-field, .ui-layer .v-tabs { pointer-events: auto; }
.ui-layer .v-card, .ui-layer .v-list { background: rgba(15, 18, 35, 0.78) !important; backdrop-filter: blur(8px); color: #fff; }
.ui-layer h1, .ui-layer h2, .ui-layer p { text-shadow: 0 1px 6px #000; color: #fff; }
</style>
