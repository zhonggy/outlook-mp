<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="modelValue" class="modal-mask" @click.self="close">
        <div class="modal" :style="{ maxWidth: width }" role="dialog">
          <div class="modal-head">
            <div>
              <div class="section-tag" v-if="tag">{{ tag }}</div>
              <h3 class="modal-title">{{ title }}</h3>
            </div>
            <button class="modal-close" @click="close" aria-label="关闭">
              <X :size="16" :stroke-width="1.8" />
            </button>
          </div>
          <div class="modal-body">
            <slot />
          </div>
          <div v-if="$slots.footer" class="modal-foot">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'

defineProps<{
  modelValue: boolean
  title: string
  tag?: string
  width?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [boolean] }>()

function close() {
  emit('update:modelValue', false)
}
</script>

<style scoped>
.modal-mask {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: var(--modal-mask);
  backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.modal {
  width: 100%;
  background: var(--bg-overlay);
  border: 1px solid var(--line-strong);
  border-radius: 6px;
  box-shadow: var(--shadow-modal);
  overflow: hidden;
}
.modal-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 22px 24px 0;
}
.modal-title {
  margin: 6px 0 0;
  font-size: 20px;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.modal-close {
  background: none;
  border: 1px solid var(--line);
  border-radius: 4px;
  color: var(--text-dim);
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: all var(--dur) var(--ease-out);
}
.modal-close:hover { color: var(--text); border-color: var(--line-strong); transform: rotate(90deg); }
.modal-body { padding: 18px 24px; }
.modal-foot {
  padding: 14px 24px 20px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  border-top: 1px solid var(--line);
}

.modal-enter-active, .modal-leave-active { transition: opacity 0.3s var(--ease-out); }
.modal-enter-active .modal { animation: modal-in 0.4s var(--ease-spring); }
.modal-leave-active .modal { animation: modal-in 0.25s var(--ease-out) reverse; }
.modal-enter-from, .modal-leave-to { opacity: 0; }
@keyframes modal-in {
  from { opacity: 0; transform: translateY(26px) scale(0.97); }
  to   { opacity: 1; transform: translateY(0) scale(1); }
}
</style>
