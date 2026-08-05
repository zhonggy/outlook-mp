import { defineStore } from 'pinia'
import { authApi } from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('om_token') || '',
    username: localStorage.getItem('om_username') || '',
  }),
  getters: {
    isLoggedIn: (s) => !!s.token,
  },
  actions: {
    async login(username: string, password: string) {
      const { data } = await authApi.login(username, password)
      this.token = data.token
      this.username = data.username
      localStorage.setItem('om_token', data.token)
      localStorage.setItem('om_username', data.username)
    },
    logout() {
      this.token = ''
      this.username = ''
      localStorage.removeItem('om_token')
      localStorage.removeItem('om_username')
    },
  },
})
