import api from "@/lib/admin-client";
import type { GeoCity, GeoCountry, GeoPage } from "@/types/api";

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

export type LocalizedNames = { zh: string; en: string; ja: string; ko: string };
export type BaziBaseSummary = { version: string; valid: boolean; validation_error?: string; elements: number; stems: number; branches: number; relations: number; ten_god_rules: number };
export type BaziBasePage<T = Record<string, unknown>> = { items: T[]; total: number; page: number; page_size: number; version: string };

export async function fetchBaziBaseSummary(): Promise<BaziBaseSummary> {
  const { data } = await api.get("/admin/bazi-base/summary");
  return unwrap<BaziBaseSummary>(data);
}

export async function fetchBaziBasePage(category: string, params: { q?: string; type?: string; page?: number; page_size?: number }): Promise<BaziBasePage> {
  const { data } = await api.get(`/admin/bazi-base/${category}`, { params });
  return unwrap<BaziBasePage>(data);
}

export type AnnualCalendarYear = { year:number; ganzhi:string; stem:string; branch:string; stem_element:string; branch_element:string; stem_yin_yang:string; branch_yin_yang:string; zodiac:string; cycle_index:number; data_version:string };
export async function fetchAnnualCalendar(params:{start_year?:number;end_year?:number;page?:number;page_size?:number}={}){
  const {data}=await api.get("/admin/bazi-base/annual-calendar",{params});
  return unwrap<BaziBasePage<AnnualCalendarYear>>(data);
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

export interface ResourceAction {
  name: string;
  label: string;
}

export interface ResourceSchema {
  name: string;
  fields: ResourceField[];
  actions?: ResourceAction[];
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

export async function runResourceAction(
  resource: string,
  id: string | number,
  action: string,
  params: Record<string, unknown> = {}
): Promise<unknown> {
  const { data } = await api.post(`/admin/resources/${resource}/${id}/actions/${action}`, params);
  return unwrap<unknown>(data);
}

export interface ReportFactsResponse {
  report_id: number;
  snapshot: {
    report_id: number;
    input_snapshot: Record<string, unknown>;
    time_calculation_snapshot: Record<string, unknown>;
    chart_snapshot: Record<string, unknown>;
    facts: Record<string, unknown>;
    chart_hash: string;
    facts_hash: string;
    created_at: string;
  };
}

export interface ReportLLMCallPage {
  report_id: number;
  items: Record<string, unknown>[];
  total: number;
  page: number;
  page_size: number;
}

export async function fetchReportFacts(id: string | number): Promise<ReportFactsResponse> {
  const { data } = await api.get(`/admin/reports/${id}/facts`);
  return unwrap<ReportFactsResponse>(data);
}

export async function fetchReportLLMCalls(id: string | number, page = 1, pageSize = 20): Promise<ReportLLMCallPage> {
  const { data } = await api.get(`/admin/reports/${id}/llm-calls`, { params: { page, page_size: pageSize } });
  return unwrap<ReportLLMCallPage>(data);
}

export interface PromptPreviewResponse {
  registry_version: string;
  prompt_version: string;
  locale: { code: string; name: string; instruction: string };
  chapter: { no: number; key: string; name: string; purpose: string; required_facts: string[]; default_facts: string[]; sections:{key:string;name:string}[]; source_prompt:string };
  system_prompt: string;
  user_prompt: string;
  input_facts: Record<string, unknown>;
  output_schema: Record<string, unknown>;
  facts_hash: string;
  dictionary_version: string;
}

export interface PromptRegistryResponse {
  version: string;
  chapters: PromptPreviewResponse["chapter"][];
  locales: PromptPreviewResponse["locale"][];
}

export async function fetchPromptRegistry(): Promise<PromptRegistryResponse> {
  const { data } = await api.get("/admin/prompt-registry");
  return unwrap<PromptRegistryResponse>(data);
}

export async function previewCalculationPrompt(id:string|number, versionId:number, chapterKey:string, locale:string, factKeys:string[]):Promise<PromptPreviewResponse>{
  const {data}=await api.post(`/admin/calculation-archives/${id}/prompt-preview`,{version_id:versionId,chapter_key:chapterKey,locale,fact_keys:factKeys});
  return unwrap<PromptPreviewResponse>(data);
}
export async function fetchPromptConfigs(){const {data}=await api.get("/admin/prompt-configs");return unwrap<{configured:Record<string,string[]>}>(data)}
export async function savePromptConfig(chapterKey:string,factKeys:string[]){const {data}=await api.put(`/admin/prompt-configs/${chapterKey}`,{fact_keys:factKeys});return unwrap<{chapter_key:string;fact_keys:string[]}>(data)}
export async function resetPromptConfig(chapterKey:string){const {data}=await api.delete(`/admin/prompt-configs/${chapterKey}`);return unwrap<{chapter_key:string;fact_keys:string[]}>(data)}

export async function previewReportPrompt(id: string | number, chapterKey: string, locale: string): Promise<PromptPreviewResponse> {
  const { data } = await api.post(`/admin/reports/${id}/prompt-preview`, { chapter_key: chapterKey, locale });
  return unwrap<PromptPreviewResponse>(data);
}

export interface CalculationInput {
  name: string; gender: number; calendar_type: number; year: number; month: number; day: number;
  hour: number; minute: number; is_leap_month: boolean; locale: string;
  location: { country_code?: string; country_name?: string; region_code?: string; region_name?: string; city?: string; place_id?: string; display_name?: string; latitude: number; longitude: number; timezone_id: string; has_coordinates: boolean };
}
export interface CalculationArchive { id:number; name:string; status:string; input:CalculationInput; latest_version_id?:number; latest_version_no:number; last_error?:string; created_at:string; updated_at:string }
export interface CalculationVersion { id:number; archive_id:number; version_no:number; input_snapshot:Record<string,unknown>; time_calculation_snapshot:Record<string,unknown>; chart_snapshot:Record<string,unknown>; facts_snapshot:Record<string,unknown>; chart_hash:string; facts_hash:string; created_at:string }
export interface CalculationPage { items:CalculationArchive[]; total:number; page:number; page_size:number }
export async function fetchCalculations(q="",page=1,pageSize=10){const {data}=await api.get("/admin/calculation-archives",{params:{q,page,page_size:pageSize}});return unwrap<CalculationPage>(data)}
export async function fetchCalculation(id:string|number){const {data}=await api.get(`/admin/calculation-archives/${id}`);return unwrap<{archive:CalculationArchive;versions:CalculationVersion[]}>(data)}
export async function createCalculation(input:CalculationInput){const {data}=await api.post("/admin/calculation-archives",input);return unwrap<CalculationArchive>(data)}
export async function updateCalculation(id:number,input:CalculationInput){const {data}=await api.patch(`/admin/calculation-archives/${id}`,input);return unwrap<CalculationArchive>(data)}
export async function recalculateCalculation(id:number){const {data}=await api.post(`/admin/calculation-archives/${id}/recalculate`);return unwrap<CalculationArchive>(data)}
export async function deleteCalculation(id:number){await api.delete(`/admin/calculation-archives/${id}`)}
export async function batchDeleteCalculations(ids:number[]){await api.post("/admin/calculation-archives/batch-delete",{ids})}
export async function fetchAdminGeoCountries(locale="zh",page=1,pageSize=100){const {data}=await api.get("/admin/geo/countries",{params:{locale,page,page_size:pageSize}});return unwrap<GeoPage<GeoCountry>>(data)}
export async function fetchAdminGeoCities(countryCode:string,q="",locale="zh",page=1,pageSize=20){const {data}=await api.get("/admin/geo/cities",{params:{country_code:countryCode,q,locale,page,page_size:pageSize}});return unwrap<GeoPage<GeoCity>>(data)}
