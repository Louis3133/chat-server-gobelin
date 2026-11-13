export interface Channel {
  id: number
  name: string
  owner_id?: number | null
  creation_date?: string
  bio?: string
  member_ids?: number[]
}

// Apparence d'une planète, telle que l'API la stocke (PUT /me/planet)
export interface PlanetLook {
  planet_color: string
  planet_size: number
  planet_rings: boolean
  planet_tilt: number
  planet_ring_color: string
  planet_ring_size: number
  planet_spin: number
  planet_moon: boolean
}

// Profil affiché au clic sur une planète (PUT /me/profile)
export interface Profile {
  planet_name: string
  pronouns: string
  bio: string
}

export interface UserMe extends Partial<PlanetLook>, Partial<Profile> {
  planet_texture_version?: number
  id: number
  username: string
  channels: Channel[]
  is_admin?: boolean
}

export interface PublicUser extends Partial<PlanetLook>, Partial<Profile> {
  planet_texture_version?: number
  id: number
  username: string
}

export interface Message {
  id: number
  content: string
  user_id: number
  user: string
  recipient_id: number | null
  recipient: string | null
  creation_date: string
  deliver_at: string
  in_transit: boolean
  rocket_removed?: boolean
}

export interface Planet extends Partial<Profile> {
  id: number
  username: string
  me: boolean
  distance: number | null
  delay_seconds: number | null
  exchanges: number
  color?: string
  size?: number
  rings?: boolean
  tilt?: number
  ring_color?: string
  ring_size?: number
  spin?: number
  moon?: boolean
  texture_version?: number
}

export interface SolarSystem {
  channel: Channel
  planets: Planet[]
  min_distance: number
  max_distance: number
  time_scale: number
}

const TOKEN_KEY = 'chat_goblin_token'

const textures = new Map<string, string>()
const knownMessages = new Map<number, RawMessage>()
const inFlight = new Map<string, Promise<unknown>>()

const COLOR_RE = /^#[0-9a-fA-F]{6}$/
const MAX_CONTENT = 100000
const MAX_TEXTURE_BYTES = 256 << 10

// Erreur au format de $fetch, lue par apiErrorMessage
const httpError = (status: number, message: string) => Object.assign(new Error(message), { response: { status }, data: message })

const checkLook = (look: PlanetLook) => {
  if (!COLOR_RE.test(look.planet_color)) return 'planet_color must look like #rrggbb'
  if (!(look.planet_size >= 0.4 && look.planet_size <= 1)) return 'planet_size must be between 0.4 and 1'
  if (!(look.planet_tilt >= -1.2 && look.planet_tilt <= 1.2)) return 'planet_tilt must be between -1.2 and 1.2'
  if (!COLOR_RE.test(look.planet_ring_color)) return 'planet_ring_color must look like #rrggbb'
  if (!(look.planet_ring_size >= 1.5 && look.planet_ring_size <= 4)) return 'planet_ring_size must be between 1.5 and 4'
  if (!(look.planet_spin >= -0.1 && look.planet_spin <= 0.1)) return 'planet_spin must be between -0.1 and 0.1'
  if (look.planet_rings && look.planet_moon) return 'a planet has either rings or a moon, not both'
  return null
}

const readAsDataUrl = (blob: Blob) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as string)
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(blob)
  })

