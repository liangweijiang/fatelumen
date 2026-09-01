"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { fetchReportFacts, fetchReportLLMCalls, fetchResourceDetail } from "@/lib/admin-api";
import { PromptPreviewPanel } from "./PromptPreviewPanel";
import { ValidationTracePanel } from "./ValidationTracePanel";

function JsonPanel({ title, value }: { title: string; value: unknown }) {
  return (
    <details className="rounded-md border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }} open>
      <summary className="cursor-pointer px-4 py-3 text-[15px] font-medium" style={{ color: "var(--ink)" }}>{title}</summary>
      <pre className="max-h-[520px] overflow-auto border-t p-4 text-[12px] leading-6" style={{ borderColor: "var(--line-soft)", color: "var(--ink-soft)" }}>{JSON.stringify(value, null, 2)}</pre>
    </details>
  );
}

export default function AdminReportDetailPage() {
  const { id } = useParams<{ id: string }>();
  const reportQ = useQuery({ queryKey: ["admin-report", id], queryFn: () => fetchResourceDetail("reports", id) });
  const factsQ = useQuery({ queryKey: ["admin-report-facts", id], queryFn: () => fetchReportFacts(id), retry: false });
  const callsQ = useQuery({ queryKey: ["admin-report-llm-calls", id], queryFn: () => fetchReportLLMCalls(id), retry: false });

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-[24px] font-medium" style={{ color: "var(--ink)" }}>报告详情 #{id}</h1><p className="mt-1 text-[13px]" style={{ color: "var(--ink-faint)" }}>基础数据为生成前固化的只读快照，可用于复现与排查。</p></div>
        <Link href="/admin/reports" className="rounded-md border px-4 py-2 text-[14px]" style={{ borderColor: "var(--line)", color: "var(--gold-deep)" }}>返回列表</Link>
      </div>

      <JsonPanel title="报告信息" value={reportQ.data ?? (reportQ.isLoading ? "加载中…" : "加载失败")} />

      <section className="space-y-3">
        <h2 className="text-[19px] font-medium" style={{ color: "var(--ink)" }}>基础数据详情</h2>
        {factsQ.isLoading ? <p>加载中…</p> : factsQ.isError ? <p style={{ color: "var(--ink-faint)" }}>该报告尚无基础数据快照；旧报告不会自动补写。</p> : <>
          <div className="grid gap-3 md:grid-cols-2"><JsonPanel title="输入快照" value={factsQ.data?.snapshot.input_snapshot} /><JsonPanel title="时间换算快照" value={factsQ.data?.snapshot.time_calculation_snapshot} /></div>
          <JsonPanel title="命盘快照" value={factsQ.data?.snapshot.chart_snapshot} />
          <JsonPanel title="完整确定性事实" value={factsQ.data?.snapshot.facts} />
        </>}
      </section>

      <PromptPreviewPanel reportId={id} />

      <ValidationTracePanel reportId={id} />

      <section className="space-y-3">
        <h2 className="text-[19px] font-medium" style={{ color: "var(--ink)" }}>DeepSeek 调用记录</h2>
        {callsQ.isLoading ? <p>加载中…</p> : (callsQ.data?.items.length ?? 0) === 0 ? <div className="rounded-md border p-5 text-[14px]" style={{ borderColor: "var(--line)", color: "var(--ink-faint)" }}>本阶段尚未接入新的调用追溯，暂无记录。</div> : callsQ.data?.items.map((call, index) => <JsonPanel key={String(call.id ?? index)} title={`调用 #${String(call.id ?? index + 1)}`} value={call} />)}
      </section>
    </div>
  );
}
