// Sons du site (public/sounds), joués avec Web Audio : le volume marche aussi sur iPhone
const SOUNDS = {
  fusee: '/sounds/fusee.mp3',
  decollage: '/sounds/decollage.mp3',
  atterrissage: '/sounds/atterrissage.mp3',
  zibzib: '/sounds/zibzib.mp3',
  disabled: '/sounds/disabled.mp3',
  delete: '/sounds/delete.mp3',
} as const

export type SoundName = keyof typeof SOUNDS

// Musique de fond : lue en continu (pas décodée d'un bloc), basse, avec une montée douce
const MUSIC = '/sounds/musique.mp3'
export const MUSIC_VOLUME = 0.1
const MUSIC_FADE = 4
let music: HTMLAudioElement | null = null
let musicGain: GainNode | null = null

let context: AudioContext | null = null
const buffers = new Map<SoundName, Promise<AudioBuffer>>()
const loops = new Map<SoundName, { source?: AudioBufferSourceNode; gain: GainNode; stopped: boolean }>()

const audio = () => (context ??= new AudioContext())
const clampVolume = (volume: number) => Math.min(1, Math.max(0, volume))

const load = (name: SoundName) => {
  if (!buffers.has(name)) {
    const buffer = fetch(SOUNDS[name]).then((r) => r.arrayBuffer()).then((data) => audio().decodeAudioData(data))
    buffer.catch(() => buffers.delete(name))
    buffers.set(name, buffer)
  }
  return buffers.get(name)!
}

const start = async (name: SoundName, gain: GainNode, loop: boolean, offset = 0) => {
  const source = audio().createBufferSource()
  source.buffer = await load(name)
  source.loop = loop
  source.connect(gain)
  if (offset < source.buffer.duration) source.start(0, Math.max(0, offset))
  return source
}

const startMusic = () => {
  if (!music) {
    music = new Audio(MUSIC)
    music.loop = true
    musicGain = audio().createGain()
    musicGain.gain.setValueAtTime(0, audio().currentTime)
    musicGain.gain.linearRampToValueAtTime(MUSIC_VOLUME, audio().currentTime + MUSIC_FADE)
    audio().createMediaElementSource(music).connect(musicGain).connect(audio().destination)
  }
  if (music.paused) music.play().catch(() => {})
}

const setMusicVolume = (volume: number) => {
  if (!musicGain) return
  musicGain.gain.cancelScheduledValues(audio().currentTime)
  musicGain.gain.value = clampVolume(volume)
}

export const useSound = () => {
  // Les navigateurs gardent le son coupé jusqu'au premier geste
  const unlock = () => {
    if (audio().state === 'suspended') audio().resume()
    startMusic()
    for (const name of Object.keys(SOUNDS) as SoundName[]) load(name).catch(() => {})
  }

  // offset : secondes déjà écoulées, pour caler le son sur une animation en cours
  const play = (name: SoundName, volume = 1, offset = 0) => {
    const gain = audio().createGain()
    gain.gain.value = clampVolume(volume)
    gain.connect(audio().destination)
    start(name, gain, false, offset).catch(() => {})
    return { setVolume: (v: number) => { gain.gain.value = clampVolume(v) } }
  }

  const loop = (name: SoundName, volume = 1) => {
    const current = loops.get(name)
    if (current) {
      current.gain.gain.value = clampVolume(volume)
      return
    }
    const gain = audio().createGain()
    gain.gain.value = clampVolume(volume)
    gain.connect(audio().destination)
    const entry: { source?: AudioBufferSourceNode; gain: GainNode; stopped: boolean } = { gain, stopped: false }
    loops.set(name, entry)
    start(name, gain, true).then((source) => {
      entry.source = source
      if (entry.stopped) source.stop()
    }).catch(() => loops.delete(name))
  }

  const stop = (name: SoundName) => {
    const entry = loops.get(name)
    if (!entry) return
    entry.stopped = true
    entry.source?.stop()
    entry.gain.disconnect()
    loops.delete(name)
  }

  return { unlock, startMusic, setMusicVolume, play, loop, stop }
}
