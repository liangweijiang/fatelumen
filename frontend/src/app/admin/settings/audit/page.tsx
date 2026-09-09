"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchAdminSettingAudits } from "@/lib/admin-api";

const resourceNames: Record<string, string> = { llm_provider_config: "供应商", llm_model_config: "模型", report_setting: "报告设置" };
const actionNames: Record<string, string> = { create: "新增", update: "修改", delete: "删除" };
const pageSize = 20;

export default function SettingAuditPage() {
  const [page, setPage] = useState(1);
  const [resource, setResource] = useState("");
  const audits = useQuery({ queryKey: ["admin-setting-audits", page, resource], queryFn: () => fetchAdminSettingAudits(page, pageSize, resource), retry: false });
  return <section className="border" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}>
    <header className="flex flex-wrap items-end justify-between gap-4 border-b px-5 py-4" style={{ borderColor: "var(--line)" }}><div><h2 className="text-lg font-medium">配置变更记录</h2><p className="mt-1 text-sm" style={{ color: "var(--ink-faint)" }}>记录安全字段的修改前后值；密钥只记录是否发生更换。</p></div><label className="text-sm"><span className="mr-2">配置类型</span><select value={resource} onChange={event => { setResource(event.target.value); setPage(1); }} className="border px-3 py-2" style={{ borderColor: "var(--line)", background: "var(--bg-card)" }}><option value="">全部</option><option value="llm_provider_config">供应商</option><option value="llm_model_config">模型</option><option value="report_setting">报告设置</option></select></label></header>
    {audits.isLoading ? <p role="status" className="p-5 text-sm">正在读取变更记录…</p> : audits.isError ? <p role="alert" className="p-5 text-sm text-red-800">变更记录读取失败。</p> : !audits.data?.items.length ? <p className="p-8 text-center text-sm" style={{ color: "var(--ink-faint)" }}>暂无配置变更记录。</p> : <div className="overflow-x-auto"><table className="w-full min-w-[900px] text-left text-sm"><thead><tr><th className="px-5 py-3">时间</th><th className="px-5 py-3">管理员</th><th className="px-5 py-3">类型</th><th className="px-5 py-3">操作</th><th className="px-5 py-3">修改前</th><th className="px-5 py-3">修改后</th></tr></thead><tbody>{audits.data.items.map(item => <tr key={item.id} className="border-t align-top" style={{ borderColor: "var(--line-soft)" }}><td className="whitespace-nowrap px-5 py-3">{new Date(item.created_at).toLocaleString("zh-CN", { hour12: false })}</td><td className="px-5 py-3">{item.admin_name || `#${item.admin_id}`}</td><td className="px-5 py-3">{resourceNames[item.resource] ?? item.resource}</td><td className="px-5 py-3">{actionNames[item.action] ?? item.action}</td><td className="max-w-xs px-5 py-3 font-mono text-xs"><pre className="whitespace-pre-wrap break-all">{item.detail?.before ? JSON.stringify(item.detail.before, null, 2) : "—"}</pre></td><td className="max-w-xs px-5 py-3 font-mono text-xs"><pre className="whitespace-pre-wrap break-all">{item.detail?.after ? JSON.stringify(item.detail.after, null, 2) : "—"}</pre></td></tr>)}</tbody></table></div>}
    <footer className="flex items-center justify-end gap-3 border-t px-5 py-4 text-sm" style={{ borderColor: "var(--line-soft)" }}><span>第 {page} 页 · 共 {audits.data?.total ?? 0} 条</span><button type="button" className="btn-ghost" disabled={page === 1} onClick={() => setPage(value => value - 1)}>上一页</button><button type="button" className="btn-ghost" disabled={!audits.data || page * pageSize >= audits.data.total} onClick={() => setPage(value => value + 1)}>下一页</button></footer>
  </section>;
}
