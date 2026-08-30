"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import adminApi from "@/lib/admin-client";
import { getAdminToken, removeAdminToken } from "@/lib/admin-auth-storage";

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1 } } });
const nav = [
  { href: "/admin", label: "数据概览" },
  { href: "/admin/users", label: "用户管理" },
  { href: "/admin/content/knowledge", label: "内容管理" },
  { href: "/admin/pricing", label: "定价套餐" },
  { href: "/admin/base-data/geo", label: "基础数据" },
  { href: "/admin/report-workspace/calculations", label: "报告工作台" },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const isLoginPage = pathname === "/admin/login";

  useEffect(() => {
	if (isLoginPage) return;
    if (!getAdminToken()) { router.replace("/admin/login"); return; }
    let alive = true;
    adminApi.get("/admin/auth/me").then(() => alive && setReady(true)).catch(() => router.replace("/admin/login"));
    return () => { alive = false; };
	}, [isLoginPage, router]);

	if (isLoginPage) return <>{children}</>;
  if (!ready) return null;
  return <QueryClientProvider client={queryClient}>
    <div className="min-h-screen" style={{ background: "var(--bg)" }}>
      <header className="flex items-center justify-between border-b px-8 py-4" style={{ borderColor: "var(--line)" }}>
        <div className="flex items-center gap-8">
          <span className="font-[var(--serif)] text-xl font-medium" style={{ color: "var(--ink)" }}>FateLumen 管理后台</span>
          <nav className="flex flex-wrap gap-5">{nav.map((item) => {
            const active = pathname === item.href
              || (item.href.startsWith("/admin/base-data") && pathname.startsWith("/admin/base-data"))
              || (item.href.startsWith("/admin/report-workspace") && pathname.startsWith("/admin/report-workspace"))
              || (item.href.startsWith("/admin/content") && pathname.startsWith("/admin/content"));
            return <Link key={item.href} href={item.href} className="text-sm" style={{ color: active ? "var(--gold-deep)" : "var(--ink-soft)" }}>{item.label}</Link>;
          })}</nav>
        </div>
        <button type="button" onClick={() => { void adminApi.post("/admin/auth/logout"); removeAdminToken(); router.replace("/admin/login"); }} className="text-sm hover:underline" style={{ color: "var(--ink-faint)" }}>退出登录</button>
      </header>
      <main className="px-8 py-8">{children}</main>
    </div>
  </QueryClientProvider>;
}
