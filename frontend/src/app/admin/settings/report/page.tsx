"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Save } from "lucide-react";
import { fetchAdminReportSettings, saveAdminReportSettings } from "@/lib/admin-api";

export default function ReportSettingsPage() {
  const queryClient = useQueryClient();
  const settings = useQuery({ queryKey: ["admin-report-settings"], queryFn: fetchAdminReportSettings, retry: false });
  const [count, setCount] = useState(3);
  const [saved, setSaved] = useState(false);
  useEffect(() => { if (settings.data) setCount(settings.data.chapter_concurrency); }, [settings.data]);
  const save = useMutation({
    mutationFn: () => saveAdminReportSettings({ chapter_concurrency: count }),
    onSuccess: data => { queryClient.setQueryData(["admin-report-settings"], data); setSaved(true); },
    onError: () => setSaved(false),
  });
  const update = (value: number) => { setCount(Math.min(10, Math.max(1, value))); setSaved(false); };

  return <section className="max-w-4xl border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}>
    <header className="border-b px-5 py-4" style={{ borderColor: "var(--line)" }}><h2 className="text-lg font-medium">完整报告执行</h2><p className="mt-1 text-sm" style={{ color: "var(--ink-faint)" }}>这里只控制十章同时生成的数量，其他调用规则由程序统一管理。</p></header>
    <div className="p-5">
      {settings.isLoading ? <p role="status" className="text-sm">正在读取设置…</p> : settings.isError ? <p role="alert" className="text-sm text-red-800">读取设置失败，请刷新后重试。</p> : <><label htmlFor="chapter-concurrency" className="font-medium">十章并发数</label><div className="mt-4 grid gap-5 md:grid-cols-[1fr_8rem]"><input id="chapter-concurrency" type="range" min="1" max="10" value={count} onChange={event => update(Number(event.target.value))} style={{ accentColor: "var(--gold-deep)" }} /><input aria-label="十章并发数数值" type="number" min="1" max="10" value={count} onChange={event => update(Number(event.target.value))} className="border px-3 py-2 text-center text-lg" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }} /></div><div className="mt-5 border-l-2 pl-4 text-sm leading-6" style={{ borderColor: "var(--gold-deep)", color: "var(--ink-soft)" }}>当前设置表示同一时间最多生成 <strong>{count}</strong> 个章节。报告启动时会冻结该数值，后续修改只影响新报告。</div></>}
      {save.isError && <p role="alert" className="mt-4 text-sm text-red-800">保存失败，原设置未改变。</p>}{saved && <p role="status" className="mt-4 text-sm text-emerald-800">报告设置已保存。</p>}
    </div>
    <footer className="flex justify-end border-t px-5 py-4" style={{ borderColor: "var(--line-soft)" }}><button type="button" disabled={settings.isLoading || settings.isError || save.isPending} onClick={() => save.mutate()} className="btn-gold flex items-center gap-2 px-5 py-2"><Save size={15} />{save.isPending ? "保存中…" : "保存报告设置"}</button></footer>
  </section>;
}
