<template>
  <div class="pager" v-if="pages > 1">
    <button class="pg-btn" :disabled="page <= 1" @click="go(page - 1)" aria-label="上一页">
      <ChevronLeft :size="14" />
    </button>
    <button
      v-for="p in windowPages"
      :key="p"
      class="pg-btn pg-num"
      :class="{ active: p === page }"
      @click="go(p)"
    >
      {{ p }}
    </button>
    <button class="pg-btn" :disabled="page >= pages" @click="go(page + 1)" aria-label="下一页">
      <ChevronRight :size="14" />
    </button>
    <span class="pg-total mono">{{ total }} 条</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  page: number
  size: number
  total: number
}>(), { size: 20 })

const emit = defineEmits<{ change: [number] }>()

const pages = computed(() => Math.max(1, Math.ceil(props.total / props.size)))

const windowPages = computed(() => {
  const p = props.page
  const n = pages.value
  const set = new Set<number>([1, n, p - 1, p, p + 1])
  return [...set].filter((x) => x >= 1 && x <= n).sort((a, b) => a - b)
})

function go(p: number) {
  if (p >= 1 && p <= pages.value && p !== props.page) emit('change', p)
}
</script>

<style scoped>
.pager { display: flex; align-items: center; gap: 6px; justify-content: flex-end; margin-top: 18px; }
.pg-btn {
  min-width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  background: transparent;
  border: 1px solid var(--line);
  border-radius: var(--radius);
  color: var(--text-dim);
  font-family: var(--font-mono);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.25s var(--ease-out);
}
.pg-btn:hover:not(:disabled) { border-color: var(--acid); color: var(--acid); }
.pg-btn:disabled { opacity: 0.35; cursor: not-allowed; }
.pg-num.active {
  background: var(--acid);
  border-color: var(--acid);
  color: var(--acid-ink);
  font-weight: 700;
}
.pg-total { margin-left: 10px; font-size: 11px; color: var(--text-faint); }
</style>
