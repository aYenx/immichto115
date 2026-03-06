import { createRouter, createWebHistory } from "vue-router";
import { defineAsyncComponent } from "vue";
import { api } from "./api";

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
      ],
    },
  ],
});

router.beforeEach(async (to, _from, next) => {
  try {
    const status = await api.getSystemStatus()
    if (!status.setup_complete && to.name !== 'wizard') {
      next({ name: 'wizard' })
    } else if (status.setup_complete && to.name === 'wizard') {
      next({ name: 'dashboard' })
    } else {
      next()
    }
  } catch (error) {
    console.error('Failed to get system status', error)
    next()
  }
})

export default router;
