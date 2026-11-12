export interface Channel {
  id: number
  name: string
  owner_id?: number | null
  creation_date?: string
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

export const useChatApi = () => {
  const baseURL = useRuntimeConfig().public.apiBase

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

  // En-tête Bearer obligatoire pour les routes protégées
  const getHeaders = () => {
    const token = getToken()
    return token ? { Authorization: `Bearer ${token}` } : {}
  }

  // Toutes les requêtes passent par ici. L'API ne met pas toujours Content-Type: application/json,
  // donc on force le parsing JSON. Un 401 sur une route protégée vide le token et renvoie au login.
  const request = async <T>(path: string, options: Record<string, any> = {}, auth = true): Promise<T> => {
    try {
      return await $fetch<T>(path, {
        baseURL,
        responseType: 'json',
        ...options,
        headers: auth ? getHeaders() : {},
      })
    } catch (error: any) {
      if (auth && error?.response?.status === 401) {
        clearToken()
        await navigateTo('/')
      }
      throw error
    }
  }

  // POST /login -> renvoie le JWT en texte brut
  const login = async (username: string, password: string): Promise<string> => {
    const token = await request<string>('/login', {
      method: 'POST',
      body: { username, password },
      responseType: 'text',
    }, false)
    setToken(token)
    return token
  }

  // POST /user -> création de compte
  const register = (username: string, password: string) =>
    request<UserMe>('/user', { method: 'POST', body: { username, password } }, false)

  // GET /users et GET /channels sont publics (décor du formulaire de connexion)
  const getUsers = () => request<PublicUser[]>('/users', {}, false)
  const getChannels = () => request<Channel[]>('/channels', {}, false)

  // GET /me -> profil et liste des salons de l'utilisateur
  const getMe = () => request<UserMe>('/me')

  // PUT /me/planet -> enregistrer l'apparence complète de sa planète
  const updatePlanet = (look: PlanetLook) => request<PublicUser>('/me/planet', { method: 'PUT', body: look })

  // PUT /me/planet/texture -> importer une texture (image déjà réduite), DELETE -> revenir à la roche
  const uploadTexture = (image: Blob) => request<PublicUser>('/me/planet/texture', { method: 'PUT', body: image })
  const deleteTexture = () => request<PublicUser>('/me/planet/texture', { method: 'DELETE' })

  // PUT /me/profile -> nom de sa planète, pronoms et bio
  const updateProfile = (profile: Profile) => request<PublicUser>('/me/profile', { method: 'PUT', body: profile })

  // GET /user/texture est public : une simple URL, versionnée pour le cache du navigateur
  const textureUrl = (userId: number, version: number) => `${baseURL}/user/texture?user=${userId}&v=${version}`

  // POST /channel -> créer un salon (réservé aux admins : 403 sinon)
  const createChannel = (name: string) =>
    request<Channel>('/channel', { method: 'POST', body: { name } })

  // POST /channel/enter -> rejoindre un salon existant par son nom
  const enterChannel = async (channelName: string): Promise<void> => {
    await request<void>('/channel/enter', { method: 'POST', body: { channel: channelName } })
  }

  // DELETE /channel/message/rocket?message=ID -> retirer la fusée d'une lettre reçue (la lettre reste)
  // DELETE /channel/message -> annuler sa lettre encore en route
  const cancelLetter = async (messageId: number): Promise<void> => {
    await request<void>('/channel/message', { method: 'DELETE', params: { message: messageId }, responseType: 'text' })
  }

  const removeRocket = async (messageId: number): Promise<void> => {
    await request<void>('/channel/message/rocket', { method: 'DELETE', params: { message: messageId }, responseType: 'text' })
  }

  // DELETE /channel?channel=ID -> supprimer un salon et ses lettres (admins ; "general" est protégé)
  const deleteChannel = async (channelId: number): Promise<void> => {
    await request<void>('/channel', { method: 'DELETE', params: { channel: channelId }, responseType: 'text' })
  }

  // DELETE /user?user=ID -> supprimer un utilisateur et ses lettres (admins ; "internal-clock" est protégé)
  const deleteUser = async (userId: number): Promise<void> => {
    await request<void>('/user', { method: 'DELETE', params: { user: userId }, responseType: 'text' })
  }

  // GET /channel/system -> soleil, planètes et distances de l'utilisateur
  const getSystem = (channelId: number) =>
    request<SolarSystem>('/channel/system', { params: { channel: channelId } })

  const getMessages = async (channelId: number): Promise<Message[]> =>
    (await request<Message[] | null>('/channel/message', { params: { channel: channelId } })) ?? []

  // Sans recipientId : message de salon, livré tout de suite. Avec : lettre, livrée à deliver_at.
  const sendMessage = (channelId: number, content: string, recipientId?: number) =>
    request<Message>('/channel/message', {
      method: 'POST',
      body: { channel: channelId, content, ...(recipientId ? { recipient_id: recipientId } : {}) },
    })

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
