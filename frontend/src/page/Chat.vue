<script setup lang="ts">
import {computed, nextTick, onMounted, onUnmounted, ref, watch} from "vue";
import {ElScrollbar} from "element-plus";
import {ArrowDownBold, ArrowRightBold, Delete, Promotion, RefreshRight} from "@element-plus/icons-vue";
import DOMPurify from "dompurify";
import {marked} from "marked";
import {LoadAppConfig} from "../../bindings/ic-wails/internal/config/applicationconfigmanager";
import {AI_CHAT_STREAM_ENDPOINT} from "@/service/AiApiPaths";
import {
  getModelOptionLabel,
  getModelOptionTagText,
  getModelOptionTagType,
  getUnavailableModelHint,
  isUnavailableModelError,
  normalizeModelName,
  requestAiModelList,
  type AiModelOption,
} from "@/service/AiModelService";
import {
  requestChatSessionDetail,
  requestChatSessionList,
  type ChatHistoryMessage,
  type ChatSessionSummary,
} from "@/service/AiChatService";

interface RuntimeAppConfig {
  Ai?: {
    Model?: string;
  };
}

interface AiUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
}

interface AiModelStats {
  model: string;
  totalCalls: number;
  avgTokens: number;
  avgFirstTokenMs: number;
}

interface StreamChunk {
  sessionId: string;
  roundId?: string;
  model?: string;
  content?: string;
  reasoningContent?: string;
  usage?: AiUsage;
  durationMs?: number;
  firstTokenMs?: number;
  modelStats?: AiModelStats;
  done?: boolean;
  error?: string;
}

interface ChatMessage {
  role: "user" | "assistant" | "system";
  content: string;
  reasoning?: string;
  loading?: boolean;
  model?: string;
  usage?: AiUsage;
  durationMs?: number;
  firstTokenMs?: number;
  modelStats?: AiModelStats;
  error?: string;
}

interface CompareMessage {
  model: string;
  content: string;
  reasoning?: string;
  loading?: boolean;
  error?: string;
  usage?: AiUsage;
  durationMs?: number;
  firstTokenMs?: number;
  modelStats?: AiModelStats;
}

interface CompareRound {
  id: string;
  question: string;
  responses: Record<string, CompareMessage>;
}

interface ApiResponse<T> {
  success: boolean;
  code: number;
  data?: T;
  message: string;
}

type ChatLoadingKey = "models" | "sessionList" | "sessionDetail";

const SESSION_PANEL_COLLAPSED_STORAGE_KEY = "chat:sessions-collapsed";

const readStoredSessionsCollapsed = () => {
  if (typeof window === "undefined") {
    return false;
  }
  try {
    return window.localStorage.getItem(SESSION_PANEL_COLLAPSED_STORAGE_KEY) === "true";
  } catch {
    return false;
  }
};

const singleMessages = ref<ChatMessage[]>([]);
const compareRounds = ref<CompareRound[]>([]);
const inputText = ref("");
const sending = ref(false);
const compareMode = ref(false);
const compareLimit = 3;
const sessionLimit = 5;
const maxSingleContextMessages = 20;
const sessionPreviewLimit = 36;
const sessionsCollapsed = ref(readStoredSessionsCollapsed());

const scrollbarRef = ref<InstanceType<typeof ElScrollbar> | null>(null);
const cleanupFns = ref<Array<() => void>>([]);

const loadingCounters = ref<Record<ChatLoadingKey, number>>({
  models: 0,
  sessionList: 0,
  sessionDetail: 0,
});

const defaultModel = ref("");
const currentModel = ref("");
const availableModels = ref<AiModelOption[]>([]);
const selectedCompareModels = ref<string[]>([]);
const modelTip = ref("");
const hasRemoteModelList = ref(false);
const invalidModelHints = ref<Record<string, string>>({});

const activeSessionId = ref("");
const sessionList = ref<ChatSessionSummary[]>([]);

const modelLoading = computed(() => loadingCounters.value.models > 0);
const sessionsLoading = computed(() => loadingCounters.value.sessionList > 0);
const sessionLoading = computed(() => loadingCounters.value.sessionDetail > 0);
const chatDataLoading = computed(() => modelLoading.value || sessionsLoading.value || sessionLoading.value);
const chatLoadingText = computed(() => {
  const labels: string[] = [];
  if (sessionsLoading.value) {
    labels.push("历史记录");
  }
  if (sessionLoading.value) {
    labels.push("当前会话记录");
  }
  if (modelLoading.value) {
    labels.push("模型列表");
  }
  if (labels.length === 0) {
    return "正在加载";
  }
  return `正在加载${labels.join("、")}`;
});

const activeSessionSummary = computed(() => sessionList.value.find((item) => item.sessionId === activeSessionId.value) || null);
const inputExpanded = computed(() => /\r?\n/.test(inputText.value));

const activeSessionMetrics = computed(() => {
  if (compareMode.value || singleMessages.value.length === 0) {
    return null;
  }

  const assistantMessages = singleMessages.value.filter((message) => message.role === "assistant");
  if (assistantMessages.length === 0) {
    return null;
  }

  let promptTokens = 0;
  let completionTokens = 0;
  let totalTokens = 0;
  let durationSum = 0;
  let durationCount = 0;
  let firstTokenSum = 0;
  let firstTokenCount = 0;

  assistantMessages.forEach((message) => {
    if (message.usage) {
      promptTokens += message.usage.promptTokens || 0;
      completionTokens += message.usage.completionTokens || 0;
      totalTokens += message.usage.totalTokens || 0;
    }
    if (typeof message.durationMs === "number") {
      durationSum += message.durationMs;
      durationCount += 1;
    }
    if (typeof message.firstTokenMs === "number") {
      firstTokenSum += message.firstTokenMs;
      firstTokenCount += 1;
    }
  });

  return {
    roundCount: singleMessages.value.filter((message) => message.role === "user" && message.content.trim()).length,
    promptTokens,
    completionTokens,
    totalTokens,
    avgDurationMs: durationCount > 0 ? Math.round(durationSum / durationCount) : undefined,
    avgFirstTokenMs: firstTokenCount > 0 ? Math.round(firstTokenSum / firstTokenCount) : undefined,
  };
});

marked.setOptions({
  gfm: true,
  breaks: true,
});

