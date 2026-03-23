import {createRouter, createWebHistory, RouteRecordRaw} from 'vue-router'
import Home from '@/page/Home.vue'
import Chat from '@/page/Chat.vue'
import UndefinedRouter from '@/components/UndefinedRouter.vue'

const routes: RouteRecordRaw[] = [
    {
        path: '/',
        name: 'home',
        component: Home,
        meta: {
            title: '主页'
        }
    },
    {
        path: '/chat',
        name: 'chat',
        component: Chat,
        meta: {
            title: '智能助理'
        }
    },
    {
        path: '/:pathMatch(.*)*',
        name: 'not-found',
        component: UndefinedRouter,
        meta: {
            title: '页面不存在'
        }
    }
]

const router = createRouter({
    history: createWebHistory('/static/'),
    routes
})

router.afterEach((to) => {
    document.title = `IC - ${String(to.meta.title || '应用')}`
})

export default router