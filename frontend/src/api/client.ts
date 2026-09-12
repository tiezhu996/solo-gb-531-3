import type { ApiEnvelope, ApiErrorBody } from '../types/common'

export const tokenKey = 'hazop_coverage_token'
export const userKey = 'hazop_coverage_user'

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly details: unknown,
    public readonly requestId: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<ApiEnvelope<T>> {
  const headers = new Headers(options.headers)
  const token = localStorage.getItem(tokenKey)
  if (options.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const response = await fetch(`/api/v1${path}`, { ...options, headers })
  const payload = await response.json().catch(() => null) as ApiEnvelope<T> | ApiErrorBody | null
  if (!response.ok) {
    const body = payload as ApiErrorBody | null
    if (response.status === 401) {
      localStorage.removeItem(tokenKey)
      localStorage.removeItem(userKey)
      window.dispatchEvent(new Event('hazop-auth-expired'))
    }
    throw new ApiError(
      response.status,
      body?.error?.code ?? body?.code ?? 'HTTP_ERROR',
      body?.error?.message ?? body?.message ?? `Request failed with HTTP ${response.status}`,
      body?.error?.details,
      body?.request_id ?? response.headers.get('X-Request-ID') ?? '',
    )
  }
  return payload as ApiEnvelope<T>
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const result = await request<T>(path, options)
  return result.data
}

export const json = (method: string, body?: unknown, headers?: HeadersInit): RequestInit => ({
  method,
  headers,
  body: body === undefined ? undefined : JSON.stringify(body),
})

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return `${error.message}${error.requestId ? ` (request ${error.requestId})` : ''}`
  }
  return error instanceof Error ? error.message : '请求未完成，请稍后重试。'
}

export function query(params: Record<string, string | number | undefined>): string {
  const values = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') values.set(key, String(value))
  })
  const encoded = values.toString()
  return encoded ? `?${encoded}` : ''
}
