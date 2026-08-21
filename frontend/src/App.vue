<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { login, register } from './api'
import AuthModal from './components/AuthModal.vue'

const router = useRouter()

const showAuth = ref(false)
const authMode = ref('login')

const token = ref(localStorage.getItem('nikoblog_token') || '')
const user = ref(JSON.parse(localStorage.getItem('nikoblog_user') || 'null'))

const isLoggedIn = computed(() => !!token.value)
const isAdmin = computed(() => user.value?.role === 'admin')

// 深色模式（默认跟随系统）
const dark = ref(localStorage.getItem('nikoblog_dark') === '1')
const applyDark = () => {
  document.documentElement.classList.toggle('dark', dark.value)
  localStorage.setItem('nikoblog_dark', dark.value ? '1' : '0')
}
const toggleDark = () => {
  dark.value = !dark.value
  applyDark()
}

function openAuth(mode) {
  authMode.value = mode
  showAuth.value = true
}

async function handleAuth(payload) {
  const res = payload.mode === 'register'
    ? await register({
        username: payload.username,
        password: payload.password,
        nickname: payload.nickname,
        email: payload.email,
        security_questions: payload.security_questions
      })
    : await login({ username: payload.username, password: payload.password })
  if (payload.mode === 'login') {
    token.value = res.token
    user.value = res.user
    localStorage.setItem('nikoblog_token', res.token)
    localStorage.setItem('nikoblog_user', JSON.stringify(res.user))
    showAuth.value = false
  } else {
    alert('注册成功，请登录')
    authMode.value = 'login'
  }
}

function logout() {
  token.value = ''
  user.value = null
  localStorage.removeItem('nikoblog_token')
  localStorage.removeItem('nikoblog_user')
  router.push('/')
}

function goProfile() {
  router.push('/profile')
}

// 更新当前用户头像（个人中心上传头像后调用）
function updateAvatar(url) {
  if (user.value) {
    user.value.avatar = url
    localStorage.setItem('nikoblog_user', JSON.stringify(user.value))
  }
}

function goAdmin() {
  router.push('/admin')
}

function goHome() {
  router.push('/')
}

onMounted(() => {
  applyDark()
})
</script>

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 transition-colors">
    <!-- 顶部导航 -->
    <header class="sticky top-0 z-20 bg-white/80 dark:bg-gray-800/80 backdrop-blur border-b border-gray-200 dark:border-gray-700">
      <div class="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
        <button @click="goHome" class="flex items-center gap-2">
          <span class="text-xl font-bold text-blue-600 dark:text-blue-400">nikoblog</span>
          <span class="text-xs text-gray-400 hidden sm:inline">极简微博客</span>
        </button>
        <div class="flex items-center gap-3">
          <button
            @click="toggleDark"
            class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-300"
            :title="dark ? '切换到浅色模式' : '切换到深色模式'"
          >
            {{ dark ? '☀️' : '🌙' }}
          </button>
          <template v-if="isLoggedIn">
            <div class="flex items-center gap-2">
              <div class="w-7 h-7 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center text-blue-600 dark:text-blue-300 text-xs font-bold overflow-hidden">
                <img
                  v-if="user?.avatar"
                  :src="user.avatar"
                  class="w-full h-full object-cover"
                  alt="头像"
                />
                <template v-else>{{ (user?.nickname || user?.username || '?').charAt(0) }}</template>
              </div>
              <span class="text-sm text-gray-600 dark:text-gray-300">{{ user?.nickname || user?.username }}</span>
            </div>
            <button
              @click="goProfile"
              class="px-3 py-1.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700"
              title="个人中心"
            >个人</button>
            <button
              v-if="isAdmin"
              @click="goAdmin"
              class="px-3 py-1.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700"
              title="后台管理"
            >管理</button>
            <button
              @click="logout"
              class="px-3 py-1.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700"
            >退出</button>
          </template>
          <template v-else>
            <button
              @click="openAuth('login')"
              class="px-3 py-1.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700"
            >登录</button>
            <button
              @click="openAuth('register')"
              class="px-3 py-1.5 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700"
            >注册</button>
          </template>
        </div>
      </div>
    </header>

    <!-- 路由视图 -->
    <router-view
      :user="user"
      :is-logged-in="isLoggedIn"
      :is-admin="isAdmin"
      @open-auth="openAuth"
      @avatar-updated="updateAvatar"
    />

    <!-- 登录/注册弹窗 -->
    <AuthModal
      v-if="showAuth"
      :mode="authMode"
      @close="showAuth = false"
      @submit="handleAuth"
      @switch-mode="authMode = $event"
    />
  </div>
</template>
