"use client";

import { useParams } from "next/navigation";
import { ChangeEvent, useEffect, useMemo, useState } from "react";
import api from "@/lib/admin-client";

type Status = "draft" | "preview" | "published" | "unpublished";
type Item = {
  id: number;
  content_key: string;
  slug: string;
  category: string;
  type: string;
  title: string;
  summary: string;
  locale: string;
  status: Status;
  markdown: string;
  tags: string[];
  sort_order: number;
  pinned: boolean;
};

const locales = ["zh", "en", "ja", "ko"];
const localeName: Record<string, string> = { zh: "中文", en: "EN", ja: "日本語", ko: "한국어" };
const labels: Record<string, string> = { knowledge: "八字知识", faq: "常见问题", case: "客户案例" };
const presets: Record<string, { category: string; tags: string[] }> = {
  knowledge: { category: "基础知识", tags: ["八字", "入门"] },
  faq: { category: "报告内容", tags: ["报告", "常见问题"] },
  case: { category: "客户案例", tags: ["案例"] },
};
const empty = (locale = "zh"): Partial<Item> => ({ locale, tags: [], sort_order: 0, pinned: false, markdown: "", summary: "", status: "draft" });

function localSummary(markdown: string) {
  const plain = markdown
    .replace(/<[^>]*>/g, " ")
    .replace(/!\[[^\]]*]\([^)]*\)/g, " ")
    .replace(/\[([^\]]+)]\([^)]*\)/g, "$1")
    .replace(/[#>*_`~-]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
  return Array.from(plain).slice(0, 100).join("");
}

export default function ContentPage() {
  const { type } = useParams<{ type: string }>();
  const [rows, setRows] = useState<Item[]>([]);
  const [edit, setEdit] = useState<Partial<Item>>(empty());
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("all");
  const [editorOpen, setEditorOpen] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [summaryAuto, setSummaryAuto] = useState(true);
  const [uploading, setUploading] = useState(false);

  const load = async () => {
    try {
      const response = await api.get(`/admin/content/${type}`);
      setRows(response.data.data ?? response.data);
      setError("");
    } catch {
      setError("加载失败，请稍后重试。");
    }
  };

  useEffect(() => {
    setEdit(empty());
    setSelected(new Set());
    void load();
  }, [type]);

  const groups = useMemo(
    () =>
      Object.values(
        rows.reduce<Record<string, Item[]>>((all, row) => {
          (all[row.content_key] ??= []).push(row);
          return all;
        }, {}),
      )
        .filter((items) => {
          const zh = items.find((item) => item.locale === "zh") ?? items[0];
          return `${zh.title} ${zh.category} ${(zh.tags ?? []).join(" ")}`.toLowerCase().includes(query.toLowerCase()) && (status === "all" || zh.status === status);
        })
        .sort((a, b) => Number(b[0].pinned) - Number(a[0].pinned) || a[0].sort_order - b[0].sort_order),
    [rows, query, status],
  );

  const visibleKeys = groups.map((items) => items[0].content_key);
  const allVisibleSelected = visibleKeys.length > 0 && visibleKeys.every((key) => selected.has(key));

  const toggleAll = () => {
    setSelected((current) => {
      const next = new Set(current);
      if (allVisibleSelected) visibleKeys.forEach((key) => next.delete(key));
      else visibleKeys.forEach((key) => next.add(key));
      return next;
    });
  };

  const toggleOne = (key: string) => {
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const openNew = () => {
    const preset = presets[type] ?? presets.knowledge;
    setEdit({ ...empty(), category: preset.category, tags: preset.tags });
    setSummaryAuto(true);
    setError("");
    setEditorOpen(true);
  };

  const openEdit = (items: Item[]) => {
    const preferred = items.find((item) => item.locale === "zh") ?? items[0];
    setEdit({ ...preferred, tags: preferred.tags ?? [] });
    setSummaryAuto(false);
    setError("");
    setEditorOpen(true);
  };

  const switchLocale = (locale: string) => {
    const sibling = rows.find((row) => row.content_key === edit.content_key && row.locale === locale);
    setEdit(
      sibling
        ? { ...sibling, tags: sibling.tags ?? [] }
        : { ...empty(locale), content_key: edit.content_key, type, sort_order: edit.sort_order, pinned: edit.pinned, category: edit.category, tags: edit.tags },
    );
    setSummaryAuto(!sibling?.summary);
  };

  const importMarkdown = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (!file.name.toLowerCase().endsWith(".md")) {
      setError("仅支持上传 .md 文件。");
      return;
    }
    try {
      setUploading(true);
      setError("");
      const body = new FormData();
      body.append("file", file);
      const response = await api.post("/admin/content/import-markdown", body);
      const parsed = response.data.data ?? response.data;
      setEdit((current) => ({ ...current, title: parsed.title, markdown: parsed.markdown, summary: parsed.summary }));
      setSummaryAuto(true);
      setNotice(`已导入 ${file.name}，请确认标题、正文和摘要后保存。`);
    } catch {
      setError("Markdown 上传失败：仅支持不超过 2MB 的非空 .md 文件。");
    } finally {
      setUploading(false);
    }
  };

  const updateMarkdown = (markdown: string) => {
    setEdit((current) => ({ ...current, markdown, summary: summaryAuto ? localSummary(markdown) : current.summary }));
  };

  const save = async () => {
    try {
      setError("");
      if (!edit.title || !edit.markdown) throw new Error();
      if (edit.id) await api.patch(`/admin/content/${type}/${edit.id}`, edit);
      else await api.post("/admin/content", { ...edit, type });
      await load();
      setEditorOpen(false);
      setNotice("内容已保存。");
    } catch {
      setError("保存失败：请填写标题并上传或输入 Markdown 正文。");
    }
  };

  const changeState = async (next: Status) => {
    if (!edit.id) return;
    if (next === "unpublished" && !window.confirm("确认取消发布吗？取消后客户端将不再显示该语言内容。")) return;
    await api.post(`/admin/content/${type}/${edit.id}/state`, { status: next });
    await load();
    setEdit({ ...edit, status: next });
  };

  const batchPin = async (pinned: boolean) => {
    if (selected.size === 0) return;
    try {
      await api.post(`/admin/content/${type}/batch-pin`, { content_keys: Array.from(selected), pinned });
      await load();
      setNotice(`已${pinned ? "置顶" : "取消置顶"} ${selected.size} 条内容。`);
      setSelected(new Set());
    } catch {
      setError("批量置顶操作失败。");
    }
  };

  const batchDelete = async () => {
    if (selected.size === 0 || !window.confirm(`确认删除选中的 ${selected.size} 条内容及其全部语言版本吗？此操作无法撤销。`)) return;
    try {
      await api.post(`/admin/content/${type}/batch-delete`, { content_keys: Array.from(selected) });
      await load();
      setNotice(`已删除 ${selected.size} 条内容及其语言版本。`);
      setSelected(new Set());
    } catch {
      setError("批量删除失败。");
    }
  };

  if (!labels[type]) return <p>不支持的内容类型。</p>;

  return (
    <div>
      <div className="mb-5 flex items-start justify-between gap-4">
        <div><h1 className="mb-2 text-2xl font-medium">{labels[type]}</h1><p className="text-sm" style={{ color: "var(--ink-soft)" }}>一条列表代表一份内容；语言版本在同一内容内维护。</p></div>
        <button onClick={openNew} className="shrink-0 rounded bg-amber-700 px-4 py-2 text-white">＋ 新增内容</button>
      </div>
      {notice && <p className="mb-3 rounded border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">{notice}</p>}
      {error && !editorOpen && <p className="mb-3 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</p>}
      <div className="mb-3 flex flex-wrap gap-3">
        <input className="min-w-64 flex-1 rounded border p-2" placeholder="搜索标题、分类或标签" value={query} onChange={(event) => setQuery(event.target.value)} />
        <select className="rounded border px-3" value={status} onChange={(event) => setStatus(event.target.value)}><option value="all">全部状态</option><option value="published">已发布</option><option value="draft">草稿</option><option value="preview">预览</option><option value="unpublished">已取消发布</option></select>
      </div>
      <div className="mb-3 flex min-h-10 flex-wrap items-center gap-2">
        <span className="mr-2 text-sm" style={{ color: "var(--ink-soft)" }}>已选 {selected.size} 条</span>
        <button disabled={!selected.size} onClick={() => void batchPin(true)} className="rounded border px-3 py-2 text-sm disabled:opacity-40">批量置顶</button>
        <button disabled={!selected.size} onClick={() => void batchPin(false)} className="rounded border px-3 py-2 text-sm disabled:opacity-40">取消置顶</button>
        <button disabled={!selected.size} onClick={() => void batchDelete()} className="rounded border border-red-300 px-3 py-2 text-sm text-red-700 disabled:opacity-40">批量删除</button>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead className="border-y text-xs" style={{ color: "var(--ink-soft)" }}><tr><th className="w-10 p-3"><input type="checkbox" aria-label="全选当前列表" checked={allVisibleSelected} onChange={toggleAll} /></th><th className="p-3">内容</th><th className="p-3">语言版本</th><th className="p-3">置顶</th><th className="p-3">排序</th><th className="p-3">状态</th><th className="p-3">操作</th></tr></thead>
          <tbody>{groups.map((items) => { const zh = items.find((item) => item.locale === "zh") ?? items[0]; return <tr key={zh.content_key} className="border-b"><td className="p-3"><input type="checkbox" aria-label={`选择 ${zh.title}`} checked={selected.has(zh.content_key)} onChange={() => toggleOne(zh.content_key)} /></td><td className="p-3"><b>{zh.title}</b><div className="mt-1 text-xs" style={{ color: "var(--ink-soft)" }}>分类：{zh.category || "未分类"}　标签：{(zh.tags ?? []).join("、") || "无"}</div></td><td className="p-3 text-xs" style={{ color: "var(--gold-deep)" }}>{items.map((item) => localeName[item.locale] ?? item.locale).join(" · ")}　{items.length} / 4</td><td className="p-3">{zh.pinned ? "是" : "否"}</td><td className="p-3">{zh.sort_order}</td><td className="p-3">{zh.status === "published" ? "已发布" : zh.status === "draft" ? "草稿" : zh.status === "preview" ? "预览" : "已取消发布"}</td><td className="p-3"><button onClick={() => openEdit(items)} style={{ color: "var(--gold-deep)" }}>编辑</button></td></tr>; })}</tbody>
        </table>
        {groups.length === 0 && <p className="p-8 text-center text-sm" style={{ color: "var(--ink-soft)" }}>暂无内容</p>}
      </div>

      {editorOpen && <div className="fixed inset-0 z-50 grid place-items-center bg-black/35 p-5" onMouseDown={(event) => { if (event.target === event.currentTarget) setEditorOpen(false); }}><section role="dialog" aria-modal="true" className="max-h-[88vh] w-full max-w-3xl overflow-y-auto border p-6 shadow-2xl" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
        <div className="mb-4 flex items-center justify-between"><b>{edit.id ? "编辑内容" : "新增内容"}</b><button onClick={() => setEditorOpen(false)} className="text-2xl" aria-label="关闭">×</button></div>
        <div className="mb-4 rounded p-3 text-sm" style={{ background: "var(--bg-soft)", color: "var(--ink-soft)" }}>共用设置：分类、标签、排序、置顶、发布状态与语言覆盖。</div>
        <div className="mb-5 flex gap-2 border-b">{locales.map((locale) => <button key={locale} onClick={() => switchLocale(locale)} className={`px-3 py-2 text-sm ${edit.locale === locale ? "border-b-2 border-amber-700 text-amber-800" : ""}`}>{localeName[locale]}</button>)}</div>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="block text-sm">标题<input className="mt-1 w-full rounded border p-2" value={edit.title ?? ""} onChange={(event) => setEdit({ ...edit, title: event.target.value })} /></label>
          <label className="block text-sm">分类<input className="mt-1 w-full rounded border p-2" value={edit.category ?? ""} onChange={(event) => setEdit({ ...edit, category: event.target.value })} /></label>
          <label className="block text-sm md:col-span-2">标签（逗号分隔）<input className="mt-1 w-full rounded border p-2" value={(edit.tags ?? []).join(", ")} onChange={(event) => setEdit({ ...edit, tags: event.target.value.split(",").map((tag) => tag.trim()).filter(Boolean) })} /></label>
          <label className="block text-sm">排序<input type="number" className="mt-1 w-full rounded border p-2" value={edit.sort_order ?? 0} onChange={(event) => setEdit({ ...edit, sort_order: Number(event.target.value) })} /></label>
          <label className="flex items-center gap-2 pt-7 text-sm"><input type="checkbox" checked={!!edit.pinned} onChange={(event) => setEdit({ ...edit, pinned: event.target.checked })} /> 置顶</label>
          <div className="md:col-span-2"><div className="mb-2 flex items-center justify-between"><span className="text-sm">{type === "faq" ? "回答" : "正文"}</span><label className="cursor-pointer rounded border px-3 py-2 text-sm">{uploading ? "上传解析中…" : "上传 Markdown（.md）"}<input disabled={uploading} type="file" accept=".md,text/markdown" className="sr-only" onChange={(event) => void importMarkdown(event)} /></label></div><textarea className="min-h-56 w-full rounded border p-3 font-mono text-sm" placeholder="上传 Markdown（.md），也可直接编辑" value={edit.markdown ?? ""} onChange={(event) => updateMarkdown(event.target.value)} /></div>
          <label className="block text-sm md:col-span-2">摘要（默认取清洗后正文前 100 字，可修改）<textarea className="mt-1 min-h-24 w-full rounded border p-3 text-sm" value={edit.summary ?? ""} onChange={(event) => { setSummaryAuto(false); setEdit({ ...edit, summary: event.target.value }); }} /><span className="mt-1 flex items-center justify-between text-xs" style={{ color: "var(--ink-soft)" }}><span>{summaryAuto ? "自动跟随正文" : "已人工修改"}</span><button type="button" onClick={() => { setSummaryAuto(true); setEdit({ ...edit, summary: localSummary(edit.markdown ?? "") }); }} style={{ color: "var(--gold-deep)" }}>重新自动生成</button></span></label>
        </div>
        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
        <div className="mt-5 flex flex-wrap justify-end gap-2"><button onClick={() => setEditorOpen(false)} className="rounded border px-4 py-2">取消</button><button onClick={() => void save()} className="rounded bg-amber-700 px-4 py-2 text-white">保存当前语言</button>{edit.id && <><button onClick={() => void changeState("published")} className="rounded border px-4 py-2">发布</button><button onClick={() => void changeState("unpublished")} className="rounded border px-4 py-2">取消发布</button></>}</div>
      </section></div>}
    </div>
  );
}
