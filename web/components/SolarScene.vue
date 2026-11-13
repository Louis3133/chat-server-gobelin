<template>
  <div ref="container" class="scene" :class="{ 'scene--overview': overview, 'scene--customizing': customizing }">
    <Teleport to="body"><div ref="tweaksEl" class="tweaks" /></Teleport>
  </div>
</template>

<script setup lang="ts">
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { CSS2DObject, CSS2DRenderer } from 'three/addons/renderers/CSS2DRenderer.js'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
import { Line2 } from 'three/addons/lines/Line2.js'
import { LineGeometry } from 'three/addons/lines/LineGeometry.js'
import { LineMaterial } from 'three/addons/lines/LineMaterial.js'
import { Pane, type FolderApi } from 'tweakpane'
import * as TweakpanePluginMedia from 'tweakpane-plugin-media'

// Textures (texels nets, façon DS)
const loadPixelTexture = (path: string, srgb = false) => {
  const t = new THREE.TextureLoader().load(path)
  t.magFilter = THREE.NearestFilter
  t.minFilter = THREE.NearestMipmapNearestFilter
  if (srgb) t.colorSpace = THREE.SRGBColorSpace
  return t
}
const texture = loadPixelTexture('/textures/planets/rock_boulder_dry_diff_1k.jpg', true)
const normalTexture = loadPixelTexture('/textures/planets/rock_boulder_dry_nor_gl_1k.jpg')
const armTexture = loadPixelTexture('/textures/planets/rock_boulder_dry_arm_1k.jpg')
const sunTexture = loadPixelTexture('/textures/planets/2k_venus_surface.jpg', true)

// Textures importées par les utilisateurs, chargées une fois par URL
const importedTextures = new Map<string, THREE.Texture>()
const textureAt = (url: string | null | undefined) => {
  if (!url) return texture
  if (!importedTextures.has(url)) importedTextures.set(url, loadPixelTexture(url, true))
  return importedTextures.get(url)!
}

const props = defineProps<{
  system: SolarSystem | null
  decor: SolarSystem | null
  channels: Channel[]
  messages: Message[]
  selectedId: number | null
  followedLetter: string | null
  openedLetter: string | null
  customizing: { id: number; look: PlanetLook; texture: string | null } | null
  focusArea: { top: number; bottom: number } | null
  overview: boolean
}>()
const emit = defineEmits<{ select: [id: number]; open: [channel: Channel]; sun: []; read: [key: string]; rocket: [key: string | null] }>()
const api = useChatApi()

const container = ref<HTMLDivElement>()
const tweaksEl = ref<HTMLDivElement>()

// Réglages Tweakpane (#debug dans l'URL)
const debug = useState<boolean>('debug', () => false).value
const tweaks = {
  sunSize: 1,
  pixelSize: 3,
  rocketSize: 0.4,
  smokeSize: 0.04,
  landing: 0.2,
  hearing: 8,
  music: MUSIC_VOLUME,
  rockets: 0.5,
  haloWidth: 0.28,
  haloColor: '#bfe6ff',
  haloOpacity: 0.35,
  background: '#05060f',
  orbitColor: '#445566',
  orbitWidth: 1.5,
  orbitOpacity: 0.5,
  starCount: 1500,
  starSize: 2,
  starColor: '#ffffff',
  starBrightness: 0.4,
  twinkleAmount: 1,
  twinkleSpeed: 1,
  shootingRate: 4,
  shootingSize: 2,
  shootingSpeed: 1,
}
let pane: Pane | undefined
let planetsFolder: FolderApi | undefined

// Apparence de chaque planète (#debug : réglable, gardée dans le navigateur)
const LOOKS_KEY = 'planet-looks'
interface Look {
  size: number
  color: string
  texture: THREE.Texture
  tilt: number
  rings: boolean
  ringColor: string
  ringSize: number
  spin: number
  moon: boolean
}
const looks = new Map<string, Look>()
const savedLooks = (): Record<string, Partial<Look>> => {
  try {
    return JSON.parse(localStorage.getItem(LOOKS_KEY) ?? '{}')
  } catch {
    return {}
  }
}
const lookValues = () => Object.fromEntries([...looks].map(([name, { texture, ...rest }]) => [name, rest]))
const saveLooks = () => localStorage.setItem(LOOKS_KEY, JSON.stringify(lookValues()))

// Caméras : distance et angle de chaque vue (#debug : réglables, gardés dans le navigateur)
const CAMERAS_KEY = 'camera-views'
interface CameraView {
  distance: number
  fixedAngle: boolean
  elevation: number
  azimuth: number
}
const cameraViews: Record<'overview' | 'channel' | 'planet' | 'rocket' | 'customize', CameraView> = {
  overview: { distance: 24, fixedAngle: true, elevation: 20, azimuth: 0 },
  channel: { distance: 24, fixedAngle: false, elevation: 66, azimuth: 0 },
  planet: { distance: 5, fixedAngle: false, elevation: 30, azimuth: 0 },
  rocket: { distance: 2.6, fixedAngle: false, elevation: 30, azimuth: 0 },
  customize: { distance: 7, fixedAngle: true, elevation: 5, azimuth: 0 },
}
if (debug) {
  try {
    const saved = JSON.parse(localStorage.getItem(CAMERAS_KEY) ?? '{}')
    for (const [name, view] of Object.entries(cameraViews)) Object.assign(view, saved[name])
  } catch {}
}
const CAMERA_LABELS = { overview: "Vue d'ensemble", channel: 'Salon', planet: 'Planète sélectionnée', rocket: 'Suivi de fusée', customize: 'Personnalisation' }
const viewDirection = (view: CameraView) =>
  new THREE.Vector3().setFromSphericalCoords(1, THREE.MathUtils.degToRad(90 - view.elevation), THREE.MathUtils.degToRad(view.azimuth))

// État de la scène
const DISTANCE_UNIT = 0.9
const SPIN_SPEED = 0.00002
const MIN_RADIUS = 3
const LOW_POLY = { planet: [10, 7], sun: [14, 9] } as const
const ROCKET_HIT_RADIUS = 0.6
const MAX_DISTANCE = 60
const SYSTEM_MARGIN = 20
const SUN_HIT_RADIUS = 10
const SUN_HOVER_SCALE = 2
const UP = new THREE.Vector3(0, 1, 0)
const ORIGIN = new THREE.Vector3()

interface PlanetNode {
  mesh: THREE.Mesh
  label: CSS2DObject
  key: string
  center: THREE.Vector3
  target: THREE.Vector3
  me: boolean
}

let renderer: THREE.WebGLRenderer
let labelRenderer: CSS2DRenderer
let scene: THREE.Scene
let camera: THREE.PerspectiveCamera
let controls: OrbitControls
let frame = 0
let observer: ResizeObserver
let downAt = { x: 0, y: 0 }
let adjustingZoom = false
let snapOverview = false
let ready = false
const planets = new Map<string, PlanetNode>()
const suns = new Map<string, THREE.Group>()
const orbits = new THREE.Group()
let activeKey = 'general'
const activeCenter = new THREE.Vector3()
const overviewCenter = new THREE.Vector3()
let overviewFit = 0
let stars: THREE.Points
const letters = new THREE.Group()

// Fusées des lettres : modèle chargé une fois, nez tourné vers +Z
const ROCKET_LENGTH = 0.7
let rocketModel: THREE.Object3D | null = null
let followedKey: string | null = null

// Fusée suivie qui explose : la caméra reste un instant sur l'explosion
const POOF_LINGER = 1500
const poofSpot = new THREE.Object3D()
let poofUntil = 0

const followedRocket = () =>
  (followedKey ? letters.children.find((d) => d.userData.key === followedKey) : undefined) ?? (Date.now() < poofUntil ? poofSpot : undefined)

const stopFollowing = () => {
  if (!followedKey) return
  followedKey = null
  adjustingZoom = true
  emit('rocket', null)
}

const rocketAt = (event: MouseEvent) => {
  const { ray } = rayAt(event)
  return letters.children
    .filter((d) => ray.distanceToPoint(d.position) < ROCKET_HIT_RADIUS * tweaks.rocketSize)
    .sort((a, b) => a.position.distanceTo(camera.position) - b.position.distanceTo(camera.position))[0]
}

