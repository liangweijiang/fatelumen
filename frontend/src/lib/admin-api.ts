import api from "@/lib/api/client";

export interface Stats {
  users: { total: number; today_new: number };
  orders: { total: number; by_status: Record<string, number> };
  revenue: { total_cents: number; today_cents: number; currency: string };
  reports: { total: number; by_status: Record<string, number>; unlocked_count: number };
}

export interface AdminUserItem {
  id: number;
  email: string;
  name: string;
  role: string;
  active: boolean;
  unlimited: boolean;
  created_at: string;
}

export interface AdminUsersPage {
  items: AdminUserItem[];
  total: number;
  page: number;
  page_size: number;
}

function unwrap<T>(data: unknown): T {
  return ((data as Record<string, unknown>)?.data ?? data) as T;
}

export async function fetchStats(): Promise<Stats> {
  const { data } = await api.get("/admin/stats");
  return unwrap<Stats>(data);
}

export async function fetchUsers(keyword = "", page = 1, pageSize = 20): Promise<AdminUsersPage> {
  const { data } = await api.get("/admin/users", { params: { keyword, page, page_size: pageSize } });
  return unwrap<AdminUsersPage>(data);
}

export async function setUserUnlimited(id: number, unlimited: boolean): Promise<void> {
  await api.patch(`/admin/users/${id}/unlimited`, { unlimited });
}
