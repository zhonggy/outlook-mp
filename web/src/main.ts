import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import AppToast from './components/ui/AppToast.vue'
import { applyStoredTheme } from './composables/useTheme'
import './styles/main.css'

// 挂载前尽早应用主题，避免刷新瞬间的明暗闪烁
applyStoredTheme()

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

// 全局 Toast 容器（独立挂载点）
const toastHost = document.createElement('div')
toastHost.id = 'toast-host'
document.body.appendChild(toastHost)
createApp(AppToast).mount(toastHost)