// Nez : le cône rouge (Tip) pivote sur son bord, comme un clapet, pour révéler la lettre
const NOSE_HINGE = new THREE.Vector3(0.42, 0, 0.266)
const NOSE_OPEN = 1.9
const addNoseHinge = (model: THREE.Object3D) => {
  const tip = model.getObjectByName('Tip')
  if (!tip?.parent) return
  const hinge = new THREE.Group()
  hinge.name = 'NoseHinge'
  hinge.position.copy(NOSE_HINGE)
  tip.parent.add(hinge)
  hinge.add(tip)
  tip.position.sub(NOSE_HINGE)
}

const animateNose = (dot: THREE.Object3D) => {
  const hinge = dot.userData.hinge as THREE.Object3D | undefined
  if (!hinge) return
  const goal = dot.userData.key === props.openedLetter ? NOSE_OPEN : 0
  hinge.rotation.y = THREE.MathUtils.lerp(hinge.rotation.y, goal, 0.12)
}

// Seul le destinataire ouvre une fusée, et seulement une fois posée
const readable = (dot?: THREE.Object3D) => !!dot?.userData.toMe && Date.now() >= dot.userData.end

const loadRocket = () => {
  new GLTFLoader().load('/models/rocket/scene.gltf', (gltf) => {
    const model = gltf.scene
    model.traverse((o) => {
      if (!(o instanceof THREE.Mesh)) return
      const material = o.material as THREE.MeshStandardMaterial
      for (const map of [material.map, material.normalMap, material.roughnessMap, material.metalnessMap]) {
        if (!map) continue
        map.magFilter = THREE.NearestFilter
        map.minFilter = THREE.NearestMipmapNearestFilter
        map.needsUpdate = true
      }
    })
    addNoseHinge(model)
    const box = new THREE.Box3().setFromObject(model)
    model.position.sub(box.getCenter(new THREE.Vector3()))
    const pivot = new THREE.Group()
    pivot.add(model)
    pivot.rotation.x = Math.PI / 2
    pivot.scale.setScalar(ROCKET_LENGTH / box.getSize(new THREE.Vector3()).y)
    rocketModel = new THREE.Group().add(pivot)
    syncLetters()
  })
}

// Fumée : particules carrées, pixelisées par le rendu basse résolution
const SMOKE_MAX = 400
const SMOKE_LIFE = 1400
const SMOKE_EVERY = 45
const smokeOrigins = new Float32Array(SMOKE_MAX * 3)
const smokeDrift = new Float32Array(SMOKE_MAX * 3)
const smokeBorn = new Float64Array(SMOKE_MAX).fill(-Infinity)
const smokeShade = new Float32Array(SMOKE_MAX)
let smokeNext = 0
const smokeGeometry = new THREE.BufferGeometry()
smokeGeometry.setAttribute('position', new THREE.BufferAttribute(new Float32Array(SMOKE_MAX * 3), 3))
smokeGeometry.setAttribute('color', new THREE.BufferAttribute(new Float32Array(SMOKE_MAX * 4), 4))
const smoke = new THREE.Points(smokeGeometry, new THREE.PointsMaterial({
  size: tweaks.smokeSize,
  vertexColors: true,
  transparent: true,
  depthWrite: false,
}))
smoke.frustumCulled = false
const TAIL = new THREE.Vector3(0, 0, -ROCKET_LENGTH / 2)

// Vol : décollage depuis la surface, arc, puis vol stationnaire, retournement et atterrissage sur la queue
const ROCKET_ARC = 0.35
const ROCKET_LIFT = 0.8
const HOVER_AT = 0.85
const NOSE = new THREE.Vector3(0, 0, 1)
const landSpots = new Map<string, THREE.Quaternion>()

// Décalage propre à chaque lettre : deux fusées vers la même planète ne se posent pas au même endroit
const LAND_SPREAD = 0.9
const spreadOf = (key: string) => {
  let h = 0
  for (const c of key) h = (h * 31 + c.charCodeAt(0)) % 100000
  return new THREE.Vector3(rand(h, 7) - 0.5, (rand(h, 8) - 0.5) * 0.5, rand(h, 9) - 0.5).multiplyScalar(LAND_SPREAD)
}
const smooth = (t: number) => t * t * (3 - 2 * t)
const clamp01 = (t: number) => THREE.MathUtils.clamp(t, 0, 1)
const surfaceOf = (mesh: THREE.Mesh) => (mesh.geometry as THREE.SphereGeometry).parameters.radius * mesh.scale.x

const fly = (rocket: THREE.Object3D, from: THREE.Mesh, to: THREE.Mesh, progress: number) => {
  const half = (ROCKET_LENGTH / 2) * tweaks.rocketSize
  const landing = tweaks.landing
  const cruiseEnd = 1 - landing
  const dir = to.position.clone().sub(from.position).normalize()
  const launch = dir.clone().addScaledVector(UP, ROCKET_LIFT).normalize()

  // Point d'atterrissage fixe pendant la descente, accroché à la planète une fois posée : il tourne alors avec elle
  let land = dir.clone().negate().addScaledVector(UP, ROCKET_LIFT).add(spreadOf(rocket.userData.key)).normalize()
  let landed = new THREE.Quaternion().setFromUnitVectors(NOSE, land)
  if (progress >= 1) {
    const key = rocket.userData.key
    if (!landSpots.has(key)) landSpots.set(key, to.quaternion.clone().invert().multiply(landed))
    landed = to.quaternion.clone().multiply(landSpots.get(key)!)
    land = NOSE.clone().applyQuaternion(landed)
  }

  const start = from.position.clone().addScaledVector(launch, surfaceOf(from) + half)
  const end = to.position.clone().addScaledVector(land, surfaceOf(to) + half)
  const reach = start.distanceTo(end) * ROCKET_ARC
  const path = new THREE.CubicBezierCurve3(
    start,
    start.clone().addScaledVector(launch, reach),
    end.clone().addScaledVector(land, reach),
    end,
  )

  const descent = clamp01((progress - cruiseEnd) / Math.max(landing, 0.001))
  const s = progress < cruiseEnd
    ? HOVER_AT * smooth(progress / cruiseEnd)
    : HOVER_AT + (1 - HOVER_AT) * smooth(descent)
  rocket.position.copy(path.getPoint(s))
  const heading = new THREE.Quaternion().setFromUnitVectors(NOSE, path.getTangent(Math.min(s, 0.999)))
  rocket.quaternion.copy(heading).slerp(landed, smooth(clamp01(descent / 0.5)))
}

const puff = (rocket: THREE.Object3D, now: number) => {
  rocket.updateMatrixWorld()
  const at = rocket.localToWorld(TAIL.clone())
  const i = smokeNext
  smokeNext = (smokeNext + 1) % SMOKE_MAX
  smokeOrigins.set([at.x, at.y, at.z], i * 3)
  const drift = new THREE.Vector3().randomDirection().multiplyScalar(0.00012 + Math.random() * 0.0002)
  smokeDrift.set([drift.x, drift.y, drift.z], i * 3)
  smokeBorn[i] = now
  smokeShade[i] = 0.55 + Math.random() * 0.35
}

// Pouf : une bouffée de fumée qui part dans tous les sens, quand une fusée est retirée
const POOF_COUNT = 40
const poof = (at: THREE.Vector3, now: number) => {
  for (let n = 0; n < POOF_COUNT; n++) {
    const i = smokeNext
    smokeNext = (smokeNext + 1) % SMOKE_MAX
    smokeOrigins.set([at.x, at.y, at.z], i * 3)
    const drift = new THREE.Vector3().randomDirection().multiplyScalar((0.0004 + Math.random() * 0.0006) * tweaks.rocketSize)
    smokeDrift.set([drift.x, drift.y, drift.z], i * 3)
    smokeBorn[i] = now
    smokeShade[i] = 0.75 + Math.random() * 0.25
  }
}

