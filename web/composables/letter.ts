export type LetterBlock = {
  top: number,
  left: number,
  text: string,
  id: number,
  rotation: number,
  fontSize: number,
  color: string,
  ruled: boolean,
}

export type Stroke = {
  color: string,
  width: number,
  points: number[],
}

export type LetterImage = {
  id: number,
  top: number,
  left: number,
  width: number,
  height: number,
  rotation: number,
  src: string,
}

export type LetterLines = {
  spacing: number,
  opacity: number,
}

export type Letter = {
  blocks: LetterBlock[],
  strokes: Stroke[],
  images: LetterImage[],
  lines: LetterLines | null,
}

export const LETTER_WIDTH = 900
export const LETTER_HEIGHT = 1200

// Même limite que l'API (maxContentLength dans handlers/message.go)
export const LETTER_MAX_LENGTH = 100000

const round = (value: number) => Math.round(value * 10) / 10

// Format stocké dans messages.content
export const serializeLetter = (blocks: LetterBlock[], strokes: Stroke[], images: LetterImage[], lines: LetterLines | null = null) =>
  JSON.stringify({
    v: 1,
    blocks: blocks.map((b) => [round(b.top), round(b.left), round(b.rotation), round(b.fontSize), b.text, b.color, ...(b.ruled ? [1] : [])]),
    strokes: strokes.map((s) => [s.color, round(s.width), simplifyPoints(s.points).map(Math.round)]),
    images: images.map((i) => [round(i.top), round(i.left), round(i.width), round(i.height), i.src, round(i.rotation)]),
    ...(lines && { lines: [round(lines.spacing), Math.round(lines.opacity * 100) / 100] }),
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

// Lignes
export const LINES_SPACING = { min: 32, max: 128, default: 64 }
export const LINES_OPACITY = { min: 0.1, max: 1, default: 0.6 }

const clamp = (value: number, { min, max }: { min: number, max: number }) => Math.min(Math.max(value, min), max)

const parseLines = (raw: unknown): LetterLines | null => {
  if (!Array.isArray(raw)) return null
  const [spacing, opacity] = raw
  if (typeof spacing !== 'number' || typeof opacity !== 'number') return null
  return { spacing: clamp(spacing, LINES_SPACING), opacity: clamp(opacity, LINES_OPACITY) }
}

export const linesBackground = (lines: LetterLines | null) => {
  if (!lines) return undefined
  const { spacing, opacity } = lines
  return `linear-gradient(transparent 0 ${spacing - 3}px, rgba(255, 255, 255, ${opacity}) ${spacing - 3}px ${spacing}px) 40px 0 / calc(100% - 80px) ${spacing}px repeat-y`
}

// Texte posé sur les lignes (Pixelify Sans : 0.92em au-dessus de la ligne de base, 0.28em en dessous)
export const LINES_TEXT_RATIO = 0.5

export const linesFontSize = (spacing: number) => Math.round(spacing * LINES_TEXT_RATIO)

const baselineOffset = (spacing: number) => spacing * (1 - (1 - 1.2 * LINES_TEXT_RATIO) / 2 - 0.92 * LINES_TEXT_RATIO) - 3

export const lineTop = (row: number, spacing: number) => Math.max(0, row) * spacing + baselineOffset(spacing)

export const nearestLine = (top: number, spacing: number) => Math.round((top - baselineOffset(spacing)) / spacing)

export const blockStyle = (block: LetterBlock, lines: LetterLines | null) => ({
  top: block.top + 'px',
  left: block.left + 'px',
  transform: `rotate(${block.rotation}deg)`,
  fontSize: block.fontSize + 'px',
  lineHeight: block.ruled && lines ? lines.spacing + 'px' : undefined,
  color: block.color,
})

// Renvoie null pour un message en texte simple (messages de salon, anciennes lettres)
export const parseLetter = (content: string): Letter | null => {
  if (!content.startsWith('{')) return null
  try {
    const data = JSON.parse(content)
    if (data?.v !== 1 || !Array.isArray(data.blocks)) return null
    return {
      blocks: data.blocks.map(([top, left, rotation, fontSize, text, color = '#ffffff', ruled = 0]: [number, number, number, number, string, string?, number?], id: number) =>
        ({ id, top, left, rotation, fontSize, text, color, ruled: ruled === 1 })),
      strokes: (data.strokes ?? []).map(([color, width, points]: [string, number, number[]]) => ({ color, width, points })),
      images: (data.images ?? []).map(([top, left, width, height, src, rotation = 0]: [number, number, number, number, string, number?], id: number) =>
        ({ id, top, left, width, height, rotation, src })),
      lines: parseLines(data.lines),
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
