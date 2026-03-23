<script setup lang="ts">
import {showSetting} from "@/service/ApplicationService.js";
import {computed, ref, watch} from "vue";
import {LoadAppConfig, SaveAppConfig} from "../../bindings/ic-wails/internal/config/applicationconfigmanager";
import {currentTheme, setThemeMode, themeMode, ThemeMode} from "@/service/ThemeService";
import {
  getModelOptionLabel,
  getModelOptionTagText,
  getModelOptionTagType,
  normalizeModelName,
  requestAiModelList,
  type AiModelOption,
} from "@/service/AiModelService";

interface RuntimeAppConfig {
  App: {
    Name: string;
    Version: string;
  };
  Window: {
    Width: number;
    Height: number;
    X: string;
    Y: string;
  };
  Ai: {
    ApiKey: string;
    Model: string;
    BaseUrl: string;
  };
}

const loading = ref<boolean>(false)
const modelLoading = ref<boolean>(false)
const modelOptions = ref<AiModelOption[]>([])
const modelTip = ref<string>("")
const hasRemoteModelList = ref<boolean>(false)

const config = ref<RuntimeAppConfig | null>(null)

const displayModelOptions = computed(() => {
  const merged = new Map<string, AiModelOption>()
  modelOptions.value.forEach((item) => {
    const normalized = normalizeModelName(item.value)
    if (normalized) {
      merged.set(normalized, item)
    }
  })

  const current = normalizeModelName(config.value?.Ai?.Model)
  if (current && !merged.has(current)) {
    merged.set(current, {
      value: current,
      label: current,
      source: "manual",
      available: false,
      status: hasRemoteModelList.value ? "manual" : "fallback",
      hint: hasRemoteModelList.value ? "该模型不在远程可用列表中" : "当前模型列表未经过远程校验",
    })
  }

  return Array.from(merged.values())
})

const loadConfig = async () => {
  loading.value = true
  try {
    const result = await LoadAppConfig() as unknown as RuntimeAppConfig | null
    config.value = result || null
  } finally {
    loading.value = false
  }
}

const loadModelOptions = async (forceRefresh = false) => {
  modelLoading.value = true
  modelTip.value = ""
  try {
    const result = await requestAiModelList(forceRefresh)
    modelOptions.value = result.models
    hasRemoteModelList.value = !result.fallback
    modelTip.value = result.message || ""
  } catch (error: any) {
    hasRemoteModelList.value = false
    modelTip.value = error?.message || "模型列表获取失败，可手动输入模型"
  } finally {
    modelLoading.value = false
  }
}

const saveSetting = () => {
  if (config.value) {
    loading.value = true
    SaveAppConfig(config.value as never).then(() => {
      showSetting.value = false
    }).finally(() => {
      loading.value = false
    })
  } else {
    showSetting.value = false
  }
}

watch(showSetting, (visible) => {
  if (visible) {
    void Promise.allSettled([loadConfig(), loadModelOptions()])
  }
}, {immediate: true})

const resetDefaultModelTip = '这里配置的是应用启动后或新建会话时使用的默认模型。'
const themeOptions: Array<{label: string, value: ThemeMode}> = [
  {label: '跟随系统', value: 'auto'},
  {label: '浅色', value: 'light'},
  {label: '深色', value: 'dark'},
]

const currentThemeLabel = computed(() => currentTheme.value === 'dark' ? '深色' : '浅色')

type SettingSection = 'ai' | 'appearance' | 'about'

const currentSection = ref<SettingSection>('ai')

const settingSections: Array<{ label: string, value: SettingSection }> = [
  {label: 'AI 模型', value: 'ai'},
  {label: '界面', value: 'appearance'},
  {label: '关于', value: 'about'},
]

const handleThemeModeChange = (mode: ThemeMode) => {
  setThemeMode(mode)
}
</script>

