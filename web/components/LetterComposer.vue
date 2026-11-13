<template>
  <v-dialog class="dialog-letter" v-model="open" fullscreen>
    <div class="dialog-wrapper">
    <div @click.self="selectedId = null" class="dialog-content">
      <div class="letter-header">
      <div :class="{ hidden: previewing }" class="tools">
        <v-btn :class="{ active: tool === 'draw' }" @click="toggleTool('draw')" aria-label="Dessiner" class="tool-case"><img src="/icons/composer/feather.svg" alt=""></v-btn>
        <v-btn :class="{ active: tool === 'text' }" @click="toggleTool('text')" aria-label="Écrire" class="tool-case"><img src="/icons/composer/text.svg" alt=""></v-btn>
        <v-btn :class="{ active: showLines }" @click="showLines = !showLines" aria-label="Lignes" class="tool-case"><img src="/icons/composer/rows.svg" alt=""></v-btn>
        <v-btn :class="{ active: tool === 'resize' }" @click="toggleTool('resize')" aria-label="Redimensionner" class="tool-case"><img src="/icons/composer/crop.svg" alt=""></v-btn>
        <v-btn :class="{ active: paletteOpen }" @click="paletteOpen = !paletteOpen" aria-label="Couleur" class="tool-case"><span class="pipette">
            <img src="/icons/composer/pipette.svg" alt="">
            <svg :fill="color" viewBox="-0.5 -0.5 12 12" shape-rendering="crispEdges" aria-hidden="true" class="pipette-cap">
              <rect x="8" y="1" width="1" height="1" />
              <rect x="7" y="2" width="1" height="1" />
              <rect x="8" y="2" width="1" height="1" />
              <rect x="9" y="2" width="1" height="1" />
              <rect x="6" y="3" width="1" height="1" />
              <rect x="7" y="3" width="1" height="1" />
              <rect x="8" y="3" width="1" height="1" />
              <rect x="7" y="4" width="1" height="1" />
            </svg>
          </span>
        </v-btn>
        <v-btn @click="imageInput?.click()" aria-label="Image" class="tool-case"><img src="/icons/composer/image-sharp.svg" alt=""></v-btn>
        <input ref="imageInput" @change="addImage" type="file" accept="image/*" hidden>
      </div>
      <h2 v-if="previewing" class="preview-label text-heading">Ça te plaît&nbsp;?</h2>
      <div v-if="tool === 'draw' && !paletteOpen && !previewing" class="palette widths pixel-frame">
        <button v-for="width in STROKE_WIDTHS" :key="width" :class="{ current: width === strokeWidth }" :aria-label="`Épaisseur ${width}`" @click="strokeWidth = width" class="swatch width">
          <span :style="{ width: width + 'px', height: width + 'px', background: color }" class="width-dot"></span>
        </button>
      </div>
      <div v-if="paletteOpen && !previewing" class="palette pixel-frame">
        <button v-for="swatch in PALETTE" :key="swatch.value" :class="{ current: swatch.value === color }" :style="{ background: swatch.value }" :aria-label="swatch.name" @click="pickColor(swatch.value)" class="swatch"></button>
      </div>
      </div>
      <div ref="frame" :style="{ height: `calc(${frameHeight}px + 2 * var(--frame-width))` }" class="letter-frame pixel-frame">
      <div @click.self="addBlock" :class="{ 'adding-text': tool === 'text', 'drawing': tool === 'draw', 'resizing': tool === 'resize', 'ruled': showLines && !previewing, previewing }" :style="{ transform: `scale(${scale})`, '--handle-scale': 1 / scale }" class="letter-content">
        <TapedImage v-for="image in images" :key="image.id" :image="image" :class="{ selected: image.id === selectedId }" @pointerdown="selectedId = image.id; startMove($event, moveTarget(image))" @pointermove="move" @pointerup="stopMove" @pointercancel="stopMove" class="letter-image">
          <span @pointerdown.stop="startRotate($event, rotateTarget(image))" @pointermove="rotate" @pointerup="stopRotate" @pointercancel="stopRotate" class="rotate-handle">↻</span>
          <span @pointerdown.stop @click="removeImage(image)" class="delete-handle">×</span>
          <span @pointerdown.stop="startResize($event, resizeImage(image))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top left"></span>
          <span @pointerdown.stop="startResize($event, resizeImage(image))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top right"></span>
          <span @pointerdown.stop="startResize($event, resizeImage(image))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom left"></span>
          <span @pointerdown.stop="startResize($event, resizeImage(image))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom right"></span>
        </TapedImage>
        <canvas ref="canvas" @pointerdown="startStroke" @pointermove="extendStroke" @pointerup="endStroke" @pointercancel="endStroke" class="drawing-layer" width="900" height="1200"></canvas>
        <template v-if="tool === 'resize'">
          <div :class="{ selected: drawing.id === selectedId }" @pointerdown="selectedId = drawing.id" v-for="drawing in drawings" :key="drawing.id" :style="drawingStyle(drawing)" class="letter-drawing">
            <span @pointerdown="startRotate($event, rotateDrawing(drawing))" @pointermove="rotate" @pointerup="stopRotate" @pointercancel="stopRotate" class="rotate-handle">↻</span>
            <span @pointerdown.stop @click="removeDrawing(drawing)" class="delete-handle">×</span>
            <span @pointerdown="startMove($event, moveDrawing(drawing))" @pointermove="move" @pointerup="stopMove" @pointercancel="stopMove" class="move-handle">+</span>
            <span @pointerdown="startResize($event, resizeDrawing(drawing))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top left"></span>
            <span @pointerdown="startResize($event, resizeDrawing(drawing))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top right"></span>
            <span @pointerdown="startResize($event, resizeDrawing(drawing))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom left"></span>
            <span @pointerdown="startResize($event, resizeDrawing(drawing))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom right"></span>
          </div>
        </template>
        <div :class="{ selected: block.id === selectedId }" @pointerdown="selectedId = block.id" v-for="block in texts" :key="block.id" :style="{ top: block.top + 'px', left: block.left + 'px', transform: `rotate(${block.rotation}deg)`, fontSize: block.fontSize + 'px', color: block.color }" class="letter-block">
          <p @blur="updateText(block, $event)" :contenteditable="!previewing" class="letter-text">{{ block.text }}</p>
          <span @pointerdown="startRotate($event, rotateTarget(block))" @pointermove="rotate" @pointerup="stopRotate" @pointercancel="stopRotate" class="rotate-handle">↻</span>
          <span @pointerdown.stop @click="removeText(block)" class="delete-handle">×</span>
          <span @pointerdown="startMove($event, moveTarget(block))" @pointermove="move" @pointerup="stopMove" @pointercancel="stopMove" class="move-handle">+</span>
          <span @pointerdown="startResize($event, resizeText(block))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top left"></span>
          <span @pointerdown="startResize($event, resizeText(block))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner top right"></span>
          <span @pointerdown="startResize($event, resizeText(block))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom left"></span>
          <span @pointerdown="startResize($event, resizeText(block))" @pointermove="resize" @pointerup="stopResize" @pointercancel="stopResize" class="selected-corner bottom right"></span>
        </div>
      </div>
      </div>
    </div>
    <div class="letter-actions">
      <template v-if="!previewing">
        <v-btn @click="open = false">Annuler</v-btn>
        <v-btn :disabled="isEmpty" @click="validate">Valider</v-btn>
      </template>
      <template v-else>
        <v-btn :disabled="sending" @click="previewing = false">Non</v-btn>
        <v-btn color="primary" :loading="sending" @click="send">Oui</v-btn>
      </template>
    </div>
    </div>
  </v-dialog>
