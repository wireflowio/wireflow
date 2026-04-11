// router/index.ts
import NProgress from 'nprogress'
import 'nprogress/nprogress.css'
import {Router} from "vue-router";

// 配置（可选）
NProgress.configure({ showSpinner: false })

export function setupProgressGuard(router: Router) {
    router.beforeEach((to, from, next) => {
       NProgress.start()
       next()
    })

    router.afterEach(() => {
        NProgress.done() // 结束加载
    })
}
