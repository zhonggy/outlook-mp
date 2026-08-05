<template>
  <span class="countup mono">{{ display }}</span>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  value: number
  duration?: number
}>(), { duration: 900 })

const display = ref('0')
let raf = 0

function animate(to: number) {
  cancelAnimationFrame(raf)
  const from = parseInt(display.value.replace(/\D/g, ''), 10) || 0
  const start = performance.now()
  const tick = (now: number) => {
    const t = Math.min(1, (now - start) / props.duration)
    const eased = 1 - Math.pow(1 - t, 3)
    display.value = Math.round(from + (to - from) * eased).toLocaleString()
    if (t < 1) raf = requestAnimationFrame(tick)
  }
  raf = requestAnimationFrame(tick)
}

onMounted(() => animate(props.value))
watch(() => props.value, (v) => animate(v))
</script>
