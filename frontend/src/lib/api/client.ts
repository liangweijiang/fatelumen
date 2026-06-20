import axios from "axios";
import { getToken, removeToken } from "@/lib/auth-storage";

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
            if (typeof window !== "undefined" && body.code === 4011) {
                removeToken();
                window.location.href = "/login";
            }
            return Promise.reject(new Error(body.msg || "请求失败，请稍后再试"));
        }
        return response;
    },
  (error) => {
    if (typeof window !== "undefined") {
      if (error.response?.status === 401) {
        removeToken();
        window.location.href = "/login";
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
