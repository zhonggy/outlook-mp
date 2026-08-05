<template>
  <div class="login">
    <!-- 背景装饰：超大描边字 -->
    <div class="bg-word" aria-hidden="true">
      <span class="w1 display outline-text">MAIL</span>
      <span class="w2 display outline-text">OPS</span>
    </div>
    <div class="scanline" aria-hidden="true" />

    <div class="login-grid">
      <!-- 左侧：品牌陈述 -->
      <section class="brand">
        <div class="brand-tag section-tag reveal" style="--d:.05s">Outlook Manager</div>
        <h1 class="display display-xl">
          <span class="reveal" style="--d:.12s">账号</span>
          <span class="reveal acid" style="--d:.2s">运营</span>
          <span class="reveal" style="--d:.28s">中枢</span>
        </h1>
        <p class="brand-sub reveal" style="--d:.38s">
          注册 · 保活 · 收信 · 检测 —— 全自动邮箱资产管理系统。
          长期驻守服务器，让每一枚账号保持鲜活。
        </p>
        <div class="brand-metrics reveal" style="--d:.48s">
          <div class="metric">
            <span class="metric-num mono">24/7</span>
            <span class="metric-label">无人值守</span>
          </div>
          <div class="metric">
            <span class="metric-num mono">Token</span>
            <span class="metric-label">自动焕新</span>
          </div>
          <div class="metric">
            <span class="metric-num mono">Graph</span>
            <span class="metric-label">实时收信</span>
          </div>
        </div>
      </section>

      <!-- 右侧：登录卡 -->
      <section class="panel login-panel reveal" style="--d:.3s">
        <div class="lp-head">
          <span class="dot acid" />
          <span class="mono lp-title">AUTH / 接入控制台</span>
        </div>
        <form @submit.prevent="onLogin">
          <AppInput v-model="form.username" label="用户名" placeholder="admin" :icon="User" />
          <AppInput v-model="form.password" label="密码" type="password" placeholder="••••••••"
            :icon="Lock" @enter="onLogin" />
          <AppButton variant="primary" size="lg" block :loading="loading" :icon="ArrowRight">
            进入系统
          </AppButton>
        </form>
        <div class="lp-foot mono">SECURE CHANNEL · JWT / HS256</div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock, ArrowRight } from 'lucide-vue-next'
import { toast } from '../components/ui/toast'
import AppInput from '../components/ui/AppInput.vue'
import AppButton from '../components/ui/AppButton.vue'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const loading = ref(false)
const form = reactive({ username: '', password: '' })

async function onLogin() {
  if (!form.username || !form.password) {
    toast.warn('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    toast.ok('接入成功，欢迎回来')
    router.push((route.query.redirect as string) || '/')
  } catch {
    /* 拦截器已提示 */
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  padding: 40px 24px;
}

/* 背景大字 */
.bg-word {
  position: absolute;
  inset: 0;
  pointer-events: none;
  overflow: hidden;
}
.bg-word .display { font-size: clamp(10rem, 26vw, 22rem); position: absolute; }
.w1 { top: -6%; right: -4%; transform: rotate(4deg); }
.w2 { bottom: -10%; left: -3%; transform: rotate(-3deg); }

/* 扫描线 */
.scanline {
  position: absolute;
  top: 0; bottom: 0;
  width: 25%;
  background: linear-gradient(90deg, transparent, color-mix(in srgb, var(--acid) 3%, transparent), transparent);
  animation: scan 9s linear infinite;
  pointer-events: none;
}

.login-grid {
  display: grid;
  grid-template-columns: 1.2fr 420px;
  gap: 80px;
  align-items: center;
  max-width: 1180px;
  width: 100%;
  position: relative;
}

.brand h1 { display: flex; flex-direction: column; }
.brand-sub {
  margin: 28px 0 0;
  color: var(--text-dim);
  font-size: 16px;
  max-width: 460px;
}
.brand-metrics {
  display: flex;
  gap: 48px;
  margin-top: 48px;
  padding-top: 28px;
  border-top: 1px solid var(--line);
}
.metric { display: flex; flex-direction: column; gap: 2px; }
.metric-num { font-size: 20px; font-weight: 700; color: var(--acid); }
.metric-label { font-size: 12px; color: var(--text-faint); letter-spacing: 0.1em; }

/* 登录卡 */
.login-panel {
  padding: 28px;
  backdrop-filter: blur(8px);
  background: color-mix(in srgb, var(--bg-raised) 82%, transparent);
}
.login-panel form { display: flex; flex-direction: column; gap: 18px; }
.lp-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--line);
}
.lp-title { font-size: 11px; letter-spacing: 0.22em; color: var(--text-dim); }
.lp-foot {
  margin-top: 22px;
  padding-top: 14px;
  border-top: 1px solid var(--line);
  font-size: 10px;
  letter-spacing: 0.2em;
  color: var(--text-faint);
  text-align: center;
}

@media (max-width: 960px) {
  .login-grid { grid-template-columns: 1fr; gap: 40px; }
  .brand-metrics { gap: 28px; }
}
</style>
