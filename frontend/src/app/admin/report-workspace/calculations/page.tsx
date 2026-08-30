"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Pencil, Plus, RotateCcw, Search, Trash2 } from "lucide-react";
import {
  batchDeleteCalculations, CalculationArchive, CalculationInput, createCalculation,
  deleteCalculation, fetchCalculations, recalculateCalculation, updateCalculation,
} from "@/lib/admin-api";
import { CalculationFormDialog, emptyCalculation } from "./CalculationFormDialog";

type Notice = { tone: "success" | "error"; message: string } | null;

export default function CalculationArchivePage() {
  const [rows, setRows] = useState<CalculationArchive[]>([]);
  const [total, setTotal] = useState(0), [page, setPage] = useState(1);
  const [query, setQuery] = useState(""), [selected, setSelected] = useState<number[]>([]);
  const [open, setOpen] = useState(false), [editingID, setEditingID] = useState<number | null>(null);
  const [form, setForm] = useState<CalculationInput>(emptyCalculation), [busy, setBusy] = useState(false);
  const [recalculatingID, setRecalculatingID] = useState<number | null>(null);
  const [highlightedID, setHighlightedID] = useState<number | null>(null);
  const [notice, setNotice] = useState<Notice>(null), [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setError("");
      const result = await fetchCalculations(query, page, 10);
      setRows(result.items); setTotal(result.total);
    } catch { setError("计算档案暂时无法读取，请确认后端服务已启动。"); }
  }, [query, page]);
  useEffect(() => { void load(); }, [load]);

  const submit = async (value: CalculationInput) => {
    setBusy(true); setError("");
    try {
      if (editingID) await updateCalculation(editingID, value); else await createCalculation(value);
      setOpen(false); setEditingID(null); setForm(emptyCalculation); setPage(1); await load();
    } catch (caught: unknown) {
      const message = (caught as { response?: { data?: { msg?: string } } })?.response?.data?.msg;
      setError(message || "计算失败，请检查出生地、经纬度和时区。"); throw caught;
    } finally { setBusy(false); }
  };

  const recalculate = async (archive: CalculationArchive) => {
    if (recalculatingID !== null) return;
    const startedAt = performance.now();
    setNotice(null); setError(""); setRecalculatingID(archive.id);
    try {
      const updated = await recalculateCalculation(archive.id);
      await load();
      const seconds = ((performance.now() - startedAt) / 1000).toFixed(2);
      setHighlightedID(archive.id);
      setNotice({ tone: "success", message: `“${archive.name}”重新计算完成，已生成第 ${updated.latest_version_no} 版，用时 ${seconds} 秒。` });
      window.setTimeout(() => setHighlightedID((id) => id === archive.id ? null : id), 3000);
    } catch (caught: unknown) {
      const message = (caught as { response?: { data?: { msg?: string } } })?.response?.data?.msg;
      setNotice({ tone: "error", message: message || `“${archive.name}”重新计算失败，请稍后重试。` });
    } finally { setRecalculatingID(null); }
  };

  const remove = async (ids: number[]) => {
    if (!ids.length || !confirm(`确定删除 ${ids.length} 条计算档案及其全部版本？`)) return;
    if (ids.length === 1) await deleteCalculation(ids[0]); else await batchDeleteCalculations(ids);
    setSelected([]); await load();
  };

  const pages = Math.max(1, Math.ceil(total / 10));
  return <section>
    <header className="mb-5 flex flex-wrap items-end justify-between gap-4"><div><h2 className="text-2xl">计算档案</h2><p className="mt-1 text-sm" style={{ color: "var(--ink-soft)" }}>出生资料先计算并形成不可变版本；Prompt 与正式报告只引用同一份确定性结果。</p></div><button className="btn-gold flex items-center gap-2 px-4 py-2 text-sm" onClick={() => { setEditingID(null); setForm(emptyCalculation); setOpen(true); }}><Plus size={16} />新增计算档案</button></header>
    <div className="mb-4 flex gap-2"><label className="flex flex-1 items-center gap-2 border px-3 py-2" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><Search size={15} /><input value={query} onChange={(event) => { setQuery(event.target.value); setPage(1); }} className="flex-1 bg-transparent text-sm outline-none" placeholder="搜索档案名称" /></label></div>
    {error && <div role="alert" className="mb-4 border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-800">{error}</div>}
    {notice && <div role="status" className={`mb-4 border px-4 py-3 text-sm ${notice.tone === "success" ? "border-emerald-300 bg-emerald-50 text-emerald-900" : "border-red-300 bg-red-50 text-red-800"}`}>{notice.message}</div>}
    {selected.length > 0 && <div className="mb-3 flex items-center gap-4 border px-4 py-2 text-sm" style={{ borderColor: "var(--line)", background: "var(--gold-soft)" }}><span>已选择 {selected.length} 项</span><button className="flex items-center gap-1 text-red-800" onClick={() => void remove(selected)}><Trash2 size={14} />批量删除</button></div>}
    <div className="overflow-x-auto border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><table className="w-full min-w-[1120px] text-left text-sm"><thead><tr className="border-b" style={{ borderColor: "var(--line)", background: "var(--bg-soft)" }}><th className="px-3 py-3"></th>{["档案名称", "出生时间", "出生地 / 时区", "状态", "最新版本", "更新时间", "操作"].map((label) => <th className="px-3 py-3 font-medium" key={label}>{label}</th>)}</tr></thead><tbody>
      {rows.length === 0 ? <tr><td colSpan={8} className="px-4 py-16 text-center" style={{ color: "var(--ink-faint)" }}>暂无计算档案</td></tr> : rows.map((archive) => { const input = archive.input, calculating = recalculatingID === archive.id; return <tr className="border-b transition-colors" style={{ borderColor: "var(--line-soft)", background: highlightedID === archive.id ? "var(--gold-soft)" : undefined }} key={archive.id}>
        <td className="px-3"><input aria-label={`选择${archive.name}`} type="checkbox" checked={selected.includes(archive.id)} onChange={(event) => setSelected((value) => event.target.checked ? [...value, archive.id] : value.filter((id) => id !== archive.id))} /></td>
        <td className="px-3 py-4"><b>{archive.name}</b><div className="text-xs opacity-60">#{archive.id}</div></td>
        <td className="px-3">{input.year}-{String(input.month).padStart(2, "0")}-{String(input.day).padStart(2, "0")} {String(input.hour).padStart(2, "0")}:{String(input.minute).padStart(2, "0")}</td>
        <td className="px-3"><div>{input.location?.display_name || "经纬度输入"}</div><small>{input.location?.timezone_id}</small></td>
        <td className="px-3">{calculating ? "重新计算中…" : archive.status === "ready" ? "计算完成" : archive.status === "failed" ? "计算失败" : "计算中"}</td>
        <td className="px-3">第 {archive.latest_version_no} 版</td><td className="px-3 text-xs">{new Date(archive.updated_at).toLocaleString("zh-CN")}</td>
        <td className="px-3"><div className="flex flex-wrap items-center gap-x-3 gap-y-2 whitespace-nowrap"><Link href={`/admin/report-workspace/calculations/${archive.id}`} className="text-sm" style={{ color: "var(--gold-deep)" }}>查看</Link><button className="inline-flex items-center gap-1 text-sm" disabled={calculating} onClick={() => { setEditingID(archive.id); setForm(archive.input); setOpen(true); }}><Pencil size={14} />修改</button><button className="inline-flex items-center gap-1 text-sm disabled:cursor-wait disabled:opacity-50" disabled={recalculatingID !== null} onClick={() => void recalculate(archive)}><RotateCcw size={14} className={calculating ? "animate-spin" : ""} />{calculating ? "计算中" : "重新计算"}</button><button className="inline-flex items-center gap-1 text-sm text-red-800" disabled={calculating} onClick={() => void remove([archive.id])}><Trash2 size={14} />删除</button></div></td>
      </tr>; })}
    </tbody></table></div>
    <footer className="mt-4 flex justify-between text-sm"><span>共 {total} 条 · 第 {page}/{pages} 页</span><div className="flex gap-2"><button className="btn-ghost px-3 py-1" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</button><button className="btn-ghost px-3 py-1" disabled={page >= pages} onClick={() => setPage(page + 1)}>下一页</button></div></footer>
    {open && <CalculationFormDialog initial={form} editing={editingID !== null} busy={busy} serverError={error} onClose={() => setOpen(false)} onSubmit={submit} />}
  </section>;
}
