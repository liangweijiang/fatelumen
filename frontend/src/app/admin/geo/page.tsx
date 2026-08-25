"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import api from "@/lib/admin-client";

type GeoRow = {
  id: number;
  country_code: string;
  country_name: string;
  region_name: string;
  latitude: number;
  longitude: number;
};

type PageData = { items: GeoRow[]; total: number; page: number; page_size: number };

const PAGE_SIZE = 20;

export default function AdminGeoPage() {
  const [items, setItems] = useState<GeoRow[]>([]);
  const [query, setQuery] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<GeoRow | null>(null);
  const [latitude, setLatitude] = useState("");
  const [longitude, setLongitude] = useState("");
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const response = await api.get("/admin/geo", { params: { q: activeQuery, page, page_size: PAGE_SIZE } });
      const data = (response.data.data ?? response.data) as PageData;
      setItems(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch {
      setError("地理数据加载失败，请稍后重试。");
    } finally {
      setLoading(false);
    }
  }, [activeQuery, page]);

  useEffect(() => { void load(); }, [load]);

  const search = (event: FormEvent) => {
    event.preventDefault();
    setPage(1);
    setActiveQuery(query.trim());
  };

  const openEditor = (row: GeoRow) => {
    setEditing(row);
    setLatitude(String(row.latitude));
    setLongitude(String(row.longitude));
    setError("");
  };

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (!editing) return;
    const lat = Number(latitude);
    const lng = Number(longitude);
    if (!Number.isFinite(lat) || lat < -90 || lat > 90 || !Number.isFinite(lng) || lng < -180 || lng > 180) {
      setError("请输入有效经纬度：纬度 -90～90，经度 -180～180。");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await api.patch(`/admin/geo/${editing.id}`, { latitude: lat, longitude: lng });
      setEditing(null);
      await load();
    } catch {
      setError("经纬度保存失败，请稍后重试。");
    } finally {
      setSaving(false);
    }
  };

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return <div className="mx-auto max-w-6xl">
    <header className="mb-6">
      <h1 className="text-2xl font-medium">地理数据</h1>
      <p className="mt-1 text-sm" style={{ color: "var(--ink-soft)" }}>查看国家、地区及对应经纬度；发现错误时可直接修正。</p>
    </header>

    <form onSubmit={search} className="mb-5 flex max-w-xl gap-2">
      <label htmlFor="geo-search" className="sr-only">搜索国家或地区</label>
      <input id="geo-search" type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索国家或地区" className="min-w-0 flex-1 border px-3 py-2" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }} />
      <button type="submit" className="btn-gold px-5 py-2">搜索</button>
    </form>

    {error && !editing && <p role="alert" className="mb-4 border px-4 py-3 text-sm text-red-800" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>{error}</p>}

    <div className="overflow-x-auto border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}>
      <table className="w-full min-w-[720px] border-collapse text-left text-sm">
        <thead><tr className="border-b" style={{ borderColor: "var(--line)" }}><th className="px-4 py-3 font-medium">国家</th><th className="px-4 py-3 font-medium">地区</th><th className="px-4 py-3 font-medium">经度</th><th className="px-4 py-3 font-medium">纬度</th><th className="px-4 py-3 text-right font-medium">操作</th></tr></thead>
        <tbody>
          {loading ? <tr><td colSpan={5} className="px-4 py-12 text-center" aria-live="polite">加载中…</td></tr> : items.length === 0 ? <tr><td colSpan={5} className="px-4 py-12 text-center">没有找到匹配的地理数据</td></tr> : items.map((row) => <tr key={row.id} className="border-b last:border-b-0" style={{ borderColor: "var(--line-soft)" }}><td className="px-4 py-3">{row.country_name} <span className="text-xs" style={{ color: "var(--ink-faint)" }}>{row.country_code}</span></td><td className="px-4 py-3">{row.region_name}</td><td className="px-4 py-3 tabular-nums">{row.longitude}</td><td className="px-4 py-3 tabular-nums">{row.latitude}</td><td className="px-4 py-3 text-right"><button type="button" onClick={() => openEditor(row)} className="px-2 py-1" style={{ color: "var(--gold-deep)" }}>修改</button></td></tr>) }
        </tbody>
      </table>
    </div>

    <footer className="mt-4 flex items-center justify-between text-sm"><span style={{ color: "var(--ink-soft)" }}>共 {total.toLocaleString()} 条</span><div className="flex items-center gap-3"><button type="button" disabled={page <= 1 || loading} onClick={() => setPage((value) => value - 1)} className="border px-3 py-1.5 disabled:opacity-40" style={{ borderColor: "var(--line)" }}>上一页</button><span>第 {page} / {pageCount} 页</span><button type="button" disabled={page >= pageCount || loading} onClick={() => setPage((value) => value + 1)} className="border px-3 py-1.5 disabled:opacity-40" style={{ borderColor: "var(--line)" }}>下一页</button></div></footer>

    {editing && <div className="fixed inset-0 z-50 grid place-items-center bg-black/35 p-4" onMouseDown={(event) => { if (event.target === event.currentTarget) setEditing(null); }}><section role="dialog" aria-modal="true" aria-labelledby="geo-edit-title" className="w-full max-w-md border p-6 shadow-2xl" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}><div className="mb-5 flex items-start justify-between"><div><h2 id="geo-edit-title" className="text-xl">修改经纬度</h2><p className="mt-1 text-sm" style={{ color: "var(--ink-soft)" }}>{editing.country_name} · {editing.region_name}</p></div><button type="button" onClick={() => setEditing(null)} aria-label="关闭" className="text-2xl leading-none">×</button></div><form onSubmit={save} className="space-y-4"><label className="block text-sm">经度<input autoFocus inputMode="decimal" value={longitude} onChange={(event) => setLongitude(event.target.value)} className="mt-1 w-full border px-3 py-2" style={{ background: "var(--bg)", borderColor: "var(--line)" }} /></label><label className="block text-sm">纬度<input inputMode="decimal" value={latitude} onChange={(event) => setLatitude(event.target.value)} className="mt-1 w-full border px-3 py-2" style={{ background: "var(--bg)", borderColor: "var(--line)" }} /></label>{error && <p role="alert" className="text-sm text-red-700">{error}</p>}<div className="flex justify-end gap-2 pt-2"><button type="button" onClick={() => setEditing(null)} className="btn-ghost px-4 py-2">取消</button><button type="submit" disabled={saving} className="btn-gold px-4 py-2 disabled:opacity-50">{saving ? "保存中…" : "保存"}</button></div></form></section></div>}
  </div>;
}
