<template>
  <Teleport to="body">
    <div class="toast-stack">
      <TransitionGroup name="toast">
        <div
          v-for="t in items.items"
          :key="t.id"
          class="toast"
          :class="[`toast-${t.tone}`, { leaving: t.leaving }]"
        >
          <component :is="icons[t.tone]" :size="15" :stroke-width="2" />
          <span>{{ t.text }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { CheckCircle2, XCircle, AlertTriangle, Info } from 'lucide-vue-next'
import { toastItems as items } from './toast'

const icons = {
  ok: CheckCircle2,
  bad: XCircle,
  warn: AlertTriangle,
  info: Info,
}
</script>

<style scoped>
.toast-stack {
  position: fixed;
  top: 24px;
  right: 24px;
  z-index: 200;
  display: flex;
  flex-direction: column;
  gap: 10px;
  pointer-events: none;
}
.toast {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 18px;
  background: var(--bg-overlay);
  border: 1px solid var(--line-strong);
  border-radius: var(--radius);
  font-size: 13px;
  font-weight: 500;
  box-shadow: var(--shadow-overlay);
  pointer-events: auto;
}
.toast-ok   { border-left: 2px solid var(--ok);   color: var(--text); }
.toast-ok svg   { color: var(--ok); }
.toast-bad  { border-left: 2px solid var(--bad);  color: var(--text); }
.toast-bad svg  { color: var(--bad); }
.toast-warn { border-left: 2px solid var(--warn); color: var(--text); }
.toast-warn svg { color: var(--warn); }
.toast-info { border-left: 2px solid var(--info); color: var(--text); }
.toast-info svg { color: var(--info); }

.toast-enter-active { animation: toast-in 0.45s var(--ease-spring); }
.toast-enter-from { opacity: 0; }
.toast-leave-active { transition: all 0.3s var(--ease-out); }
.toast-leave-to { opacity: 0; transform: translateX(30px); }
@keyframes toast-in {
  from { opacity: 0; transform: translateX(40px) scale(0.95); }
  to   { opacity: 1; transform: translateX(0) scale(1); }
}
</style>
