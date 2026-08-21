<script setup>
import { computed } from 'vue'

// 角色徽章：根据角色显示不同颜色的标识
// admin=博主（金色）、user=会员（蓝色）、游客（灰色）
const props = defineProps({
  // 用户对象（含 role 字段）；为 null 表示游客
  user: { type: Object, default: null },
  // 是否为游客（当 user 为 null 时强制显示游客徽章）
  isGuest: { type: Boolean, default: false }
})

const badge = computed(() => {
  // 游客：user 为 null 或显式标记为游客
  if (props.isGuest || !props.user) {
    return { label: '游客', cls: 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400' }
  }
  if (props.user.role === 'admin') {
    return { label: '博主', cls: 'bg-amber-100 dark:bg-amber-900/60 text-amber-700 dark:text-amber-300' }
  }
  // 默认会员
  return { label: '会员', cls: 'bg-blue-100 dark:bg-blue-900/60 text-blue-600 dark:text-blue-300' }
})
</script>

<template>
  <span
    class="inline-block text-[10px] leading-none px-1.5 py-1 rounded-full font-medium align-middle shrink-0"
    :class="badge.cls"
  >{{ badge.label }}</span>
</template>
