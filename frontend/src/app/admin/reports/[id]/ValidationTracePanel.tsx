"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, CheckCircle2, ChevronRight, X, XCircle } from "lucide-react";
import {
  fetchReportChapter,
  fetchReportChapters,
  fetchReportValidation,
  fetchReportValidations,
  ReportValidationResult,
  ReportValidationRule,
} from "@/lib/admin-api";

const statusLabel: Record<string, string> = { passed: "通过", failed: "未通过", pending: "等待校验", succeeded: "已完成", running: "生成中", rejected: "已拒绝" };

function Status({ passed, label }: { passed: boolean; label?: string }) {
  return <span className={`inline-flex items-center gap-1.5 text-sm ${passed ? "text-emerald-800" : "text-red-800"}`}>{passed ? <CheckCircle2 size={15} /> : <XCircle size={15} />}{label ?? (passed ? "通过" : "未通过")}</span>;
}

function Rules({ result }: { result?: ReportValidationResult }) {
  if (!result?.rules?.length) return <p className="py-8 text-center text-sm" style={{ color: "var(--ink-faint)" }}>暂无逐项校验记录</p>;
  return <div className="divide-y" style={{ borderColor: "var(--line-soft)" }}>{result.rules.map((rule, index) => <Rule key={`${rule.code}-${index}`} rule={rule} />)}</div>;
}

function Rule({ rule }: { rule: ReportValidationRule }) {
  return <article className="grid gap-3 py-4 md:grid-cols-[12rem_1fr]">
    <div><Status passed={rule.passed} /><h4 className="mt-1 font-medium">{rule.name}</h4><code className="mt-1 block text-xs" style={{ color: "var(--ink-faint)" }}>{rule.code}</code></div>
    <div className="space-y-2 text-sm leading-6">
      {rule.message && <p>{rule.message}</p>}
      {(rule.expected || rule.actual) && <dl className="grid gap-2 sm:grid-cols-2"><div><dt style={{ color: "var(--ink-faint)" }}>期望</dt><dd>{rule.expected || "—"}</dd></div><div><dt style={{ color: "var(--ink-faint)" }}>实际</dt><dd>{rule.actual || "—"}</dd></div></dl>}
      {!!rule.affected_chapters?.length && <p><span style={{ color: "var(--ink-faint)" }}>涉及章节：</span>{rule.affected_chapters.map(no => `第 ${no} 章`).join("、")}</p>}
      {!!rule.evidence_refs?.length && <details><summary className="cursor-pointer" style={{ color: "var(--gold-deep)" }}>查看证据路径</summary><ul className="mt-2 space-y-1 font-mono text-xs">{rule.evidence_refs.map(ref => <li key={ref}>{ref}</li>)}</ul></details>}
    </div>
  </article>;
}

function DetailDialog({ title, loading, result, onClose }: { title: string; loading: boolean; result?: ReportValidationResult; onClose: () => void }) {
  const closeRef = useRef<HTMLButtonElement>(null);
  useEffect(() => { closeRef.current?.focus(); const close = (event: KeyboardEvent) => event.key === "Escape" && onClose(); window.addEventListener("keydown", close); return () => window.removeEventListener("keydown", close); }, [onClose]);
  return <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" role="presentation" onMouseDown={event => event.target === event.currentTarget && onClose()}>
    <section role="dialog" aria-modal="true" aria-labelledby="validation-dialog-title" className="max-h-[88vh] w-full max-w-4xl overflow-hidden border shadow-2xl" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}>
      <header className="flex items-start justify-between border-b px-5 py-4" style={{ borderColor: "var(--line)" }}><div><h3 id="validation-dialog-title" className="text-xl">{title}</h3><p className="mt-1 text-sm" style={{ color: "var(--ink-faint)" }}>校验结果来自报告生成时保存的冻结记录，不按当前规则重新计算。</p></div><button ref={closeRef} type="button" onClick={onClose} aria-label="关闭校验详情" className="btn-ghost p-2"><X size={18} /></button></header>
      <div className="max-h-[72vh] overflow-y-auto px-5">{loading ? <p className="py-10 text-center text-sm">正在读取校验记录…</p> : <Rules result={result} />}</div>
    </section>
  </div>;
}

