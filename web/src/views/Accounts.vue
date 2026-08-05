<template>
  <div class="page">
    <header class="page-head">
      <div class="section-tag reveal">02 / Accounts</div>
      <h1 class="display display-lg reveal" style="--d:.08s">账号管理</h1>
      <p class="page-sub reveal" style="--d:.14s">
        共 <span class="acid mono">{{ total }}</span> 枚账号资产。支持导入、导出、批量操作与单账号运维。
      </p>
    </header>

    <!-- 筛选 + 操作 -->
    <div class="toolbar reveal" style="--d:.2s">
      <div class="filters">
        <AppInput v-model="query.keyword" placeholder="搜索邮箱…" :icon="SearchIcon" @enter="load(1)" />
        <AppSelect v-model="query.status" placeholder="全部状态" :options="statusOptions" />
        <AppSelect v-model="query.group" placeholder="全部分组" :options="groupOptions" />
        <AppButton :icon="SearchIcon" @click="load(1)">查询</AppButton>
      </div>
      <div class="ops">
        <AppButton variant="primary" :icon="Plus" @click="showCreate = true">添加</AppButton>
        <AppButton :icon="Upload" @click="showImport = true">导入</AppButton>
        <AppButton :icon="Download" @click="openExport">导出</AppButton>
        <AppButton :icon="HeartPulse" :loading="checking" @click="onBatchTask('check')">
          健康检测{{ selection.size ? `(${selection.size})` : '' }}
        </AppButton>
        <AppButton :icon="RefreshCw" :loading="refreshing" @click="onBatchTask('refresh')">
          令牌刷新{{ selection.size ? `(${selection.size})` : '' }}
        </AppButton>
        <AppButton variant="danger" :icon="Eraser" @click="onDeleteDead">清理失效</AppButton>
        <AppButton variant="danger" :icon="Trash2" :disabled="!selection.size" @click="onBatchDelete">
          删除{{ selection.size ? `(${selection.size})` : '' }}
        </AppButton>
      </div>
    </div>

    <!-- 账号表 -->
    <section class="panel reveal" style="--d:.28s">
      <div class="table-head mono">
        <span class="col-check">
          <input type="checkbox" :checked="allChecked" @change="toggleAll" />
        </span>
        <span class="col-mail">邮箱</span>
        <span class="col-status">状态</span>
        <span class="col-group">分组</span>
        <span class="col-tags">标签</span>
        <span class="col-time">最近刷新</span>
        <span class="col-act">操作</span>
      </div>

      <div v-if="loading" class="loading mono">LOADING …</div>
      <AppEmpty v-else-if="!items.length" text="没有匹配的账号" hint="调整筛选条件，或导入新账号" />

      <div v-else class="table-body">
        <div v-for="(row, i) in items" :key="row.id" class="trow"
          :style="{ '--d': `${0.3 + Math.min(i, 12) * 0.03}s` }">
          <span class="col-check">
            <input type="checkbox" :checked="selection.has(row.id)" @change="toggle(row.id)" />
          </span>
          <span class="col-mail">
            <RouterLink :to="`/accounts/${row.id}`" class="mail-link">{{ row.email }}</RouterLink>
            <span class="mail-src mono">{{ row.source }}</span>
          </span>
          <span class="col-status">
            <AppTag :tone="statusTone(row.status)">{{ statusLabel(row.status) }}</AppTag>
          </span>
          <span class="col-group dim">{{ row.group_name || '—' }}</span>
          <span class="col-tags">
            <span v-for="t in splitTags(row.tags)" :key="t" class="mini-tag mono">{{ t }}</span>
            <span v-if="!row.tags" class="faint">—</span>
          </span>
          <span class="col-time mono dim">{{ fmtTime(row.last_refresh_at) }}</span>
          <span class="col-act">
            <button class="act" title="刷新 Token" @click="onAction(row, 'refresh')"><RefreshCw :size="14" /></button>
            <button class="act" title="健康检测" @click="onAction(row, 'check')"><HeartPulse :size="14" /></button>
            <button class="act" title="编辑" @click="onEdit(row)"><Pencil :size="14" /></button>
            <button class="act act-bad" title="删除" @click="onDelete(row)"><Trash2 :size="14" /></button>
          </span>
        </div>
      </div>

      <div class="table-foot">
        <AppPagination :page="query.page" :size="query.size" :total="total" @change="load" />
      </div>
    </section>

    <!-- 添加账号 -->
    <AppModal v-model="showCreate" title="添加账号" tag="NEW ACCOUNT" width="480px">
      <div class="form-grid">
        <AppInput v-model="createForm.email" label="邮箱 *" placeholder="xxx@hotmail.com" />
        <AppInput v-model="createForm.password" label="密码" />
        <AppInput v-model="createForm.client_id" label="Client ID" />
        <AppInput v-model="createForm.refresh_token" label="Refresh Token" type="textarea" :rows="2" />
        <AppInput v-model="createForm.group_name" label="分组" />
        <AppInput v-model="createForm.tags" label="标签" placeholder="逗号分隔" />
      </div>
      <template #footer>
        <AppButton @click="showCreate = false">取消</AppButton>
        <AppButton variant="primary" :loading="saving" @click="onCreate">保存</AppButton>
      </template>
    </AppModal>

    <!-- 编辑账号 -->
    <AppModal v-model="showEdit" title="编辑账号" tag="EDIT" width="480px">
      <div class="form-grid">
        <AppInput :model-value="editForm.email" label="邮箱" disabled />
        <AppInput v-model="editForm.password" label="密码" />
        <AppInput v-model="editForm.group_name" label="分组" />
        <AppInput v-model="editForm.tags" label="标签" placeholder="逗号分隔" />
        <AppInput v-model="editForm.proxy" label="代理" placeholder="留空使用全局代理" />
        <AppInput v-model="editForm.remark" label="备注" type="textarea" :rows="2" />
      </div>
      <template #footer>
        <AppButton @click="showEdit = false">取消</AppButton>
        <AppButton variant="primary" :loading="saving" @click="onSaveEdit">保存</AppButton>
      </template>
    </AppModal>

    <!-- 导入 -->
    <AppModal v-model="showImport" title="批量导入" tag="IMPORT" width="600px">
      <div class="import-hint mono">
        支持格式：email----密码----client_id----refresh_token / 邮箱: x | 密码: y / JSON / CSV
      </div>
      <AppInput v-model="importText" type="textarea" :rows="10" placeholder="粘贴账号，每行一个" />
      <template #footer>
        <AppButton @click="showImport = false">取消</AppButton>
        <AppButton variant="primary" :loading="saving" @click="onImport">开始导入</AppButton>
      </template>
    </AppModal>

    <!-- 导出 -->
    <AppModal v-model="showExport" title="导出账号" tag="EXPORT" width="440px">
      <div class="form-grid">
        <AppSelect v-model="exportForm.format" label="导出格式" :options="exportFormatOptions" block />
        <AppSelect v-model="exportForm.status" label="账号状态" :options="exportStatusOptions" block />
        <AppSelect v-model="exportForm.limit" label="导出数量" :options="exportLimitOptions" block />
      </div>
      <p v-if="query.keyword || query.group" class="export-hint mono">
        同时应用当前筛选：<template v-if="query.keyword">关键词「{{ query.keyword }}」</template>
        <template v-if="query.keyword && query.group"> · </template>
        <template v-if="query.group">分组「{{ query.group }}」</template>
      </p>
      <template #footer>
        <AppButton @click="showExport = false">取消</AppButton>
        <AppButton variant="primary" :icon="Download" :loading="exporting" @click="doExport">导出</AppButton>
      </template>
    </AppModal>

    <!-- 批量任务确认（健康检测 / 令牌刷新，无选中=全量，有选中=仅选中） -->
    <AppModal v-model="batchConfirm.show" :title="batchConfirm.title" tag="BATCH" width="420px">
      <p class="confirm-text">{{ batchConfirm.text }}</p>
      <template #footer>
        <AppButton @click="batchConfirm.show = false">取消</AppButton>
        <AppButton variant="primary" :icon="Play" @click="batchConfirm.run">开始执行</AppButton>
      </template>
    </AppModal>

    <!-- 删除确认 -->
    <AppModal v-model="confirm.show" title="确认删除" tag="DANGER ZONE" width="400px">
      <p class="confirm-text">{{ confirm.text }}</p>
      <template #footer>
        <AppButton @click="confirm.show = false">取消</AppButton>
        <AppButton variant="danger" :loading="saving" @click="confirm.run">确认删除</AppButton>
      </template>
    </AppModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import dayjs from 'dayjs'
