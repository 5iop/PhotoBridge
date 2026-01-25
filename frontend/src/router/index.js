import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/admin'
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/admin/Login.vue')
  },
  {
    path: '/admin',
    name: 'Dashboard',
    component: () => import('../views/admin/Dashboard.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/admin/project/:id',
    name: 'ProjectDetail',
    component: () => import('../views/admin/ProjectDetail.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/admin/project/:id/links',
    name: 'LinkManage',
    component: () => import('../views/admin/LinkManage.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/share/:token',
    name: 'Gallery',
    component: () => import('../views/client/Gallery.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router
