<script setup>
import { ref, computed, onMounted } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { fetchComments, createComment, deleteComment, uploadImage } from '../api'
import RoleBadge from './RoleBadge.vue'

const props = defineProps({
  memo: { type: Object, required: true },
  user: { type: Object, default: null },
  isLoggedIn: { type: Boolean, default: false },
  isAdmin: { type: Boolean, default: false },
  allowGuestComment: { type: Boolean, default: false }
})
const emit = defineEmits(['close'])

// 完整内容渲染（Markdown + XSS 过滤）
const renderedContent = computed(() => {
  if (!props.memo.content) return ''
  const rawHtml = marked.parse(props.memo.content)
  return DOMPurify.sanitize(rawHtml)
})

// 图片放大（lightbox）
const lightboxIndex = ref(-1)
const lightboxOpen = computed(() => lightboxIndex.value >= 0)
const currentImage = computed(() =>
  lightboxOpen.value ? props.memo.images[lightboxIndex.value] : ''
)

function openLightbox(i) {
  lightboxIndex.value = i
}
function closeLightbox() {
  lightboxIndex.value = -1
}
function prevImage() {
  if (props.memo.images.length === 0) return
  lightboxIndex.value = (lightboxIndex.value - 1 + props.memo.images.length) % props.memo.images.length
}
function nextImage() {
  if (props.memo.images.length === 0) return
  lightboxIndex.value = (lightboxIndex.value + 1) % props.memo.images.length
}

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit'
  })
}

// ===== 评论 =====
const comments = ref([])
const commentsLoading = ref(false)
const commentContent = ref('')
const guestName = ref('')
const submitting = ref(false)
// 评论附图（仅登录用户可上传，最多 5 张）
const commentImages = ref([])
const commentUploading = ref(false)
const commentFileInput = ref(null)

async function loadComments() {
  commentsLoading.value = true
  try {
    comments.value = await fetchComments(props.memo.id)
  } catch (e) {
    console.error('评论加载失败', e)
  } finally {
    commentsLoading.value = false
  }
}

// 是否可发表评论：登录用户始终可；游客需开启免注册评论
const canComment = computed(() => props.isLoggedIn || props.allowGuestComment)

// 评论图片上传（仅登录用户可上传）
async function handleCommentFileChange(e) {
  const files = Array.from(e.target.files || [])
  if (files.length === 0) return
  commentUploading.value = true
  try {
    for (const file of files) {
      if (commentImages.value.length >= 5) {
        alert('评论图片最多 5 张')
        break
      }
      if (file.size > 5 * 1024 * 1024) {
        alert('图片不能超过 5MB: ' + file.name)
        continue
      }
      const res = await uploadImage(file)
      commentImages.value.push(res.url)
    }
  } catch (err) {
    alert('上传失败: ' + err.message)
  } finally {
    commentUploading.value = false
    if (commentFileInput.value) commentFileInput.value.value = ''
  }
}

function removeCommentImage(index) {
  commentImages.value.splice(index, 1)
}

async function submitComment() {
  const content = commentContent.value.trim()
  if (!content) {
    alert('请输入评论内容')
    return
  }
  // 游客需填昵称
  if (!props.isLoggedIn && !guestName.value.trim()) {
    alert('请填写昵称')
    return
  }
  submitting.value = true
  try {
    const data = { content }
    if (!props.isLoggedIn) {
      data.guest_name = guestName.value.trim()
    } else if (commentImages.value.length > 0) {
      // 仅登录用户可附图
      data.images = commentImages.value
    }
    const created = await createComment(props.memo.id, data)
    comments.value.push(created)
    commentContent.value = ''
    guestName.value = ''
    commentImages.value = []
  } catch (e) {
    alert('评论失败: ' + e.message)
  } finally {
    submitting.value = false
  }
}

// 删除评论：作者本人或 admin
function canDeleteComment(comment) {
  if (props.isAdmin) return true
  return props.isLoggedIn && comment.user_id != null && comment.user_id === props.user?.id
}

async function removeComment(comment) {
  if (!confirm('确定删除这条评论？')) return
  try {
    await deleteComment(comment.id)
    comments.value = comments.value.filter(c => c.id !== comment.id)
  } catch (e) {
    alert('删除失败: ' + e.message)
  }
}

function commentAuthorName(comment) {
  if (comment.user) return comment.user.nickname || comment.user.username
  return comment.guest_name || '游客'
}

