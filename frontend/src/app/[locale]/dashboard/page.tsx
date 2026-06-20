"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { getMe, listReports } from "@/lib/api/endpoints";
import type { User, Report } from "@/types/api";
import Link from "next/link";

export default function DashboardPage() {
  const t = useTranslations("dashboard");
  const router = useRouter();
  const params = useParams();
  const locale = (params?.locale as string) || "en";

  const [user, setUser] = useState<User | null>(null);
  const [reports, setReports] = useState<Report[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    async function load() {
      try {
        const [me, reps] = await Promise.all([getMe(), listReports()]);
        if (alive) {
          setUser(me);
          setReports(reps);
        }
      } catch {
        // silently handle
      } finally {
        if (alive) setLoading(false);
      }
    }
    load();
    return () => {
      alive = false;
    };
  }, []);

  function statusChip(status: string) {
    if (status === "done") return <Chip color="done" label={t("status_done")} />;
    if (status === "failed") return <Chip color="failed" label={t("status_failed")} />;
    return <Chip color="pending" label={t("status_pending")} />;
  }

  function Chip({ color, label }: { color: string; label: string }) {
    const colors: Record<string, { bg: string; text: string }> = {
      pending: { bg: "var(--gold-soft)", text: "var(--ink)" },
      processing: { bg: "var(--gold-soft)", text: "var(--ink)" },
      done: { bg: "var(--gold)", text: "var(--bg-card)" },
      failed: { bg: "oklch(80% 0.04 10)", text: "var(--bg-card)" },
    };
    const c = colors[color] || colors.pending;
    return (
      <span
        className="rounded-full px-3 py-1 text-[11px] font-semibold tracking-[.3px]"
        style={{ background: c.bg, color: c.text }}
      >
        {label}
      </span>
    );
  }

  return (
    <div
      className="relative min-h-screen px-5 py-10 md:px-10 md:py-16"
      style={{ background: "var(--bg)" }}
    >
      <div className="mx-auto max-w-[760px]">
        {/* Header */}
        <div className="mb-10 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1
              className="text-[28px] font-semibold tracking-[-0.3px]"
              style={{ fontFamily: "var(--serif-d)", color: "var(--ink)" }}
            >
              {t("title")}
            </h1>
            {user && (
              <p className="mt-1 text-[14px]" style={{ color: "var(--ink-soft)" }}>
                {user.name || (user.email ? user.email.split("@")[0] : "用户")}
                {user.credits != null && (
                  <span className="ml-3" style={{ color: "var(--gold-deep)" }}>
                    {t("credits")}: {user.credits}
                  </span>
                )}
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={() => router.push(`/${locale}/calculate`)}
            className="rounded-full px-6 py-3 text-[15px] font-semibold tracking-[.3px] transition-all"
            style={{
              fontFamily: "var(--serif-d)",
              background: "var(--gold-deep)",
              color: "var(--bg-card)",
            }}
          >
            {t("newReading")}
          </button>
        </div>

        {/* Reports list */}
        {loading ? (
          <div className="flex justify-center py-20">
            <div
              className="h-8 w-8 animate-spin rounded-full border-[3px]"
              style={{ borderColor: "var(--line)", borderTopColor: "var(--gold)" }}
            />
          </div>
        ) : reports.length === 0 ? (
          <div className="py-20 text-center">
            <p className="text-[15px]" style={{ color: "var(--ink-faint)" }}>
              {t("empty")}
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {reports.map((r) => (
              <Link
                key={r.id}
                href={`/${locale}/reports/${r.id}`}
                className="flex items-center justify-between rounded-2xl border p-5 transition-all hover:shadow-md"
                style={{
                  background: "var(--bg-card)",
                  borderColor: "var(--line)",
                }}
              >
                <div className="flex items-center gap-4">
                  {statusChip(r.status)}
                  <div>
                    <p className="text-[14px] font-semibold" style={{ color: "var(--ink)" }}>
                      #{r.id}
                    </p>
                    <p className="text-[12px]" style={{ color: "var(--ink-faint)" }}>
                      {r.locale.toUpperCase()} · {new Date(r.created_at).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                <span
                  className="text-[13px] font-semibold tracking-[.3px]"
                  style={{ color: "var(--gold-deep)" }}
                >
                  {t("viewReport")}
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
