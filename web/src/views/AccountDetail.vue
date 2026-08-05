<template>
  <div class="page">
    <header class="page-head">
      <div class="section-tag reveal">
        <RouterLink to="/accounts" class="back mono">← 账号管理</RouterLink>
      </div>
      <h1 class="display display-md reveal mono" style="--d:.08s">{{ account?.email || '加载中' }}</h1>
      <div class="head-meta reveal" style="--d:.14s" v-if="account">
        <AppTag :tone="statusTone(account.status)">{{ statusLabel(account.status) }}</AppTag>
        <span class="dim">来源 {{ account.source }}</span>
        <span class="dim" v-if="account.group_name">分组 {{ account.group_name }}</span>
      </div>
    </header>

    <div class="detail-grid">
      <!-- 左：信息 + 日志 -->
      <div class="col-main">
        <section class="panel block reveal" style="--d:.2s">
          <div class="block-head">
            <span class="section-tag">凭据与状态</span>
            <div class="head-ops">
              <AppButton size="sm" variant="primary" :icon="RefreshCw" :loading="acting" @click="doRefresh">刷新</AppButton>
              <AppButton size="sm" :icon="HeartPulse" :loading="acting" @click="doCheck">检测</AppButton>
              <AppButton size="sm" :icon="Inbox" :loading="fetching" @click="doFetchMails">收信</AppButton>
            </div>
          </div>
          <div class="kv-grid" v-if="account">
            <div class="kv"><span class="kv-k mono">密码</span><span class="kv-v mono">{{ account.password || '—' }}</span></div>
            <div class="kv"><span class="kv-k mono">CLIENT ID</span><span class="kv-v mono">{{ short(account.client_id) }}</span></div>
            <div class="kv"><span class="kv-k mono">TOKEN 过期</span><span class="kv-v mono">{{ fmt(account.token_expires_at) }}</span></div>
            <div class="kv"><span class="kv-k mono">最近刷新</span><span class="kv-v mono">{{ fmt(account.last_refresh_at) }}</span></div>
            <div class="kv"><span class="kv-k mono">最近检测</span><span class="kv-v mono">{{ fmt(account.last_check_at) }}</span></div>
            <div class="kv"><span class="kv-k mono">最近保活</span><span class="kv-v mono">{{ fmt(account.last_keepalive_at) }}</span></div>
            <div class="kv"><span class="kv-k mono">最近收信</span><span class="kv-v mono">{{ fmt(account.last_mail_at) }}</span></div>
            <div class="kv"><span class="kv-k mono">创建时间</span><span class="kv-v mono">{{ fmt(account.created_at) }}</span></div>
          </div>
          <div v-if="account?.status_reason" class="reason">
            <AlertTriangle :size="14" />
            <span>{{ account.status_reason }}</span>
          </div>
        </section>

        <section class="panel block reveal" style="--d:.28s">
          <div class="block-head"><span class="section-tag">最近任务</span></div>
          <div v-if="logs.length" class="log-list">
            <div v-for="log in logs" :key="log.id" class="log-row">
              <span class="dot" :class="log.status === 'success' ? 'ok' : log.status === 'skip' ? 'info' : 'bad'" />
              <span class="log-type mono">{{ taskLabel(log.task_type) }}</span>
              <span class="log-msg">{{ log.message }}</span>
              <span class="log-time mono">{{ dayjs(log.created_at).format('MM-DD HH:mm:ss') }}</span>
            </div>
          </div>
          <AppEmpty v-else text="暂无任务记录" />
        </section>
      </div>

      <!-- 右：收信（收件箱 + 垃圾邮件合并） -->
      <section class="panel block col-mail reveal" style="--d:.36s">
        <div class="block-head">
          <span class="section-tag">收信 · {{ mailTotal }}</span>
          <AppButton size="sm" :icon="RefreshCw" :loading="fetching" @click="doFetchMails">在线收信</AppButton>
        </div>
        <div v-if="mails.length" class="mail-list">
          <div v-for="m in mails" :key="m.id" class="mail-item" @click="openMail(m)">
            <div class="mail-top">
              <span class="mail-from">{{ m.from_addr }}</span>
              <span class="mail-time mono">{{ fmt(m.received_at) }}</span>
            </div>
            <div class="mail-subject">
              <span v-if="m.folder === 'junk'" class="mail-junk mono">垃圾</span>
              {{ m.subject || '(无主题)' }}
              <span v-if="m.code" class="mail-code mono">{{ m.code }}</span>
            </div>
            <div class="mail-preview">{{ m.body_preview }}</div>
          </div>
        </div>
        <AppEmpty v-else text="暂无邮件" hint="点击右上角在线收信（含垃圾邮件）" />
        <AppPagination :page="mailPage" :size="20" :total="mailTotal" @change="(p) => { mailPage = p; loadMails() }" />
      </section>
    </div>

    <!-- 邮件详情 -->
    <AppModal v-model="showMail" :title="currentMail?.subject || '邮件详情'" tag="MESSAGE" width="680px">
      <div v-if="currentMail" class="mail-detail">
        <div class="mail-meta mono">
          <span>{{ currentMail.from_addr }}</span>
          <span>{{ fmt(currentMail.received_at) }}</span>
        </div>
        <div v-if="currentMail.code" class="code-box">
          <span class="code-label mono">验证码</span>
          <span class="code-value mono">{{ currentMail.code }}</span>
          <AppButton size="sm" :icon="Copy" @click="copyCode(currentMail.code)">复制</AppButton>
        </div>
        <div v-if="mailBodyLoading" class="mail-loading mono">LOADING …</div>
        <div v-else class="mail-body" v-html="currentMail.body || currentMail.body_preview"></div>
      </div>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import dayjs from 'dayjs'
