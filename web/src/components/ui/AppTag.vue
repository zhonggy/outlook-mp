<template>
  <span class="tag" :class="`tag-${tone}`">
    <span class="dot" :class="dotTone" />
    <slot />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  tone?: 'ok' | 'bad' | 'warn' | 'info' | 'acid'
}>(), { tone: 'info' })

const dotTone = computed(() => props.tone)
</script>

<style scoped>
.tag {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 3px 10px;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.06em;
  border-radius: 100px;
  border: 1px solid;
  white-space: nowrap;
}
.tag-ok   { color: var(--ok);   border-color: rgba(61, 220, 132, 0.3);  background: rgba(61, 220, 132, 0.07); }
.tag-bad  { color: var(--bad);  border-color: rgba(255, 84, 112, 0.3);  background: rgba(255, 84, 112, 0.07); }
.tag-warn { color: var(--warn); border-color: rgba(255, 178, 36, 0.3);  background: rgba(255, 178, 36, 0.07); }
.tag-info { color: var(--info); border-color: rgba(122, 129, 148, 0.3); background: rgba(122, 129, 148, 0.07); }
.tag-acid { color: var(--acid); border-color: color-mix(in srgb, var(--acid) 30%, transparent); background: var(--acid-glow); }
</style>
