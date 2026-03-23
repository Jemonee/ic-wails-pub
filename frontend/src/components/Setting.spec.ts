import { computed, defineComponent, h, nextTick, ref } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

type ThemeMode = 'auto' | 'light' | 'dark'
type ResolvedTheme = 'light' | 'dark'

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

interface AiModelOption {
  value: string;
  label: string;
  source?: string;
  available?: boolean;
  status?: 'available' | 'unavailable' | 'fallback' | 'manual';
  hint?: string;
}

vi.mock('@/service/ApplicationService.js', async () => {
  const { ref } = await import('vue')
  return {
    showSetting: ref(false),
  }
})

vi.mock('../../bindings/ic-wails/internal/config/applicationconfigmanager', () => ({
  LoadAppConfig: vi.fn(),
  SaveAppConfig: vi.fn(),
}))

vi.mock('@/service/ThemeService', async () => {
  const { computed, ref } = await import('vue')
  const themeMode = ref<ThemeMode>('auto')
  const resolvedTheme = ref<ResolvedTheme>('light')
  const setThemeMode = vi.fn((mode: ThemeMode) => {
    themeMode.value = mode
    resolvedTheme.value = mode === 'auto' ? 'light' : mode
  })

  return {
    currentTheme: computed(() => resolvedTheme.value),
    setThemeMode,
    themeMode,
    ThemeMode: {},
    __resolvedTheme: resolvedTheme,
  }
})

vi.mock('@/service/AiModelService', () => ({
  normalizeModelName: (modelName?: string | null) => modelName?.trim() || '',
  requestAiModelList: vi.fn(),
  getModelOptionLabel: (item: AiModelOption) => {
    if (item.status === 'unavailable') {
      return `${item.label}（不可用）`
    }
    if (item.status === 'fallback') {
      return `${item.label}（未校验）`
    }
    if (item.status === 'manual') {
      return `${item.label}（手动输入）`
    }
    return item.label
  },
  getModelOptionTagText: (item: AiModelOption) => {
    if (item.status === 'unavailable') {
      return '不可用'
    }
    if (item.status === 'fallback') {
      return '未校验'
    }
    if (item.status === 'manual') {
      return '手动输入'
    }
    return ''
  },
  getModelOptionTagType: (item: AiModelOption) => {
    if (item.status === 'unavailable') {
      return 'danger'
    }
    if (item.status === 'fallback') {
      return 'warning'
    }
    return 'info'
  },
}))

import { showSetting } from '@/service/ApplicationService.js'
import { requestAiModelList } from '@/service/AiModelService'
import { __resolvedTheme as resolvedThemeRef, setThemeMode, themeMode as themeModeRef } from '@/service/ThemeService'
import { LoadAppConfig, SaveAppConfig } from '../../bindings/ic-wails/internal/config/applicationconfigmanager'
import Setting from './Setting.vue'

const ElDialogStub = defineComponent({
  name: 'ElDialog',
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () => props.modelValue
      ? h('div', { 'data-test': 'dialog' }, [
        slots.header?.(),
        slots.default?.(),
      ])
      : h('div')
  },
})

const ElFormStub = defineComponent({
  name: 'ElForm',
  setup(_, { slots }) {
    return () => h('form', slots.default?.())
  },
})

const ElFormItemStub = defineComponent({
  name: 'ElFormItem',
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const ElInputStub = defineComponent({
  name: 'ElInput',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [String, Number],
      default: '',
    },
    type: {
      type: String,
      default: 'text',
    },
    placeholder: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('input', {
      value: props.modelValue,
      type: props.type,
      placeholder: props.placeholder,
      onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
    })
  },
})

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  setup(_, { slots }) {
    return () => h('div', { 'data-test': 'select' }, slots.default?.())
  },
})

const ElOptionStub = defineComponent({
  name: 'ElOption',
  props: {
    value: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', { 'data-option-value': props.value }, slots.default?.())
  },
})

