import http from './http'

// ===== 认证 =====
export const register = (data) => http.post('/api/auth/register', data)
export const login = (data) => http.post('/api/auth/login', data)

// ===== 密保找回 =====
export const getSecurityQuestion = (data) => http.post('/api/auth/security/question', data)
export const forgotUsername = (data) => http.post('/api/auth/forgot/username', data)
export const forgotPassword = (data) => http.post('/api/auth/forgot/password', data)
export const updateSecurity = (data) => http.put('/api/auth/security', data)
export const changePassword = (data) => http.put('/api/auth/password', data)

// ===== 博文 =====
export const fetchMemos = (params) => http.get('/api/memos', { params })
export const fetchMemo = (id) => http.get(`/api/memos/${id}`)
export const createMemo = (data) => http.post('/api/memos', data)
export const updateMemo = (id, data) => http.put(`/api/memos/${id}`, data)
export const deleteMemo = (id) => http.delete(`/api/memos/${id}`)
export const searchMemos = (params) => http.get('/api/memos/search', { params })

// ===== 标签 =====
export const fetchTags = () => http.get('/api/tags')
export const fetchHotTags = () => http.get('/api/tags/hot')

// ===== 评论 =====
export const fetchCommentSettings = () => http.get('/api/settings/comments')
export const fetchComments = (memoId) => http.get(`/api/memos/${memoId}/comments`)
export const createComment = (memoId, data) => http.post(`/api/memos/${memoId}/comments`, data)
export const deleteComment = (id) => http.delete(`/api/comments/${id}`)
// 我评论过的博文（用户中心"回复过的主题"）
export const fetchMyCommentedMemos = () => http.get('/api/memos/commented')

// ===== 上传 =====
export const uploadImage = (file) => {
  const form = new FormData()
  form.append('file', file)
  return http.post('/api/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}
// 上传头像（最大 2MB，上传后自动更新当前用户头像）
export const uploadAvatar = (file) => {
  const form = new FormData()
  form.append('file', file)
  return http.post('/api/upload/avatar', form, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// ===== 后台管理（仅 admin） =====
// 博客设置
export const fetchSettings = () => http.get('/api/admin/settings')
export const updateSettings = (data) => http.put('/api/admin/settings', data)
// 用户管理
export const fetchAdminUsers = (params) => http.get('/api/admin/users', { params })
export const updateUserRole = (id, role) => http.put(`/api/admin/users/${id}/role`, { role })
export const deleteUser = (id) => http.delete(`/api/admin/users/${id}`)
// 文章管理
export const fetchAdminMemos = (params) => http.get('/api/admin/memos', { params })
export const deleteAdminMemo = (id) => http.delete(`/api/admin/memos/${id}`)
export const pinMemo = (id, data) => http.put(`/api/admin/memos/${id}/pin`, data)
// TAG 管理
export const fetchAdminTags = () => http.get('/api/admin/tags')
export const deleteAdminTag = (id) => http.delete(`/api/admin/tags/${id}`)

// ===== AI 能力（仅 admin） =====
// AI 润色（调用配置好的 AI 模型优化博文内容）
export const polishContent = (content) => http.post('/api/admin/ai/polish', { content })
