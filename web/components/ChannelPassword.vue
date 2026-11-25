<template>
  <Teleport to="body">
    <div v-if="channel" class="channel-password">
      <form class="channel-password__panel pixel-frame" @submit.prevent="submit">
        <h1 class="text-heading">{{ channelLabel(channel) }}</h1>
        <p class="text-body">Ce salon est protégé : entre son mot de passe une seule fois pour le rejoindre.</p>
        <label class="field">
          <span v-if="password" class="field-label">Mot de passe du salon</span>
          <input v-model="password" type="password" placeholder="Mot de passe du salon" :readonly="busy" autocomplete="off" autofocus>
        </label>
        <p v-if="error" class="channel-password__error text-body">{{ error }}</p>
        <button class="pixel-btn wide" type="submit" :disabled="busy || !password">{{ busy ? '…' : 'Entrer' }}</button>
        <button class="switch" type="button" :disabled="busy" @click="close">Annuler</button>
      </form>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const channel = defineModel<Channel | null>({ default: null })
const emit = defineEmits<{ entered: [channel: Channel] }>()
const api = useChatApi()

const password = ref('')
const error = ref('')
const busy = ref(false)

watch(channel, () => {
  password.value = ''
  error.value = ''
})

const close = () => {
  channel.value = null
}

const submit = async () => {
  if (!channel.value) return
  busy.value = true
  error.value = ''
  try {
    await api.enterChannel(channel.value.name, password.value)
    const entered = channel.value
    channel.value = null
    emit('entered', entered)
  } catch (e: any) {
    error.value = e?.response?.status === 403 ? 'Mot de passe incorrect' : apiErrorMessage(e)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.channel-password {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: var(--space-2);
}

.channel-password__panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  width: 400px;
  max-width: 100%;
  padding: var(--space-4) var(--space-3);
  pointer-events: auto;
}

.channel-password__panel h1,
.channel-password__panel p {
  margin: 0;
  text-align: center;
}

.field {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 52px;
  padding: 4px var(--space-1);
  background: #fff;
  color: #000;
}

.field-label {
  font-size: 11px;
  opacity: 0.7;
}

.field input {
  width: 100%;
  color: #000;
  font-size: var(--body-size);
  outline: none;
}

.field input::placeholder {
  color: #737373;
  opacity: 1;
}

.channel-password__error {
  color: #ff8a80;
}

.wide {
  width: 100% !important;
  padding: var(--space-2);
  font-size: var(--subtitle-size);
}

.switch {
  align-self: center;
  color: #fff;
  font-size: var(--subtitle-size);
  transition: color 0.15s;
}

.switch:hover {
  color: #ffd54f;
}
</style>