const ElTagStub = defineComponent({
  name: 'ElTag',
  setup(_, { slots }) {
    return () => h('span', slots.default?.())
  },
})

const ElButtonStub = defineComponent({
  name: 'ElButton',
  props: {
    loading: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['click'],
  setup(props, { slots, emit }) {
    return () => h('button', {
      disabled: props.loading,
      onClick: () => emit('click'),
    }, slots.default?.())
  },
})

const ElDividerStub = defineComponent({
  name: 'ElDivider',
  setup() {
    return () => h('hr')
  },
})

const ElRadioGroupStub = defineComponent({
  name: 'ElRadioGroup',
  emits: ['update:modelValue'],
  setup(_, { slots }) {
    return () => h('div', { 'data-test': 'radio-group' }, slots.default?.())
  },
})

const ElRadioButtonStub = defineComponent({
  name: 'ElRadioButton',
  props: {
    value: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('button', { 'data-radio-value': props.value }, slots.default?.())
  },
})

const baseConfig = (overrides?: Partial<RuntimeAppConfig>): RuntimeAppConfig => ({
  App: {
    Name: 'IC-Wails',
    Version: '0.1.0',
    ...(overrides?.App || {}),
  },
  Window: {
    Width: 1280,
    Height: 800,
    X: '0',
    Y: '0',
    ...(overrides?.Window || {}),
  },
  Ai: {
    ApiKey: 'test-api-key',
    Model: 'doubao-pro',
    BaseUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    ...(overrides?.Ai || {}),
  },
})

const mountSetting = () => mount(Setting, {
  global: {
    components: {
      ElDialog: ElDialogStub,
      ElForm: ElFormStub,
      ElFormItem: ElFormItemStub,
      ElInput: ElInputStub,
      ElSelect: ElSelectStub,
      ElOption: ElOptionStub,
      ElTag: ElTagStub,
      ElButton: ElButtonStub,
      ElDivider: ElDividerStub,
      ElRadioGroup: ElRadioGroupStub,
      ElRadioButton: ElRadioButtonStub,
    },
    directives: {
      loading: {
        mounted() {},
        updated() {},
      },
    },
  },
})

const settleComponent = async () => {
  await flushPromises()
  await nextTick()
}

const findButtonByText = (wrapper: VueWrapper<any>, text: string) => {
  const button = wrapper.findAll('button').find((item) => item.text().trim() === text)
  if (!button) {
    throw new Error(`未找到按钮: ${text}`)
  }
  return button
}

beforeEach(() => {
  showSetting.value = false
  themeModeRef.value = 'auto'
  resolvedThemeRef.value = 'light'
  vi.mocked(LoadAppConfig).mockReset()
  vi.mocked(SaveAppConfig).mockReset()
  vi.mocked(requestAiModelList).mockReset()
  vi.mocked(setThemeMode).mockClear()
})

afterEach(() => {
  showSetting.value = false
})

describe('Setting.vue', () => {
  it('打开设置时会加载配置和模型列表', async () => {
    vi.mocked(LoadAppConfig).mockResolvedValue(baseConfig())
    vi.mocked(requestAiModelList).mockResolvedValue({
      models: [{ value: 'doubao-pro', label: '豆包 Pro', status: 'available' }],
      fallback: false,
      message: '模型列表已同步',
    })
    showSetting.value = true

    const wrapper = mountSetting()
    await settleComponent()

    expect(LoadAppConfig).toHaveBeenCalledTimes(1)
    expect(requestAiModelList).toHaveBeenCalledWith(false)
    const inputs = wrapper.findAll('input')
    expect(inputs[0]?.element.value).toBe('test-api-key')
    expect(wrapper.text()).toContain('模型列表已同步')

    wrapper.unmount()
  })

  it('当前模型不在远程列表中时会展示手动输入提示', async () => {
    vi.mocked(LoadAppConfig).mockResolvedValue(baseConfig({
      Ai: {
        ApiKey: 'custom-key',
        Model: 'custom-model',
        BaseUrl: 'https://example.com/api',
      },
    }))
    vi.mocked(requestAiModelList).mockResolvedValue({
      models: [{ value: 'doubao-pro', label: '豆包 Pro', status: 'available' }],
      fallback: false,
      message: '',
    })
    showSetting.value = true

    const wrapper = mountSetting()
    await settleComponent()

    expect(wrapper.text()).toContain('手动输入')
    expect(wrapper.text()).toContain('该模型不在远程可用列表中')

    wrapper.unmount()
  })

  it('点击刷新模型列表会使用 forceRefresh 重新拉取', async () => {
    vi.mocked(LoadAppConfig).mockResolvedValue(baseConfig())
    vi.mocked(requestAiModelList)
      .mockResolvedValueOnce({
        models: [{ value: 'doubao-pro', label: '豆包 Pro', status: 'available' }],
        fallback: false,
        message: '首次加载完成',
      })
      .mockResolvedValueOnce({
        models: [{ value: 'doubao-pro', label: '豆包 Pro 2', status: 'available' }],
        fallback: false,
        message: '刷新完成',
      })
    showSetting.value = true

    const wrapper = mountSetting()
    await settleComponent()
    await findButtonByText(wrapper, '刷新模型列表').trigger('click')
    await settleComponent()

    expect(requestAiModelList).toHaveBeenNthCalledWith(1, false)
    expect(requestAiModelList).toHaveBeenNthCalledWith(2, true)
    expect(wrapper.text()).toContain('刷新完成')

    wrapper.unmount()
  })

  it('模型列表加载失败时会显示错误提示', async () => {
    vi.mocked(LoadAppConfig).mockResolvedValue(baseConfig({
      Ai: {
        ApiKey: 'error-key',
        Model: 'missing-model',
        BaseUrl: 'https://example.com/api',
      },
    }))
    vi.mocked(requestAiModelList).mockRejectedValue(new Error('模型接口异常'))
    showSetting.value = true

    const wrapper = mountSetting()
    await settleComponent()

    expect(wrapper.text()).toContain('模型接口异常')
    expect(wrapper.text()).toContain('未校验')

    wrapper.unmount()
  })

  it('切换主题模式时会调用主题服务并更新当前主题显示', async () => {
    vi.mocked(LoadAppConfig).mockResolvedValue(baseConfig())
    vi.mocked(requestAiModelList).mockResolvedValue({
      models: [{ value: 'doubao-pro', label: '豆包 Pro', status: 'available' }],
      fallback: false,
      message: '',
    })
    showSetting.value = true

    const wrapper = mountSetting()
    await settleComponent()

    wrapper.findComponent(ElRadioGroupStub).vm.$emit('update:modelValue', 'dark')
    await nextTick()

    expect(setThemeMode).toHaveBeenCalledWith('dark')
    expect(wrapper.text()).toContain('当前系统生效主题：深色')

    wrapper.unmount()
  })

  it('保存设置后会提交配置并关闭弹窗', async () => {
    vi.mocked(LoadAppConfig).mockResolvedValue(baseConfig())
    vi.mocked(SaveAppConfig).mockResolvedValue()
    vi.mocked(requestAiModelList).mockResolvedValue({
      models: [{ value: 'doubao-pro', label: '豆包 Pro', status: 'available' }],
      fallback: false,
      message: '',
    })
    showSetting.value = true

    const wrapper = mountSetting()
    await settleComponent()
    await findButtonByText(wrapper, '保存').trigger('click')
    await settleComponent()

    expect(SaveAppConfig).toHaveBeenCalledWith(expect.objectContaining({
      Ai: expect.objectContaining({
        ApiKey: 'test-api-key',
        Model: 'doubao-pro',
      }),
    }))
    expect(showSetting.value).toBe(false)

    wrapper.unmount()
  })
})