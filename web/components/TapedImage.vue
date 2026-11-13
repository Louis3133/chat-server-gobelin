<template>
  <div :style="{ top: image.top + 'px', left: image.left + 'px', width: image.width + 'px', height: image.height + 'px', transform: `rotate(${image.rotation}deg)` }" class="taped-image">
    <img :src="image.src" draggable="false" alt="" class="picture">
    <img :style="tape" src="/scotch.svg" alt="" class="scotch top-left">
    <img :style="tape" src="/scotch.svg" alt="" class="scotch bottom-right">
    <slot />
  </div>
</template>

<style scoped>

.taped-image {
  position: absolute;
  transform-origin: center;
}

.picture {
  display: block;
  width: 100%;
  height: 100%;
  user-select: none;
}

.scotch {
  position: absolute;
  width: var(--tape-width);
  height: var(--tape-height);
  pointer-events: none;
}

.scotch.top-left {
  top: calc(var(--tape-height) / -2);
  left: calc(var(--tape-width) / -2);
}

.scotch.bottom-right {
  right: calc(var(--tape-width) / -2);
  bottom: calc(var(--tape-height) / -2);
}

</style>

<script setup lang="ts">
const props = defineProps<{
  image: LetterImage,
}>()

// Scotch proportionnel à l'image, aux proportions du SVG (82 x 74)
const tape = computed(() => {
  const width = Math.sqrt(props.image.width * props.image.height) * 0.3
  return { '--tape-width': width + 'px', '--tape-height': width * 74 / 82 + 'px' }
})
</script>
