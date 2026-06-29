// @ts-ignore
import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'
import devices from './pages/devices/devices.vue' 

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    name: 'home',
    component: devices
  },
  {
    path: '/upload/:id',
    name: 'upload',
    component: () => import('./pages/upload/drop.vue'),
    props: true 
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