export const useChatApi = () => {
  const config = useRuntimeConfig().public
  const baseURL = config.apiBase as string
  const namespace = config.apiNamespace as string
  const admins = String(config.adminUsernames ?? '').split(',').map((s) => s.trim()).filter(Boolean)
  const rules = makeLetterRules(config)

  const getToken = (): string | null => {
    if (import.meta.client) {
      return localStorage.getItem(TOKEN_KEY)
    }
    return null
  }

  const setToken = (token: string) => {
    if (import.meta.client) {
      localStorage.setItem(TOKEN_KEY, token)
    }
  }

  const clearToken = () => {
    if (import.meta.client) {
      localStorage.removeItem(TOKEN_KEY)
    }
  }

  const getHeaders = (): Record<string, string> => {
    const token = getToken()
    return token ? { Authorization: `Bearer ${token}` } : {}
  }

  // Requêtes vers l'API partagée : namespace toujours en paramètre (l'en-tête Namespace passe mal le CORS), GET identiques simultanés regroupés
  const request = <T>(path: string, options: Record<string, any> = {}, auth = true): Promise<T> => {
    const params = { namespace, ...options.params }
    const key = !options.method || options.method === 'GET' ? JSON.stringify([path, params, auth && getToken()]) : null
    if (key && inFlight.has(key)) return inFlight.get(key) as Promise<T>

    const run = (async () => {
      try {
        return await $fetch<T>(path, {
          baseURL,
          responseType: 'json',
          ...options,
          params,
          headers: auth ? getHeaders() : {},
        })
      } catch (error: any) {
        if (auth && error?.response?.status === 401) {
          clearToken()
          await navigateTo('/')
        }
        throw error
      } finally {
        if (key) inFlight.delete(key)
      }
    })()
    if (key) inFlight.set(key, run)
    return run
  }

  const remember = (world: World) => {
    for (const u of world.users) {
      const texture = world.textures.get(u.id)
      if (texture) textures.set(`${u.id}:${u.planet_texture_version}`, texture)
    }
    return world
  }

  const loadWorld = async () => remember(buildWorld((await request<RawUser[] | null>('/users', {}, false)) ?? [], namespace, admins))
  const loadMe = () => request<RawUser>('/me')
  const loadRawMessages = async (channelId: number) => {
    const raws = (await request<RawMessage[] | null>('/channel/message', { params: { channel: channelId } })) ?? []
    for (const m of raws) knownMessages.set(m.id, m)
    return raws
  }

  const isAdmin = (username: string) => admins.includes(username)

  // Métadonnées de l'utilisateur connecté : relues juste avant chaque écriture, PUT /user remplace tout le namespace
  const saveMeta = async (meta: UserMeta) => {
    const user = await request<RawUser>('/user', { method: 'PUT', body: { metadata: meta } })
    return metaOf<UserMeta>(user.metadata, namespace) ?? meta
  }

  const updateMeta = async (change: (meta: UserMeta, me: RawUser) => UserMeta): Promise<PublicUser> => {
    const me = await loadMe()
    const meta = await saveMeta(change(metaOf<UserMeta>(me.metadata, namespace) ?? {}, me))
    const user = toPublicUser(me.id, me.username, meta)
    if (meta.planet_texture) textures.set(`${me.id}:${user.planet_texture_version}`, meta.planet_texture)
    return user
  }

  // Salon dont l'utilisateur est membre (403 sinon, 404 s'il n'existe pas)
  const memberChannel = (me: RawUser, world: World, channelId: number): Channel => {
    const mine = mirrorChannels(me.channels, namespace).find((c) => c.id === channelId)
    if (!mine || world.deletedChannels.has(channelId)) {
      if (world.channels.some((c) => c.id === channelId)) throw httpError(403, 'Forbidden')
      throw httpError(404, 'Channel not found')
    }
    const known = world.channels.find((c) => c.id === channelId)
    return { id: mine.id, name: mine.name, owner_id: mine.owner_id, creation_date: mine.creation_date, bio: known?.bio ?? '' }
  }

  const membersOf = (me: RawUser, world: World, channelId: number): PublicUser[] => {
    const ids = new Set(world.channels.find((c) => c.id === channelId)?.member_ids ?? [])
    const members = world.users.filter((u) => ids.has(u.id) && u.id !== me.id)
    const meUser = world.users.find((u) => u.id === me.id) ?? toPublicUser(me.id, me.username, metaOf<UserMeta>(me.metadata, namespace))
    return [...members, meUser].sort((a, b) => a.id - b.id)
  }

  const login = async (username: string, password: string): Promise<string> => {
    const token = await request<string>('/login', {
      method: 'POST',
      body: { username, password },
      responseType: 'text',
    }, false)
    setToken(token)
    const [me, world] = await Promise.all([loadMe(), loadWorld()])
    if (world.hiddenUsers.has(me.id)) {
      clearToken()
      throw httpError(401, 'Unauthorized')
    }
    return token
  }

  const register = async (username: string, password: string) => {
    const users = (await request<RawUser[] | null>('/users', {}, false)) ?? []
    if (users.some((u) => u.username === username)) throw httpError(400, 'Ce nom est déjà pris')
    return request<UserMe>('/user', { method: 'POST', body: { username, password, metadata: { ...DEFAULT_LOOK, channels: [] } } }, false)
  }

  const getUsers = async () => (await loadWorld()).users
  const getChannels = async () => (await loadWorld()).channels

  // GET /me, puis recopie de ses salons dans ses métadonnées : c'est ce qui les rend visibles sans token
  const getMe = async (): Promise<UserMe> => {
    const [me, world] = await Promise.all([loadMe(), loadWorld()])
    if (world.hiddenUsers.has(me.id)) {
      clearToken()
      await navigateTo('/')
      throw httpError(401, 'Unauthorized')
    }
    let meta = metaOf<UserMeta>(me.metadata, namespace)
    const mirror = mirrorChannels(me.channels, namespace)
    if (!meta || !sameMirror(meta.channels, mirror)) meta = await saveMeta({ ...DEFAULT_LOOK, ...meta, channels: mirror })
    const user = toPublicUser(me.id, me.username, meta)
    if (meta.planet_texture) textures.set(`${me.id}:${user.planet_texture_version}`, meta.planet_texture)
    const channels = mirror
      .filter((c) => !world.deletedChannels.has(c.id))
      .map((c) => ({ id: c.id, name: c.name, owner_id: c.owner_id, creation_date: c.creation_date, bio: world.channels.find((x) => x.id === c.id)?.bio ?? '' }))
    return { ...user, channels, is_admin: isAdmin(me.username) }
  }

  const updatePlanet = (look: PlanetLook) => {
    const problem = checkLook(look)
    if (problem) return Promise.reject(httpError(400, problem))
    return updateMeta((meta) => ({ ...meta, ...look }))
  }

  const uploadTexture = async (image: Blob) => {
    if (image.size > MAX_TEXTURE_BYTES) throw httpError(413, `texture must be at most ${MAX_TEXTURE_BYTES >> 10} KB`)
    if (!['image/jpeg', 'image/png'].includes(image.type)) throw httpError(400, 'texture must be a JPEG or PNG image')
    const texture = await readAsDataUrl(image)
    return updateMeta((meta) => ({ ...meta, planet_texture: texture, planet_texture_version: (meta.planet_texture_version ?? 0) + 1 }))
  }

  const deleteTexture = () => updateMeta((meta) => ({ ...meta, planet_texture: null }))

  const updateProfile = (profile: Profile) => {
    const clean = { planet_name: profile.planet_name.trim(), pronouns: profile.pronouns.trim(), bio: profile.bio.trim() }
    if ([...clean.planet_name].length > 40) return Promise.reject(httpError(400, 'planet_name must be at most 40 characters'))
    if ([...clean.pronouns].length > 30) return Promise.reject(httpError(400, 'pronouns must be at most 30 characters'))
    if ([...clean.bio].length > 500) return Promise.reject(httpError(400, 'bio must be at most 500 characters'))
    return updateMeta((meta) => ({ ...meta, ...clean }))
  }

  // Texture importée, rangée en data URL dans les métadonnées
  const textureUrl = (userId: number, version: number) => textures.get(`${userId}:${version}`) ?? ''

  // Création réservée aux admins (NUXT_PUBLIC_ADMIN_USERNAMES) ; le créateur devient propriétaire
  const createChannel = async (name: string) => {
    const me = await loadMe()
    if (!isAdmin(me.username)) throw httpError(403, 'Forbidden')
    if (!name.trim()) throw httpError(400, 'name required')
    const created = await request<RawChannel>('/channel', { method: 'POST', body: { name: name.trim(), metadata: { owner_id: me.id } } })
    await getMe()
    return { id: created.id, name: created.name, owner_id: me.id, creation_date: created.creation_date } as Channel
  }

  // Sans effet si on y est déjà : l'API partagée ajouterait un doublon
  const enterChannel = async (channelName: string): Promise<void> => {
    const [me, world] = await Promise.all([loadMe(), loadWorld()])
    if ([...world.deletedChannels.values()].includes(channelName)) throw httpError(404, 'Channel not found')
    if (mirrorChannels(me.channels, namespace).some((c) => c.name === channelName)) return
    try {
      await request<RawChannel>('/channel/enter', { method: 'POST', body: { channel: channelName } })
    } catch (error: any) {
      if (error?.response?.status === 500) throw httpError(404, 'Channel not found')
      throw error
    }
    await getMe()
  }

  // Annuler : la lettre est vidée et marquée annulée (l'API partagée ne supprime rien)
  const cancelLetter = async (messageId: number): Promise<void> => {
    const me = await loadMe()
    const raw = knownMessages.get(messageId)
    const meta = raw && letterMetaOf(raw, namespace)
    if (!raw || !meta || meta.cancelled || raw.user_id !== me.id || Date.parse(deliverAt(raw, meta)) <= Date.now()) {
      throw httpError(404, 'Letter not found, not yours, or already arrived')
    }
    const cancelled: LetterMeta = { ...meta, letter: '', cancelled: true }
    await request('/channel/message', { method: 'PUT', body: { id: raw.id, content: '✉️ Lettre annulée', metadata: cancelled } })
    knownMessages.set(raw.id, { ...raw, metadata: { [namespace]: cancelled } })
  }

  const removeRocket = async (messageId: number): Promise<void> => {
    const me = await loadMe()
    const raw = knownMessages.get(messageId)
    const meta = raw && letterMetaOf(raw, namespace)
    if (!raw || !meta || meta.cancelled || meta.to !== me.id || Date.parse(deliverAt(raw, meta)) > Date.now()) {
      throw httpError(404, 'Letter not found, not yours, or still in transit')
    }
    const removed: LetterMeta = { ...meta, rocket_removed: true }
    await request('/channel/message', { method: 'PUT', body: { id: raw.id, content: '', metadata: removed } })
    knownMessages.set(raw.id, { ...raw, metadata: { [namespace]: removed } })
  }

  // Suppressions (admins) : un marquage dans ses métadonnées, qu'Epistellaire respecte partout
  const deleteChannel = async (channelId: number): Promise<void> => {
    const [me, world] = await Promise.all([loadMe(), loadWorld()])
    if (!isAdmin(me.username)) throw httpError(403, 'Forbidden')
    const channel = world.channels.find((c) => c.id === channelId)
    if (!channel) throw httpError(404, 'Channel not found')
    if (channel.name === 'general') throw httpError(400, 'The general channel cannot be deleted')
    await updateMeta((meta) => ({ ...meta, channel_edits: { ...meta.channel_edits, [channelId]: { ...meta.channel_edits?.[channelId], deleted: true, at: Date.now() } } }))
  }

  const deleteUser = async (userId: number): Promise<void> => {
    const [me, world] = await Promise.all([loadMe(), loadWorld()])
    if (!isAdmin(me.username)) throw httpError(403, 'Forbidden')
    const user = world.users.find((u) => u.id === userId)
    if (!user) throw httpError(404, 'User not found')
    if (user.username === 'internal-clock') throw httpError(400, 'The internal clock user cannot be deleted')
    await updateMeta((meta) => ({ ...meta, hidden_users: [...new Set([...(meta.hidden_users ?? []), userId])] }))
  }

  // Bio du salon : admins et propriétaire, rangée dans leurs métadonnées (la plus récente l'emporte)
  const updateChannelBio = async (channelId: number, bio: string): Promise<Channel> => {
    const [me, world] = await Promise.all([loadMe(), loadWorld()])
    const channel = world.channels.find((c) => c.id === channelId)
    if (!channel) throw httpError(404, 'Channel not found')
    if (!isAdmin(me.username) && channel.owner_id !== me.id) throw httpError(403, 'Forbidden')
    const clean = bio.trim()
    if ([...clean].length > 500) throw httpError(400, 'bio must be at most 500 characters')
    await updateMeta((meta) => ({ ...meta, channel_edits: { ...meta.channel_edits, [channelId]: { ...meta.channel_edits?.[channelId], bio: clean, at: Date.now() } } }))
    return { ...channel, bio: clean }
  }

  // Système solaire calculé ici : membres, apparences et SES distances, tirées de ses lettres
  const getSystem = async (channelId: number): Promise<SolarSystem> => {
    const [me, world, raws] = await Promise.all([loadMe(), loadWorld(), loadRawMessages(channelId)])
    const channel = memberChannel(me, world, channelId)
    const now = Date.now()
    const planets = membersOf(me, world, channelId).map((u): Planet => {
      const self = u.id === me.id
      const pair = self ? null : pairState(raws, namespace, me.id, u.id, rules, now)
      return {
        id: u.id,
        username: u.username,
        me: self,
        distance: pair ? pair.distance : null,
        delay_seconds: pair ? letterDelayMs(rules, pair.distance) / 1000 : null,
        exchanges: pair?.exchanges ?? 0,
        color: u.planet_color,
        size: u.planet_size,
        rings: u.planet_rings,
        tilt: u.planet_tilt,
        ring_color: u.planet_ring_color,
        ring_size: u.planet_ring_size,
        spin: u.planet_spin,
        moon: u.planet_moon,
        texture_version: u.planet_texture_version,
        planet_name: u.planet_name,
        pronouns: u.pronouns,
        bio: u.bio,
      }
    })
    return { channel, planets, min_distance: rules.minDistance, max_distance: rules.maxDistance, time_scale: rules.timeScale }
  }

  const getMessages = async (channelId: number): Promise<Message[]> => {
    const [me, world, raws] = await Promise.all([loadMe(), loadWorld(), loadRawMessages(channelId)])
    memberChannel(me, world, channelId)
    const names = new Map(world.users.map((u) => [u.id, u.username]))
    return visibleMessages(raws, namespace, me.id, names, world.hiddenUsers, Date.now())
  }

  // Sans recipientId : message de salon, livré tout de suite. Avec : lettre, dont le délai dépend de la distance
  const sendMessage = async (channelId: number, content: string, recipientId?: number): Promise<Message> => {
    if (!content || content.length > MAX_CONTENT) throw httpError(400, `content must be between 1 and ${MAX_CONTENT} characters`)
    const [me, world, raws] = await Promise.all([loadMe(), loadWorld(), recipientId ? loadRawMessages(channelId) : Promise.resolve([])])
    memberChannel(me, world, channelId)

    if (!recipientId) {
      const sent = await request<RawMessage>('/channel/message', { method: 'POST', body: { channel: channelId, content } })
      return visibleMessages([sent], namespace, me.id, new Map(), new Set(), Date.now())[0]!
    }

    if (recipientId === me.id) throw httpError(400, 'cannot send a letter to yourself')
    const recipient = membersOf(me, world, channelId).find((u) => u.id === recipientId)
    if (!recipient) throw httpError(400, 'recipient is not in this channel')

    const pair = pairState(raws, namespace, me.id, recipientId, rules, Date.now())
    const distance = pair.isReply ? approachedDistance(rules, pair.distance) : pair.distance
    const meta: LetterMeta = {
      to: recipientId,
      to_name: recipient.username,
      delay_ms: letterDelayMs(rules, distance),
      distance,
      reply: pair.isReply,
      letter: content,
    }
    const sent = await request<RawMessage>('/channel/message', {
      method: 'POST',
      body: { channel: channelId, content: `✉️ Lettre de ${me.username} pour ${recipient.username}`, metadata: meta },
    })
    const raw = { ...sent, metadata: { [namespace]: meta } }
    knownMessages.set(raw.id, raw)
    const [letter] = visibleMessages([raw], namespace, me.id, new Map([[recipientId, recipient.username]]), new Set(), Date.now())
    return { ...letter!, in_transit: meta.delay_ms > 0 }
  }

  return {
    getToken,
    setToken,
    clearToken,
    login,
    register,
    getUsers,
    getChannels,
    getMe,
    updatePlanet,
    updateProfile,
    cancelLetter,
    uploadTexture,
    deleteTexture,
    textureUrl,
    createChannel,
    enterChannel,
    deleteChannel,
    deleteUser,
    getSystem,
    updateChannelBio,
    getMessages,
    sendMessage,
    removeRocket,
  }
}

// Message lisible pour une erreur $fetch
export const apiErrorMessage = (error: any): string => {
  const status = error?.response?.status
  const detail = typeof error?.data === 'string' ? error.data.trim() : ''
  if (status === 401) return 'Identifiants invalides ou session expirée'
  if (status === 403) return detail || 'Accès refusé'
  if (status === 404) return detail || 'Introuvable'
  if (status === 400) return detail || 'Requête invalide'
  if (status === 413) return detail || 'Fichier trop lourd'
  if (status) return 'Erreur du serveur'
  return 'Impossible de joindre le serveur'
}
