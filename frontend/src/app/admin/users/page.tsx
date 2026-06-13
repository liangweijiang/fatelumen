"use client";

import { useEffect, useState, useCallback } from "react";
import { fetchUsers, type AdminUserItem } from "@/lib/admin-api";

export default function AdminUsers() {
  const [items, setItems] = useState<AdminUserItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [err, setErr] = useState("");
  const pageSize = 20;

  const load = useCallback(() => {
    setErr("");
    fetchUsers(keyword, page, pageSize)
      .then((r) => { setItems(r.items || []); setTotal(r.total || 0); })
      .catch((e: unknown) => setErr((e as { message?: string })?.message || "无权查阅名册"));
  }, [keyword, page]);

  useEffect(() => { load(); }, [load]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div>
      <h1 className="mb-6 font-[var(--serif)] text-[26px] font-medium" style={{ color: "var(--ink)" }}>命主名册</h1>
      <div className="mb-5 flex gap-3">
        <input
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter") { setPage(1); load(); } }}
          placeholder="按邮箱或称谓查找"
          className="w-72 rounded-md border px-3 py-2 text-[14px]"
          style={{ background: "var(--bg-card)", borderColor: "var(--line)", color: "var(--ink)" }}
        />
        <button type="button" onClick={() => { setPage(1); load(); }} className="rounded-md px-4 py-2 text-[14px]" style={{ background: "var(--gold-deep)", color: "var(--bg)" }}>查找</button>
      </div>
      {err && <p className="mb-4 text-[14px]" style={{ color: "var(--fire, #b8473e)" }}>{err}</p>}
      <div className="overflow-hidden rounded-xl border" style={{ borderColor: "var(--line)" }}>
        <table className="w-full text-[14px]">
          <thead>
            <tr style={{ background: "var(--bg-soft)", color: "var(--ink-soft)" }}>
              <th className="px-4 py-3 text-left font-medium">编号</th>
              <th className="px-4 py-3 text-left font-medium">邮箱</th>
              <th className="px-4 py-3 text-left font-medium">称谓</th>
              <th className="px-4 py-3 text-left font-medium">身份</th>
              <th className="px-4 py-3 text-left font-medium">状态</th>
              <th className="px-4 py-3 text-left font-medium">入阁时间</th>
            </tr>
          </thead>
          <tbody style={{ color: "var(--ink)" }}>
            {items.map((u) => (
              <tr key={u.id} className="border-t" style={{ borderColor: "var(--line-soft, var(--line))" }}>
                <td className="px-4 py-3">{u.id}</td>
                <td className="px-4 py-3">{u.email}</td>
                <td className="px-4 py-3">{u.name}</td>
                <td className="px-4 py-3">{u.role === "admin" ? "阁主" : "命主"}</td>
                <td className="px-4 py-3">{u.active ? "在册" : "已封"}</td>
                <td className="px-4 py-3" style={{ color: "var(--ink-faint)" }}>{u.created_at?.slice(0, 10)}</td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-8 text-center" style={{ color: "var(--ink-faint)" }}>暂无命主</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="mt-5 flex items-center justify-between text-[14px]" style={{ color: "var(--ink-soft)" }}>
        <span>共 {total} 位命主</span>
        <div className="flex items-center gap-4">
          <button type="button" disabled={page <= 1} onClick={() => setPage(page - 1)} className="disabled:opacity-40">上一页</button>
          <span>{page} / {totalPages}</span>
          <button type="button" disabled={page >= totalPages} onClick={() => setPage(page + 1)} className="disabled:opacity-40">下一页</button>
        </div>
      </div>
    </div>
  );
}
