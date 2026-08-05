import axios, { AxiosInstance } from 'axios'
import { toast } from '../components/ui/toast'

const api: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('om_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (resp) => resp,
  (error) => {
    const status = error.response?.status
    const msg = error.response?.data?.error || error.message || '请求失败'
    if (status === 401) {
      localStorage.removeItem('om_token')
      if (location.pathname !== '/login') {
        location.href = '/login'
      }
    } else {
      toast.bad(msg)
    }
    return Promise.reject(error)
  },
)

export default api

// ---- 类型 ----
export interface Account {
  id: number
  email: string
  password?: string
  client_id: string
  refresh_token?: string
  status: string
  status_reason: string
  fail_count: number
  tags: string
  group_name: string
  remark: string
  proxy: string
  source: string
  token_expires_at?: string
  last_refresh_at?: string
  last_check_at?: string
  last_keepalive_at?: string
  last_mail_at?: string
  created_at: string
  updated_at: string
}

export interface MailMessage {
  id: number
  account_id: number
  message_id: string
  subject: string
  from_addr: string
  body_preview: string
  body?: string
  received_at: string
  is_read: boolean
  folder?: string  // inbox / junk
  code?: string
}

export interface TaskLog {
  id: number
  task_type: string
  email: string
  status: string
  message: string
  duration_ms: number
  created_at: string
}

export interface ListResp<T> {
  items: T[]
  total: number
}

// ---- API 方法 ----
export const authApi = {
  login: (username: string, password: string) =>
    api.post('/auth/login', { username, password }),
  changePassword: (old_password: string, new_password: string) =>
    api.post('/auth/change-password', { old_password, new_password }),
}

export const accountApi = {
  list: (params: Record<string, unknown>) => api.get<ListResp<Account>>('/accounts', { params }),
  get: (id: number) => api.get(`/accounts/${id}`),
  create: (data: Partial<Account>) => api.post('/accounts', data),
  update: (id: number, data: Partial<Account>) => api.put(`/accounts/${id}`, data),
  remove: (id: number) => api.delete(`/accounts/${id}`),
  batchDelete: (ids: number[]) => api.post('/accounts/batch-delete', { ids }),
  deleteByStatus: (status: string) =>
    api.post<{ count: number }>('/accounts/delete-by-status', { status }),
  importText: (text: string) =>
    api.post('/accounts/import', text, { headers: { 'Content-Type': 'text/plain' } }),
  exportUrl: (params: Record<string, unknown>) => {
    const q = new URLSearchParams(params as Record<string, string>).toString()
    return `/api/v1/accounts/export?${q}`
  },
  refresh: (id: number) => api.post(`/accounts/${id}/refresh`),
  check: (id: number) => api.post(`/accounts/${id}/check`),
  refreshAll: () => api.post('/accounts/refresh-all'),
  // 全量检测耗时与账号数成正比，放宽到 10 分钟
  checkAll: () =>
    api.post<{ total: number; success: number; fail: number; skip: number }>(
      '/accounts/check-all', null, { timeout: 10 * 60 * 1000 }),
  groups: () => api.get('/accounts/groups'),
}

export const mailApi = {
  list: (accountId: number, params: Record<string, unknown>) =>
    api.get<ListResp<MailMessage>>(`/accounts/${accountId}/mails`, { params }),
  fetch: (accountId: number) => api.post(`/accounts/${accountId}/mails/fetch`),
  detail: (id: number) => api.get<MailMessage>(`/mails/${id}`),
}

export const taskApi = {
  logs: (params: Record<string, unknown>) => api.get<ListResp<TaskLog>>('/tasks/logs', { params }),
  schedule: () => api.get('/tasks/schedule'),
  setSchedule: (data: Record<string, unknown>) => api.put('/tasks/schedule', data),
  cleanupLogs: (all = false) =>
    api.post<{ deleted: number; remaining: number }>('/tasks/logs/cleanup', all ? { all: true } : {}),
}

export const statsApi = {
  dashboard: () => api.get('/stats/dashboard'),
}

export const apikeyApi = {
  list: () => api.get('/apikeys'),
  create: (name: string) => api.post('/apikeys', { name }),
  remove: (id: number) => api.delete(`/apikeys/${id}`),
}
