import type { Bootstrap, ImportResult, LinkHistoryEntry, LinkInput, Preferences, RestoreTarget, Revision } from './types'

async function request<T>(path:string, init:RequestInit = {}):Promise<T> {
  const response = await fetch(path, { credentials:'same-origin', headers:{ 'Content-Type':'application/json', ...(init.headers ?? {}) }, ...init })
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error:{ message:response.statusText } }))
    if (response.status === 401) {
      const returnTo = `${location.pathname}${location.search}${location.hash}`
      location.assign(`/auth/login?return=${encodeURIComponent(returnTo)}`)
    }
    throw new Error(body.error?.message ?? `Request failed (${response.status})`)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  bootstrap: () => request<Bootstrap>('/api/v1/bootstrap'),
  savePreferences: (value:Preferences) => request<Preferences>('/api/v1/me/preferences', { method:'PUT', body:JSON.stringify(value) }),
  hide: (id:number, hidden:boolean) => request<void>(`/api/v1/me/hidden-links/${id}`, { method:hidden?'PUT':'DELETE' }),
  create: (input:LinkInput) => request<{id:number}>('/api/v1/links', { method:'POST', body:JSON.stringify(input) }),
  update: (id:number, input:LinkInput) => request<void>(`/api/v1/links/${id}`, { method:'PUT', body:JSON.stringify(input) }),
  remove: (id:number, revision:number) => request<void>(`/api/v1/links/${id}?revision=${revision}`, { method:'DELETE' }),
  revisions: (id:number) => request<Revision[]>(`/api/v1/links/${id}/revisions`),
  restore: (id:number, revision:number) => request<void>(`/api/v1/links/${id}/restore/${revision}`, { method:'POST' }),
  linkHistory: () => request<LinkHistoryEntry[]>('/api/v1/link-history'),
  restoreHistory: (items:RestoreTarget[]) => request<void>('/api/v1/link-history/restore', { method:'POST', body:JSON.stringify({items}) }),
  importCatalog: (file:File) => request<ImportResult>('/api/v1/catalog/import', { method:'POST', headers:{'Content-Type':'application/yaml'}, body:file }),
  logout: () => request<{redirectUrl:string}>('/auth/logout', { method:'POST' }),
}
