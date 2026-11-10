<template>
  <div class="info-menu">
    <div class="info-menu__bar">
      <v-btn :icon="open ? 'mdi-close' : 'mdi-menu'" color="white" variant="outlined" @click="open = !open" />
      <v-btn v-if="backLabel" icon="mdi-arrow-left" color="white" variant="outlined" :title="backLabel" :aria-label="backLabel" @click="emit('back')" />
      <slot name="actions" />
      <span v-if="title" class="info-menu__title">{{ title }}</span>
    </div>
    <Transition name="info-menu-slide">
      <div v-show="open" class="info-menu__panel">
        <slot />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title?: string; backLabel?: string }>()
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

.info-menu__bar :deep(.v-btn) {
  background: rgba(15, 18, 35, 0.78) !important;
}

.info-menu__title {
  min-width: 0;
  overflow: hidden;
  color: #fff;
  font: 600 18px/1 monospace;
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