import {
  Search as SearchIcon, Plus, Upload, Download, Trash2, RefreshCw, HeartPulse, Pencil, Eraser, Play,
} from 'lucide-vue-next'
import { accountApi, type Account } from '../api'
import { toast } from '../components/ui/toast'
import AppButton from '../components/ui/AppButton.vue'
import AppInput from '../components/ui/AppInput.vue'
import AppSelect from '../components/ui/AppSelect.vue'
import AppTag from '../components/ui/AppTag.vue'
import AppModal from '../components/ui/AppModal.vue'
import AppPagination from '../components/ui/AppPagination.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'

const route = useRoute()

const STATUS: Record<string, { label: string; tone: any }> = {
  healthy: { label: '健康', tone: 'ok' },
  dead: { label: '失效', tone: 'bad' },
  locked: { label: '锁定', tone: 'warn' },
  error: { label: '异常', tone: 'warn' },
  unknown: { label: '未检测', tone: 'info' },
}
const statusOptions = Object.entries(STATUS).map(([value, v]) => ({ value, label: v.label }))

const items = ref<Account[]>([])
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const selection = ref<Set<number>>(new Set())
const groups = ref<string[]>([])

const query = reactive({
  keyword: '',
  status: (route.query.status as string) || '',
  group: '',
  page: 1,
  size: 20,
})

