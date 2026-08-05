<template>
  <label class="field" :class="{ block }">
    <span v-if="label" class="field-label">{{ label }}</span>
    <div class="select-wrap">
      <select
        class="select"
        :value="modelValue"
        @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
      >
        <option v-if="placeholder" value="">{{ placeholder }}</option>
        <option v-for="opt in options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>
      <ChevronDown :size="14" :stroke-width="1.8" class="select-arrow" />
    </div>
  </label>
</template>

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next'

withDefaults(defineProps<{
  modelValue: string
  options: { value: string; label: string }[]
  label?: string
  placeholder?: string
  /** 占满父容器宽度（表单场景）；默认 inline-block 适配工具条 */
  block?: boolean
}>(), { block: false })

defineEmits<{ 'update:modelValue': [string] }>()
</script>

<style scoped>
.field { display: inline-block; }
.field.block { display: block; width: 100%; }
.field-label {
  display: block;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-faint);
  margin-bottom: 8px;
}
.select-wrap { position: relative; display: inline-block; min-width: 130px; }
.field.block .select-wrap { display: block; width: 100%; }
.select {
  width: 100%;
  appearance: none;
  background: var(--bg);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 13px;
  padding: 9px 34px 9px 12px;
  cursor: pointer;
  outline: none;
  transition: all var(--dur) var(--ease-out);
}
.select:hover, .select:focus { border-color: var(--acid); }
.select:focus { box-shadow: 0 0 0 3px var(--acid-glow); }
.select option { background: var(--bg-overlay); color: var(--text); }
.select-arrow {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-faint);
  pointer-events: none;
}
</style>
