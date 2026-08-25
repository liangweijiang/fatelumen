// ── User ──
export interface User {
  id: number;
  email: string;
  name: string;
  avatar_url: string;
  credits: number;
  locale: string;
  role: "user" | "admin";
  active: boolean;
  created_at: string;
  updated_at: string;
}

// ── BirthProfile ──
export interface BirthProfile {
  id: number;
  user_id: number;
  display_name: string;
  gender: number; // 0=female, 1=male
  calendar_type: number; // 0=solar, 1=lunar
  birth_year: number;
  birth_month: number;
  birth_day: number;
  birth_hour: number; // -1=unknown
  birth_minute: number;
  is_leap_month: number;
  birth_place: string;
  timezone: string;
  longitude: number;
  created_at: string;
  updated_at: string;
}

export interface CreateProfilePayload {
  display_name?: string;
  gender: number;
  calendar_type: number;
  birth_year: number;
  birth_month: number;
  birth_day: number;
  birth_hour: number;
  birth_minute: number;
  is_leap_month: boolean;
  birth_place?: string;
	country_code?: string;
	country_name?: string;
	region_code?: string;
	region_name?: string;
	city?: string;
	place_id?: string;
  timezone?: string;
  longitude?: number;
  latitude?: number;
	has_coordinates?: boolean;
}

// ── Chart ──
export interface Pillar {
  stem: string;
  branch: string;
  stem_element: string;
  branch_element: string;
  ten_god_stem: string;
  ten_god_hidden: string[];
  hidden_stems: string[];
  nayin: string;
}

export interface Pillars {
  year: Pillar;
  month: Pillar;
  day: Pillar;
  hour: Pillar;
}

export interface LuckCycle {
  ganzhi: string;
  start_age: number;
  start_year: number;
  element?: string;
}

export interface DayMaster {
  stem: string;
  element: string;
  yin_yang: string;
}

export interface Strength {
  level: string;
  score: number;
  favorable: string[];
  unfavorable: string[];
}

export interface CurrentYearFortune {
  year: number;
  stem: string;
  branch: string;
  element: string;
}

export interface AnnualFortune {
  year: number;
  age?: number;
  ganzhi: string;
  stem: string;
  branch: string;
  element: string;
  luck_cycle_ganzhi?: string;
  luck_cycle_start_age?: number;
  luck_cycle_start_year?: number;
}

export interface ChartMeta {
  solar_date: string;
  lunar_date: string;
  gender: string;
  calc_lib: string;
  calc_version: string;
}

export interface ChartData {
  pillars: Pillars;
  day_master: DayMaster;
  five_elements_count: Record<string, number>;
  strength: Strength;
  luck_cycles: LuckCycle[];
  current_year_fortune?: CurrentYearFortune;
  annual_fortunes?: AnnualFortune[];
  hour_unknown: boolean;
  meta: ChartMeta;
}

export interface Chart {
  id: number;
  profile_id: number;
  chart_hash: string;
  chart_data: ChartData;
  created_at: string;
}

export interface CreateChartPayload {
  profile_id: number;
}

// ── Reading ──
export interface QuickContent {
  summary_line: string;
  personality: string;
  strengths: string[];
  weaknesses: string[];
  element_note: string;
}

export interface Reading {
  id: number;
  user_id: number;
  profile_id: number;
  chart_id: number;
  locale: string;
  content: QuickContent;
  image_url: string;
  status: string;
  created_at: string;
}

export interface CreateQuickReadingPayload {
  profile_id: number;
  locale?: string;
}

// ── Report ──
export interface YearlyFortuneItem {
  year: number;
  note: string;
}

export interface Chapter {
  no: number;
  key: string;
  title: string;
  body: string;
  strength_score?: number;
  cycles?: CycleNote[];
  years?: YearNote[];
  tags?: string[];
}

export interface CycleNote {
  ganzhi: string;
  start_age: number;
  start_year: number;
  note: string;
}

export interface YearNote {
  year: number;
  ganzhi: string;
  note: string;
}

export interface ReportContent {
  locale: string;
  summary_line: string;
  summary: string;
  personality: string;
  career: string;
  relationship: string;
  health: string;
  yearly_fortune: YearlyFortuneItem[];
  suggestions: string[];
  chapters?: Chapter[];
}

export interface Report {
  id: number;
  user_id: number;
  profile_id: number;
  chart_id: number;
  order_id?: number;
  locale: string;
  status: "pending" | "processing" | "done" | "failed";
  pay_method: string;
  content: ReportContent;
  pdf_url: string;
  error_msg: string;
  retry_count: number;
  paid: boolean;
  locked?: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateReportPayload {
  profile_id: number;
  locale?: string;
}

// ── Order ──
export interface Order {
  id: number;
  user_id: number;
  report_id: number;
  type: string;
  sku: string;
  amount_cents: number;
  currency: string;
  credits_granted: number;
  provider: string;
  provider_ref: string;
  provider_txn_id: string;
  provider_meta: unknown;
  status: "created" | "pending" | "paid" | "failed" | "refunded";
  created_at: string;
  updated_at: string;
}

export interface CreateOrderPayload {
  report_id: number;
  provider: string;
  sku?: string;
}

export interface CreateOrderResult {
  order_id: number;
  status: string;
  checkout_url: string;
}

export interface UnlockReportResult {
  report_id: number;
  unlocked: boolean;
}

export interface FreeChartLocation {
  country_code?: string; country_name?: string; region_code?: string; region_name?: string;
  city?: string; place_id?: string; display_name?: string;
  latitude: number; longitude: number; timezone_id: string;
}
export interface FreeChartLocationInput extends Partial<FreeChartLocation> { has_coordinates: boolean }
export interface FreeChartPayload {
  gender: 0 | 1; calendar_type: 0 | 1; year: number; month: number; day: number;
  hour: number; minute: number; is_leap_month: boolean; location: FreeChartLocationInput;
}
export interface FreeChartResult {
  id: number; chart_hash: string; location: FreeChartLocation; pillars: Pillars; day_master: DayMaster;
  five_elements_count: Record<string, number>; lunar_date: string; solar_date: string; zodiac: string;
  gender: 0 | 1; calendar_type: 0 | 1; birth_year: number; birth_month: number; birth_day: number;
  birth_hour: number; birth_minute: number; is_leap_month: boolean; created_at: string;
	time_calculation: { timezone_id: string; local_civil_time: string; true_solar_time: string };
}
export interface FreeChartPage { items: FreeChartResult[]; total: number; page: number; page_size: number }

export interface GeoCountry {
  code: string; name_en: string; name_zh: string; name_ja: string; name_ko: string;
}
export interface GeoCity {
  id: number; country_code: string; admin1_code: string; admin1_name: string;
  name_en: string; name_zh: string; name_ja: string; name_ko: string;
  latitude: number; longitude: number; timezone_id: string;
}
export interface GeoPage<T> { items: T[]; total: number; page: number; page_size: number }

// ── Auth ──
export interface AuthProvider {
  id: string;
  name: string;
  login_url: string;
}

export interface AuthUser {
  token: string;
  user: User;
}

// ── Common ──
export interface PaginatedList<T> {
  items: T[];
  total: number;
}

export interface APIError {
  error: string;
  message?: string;
}
