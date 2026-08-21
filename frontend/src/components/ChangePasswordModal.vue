<script setup>
import { ref } from 'vue'
import { changePassword } from '../api'

const emit = defineEmits(['close', 'updated'])

const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const submitting = ref(false)

async function save() {
  if (!oldPassword.value) {
    alert('请输入原密码')
    return
  }
  if (newPassword.value.length < 6) {
    alert('新密码长度至少 6 位')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    alert('两次输入的新密码不一致')
    return
  }
  submitting.value = true
  try {
    await changePassword({
      old_password: oldPassword.value,
      new_password: newPassword.value
    })
    alert('密码修改成功')
    emit('updated')
    emit('close')
  } catch (e) {
    alert('修改失败: ' + (e.response?.data?.error || e.message))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="emit('close')">
    <div class="bg-white dark:bg-gray-800 rounded-2xl p-6 w-full max-w-md shadow-xl max-h-[90vh] overflow-y-auto">
      <h2 class="text-lg font-bold mb-2 text-center">修改密码</h2>
      <p class="text-xs text-gray-500 text-center mb-4">验证原密码后直接设置新密码；忘记密码请走找回流程</p>

      <div class="space-y-3">
        <input v-model="oldPassword" type="password" placeholder="原密码"
          class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm" />
        <input v-model="newPassword" type="password" placeholder="新密码（至少 6 位）"
          class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm" />
        <input v-model="confirmPassword" type="password" placeholder="确认新密码"
          class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm" />
      </div>

      <button @click="save" :disabled="submitting"
        class="w-full mt-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
        {{ submitting ? '保存中...' : '确认修改' }}
      </button>
      <button @click="emit('close')" class="w-full mt-2 py-2 text-sm text-gray-500 hover:underline">取消</button>
    </div>
  </div>
</template>
