<template>
  <span class="birthday-field">
    <select v-model="day" :disabled="disabled" aria-label="Jour">
      <option :value="null">Jour</option>
      <option v-for="d in days" :key="d" :value="d">{{ d }}</option>
    </select>
    <select v-model="month" :disabled="disabled" aria-label="Mois">
      <option :value="null">Mois</option>
      <option v-for="(name, i) in MONTHS" :key="name" :value="i + 1">{{ name }}</option>
    </select>
  </span>
</template>

<script setup lang="ts">
defineProps<{ disabled?: boolean }>()
const model = defineModel<Birthday | null>({ default: null })

const MONTHS = ['janvier', 'février', 'mars', 'avril', 'mai', 'juin', 'juillet', 'août', 'septembre', 'octobre', 'novembre', 'décembre']

const day = ref<number | null>(model.value?.day ?? null)
const month = ref<number | null>(model.value?.month ?? null)
const days = computed(() => Array.from({ length: month.value ? new Date(2000, month.value, 0).getDate() : 31 }, (_, i) => i + 1))

watch(model, (value) => {
  day.value = value?.day ?? null
  month.value = value?.month ?? null
})

watch([day, month], ([d, m]) => {
  if (d && m && d > days.value.length) day.value = days.value.length
  const next = day.value && month.value ? { month: month.value, day: day.value } : null
  if (JSON.stringify(next) !== JSON.stringify(model.value)) model.value = next
})
</script>

<style scoped>
.birthday-field {
  display: flex;
  gap: var(--space-2);
}

.birthday-field select {
  flex: 1;
  min-width: 0;
  color: inherit;
  font: inherit;
  outline: none;
  cursor: pointer;
}

.birthday-field option {
  color: #000;
}
</style>
