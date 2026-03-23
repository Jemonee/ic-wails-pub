import {AI_CHAT_SESSIONS_ENDPOINT} from "@/service/AiApiPaths"

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

export interface ChatHistoryMessage {
  role: "user" | "assistant" | "system";
  content: string;
  reasoningContent?: string;
  model?: string;
  usage?: AiUsage;
  durationMs?: number;
  firstTokenMs?: number;
  modelStats?: AiModelStats;
  error?: string;
}

export interface ChatSessionSummary {
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

interface ChatSessionDetail {
  sessionId: string;
  messages: ChatHistoryMessage[];
}

interface ApiResponse<T> {
  success: boolean;
  code: number;
  data?: T;
  message: string;
}

const parseApiResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    throw new Error(`接口请求失败: HTTP ${response.status}`);
  }
  const result = await response.json() as ApiResponse<T>;
  if (!result.success || result.data === undefined) {
    throw new Error(result.message || "接口调用失败");
  }
  return result.data;
};

export const requestChatSessionList = async (limit = 5): Promise<ChatSessionSummary[]> => {
  const response = await fetch(`${AI_CHAT_SESSIONS_ENDPOINT}?limit=${encodeURIComponent(String(limit))}`, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
  });

  return parseApiResponse<ChatSessionSummary[]>(response);
};

export const requestChatSessionDetail = async (sessionId: string): Promise<ChatSessionDetail> => {
  const response = await fetch(`${AI_CHAT_SESSIONS_ENDPOINT}/${encodeURIComponent(sessionId)}`, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
  });

  return parseApiResponse<ChatSessionDetail>(response);
};