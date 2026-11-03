/** Para onde voltar depois de entrar: só caminhos do próprio app. */
export function nextPath(next: unknown, fallback = '/perfil'): string {
  return typeof next === 'string' && next.startsWith('/') && !next.startsWith('//') && !next.startsWith('/\\') ? next : fallback
}
