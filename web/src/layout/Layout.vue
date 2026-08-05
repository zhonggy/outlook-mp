<template>
  <div class="shell">
    <!-- 侧边导航 -->
    <aside class="side">
      <div class="brand">
        <div class="brand-mark">
          <Mail :size="20" :stroke-width="1.6" />
        </div>
        <div class="brand-text">
          <span class="brand-name">OUTLOOK</span>
          <span class="brand-sub mono">MANAGER v1.0</span>
        </div>
      </div>

      <nav class="nav">
        <RouterLink
          v-for="(item, i) in menus"
          :key="item.to"
          :to="item.to"
          class="nav-item"
          :class="{ active: isActive(item.to) }"
          :style="{ '--d': `${0.05 + i * 0.06}s` }"
        >
          <span class="nav-idx mono">{{ String(i + 1).padStart(2, '0') }}</span>
          <component :is="item.icon" :size="16" :stroke-width="1.6" class="nav-icon" />
          <span class="nav-label">{{ item.label }}</span>
          <span class="nav-line" />
        </RouterLink>
      </nav>

      <div class="side-foot">
        <div class="svc">
          <span class="dot ok" />
          <span class="svc-text mono">SERVICE ONLINE</span>
        </div>
        <div class="user-row">
          <div class="avatar">{{ initials }}</div>
          <div class="user-meta">
            <span class="user-name">{{ auth.username || 'admin' }}</span>
            <span class="user-role mono">ADMINISTRATOR</span>
          </div>
          <button class="logout" title="退出登录" @click="onLogout">
            <LogOut :size="15" :stroke-width="1.6" />
          </button>
        </div>
      </div>
    </aside>

    <!-- 主区域 -->
    <main class="content">
      <header class="topbar">
        <div class="crumb mono">
          <span class="faint">OM</span>
          <ChevronRight :size="12" class="faint" />
          <span>{{ route.meta.title || 'Console' }}</span>
        </div>
        <div class="topbar-right">
          <button
            class="theme-toggle"
            :title="isDark ? '切换到浅色' : '切换到深色'"
            :aria-label="isDark ? '切换到浅色' : '切换到深色'"
            @click="toggle"
          >
            <Transition name="theme-icon" mode="out-in">
              <Sun v-if="isDark" :size="16" :stroke-width="1.8" key="sun" />
              <Moon v-else :size="16" :stroke-width="1.8" key="moon" />
            </Transition>
          </button>
          <div class="clock mono">{{ clock }}</div>
        </div>
      </header>
      <div class="view">
        <RouterView v-slot="{ Component }">
          <Transition name="view" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Mail, Gauge, Users, ScrollText, Settings, LogOut, ChevronRight, Sun, Moon,
} from 'lucide-vue-next'
import { useAuthStore } from '../stores/auth'
import { useTheme } from '../composables/useTheme'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isDark, toggle } = useTheme()

const menus = [
  { to: '/dashboard', label: '仪表盘', icon: Gauge },
  { to: '/accounts', label: '账号管理', icon: Users },
  { to: '/tasks', label: '任务日志', icon: ScrollText },
  { to: '/settings', label: '系统设置', icon: Settings },
]

const clock = ref('')
let timer = 0
function tick() {
  clock.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
}

const initials = computed(() => (auth.username || 'A').slice(0, 1).toUpperCase())

function isActive(to: string) {
  return route.path === to || route.path.startsWith(to + '/')
}

function onLogout() {
  auth.logout()
  router.push('/login')
}