export function ValidationTracePanel({ reportId }: { reportId: string }) {
  const [selection, setSelection] = useState<{ type: "report" | "chapter"; id: number; title: string }>();
  const validations = useQuery({ queryKey: ["admin-report-validations", reportId], queryFn: () => fetchReportValidations(reportId), retry: false });
  const chapters = useQuery({ queryKey: ["admin-report-chapters", reportId], queryFn: () => fetchReportChapters(reportId), retry: false });
  const reportDetail = useQuery({ queryKey: ["admin-report-validation", reportId, selection?.id], queryFn: () => fetchReportValidation(reportId, selection!.id), enabled: selection?.type === "report", retry: false });
  const chapterDetail = useQuery({ queryKey: ["admin-report-chapter", reportId, selection?.id], queryFn: () => fetchReportChapter(reportId, selection!.id), enabled: selection?.type === "chapter", retry: false });
  const result = selection?.type === "report" ? reportDetail.data?.payload.validation_result : chapterDetail.data?.payload.validation_result;

  return <section aria-labelledby="validation-trace-title" className="space-y-4">
    <div><h2 id="validation-trace-title" className="text-[19px] font-medium">校验记录</h2><p className="mt-1 text-[13px]" style={{ color: "var(--ink-faint)" }}>先看十章汇总结论，再按章节查看生成后的六层校验依据。</p></div>
    {(validations.isError || chapters.isError) && <div role="alert" className="flex items-center gap-2 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800"><AlertTriangle size={16} />校验记录读取失败，请确认数据库迁移和后端服务已更新。</div>}
    <div className="grid gap-4 xl:grid-cols-[minmax(20rem,0.8fr)_minmax(32rem,1.7fr)]">
      <div className="border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><h3 className="border-b px-4 py-3 font-medium" style={{ borderColor: "var(--line-soft)" }}>报告汇总校验</h3><div className="divide-y" style={{ borderColor: "var(--line-soft)" }}>{validations.isLoading ? <p className="p-4 text-sm">加载中…</p> : validations.data?.items.length ? validations.data.items.map(run => <button type="button" key={run.id} onClick={() => setSelection({ type: "report", id: run.id, title: `第 ${run.round_no} 轮报告汇总校验` })} className="flex w-full items-center justify-between gap-3 px-4 py-4 text-left hover:bg-black/[0.025]"><div><Status passed={run.status === "passed"} label={statusLabel[run.status] ?? run.status} /><p className="mt-1 text-xs" style={{ color: "var(--ink-faint)" }}>{run.validator_version} · {new Date(run.started_at).toLocaleString("zh-CN")}</p>{run.error_summary && <p className="mt-2 text-sm">{run.error_summary}</p>}</div><ChevronRight size={17} /></button>) : <p className="p-4 text-sm" style={{ color: "var(--ink-faint)" }}>报告尚未进入十章汇总校验。</p>}</div></div>
      <div className="border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><h3 className="border-b px-4 py-3 font-medium" style={{ borderColor: "var(--line-soft)" }}>十章生成后校验</h3><div className="grid sm:grid-cols-2">{chapters.isLoading ? <p className="p-4 text-sm">加载中…</p> : chapters.data?.items.map(chapter => <button type="button" key={chapter.id} onClick={() => setSelection({ type: "chapter", id: chapter.id, title: `第 ${chapter.chapter_no} 章 · ${chapter.title}` })} className="flex items-center justify-between gap-3 border-b px-4 py-3 text-left hover:bg-black/[0.025] sm:odd:border-r" style={{ borderColor: "var(--line-soft)" }}><div><p className="font-medium">{String(chapter.chapter_no).padStart(2, "0")}　{chapter.title}</p><p className="mt-1 text-xs" style={{ color: "var(--ink-faint)" }}>{statusLabel[chapter.validation_status] ?? chapter.validation_status} · 尝试 {chapter.attempt_count} 次</p></div><Status passed={chapter.validation_status === "passed"} label="" /></button>)}</div></div>
    </div>
    {selection && <DetailDialog title={selection.title} loading={reportDetail.isFetching || chapterDetail.isFetching} result={result} onClose={() => setSelection(undefined)} />}
  </section>;
}
