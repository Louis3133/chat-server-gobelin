<template>
  <div class="info-menu">
    <div class="info-menu__bar">
      <v-btn v-if="!backOnly && $slots.default" :icon="open || icon ? undefined : 'mdi-menu'" color="white" variant="outlined" :aria-label="open ? 'Fermer le menu' : 'Ouvrir le menu'" @click="open = !open">
        <img v-if="open" class="pixel-icon" src="/icons/actions/close.svg" alt="">
        <img v-else-if="icon" class="pixel-icon" :src="icon" alt="">
      </v-btn>
      <v-btn v-if="backLabel" icon="mdi-arrow-left" color="white" variant="outlined" :title="backLabel" :aria-label="backLabel" @click="emit('back')" />
      <template v-if="!backOnly">
        <slot name="actions" />
        <span v-if="title" class="info-menu__title">{{ title }}</span>
      </template>
    </div>
    <Transition name="info-menu-slide">
      <div v-if="$slots.default" v-show="open && !backOnly" class="info-menu__panel">
        <slot />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title?: string; backLabel?: string; backOnly?: boolean; icon?: string }>()
const emit = defineEmits<{ back: [] }>()

const open = ref(false)
</script>

<style scoped>
.info-menu__bar {
  position: fixed;
  top: 16px;
  left: 16px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: calc(100vw - 200px);
}

.info-menu__title {
  min-width: 0;
  overflow: hidden;
  color: #fff;
  font-size: var(--subtitle-medium-size);
  font-weight: var(--subtitle-medium-weight);
  text-shadow: 1px 1px 0 #000;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.info-menu__panel {
  position: fixed;
  top: 80px;
  left: 16px;
  bottom: 16px;
  z-index: 10;
  width: min(460px, calc(100vw - 32px));
  overflow-y: auto;
  pointer-events: auto;
}

@media (max-width: 599px) {
  .info-menu__bar {
    gap: 8px;
    max-width: calc(100vw - 88px);
  }

  .info-menu__title {
    font-size: 15px;
  }

  .info-menu__panel {
    top: 72px;
    left: 8px;
    bottom: 8px;
    width: calc(100vw - 16px);
  }
}

.info-menu-slide-enter-active,
.info-menu-slide-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.info-menu-slide-enter-from,
.info-menu-slide-leave-to {
  transform: translateX(-24px);
  opacity: 0;
}
</style>
