<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  colorGroups,
  iconNames,
  radiusScale,
  readToken,
  spaceScale,
  trackColorNames,
  trackColors,
  typeScale,
  TlAppHeader,
  TlAvatar,
  TlBadge,
  TlButton,
  TlCard,
  TlCheckbox,
  TlChip,
  TlColorPicker,
  TlCompareTracks,
  TlEventItem,
  TlField,
  TlIcon,
  TlInput,
  TlLogo,
  TlOverviewBar,
  TlRadioGroup,
  TlSegmented,
  TlSelect,
  TlStat,
  TlTabs,
  TlTextarea,
  TlTimelineCard,
  TlTrackLabel,
  type CompareLane,
  type TrackColor,
} from '@/design-system'
import { generatedTimeline } from '@/data/mock'
import { chronological, windowOf } from '@/data/windowing'
import DsDemo from './design-system/DsDemo.vue'
import DsSection from './design-system/DsSection.vue'

const sections = [
  { id: 'cores', label: 'Cores' },
  { id: 'tipografia', label: 'Tipografia' },
  { id: 'espaco', label: 'Espaço e raios' },
  { id: 'icones', label: 'Ícones' },
  { id: 'botoes', label: 'Botões' },
  { id: 'campos', label: 'Campos' },
  { id: 'selecao', label: 'Seleção' },
  { id: 'selos', label: 'Selos e avatares' },
  { id: 'cartoes', label: 'Cartões' },
  { id: 'timeline', label: 'Timeline' },
  { id: 'comparar', label: 'Comparação' },
  { id: 'timeline-grande', label: 'Timelines grandes' },
  { id: 'cabecalho', label: 'Cabeçalho' },
]

// Valores reais dos tokens, lidos do CSS para não duplicar os hex aqui.
const values = ref<Record<string, string>>({})
onMounted(() => {
  const names = [
    ...colorGroups.flatMap((g) => g.tokens.map((t) => t.name)),
    ...trackColors.flatMap((c) => [`--tl-track-${c}`, `--tl-track-${c}-soft`]),
    ...typeScale.map((t) => t.name),
    ...spaceScale,
    ...radiusScale,
  ]
  values.value = Object.fromEntries(names.map((n) => [n, readToken(n)]))
})

// Estado dos exemplos interativos
const titulo = ref('Renúncia e impeachment de Fernando Collor')
const descricao = ref('Collor renuncia no dia do julgamento no Senado, que mantém a perda dos direitos políticos por oito anos.')
const mes = ref('Dezembro')
const aproximada = ref(false)
const periodo = ref(true)
const visibilidade = ref('publica')
const era = ref('Tudo')
const modo = ref('trilhas')
const aba = ref('timelines')
const cor = ref<TrackColor>('b')
const selecionado = ref('b-1992')
const erroFonte = 'Informe um link para a fonte.'
const urlPlaceholder = 'https://'

const eras = ['Tudo', 'Colônia', 'Império', 'República Velha e Era Vargas', 'Ditadura militar', 'Nova República']

const lanes: CompareLane[] = [
  {
    id: 'brasil',
    name: 'História do Brasil',
    color: 'a',
    events: [
      { id: 'a-1985', year: 1985, title: 'Fim do regime militar' },
      { id: 'a-1988', year: 1988, title: 'Constituição Cidadã' },
      { id: 'a-1989', year: 1989, title: 'Primeira eleição direta desde 1960' },
      { id: 'a-1994', year: 1994, title: 'Plano Real' },
      { id: 'a-2002', year: 2002, title: 'Lula é eleito presidente' },
      { id: 'a-2016', year: 2016, title: 'Impeachment de Dilma Rousseff' },
    ],
  },
  {
    id: 'escandalos',
    name: 'Escândalos políticos',
    color: 'b',
    events: [
      { id: 'b-1992', year: 1992, title: 'Esquema PC Farias e queda de Collor' },
      { id: 'b-1993', year: 1993, title: 'Anões do Orçamento' },
      { id: 'b-2005', year: 2005, title: 'Mensalão' },
      { id: 'b-2014', year: 2014, title: 'Operação Lava Jato' },
      { id: 'b-2017', year: 2017, title: 'Delação da JBS' },
    ],
  },
]

