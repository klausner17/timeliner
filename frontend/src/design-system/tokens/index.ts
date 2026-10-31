/**
 * Catálogo dos tokens definidos em tokens.css, usado pela vitrine e pelos tipos
 * dos componentes. Os valores ficam só no CSS; aqui ficam nomes e usos.
 */

export type TokenGroup = { title: string; tokens: { name: string; use: string }[] }

export const colorGroups: TokenGroup[] = [
  {
    title: 'Superfícies',
    tokens: [
      { name: '--tl-color-bg', use: 'Fundo da página' },
      { name: '--tl-color-surface', use: 'Cartões, cabeçalho, formulários' },
      { name: '--tl-color-surface-muted', use: 'Blocos dentro de cartões' },
      { name: '--tl-color-surface-sunken', use: 'Fundo de controles segmentados' },
      { name: '--tl-color-surface-inverse', use: 'Botão escuro, chamadas' },
    ],
  },
  {
    title: 'Texto',
    tokens: [
      { name: '--tl-color-text', use: 'Títulos e texto principal' },
      { name: '--tl-color-text-body', use: 'Parágrafos em cartões' },
      { name: '--tl-color-text-muted', use: 'Metadados, legendas' },
      { name: '--tl-color-text-subtle', use: 'Ícones decorativos' },
      { name: '--tl-color-text-inverse', use: 'Texto sobre fundo escuro' },
    ],
  },
  {
    title: 'Bordas',
    tokens: [
      { name: '--tl-color-border', use: 'Borda de cartões' },
      { name: '--tl-color-border-subtle', use: 'Divisórias internas' },
      { name: '--tl-color-border-strong', use: 'Borda de campos e botões' },
      { name: '--tl-color-rail', use: 'Trilho vertical da timeline' },
    ],
  },
  {
    title: 'Ação e estado',
    tokens: [
      { name: '--tl-color-primary', use: 'Ação principal, links' },
      { name: '--tl-color-primary-soft', use: 'Fundo de destaque azul' },
      { name: '--tl-color-success', use: 'Aprovar' },
      { name: '--tl-color-success-soft', use: 'Selo "Aprovada", "Pública"' },
      { name: '--tl-color-warning', use: 'Alerta' },
      { name: '--tl-color-warning-soft', use: 'Selo "Rascunho"' },
      { name: '--tl-color-neutral-soft', use: 'Chips e selos neutros' },
    ],
  },
]

export const trackColors = ['a', 'b', 'c', 'd'] as const
export type TrackColor = (typeof trackColors)[number]

export const trackColorNames: Record<TrackColor, string> = {
  a: 'Timeline A (azul)',
  b: 'Timeline B (laranja)',
  c: 'Timeline C (verde)',
  d: 'Timeline D (roxo)',
}

/** Cores das faixas de período, na ordem em que um período sem cor escolhida recebe uma. */
export const periodColors = ['azul', 'laranja', 'verde', 'roxo', 'amarelo', 'vermelho', 'cinza'] as const
export type PeriodColor = (typeof periodColors)[number]

export const periodColorNames: Record<PeriodColor, string> = {
  azul: 'Azul',
  laranja: 'Laranja',
  verde: 'Verde',
  roxo: 'Roxo',
  amarelo: 'Amarelo',
  vermelho: 'Vermelho',
  cinza: 'Cinza',
}

export const typeScale = [
  { name: '--tl-text-6xl', sample: 'Título de página inicial', display: true },
  { name: '--tl-text-5xl', sample: 'Título de timeline', display: true },
  { name: '--tl-text-4xl', sample: 'Título de tela', display: true },
  { name: '--tl-text-3xl', sample: 'Título de seção', display: true },
  { name: '--tl-text-xl', sample: 'Título de cartão', display: true },
  { name: '--tl-text-lg', sample: 'Título de evento', display: false },
  { name: '--tl-text-base', sample: 'Texto padrão da interface', display: false },
  { name: '--tl-text-md', sample: 'Texto secundário e chips', display: false },
  { name: '--tl-text-sm', sample: 'Metadados de cartão', display: false },
  { name: '--tl-text-xs', sample: 'Rótulos de eixo', display: false },
]

export const spaceScale = [1, 2, 3, 4, 5, 6, 7, 8, 10, 12, 14, 18].map((n) => `--tl-space-${n}`)

export const radiusScale = ['sm', 'md', 'lg', 'xl', '2xl', 'pill'].map((n) => `--tl-radius-${n}`)

/** Lê o valor atual de um token no documento. */
export function readToken(name: string, el: Element = document.documentElement): string {
  return getComputedStyle(el).getPropertyValue(name).trim()
}
