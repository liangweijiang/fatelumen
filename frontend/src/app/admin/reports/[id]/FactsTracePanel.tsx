"use client";

import { useEffect, useRef, useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { Braces, X } from "lucide-react";
import { fetchReportFacts } from "@/lib/admin-api";
import { displayCode, displayDateTime, displayNumber, displayUTCOffset } from "@/lib/bazi-display";
import { displayDiagnostic } from "@/lib/bazi-display/diagnostics";
import { displayArrayItemLabel, displayFieldLabel } from "@/lib/bazi-display/field-labels";

function valueCategory(key: string, path: readonly string[]): string | undefined {
  if (["element", "day_element", "stem_element", "branch_element", "favorable", "unfavorable", "taboo", "enemy", "neutral", "primary", "secondary"].includes(key)) return "element";
  if (key === "ten_god") return "ten_god";
  if (key === "yin_yang") return "yin_yang";
  if (key === "root_level") return "root_level";
  if ((key === "type" || key === "code") && path.some(part => part.includes("relation"))) return "relation";
  if ((key === "type" || key === "code") && path.some(part => part.includes("pattern"))) return "pattern";
  return undefined;
}

function scalar(key: string, value: unknown, path: readonly string[] = []): string {
  if (value === null || value === undefined || value === "") return "—";
  if (["local_civil_time", "local_standard_time", "mean_solar_time", "true_solar_time"].includes(key)) return displayDateTime(value, "zh");
  if (["historical_utc_offset_seconds", "standard_utc_offset_seconds"].includes(key)) return displayUTCOffset(value, "zh");
  if (key === "calendar_type") return Number(value) === 1 ? "农历" : "公历";
  if (key === "gender") return Number(value) === 2 ? "坤 · 女" : "乾 · 男";
  if (typeof value === "number") return displayNumber(value);
  if (["reason", "reasons", "reject_reasons"].includes(key)) return displayDiagnostic(value, "zh");
  return displayCode(value, "zh", valueCategory(key, path));
}

const isNested = (value: unknown) => value !== null && typeof value === "object";

function ScalarRows({ value, path = [] }: { value: Record<string, unknown>; path?: string[] }) {
  const entries = Object.entries(value).filter(([, item]) => !isNested(item));
  if (!entries.length) return null;
  return <dl>{entries.map(([key, item]) => <div key={key} className="grid gap-1 border-b px-5 py-3 last:border-b-0 sm:grid-cols-[13rem_minmax(0,1fr)]" style={{ borderColor: "var(--line-soft)" }}><dt className="text-sm" style={{ color: "var(--ink-faint)" }}>{displayFieldLabel(key, path)}</dt><dd className="break-words text-sm leading-6">{scalar(key, item, path)}</dd></div>)}</dl>;
}

function DataNode({ name, value, depth = 0, path = [] }: { name: string; value: unknown; depth?: number; path?: string[] }) {
  if (!isNested(value)) return <div className="grid gap-1 border-b px-5 py-3 sm:grid-cols-[13rem_minmax(0,1fr)]" style={{ borderColor: "var(--line-soft)" }}><span className="text-sm" style={{ color: "var(--ink-faint)" }}>{displayFieldLabel(name, path)}</span><span className="text-sm">{scalar(name, value, path)}</span></div>;
  const array = Array.isArray(value);
  const record = array ? undefined : value as Record<string, unknown>;
  const size = array ? value.length : Object.keys(record!).length;
  const defaultOpen = depth === 0 && ["meta", "pillars", "day_master", "element_strength", "day_master_strength", "useful_god"].includes(name);
  const currentPath = [...path, name];
  return <details className="border-b last:border-b-0" style={{ borderColor: "var(--line-soft)" }} open={defaultOpen}><summary className="cursor-pointer px-5 py-3 text-sm"><span className="font-medium">{displayFieldLabel(name, path)}</span><span className="ml-2 text-xs" style={{ color: "var(--ink-faint)" }}>{array ? `${size} 条记录` : `${size} 项数据`}</span></summary><div className="border-t pl-3 sm:pl-6" style={{ borderColor: "var(--line-soft)" }}>{array ? (value.length ? value.map((item, index) => isNested(item) ? <DataNode key={index} name={displayArrayItemLabel(name, index, path)} value={item} depth={depth + 1} path={currentPath} /> : <div key={index} className="border-b px-5 py-3 text-sm last:border-b-0" style={{ borderColor: "var(--line-soft)" }}>{scalar(name, item, path)}</div>) : <p className="px-5 py-3 text-sm" style={{ color: "var(--ink-faint)" }}>暂无数据</p>) : <><ScalarRows value={record!} path={currentPath} />{Object.entries(record!).filter(([, item]) => isNested(item)).map(([key, item]) => <DataNode key={key} name={key} value={item} depth={depth + 1} path={currentPath} />)}</>}</div></details>;
}

function ReadablePanel({ title, value, onRaw, compact = false }: { title: string; value: unknown; onRaw: () => void; compact?: boolean }) {
  const record = value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {};
  const scalarCount = Object.values(record).filter(item => !isNested(item)).length;
  const groups = Object.entries(record).filter(([, item]) => isNested(item));
  return <details className="border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><summary className="cursor-pointer border-b px-5 py-4" style={{ borderColor: "var(--line)" }}><span className="text-lg">{title}</span><span className="ml-3 text-xs" style={{ color: "var(--ink-faint)" }}>{compact ? `${scalarCount} 项基础字段` : `${scalarCount} 项摘要 · ${groups.length} 组分层数据`}</span></summary><div className="flex justify-end border-b px-5 py-3" style={{ borderColor: "var(--line-soft)" }}><button type="button" onClick={onRaw} className="btn-ghost inline-flex items-center gap-2 border px-3 py-2 text-sm"><Braces size={15} />查看原始 JSON</button></div><ScalarRows value={record} />{groups.map(([key, item]) => <DataNode key={key} name={key} value={item} />)}</details>;
}

function RawDialog({ title, value, onClose }: { title: string; value: unknown; onClose: () => void }) {
  const closeRef = useRef<HTMLButtonElement>(null);
  useEffect(() => { closeRef.current?.focus(); const keydown = (event: KeyboardEvent) => event.key === "Escape" && onClose(); window.addEventListener("keydown", keydown); return () => window.removeEventListener("keydown", keydown); }, [onClose]);
  return <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" role="presentation" onMouseDown={event => event.target === event.currentTarget && onClose()}><section role="dialog" aria-modal="true" aria-labelledby="raw-json-title" className="max-h-[88vh] w-full max-w-5xl overflow-hidden border shadow-2xl" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><header className="flex items-center justify-between border-b px-5 py-4" style={{ borderColor: "var(--line)" }}><div><h2 id="raw-json-title" className="text-xl">{title} · 原始 JSON</h2><p className="mt-1 text-xs" style={{ color: "var(--ink-faint)" }}>报告生成时保存的原始冻结数据，仅供追溯。</p></div><button ref={closeRef} type="button" onClick={onClose} aria-label="关闭原始JSON" className="btn-ghost p-2"><X size={18} /></button></header><pre className="max-h-[72vh] overflow-auto whitespace-pre-wrap p-5 text-xs leading-6">{JSON.stringify(value, null, 2)}</pre></section></div>;
}

export function FactsTracePanel({ reportId }: { reportId: string }) {
  const [raw, setRaw] = useState<{ title: string; value: unknown }>();
  const specs = [{ section: "input" as const, title: "输入资料", compact: true }, { section: "time" as const, title: "时间换算", compact: true }, { section: "chart" as const, title: "命盘快照", compact: false }, { section: "interpretation" as const, title: "确定性事实", compact: false }];
  const queries = useQueries({ queries: specs.map(spec => ({ queryKey: ["admin-report-facts", reportId, spec.section], queryFn: () => fetchReportFacts(reportId, spec.section), retry: false })) });
  if (queries.some(query => query.isLoading)) return <p role="status" className="border p-6 text-sm" style={{ borderColor: "var(--line)" }}>正在分区读取冻结基础数据…</p>;
  if (queries.some(query => query.isError || !query.data)) return <p role="alert" className="border p-6 text-sm text-red-800" style={{ borderColor: "var(--line)" }}>基础数据快照读取失败。</p>;
  const panels = specs.map((spec, index) => ({ title: spec.title, compact: spec.compact, value: queries[index].data!.value }));
  return <div className="space-y-4"><p className="text-sm" style={{ color: "var(--ink-faint)" }}>基础字段直接展开；命盘、十神、关系、运势和规则等复杂数据按层级折叠。所有内容均来自报告生成时的冻结快照。</p>{panels.map(panel => <ReadablePanel key={panel.title} {...panel} onRaw={() => setRaw(panel)} />)}{raw && <RawDialog {...raw} onClose={() => setRaw(undefined)} />}</div>;
}
