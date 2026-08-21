<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { deleteMemo } from '../api'
import MemoDetailModal from './MemoDetailModal.vue'
import RoleBadge from './RoleBadge.vue'

const router = useRouter()

const props = defineProps({
  memo: { type: Object, required: true },
  currentUserId: { type: Number, default: null },
  user: { type: Object, default: null },
  isLoggedIn: { type: Boolean, default: false },
  isAdmin: { type: Boolean, default: false },
  allowGuestComment: { type: Boolean, default: false }
})
const emit = defineEmits(['deleted', 'edit'])

// 渲染 Markdown 并通过 DOMPurify 过滤 XSS
const renderedContent = computed(() => {
  if (!props.memo.content) return ''
  const rawHtml = marked.parse(props.memo.content)
  return DOMPurify.sanitize(rawHtml)
})

const isOwner = computed(() => props.currentUserId && props.memo.user_id === props.currentUserId)

// 置顶状态：pinned_at 非空且 pin_expire_at 未过期（到期自动取消置顶）
const isPinned = computed(() => {
  if (!props.memo.pinned_at) return false
  if (props.memo.pin_expire_at && new Date(props.memo.pin_expire_at) <= new Date()) return false
  return true
})

// 长文截断：超过阈值显示"展开全文"
const CONTENT_CHAR_LIMIT = 300
const isLong = computed(() => (props.memo.content || '').length > CONTENT_CHAR_LIMIT)
const expanded = ref(false)
const showFull = computed(() => !isLong.value || expanded.value)

// 详情弹窗
const showDetail = ref(false)
// 图片放大（列表内直接查看大图）
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
  const now = new Date()
  const diff = now - d
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + ' 分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + ' 小时前'
  return d.toLocaleDateString('zh-CN')
}

async function remove() {
  if (!confirm('确定删除该博文？')) return
  try {
    await deleteMemo(props.memo.id)
    emit('deleted')
  } catch (e) {
    alert('删除失败: ' + e.message)
  }
}

// 编辑入口：把当前博文回填到顶部 Composer
function edit() {
  emit('edit', props.memo)
}

// 跳转到独立详情页（可分享的 URL）
function goDetail() {
  router.push(`/m/${props.memo.id}`)
}

// 复制独立分享链接
async function copyShareLink() {
  const url = `${window.location.origin}/m/${props.memo.id}`
  try {
    await navigator.clipboard.writeText(url)
    alert('已复制分享链接: ' + url)
  } catch (e) {
    alert('复制失败，请手动复制: ' + url)
  }
}
</script>

<template>
  <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 shadow-sm">
    <!-- 头部：作者 + 时间 -->
    <div class="flex items-center justify-between mb-3">
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
          <button
            @click="goDetail"
            class="text-xs text-gray-400 hover:text-blue-500"
            title="查看独立详情页"
          >{{ formatTime(memo.created_at) }}</button>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <span
          v-if="isPinned"
          class="text-xs px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300"
        >📌 置顶</span>
        <span
          v-if="memo.visibility === 'private'"
          class="text-xs px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900 text-amber-700 dark:text-amber-300"
        >🔒 仅自己</span>
        <button
          @click="copyShareLink"
          class="text-xs text-gray-400 hover:text-blue-500"
          title="复制分享链接"
        >🔗</button>
        <button
          v-if="isAdmin"
          @click="edit"
          class="text-xs text-gray-400 hover:text-blue-500"
        >编辑</button>
        <button
          v-if="isAdmin"
          @click="remove"
          class="text-xs text-gray-400 hover:text-red-500"
        >删除</button>
      </div>
    </div>

    <!-- 内容（Markdown 渲染 + XSS 过滤；长文截断） -->
    <div
      v-if="memo.content"
      class="memo-content text-sm"
      :class="{ 'line-clamp-6': isLong && !expanded }"
      v-html="renderedContent"
    ></div>
    <button
      v-if="isLong"
      @click="expanded = !expanded"
      class="mt-1 text-xs text-blue-500 hover:text-blue-600"
    >{{ expanded ? '收起' : '展开全文' }}</button>

    <!-- 图片：点击查看大图 -->
    <div v-if="memo.images && memo.images.length > 0" class="grid gap-2 mt-3"
      :class="memo.images.length === 1 ? 'grid-cols-1' : 'grid-cols-2'">
      <img
        v-for="(img, i) in memo.images"
        :key="i"
        :src="img"
        class="w-full rounded-lg object-cover max-h-80 cursor-zoom-in"
        loading="lazy"
        @click="openLightbox(i)"
      />
    </div>

    <!-- 标签 -->
    <div v-if="memo.tags && memo.tags.length > 0" class="flex flex-wrap gap-2 mt-3">
      <span
        v-for="tag in memo.tags"
        :key="tag.id"
        class="text-xs px-2 py-0.5 rounded-full bg-blue-50 dark:bg-blue-900/50 text-blue-600 dark:text-blue-300"
      >#{{ tag.name }}</span>
    </div>

    <!-- 查看完整详情入口 -->
    <button
      @click="showDetail = true"
      class="mt-3 w-full py-2 text-xs text-gray-400 hover:text-blue-500 border-t border-gray-100 dark:border-gray-700"
    >查看完整内容</button>
  </div>

  <!-- 详情弹窗 -->
  <MemoDetailModal
    v-if="showDetail"
    :memo="memo"
    :user="user"
    :is-logged-in="isLoggedIn"
    :is-admin="isAdmin"
    :allow-guest-comment="allowGuestComment"
    @close="showDetail = false"
  />

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
</template>
