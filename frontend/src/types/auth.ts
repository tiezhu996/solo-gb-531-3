export type UserRole = 'admin' | 'process_engineer' | 'safety_reviewer' | 'auditor'

export interface SessionUser {
  id: number
  username: string
  display_name?: string
  role: UserRole
}

export interface LoginRequest { username: string; password: string }

export interface LoginResponse {
  token: string
  expires_at?: string
  user: SessionUser
}
