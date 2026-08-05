<template>
  <div class="empty">
    <div class="empty-icon">
      <component :is="icon" :size="26" :stroke-width="1.2" />
    </div>
    <p class="empty-text">{{ text }}</p>
    <p v-if="hint" class="empty-hint">{{ hint }}</p>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { Inbox } from 'lucide-vue-next'
import type { Component } from 'vue'

withDefaults(defineProps<{
  text?: string
  hint?: string
  icon?: Component
}>(), {
  text: '暂无数据',
  // 用工厂函数返回，避免 Vue 把函数式组件（lucide 图标）当作 prop 默认工厂调用，
  // 否则 icon 未显式传入时会以缺失 context 调用 Inbox，触发
  // "Cannot destructure property 'slots' of 'undefined'"。
  icon: () => Inbox,
})
</script>

<style scoped>
.empty {
  padding: 56px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.empty-icon {
  width: 56px;
  height: 56px;
  border: 1px dashed var(--line-strong);
  border-radius: 50%;
  display: grid;
  place-items: center;
  color: var(--text-faint);
  margin-bottom: 8px;
}
.empty-text { margin: 0; color: var(--text-dim); font-weight: 600; }
.empty-hint { margin: 0; color: var(--text-faint); font-size: 12px; }
</style>
