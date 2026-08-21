import axios from 'axios'

// 统一 axios 实例，baseURL 为空（走 vite 代理 /api）
const http = axios.create({
  baseURL: '',
  timeout: 15000
})

// 请求拦截器：自动附加 JWT
http.interceptors.request.use((config) => {
  const token = localStorage.getItem('nikoblog_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：统一错误提示
http.interceptors.response.use(
  (res) => res.data,
  (err) => {
    const msg =
      err.response?.data?.error || err.message || '网络请求失败'
    return Promise.reject(new Error(msg))
  }
)

export default http
