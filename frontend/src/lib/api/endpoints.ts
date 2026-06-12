import api from "@/lib/api/client";
import type {
  AuthProvider,
  User,
  BirthProfile,
  CreateProfilePayload,
  Chart,
  CreateChartPayload,
  Reading,
  CreateQuickReadingPayload,
  Report,
  CreateReportPayload,
  Order,
  CreateOrderPayload,
} from "@/types/api";

// ── Auth ──
export async function getAuthProviders(): Promise<AuthProvider[]> {
  const { data } = await api.get("/auth/providers");
  return data.data ?? data;
}

export async function getMe(): Promise<User> {
  const { data } = await api.get("/me");
  return data.data ?? data;
}

export async function updateMe(payload: Partial<Pick<User, "name" | "locale">>): Promise<User> {
  const { data } = await api.patch("/me", payload);
  return data.data ?? data;
}

export async function logout(): Promise<void> {
  await api.post("/auth/logout");
}

// ── Profiles ──
export async function listProfiles(): Promise<BirthProfile[]> {
  const { data } = await api.get("/profiles");
  return data.data ?? data;
}

export async function createProfile(payload: CreateProfilePayload): Promise<BirthProfile> {
  const { data } = await api.post("/profiles", payload);
  return data.data ?? data;
}

export async function getProfile(id: number): Promise<BirthProfile> {
  const { data } = await api.get(`/profiles/${id}`);
  return data.data ?? data;
}

export async function removeProfile(id: number): Promise<void> {
  await api.delete(`/profiles/${id}`);
}

// ── Charts ──
export async function createChart(payload: CreateChartPayload): Promise<Chart> {
  const { data } = await api.post("/charts", payload);
  return data.data ?? data;
}

export async function getChart(id: number): Promise<Chart> {
  const { data } = await api.get(`/charts/${id}`);
  return data.data ?? data;
}

// ── Readings ──
export async function createQuickReading(payload: CreateQuickReadingPayload): Promise<Reading> {
  const { data } = await api.post("/readings/quick", payload);
  return data.data ?? data;
}

export async function getReading(id: number): Promise<Reading> {
  const { data } = await api.get(`/readings/${id}`);
  return data.data ?? data;
}

export async function listReadings(): Promise<Reading[]> {
  const { data } = await api.get("/readings");
  return data.data ?? data;
}

// ── Reports ──
export async function createReport(payload: CreateReportPayload): Promise<Report> {
  const { data } = await api.post("/reports", payload);
  return data.data ?? data;
}

export async function getReport(id: number): Promise<Report> {
  const { data } = await api.get(`/reports/${id}`);
  return data.data ?? data;
}

export async function listReports(): Promise<Report[]> {
  const { data } = await api.get("/reports");
  return data.data ?? data;
}

// ── Orders ──
export async function createOrder(payload: CreateOrderPayload): Promise<Order> {
  const { data } = await api.post("/orders", payload);
  return data.data ?? data;
}

export async function getOrder(id: number): Promise<Order> {
  const { data } = await api.get(`/orders/${id}`);
  return data.data ?? data;
}

export async function listOrders(): Promise<Order[]> {
  const { data } = await api.get("/orders");
  return data.data ?? data;
}
