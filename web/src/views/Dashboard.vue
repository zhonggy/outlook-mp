<template>
  <div class="page">
    <header class="page-head">
      <div class="section-tag reveal">01 / Overview</div>
      <h1 class="display display-lg reveal" style="--d:.08s">运行总览</h1>
      <p class="page-sub reveal" style="--d:.14s">账号资产、健康度与自动化任务的实时全景。</p>
    </header>

    <!-- 大数字 -->
    <div class="stats">
      <div v-for="(s, i) in statCards" :key="s.label" class="panel stat reveal"
        :style="{ '--d': `${0.18 + i * 0.07}s` }">
        <div class="stat-top">
          <component :is="s.icon" :size="16" :stroke-width="1.6" class="stat-icon" :style="{ color: s.color }" />
          <span class="stat-tag mono">{{ s.tag }}</span>
        </div>
        <div class="stat-num"><CountUp :value="s.value" /></div>
        <div class="stat-label">{{ s.label }}</div>
        <div class="stat-bar"><span :style="{ width: s.ratio + '%', background: s.color }" /></div>
      </div>
    </div>

    <div class="grid-2">
      <!-- 状态分布 -->
      <section class="panel block reveal" style="--d:.5s">
        <div class="block-head">
          <span class="section-tag">02 / 健康分布</span>
        </div>
        <div class="dist">
          <div v-for="row in distRows" :key="row.key" class="dist-row" @click="goAccounts(row.key)">
            <span class="dot" :class="row.dot" />
            <span class="dist-label">{{ row.label }}</span>
            <div class="dist-track">
              <span class="dist-fill" :class="row.key" :style="{ width: distPct(row.key) + '%' }" />
            </div>
            <span class="dist-num mono">{{ counts[row.key] || 0 }}</span>
          </div>
        </div>
        <div v-if="!total" class="dist-empty">
          <AppEmpty text="还没有账号" hint="去账号管理导入，或等待自动化系统上传" />
        </div>
      </section>

      <!-- 调度状态 -->
      <section class="panel block reveal" style="--d:.58s">
        <div class="block-head">
          <span class="section-tag">03 / 自动化任务</span>
          <AppTag :tone="schedule.enabled ? 'ok' : 'info'">
            {{ schedule.enabled ? '运行中' : '已停用' }}
          </AppTag>
        </div>
        <div class="sched">
          <div v-for="t in schedRows" :key="t.key" class="sched-row">
            <component :is="t.icon" :size="15" :stroke-width="1.6" class="sched-icon" />
            <div class="sched-meta">
              <span class="sched-name">{{ t.label }}</span>
              <span class="sched-iv mono">每 {{ schedule[t.key]?.interval || '-' }}</span>
            </div>
            <AppTag v-if="schedule[t.key]?.due" tone="warn">待执行</AppTag>
            <span v-else class="sched-last mono">{{ fmtLast(schedule[t.key]?.last_run) }}</span>
          </div>
        </div>
      </section>
    </div>

    <!-- 最近活动 -->
    <section class="panel block reveal" style="--d:.66s">
      <div class="block-head">
        <span class="section-tag">04 / 最近任务</span>
        <RouterLink to="/tasks" class="more mono">全部日志 →</RouterLink>
      </div>
      <div v-if="recentLogs.length" class="log-list">
        <div v-for="log in recentLogs" :key="log.id" class="log-row">
          <span class="dot" :class="logTone(log.status)" />
          <span class="log-type mono">{{ taskLabel(log.task_type) }}</span>
          <span class="log-email">{{ log.email || '全局' }}</span>
          <span class="log-msg">{{ log.message }}</span>
          <span class="log-time mono">{{ fmtTime(log.created_at) }}</span>
        </div>
      </div>
      <AppEmpty v-else text="暂无任务记录" hint="定时任务到点后自动产生日志" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import {
  Users, ShieldCheck, Skull, MailPlus, RefreshCw, HeartPulse, Zap, Inbox,
} from 'lucide-vue-next'
import { statsApi, taskApi, type TaskLog } from '../api'
import CountUp from '../components/ui/CountUp.vue'
import AppTag from '../components/ui/AppTag.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

const router = useRouter()
const data = ref<Record<string, any>>({})
const schedule = ref<Record<string, any>>({})
const recentLogs = ref<TaskLog[]>([])

const counts = computed(() => data.value.status_counts || {})
const total = computed(() => data.value.total_accounts || 0)

// 颜色引用 CSS 变量，随浅/深主题自动适配（勿写死十六进制）
const statCards = computed(() => [
  { label: '账号总数', tag: 'TOTAL', value: total.value, icon: Users, color: 'var(--neutral)', ratio: 100 },
  { label: '健康账号', tag: 'HEALTHY', value: counts.value.healthy || 0, icon: ShieldCheck, color: 'var(--ok)', ratio: pctOf(counts.value.healthy) },
  { label: '失效账号', tag: 'DEAD', value: counts.value.dead || 0, icon: Skull, color: 'var(--bad)', ratio: pctOf(counts.value.dead) },
  { label: '已收邮件', tag: 'MAILS', value: data.value.mail_count || 0, icon: MailPlus, color: 'var(--acid)', ratio: 100 },
])

