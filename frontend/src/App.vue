<script setup lang="ts">
import {computed} from "vue";
import {RouterView, useRoute, useRouter} from "vue-router";
import {
  canShowCustomTitleBtn,
  getQuickMenu,
  headButtons,
  IMenu,
  isMacOS,
  titleBarDoubleClick
} from "@/service/ApplicationService";
import {ChatLineSquare, Menu} from "@element-plus/icons-vue";
import Setting from "@/components/Setting.vue";

const router = useRouter()
const route = useRoute()
const quickMenu = getQuickMenu()
const macos = isMacOS()

const currentPageTitle = computed(() => {
  return quickMenu.find((item) => item.path === route.path)?.title || 'IC'
})

const changeRouter = (menu: IMenu) => {
  if (route.path !== menu.path) {
    router.push(menu.path)
  }
}
</script>

<template>
  <div class="drag-area" @dblclick="titleBarDoubleClick">
    <div class="drag-brand" :class="{ 'drag-brand--macos': macos }">
      <div class="drag-brand__mark">
        <el-icon><ChatLineSquare /></el-icon>
      </div>
      <div class="drag-brand__text">
        <strong>IC</strong>
        <span>{{ currentPageTitle }}</span>
      </div>
    </div>
    <div class="drag-child" v-if="canShowCustomTitleBtn()">
      <div v-for="button in headButtons" :key="button.name">
        <el-tooltip
            v-if="button.name?.trim()"
            :content="button.name"
            placement="bottom"
            effect="light"
        >
          <el-button
              type="primary"
              :link="true"
              :icon="button.icon"
              @click="button.click"
              class="drag-button"
          />
        </el-tooltip>
      </div>
    </div>
  </div>
  <div class="app-container">
    <Setting />
    <div class="app-sidebar">
      <div class="app-quick-menu">
        <div v-for="menu in quickMenu" :key="menu.id">
          <el-tooltip
              :content="menu.title"
              placement="left"
          >
            <el-button
                size="default"
                text
                circle
                :icon="menu.icon"
                @click="changeRouter(menu)"
                :class="['quick-menu-button', { 'is-active': route.path === menu.path }]"
            />
          </el-tooltip>
        </div>
      </div>
      <div class="app-menu-controller">
        <el-tooltip
            :content="currentPageTitle"
            placement="left"
        >
          <el-button size="default" text circle :icon="Menu"/>
        </el-tooltip>
      </div>
    </div>
    <div class="app-main">
      <div class="app-main__inner">
        <RouterView v-slot="{ Component }">
          <transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </RouterView>
      </div>
    </div>
  </div>
</template>

<style scoped>
.drag-area {
  background: var(--bg-topbar);
  width: 100%;
  height: var(--titlebar-height);
  z-index: 1000;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-4);
  border-bottom: 1px solid var(--border-subtle);
  --wails-draggable: drag;
  user-select: none;
  cursor: default;
}

.drag-brand {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  cursor: default;
}

.drag-brand--macos {
  margin-left: 84px;
}

.drag-brand__mark {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-primary) 0%, color-mix(in srgb, var(--color-primary) 66%, white 34%) 100%);
  color: var(--text-on-primary-strong);
  box-shadow: 0 6px 16px color-mix(in srgb, var(--color-primary) 30%, transparent 70%);
}

.drag-brand__text {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  color: var(--text-primary);
  max-width: 220px;
  white-space: nowrap;
  cursor: default;
}

.drag-brand__text strong {
  font-size: var(--font-size-sm);
  letter-spacing: 0.08em;
}

.drag-brand__text span {
  font-size: 11px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
}

.drag-child {
  display: flex;
  --wails-draggable: no-drag;
  align-items: center;
  margin-right: 5px;
  flex-direction: row;
}

.drag-button {
  transition: color 0.3s ease, background-color 0.2s ease, transform 0.2s ease;
  margin-right: 4px;
}

.drag-button:hover {
  background-color: var(--color-primary-soft);
  transform: scale(1.08);
}

.app-container {
  width: 100vw;
  height: calc(100vh - var(--titlebar-height));
  display: flex;
  background: radial-gradient(circle at top right, color-mix(in srgb, var(--color-primary) 18%, transparent 82%), transparent 28%), var(--bg-shell);
}

.app-sidebar {
  width: var(--sidebar-width);
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-4) 0 var(--space-3);
  border-right: 1px solid var(--border-subtle);
  background: var(--bg-sidebar);
  backdrop-filter: var(--glass-blur);
}

.app-quick-menu {
  flex: 1;
  display: flex;
  width: 100%;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
}

.quick-menu-button {
  width: var(--sidebar-button-size);
  height: var(--sidebar-button-size);
  border-radius: var(--radius-lg);
  color: var(--text-secondary);
}

.quick-menu-button.is-active {
  color: var(--color-primary);
  background: var(--color-primary-soft);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-primary) 18%, transparent 82%);
}

.app-menu-controller {
  height: calc(var(--sidebar-button-size) + 4px);
  display: flex;
  align-items: center;
}

.app-main {
  flex: 1;
  height: calc(100vh - var(--titlebar-height));
  overflow: hidden;
  padding: var(--layout-page-padding);
}

.app-main__inner {
  width: min(100%, var(--layout-content-max-width));
  height: 100%;
  margin: 0 auto;
}

.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.page-fade-enter-from,
.page-fade-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

@media (max-width: 960px) {
  .drag-brand--macos {
    margin-left: 76px;
  }

  .drag-brand__text {
    max-width: 140px;
  }
}
</style>
