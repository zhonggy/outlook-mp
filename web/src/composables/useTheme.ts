/**
 * 主题管理：浅色 / 深色 / 跟随系统。
 *
 * - 首次访问无记录时为 'auto'，跟随操作系统的 prefers-color-scheme。
 * - 用户手动切换后将选择写入 localStorage 长期记忆。
 * - 实际生效的主题通过 <html data-theme="dark|light"> 驱动 CSS 变量。
 *
 * 为避免刷新瞬间的主题闪烁（FOUC），main.ts 在 mount 前会调用 applyStoredTheme()
 * 尽早把 data-theme 打到 <html> 上。
 */
import { computed, ref } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'om_theme'

const media = window.matchMedia('(prefers-color-scheme: light)')

function readStored(): ThemeMode {
  const v = localStorage.getItem(STORAGE_KEY)
  return v === 'light' || v === 'dark' || v === 'auto' ? v : 'auto'
}

function systemTheme(): ResolvedTheme {
  return media.matches ? 'light' : 'dark'
}

function resolve(mode: ThemeMode): ResolvedTheme {
  return mode === 'auto' ? systemTheme() : mode
}

// 模块级单例状态：全局共享同一份主题
const mode = ref<ThemeMode>(readStored())
const resolved = ref<ResolvedTheme>(resolve(mode.value))

function apply() {
  resolved.value = resolve(mode.value)
  document.documentElement.setAttribute('data-theme', resolved.value)
}

// 系统主题变化时，仅在 auto 模式下自动跟随
media.addEventListener('change', () => {
  if (mode.value === 'auto') apply()
})

/** 在应用挂载前调用，尽早写入 data-theme，避免主题闪烁。 */
export function applyStoredTheme() {
  apply()
}

export function useTheme() {
  function setMode(next: ThemeMode) {
    mode.value = next
    localStorage.setItem(STORAGE_KEY, next)
    apply()
  }

  /** 顶栏一键切换：在浅/深之间翻转（显式记忆，不再跟随系统）。 */
  function toggle() {
    setMode(resolved.value === 'dark' ? 'light' : 'dark')
  }

  const isDark = computed(() => resolved.value === 'dark')

  return { mode, resolved, isDark, setMode, toggle }
}
