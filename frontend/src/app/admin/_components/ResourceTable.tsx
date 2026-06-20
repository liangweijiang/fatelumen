"use client";

import { useEffect, useState, useCallback } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  fetchResourceSchema,
  fetchResourceList,
  fetchResourceDetail,
  runResourceAction,
  type ResourceField,
} from "@/lib/admin-api";

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
  const [activeId, setActiveId] = useState<string | null>(null);
  const [acting, setActing] = useState(false);
  const pageSize = 20;
  const queryClient = useQueryClient();

  const schemaQ = useQuery({
    queryKey: ["admin-schema", resource],
    queryFn: () => fetchResourceSchema(resource),
  });

  const listQ = useQuery({
    queryKey: ["admin-list", resource, page, search],
    queryFn: () => fetchResourceList(resource, { page, page_size: pageSize, search }),
  });

  const detailQ = useQuery({
    queryKey: ["admin-detail", resource, activeId],
    queryFn: () => fetchResourceDetail(resource, activeId as string),
    enabled: activeId !== null,
  });

  useEffect(() => {
    setPage(1);
    setActiveId(null);
  }, [resource]);

  const onSearch = useCallback(() => {
    setPage(1);
    setSearch(searchInput.trim());
  }, [searchInput]);

  const onAction = useCallback(
    async (actionName: string) => {
      if (activeId === null) return;
      let params: Record<string, unknown> = {};
      if (actionName === "unlock") {
        const reason = window.prompt("解锁原因（可留空）", "");
        params = { reason: reason ?? "" };
      }
      setActing(true);
      try {
        await runResourceAction(resource, activeId, actionName, params);
        await queryClient.invalidateQueries({ queryKey: ["admin-list", resource] });
        await queryClient.invalidateQueries({ queryKey: ["admin-detail", resource, activeId] });
      } catch {
        window.alert("操作失败，请稍后重试");
      } finally {
        setActing(false);
      }
    },
    [activeId, resource, queryClient]
  );

  const fields = (schemaQ.data?.fields ?? []).filter((f) => !f.hidden);
  const actions = schemaQ.data?.actions ?? [];
  const items = listQ.data?.items ?? [];
  const total = listQ.data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const detail = (detailQ.data ?? {}) as Record<string, unknown>;

  if (schemaQ.isError || listQ.isError) {
    return (
      <div className="rounded-md border p-6 text-[14px]" style={{ borderColor: "var(--line)", color: "var(--ink-soft)" }}>
        数据加载失败，请稍后重试。
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
              <th className="border-b px-4 py-3 text-left font-medium" style={{ borderColor: "var(--line)", color: "var(--ink)" }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {listQ.isLoading ? (
              <tr><td colSpan={fields.length + 1} className="px-4 py-8 text-center" style={{ color: "var(--ink-faint)" }}>加载中…</td></tr>
            ) : items.length === 0 ? (
              <tr><td colSpan={fields.length + 1} className="px-4 py-8 text-center" style={{ color: "var(--ink-faint)" }}>暂无数据</td></tr>
            ) : (
              items.map((row, i) => {
                const r = row as Record<string, unknown>;
                const rowId = String(r.id ?? "");
                return (
                  <tr key={i} style={{ background: i % 2 ? "transparent" : "var(--bg-card)" }}>
                    {fields.map((f) => (
                      <td key={f.key} className="border-b px-4 py-3" style={{ borderColor: "var(--line-soft)", color: "var(--ink-soft)" }}>
                        {renderCell(f, r[f.key])}
                      </td>
                    ))}
                    <td className="border-b px-4 py-3" style={{ borderColor: "var(--line-soft)" }}>
                      <button
                        type="button"
                        onClick={() => setActiveId(rowId)}
                        className="rounded-md border px-3 py-1 text-[13px]"
                        style={{ borderColor: "var(--line)", color: "var(--gold-deep)" }}
                      >
                        查看
                      </button>
                    </td>
                  </tr>
                );
              })
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

      {activeId !== null && (
        <div
          className="fixed inset-0 z-50 flex justify-end"
          style={{ background: "rgba(0,0,0,0.3)" }}
          onClick={() => setActiveId(null)}
        >
          <div
            className="h-full w-full max-w-md overflow-y-auto p-6"
            style={{ background: "var(--bg-card)" }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-[18px] font-medium" style={{ color: "var(--ink)" }}>详情 #{activeId}</h2>
              <button type="button" onClick={() => setActiveId(null)} className="text-[14px]" style={{ color: "var(--ink-faint)" }}>关闭</button>
            </div>

            {detailQ.isLoading ? (
              <div className="py-8 text-center text-[14px]" style={{ color: "var(--ink-faint)" }}>加载中…</div>
            ) : (
              <div className="space-y-3">
                {Object.entries(detail).map(([k, v]) => (
                  <div key={k} className="flex gap-3 text-[14px]">
                    <span className="w-32 shrink-0" style={{ color: "var(--ink-faint)" }}>{k}</span>
                    <span style={{ color: "var(--ink-soft)" }}>{typeof v === "object" ? JSON.stringify(v) : String(v)}</span>
                  </div>
                ))}
              </div>
            )}

            {actions.length > 0 && (
              <div className="mt-6 flex flex-wrap gap-3 border-t pt-4" style={{ borderColor: "var(--line)" }}>
                {actions.map((a) => (
                  <button
                    key={a.name}
                    type="button"
                    disabled={acting}
                    onClick={() => onAction(a.name)}
                    className="rounded-md px-4 py-2 text-[14px] disabled:opacity-50"
                    style={{ background: "var(--gold-deep)", color: "var(--bg-card)" }}
                  >
                    {a.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
