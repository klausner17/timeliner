import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, getToken, setToken } from '../api'
import { canEdit, getTimeline, initSession, listContributions, listTimelines, loadEvents, login, logout, register, confirmPasswordReset, rangeLabel, relatedTo, session } from '../store'
import type { Timeline } from '../types'

type Call = { url: string; init: RequestInit }
let calls: Call[]
let replies: (() => Response)[]

const json = (status: number, body: unknown) => () => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

beforeEach(async () => {
  calls = []
  replies = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, init: RequestInit) => {
      calls.push({ url, init })
      const next = replies.shift()
      if (!next) throw new TypeError('sem resposta')
      return next()
    }),
  )
  await logout()
  calls = []
  replies = []
})

afterEach(() => vi.unstubAllGlobals())

const timeline = (over: Partial<Timeline> = {}): Timeline => ({
  slug: 'a',
  title: 'A',
  description: '',
  category: 'História',
  color: 'a',
  owner: { handle: 'klausner', name: 'Klausner', tint: 'a' },
  visibility: 'publica',
  contributions: 'fechada',
  draft: false,
  cyclic: false,
  eras: [],
  tags: [],
  startYear: 1900,
  endYear: 2000,
  eventCount: 0,
  contributorCount: 1,
  pendingSuggestions: 0,
  followers: 0,
  following: false,
  points: [],
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-01T00:00:00Z',
  ...over,
})

describe('cliente da API', () => {
  it('monta a query sem os vazios e manda o token', async () => {
    setToken('t1')
    replies.push(json(200, { items: [], page: 1, pageSize: 20, total: 0 }))
    await listTimelines({ q: '', cat: 'Esporte', owner: 'me', pageSize: 48 })
    expect(calls[0].url).toBe('/api/timelines?cat=Esporte&owner=me&pageSize=48')
    expect((calls[0].init.headers as Record<string, string>).Authorization).toBe('Bearer t1')
  })

  it('transforma o erro da API em ApiError com a mensagem e os campos', async () => {
    replies.push(json(400, { error: { code: 'invalid', message: 'Confira os campos.', fields: { email: 'E-mail inválido.' } } }))
    const err = await login('x', 'y').catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 400, code: 'invalid', message: 'Confira os campos.', fields: { email: 'E-mail inválido.' } })
  })

  it('sem rede vira erro de conexão', async () => {
    const err = await listTimelines().catch((e) => e)
    expect(err).toMatchObject({ status: 0, code: 'network' })
  })
})

describe('sessão', () => {
  it('login guarda o token e o usuário; 401 depois limpa os dois', async () => {
    const user = { handle: 'klausner', name: 'Klausner', tint: 'a', email: 'k@x', bio: '', memberSince: '2026-10-01T00:00:00Z' }
    replies.push(json(200, { token: 'abc', user }))
    await login('k@x', 'senha')
    expect(getToken()).toBe('abc')
    expect(session.user?.handle).toBe('klausner')

    replies.push(json(401, { error: { code: 'unauthorized', message: 'Sessão expirada.' } }))
    await listTimelines().catch(() => undefined)
    expect(getToken()).toBeNull()
    expect(session.user).toBeNull()
  })

  it('initSession sem token não chama a API', async () => {
    await initSession()
    expect(calls).toHaveLength(0)
    expect(session.ready).toBe(true)
  })
})

describe('leituras', () => {
  it('getTimeline devolve undefined no 404 e tira os null', async () => {
    replies.push(json(404, { error: { code: 'not_found', message: 'Não encontrada.' } }))
    expect(await getTimeline('nada')).toBeUndefined()

    replies.push(
      json(200, {
        ...timeline({ slug: 'b' }),
        startYear: null,
        endYear: null,
        eras: [{ name: 'X', start: 1, end: null }],
        events: [{ id: 'e1', year: 1, endYear: null, title: 'E', description: '', sourceUrl: null, by: 'klausner', era: null }],
        eventsNextCursor: null,
      }),
    )
    const t = await getTimeline('b')
    expect(t).not.toHaveProperty('startYear')
    expect(t?.eras[0]).toEqual({ name: 'X', start: 1 })
    expect(t?.events[0]).not.toHaveProperty('endYear')
    expect(rangeLabel(t!)).toBe('Sem datas')
  })

  it('loadEvents pede uma página pelo cursor', async () => {
    const ev = (id: string) => ({ id, year: 1, title: id, description: '', by: 'k' })
    replies.push(json(200, { items: [ev('3')], nextCursor: null }))
    const page = await loadEvents('a', { cursor: 'c2', limit: 200 })
    expect(page.items.map((e) => e.id)).toEqual(['3'])
    expect(calls[0].url).toBe('/api/timelines/a/events?cursor=c2&limit=200')
  })

  it('listContributions filtra pela timeline e pagina', async () => {
    replies.push(json(200, { items: [], page: 2, pageSize: 20, total: 21 }))
    const page = await listContributions('ana', { timeline: 'brasil', page: 2, pageSize: 20 })
    expect(page.total).toBe(21)
    expect(calls[0].url).toBe('/api/users/ana/contributions?timeline=brasil&page=2&pageSize=20')
  })
})

describe('ajudas', () => {
  it('canEdit só para o dono logado', () => {
    expect(canEdit(timeline())).toBe(false)
    session.user = { handle: 'klausner', name: 'K', tint: 'a', email: '', bio: '', memberSince: '', emailVerified: true }
    expect(canEdit(timeline())).toBe(true)
    expect(canEdit(timeline({ owner: { handle: 'outra', name: 'O', tint: 'neutral' } }))).toBe(false)
  })

  it('relatedTo ordena pela sobreposição e ignora sem datas', () => {
    const base = timeline({ slug: 'base', startYear: 1900, endYear: 2000 })
    const others = [
      timeline({ slug: 'pouco', startYear: 1990, endYear: 2050 }),
      timeline({ slug: 'muito', startYear: 1920, endYear: 1990 }),
      timeline({ slug: 'fora', startYear: 2001, endYear: 2010 }),
      timeline({ slug: 'semdata', startYear: undefined, endYear: undefined }),
    ]
    expect(relatedTo(base, others).map((t) => t.slug)).toEqual(['muito', 'pouco'])
  })
})

describe('conta', () => {
  const user = { handle: 'ana', name: 'Ana', tint: 'neutral', email: 'ana@x.com', bio: '', memberSince: '2026-11-01T00:00:00Z', emailVerified: false }

  it('cadastra e já entra', async () => {
    replies.push(json(201, { token: 'novo', user }))
    await register({ handle: 'ana', name: 'Ana', email: 'ana@x.com', password: 'senha-forte', website: '' })
    expect(calls[0].url).toBe('/api/auth/register')
    expect(getToken()).toBe('novo')
    expect(session.user?.handle).toBe('ana')
  })

  it('sair apaga a sessão no servidor e localmente, mesmo se a API falhar', async () => {
    setToken('t1')
    session.user = user as never
    await logout()
    expect(calls[0].url).toBe('/api/auth/logout')
    expect(calls[0].init.method).toBe('POST')
    expect(getToken()).toBeNull()
    expect(session.user).toBeNull()
  })

  it('redefinir a senha pelo link entra na conta', async () => {
    replies.push(json(200, { token: 't2', user: { ...user, emailVerified: true } }))
    await confirmPasswordReset('abc', 'nova-senha-1')
    expect(JSON.parse(String(calls[0].init.body))).toEqual({ token: 'abc', password: 'nova-senha-1' })
    expect(getToken()).toBe('t2')
    expect(session.user?.emailVerified).toBe(true)
  })
})
