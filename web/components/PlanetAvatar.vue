<template>
  <span class="planet-avatar" :style="style" />
</template>

<script setup lang="ts">
const props = defineProps<{ planet?: Planet }>()
const api = useChatApi()

// Même rendu que la scène : texture teintée par la couleur, couleur tirée du nom à défaut
const DEFAULT_TEXTURE = '/textures/planets/rock_boulder_dry_diff_1k.jpg'

const colorFor = (name: string) => {
  let h = 0
  for (const c of name) h = (h * 31 + c.charCodeAt(0)) % 360
  return `hsl(${h} 60% 55%)`
}

const style = computed(() => {
  const p = props.planet
  if (!p) return { backgroundColor: '#d9d9d9' }
  const texture = p.texture_version ? api.textureUrl(p.id, p.texture_version) : DEFAULT_TEXTURE
  return {
    backgroundColor: p.color ?? colorFor(p.username),
    backgroundImage: `radial-gradient(circle at 35% 30%, rgba(255, 255, 255, 0.35), transparent 45%, rgba(0, 0, 0, 0.55) 90%), url('${texture}')`,
  }
})
</script>

<style scoped>
.planet-avatar {
  display: block;
  flex: none;
  width: 40px;
  height: 40px;
  background-position: center;
  background-size: 100% 100%, auto 100%;
  background-blend-mode: normal, multiply;
  image-rendering: pixelated;
  mask: url('/pixel-disc.svg') 0 0 / 100% 100%;
  -webkit-mask: url('/pixel-disc.svg') 0 0 / 100% 100%;
}
</style>
