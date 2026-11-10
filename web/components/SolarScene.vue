<template>
  <div ref="container" class="scene" :class="{ 'scene--overview': overview }">
    <Teleport to="body"><div ref="tweaksEl" class="tweaks" /></Teleport>
  </div>
</template>

<script setup lang="ts">
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import { CSS2DObject, CSS2DRenderer } from 'three/addons/renderers/CSS2DRenderer.js'
import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js'
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

const props = defineProps<{
  system: SolarSystem | null
  decor: SolarSystem | null
  channels: Channel[]
  messages: Message[]
  selectedId: number | null
  overview: boolean
}>()
const emit = defineEmits<{ select: [id: number]; open: [channel: Channel]; center: [] }>()

const container = ref<HTMLDivElement>()
const tweaksEl = ref<HTMLDivElement>()

// Réglages Tweakpane (#debug dans l'URL)
const debug = useState<boolean>('debug', () => false).value
const tweaks = {
  sunSize: 1,
  pixelSize: 3,
  rocketSize: 1,
  smokeSize: 0.14,
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
const cameraViews: Record<'overview' | 'channel' | 'planet' | 'rocket', CameraView> = {
  overview: { distance: 24, fixedAngle: true, elevation: 66, azimuth: 0 },
  channel: { distance: 24, fixedAngle: false, elevation: 66, azimuth: 0 },
  planet: { distance: 5, fixedAngle: false, elevation: 30, azimuth: 0 },
  rocket: { distance: 2.6, fixedAngle: false, elevation: 30, azimuth: 0 },
}
if (debug) {
  try {
    const saved = JSON.parse(localStorage.getItem(CAMERAS_KEY) ?? '{}')
    for (const [name, view] of Object.entries(cameraViews)) Object.assign(view, saved[name])
  } catch {}
}
const CAMERA_LABELS = { overview: "Vue d'ensemble", channel: 'Salon', planet: 'Planète sélectionnée', rocket: 'Suivi de fusée' }
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

const followedRocket = () => (followedKey ? letters.children.find((d) => d.userData.key === followedKey) : undefined)

const stopFollowing = () => {
  if (!followedKey) return
  followedKey = null
  adjustingZoom = true
}

const rocketAt = (event: MouseEvent) => {
  const { ray } = rayAt(event)
  return letters.children
    .filter((d) => ray.distanceToPoint(d.position) < ROCKET_HIT_RADIUS * tweaks.rocketSize)
    .sort((a, b) => a.position.distanceTo(camera.position) - b.position.distanceTo(camera.position))[0]
}

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
  size: 0.14,
  vertexColors: true,
  transparent: true,
  depthWrite: false,
}))
smoke.frustumCulled = false
const TAIL = new THREE.Vector3(0, 0, -ROCKET_LENGTH / 2)

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
const selectionRing = new THREE.Mesh(
  new THREE.RingGeometry(0.6, 0.7, 48),
  new THREE.MeshBasicMaterial({ color: 0xffffff, side: THREE.DoubleSide }),
)
const hoverRing = new THREE.Mesh(
  new THREE.RingGeometry(0.85, 1, 48),
  new THREE.MeshBasicMaterial({ color: 0xffd54f, side: THREE.DoubleSide }),
)
let hoveredSun: THREE.Group | undefined

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

const BASE_SIZE = 0.55