const showCreate = ref(false)
const showEdit = ref(false)
const showImport = ref(false)
const showExport = ref(false)
const exporting = ref(false)
const checking = ref(false)
const refreshing = ref(false)
const batchConfirm = reactive({ show: false, title: '', text: '', run: () => {} })
const importText = ref('')
const createForm = reactive({ email: '', password: '', client_id: '', refresh_token: '', group_name: '', tags: '' })
const editForm = reactive({ id: 0, email: '', password: '', group_name: '', tags: '', proxy: '', remark: '' })
const exportForm = reactive({ format: 'txt', status: '', limit: '' })
const confirm = reactive({ show: false, text: '', run: () => {} })

const exportFormatOptions = [
  { value: 'txt', label: 'TXT（邮箱----密码----client_id----refresh_token）' },
  { value: 'csv', label: 'CSV（表格，含状态/分组/标签）' },
  { value: 'json', label: 'JSON（完整字段）' },
]
// 状态默认跟随列表当前筛选，也可在弹窗里改选
const exportStatusOptions = computed(() => [{ value: '', label: '全部状态' }, ...statusOptions])
const exportLimitOptions = [
  { value: '', label: '全部' },
  { value: '50', label: '前 50 条' },
  { value: '100', label: '前 100 条' },
  { value: '200', label: '前 200 条' },
  { value: '500', label: '前 500 条' },
  { value: '1000', label: '前 1000 条' },
]

const groupOptions = computed(() => groups.value.map((g) => ({ value: g, label: g })))
const allChecked = computed(() => items.value.length > 0 && items.value.every((a) => selection.value.has(a.id)))

function statusLabel(s: string) { return STATUS[s]?.label || s }
function statusTone(s: string) { return STATUS[s]?.tone || 'info' }
function splitTags(s: string) { return s ? s.split(',').map((t) => t.trim()).filter(Boolean) : [] }
function fmtTime(t?: string) { return t ? dayjs(t).format('MM-DD HH:mm') : '—' }

async function load(page?: number) {
  if (page) query.page = page
  loading.value = true
  try {
    const { data } = await accountApi.list({ ...query })
    items.value = data.items || []
    total.value = data.total
    selection.value = new Set()
  } finally {
    loading.value = false
  }
}

function toggle(id: number) {
  const next = new Set(selection.value)
  next.has(id) ? next.delete(id) : next.add(id)
  selection.value = next
}
function toggleAll() {
  selection.value = allChecked.value ? new Set() : new Set(items.value.map((a) => a.id))
}

async function onCreate() {
  if (!createForm.email) return toast.warn('邮箱必填')
  saving.value = true
  try {
    await accountApi.create(createForm)
    toast.ok('账号已添加')
    showCreate.value = false
    Object.assign(createForm, { email: '', password: '', client_id: '', refresh_token: '', group_name: '', tags: '' })
    load(1)
  } finally { saving.value = false }
}