const overviewYears = [1500, 1549, 1580, 1630, 1654, 1695, 1720, 1763, 1789, 1808, 1815, 1822, 1831, 1840, 1850, 1865, 1888, 1889, 1910, 1922, 1930, 1937, 1945, 1954, 1964, 1968, 1985, 1988, 1994, 2002, 2016]

// Timeline grande: 20 mil eventos gerados, com zoom e faixas de anos.
const big = chronological(generatedTimeline().events)
const bigView = ref<[number, number]>([-2000, 2030])
const bigStart = computed({ get: () => bigView.value[0], set: (v) => (bigView.value = [v, bigView.value[1]]) })
const bigEnd = computed({ get: () => bigView.value[1], set: (v) => (bigView.value = [bigView.value[0], v]) })
const bigWindow = computed(() => windowOf(big, bigView.value[0], bigView.value[1], { buckets: 60 }))
const bigLanes = computed<CompareLane[]>(() => [{ id: 'grande', name: '20 mil eventos', color: 'd', events: bigWindow.value.events, buckets: bigWindow.value.buckets }])
const bigOverview = windowOf(big, -2000, 2030, { buckets: 100, maxEvents: 1 }).buckets
const bigSelected = ref<string>()

const usage = {
  tokens: `/* Use sempre os tokens semânticos */
.minha-caixa {
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-xl);
  padding: var(--tl-space-5);
}`,
  buttons: `<TlButton variant="primary" icon="plus">Sugerir evento</TlButton>
<TlButton variant="secondary" :to="{ name: 'comparar' }" icon="compare">Comparar com…</TlButton>
<TlButton icon-only icon="share" label="Compartilhar" />`,
  fields: `<TlField label="Título" v-slot="{ id }">
  <TlInput :id="id" v-model="titulo" />
</TlField>`,
  compare: `<TlCompareTracks :lanes="lanes" :start="1980" :end="2020" :highlight-year="1992" v-model:selected="eventoId" />`,
  big: `<TlOverviewBar v-model:window="janela" :start="-2000" :end="2030" :buckets="histograma" interactive />
<TlCompareTracks v-model:start="inicio" v-model:end="fim" :lanes="lanes" :min="-2000" :max="2030" zoomable />
<!-- lanes[i].buckets: [{ from, to, count, sample }] quando a API devolve mode=faixas -->`,
}
</script>

