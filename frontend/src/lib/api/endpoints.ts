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
  CreateOrderResult,
  UnlockReportResult,
  FreeChartPayload,
  FreeChartResult,
  FreeChartPage,
  FreeChartLocation,
  FreeChartLocationInput,
  GeoCountry,
  GeoCity,
  GeoPage,
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
export async function createReport(payload: CreateReportPayload): Promise<{ report_id: number; status: string }> {
  const { data } = await api.post("/reports", payload);
  return data.data ?? data;
}

// ── Free charts ──
export async function resolveFreeChartLocation(payload: FreeChartLocationInput): Promise<FreeChartLocation> {
  const { data } = await api.post("/locations/resolve", payload);
  return data.data ?? data;
}
export async function createFreeChart(payload: FreeChartPayload): Promise<FreeChartResult> {
  const { data } = await api.post("/free-charts", payload); return data.data ?? data;
}
export async function listFreeCharts(page: number, pageSize: number): Promise<FreeChartPage> {
  const { data } = await api.get("/free-charts", { params: { page, page_size: pageSize } }); return data.data ?? data;
}
export async function getFreeChart(id: number): Promise<FreeChartResult> {
  const { data } = await api.get(`/free-charts/${id}`); return data.data ?? data;
}
export async function deleteFreeChart(id: number): Promise<void> { await api.delete(`/free-charts/${id}`); }
export async function batchDeleteFreeCharts(ids: number[]): Promise<number> {
  const { data } = await api.post("/free-charts/batch-delete", { ids }); return (data.data ?? data).deleted_count;
}
export async function listGeoCountries(locale: string, page = 1, pageSize = 100): Promise<GeoPage<GeoCountry>> {
  const { data } = await api.get("/geo/countries", { params: { locale, page, page_size: pageSize } });
  return data.data ?? data;
}
export async function listGeoCities(countryCode: string, query: string, locale: string, page = 1, pageSize = 50): Promise<GeoPage<GeoCity>> {
  const { data } = await api.get("/geo/cities", { params: { country_code: countryCode, q: query, locale, page, page_size: pageSize } });
  return data.data ?? data;
}

export async function updateProfile(id: number, payload: CreateProfilePayload): Promise<BirthProfile> {
  const { data } = await api.patch(`/profiles/${id}`, payload);
  return data.data ?? data;
}

export type ReportProfileSaveAction = "existing" | "none" | "new" | "update";

export async function createReportFromInput(payload: {
  profile: CreateProfilePayload;
  save_action: ReportProfileSaveAction;
  target_profile_id?: number;
  locale: string;
}): Promise<{ report_id: number; status: string; profile_id: number; profile_saved: boolean; save_action: ReportProfileSaveAction }> {
  const { data } = await api.post("/reports/from-input", payload);
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

export async function unlockReportWithCredits(id: number): Promise<UnlockReportResult> {
  const { data } = await api.post(`/reports/${id}/unlock`);
  return data.data ?? data;
}

// ── Orders ──
export async function createOrder(payload: CreateOrderPayload): Promise<CreateOrderResult> {
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
