<template>
  <v-dialog v-model="open">
    <div class="dialog-wrapper">
      <div class="dialog-content">
        <div class="letter-content">
          <div v-for="block in blocks" :key="block.id" :style="{ top: block.top + 'px', left: block.left + 'px', transform: `rotate(${block.rotation}deg)`, fontSize: block.fontSize + 'px' }" class="letter-block">
            <p class="letter-text">{{ block.text }}</p>
          </div>
        </div>
      </div>
      <div class="letter-actions">
        <span class="text-caption px-2">De {{ from }}</span>
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
  border-radius: 16px;
  background: white;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
}

.dialog-content {
  background: white;
  height: 90vh;
  width: 90vw;
  justify-content: center;
  display: flex;
  border-radius: 24px;
  overflow-y: scroll;
  padding: 24px;
}

.letter-content {
  display: flex;
  flex-shrink: 0;
  width: 900px;
  height: 1200px;
  position: relative;
  background: rgb(216, 216, 216);
}

.letter-block {
  position: absolute;
  transform-origin: center;
}

.letter-text {
  margin: unset;
  white-space: pre-wrap;
}

</style>

<script setup lang="ts">
defineProps<{
  blocks: LetterBlock[],
  from: string,
}>()

const open = defineModel<boolean>({ default: false })
</script>