<template>
  <el-dialog
    class="setting-dialog"
    v-model="showSetting"
    width="80vw"
    :style="{ maxWidth: '980px', maxHeight: 'min(78vh, 760px)', minHeight: '620px' }"
    header-class="setting-dialog__header"
    body-class="setting-dialog__body"
    :modal="true"
    :show-close="true"
    align-center
    :close-on-click-modal="false"
  >
    <template #header>
      <div class="setting-header">
        <h2 class="setting-header__title">设置</h2>
        <p class="setting-header__subtitle">统一管理模型接入、界面主题与应用基础信息。</p>
        <el-divider />
      </div>
    </template>
    <template #default>
      <div class="app-setting" v-loading="loading">
        <div class="setting-main">
          <div class="setting-layout">
            <nav class="setting-nav" aria-label="设置分类">
              <button
                v-for="section in settingSections"
                :key="section.value"
                type="button"
                class="setting-nav__button"
                :class="{ 'is-active': currentSection === section.value }"
                @click="currentSection = section.value"
              >
                {{ section.label }}
              </button>
            </nav>

            <section class="setting-content">
              <div v-show="currentSection === 'ai'" class="setting-pane">
                <el-form class="setting-form" label-position="top" v-if="config">
                <el-form-item label="API Key">
                  <el-input
                    v-model="config.Ai.ApiKey"
                    type="password"
                    size="large"
                    show-password
                    placeholder="填写火山引擎 Ark API Key"
                  />
                </el-form-item>
                <el-form-item label="模型">
                  <el-select
                    v-model="config.Ai.Model"
                    filterable
                    allow-create
                    default-first-option
                    clearable
                    :loading="modelLoading"
                    size="large"
                    placeholder="请选择或输入模型"
                    style="width: 100%"
                  >
                    <el-option
                      v-for="item in displayModelOptions"
                      :key="item.value"
                      :label="getModelOptionLabel(item)"
                      :value="item.value"
                    >
                      <div class="setting-model-option">
                        <span class="setting-model-option__label">{{ item.label }}</span>
                        <div class="setting-model-option__meta">
                          <el-tag v-if="getModelOptionTagText(item)" :type="getModelOptionTagType(item)" effect="plain" size="small">
                            {{ getModelOptionTagText(item) }}
                          </el-tag>
                          <span v-if="item.hint" class="setting-model-option__hint">{{ item.hint }}</span>
                        </div>
                      </div>
                    </el-option>
                  </el-select>
                  <div class="setting-model-tools">
                    <el-button link type="primary" :loading="modelLoading" @click="loadModelOptions(true)">
                      刷新模型列表
                    </el-button>
                    <span class="setting-tip">未命中列表时可直接输入模型名</span>
                  </div>
                  <div v-if="modelTip" class="setting-tip">{{ modelTip }}</div>
                  <div class="setting-tip">{{ resetDefaultModelTip }}</div>
                </el-form-item>
                <el-form-item label="接口地址">
                  <el-input
                    v-model="config.Ai.BaseUrl"
                    size="large"
                    placeholder="https://ark.cn-beijing.volces.com/api/v3"
                  />
                </el-form-item>
                </el-form>
              </div>

              <div v-show="currentSection === 'appearance'" class="setting-pane setting-panel">
                <div class="setting-panel__title">主题模式</div>
                <el-radio-group :model-value="themeMode" @update:model-value="handleThemeModeChange">
                  <el-radio-button
                    v-for="option in themeOptions"
                    :key="option.value"
                    :label="option.value"
                    :value="option.value"
                  >
                    {{ option.label }}
                  </el-radio-button>
                </el-radio-group>
                <div class="setting-tip">当前系统生效主题：{{ currentThemeLabel }}</div>
              </div>

              <div v-show="currentSection === 'about'" class="setting-pane setting-about">
                <p>{{ config?.App.Name }}</p>
                <p>版本信息：{{ config?.App.Version }}</p>
              </div>
            </section>
          </div>
        </div>
        <div class="setting-button">
          <el-button type="primary" @click="saveSetting">保存</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<style>
.setting-dialog {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-2xl);
  background: var(--bg-card);
  border: 1px solid var(--border-strong);
  box-shadow: var(--shadow-soft);
}

.setting-dialog__header {
  margin-right: 0;
  padding: var(--space-6) var(--space-7) 0;
}

.setting-dialog__body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 0 var(--space-7) var(--space-7);
}

.setting-dialog .el-dialog__headerbtn {
  top: var(--space-6);
  right: var(--space-6);
}

.setting-dialog .el-dialog__headerbtn .el-dialog__close {
  color: var(--text-muted);
}

.setting-dialog .el-dialog__headerbtn:hover .el-dialog__close {
  color: var(--text-primary);
}

@media (max-width: 900px) {
  .setting-dialog {
    width: 92vw !important;
    max-height: min(84vh, 760px) !important;
    min-height: 0 !important;
  }

  .setting-dialog__header {
    padding: var(--space-5) var(--space-5) 0;
  }

  .setting-dialog__body {
    padding: 0 var(--space-5) var(--space-5);
  }
}
</style>

<style scoped>
.setting-header__title {
  margin: 0;
  font-size: var(--font-size-2xl);
  line-height: var(--line-height-compact);
  color: var(--text-primary);
}

.setting-header__subtitle {
  margin: var(--space-2) 0 0;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
}

.app-setting {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.setting-main {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.setting-layout {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 156px minmax(0, 1fr);
  gap: var(--space-5);
}

.setting-nav {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-subtle);
  background: color-mix(in srgb, var(--bg-card-muted) 88%, transparent);
}

.setting-nav__button {
  min-height: 48px;
  padding: 0 var(--space-4);
  border: 0;
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--text-secondary);
  text-align: left;
  font-size: var(--font-size-sm);
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.setting-nav__button:hover {
  background: color-mix(in srgb, var(--color-primary-soft) 60%, transparent);
  color: var(--text-primary);
}

.setting-nav__button.is-active {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  box-shadow: inset 2px 0 0 var(--color-primary);
}

.setting-content {
  min-height: 0;
  overflow: auto;
  padding: var(--space-5);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-subtle);
  background: color-mix(in srgb, var(--bg-card) 96%, transparent);
}

.setting-pane {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-height: 100%;
}

.setting-form {
  max-width: 620px;
}

.setting-tip {
  margin-top: 6px;
  color: var(--text-muted);
  font-size: var(--font-size-xs);
  line-height: var(--line-height-comfortable);
}

.setting-model-tools {
  margin-top: var(--space-2);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.setting-panel {
  max-width: 560px;
  color: var(--text-primary);
}

.setting-panel__title {
  margin-bottom: var(--space-3);
  font-size: var(--font-size-md);
  font-weight: 600;
}

.setting-model-option {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 2px 0;
}

.setting-model-option__label {
  color: var(--text-primary);
}

.setting-model-option__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.setting-model-option__hint {
  color: var(--text-muted);
  font-size: var(--font-size-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.setting-about {
  justify-content: center;
  color: var(--text-primary);
}

.setting-button {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-3);
  border-top: 1px solid var(--border-subtle);
}

@media (max-width: 900px) {
  .setting-layout {
    grid-template-columns: 1fr;
    gap: var(--space-4);
  }

  .setting-nav {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .setting-nav__button {
    text-align: center;
    padding: 0 var(--space-3);
  }

  .setting-content {
    padding: var(--space-4);
  }

  .setting-model-tools {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>