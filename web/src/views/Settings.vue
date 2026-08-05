<template>
  <div class="page">
    <header class="page-head">
      <div class="section-tag reveal">04 / System</div>
      <h1 class="display display-lg reveal" style="--d:.08s">系统设置</h1>
      <p class="page-sub reveal" style="--d:.14s">调度策略、自动化对接密钥与账户安全。</p>
    </header>

    <div class="settings-grid">
      <!-- 调度 -->
      <section class="panel block reveal" style="--d:.2s">
        <div class="block-head">
          <span class="section-tag">定时任务调度</span>
          <button class="switch" :class="{ on: schedule.enabled }" @click="toggleEnabled">
            <span class="knob" />
          </button>
        </div>
        <div class="form-grid">
          <AppSelect v-model="intervals.refresh" label="Token 刷新周期" :options="intervalOptions.refresh" block />
          <AppSelect v-model="intervals.health" label="健康检测周期" :options="intervalOptions.health" block />
          <AppSelect v-model="intervals.keepalive" label="保活周期" :options="intervalOptions.keepalive" block />
          <AppSelect v-model="intervals.mail" label="收信周期" :options="intervalOptions.mail" block />
        </div>
        <div class="block-foot">
          <span class="hint mono">「停用」可暂停单项任务，保存后生效</span>
          <AppButton variant="primary" :icon="Save" :loading="saving" @click="saveIntervals">保存调度</AppButton>
        </div>
      </section>

      <!-- API Key -->
      <section class="panel block reveal" style="--d:.24s">
        <div class="block-head">
          <span class="section-tag">API 密钥</span>
          <AppButton size="sm" variant="primary" :icon="Plus" @click="showCreateKey = true">创建</AppButton>
        </div>
        <div v-if="apiKeys.length" class="key-list">
          <div v-for="k in apiKeys" :key="k.id" class="key-row">
            <div class="key-meta">
              <span class="key-name">{{ k.name }}</span>
              <span class="key-value mono">{{ k.key }}</span>
            </div>
            <span class="key-used mono">{{ k.last_used_at ? dayjs(k.last_used_at).format('MM-DD HH:mm') : '从未使用' }}</span>
            <div class="key-ops">
              <button class="act" title="复制" @click="copy(k.key)"><Copy :size="14" /></button>
              <button class="act act-bad" title="删除" @click="removeKey(k.id)"><Trash2 :size="14" /></button>
            </div>
          </div>
        </div>
        <AppEmpty v-else text="暂无密钥" hint="创建一个用于自动化系统对接" :icon="KeyRound" />

        <!-- 上传接口说明 -->
        <div class="api-docs">
          <div class="docs-head">
            <span class="section-tag">上传接口说明</span>
            <button class="act" title="复制调用示例" @click="copy(curlExample)"><Copy :size="14" /></button>
          </div>
          <div class="docs-row">
            <span class="docs-k mono">端点</span>
            <span class="docs-v mono">POST {{ origin }}/api/v1/ingest/accounts</span>
          </div>
          <div class="docs-row">
            <span class="docs-k mono">认证</span>
            <span class="docs-v mono">请求头 X-API-Key: &lt;上方创建的密钥&gt;</span>
          </div>
          <div class="docs-row">
            <span class="docs-k mono">行为</span>
            <span class="docs-v">按 email 幂等：已存在则更新凭据，不存在则新建；非法邮箱自动跳过。返回 {created, updated, skipped, errors} 汇总。</span>
          </div>
          <div class="docs-row">
            <span class="docs-k mono">字段</span>
            <span class="docs-v mono docs-fields">
              email* 邮箱 · password 密码 · client_id 应用 ID · refresh_token 刷新令牌 · tags 标签(逗号分隔) · group_name 分组 · remark 备注
            </span>
          </div>
          <pre class="docs-code mono">{{ curlExample }}</pre>
        </div>
      </section>

      <!-- 日志清理 -->
      <section class="panel block reveal" style="--d:.28s">
        <div class="block-head">
          <span class="section-tag">日志清理</span>
          <span class="hint mono">当前 {{ logTotal }} 条</span>
        </div>
        <div class="form-grid">
          <AppSelect v-model="logRetention" label="日志最长保留" :options="retentionOptions" block />
        </div>
        <div class="block-foot">
          <span class="hint mono">过期日志由调度器每小时自动清理</span>
          <div class="foot-ops">
            <AppButton :icon="Eraser" :loading="cleaning" @click="cleanupNow">清理过期</AppButton>
            <AppButton variant="danger" :icon="Trash2" @click="onClearAll">清空</AppButton>
            <AppButton variant="primary" :icon="Save" :loading="saving" @click="saveRetention">保存</AppButton>
          </div>
        </div>
      </section>

      <!-- 密码 -->
      <section class="panel block reveal" style="--d:.36s">
        <div class="block-head"><span class="section-tag">账户安全</span></div>
        <div class="form-grid">
          <AppInput v-model="pwd.old_password" label="原密码" type="password" :icon="Lock" />
          <AppInput v-model="pwd.new_password" label="新密码" type="password" :icon="Lock" />
        </div>
        <div class="block-foot">
          <span class="hint mono">至少 6 位，修改后需重新登录</span>
          <AppButton variant="primary" :icon="Save" :loading="saving" @click="changePwd">修改密码</AppButton>
        </div>
      </section>
    </div>

    <!-- 创建密钥 -->
    <AppModal v-model="showCreateKey" title="创建 API 密钥" tag="NEW KEY" width="420px">
      <AppInput v-model="newKeyName" label="密钥名称" placeholder="如 自动注册器" @enter="createKey" />
      <template #footer>
        <AppButton @click="showCreateKey = false">取消</AppButton>
        <AppButton variant="primary" @click="createKey">创建</AppButton>
      </template>
    </AppModal>

    <!-- 删除密钥确认 -->
    <AppModal v-model="confirmDel.show" title="删除密钥" tag="DANGER ZONE" width="400px">
      <p class="confirm-text">删除后使用该密钥的自动化上传将立即失效。</p>
      <template #footer>
        <AppButton @click="confirmDel.show = false">取消</AppButton>
        <AppButton variant="danger" @click="confirmDel.run">确认删除</AppButton>
      </template>
    </AppModal>

    <!-- 清空日志确认 -->
    <AppModal v-model="confirmClear.show" title="清空任务日志" tag="DANGER ZONE" width="400px">
      <p class="confirm-text">即将删除全部 {{ logTotal }} 条任务日志（含错误历史），此操作不可恢复。只想删旧日志的话，请改用「清理过期」。</p>
      <template #footer>
        <AppButton @click="confirmClear.show = false">取消</AppButton>
        <AppButton variant="danger" :loading="cleaning" @click="clearAll">确认清空</AppButton>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { Save, Plus, Copy, Trash2, Lock, KeyRound, Eraser } from 'lucide-vue-next'
