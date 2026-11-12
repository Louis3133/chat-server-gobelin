export type LetterBlock = {
  top: number,
  left: number,
  text: string,
  id: number,
  rotation: number,
  fontSize: number,
  color: string,
}

export type Stroke = {
  color: string,
  width: number,
  points: number[],
}

export type Letter = {
  blocks: LetterBlock[],
  strokes: Stroke[],
}

export const LETTER_WIDTH = 900
export const LETTER_HEIGHT = 1200

// Même limite que l'API (maxContentLength dans handlers/message.go)
export const LETTER_MAX_LENGTH = 100000

const round = (value: number) => Math.round(value * 10) / 10

// Format stocké dans messages.content
export const serializeLetter = (blocks: LetterBlock[], strokes: Stroke[]) =>
  JSON.stringify({
    v: 1,
    blocks: blocks.map((b) => [round(b.top), round(b.left), round(b.rotation), round(b.fontSize), b.text, b.color]),
    strokes: strokes.map((s) => [s.color, round(s.width), simplifyPoints(s.points).map(Math.round)]),
  })

// Les points à moins de 3 px du précédent n'apportent rien au tracé
const simplifyPoints = (points: number[]) => {
  const kept = points.slice(0, 2)
  for (let i = 2; i < points.length; i += 2) {
    const isLast = i === points.length - 2
    const far = Math.hypot(points[i]! - kept[kept.length - 2]!, points[i + 1]! - kept[kept.length - 1]!) >= 3
    if (far || isLast) kept.push(points[i]!, points[i + 1]!)
  }
  return kept
}

// Renvoie null pour un message en texte simple (messages de salon, anciennes lettres)
export const parseLetter = (content: string): Letter | null => {
  if (!content.startsWith('{')) return null
  try {
    const data = JSON.parse(content)
    if (data?.v !== 1 || !Array.isArray(data.blocks)) return null
    return {
      blocks: data.blocks.map(([top, left, rotation, fontSize, text, color = '#ffffff']: [number, number, number, number, string, string?], id: number) =>
        ({ id, top, left, rotation, fontSize, text, color })),
      strokes: (data.strokes ?? []).map(([color, width, points]: [string, number, number[]]) => ({ color, width, points })),
    }
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