const animateSmoke = (now: number) => {
  const positions = smokeGeometry.attributes.position as THREE.BufferAttribute
  const colors = smokeGeometry.attributes.color as THREE.BufferAttribute
  for (let i = 0; i < SMOKE_MAX; i++) {
    const age = now - smokeBorn[i]!
    if (age >= SMOKE_LIFE) {
      colors.setW(i, 0)
      continue
    }
    const t = age / SMOKE_LIFE
    positions.setXYZ(
      i,
      smokeOrigins[i * 3]! + smokeDrift[i * 3]! * age,
      smokeOrigins[i * 3 + 1]! + smokeDrift[i * 3 + 1]! * age,
      smokeOrigins[i * 3 + 2]! + smokeDrift[i * 3 + 2]! * age,
    )
    const shade = smokeShade[i]!
    if (t < 0.12) colors.setXYZ(i, 1, 0.65, 0.25)
    else colors.setXYZ(i, shade, shade, shade)
    colors.setW(i, Math.ceil((1 - t) * 4) / 4)
  }
  positions.needsUpdate = true
  colors.needsUpdate = true
}
// Halo pixelisé autour de la planète sélectionnée : anneau lumineux par paliers
const HALO_SIZE = 2.9
const HALO_STEPS = 4
const makeHaloTexture = (width: number) => {
  const n = 32
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = n
  const ctx = canvas.getContext('2d')!
  const image = ctx.createImageData(n, n)
  for (let y = 0; y < n; y++) {
    for (let x = 0; x < n; x++) {
      const d = Math.hypot(x + 0.5 - n / 2, y + 0.5 - n / 2) / (n / 2)
      const glow = d < 0.72 ? Math.max(0, 1 - (0.72 - d) / 0.08) : Math.max(0, 1 - (d - 0.72) / width)
      const alpha = Math.round(glow * HALO_STEPS) / HALO_STEPS
      image.data.set([255, 255, 255, alpha * 255], (y * n + x) * 4)
    }
  }
  ctx.putImageData(image, 0, 0)
  const texture = new THREE.CanvasTexture(canvas)
  texture.magFilter = THREE.NearestFilter
  texture.minFilter = THREE.NearestFilter
  return texture
}
const selectionHalo = new THREE.Sprite(new THREE.SpriteMaterial({
  map: makeHaloTexture(tweaks.haloWidth),
  color: tweaks.haloColor,
  opacity: tweaks.haloOpacity,
  transparent: true,
  blending: THREE.AdditiveBlending,
  depthWrite: false,
}))
selectionHalo.visible = false

// Survol d'une planète : même halo, plus discret
const HOVER_HALO_OPACITY = 0.5
const hoverHalo = new THREE.Sprite()
hoverHalo.visible = false
const syncHoverHalo = () => {
  hoverHalo.material.dispose()
  hoverHalo.material = selectionHalo.material.clone()
  hoverHalo.material.opacity = tweaks.haloOpacity * HOVER_HALO_OPACITY
}
syncHoverHalo()

const updateHalo = () => {
  const material = selectionHalo.material
  material.map?.dispose()
  material.map = makeHaloTexture(tweaks.haloWidth)
  material.color.set(tweaks.haloColor)
  material.opacity = tweaks.haloOpacity
  material.needsUpdate = true
  syncHoverHalo()
}

const hoverRing = new THREE.Mesh(
  new THREE.RingGeometry(0.85, 1, 48),
  new THREE.MeshBasicMaterial({ color: 0xffd54f, side: THREE.DoubleSide }),
)
let hoveredSun: THREE.Group | undefined
let hoveredPlanet: string | null = null

// Utilitaires
const colorFor = (name: string) => {
  let h = 0
  for (const c of name) h = (h * 31 + c.charCodeAt(0)) % 360
  return new THREE.Color().setHSL(h / 360, 0.6, 0.55)
}

// Noms en HTML : nets malgré la pixelisation
const makeLabel = (text: string, big = false) => {
  const el = document.createElement('div')
  el.className = big ? 'planet-label planet-label--sun' : 'planet-label'
  el.textContent = text
  return new CSS2DObject(el)
}

// Nom de la planète, pseudo en dessous
const setPlanetLabel = (el: HTMLElement, p: Planet) => {
  const username = p.me ? `${p.username} (moi)` : p.username
  const name = p.planet_name?.trim()
  el.classList.toggle('planet-label--me', p.me)
  if (el.dataset.name === name && el.dataset.username === username) return
  el.dataset.name = name ?? ''
  el.dataset.username = username
  const title = document.createElement('span')
  title.className = 'planet-label__name'
  title.textContent = name || username
  el.replaceChildren(title)
  if (!name) return
  const sub = document.createElement('span')
  sub.className = 'planet-label__username'
  sub.textContent = username
  el.append(sub)
}

const BASE_SIZE = 0.55

const lookFor = (p: Planet) => {
  const name = p.username
  let look = looks.get(name)
  if (!look) {
    look = {
      size: p.size ?? BASE_SIZE,
      color: p.color ?? `#${colorFor(name).getHexString()}`,
      texture: p.texture_version ? textureAt(api.textureUrl(p.id, p.texture_version)) : texture,
      tilt: p.tilt ?? 0,
      rings: p.rings ?? false,
      ringColor: p.ring_color ?? '#d9c9a3',
      ringSize: p.ring_size ?? 2,
      spin: p.spin ?? 0.01,
      moon: p.moon ?? false,
      ...(debug ? savedLooks()[name] : {}),
    }
    looks.set(name, look)
    addLookFolder(name, look)
  }
  return look
}

// Lune : enfant de la planète, donc proportionnelle à sa taille et dans son plan incliné
const MOON_SIZE = 0.27
const MOON_DISTANCE = 2.2
const MOON_SPEED = 0.0008
const applyMoon = (mesh: THREE.Mesh, look: Look) => {
  let moon = mesh.userData.moon as THREE.Group | undefined
  if (look.moon && !moon) {
    const r = (mesh.geometry as THREE.SphereGeometry).parameters.radius
    const body = new THREE.Mesh(
      new THREE.SphereGeometry(r * MOON_SIZE, 8, 6),
      new THREE.MeshStandardMaterial({ flatShading: true, map: texture, color: '#b8b8c0' }),
    )
    body.position.x = r * MOON_DISTANCE
    moon = new THREE.Group().add(body)
    mesh.userData.moon = moon
    mesh.add(moon)
  }
  if (moon) moon.visible = look.moon
}

// Orbite de la lune : on retire la rotation propre de la planète, la lune garde sa vitesse
const orbitMoon = (mesh: THREE.Mesh, now: number) => {
  const moon = mesh.userData.moon as THREE.Group | undefined
  if (moon?.visible) moon.rotation.y = now * MOON_SPEED - mesh.rotation.y
}

const applyLook = (mesh: THREE.Mesh, look: Look) => {
  const material = mesh.material as THREE.MeshStandardMaterial
  material.color.set(look.color)
  if (material.map !== look.texture) {
    look.texture.magFilter = THREE.NearestFilter
    look.texture.minFilter = THREE.NearestMipmapNearestFilter
    material.map = look.texture
    material.needsUpdate = true
  }
  mesh.scale.setScalar(look.size / BASE_SIZE)
  mesh.rotation.z = look.tilt
  applyMoon(mesh, look)
  let ring = mesh.userData.ring as THREE.Mesh | undefined
  if (look.rings && !ring) {
    ring = new THREE.Mesh(
      new THREE.BufferGeometry(),
      new THREE.MeshStandardMaterial({ side: THREE.DoubleSide, flatShading: true }),
    )
    ring.rotation.x = Math.PI / 2
    mesh.userData.ring = ring
    mesh.add(ring)
  }
  if (!ring) return
  ring.visible = look.rings
  ;(ring.material as THREE.MeshStandardMaterial).color.set(look.ringColor)
  if (ring.userData.size !== look.ringSize) {
    const r = (mesh.geometry as THREE.SphereGeometry).parameters.radius
    ring.geometry.dispose()
    ring.geometry = new THREE.RingGeometry(r * 1.3, r * look.ringSize, 32)
    ring.userData.size = look.ringSize
  }
}

// Personnalisation de sa planète : aperçu en direct, sur sa planète de general
const customizedNode = () => (props.customizing ? planets.get(`general:${props.customizing.id}`) : undefined)
const previewLook = () => {
  const node = customizedNode()
  if (!node) return
  const { look: l, texture: url } = props.customizing!
  const name = node.mesh.userData.username
  Object.assign(looks.get(name)!, {
    texture: textureAt(url),
    size: l.planet_size,
    color: l.planet_color,
    tilt: l.planet_tilt,
    rings: l.planet_rings,
    ringColor: l.planet_ring_color,
    ringSize: l.planet_ring_size,
    spin: l.planet_spin,
    moon: l.planet_moon,
  })
  refreshLook(name)
}

const refreshLook = (name: string) => {
  const look = looks.get(name)!
  for (const node of planets.values()) if (node.mesh.userData.username === name) applyLook(node.mesh, look)
}