</template>

<style scoped>

.tool-case.active :deep(.v-btn__overlay) {
  opacity: .2;
}

.pipette {
  position: relative;
}

.pipette-cap {
  position: absolute;
  inset: 0;
  width: 32px;
  height: 32px;
}

.tool-case img {
  display: block;
  width: 32px;
  height: 32px;
}

.letter-header {
  position: relative;
  display: grid;
  place-items: center;
  padding-bottom: 24px;
}

.letter-header > * {
  grid-area: 1 / 1;
}

.tools {
  display: grid;
  grid-template-columns: repeat(3, auto);
  gap: 24px;
}

.palette {
  position: absolute;
  top: 100%;
  left: 50%;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(4, auto);
  gap: 12px;
  padding: 12px;
  transform: translateX(-50%);
}

.swatch {
  width: 32px;
  height: 32px;
  border: 3px solid transparent;
  cursor: pointer;
}

.widths {
  grid-template-columns: repeat(5, auto);
}

.swatch.width {
  display: grid;
  place-items: center;
  background: transparent;
}

.width-dot {
  border-radius: 50%;
}

.swatch.current {
  border-color: #ffffff;
  outline: 2px solid #000000;
  outline-offset: -5px;
}

.tools.hidden {
  visibility: hidden;
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

.drawing-layer {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;

}

.drawing .letter-block {
  opacity: .5;
  pointer-events: none;
}

.drawing .drawing-layer {
  pointer-events: auto;
  cursor: crosshair;
  touch-action: none;
}

.letter-drawing {
  position: absolute;
}

.letter-image {
  cursor: move;
  touch-action: none;
}

.drawing .letter-image {
  pointer-events: none;
}

.resizing .letter-drawing:not(.selected) {
  outline: 2px dashed rgba(255, 255, 255, .4);
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

.selected .delete-handle{
  display: block;
}

.resizing .selected .selected-corner{
  display: block;
  width: var(--corner-size);
  height: var(--corner-size);
  background: rgb(13, 153, 255);
  position: absolute;
  transform: translate(-50%, -50%) scale(var(--handle-scale, 1));
}

.selected-corner::before {
  content: '';
  position: absolute;
  inset: -10px;
}

.selected-corner.top {
  top: 0;
}

.selected-corner.left {
  left: 0;
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
  left: 100%;
}

.selected-corner.bottom {
  top: 100%;
}

.letter-text {
  margin: unset;
  min-width: 10px;
  white-space: pre-wrap;
}

.rotate-handle,
.move-handle,
.delete-handle {
  position: absolute;
  width: var(--handle-size);
  height: var(--handle-size);
  border-radius: 50%;
  background: rgb(13, 153, 255);
  color: white;
  font-size: calc(var(--handle-size) * .7);
  line-height: var(--handle-size);
  text-align: center;
  transform: translate(-50%, -50%) scale(var(--handle-scale, 1));
  user-select: none;
  display: none;
  touch-action: none;
}

.rotate-handle::before,
.move-handle::before,
.delete-handle::before {
  content: '';
  position: absolute;
  inset: -12px;
}

.rotate-handle {
  top: calc(-1 * var(--handle-distance));
  left: calc(50% - var(--handle-spacing) / 2);
  cursor: grab;
}

.move-handle {
  top: calc(100% + var(--handle-distance));
  left: 50%;
  cursor: move;
}

.delete-handle {
  top: calc(-1 * var(--handle-distance));
  left: calc(50% + var(--handle-spacing) / 2);
  background: rgb(255, 77, 77);
  cursor: pointer;
}

.letter-text:empty::before{
  content: 'écris ici';
}

.dialog-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
}

.letter-actions {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  justify-content: center;
  gap: 24px;
  padding: 48px 32px 32px;
  background: linear-gradient(transparent, rgba(0, 0, 0, .9) 60%);
  pointer-events: none;
}

.letter-actions > * {
  pointer-events: auto;
}

.dialog-content {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow-y: scroll;
  padding: 32px 32px 128px;
  background: rgba(0, 0, 0, .8);
  clip-path: none;
}

.letter-content.adding-text {
  cursor: text;
}

.letter-frame {
  position: relative;
  width: 100%;
  max-width: calc(900px + 2 * var(--frame-width));
  margin-inline: auto;
  flex-shrink: 0;
}

.letter-content {
  --handle-size: 20px;
  --corner-size: 10px;
  --handle-distance: calc((var(--handle-size) / 2 + 14px) * var(--handle-scale, 1));
  --handle-spacing: calc((var(--handle-size) + 24px) * var(--handle-scale, 1));
  display: flex;
  width: 900px;
  height: 1200px;
  position: absolute;
  top: 0;
  left: 0;
  transform-origin: top left;
  color: white;
}

.letter-content.ruled {
  background: linear-gradient(transparent 0 61px, rgba(255, 255, 255, .6) 61px 64px) 40px 0 / calc(100% - 80px) 64px repeat-y;
}

.letter-content.previewing {
  pointer-events: none;
}


@media (pointer: coarse) {
  .letter-content {
    --handle-size: 36px;
    --corner-size: 22px;
  }
}
</style>

<script setup lang="ts">


const selectedId = ref<number | null>(null)
const tool = ref<'select' | 'text' | 'draw' | 'resize'>('select')
const previewing = ref(false)
const showLines = ref(false)

// Color

const PALETTE = [
  { name: 'Blanc', value: '#ffffff' },
  { name: 'Jaune', value: '#ffd84d' },
  { name: 'Orange', value: '#ff8c42' },
  { name: 'Rouge', value: '#ff4d4d' },
  { name: 'Rose', value: '#ff7eb6' },
  { name: 'Violet', value: '#a77bff' },
  { name: 'Bleu', value: '#4da6ff' },
  { name: 'Vert', value: '#5ed67a' },
]

const color = ref(PALETTE[0]!.value)
const paletteOpen = ref(false)

// Stroke width

const STROKE_WIDTHS = [2.5, 4, 8, 14, 22]
const strokeWidth = ref(2.5)

const pickColor = (value: string) => {
  color.value = value
  paletteOpen.value = false

  const block = texts.value.find(item => item.id === selectedId.value)
  if (block) block.color = value

  const drawing = drawings.value.find(item => item.id === selectedId.value)
  if (drawing) {
    drawing.strokes.forEach(stroke => { stroke.color = value })
    redraw()
  }
}

// Scale

const frame = ref<HTMLElement | null>(null)
const scale = ref(1)
const frameHeight = computed(() => LETTER_HEIGHT * scale.value)
const frameObserver = new ResizeObserver(([entry]) => {
  if (entry) scale.value = entry.contentRect.width / LETTER_WIDTH
})

watch(frame, (element, previous) => {
  if (previous) frameObserver.unobserve(previous)
  if (element) frameObserver.observe(element)
})

onBeforeUnmount(() => frameObserver.disconnect())

const sheetPoint = (event: MouseEvent) => {
  const sheet = event.currentTarget as HTMLElement
  const rect = sheet.getBoundingClientRect()
  return {
    x: (event.clientX - rect.left) / scale.value - sheet.clientLeft,
    y: (event.clientY - rect.top) / scale.value - sheet.clientTop,
  }
}

const canvasPoint = (event: PointerEvent) => {
  const canvasElement = event.currentTarget as HTMLCanvasElement
  const rect = canvasElement.getBoundingClientRect()
  return [
    (event.clientX - rect.left) * canvasElement.width / rect.width,
    (event.clientY - rect.top) * canvasElement.height / rect.height,
  ]
}
const sending = ref(false)
const api = useChatApi()
let idCount = 3

type SelectedBlock = {
  centerX: number,
  centerY: number,
  pointerAngle: number,
  apply: (angle: number) => void,
}

type Moving = {
  startX: number,
  startY: number,
  apply: (dx: number, dy: number) => void,
}

type Resizing = {
  centerX: number,
  centerY: number,
  pointerDistance : number,
  apply: (factor: number) => void,
}


let selectedBlock: SelectedBlock | null = null
let resizing: Resizing | null = null
let moving: Moving | null = null

const texts = ref<LetterBlock[]>([])

const toggleTool = (name: 'text' | 'draw' | 'resize') => {
  if (tool.value === name) {
    tool.value = 'select'
  } else {
    tool.value = name
  }
}

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
  if (tool.value !== 'text') return
  tool.value = 'select';

  const sheet = event.currentTarget as HTMLElement
  const point = sheetPoint(event)
  texts.value.push({fontSize:32, id: idCount, top: point.y, left: point.x, text:'', rotation: 0, color: color.value })
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

const startRotate = (event: PointerEvent, apply: (angle: number) => void) => {
  const handle = event.currentTarget as HTMLElement
  const rect = (handle.parentElement as HTMLElement).getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 2
  handle.setPointerCapture(event.pointerId)

  selectedBlock = {
    centerX,
    centerY,
    pointerAngle: getPointerAngle(centerX, centerY, event),
    apply,
  }
}

const rotate = (event: PointerEvent) => {
  if (!selectedBlock) return

  const angle = getPointerAngle(selectedBlock.centerX, selectedBlock.centerY, event)
  selectedBlock.apply(angle - selectedBlock.pointerAngle)
}

const rotateTarget = (target: { rotation: number }) => {
  const initial = target.rotation

  return (angle: number) => {
    target.rotation = initial + angle
  }
}

const rotateDrawing = (drawing: Drawing) => {
  const bounds = drawingBounds(drawing)
  const centerX = bounds.left + bounds.width / 2
  const centerY = bounds.top + bounds.height / 2
  const initial = drawing.strokes.map(stroke => [...stroke.points])

  return (angle: number) => {
    const cos = Math.cos(angle * Math.PI / 180)
    const sin = Math.sin(angle * Math.PI / 180)
    drawing.strokes.forEach((stroke, i) => {
      const points = initial[i]!
      stroke.points = points.map((value, j) => {
        const dx = (j % 2 === 0 ? value : points[j - 1]!) - centerX
        const dy = (j % 2 === 0 ? points[j + 1]! : value) - centerY
        return j % 2 === 0 ? centerX + dx * cos - dy * sin : centerY + dx * sin + dy * cos
      })
    })
    redraw()
  }
}

const stopRotate = () => {
  selectedBlock = null
}

// Move

const startMove = (event: PointerEvent, apply: (dx: number, dy: number) => void) => {
  const handle = event.currentTarget as HTMLElement
  handle.setPointerCapture(event.pointerId)

  moving = {
    startX: event.clientX,
    startY: event.clientY,
    apply,
  }
}

const move = (event: PointerEvent) => {
  if (!moving) return

  moving.apply((event.clientX - moving.startX) / scale.value, (event.clientY - moving.startY) / scale.value)
}

const moveTarget = (target: { top: number, left: number }) => {
  const { top, left } = target

  return (dx: number, dy: number) => {
    target.left = left + dx
    target.top = top + dy
  }
}

const moveDrawing = (drawing: Drawing) => {
  const initial = drawing.strokes.map(stroke => [...stroke.points])

  return (dx: number, dy: number) => {
    drawing.strokes.forEach((stroke, i) => {
      stroke.points = initial[i]!.map((value, j) => value + (j % 2 === 0 ? dx : dy))
    })
    redraw()
  }
}

const stopMove = () => {
  moving = null
}

// Resize

const startResize = (event: PointerEvent, apply: (factor: number) => void) => {
  const handle = event.currentTarget as HTMLElement
  const rect = (handle.parentElement as HTMLElement).getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 2
  handle.setPointerCapture(event.pointerId)

  resizing = {
    centerX,
    centerY,
    pointerDistance: getDistance(centerX, centerY, event),
    apply,
  }
}

const resize = (event: PointerEvent) => {
  if (!resizing) return

  const distance = getDistance(resizing.centerX, resizing.centerY, event)
  resizing.apply(distance / resizing.pointerDistance)
}

const stopResize = () => {
  resizing = null
}

const resizeText = (block: LetterBlock) => {
  const initSize = block.fontSize

  return (factor: number) => {
    block.fontSize = Math.min(Math.max(initSize * factor, 8), 120)
  }
}

const resizeDrawing = (drawing: Drawing) => {
  const bounds = drawingBounds(drawing)
  const centerX = bounds.left + bounds.width / 2
  const centerY = bounds.top + bounds.height / 2
  const size = Math.max(bounds.width, bounds.height)
  const initial = drawing.strokes.map(stroke => ({ width: stroke.width, points: [...stroke.points] }))

  return (factor: number) => {
    const clamped = Math.min(Math.max(factor, 20 / size), LETTER_HEIGHT / size)
    drawing.strokes.forEach((stroke, i) => {
      stroke.width = initial[i]!.width * clamped
      stroke.points = initial[i]!.points.map((value, j) =>
        j % 2 === 0 ? centerX + (value - centerX) * clamped : centerY + (value - centerY) * clamped)
    })
    redraw()
  }
}

const resizeImage = (image: LetterImage) => {
  const centerX = image.left + image.width / 2
  const centerY = image.top + image.height / 2
  const { width, height } = image
  const size = Math.max(width, height)

  return (factor: number) => {
    const clamped = Math.min(Math.max(factor, 60 / size), LETTER_WIDTH / size)
    image.width = width * clamped
    image.height = height * clamped
    image.left = centerX - image.width / 2
    image.top = centerY - image.height / 2
  }
}

// Image

const IMAGE_MAX_SIDE = 800
const IMAGE_MAX_LENGTH = 40000
const IMAGE_DISPLAY_SIZE = 400

const imageInput = ref<HTMLInputElement | null>(null)
const images = ref<LetterImage[]>([])

const encodeImage = (bitmap: ImageBitmap, budget: number) => {
  const encoder = document.createElement('canvas')
  const ctx = encoder.getContext('2d')
  let side = IMAGE_MAX_SIDE

  while (ctx && side >= 64) {
    const ratio = Math.min(1, side / Math.max(bitmap.width, bitmap.height))
    encoder.width = Math.round(bitmap.width * ratio)
    encoder.height = Math.round(bitmap.height * ratio)
    ctx.drawImage(bitmap, 0, 0, encoder.width, encoder.height)
    const src = encoder.toDataURL('image/jpeg', 0.75)
    if (src.length <= budget) return src
    side *= 0.8
  }
  return null
}

const addImage = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return

  let bitmap: ImageBitmap
  try {
    bitmap = await createImageBitmap(file)
  } catch {
    emit('failed', 'Image illisible')
    return
  }

  const used = serializeLetter(texts.value, drawings.value.flatMap(drawing => drawing.strokes), images.value).length
  const src = encodeImage(bitmap, Math.min(IMAGE_MAX_LENGTH, LETTER_MAX_LENGTH - used - 100))
  const ratio = IMAGE_DISPLAY_SIZE / Math.max(bitmap.width, bitmap.height)
  const width = bitmap.width * ratio
  const height = bitmap.height * ratio
  bitmap.close()

  if (!src) {
    emit('failed', 'Plus de place pour une image dans cette lettre')
    return
  }

  images.value.push({ id: idCount, top: (LETTER_HEIGHT - height) / 2, left: (LETTER_WIDTH - width) / 2, width, height, rotation: 0, src })
  selectedId.value = idCount
  idCount += 1
  tool.value = 'resize'
}