function onEdit(row: Account) {
  Object.assign(editForm, {
    id: row.id, email: row.email, password: row.password || '',
    group_name: row.group_name, tags: row.tags, proxy: row.proxy, remark: row.remark,
  })
  showEdit.value = true
}

async function onSaveEdit() {
  saving.value = true
  try {
    await accountApi.update(editForm.id, {
      password: editForm.password, group_name: editForm.group_name,
      tags: editForm.tags, proxy: editForm.proxy, remark: editForm.remark,
    })
    toast.ok('已保存')
    showEdit.value = false
    load()
  } finally { saving.value = false }
}

function onDelete(row: Account) {
  confirm.text = `即将删除账号 ${row.email}，此操作不可恢复。`
  confirm.show = true
  confirm.run = async () => {
    saving.value = true
    try {
      await accountApi.remove(row.id)
      toast.ok('已删除')
      confirm.show = false
      load()
    } finally { saving.value = false }
  }
}

function onBatchDelete() {
  confirm.text = `即将删除选中的 ${selection.value.size} 个账号，此操作不可恢复。`
  confirm.show = true
  confirm.run = async () => {
    saving.value = true
    try {
      await accountApi.batchDelete([...selection.value])
      toast.ok('批量删除完成')
      confirm.show = false
      load()
    } finally { saving.value = false }
  }
}

async function onImport() {
  if (!importText.value.trim()) return toast.warn('请粘贴账号内容')
  saving.value = true
  try {
    const { data } = await accountApi.importText(importText.value)
    toast.ok(`导入完成：新建 ${data.created}，更新 ${data.updated}，跳过 ${data.skipped}`)
    showImport.value = false
    importText.value = ''
    load(1)
  } finally { saving.value = false }
}

function openExport() {
  exportForm.status = query.status
  showExport.value = true
}

async function doExport() {
  exporting.value = true
  try {
    const params: Record<string, string> = { format: exportForm.format }
    if (exportForm.status) params.status = exportForm.status
    if (exportForm.limit) params.limit = exportForm.limit
    if (query.keyword) params.keyword = query.keyword
    if (query.group) params.group = query.group
    const token = localStorage.getItem('om_token') || ''
    const r = await fetch(accountApi.exportUrl(params), { headers: { Authorization: `Bearer ${token}` } })
    if (!r.ok) throw new Error(`HTTP ${r.status}`)
    const blob = await r.blob()
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `accounts-${dayjs().format('YYYYMMDD-HHmmss')}.${exportForm.format}`
    a.click()
    URL.revokeObjectURL(a.href)
    toast.ok('导出成功')
    showExport.value = false
  } catch {
    toast.bad('导出失败，请重试')
  } finally {
    exporting.value = false
  }
}

// 批量任务：无选中 → 全量；有选中 → 仅选中的账号
function onBatchTask(kind: 'check' | 'refresh') {
  const n = selection.value.size
  const action = kind === 'check' ? '健康检测' : '令牌刷新'
  batchConfirm.title = n ? `${action}（选中 ${n} 个）` : `一键${action}`
  batchConfirm.text = n
    ? `将对选中的 ${n} 个账号执行${action}，逐账号请求微软接口。`
    : `将对全部账号逐一执行${action}。账号较多时可能需要几分钟，期间按钮保持加载状态，请勿重复点击。`
  batchConfirm.show = true
  batchConfirm.run = () => runBatchTask(kind)
}

async function runBatchTask(kind: 'check' | 'refresh') {
  batchConfirm.show = false
  const busy = kind === 'check' ? checking : refreshing
  busy.value = true
  try {
    const ids = [...selection.value]
    const { data } = kind === 'check'
      ? (ids.length ? await accountApi.batchCheck(ids) : await accountApi.checkAll())
      : (ids.length ? await accountApi.batchRefresh(ids) : await accountApi.refreshAll())
    toast.ok(`${kind === 'check' ? '检测' : '刷新'}完成：共 ${data.total}，成功 ${data.success}，失败 ${data.fail}，跳过 ${data.skip}`)
    load()
  } catch { /* 拦截器已提示 */ } finally {
    busy.value = false
  }
}

