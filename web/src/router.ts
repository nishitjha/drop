// @ts-ignore
import { createRouter, createWebHistory } from "vue-router";
import devices from "@/views/devices.vue";

const routes: Array<any> = [
  {
    path: "/",
    name: "home",
    component: devices,
  },
  {
    path: "/share/:deviceId",
    name: "share",
    component: () => import("@/views/share.vue"),
    props: true,
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
