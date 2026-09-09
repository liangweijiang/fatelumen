"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { createReportFromInput, listGeoCities, listGeoCountries } from "@/lib/api/endpoints";
import type { CreateProfilePayload, GeoCity, GeoCountry } from "@/types/api";
import { normalizeLocale } from "@/lib/location-options";
import { timeZoneOptions } from "@/lib/timezone-options";
import { CitySearchField } from "./CitySearchField";
import { CountrySearchField } from "@/components/form/CountrySearchField";

function stripLeadingZero(v: string): number {
  if (v === "" || v === "-") return 0;
  const n = Number(v);
  return Number.isNaN(n) ? 0 : n;
}

function daysInSolarMonth(year: number, month: number): number {
  if (month === 2) {
    const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
    return leap ? 29 : 28;
  }
  return [4, 6, 9, 11].includes(month) ? 30 : 31;
}

function localizedName(item: GeoCountry | GeoCity, locale: "en" | "zh" | "ja" | "ko") {
  return item[`name_${locale}`] || item.name_en;
}

export default function CalculatePage() {
  const t = useTranslations("calculate");
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";
  const locationLocale = normalizeLocale(locale);

  const [submitting, setSubmitting] = useState(false);

  const [calendarType, setCalendarType] = useState(0);
  const [gender, setGender] = useState(1);
  const [birthYear, setBirthYear] = useState(1990);
  const [birthMonth, setBirthMonth] = useState(1);
  const [birthDay, setBirthDay] = useState(1);
  const [birthHour, setBirthHour] = useState(12);
  const [birthMinute, setBirthMinute] = useState(0);
  const [isLeapMonth, setIsLeapMonth] = useState(false);
  const [countryCode, setCountryCode] = useState("");
  const [countryQuery, setCountryQuery] = useState("");
  const [regionQuery, setRegionQuery] = useState("");
  const [geoCountries, setGeoCountries] = useState<GeoCountry[]>([]);
  const [geoCities, setGeoCities] = useState<GeoCity[]>([]);
  const [selectedCity, setSelectedCity] = useState<GeoCity | null>(null);
  const [cityLoading, setCityLoading] = useState(false);
  const [cityLoadingMore, setCityLoadingMore] = useState(false);
  const [cityPage, setCityPage] = useState(1);
  const [cityTotal, setCityTotal] = useState(0);
  const [longitude, setLongitude] = useState("");
  const [latitude, setLatitude] = useState("");
  const [timezone, setTimezone] = useState("UTC");
  const [displayName, setDisplayName] = useState("");

  const selectedCountry = geoCountries.find((country) => country.code === countryCode);
  const countryOptions = useMemo(() => geoCountries.map((item) => ({ code: item.code, label: localizedName(item, locationLocale) })), [geoCountries, locationLocale]);
  const timezoneOptions = useMemo(() => timeZoneOptions(new Date(birthYear, Math.max(0, birthMonth - 1), Math.max(1, birthDay), birthHour, birthMinute)), [birthDay, birthHour, birthMinute, birthMonth, birthYear]);

  useEffect(() => { setTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"); }, []);
  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const first = await listGeoCountries(locationLocale, 1, 100);
        const pages = Math.ceil(first.total / first.page_size);
        const rest = await Promise.all(Array.from({ length: Math.max(0, pages - 1) }, (_, index) => listGeoCountries(locationLocale, index + 2, 100)));
        if (active) setGeoCountries([...(first.items ?? []), ...rest.flatMap((page) => page.items ?? [])]);
      } catch { if (active) toast.error(t("locationUnavailable")); }
    })();
    return () => { active = false; };
  }, [locationLocale, t]);

  useEffect(() => {
    const query = regionQuery.trim();
    if (!countryCode) { setGeoCities([]); setCityTotal(0); setCityLoading(false); return; }
    if (selectedCity && localizedName(selectedCity, locationLocale) === regionQuery) { setCityLoading(false); return; }
    let active = true;
    setCityLoading(true);
    const timer = window.setTimeout(() => {
      void listGeoCities(countryCode, query, locationLocale, 1).then((result) => {
        if (active) { setGeoCities(result.items ?? []); setCityTotal(result.total ?? 0); setCityPage(1); }
      }).catch(() => { if (active) toast.error(t("locationUnavailable")); }).finally(() => { if (active) setCityLoading(false); });
    }, 250);
    return () => { active = false; window.clearTimeout(timer); };
  }, [countryCode, locationLocale, regionQuery, selectedCity, t]);

  const loadMoreCities = useCallback(async () => {
    if (!countryCode || cityLoadingMore || geoCities.length >= cityTotal) return;
    setCityLoadingMore(true);
    try {
      const next = cityPage + 1;
      const result = await listGeoCities(countryCode, regionQuery.trim(), locationLocale, next);
      setGeoCities((current) => { const seen = new Set(current.map((city) => city.id)); return [...current, ...result.items.filter((city) => !seen.has(city.id))]; });
      setCityPage(next); setCityTotal(result.total ?? cityTotal);
    } catch { toast.error(t("locationUnavailable")); }
    finally { setCityLoadingMore(false); }
  }, [cityLoadingMore, cityPage, cityTotal, countryCode, geoCities.length, locationLocale, regionQuery, t]);

  function buildProfileInput(): CreateProfilePayload | null {
    const lng = longitude.trim() ? Number(longitude) : undefined;
    const lat = latitude.trim() ? Number(latitude) : undefined;
    const maxDay = calendarType === 1 ? 30 : daysInSolarMonth(birthYear, birthMonth);
    const completePlace = Boolean(selectedCountry && selectedCity);
    const hasPlaceInput = Boolean(countryQuery.trim() || regionQuery.trim());
    const completeCoordinates = lng !== undefined && lat !== undefined && Number.isFinite(lng) && Number.isFinite(lat);
    const hasCoordinateInput = Boolean(longitude.trim() || latitude.trim());
    if (birthYear < 1 || birthYear > 9999 || birthMonth < 1 || birthMonth > 12 ||
      birthDay < 1 || birthDay > maxDay || birthHour < 0 || birthHour > 23 ||
      birthMinute < 0 || birthMinute > 59 ||
      (hasPlaceInput && !completePlace) ||
      (hasCoordinateInput && !completeCoordinates) ||
      (lng !== undefined && (Number.isNaN(lng) || lng < -180 || lng > 180)) ||
      (lat !== undefined && (Number.isNaN(lat) || lat < -90 || lat > 90))) {
      toast.error(t("invalidInput"));
      return null;
    }
    if (!completePlace && !completeCoordinates) { toast.error(t("locationRequired")); return null; }
    if (completePlace && completeCoordinates && (Math.abs(lng! - selectedCity!.longitude) > 0.0001 || Math.abs(lat! - selectedCity!.latitude) > 0.0001)) { toast.error(t("locationConflict")); return null; }
    if (!timezone.trim() || !timezoneOptions.some((item) => item.value === timezone)) { toast.error(t("timezoneInvalid")); return null; }
    const birthPlace = completePlace ? `${selectedCity!.name_en}, ${selectedCountry!.name_en}` : undefined;
    return {
      calendar_type: calendarType,
      gender,
      birth_year: birthYear,
      birth_month: birthMonth,
      birth_day: birthDay,
      birth_hour: birthHour,
      birth_minute: birthMinute,
      is_leap_month: isLeapMonth,
      birth_place: birthPlace,
      country_code: selectedCountry?.code,
      country_name: selectedCountry?.name_en,
      region_code: selectedCity?.admin1_code,
      region_name: selectedCity?.name_en,
      city: selectedCity?.name_en,
      place_id: selectedCity ? String(selectedCity.id) : undefined,
      timezone,
      longitude: lng,
      latitude: lat,
      has_coordinates: completeCoordinates,
      display_name: displayName.trim() || undefined,
    };
  }

  async function handleSubmit() {
    const profileInput = buildProfileInput();
    if (!profileInput) return;
    setSubmitting(true);
    try {
      const report = await createReportFromInput({ profile: profileInput, save_action: "new", locale });
      router.push(`/${locale}/reports/${report.report_id}`);
    } catch (error) {
      toast.error(error instanceof Error && error.message === "insufficient credits" ? t("insufficientCredits") : t("error"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div
      className="relative min-h-screen px-5 py-10 md:px-10 md:py-16"
      style={{ background: "var(--bg)" }}
    >
      <div className="mx-auto max-w-[640px]">
        {/* Header */}
        <div className="mb-10 text-center">
          <h1
            className="mb-3 text-[32px] font-semibold leading-tight tracking-[-0.3px]"
            style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
          >
            {t("title")}
          </h1>
          <p
            className="text-[15px]"
            style={{ color: "var(--ink-soft)" }}
          >
            {t("sub")}
          </p>
        </div>

        {/* Form Card */}
        <div
          className="mb-8 rounded-2xl border p-7 md:p-10"
          style={{
            background: "var(--bg-card)",
            borderColor: "var(--line)",
            boxShadow: "0 10px 36px -18px oklch(35% 0.04 60 / 0.4)",
          }}
        >
          {/* Display name */}
          <div className="mb-6">
            <label
              className="mb-1 block text-[12px] tracking-[.3px]"
              style={{ color: "var(--ink-faint)" }}
            >
              {t("displayName")}
            </label>
            <input
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
              style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink)" }}
            />
          </div>

          {/* Calendar type */}
          <div className="mb-6">
            <label
              className="mb-2 block text-[13px] font-semibold tracking-[.4px]"
              style={{ color: "var(--ink)" }}
            >
              {t("calendar")}
            </label>
            <div className="flex rounded-full border p-1" style={{ borderColor: "var(--line)" }}>
              <button
                type="button"
                onClick={() => setCalendarType(0)}
                className="flex-1 rounded-full px-4 py-2 text-[14px] font-semibold transition-all"
                style={{
                  background: calendarType === 0 ? "var(--gold-deep)" : "transparent",
                  color: calendarType === 0 ? "var(--bg-card)" : "var(--ink-soft)",
                }}
              >
                {t("solar")}
              </button>
              <button
                type="button"
                onClick={() => setCalendarType(1)}
                className="flex-1 rounded-full px-4 py-2 text-[14px] font-semibold transition-all"
                style={{
                  background: calendarType === 1 ? "var(--gold-deep)" : "transparent",
                  color: calendarType === 1 ? "var(--bg-card)" : "var(--ink-soft)",
                }}
              >
                {t("lunar")}
              </button>
            </div>
          </div>

          {/* Birth date fields */}
          <div className="mb-6 grid grid-cols-3 gap-3">
            <div>
              <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
                {t("birthYear")}
              </label>
              <input
                type="number"
                value={birthYear}
                onChange={(e) => setBirthYear(stripLeadingZero(e.target.value))}
                className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
                style={{
                  background: "var(--bg)",
                  borderColor: "var(--line)",
                  color: "var(--ink)",
                }}
              />
            </div>
            <div>
              <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
                {t("birthMonth")}
              </label>
              <input
                type="number"
                value={birthMonth}
                onChange={(e) => setBirthMonth(stripLeadingZero(e.target.value))}
                min={1}
                max={12}
                className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
                style={{
                  background: "var(--bg)",
                  borderColor: "var(--line)",
                  color: "var(--ink)",
                }}
              />
            </div>
            <div>
              <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
                {t("birthDay")}
              </label>
              <input
                type="number"
                value={birthDay}
                onChange={(e) => setBirthDay(stripLeadingZero(e.target.value))}
                min={1}
                max={31}
                className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
                style={{
                  background: "var(--bg)",
                  borderColor: "var(--line)",
                  color: "var(--ink)",
                }}
              />
            </div>
          </div>

          {/* Birth time */}
          <div className="mb-6 grid grid-cols-2 gap-3">
            <div>
              <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
                {t("birthHour")}
              </label>
              <input
                type="number"
                value={birthHour}
                onChange={(e) => setBirthHour(stripLeadingZero(e.target.value))}
                min={0}
                max={23}
                className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
                style={{
                  background: "var(--bg)",
                  borderColor: "var(--line)",
                  color: "var(--ink)",
                }}
              />
            </div>
            <div>
              <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
                {t("birthMinute")}
              </label>
              <input
                type="number"
                value={birthMinute}
                onChange={(e) => setBirthMinute(stripLeadingZero(e.target.value))}
                min={0}
                max={59}
                className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
                style={{
                  background: "var(--bg)",
                  borderColor: "var(--line)",
                  color: "var(--ink)",
                }}
              />
            </div>
          </div>

          {/* Leap month (lunar only) */}
          {calendarType === 1 && (
            <label className="mb-6 flex items-center gap-3 text-sm text-[var(--ink-soft)]">
              <input
                type="checkbox"
                checked={isLeapMonth}
                onChange={(event) => setIsLeapMonth(event.target.checked)}
                className="h-4 w-4 accent-[var(--gold-deep)]"
              />
              {t("leapMonth")}
            </label>
          )}

          {/* Birth place */}
          <div className="mb-6">
            <label className="mb-2 block text-[13px] font-semibold tracking-[.4px]" style={{ color: "var(--ink)" }}>
              {t("birthPlace")}
            </label>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label htmlFor="birth-country" className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>{t("country")}</label>
                <CountrySearchField id="birth-country" value={countryQuery} selectedCode={countryCode} options={countryOptions} placeholder={t("countrySearch")} ariaLabel={t("country")} emptyText={t("noLocationResults")} className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2" onChange={(query) => { setCountryQuery(query); setCountryCode(""); setSelectedCity(null); setRegionQuery(""); }} onSelect={(country) => { setCountryQuery(country.label); setCountryCode(country.code); setSelectedCity(null); setRegionQuery(""); }} />
              </div>
              <div>
                <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>{t("region")}</label>
                <CitySearchField value={regionQuery} disabled={!selectedCountry} loading={cityLoading} loadingMore={cityLoadingMore} options={geoCities} total={cityTotal} selectedID={selectedCity?.id} placeholder={selectedCountry ? t("regionSearch") : t("selectCountryFirst")} ariaLabel={t("region")} loadingText={t("citySearching")} emptyText={t("noLocationResults")} statusText={t("cityCount", { shown: geoCities.length, total: cityTotal })} getName={(city) => localizedName(city, locationLocale)} onChange={(query) => { setRegionQuery(query); setSelectedCity(null); }} onSelect={(city) => { setSelectedCity(city); setRegionQuery(localizedName(city, locationLocale)); setLongitude(String(city.longitude)); setLatitude(String(city.latitude)); setTimezone(city.country_code === "CN" ? "Asia/Shanghai" : city.timezone_id); }} onLoadMore={() => void loadMoreCities()} />
              </div>
            </div>
            <p className="my-3 text-center text-[12px]" style={{ color: "var(--ink-faint)" }}>
              {t("locationOrCoordinates")}
            </p>
            <div className="mt-3 grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
                  {t("longitude")}
                </label>
                <input
                  type="text"
                  inputMode="decimal"
                  value={longitude}
                  onChange={(e) => setLongitude(e.target.value)}
                  placeholder="116.4074"
                  className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
                  style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink)" }}
                />
              </div>
              <div>
                <label className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
                  {t("latitude")}
                </label>
                <input
                  type="text"
                  inputMode="decimal"
                  value={latitude}
                  onChange={(e) => setLatitude(e.target.value)}
                  placeholder="39.9042"
                  className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
                  style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink)" }}
                />
              </div>
            </div>
            <p className="mt-2 text-[12px]" style={{ color: "var(--ink-faint)" }}>
              {t("placeHint")}
            </p>
            <div className="mt-4">
              <label htmlFor="birth-timezone" className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>{t("timezone")}</label>
              <input id="birth-timezone" type="search" list="calculate-timezones" value={timezone} onChange={(event) => setTimezone(event.target.value)} placeholder={t("timezoneSearch")} autoComplete="off" className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2" style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink)" }} />
              <datalist id="calculate-timezones">{timezoneOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</datalist>
              <p className="mt-2 text-[12px]" style={{ color: "var(--ink-faint)" }}>{t("timezoneHint")}</p>
            </div>
          </div>

          {/* Gender */}
          <div className="mb-6">
            <label
              className="mb-2 block text-[13px] font-semibold tracking-[.4px]"
              style={{ color: "var(--ink)" }}
            >
              {t("gender")}
            </label>
            <div className="flex rounded-full border p-1" style={{ borderColor: "var(--line)" }}>
              <button
                type="button"
                onClick={() => setGender(1)}
                className="flex-1 rounded-full px-4 py-2 text-[14px] font-semibold transition-all"
                style={{
                  background: gender === 1 ? "var(--gold-deep)" : "transparent",
                  color: gender === 1 ? "var(--bg-card)" : "var(--ink-soft)",
                }}
              >
                {t("male")}
              </button>
              <button
                type="button"
                onClick={() => setGender(0)}
                className="flex-1 rounded-full px-4 py-2 text-[14px] font-semibold transition-all"
                style={{
                  background: gender === 0 ? "var(--gold-deep)" : "transparent",
                  color: gender === 0 ? "var(--bg-card)" : "var(--ink-soft)",
                }}
              >
                {t("female")}
              </button>
            </div>
          </div>

        </div>

        {/* Submit */}
        <button
          type="button"
          onClick={handleSubmit}
          disabled={submitting}
          className="w-full rounded-full py-3.5 text-[16px] font-semibold tracking-[.3px] transition-all"
          style={{
            fontFamily: "var(--serif-d)",
            background: submitting ? "var(--ink-faint)" : "var(--gold-deep)",
            color: "var(--bg-card)",
            opacity: submitting ? 0.6 : 1,
            cursor: submitting ? "not-allowed" : "pointer",
          }}
        >
          {submitting ? t("submitting") : t("submit")}
        </button>
      </div>

    </div>
  );
}
