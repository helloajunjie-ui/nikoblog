<script setup>
defineProps({
  tags: { type: Array, default: () => [] },
  activeTag: { type: String, default: '' },
  user: { type: Object, default: null },
  isLoggedIn: { type: Boolean, default: false }
})
const emit = defineEmits(['select-tag', 'open-auth'])
</script>

<template>
  <div class="space-y-4">
    <!-- 用户信息卡片 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 shadow-sm">
      <template v-if="isLoggedIn && user">
        <div class="flex items-center gap-3">
          <div class="w-14 h-14 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center text-blue-600 dark:text-blue-300 text-xl font-bold overflow-hidden">
            <img
              v-if="user.avatar"
              :src="user.avatar"
              class="w-full h-full object-cover"
              alt="头像"
            />
            <template v-else>{{ (user.nickname || user.username || '?').charAt(0) }}</template>
          </div>
          <div>
            <div class="font-bold">{{ user.nickname || user.username }}</div>
            <div class="text-xs text-gray-400">@{{ user.username }}</div>
          </div>
        </div>
      </template>
      <template v-else>
        <div class="text-center py-2">
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">登录后即可发布博文</p>
          <button
            @click="emit('open-auth', 'login')"
            class="px-4 py-2 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700"
          >立即登录</button>
        </div>
      </template>
    </div>

    <!-- 标签列表 -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 shadow-sm">
      <h3 class="text-sm font-bold mb-3 text-gray-500 dark:text-gray-400">热门标签</h3>
      <div v-if="tags.length === 0" class="text-xs text-gray-400">暂无标签</div>
      <div v-else class="flex flex-wrap gap-2">
        <button
          v-for="tag in tags"
          :key="tag.id"
          @click="emit('select-tag', tag.name)"
          class="text-sm px-3 py-1 rounded-full border transition-colors"
          :class="activeTag === tag.name
            ? 'bg-blue-600 text-white border-blue-600'
            : 'border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700'"
        >
          #{{ tag.name }}
          <span class="text-xs opacity-70">({{ tag.memo_count }})</span>
        </button>
      </div>
    </div>
  </div>
</template>
