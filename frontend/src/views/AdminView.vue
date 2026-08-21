<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  fetchSettings, updateSettings,
  fetchAdminUsers, updateUserRole, deleteUser,
  fetchAdminMemos, deleteAdminMemo, pinMemo,
  fetchAdminTags, deleteAdminTag
} from '../api'

const props = defineProps({
  user: { type: Object, default: null },
  isLoggedIn: { type: Boolean, default: false }
})

const router = useRouter()
const activeTab = ref('settings')

// ===== 博客设置 =====
const settings = ref({ blog_name: '', blog_desc: '', allow_register: true, allow_comment: true, allow_guest_comment: false, ai_api_url: '', ai_api_key: '', ai_model: '', deal_source_url: '', deal_cron_expr: '', ai_system_prompt: '' })
const savingSettings = ref(false)

async function loadSettings() {
  try {
    settings.value = await fetchSettings()
  } catch (e) {
    alert('读取设置失败: ' + e.message)
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    await updateSettings({
      blog_name: settings.value.blog_name,
      blog_desc: settings.value.blog_desc,
      allow_register: settings.value.allow_register,
      allow_comment: settings.value.allow_comment,
      allow_guest_comment: settings.value.allow_guest_comment,
      ai_api_url: settings.value.ai_api_url,
      ai_api_key: settings.value.ai_api_key,
      ai_model: settings.value.ai_model,
      deal_source_url: settings.value.deal_source_url,
      deal_cron_expr: settings.value.deal_cron_expr,
      ai_system_prompt: settings.value.ai_system_prompt
    })
    alert('设置已保存')
  } catch (e) {
    alert('保存失败: ' + e.message)
  } finally {
    savingSettings.value = false
  }
}

// ===== 用户管理 =====
const users = ref([])
const usersTotal = ref(0)
const usersPage = ref(1)
const usersPageSize = 20

async function loadUsers() {
  try {
    const res = await fetchAdminUsers({ page: usersPage.value, page_size: usersPageSize })
    users.value = res.items || []
    usersTotal.value = res.total || 0
  } catch (e) {
    alert('加载用户失败: ' + e.message)
  }
}

async function changeRole(u) {
  const newRole = u.role === 'admin' ? 'user' : 'admin'
  if (!confirm(`确认将用户 @${u.username} 的角色改为 ${newRole === 'admin' ? '管理员' : '普通用户'}？`)) return
  try {
    await updateUserRole(u.id, newRole)
    u.role = newRole
    alert('角色已更新')
  } catch (e) {
    alert('修改角色失败: ' + e.message)
  }
}

async function removeUser(u) {
  if (!confirm(`确认删除用户 @${u.username}？其所有博文将一并删除！`)) return
  try {
    await deleteUser(u.id)
    alert('用户已删除')
    loadUsers()
  } catch (e) {
    alert('删除失败: ' + e.message)
  }
}

// ===== 文章管理 =====
const memos = ref([])
const memosTotal = ref(0)
const memosPage = ref(1)
const memosPageSize = 20

async function loadMemos() {
  try {
    const res = await fetchAdminMemos({ page: memosPage.value, page_size: memosPageSize })
    memos.value = res.items || []
    memosTotal.value = res.total || 0
  } catch (e) {
    alert('加载文章失败: ' + e.message)
  }
}

async function removeMemo(m) {
  if (!confirm('确认删除该博文？')) return
  try {
    await deleteAdminMemo(m.id)
    alert('博文已删除')
    loadMemos()
  } catch (e) {
    alert('删除失败: ' + e.message)
  }
}

// ===== 置顶管理 =====
// 判断博文当前是否处于置顶状态（考虑截止时间：已过期视为未置顶）
function isPinned(m) {
  if (!m.pinned_at) return false
  if (m.pin_expire_at && new Date(m.pin_expire_at) <= new Date()) return false
  return true
}

