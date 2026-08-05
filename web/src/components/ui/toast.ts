import { reactive, readonly } from 'vue'

export type ToastTone = 'ok' | 'bad' | 'warn' | 'info'

export interface ToastItem {
  id: number
  tone: ToastTone
  text: string
  leaving?: boolean
}

const state = reactive<{ items: ToastItem[] }>({ items: [] })
let seq = 0

function push(tone: ToastTone, text: string, duration = 3200) {
  const id = ++seq
  state.items.push({ id, tone, text })
  setTimeout(() => dismiss(id), duration)
}

function dismiss(id: number) {
  const item = state.items.find((t) => t.id === id)
  if (item) item.leaving = true
  setTimeout(() => {
    const idx = state.items.findIndex((t) => t.id === id)
    if (idx >= 0) state.items.splice(idx, 1)
  }, 350)
}

export const toast = {
  ok: (text: string) => push('ok', text),
  bad: (text: string) => push('bad', text, 4200),
  warn: (text: string) => push('warn', text),
  info: (text: string) => push('info', text),
}

export const toastState = readonly(state)
export const toastItems = state
