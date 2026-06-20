"use client";

import { useEffect, useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchResourceSchema, fetchResourceList, type ResourceField } from "@/lib/admin-api";

function renderCell(field: ResourceField, value: unknown): string {
  if (value === null || value === undefined) return "—";
  if (field.type === "bool") return value ? "是" : "否";
  if (field.type === "enum") {
    const opt = field.enum?.find((e) => e.value === String(value));
    return opt ? opt.label : String(value);
  }
  if (field.type === "datetime") {
    const d = new Date(String(value));
    return Number.isNaN(d.getTime()) ? String(value) : d.toLocaleString("zh-CN");
  }
  if (field.type === "money") {
    const n = Number(value);
    return Number.isNaN(n) ? String(value) : `¥${(n / 100).toFixed(2)}`;
  }
  return String(value);
}

export default function ResourceTable({ resource }: { resource: string }) {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const pageSize = 20;

  const schemaQ = useQuery({
    queryKey: ["admin-schema", resource],
    queryFn: () => fetchResourceSchema(resource),
  });

  const listQ = useQuery({
    queryKey: ["admin-list", resource, page, search],
    queryFn: () => fetchResourceList(resource, { page, page_size: pageSize, search }),
  });

  useEffect(() => {
    setPage(1);
  }, [resource]);

  const onSearch = useCallback(() => {
    setPage(1);
    setSearch(searchInput.trim());
  }, [searchInput]);

  const fields = (schemaQ.data?.fields ?? []).filter((f) => !f.hidden);
  const items = listQ.data?.items ?? [];
  const total = listQ.data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  if (schemaQ.isError || listQ.isError) {
    return (
      <div className="rounded-md border p-6 text-[14px]" style={{ borderColor: "var(--line)", color: "var(--ink-soft)" }}>
        数据加载失败,请稍后重试。
      </div>
    );
  }

  return (
    <div>
      <div className="mb-4 flex items-center gap-3">
        <input
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") onSearch(); }}
          placeholder="搜索关键词"
          className="rounded-md border px-3 py-2 text-[14px] outline-none"
          style={{ borderColor: "var(--line)", background: "var(--bg-card)", color: "var(--ink)" }}
        />
        <button
          type="button"
          onClick={onSearch}
          className="rounded-md px-4 py-2 text-[14px]"
          style={{ background: "var(--gold-deep)", color: "var(--bg-card)" }}
        >
          搜索
        </button>
      </div>

      <div className="overflow-x-auto rounded-md border" style={{ borderColor: "var(--line)" }}>
        <table className="w-full border-collapse text-[14px]">
          <thead>
            <tr style={{ background: "var(--bg-soft)" }}>
              {fields.map((f) => (
                <th key={f.key} className="border-b px-4 py-3 text-left font-medium" style={{ borderColor: "var(--line)", color: "var(--ink)" }}>
                  {f.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {listQ.isLoading ? (
              <tr><td colSpan={fields.length} className="px-4 py-8 text-center" style={{ color: "var(--ink-faint)" }}>加载中…</td></tr>
            ) : items.length === 0 ? (
              <tr><td colSpan={fields.length} className="px-4 py-8 text-center" style={{ color: "var(--ink-faint)" }}>暂无数据</td></tr>
            ) : (
              items.map((row, i) => (
                <tr key={i} style={{ background: i % 2 ? "transparent" : "var(--bg-card)" }}>
                  {fields.map((f) => (
                    <td key={f.key} className="border-b px-4 py-3" style={{ borderColor: "var(--line-soft)", color: "var(--ink-soft)" }}>
                      {renderCell(f, (row as Record<string, unknown>)[f.key])}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="mt-4 flex items-center justify-between text-[14px]" style={{ color: "var(--ink-soft)" }}>
        <span>共 {total} 条</span>
        <div className="flex items-center gap-3">
          <button type="button" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))} className="rounded-md border px-3 py-1 disabled:opacity-40" style={{ borderColor: "var(--line)" }}>上一页</button>
          <span>{page} / {totalPages}</span>
          <button type="button" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))} className="rounded-md border px-3 py-1 disabled:opacity-40" style={{ borderColor: "var(--line)" }}>下一页</button>
        </div>
      </div>
    </div>
  );
}