const distRows = [
  { key: 'healthy', label: '健康', dot: 'ok' },
  { key: 'dead', label: '失效', dot: 'bad' },
  { key: 'locked', label: '锁定', dot: 'warn' },
  { key: 'error', label: '异常', dot: 'warn' },
  { key: 'unknown', label: '未检测', dot: 'info' },
]

const schedRows = [
  { key: 'refresh', label: 'Token 刷新', icon: RefreshCw },
  { key: 'health', label: '健康检测', icon: HeartPulse },
  { key: 'keepalive', label: '保活', icon: Zap },
  { key: 'mail', label: '收信', icon: Inbox },
]

function pctOf(n?: number) {
  return total.value ? Math.round(((n || 0) / total.value) * 100) : 0
}
function distPct(key: string) {
  return pctOf(counts.value[key])
}
function fmtLast(t?: string) {
  return t ? '上次 ' + dayjs(t).format('MM-DD HH:mm') : '从未执行'
}
function fmtTime(t: string) {
  return dayjs(t).format('MM-DD HH:mm:ss')
}
function logTone(s: string) {
  return s === 'success' ? 'ok' : s === 'skip' ? 'info' : 'bad'
}
function taskLabel(t: string) {
  return ({ refresh: '刷新', health: '检测', keepalive: '保活', mail: '收信' } as any)[t] || t
}
function goAccounts(status: string) {
  router.push({ path: '/accounts', query: { status } })
}

onMounted(async () => {
  const [d, s, l] = await Promise.all([
    statsApi.dashboard(),
    taskApi.schedule(),
    taskApi.logs({ page: 1, size: 8 }),
  ])
  data.value = d.data
  schedule.value = s.data
  recentLogs.value = l.data.items || []
})
</script>

<style scoped>
/* 大数字卡 */
.stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-bottom: 14px;
}
.stat { padding: 22px 22px 18px; overflow: hidden; transition: all 0.35s var(--ease-out); }
.stat:hover { transform: translateY(-3px); border-color: var(--line-strong); }
.stat-top { display: flex; justify-content: space-between; align-items: center; }
.stat-tag { font-size: 10px; letter-spacing: 0.2em; color: var(--text-faint); }
.stat-num {
  font-size: clamp(2.4rem, 4vw, 3.4rem);
  font-weight: 800;
  letter-spacing: -0.03em;
  line-height: 1;
  margin: 14px 0 4px;
}
.stat-label { color: var(--text-dim); font-size: 13px; }
.stat-bar {
  margin-top: 16px;
  height: 2px;
  background: var(--line);
  border-radius: 2px;
  overflow: hidden;
}
.stat-bar span { display: block; height: 100%; transition: width 1s var(--ease-out); }

/* 两栏 */
.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  margin-bottom: 14px;
}
.block { padding: 22px 24px; }
.block-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.more { font-size: 11px; letter-spacing: 0.1em; color: var(--text-faint); transition: color 0.25s; }
.more:hover { color: var(--acid); }

/* 状态分布 */
.dist { display: flex; flex-direction: column; gap: 14px; }
.dist-row {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
  transition: background 0.25s;
}
.dist-row:hover { background: var(--bg-hover); }
.dist-label { width: 52px; font-size: 13px; color: var(--text-dim); flex: none; }
.dist-track {
  flex: 1;
  height: 6px;
  background: var(--bg);
  border-radius: 4px;
  overflow: hidden;
}
.dist-fill { display: block; height: 100%; border-radius: 4px; transition: width 1s var(--ease-out); }
.dist-fill.healthy { background: var(--ok); }
.dist-fill.dead { background: var(--bad); }
.dist-fill.locked, .dist-fill.error { background: var(--warn); }
.dist-fill.unknown { background: var(--info); }
.dist-num { width: 40px; text-align: right; font-weight: 700; flex: none; }
.dist-empty { padding: 12px 0; }

/* 调度 */
.sched { display: flex; flex-direction: column; }
.sched-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 13px 4px;
  border-bottom: 1px solid var(--line);
}
.sched-row:last-child { border-bottom: none; }
.sched-icon { color: var(--acid); flex: none; }
.sched-meta { flex: 1; display: flex; flex-direction: column; }
.sched-name { font-weight: 600; font-size: 13.5px; }
.sched-iv { font-size: 11px; color: var(--text-faint); }
.sched-last { font-size: 11px; color: var(--text-faint); }

/* 日志流 */
.log-list { display: flex; flex-direction: column; }
.log-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 11px 6px;
  border-bottom: 1px solid var(--line);
  font-size: 13px;
  transition: background 0.25s;
}
.log-row:hover { background: var(--bg-hover); }
.log-row:last-child { border-bottom: none; }
.log-type { font-size: 11px; color: var(--acid); width: 40px; flex: none; }
.log-email { color: var(--text); width: 220px; flex: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.log-msg { flex: 1; color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.log-time { font-size: 11px; color: var(--text-faint); flex: none; }

@media (max-width: 1100px) {
  .stats { grid-template-columns: repeat(2, 1fr); }
  .grid-2 { grid-template-columns: 1fr; }
}
</style>
