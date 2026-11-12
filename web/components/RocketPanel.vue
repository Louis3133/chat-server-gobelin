<template>
  <Teleport to="body">
    <div class="rocket-panel scene-panel pixel-frame">
      <div>
        <h2 class="text-heading">Lettre pour {{ recipient }}</h2>
        <p class="text-subtitle-medium">{{ remaining > 0 ? `Arrivée dans ${formatRemaining(remaining)}` : 'Arrivée imminente' }}</p>
      </div>
      <v-btn :loading="cancelling" :disabled="remaining <= 0" @click="emit('cancel')">
        <template #prepend><img class="pixel-icon" src="/icons/actions/close.svg" alt=""></template>
        Annuler la lettre
      </v-btn>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const props = defineProps<{ letter: Message, recipient: string, cancelling?: boolean }>()
const emit = defineEmits<{ cancel: [], arrived: [] }>()

const now = ref(Date.now())
const timer = setInterval(() => { now.value = Date.now() }, 1000)
onBeforeUnmount(() => clearInterval(timer))

const remaining = computed(() => Math.max(0, Math.ceil((new Date(props.letter.deliver_at).getTime() - now.value) / 1000)))
watch(remaining, (seconds) => { if (seconds <= 0) emit('arrived') })

const formatRemaining = (seconds: number) => {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h) return `${h} h ${String(m).padStart(2, '0')} min`
  if (m) return `${m} min ${String(s).padStart(2, '0')} s`
  return `${s} s`
}
</script>

<style scoped>
.rocket-panel {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-3);
}

.rocket-panel h2,
.rocket-panel p {
  margin: 0;
}

.rocket-panel > div {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
</style>