const addLookFolder = (name: string, look: Look) => {
  if (!planetsFolder) return
  const folder = planetsFolder.addFolder({ title: name, expanded: false })
  folder.addBinding(look, 'size', { label: 'Taille', min: 0.2, max: 2, step: 0.01 })
  folder.addBinding(look, 'color', { label: 'Couleur' })
  folder.addBinding(look, 'texture', { label: 'Texture', view: 'texture', height: 80, showMonitor: true })
  folder.addBinding(look, 'tilt', { label: 'Inclinaison', min: -1.2, max: 1.2, step: 0.05 })
  folder.addBinding(look, 'rings', { label: 'Anneau' })
  folder.addBinding(look, 'ringColor', { label: 'Couleur anneau' })
  folder.addBinding(look, 'ringSize', { label: 'Taille anneau', min: 1.5, max: 4, step: 0.05 })
  folder.addBinding(look, 'moon', { label: 'Lune' })
  folder.addBinding(look, 'spin', { label: 'Rotation', min: -0.1, max: 0.1, step: 0.001 })
  folder.on('change', () => {
    refreshLook(name)
    saveLooks()
  })
}

const removeFromScene = (obj: THREE.Object3D) => {
  obj.traverse((o) => { if (o instanceof CSS2DObject) o.element.remove() })
  scene.remove(obj)
}

const rand = (id: number, salt: number) => {
  const x = Math.sin(id * 12.9898 + salt * 78.233) * 43758.5453
  return x - Math.floor(x)
}

const place = (radius: number, angle: number, tilt: number) => {
  const z = Math.sin(angle) * radius
  return new THREE.Vector3(Math.cos(angle) * radius, z * Math.sin(tilt), z * Math.cos(tilt))
}

const spinAt = (now: number) => now * SPIN_SPEED

const progressOf = (start: number, end: number, now: number) =>
  Math.min(1, Math.max(0, (now - start) / Math.max(1, end - start)))

// Sons des fusées : décollage, vol, atterrissage (fini au moment du contact), plus forts quand la caméra est proche
const sound = useSound()
const TAKEOFF_MS = 5500
const LANDING_MS = 6000
const rocketPhases = new Map<string, string>()
const rocketVoices = new Map<string, { setVolume: (volume: number) => void }>()

const rocketSounds = (now: number) => {
  let flight = 0
  const seen = new Set<string>()
  for (const dot of letters.children) {
    const { key, start, end } = dot.userData
    const phase = now >= end ? 'landed' : end - now <= LANDING_MS ? 'landing' : now - start < TAKEOFF_MS ? 'takeoff' : 'flight'
    const near = clamp01(1 - camera.position.distanceTo(dot.position) / tweaks.hearing) * tweaks.rockets
    if (phase !== rocketPhases.get(key)) {
      rocketVoices.delete(key)
      if (phase === 'takeoff') rocketVoices.set(key, sound.play('decollage', near, (now - start) / 1000))
      if (phase === 'landing') rocketVoices.set(key, sound.play('atterrissage', near, (now - end + LANDING_MS) / 1000))
    }
    rocketVoices.get(key)?.setVolume(near)
    if (phase === 'flight') flight = Math.max(flight, near)
    rocketPhases.set(key, phase)
    seen.add(key)
  }
  for (const key of rocketPhases.keys()) {
    if (seen.has(key)) continue
    rocketPhases.delete(key)
    rocketVoices.get(key)?.setVolume(0)
    rocketVoices.delete(key)
  }
  if (flight > 0) sound.loop('fusee', flight)
  else sound.stop('fusee')
}

// Mise en page : place fixe par id, les proches viennent à côté de moi
const besideMe = (mine: { radius: number; angle: number }, distance: number, side: number, id: number) => {
  const d = distance * DISTANCE_UNIT
  const at = (dir: number) => {
    const x = Math.cos(mine.angle) * mine.radius + Math.cos(dir) * d
    const z = Math.sin(mine.angle) * mine.radius + Math.sin(dir) * d
    return { radius: Math.hypot(x, z), angle: Math.atan2(z, x) }
  }
  const spot = at(mine.angle + side * (Math.PI / 2 + (rand(id, 3) - 0.5) * 1.2))
  return spot.radius >= MIN_RADIUS ? spot : at(mine.angle)
}

const layout = (sys: SolarSystem) => {
  const out = new Map<number, { pos: THREE.Vector3; radius: number; tilt: number }>()
  const sorted = [...sys.planets].sort((a, b) => a.id - b.id)
  const slots = sorted.map((p, i) => ({
    p,
    radius: 4 + i * 1.4,
    angle: i * 2.399963,
    tilt: (rand(p.id, 2) - 0.5) * 0.2,
  }))
  const mine = slots.find((s) => s.p.me)
  slots.forEach((s, i) => {
    let { radius, angle, tilt } = s
    if (mine && !s.p.me && s.p.distance != null && s.p.distance < sys.max_distance) {
      ;({ radius, angle } = besideMe(mine, s.p.distance, i % 2 ? -1 : 1, s.p.id))
      tilt = mine.tilt
    }
    out.set(s.p.id, { pos: place(radius, angle, tilt), radius, tilt })
  })
  return out
}

// Salons : un soleil chacun, general au centre, chacun peuplé de ses membres
const keyOf = (c: Channel) => (c.id === 0 || c.name === 'general' ? 'general' : `c${c.id}`)

const centers = () => {
  const all = [...props.channels]
  if (props.system && !all.some((c) => c.id === props.system!.channel.id)) all.push(props.system.channel)
  const out = new Map<string, { channel: Channel | null; pos: THREE.Vector3 }>([['general', { channel: null, pos: new THREE.Vector3() }]])
  all
    .filter((c) => keyOf(c) !== 'general')
    .sort((a, b) => a.id - b.id)
    .forEach((c, i) => {
      const angle = 0.8 + i * 2.399963
      const dist = 90 + i * 15
      out.set(keyOf(c), {
        channel: c,
        pos: new THREE.Vector3(Math.cos(angle) * dist, (rand(c.id, 4) - 0.5) * 30, Math.sin(angle) * dist),
      })
    })
  return out
}

const makeSun = (name: string | null) => {
  const group = new THREE.Group()
  group.add(new THREE.Mesh(
    new THREE.SphereGeometry(1.6, ...LOW_POLY.sun),
    new THREE.MeshStandardMaterial({ map: sunTexture, flatShading: true }),
  ))
  group.add(new THREE.PointLight(0xffffff, 200))
  const hit = new THREE.Mesh(new THREE.SphereGeometry(SUN_HIT_RADIUS, 8, 6), new THREE.MeshBasicMaterial({ visible: false }))
  hit.userData.sun = group
  group.add(hit)
  if (name) {
    const label = makeLabel(name, true)
    label.position.set(0, 3.5, 0)
    label.element.addEventListener('click', () => openSun(group))
    label.element.addEventListener('mouseenter', () => { if (props.overview) setHovered(group) })
    label.element.addEventListener('mouseleave', () => setHovered(undefined))
    group.userData.label = label.element
    group.add(label)
  }
  return group
}

// Vue d'ensemble : survol et clic d'un soleil pour choisir son salon
const setHovered = (group?: THREE.Group) => {
  if (hoveredSun === group) return
  hoveredSun?.userData.label?.classList.remove('planet-label--hover')
  hoveredSun = group
  hoveredSun?.userData.label?.classList.add('planet-label--hover')
  renderer.domElement.style.cursor = group ? 'pointer' : ''
}

// Dans un salon : survol et clic d'une planète ou de son nom
const planetsInteractive = () => !props.overview && !props.customizing

const setHoveredPlanet = (k: string | null) => {
  if (!planetsInteractive()) k = null
  if (hoveredPlanet === k) return
  if (hoveredPlanet) planets.get(hoveredPlanet)?.label.element.classList.remove('planet-label--pointed')
  hoveredPlanet = k
  if (k) planets.get(k)?.label.element.classList.add('planet-label--pointed')
}

const selectPlanet = (k: string) => {
  const node = planets.get(k)
  if (!node || !planetsInteractive()) return
  stopFollowing()
  emit('select', node.mesh.userData.id)
}

const planetAt = (event: MouseEvent) => {
  const active = [...planets.entries()].filter(([, n]) => n.key === activeKey)
  const hit = rayAt(event).intersectObjects(active.map(([, n]) => n.mesh), false)[0]
  return active.find(([, n]) => n.mesh === hit?.object)?.[0] ?? null
}

const openSun = (group: THREE.Object3D) => {
  if (props.overview && !props.customizing && group.userData.channel) emit('open', group.userData.channel)
}

