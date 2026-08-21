<script setup>
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import SecuritySettingsModal from '../components/SecuritySettingsModal.vue'
import { fetchMyCommentedMemos, uploadAvatar } from '../api'

const props = defineProps({
  user: { type: Object, default: null },
  isLoggedIn: { type: Boolean, default: false }
})

const emit = defineEmits(['avatar-updated'])

const router = useRouter()
const showSecurity = ref(false)

// 头像上传
const avatarUploading = ref(false)
const avatarInput = ref(null)

function triggerAvatarSelect() {
  avatarInput.value?.click()
}

async function handleAvatarChange(e) {
  const file = e.target.files?.[0]
  e.target.value = '' // 允许重复选择同一文件
  if (!file) return

  // 前端校验大小（最大 2MB）
  if (file.size > 2 * 1024 * 1024) {
    alert('头像文件过大，最大允许 2MB')
    return
  }

  avatarUploading.value = true
  try {
    const res = await uploadAvatar(file)
    // 通知 App.vue 更新当前用户头像（同步 localStorage）
    emit('avatar-updated', res.url)
  } catch (err) {
    alert('头像上传失败: ' + (err.response?.data?.error || err.message))
  } finally {
    avatarUploading.value = false
  }
}

// 我回复过的主题
const commentedMemos = ref([])
const commentedLoading = ref(false)

async function loadCommentedMemos() {
  if (!props.isLoggedIn) return
  commentedLoading.value = true
  try {
    const res = await fetchMyCommentedMemos()
    commentedMemos.value = res.items || []
  } catch (e) {
    console.error('加载回复过的主题失败', e)
  } finally {
    commentedLoading.value = false
  }
}

// 登录状态变化时加载
watch(() => props.isLoggedIn, (v) => {
  if (v) loadCommentedMemos()
})

// 点击主题：跳转首页并自动打开该博文详情弹窗
function openMemo(memoId) {
  router.push({ path: '/', query: { open: memoId } })
}

function goHome() {
  router.push('/')
}
</script>

<template>
  <main class="max-w-2xl mx-auto px-4 py-6">
    <div class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
      <h2 class="text-xl font-bold mb-4">个人中心</h2>

      <!-- 未登录提示 -->
      <div v-if="!isLoggedIn" class="text-center py-10 text-gray-400">
        请先登录
        <button @click="goHome" class="block mx-auto mt-4 px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700">
          返回首页登录
        </button>
      </div>

      <!-- 已登录：个人资料 -->
      <div v-else class="space-y-4">
        <div class="flex items-center gap-4">
          <div class="relative shrink-0">
            <div class="w-16 h-16 rounded-full bg-blue-600 text-white flex items-center justify-center text-2xl font-bold overflow-hidden">
              <img
                v-if="user?.avatar"
                :src="user.avatar"
                class="w-full h-full object-cover"
                alt="头像"
              />
              <template v-else>{{ (user?.nickname || user?.username || '?').charAt(0).toUpperCase() }}</template>
            </div>
            <button
              @click="triggerAvatarSelect"
              :disabled="avatarUploading"
              class="absolute -bottom-1 -right-1 w-7 h-7 rounded-full bg-blue-600 text-white text-sm flex items-center justify-center shadow hover:bg-blue-700 disabled:opacity-50"
              title="更换头像"
            >{{ avatarUploading ? '…' : '✎' }}</button>
            <input
              ref="avatarInput"
              type="file"
              accept="image/jpeg,image/png,image/gif,image/webp"
              class="hidden"
              @change="handleAvatarChange"
            />
          </div>
          <div>
            <div class="text-lg font-semibold">{{ user?.nickname || user?.username }}</div>
            <div class="text-sm text-gray-500">@{{ user?.username }}</div>
            <div class="text-xs text-gray-400 mt-1">头像最大 2MB</div>
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
          <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
            <div class="text-gray-500 text-xs">昵称</div>
            <div>{{ user?.nickname || '未设置' }}</div>
          </div>
          <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
            <div class="text-gray-500 text-xs">邮箱</div>
            <div>{{ user?.email || '未设置' }}</div>
          </div>
          <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
            <div class="text-gray-500 text-xs">角色</div>
            <div>{{ user?.role === 'admin' ? '管理员' : '普通用户' }}</div>
          </div>
          <div class="bg-gray-50 dark:bg-gray-700 rounded-lg p-3">
            <div class="text-gray-500 text-xs">注册时间</div>
            <div>{{ user?.created_at ? new Date(user.created_at).toLocaleDateString() : '-' }}</div>
          </div>
        </div>

        <!-- 密保设置（个人页专属） -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h3 class="font-semibold mb-2">账号安全</h3>
          <p class="text-xs text-gray-500 mb-3">设置密保问题，用于忘记密码时找回账号</p>
          <button
            @click="showSecurity = true"
            class="px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 text-sm"
          >修改密保问题</button>
        </div>

        <!-- 我回复过的主题 -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h3 class="font-semibold mb-2">我回复过的主题</h3>
          <p class="text-xs text-gray-500 mb-3">点击可快速跳转到对应博文</p>

          <div v-if="commentedLoading" class="text-sm text-gray-400 py-3">加载中...</div>
          <div v-else-if="commentedMemos.length === 0" class="text-sm text-gray-400 py-3">
            暂无回复记录
          </div>
          <div v-else class="space-y-2">
            <button
              v-for="memo in commentedMemos"
              :key="memo.id"
              @click="openMemo(memo.id)"
              class="w-full text-left px-3 py-2 rounded-lg border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-sm text-gray-800 dark:text-gray-200 line-clamp-1 flex-1">
                  {{ memo.content || '(无内容)' }}
                </span>
                <span class="text-xs text-gray-400 shrink-0">#{{ memo.id }}</span>
              </div>
              <div class="text-xs text-gray-400 mt-1">
                {{ memo.user?.nickname || memo.user?.username }} · {{ new Date(memo.created_at).toLocaleDateString() }}
              </div>
            </button>
          </div>
        </div>

        <div class="flex gap-2 pt-2">
          <button @click="goHome" class="px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 text-sm">
            返回首页
          </button>
        </div>
      </div>
    </div>

    <!-- 修改密保弹窗 -->
    <SecuritySettingsModal
      v-if="showSecurity"
      @close="showSecurity = false"
    />
  </main>
</template>
