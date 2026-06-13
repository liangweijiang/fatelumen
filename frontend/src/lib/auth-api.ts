import api from "@/lib/api/client";
import { setToken } from "@/lib/auth-storage";

export interface LoginResult {
  user_id: number;
  email: string;
  name: string;
  avatar_url: string;
  token: string;
}

function unwrap<T>(data: unknown): T {
  return ((data as Record<string, unknown>)?.data ?? data) as T;
}

export async function register(email: string, password: string, name: string): Promise<LoginResult> {
  const { data } = await api.post("/auth/register", { email, password, name });
  const result = unwrap<LoginResult>(data);
  if (result?.token) setToken(result.token);
  return result;
}

export async function login(email: string, password: string): Promise<LoginResult> {
  const { data } = await api.post("/auth/login", { email, password });
  const result = unwrap<LoginResult>(data);
  if (result?.token) setToken(result.token);
  return result;
}
