/** Ano negativo é a.C.: -500 vira "500 a.C."; os outros saem como número. */
export function formatYear(year: number): string {
  return year < 0 ? `${-year} a.C.` : `${year}`
}

/** "1930–1945", "500–450 a.C." ou "27 a.C.–14 d.C.". */
export function formatYearRange(from: number, to: number): string {
  if (from === to) return formatYear(from)
  if (to < 0) return `${-from}–${-to} a.C.`
  if (from < 0) return `${-from} a.C.–${to} d.C.`
  return `${from}–${to}`
}

/** Antes ou depois de Cristo. */
export type Epoch = 'dC' | 'aC'

export const epochOptions: { value: Epoch; label: string }[] = [
  { value: 'dC', label: 'd.C.' },
  { value: 'aC', label: 'a.C.' },
]

/** Ano com sinal a partir do número digitado e de a.C./d.C.; undefined se não for um inteiro a partir de 1. */
export function signedYear(text: string, epoch: Epoch): number | undefined {
  const n = Number(text.trim())
  if (!text.trim() || !Number.isInteger(n) || n < 1) return undefined
  return epoch === 'aC' ? -n : n
}

/** Número e a.C./d.C. do formulário a partir do ano com sinal. */
export function yearFields(year: number | undefined): { text: string; epoch: Epoch } {
  if (year === undefined) return { text: '', epoch: 'dC' }
  return { text: String(Math.abs(year)), epoch: year < 0 ? 'aC' : 'dC' }
}
