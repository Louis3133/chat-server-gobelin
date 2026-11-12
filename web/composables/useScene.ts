// État partagé entre la scène plein écran (app.vue) et les pages qui la pilotent.
// `system` est le salon réel affiché ; sans lui, la scène montre un décor fait des habitants publics.
export const useScene = () => ({
  system: useState<SolarSystem | null>('scene-system', () => null),
  messages: useState<Message[]>('scene-messages', () => []),
  selectedId: useState<number | null>('scene-selected', () => null),
  // Modale de la planète cliquée (nom, créateur, bio)
  profileOpen: useState<boolean>('scene-profile-open', () => false),
  // Modale du salon ouverte au clic sur son soleil (nom, nombre de planètes, bio)
  channelOpen: useState<boolean>('scene-channel-open', () => false),
  // Fusée cliquée (clé de sa lettre) : son panneau ne s'affiche que pour l'expéditeur d'une lettre en route
  clickedRocket: useState<string | null>('scene-clicked-rocket', () => null),
  decor: useState<SolarSystem | null>('scene-decor', () => null),
  channels: useState<Channel[]>('scene-channels', () => []),
  followedLetter: useState<string | null>('scene-followed-letter', () => null),
  // Fusée posée que le destinataire vient d'ouvrir : son nez reste ouvert tant que la lettre est affichée
  openedLetter: useState<string | null>('scene-opened-letter', () => null),
  // texture : URL de l'image (importée ou en aperçu), null pour la roche par défaut
  customizing: useState<{ id: number; look: PlanetLook; texture: string | null } | null>('scene-customizing', () => null),
  // Espace libre entre les panneaux de la personnalisation (px depuis le haut de l'écran) : la planète s'y centre
  focusArea: useState<{ top: number; bottom: number } | null>('scene-focus-area', () => null),
})

// Identifie la fusée d'une lettre, côté scène comme côté pages
export const letterKey = (m: Message) => `${m.user_id}:${m.recipient_id}:${m.creation_date}`

// Décor de l'accueil : general vu de l'extérieur
export const buildDecor = (users: PublicUser[]): SolarSystem => ({
  channel: { id: 0, name: '' },
  planets: users.filter((u) => u.username !== 'internal-clock').map((u) => ({
    id: u.id,
    username: u.username,
    planet_name: u.planet_name,
    me: false,
    distance: null,
    delay_seconds: null,
    exchanges: 0,
    color: u.planet_color,
    size: u.planet_size,
    rings: u.planet_rings,
    tilt: u.planet_tilt,
    ring_color: u.planet_ring_color,
    ring_size: u.planet_ring_size,
    spin: u.planet_spin,
    moon: u.planet_moon,
    texture_version: u.planet_texture_version,
  })),
  min_distance: 1,
  max_distance: 10,
  time_scale: 1,
})
