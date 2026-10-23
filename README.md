# Timeliner

Site para criar timelines, contribuir com as de outras pessoas e comparar duas histórias no mesmo eixo de tempo.

## Estrutura

| Pasta | O que tem |
| --- | --- |
| `frontend/` | App web em Vue 3, Vite e TypeScript, com o design system |
| `backend/` | API em Go (`net/http` da biblioteca padrão) |

## Rodando

Frontend:

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173
npm test           # testes com Vitest
npm run build      # checagem de tipos + build de produção
```

Backend:

```bash
cd backend
go run ./cmd/api   # http://localhost:8080/api/health
go test ./...
```

Em desenvolvimento, o Vite repassa `/api` para `http://localhost:8080`, então o frontend chama a API pelo mesmo endereço.

## Telas

Ainda sem backend: os dados vêm de `frontend/src/data/mock.ts` e ficam num store reativo (`frontend/src/data/store.ts`) salvo no `localStorage`. Quando a API existir, só o store muda.

| Rota | Tela |
| --- | --- |
| `/` | Início e busca (`?q=` e `?cat=`) |
| `/t/:slug` | Ver timeline, na vertical ou na horizontal, com filtro por período |
| `/t/:slug/contribuir` | Sugerir evento; para o dono, revisar sugestões |
| `/t/:slug/editar` | Editar a timeline e os eventos (só o dono) |
| `/nova` | Cria um rascunho e abre o editor |
| `/comparar?a=&b=&de=&ate=` | Comparar até 4 timelines no mesmo eixo |
| `/perfil` | Perfil com timelines, contribuições, comparações salvas e quem você segue |
| `/design-system` | Vitrine do design system |

O usuário logado é fixo (`@klausner`) até existir login.

## Design system

Tudo fica em `frontend/src/design-system/` e é importado por `@/design-system`:

- `tokens/tokens.css`: cores, tipografia, espaçamento, raios e tamanhos de controle como variáveis CSS (`--tl-*`).
- `tokens/base.css`: estilos globais (fonte, fundo, foco) e utilitários `.tl-container` e `.tl-visually-hidden`.
- `tokens/index.ts`: catálogo dos tokens e o tipo `TrackColor` (`'a' | 'b' | 'c' | 'd'`).
- `components/`: componentes `Tl*`.

A vitrine com todos os tokens e componentes está em `/design-system` (`frontend/src/views/DesignSystemView.vue`).

### Regras

- Componentes usam só tokens, nunca cores em hex. Um teste confere isso.
- Cores de timeline: A azul (`--tl-track-a`), B laranja (`--tl-track-b`), C verde e D roxo para comparações maiores.
- Títulos, anos e números em Space Grotesk; o resto em Inter.
- Alvos de toque com pelo menos 44 px e rótulos reais em todos os controles.

### Componentes

| Grupo | Componentes |
| --- | --- |
| Base | `TlButton`, `TlIcon`, `TlLogo`, `TlCard`, `TlBadge`, `TlAvatar`, `TlStat` |
| Formulário | `TlField`, `TlInput`, `TlTextarea`, `TlSelect`, `TlCheckbox`, `TlRadioGroup`, `TlColorPicker` |
| Navegação e seleção | `TlAppHeader`, `TlTabs`, `TlSegmented`, `TlChip` |
| Timeline | `TlTimelineCard`, `TlMiniTrack`, `TlTrackLabel`, `TlEventItem`, `TlOverviewBar`, `TlCompareTracks` |
