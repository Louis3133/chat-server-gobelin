import type { Channel, Message, PlanetLook, Profile, PublicUser } from '../composables/useChatApi'
import type { LetterRules } from './letterRules'

// Réponses brutes de l'API partagée
export interface RawChannel {
  id: number
  name: string
  creation_date?: string
  metadata?: Record<string, any> | null
}

export interface RawUser {
  id: number
  username: string
  channels?: RawChannel[] | null
  metadata?: Record<string, any> | null
}

export interface RawMessage {
  id: number
  channel_id: number
  content: string
  user_id: number
  user: string
  anonymous: boolean
  creation_date: string
  posted_at: string
  metadata?: Record<string, any> | null
}

// Métadonnées Epistellaire
export interface MirroredChannel {
  id: number
  name: string
  owner_id: number | null
  creation_date?: string
}

export interface ChannelEdit {
  bio?: string
  deleted?: boolean
  at: number
}

export interface UserMeta extends Partial<PlanetLook>, Partial<Profile> {
  planet_texture?: string | null
  planet_texture_version?: number
  channels?: MirroredChannel[]
  channel_edits?: Record<string, ChannelEdit>
  hidden_users?: number[]
}

export interface LetterMeta {
  to: number
  to_name: string
  delay_ms: number
  distance: number
  reply: boolean
  letter: string
  cancelled?: boolean
  rocket_removed?: boolean
}

export const DEFAULT_LOOK: PlanetLook = {
  planet_color: '#8d9db6',
  planet_size: 0.55,
  planet_rings: false,
  planet_tilt: 0,
  planet_ring_color: '#d9c9a3',
  planet_ring_size: 2,
  planet_spin: 0.01,
  planet_moon: false,
}

export const metaOf = <T>(metadata: Record<string, any> | null | undefined, namespace: string): T | null => {
  const value = metadata?.[namespace]
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as T) : null
}

export const letterMetaOf = (m: RawMessage, namespace: string) => {
  const meta = metaOf<LetterMeta>(m.metadata, namespace)
  return meta && typeof meta.to === 'number' ? meta : null
}

// Utilisateurs
export const toPublicUser = (id: number, username: string, meta: UserMeta | null): PublicUser => ({
  id,
  username,
  planet_color: meta?.planet_color ?? DEFAULT_LOOK.planet_color,
  planet_size: meta?.planet_size ?? DEFAULT_LOOK.planet_size,
  planet_rings: meta?.planet_rings ?? DEFAULT_LOOK.planet_rings,
  planet_tilt: meta?.planet_tilt ?? DEFAULT_LOOK.planet_tilt,
  planet_ring_color: meta?.planet_ring_color ?? DEFAULT_LOOK.planet_ring_color,
  planet_ring_size: meta?.planet_ring_size ?? DEFAULT_LOOK.planet_ring_size,
  planet_spin: meta?.planet_spin ?? DEFAULT_LOOK.planet_spin,
  planet_moon: meta?.planet_moon ?? DEFAULT_LOOK.planet_moon,
  planet_name: meta?.planet_name ?? '',
  pronouns: meta?.pronouns ?? '',
  bio: meta?.bio ?? '',
  planet_texture_version: meta?.planet_texture ? meta.planet_texture_version || 1 : 0,
})

// Salons de l'utilisateur, sans doublon (l'API partagée en crée à chaque entrée)
export const mirrorChannels = (channels: RawChannel[] | null | undefined, namespace: string): MirroredChannel[] => {
  const seen = new Map<number, MirroredChannel>()
  for (const c of channels ?? []) {
    if (seen.has(c.id)) continue
    const owner = metaOf<{ owner_id?: number }>(c.metadata, namespace)?.owner_id
    seen.set(c.id, { id: c.id, name: c.name, owner_id: typeof owner === 'number' ? owner : null, creation_date: c.creation_date })
  }
  return [...seen.values()].sort((a, b) => a.id - b.id)
}

export const sameMirror = (a: MirroredChannel[] | undefined, b: MirroredChannel[]) =>
  !!a && a.length === b.length && a.every((c, i) => c.id === b[i]!.id && c.name === b[i]!.name && c.owner_id === b[i]!.owner_id)

// Vue d'ensemble tirée de GET /users : habitants, salons, suppressions
export interface World {
  users: PublicUser[]
  channels: Channel[]
  hiddenUsers: Set<number>
  deletedChannels: Map<number, string>
  textures: Map<number, string>
}

