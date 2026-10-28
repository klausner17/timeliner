import type { TimelineEvent } from './types'

/**
 * Milhares de eventos inventados, para a vitrine mostrar faixas de anos e paginação
 * funcionando. Saem sempre iguais (gerador com semente fixa).
 */
export function generatedEvents(count = 20000): TimelineEvent[] {
  let x = 42
  const rand = () => ((x = (x * 1103515245 + 12345) % 2147483648) / 2147483648)
  const events: TimelineEvent[] = []
  for (let i = 0; i < count; i++) {
    // Mais eventos perto de hoje, como numa linha do tempo histórica de verdade.
    const year = 2025 - Math.floor(4000 * rand() * rand() * rand())
    events.push({ id: `g${i + 1}`, year, title: `Evento ${i + 1}`, description: 'Gerado para demonstrar timelines grandes.', by: 'memoria' })
  }
  return events
}
