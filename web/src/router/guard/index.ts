import Router from "@/router";
import {setupAuthGuard} from "@/router/guard/authGuard";
import {setupProgressGuard} from "@/router/guard/progressGuard";

export function setupRouterGuard(router: Router) {
    setupAuthGuard(router)
    setupProgressGuard(router)
}