import { apikeyApi, authApi, taskApi } from '../api'
import { toast } from '../components/ui/toast'
import AppButton from '../components/ui/AppButton.vue'
import AppInput from '../components/ui/AppInput.vue'
import AppSelect from '../components/ui/AppSelect.vue'
import AppModal from '../components/ui/AppModal.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

const schedule = reactive<Record<string, any>>({ enabled: true })
const intervals = reactive({ refresh: '', health: '', keepalive: '', mail: '' })
const saving = ref(false)
const pwd = reactive({ old_password: '', new_password: '' })
const apiKeys = ref<any[]>([])
const showCreateKey = ref(false)
const newKeyName = ref('')
const confirmDel = reactive({ show: false, run: () => {} })

// 日志清理：保留期（Go duration 字符串，"0" = 永久）+ 当前日志数
const logRetention = ref('720h')
const logTotal = ref(0)
const cleaning = ref(false)
const confirmClear = reactive({ show: false })
const retentionOptions = [
  { value: '72h', label: '3 天' },
  { value: '168h', label: '7 天' },
  { value: '360h', label: '15 天' },
  { value: '720h', label: '30 天（默认）' },
  { value: '2160h', label: '90 天' },
  { value: '0', label: '永久保留' },
]

// 上传接口文档：地址取当前访问源，示例可直接复制使用
const origin = window.location.origin
const curlExample = computed(() => `curl -X POST ${origin}/api/v1/ingest/accounts \\
  -H "X-API-Key: <上方创建的密钥>" \\
  -H "Content-Type: application/json" \\
  -d '[{"email":"a@hotmail.com","password":"密码","client_id":"xxx","refresh_token":"xxx"}]'`)

