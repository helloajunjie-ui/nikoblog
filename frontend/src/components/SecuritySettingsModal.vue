<script setup>
import { ref } from 'vue'
import { updateSecurity } from '../api'

const emit = defineEmits(['close', 'updated'])

const securityQuestions = ref([
  { question: '', answer: '' },
  { question: '', answer: '' },
  { question: '', answer: '' }
])
const submitting = ref(false)

async function save() {
  for (const qa of securityQuestions.value) {
    if (!qa.question.trim() || !qa.answer.trim()) {
      alert('请完整填写 3 个密保问题及答案')
      return
    }
  }
  submitting.value = true
  try {
    await updateSecurity({
      security_questions: securityQuestions.value.map(qa => ({
        question: qa.question.trim(),
        answer_hash: qa.answer.trim()
      }))
    })
    alert('密保更新成功')
    emit('updated')
    emit('close')
  } catch (e) {
    alert('更新失败: ' + e.message)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="emit('close')">
    <div class="bg-white dark:bg-gray-800 rounded-2xl p-6 w-full max-w-md shadow-xl max-h-[90vh] overflow-y-auto">
      <h2 class="text-lg font-bold mb-2 text-center">修改密保问题</h2>
      <p class="text-xs text-gray-500 text-center mb-4">设置 3 个密保问题，用于找回账号</p>

      <div class="space-y-3">
        <div v-for="(qa, i) in securityQuestions" :key="i" class="space-y-1">
          <input v-model="qa.question" type="text" :placeholder="`密保问题 ${i + 1}`"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm" />
          <input v-model="qa.answer" type="text" :placeholder="`答案 ${i + 1}`"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm" />
        </div>
      </div>

      <button @click="save" :disabled="submitting"
        class="w-full mt-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
        {{ submitting ? '保存中...' : '保存' }}
      </button>
      <button @click="emit('close')" class="w-full mt-2 py-2 text-sm text-gray-500 hover:underline">取消</button>
    </div>
  </div>
</template>