const removeText = (block: LetterBlock) => {
  texts.value = texts.value.filter(item => item.id !== block.id)
  selectedId.value = null
}

const removeDrawing = (drawing: Drawing) => {
  drawings.value = drawings.value.filter(item => item.id !== drawing.id)
  selectedId.value = null
  redraw()
}

const removeImage = (image: LetterImage) => {
  images.value = images.value.filter(item => item.id !== image.id)
  selectedId.value = null
}

// Draw

type Drawing = {
  id: number,
  strokes: Stroke[],
}

const canvas = ref<HTMLCanvasElement | null>(null)
const drawings = ref<Drawing[]>([])
const isEmpty = computed(() => !texts.value.length && !drawings.value.length && !images.value.length)
let currentDrawing: Drawing | null = null
let currentStroke: Stroke | null = null

watch(tool, () => {
  currentDrawing = null
  if (drawings.value.some(drawing => drawing.id === selectedId.value)) selectedId.value = null
})

const redraw = () => {
  const ctx = canvas.value?.getContext('2d')
  if (ctx) drawStrokes(ctx, drawings.value.flatMap(drawing => drawing.strokes))
}

const startStroke = (event: PointerEvent) => {
  (event.currentTarget as HTMLCanvasElement).setPointerCapture(event.pointerId)

  if (!currentDrawing) {
    drawings.value.push({ id: idCount, strokes: [] })
    currentDrawing = drawings.value[drawings.value.length - 1]!
    idCount += 1
  }

  currentDrawing.strokes.push({ color: color.value, width: strokeWidth.value, points: canvasPoint(event) })
  currentStroke = currentDrawing.strokes[currentDrawing.strokes.length - 1]!
  redraw()
}

