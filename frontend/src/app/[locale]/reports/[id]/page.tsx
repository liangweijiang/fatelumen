"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getReport, getChart } from "@/lib/api/endpoints";
import type { Report, Chart } from "@/types/api";
import api from "@/lib/api/client";
import CheckoutBlock from "@/components/report/CheckoutBlock";

export default function ReportPage() {
  const t = useTranslations("report");
  const params = useParams();
  const id = Number(params?.id);
  const idValid = Number.isInteger(id) && id > 0;

  const [report, setReport] = useState<Report | null>(null);
  const [chart, setChart] = useState<Chart | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [pdfLoading, setPdfLoading] = useState(false);

  const poll = useCallback(async () => {
    if (!idValid) {
      setError(true);
      setLoading(false);
      return true;
    }
    try {
      const r = await getReport(id);
      setReport(r);
      if (r.status === "done" || r.status === "failed") {
        setLoading(false);
        setError(r.status === "failed");
        if (r.status === "done" && r.chart_id) {
          try {
            const c = await getChart(r.chart_id);
            setChart(c);
          } catch {
            // 命盘拉取失败不阻断报告展示
          }
        }
        return true;
      }
      return false;
    } catch {
      setError(true);
      setLoading(false);
      return true;
    }
  }, [id, idValid]);

  useEffect(() => {
    let alive = true;
    let timer: ReturnType<typeof setTimeout>;

    async function loop() {
      while (alive) {
        const done = await poll();
        if (done) break;
        await new Promise((r) => {
          timer = setTimeout(r, 3000);
        });
      }
    }

    loop();
    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [poll]);

  async function handleDownloadPdf() {
    if (!report) return;
    setPdfLoading(true);
    try {
      const { data } = await api.post(`/reports/${report.id}/pdf`);
      const pdfUrl = (data as Record<string, unknown>)?.data
        ? ((data as Record<string, unknown>).data as Record<string, string>)?.pdf_url
        : (data as Record<string, string>)?.pdf_url;
      if (pdfUrl) {
        window.open(pdfUrl, "_blank");
      }
      toast.success(t("downloadPdf"));
    } catch {
      toast.error(t("failed"));
    } finally {
      setPdfLoading(false);
    }
  }

  // Loading state
  if (loading && !error) {
    return (
      <div
        className="flex min-h-screen flex-col items-center justify-center gap-4"
        style={{ background: "var(--bg)" }}
      >
        <div
          className="h-10 w-10 animate-spin rounded-full border-[3px]"
          style={{ borderColor: "var(--line)", borderTopColor: "var(--gold)" }}
        />
        <p
          className="text-[15px] tracking-[.3px]"
          style={{ fontFamily: "var(--serif-d)", color: "var(--ink-soft)" }}
        >
          {t("loading")}
        </p>
      </div>
    );
  }

  // Failed state
  if (error || !report || report.status === "failed") {
    return (
      <div
        className="flex min-h-screen flex-col items-center justify-center gap-4 px-5"
        style={{ background: "var(--bg)" }}
      >
        <p
          className="text-[18px] font-semibold"
          style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
        >
          {t("failed")}
        </p>
        {report?.error_msg && (
          <p className="text-[13px]" style={{ color: "var(--ink-faint)" }}>
            {report.error_msg}
          </p>
        )}
        <button
          type="button"
          onClick={() => window.location.reload()}
          className="rounded-full px-6 py-2.5 text-[14px] font-semibold"
          style={{ background: "var(--gold-deep)", color: "var(--bg-card)" }}
        >
          {t("retry")}
        </button>
      </div>
    );
  }

  // Done state — render full report
  const content = report.content;

  return (
    <div
      className="relative min-h-screen px-5 py-10 md:px-10 md:py-16"
      style={{ background: "var(--bg)" }}
    >
      <div className="mx-auto max-w-[760px]">
        {/* Header */}
        <div className="mb-10 text-center">
          <div className="mb-4 flex items-center justify-center gap-3">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="40" height="40">
              <rect width="32" height="32" rx="9" fill="oklch(25% 0.018 58)" />
              <g transform="translate(4.36 6.4) scale(0.6)">
                <path d="M21.5 5.5a11 11 0 1 0 0 21 8.5 8.5 0 0 1 0-21z" fill="oklch(66% 0.115 84)" />
                <circle cx="22.5" cy="13" r="2.4" fill="oklch(66% 0.115 84)" />
                <g stroke="oklch(66% 0.115 84)" strokeWidth="1.4" strokeLinecap="round">
                  <path d="M22.5 7.2v2M22.5 16.8v2M16.7 13h2M26.3 13h2M18.6 9.1l1.4 1.4M25 9.1l-1.4 1.4" />
                </g>
              </g>
            </svg>
            <span
              className="text-[22px] font-semibold tracking-[.3px]"
              style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
            >
              FateLumen
            </span>
          </div>
          {content?.summary_line && (
            <p
              className="text-[20px] leading-relaxed"
              style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
            >
              {content.summary_line}
            </p>
          )}
        </div>

        {/* Overview: Pillars */}
        {report.chart_id && chart?.chart_data && (
          <div
            className="mb-10 rounded-2xl border p-8"
            style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}
          >
            <h2
              className="mb-1 text-center text-[18px] font-semibold"
              style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
            >
              {t("pillars")}
            </h2>
            <div className="mb-5 text-center text-[12px]" style={{ color: "var(--ink-faint)" }}>
              {chart.chart_data.meta?.solar_date}
              {chart.chart_data.meta?.lunar_date ? ` · 农历 ${chart.chart_data.meta.lunar_date}` : ""}
              {(chart.chart_data.meta as { zodiac?: string })?.zodiac ? ` · 属${(chart.chart_data.meta as { zodiac?: string }).zodiac}` : ""}
            </div>
            <div className="grid grid-cols-4 gap-3 text-center">
              {([
                { pos: "hour", label: "时柱" },
                { pos: "day", label: "日柱" },
                { pos: "month", label: "月柱" },
                { pos: "year", label: "年柱" },
              ] as const).map(({ pos, label }) => {
                const p = chart.chart_data.pillars[pos];
                const wx: Record<string, string> = {
                  "木": "#5c7060", "火": "#b8473e", "土": "#b89048", "金": "#9a9486", "水": "#3f5a6b",
                };
                return (
                  <div key={pos} className="flex flex-col items-center">
                    <span className="mb-2 text-[11px] tracking-[1px]" style={{ color: "var(--ink-faint)" }}>
                      {label}
                    </span>
                    <span
                      className="text-[34px] leading-[1.1] font-semibold"
                      style={{ fontFamily: "var(--serif-c)", color: wx[p?.stem_element] || "var(--ink)" }}
                    >
                      {p?.stem || "—"}
                    </span>
                    <span
                      className="text-[34px] leading-[1.1] font-semibold"
                      style={{ fontFamily: "var(--serif-c)", color: wx[p?.branch_element] || "var(--ink)" }}
                    >
                      {p?.branch || "—"}
                    </span>
                  </div>
                );
              })}
            </div>
            {chart.chart_data.five_elements_count && (
              <div className="mt-5 flex items-center justify-center gap-5">
                {(["木", "火", "土", "金", "水"] as const).map((e) => {
                  const wx: Record<string, string> = {
                    "木": "#5c7060", "火": "#b8473e", "土": "#b89048", "金": "#9a9486", "水": "#3f5a6b",
                  };
                  return (
                    <span key={e} className="flex items-center gap-1 text-[12px]" style={{ color: wx[e] }}>
                      <span className="inline-block h-2 w-2 rounded-full" style={{ background: wx[e] }} />
                      {e} {chart.chart_data.five_elements_count[e] ?? 0}
                    </span>
                  );
                })}
              </div>
            )}
            {content?.summary && (
              <p className="mt-6 text-[14px] leading-relaxed" style={{ color: "var(--ink-soft)" }}>
                {content.summary}
              </p>
            )}
          </div>
        )}

        {/* Yearly Fortune */}
        {content?.yearly_fortune && content.yearly_fortune.length > 0 && (
          <div
            className="mb-10 rounded-2xl border p-8"
            style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}
          >
            <h2
              className="mb-5 text-[18px] font-semibold"
              style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
            >
              {t("yearlyTitle")}
            </h2>
            <div className="flex flex-col gap-3">
              {content.yearly_fortune.map((y) => (
                <div
                  key={y.year}
                  className="rounded-xl border-l-4 py-3 pl-4 pr-3"
                  style={{ borderColor: "var(--gold)", background: "var(--bg-soft)" }}
                >
                  <div className="mb-1 text-[15px] font-semibold" style={{ color: "var(--gold-deep)" }}>
                    {y.year}
                  </div>
                  <div className="text-[14px] leading-relaxed" style={{ color: "var(--ink-soft)" }}>
                    {y.note}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Chapters */}
        {content?.chapters && content.chapters.length > 0 && content.chapters.map((ch) => (
          <div
            key={ch.no}
            className="mb-8 rounded-2xl border p-8"
            style={{
              background: "var(--bg-card)",
              borderColor: "var(--line)",
            }}
          >
            <div className="mb-2 flex items-center gap-2">
              <span
                className="rounded-full px-3 py-1 text-[11px] font-semibold tracking-[.3px]"
                style={{ background: "var(--gold-soft)", color: "var(--ink)" }}
              >
                {t("chapterPrefix")}{ch.no}{t("chapterSuffix")}
              </span>
              {ch.tags?.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full px-2.5 py-0.5 text-[11px]"
                  style={{ background: "var(--gold-soft)", color: "var(--ink)" }}
                >
                  {tag}
                </span>
              ))}
            </div>
            <h2
              className="mb-4 text-[20px] font-semibold"
              style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
            >
              {ch.title}
            </h2>
            {ch.strength_score != null && (
              <div
                className="mb-4 inline-block rounded-full border px-4 py-1.5 text-[13px] font-semibold"
                style={{ borderColor: "var(--gold)", color: "var(--ink)" }}
              >
                {t("strength")}: {ch.strength_score}
              </div>
            )}
            <p className="text-[14px] leading-[1.85] text-justify" style={{ color: "var(--ink-soft)" }}>
              {ch.body}
            </p>

            {/* Cycles table */}
            {ch.cycles && ch.cycles.length > 0 && (
              <div className="mt-6">
                <h3
                  className="mb-3 text-[15px] font-semibold"
                  style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
                >
                  {t("cycleTitle")}
                </h3>
                <table className="w-full border-collapse">
                  <thead>
                    <tr style={{ background: "var(--bg)" }}>
                      <th className="px-4 py-2.5 text-left text-[12px]" style={{ color: "var(--ink-faint)" }}>
                        {t("yearTitle")}
                      </th>
                      <th className="px-4 py-2.5 text-left text-[12px]" style={{ color: "var(--ink-faint)" }}>
                        —
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {ch.cycles.map((c, i) => (
                      <tr key={i} style={{ borderBottom: `1px solid var(--line)` }}>
                        <td className="px-4 py-2.5 text-[13px] font-semibold" style={{ color: "var(--gold-deep)" }}>
                          {c.ganzhi} ({c.start_year})
                        </td>
                        <td className="px-4 py-2.5 text-[13px]" style={{ color: "var(--ink-soft)" }}>
                          {c.note}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Years table */}
            {ch.years && ch.years.length > 0 && (
              <div className="mt-6">
                <h3
                  className="mb-3 text-[15px] font-semibold"
                  style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
                >
                  {t("yearTitle")}
                </h3>
                <table className="w-full border-collapse">
                  <thead>
                    <tr style={{ background: "var(--bg)" }}>
                      <th className="px-4 py-2.5 text-left text-[12px]" style={{ color: "var(--ink-faint)" }}>
                        {t("yearTitle")}
                      </th>
                      <th className="px-4 py-2.5 text-left text-[12px]" style={{ color: "var(--ink-faint)" }}>
                        —
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {ch.years.map((y, i) => (
                      <tr key={i} style={{ borderBottom: `1px solid var(--line)` }}>
                        <td className="px-4 py-2.5 text-[13px] font-semibold" style={{ color: "var(--gold-deep)" }}>
                          {y.year} {y.ganzhi}
                        </td>
                        <td className="px-4 py-2.5 text-[13px]" style={{ color: "var(--ink-soft)" }}>
                          {y.note}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        ))}

        <CheckoutBlock reportId={id} />

        {/* Bottom action bar */}
        <div className="sticky bottom-4 mt-10">
          <div
            className="mx-auto flex max-w-[400px] items-center justify-center rounded-full border p-2"
            style={{
              background: "var(--bg-card)",
              borderColor: "var(--line)",
              boxShadow: "0 8px 30px rgba(0,0,0,0.12)",
            }}
          >
            <button
              type="button"
              onClick={handleDownloadPdf}
              disabled={pdfLoading}
              className="w-full rounded-full py-3 text-[15px] font-semibold tracking-[.3px] transition-all"
              style={{
                fontFamily: "var(--serif-d)",
                background: pdfLoading ? "var(--ink-faint)" : "var(--gold-deep)",
                color: "var(--bg-card)",
                cursor: pdfLoading ? "not-allowed" : "pointer",
              }}
            >
              {pdfLoading ? t("generating") : t("downloadPdf")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
