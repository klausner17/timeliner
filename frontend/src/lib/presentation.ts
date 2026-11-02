/*
 * Agrupamento dos eventos por período (usado na timeline vertical) e os slides do modo
 * apresentação: abertura, um slide por período quando ele começa, um por evento e o fim.
 */
import { datePos, innermostPeriod, parentPeriod, periodRange, type PeriodColor, type YearFormat } from '@/design-system'
import type { Era, TimelineEvent } from '@/data/types'

export type PeriodGroup = { name: string; range?: string; color?: PeriodColor; parent?: string; events: TimelineEvent[] }

export const OUTSIDE_PERIODS = 'Fora dos períodos'

/**
 * Agrupa os eventos por período, na ordem do tempo. O evento marcado com um período vai
 * para ele; o sem marca vai para o período mais curto que contém o ano (1940 cai no
 * Estado Novo, não na Era Vargas). Eventos seguidos do mesmo período formam uma seção;
 * os que não cabem em nenhum ficam em "Fora dos períodos", no lugar deles no tempo.
 */
export function groupByPeriod(
  events: readonly TimelineEvent[],
  periods: readonly (Era & { color: PeriodColor })[],
  years: YearFormat,
  axisEnd: number,
): PeriodGroup[] {
  if (!periods.length) return [{ name: '', events: [...events] }]
  const list = [...periods]
  const openEnd = years.openEnd(axisEnd)
  const byName = new Map(list.map((p) => [p.name, p]))
  const out: PeriodGroup[] = []
  for (const e of events) {
    const p = (e.era ? byName.get(e.era) : undefined) ?? innermostPeriod(list, datePos(e.year, e.month, e.day), openEnd)
    const name = p?.name ?? OUTSIDE_PERIODS
    const last = out[out.length - 1]
    if (last?.name === name) {
      last.events.push(e)
      continue
    }
    out.push(
      p
        ? { name, range: periodRange(p, years), color: p.color, parent: parentPeriod<Era>(list, p, openEnd)?.name, events: [e] }
        : { name, events: [e] },
    )
  }
  return out
}

export type Slide =
  | { kind: 'intro'; key: string }
  | { kind: 'period'; key: string; group: PeriodGroup }
  | { kind: 'event'; key: string; event: TimelineEvent; group: PeriodGroup; number: number }
  | { kind: 'end'; key: string }

/**
 * Os slides na ordem: a abertura, e para cada grupo um slide do período (só nos períodos
 * de verdade) seguido dos eventos. O fim só entra quando todos os eventos chegaram. Novas
 * páginas de eventos só acrescentam slides depois dos que já existem, então a posição
 * de quem está apresentando não muda.
 */
export function buildSlides(groups: readonly PeriodGroup[], complete: boolean): Slide[] {
  const out: Slide[] = [{ kind: 'intro', key: 'intro' }]
  let number = 0
  groups.forEach((g, gi) => {
    if (g.name && g.color) out.push({ kind: 'period', key: `p${gi}-${g.name}`, group: g })
    for (const e of g.events) out.push({ kind: 'event', key: `e-${e.id}`, event: e, group: g, number: ++number })
  })
  if (complete) out.push({ kind: 'end', key: 'end' })
  return out
}

export type EventLink = { url: string; label: string }
export type EventMedia = { imageUrl?: string; imageAlt?: string; imageCaption?: string; longDescription?: string; links: EventLink[] }

const str = (v: unknown) => (typeof v === 'string' && v.trim() ? v.trim() : undefined)
const isHttp = (v: string | undefined): boolean => !!v && /^https?:\/\//i.test(v)

/**
 * Imagem, links e descrição longa do evento, quando existirem. Os campos de mídia ainda
 * não fazem parte do tipo do evento, então são lidos sem depender do formato exato:
 * `imageUrl` (ou `image` como texto ou `{ url, alt, caption }`), `imageAlt`,
 * `imageCaption`, `longDescription` e `links` (textos ou `{ url, label | title }`).
 * A fonte (`sourceUrl`) entra como o primeiro link.
 */
export function mediaOf(e: TimelineEvent): EventMedia {
  const r = e as TimelineEvent & Record<string, unknown>
  const img = r.image && typeof r.image === 'object' ? (r.image as Record<string, unknown>) : undefined
  const imageUrl = str(r.imageUrl) ?? str(r.image) ?? str(img?.url)
  const links: EventLink[] = []
  const seen = new Set<string>()
  const add = (url: string | undefined, label: string | undefined) => {
    if (!url || !isHttp(url) || seen.has(url)) return
    seen.add(url)
    links.push({ url, label: label ?? hostOf(url) })
  }
  add(str(e.sourceUrl), 'Fonte')
  if (Array.isArray(r.links)) {
    for (const l of r.links) {
      if (typeof l === 'string') add(str(l), undefined)
      else if (l && typeof l === 'object') {
        const o = l as Record<string, unknown>
        add(str(o.url) ?? str(o.href), str(o.label) ?? str(o.title))
      }
    }
  }
  const longDescription = str(r.longDescription) ?? str(r.details)
  return {
    imageUrl: imageUrl && (isHttp(imageUrl) || imageUrl.startsWith('/')) ? imageUrl : undefined,
    imageAlt: str(r.imageAlt) ?? str(img?.alt),
    imageCaption: str(r.imageCaption) ?? str(img?.caption),
    longDescription: longDescription && longDescription !== e.description.trim() ? longDescription : undefined,
    links,
  }
}

function hostOf(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

/** Posição de 0 a 1 do evento entre o início e o fim do eixo. */
export function axisFraction(e: Pick<TimelineEvent, 'year' | 'month' | 'day'>, start: number, end: number): number {
  if (end <= start) return 0
  return Math.min(1, Math.max(0, (datePos(e.year, e.month, e.day) - start) / (end - start)))
}
