<script setup>
import { ref, watch } from 'vue'
import {
  getSecurityQuestion,
  forgotUsername,
  forgotPassword
} from '../api'

const props = defineProps({
  mode: { type: String, default: 'login' }
})
const emit = defineEmits(['close', 'submit', 'switch-mode'])

// 视图：login / register / forgot
const view = ref('login')
// 找回类型：username / password
const forgotType = ref('username')
// 找回步骤：1=输入信息取问题, 2=答题
const forgotStep = ref(1)
const securityQuestion = ref('')
const securityLockMsg = ref('')

// 表单字段
const username = ref('')
const password = ref('')
const nickname = ref('')
const email = ref('')
const securityQuestions = ref([
  { question: '', answer: '' },
  { question: '', answer: '' },
  { question: '', answer: '' }
])
const submitting = ref(false)

// 找回表单
const forgotEmail = ref('')
const forgotUsernameInput = ref('')
const forgotAnswer = ref('')
const newPassword = ref('')

watch(() => props.mode, () => {
  view.value = 'login'
  resetForm()
})

function resetForm() {
  username.value = ''
  password.value = ''
  nickname.value = ''
  email.value = ''
  securityQuestions.value = [
    { question: '', answer: '' },
    { question: '', answer: '' },
    { question: '', answer: '' }
  ]
  forgotEmail.value = ''
  forgotUsernameInput.value = ''
  forgotAnswer.value = ''
  newPassword.value = ''
  securityQuestion.value = ''
  securityLockMsg.value = ''
  forgotStep.value = 1
}

function switchView(v) {
  view.value = v
  resetForm()
}

async function submit() {
  if (view.value === 'login') {
    if (!username.value || !password.value) {
      alert('请填写用户名和密码')
      return
    }
    submitting.value = true
    try {
      await emit('submit', {
        mode: 'login',
        username: username.value,
        password: password.value
      })
    } finally {
      submitting.value = false
    }
    return
  }

  if (view.value === 'register') {
    if (!username.value || !password.value || !email.value) {
      alert('请填写用户名、密码和邮箱')
      return
    }
    if (password.value.length < 6) {
      alert('密码至少 6 位')
      return
    }
    // 校验 3 个密保问答
    for (const qa of securityQuestions.value) {
      if (!qa.question.trim() || !qa.answer.trim()) {
        alert('请完整填写 3 个密保问题及答案')
        return
      }
    }
    submitting.value = true
    try {
      await emit('submit', {
        mode: 'register',
        username: username.value,
        password: password.value,
        nickname: nickname.value,
        email: email.value,
        security_questions: securityQuestions.value.map(qa => ({
          question: qa.question.trim(),
          answer_hash: qa.answer.trim()
        }))
      })
    } finally {
      submitting.value = false
    }
  }
}

// ===== 找回流程 =====
async function fetchQuestion() {
  securityLockMsg.value = ''
  if (forgotType.value === 'password' && !forgotUsernameInput.value) {
    alert('请填写用户名')
    return
  }
  if (!forgotEmail.value) {
    alert('请填写注册邮箱')
    return
  }
  submitting.value = true
  try {
    const data = { email: forgotEmail.value }
    if (forgotType.value === 'password') data.username = forgotUsernameInput.value
    const res = await getSecurityQuestion(data)
    securityQuestion.value = res.question
    forgotStep.value = 2
  } catch (e) {
    securityLockMsg.value = e.message
  } finally {
    submitting.value = false
  }
}