// 周期预设：value 为 Go duration 字符串（后端 time.ParseDuration 解析，"0" 表示停用）
type Opt = { value: string; label: string }
const PRESETS: Record<'refresh' | 'health' | 'keepalive' | 'mail', Opt[]> = {
  refresh: [
    { value: '1h', label: '每 1 小时' },
    { value: '2h', label: '每 2 小时' },
    { value: '4h', label: '每 4 小时' },
    { value: '6h', label: '每 6 小时' },
    { value: '8h', label: '每 8 小时' },
    { value: '12h', label: '每 12 小时' },
    { value: '24h', label: '每 24 小时' },
    { value: '0', label: '停用' },
  ],
  health: [
    { value: '30m', label: '每 30 分钟' },
    { value: '1h', label: '每 1 小时' },
    { value: '2h', label: '每 2 小时' },
    { value: '4h', label: '每 4 小时' },
    { value: '6h', label: '每 6 小时' },
    { value: '12h', label: '每 12 小时' },
    { value: '24h', label: '每 24 小时' },
    { value: '0', label: '停用' },
  ],
  keepalive: [
    { value: '6h', label: '每 6 小时' },
    { value: '12h', label: '每 12 小时' },
    { value: '24h', label: '每 24 小时' },
    { value: '48h', label: '每 48 小时' },
    { value: '72h', label: '每 72 小时' },
    { value: '0', label: '停用' },
  ],
  mail: [
    { value: '5m', label: '每 5 分钟' },
    { value: '10m', label: '每 10 分钟' },
    { value: '15m', label: '每 15 分钟' },
    { value: '30m', label: '每 30 分钟' },
    { value: '1h', label: '每 1 小时' },
    { value: '2h', label: '每 2 小时' },
    { value: '0', label: '停用' },
  ],
}

// 后端返回 Go duration 字符串（如 12h0m0s / 30m0s / 0s），归一化为预设值（12h / 30m / 0）
function normalizeInterval(raw: string): string {
  if (!raw || raw === '0') return '0'
  const re = /(\d+(?:\.\d+)?)(h|m|s)/g
  let totalSec = 0
  let m: RegExpExecArray | null
  let matched = false
  while ((m = re.exec(raw))) {
    matched = true
    totalSec += parseFloat(m[1]) * (m[2] === 'h' ? 3600 : m[2] === 'm' ? 60 : 1)
  }
  if (!matched) return raw
  if (totalSec === 0) return '0'
  if (totalSec % 3600 === 0) return `${totalSec / 3600}h`
  if (totalSec % 60 === 0) return `${totalSec / 60}m`
  return `${totalSec}s`
}

// 当前值不在预设中时（历史自定义值），临时并入选项避免下拉显示空白
const intervalOptions = computed(() => {
  const out = {} as Record<keyof typeof PRESETS, Opt[]>
  for (const key of Object.keys(PRESETS) as (keyof typeof PRESETS)[]) {
    const list = [...PRESETS[key]]
    const cur = intervals[key]
    if (cur && !list.some(o => o.value === cur)) {
      list.unshift({ value: cur, label: `自定义（${cur}）` })
    }
    out[key] = list
  }
  return out
})

async function loadSchedule() {
  const { data } = await taskApi.schedule()
  Object.assign(schedule, data)
  intervals.refresh = normalizeInterval(data.refresh?.interval || '')
  intervals.health = normalizeInterval(data.health?.interval || '')
  intervals.keepalive = normalizeInterval(data.keepalive?.interval || '')
  intervals.mail = normalizeInterval(data.mail?.interval || '')
  logRetention.value = normalizeInterval(data.log_retention || '720h')
}

async function loadLogTotal() {
  const { data } = await taskApi.logs({ page: 1, size: 1 })
  logTotal.value = data.total
}

async function saveRetention() {
  saving.value = true
  try {
    await taskApi.setSchedule({ log_retention: logRetention.value })
    toast.ok('日志保留期已保存')
  } finally { saving.value = false }
}

async function cleanupNow() {
  cleaning.value = true
  try {
    const { data } = await taskApi.cleanupLogs()
    if (data.deleted > 0) {
      toast.ok(`已清理 ${data.deleted} 条过期日志`)
    } else {
      const label = retentionOptions.find((o) => o.value === logRetention.value)?.label || logRetention.value
      toast.info(`没有超过保留期（${label.replace('（默认）', '')}）的日志；想全部删除请用「清空」`)
    }
    logTotal.value = data.remaining
  } finally { cleaning.value = false }
}

function onClearAll() {
  if (!logTotal.value) return toast.info('当前没有日志')
  confirmClear.show = true
}

async function clearAll() {
  cleaning.value = true
  try {
    const { data } = await taskApi.cleanupLogs(true)
    toast.ok(`已清空 ${data.deleted} 条日志`)
    confirmClear.show = false
    logTotal.value = data.remaining
  } finally { cleaning.value = false }
}

async function toggleEnabled() {
  schedule.enabled = !schedule.enabled
  await taskApi.setSchedule({ enabled: schedule.enabled })
  toast.ok(schedule.enabled ? '调度器已启用' : '调度器已停用')
}

