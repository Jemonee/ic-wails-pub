import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

interface RuntimeAppConfig {
  Ai?: {
    Model?: string;
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

interface ChatSessionSummary {
  sessionId: string;
  title: string;
  preview?: string;
  model?: string;
  roundCount: number;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  avgDurationMs: number;
  avgFirstTokenMs: number;
  lastActiveTime: string;
}

vi.mock('../../bindings/ic-wails/internal/config/applicationconfigmanager', () => ({
  LoadAppConfig: vi.fn(),
}))

vi.mock('@/service/AiModelService', () => ({
  requestAiModelList: vi.fn(),
  normalizeModelName: (modelName?: string | null) => modelName?.trim() || '',
  isUnavailableModelError: (message?: string | null) => /InvalidEndpointOrModel\.NotFound|does not exist or you do not have access|model or endpoint/i.test((message || '').trim()),
  getUnavailableModelHint: (message?: string | null) => (message || '').trim() || '该模型暂时不可用',
  getModelOptionLabel: (item: AiModelOption) => item.label,
  getModelOptionTagText: () => '',
  getModelOptionTagType: () => 'info',
}))

vi.mock('@/service/AiChatService', () => ({
  requestChatSessionList: vi.fn(),
  requestChatSessionDetail: vi.fn(),
}))

import { LoadAppConfig } from '../../bindings/ic-wails/internal/config/applicationconfigmanager'
import { requestAiModelList } from '@/service/AiModelService'
import { requestChatSessionDetail, requestChatSessionList } from '@/service/AiChatService'
import Chat from './Chat.vue'

const ElScrollbarStub = defineComponent({
  name: 'ElScrollbar',
  setup(_, { slots, expose }) {
    expose({
      setScrollTop: vi.fn(),
    })
    return () => h('div', { 'data-test': 'scrollbar' }, slots.default?.())
  },
})

const ElSwitchStub = defineComponent({
  name: 'ElSwitch',
  props: {
    modelValue: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('input', {
      type: 'checkbox',
      checked: props.modelValue,
      'data-test': 'compare-switch',
      onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).checked),
    })
  },
})

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [String, Array],
      default: '',
    },
    multiple: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, slots, attrs }) {
    return () => h('select', {
      ...attrs,
      multiple: props.multiple,
      value: props.modelValue,
      onChange: (event: Event) => {
        const target = event.target as HTMLSelectElement
        if (props.multiple) {
          emit('update:modelValue', Array.from(target.selectedOptions).map((item) => item.value))
          return
        }
        emit('update:modelValue', target.value)
      },
    }, slots.default?.())
  },
})

const ElOptionStub = defineComponent({
  name: 'ElOption',
  props: {
    value: {
      type: String,
      default: '',
    },
    label: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    return () => h('option', { value: props.value }, props.label || props.value)
  },
})

const ElButtonStub = defineComponent({
  name: 'ElButton',
  props: {
    loading: {
      type: Boolean,
      default: false,
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['click'],
  setup(props, { slots, emit, attrs }) {
    return () => h('button', {
      ...attrs,
      disabled: props.disabled || props.loading,
      onClick: () => emit('click'),
    }, slots.default?.())
  },
})

const ElInputStub = defineComponent({
  name: 'ElInput',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: String,
      default: '',
    },
    type: {
      type: String,
      default: 'text',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () => props.type === 'textarea'
      ? h('textarea', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
      })
      : h('input', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      })
  },
})

const ElTagStub = defineComponent({
  name: 'ElTag',
  setup(_, { slots }) {
    return () => h('span', slots.default?.())
  },
})

