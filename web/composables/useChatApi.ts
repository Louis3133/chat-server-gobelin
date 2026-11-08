export interface Channel {
  id: number
  name: string
  owner_id?: number | null
  creation_date?: string
  member_ids?: number[]
}

export interface UserMe {
  id: number
  username: string
  channels: Channel[]
  is_admin?: boolean
}

export interface PublicUser {
  id: number
  username: string
}

export interface Message {
  content: string
  user_id: number
  user: string
  recipient_id: number | null
  recipient: string | null
  creation_date: string
  deliver_at: string
  in_transit: boolean
}

export interface Planet {
  id: number
  username: string
  me: boolean
  distance: number | null
  delay_seconds: number | null
  exchanges: number
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

  // POST /channel -> créer un salon (réservé aux admins : 403 sinon)
  const createChannel = (name: string) =>
    request<Channel>('/channel', { method: 'POST', body: { name } })

  // POST /channel/enter -> rejoindre un salon existant par son nom
  const enterChannel = async (channelName: string): Promise<void> => {
    await request<void>('/channel/enter', { method: 'POST', body: { channel: channelName } })
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
    createChannel,
    enterChannel,
    deleteChannel,
    deleteUser,
    getSystem,
    getMessages,
    sendMessage,
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
  if (status) return 'Erreur du serveur'
  return 'Impossible de joindre le serveur'
}
