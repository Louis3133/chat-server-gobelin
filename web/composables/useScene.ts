// État partagé entre la scène plein écran (app.vue) et les pages qui la pilotent.
// `system` est le salon réel affiché ; sans lui, la scène montre un décor fait des habitants publics.
export const useScene = () => ({
  system: useState<SolarSystem | null>('scene-system', () => null),
  messages: useState<Message[]>('scene-messages', () => []),
  selectedId: useState<number | null>('scene-selected', () => null),
  decor: useState<SolarSystem | null>('scene-decor', () => null),
  channels: useState<Channel[]>('scene-channels', () => []),
})

// Décor de l'accueil : general vu de l'extérieur
export const buildDecor = (users: PublicUser[]): SolarSystem => ({
  channel: { id: 0, name: '' },
  planets: users.filter((u) => u.username !== 'internal-clock').map((u) => ({
    id: u.id,
    username: u.username,
    me: false,
    distance: null,
    delay_seconds: null,
    exchanges: 0,
  })),
  min_distance: 1,
  max_distance: 10,
  time_scale: 1,
})
