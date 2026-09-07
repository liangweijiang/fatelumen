"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, ArrowLeft, FileText, ListChecks, ScrollText, Workflow } from "lucide-react";
import { fetchAdminReportOverview, fetchAdminReportResult } from "@/lib/admin-api";
import { CallTracePanel } from "./CallTracePanel";
import { ChapterTracePanel } from "./ChapterTracePanel";
import { FactsTracePanel } from "./FactsTracePanel";
import { ReportOverviewPanel } from "./ReportOverviewPanel";

type Tab = "overview" | "chapters" | "calls" | "facts" | "result";
const tabs: { key: Tab; label: string; icon: typeof Workflow }[] = [
  { key: "overview", label: "执行总览", icon: Workflow }, { key: "chapters", label: "十章正文", icon: ListChecks },
  { key: "calls", label: "模型调用", icon: ScrollText },
  { key: "facts", label: "冻结基础数据", icon: FileText }, { key: "result", label: "最终报告", icon: FileText },
];

function FinalReportPanel({ reportId }: { reportId: string }) {
  const result = useQuery({ queryKey: ["admin-report-result", reportId], queryFn: () => fetchAdminReportResult(reportId), retry: false });
  if (result.isLoading) return <p className="border p-6 text-sm" style={{ borderColor: "var(--line)" }}>正在按需读取最终报告…</p>;
  if (result.isError || !result.data) return <p role="alert" className="border p-6 text-sm text-red-800" style={{ borderColor: "var(--line)" }}>最终报告尚未生成或读取失败。</p>;
  return <div className="space-y-4"><div className="grid gap-3 border p-4 text-sm sm:grid-cols-2 lg:grid-cols-4" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><p>语言：{result.data.locale}</p><p>章节：{result.data.content.chapters?.length ?? 0}</p><p>渲染版本：{result.data.render_version}</p><p>PDF：{result.data.pdf_url ? "已生成" : "尚未接入"}</p><p className="break-all font-mono text-xs sm:col-span-2 lg:col-span-4">内容哈希：{result.data.content_hash}</p></div><div className="space-y-5">{result.data.content.chapters?.map(chapter => <article key={chapter.key} className="border p-6" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><p className="text-xs text-amber-800">第 {chapter.no} 章</p><h2 className="mt-1 text-xl">{chapter.title}</h2><p className="mt-4 whitespace-pre-wrap text-[15px] leading-8">{chapter.body}</p></article>)}</div></div>;
}

export default function AdminReportDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [tab, setTab] = useState<Tab>("overview");
  const overview = useQuery({
    queryKey: ["admin-report-overview", id], queryFn: () => fetchAdminReportOverview(id), retry: false,
    refetchInterval: query => query.state.data && !["completed", "failed"].includes(query.state.data.report.status) ? 5000 : false,
  });
  return <div className="space-y-5">
    <header className="flex flex-wrap items-end justify-between gap-3"><div><Link href="/admin/reports" className="inline-flex items-center gap-1 text-sm text-amber-800"><ArrowLeft size={15} />返回报告列表</Link><h1 className="mt-3 text-[24px] font-medium">报告详情 #{id}</h1><p className="mt-1 text-[13px]" style={{ color: "var(--ink-faint)" }}>完整调用链只读追溯；Prompt、输出和事实大字段仅在点击时加载。</p></div>{overview.data && <div className="text-right text-sm"><p>{overview.data.report.chapter_succeeded}/{overview.data.report.chapter_total} 章完成</p><p className="mt-1" style={{ color: "var(--ink-faint)" }}>{overview.data.report.status}</p></div>}</header>
    {overview.isError && <div role="alert" className="flex items-center gap-2 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800"><AlertTriangle size={16} />报告总览读取失败。</div>}
    <nav className="flex overflow-x-auto border-b" style={{ borderColor: "var(--line)" }} aria-label="报告详情栏目">{tabs.map(item => { const Icon = item.icon; return <button type="button" key={item.key} onClick={() => setTab(item.key)} aria-current={tab === item.key ? "page" : undefined} className={`inline-flex shrink-0 items-center gap-2 border-b-2 px-4 py-3 text-sm ${tab === item.key ? "border-amber-700 text-amber-900" : "border-transparent"}`}><Icon size={15} />{item.label}</button>; })}</nav>
    {overview.isLoading ? <div className="space-y-3" aria-label="正在加载报告总览"><div className="h-24 animate-pulse bg-black/[0.05]" /><div className="h-52 animate-pulse bg-black/[0.05]" /></div> : overview.data && <>{tab === "overview" && <ReportOverviewPanel data={overview.data} reportId={id} />}{tab === "chapters" && <ChapterTracePanel reportId={id} locale={overview.data.report.locale} />}{tab === "calls" && <CallTracePanel reportId={id} />}{tab === "facts" && <FactsTracePanel reportId={id} />}{tab === "result" && <FinalReportPanel reportId={id} />}</>}
  </div>;
}
