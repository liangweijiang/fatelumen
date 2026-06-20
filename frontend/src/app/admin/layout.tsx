"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { getToken, removeToken } from "@/lib/auth-storage";
import { fetchMe } from "@/lib/admin-api";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const adminQueryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
});

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login");
      return;
    }
    let alive = true;
    fetchMe()
      .then((me) => {
        if (!alive) return;
        if (me.role !== "admin") {
          router.replace("/");
          return;
        }
        setReady(true);
      })
      .catch(() => {
        if (!alive) return;
        router.replace("/login");
      });
    return () => { alive = false; };
  }, [router]);

  if (!ready) return null;

  const nav = [
    { href: "/admin", label: "数据概览" },
    { href: "/admin/users", label: "用户管理" },
    { href: "/admin/orders", label: "订单管理" },
    { href: "/admin/reports", label: "报告管理" },
  ];

  return (
    <QueryClientProvider client={adminQueryClient}>
    <div className="min-h-screen" style={{ background: "var(--bg)" }}>
      <header className="flex items-center justify-between border-b px-8 py-4" style={{ borderColor: "var(--line)" }}>
        <div className="flex items-center gap-8">
          <span className="font-[var(--serif)] text-[20px] font-medium" style={{ color: "var(--ink)" }}>FateLumen 管理后台</span>
          <nav className="flex gap-6">
            {nav.map((n) => (
              <Link key={n.href} href={n.href} className="text-[15px]" style={{ color: pathname === n.href ? "var(--gold-deep)" : "var(--ink-soft)" }}>
                {n.label}
              </Link>
            ))}
          </nav>
        </div>
        <button
          type="button"
          onClick={() => { removeToken(); router.replace("/login"); }}
          className="text-[14px] font-light hover:underline"
          style={{ color: "var(--ink-faint)" }}
        >
          退出登录
        </button>
      </header>
      <main className="px-8 py-8">{children}</main>
    </div>
    </QueryClientProvider>
  );
}
