/** Ícones de traço usados nas telas. Cada um é uma lista de paths SVG. */
type IconDef = { viewBox: string; paths: string[]; fill?: boolean }

const defs = {
  plus: { viewBox: '0 0 16 16', paths: ['M8 3v10', 'M3 8h10'] },
  compare: { viewBox: '0 0 16 16', paths: ['M2 5h12', 'M2 11h12'] },
  swap: { viewBox: '0 0 20 20', paths: ['M4 7h12l-3-3', 'M16 13H4l3 3'] },
  share: { viewBox: '0 0 18 18', paths: ['M9 2v10', 'M5 6l4-4 4 4', 'M3 11v4h12v-4'] },
  back: { viewBox: '0 0 14 14', paths: ['M9 2L4 7l5 5'] },
  search: { viewBox: '0 0 18 18', paths: ['M8 14A6 6 0 1 0 8 2a6 6 0 0 0 0 12z', 'M16 16l-3.5-3.5'] },
  check: { viewBox: '0 0 16 16', paths: ['M3 8.5l3.5 3.5L13 4.5'] },
  close: { viewBox: '0 0 16 16', paths: ['M4 4l8 8', 'M12 4l-8 8'] },
  edit: { viewBox: '0 0 18 18', paths: ['M12 3l3 3-8.5 8.5H3.5v-3z'] },
  trash: { viewBox: '0 0 18 18', paths: ['M3 5h12', 'M7 5V3h4v2', 'M5 5l1 10h6l1-10'] },
  image: {
    viewBox: '0 0 24 24',
    paths: ['M5 4h14a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z', 'M9 12a2 2 0 1 0 0-4 2 2 0 0 0 0 4z', 'M21 16l-5-5-9 9'],
  },
  drag: {
    viewBox: '0 0 12 18',
    fill: true,
    paths: [
      'M3 1.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z',
      'M9 1.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z',
      'M3 7.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z',
      'M9 7.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z',
      'M3 13.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z',
      'M9 13.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3z',
    ],
  },
} satisfies Record<string, IconDef>

export type IconName = keyof typeof defs
export const icons: Record<IconName, IconDef> = defs
export const iconNames = Object.keys(icons) as IconName[]