<template>
  <div class="ds">
    <TlAppHeader
      :nav="[
        { label: 'Explorar', to: '/' },
        { label: 'Comparar', to: '/' },
        { label: 'Design system', to: '/design-system', current: true },
      ]"
      create-to="/"
      profile-to="/"
      user-name="Klausner"
    />

    <section class="ds-hero">
      <div class="tl-container ds-hero__inner">
        <p class="ds-eyebrow">Timeliner · design system</p>
        <h1 class="ds-hero__title">Tokens e componentes</h1>
        <p class="ds-hero__lead">
          A base visual das telas do Timeliner em Vue: fundo claro, títulos em Space Grotesk, texto em Inter, azul para a timeline A e
          laranja para a B. Tudo aqui vem de <code>src/design-system</code>.
        </p>
      </div>
    </section>

    <div class="tl-container ds-layout">
      <nav class="ds-toc" aria-label="Seções do design system">
        <a v-for="s in sections" :key="s.id" :href="`#${s.id}`">{{ s.label }}</a>
      </nav>

      <main class="ds-main">
        <!-- Cores -->
        <DsSection
          id="cores"
          title="Cores"
          description="Os componentes usam só os tokens semânticos (--tl-color-*). As cores de trilha identificam cada timeline numa comparação: A azul, B laranja, e C e D para quando houver mais de duas."
          :usage="usage.tokens"
        >
          <div class="ds-tracks">
            <div v-for="c in trackColors" :key="c" class="ds-track">
              <div class="ds-track__chip" :style="{ background: `var(--tl-track-${c})` }"></div>
              <div class="ds-track__soft" :style="{ background: `var(--tl-track-${c}-soft)` }"></div>
              <div class="ds-track__text">
                <strong>{{ trackColorNames[c] }}</strong>
                <code>--tl-track-{{ c }}</code>
                <span class="ds-muted">{{ values[`--tl-track-${c}`] }} · soft {{ values[`--tl-track-${c}-soft`] }}</span>
              </div>
            </div>
          </div>
          <div v-for="g in colorGroups" :key="g.title" class="ds-color-group">
            <h3 class="ds-h3">{{ g.title }}</h3>
            <div class="ds-swatches">
              <div v-for="t in g.tokens" :key="t.name" class="ds-swatch">
                <div class="ds-swatch__color" :style="{ background: `var(${t.name})` }"></div>
                <code>{{ t.name }}</code>
                <span class="ds-muted">{{ values[t.name] }} · {{ t.use }}</span>
              </div>
            </div>
          </div>
        </DsSection>

        <!-- Tipografia -->
        <DsSection
          id="tipografia"
          title="Tipografia"
          description="Space Grotesk (500–700) para títulos, anos e números. Inter (400–600) para todo o resto. Títulos grandes usam espaçamento de letras -0.02em."
        >
          <DsDemo stack>
            <div v-for="t in typeScale" :key="t.name" class="ds-type">
              <span class="ds-type__meta"><code>{{ t.name }}</code> {{ values[t.name] }}</span>
              <span
                class="ds-type__sample"
                :style="{ fontSize: `var(${t.name})`, fontFamily: t.display ? 'var(--tl-font-display)' : 'var(--tl-font-body)', fontWeight: t.display ? 700 : 400 }"
                >{{ t.sample }}</span
              >
            </div>
          </DsDemo>
          <DsDemo label="Sobrelinha">
            <p class="ds-eyebrow">Timelines colaborativas</p>
          </DsDemo>
        </DsSection>

        <!-- Espaço -->
        <DsSection id="espaco" title="Espaço e raios" description="Escala de 4 em 4 px. Raios pequenos em campos, médios em cartões, pílula em botões e chips.">
          <DsDemo label="Espaçamento" stack>
            <div v-for="s in spaceScale" :key="s" class="ds-space">
              <code>{{ s }}</code>
              <span class="ds-space__bar" :style="{ width: `var(${s})` }"></span>
              <span class="ds-muted">{{ values[s] }}</span>
            </div>
          </DsDemo>
          <DsDemo label="Raios">
            <div v-for="r in radiusScale" :key="r" class="ds-radius">
              <div class="ds-radius__box" :style="{ borderRadius: `var(${r})` }"></div>
              <code>{{ r }}</code>
            </div>
          </DsDemo>
        </DsSection>

        <!-- Ícones -->
        <DsSection id="icones" title="Ícones" description="Ícones de traço em SVG. Herdam a cor do texto; quando ficam sozinhos num botão, o botão recebe o rótulo.">
          <DsDemo>
            <div v-for="n in iconNames" :key="n" class="ds-icon">
              <TlIcon :name="n" :size="20" />
              <code>{{ n }}</code>
            </div>
          </DsDemo>
          <DsDemo label="Logo">
            <TlLogo />
            <TlLogo :wordmark="false" :size="40" />
          </DsDemo>
        </DsSection>

        <!-- Botões -->
        <DsSection
          id="botoes"
          title="Botões"
          description="Pílula nas ações de página e arredondado dentro de formulários. Viram link quando recebem to (rota) ou href. Altura mínima de 44 px."
          :usage="usage.buttons"
        >
          <DsDemo label="Variantes">
            <TlButton variant="primary" icon="plus">Sugerir evento</TlButton>
            <TlButton variant="dark" icon="plus">Criar timeline</TlButton>
            <TlButton variant="secondary" icon="compare">Comparar com…</TlButton>
            <TlButton variant="ghost">Cancelar</TlButton>
            <TlButton variant="success">Aprovar</TlButton>
            <TlButton variant="dashed">+ Adicionar timeline</TlButton>
          </DsDemo>
          <DsDemo label="Tamanhos e formatos">
            <TlButton variant="secondary" size="sm">Pequeno</TlButton>
            <TlButton variant="secondary">Médio</TlButton>
            <TlButton variant="primary" size="lg">Buscar</TlButton>
            <TlButton variant="secondary" shape="rounded">Convidar editores</TlButton>
            <TlButton variant="secondary" icon-only icon="share" label="Compartilhar" />
            <TlButton variant="secondary" size="lg" icon-only icon="swap" label="Trocar A e B" />
            <TlButton variant="primary" disabled>Desativado</TlButton>
          </DsDemo>
          <div class="ds-inverse">
            <div>
              <h3 class="ds-inverse__title">Tem uma história para contar?</h3>
              <p class="ds-inverse__text">Botão claro sobre fundo escuro.</p>
            </div>
            <TlButton variant="inverse">Criar minha timeline</TlButton>
          </div>
        </DsSection>

        <!-- Campos -->
        <DsSection
          id="campos"
          title="Campos"
          description="TlField cuida do rótulo, da dica e do id; o campo recebe o id pelo slot. Erros trocam a dica e marcam o campo."
          :usage="usage.fields"
        >
          <DsDemo stack>
            <div class="ds-grid-3">
              <TlField v-slot="{ id }" label="Dia" tone="muted"><TlInput :id="id" model-value="29" /></TlField>
              <TlField v-slot="{ id }" label="Mês" tone="muted">
                <TlSelect :id="id" v-model="mes" :options="['Outubro', 'Novembro', 'Dezembro']" />
              </TlField>
              <TlField v-slot="{ id }" label="Ano" tone="muted"><TlInput :id="id" model-value="1992" /></TlField>
            </div>
            <div>
              <TlCheckbox v-model="aproximada" label="Data aproximada (mostrar só o ano)" />
              <TlCheckbox v-model="periodo" label="É um período (tem data de fim)" />
            </div>
            <TlField v-slot="{ id }" label="Título"><TlInput :id="id" v-model="titulo" /></TlField>
            <TlField v-slot="{ id, describedby }" label="Descrição" hint="Conte o essencial em poucas linhas.">
              <TlTextarea :id="id" v-model="descricao" :aria-describedby="describedby" :maxlength="280" />
            </TlField>
            <TlField v-slot="{ id, describedby, invalid }" label="Fonte (obrigatória)" :error="erroFonte">
              <TlInput :id="id" type="url" :placeholder="urlPlaceholder" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
          </DsDemo>
          <DsDemo label="Busca">
            <form class="ds-search" role="search" @submit.prevent>
              <label for="ds-busca" class="tl-visually-hidden">Buscar timelines</label>
              <TlInput id="ds-busca" type="search" size="lg" shape="pill" placeholder="Buscar: Brasil Império, Copa do Mundo, internet…" />
              <TlButton type="submit" variant="primary" size="lg">Buscar</TlButton>
            </form>
          </DsDemo>
          <DsDemo label="Opções">
            <TlRadioGroup
              v-model="visibilidade"
              legend="Visibilidade"
              :options="[
                { value: 'publica', label: 'Pública' },
                { value: 'link', label: 'Só com o link' },
                { value: 'privada', label: 'Privada' },
              ]"
            />
          </DsDemo>
        </DsSection>

        <!-- Seleção -->
        <DsSection id="selecao" title="Seleção" description="Chips de filtro, chips de categoria, controle segmentado para modos de visualização, abas e seletor de cor.">
          <DsDemo label="Chips de filtro (período)">
            <TlChip v-for="e in eras" :key="e" :pressed="era === e" @click="era = e">{{ e }}</TlChip>
          </DsDemo>
          <DsDemo label="Chips de categoria e tags">
            <TlChip kind="tag" href="#selecao">História</TlChip>
            <TlChip kind="tag" href="#selecao">Política</TlChip>
            <TlChip kind="tag" href="#selecao">Esporte</TlChip>
            <TlChip kind="tag" size="sm">brasil</TlChip>
            <TlChip kind="tag" size="sm">educação</TlChip>
          </DsDemo>
          <DsDemo label="Controle segmentado">
            <TlSegmented
              v-model="modo"
              label="Modo de comparação"
              :options="[
                { value: 'trilhas', label: 'Trilhas' },
                { value: 'lado', label: 'Lado a lado' },
              ]"
            />
          </DsDemo>
          <DsDemo label="Abas" stack>
            <TlTabs
              v-model="aba"
              label="Seções do perfil"
              :tabs="[
                { value: 'timelines', label: 'Timelines', count: 4 },
                { value: 'contribuicoes', label: 'Contribuições', count: 18 },
                { value: 'comparacoes', label: 'Comparações salvas' },
                { value: 'seguindo', label: 'Seguindo' },
              ]"
            >
              <p class="ds-muted ds-tab-panel">Conteúdo da aba “{{ aba }}”.</p>
            </TlTabs>
          </DsDemo>
          <DsDemo label="Cor da timeline">
            <TlColorPicker v-model="cor" />
            <TlTrackLabel :color="cor">{{ trackColorNames[cor] }}</TlTrackLabel>
          </DsDemo>
        </DsSection>

        <!-- Selos -->
        <DsSection id="selos" title="Selos e avatares" description="Selos indicam estado de timelines e contribuições. Avatares mostram a inicial e o nome no title.">
          <DsDemo label="Estados">
            <TlBadge tone="success">Pública</TlBadge>
            <TlBadge tone="neutral">Privada</TlBadge>
            <TlBadge tone="warning">Rascunho</TlBadge>
            <TlBadge tone="success" size="md">Aprovada</TlBadge>
            <TlBadge tone="info" size="md">Pendente</TlBadge>
            <TlBadge tone="neutral" size="md">Recusada</TlBadge>
            <TlBadge tone="success">Aberta a contribuições</TlBadge>
          </DsDemo>
          <DsDemo label="Avatares">
            <TlAvatar name="@historia.br" size="xs" tint="a" />
            <TlAvatar name="@ana.m" size="sm" tint="b" />
            <TlAvatar name="@joaovitor" size="sm" tint="c" />
            <TlAvatar name="@memoria" size="sm" tint="d" />
            <TlAvatar name="Mais 5 contribuidores" initial="+5" size="sm" />
            <TlAvatar name="Klausner" size="md" tint="a" />
            <TlAvatar name="Klausner" size="lg" tint="a" />
            <TlAvatar name="Klausner" size="xl" tint="a" />
          </DsDemo>
          <DsDemo label="Números do perfil" stack>
            <dl class="ds-stats">
              <TlStat label="Timelines" :value="4" />
              <TlStat label="Eventos criados" :value="73" />
              <TlStat label="Contribuições aprovadas" :value="18" />
              <TlStat label="Seguidores" :value="36" />
            </dl>
          </DsDemo>
        </DsSection>

        <!-- Cartões -->
        <DsSection id="cartoes" title="Cartões" description="TlCard é a superfície base. TlTimelineCard é o cartão das listas de timelines, com a faixa de pontos na cor da timeline.">
          <div class="ds-cards">
            <TlTimelineCard
              title="História do Brasil"
              description="Do descobrimento à Nova República, com os marcos políticos e sociais."
              color="a"
              :points="[2, 22, 48, 66, 90]"
              range="1500–2024"
              :events="86"
              :contributors="12"
              author="@historia.br"
              href="#cartoes"
            />
            <TlTimelineCard
              title="Escândalos políticos do Brasil"
              description="Casos de corrupção que marcaram a política brasileira desde a redemocratização."
              color="b"
              :points="[10, 18, 35, 58, 82]"
              range="1988–2024"
              :events="41"
              :contributors="7"
              author="@klausner"
              :badge="{ label: 'Pública', tone: 'success' }"
              href="#cartoes"
            />
            <TlCard tone="dashed" href="#cartoes" style="min-height: 200px">
              <TlIcon name="plus" :size="24" />
              Nova timeline
            </TlCard>
          </div>
          <div class="ds-cards">
            <TlCard href="#cartoes">
              <TlTrackLabel color="a">História do Brasil</TlTrackLabel>
              <TlTrackLabel color="b">Escândalos políticos do Brasil</TlTrackLabel>
              <p class="ds-muted">Veja como crises e reformas aparecem na mesma década.</p>
            </TlCard>
            <TlCard tone="muted">
              <strong>Cartão neutro</strong>
              <p class="ds-muted">Para blocos dentro de outras superfícies.</p>
            </TlCard>
            <TlCard tone="inverse">
              <strong>Coincidências</strong>
              <p style="color: var(--tl-color-text-inverse-muted)">Cartão escuro para destaques.</p>
            </TlCard>
          </div>
        </DsSection>

        <!-- Timeline -->
        <DsSection
          id="timeline"
          title="Timeline"
          description="O evento na timeline vertical (ano, trilho e cartão), a versão de sugestão com borda tracejada e o minimapa com a janela visível."
        >
          <TlOverviewBar :start="1500" :end="2024" :years="overviewYears" :window="[1888, 2024]" :ticks="[1500, 1600, 1700, 1800, 1900, 2024]" />
          <div class="ds-events">
            <TlEventItem
              year="1888"
              date="13 de maio de 1888"
              title="Lei Áurea"
              description="Assinada pela princesa Isabel, abole a escravidão no país."
              by="@joaovitor"
              source-url="https://pt.wikipedia.org/wiki/Lei_%C3%81urea"
            />
            <TlEventItem
              year="1889"
              date="15 de novembro de 1889"
              title="Proclamação da República"
              description="O marechal Deodoro da Fonseca lidera o movimento que encerra a monarquia."
              by="@historia.br"
            />
            <TlEventItem
              year="1992"
              date="29 de dezembro de 1992"
              title="Renúncia e impeachment de Fernando Collor"
              description="Collor renuncia no dia do julgamento no Senado, que mantém a perda dos direitos políticos por oito anos."
              suggestion
            >
              <template #actions>
                <span class="ds-actions">
                  <TlButton size="sm" variant="secondary">Recusar</TlButton>
                  <TlButton size="sm" variant="success" icon="check">Aprovar</TlButton>
                </span>
              </template>
            </TlEventItem>
          </div>
        </DsSection>

        <!-- Comparação -->
        <DsSection
          id="comparar"
          title="Comparação"
          description="Trilhas paralelas no mesmo eixo de anos. Os rótulos se alternam em três alturas para não se sobreporem; clique num evento para selecioná-lo."
          :usage="usage.compare"
        >
          <TlCompareTracks v-model:selected="selecionado" :lanes="lanes" :start="1980" :end="2020" :highlight-year="1992" />
          <p class="ds-muted">Selecionado: <code>{{ selecionado }}</code></p>
        </DsSection>

        <DsSection
          id="timeline-grande"
          title="Timelines grandes"
          description="Com muitos eventos, a trilha recebe faixas de anos (buckets) e mostra grupos; clicar num grupo aproxima. Arraste o eixo, use os botões ou ctrl + roda do mouse para o zoom, ou mova a janela do minimapa. Poucos eventos no trecho aparecem um a um."
          :usage="usage.big"
        >
          <TlOverviewBar v-model:window="bigView" :start="-2000" :end="2030" :buckets="bigOverview" color="d" interactive />
          <TlCompareTracks v-model:selected="bigSelected" v-model:start="bigStart" v-model:end="bigEnd" :lanes="bigLanes" :min="-2000" :max="2030" zoomable />
          <p class="ds-muted">{{ bigWindow.total.toLocaleString('pt-BR') }} eventos entre {{ bigView[0] }} e {{ bigView[1] }} ({{ bigWindow.mode }})</p>
        </DsSection>

        <!-- Cabeçalho -->
        <DsSection id="cabecalho" title="Cabeçalho" description="TlAppHeader, o mesmo topo de todas as telas: logo, navegação, ação de criar e avatar para o perfil.">
          <div class="ds-header-demo">
            <TlAppHeader
              :nav="[
                { label: 'Explorar', to: '/', current: true },
                { label: 'Comparar', to: '/' },
                { label: 'Minhas timelines', to: '/' },
              ]"
              create-to="/"
              profile-to="/"
              user-name="Klausner"
            />
          </div>
        </DsSection>
      </main>
    </div>
  </div>