const modelOptions = computed(() => {
  const merged = new Map<string, AiModelOption>();
  availableModels.value.forEach((item) => {
    const normalized = normalizeModelName(item?.value);
    if (normalized) {
      merged.set(normalized, item);
    }
  });

  [defaultModel.value, currentModel.value, ...selectedCompareModels.value]
    .map((item) => normalizeModelName(item))
    .filter((item): item is string => Boolean(item))
    .forEach((item) => {
      const invalidHint = invalidModelHints.value[item];
      const existing = merged.get(item);
      if (existing) {
        if (invalidHint) {
          merged.set(item, {
            ...existing,
            available: false,
            status: "unavailable",
            hint: invalidHint,
          });
        }
        return;
      }

      merged.set(item, {
        value: item,
        label: item,
        source: "manual",
        available: false,
        status: invalidHint ? "unavailable" : (hasRemoteModelList.value ? "manual" : "fallback"),
        hint: invalidHint || (hasRemoteModelList.value ? "该模型不在远程可用列表中" : "当前模型列表未经过远程校验"),
      });
    });

  return Array.from(merged.values());
});

const markModelUnavailable = (modelName: string, message?: string) => {
  const normalized = normalizeModelName(modelName);
  if (!normalized || !isUnavailableModelError(message)) {
    return;
  }
  invalidModelHints.value = {
    ...invalidModelHints.value,
    [normalized]: getUnavailableModelHint(message),
  };
};

const clearModelUnavailable = (modelName: string) => {
  const normalized = normalizeModelName(modelName);
  if (!normalized || !invalidModelHints.value[normalized]) {
    return;
  }
  const next = {...invalidModelHints.value};
  delete next[normalized];
  invalidModelHints.value = next;
};

const addCleanup = (off: () => void) => {
  cleanupFns.value.push(off);
};

const clearSubscriptions = () => {
  cleanupFns.value.forEach((off) => off());
  cleanupFns.value = [];
};

const removeCleanup = (off: () => void) => {
  cleanupFns.value = cleanupFns.value.filter((item) => item !== off);
};

const trackLoading = (key: ChatLoadingKey) => {
  loadingCounters.value = {
    ...loadingCounters.value,
    [key]: loadingCounters.value[key] + 1,
  };

  let finished = false;
  return () => {
    if (finished) {
      return;
    }
    finished = true;
    loadingCounters.value = {
      ...loadingCounters.value,
      [key]: Math.max(0, loadingCounters.value[key] - 1),
    };
  };
};

