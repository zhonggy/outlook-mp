<template>
  <button
    class="btn"
    :class="[`btn-${variant}`, `btn-${size}`, { 'is-loading': loading, 'btn-block': block }]"
    :disabled="disabled || loading"
  >
    <span v-if="loading" class="spinner" />
    <component :is="icon" v-else-if="icon" :size="iconSize" :stroke-width="1.8" class="btn-icon" />
    <span class="btn-label"><slot /></span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'primary' | 'ghost' | 'danger' | 'text'
  size?: 'sm' | 'md' | 'lg'
  icon?: Component
  loading?: boolean
  disabled?: boolean
  block?: boolean
}>(), {
  variant: 'ghost',
  size: 'md',
  loading: false,
  disabled: false,
  block: false,
})

const iconSize = computed(() => ({ sm: 13, md: 15, lg: 17 }[props.size]))
</script>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-family: var(--font-sans);
  font-weight: 600;
  font-size: 13px;
  letter-spacing: 0.02em;
  border-radius: var(--radius);
  border: 1px solid transparent;
  cursor: pointer;
  transition: all var(--dur) var(--ease-out);
  white-space: nowrap;
  user-select: none;
}
.btn:disabled { opacity: 0.45; cursor: not-allowed; }
.btn:not(:disabled):active { transform: translateY(1px) scale(0.98); }

.btn-sm  { padding: 5px 12px;  font-size: 12px; }
.btn-md  { padding: 9px 18px; }
.btn-lg  { padding: 13px 28px; font-size: 14px; }

.btn-primary {
  background: var(--acid);
  color: var(--acid-ink);
  border-color: var(--acid);
}
.btn-primary:not(:disabled):hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 28px -6px color-mix(in srgb, var(--acid) 45%, transparent);
}

.btn-ghost {
  background: transparent;
  color: var(--text);
  border-color: var(--line-strong);
}
.btn-ghost:not(:disabled):hover {
  border-color: var(--acid);
  color: var(--acid);
  transform: translateY(-2px);
}

.btn-danger {
  background: transparent;
  color: var(--bad);
  border-color: rgba(255, 84, 112, 0.4);
}
.btn-danger:not(:disabled):hover {
  background: rgba(255, 84, 112, 0.1);
  border-color: var(--bad);
}

.btn-text {
  background: transparent;
  color: var(--text-dim);
  padding-left: 8px;
  padding-right: 8px;
}
.btn-text:not(:disabled):hover { color: var(--acid); }

.btn-block { width: 100%; justify-content: center; }

.btn-icon { flex: none; }

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  flex: none;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