</template>

<style scoped>
.ds-hero {
  background: var(--tl-color-surface);
  border-bottom: 1px solid var(--tl-color-border);
}
.ds-hero__inner {
  padding-top: var(--tl-space-12);
  padding-bottom: var(--tl-space-10);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
}
.ds-eyebrow {
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-semibold);
  letter-spacing: var(--tl-tracking-eyebrow);
  text-transform: uppercase;
  color: var(--tl-color-primary);
}
.ds-hero__title {
  font-size: var(--tl-text-5xl);
  line-height: var(--tl-leading-tight);
  letter-spacing: var(--tl-tracking-tight);
}
.ds-hero__lead {
  max-width: 680px;
  font-size: var(--tl-text-lg);
  color: var(--tl-color-text-muted);
}
.ds-layout {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: var(--tl-space-10);
  padding-top: var(--tl-space-10);
  padding-bottom: var(--tl-space-18);
}
.ds-toc {
  position: sticky;
  top: var(--tl-space-6);
  align-self: start;
  display: flex;
  flex-direction: column;
}
.ds-toc a {
  padding: 6px 0;
  color: var(--tl-color-text-muted);
  text-decoration: none;
  font-size: var(--tl-text-md);
}
.ds-toc a:hover {
  color: var(--tl-color-text);
}
.ds-main {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-12);
  min-width: 0;
}
.ds-h3 {
  font-size: var(--tl-text-lg);
  margin-bottom: var(--tl-space-3);
}
.ds-muted {
  color: var(--tl-color-text-muted);
  font-size: var(--tl-text-sm);
}
code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.9em;
}

