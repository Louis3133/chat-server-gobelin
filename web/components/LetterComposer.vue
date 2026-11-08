<template>
  <v-dialog class="dialog-letter" v-model="open">
    <div class="dialog-wrapper">
    <div @click.self="selectedId = null" class="dialog-content">
      <div class="tools">
        <v-btn :color="addingText ? 'primary' : undefined" @click="addingText = !addingText" class="tool-case font-tool" prepend-icon="mdi-format-text">
          Texte
        </v-btn>
        <v-btn class="tool-case font-tool">
          test
        </v-btn>
        <v-btn class="tool-case font-tool">
          test
        </v-btn>
      </div>
      <div @click.self="addBlock" :class="{ 'adding-text': addingText }" class="letter-content">
        <div :class="{ selected: block.id === selectedId }" @pointerdown="selectedId = block.id" v-for="block in texts" :key="block.id" :style="{ top: block.top + 'px', left: block.left + 'px', transform: `rotate(${block.rotation}deg)`, fontSize: block.fontSize + 'px' }" class="letter-block">
          <p @blur="updateText(block, $event)" contenteditable class="letter-text">{{ block.text }}</p>
          <span @pointerdown="startRotate(block, $event)" @pointermove="rotate" @pointerup="stopRotate" @pointercancel="stopRotate" class="rotate-handle">↻</span>
          <span @pointerdown="startMove(block, $event)" @pointermove="move" @pointerup="stopMove" @pointercancel="stopMove" class="move-handle">+</span>
          <span @pointerdown="startResize(block, $event)" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top left"></span>
          <span @pointerdown="startResize(block, $event)" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top right"></span>
          <span @pointerdown="startResize(block, $event)" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom left"></span>
          <span @pointerdown="startResize(block, $event)" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom right"></span>
        </div>
      </div>
    </div>
    <div class="letter-actions">
      <v-btn variant="text" :disabled="sending" @click="open = false">Fermer</v-btn>
      <v-btn color="primary" prepend-icon="mdi-send" :loading="sending" :disabled="!texts.length" @click="send">
        Envoyer à {{ recipient.username }}
      </v-btn>
    </div>
    </div>
  </v-dialog>
</template>

<style scoped>