import { RefreshCw, HeartPulse, Inbox, AlertTriangle, Copy } from 'lucide-vue-next'
import { accountApi, mailApi, type Account, type MailMessage, type TaskLog } from '../api'
import { toast } from '../components/ui/toast'
import AppButton from '../components/ui/AppButton.vue'
import AppTag from '../components/ui/AppTag.vue'
import AppModal from '../components/ui/AppModal.vue'
import AppPagination from '../components/ui/AppPagination.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

const route = useRoute()
const id = Number(route.params.id)

const STATUS: Record<string, { label: string; tone: any }> = {
  healthy: { label: '健康', tone: 'ok' },
  dead: { label: '失效', tone: 'bad' },
  locked: { label: '锁定', tone: 'warn' },
  error: { label: '异常', tone: 'warn' },
  unknown: { label: '未检测', tone: 'info' },
}

const account = ref<Account | null>(null)
const logs = ref<TaskLog[]>([])
const mails = ref<MailMessage[]>([])
const mailTotal = ref(0)
const mailPage = ref(1)
const acting = ref(false)
const fetching = ref(false)
const showMail = ref(false)
const mailBodyLoading = ref(false)
const currentMail = ref<MailMessage | null>(null)

function statusLabel(s: string) { return STATUS[s]?.label || s }
function statusTone(s: string) { return STATUS[s]?.tone || 'info' }
function fmt(t?: string) { return t ? dayjs(t).format('MM-DD HH:mm') : '—' }
function short(s?: string) { return s && s.length > 24 ? s.slice(0, 24) + '…' : s || '—' }
function taskLabel(t: string) {
  return ({ refresh: '刷新', health: '检测', keepalive: '保活', mail: '收信' } as any)[t] || t
}

async function loadDetail() {
  const { data } = await accountApi.get(id)
  account.value = data.account
  logs.value = data.recent_logs || []
}

async function loadMails() {
  const { data } = await mailApi.list(id, { page: mailPage.value, size: 20 })
  mails.value = data.items || []
  mailTotal.value = data.total
}

async function doRefresh() {
  acting.value = true
  try {
    await accountApi.refresh(id)
    toast.ok('Token 刷新成功')
    loadDetail()
  } catch { /* 已提示 */ } finally { acting.value = false }
}

async function doCheck() {
  acting.value = true
  try {
    await accountApi.check(id)
    toast.ok('账号健康')
    loadDetail()
  } catch { /* 已提示 */ } finally { acting.value = false }
}

async function doFetchMails() {
  fetching.value = true
  try {
    const { data } = await mailApi.fetch(id)
    toast.ok(`收信完成：收件箱 ${data.inbox} + 垃圾 ${data.junk}，新增 ${data.new} 封`)
    loadMails()
    loadDetail()
  } catch { /* 已提示 */ } finally { fetching.value = false }
}

