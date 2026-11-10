export type LetterBlock = {
  top: number,
  left: number,
  text: string,
  id: number,
  rotation: number,
  fontSize: number,
}

export type Stroke = {
  color: string,
  width: number,
  points: number[],
}

export const LETTER_WIDTH = 900
export const LETTER_HEIGHT = 1200

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

// Dessin
export const drawStrokes = (ctx: CanvasRenderingContext2D, strokes: Stroke[]) => {
  ctx.clearRect(0, 0, LETTER_WIDTH, LETTER_HEIGHT)
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  for (const stroke of strokes) {
    ctx.strokeStyle = stroke.color
    ctx.lineWidth = stroke.width
    ctx.beginPath()
    ctx.moveTo(stroke.points[0]!, stroke.points[1]!)
    for (let i = 2; i < stroke.points.length; i += 2) {
      ctx.lineTo(stroke.points[i]!, stroke.points[i + 1]!)
    }
    ctx.stroke()
  }
}
