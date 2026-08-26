import axios from "axios";

type APIErrorPayload = {
  error?: string;
  message?: string;
};

export function apiErrorMessage(
  error: unknown,
  fallback = "操作失败，请稍后重试",
) {
  if (axios.isAxiosError<APIErrorPayload>(error)) {
    return (
      error.response?.data?.error ||
      error.response?.data?.message ||
      error.message ||
      fallback
    );
  }
  if (error instanceof Error) return error.message || fallback;
  return fallback;
}
