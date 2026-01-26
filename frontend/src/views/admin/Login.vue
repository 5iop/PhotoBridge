<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }

  loading.value = true
  error.value = ''

  try {
    await auth.login(username.value, password.value)
    router.push('/admin')
  } catch (err) {
    error.value = err.response?.data?.error || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4">
    <div class="card p-8 w-full max-w-md">
      <!-- Logo -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 mb-4">
          <svg class="w-16 h-16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="12" cy="12" r="4" stroke="#f6821f" stroke-width="1.5"/>
            <path d="M22 12C22 16.714 22 19.0711 20.5355 20.5355C19.0711 22 16.714 22 12 22C7.28595 22 4.92893 22 3.46447 20.5355C2 19.0711 2 16.714 2 12C2 7.28595 2 4.92893 3.46447 3.46447C4.92893 2 7.28595 2 12 2C16.714 2 19.0711 2 20.5355 3.46447C21.5093 4.43821 21.8356 5.80655 21.9449 8" stroke="#f6821f" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </div>
        <h1 class="text-2xl font-bold text-cf-text">PhotoBridge</h1>
        <p class="text-cf-muted mt-1">管理员登录</p>
      </div>

      <!-- Error message -->
      <div v-if="error" class="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
        {{ error }}
      </div>

      <!-- Form -->
      <form @submit.prevent="handleLogin" class="space-y-4">
        <div>
          <label class="label">用户名</label>
          <input
            v-model="username"
            type="text"
            class="input"
            placeholder="请输入用户名"
            autocomplete="username"
          />
        </div>

        <div>
          <label class="label">密码</label>
          <input
            v-model="password"
            type="password"
            class="input"
            placeholder="请输入密码"
            autocomplete="current-password"
          />
        </div>

        <button
          type="submit"
          class="btn btn-primary w-full py-3"
          :disabled="loading"
        >
          <svg v-if="loading" class="w-5 h-5 spinner" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span v-else>登 录</span>
        </button>
      </form>
    </div>
  </div>
</template>