const lookFor = (p: Planet) => {
  const name = p.username
  let look = looks.get(name)
  if (!look) {
    look = {
      size: p.size ?? BASE_SIZE,
      color: p.color ?? `#${colorFor(name).getHexString()}`,
      texture,
      tilt: 0,
      rings: p.rings ?? false,
      ringColor: '#d9c9a3',
      ringSize: 2,
      spin: 0.01,
      ...(debug ? savedLooks()[name] : {}),
    }
    looks.set(name, look)
    addLookFolder(name, look)
  }
  return look
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
  folder.addBinding(look, 'rings', { label: 'Anneaux' })
  folder.addBinding(look, 'ringColor', { label: 'Couleur anneaux' })
  folder.addBinding(look, 'ringSize', { label: 'Taille anneaux', min: 1.5, max: 4, step: 0.05 })
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

const openSun = (group: THREE.Object3D) => {
  if (props.overview && group.userData.channel) emit('open', group.userData.channel)
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

// Dans un salon, le soleil ramène à la vue du salon quand une planète est sélectionnée
const activeSunAt = (event: MouseEvent) => {
  const sun = suns.get(activeKey)?.children[0]
  return props.selectedId != null && !!sun && rayAt(event).intersectObject(sun, false).length > 0
}

// Synchronisation avec les données
const syncSystem = () => {
  if (!ready) return
  const where = centers()
  activeKey = props.system ? keyOf(props.system.channel) : 'general'
  activeCenter.copy(where.get(activeKey)?.pos ?? ORIGIN)
  fitOverview([...where.values()].map((w) => w.pos))

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
    const label = makeLabel(p.me ? `${p.username} (moi)` : p.username)
    label.position.set(0, 1.4, 0)
    mesh.add(label)
    scene.add(mesh)
    node = { mesh, label, key, center: center.clone(), target: t.pos.clone(), me: p.me }
    planets.set(k, node)
  }
  node.target.copy(t.pos)
  const ring = new THREE.LineLoop(
    new THREE.BufferGeometry().setFromPoints(new THREE.Path().absarc(0, 0, t.radius, 0, Math.PI * 2).getSpacedPoints(128)),
    new THREE.LineBasicMaterial({ color: 0x445566, transparent: true, opacity: 0.5 }),
  )
  ring.rotation.x = Math.PI / 2 - t.tilt
  rings.add(ring)
}

// Lettres en transit
const syncLetters = () => {
  if (!ready) return
  letters.clear()
  for (const m of props.messages) {
    if (!m.in_transit || !m.recipient_id) continue
    const from = planets.get(`${activeKey}:${m.user_id}`)
    const to = planets.get(`${activeKey}:${m.recipient_id}`)
    if (!from || !to) continue
    const dot = rocketModel?.clone() ?? new THREE.Mesh(new THREE.SphereGeometry(0.2, 12, 12), new THREE.MeshBasicMaterial({ color: 0xffd54f }))
    dot.userData = {
      key: `${m.user_id}:${m.recipient_id}:${m.creation_date}`,
      lastPuff: 0,
      from: from.mesh,
      to: to.mesh,
      start: new Date(m.creation_date).getTime(),
      end: new Date(m.deliver_at).getTime(),
    }
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

// Caméra : suit la sélection, le soleil du salon ouvert ou l'ensemble des salons
const followSelection = (selected?: PlanetNode, rocket?: THREE.Object3D) => {
  const wanted = rocket ? rocket.position : selected ? selected.mesh.position : props.overview ? overviewCenter : activeCenter
  const before = controls.target.clone()
  controls.target.lerp(wanted, 0.04)
  camera.position.add(controls.target.clone().sub(before))

  if (adjustingZoom) {
    const offset = camera.position.clone().sub(controls.target)
    const view = rocket ? cameraViews.rocket : selected ? cameraViews.planet : props.overview ? cameraViews.overview : cameraViews.channel
    const goal = rocket || selected ? view.distance : props.overview ? overviewDistance() : channelDistance()
    const wantedDir = view.fixedAngle ? viewDirection(view) : null
    const next = THREE.MathUtils.lerp(offset.length(), goal, 0.05)
    const dir = offset.normalize()
    if (wantedDir) dir.lerp(wantedDir, 0.05).normalize()
    camera.position.copy(controls.target).add(dir.multiplyScalar(next))
    if (Math.abs(next - goal) < 0.05 && (!wantedDir || dir.angleTo(wantedDir) < 0.01)) adjustingZoom = false
  }
  const farthest = Math.max(MAX_DISTANCE, channelDistance() * 1.2)
  controls.maxDistance = props.overview || adjustingZoom ? Math.max(farthest, overviewDistance() * 1.2) : farthest
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
    node.label.visible = !props.overview && node.key === activeKey
  }
  for (const dot of letters.children) {
    const { from, to, start, end } = dot.userData
    const progress = progressOf(start, end, now)
    dot.position.lerpVectors(from.position, to.position, progress)
    dot.scale.setScalar(tweaks.rocketSize)
    dot.lookAt(to.position)
    if (rocketModel && progress < 1 && now - dot.userData.lastPuff >= SMOKE_EVERY) {
      dot.userData.lastPuff = now
      puff(dot, now)
    }
  }
  ;(smoke.material as THREE.PointsMaterial).size = tweaks.smokeSize
  animateSmoke(now)
  const selected = props.selectedId != null ? planets.get(`${activeKey}:${props.selectedId}`) : undefined
  selectionRing.visible = !!selected
  if (selected) {
    selectionRing.position.copy(selected.mesh.position)
    selectionRing.scale.setScalar(selected.mesh.scale.x)
    selectionRing.quaternion.copy(camera.quaternion)
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
  followSelection(selected, rocket)
  controls.update()
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
  starMaterial.uniforms.scale.value = renderer.getDrawingBufferSize(new THREE.Vector2()).y / 2
  labelRenderer.setSize(el.clientWidth, el.clientHeight)
  camera.aspect = el.clientWidth / el.clientHeight
  camera.updateProjectionMatrix()
}

const onPointerDown = (e: PointerEvent) => { downAt = { x: e.clientX, y: e.clientY } }

const onPointerMove = (event: PointerEvent) => {
  setHovered(props.overview ? sunAt(event) : undefined)
  if (!props.overview) renderer.domElement.style.cursor = activeSunAt(event) ? 'pointer' : ''
}

const onClick = (event: MouseEvent) => {
  if (Math.hypot(event.clientX - downAt.x, event.clientY - downAt.y) > 5) return
  if (props.overview) {
    const sun = sunAt(event)
    if (sun) openSun(sun)
    return
  }
  const rocket = rocketAt(event)
  if (rocket) {
    followedKey = rocket.userData.key
    adjustingZoom = true
    return
  }
  stopFollowing()
  const active = [...planets.values()].filter((n) => n.key === activeKey)
  const hit = rayAt(event).intersectObjects(active.map((n) => n.mesh), false)[0]
  if (hit) emit('select', hit.object.userData.id)
  else if (activeSunAt(event)) emit('center')
}

// Étoiles : chacune scintille doucement à son rythme
const starMaterial = new THREE.ShaderMaterial({
  uniforms: { time: { value: 0 }, size: { value: 2 }, scale: { value: 1 } },
  vertexShader: `
    attribute float phase;
    uniform float time;
    uniform float size;
    uniform float scale;
    varying float vLight;
    void main() {
      vec4 mv = modelViewMatrix * vec4(position, 1.0);
      float speed = 1.2 + fract(phase * 7.31) * 2.8;
      float flash = pow(0.5 + 0.5 * sin(time * speed + phase * 6.2831), 6.0);
      vLight = 0.4 + 0.6 * flash;
      gl_PointSize = size * (scale / -mv.z) * (1.0 + flash);
      gl_Position = projectionMatrix * mv;
    }
  `,
  fragmentShader: `
    varying float vLight;
    void main() {
      gl_FragColor = vec4(vec3(vLight), 1.0);
    }
  `,
})

const makeStars = () => {
  const count = 1500
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
    size: 2,
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
    speed: 0.2 + Math.random() * 0.1,
    length: 150 + Math.random() * 70,
  }
  shootingStars.add(trail)
}

const animateShootingStars = (now: number) => {
  if (now >= nextShootingStar) {
    if (nextShootingStar) launchShootingStar(now)
    nextShootingStar = now + 8000 + Math.random() * 12000
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
  scene.background = new THREE.Color(0x05060f)
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
  controls.addEventListener('start', () => { adjustingZoom = false })
  controls.maxPolarAngle = Math.PI * 0.49

  stars = makeStars()
  scene.add(stars, shootingStars, new THREE.AmbientLight(0xffffff, 1.5), orbits, letters, smoke, selectionRing, hoverRing)
  loadRocket()

  renderer.domElement.addEventListener('pointerdown', onPointerDown)
  renderer.domElement.addEventListener('click', onClick)
  renderer.domElement.addEventListener('pointermove', onPointerMove)
  observer = new ResizeObserver(resize)
  observer.observe(el)
  ready = true
  adjustingZoom = props.overview
  syncSystem()
  syncLetters()
  animate()

  if (debug) {
    pane = new Pane({ title: 'Réglages', container: tweaksEl.value })
    pane.addBinding(tweaks, 'sunSize', { label: 'Soleil', min: 0.3, max: 2.4, step: 0.05 })
    pane.addBinding(tweaks, 'pixelSize', { label: 'Pixels', min: 1, max: 6, step: 1 }).on('change', resize)
    pane.addBinding(tweaks, 'rocketSize', { label: 'Fusée', min: 0.2, max: 2, step: 0.05 })
    pane.addBinding(tweaks, 'smokeSize', { label: 'Fumée', min: 0.02, max: 0.4, step: 0.01 })
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
watch(() => props.system?.channel.id, () => { followedKey = null; adjustingZoom = true })
watch(() => props.messages, syncLetters)
watch(() => props.selectedId, () => { followedKey = null; adjustingZoom = true })
watch(() => props.overview, () => {
  followedKey = null
  adjustingZoom = true
  setHovered(undefined)
})

onBeforeUnmount(() => {
  ready = false
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
  color: #fff;
  font: 600 12px/1 monospace;
  text-shadow: 1px 1px 0 #000;
  white-space: nowrap;
}
.scene :deep(.planet-label--sun) {
  font-size: 16px;
  transition: font-size 0.15s, color 0.15s;
}
.scene :deep(.planet-label--hover) {
  font-size: 20px;
  color: #ffd54f;
}
.scene--overview :deep(.planet-label--sun) {
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