.ds-tracks {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--tl-space-4);
}
.ds-track {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-xl);
  overflow: hidden;
  background: var(--tl-color-surface);
}
.ds-track__chip {
  height: 64px;
}
.ds-track__soft {
  height: 24px;
}
.ds-track__text {
  padding: var(--tl-space-3) var(--tl-space-4) var(--tl-space-4);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ds-swatches {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--tl-space-3);
}
.ds-swatch {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: var(--tl-space-3);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-lg);
  background: var(--tl-color-surface);
}
.ds-swatch__color {
  height: 48px;
  border-radius: var(--tl-radius-sm);
  border: 1px solid var(--tl-color-border);
  margin-bottom: 4px;
}

.ds-type {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-bottom: var(--tl-space-3);
  border-bottom: 1px solid var(--tl-color-border-subtle);
}
.ds-type__meta {
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
}
.ds-type__sample {
  line-height: 1.2;
  letter-spacing: -0.01em;
}

.ds-space {
  display: grid;
  grid-template-columns: 130px auto 1fr;
  align-items: center;
  gap: var(--tl-space-3);
  font-size: var(--tl-text-sm);
}
.ds-space__bar {
  height: 12px;
  border-radius: 2px;
  background: var(--tl-color-primary);
}
.ds-radius {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  font-size: var(--tl-text-xs);
}
.ds-radius__box {
  width: 72px;
  height: 56px;
  background: var(--tl-color-primary-soft);
  border: 1.5px solid var(--tl-color-primary);
}
.ds-icon {
  width: 96px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: var(--tl-space-3) 0;
  font-size: var(--tl-text-xs);
}

