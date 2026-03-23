import {AI_MODELS_ENDPOINT} from "@/service/AiApiPaths"

export interface AiModelOption {
  value: string;
  label: string;
  source?: string;
  available?: boolean;
  status?: "available" | "unavailable" | "fallback" | "manual";
  hint?: string;
}

interface AiModelListResponse {
  models: AiModelOption[];
  fallback: boolean;
  message?: string;
}

interface ApiResponse<T> {
  success: boolean;
  code: number;
  data?: T;
  message: string;
}

export const requestAiModelList = async (forceRefresh = false): Promise<AiModelListResponse> => {
  const query = forceRefresh ? "?refresh=1" : "";
  const response = await fetch(`${AI_MODELS_ENDPOINT}${query}`, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    throw new Error(`模型列表接口请求失败: HTTP ${response.status}`);
  }

  const result = await response.json() as ApiResponse<AiModelListResponse>;
  if (!result.success || !result.data) {
    throw new Error(result.message || "模型列表获取失败");
  }

  return {
    models: Array.isArray(result.data.models) ? result.data.models : [],
    fallback: result.data.fallback,
    message: result.data.message || "",
  };
};

export const normalizeModelName = (modelName?: string | null) => modelName?.trim() || "";

export const isUnavailableModelError = (message?: string | null) => {
  const text = (message || "").trim();
  if (!text) {
    return false;
  }
  return /InvalidEndpointOrModel\.NotFound|does not exist or you do not have access|model or endpoint/i.test(text);
};

export const getUnavailableModelHint = (message?: string | null) => {
  if (isUnavailableModelError(message)) {
    return "该模型当前不可用，或当前账号无权限访问";
  }
  return (message || "").trim() || "该模型暂时不可用";
};

export const getModelOptionLabel = (item: AiModelOption) => {
  if (item.status === "unavailable") {
    return `${item.label}（不可用）`;
  }
  if (item.status === "fallback") {
    return `${item.label}（未校验）`;
  }
  if (item.status === "manual") {
    return `${item.label}（手动输入）`;
  }
  return item.label;
};

export const getModelOptionTagText = (item: AiModelOption) => {
  if (item.status === "unavailable") {
    return "不可用";
  }
  if (item.status === "fallback") {
    return "未校验";
  }
  if (item.status === "manual") {
    return "手动输入";
  }
  return "";
};

export const getModelOptionTagType = (item: AiModelOption) => {
  if (item.status === "unavailable") {
    return "danger";
  }
  if (item.status === "fallback") {
    return "warning";
  }
  if (item.status === "manual") {
    return "info";
  }
  return "info";
};
