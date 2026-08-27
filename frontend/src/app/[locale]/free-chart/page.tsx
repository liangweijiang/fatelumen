"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { History, LockKeyhole, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { getToken } from "@/lib/auth-storage";
import { normalizeLocale } from "@/lib/location-options";
import { timeZoneOptions } from "@/lib/timezone-options";
import { batchDeleteFreeCharts, createFreeChart, deleteFreeChart, getFreeChart, listFreeCharts, listGeoCities, listGeoCountries } from "@/lib/api/endpoints";
import type { FreeChartLocationInput, FreeChartResult, GeoCity, GeoCountry } from "@/types/api";
import { ChartResultDialog } from "./ChartResultDialog";
import { CitySearchField } from "./CitySearchField";
import { HistoryDialog } from "./HistoryDialog";
import type { ChartRecord } from "./types";
import { CountrySearchField } from "@/components/form/CountrySearchField";

function numberValue(value: string) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function localizedName(item: GeoCountry | GeoCity, locale: "en" | "zh" | "ja" | "ko") {
  return item[`name_${locale}`] || item.name_en;
}

function toRecord(result: FreeChartResult, locale: string): ChartRecord {
  const p = result.pillars;
  const pillars = [
    { key: "yearPillar" as const, ...p.year }, { key: "monthPillar" as const, ...p.month },
    { key: "dayPillar" as const, ...p.day }, { key: "hourPillar" as const, ...p.hour },
  ].map(item => ({ key:item.key, stem:item.stem, branch:item.branch, stemElement:item.stem_element, branchElement:item.branch_element }));
  const names=[result.location.country_name,result.location.region_name||result.location.city].filter(Boolean).join(" · ");
  return {id:result.id,solarDate:result.solar_date||`${result.birth_year}-${result.birth_month}-${result.birth_day} ${result.birth_hour}:${result.birth_minute}`,lunarDate:result.lunar_date,gender:result.gender,place:names||result.location.display_name||"—",coordinates:`${result.location.longitude.toFixed(4)}, ${result.location.latitude.toFixed(4)}`,timezone:result.time_calculation?.timezone_id||result.location.timezone_id,trueSolarTime:result.time_calculation?.true_solar_time||"—",dayMaster:`${result.day_master.stem}${result.day_master.element}`,createdAt:new Date(result.created_at).toLocaleString(locale),pillars};
}