const ElTooltipStub = defineComponent({
  name: 'ElTooltip',
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const baseConfig = (model = 'gpt-4o'): RuntimeAppConfig => ({
  Ai: {
    Model: model,
  },
})

const baseModels = (items?: string[]) => ({
  models: (items || ['gpt-4o', 'deepseek-r1', 'glm-4', 'claude-3-5']).map((item) => ({
    value: item,
    label: item,
    available: true,
    status: 'available' as const,
  })),
  fallback: false,
  message: '',
})

const createNdjsonResponse = (chunks: Array<Record<string, unknown>>) => {
  const encoder = new TextEncoder()
  const stream = new ReadableStream({
    start(controller) {
      chunks.forEach((chunk) => {
        controller.enqueue(encoder.encode(`${JSON.stringify(chunk)}\n`))
      })
      controller.close()
    },
  })

  return new Response(stream, {
    status: 200,
    headers: {
      'Content-Type': 'application/x-ndjson',
    },
  })
}

const mountChat = () => mount(Chat, {
  global: {
    components: {
      ElScrollbar: ElScrollbarStub,
      ElSwitch: ElSwitchStub,
      ElSelect: ElSelectStub,
      ElOption: ElOptionStub,
      ElButton: ElButtonStub,
      ElInput: ElInputStub,
      ElTag: ElTagStub,
      ElTooltip: ElTooltipStub,
    },
    directives: {
      loading: {
        mounted() {},
        updated() {},
      },
    },
  },
})

const settle = async () => {
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

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  vi.mocked(LoadAppConfig).mockReset()
  vi.mocked(requestAiModelList).mockReset()
  vi.mocked(requestChatSessionList).mockReset()
  vi.mocked(requestChatSessionDetail).mockReset()
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)

  vi.mocked(LoadAppConfig).mockResolvedValue(baseConfig())
  vi.mocked(requestAiModelList).mockResolvedValue(baseModels())
  vi.mocked(requestChatSessionList).mockResolvedValue([] as ChatSessionSummary[])
  vi.mocked(requestChatSessionDetail).mockResolvedValue({ sessionId: '', messages: [] })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Chat.vue', () => {
  it('支持单模型 AI 流式对话并展示统计信息', async () => {
    fetchMock.mockResolvedValueOnce(createNdjsonResponse([
      { model: 'gpt-4o', content: '你好，', firstTokenMs: 120 },
      {
        model: 'gpt-4o',
        content: '我是测试助手。',
        usage: { promptTokens: 4, completionTokens: 6, totalTokens: 10 },
        durationMs: 800,
        firstTokenMs: 120,
        done: true,
      },
    ]))

    const wrapper = mountChat()
    await settle()

    await wrapper.find('textarea').setValue('你好')
    await findButtonByText(wrapper, '发送').trigger('click')
    await settle()
    await settle()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [, requestInit] = fetchMock.mock.calls[0]
    const payload = JSON.parse(String(requestInit?.body))
    expect(payload.model).toBe('gpt-4o')
    expect(payload.mode).toBe('single')
    expect(payload.stream).toBe(true)
    expect(wrapper.text()).toContain('你好')
    expect(wrapper.text()).toContain('我是测试助手。')
    expect(wrapper.text()).toContain('tokens: 10')
    expect(wrapper.text()).toContain('耗时: 0.800s')

    wrapper.unmount()
  })

  it('单模型 AI 对话请求失败时会展示错误信息', async () => {
    fetchMock.mockRejectedValueOnce(new Error('InvalidEndpointOrModel.NotFound'))

    const wrapper = mountChat()
    await settle()

    await wrapper.find('textarea').setValue('测试失败路径')
    await findButtonByText(wrapper, '发送').trigger('click')
    await settle()
    await settle()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('测试失败路径')
    expect(wrapper.text()).toContain('请求失败: InvalidEndpointOrModel.NotFound')

    wrapper.unmount()
  })

  it('支持多模型对比并行回答', async () => {
    fetchMock
      .mockResolvedValueOnce(createNdjsonResponse([
        { model: 'gpt-4o', content: '模型 A 回答', done: true },
      ]))
      .mockResolvedValueOnce(createNdjsonResponse([
        { model: 'deepseek-r1', content: '模型 B 回答', done: true },
      ]))

    const wrapper = mountChat()
    await settle()

    await wrapper.find('[data-test="compare-switch"]').setValue(true)
    await settle()

    const compareSelect = wrapper.find('select[multiple]')
    await compareSelect.setValue(['gpt-4o', 'deepseek-r1'])
    await settle()

    await wrapper.find('textarea').setValue('请比较回答')
    await findButtonByText(wrapper, '发送并对比').trigger('click')
    await settle()
    await settle()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    const firstPayload = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))
    const secondPayload = JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body))
    expect(firstPayload.mode).toBe('compare')
    expect(secondPayload.mode).toBe('compare')
    expect([firstPayload.model, secondPayload.model]).toEqual(['gpt-4o', 'deepseek-r1'])
    expect(wrapper.text()).toContain('请比较回答')
    expect(wrapper.text()).toContain('模型 A 回答')
    expect(wrapper.text()).toContain('模型 B 回答')
    expect(wrapper.text()).toContain('gpt-4o')
    expect(wrapper.text()).toContain('deepseek-r1')

    wrapper.unmount()
  })

  it('多模型对比时支持混合成功与失败结果', async () => {
    fetchMock
      .mockResolvedValueOnce(createNdjsonResponse([
        { model: 'gpt-4o', content: '模型 A 成功回答', done: true },
      ]))
      .mockRejectedValueOnce(new Error('InvalidEndpointOrModel.NotFound'))

    const wrapper = mountChat()
    await settle()

    await wrapper.find('[data-test="compare-switch"]').setValue(true)
    await settle()

    const compareSelect = wrapper.find('select[multiple]')
    await compareSelect.setValue(['gpt-4o', 'deepseek-r1'])
    await settle()

    await wrapper.find('textarea').setValue('混合结果测试')
    await findButtonByText(wrapper, '发送并对比').trigger('click')
    await settle()
    await settle()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('混合结果测试')
    expect(wrapper.text()).toContain('模型 A 成功回答')
    expect(wrapper.text()).toContain('请求失败: InvalidEndpointOrModel.NotFound')
    expect(wrapper.text()).toContain('gpt-4o')
    expect(wrapper.text()).toContain('deepseek-r1')

    wrapper.unmount()
  })

  it('多模型对比最多只会请求前三个模型', async () => {
    fetchMock
      .mockResolvedValueOnce(createNdjsonResponse([{ model: 'gpt-4o', content: 'A', done: true }]))
      .mockResolvedValueOnce(createNdjsonResponse([{ model: 'deepseek-r1', content: 'B', done: true }]))
      .mockResolvedValueOnce(createNdjsonResponse([{ model: 'glm-4', content: 'C', done: true }]))

    const wrapper = mountChat()
    await settle()

    await wrapper.find('[data-test="compare-switch"]').setValue(true)
    await settle()

    const compareSelect = wrapper.find('select[multiple]')
    await compareSelect.setValue(['gpt-4o', 'deepseek-r1', 'glm-4', 'claude-3-5'])
    await settle()

    await wrapper.find('textarea').setValue('限制模型数量')
    await findButtonByText(wrapper, '发送并对比').trigger('click')
    await settle()
    await settle()

    expect(fetchMock).toHaveBeenCalledTimes(3)
    const requestedModels = fetchMock.mock.calls.map((call) => JSON.parse(String(call[1]?.body)).model)
    expect(requestedModels).toEqual(['gpt-4o', 'deepseek-r1', 'glm-4'])

    wrapper.unmount()
  })

  it('多模型对比未选择模型时会给出前置提示且不发起请求', async () => {
    vi.mocked(LoadAppConfig).mockResolvedValue({ Ai: { Model: '' } })
    vi.mocked(requestAiModelList).mockResolvedValue({
      models: [],
      fallback: true,
      message: '',
    })

    const wrapper = mountChat()
    await settle()

    await wrapper.find('[data-test="compare-switch"]').setValue(true)
    await settle()

    await wrapper.find('textarea').setValue('未选择模型')
    await findButtonByText(wrapper, '发送并对比').trigger('click')
    await settle()

    expect(fetchMock).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('请先在上方选择至少一个模型')

    wrapper.unmount()
  })
})