import { api, json, query } from './client'
import { normalizePage, type PageData } from '../types/common'
import type { ProcessNode, ProcessNodeInput } from '../types/process-node'

export async function listProcessNodes(search = ''): Promise<PageData<ProcessNode>> {
  return normalizePage(await api<PageData<ProcessNode> | ProcessNode[]>(`/process-nodes${query({ search })}`))
}
export const getProcessNode = (id: number) => api<ProcessNode>(`/process-nodes/${id}`)
export const createProcessNode = (input: ProcessNodeInput) => api<ProcessNode>('/process-nodes', json('POST', input))
export const updateProcessNode = (id: number, input: ProcessNodeInput) => api<ProcessNode>(`/process-nodes/${id}`, json('PUT', input))
export const deactivateProcessNode = (id: number) => api<ProcessNode>(`/process-nodes/${id}/deactivate`, json('POST'))