async function openMail(m: MailMessage) {
  currentMail.value = m
  showMail.value = true
  if (!m.body) {
    mailBodyLoading.value = true
    try {
      const { data } = await mailApi.detail(m.id)
      currentMail.value = data
    } catch { /* 已提示 */ } finally { mailBodyLoading.value = false }
  }
}

function copyCode(code: string) {
  navigator.clipboard.writeText(code).then(() => toast.ok('验证码已复制'))
}

onMounted(() => {
  loadDetail()
  loadMails()
})
</script>

<style scoped>
.back { color: var(--text-faint); font-size: 11px; letter-spacing: 0.15em; transition: color 0.25s; }
.back:hover { color: var(--acid); }
.head-meta { display: flex; align-items: center; gap: 14px; font-size: 13px; }

.detail-grid {
  display: grid;
  /* minmax(0, …)：允许轨道收缩到内容最小宽度以下，
     否则超长日志 URL 会撑破轨道，把右列收信面板挤出视口 */
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
  gap: 14px;
  align-items: start;
}
.col-main { display: flex; flex-direction: column; gap: 14px; min-width: 0; }
.col-mail { min-width: 0; }
.block { padding: 22px 24px; }
.block-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
}
.head-ops { display: flex; gap: 8px; }

.kv-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1px;
  background: var(--line);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  overflow: hidden;
}
.kv { background: var(--bg-raised); padding: 12px 16px; display: flex; flex-direction: column; gap: 3px; }
.kv-k { font-size: 10px; letter-spacing: 0.18em; color: var(--text-faint); }
.kv-v { font-size: 13px; color: var(--text); word-break: break-all; }

.reason {
  margin-top: 14px;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 14px;
  border: 1px solid rgba(255, 178, 36, 0.3);
  border-radius: var(--radius);
  color: var(--warn);
  font-size: 12px;
}

.log-list { display: flex; flex-direction: column; }
.log-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 4px;
  border-bottom: 1px solid var(--line);
  font-size: 13px;
}
.log-row:last-child { border-bottom: none; }
.log-type { font-size: 11px; color: var(--acid); width: 36px; flex: none; }
.log-msg { flex: 1; min-width: 0; color: var(--text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.log-time { font-size: 11px; color: var(--text-faint); flex: none; }

/* 收件箱 */
.mail-list { display: flex; flex-direction: column; }
.mail-item {
  padding: 13px 10px;
  border-bottom: 1px solid var(--line);
  cursor: pointer;
  transition: all 0.25s var(--ease-out);
  border-radius: 4px;
}
.mail-item:hover { background: var(--bg-hover); transform: translateX(4px); }
.mail-item:last-child { border-bottom: none; }
.mail-top { display: flex; justify-content: space-between; font-size: 11px; color: var(--text-faint); }
.mail-from { max-width: 70%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mail-subject {
  font-weight: 700;
  margin: 5px 0 3px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13.5px;
}
.mail-code {
  background: var(--acid);
  color: var(--acid-ink);
  padding: 1px 8px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.1em;
}
.mail-junk {
  flex: none;
  padding: 1px 7px;
  border: 1px solid color-mix(in srgb, var(--warn) 45%, transparent);
  border-radius: 3px;
  font-size: 10px;
  color: var(--warn);
  letter-spacing: 0.08em;
}
.mail-preview {
  font-size: 12px;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 邮件详情 */
.mail-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-faint);
  margin-bottom: 14px;
}
.code-box {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  border: 1px solid color-mix(in srgb, var(--acid) 35%, transparent);
  background: var(--acid-glow);
  border-radius: var(--radius);
  margin-bottom: 16px;
}
.code-label { font-size: 10px; letter-spacing: 0.2em; color: var(--text-dim); }
.code-value { font-size: 24px; font-weight: 800; letter-spacing: 0.2em; color: var(--acid); flex: 1; }
.mail-loading { text-align: center; padding: 32px; color: var(--text-faint); letter-spacing: 0.3em; }
.mail-body {
  max-height: 440px;
  overflow-y: auto;
  font-size: 13.5px;
  color: var(--text);
  background: var(--bg);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  padding: 16px;
}
.mail-body :deep(a) { color: var(--acid); }
.mail-body :deep(img) { max-width: 100%; }

@media (max-width: 1100px) {
  .detail-grid { grid-template-columns: 1fr; }
}
</style>
