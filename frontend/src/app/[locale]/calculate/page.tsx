"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { createProfile, createReport } from "@/lib/api/endpoints";

const timezones = [
  { value: "Asia/Shanghai", label: "UTC+08:00 中国" },
  { value: "Asia/Tokyo", label: "UTC+09:00 日韩" },
  { value: "Europe/London", label: "UTC+00:00 格林尼治 (UTC)" },
  { value: "America/New_York", label: "UTC-05:00 美东" },
  { value: "America/Los_Angeles", label: "UTC-08:00 美西" },
];

export default function CalculatePage() {
  const t = useTranslations("calculate");
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";

  const [submitting, setSubmitting] = useState(false);

  const [calendarType, setCalendarType] = useState(0);
  const [gender, setGender] = useState(1);
  const [birthYear, setBirthYear] = useState(1990);
  const [birthMonth, setBirthMonth] = useState(1);
  const [birthDay, setBirthDay] = useState(1);
  const [birthHour, setBirthHour] = useState(12);
  const [birthMinute, setBirthMinute] = useState(0);
  const [isLeapMonth, setIsLeapMonth] = useState(0);
  const [timezone, setTimezone] = useState("Asia/Shanghai");
  const [displayName, setDisplayName] = useState("");
  const [depth, setDepth] = useState<"quick" | "deep">("deep");

  async function handleSubmit() {
    setSubmitting(true);
    try {
      const profile = await createProfile({
        calendar_type: calendarType,
        gender,
        birth_year: birthYear,
        birth_month: birthMonth,
        birth_day: birthDay,
        birth_hour: birthHour,
        birth_minute: birthMinute,
        is_leap_month: isLeapMonth,
        timezone,
        display_name: displayName || undefined,
      });

      if (depth === "quick") {
        router.push(`/${locale}/reading/${profile.id}`);
        return;
      }

      const report = await createReport({
        profile_id: profile.id,
        locale,
      });

      router.push(`/${locale}/reports/${report.id}`);
    } catch {
      toast.error(t("error"));
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
                onChange={(e) => setBirthYear(Number(e.target.value))}
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
                onChange={(e) => setBirthMonth(Number(e.target.value))}
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
                onChange={(e) => setBirthDay(Number(e.target.value))}
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
                onChange={(e) => setBirthHour(Number(e.target.value))}
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
                onChange={(e) => setBirthMinute(Number(e.target.value))}
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
            <div className="mb-6 flex items-center gap-3">
              <label
                className="text-[14px] tracking-[.3px]"
                style={{ color: "var(--ink)" }}
              >
                {t("leapMonth")}
              </label>
              <button
                type="button"
                onClick={() => setIsLeapMonth(isLeapMonth ? 0 : 1)}
                className="rounded-full px-4 py-1.5 text-[13px] font-semibold transition-all"
                style={{
                  background: isLeapMonth ? "var(--gold)" : "var(--bg)",
                  color: isLeapMonth ? "var(--bg-card)" : "var(--ink-soft)",
                  border: `1px solid ${isLeapMonth ? "var(--gold)" : "var(--line)"}`,
                }}
              >
                {isLeapMonth ? "✓" : "—"}
              </button>
            </div>
          )}

          {/* Timezone */}
          <div className="mb-6">
            <label
              className="mb-2 block text-[13px] font-semibold tracking-[.4px]"
              style={{ color: "var(--ink)" }}
            >
              {t("timezone")}
            </label>
            <select
              value={timezone}
              onChange={(e) => setTimezone(e.target.value)}
              className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
              style={{
                background: "var(--bg)",
                borderColor: "var(--line)",
                color: "var(--ink)",
              }}
            >
              {timezones.map((tz) => (
                <option key={tz.value} value={tz.value}>
                  {tz.label}
                </option>
              ))}
            </select>
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

          {/* Display name */}
          <div className="mb-2">
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
              placeholder=""
              className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2"
              style={{
                background: "var(--bg)",
                borderColor: "var(--line)",
                color: "var(--ink)",
              }}
            />
          </div>
        </div>

        {/* Depth cards */}
        <div className="mb-8">
          <label
            className="mb-3 block text-[13px] font-semibold tracking-[.4px]"
            style={{ color: "var(--ink)" }}
          >
            {t("depth")}
          </label>
          <div className="grid grid-cols-2 gap-4">
            {/* Quick card */}
            <button
              type="button"
              onClick={() => setDepth("quick")}
              className="relative rounded-2xl border p-5 text-left transition-all"
              style={{
                background: depth === "quick" ? "var(--bg-card)" : "var(--bg)",
                borderColor: depth === "quick" ? "var(--gold)" : "var(--line)",
                boxShadow: depth === "quick" ? "0 0 0 2px var(--gold)" : "none",
              }}
            >
              <p
                className="mb-1 text-[16px] font-semibold"
                style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
              >
                {t("quickCard")}
              </p>
              <p className="text-[12px]" style={{ color: "var(--ink-faint)" }}>
                {t("quickDesc")}
              </p>
            </button>

            {/* Deep card */}
            <button
              type="button"
              onClick={() => setDepth("deep")}
              className="relative rounded-2xl border p-5 text-left transition-all"
              style={{
                background: depth === "deep" ? "var(--bg-card)" : "var(--bg)",
                borderColor: depth === "deep" ? "var(--gold)" : "var(--line)",
                boxShadow: depth === "deep" ? "0 0 0 2px var(--gold)" : "none",
              }}
            >
              <span
                className="absolute right-3 top-3 rounded-full px-2.5 py-0.5 text-[11px] font-semibold tracking-[.3px]"
                style={{ background: "var(--gold-deep)", color: "var(--bg-card)" }}
              >
                {t("deepBadge")}
              </span>
              <p
                className="mb-1 text-[16px] font-semibold"
                style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
              >
                {t("deepCard")}
              </p>
              <p className="text-[12px]" style={{ color: "var(--ink-faint)" }}>
                {t("deepDesc")}
              </p>
            </button>
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
