<template>
  <div class="auth">
    <div class="auth-panel pixel-frame">
      <template v-if="mode === 'home'">
        <h1 class="title text-heading">Epistellaire</h1>
        <button class="pixel-btn wide" type="button" @click="open('register')">Créer un compte</button>
        <button class="pixel-btn wide" type="button" @click="open('login')">Connexion</button>
      </template>

      <form v-else-if="mode === 'pseudo'" class="auth-form" @submit.prevent="onPseudo">
        <h1 class="text-heading">Choisir son pseudo</h1>
        <label class="field">
          <span class="field-label">Pseudo</span>
          <input v-model="pseudo" :maxlength="pseudoMax" autocomplete="nickname" spellcheck="false" autofocus>
        </label>
        <button class="pixel-btn wide" type="submit" :disabled="!pseudo.trim()">Continuer</button>
      </form>

      <form v-else class="auth-form" @submit.prevent="onSubmit">
        <h1 class="text-heading">{{ mode === 'login' ? 'Connexion' : 'Créer un compte' }}</h1>
        <label class="field">
          <span class="field-label">Nom d'utilisateur</span>
          <input v-model="username" :readonly="loading" autocomplete="username" autocapitalize="off" spellcheck="false">
        </label>
        <div class="password-row">
          <label class="field">
            <span class="field-label">Mot de passe</span>
            <input v-model="password" :readonly="loading" :type="showPassword ? 'text' : 'password'" :autocomplete="mode === 'login' ? 'current-password' : 'new-password'">
          </label>
          <button class="eye" type="button" :aria-label="showPassword ? 'Masquer le mot de passe' : 'Afficher le mot de passe'" @click="showPassword = !showPassword">
            <v-icon :icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'" color="black" />
          </button>
        </div>
        <button class="pixel-btn wide" type="submit" :disabled="!canSubmit">
          {{ loading ? '…' : mode === 'login' ? 'Se connecter' : 'Créer le compte' }}
        </button>
        <button class="switch" type="button" @click="open(mode === 'login' ? 'register' : 'login')">
          {{ mode === 'login' ? 'Créer un compte' : "J'ai déjà un compte" }}
        </button>
      </form>
    </div>

    <v-snackbar v-model="snackbar" color="red" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </div>
</template>

<script setup lang="ts">
const api = useChatApi()

const mode = ref<'home' | 'login' | 'register' | 'pseudo'>('home')
const loading = ref(false)
const username = ref('')
const password = ref('')
const showPassword = ref(false)
const pseudo = ref('')
const pseudoMax = PSEUDO_MAX
const snackbar = ref(false)
const snackbarText = ref('')

const canSubmit = computed(() => !loading.value && !!username.value.trim() && !!password.value)

const open = (next: 'login' | 'register') => {
  mode.value = next
  showPassword.value = false
}

const fail = (error: unknown) => {
  snackbarText.value = apiErrorMessage(error)
  snackbar.value = true
}

const onSubmit = async () => {
  if (!canSubmit.value) return
  loading.value = true
  try {
    const isRegister = mode.value === 'register'
    if (isRegister) {
      await api.register(username.value, password.value)
    }
    await api.login(username.value, password.value)
    if (isRegister) {
      mode.value = 'pseudo'
      return
    }
    await navigateTo('/channels')
  } catch (error) {
    fail(error)
  } finally {
    loading.value = false
  }
}

const onPseudo = async () => {
  if (!pseudo.value.trim()) return
  savePseudo(pseudo.value)
  await navigateTo('/welcome')
}

onMounted(async () => {
  if (api.getToken()) {
    await navigateTo('/channels')
    return
  }
})
</script>

<style scoped>
.auth {
  position: fixed;
  inset: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: var(--space-2);
  pointer-events: auto;
}

.auth-panel {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: var(--space-4);
  width: 400px;
  max-width: 100%;
  min-height: 460px;
  padding: var(--space-4) var(--space-3);
  pointer-events: auto;
}

.title {
  text-align: center;
  text-transform: uppercase;
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.auth-form h1 {
  text-align: center;
}

.wide {
  width: 100% !important;
  padding: var(--space-2);
  font-size: var(--subtitle-size);
}

.wide:disabled {
  opacity: 0.5;
  cursor: default;
}

.field {
  display: flex;
  flex: 1;
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
  color: #000;
  font-size: var(--body-size);
  outline: none;
}

.password-row {
  display: flex;
}

.eye {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 56px;
  border-left: 4px solid;
  border-image: linear-gradient(180deg, #fff -2.13%, #737373 100%) 1;
  background: #fff;
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
