export interface ApiEnvelope<T> {
  code: string
  message: string
  data: T
  request_id: string
}

export interface PageData<T> {
  items: T[]
  total: number
}

export interface ApiErrorBody {
  code?: string
  message?: string
  error?: { code?: string; message?: string; details?: unknown }
  request_id?: string
}

export type EntityID = number

export function normalizePage<T>(data: PageData<T> | T[]): PageData<T> {
  return Array.isArray(data) ? { items: data, total: data.length } : data
}