const rayAt = (event: MouseEvent) => {
  const rect = renderer.domElement.getBoundingClientRect()
  const pointer = new THREE.Vector2(
    ((event.clientX - rect.left) / rect.width) * 2 - 1,
    -((event.clientY - rect.top) / rect.height) * 2 + 1,
  )
  const ray = new THREE.Raycaster()
  ray.setFromCamera(pointer, camera)
  return ray
}

const sunAt = (event: MouseEvent) => {
  const hits = [...suns.values()].flatMap((g) => g.children.filter((o) => o.userData.sun))
  return rayAt(event).intersectObjects(hits, false)[0]?.object.userData.sun as THREE.Group | undefined
}

// Dans un salon, le soleil ouvre la fiche du salon (et ramène à la vue du salon)
const activeSunAt = (event: MouseEvent) => {
  const sun = suns.get(activeKey)?.children[0]
  return !!sun && rayAt(event).intersectObject(sun, false).length > 0
}

// Orbites : une matière partagée, épaisseur en pixels
const orbitMaterial = new LineMaterial({ transparent: true, depthWrite: false })
const updateOrbits = () => {
  orbitMaterial.color.set(tweaks.orbitColor)
  orbitMaterial.linewidth = tweaks.orbitWidth
  orbitMaterial.opacity = tweaks.orbitOpacity
}
updateOrbits()

// Synchronisation avec les données
const syncSystem = () => {
  if (!ready) return
  const where = centers()
  activeKey = props.system ? keyOf(props.system.channel) : 'general'
  activeCenter.copy(where.get(activeKey)?.pos ?? ORIGIN)
  fitOverview([...where.values()].map((w) => w.pos))
  if (snapOverview) placeOverview()

  for (const [key, group] of suns) {
    if (!where.has(key)) {
      removeFromScene(group)
      suns.delete(key)
    }
  }
  for (const [key, { channel, pos }] of where) {
    if (!suns.has(key)) {
      const group = makeSun(channel?.name ?? 'general')
      scene.add(group)
      suns.set(key, group)
    }
    suns.get(key)!.position.copy(pos)
    suns.get(key)!.userData.channel = channel ?? props.channels.find((c) => keyOf(c) === key) ?? null
  }

  const shown: { key: string; sys: SolarSystem }[] = []
  const general = activeKey === 'general' && props.system ? props.system : props.decor
  if (general) shown.push({ key: 'general', sys: general })
  if (props.system && activeKey !== 'general') shown.push({ key: activeKey, sys: props.system })
  for (const [key, { channel }] of where) {
    if (!channel || shown.some((s) => s.key === key)) continue
    const members = new Set(channel.member_ids ?? [])
    const planets = props.decor?.planets.filter((p) => members.has(p.id)) ?? []
    if (planets.length) shown.push({ key, sys: { ...props.decor!, channel, planets } })
  }

  const wanted = new Set(shown.flatMap(({ key, sys }) => sys.planets.map((p) => `${key}:${p.id}`)))
  for (const [k, node] of planets) {
    if (!wanted.has(k)) {
      removeFromScene(node.mesh)
      planets.delete(k)
    }
  }
  orbits.clear()
  for (const { key, sys } of shown) {
    const center = where.get(key)?.pos ?? ORIGIN
    const rings = new THREE.Group()
    rings.position.copy(center)
    orbits.add(rings)
    const targets = layout(sys)
    for (const p of sys.planets) syncPlanet(key, center, p, targets.get(p.id)!, rings)
  }
}

const syncPlanet = (
  key: string,
  center: THREE.Vector3,
  p: Planet,
  t: { pos: THREE.Vector3; radius: number; tilt: number },
  rings: THREE.Group,
) => {
  const k = `${key}:${p.id}`
  let node = planets.get(k)
  const from = node && node.me !== p.me ? node.mesh.position.clone() : null
  if (node && from) {
    removeFromScene(node.mesh)
    planets.delete(k)
    node = undefined
  }
  if (!node) {
    const mesh = new THREE.Mesh(
      new THREE.SphereGeometry(p.me ? 0.7 : BASE_SIZE, ...LOW_POLY.planet),
      new THREE.MeshStandardMaterial({
        flatShading: true,
        map: texture,
        normalMap: normalTexture,
        aoMap: armTexture,
        roughnessMap: armTexture,
        metalnessMap: armTexture,
        color: colorFor(p.username),
      }),
    )
    mesh.userData.id = p.id
    mesh.userData.username = p.username
    mesh.rotation.order = 'ZYX'
    applyLook(mesh, lookFor(p))
    mesh.position.copy(from ?? t.pos.clone().applyAxisAngle(UP, spinAt(Date.now())).add(center))
    const label = makeLabel('')
    label.center.set(0.5, 1)
    label.position.set(0, 1.1, 0)
    label.element.addEventListener('mouseenter', () => setHoveredPlanet(k))
    label.element.addEventListener('mouseleave', () => { if (hoveredPlanet === k) setHoveredPlanet(null) })
    label.element.addEventListener('click', () => selectPlanet(k))
    mesh.add(label)
    scene.add(mesh)
    node = { mesh, label, key, center: center.clone(), target: t.pos.clone(), me: p.me }
    planets.set(k, node)
  }
  setPlanetLabel(node.label.element, p)
  node.target.copy(t.pos)
  const points = new THREE.Path().absarc(0, 0, t.radius, 0, Math.PI * 2).getSpacedPoints(128)
  const ring = new Line2(new LineGeometry().setPositions(points.flatMap((v) => [v.x, v.y, 0])), orbitMaterial)
  ring.rotation.x = Math.PI / 2 - t.tilt
  rings.add(ring)
}

// Lettres en transit
const LANDED_MAX = 30

const syncLetters = () => {
  if (!ready) return
  const removed = new Set(props.messages.filter((m) => m.rocket_removed).map(letterKey))
  for (const dot of letters.children) {
    if (!removed.has(dot.userData.key)) continue
    poof(dot.position, Date.now())
    if (dot.userData.key === followedKey) {
      poofSpot.position.copy(dot.position)
      poofUntil = Date.now() + POOF_LINGER
    }
  }
  letters.clear()
  const meId = props.system?.planets.find((p) => p.me)?.id
  const sent = props.messages.filter((m) => m.recipient_id && !m.rocket_removed)
  const landed = sent
    .filter((m) => !m.in_transit)
    .sort((a, b) => new Date(b.deliver_at).getTime() - new Date(a.deliver_at).getTime())
    .slice(0, LANDED_MAX)
  for (const m of [...sent.filter((m) => m.in_transit), ...landed]) {
    const from = planets.get(`${activeKey}:${m.user_id}`)
    const to = planets.get(`${activeKey}:${m.recipient_id}`)
    if (!from || !to) continue
    const dot = rocketModel?.clone() ?? new THREE.Mesh(new THREE.SphereGeometry(0.2, 12, 12), new THREE.MeshBasicMaterial({ color: 0xffd54f }))
    dot.userData = {
      key: letterKey(m),
      lastPuff: 0,
      from: from.mesh,
      to: to.mesh,
      start: new Date(m.creation_date).getTime(),
      end: new Date(m.deliver_at).getTime(),
      toMe: m.recipient_id === meId,
      hinge: dot.getObjectByName('NoseHinge'),
    }
    if (dot.userData.hinge && dot.userData.key === props.openedLetter) dot.userData.hinge.rotation.y = NOSE_OPEN
    letters.add(dot)
  }
}

// Vue d'ensemble : cadre tous les soleils depuis le dessus
const fitOverview = (points: THREE.Vector3[]) => {
  const sphere = new THREE.Sphere().setFromPoints(points)
  overviewCenter.copy(sphere.center)
  const halfFov = THREE.MathUtils.degToRad(camera.fov / 2)
  const narrowest = Math.min(halfFov, Math.atan(Math.tan(halfFov) * camera.aspect))
  overviewFit = (sphere.radius + SYSTEM_MARGIN) / Math.sin(narrowest)
}

// Écran en hauteur : on recule pour que le salon tienne en largeur
const channelDistance = () => cameraViews.channel.distance / Math.min(1, camera.aspect)
const overviewDistance = () => Math.max(cameraViews.overview.distance, overviewFit)

// Arrivée sur le site : la caméra est posée d'emblée sur la vue d'ensemble, sans glisser
const placeOverview = () => {
  controls.target.copy(overviewCenter)
  camera.position.copy(overviewCenter).add(viewDirection(cameraViews.overview).multiplyScalar(overviewDistance()))
  adjustingZoom = false
}

