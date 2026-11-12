<template>
  <div class="pixel-slider" :style="{ '--ratio': ratio }">
    <div class="track">
      <div class="fill" />
    </div>
    <span class="thumb" />
    <input v-model.number="model" type="range" :min="min" :max="max" :step="step" :aria-label="label">
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ min: number, max: number, step: number, label?: string }>()
const model = defineModel<number>({ required: true })

const ratio = computed(() => (model.value - props.min) / (props.max - props.min))
</script>

<style scoped>
.pixel-slider {
  --thumb: 28px;
  --track: 24px;
  --notch: 4px;
  position: relative;
  height: var(--thumb);
}

.track {
  position: absolute;
  top: calc((var(--thumb) - var(--track)) / 2);
  right: 0;
  left: 0;
  height: var(--track);
  background: #666;
  clip-path: polygon(
    0 var(--notch), var(--notch) var(--notch), var(--notch) 0,
    calc(100% - var(--notch)) 0, calc(100% - var(--notch)) var(--notch), 100% var(--notch),
    100% calc(100% - var(--notch)), calc(100% - var(--notch)) calc(100% - var(--notch)), calc(100% - var(--notch)) 100%,
    var(--notch) 100%, var(--notch) calc(100% - var(--notch)), 0 calc(100% - var(--notch))
  );
}

.fill {
  width: calc(var(--ratio) * (100% - var(--thumb)) + var(--thumb) / 2);
  height: 100%;
  background: var(--Zib);
}

.thumb {
  position: absolute;
  top: 0;
  left: calc(var(--ratio) * (100% - var(--thumb)));
  width: var(--thumb);
  height: var(--thumb);
  background: #d9d9d9;
  mask: url('/pixel-disc.svg') 0 0 / 100% 100%;
  -webkit-mask: url('/pixel-disc.svg') 0 0 / 100% 100%;
  pointer-events: none;
}

input {
  position: absolute;
  inset: 0;
  width: 100%;
  margin: 0;
  opacity: 0;
  cursor: pointer;
}
</style>