function onDeleteDead() {
  accountApi.list({ status: 'dead', page: 1, size: 1 }).then(({ data }) => {
    if (!data.total) return toast.info('当前没有失效账号')
    confirm.text = `即将删除全部 ${data.total} 个失效账号，此操作不可恢复。`
    confirm.show = true
    confirm.run = async () => {
      saving.value = true
      try {
        const { data: res } = await accountApi.deleteByStatus('dead')
        toast.ok(`已清理 ${res.count} 个失效账号`)
        confirm.show = false
        load(1)
      } finally { saving.value = false }
    }
  })
}

async function onAction(row: Account, kind: 'refresh' | 'check') {
  try {
    if (kind === 'refresh') {
      await accountApi.refresh(row.id)
      toast.ok(`${row.email} 刷新成功`)
    } else {
      await accountApi.check(row.id)
      toast.ok(`${row.email} 健康`)
    }
  } catch { /* 拦截器已提示 */ }
  load()
}

onMounted(async () => {
  load()
  const { data } = await accountApi.groups()
  groups.value = data.groups || []
})
</script>

<style scoped>
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}
.filters { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.filters :deep(.field-wrap) { min-width: 200px; }
.ops { display: flex; gap: 10px; flex-wrap: wrap; }

/* 表格 */
.table-head {
  display: grid;
  grid-template-columns: 40px 2.4fr 110px 90px 1.2fr 130px 150px;
  gap: 12px;
  padding: 14px 20px;
  font-size: 10px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--text-faint);
  border-bottom: 1px solid var(--line);
  align-items: center;
}
.trow {
  display: grid;
  grid-template-columns: 40px 2.4fr 110px 90px 1.2fr 130px 150px;
  gap: 12px;
  padding: 13px 20px;
  border-bottom: 1px solid var(--line);
  align-items: center;
  font-size: 13px;
  opacity: 0;
  animation: rise 0.5s var(--ease-out) forwards;
  animation-delay: var(--d, 0s);
  transition: background 0.25s, transform 0.25s var(--ease-out);
  position: relative;
}
.trow:hover { background: var(--bg-hover); }
.trow::before {
  content: "";
  position: absolute;
  left: 0; top: 0; bottom: 0;
  width: 2px;
  background: var(--acid);
  transform: scaleY(0);
  transition: transform 0.3s var(--ease-spring);
}
.trow:hover::before { transform: scaleY(1); }
.trow:last-child { border-bottom: none; }

input[type="checkbox"] {
  width: 15px; height: 15px;
  accent-color: var(--acid);
  cursor: pointer;
}

.mail-link { color: var(--text); font-weight: 600; display: block; transition: color 0.25s; }
.mail-link:hover { color: var(--acid); }
.mail-src { font-size: 10px; color: var(--text-faint); letter-spacing: 0.06em; }

.mini-tag {
  display: inline-block;
  padding: 2px 8px;
  margin-right: 4px;
  border: 1px solid var(--line-strong);
  border-radius: 100px;
  font-size: 10.5px;
  color: var(--text-dim);
}

.col-act { display: flex; gap: 4px; }
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
.act:hover { color: var(--acid); border-color: var(--line-strong); background: var(--bg); }
.act-bad:hover { color: var(--bad); border-color: rgba(255, 84, 112, 0.4); }

.table-foot { padding: 14px 20px; }
.loading { padding: 48px; text-align: center; color: var(--text-faint); letter-spacing: 0.3em; }

.form-grid { display: flex; flex-direction: column; gap: 14px; }
.import-hint {
  font-size: 11px;
  color: var(--text-dim);
  padding: 10px 12px;
  border: 1px dashed var(--line-strong);
  border-radius: var(--radius);
  margin-bottom: 14px;
  line-height: 1.8;
}
.confirm-text { color: var(--text-dim); margin: 4px 0 8px; }
.export-hint {
  margin: 14px 0 0;
  font-size: 11px;
  color: var(--text-faint);
  padding: 8px 12px;
  border: 1px dashed var(--line-strong);
  border-radius: var(--radius);
}

@media (max-width: 1100px) {
  .table-head { display: none; }
  .trow { grid-template-columns: 40px 1fr; row-gap: 8px; }
  .col-status, .col-group, .col-tags, .col-time, .col-act { grid-column: 2; }
}
</style>