// Caméra : suit la sélection, le soleil du salon ouvert ou l'ensemble des salons
// Personnalisation : on tourne autour de sa planète, sans zoom ni déplacement
const lockControls = () => {
  controls.enableZoom = !props.customizing
  controls.enablePan = !props.customizing
}

// Personnalisation : la plus grande planète possible remplit une part de l'espace libre entre les panneaux
const CUSTOMIZE_FILL = 0.6
const customizeDistance = (node: PlanetNode) => {
  const area = props.focusArea
  const h = container.value?.clientHeight ?? 0
  if (!area || !h) return cameraViews.customize.distance
  const largest = (node.mesh.geometry as THREE.SphereGeometry).parameters.radius / BASE_SIZE
  const free = Math.max(area.bottom - area.top, 1)
  return (largest * h) / (CUSTOMIZE_FILL * free * Math.tan(THREE.MathUtils.degToRad(camera.fov / 2)))
}

let lastCustomized: THREE.Vector3 | null = null

const followSelection = (selected?: PlanetNode, rocket?: THREE.Object3D, customized?: PlanetNode) => {
  const wanted = customized ? customized.mesh.position : rocket ? rocket.position : selected ? selected.mesh.position : props.overview ? overviewCenter : activeCenter
  if (customized && lastCustomized) {
    const step = wanted.clone().sub(lastCustomized)
    controls.target.add(step)
    camera.position.add(step)
  }
  lastCustomized = customized ? wanted.clone() : null
  const before = controls.target.clone()
  controls.target.lerp(wanted, 0.04)
  camera.position.add(controls.target.clone().sub(before))

  if (adjustingZoom) {
    const offset = camera.position.clone().sub(controls.target)
    const view = customized ? cameraViews.customize : rocket ? cameraViews.rocket : selected ? cameraViews.planet : props.overview ? cameraViews.overview : cameraViews.channel
    const goal = customized ? customizeDistance(customized) : rocket || selected ? view.distance : props.overview ? overviewDistance() : channelDistance()
    const wantedDir = view.fixedAngle ? viewDirection(view) : null
    const next = THREE.MathUtils.lerp(offset.length(), goal, 0.05)
    const dir = offset.normalize()
    if (wantedDir) dir.lerp(wantedDir, 0.05).normalize()
    camera.position.copy(controls.target).add(dir.multiplyScalar(next))
    if (Math.abs(next - goal) < 0.05 && (!wantedDir || dir.angleTo(wantedDir) < 0.01)) adjustingZoom = false
  } else if (customized) {
    const offset = camera.position.clone().sub(controls.target)
    offset.setLength(THREE.MathUtils.lerp(offset.length(), customizeDistance(customized), 0.05))
    camera.position.copy(controls.target).add(offset)
  }
  const farthest = Math.max(MAX_DISTANCE, channelDistance() * 1.2)
  controls.maxDistance = props.overview || adjustingZoom ? Math.max(farthest, overviewDistance() * 1.2) : farthest
}

// Mobile : la planète ou la fusée suivie remonte pour laisser la place au panneau du bas
// Personnalisation : la planète se centre dans l'espace libre entre les panneaux
const MOBILE_WIDTH = 600
const MOBILE_LIFT = 0.22
let viewLift = 0

const liftView = (focused: boolean, customizing: boolean) => {
  const el = container.value
  if (!el) return
  const { clientWidth: w, clientHeight: h } = el
  const area = customizing ? props.focusArea : null
  const goal = area ? 0.5 - (area.top + area.bottom) / 2 / h : focused && w < MOBILE_WIDTH ? MOBILE_LIFT : 0
  viewLift = Math.abs(goal - viewLift) < 0.001 ? goal : THREE.MathUtils.lerp(viewLift, goal, 0.06)
  if (viewLift) camera.setViewOffset(w, h, 0, viewLift * h, w, h)
  else camera.clearViewOffset()
}

// Étiquettes cachées par un objet 3D plus proche
const occlusion = new THREE.Raycaster()
const labelAt = new THREE.Vector3()

const isOwnedBy = (object: THREE.Object3D, owner: THREE.Object3D) => {
  for (let o: THREE.Object3D | null = object; o; o = o.parent) if (o === owner) return true
  return false
}

const hideCoveredLabels = () => {
  const visible = [...planets.values()].filter((n) => n.label.visible)
  if (!visible.length) return
  const sun = suns.get(activeKey)?.children[0]
  const blockers = [...visible.map((n) => n.mesh), ...letters.children, ...(sun ? [sun] : [])]
  for (const node of visible) {
    node.label.getWorldPosition(labelAt)
    const distance = labelAt.distanceTo(camera.position)
    occlusion.set(camera.position, labelAt.sub(camera.position).normalize())
    occlusion.far = distance - 0.05
    const covered = occlusion.intersectObjects(blockers, true).some((hit) => hit.object instanceof THREE.Mesh && !isOwnedBy(hit.object, node.mesh))
    node.label.element.classList.toggle('planet-label--covered', covered)
  }
}

// Boucle d'animation
const animate = () => {
  frame = requestAnimationFrame(animate)
  const now = Date.now()
  const spin = spinAt(now)
  for (const rings of orbits.children) rings.rotation.y = spin
  for (const node of planets.values()) {
    node.mesh.position.lerp(node.target.clone().applyAxisAngle(UP, spin).add(node.center), 0.05)
    node.mesh.rotation.y += looks.get(node.mesh.userData.username)?.spin ?? 0.01
    orbitMoon(node.mesh, now)
    node.label.visible = !props.overview && node.key === activeKey
  }
  for (const dot of letters.children) {
    const { from, to, start, end } = dot.userData
    const progress = progressOf(start, end, now)
    dot.scale.setScalar(tweaks.rocketSize)
    fly(dot, from, to, progress)
    animateNose(dot)
    if (rocketModel && progress < 1 && now - dot.userData.lastPuff >= SMOKE_EVERY) {
      dot.userData.lastPuff = now
      puff(dot, now)
    }
  }
  rocketSounds(now)
  ;(smoke.material as THREE.PointsMaterial).size = tweaks.smokeSize
  animateSmoke(now)
  const selected = props.selectedId != null ? planets.get(`${activeKey}:${props.selectedId}`) : undefined
  selectionHalo.visible = !!selected
  if (selected) {
    selectionHalo.position.copy(selected.mesh.position)
    selectionHalo.scale.setScalar(surfaceOf(selected.mesh) * HALO_SIZE * (1 + Math.sin(now / 400) * 0.02))
  }
  if (hoveredPlanet && (!planets.has(hoveredPlanet) || !planetsInteractive())) setHoveredPlanet(null)
  const hovered = hoveredPlanet ? planets.get(hoveredPlanet) : undefined
  hoverHalo.visible = !!hovered && hovered !== selected
  if (hovered) {
    hoverHalo.position.copy(hovered.mesh.position)
    hoverHalo.scale.setScalar(surfaceOf(hovered.mesh) * HALO_SIZE)
  }
  for (const [key, group] of suns) {
    const sun = group.children[0]
    const goal = (key === 'general' ? tweaks.sunSize : 1) * (group === hoveredSun ? SUN_HOVER_SCALE : 1)
    sun.scale.setScalar(THREE.MathUtils.lerp(sun.scale.x, goal, 0.15))
  }
  hoverRing.visible = !!hoveredSun
  if (hoveredSun) {
    hoverRing.position.copy(hoveredSun.position)
    hoverRing.quaternion.copy(camera.quaternion)
    hoverRing.scale.setScalar(camera.position.distanceTo(hoveredSun.position) * 0.03 * (1 + Math.sin(now / 200) * 0.08))
  }
  const rocket = followedRocket()
  if (followedKey && !rocket) stopFollowing()
  followSelection(selected, rocket, customizedNode())
  liftView(!!(selected || rocket || customizedNode()), !!customizedNode())
  controls.update()
  hideCoveredLabels()
  stars.position.copy(camera.position)
  shootingStars.position.copy(camera.position)
  animateShootingStars(now)
  starMaterial.uniforms.time.value = performance.now() / 1000
  renderer.render(scene, camera)
  labelRenderer.render(scene, camera)
}

// Interaction
const resize = () => {
  const el = container.value
  if (!el) return
  renderer.setPixelRatio(1 / tweaks.pixelSize)
  renderer.setSize(el.clientWidth, el.clientHeight)
  const buffer = renderer.getDrawingBufferSize(new THREE.Vector2())
  starMaterial.uniforms.scale.value = buffer.y / 2
  orbitMaterial.resolution.copy(buffer)
  labelRenderer.setSize(el.clientWidth, el.clientHeight)
  camera.aspect = el.clientWidth / el.clientHeight
  camera.updateProjectionMatrix()
}

