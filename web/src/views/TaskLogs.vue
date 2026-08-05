<template>
  <div class="page">
    <header class="page-head">
      <div class="section-tag reveal">03 / Activity</div>
      <h1 class="display display-lg reveal" style="--d:.08s">任务日志</h1>
      <p class="page-sub reveal" style="--d:.14s">自动化任务的完整执行轨迹。</p>
    </header>

    <div class="toolbar reveal" style="--d:.2s">
      <AppSelect v-model="query.type" placeholder="全部类型" :options="typeOptions" />
      <AppSelect v-model="query.status" placeholder="全部结果" :options="statusOptions" />
      <AppButton :icon="SearchIcon" @click="load(1)">查询</AppButton>
    </div>

    <section class="panel reveal" style="--d:.28s">
      <div v-if="loading" class="loading mono">LOADING …</div>
      <AppEmpty v-else-if="!items.length" text="暂无日志" :icon="ScrollText" />
      <div v-else class="log-list">
        <div v-for="(log, i) in items" :key="log.id" class="log-row"
          :style="{ '--d': `${0.3 + Math.min(i, 12) * 0.025}s` }">
          <span class="dot" :class="tone(log.status)" />
          <span class="log-type mono">{{ taskLabel(log.task_type) }}</span>
          <span class="log-email">{{ log.email || '全局任务' }}</span>
          <span class="log-msg">{{ log.message }}</span>
          <AppTag :tone="tagTone(log.status)">{{ statusLabel(log.status) }}</AppTag>
          <span class="log-dur mono">{{ (log.duration_ms / 1000).toFixed(1) }}s</span>
          <span class="log-time mono">{{ dayjs(log.created_at).format('MM-DD HH:mm:ss') }}</span>
        </div>
      </div>
      <div class="table-foot">
        <AppPagination :page="query.page" :size="query.size" :total="total" @change="load" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { Search as SearchIcon, ScrollText } from 'lucide-vue-next'
import { taskApi, type TaskLog } from '../api'
import AppButton from '../components/ui/AppButton.vue'
import AppSelect from '../components/ui/AppSelect.vue'
import AppTag from '../components/ui/AppTag.vue'
import AppPagination from '../components/ui/AppPagination.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

const typeOptions = [
  { value: 'refresh', label: 'Token 刷新' },
  { value: 'health', label: '健康检测' },
  { value: 'keepalive', label: '保活' },
  { value: 'mail', label: '收信' },
]
const statusOptions = [
  { value: 'success', label: '成功' },
  { value: 'fail', label: '失败' },
  { value: 'skip', label: '跳过' },
]

const items = ref<TaskLog[]>([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ type: '', status: '', page: 1, size: 20 })

function taskLabel(t: string) {
  return ({ refresh: '刷新', health: '检测', keepalive: '保活', mail: '收信' } as any)[t] || t
}
function statusLabel(s: string) {
  return ({ success: '成功', fail: '失败', skip: '跳过' } as any)[s] || s
}
function tone(s: string) {
  return s === 'success' ? 'ok' : s === 'skip' ? 'info' : 'bad'
}
function tagTone(s: string) {
  return s === 'success' ? 'ok' : s === 'skip' ? 'info' : 'bad'
}

async function load(page?: number) {
  if (page) query.page = page
  loading.value = true
  try {
    const { data } = await taskApi.logs({ ...query })
    items.value = data.items || []
    total.value = data.total
  } finally {
    loading.value = false
  }
}

onMounted(() => load())
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 16px; }
.loading { padding: 48px; text-align: center; color: var(--text-faint); letter-spacing: 0.3em; }

.log-list { display: flex; flex-direction: column; }
.log-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--line);
  font-size: 13px;
  opacity: 0;
  animation: rise 0.45s var(--ease-out) forwards;
  animation-delay: var(--d, 0s);
  transition: background 0.25s;
}
.log-row:hover { background: var(--bg-hover); }
.log-row:last-child { border-bottom: none; }
.log-type { font-size: 11px; color: var(--acid); width: 36px; flex: none; }
.log-email { width: 240px; flex: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.log-msg { flex: 1; min-width: 0; color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.log-dur { font-size: 11px; color: var(--text-faint); width: 48px; text-align: right; flex: none; }
.log-time { font-size: 11px; color: var(--text-faint); flex: none; }
.table-foot { padding: 14px 20px; }
</style>
