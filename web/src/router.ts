import { createRouter, createWebHistory } from "vue-router";
import { defineAsyncComponent } from "vue";
import { api, handleAuthFailure } from "./api";

// 缓存 setup_complete 状态，避免每次导航都请求 API
let setupComplete: boolean | null = null;

export function markSetupComplete() {
  setupComplete = true;
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/setup",
      name: "wizard",
      component: defineAsyncComponent(() => import("./views/Wizard.vue")),
    },
    {
      path: "/",
      name: "layout",
      component: defineAsyncComponent(() => import("./components/Layout.vue")),
      redirect: "/dashboard",
      children: [
        {
          path: "dashboard",
          name: "dashboard",
          component: defineAsyncComponent(
            () => import("./views/Dashboard.vue"),
          ),
        },
        {
          path: "explore",
          name: "explore",
          component: defineAsyncComponent(
            () => import("./views/RestoreExplorer.vue"),
          ),
        },
        {
          path: "settings",
          name: "settings",
          component: defineAsyncComponent(
            () => import("./views/Settings.vue"),
          ),
        },
      ],
    },
  ],
});

router.beforeEach(async (to, _from, next) => {
  // 如果已经确认 setup 完成，仅阻止再次访问 wizard
  if (setupComplete === true) {
    if (to.name === "wizard") {
      next({ name: "dashboard" });
    } else {
      next();
    }
    return;
  }

  // 仅在首次导航或未确认状态时请求 API
  try {
    const status = await api.getSystemStatus();
    setupComplete = status.setup_complete;

    if (!status.setup_complete && to.name !== "wizard") {
      next({ name: "wizard" });
    } else if (status.setup_complete && to.name === "wizard") {
      next({ name: "dashboard" });
    } else {
      next();
    }
  } catch (error) {
    if (handleAuthFailure(error)) {
      next(false);
      return;
    }

    console.error("Failed to get system status", error);
    // API 不可达时直接放行，避免阻塞页面
    next();
  }
});

export default router;