// 置顶/取消置顶
async function togglePin(m) {
  if (isPinned(m)) {
    // 取消置顶
    if (!confirm('确认取消该博文的置顶？')) return
    try {
      await pinMemo(m.id, { pinned: false })
      alert('已取消置顶')
      loadMemos()
    } catch (e) {
      alert('取消置顶失败: ' + e.message)
    }
    return
  }
  // 置顶：读取截止时间输入（可选）
  const expireInput = m._pinExpire || ''
  let expireAt = ''
  if (expireInput) {
    const d = new Date(expireInput)
    if (isNaN(d.getTime())) {
      alert('截止时间格式无效，请使用 YYYY-MM-DDTHH:mm 格式，或留空表示永久置顶')
      return
    }
    expireAt = d.toISOString()
  }
  try {
    await pinMemo(m.id, { pinned: true, expire_at: expireAt })
    alert(expireAt ? '已置顶，到期自动取消' : '已永久置顶')
    loadMemos()
  } catch (e) {
    alert('置顶失败: ' + e.message)
  }
}

// ===== TAG 管理 =====
const tags = ref([])

async function loadTags() {
  try {
    const res = await fetchAdminTags()
    tags.value = res.items || []
  } catch (e) {
    alert('加载标签失败: ' + e.message)
  }
}

async function removeTag(t) {
  if (!confirm(`确认删除标签 #${t.name}？`)) return
  try {
    await deleteAdminTag(t.id)
    alert('标签已删除')
    loadTags()
  } catch (e) {
    alert('删除失败: ' + e.message)
  }
}

function switchTab(tab) {
  activeTab.value = tab
  if (tab === 'users') loadUsers()
  else if (tab === 'memos') loadMemos()
  else if (tab === 'tags') loadTags()
}

function goHome() {
  router.push('/')
}

onMounted(() => {
  loadSettings()
})
</script>

