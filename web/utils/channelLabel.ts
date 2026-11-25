// Nom affiché d'un salon : general est le salon libre
export const channelLabel = (c: { name: string }) =>
  c.name === 'general' ? 'Libre' : c.name.charAt(0).toUpperCase() + c.name.slice(1)
