<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { fetchMemos, fetchHotTags, searchMemos, fetchCommentSettings, fetchMemo } from '../api'
import MemoComposer from '../components/MemoComposer.vue'
import MemoCard from '../components/MemoCard.vue'
import MemoDetailModal from '../components/MemoDetailModal.vue'
import TagSidebar from '../components/TagSidebar.vue'
import AuthModal from '../components/AuthModal.vue'
import SecuritySettingsModal from '../components/SecuritySettingsModal.vue'

const route = useRoute()

const props = defineProps({
  user: { type: Object, default: null },
  isLoggedIn: { type: Boolean, default: false }
})

const emit = defineEmits(['open-auth', 'logout'])

const memos = ref([])
const tags = ref([])
const loading = ref(false)
const showAuth = ref(false)
const showSecurity = ref(false)
const authMode = ref('login')
const keyword = ref('')
const activeTag = ref('')
const allowGuestComment = ref(false)
// 发布/编辑框组件引用（用于把待编辑博文回填进 Composer）
const composerRef = ref(null)

// 从用户中心"回复过的主题"跳转过来时，自动打开的博文详情
const detailMemo = ref(null)

async function openMemoDetail(id) {
  try {
    const memo = await fetchMemo(id)
    detailMemo.value = memo
  } catch (e) {
    alert('打开博文失败: ' + e.message)
  }
}

async function loadMemos() {
  loading.value = true
  try {
    const params = { page: 1, page_size: 50 }
    if (keyword.value) {
      params.q = keyword.value
    }
    if (activeTag.value) {
      params.tag = activeTag.value
    }
    const res = keyword.value || activeTag.value
      ? await searchMemos(params)
      : await fetchMemos(params)
    memos.value = res.items || []
  } catch (e) {
    alert('加载失败: ' + e.message)
  } finally {
    loading.value = false
  }
}

async function loadTags() {
  try {
    // 热门标签：后端返回 [{tagName, count}]，映射为 TagSidebar 需要的 {name, memo_count}
    const res = await fetchHotTags()
    tags.value = (res.items || []).map(t => ({
      name: t.tagName,
      memo_count: t.count
    }))
  } catch (e) {
    console.error('标签加载失败', e)
  }
}

function openAuth(mode) {
  authMode.value = mode
  showAuth.value = true
}

function onSearch() {
  loadMemos()
}

function onSelectTag(tag) {
  activeTag.value = activeTag.value === tag ? '' : tag
  loadMemos()
}

function onPublished() {
  loadMemos()
  loadTags()
}

// 编辑入口：把博文回填到顶部 Composer
function onEdit(memo) {
  composerRef.value?.startEdit(memo)
}

async function loadCommentSettings() {
  try {
    const res = await fetchCommentSettings()
    allowGuestComment.value = !!res.allow_guest_comment
  } catch (e) {
    console.error('评论设置加载失败', e)
  }
}

onMounted(() => {
  loadMemos()
  loadTags()
  loadCommentSettings()
  // 从用户中心"回复过的主题"跳转过来时，自动打开对应博文详情
  const openId = route.query.open
  if (openId) {
    openMemoDetail(openId)
  }
})
</script>

<template>
  <main class="max-w-5xl mx-auto px-4 py-6 grid grid-cols-1 md:grid-cols-[280px_1fr] gap-6">
    <!-- 左栏：用户信息 + 标签列表 -->
    <aside class="order-2 md:order-1">
      <TagSidebar
        :tags="tags"
        :active-tag="activeTag"
        :user="user"
        :is-logged-in="isLoggedIn"
        @select-tag="onSelectTag"
        @open-auth="openAuth"
      />
    </aside>

    <!-- 右栏：微博信息流 -->
    <section class="order-1 md:order-2 space-y-4">
      <!-- 搜索框 -->
      <div class="flex gap-2">
        <input
          v-model="keyword"
          @keyup.enter="onSearch"
          type="text"
          placeholder="搜索内容或 #标签..."
          class="flex-1 px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          @click="onSearch"
          class="px-4 py-2 rounded-lg bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600"
        >搜索</button>
      </div>

      <!-- 发布框（仅博主 admin 可发布） -->
      <MemoComposer
        v-if="isLoggedIn && user?.role === 'admin'"
        ref="composerRef"
        @published="onPublished"
      />

      <!-- 信息流 -->
      <div v-if="loading" class="text-center py-10 text-gray-400">加载中...</div>
      <div v-else-if="memos.length === 0" class="text-center py-10 text-gray-400">
        暂无内容
      </div>
      <div v-else class="space-y-4">
        <MemoCard
          v-for="memo in memos"
          :key="memo.id"
          :memo="memo"
          :current-user-id="user?.id"
          :user="user"
          :is-logged-in="isLoggedIn"
          :is-admin="user?.role === 'admin'"
          :allow-guest-comment="allowGuestComment"
          @deleted="loadMemos"
          @edit="onEdit"
        />
      </div>
    </section>

    <!-- 登录/注册弹窗 -->
    <AuthModal
      v-if="showAuth"
      :mode="authMode"
      @close="showAuth = false"
      @submit="emit('open-auth', $event)"
      @switch-mode="authMode = $event"
    />

    <!-- 修改密保弹窗 -->
    <SecuritySettingsModal
      v-if="showSecurity"
      @close="showSecurity = false"
    />

    <!-- 从"回复过的主题"跳转过来时自动打开的博文详情弹窗 -->
    <MemoDetailModal
      v-if="detailMemo"
      :memo="detailMemo"
      :user="user"
      :is-logged-in="isLoggedIn"
      :is-admin="user?.role === 'admin'"
      :allow-guest-comment="allowGuestComment"
      @close="detailMemo = null"
    />
  </main>
</template>