// 评论图片放大（lightbox）
const commentLightboxIndex = ref(-1)
const commentLightboxList = ref([])
const commentLightboxOpen = computed(() => commentLightboxIndex.value >= 0)
const currentCommentImage = computed(() =>
  commentLightboxOpen.value ? commentLightboxList.value[commentLightboxIndex.value] : ''
)

function openCommentLightbox(images, i) {
  commentLightboxList.value = images
  commentLightboxIndex.value = i
}
function closeCommentLightbox() {
  commentLightboxIndex.value = -1
  commentLightboxList.value = []
}
function prevCommentImage() {
  if (commentLightboxList.value.length === 0) return
  commentLightboxIndex.value = (commentLightboxIndex.value - 1 + commentLightboxList.value.length) % commentLightboxList.value.length
}
function nextCommentImage() {
  if (commentLightboxList.value.length === 0) return
  commentLightboxIndex.value = (commentLightboxIndex.value + 1) % commentLightboxList.value.length
}

onMounted(loadComments)
</script>

<template>
  <!-- 详情弹窗 -->
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" @click.self="emit('close')">
    <div class="bg-white dark:bg-gray-800 rounded-2xl w-full max-w-2xl shadow-xl max-h-[90vh] flex flex-col overflow-hidden">
      <!-- 头部 -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center text-blue-600 dark:text-blue-300 font-bold overflow-hidden">
            <img
              v-if="memo.user?.avatar"
              :src="memo.user.avatar"
              class="w-full h-full object-cover"
              alt="头像"
            />
            <template v-else>{{ (memo.user?.nickname || memo.user?.username || '?').charAt(0) }}</template>
          </div>
          <div>
            <div class="font-medium text-sm flex items-center gap-1.5">
              {{ memo.user?.nickname || memo.user?.username }}
              <RoleBadge :user="memo.user" />
            </div>
            <div class="text-xs text-gray-400">{{ formatTime(memo.created_at) }}</div>
          </div>
        </div>
        <button @click="emit('close')" class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 text-xl leading-none px-2">×</button>
      </div>

      <!-- 正文（完整展示，可滚动） -->
      <div class="flex-1 overflow-y-auto px-5 py-4">
        <div v-if="memo.content" class="memo-content text-sm leading-relaxed" v-html="renderedContent"></div>

        <!-- 图片：完整展示，点击放大 -->
        <div v-if="memo.images && memo.images.length > 0" class="grid gap-3 mt-4"
          :class="memo.images.length === 1 ? 'grid-cols-1' : 'grid-cols-2'">
          <img
            v-for="(img, i) in memo.images"
            :key="i"
            :src="img"
            class="w-full rounded-lg cursor-zoom-in border border-gray-200 dark:border-gray-700"
            loading="lazy"
            @click="openLightbox(i)"
          />
        </div>

        <!-- 标签 -->
        <div v-if="memo.tags && memo.tags.length > 0" class="flex flex-wrap gap-2 mt-4">
          <span
            v-for="tag in memo.tags"
            :key="tag.id"
            class="text-xs px-2 py-0.5 rounded-full bg-blue-50 dark:bg-blue-900/50 text-blue-600 dark:text-blue-300"
          >#{{ tag.name }}</span>
        </div>

        <!-- 评论区域 -->
        <div class="mt-6 border-t border-gray-200 dark:border-gray-700 pt-4">
          <div class="text-sm font-medium mb-3">评论 ({{ comments.length }})</div>

          <!-- 评论列表 -->
          <div v-if="commentsLoading" class="text-xs text-gray-400 py-2">评论加载中...</div>
          <div v-else-if="comments.length === 0" class="text-xs text-gray-400 py-2">暂无评论</div>
          <div v-else class="space-y-3">
            <div v-for="comment in comments" :key="comment.id" class="flex items-start gap-2">
              <div class="w-7 h-7 rounded-full bg-gray-100 dark:bg-gray-700 flex items-center justify-center text-gray-500 dark:text-gray-300 text-xs font-bold shrink-0 overflow-hidden">
                <img
                  v-if="comment.user?.avatar"
                  :src="comment.user.avatar"
                  class="w-full h-full object-cover"
                  alt="头像"
                />
                <template v-else>{{ commentAuthorName(comment).charAt(0) }}</template>
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="text-xs font-medium flex items-center gap-1">
                    {{ commentAuthorName(comment) }}
                    <RoleBadge :user="comment.user" :is-guest="!comment.user" />
                  </span>
                  <span class="text-[10px] text-gray-400">{{ formatTime(comment.created_at) }}</span>
                  <button
                    v-if="canDeleteComment(comment)"
                    @click="removeComment(comment)"
                    class="text-[10px] text-gray-400 hover:text-red-500 ml-auto"
                  >删除</button>
                </div>
                <p class="text-sm text-gray-700 dark:text-gray-300 break-words">{{ comment.content }}</p>
                <!-- 评论附图：缩略图，点击放大 -->
                <div v-if="comment.images && comment.images.length > 0" class="flex flex-wrap gap-2 mt-2">
                  <img
                    v-for="(img, i) in comment.images"
                    :key="i"
                    :src="img"
                    class="w-20 h-20 object-cover rounded-lg cursor-zoom-in border border-gray-200 dark:border-gray-700"
                    loading="lazy"
                    @click="openCommentLightbox(comment.images, i)"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- 评论输入框 -->
          <div v-if="canComment" class="mt-4">
            <input
              v-if="!isLoggedIn"
              v-model="guestName"
              type="text"
              placeholder="你的昵称"
              class="w-full mb-2 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <!-- 评论附图预览（仅登录用户） -->
            <div v-if="isLoggedIn && commentImages.length > 0" class="flex flex-wrap gap-2 mb-2">
              <div v-for="(img, i) in commentImages" :key="i" class="relative">
                <img :src="img" class="w-16 h-16 object-cover rounded-lg border border-gray-200 dark:border-gray-700" />
                <button
                  @click="removeCommentImage(i)"
                  class="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-red-500 text-white text-xs leading-none flex items-center justify-center"
                >×</button>
              </div>
            </div>
            <div class="flex gap-2">
              <input
                v-model="commentContent"
                @keyup.enter="submitComment"
                type="text"
                placeholder="写下你的评论..."
                class="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <!-- 图片上传按钮（仅登录用户可附图） -->
              <button
                v-if="isLoggedIn"
                @click="commentFileInput && commentFileInput.click()"
                :disabled="commentUploading || commentImages.length >= 5"
                class="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-300 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50"
                title="上传图片（最多5张）"
              >{{ commentUploading ? '上传中...' : '🖼️' }}</button>
              <input
                ref="commentFileInput"
                type="file"
                accept="image/*"
                multiple
                class="hidden"
                @change="handleCommentFileChange"
              />
              <button
                @click="submitComment"
                :disabled="submitting"
                class="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-sm disabled:opacity-50"
              >{{ submitting ? '发送中...' : '评论' }}</button>
            </div>
          </div>
          <div v-else class="mt-4 text-xs text-gray-400">
            博主未开放游客评论，请先登录后再评论。
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- 图片放大 Lightbox -->
  <div v-if="lightboxOpen" class="fixed inset-0 z-[60] bg-black/90 flex items-center justify-center" @click="closeLightbox">
    <button class="absolute top-4 right-4 text-white text-3xl px-3 hover:text-gray-300" @click.stop="closeLightbox">×</button>
    <button
      v-if="memo.images.length > 1"
      class="absolute left-4 text-white text-4xl px-3 hover:text-gray-300"
      @click.stop="prevImage"
    >‹</button>
    <img :src="currentImage" class="max-w-[90vw] max-h-[90vh] object-contain" @click.stop />
    <button
      v-if="memo.images.length > 1"
      class="absolute right-4 text-white text-4xl px-3 hover:text-gray-300"
      @click.stop="nextImage"
    >›</button>
    <div v-if="memo.images.length > 1" class="absolute bottom-4 text-white text-sm">
      {{ lightboxIndex + 1 }} / {{ memo.images.length }}
    </div>
  </div>

  <!-- 评论图片放大 Lightbox -->
  <div v-if="commentLightboxOpen" class="fixed inset-0 z-[60] bg-black/90 flex items-center justify-center" @click="closeCommentLightbox">
    <button class="absolute top-4 right-4 text-white text-3xl px-3 hover:text-gray-300" @click.stop="closeCommentLightbox">×</button>
    <button
      v-if="commentLightboxList.length > 1"
      class="absolute left-4 text-white text-4xl px-3 hover:text-gray-300"
      @click.stop="prevCommentImage"
    >‹</button>
    <img :src="currentCommentImage" class="max-w-[90vw] max-h-[90vh] object-contain" @click.stop />
    <button
      v-if="commentLightboxList.length > 1"
      class="absolute right-4 text-white text-4xl px-3 hover:text-gray-300"
      @click.stop="nextCommentImage"
    >›</button>
    <div v-if="commentLightboxList.length > 1" class="absolute bottom-4 text-white text-sm">
      {{ commentLightboxIndex + 1 }} / {{ commentLightboxList.length }}
    </div>
  </div>
</template>