const makeUid = () => `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

const isAbortError = (error: unknown) => error instanceof DOMException && error.name === "AbortError";

const requestChatStream = async (
  payload: Record<string, unknown>,
  onChunk: (chunk: StreamChunk) => void,
  signal?: AbortSignal,
) => {
  const response = await fetch(AI_CHAT_STREAM_ENDPOINT, {
    method: "POST",
    headers: {
      Accept: "application/x-ndjson",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
    signal,
  });

  if (!response.ok) {
    throw new Error(`聊天接口请求失败: HTTP ${response.status}`);
  }

  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    const result = await response.json() as ApiResponse<unknown>;
    throw new Error(result.message || "聊天接口调用失败");
  }

  if (!response.body) {
    throw new Error("当前环境不支持流式响应");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const {value, done} = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), {stream: !done});

    let newlineIndex = buffer.indexOf("\n");
    while (newlineIndex >= 0) {
      const line = buffer.slice(0, newlineIndex).trim();
      buffer = buffer.slice(newlineIndex + 1);
      if (line) {
        onChunk(JSON.parse(line) as StreamChunk);
      }
      newlineIndex = buffer.indexOf("\n");
    }

    if (done) {
      break;
    }
  }

  const tail = buffer.trim();
  if (tail) {
    onChunk(JSON.parse(tail) as StreamChunk);
  }
};

const loadDefaultModel = async () => {
  const config = await LoadAppConfig() as unknown as RuntimeAppConfig | null;
  defaultModel.value = config?.Ai?.Model || "";
  if (!currentModel.value) {
    currentModel.value = defaultModel.value;
  }
};

const loadModelOptions = async (forceRefresh = false) => {
  const finishLoading = trackLoading("models");
  modelTip.value = "";
  try {
    const result = await requestAiModelList(forceRefresh);
    availableModels.value = result.models;
    hasRemoteModelList.value = !result.fallback;
    modelTip.value = result.message || "";
    result.models.forEach((item) => {
      if (item.available) {
        clearModelUnavailable(item.value);
      }
    });
  } catch (error: any) {
    hasRemoteModelList.value = false;
    modelTip.value = error?.message || "模型列表获取失败，可手动输入模型";
  } finally {
    finishLoading();
  }
};

const loadChatSessionSummaries = async () => {
  const finishLoading = trackLoading("sessionList");
  try {
    const result = await requestChatSessionList(sessionLimit);
    sessionList.value = result;
    return result;
  } catch {
    sessionList.value = [];
    return [];
  } finally {
    finishLoading();
  }
};

const scrollToBottom = async () => {
  await nextTick();
  scrollbarRef.value?.setScrollTop(99999);
};

const formatNumber = (value: number | undefined, digits = 2) => {
  if (value === undefined || Number.isNaN(value)) {
    return "-";
  }
  return Number(value).toFixed(digits);
};

const formatSecondsFromMs = (value: number | undefined) => {
  if (value === undefined || Number.isNaN(value)) {
    return "-";
  }
  return `${(Number(value) / 1000).toFixed(3)}s`;
};

const renderMarkdown = (content: string) => {
  const normalized = String(content || "");
  const html = marked.parse(normalized) as string;
  return DOMPurify.sanitize(html);
};

const normalizeSessionSnippet = (content: string) => {
  const text = String(content || "")
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/!\[[^\]]*\]\([^)]*\)/g, " ")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/[>#*_~|-]/g, " ")
    .replace(/\s+/g, " ")
    .trim();

  if (text.length <= sessionPreviewLimit) {
    return text;
  }
  return `${text.slice(0, sessionPreviewLimit)}...`;
};

const upsertSessionSummary = (summary: ChatSessionSummary) => {
  sessionList.value = [summary, ...sessionList.value.filter((item) => item.sessionId !== summary.sessionId)].slice(0, sessionLimit);
};

const buildLocalSessionSummary = (): ChatSessionSummary | null => {
  if (compareMode.value || !activeSessionId.value) {
    return null;
  }

  const userMessages = singleMessages.value.filter((message) => message.role === "user" && message.content.trim());
  if (userMessages.length === 0) {
    return null;
  }

  const assistantMessages = singleMessages.value.filter((message) => message.role === "assistant" && (message.content.trim() || message.error));
  const lastAssistantMessage = assistantMessages[assistantMessages.length - 1];
  const lastRelevantMessage = [...singleMessages.value].reverse().find((message) => message.content.trim() || message.error);

  let promptTokens = 0;
  let completionTokens = 0;
  let totalTokens = 0;
  let durationSum = 0;
  let durationCount = 0;
  let firstTokenSum = 0;
  let firstTokenCount = 0;

  assistantMessages.forEach((message) => {
    if (message.usage) {
      promptTokens += message.usage.promptTokens || 0;
      completionTokens += message.usage.completionTokens || 0;
      totalTokens += message.usage.totalTokens || 0;
    }
    if (typeof message.durationMs === "number") {
      durationSum += message.durationMs;
      durationCount += 1;
    }
    if (typeof message.firstTokenMs === "number") {
      firstTokenSum += message.firstTokenMs;
      firstTokenCount += 1;
    }
  });

  const title = normalizeSessionSnippet(userMessages[0]?.content || "") || "未命名会话";
  const preview = normalizeSessionSnippet(lastAssistantMessage?.content || lastAssistantMessage?.error || lastRelevantMessage?.content || "");
  const model = lastAssistantMessage?.model || currentModel.value || defaultModel.value || "";

  return {
    sessionId: activeSessionId.value,
    title,
    preview,
    model,
    roundCount: userMessages.length,
    promptTokens,
    completionTokens,
    totalTokens,
    avgDurationMs: durationCount > 0 ? Math.round(durationSum / durationCount) : 0,
    avgFirstTokenMs: firstTokenCount > 0 ? Math.round(firstTokenSum / firstTokenCount) : 0,
    lastActiveTime: new Date().toISOString(),
  };
};

const persistCurrentSessionSummary = () => {
  const summary = buildLocalSessionSummary();
  if (!summary) {
    return;
  }
  upsertSessionSummary(summary);
  void refreshSessionSummaries();
};

const buildMetricsText = (payload: {
  usage?: AiUsage;
  firstTokenMs?: number;
  durationMs?: number;
  modelStats?: AiModelStats;
}) => {
  const sections: string[] = [];
  if (payload.usage) {
    sections.push(`tokens: ${payload.usage.totalTokens} (P${payload.usage.promptTokens}/C${payload.usage.completionTokens})`);
  }
  if (payload.firstTokenMs !== undefined) {
    sections.push(`首 token: ${formatSecondsFromMs(payload.firstTokenMs)}`);
  }
  if (payload.durationMs !== undefined) {
    sections.push(`耗时: ${formatSecondsFromMs(payload.durationMs)}`);
  }
  if (payload.modelStats) {
    sections.push(`平均 tokens: ${formatNumber(payload.modelStats.avgTokens)}`);
    sections.push(`平均首 token: ${formatSecondsFromMs(payload.modelStats.avgFirstTokenMs)}`);
  }
  return sections.join(" | ");
};

const getTargetModels = () => {
  if (compareMode.value) {
    const unique = Array.from(new Set(selectedCompareModels.value.map((item) => item.trim()).filter(Boolean)));
    if (unique.length === 0) {
      const fallback = (currentModel.value || defaultModel.value || "").trim();
      return fallback ? [fallback] : [];
    }
    return unique.slice(0, compareLimit);
  }

  const single = (currentModel.value || defaultModel.value || "").trim();
  return single ? [single] : [];
};

const buildSingleContextMessages = () => {
  const filtered = singleMessages.value
    .filter((message) => !message.loading && (message.role === "user" || message.role === "assistant"))
    .map((message) => ({role: message.role, content: message.content}));

  if (filtered.length <= maxSingleContextMessages + 1) {
    return filtered;
  }

  const latest = filtered[filtered.length - 1];
  const previous = filtered.slice(0, -1).slice(-maxSingleContextMessages);
  return [...previous, latest];
};

const buildSingleRequestPayload = (modelName: string, roundId: string, sessionId: string) => ({
  messages: buildSingleContextMessages(),
  model: modelName,
  roundId,
  sessionId,
  mode: "single",
});

const updateCompareResponse = (roundId: string, modelName: string, patch: Partial<CompareMessage>) => {
  const round = compareRounds.value.find((item) => item.id === roundId);
  if (!round) {
    return;
  }
  const existing = round.responses[modelName] || {model: modelName, content: "", loading: true};
  round.responses[modelName] = {
    ...existing,
    ...patch,
  };
};

const mapHistoryMessage = (message: ChatHistoryMessage): ChatMessage => ({
  role: message.role,
  content: message.content,
  reasoning: message.reasoningContent,
  model: message.model,
  usage: message.usage,
  durationMs: message.durationMs,
  firstTokenMs: message.firstTokenMs,
  modelStats: message.modelStats,
  error: message.error,
});

const inferModelFromMessages = (messages: ChatMessage[]) => {
  for (let idx = messages.length - 1; idx >= 0; idx -= 1) {
    if (messages[idx].role === "assistant" && messages[idx].model) {
      return messages[idx].model || "";
    }
  }
  return "";
};

const selectSession = async (sessionId: string) => {
  if (!sessionId || sending.value) {
    return;
  }

  const finishLoading = trackLoading("sessionDetail");
  compareMode.value = false;
  compareRounds.value = [];

  try {
    const detail = await requestChatSessionDetail(sessionId);
    const messages = Array.isArray(detail.messages) ? detail.messages.map(mapHistoryMessage) : [];
    singleMessages.value = messages;
    activeSessionId.value = detail.sessionId;

    const inferredModel = inferModelFromMessages(messages);
    if (inferredModel) {
      currentModel.value = inferredModel;
    } else if (!currentModel.value) {
      currentModel.value = defaultModel.value;
    }

    await scrollToBottom();
  } finally {
    finishLoading();
  }
};

const refreshSessionSummaries = async () => {
  try {
    sessionList.value = await requestChatSessionList(sessionLimit);
  } catch {
    // ignore refresh failures and keep the in-memory conversation available
  }
};

const sendSingleStream = async (text: string, modelName: string) => {
  const sessionId = activeSessionId.value || makeUid();
  activeSessionId.value = sessionId;

  singleMessages.value.push({role: "user", content: text});
  inputText.value = "";
  await scrollToBottom();

  const assistantIdx = singleMessages.value.length;
  singleMessages.value.push({
    role: "assistant",
    content: "",
    loading: true,
    model: modelName,
  });
  await scrollToBottom();

  const roundId = makeUid();
  const controller = new AbortController();
  const cleanup = () => controller.abort();
  addCleanup(cleanup);

  try {
    let fullContent = "";
    let fullReasoning = "";

    await requestChatStream({
      ...buildSingleRequestPayload(modelName, roundId, sessionId),
      stream: true,
    }, (chunk) => {
      if (chunk.error) {
        markModelUnavailable(chunk.model || modelName, chunk.error);
        singleMessages.value[assistantIdx] = {
          role: "assistant",
          content: fullContent || `错误: ${chunk.error}`,
          reasoning: fullReasoning || undefined,
          model: chunk.model || modelName,
          usage: chunk.usage,
          firstTokenMs: chunk.firstTokenMs,
          durationMs: chunk.durationMs,
          modelStats: chunk.modelStats,
          error: chunk.error,
        };
        persistCurrentSessionSummary();
        sending.value = false;
        return;
      }

      if (chunk.content) {
        fullContent += chunk.content;
      }
      if (chunk.reasoningContent) {
        fullReasoning += chunk.reasoningContent;
      }

      singleMessages.value[assistantIdx] = {
        role: "assistant",
        content: fullContent,
        reasoning: fullReasoning || undefined,
        loading: !chunk.done,
        model: chunk.model || modelName,
        usage: chunk.usage,
        firstTokenMs: chunk.firstTokenMs,
        durationMs: chunk.durationMs,
        modelStats: chunk.modelStats,
      };
      void scrollToBottom();

      if (chunk.done) {
        clearModelUnavailable(chunk.model || modelName);
        persistCurrentSessionSummary();
        sending.value = false;
      }
    }, controller.signal);
  } catch (error: any) {
    if (isAbortError(error)) {
      return;
    }

    markModelUnavailable(modelName, error?.message || String(error));
    singleMessages.value[assistantIdx] = {
      role: "assistant",
      content: `请求失败: ${error?.message || error}`,
      model: modelName,
      error: error?.message || String(error),
    };
    persistCurrentSessionSummary();
    sending.value = false;
    await scrollToBottom();
  } finally {
    removeCleanup(cleanup);
  }
};

const sendCompareStream = async (text: string, targetModels: string[]) => {
  const roundId = makeUid();
  const round: CompareRound = {
    id: roundId,
    question: text,
    responses: {},
  };
  targetModels.forEach((item) => {
    round.responses[item] = {
      model: item,
      content: "",
      loading: true,
    };
  });
  compareRounds.value.push(round);
  inputText.value = "";
  await scrollToBottom();

  let pending = targetModels.length;
  const doneOne = () => {
    pending -= 1;
    if (pending <= 0) {
      sending.value = false;
    }
  };

  targetModels.forEach(async (modelName) => {
    const controller = new AbortController();
    const cleanup = () => controller.abort();
    addCleanup(cleanup);

    try {
      let fullContent = "";
      let fullReasoning = "";

      await requestChatStream({
        messages: [{role: "user", content: text}],
        model: modelName,
        stream: true,
        roundId,
        mode: "compare",
      }, (chunk) => {
        if (chunk.error) {
          markModelUnavailable(chunk.model || modelName, chunk.error);
          updateCompareResponse(roundId, modelName, {
            loading: false,
            error: chunk.error,
            content: fullContent || `错误: ${chunk.error}`,
            reasoning: fullReasoning || undefined,
            usage: chunk.usage,
            firstTokenMs: chunk.firstTokenMs,
            durationMs: chunk.durationMs,
            modelStats: chunk.modelStats,
          });
          doneOne();
          return;
        }

        if (chunk.content) {
          fullContent += chunk.content;
        }
        if (chunk.reasoningContent) {
          fullReasoning += chunk.reasoningContent;
        }

        updateCompareResponse(roundId, modelName, {
          loading: !chunk.done,
          content: fullContent,
          reasoning: fullReasoning || undefined,
          usage: chunk.usage,
          firstTokenMs: chunk.firstTokenMs,
          durationMs: chunk.durationMs,
          modelStats: chunk.modelStats,
        });
        void scrollToBottom();

        if (chunk.done) {
          clearModelUnavailable(chunk.model || modelName);
          doneOne();
        }
      }, controller.signal);
    } catch (error: any) {
      if (isAbortError(error)) {
        return;
      }
      markModelUnavailable(modelName, error?.message || String(error));
      updateCompareResponse(roundId, modelName, {
        loading: false,
        error: error?.message || String(error),
        content: `请求失败: ${error?.message || error}`,
      });
      doneOne();
    } finally {
      removeCleanup(cleanup);
    }
  });
};

const sendMessageStream = async () => {
  const text = inputText.value.trim();
  if (!text || sending.value || sessionLoading.value) {
    return;
  }

  const targets = getTargetModels();
  if (targets.length === 0) {
    if (compareMode.value) {
      compareRounds.value.push({
        id: makeUid(),
        question: text,
        responses: {
          "未指定模型": {
            model: "未指定模型",
            content: "请先在上方选择至少一个模型",
            loading: false,
            error: "未指定模型",
          },
        },
      });
      inputText.value = "";
    } else {
      singleMessages.value.push({role: "assistant", content: "请先在上方选择模型", model: "未指定模型"});
    }
    return;
  }

  sending.value = true;

  if (compareMode.value) {
    await sendCompareStream(text, targets);
    return;
  }

  await sendSingleStream(text, targets[0]);
};

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault();
    void sendMessageStream();
  }
};

const createNewConversation = async () => {
  persistCurrentSessionSummary();
  clearSubscriptions();
  sending.value = false;
  compareMode.value = false;
  singleMessages.value = [];
  compareRounds.value = [];
  inputText.value = "";
  activeSessionId.value = "";
  selectedCompareModels.value = [];
  await loadDefaultModel();
  currentModel.value = defaultModel.value;
};

const clearMessages = () => {
  if (compareMode.value) {
    compareRounds.value = [];
    return;
  }
  persistCurrentSessionSummary();
  singleMessages.value = [];
  activeSessionId.value = "";
};

const toggleSessionsCollapsed = () => {
  sessionsCollapsed.value = !sessionsCollapsed.value;
};

const getRoundResponses = (round: CompareRound) => Object.values(round.responses);

const formatSessionTime = (value: string) => {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return new Intl.DateTimeFormat("zh-CN", {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
};

const buildSessionSummaryMetrics = (session: ChatSessionSummary) => {
  const parts: string[] = [];
  if (session.totalTokens > 0) {
    parts.push(`总 tokens ${session.totalTokens}`);
  }
  if (session.avgDurationMs > 0) {
    parts.push(`平均响应 ${formatSecondsFromMs(session.avgDurationMs)}`);
  }
  if (session.avgFirstTokenMs > 0) {
    parts.push(`平均首 token ${formatSecondsFromMs(session.avgFirstTokenMs)}`);
  }
  return parts;
};

watch(sessionsCollapsed, (collapsed) => {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(SESSION_PANEL_COLLAPSED_STORAGE_KEY, String(collapsed));
  } catch {
    // ignore storage failures
  }
});

watch(compareMode, (enabled) => {
  if (!enabled) {
    return;
  }
  if (selectedCompareModels.value.length === 0) {
    const fallback = (currentModel.value || defaultModel.value || "").trim();
    if (fallback) {
      selectedCompareModels.value = [fallback];
    }
  }
});

watch(selectedCompareModels, (current) => {
  if (current.length > compareLimit) {
    selectedCompareModels.value = current.slice(0, compareLimit);
  }
});

onMounted(async () => {
  const [defaultModelResult, , sessionResult] = await Promise.allSettled([
    loadDefaultModel(),
    loadModelOptions(),
    loadChatSessionSummaries(),
  ]);

  if (defaultModelResult.status === "rejected") {
    defaultModel.value = "";
  }

  if (sessionResult.status === "fulfilled" && !activeSessionId.value && sessionResult.value.length > 0) {
    void selectSession(sessionResult.value[0].sessionId);
  }

  if (!activeSessionId.value && !currentModel.value) {
    currentModel.value = defaultModel.value;
  }
});

onUnmounted(() => {
  clearSubscriptions();
});
</script>

<template>
  <div
    class="chat-container"
    v-loading="chatDataLoading"
    :element-loading-text="chatLoadingText"
    element-loading-background="color-mix(in srgb, var(--bg-card) 72%, transparent)"
  >
    <div class="chat-toolbar">
      <div class="chat-toolbar__meta">
        <div>
          <h1>智能助理</h1>
          <p v-if="!compareMode && activeSessionSummary">
            当前会话：{{ activeSessionSummary.title }}
          </p>
          <p v-else>
            支持单模型连续对话，或最多 3 个模型并行回答同一问题进行对比。
          </p>
        </div>
      </div>
      <div class="chat-toolbar__controls">
        <el-switch v-model="compareMode" active-text="多模型对比" inline-prompt />

        <el-select
          v-if="!compareMode"
          v-model="currentModel"
          filterable
          allow-create
          default-first-option
          clearable
          :loading="modelLoading"
          placeholder="当前会话模型"
          style="width: 260px"
        >
          <el-option
            v-for="item in modelOptions"
            :key="item.value"
            :label="getModelOptionLabel(item)"
            :value="item.value"
          >
            <div class="model-option">
              <span class="model-option__label">{{ item.label }}</span>
              <div class="model-option__meta">
                <el-tag v-if="getModelOptionTagText(item)" :type="getModelOptionTagType(item)" effect="plain" size="small">
                  {{ getModelOptionTagText(item) }}
                </el-tag>
                <span v-if="item.hint" class="model-option__hint">{{ item.hint }}</span>
              </div>
            </div>
          </el-option>
        </el-select>

        <el-select
          v-else
          v-model="selectedCompareModels"
          multiple
          filterable
          allow-create
          default-first-option
          collapse-tags
          collapse-tags-tooltip
          :max-collapse-tags="2"
          :loading="modelLoading"
          placeholder="选择最多 3 个对比模型"
          style="width: 320px"
        >
          <el-option
            v-for="item in modelOptions"
            :key="item.value"
            :label="getModelOptionLabel(item)"
            :value="item.value"
          >
            <div class="model-option">
              <span class="model-option__label">{{ item.label }}</span>
              <div class="model-option__meta">
                <el-tag v-if="getModelOptionTagText(item)" :type="getModelOptionTagType(item)" effect="plain" size="small">
                  {{ getModelOptionTagText(item) }}
                </el-tag>
                <span v-if="item.hint" class="model-option__hint">{{ item.hint }}</span>
              </div>
            </div>
          </el-option>
        </el-select>

        <el-button link type="primary" :loading="modelLoading" @click="loadModelOptions(true)">
          刷新模型
        </el-button>

        <el-tooltip :content="sessionsCollapsed ? '展开历史会话栏' : '折叠历史会话栏'" placement="bottom">
          <el-button
            circle
            text
            class="chat-toolbar__icon-button"
            :class="{ 'is-collapsed': sessionsCollapsed }"
            :icon="sessionsCollapsed ? ArrowRightBold : ArrowDownBold"
            @click="toggleSessionsCollapsed"
          />
        </el-tooltip>

        <el-button :icon="RefreshRight" @click="createNewConversation" :disabled="sending || sessionLoading">
          新建会话
        </el-button>
      </div>

      <div v-if="activeSessionMetrics && !compareMode" class="chat-toolbar__session-summary">
        <span class="chat-toolbar__session-pill">输入 {{ activeSessionMetrics.promptTokens }}</span>
        <span class="chat-toolbar__session-pill">输出 {{ activeSessionMetrics.completionTokens }}</span>
        <span class="chat-toolbar__session-pill">总 tokens {{ activeSessionMetrics.totalTokens }}</span>
        <span class="chat-toolbar__session-pill">平均响应 {{ formatSecondsFromMs(activeSessionMetrics.avgDurationMs) }}</span>
        <span class="chat-toolbar__session-pill">平均首 token {{ formatSecondsFromMs(activeSessionMetrics.avgFirstTokenMs) }}</span>
        <span class="chat-toolbar__session-pill">{{ activeSessionMetrics.roundCount }} 轮</span>
      </div>
    </div>

    <div v-if="modelTip" class="chat-model-tip">{{ modelTip }}</div>

    <div class="chat-body" :class="{ 'chat-body--sessions-collapsed': sessionsCollapsed }">
      <aside class="chat-sessions" :class="{ 'is-collapsed': sessionsCollapsed }">
        <div class="chat-sessions__inner">
          <div class="chat-sessions__header">
            <div>
              <strong>历史会话</strong>
              <span>默认展示最近 5 个</span>
            </div>
            <el-button text type="primary" @click="createNewConversation" :disabled="sending || sessionLoading">
              新会话
            </el-button>
          </div>

          <div v-if="!sessionsLoading && sessionList.length === 0" class="chat-sessions__empty">
            暂无历史会话
          </div>
          <div v-else class="chat-sessions__list">
            <button
              v-for="session in sessionList"
              :key="session.sessionId"
              type="button"
              class="chat-session-card"
              :class="{ 'is-active': session.sessionId === activeSessionId }"
              @click="selectSession(session.sessionId)"
            >
              <div class="chat-session-card__head">
                <strong>{{ session.title }}</strong>
                <span>{{ formatSessionTime(session.lastActiveTime) }}</span>
              </div>
              <p>{{ session.preview || "暂无摘要" }}</p>
              <div v-if="buildSessionSummaryMetrics(session).length > 0" class="chat-session-card__stats">
                <span
                  v-for="metric in buildSessionSummaryMetrics(session)"
                  :key="metric"
                  class="chat-session-card__stat"
                >
                  {{ metric }}
                </span>
              </div>
              <div class="chat-session-card__meta">
                <span>{{ session.model || "未记录模型" }}</span>
                <span>{{ session.roundCount }} 轮</span>
              </div>
            </button>
          </div>
        </div>
      </aside>

      <div class="chat-panel">
        <el-scrollbar ref="scrollbarRef" class="chat-messages">
          <div v-if="!compareMode && !sessionLoading && singleMessages.length === 0" class="chat-empty">
            <div class="chat-empty-card">
              <strong>开始与 AI 对话吧</strong>
              <span>当前新会话默认模型：{{ defaultModel || "未配置" }}</span>
            </div>
          </div>

          <template v-if="!compareMode">
            <div
              v-for="(msg, idx) in singleMessages"
              :key="idx"
              class="chat-message"
              :class="msg.role"
            >
              <div class="message-role">{{ msg.role === "user" ? "我" : "AI" }}</div>
              <div class="message-body">
                <div v-if="msg.reasoning" class="message-reasoning">
                  <details :open="Boolean(msg.loading)">
                    <summary>{{ msg.loading ? '思考中' : '思考过程' }}</summary>
                    <pre>{{ msg.reasoning }}</pre>
                  </details>
                </div>
                <div class="message-content">
                  <span v-if="msg.loading && !msg.content" class="typing-indicator">思考中...</span>
                  <div
                    v-else-if="msg.role === 'assistant'"
                    class="message-markdown markdown-body"
                    v-html="renderMarkdown(msg.content)"
                  />
                  <pre v-else>{{ msg.content }}</pre>
                </div>
                <div v-if="msg.role === 'assistant'" class="message-metrics">
                  <span>模型: {{ msg.model || "未知" }}</span>
                  <span v-if="buildMetricsText(msg)">{{ buildMetricsText(msg) }}</span>
                </div>
              </div>
            </div>
          </template>

          <div v-if="compareMode && compareRounds.length === 0" class="chat-empty">
            <div class="chat-empty-card">
              <strong>对比模式已开启</strong>
              <span>输入同一问题后，模型会并行回答并显示统计指标。</span>
            </div>
          </div>

          <div v-if="compareMode" class="compare-list">
            <div v-for="round in compareRounds" :key="round.id" class="compare-round">
              <div class="compare-round__question">
                <span class="compare-round__label">问题</span>
                <p>{{ round.question }}</p>
              </div>
              <div class="compare-grid">
                <div
                  v-for="message in getRoundResponses(round)"
                  :key="message.model"
                  class="compare-card"
                >
                  <div class="compare-card__header">
                    <strong>{{ message.model }}</strong>
                    <el-tag v-if="message.loading" type="warning" effect="plain" size="small">生成中</el-tag>
                    <el-tag v-else-if="message.error" type="danger" effect="plain" size="small">失败</el-tag>
                    <el-tag v-else type="success" effect="plain" size="small">完成</el-tag>
                  </div>
                  <div v-if="buildMetricsText(message)" class="compare-card__metrics">{{ buildMetricsText(message) }}</div>
                  <div v-if="message.reasoning" class="message-reasoning">
                    <details :open="Boolean(message.loading)">
                      <summary>{{ message.loading ? '思考中' : '思考过程' }}</summary>
                      <pre>{{ message.reasoning }}</pre>
                    </details>
                  </div>
                  <div class="compare-card__content">
                    <span v-if="message.loading && !message.content" class="typing-indicator">思考中...</span>
                    <div
                      v-else
                      class="compare-card__markdown markdown-body"
                      v-html="renderMarkdown(message.content)"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-scrollbar>

        <div class="chat-input-area">
          <div class="chat-input-actions">
            <el-tag v-if="!compareMode" type="info" effect="plain">
              当前模型：{{ currentModel || defaultModel || "未指定" }}
            </el-tag>
            <el-tag v-else type="info" effect="plain">
              对比模型：{{ getTargetModels().join("、") || "未指定" }}
            </el-tag>
            <el-button
              v-if="!inputExpanded"
              size="small"
              @click="clearMessages"
              :disabled="sending || sessionLoading"
              text
              :icon="Delete"
            >
              清空当前消息
            </el-button>
          </div>
          <div class="chat-input-row" :class="{ 'is-expanded': inputExpanded }">
            <el-input
              v-model="inputText"
              type="textarea"
              :autosize="inputExpanded ? { minRows: 3, maxRows: 5 } : { minRows: 1, maxRows: 1 }"
              placeholder="输入消息，Enter 发送，Shift+Enter 换行"
              :disabled="sending || sessionLoading"
              @keydown="handleKeydown"
            />
            <div class="chat-input-buttons" :class="{ 'is-expanded': inputExpanded }">
              <el-button
                v-if="inputExpanded"
                @click="clearMessages"
                :disabled="sending || sessionLoading"
                class="chat-clear-button"
                :icon="Delete"
              >
                清空当前消息
              </el-button>
              <el-button
                type="primary"
                :icon="Promotion"
                :loading="sending"
                @click="sendMessageStream"
                class="send-button"
              >
                {{ compareMode ? "发送并对比" : "发送" }}
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  border-radius: var(--radius-2xl);
  background: var(--bg-card-muted);
  border: 1px solid var(--border-strong);
  box-shadow: var(--shadow-soft);
  backdrop-filter: var(--glass-blur);
}

.chat-toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
  gap: var(--space-4);
  padding: 18px 28px;
  border-bottom: 1px solid var(--border-subtle);
  cursor: default;
}

.chat-toolbar__meta h1 {
  margin: 0 0 6px;
  font-size: 42px;
  line-height: var(--line-height-compact);
  color: var(--text-primary);
}

.chat-toolbar__meta p {
  margin: 0;
  color: var(--text-tertiary);
  font-size: 12px;
  line-height: 1.45;
}

.chat-toolbar__meta,
.chat-toolbar__meta h1,
.chat-toolbar__meta p,
.chat-toolbar__session-summary,
.chat-toolbar__session-pill,
.chat-model-tip {
  user-select: none;
  cursor: default;
}

.chat-toolbar__session-summary {
  display: flex;
  grid-column: 1 / -1;
  flex-wrap: wrap;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.chat-toolbar__session-pill {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid var(--border-subtle);
  background: color-mix(in srgb, var(--bg-soft) 68%, transparent);
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1;
}

.chat-toolbar__controls {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.chat-toolbar__controls :deep(.el-select__wrapper),
.chat-toolbar__controls :deep(.el-input__wrapper) {
  min-height: 40px;
}

.chat-toolbar__controls :deep(.el-switch) {
  --el-switch-height: 24px;
  --el-switch-button-size: 18px;
}

.chat-toolbar__controls :deep(.el-button) {
  min-height: 40px;
  padding-top: 0;
  padding-bottom: 0;
}

.chat-toolbar__icon-button {
  width: var(--sidebar-button-size);
  height: var(--sidebar-button-size);
  border-radius: var(--radius-md);
  transition: background-color 0.24s ease, color 0.24s ease, transform 0.28s ease;
}

.chat-toolbar__icon-button:hover {
  background: var(--color-primary-soft);
}

.chat-toolbar__icon-button.is-collapsed {
  background: color-mix(in srgb, var(--color-primary-soft) 72%, transparent);
  color: var(--color-primary);
}

.chat-model-tip {
  padding: 6px 28px;
  font-size: 11px;
  color: var(--text-muted);
  border-bottom: 1px dashed var(--border-subtle);
}

.chat-toolbar__controls,
.chat-toolbar__controls * {
  user-select: auto;
}

.chat-body {
  display: flex;
  flex: 1;
  min-height: 0;
}

.chat-body--sessions-collapsed .chat-panel {
  width: 100%;
}

.chat-sessions {
  width: 320px;
  flex-shrink: 0;
  overflow: hidden;
  padding: var(--space-5);
  border-right: 1px solid var(--border-subtle);
  background: color-mix(in srgb, var(--bg-soft) 78%, transparent);
  transition: width 0.28s ease, padding 0.28s ease, border-color 0.28s ease, background-color 0.28s ease;
}

.chat-sessions.is-collapsed {
  width: 0;
  padding-left: 0;
  padding-right: 0;
  border-right-color: transparent;
}

.chat-sessions__inner {
  width: 280px;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  transition: opacity 0.2s ease, transform 0.28s ease;
}

.chat-sessions.is-collapsed .chat-sessions__inner {
  opacity: 0;
  transform: translateX(-18px);
  pointer-events: none;
}

.chat-sessions__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
}

.chat-sessions__header strong {
  display: block;
  font-size: var(--font-size-md);
  color: var(--text-primary);
}

.chat-sessions__header span {
  display: block;
  margin-top: 4px;
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}

.chat-sessions__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 120px;
  border-radius: var(--radius-lg);
  border: 1px dashed var(--border-subtle);
  color: var(--text-muted);
  font-size: var(--font-size-sm);
  background: var(--bg-soft);
}

.chat-sessions__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.chat-session-card {
  width: 100%;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--bg-soft);
  padding: var(--space-4);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease, transform 0.2s ease, background 0.2s ease, box-shadow 0.2s ease;
}

.chat-session-card:hover {
  transform: translateY(-1px);
  border-color: var(--color-primary);
  box-shadow: 0 10px 24px color-mix(in srgb, var(--color-primary) 12%, transparent 88%);
}

.chat-session-card.is-active {
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--bg-bubble-user) 58%, var(--bg-soft) 42%);
}


.chat-session-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
}

.chat-session-card__head strong {
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-comfortable);
}

.chat-session-card__head span,
.chat-session-card__meta {
  color: var(--text-muted);
  font-size: var(--font-size-xs);
}

.chat-session-card p {
  margin: var(--space-3) 0 var(--space-4);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-comfortable);
  display: -webkit-box;
  line-clamp: 2;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.chat-session-card__stats {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-bottom: var(--space-4);
}

.chat-session-card__stat {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 var(--space-2);
  border-radius: var(--radius-sm);
  border: 1px solid color-mix(in srgb, var(--border-subtle) 82%, transparent);
  background: color-mix(in srgb, var(--bg-bubble-user) 42%, transparent);
  color: var(--text-tertiary);
  font-size: var(--font-size-xs);
  line-height: 1;
}

.chat-session-card.is-active .chat-session-card__stat {
  border-color: color-mix(in srgb, var(--color-primary) 20%, transparent);
  background: color-mix(in srgb, var(--color-primary-soft) 78%, white 22%);
  color: var(--color-primary-strong);
}

.chat-session-card__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.chat-panel {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
}

.chat-session-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

.chat-session-stats__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: var(--space-4);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--bg-soft) 76%, transparent);
}

.chat-session-stats__item span {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}

.chat-session-stats__item strong {
  font-size: var(--font-size-lg);
  color: var(--text-primary);
}

.chat-messages {
  flex: 1;
  padding: var(--space-6);
  overflow-y: auto;
}

.chat-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.chat-empty-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-7) calc(var(--space-7) + var(--space-2));
  border-radius: var(--radius-xl);
  background: var(--bg-soft);
  color: var(--text-tertiary);
  text-align: center;
}

.chat-empty-card strong {
  font-size: var(--font-size-xl);
  color: var(--text-primary);
}

.chat-message {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

.chat-message.user {
  flex-direction: row-reverse;
}

.message-role {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xs);
  font-weight: bold;
  color: #fff;
}

.chat-message.user .message-role {
  background: var(--color-primary);
}

.chat-message.assistant .message-role {
  background: var(--color-success);
}

.message-body {
  max-width: min(84%, 1040px);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.message-content {
  padding: var(--space-4) var(--space-5);
  border-radius: var(--radius-md);
  font-size: var(--font-size-md);
  line-height: var(--line-height-relaxed);
  word-break: break-word;
}

.message-content pre,
.message-reasoning pre,
.compare-card__content pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
}

.message-markdown,
.compare-card__markdown {
  min-height: 20px;
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.markdown-body :deep(*) {
  max-width: 100%;
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.markdown-body :deep(p) {
  margin: 0 0 var(--space-3);
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  margin: 0 0 var(--space-3);
  padding-left: 20px;
}

.markdown-body :deep(li + li) {
  margin-top: 4px;
}

.markdown-body :deep(pre) {
  margin: 0 0 var(--space-3);
  padding: var(--space-4);
  overflow-x: auto;
  border-radius: var(--radius-md);
  background: rgba(15, 23, 42, 0.9);
  color: #e2e8f0;
}

.markdown-body :deep(code) {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  font-size: var(--font-size-sm);
}

.markdown-body :deep(:not(pre) > code) {
  padding: 2px 6px;
  border-radius: 6px;
  background: rgba(148, 163, 184, 0.2);
}

.markdown-body :deep(blockquote) {
  margin: 0 0 var(--space-3);
  padding-left: var(--space-3);
  border-left: 3px solid var(--border-strong);
  color: var(--text-muted);
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0 0 var(--space-3);
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid var(--border-subtle);
  padding: var(--space-2) var(--space-3);
  text-align: left;
}

.markdown-body :deep(a) {
  color: var(--color-primary);
}

.chat-message.user .message-content {
  background: var(--bg-bubble-user);
  border-top-right-radius: 4px;
}

.chat-message.assistant .message-content {
  background: var(--bg-bubble-assistant);
  border-top-left-radius: 4px;
}

.message-reasoning {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}

.message-reasoning summary {
  cursor: pointer;
  user-select: none;
}

.message-metrics {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.compare-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.compare-round {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--bg-soft);
}

.compare-round__question {
  background: var(--bg-bubble-user);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4);
}

.compare-round__label {
  display: inline-block;
  margin-bottom: 6px;
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}

.compare-round__question p {
  margin: 0;
  color: var(--text-primary);
  white-space: pre-wrap;
}

.compare-grid {
  display: grid;
  gap: var(--space-3);
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  min-width: 0;
}

.compare-card {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  background: var(--bg-bubble-assistant);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
  overflow: hidden;
}

.compare-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
  min-width: 0;
}

.compare-card__header strong {
  flex: 1;
  min-width: 0;
  line-height: 1.35;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.compare-card__header :deep(.el-tag) {
  flex-shrink: 0;
}

.compare-card__metrics {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  line-height: var(--line-height-comfortable);
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.compare-card__content {
  min-width: 0;
  overflow: hidden;
}

.typing-indicator {
  color: var(--text-muted);
  animation: blink 1.2s infinite;
}

@keyframes blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

.chat-input-area {
  border-top: 1px solid var(--border-subtle);
  padding: var(--space-4) var(--space-6) var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  background: color-mix(in srgb, var(--bg-card) 88%, transparent);
}

.chat-input-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.chat-input-row {
  display: flex;
  gap: var(--space-3);
  align-items: stretch;
}

.chat-input-row.is-expanded {
  align-items: stretch;
}

.model-option {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 2px 0;
}

.model-option__label {
  color: var(--text-primary);
}

.model-option__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.model-option__hint {
  color: var(--text-muted);
  font-size: var(--font-size-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-input-row .el-input {
  flex: 1;
}

.chat-input-row :deep(.el-textarea),
.chat-input-row :deep(.el-textarea__inner),
.chat-input-row :deep(.el-textarea__wrapper) {
  height: 100%;
}

.chat-input-row :deep(.el-textarea__inner) {
  min-height: 44px !important;
  resize: none;
}

.chat-input-row.is-expanded :deep(.el-textarea__inner) {
  min-height: 96px !important;
}

.chat-input-buttons {
  width: 160px;
  display: flex;
  flex-direction: row;
  gap: var(--space-2);
  align-self: stretch;
}

.chat-input-buttons.is-expanded {
  width: 184px;
  flex-direction: column;
}

.chat-input-buttons .el-button {
  flex: 1;
  width: 100%;
  min-height: 44px;
  margin: 0;
}

.chat-clear-button {
  border-color: var(--border-subtle);
  background: color-mix(in srgb, var(--bg-soft) 72%, transparent);
  color: var(--text-secondary);
}

.send-button {
  min-width: 0;
  min-height: 44px;
}

@media (max-width: 1100px) {
  .chat-body {
    flex-direction: column;
  }

  .chat-sessions {
    width: 100%;
    padding-top: var(--space-4);
    padding-bottom: var(--space-4);
    border-right: 0;
    border-bottom: 1px solid var(--border-subtle);
  }

  .chat-sessions.is-collapsed {
    width: 100%;
    height: 0;
    padding-top: 0;
    padding-bottom: 0;
    border-bottom-color: transparent;
  }

  .chat-sessions__inner {
    width: 100%;
  }

  .chat-sessions__list {
    max-height: 220px;
  }
}

@media (max-width: 900px) {
  .chat-toolbar {
    grid-template-columns: 1fr;
  }

  .chat-toolbar__controls {
    flex-direction: column;
    align-items: stretch;
  }

  .chat-toolbar__session-summary {
    grid-column: auto;
  }

  .chat-toolbar__session-pill {
    min-height: 26px;
  }

  .message-body {
    max-width: 100%;
  }

  .chat-input-actions,
  .chat-input-row {
    flex-direction: column;
    align-items: stretch;
  }

  .chat-input-buttons {
    width: 100%;
    flex-direction: column;
  }
}
</style>
