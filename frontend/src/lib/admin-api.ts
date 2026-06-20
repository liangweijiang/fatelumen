import api from "@/lib/api/client";

export interface Me {
  id: number;
  email: string;
  name: string;
  role: string;
  unlimited: boolean;
}

export async function fetchMe(): Promise<Me> {
  const { data } = await api.get("/me");
  return unwrap<Me>(data);
}

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

// ---------- 资源驱动后台(对接 /admin/resources)----------

export interface ResourceField {
  key: string;
  label: string;
  type: string;
  enum?: { value: string; label: string }[];
  sortable?: boolean;
  filterable?: boolean;
  searchable?: boolean;
  editable?: boolean;
  hidden?: boolean;
}

export interface ResourceSchema {
  name: string;
  fields: ResourceField[];
}

export interface ResourceListResult<T = Record<string, unknown>> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export async function fetchResourceSchema(resource: string): Promise<ResourceSchema> {
  const { data } = await api.get(`/admin/resources/${resource}/_schema`);
  return unwrap<ResourceSchema>(data);
}

export async function fetchResourceList(
  resource: string,
  params: { page?: number; page_size?: number; search?: string; sort?: string } & Record<string, string | number> = {}
): Promise<ResourceListResult> {
  const { data } = await api.get(`/admin/resources/${resource}`, { params });
  return unwrap<ResourceListResult>(data);
}

export async function fetchResourceDetail(resource: string, id: string | number): Promise<Record<string, unknown>> {
  const { data } = await api.get(`/admin/resources/${resource}/${id}`);
  return unwrap<Record<string, unknown>>(data);
}