.tool-case {
  border: 1px solid;
  padding: 8px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.tools {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
}

.letter-block {
  position: absolute;
  transform-origin: center;
}

.letter-text:focus{
  outline: none;
}

.selected-corner {
  display: none;
  touch-action: none
}

.selected{
  outline: 2px solid rgb(13, 153, 255);
}

.selected .rotate-handle{
  display: block;
}

.selected .move-handle{
  display: block;
}

.selected .selected-corner{
  display: block;
  width: 10px;
  height: 10px;
  background: rgb(13, 153, 255);
  position: absolute;
}

.selected-corner.top {
  top: -6px;
}

.selected-corner.left {
  left: -6px;
}

.selected-corner.left.top {
  cursor: nwse-resize;
}

.selected-corner.right.top {
  cursor: nesw-resize;
}

.selected-corner.left.bottom {
  cursor: nesw-resize;
}

.selected-corner.right.bottom {
  cursor:  nwse-resize;
}

.selected-corner.right {
  right: -6px;
}

.selected-corner.bottom {
  bottom: -6px;
}

.letter-text {
  margin: unset;
  min-width: 10px;
  white-space: pre-wrap;
}

.rotate-handle {
  position: absolute;
  top: -32px;
  left: 50%;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgb(13, 153, 255);
  color: white;
  font-size: 14px;
  line-height: 20px;
  text-align: center;
  transform: translateX(-50%);
  cursor: grab;
  user-select: none;
  display: none;
  touch-action: none;
}

.move-handle {
  position: absolute;
  bottom: -32px;
  left: 50%;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgb(13, 153, 255);
  color: white;
  font-size: 14px;
  line-height: 20px;
  text-align: center;
  transform: translateX(-50%);
  cursor: move;
  user-select: none;
  display: none;
  touch-action: none;
}

.rotate-handle::before {
  content: '';
  position: absolute;
  inset: -12px;
}

.move-handle::before {
  content: '';
  position: absolute;
  inset: -12px;
}

.letter-text:empty::before{
  content: 'écris ici';
}

.dialog-wrapper {
  position: relative;
  margin-inline: auto;
}

.letter-actions {
  position: absolute;
  right: 24px;
  bottom: 24px;
  display: flex;
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
  margin-inline: auto;
  border-radius: 24px;
  overflow-y: scroll;
  padding: 24px;
}

.letter-content.adding-text {
  cursor: text;
}

.letter-content {
  display: flex;
  width: 900px;
  height: 1200px;
  position: relative;
  background: rgb(216, 216, 216);
}


</style>

<script setup lang="ts">

const selectedId = ref<number | null>(null)
const addingText = ref(false)
const sending = ref(false)
const api = useChatApi()
let idCount = 3

type SelectedBlock = {
  centerX: number,
  centerY: number,
  pointerAngle: number,
  blockRotation : number,
  letterBlock: LetterBlock,
}

type Moving = {
  startX: number,
  startY: number,
  startTop: number,
  startLeft: number,
  letterBlock: LetterBlock,
}

type SelectedBlockSize = {
  centerX: number,
  centerY: number,
  initSize: number,
  pointerDistance : number,
  letterBlock: LetterBlock,
}


let selectedBlock: SelectedBlock | null = null
let SelectedBlockSize: SelectedBlockSize | null = null
let moving: Moving | null = null

const texts = ref<LetterBlock[]>([])

const getDistance = (centerX: number, centerY: number, event: PointerEvent) => {
  const dx =  event.clientX - centerX;
  const dy =  event.clientY - centerY;
  const distance =Math.hypot(dx, dy);

  return distance
}

const getPointerAngle = (centerX: number, centerY: number, event: PointerEvent) => {
  const dx =  event.clientX - centerX;
  const dy =  event.clientY - centerY;
  const radiantAngle = Math.atan2(dy, dx)
  const angle = radiantAngle * 180 / Math.PI

  return angle
}

const addBlock = async (event: MouseEvent) => {
  selectedId.value = null
  if (!addingText.value) return
  addingText.value = false

  const sheet = event.currentTarget as HTMLElement
  texts.value.push({fontSize:12, id: idCount, top: event.offsetY, left: event.offsetX, text:'', rotation: 0 })
  selectedId.value = idCount
  idCount +=1

  await nextTick()
  const newText = sheet.lastElementChild?.querySelector('.letter-text') as HTMLElement | null
  newText?.focus()
}

const updateText = (block: LetterBlock, event: Event) => {
  const text = (event.target as HTMLElement).innerText

  if (text.trim().length === 0) {
    const blockIndex = texts.value.findIndex(item => item.id === block.id)
    if (blockIndex !== -1) {
      texts.value.splice(blockIndex, 1)
    }
  } else {
    block.text = text
  }
}

// Rotate

const startRotate = (block: LetterBlock, event: PointerEvent) => {
  const handle = event.currentTarget as HTMLElement
  const rect = (handle.parentElement as HTMLElement).getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 2
  handle.setPointerCapture(event.pointerId)

  selectedBlock = {
    centerX,
    centerY,
    pointerAngle: getPointerAngle(centerX, centerY, event),
    blockRotation: block.rotation,
    letterBlock: block,
  }
}

const rotate = (event: PointerEvent) => {
  if (!selectedBlock) return

  const angle = getPointerAngle(selectedBlock.centerX, selectedBlock.centerY, event)
  selectedBlock.letterBlock.rotation = selectedBlock.blockRotation + angle - selectedBlock.pointerAngle
}

const stopRotate = () => {
  selectedBlock = null
}

// Move

const startMove = (block: LetterBlock, event: PointerEvent) => {
  const handle = event.currentTarget as HTMLElement
  handle.setPointerCapture(event.pointerId)

  moving = {
    startX: event.clientX,
    startY: event.clientY,
    startTop: block.top,
    startLeft: block.left,
    letterBlock: block,
  }
}

const move = (event: PointerEvent) => {
  if (!moving) return

  moving.letterBlock.left = moving.startLeft + event.clientX - moving.startX
  moving.letterBlock.top = moving.startTop + event.clientY - moving.startY
}

const stopMove = () => {
  moving = null
}

// Resize

const startResize = (block: LetterBlock, event: PointerEvent) => {
  const handle = event.currentTarget as HTMLElement
  const rect = (handle.parentElement as HTMLElement).getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 2
  handle.setPointerCapture(event.pointerId)

  SelectedBlockSize = {
    centerX,
    centerY,
    pointerDistance: getDistance(centerX, centerY, event),
    initSize: block.fontSize,
    letterBlock: block,
  }
}

const resize = (event: PointerEvent) => {
  if (!SelectedBlockSize) return

  const distance = getDistance(SelectedBlockSize.centerX, SelectedBlockSize.centerY, event)
  const fontSize = SelectedBlockSize.initSize * (distance / SelectedBlockSize.pointerDistance)
  SelectedBlockSize.letterBlock.fontSize = Math.min(Math.max(fontSize, 8), 120)
}

const stopResize = () => {
  SelectedBlockSize = null
}



const props = defineProps<{
  channelId: number,
  recipient: Planet,
}>()

const open = defineModel<boolean>({ default: false })
const emit = defineEmits<{ sent: [message: Message], failed: [error: string] }>()


const send = async () => {
  ;(document.activeElement as HTMLElement | null)?.blur()
  await nextTick()
  if (!texts.value.length) return

  const content = serializeLetter(texts.value)
  if (content.length > LETTER_MAX_LENGTH) {
    emit('failed', 'Lettre trop longue')
    return
  }

  sending.value = true
  try {
    const sent = await api.sendMessage(props.channelId, content, props.recipient.id)
    texts.value = []
    selectedId.value = null
    open.value = false
    emit('sent', sent)
  } catch (error) {
    emit('failed', apiErrorMessage(error))
  } finally {
    sending.value = false
  }
}
</script>
