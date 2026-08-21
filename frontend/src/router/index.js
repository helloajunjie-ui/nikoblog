import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import ProfileView from '../views/ProfileView.vue'
import AdminView from '../views/AdminView.vue'
import MemoDetailView from '../views/MemoDetailView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView },
    { path: '/profile', name: 'profile', component: ProfileView },
    { path: '/admin', name: 'admin', component: AdminView },
    // 独立博文详情页：可通过 /m/:id 直接分享单篇博文
    { path: '/m/:id', name: 'memo-detail', component: MemoDetailView }
  ]
})

export default router
