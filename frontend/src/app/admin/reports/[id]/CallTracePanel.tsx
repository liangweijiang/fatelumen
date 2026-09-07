"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { CheckCircle2, ChevronLeft, ChevronRight, Clock3, X, XCircle } from "lucide-react";
import { fetchReportCallValidation, fetchReportChapters, fetchReportLLMCall, fetchReportLLMCalls } from "@/lib/admin-api";
import { ValidationResultView } from "./ValidationResultView";

const statusText: Record<string, string> = { running: "调用中", succeeded: "成功采用", rejected: "校验拒绝", failed: "调用失败" };
const formatTime = (value?: string) => value ? new Date(value).toLocaleString("zh-CN", { hour12: false }) : "—";
const pageSize = 5;

type CallDetail = Awaited<ReturnType<typeof fetchReportLLMCall>>;

function attemptResult(item: { status: string; schema_valid: boolean; validation_status: string; error_code?: string }) {
  if (item.status === "succeeded") return "校验通过并采用";
  if (item.status === "failed") return "模型调用失败";
  const code = item.error_code ?? "";
  if (code.startsWith("FMT_")) return "JSON 格式错误";
  if (code.startsWith("SCH_") || !item.schema_valid) return "Schema 结构错误";
  if (code.startsWith("FACT_")) return "冻结事实冲突";
  if (code.startsWith("LANG_") || code.startsWith("TERM_")) return "语言或术语不符";
  return item.validation_status === "failed" ? "校验拒绝" : statusText[item.status] ?? item.status;
}

function CallDetailDialog({ callId, detail, reportId, onClose }: { callId: number; detail: UseQueryResult<CallDetail, Error>; reportId: string; onClose: () => void }) {
  const closeRef = useRef<HTMLButtonElement>(null);
  const validation = useQuery({ queryKey: ["admin-report-call-validation", reportId, callId], queryFn: () => fetchReportCallValidation(reportId, callId), retry: false });
  useEffect(() => { closeRef.current?.focus(); const handler = (event: KeyboardEvent) => event.key === "Escape" && onClose(); window.addEventListener("keydown", handler); return () => window.removeEventListener("keydown", handler); }, [onClose]);
  return <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" role="presentation" onMouseDown={event => event.target === event.currentTarget && onClose()}><section role="dialog" aria-modal="true" aria-labelledby="call-detail-title" className="max-h-[90vh] w-full max-w-6xl overflow-hidden border shadow-2xl" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><header className="flex items-center justify-between border-b px-5 py-4" style={{ borderColor: "var(--line)" }}><div><h2 id="call-detail-title" className="text-xl">调用 #{callId} 追溯</h2><p className="mt-1 text-xs" style={{ color: "var(--ink-faint)" }}>读取生成时保存的数据，不重新调用模型。</p></div><button ref={closeRef} type="button" onClick={onClose} aria-label="关闭调用详情" className="btn-ghost p-2"><X size={19} /></button></header><div className="max-h-[78vh] overflow-y-auto">{detail.isLoading ? <p className="p-6 text-sm">正在读取大字段…</p> : detail.isError ? <p role="alert" className="p-6 text-sm text-red-800">调用详情读取失败</p> : detail.data && <div className="space-y-3 p-5"><div className="grid gap-3 border p-4 sm:grid-cols-2 lg:grid-cols-4" style={{ borderColor: "var(--line-soft)" }}><p>章节：{detail.data.chapter.title}</p><p>状态：{statusText[detail.data.attempt.status] ?? detail.data.attempt.status}</p><p>追踪号：<span className="font-mono text-xs">{detail.data.attempt.trace_id}</span></p><p>Token：{detail.data.attempt.total_tokens ?? "未上报"}</p><p>路由：#{detail.data.attempt.route_no}</p><p>模型：{detail.data.attempt.provider} / {detail.data.attempt.model}</p><p>耗时：{(detail.data.attempt.duration_ms / 1000).toFixed(1)} 秒</p><p>最终处理：{statusText[detail.data.attempt.status] ?? detail.data.attempt.status}</p></div><ValidationResultView result={validation.data?.validation_result} emptyText={validation.isLoading ? "正在读取调用后校验…" : "本次调用没有校验结果"} />{[["请求参数", detail.data.payload.request_parameters], ["最终 Prompt", detail.data.payload.request_prompt], ["原始输出", detail.data.payload.raw_output], ["解析输出", detail.data.payload.parsed_output], ["Schema 错误", detail.data.payload.schema_errors]].map(([title, value]) => <details key={String(title)} className="border" style={{ borderColor: "var(--line-soft)" }}><summary className="cursor-pointer px-4 py-3 text-sm">{String(title)}</summary><pre className="max-h-[30rem] overflow-auto whitespace-pre-wrap border-t p-4 text-xs leading-6" style={{ borderColor: "var(--line-soft)" }}>{typeof value === "string" ? value : JSON.stringify(value, null, 2)}</pre></details>)}</div>}</div></section></div>;
}

