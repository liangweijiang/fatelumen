import axios from "axios";
import { getToken, removeToken } from "@/lib/auth-storage";

const AUTH_REDIRECT_KEY = "fatelumen_auth_redirecting";
const SUPPORTED_LOCALES = new Set(["en", "zh", "ja", "ko"]);

function requestHadUserToken(headers: unknown): boolean {
  if (!headers || typeof headers !== "object") return false;
  const value = "get" in headers && typeof headers.get === "function"
    ? headers.get("Authorization")
    : (headers as Record<string, unknown>).Authorization;
  return typeof value === "string" && value.startsWith("Bearer ");
}

function redirectToLogin(): void {
  if (typeof window === "undefined") return;
  removeToken();

  if (window.location.pathname === "/login") return;
  if (window.sessionStorage.getItem(AUTH_REDIRECT_KEY) === "1") return;
  window.sessionStorage.setItem(AUTH_REDIRECT_KEY, "1");

  const firstSegment = window.location.pathname.split("/").filter(Boolean)[0];
  const lang = SUPPORTED_LOCALES.has(firstSegment) ? firstSegment : "en";
  const next = `${window.location.pathname}${window.location.search}`;
  const params = new URLSearchParams({ lang, next, reason: "session-expired" });
  window.location.replace(`/login?${params.toString()}`);
}

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
});

// Request interceptor: inject JWT
api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor: handle 401 and extract errors
api.interceptors.response.use(
    (response) => {
        const body = response.data;
        if (body && typeof body === "object" && "code" in body && body.code !== 0) {
            if ((body.code === 4010 || body.code === 4011) && requestHadUserToken(response.config.headers)) {
                redirectToLogin();
            }
            return Promise.reject(new Error(body.msg || "请求失败，请稍后再试"));
        }
        return response;
    },
  (error) => {
    if (typeof window !== "undefined") {
      if (error.response?.status === 401 && requestHadUserToken(error.config?.headers)) {
        redirectToLogin();
      }
    }

    const message =
      error.response?.data?.error ||
      error.response?.data?.message ||
      error.message ||
      "网络异常，请稍后再试";

    return Promise.reject(new Error(message));
  }
);

export default api;
