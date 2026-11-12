<template>
  <v-dialog v-model="open" :fullscreen="xs">
    <div class="dialog-wrapper">
      <div class="dialog-content pixel-frame">
        <div class="letter-content">
          <canvas ref="canvas" class="drawing-layer" width="900" height="1200"></canvas>
          <div v-for="block in blocks" :key="block.id" :style="{ top: block.top + 'px', left: block.left + 'px', transform: `rotate(${block.rotation}deg)`, fontSize: block.fontSize + 'px', color: block.color }" class="letter-block">
            <p class="letter-text">{{ block.text }}</p>
          </div>
        </div>
      </div>
      <div class="letter-actions pixel-frame">
        <span class="text-caption px-2">De {{ from }}</span>
        <v-btn v-if="removable" variant="text" color="red" prepend-icon="mdi-rocket-launch-outline" @click="emit('remove')">Retirer la fusée</v-btn>
        <v-btn variant="text" @click="open = false">Fermer</v-btn>
      </div>
    </div>
  </v-dialog>
</template>

<style scoped>

.dialog-wrapper {
  position: relative;
  margin-inline: auto;
}

.letter-actions {
  position: absolute;
  right: 24px;
  bottom: 24px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
}

.dialog-content {
  height: 90vh;
  width: 90vw;
  justify-content: center;
  display: flex;
  overflow-y: scroll;
  padding: 24px;
}

.letter-content {
  display: flex;
  flex-shrink: 0;
  width: 900px;
  height: 1200px;
  position: relative;
  color: white;
}

.drawing-layer {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.letter-block {
  position: absolute;
  transform-origin: center;
}

.letter-text {
  margin: unset;
  white-space: pre-wrap;
}

@media (max-width: 599px) {
  .dialog-wrapper {
    width: 100%;
    height: 100%;
  }

  .dialog-content {
    width: 100%;
    height: 100%;
    padding: 12px;
    clip-path: none;
    justify-content: flex-start;
    overflow: auto;
  }

  .letter-actions {
    right: 12px;
    bottom: 12px;
  }
}
</style>

<script setup lang="ts">
const props = defineProps<{
  blocks: LetterBlock[],
  strokes: Stroke[],
  from: string,
  removable?: boolean,
}>()
const emit = defineEmits<{ remove: [] }>()

const open = defineModel<boolean>({ default: false })
const { xs } = useDisplay()

const canvas = ref<HTMLCanvasElement | null>(null)

watch([canvas, () => props.strokes], () => {
  const ctx = canvas.value?.getContext('2d')
  if (ctx) drawStrokes(ctx, props.strokes)
})
</script>
