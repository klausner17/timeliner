/*
 * Cliente da API (api/openapi.yaml). Fica no mesmo endereço do app: em produção
 * klausner.site/app/timeliner/api, em desenvolvimento o Vite repassa /api ao Go.
 */

const BASE = `${import.meta.env.BASE_URL}api`
const TOKEN_KEY = 'timeliner:token'

export type ApiErrorCode = 'invalid' | 'unauthorized' | 'forbidden' | 'not_found' | 'conflict' | 'method_not_allowed' | 'internal' | 'unavailable' | 'network'

/** Erro da API, com a frase pronta para a tela e os erros por campo. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: ApiErrorCode,
    message: string,
    readonly fields: Record<string, string> = {},
  ) {
    super(message)
  }
}

let token: string | null = readToken()
let onUnauthorized: (() => void) | undefined

function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

export function getToken() {
  return token
}

export function setToken(value: string | null) {
  token = value
  try {
    if (value) localStorage.setItem(TOKEN_KEY, value)
    else localStorage.removeItem(TOKEN_KEY)
  } catch {
    // Sem storage (aba anônima): a sessão dura até fechar a aba.
  }
}

/** Chamado quando a API recusa o token (expirou ou foi revogado). */
export function handleUnauthorized(fn: () => void) {
  onUnauthorized = fn
}

type Query = Record<string, string | number | boolean | undefined | null>

export async function request<T>(method: string, path: string, opts: { query?: Query; body?: unknown } = {}): Promise<T> {
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(opts.query ?? {})) if (v !== undefined && v !== null && v !== '') params.set(k, String(v))
  const qs = params.toString()
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`

  let res: Response
  try {
    res = await fetch(`${BASE}${path}${qs ? `?${qs}` : ''}`, {
      method,
      headers,
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
    })
  } catch {
    throw new ApiError(0, 'network', 'Sem conexão com o servidor. Tente de novo em instantes.')
  }

  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => undefined)
  if (res.ok) return data as T

  const err = (data as { error?: { code?: ApiErrorCode; message?: string; fields?: Record<string, string> } } | undefined)?.error
  if (res.status === 401 && token) {
    setToken(null)
    onUnauthorized?.()
  }
  throw new ApiError(res.status, err?.code ?? 'internal', err?.message ?? 'Algo deu errado. Tente de novo.', err?.fields ?? {})
}

export const get = <T>(path: string, query?: Query) => request<T>('GET', path, { query })
export const post = <T>(path: string, body?: unknown) => request<T>('POST', path, { body: body ?? {} })
export const patch = <T>(path: string, body: unknown) => request<T>('PATCH', path, { body })
export const del = (path: string) => request<void>('DELETE', path)

/** Mensagem de qualquer erro, para mostrar na tela. */
export function errorMessage(e: unknown, fallback = 'Algo deu errado. Tente de novo.'): string {
  return e instanceof Error && e.message ? e.message : fallback
}
