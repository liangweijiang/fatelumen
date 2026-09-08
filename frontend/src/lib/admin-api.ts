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
  section: ReportFactsSection;
  value: Record<string, unknown>;
}

export type ReportFactsSection = "input" | "time" | "chart" | "interpretation";

export interface ReportLLMCallPage {
  report_id: number;
  items: ReportAttemptItem[];
  total: number;
  page: number;
  page_size: number;
}

export interface ReportModelStats {
  route_no: number; provider: string; model: string; attempt_count: number; succeeded_count: number;
  failed_count: number; usage_reported_count: number; prompt_tokens?: number; completion_tokens?: number;
  total_tokens?: number; duration_ms: number;
}

export interface ReportAttemptItem {
  id: number; report_id: number; chapter_id: number; attempt_no: number; route_no: number; provider: string;
  model: string; status: string; schema_valid: boolean; validation_status: string; error_code?: string;
  error_summary?: string; prompt_tokens?: number; completion_tokens?: number; total_tokens?: number;
  duration_ms: number; trace_id: string; started_at: string; finished_at?: string;
}

export interface AdminFullReportItem {
  id: number; public_id: string; user_id: number; profile_id?: number; profile_name: string; locale: string;
  status: string; current_stage: string; chapter_total: number; chapter_succeeded: number; chapter_failed: number;
  error_code?: string; error_summary?: string; started_at?: string; completed_at?: string; created_at: string;
}

export interface AdminFullReportPage {
  items: AdminFullReportItem[]; page_size: number; has_more: boolean; next_cursor: string;
}

export interface AdminFullReportOverview {
  report: AdminFullReportItem & {
    chart_id?: number; order_id?: number; source_report_id?: number; pay_method: string; paid: boolean;
    provider_chain_key: string; chapter_concurrency: number; facts_hash: string; execution_hash?: string;
    content_hash?: string; retention_policy: string; generating_at?: string; assembling_at?: string;
    rendering_at?: string; failed_at?: string; expires_at: string; updated_at: string;
  };
  model_stats: ReportModelStats[];
	render_job?: {
		id: number; report_id: number; render_version: string; status: "queued" | "running" | "succeeded" | "failed";
		attempt_count: number; max_attempts: number; error_code?: string; error_summary?: string;
		started_at?: string; finished_at?: string; created_at: string; updated_at: string;
	};
}

export interface AdminFullReportResult {
  id: number; report_id: number; locale: string; content: import("@/types/api").ReportContent; content_hash: string;
  render_version: string; pdf_storage_key?: string; pdf_url?: string; pdf_hash?: string; created_at: string;
}

export async function fetchAdminReports(params: { status?: string; locale?: string; user_id?: number; created_from?: string; created_to?: string; cursor?: string; page_size?: number } = {}) {
  const { data } = await api.get("/admin/reports", { params });
  return unwrap<AdminFullReportPage>(data);
}

export async function fetchAdminReportOverview(id: string | number) {
  const { data } = await api.get(`/admin/reports/${id}`);
  return unwrap<AdminFullReportOverview>(data);
}

export async function fetchAdminReportResult(id: string | number) {
  const { data } = await api.get(`/admin/reports/${id}/result`);
  return unwrap<AdminFullReportResult>(data);
}

export async function fetchReportLLMCall(id: string | number, callId: number) {
  const { data } = await api.get(`/admin/reports/${id}/llm-calls/${callId}`);
  return unwrap<{ attempt: ReportAttemptItem; payload: Record<string, unknown>; chapter: ReportChapterTraceItem }>(data);
}

export async function fetchReportFacts(id: string | number, section: ReportFactsSection): Promise<ReportFactsResponse> {
  const { data } = await api.get(`/admin/reports/${id}/facts`, { params: { section } });
  return unwrap<ReportFactsResponse>(data);
}

export async function fetchReportLLMCalls(id: string | number, page = 1, pageSize = 20, chapterId?: number): Promise<ReportLLMCallPage> {
  const { data } = await api.get(`/admin/reports/${id}/llm-calls`, { params: { page, page_size: pageSize, chapter_id: chapterId } });
  return unwrap<ReportLLMCallPage>(data);
}

export interface ReportValidationRule {
  code: string;
  name: string;
  stage?: string;
  severity: "error" | "warning";
  passed: boolean;
  retryable: boolean;
  affected_chapters?: number[];
  expected?: string;
  actual?: string;
  evidence_refs?: string[];
  message?: string;
}

export interface ReportValidationResult {
  validator_version: string;
  scope: "chapter" | "report";
  passed: boolean;
  retryable: boolean;
  code?: string;
  summary?: string;
  affected_chapters?: number[];
  rules: ReportValidationRule[];
  validated_at?: string;
  schema_valid?: boolean;
  errors?: string[];
  parsed?: unknown;
}

export interface ReportValidationRun {
  id: number;
  report_id: number;
  round_no: number;
  validator_version: string;
  status: string;
  retryable: boolean;
  affected_chapters: number;
  error_code?: string;
  error_summary?: string;
  started_at: string;
  finished_at?: string;
}

export interface ReportChapterTraceItem {
  id: number;
  report_id: number;
  chapter_no: number;
  chapter_key: string;
  title: string;
  status: string;
  attempt_count: number;
  selected_attempt_id?: number;
  prompt_hash?: string;
  output_hash?: string;
  schema_valid: boolean;
  validation_status: string;
  error_code?: string;
  error_summary?: string;
  started_at?: string;
  completed_at?: string;
}