const extendStroke = (event: PointerEvent) => {
  if (!currentStroke) return

  currentStroke.points.push(...canvasPoint(event))
  redraw()
}

const endStroke = () => {
  currentStroke = null
}

const drawingBounds = (drawing: Drawing) => {
  let left = Infinity
  let top = Infinity
  let right = -Infinity
  let bottom = -Infinity

  for (const stroke of drawing.strokes) {
    const half = stroke.width / 2
    for (let i = 0; i < stroke.points.length; i += 2) {
      left = Math.min(left, stroke.points[i]! - half)
      right = Math.max(right, stroke.points[i]! + half)
      top = Math.min(top, stroke.points[i + 1]! - half)
      bottom = Math.max(bottom, stroke.points[i + 1]! + half)
    }
  }

  return { left, top, width: right - left, height: bottom - top }
}

const drawingStyle = (drawing: Drawing) => {
  const bounds = drawingBounds(drawing)
  return { left: bounds.left + 'px', top: bounds.top + 'px', width: bounds.width + 'px', height: bounds.height + 'px' }
}



const props = defineProps<{
  channelId: number,
  recipient: Planet,
}>()

const open = defineModel<boolean>({ default: false })
const emit = defineEmits<{ sent: [message: Message], failed: [error: string] }>()


watch(open, isOpen => {
  if (!isOpen) previewing.value = false
})

const validate = async () => {
  ;(document.activeElement as HTMLElement | null)?.blur()
  await nextTick()
  if (isEmpty.value) return

  selectedId.value = null
  tool.value = 'select'
  paletteOpen.value = false
  previewing.value = true
}

const send = async () => {
  ;(document.activeElement as HTMLElement | null)?.blur()
  await nextTick()
  if (isEmpty.value) return

  const content = serializeLetter(texts.value, drawings.value.flatMap(drawing => drawing.strokes), images.value)
  if (content.length > LETTER_MAX_LENGTH) {
    emit('failed', 'Lettre trop longue')
    return
  }

  sending.value = true
  try {
    const sent = await api.sendMessage(props.channelId, content, props.recipient.id)
    texts.value = []
    drawings.value = []
    images.value = []
    redraw()
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
