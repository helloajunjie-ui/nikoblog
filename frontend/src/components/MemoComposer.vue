<script setup>
import { ref, computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { createMemo, updateMemo, uploadImage, polishContent } from '../api'

const emit = defineEmits(['published', 'cancel-edit'])

const content = ref('')
const visibility = ref('public')
const images = ref([])
const uploading = ref(false)
const polishing = ref(false)
const fileInput = ref(null)
const textareaRef = ref(null)
// 编辑模式：edit=编辑, preview=预览, split=分屏
const mode = ref('edit')
// 编辑状态：editingId 非空表示正在编辑某条博文
const editingId = ref(null)
const isEditing = computed(() => editingId.value !== null)

// 实时预览渲染
const previewHtml = computed(() => {
  if (!content.value) return ''
  return DOMPurify.sanitize(marked.parse(content.value))
})

// 工具栏插入 Markdown 语法
function insertSyntax(before, after = '', placeholder = '') {
  const ta = textareaRef.value
  if (!ta) return
  const start = ta.selectionStart
  const end = ta.selectionEnd
  const selected = content.value.slice(start, end) || placeholder
  const newVal = content.value.slice(0, start) + before + selected + after + content.value.slice(end)
  content.value = newVal
  // 恢复光标
  requestAnimationFrame(() => {
    ta.focus()
    const pos = start + before.length + selected.length
    ta.setSelectionRange(pos, pos)
  })
}

const toolbarActions = [
  { label: 'B', title: '加粗', cls: 'font-bold', action: () => insertSyntax('**', '**', '加粗文字') },
  { label: 'I', title: '斜体', cls: 'italic', action: () => insertSyntax('*', '*', '斜体文字') },
  { label: 'H', title: '标题', cls: '', action: () => insertSyntax('## ', '', '标题') },
  { label: '❝', title: '引用', cls: '', action: () => insertSyntax('> ', '', '引用内容') },
  { label: '•', title: '无序列表', cls: '', action: () => insertSyntax('- ', '', '列表项') },
  { label: '1.', title: '有序列表', cls: '', action: () => insertSyntax('1. ', '', '列表项') },
  { label: '</>', title: '代码块', cls: 'font-mono', action: () => insertSyntax('```\n', '\n```', '代码') },
  { label: '🔗', title: '链接', cls: '', action: () => insertSyntax('[', '](https://)', '链接文字') },
  { label: '#', title: '标签', cls: 'text-blue-600 font-bold', action: () => insertSyntax('#', '', '标签') }
]

async function handleFileChange(e) {
  const files = Array.from(e.target.files || [])
  if (files.length === 0) return
  uploading.value = true
  try {
    for (const file of files) {
      if (file.size > 5 * 1024 * 1024) {
        alert('图片不能超过 5MB: ' + file.name)
        continue
      }
      const res = await uploadImage(file)
      images.value.push(res.url)
    }
  } catch (err) {
    alert('上传失败: ' + err.message)
  } finally {
    uploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

function removeImage(index) {
  images.value.splice(index, 1)
}

// AI 润色：调用配置好的 AI 模型优化当前内容，并用结果覆盖输入框
async function polish() {
  const text = content.value.trim()
  if (!text) {
    alert('请先输入内容再使用 AI 润色')
    return
  }
  polishing.value = true
  try {
    const res = await polishContent(text)
    content.value = res.content || ''
  } catch (err) {
    alert('AI 润色失败: ' + err.message)
  } finally {
    polishing.value = false
  }
}

// 进入编辑状态：回填博文内容/图片/可见性到输入框
function startEdit(memo) {
  editingId.value = memo.id
  content.value = memo.content || ''
  images.value = Array.isArray(memo.images) ? [...memo.images] : []
  visibility.value = memo.visibility === 'private' ? 'private' : 'public'
  mode.value = 'edit'
  // 滚动到输入框并聚焦
  requestAnimationFrame(() => {
    textareaRef.value?.focus()
  })
}

// 取消编辑，回到发布状态
function cancelEdit() {
  editingId.value = null
  content.value = ''
  images.value = []
  visibility.value = 'public'
  mode.value = 'edit'
  emit('cancel-edit')
}

async function publish() {
  const text = content.value.trim()
  if (!text && images.value.length === 0) {
    alert('请输入内容或添加图片')
    return
  }
  try {
    if (isEditing.value) {
      // 编辑模式：调用 PUT 更新现有博文
      await updateMemo(editingId.value, {
        content: text,
        visibility: visibility.value,
        images: images.value
      })
    } else {
      await createMemo({
        content: text,
        visibility: visibility.value,
        images: images.value
      })
    }
    editingId.value = null
    content.value = ''
    images.value = []
    mode.value = 'edit'
    emit('published')
  } catch (err) {
    alert(isEditing.value ? '保存失败: ' + err.message : '发布失败: ' + err.message)
  }
}

defineExpose({ startEdit })
</script>

<template>
  <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden">
    <!-- 工具栏 -->
    <div class="flex items-center gap-1 px-3 py-2 border-b border-gray-100 dark:border-gray-700 flex-wrap">
      <button
        v-for="act in toolbarActions"
        :key="act.title"
        @click="act.action"
        :title="act.title"
        class="px-2 py-1 text-sm rounded hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-300"
        :class="act.cls"
      >{{ act.label }}</button>

      <div class="flex-1"></div>

      <!-- 模式切换 -->
      <div class="flex items-center gap-1 text-xs">
        <button
          @click="mode = 'edit'"
          class="px-2 py-1 rounded"
          :class="mode === 'edit' ? 'bg-blue-600 text-white' : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700'"
        >编辑</button>
        <button
          @click="mode = 'split'"
          class="px-2 py-1 rounded"
          :class="mode === 'split' ? 'bg-blue-600 text-white' : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700'"
        >分屏</button>
        <button
          @click="mode = 'preview'"
          class="px-2 py-1 rounded"
          :class="mode === 'preview' ? 'bg-blue-600 text-white' : 'text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700'"
        >预览</button>
      </div>
    </div>

    <!-- 编辑区 -->
    <div class="grid" :class="mode === 'split' ? 'grid-cols-2 divide-x divide-gray-100 dark:divide-gray-700' : 'grid-cols-1'">
      <!-- 输入区 -->
      <textarea
        v-if="mode !== 'preview'"
        ref="textareaRef"
        v-model="content"
        rows="6"
        placeholder="分享新鲜事... 支持 #标签 和 Markdown 语法"
        class="w-full resize-y bg-transparent focus:outline-none p-3 text-sm min-h-[140px]"
      ></textarea>

      <!-- 预览区 -->
      <div
        v-if="mode !== 'edit'"
        class="p-3 text-sm memo-content min-h-[140px]"
        v-html="previewHtml"
      ></div>
    </div>

    <!-- 已选图片预览 -->
    <div v-if="images.length > 0" class="flex flex-wrap gap-2 px-3 pb-2">
      <div v-for="(img, i) in images" :key="img" class="relative">
        <img :src="img" class="w-20 h-20 object-cover rounded-lg border border-gray-200 dark:border-gray-600" />
        <button
          @click="removeImage(i)"
          class="absolute -top-2 -right-2 w-5 h-5 rounded-full bg-red-500 text-white text-xs leading-none"
        >×</button>
      </div>
    </div>

    <div class="flex items-center justify-between px-3 py-2 border-t border-gray-100 dark:border-gray-700">
      <div class="flex items-center gap-2">
        <button
          @click="fileInput.click()"
          :disabled="uploading"
          class="px-3 py-1.5 text-sm rounded-lg bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50"
        >
          {{ uploading ? '上传中...' : '📷 图片' }}
        </button>
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          multiple
          class="hidden"
          @change="handleFileChange"
        />
        <button
          @click="polish"
          :disabled="polishing"
          class="px-3 py-1.5 text-sm rounded-lg bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300 hover:bg-purple-200 dark:hover:bg-purple-800 disabled:opacity-50"
        >
          {{ polishing ? '润色中...' : '✨ AI 润色' }}
        </button>
        <select
          v-model="visibility"
          class="px-2 py-1.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800"
        >
          <option value="public">公开</option>
          <option value="private">仅自己</option>
        </select>
      </div>
      <div class="flex items-center gap-2">
        <button
          v-if="isEditing"
          @click="cancelEdit"
          class="px-4 py-1.5 text-sm rounded-lg bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600"
        >取消</button>
        <button
          @click="publish"
          class="px-4 py-1.5 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700"
        >{{ isEditing ? '保存修改' : '发布' }}</button>
      </div>
    </div>
  </div>
</template>
