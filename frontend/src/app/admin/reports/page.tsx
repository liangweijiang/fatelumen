"use client";

import Link from "next/link";
import { FormEvent, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, ChevronLeft, ChevronRight, RefreshCw, Search } from "lucide-react";
import { fetchAdminReports } from "@/lib/admin-api";

const statusText: Record<string, string> = {
  pending: "待处理", preflighting: "预检中", generating: "生成中", assembling: "组装中",
  rendering: "整理结果", completed: "已完成", failed: "失败", deleting: "删除中",
};
const localeText: Record<string, string> = { zh: "中文", en: "英文", ja: "日文", ko: "韩文" };
const formatTime = (value?: string) => value ? new Date(value).toLocaleString("zh-CN", { hour12: false }) : "—";

export default function AdminReportsPage() {
  const [draft, setDraft] = useState({ userId: "", status: "", locale: "", from: "", to: "" });
  const [filters, setFilters] = useState(draft);
  const [cursors, setCursors] = useState<string[]>([""]);
  const cursor = cursors[cursors.length - 1];
  const params = useMemo(() => ({
    status: filters.status || undefined, locale: filters.locale || undefined,
    user_id: filters.userId ? Number(filters.userId) : undefined,
    created_from: filters.from ? new Date(`${filters.from}T00:00:00`).toISOString() : undefined,
    created_to: filters.to ? new Date(`${filters.to}T23:59:59.999`).toISOString() : undefined,
    cursor: cursor || undefined, page_size: 20,
  }), [cursor, filters]);
  const reports = useQuery({
    queryKey: ["admin-full-reports", params], queryFn: () => fetchAdminReports(params),
    refetchInterval: query => query.state.data?.items.some(item => !["completed", "failed"].includes(item.status)) ? 5000 : false,
  });

  function applyFilters(event: FormEvent) { event.preventDefault(); setFilters(draft); setCursors([""]); }
  return <div className="space-y-5">
    <header className="flex flex-wrap items-end justify-between gap-3">
      <div><h1 className="text-[24px] font-medium">报告管理</h1><p className="mt-1 text-[13px]" style={{ color: "var(--ink-faint)" }}>查看完整十章的生成进度、模型调用、校验记录与最终正文。终态报告只读。</p></div>
      <button type="button" onClick={() => reports.refetch()} className="btn-ghost inline-flex items-center gap-2 border px-3 py-2 text-sm"><RefreshCw size={15} />刷新</button>
    </header>
    <form onSubmit={applyFilters} className="grid gap-3 border p-4 md:grid-cols-2 xl:grid-cols-[1fr_11rem_10rem_11rem_11rem_auto]" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}>
      <label><span className="mb-1 block text-xs">用户 ID</span><input inputMode="numeric" value={draft.userId} onChange={e => setDraft({ ...draft, userId: e.target.value.replace(/\D/g, "") })} placeholder="输入用户 ID" className="w-full border px-3 py-2 text-sm" /></label>
      <label><span className="mb-1 block text-xs">状态</span><select value={draft.status} onChange={e => setDraft({ ...draft, status: e.target.value })} className="w-full border px-3 py-2 text-sm"><option value="">全部状态</option>{Object.entries(statusText).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
      <label><span className="mb-1 block text-xs">语言</span><select value={draft.locale} onChange={e => setDraft({ ...draft, locale: e.target.value })} className="w-full border px-3 py-2 text-sm"><option value="">全部语言</option>{Object.entries(localeText).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
      <label><span className="mb-1 block text-xs">开始日期</span><input type="date" value={draft.from} onChange={e => setDraft({ ...draft, from: e.target.value })} className="w-full border px-3 py-2 text-sm" /></label>
      <label><span className="mb-1 block text-xs">结束日期</span><input type="date" value={draft.to} onChange={e => setDraft({ ...draft, to: e.target.value })} className="w-full border px-3 py-2 text-sm" /></label>
      <button type="submit" className="btn-gold inline-flex self-end items-center justify-center gap-2 px-4 py-2 text-sm"><Search size={15} />查询</button>
    </form>
    {reports.isError && <div role="alert" className="flex items-center gap-2 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800"><AlertTriangle size={16} />报告列表读取失败，请检查后端服务。</div>}
    <div className="overflow-x-auto border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}>
      <table className="w-full min-w-[1060px] text-left text-sm">
        <thead><tr className="border-b" style={{ borderColor: "var(--line)" }}><th className="px-4 py-3">报告</th><th className="px-4 py-3">用户 / 档案</th><th className="px-4 py-3">语言</th><th className="px-4 py-3">状态</th><th className="px-4 py-3">十章进度</th><th className="px-4 py-3">创建时间</th><th className="px-4 py-3">完成时间</th><th className="px-4 py-3">操作</th></tr></thead>
        <tbody className="divide-y" style={{ borderColor: "var(--line-soft)" }}>
          {reports.isLoading ? Array.from({ length: 5 }).map((_, index) => <tr key={index}><td colSpan={8} className="px-4 py-3"><div className="h-5 animate-pulse bg-black/[0.05]" /></td></tr>) : reports.data?.items.length ? reports.data.items.map(item => {
            const active = !["completed", "failed"].includes(item.status);
            return <tr key={item.id}>
              <td className="px-4 py-3"><p className="font-medium">#{item.id}</p><p className="mt-1 font-mono text-xs" style={{ color: "var(--ink-faint)" }}>{item.public_id}</p></td>
              <td className="px-4 py-3"><p>用户 #{item.user_id}</p><p className="mt-1 text-xs" style={{ color: "var(--ink-faint)" }}>{item.profile_name || `档案 #${item.profile_id ?? "—"}`}</p></td>
              <td className="px-4 py-3">{localeText[item.locale] ?? item.locale}</td>
              <td className="px-4 py-3"><span className="inline-flex items-center gap-2"><span className={`h-2 w-2 ${active ? "animate-pulse bg-amber-600" : item.status === "failed" ? "bg-red-700" : "bg-emerald-700"}`} />{statusText[item.status] ?? item.status}</span>{item.error_summary && <p className="mt-1 max-w-52 truncate text-xs text-red-800" title={item.error_summary}>{item.error_summary}</p>}</td>
              <td className="px-4 py-3"><p>{item.chapter_succeeded}/{item.chapter_total}</p><div className="mt-2 h-1.5 w-24 overflow-hidden bg-black/[0.08]"><div className="h-full bg-amber-700" style={{ width: `${Math.min(100, item.chapter_total ? item.chapter_succeeded / item.chapter_total * 100 : 0)}%` }} /></div></td>
              <td className="whitespace-nowrap px-4 py-3">{formatTime(item.created_at)}</td><td className="whitespace-nowrap px-4 py-3">{formatTime(item.completed_at)}</td>
              <td className="px-4 py-3"><Link href={`/admin/reports/${item.id}`} className="text-amber-800 underline underline-offset-4">查看详情</Link></td>
            </tr>;
          }) : <tr><td colSpan={8} className="px-4 py-16 text-center" style={{ color: "var(--ink-faint)" }}>没有符合条件的报告</td></tr>}
        </tbody>
      </table>
    </div>
    <footer className="flex items-center justify-between text-sm"><span style={{ color: "var(--ink-faint)" }}>第 {cursors.length} 页 · 每页最多 20 条</span><div className="flex gap-2"><button type="button" disabled={cursors.length === 1} onClick={() => setCursors(value => value.slice(0, -1))} className="btn-ghost inline-flex items-center gap-1 border px-3 py-2 disabled:opacity-40"><ChevronLeft size={15} />上一页</button><button type="button" disabled={!reports.data?.has_more || !reports.data.next_cursor} onClick={() => setCursors(value => [...value, reports.data!.next_cursor])} className="btn-ghost inline-flex items-center gap-1 border px-3 py-2 disabled:opacity-40">下一页<ChevronRight size={15} /></button></div></footer>
  </div>;
}