export function CallTracePanel({ reportId }: { reportId: string }) {
  const [chapterId, setChapterId] = useState<number>();
  const [page, setPage] = useState(1);
  const [callId, setCallId] = useState<number>();
  const chapters = useQuery({ queryKey: ["admin-report-chapters", reportId], queryFn: () => fetchReportChapters(reportId), retry: false });
  useEffect(() => { if (!chapterId && chapters.data?.items[0]) setChapterId(chapters.data.items[0].id); }, [chapterId, chapters.data]);
  const calls = useQuery({ queryKey: ["admin-report-calls", reportId, chapterId, page], queryFn: () => fetchReportLLMCalls(reportId, page, pageSize, chapterId), enabled: !!chapterId, retry: false });
  const detail = useQuery({ queryKey: ["admin-report-call", reportId, callId], queryFn: () => fetchReportLLMCall(reportId, callId!), enabled: !!callId, retry: false });
  const selected = chapters.data?.items.find(item => item.id === chapterId);

  function selectChapter(id: number) { setChapterId(id); setPage(1); setCallId(undefined); }

  return <div className="space-y-5">
    <div className="grid gap-5 xl:grid-cols-[18rem_minmax(0,1fr)]">
      <aside className="border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }} aria-label="按章节查看模型调用">
        <h2 className="border-b px-4 py-3 text-lg" style={{ borderColor: "var(--line)" }}>十章调用记录</h2>
        {chapters.isLoading ? <p className="p-4 text-sm">正在加载十章…</p> : chapters.isError ? <p role="alert" className="p-4 text-sm text-red-800">章节读取失败</p> : chapters.data?.items.map(item => <button type="button" key={item.id} onClick={() => selectChapter(item.id)} aria-current={chapterId === item.id ? "page" : undefined} className={`flex w-full items-center gap-3 border-b px-4 py-3 text-left ${chapterId === item.id ? "bg-amber-50" : "hover:bg-black/[0.025]"}`} style={{ borderColor: "var(--line-soft)" }}>
          <span className="w-6 font-mono text-xs">{String(item.chapter_no).padStart(2, "0")}</span><span className="min-w-0 flex-1"><span className="block truncate text-sm">{item.title}</span><span className="mt-1 block text-xs" style={{ color: "var(--ink-faint)" }}>{item.attempt_count} 次调用</span></span>{item.status === "succeeded" ? <CheckCircle2 size={16} className="text-emerald-800" /> : item.status === "failed" ? <XCircle size={16} className="text-red-800" /> : <Clock3 size={16} className="text-amber-800" />}
        </button>)}
      </aside>
      <section className="min-w-0">
        <header className="mb-3 flex flex-wrap items-end justify-between gap-2"><div><p className="text-xs text-amber-800">第 {selected?.chapter_no ?? "—"} 章</p><h2 className="mt-1 text-xl">{selected?.title ?? "请选择章节"}</h2></div><p className="text-sm" style={{ color: "var(--ink-faint)" }}>共 {calls.data?.total ?? selected?.attempt_count ?? 0} 次调用</p></header>
        <div className="overflow-x-auto border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><table className="w-full min-w-[820px] text-left text-sm"><thead><tr className="border-b" style={{ borderColor: "var(--line)" }}><th className="px-4 py-3">尝试</th><th className="px-4 py-3">路由 / 模型</th><th className="px-4 py-3">结果</th><th className="px-4 py-3">Token</th><th className="px-4 py-3">耗时</th><th className="px-4 py-3">开始时间</th><th className="px-4 py-3">详情</th></tr></thead><tbody className="divide-y" style={{ borderColor: "var(--line-soft)" }}>
          {calls.isLoading ? <tr><td colSpan={7} className="p-8 text-center">正在加载本章调用记录…</td></tr> : calls.isError ? <tr><td colSpan={7} className="p-8 text-center text-red-800">本章调用记录读取失败</td></tr> : calls.data?.items.length ? calls.data.items.map(item => <tr key={item.id}><td className="px-4 py-3">第 {item.attempt_no} 次</td><td className="px-4 py-3">路由 #{item.route_no}<p className="mt-1 text-xs">{item.provider} / <span className="font-mono">{item.model}</span></p></td><td className="px-4 py-3"><span className={item.status === "succeeded" ? "text-emerald-800" : "text-red-800"}>{attemptResult(item)}</span>{item.error_summary && <p className="mt-1 max-w-64 text-xs leading-5"><span className="font-medium">原因：</span>{item.error_summary}</p>}<p className="mt-1 text-xs" style={{ color: "var(--ink-faint)" }}>后续动作：{item.status === "succeeded" ? "冻结为本章最终结果" : "按冻结策略自动尝试下一次"}</p></td><td className="px-4 py-3">{item.total_tokens ?? "未上报"}</td><td className="px-4 py-3">{(item.duration_ms / 1000).toFixed(1)} 秒</td><td className="whitespace-nowrap px-4 py-3">{formatTime(item.started_at)}</td><td className="px-4 py-3"><button type="button" onClick={() => setCallId(item.id)} className="text-amber-800 underline underline-offset-4">查看详情</button></td></tr>) : <tr><td colSpan={7} className="p-10 text-center" style={{ color: "var(--ink-faint)" }}>本章暂无调用记录</td></tr>}
        </tbody></table></div>
        <footer className="mt-3 flex items-center justify-between text-sm"><span>第 {page} 页 · 每页 {pageSize} 条</span><div className="flex gap-2"><button type="button" disabled={page === 1} onClick={() => { setPage(value => value - 1); setCallId(undefined); }} className="btn-ghost inline-flex items-center gap-1 border px-3 py-2 disabled:opacity-40"><ChevronLeft size={15} />上一页</button><button type="button" disabled={!calls.data || page * calls.data.page_size >= calls.data.total} onClick={() => { setPage(value => value + 1); setCallId(undefined); }} className="btn-ghost inline-flex items-center gap-1 border px-3 py-2 disabled:opacity-40">下一页<ChevronRight size={15} /></button></div></footer>
      </section>
    </div>
    {callId && <CallDetailDialog reportId={reportId} callId={callId} detail={detail} onClose={() => setCallId(undefined)} />}
  </div>;
}