onMounted(() => {
  tick()
  timer = window.setInterval(tick, 1000)
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.shell { display: flex; min-height: 100vh; }

/* ---------- 侧边 ---------- */
.side {
  width: var(--nav-w);
  flex: none;
  border-right: 1px solid var(--line);
  display: flex;
  flex-direction: column;
  position: sticky;
  top: 0;
  height: 100vh;
  background: color-mix(in srgb, var(--bg-raised) 60%, transparent);
  backdrop-filter: blur(6px);
}
.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 24px 22px;
  border-bottom: 1px solid var(--line);
}
.brand-mark {
  width: 38px;
  height: 38px;
  border: 1px solid var(--line-strong);
  border-radius: 6px;
  display: grid;
  place-items: center;
  color: var(--acid);
  background: var(--acid-glow);
}
.brand-text { display: flex; flex-direction: column; }
.brand-name { font-weight: 800; letter-spacing: 0.08em; font-size: 14px; }
.brand-sub { font-size: 9px; color: var(--text-faint); letter-spacing: 0.28em; }

/* 导航项 */
.nav { flex: 1; padding: 22px 14px; display: flex; flex-direction: column; gap: 4px; }
.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 12px;
  border-radius: var(--radius);
  color: var(--text-dim);
  position: relative;
  overflow: hidden;
  opacity: 0;
  animation: rise-left 0.55s var(--ease-out) forwards;
  animation-delay: var(--d, 0s);
  transition: all 0.3s var(--ease-out);
}
.nav-item:hover { color: var(--text); background: var(--bg-hover); transform: translateX(4px); }
.nav-item.active { color: var(--acid); background: var(--acid-glow); }
.nav-idx { font-size: 10px; color: var(--text-faint); width: 18px; }
.nav-item.active .nav-idx { color: var(--acid); }
.nav-icon { flex: none; }
.nav-label { font-size: 13.5px; font-weight: 600; letter-spacing: 0.04em; }
.nav-line {
  position: absolute;
  left: 0; top: 20%; bottom: 20%;
  width: 2px;
  background: var(--acid);
  transform: scaleY(0);
  transform-origin: center;
  transition: transform 0.3s var(--ease-spring);
}
.nav-item.active .nav-line { transform: scaleY(1); }

/* 侧栏底部 */
.side-foot { padding: 18px 22px; border-top: 1px solid var(--line); }
.svc { display: flex; align-items: center; gap: 8px; margin-bottom: 14px; }
.svc-text { font-size: 10px; letter-spacing: 0.22em; color: var(--text-faint); }
.user-row { display: flex; align-items: center; gap: 10px; }
.avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  background: var(--acid);
  color: var(--acid-ink);
  font-weight: 800;
  display: grid;
  place-items: center;
  font-size: 13px;
}
.user-meta { display: flex; flex-direction: column; flex: 1; min-width: 0; }
.user-name { font-size: 13px; font-weight: 700; }
.user-role { font-size: 9px; letter-spacing: 0.2em; color: var(--text-faint); }
.logout {
  background: none;
  border: 1px solid var(--line);
  border-radius: 4px;
  color: var(--text-faint);
  width: 28px; height: 28px;
  display: grid; place-items: center;
  cursor: pointer;
  transition: all 0.25s var(--ease-out);
}
.logout:hover { color: var(--bad); border-color: rgba(255, 84, 112, 0.4); }

/* ---------- 主区 ---------- */
.content { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.topbar {
  height: 52px;
  border-bottom: 1px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  position: sticky;
  top: 0;
  background: color-mix(in srgb, var(--bg) 78%, transparent);
  backdrop-filter: blur(10px);
  z-index: 10;
}
.crumb { display: flex; align-items: center; gap: 8px; font-size: 12px; letter-spacing: 0.1em; }
.topbar-right { display: flex; align-items: center; gap: 16px; }
.clock { font-size: 12px; color: var(--text-faint); letter-spacing: 0.1em; }

/* 主题切换按钮 */
.theme-toggle {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  background: transparent;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  color: var(--text-dim);
  cursor: pointer;
  transition: all 0.25s var(--ease-out);
}
.theme-toggle:hover {
  color: var(--acid);
  border-color: var(--line-strong);
  transform: translateY(-1px);
}
.theme-icon-enter-active,
.theme-icon-leave-active { transition: all 0.3s var(--ease-spring); }
.theme-icon-enter-from { opacity: 0; transform: rotate(-90deg) scale(0.5); }
.theme-icon-leave-to { opacity: 0; transform: rotate(90deg) scale(0.5); }

.view { flex: 1; }

/* 视图切换动效 */
.view-enter-active { animation: rise 0.45s var(--ease-out); }
.view-leave-active { transition: opacity 0.18s ease; }
.view-leave-to { opacity: 0; }

@media (max-width: 860px) {
  .side { width: 72px; }
  .brand-text, .nav-label, .nav-idx, .user-meta, .svc-text { display: none; }
  .nav-item { justify-content: center; }
  .user-row { justify-content: center; }
}
</style>
