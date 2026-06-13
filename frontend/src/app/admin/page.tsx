"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { fetchStats, type Stats } from "@/lib/admin-api";

export default function AdminDashboard() {
  const router = useRouter();
  const [stats, setStats] = useState<Stats | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    fetchStats().then(setStats).catch((e: unknown) => {
      setErr((e as { message?: string })?.message || "无权查阅命阁");
    });
  }, []);

  if (err) {
    return (
      <div className="mx-auto max-w-md text-center" style={{ color: "var(--ink-soft)" }}>
        <p className="mb-4">{err}</p>
        <button type="button" onClick={() => router.replace("/login")} className="underline" style={{ color: "var(--gold-deep)" }}>返回登入</button>
      </div>
    );
  }

  if (!stats) return <p style={{ color: "var(--ink-faint)" }}>推演中…</p>;

  const cards = [
    { label: "命主总数", value: stats.users.total, sub: `今日新立 ${stats.users.today_new}` },
    { label: "订单总数", value: stats.orders.total, sub: `状态分布 ${Object.keys(stats.orders.by_status).length} 类` },
    { label: "进帐合计", value: `${(stats.revenue.total_cents / 100).toFixed(2)} ${stats.revenue.currency}`, sub: `今日 ${(stats.revenue.today_cents / 100).toFixed(2)}` },
    { label: "命盘报告", value: stats.reports.total, sub: `已启 ${stats.reports.unlocked_count}` },
  ];

  return (
    <div>
      <h1 className="mb-6 font-[var(--serif)] text-[26px] font-medium" style={{ color: "var(--ink)" }}>命阁概览</h1>
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
        {cards.map((c) => (
          <div key={c.label} className="rounded-xl border p-6" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
            <p className="text-[14px] font-light" style={{ color: "var(--ink-soft)" }}>{c.label}</p>
            <p className="mt-2 font-[var(--serif)] text-[28px] font-medium" style={{ color: "var(--ink)" }}>{c.value}</p>
            <p className="mt-1 text-[13px]" style={{ color: "var(--ink-faint)" }}>{c.sub}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
