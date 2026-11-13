// Règles des lettres
export interface LetterRules {
  minDistance: number
  maxDistance: number
  minDelayMs: number
  maxDelayMs: number
  approach: number
  timeScale: number
}

const UNITS: Record<string, number> = { ms: 1, s: 1000, m: 60_000, h: 3_600_000 }

export const parseDuration = (raw: string | number): number => {
  if (typeof raw === 'number') return raw * 1000
  const parts = [...raw.trim().matchAll(/(\d+(?:\.\d+)?)(ms|s|m|h)/g)]
  if (!parts.length || parts.map((p) => p[0]).join('') !== raw.trim()) {
    throw new Error(`letters: ${JSON.stringify(raw)} n'est pas une durée (exemples : 90s, 3h)`)
  }
  return parts.reduce((total, [, value, unit]) => total + Number(value) * UNITS[unit!]!, 0)
}

export const makeLetterRules = (config: Record<string, unknown>): LetterRules => {
  const rules: LetterRules = {
    minDistance: Number(config.letterMinDistance),
    maxDistance: Number(config.letterMaxDistance),
    minDelayMs: parseDuration(config.letterMinDelay as string),
    maxDelayMs: parseDuration(config.letterMaxDelay as string),
    approach: Number(config.letterApproach),
    timeScale: Number(config.letterTimeScale),
  }
  if (!(rules.minDistance > 0)) throw new Error('letters: la distance minimale doit être > 0')
  if (!(rules.maxDistance > rules.minDistance)) throw new Error('letters: la distance maximale doit dépasser la minimale')
  if (rules.minDelayMs < 0 || rules.maxDelayMs < rules.minDelayMs) throw new Error('letters: délais incohérents')
  if (!(rules.approach >= 0 && rules.approach < 1)) throw new Error('letters: le rapprochement doit être entre 0 inclus et 1 exclus')
  if (!(rules.timeScale > 0)) throw new Error('letters: le facteur de temps doit être > 0')
  return rules
}

export const letterDelayMs = (rules: LetterRules, distance: number) => {
  const t = Math.min(1, Math.max(0, (distance - rules.minDistance) / (rules.maxDistance - rules.minDistance)))
  return (rules.minDelayMs + t * (rules.maxDelayMs - rules.minDelayMs)) / rules.timeScale
}

export const approachedDistance = (rules: LetterRules, distance: number) =>
  Math.max(rules.minDistance, distance * (1 - rules.approach))