async function saveIntervals() {
  saving.value = true
  try {
    await taskApi.setSchedule({
      refresh_interval: intervals.refresh,
      health_interval: intervals.health,
      keepalive_interval: intervals.keepalive,
      mail_interval: intervals.mail,
    })
    toast.ok('调度配置已保存')
    loadSchedule()
  } finally { saving.value = false }
}

async function changePwd() {
  if (!pwd.old_password || !pwd.new_password) return toast.warn('请输入密码')
  saving.value = true
  try {
    await authApi.changePassword(pwd.old_password, pwd.new_password)
    toast.ok('密码已修改')
    pwd.old_password = pwd.new_password = ''
  } finally { saving.value = false }
}

async function loadKeys() {
  const { data } = await apikeyApi.list()
  apiKeys.value = data.items || []
}

async function createKey() {
  if (!newKeyName.value.trim()) return toast.warn('请输入名称')
  const { data } = await apikeyApi.create(newKeyName.value.trim())
  toast.ok('密钥已创建')
  showCreateKey.value = false
  newKeyName.value = ''
  loadKeys()
}

function removeKey(id: number) {
  confirmDel.show = true
  confirmDel.run = async () => {
    await apikeyApi.remove(id)
    toast.ok('已删除')
    confirmDel.show = false
    loadKeys()
  }
}

function copy(text: string) {
  navigator.clipboard.writeText(text).then(() => toast.ok('已复制'))
}

onMounted(() => {
  loadSchedule()
  loadKeys()
  loadLogTotal()
})
</script>

<style scoped>
.settings-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  /* 不设 align-items: start：同行两块拉伸等高，底边对齐 */
}
.block {
  padding: 24px 26px;
  display: flex;
  flex-direction: column;
}
/* 与底部操作行保持最小间距（margin-top:auto 在短块中只追加空隙） */
.block .form-grid { margin-bottom: 18px; }
.block-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.form-grid { display: flex; flex-direction: column; gap: 14px; }
.block-foot {
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid var(--line);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.hint { font-size: 11px; color: var(--text-faint); }
.foot-ops { display: flex; gap: 8px; flex: none; }
.confirm-text { color: var(--text-dim); margin: 4px 0 8px; }

/* 开关 */
.switch {
  width: 44px;
  height: 24px;
  border-radius: 100px;
  border: 1px solid var(--line-strong);
  background: var(--bg);
  cursor: pointer;
  position: relative;
  transition: all 0.3s var(--ease-out);
  padding: 0;
}
.switch .knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--text-faint);
  transition: all 0.3s var(--ease-spring);
}
.switch.on { background: var(--acid); border-color: var(--acid); }
.switch.on .knob { left: 23px; background: var(--acid-ink); }

/* API Key 列表 */
.key-list { display: flex; flex-direction: column; }
.key-row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 6px;
  border-bottom: 1px solid var(--line);
  transition: background 0.25s;
}
.key-row:hover { background: var(--bg-hover); }
.key-row:last-child { border-bottom: none; }
.key-meta { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.key-name { font-weight: 700; font-size: 13px; }
.key-value {
  font-size: 11px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.key-used { font-size: 11px; color: var(--text-faint); flex: none; }
.key-ops { display: flex; gap: 4px; }

/* 上传接口文档 */
.api-docs {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--line);
}
.docs-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}
.docs-row {
  display: flex;
  gap: 12px;
  align-items: baseline;
  margin-bottom: 10px;
  line-height: 1.7;
}
.docs-k {
  flex: none;
  width: 36px;
  font-size: 11px;
  letter-spacing: 0.1em;
  color: var(--text-faint);
}
.docs-v { font-size: 12px; color: var(--text-dim); word-break: break-all; }
.docs-fields {
  padding: 8px 12px;
  border: 1px dashed var(--line-strong);
  border-radius: var(--radius);
  font-size: 11px;
  color: var(--text-faint);
  line-height: 1.9;
}
.docs-code {
  margin: 12px 0 0;
  padding: 12px 14px;
  background: var(--bg);
  border: 1px solid var(--line);
  border-radius: var(--radius);
  font-size: 11px;
  line-height: 1.9;
  color: var(--text-dim);
  white-space: pre;
  overflow-x: auto;
}
.act {
  width: 28px; height: 28px;
  display: grid;
  place-items: center;
  background: none;
  border: 1px solid transparent;
  border-radius: 4px;
  color: var(--text-faint);
  cursor: pointer;
  transition: all 0.25s var(--ease-out);
}
.act:hover { color: var(--acid); border-color: var(--line-strong); }
.act-bad:hover { color: var(--bad); border-color: rgba(255, 84, 112, 0.4); }

@media (max-width: 1100px) {
  .settings-grid { grid-template-columns: 1fr; }
}
</style>