export async function fetchReportValidations(id: string | number) {
  const { data } = await api.get(`/admin/reports/${id}/validations`);
  return unwrap<{ report_id: number; items: ReportValidationRun[] }>(data);
}

export async function fetchReportValidation(id: string | number, validationId: number) {
  const { data } = await api.get(`/admin/reports/${id}/validations/${validationId}`);
  return unwrap<{ run: ReportValidationRun; payload: { validation_result: ReportValidationResult } }>(data);
}

export async function fetchReportChapters(id: string | number) {
  const { data } = await api.get(`/admin/reports/${id}/chapters`);
  return unwrap<{ report_id: number; items: ReportChapterTraceItem[] }>(data);
}

export async function fetchReportChapter(id: string | number, chapterId: number) {
  const { data } = await api.get(`/admin/reports/${id}/chapters/${chapterId}`);
  return unwrap<{ chapter: ReportChapterTraceItem; payload: {
    semantic_digest: string; language_instruction: string; terminology_snapshot: unknown; final_prompt: string;
    output_schema: unknown; final_raw_output?: string; final_parsed_output?: unknown; validation_result?: ReportValidationResult;
  } }>(data);
}

export async function fetchReportChapterArtifact(id: string | number, chapterId: number, type: "content" | "prompt" | "terminology" | "raw_output" | "validation") {
  const { data } = await api.get(`/admin/reports/${id}/chapters/${chapterId}/artifact`, { params: { type } });
  return unwrap<{ chapter: ReportChapterTraceItem; type: string; value: unknown }>(data);
}

export async function fetchReportCallValidation(id: string | number, callId: number) {
  const { data } = await api.get(`/admin/reports/${id}/llm-calls/${callId}/validation`);
  return unwrap<{ report_id: number; call_id: number; validation_result?: ReportValidationResult }>(data);
}

export interface ReportPreflightResult {
  validator_version?: string; passed: boolean; checked_at: string; errors: string[]; warnings: string[]; chapter_count: number;
  chapters: { chapter_no: number; chapter_key: string; passed: boolean; errors: string[]; warnings: string[] }[];
  groups?: { code: string; name: string; logic: string; passed: boolean; summary: string }[];
}

export async function fetchReportPreflight(id: string | number) {
  const { data } = await api.get(`/admin/reports/${id}/execution/preflight`);
  return unwrap<{ report_id: number; result: ReportPreflightResult }>(data);
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
  semantic_digest: {version:string;locale:string;items:{fact:string;status:string;text:string}[];warnings?:string[];coverage:{selected_facts:number;covered_facts:number;unavailable_facts:number;pending_facts:number;coverage_rate:number;missing_facts?:string[];ready:boolean}};
  chapter_instruction: string;
  additive_instruction: string;
  complete_instruction: string;
  terminology_coverage: {locale:string;total:number;approved:number;draft:number;missing:number;rate:number;ready:boolean};
  phrase_coverage: {locale:string;total:number;approved:number;draft:number;missing:number;rate:number;ready:boolean};
  glossary: {source:string;target:string}[];
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

export type LLMModelPreset={id:string;name:string};
export type LLMProviderPreset={code:string;name:string;base_url:string;models:LLMModelPreset[]};
export type LLMProviderConfig={id:number;code:string;name:string;base_url:string;api_key_hint:string;enabled:boolean;created_at:string;updated_at:string};
export type LLMModelConfig={id:number;provider_id:number;provider:LLMProviderConfig;name:string;model_id:string;priority:number;max_retries:number;enabled:boolean;created_at:string;updated_at:string};
export type LLMConfigPage<T>={items:T[];total:number;page:number;page_size:number};

export async function fetchLLMCatalog(){const {data}=await api.get("/admin/llm-catalog");return unwrap<{providers:LLMProviderPreset[]}>(data)}
export async function fetchLLMProviders(page=1,pageSize=10,q=""){const {data}=await api.get("/admin/llm-providers",{params:{page,page_size:pageSize,q}});return unwrap<LLMConfigPage<LLMProviderConfig>>(data)}
export async function createLLMProvider(input:{code:string;name:string;base_url:string;api_key:string;enabled:boolean}){const {data}=await api.post("/admin/llm-providers",input);return unwrap<LLMProviderConfig>(data)}
export async function updateLLMProvider(id:number,input:{code:string;name:string;base_url:string;api_key?:string;enabled:boolean}){const {data}=await api.put(`/admin/llm-providers/${id}`,input);return unwrap<LLMProviderConfig>(data)}
export async function deleteLLMProvider(id:number){await api.delete(`/admin/llm-providers/${id}`)}
export async function fetchLLMModels(page=1,pageSize=10,q="",providerId?:number){const {data}=await api.get("/admin/llm-models",{params:{page,page_size:pageSize,q,provider_id:providerId}});return unwrap<LLMConfigPage<LLMModelConfig>>(data)}
export async function createLLMModel(input:{provider_id:number;name:string;model_id:string;priority:number;max_retries:number;enabled:boolean}){const {data}=await api.post("/admin/llm-models",input);return unwrap<LLMModelConfig>(data)}
export async function updateLLMModel(id:number,input:{provider_id:number;name:string;model_id:string;priority:number;max_retries:number;enabled:boolean}){const {data}=await api.put(`/admin/llm-models/${id}`,input);return unwrap<LLMModelConfig>(data)}
export async function deleteLLMModel(id:number){await api.delete(`/admin/llm-models/${id}`)}

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
