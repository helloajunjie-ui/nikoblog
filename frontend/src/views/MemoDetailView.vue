<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { fetchMemo, fetchComments, createComment, deleteComment, uploadImage, fetchCommentSettings } from '../api'
import RoleBadge from '../components/RoleBadge.vue'

const route = useRoute()
const router = useRouter()

const props = defineProps({
  user: { type: Object, default: null },
  isLoggedIn: { type: Boolean, default: false },
  isAdmin: { type: Boolean, default: false }
})

// ===== 博文加载 =====
const memo = ref(null)
const loading = ref(true)
const loadError = ref('')
const allowGuestComment = ref(false)

// 去掉 Markdown 标记，提取纯文本（用于 document.title）
function stripMarkdown(text) {
  if (!text) return ''
  return text
    .replace(/```[\s\S]*?```/g, ' ')   // 代码块
    .replace(/`[^`]*`/g, ' ')          // 行内代码
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ') // 图片
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1') // 链接
    .replace(/^#{1,6}\s+/gm, '')       // 标题
    .replace(/[*_~>#|]/g, ' ')         // 强调/引用等符号
    .replace(/\s+/g, ' ')
    .trim()
}

async function loadMemo(id) {
  loading.value = true
  loadError.value = ''
  memo.value = null
  try {
    const m = await fetchMemo(id)
    memo.value = m
    // 动态设置 document.title：博文正文前 15 字符（去 Markdown 标记）
    const title = stripMarkdown(m.content).slice(0, 15)
    document.title = title ? `${title} - nikoblog` : 'nikoblog'
  } catch (e) {
    loadError.value = e.message || '博文不存在或无权访问'
  } finally {
    loading.value = false
  }
}

// 路由参数变化时重新加载（如从 /m/1 跳到 /m/2）
watch(() => route.params.id, (id) => {
  if (id) loadMemo(id)
})

async function loadCommentSettings() {
  try {
    const res = await fetchCommentSettings()
    allowGuestComment.value = !!res.allow_guest_comment
  } catch (e) {
    console.error('评论设置加载失败', e)
  }
}

onMounted(() => {
  loadMemo(route.params.id)
  loadCommentSettings()
})

// ===== 内容渲染（Markdown + XSS 过滤） =====
const renderedContent = computed(() => {
  if (!memo.value?.content) return ''
  const rawHtml = marked.parse(memo.value.content)
  return DOMPurify.sanitize(rawHtml)
})

// ===== 图片放大（lightbox） =====
const lightboxIndex = ref(-1)
const lightboxOpen = computed(() => lightboxIndex.value >= 0)
const currentImage = computed(() =>
  lightboxOpen.value ? memo.value.images[lightboxIndex.value] : ''
)

function openLightbox(i) {
  lightboxIndex.value = i
}
function closeLightbox() {
  lightboxIndex.value = -1
}
function prevImage() {
  if (memo.value.images.length === 0) return
  lightboxIndex.value = (lightboxIndex.value - 1 + memo.value.images.length) % memo.value.images.length
}
function nextImage() {
  if (memo.value.images.length === 0) return
  lightboxIndex.value = (lightboxIndex.value + 1) % memo.value.images.length
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
const commentImages = ref([])
const commentUploading = ref(false)
const commentFileInput = ref(null)

async function loadComments() {
  if (!memo.value) return
  commentsLoading.value = true
  try {
    comments.value = await fetchComments(memo.value.id)
  } catch (e) {
    console.error('评论加载失败', e)
  } finally {
    commentsLoading.value = false
  }
}

// 博文加载完成后加载评论
watch(memo, (m) => {
  if (m) loadComments()
})

const canComment = computed(() => props.isLoggedIn || allowGuestComment.value)

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
      data.images = commentImages.value
    }
    const created = await createComment(memo.value.id, data)
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
</script>

<template>
  <main class="max-w-2xl mx-auto px-4 py-6">
    <!-- 返回首页 -->
    <button
      @click="router.push('/')"
      class="mb-4 text-sm text-gray-400 hover:text-blue-500"
    >← 返回首页</button>

    <!-- 加载中 -->
    <div v-if="loading" class="text-center text-gray-400 py-16">加载中...</div>

    <!-- 加载失败 -->
    <div v-else-if="loadError" class="text-center py-16">
      <p class="text-gray-500 mb-4">{{ loadError }}</p>
      <button
        @click="router.push('/')"
        class="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-sm"
      >返回首页</button>
    </div>

    <!-- 博文详情 -->
    <div v-else-if="memo" class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-5 shadow-sm">
      <!-- 头部：作者 + 时间 -->
      <div class="flex items-center justify-between mb-4">
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
        <div class="flex items-center gap-2">
          <span
            v-if="memo.pinned_at && (!memo.pin_expire_at || new Date(memo.pin_expire_at) > new Date())"
            class="text-xs px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300"
          >📌 置顶</span>
          <span
            v-if="memo.visibility === 'private'"
            class="text-xs px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900 text-amber-700 dark:text-amber-300"
          >🔒 仅自己</span>
        </div>
      </div>

      <!-- 正文（完整展示） -->
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
  </main>
</template>
