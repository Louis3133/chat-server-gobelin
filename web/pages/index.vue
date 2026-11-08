<template>
  <v-container class="d-flex align-center justify-center fill-height" max-width="420">
    <v-card class="px-6 py-8 w-100">
      <v-tabs v-model="mode" grow class="mb-4">
        <v-tab value="login">Connexion</v-tab>
        <v-tab value="register">Inscription</v-tab>
      </v-tabs>

      <v-form v-model="valid" @submit.prevent="onSubmit">
        <v-text-field v-model="username" :readonly="loading" :rules="[required]" label="Nom d'utilisateur" class="mb-2" />
        <v-text-field v-model="password" :readonly="loading" :rules="[required]" label="Mot de passe" type="password" />
        <v-btn :disabled="!valid" :loading="loading" block color="success" size="large" type="submit" variant="elevated">
          {{ mode === 'login' ? 'Se connecter' : 'Créer le compte' }}
        </v-btn>
      </v-form>
    </v-card>

    <v-snackbar v-model="snackbar" color="red" :timeout="3000">{{ snackbarText }}</v-snackbar>
  </v-container>
</template>

<script setup lang="ts">
const api = useChatApi()

const mode = ref<'login' | 'register'>('login')
const valid = ref(false)
const loading = ref(false)
const username = ref('')
const password = ref('')
const snackbar = ref(false)
const snackbarText = ref('')

const required = (v: string) => !!v || 'Champ obligatoire'

const fail = (error: unknown) => {
  snackbarText.value = apiErrorMessage(error)
  snackbar.value = true
}

const onSubmit = async () => {
  if (!valid.value) return
  loading.value = true
  try {
    if (mode.value === 'register') {
      await api.register(username.value, password.value)
    }
    await api.login(username.value, password.value)
    await navigateTo('/channels')
  } catch (error) {
    fail(error)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  if (api.getToken()) {
    await navigateTo('/channels')
    return
  }
})
</script>
