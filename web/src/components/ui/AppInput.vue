<template>
  <label class="field">
    <span v-if="label" class="field-label">{{ label }}</span>
    <div class="field-wrap" :class="{ focused }">
      <component :is="icon" v-if="icon" :size="15" :stroke-width="1.6" class="field-icon" />
      <textarea
        v-if="type === 'textarea'"
        class="field-input field-textarea"
        :value="modelValue"
        :placeholder="placeholder"
        :rows="rows"
        :disabled="disabled"
        @input="onInput"
        @focus="focused = true"
        @blur="focused = false"
      />
      <input
        v-else
        class="field-input"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        @input="onInput"
        @focus="focused = true"
        @blur="focused = false"
        @keyup.enter="$emit('enter')"
      />
    </div>
  </label>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Component } from 'vue'

withDefaults(defineProps<{
  modelValue: string
  label?: string
  placeholder?: string
  type?: string
  icon?: Component
  rows?: number
  disabled?: boolean
}>(), { type: 'text', rows: 3, disabled: false })

const emit = defineEmits<{
  'update:modelValue': [string]
  enter: []
}>()

const focused = ref(false)

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
}
</script>

<style scoped>
.field { display: block; }
.field-label {
  display: block;
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-faint);
  margin-bottom: 8px;
}
.field-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--bg);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  padding: 0 12px;
  transition: all var(--dur) var(--ease-out);
}
.field-wrap.focused {
  border-color: var(--acid);
  box-shadow: 0 0 0 3px var(--acid-glow);
}
.field-icon { color: var(--text-faint); flex: none; }
.field-input {
  flex: 1;
  background: none;
  border: none;
  outline: none;
  color: var(--text);
  font-family: var(--font-sans);
  font-size: 14px;
  padding: 11px 0;
  min-width: 0;
}
.field-input::placeholder { color: var(--text-faint); }
.field-input:disabled { color: var(--text-faint); cursor: not-allowed; }
.field-textarea { resize: vertical; line-height: 1.6; font-family: var(--font-mono); font-size: 12.5px; }
</style>