export default function FreeChartPage() {
  const t = useTranslations("freeChart");
  const params = useParams();
  const router = useRouter();
  const locale = (params.locale as string) || "en";
  const dataLocale = normalizeLocale(locale);
  const [ready, setReady] = useState(false);
  const [calendar, setCalendar] = useState<0 | 1>(0);
  const [leapMonth, setLeapMonth] = useState(false);
  const [year, setYear] = useState(1990);
  const [month, setMonth] = useState(1);
  const [day, setDay] = useState(1);
  const [hour, setHour] = useState(12);
  const [minute, setMinute] = useState(0);
  const [gender, setGender] = useState<0 | 1>(1);
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
  const [history, setHistory] = useState<ChartRecord[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [resultOpen, setResultOpen] = useState(false);
  const [activeRecord, setActiveRecord] = useState<ChartRecord | null>(null);
  const [locationDialogOpen, setLocationDialogOpen] = useState(false);

  useEffect(() => {
    if (!getToken()) {
      router.replace(`/login?lang=${locale}&next=/${locale}/free-chart`);
      return;
    }
    setTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC");
    setReady(true);
  }, [locale, router]);

  const selectedCountry = geoCountries.find((item) => item.code === countryCode);
  const countryOptions = useMemo(
    () => geoCountries.map((item) => ({ code: item.code, label: localizedName(item, dataLocale), item })),
    [dataLocale, geoCountries],
  );
  const timezoneOptions = useMemo(
    () => timeZoneOptions(new Date(year, Math.max(0, month - 1), Math.max(1, day), hour, minute)),
    [day, hour, minute, month, year],
  );

  useEffect(() => {
    if (!ready) return;
    let active = true;
    void (async () => {
      try {
        const first = await listGeoCountries(dataLocale, 1, 100);
        const pages = Math.ceil(first.total / first.page_size);
        const rest = await Promise.all(Array.from({ length: Math.max(0, pages - 1) }, (_, index) => listGeoCountries(dataLocale, index + 2, 100)));
        if (active) setGeoCountries([...(first.items ?? []), ...rest.flatMap((page) => page.items ?? [])]);
      } catch { if (active) toast.error(t("locationUnavailable")); }
    })();
    return () => { active = false; };
  }, [dataLocale, ready, t]);

  useEffect(() => {
    const query = regionQuery.trim();
    if (!countryCode) { setGeoCities([]); setCityTotal(0); setCityLoading(false); return; }
    if (selectedCity && localizedName(selectedCity, dataLocale) === regionQuery) { setCityLoading(false); return; }
    let active = true;
    setCityLoading(true);
    const timer = window.setTimeout(() => {
      void listGeoCities(countryCode, query, dataLocale, 1).then((result) => {
        if (active) { setGeoCities(result.items ?? []); setCityTotal(result.total ?? 0); setCityPage(1); }
      }).catch(() => { if (active) toast.error(t("locationUnavailable")); }).finally(() => { if (active) setCityLoading(false); });
    }, 250);
    return () => { active = false; window.clearTimeout(timer); };
  }, [countryCode, dataLocale, regionQuery, selectedCity, t]);

  const loadMoreCities=useCallback(async()=>{
    if(!countryCode||cityLoadingMore||geoCities.length>=cityTotal)return;
    setCityLoadingMore(true);
    try{const next=cityPage+1;const result=await listGeoCities(countryCode,regionQuery.trim(),dataLocale,next);setGeoCities((current)=>{const seen=new Set(current.map((city)=>city.id));return [...current,...result.items.filter((city)=>!seen.has(city.id))]});setCityPage(next);setCityTotal(result.total??cityTotal)}
    catch{toast.error(t("locationUnavailable"))}
    finally{setCityLoadingMore(false)}
  },[cityLoadingMore,cityPage,cityTotal,countryCode,dataLocale,geoCities.length,regionQuery,t]);
  const loadHistory = useCallback(async (page: number) => {
    setHistoryLoading(true);
    try { const data=await listFreeCharts(page,5); setHistory(data.items.map(item=>toRecord(item,locale))); setHistoryTotal(data.total); }
    catch { toast.error(t("historyLoadFailed")); }
    finally { setHistoryLoading(false); }
  },[locale,t]);

  useEffect(()=>{if(ready)void loadHistory(1)},[ready,loadHistory]);
  useEffect(()=>{if(historyOpen)void loadHistory(historyPage)},[historyOpen,historyPage,loadHistory]);

  async function calculate() {
    const validDate = year > 0 && month >= 1 && month <= 12 && day >= 1 && day <= 31;
    const validTime = hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59;
    const completePlace = Boolean(selectedCountry && selectedCity);
    const hasPlaceInput = Boolean(countryQuery.trim() || regionQuery.trim());
    const lng = Number(longitude);
    const lat = Number(latitude);
    const completeCoordinates = Boolean(longitude.trim() && latitude.trim()) &&
      Number.isFinite(lng) && Number.isFinite(lat) && lng >= -180 && lng <= 180 && lat >= -90 && lat <= 90;
    const hasCoordinateInput = Boolean(longitude.trim() || latitude.trim());
    if (!validDate || !validTime) {
      toast.error(t("invalid"));
      return;
    }
    if (hasPlaceInput && !completePlace) { toast.error(t("placeIncomplete")); return; }
    if (hasCoordinateInput && !completeCoordinates) { toast.error(t("coordinatesIncomplete")); return; }
    if (!completePlace && !completeCoordinates) { setLocationDialogOpen(true); return; }
    if (completePlace && completeCoordinates && (Math.abs(lng-selectedCity!.longitude)>0.0001 || Math.abs(lat-selectedCity!.latitude)>0.0001)) { toast.error(t("locationConflict")); return; }
    if (!timezone.trim() || !timezoneOptions.some((item) => item.value === timezone)) { toast.error(t("timezoneInvalid")); return; }
    const location:FreeChartLocationInput={country_code:selectedCountry?.code,country_name:selectedCountry?.name_en,region_code:selectedCity?.admin1_code,region_name:selectedCity?.name_en,city:selectedCity?.name_en,place_id:selectedCity?String(selectedCity.id):undefined,display_name:completePlace?`${selectedCity!.name_en}, ${selectedCountry!.name_en}`:undefined,latitude:completeCoordinates?lat:undefined,longitude:completeCoordinates?lng:undefined,timezone_id:timezone,has_coordinates:completeCoordinates};
    setSubmitting(true);
    try {
      const result=await createFreeChart({gender,calendar_type:calendar,year,month,day,hour,minute,is_leap_month:leapMonth,location});
      setActiveRecord(toRecord(result,locale)); setResultOpen(true); setHistoryTotal(value=>value+1);
    } catch(error) { const message=error instanceof Error?error.message:""; toast.error(message==="location_conflict"?t("locationConflict"):message==="location_validation_unavailable"?t("locationValidationUnavailable"):message.includes("location")?t("locationUnavailable"):t("chartCreateFailed")); }
    finally { setSubmitting(false); }
  }

  async function viewHistory(record:ChartRecord){try{const result=await getFreeChart(record.id);setActiveRecord(toRecord(result,locale));setResultOpen(true)}catch{toast.error(t("historyDetailFailed"))}}
  async function removeHistory(ids:number[]){try{if(ids.length===1)await deleteFreeChart(ids[0]);else await batchDeleteFreeCharts(ids);const nextTotal=Math.max(0,historyTotal-ids.length);const lastPage=Math.max(1,Math.ceil(nextTotal/5));setHistoryTotal(nextTotal);if(historyPage>lastPage)setHistoryPage(lastPage);else await loadHistory(historyPage);toast.success(t("deleteSuccess"))}catch{toast.error(t("deleteFailed"))}}

  if (!ready) {
    return (
      <div className="flex min-h-[62vh] items-center justify-center bg-[var(--bg)] text-sm text-[var(--ink-faint)]">
        <LockKeyhole className="mr-2 h-4 w-4" aria-hidden="true" />{t("loginChecking")}
      </div>
    );
  }

  const fieldClass = "w-full border border-[var(--line)] bg-[var(--bg)] px-3 py-3 text-sm text-[var(--ink)] outline-none transition focus:border-[var(--gold-deep)] focus:ring-1 focus:ring-[var(--gold-deep)] disabled:cursor-not-allowed disabled:opacity-45";
  const labelClass = "mb-1.5 block text-xs tracking-[.08em] text-[var(--ink-faint)]";

  return (
    <div className="min-h-screen bg-[var(--bg)] px-4 py-10 sm:px-6 lg:px-10 lg:py-14">
      <header className="mx-auto mb-10 max-w-[1180px] border-b border-[var(--line)] pb-7">
        <div className="mb-3 flex items-center gap-2 text-xs font-semibold uppercase tracking-[.22em] text-[var(--gold-deep)]">
          <Sparkles className="h-4 w-4" aria-hidden="true" />{t("eyebrow")}
        </div>
        <div className="flex flex-col justify-between gap-4 md:flex-row md:items-end">
          <div>
            <h1 className="font-[var(--serif)] text-3xl font-semibold tracking-[-.02em] text-[var(--ink)] sm:text-4xl">{t("title")}</h1>
            <p className="mt-3 max-w-[680px] text-sm leading-7 text-[var(--ink-soft)]">{t("sub")}</p>
          </div>
          <button type="button" onClick={() => setHistoryOpen(true)} className="inline-flex min-h-11 w-fit items-center gap-2 border border-[var(--gold)] px-4 text-sm font-semibold text-[var(--gold-deep)] transition hover:bg-[var(--gold-deep)] hover:text-[var(--bg-card)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--gold-deep)] focus-visible:ring-offset-2"><History className="h-4 w-4" aria-hidden="true" />{t("historyButton")}<span className="border-l border-current/25 pl-2 text-xs font-normal">{historyTotal}</span></button>
        </div>
      </header>

      <div className="mx-auto max-w-[860px]">
        <section aria-labelledby="chart-input-title" className="border border-[var(--line)] bg-[var(--bg-card)] p-5 sm:p-8 lg:p-10">
          <h2 id="chart-input-title" className="font-[var(--serif)] text-xl font-semibold text-[var(--ink)]">{t("inputTitle")}</h2>
          <p className="mt-2 border-b border-[var(--line-soft)] pb-5 text-xs leading-5 text-[var(--ink-faint)]">{t("inputNote")}</p>

          <div className="mt-7 space-y-7">
            <fieldset>
              <legend className="mb-2 text-sm font-semibold text-[var(--ink)]">{t("calendar")}</legend>
              <div className="grid grid-cols-2 border border-[var(--line)] p-1">
                {([0, 1] as const).map((value) => <button key={value} type="button" onClick={() => setCalendar(value)} className="min-h-10 text-sm font-semibold transition" style={{ background: calendar === value ? "var(--gold-deep)" : "transparent", color: calendar === value ? "var(--bg-card)" : "var(--ink-soft)" }}>{value === 0 ? t("solar") : t("lunar")}</button>)}
              </div>
            </fieldset>

            {calendar === 1 && <label className="flex items-center gap-3 text-sm text-[var(--ink-soft)]"><input type="checkbox" checked={leapMonth} onChange={(event) => setLeapMonth(event.target.checked)} className="h-4 w-4 accent-[var(--gold-deep)]" />{t("leapMonth")}</label>}

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              {[["year", year, setYear], ["month", month, setMonth], ["day", day, setDay]].map(([key, value, setter]) => <label key={key as string}><span className={labelClass}>{t(key as "year")}</span><input type="number" value={value as number} onChange={(event) => (setter as (value: number) => void)(numberValue(event.target.value))} className={fieldClass} /></label>)}
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <label><span className={labelClass}>{t("hour")}</span><input type="number" min="0" max="23" value={hour} onChange={(event) => setHour(numberValue(event.target.value))} className={fieldClass} /></label>
              <label><span className={labelClass}>{t("minute")}</span><input type="number" min="0" max="59" value={minute} onChange={(event) => setMinute(numberValue(event.target.value))} className={fieldClass} /></label>
            </div>

            <fieldset>
              <legend className="mb-3 text-sm font-semibold text-[var(--ink)]">{t("birthPlace")}</legend>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div><span className={labelClass}>{t("country")}</span><CountrySearchField id="free-chart-country" value={countryQuery} selectedCode={countryCode} options={countryOptions} placeholder={t("countrySearch")} ariaLabel={t("country")} emptyText={t("countryNoResults")} className={fieldClass} onChange={(query) => { setCountryQuery(query); setCountryCode(""); setSelectedCity(null); setRegionQuery(""); }} onSelect={(country) => { setCountryQuery(country.label); setCountryCode(country.code); setSelectedCity(null); setRegionQuery(""); }} /></div>
                <div><span className={labelClass}>{t("region")}</span><CitySearchField value={regionQuery} disabled={!selectedCountry} loading={cityLoading} loadingMore={cityLoadingMore} options={geoCities} total={cityTotal} selectedID={selectedCity?.id} placeholder={selectedCountry ? t("regionSearch") : t("selectCountry")} ariaLabel={t("region")} searchHint={t("citySearchHint")} loadingText={t("citySearching")} emptyText={t("cityNoResults")} statusText={t("cityCount",{shown:geoCities.length,total:cityTotal})} getName={(city)=>localizedName(city,dataLocale)} onChange={(query)=>{setRegionQuery(query);setSelectedCity(null);}} onSelect={(city)=>{setSelectedCity(city);setRegionQuery(localizedName(city,dataLocale));setLongitude(String(city.longitude));setLatitude(String(city.latitude));setTimezone(city.country_code === "CN" ? "Asia/Shanghai" : city.timezone_id);}} onLoadMore={()=>void loadMoreCities()} className={fieldClass} /></div>
              </div>
              <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                <label><span className={labelClass}>{t("longitude")}</span><input aria-describedby="free-chart-place-hint" value={longitude} inputMode="decimal" placeholder="116.4074" onChange={(event) => setLongitude(event.target.value)} className={fieldClass} /></label>
                <label><span className={labelClass}>{t("latitude")}</span><input aria-describedby="free-chart-place-hint" value={latitude} inputMode="decimal" placeholder="39.9042" onChange={(event) => setLatitude(event.target.value)} className={fieldClass} /></label>
              </div>
              <p id="free-chart-place-hint" className="mt-2 text-xs leading-5 text-[var(--ink-faint)]">{t("placeHint")}</p>
              <label className="mt-4 block"><span className={labelClass}>{t("timezone")}</span><input list="free-chart-timezones" value={timezone} onChange={(event)=>setTimezone(event.target.value)} placeholder={t("timezoneSearch")} autoComplete="off" className={fieldClass} /><datalist id="free-chart-timezones">{timezoneOptions.map((item)=><option key={item.value} value={item.value}>{item.label}</option>)}</datalist><span className="mt-2 block text-xs leading-5 text-[var(--ink-faint)]">{t("timezoneHint")}</span></label>
            </fieldset>

            <fieldset>
              <legend className="mb-2 text-sm font-semibold text-[var(--ink)]">{t("gender")}</legend>
              <div className="grid grid-cols-2 border border-[var(--line)] p-1">
                {([1, 0] as const).map((value) => <button key={value} type="button" onClick={() => setGender(value)} className="min-h-10 text-sm font-semibold transition" style={{ background: gender === value ? "var(--gold-deep)" : "transparent", color: gender === value ? "var(--bg-card)" : "var(--ink-soft)" }}>{value === 1 ? t("male") : t("female")}</button>)}
              </div>
            </fieldset>

            <button type="button" disabled={submitting} onClick={calculate} className="min-h-12 w-full bg-[var(--gold-deep)] font-[var(--serif)] text-base font-semibold text-[var(--bg-card)] transition hover:brightness-95 disabled:opacity-60 focus:outline-none focus:ring-2 focus:ring-[var(--gold-deep)] focus:ring-offset-2 focus:ring-offset-[var(--bg-card)]">{submitting?t("submitting"):t("submit")}</button>
          </div>
        </section>

      </div>
      <HistoryDialog open={historyOpen} onOpenChange={setHistoryOpen} records={history} total={historyTotal} page={historyPage} pageSize={5} loading={historyLoading} onPageChange={setHistoryPage} onDelete={(ids)=>void removeHistory(ids)} onView={(record)=>void viewHistory(record)} t={(key, values) => t(key as never, values as never)} />
      <ChartResultDialog open={resultOpen} onOpenChange={setResultOpen} record={activeRecord} t={(key) => t(key as never)} />
      {locationDialogOpen && <div className="fixed inset-0 z-[70] grid place-items-center bg-black/35 p-4" onMouseDown={(event)=>{if(event.target===event.currentTarget)setLocationDialogOpen(false)}}><section role="alertdialog" aria-modal="true" aria-labelledby="location-required-title" className="w-full max-w-sm border border-[var(--line)] bg-[var(--bg-card)] p-6 shadow-2xl"><h2 id="location-required-title" className="font-[var(--serif)] text-xl font-semibold text-[var(--ink)]">{t("locationDialogTitle")}</h2><p className="mt-3 text-sm leading-6 text-[var(--ink-soft)]">{t("locationRequired")}</p><div className="mt-6 flex justify-end"><button autoFocus type="button" onClick={()=>setLocationDialogOpen(false)} className="min-h-10 bg-[var(--gold-deep)] px-5 text-sm font-semibold text-[var(--bg-card)]">{t("confirm")}</button></div></section></div>}
    </div>
  );
}