.ds-inverse {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-5);
  padding: var(--tl-space-8);
  background: var(--tl-color-surface-inverse);
  color: var(--tl-color-text-inverse);
  border-radius: var(--tl-radius-2xl);
}
.ds-inverse__title {
  font-size: var(--tl-text-2xl);
}
.ds-inverse__text {
  color: var(--tl-color-text-inverse-muted);
}

.ds-grid-3 {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}
.ds-search {
  width: 100%;
  max-width: 640px;
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-2);
}
.ds-search .tl-input {
  flex: 1 1 280px;
  width: auto;
}
.ds-tab-panel {
  padding-top: var(--tl-space-4);
}
.ds-stats {
  /* Sem largura explícita, a grade auto-fit dentro de flex em coluna cresce na altura. */
  width: 100%;
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: var(--tl-space-3);
}
.ds-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--tl-space-5);
}
.ds-events {
  max-width: 760px;
}
.ds-actions {
  display: inline-flex;
  gap: var(--tl-space-2);
  margin-left: auto;
}
.ds-header-demo {
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-xl);
  overflow: hidden;
}

@media (max-width: 860px) {
  .ds-layout {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--tl-space-6);
  }
  .ds-toc {
    position: static;
    flex-direction: row;
    flex-wrap: wrap;
    gap: 4px 16px;
  }
  .ds-hero__title {
    font-size: var(--tl-text-4xl);
  }
}
</style>
