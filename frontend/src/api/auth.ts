import { api, json } from './client'
import type { LoginRequest, LoginResponse } from '../types/auth'

export const login = (input: LoginRequest) => api<LoginResponse>('/auth/login', json('POST', input))