const onPointerDown = (e: PointerEvent) => { downAt = { x: e.clientX, y: e.clientY } }

const onPointerMove = (event: PointerEvent) => {
  if (props.customizing) return
  setHovered(props.overview ? sunAt(event) : undefined)
  if (props.overview) return
  const rocket = rocketAt(event)
  setHoveredPlanet(rocket ? null : planetAt(event))
  renderer.domElement.style.cursor = hoveredPlanet || activeSunAt(event) || readable(rocket) ? 'pointer' : ''
}

const onClick = (event: MouseEvent) => {
  if (props.customizing || Math.hypot(event.clientX - downAt.x, event.clientY - downAt.y) > 5) return
  if (props.overview) {
    const sun = sunAt(event)
    if (sun) openSun(sun)
    return
  }
  const rocket = rocketAt(event)
  if (rocket) {
    followedKey = rocket.userData.key
    adjustingZoom = true
    emit('rocket', rocket.userData.key)
    if (readable(rocket)) emit('read', rocket.userData.key)
    return
  }
  const planet = planetAt(event)
  if (planet) return selectPlanet(planet)
  stopFollowing()
  if (activeSunAt(event)) emit('sun')
}

// Étoiles : chacune scintille doucement à son rythme
const starMaterial = new THREE.ShaderMaterial({
  uniforms: {
    time: { value: 0 },
    size: { value: tweaks.starSize },
    scale: { value: 1 },
    color: { value: new THREE.Color(tweaks.starColor) },
    brightness: { value: tweaks.starBrightness },
    twinkle: { value: tweaks.twinkleAmount },
    speed: { value: tweaks.twinkleSpeed },
  },
  vertexShader: `
    attribute float phase;
    uniform float time;
    uniform float size;
    uniform float scale;
    uniform float brightness;
    uniform float twinkle;
    uniform float speed;
    varying float vLight;
    void main() {
      vec4 mv = modelViewMatrix * vec4(position, 1.0);
      float rate = (1.2 + fract(phase * 7.31) * 2.8) * speed;
      float flash = pow(0.5 + 0.5 * sin(time * rate + phase * 6.2831), 6.0) * twinkle;
      vLight = brightness + (1.0 - brightness) * flash;
      gl_PointSize = size * (scale / -mv.z) * (1.0 + flash);
      gl_Position = projectionMatrix * mv;
    }
  `,
  fragmentShader: `
    uniform vec3 color;
    varying float vLight;
    void main() {
      gl_FragColor = vec4(color * vLight, 1.0);
    }
  `,
})

const updateStars = () => {
  const u = starMaterial.uniforms
  u.size.value = tweaks.starSize
  u.color.value.set(tweaks.starColor)
  u.brightness.value = tweaks.starBrightness
  u.twinkle.value = tweaks.twinkleAmount
  u.speed.value = tweaks.twinkleSpeed
  stars?.geometry.setDrawRange(0, tweaks.starCount)
}

const STARS_MAX = 8000
const makeStars = () => {
  const count = STARS_MAX
  const positions = new Float32Array(count * 3)
  const phases = new Float32Array(count)
  for (let i = 0; i < count; i++) {
    const v = new THREE.Vector3().randomDirection().multiplyScalar(500 + Math.random() * 200)
    positions.set([v.x, v.y, v.z], i * 3)
    phases[i] = Math.random()
  }
  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
  geometry.setAttribute('phase', new THREE.BufferAttribute(phases, 1))
  geometry.setDrawRange(0, tweaks.starCount)
  return new THREE.Points(geometry, starMaterial)
}

// Étoiles filantes : de temps en temps, une traînée au loin
const SHOOTING_STAR_DISTANCE = 550
const SHOOTING_STAR_DOTS = 40
const shootingStars = new THREE.Group()
let nextShootingStar = 0

const launchShootingStar = (now: number) => {
  const forward = camera.getWorldDirection(new THREE.Vector3())
  const start = forward.add(new THREE.Vector3().randomDirection().multiplyScalar(0.6)).setLength(SHOOTING_STAR_DISTANCE)
  const dir = new THREE.Vector3().randomDirection().projectOnPlane(start).normalize()
  const geometry = new THREE.BufferGeometry()
  geometry.setAttribute('position', new THREE.BufferAttribute(new Float32Array(SHOOTING_STAR_DOTS * 3), 3))
  const fade = Array.from({ length: SHOOTING_STAR_DOTS }, (_, i) => (1 - i / SHOOTING_STAR_DOTS) ** 2)
  geometry.setAttribute('color', new THREE.Float32BufferAttribute(fade.flatMap((f) => [f, f, f]), 3))
  const trail = new THREE.Points(geometry, new THREE.PointsMaterial({
    size: tweaks.shootingSize,
    sizeAttenuation: false,
    vertexColors: true,
    transparent: true,
    blending: THREE.AdditiveBlending,
    depthWrite: false,
  }))
  trail.frustumCulled = false
  trail.userData = {
    start,
    dir,
    born: now,
    life: 900 + Math.random() * 600,
    speed: (0.2 + Math.random() * 0.1) * tweaks.shootingSpeed,
    length: 150 + Math.random() * 70,
  }
  shootingStars.add(trail)
}

const animateShootingStars = (now: number) => {
  if (now >= nextShootingStar) {
    if (nextShootingStar && tweaks.shootingRate > 0) launchShootingStar(now)
    nextShootingStar = now + (60000 / Math.max(tweaks.shootingRate, 0.5)) * (0.5 + Math.random())
  }
  for (const trail of [...shootingStars.children] as THREE.Points[]) {
    const { start, dir, born, life, speed, length } = trail.userData
    const t = (now - born) / life
    const material = trail.material as THREE.PointsMaterial
    if (t >= 1) {
      shootingStars.remove(trail)
      trail.geometry.dispose()
      material.dispose()
      continue
    }
    const head = start.clone().addScaledVector(dir, (now - born) * speed)
    const span = length * Math.min(1, t * 3)
    const positions = trail.geometry.attributes.position as THREE.BufferAttribute
    for (let i = 0; i < SHOOTING_STAR_DOTS; i++) {
      const dot = head.clone().addScaledVector(dir, -span * (i / SHOOTING_STAR_DOTS))
      positions.setXYZ(i, dot.x, dot.y, dot.z)
    }
    positions.needsUpdate = true
    material.opacity = Math.min(1, Math.sin(Math.PI * t) * 1.5)
  }
}