export const buildWorld = (raws: RawUser[], namespace: string, admins: string[]): World => {
  const metas = raws
    .filter((u) => u.username !== 'internal-clock')
    .map((u) => ({ user: u, meta: metaOf<UserMeta>(u.metadata, namespace) }))
    .filter((x): x is { user: RawUser; meta: UserMeta } => !!x.meta)

  const isAdmin = (username: string) => admins.includes(username)
  const hiddenUsers = new Set(metas.filter((x) => isAdmin(x.user.username)).flatMap((x) => x.meta.hidden_users ?? []))
  const visible = metas.filter((x) => !hiddenUsers.has(x.user.id)).sort((a, b) => a.user.id - b.user.id)

  const channels = new Map<number, Channel & { member_ids: number[] }>()
  for (const { user, meta } of visible) {
    for (const c of meta.channels ?? []) {
      const known = channels.get(c.id) ?? { id: c.id, name: c.name, owner_id: c.owner_id, creation_date: c.creation_date, bio: '', member_ids: [] }
      if (!known.member_ids.includes(user.id)) known.member_ids.push(user.id)
      channels.set(c.id, known)
    }
  }

  const deletedChannels = new Map<number, string>()
  const bioAt = new Map<number, number>()
  for (const { user, meta } of metas) {
    for (const [key, edit] of Object.entries(meta.channel_edits ?? {})) {
      const channel = channels.get(Number(key))
      if (!channel || !(isAdmin(user.username) || channel.owner_id === user.id)) continue
      if (edit.deleted && isAdmin(user.username)) deletedChannels.set(channel.id, channel.name)
      if (typeof edit.bio === 'string' && edit.at > (bioAt.get(channel.id) ?? -1)) {
        channel.bio = edit.bio
        bioAt.set(channel.id, edit.at)
      }
    }
  }

  const textures = new Map<number, string>()
  for (const { user, meta } of visible) if (meta.planet_texture) textures.set(user.id, meta.planet_texture)

  return {
    users: visible.map(({ user, meta }) => toPublicUser(user.id, user.username, meta)),
    channels: [...channels.values()]
      .filter((c) => !deletedChannels.has(c.id))
      .sort((a, b) => a.id - b.id),
    hiddenUsers,
    deletedChannels,
    textures,
  }
}

// Lettres
export const deliverAt = (m: RawMessage, meta: LetterMeta) => new Date(Date.parse(m.creation_date) + meta.delay_ms).toISOString()

export const letterInvolves = (meta: LetterMeta, m: RawMessage, a: number, b: number) =>
  (m.user_id === a && meta.to === b) || (m.user_id === b && meta.to === a)

export interface PairState {
  distance: number
  exchanges: number
  isReply: boolean
}

// Distance entre deux planètes : celle enregistrée par la dernière lettre du couple (annulées exclues)
export const pairState = (raws: RawMessage[], namespace: string, me: number, other: number, rules: LetterRules, now: number): PairState => {
  const letters = raws
    .map((m) => ({ m, meta: letterMetaOf(m, namespace) }))
    .filter((x): x is { m: RawMessage; meta: LetterMeta } => !!x.meta && !x.meta.cancelled && letterInvolves(x.meta, x.m, me, other))
    .sort((a, b) => a.m.id - b.m.id)
  const last = letters[letters.length - 1]
  return {
    distance: last && Number.isFinite(last.meta.distance) ? last.meta.distance : rules.maxDistance,
    exchanges: letters.filter((x) => x.meta.reply).length,
    isReply: !!last && last.m.user_id === other && Date.parse(deliverAt(last.m, last.meta)) <= now,
  }
}

// Ce que l'utilisateur a le droit de voir : messages de salon, ses lettres, et celles qu'on lui écrit (vides tant qu'elles sont en route)
export const visibleMessages = (raws: RawMessage[], namespace: string, me: number, names: Map<number, string>, hidden: Set<number>, now: number): Message[] => {
  const out: Message[] = []
  for (const m of raws) {
    const meta = letterMetaOf(m, namespace)
    if (!meta) {
      out.push({ id: m.id, content: m.content, user_id: m.user_id, user: m.user, recipient_id: null, recipient: null, creation_date: m.creation_date, deliver_at: m.posted_at, in_transit: false })
      continue
    }
    if (meta.cancelled || (m.user_id !== me && meta.to !== me)) continue
    if (hidden.has(m.user_id) || hidden.has(meta.to)) continue
    const deliver = deliverAt(m, meta)
    const inTransit = Date.parse(deliver) > now
    out.push({
      id: m.id,
      content: inTransit && m.user_id !== me ? '' : meta.letter,
      user_id: m.user_id,
      user: m.user,
      recipient_id: meta.to,
      recipient: names.get(meta.to) ?? meta.to_name ?? null,
      creation_date: m.creation_date,
      deliver_at: deliver,
      in_transit: inTransit,
      rocket_removed: !!meta.rocket_removed,
    })
  }
  return out.sort((a, b) => Date.parse(a.deliver_at) - Date.parse(b.deliver_at) || a.id - b.id)
}
