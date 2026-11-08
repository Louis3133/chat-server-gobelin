export type LetterBlock = {
  top: number,
  left: number,
  text: string,
  id: number,
  rotation: number,
  fontSize: number,
}

// Même limite que l'API (maxContentLength dans handlers/message.go)
export const LETTER_MAX_LENGTH = 5000

const round = (value: number) => Math.round(value * 10) / 10

// Format stocké dans messages.content
export const serializeLetter = (blocks: LetterBlock[]) =>
  JSON.stringify({
    v: 1,
    blocks: blocks.map((b) => [round(b.top), round(b.left), round(b.rotation), round(b.fontSize), b.text]),
  })

// Renvoie null pour un message en texte simple (messages de salon, anciennes lettres)
export const parseLetter = (content: string): LetterBlock[] | null => {
  if (!content.startsWith('{')) return null
  try {
    const data = JSON.parse(content)
    if (data?.v !== 1 || !Array.isArray(data.blocks)) return null
    return data.blocks.map(([top, left, rotation, fontSize, text]: [number, number, number, number, string], id: number) =>
      ({ id, top, left, rotation, fontSize, text }))
  } catch {
    return null
  }
}