// Montage
onMounted(() => {
  const el = container.value!
  scene = new THREE.Scene()
  scene.background = new THREE.Color(tweaks.background)
  camera = new THREE.PerspectiveCamera(50, el.clientWidth / el.clientHeight, 0.1, 2000)
  camera.position.copy(viewDirection(cameraViews.overview)).multiplyScalar(channelDistance())
  renderer = new THREE.WebGLRenderer({ antialias: false })
  el.appendChild(renderer.domElement)
  labelRenderer = new CSS2DRenderer()
  labelRenderer.domElement.className = 'labels'
  el.appendChild(labelRenderer.domElement)
  resize()

  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.minDistance = 2.5
  controls.maxDistance = MAX_DISTANCE
  controls.addEventListener('start', () => { adjustingZoom = snapOverview = false })
  controls.maxPolarAngle = Math.PI * 0.49
  lockControls()

  stars = makeStars()
  scene.add(stars, shootingStars, new THREE.AmbientLight(0xffffff, 1.5), orbits, letters, smoke, selectionHalo, hoverHalo, hoverRing)
  loadRocket()

  renderer.domElement.addEventListener('pointerdown', onPointerDown)
  renderer.domElement.addEventListener('click', onClick)
  renderer.domElement.addEventListener('pointermove', onPointerMove)
  observer = new ResizeObserver(resize)
  observer.observe(el)
  ready = true
  snapOverview = props.overview
  syncSystem()
  syncLetters()
  animate()

  if (debug) {
    pane = new Pane({ title: 'Réglages', container: tweaksEl.value })
    const sceneFolder = pane.addFolder({ title: 'Scène', expanded: false })
    sceneFolder.addBinding(tweaks, 'background', { label: 'Fond' })
      .on('change', () => (scene.background as THREE.Color).set(tweaks.background))
    sceneFolder.addBinding(tweaks, 'sunSize', { label: 'Soleil', min: 0.3, max: 2.4, step: 0.05 })
    sceneFolder.addBinding(tweaks, 'pixelSize', { label: 'Pixels', min: 1, max: 6, step: 0.5 }).on('change', resize)

    const orbitsFolder = pane.addFolder({ title: 'Orbites', expanded: false })
    orbitsFolder.addBinding(tweaks, 'orbitColor', { label: 'Couleur' })
    orbitsFolder.addBinding(tweaks, 'orbitWidth', { label: 'Épaisseur', min: 0.5, max: 6, step: 0.1 })
    orbitsFolder.addBinding(tweaks, 'orbitOpacity', { label: 'Opacité', min: 0, max: 1, step: 0.01 })
    orbitsFolder.on('change', updateOrbits)

    const starsFolder = pane.addFolder({ title: 'Étoiles', expanded: false })
    starsFolder.addBinding(tweaks, 'starCount', { label: 'Densité', min: 0, max: STARS_MAX, step: 100 })
    starsFolder.addBinding(tweaks, 'starSize', { label: 'Taille', min: 0.5, max: 6, step: 0.1 })
    starsFolder.addBinding(tweaks, 'starColor', { label: 'Couleur' })
    starsFolder.addBinding(tweaks, 'starBrightness', { label: 'Luminosité', min: 0, max: 1, step: 0.01 })
    starsFolder.addBinding(tweaks, 'twinkleAmount', { label: 'Scintillement', min: 0, max: 2, step: 0.05 })
    starsFolder.addBinding(tweaks, 'twinkleSpeed', { label: 'Vitesse', min: 0, max: 4, step: 0.05 })
    starsFolder.on('change', updateStars)

    const shootingFolder = pane.addFolder({ title: 'Étoiles filantes', expanded: false })
    shootingFolder.addBinding(tweaks, 'shootingRate', { label: 'Par minute', min: 0, max: 60, step: 0.5 })
      .on('change', () => { nextShootingStar = Date.now() })
    shootingFolder.addBinding(tweaks, 'shootingSize', { label: 'Taille', min: 1, max: 6, step: 0.5 })
    shootingFolder.addBinding(tweaks, 'shootingSpeed', { label: 'Vitesse', min: 0.2, max: 3, step: 0.05 })
    shootingFolder.addButton({ title: 'En lancer une' }).on('click', () => launchShootingStar(Date.now()))

    const rocketsFolder = pane.addFolder({ title: 'Fusées', expanded: false })
    rocketsFolder.addBinding(tweaks, 'rocketSize', { label: 'Taille', min: 0.2, max: 2, step: 0.05 })
    rocketsFolder.addBinding(tweaks, 'smokeSize', { label: 'Fumée', min: 0.02, max: 0.4, step: 0.01 })
    rocketsFolder.addBinding(tweaks, 'landing', { label: 'Atterrissage', min: 0.05, max: 0.6, step: 0.01 })
    rocketsFolder.addBinding(tweaks, 'hearing', { label: 'Portée du son', min: 1, max: 30, step: 0.5 })

    const soundsFolder = pane.addFolder({ title: 'Sons', expanded: false })
    soundsFolder.addBinding(tweaks, 'music', { label: 'Musique', min: 0, max: 1, step: 0.01 })
      .on('change', () => sound.setMusicVolume(tweaks.music))
    soundsFolder.addBinding(tweaks, 'rockets', { label: 'Fusées', min: 0, max: 1, step: 0.01 })

    const haloFolder = pane.addFolder({ title: 'Halo de sélection', expanded: false })
    haloFolder.addBinding(tweaks, 'haloWidth', { label: 'Épaisseur', min: 0.04, max: 0.28, step: 0.01 })
    haloFolder.addBinding(tweaks, 'haloColor', { label: 'Couleur' })
    haloFolder.addBinding(tweaks, 'haloOpacity', { label: 'Intensité', min: 0, max: 1, step: 0.01 })
    haloFolder.on('change', updateHalo)

    pane.registerPlugin(TweakpanePluginMedia)
    const camerasFolder = pane.addFolder({ title: 'Caméras', expanded: false })
    camerasFolder.addButton({ title: 'Copier les caméras (JSON)' }).on('click', () => {
      navigator.clipboard.writeText(JSON.stringify(cameraViews, null, 2))
    })
    camerasFolder.addButton({ title: 'Réinitialiser' }).on('click', () => {
      localStorage.removeItem(CAMERAS_KEY)
      location.reload()
    })
    for (const [name, view] of Object.entries(cameraViews)) {
      const folder = camerasFolder.addFolder({ title: CAMERA_LABELS[name as keyof typeof cameraViews], expanded: false })
      folder.addBinding(view, 'distance', { label: 'Distance', min: 1, max: 120, step: 0.1 })
      folder.addBinding(view, 'fixedAngle', { label: 'Angle imposé' })
      folder.addBinding(view, 'elevation', { label: 'Hauteur (°)', min: 5, max: 89, step: 1 })
      folder.addBinding(view, 'azimuth', { label: 'Côté (°)', min: -180, max: 180, step: 1 })
    }
    camerasFolder.on('change', () => {
      adjustingZoom = true
      localStorage.setItem(CAMERAS_KEY, JSON.stringify(cameraViews))
    })
    planetsFolder = pane.addFolder({ title: 'Planètes', expanded: false })
    planetsFolder.addButton({ title: 'Copier les réglages (JSON)' }).on('click', () => {
      navigator.clipboard.writeText(JSON.stringify(lookValues(), null, 2))
    })
    planetsFolder.addButton({ title: 'Tout réinitialiser' }).on('click', () => {
      localStorage.removeItem(LOOKS_KEY)
      location.reload()
    })
    for (const [name, look] of looks) addLookFolder(name, look)
  }
})

watch(() => [props.system, props.decor, props.channels], () => { syncSystem(); syncLetters() })
watch(() => props.system?.channel.id, () => { followedKey = null; adjustingZoom = true; snapOverview = false })
watch(() => props.messages, syncLetters)
watch(() => props.followedLetter, (key) => {
  if (!key) return
  followedKey = key
  adjustingZoom = true
  snapOverview = false
})
watch(() => props.selectedId, () => { followedKey = null; adjustingZoom = true; snapOverview = false })
watch(() => props.customizing, previewLook, { deep: true })
watch(() => props.decor, previewLook)
watch(() => props.customizing?.id, () => {
  adjustingZoom = true
  snapOverview = false
  if (controls) lockControls()
  setHovered(undefined)
})
watch(() => props.overview, () => {
  followedKey = null
  adjustingZoom = true
  snapOverview = false
  setHovered(undefined)
})

onBeforeUnmount(() => {
  ready = false
  sound.stop('fusee')
  pane?.dispose()
  cancelAnimationFrame(frame)
  observer?.disconnect()
  controls?.dispose()
  renderer?.dispose()
  labelRenderer?.domElement.remove()
})
</script>

<style scoped>
.scene {
  position: relative;
  width: 100%;
  height: 100%;
}
.scene :deep(.labels) {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.scene :deep(.planet-label) {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  color: #fff;
  font: 600 16px/1 var(--font-family);
  text-shadow: 1px 1px 0 #000;
  white-space: nowrap;
}
.scene :deep(.planet-label:not(.planet-label--sun)) {
  transition: opacity 0.15s;
}
.scene :deep(.planet-label--covered) {
  opacity: 0;
}
.scene :deep(.planet-label__username) {
  font-size: 12px;
  font-weight: 400;
  opacity: 0.85;
}
.scene :deep(.planet-label--sun) {
  font-size: 20px;
  transition: font-size 0.15s, color 0.15s;
}
.scene :deep(.planet-label--hover) {
  font-size: 24px;
  color: #ffd54f;
}
.scene :deep(.planet-label--me) {
  color: #ffd54f;
}
.scene:not(.scene--overview):not(.scene--customizing) :deep(.planet-label:not(.planet-label--sun):not(.planet-label--covered)) {
  pointer-events: auto;
  cursor: pointer;
}
.scene :deep(.planet-label__name) {
  transition: color 0.15s;
}
.scene :deep(.planet-label--pointed .planet-label__name) {
  color: #d3fbb2;
}
.scene--overview:not(.scene--customizing) :deep(.planet-label--sun) {
  pointer-events: auto;
  cursor: pointer;
}
.scene :deep(canvas) {
  image-rendering: pixelated;
}
.tweaks {
  position: fixed;
  left: 16px;
  bottom: 16px;
  width: 280px;
  max-height: calc(100vh - 32px);
  overflow-y: auto;
  z-index: 20;
}
</style>