<template>
  <main class="max-w-5xl mx-auto px-4 py-6">
    <!-- 非 admin 提示 -->
    <div v-if="!isLoggedIn || user?.role !== 'admin'" class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-10 text-center shadow-sm">
      <div class="text-4xl mb-3">🔒</div>
      <h2 class="text-xl font-bold mb-2">无管理员权限</h2>
      <p class="text-gray-500 text-sm mb-4">此页面仅博主（管理员）可访问</p>
      <button @click="goHome" class="px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 text-sm">
        返回首页
      </button>
    </div>

    <!-- 管理面板 -->
    <div v-else class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-xl font-bold">后台管理</h2>
        <button @click="goHome" class="px-3 py-1.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700">
          返回首页
        </button>
      </div>

      <!-- Tab 导航 -->
      <div class="flex gap-2 border-b border-gray-200 dark:border-gray-700 pb-2 overflow-x-auto">
        <button
          v-for="tab in [{k:'settings',l:'博客设置'},{k:'memos',l:'文章管理'},{k:'tags',l:'TAG管理'},{k:'users',l:'用户管理'}]"
          :key="tab.k"
          @click="switchTab(tab.k)"
          class="px-4 py-2 rounded-lg text-sm whitespace-nowrap"
          :class="activeTab === tab.k
            ? 'bg-blue-600 text-white'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'"
        >{{ tab.l }}</button>
      </div>

      <!-- 博客设置 -->
      <div v-if="activeTab === 'settings'" class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">博客名称</label>
          <input v-model="settings.blog_name" type="text"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">博客描述</label>
          <textarea v-model="settings.blog_desc" rows="2"
            class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500"></textarea>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-sm">开放注册</span>
          <button @click="settings.allow_register = !settings.allow_register"
            class="w-11 h-6 rounded-full transition-colors"
            :class="settings.allow_register ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'">
            <span class="block w-5 h-5 bg-white rounded-full transition-transform"
              :class="settings.allow_register ? 'translate-x-5' : 'translate-x-0.5'"></span>
          </button>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-sm">开放评论</span>
          <button @click="settings.allow_comment = !settings.allow_comment"
            class="w-11 h-6 rounded-full transition-colors"
            :class="settings.allow_comment ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'">
            <span class="block w-5 h-5 bg-white rounded-full transition-transform"
              :class="settings.allow_comment ? 'translate-x-5' : 'translate-x-0.5'"></span>
          </button>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-sm">允许游客（免注册）评论</span>
          <button @click="settings.allow_guest_comment = !settings.allow_guest_comment"
            class="w-11 h-6 rounded-full transition-colors"
            :class="settings.allow_guest_comment ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'">
            <span class="block w-5 h-5 bg-white rounded-full transition-transform"
              :class="settings.allow_guest_comment ? 'translate-x-5' : 'translate-x-0.5'"></span>
          </button>
        </div>
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <div class="text-sm font-medium mb-3">AI 服务配置（OpenAI 兼容）</div>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-1">API 地址</label>
              <input v-model="settings.ai_api_url" type="text" placeholder="https://api.openai.com/v1"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">API Key</label>
              <input v-model="settings.ai_api_key" type="password" placeholder="sk-..."
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">模型名称</label>
              <input v-model="settings.ai_model" type="text" placeholder="gpt-4o-mini"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
          </div>
        </div>
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <div class="text-sm font-medium mb-3">自动任务（定时抓取 RSS → AI 洗稿 → 自动发布）</div>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-1">数据源 RSS 链接</label>
              <input v-model="settings.deal_source_url" type="text" placeholder="https://example.com/feed.xml"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">执行计划（cron 表达式）</label>
              <input v-model="settings.deal_cron_expr" type="text" placeholder="0 10,16 * * *"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500" />
              <p class="text-xs text-gray-400 mt-1">标准 5 段 cron，例如 <code>0 10,16 * * *</code> 表示每天 10 点和 16 点执行</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">AI 洗稿指令 (System Prompt)</label>
              <textarea v-model="settings.ai_system_prompt" rows="8" placeholder="留空则使用内置的『毒舌导购专家』兜底 Prompt"
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-xs leading-relaxed"></textarea>
              <p class="text-xs text-gray-400 mt-1">用于调教 AI 洗稿的人设与输出格式。留空时系统自动使用内置默认 Prompt，不会崩溃。</p>
            </div>
          </div>
        </div>
        <button @click="saveSettings" :disabled="savingSettings"
          class="px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
          {{ savingSettings ? '保存中...' : '保存设置' }}
        </button>
      </div>

      <!-- 文章管理 -->
      <div v-else-if="activeTab === 'memos'" class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <div class="flex items-center justify-between mb-4">
          <span class="text-sm text-gray-500">共 {{ memosTotal }} 篇博文</span>
          <button @click="loadMemos" class="px-3 py-1 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700">刷新</button>
        </div>
        <div v-if="memos.length === 0" class="text-center py-8 text-gray-400">暂无博文</div>
        <div v-else class="space-y-2">
          <div v-for="m in memos" :key="m.id"
            class="flex items-center justify-between gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-700">
            <div class="min-w-0 flex-1">
              <div class="text-sm truncate">
                <span v-if="isPinned(m)" class="mr-1 text-xs">📌</span>{{ m.content }}
              </div>
              <div class="text-xs text-gray-500 mt-1">
                @{{ m.user?.username || '未知' }} · {{ new Date(m.created_at).toLocaleString() }}
                <span class="ml-2 px-1.5 py-0.5 rounded text-xs"
                  :class="m.visibility === 'private' ? 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300' : 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'">
                  {{ m.visibility === 'private' ? '私密' : '公开' }}
                </span>
                <span v-if="isPinned(m)" class="ml-2 px-1.5 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300">
                  📌 置顶{{ m.pin_expire_at ? ' · ' + new Date(m.pin_expire_at).toLocaleString() : '（永久）' }}
                </span>
              </div>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <template v-if="!isPinned(m)">
                <input
                  v-model="m._pinExpire"
                  type="datetime-local"
                  class="text-xs px-2 py-1 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800"
                  title="置顶截止时间（留空=永久置顶）"
                />
                <button @click="togglePin(m)"
                  class="px-2 py-1 text-xs rounded-lg bg-blue-100 dark:bg-blue-900 text-blue-600 dark:text-blue-300 hover:bg-blue-200 dark:hover:bg-blue-800">置顶</button>
              </template>
              <button v-else @click="togglePin(m)"
                class="px-2 py-1 text-xs rounded-lg bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-500">取消置顶</button>
              <button @click="removeMemo(m)"
                class="px-2 py-1 text-xs rounded-lg bg-red-100 dark:bg-red-900 text-red-600 dark:text-red-300 hover:bg-red-200 dark:hover:bg-red-800">删除</button>
            </div>
          </div>
        </div>
        <div v-if="memosTotal > memosPageSize" class="flex justify-center gap-2 mt-4">
          <button @click="memosPage--; loadMemos()" :disabled="memosPage <= 1"
            class="px-3 py-1 text-sm rounded-lg border border-gray-300 dark:border-gray-600 disabled:opacity-40">上一页</button>
          <span class="px-3 py-1 text-sm">{{ memosPage }}</span>
          <button @click="memosPage++; loadMemos()" :disabled="memosPage * memosPageSize >= memosTotal"
            class="px-3 py-1 text-sm rounded-lg border border-gray-300 dark:border-gray-600 disabled:opacity-40">下一页</button>
        </div>
      </div>

      <!-- TAG 管理 -->
      <div v-else-if="activeTab === 'tags'" class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <div class="flex items-center justify-between mb-4">
          <span class="text-sm text-gray-500">共 {{ tags.length }} 个标签</span>
          <button @click="loadTags" class="px-3 py-1 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700">刷新</button>
        </div>
        <div v-if="tags.length === 0" class="text-center py-8 text-gray-400">暂无标签</div>
        <div v-else class="flex flex-wrap gap-2">
          <div v-for="t in tags" :key="t.id"
            class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-gray-100 dark:bg-gray-700">
            <span class="text-sm">#{{ t.name }}</span>
            <span class="text-xs text-gray-500">{{ t.memo_count }}</span>
            <button @click="removeTag(t)" class="text-red-500 hover:text-red-700 text-xs">✕</button>
          </div>
        </div>
      </div>

      <!-- 用户管理 -->
      <div v-else-if="activeTab === 'users'" class="bg-white dark:bg-gray-800 rounded-2xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        <div class="flex items-center justify-between mb-4">
          <span class="text-sm text-gray-500">共 {{ usersTotal }} 个用户</span>
          <button @click="loadUsers" class="px-3 py-1 text-sm rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700">刷新</button>
        </div>
        <div v-if="users.length === 0" class="text-center py-8 text-gray-400">暂无用户</div>
        <div v-else class="space-y-2">
          <div v-for="u in users" :key="u.id"
            class="flex items-center justify-between gap-3 p-3 rounded-lg bg-gray-50 dark:bg-gray-700">
            <div class="min-w-0 flex-1">
              <div class="text-sm font-medium">
                {{ u.nickname || u.username }}
                <span class="ml-2 px-1.5 py-0.5 rounded text-xs"
                  :class="u.role === 'admin' ? 'bg-purple-100 dark:bg-purple-900 text-purple-700 dark:text-purple-300' : 'bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300'">
                  {{ u.role === 'admin' ? '管理员' : '用户' }}
                </span>
              </div>
              <div class="text-xs text-gray-500 mt-1">@{{ u.username }} · {{ new Date(u.created_at).toLocaleDateString() }}</div>
            </div>
            <div class="flex gap-2 shrink-0">
              <button @click="changeRole(u)"
                class="px-2 py-1 text-xs rounded-lg border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-600">
                {{ u.role === 'admin' ? '降为用户' : '设为管理员' }}
              </button>
              <button @click="removeUser(u)"
                class="px-2 py-1 text-xs rounded-lg bg-red-100 dark:bg-red-900 text-red-600 dark:text-red-300 hover:bg-red-200 dark:hover:bg-red-800">删除</button>
            </div>
          </div>
        </div>
        <div v-if="usersTotal > usersPageSize" class="flex justify-center gap-2 mt-4">
          <button @click="usersPage--; loadUsers()" :disabled="usersPage <= 1"
            class="px-3 py-1 text-sm rounded-lg border border-gray-300 dark:border-gray-600 disabled:opacity-40">上一页</button>
          <span class="px-3 py-1 text-sm">{{ usersPage }}</span>
          <button @click="usersPage++; loadUsers()" :disabled="usersPage * usersPageSize >= usersTotal"
            class="px-3 py-1 text-sm rounded-lg border border-gray-300 dark:border-gray-600 disabled:opacity-40">下一页</button>
        </div>
      </div>
    </div>
  </main>
</template>