async function submitAnswer() {
  if (!forgotAnswer.value) {
    alert('请填写密保答案')
    return
  }
  submitting.value = true
  try {
    if (forgotType.value === 'username') {
      const res = await forgotUsername({
        email: forgotEmail.value,
        security_answer: forgotAnswer.value
      })
      alert('您的用户名是：' + res.username)
      switchView('login')
    } else {
      if (!newPassword.value || newPassword.value.length < 6) {
        alert('新密码至少 6 位')
        return
      }
      await forgotPassword({
        username: forgotUsernameInput.value,
        email: forgotEmail.value,
        security_answer: forgotAnswer.value,
        new_password: newPassword.value
      })
      alert('密码重置成功，请用新密码登录')
      switchView('login')
    }
  } catch (e) {
    securityLockMsg.value = e.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="emit('close')">
    <div class="bg-white dark:bg-gray-800 rounded-2xl p-6 w-full max-w-md shadow-xl max-h-[90vh] overflow-y-auto">
      <!-- ===== 登录 ===== -->
      <template v-if="view === 'login'">
        <h2 class="text-lg font-bold mb-4 text-center">登录</h2>
        <div class="space-y-3">
          <input v-model="username" type="text" placeholder="用户名"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <input v-model="password" type="password" placeholder="密码"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <button @click="submit" :disabled="submitting"
          class="w-full mt-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
          {{ submitting ? '处理中...' : '登录' }}
        </button>
        <div class="flex items-center justify-between mt-3 text-sm">
          <button class="text-blue-600 hover:underline" @click="switchView('register')">没有账号？去注册</button>
          <button class="text-gray-500 hover:underline" @click="switchView('forgot')">忘记密码/用户名？</button>
        </div>
      </template>

      <!-- ===== 注册 ===== -->
      <template v-else-if="view === 'register'">
        <h2 class="text-lg font-bold mb-4 text-center">注册</h2>
        <div class="space-y-3">
          <input v-model="username" type="text" placeholder="用户名（3-64位）"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <input v-model="nickname" type="text" placeholder="昵称（可选）"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <input v-model="email" type="email" placeholder="邮箱（用于找回账号）"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <input v-model="password" type="password" placeholder="密码（至少6位）"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />

          <div class="pt-2 border-t border-gray-200 dark:border-gray-700">
            <p class="text-xs text-gray-500 mb-2">设置 3 个密保问题（请牢记，用于找回账号）</p>
            <div v-for="(qa, i) in securityQuestions" :key="i" class="space-y-1 mb-2">
              <input v-model="qa.question" type="text" :placeholder="`密保问题 ${i + 1}`"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm" />
              <input v-model="qa.answer" type="text" :placeholder="`答案 ${i + 1}`"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm" />
            </div>
          </div>
        </div>
        <button @click="submit" :disabled="submitting"
          class="w-full mt-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
          {{ submitting ? '处理中...' : '注册' }}
        </button>
        <p class="text-center text-sm mt-3 text-gray-500">
          已有账号？
          <button class="text-blue-600 hover:underline" @click="switchView('login')">去登录</button>
        </p>
      </template>

      <!-- ===== 找回 ===== -->
      <template v-else>
        <h2 class="text-lg font-bold mb-4 text-center">找回账号</h2>

        <!-- 找回类型选择 -->
        <div class="flex gap-2 mb-4">
          <button @click="forgotType = 'username'; resetForm()"
            class="flex-1 py-2 text-sm rounded-lg border"
            :class="forgotType === 'username' ? 'bg-blue-600 text-white border-blue-600' : 'border-gray-300 dark:border-gray-600'">
            找回用户名
          </button>
          <button @click="forgotType = 'password'; resetForm()"
            class="flex-1 py-2 text-sm rounded-lg border"
            :class="forgotType === 'password' ? 'bg-blue-600 text-white border-blue-600' : 'border-gray-300 dark:border-gray-600'">
            重置密码
          </button>
        </div>

        <!-- 步骤 1：输入信息 -->
        <div v-if="forgotStep === 1" class="space-y-3">
          <input v-model="forgotEmail" type="email" placeholder="注册邮箱"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <input v-if="forgotType === 'password'" v-model="forgotUsernameInput" type="text" placeholder="用户名"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <button @click="fetchQuestion" :disabled="submitting"
            class="w-full py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {{ submitting ? '处理中...' : '下一步' }}
          </button>
        </div>

        <!-- 步骤 2：答题 -->
        <div v-else class="space-y-3">
          <div class="p-3 rounded-lg bg-gray-100 dark:bg-gray-700 text-sm">
            <p class="text-gray-500 mb-1">密保问题：</p>
            <p class="font-medium">{{ securityQuestion }}</p>
          </div>
          <input v-model="forgotAnswer" type="text" placeholder="密保答案"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <input v-if="forgotType === 'password'" v-model="newPassword" type="password" placeholder="新密码（至少6位）"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
          <button @click="submitAnswer" :disabled="submitting"
            class="w-full py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {{ submitting ? '处理中...' : (forgotType === 'username' ? '找回用户名' : '重置密码') }}
          </button>
          <button @click="forgotStep = 1" class="w-full py-2 text-sm text-gray-500 hover:underline">上一步</button>
        </div>

        <!-- 错误/锁定提示 -->
        <p v-if="securityLockMsg" class="mt-3 text-sm text-red-500 text-center">{{ securityLockMsg }}</p>

        <p class="text-center text-sm mt-3 text-gray-500">
          想起密码了？
          <button class="text-blue-600 hover:underline" @click="switchView('login')">去登录</button>
        </p>
      </template>
    </div>
  </div>
</template>
