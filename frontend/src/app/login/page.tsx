"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { login, register } from "@/lib/auth-api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function LoginPage() {
  const router = useRouter();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    setLoading(true);
    try {
      if (mode === "register") {
        await register(email, password, name);
      } else {
        await login(email, password);
      }
      router.push("/admin");
    } catch (e: unknown) {
      setErr((e as { message?: string })?.message || "操作失败，请稍后再试");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-6" style={{ background: "var(--bg)" }}>
      <div className="w-full max-w-[420px] rounded-xl border p-9" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
        <h1 className="mb-2 text-center font-[var(--serif)] text-[32px] font-medium" style={{ color: "var(--ink)" }}>
          FateLumen
        </h1>
        <p className="mb-8 text-center text-[15px] font-light" style={{ color: "var(--ink-soft)" }}>
          {mode === "login" ? "登入命理阁，开启您的推演之旅" : "立此命牒，记录您的命盘流年"}
        </p>
        <form onSubmit={handleSubmit} className="space-y-5">
          {mode === "register" && (
            <div className="space-y-2">
              <Label htmlFor="name" style={{ color: "var(--ink-soft)" }}>称谓</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} placeholder="如何称呼您" />
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="email" style={{ color: "var(--ink-soft)" }}>邮箱</Label>
            <Input id="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password" style={{ color: "var(--ink-soft)" }}>密码</Label>
            <Input id="password" type="password" required value={password} onChange={(e) => setPassword(e.target.value)} placeholder="至少 8 位" />
          </div>
          {err && <p className="text-[14px]" style={{ color: "var(--fire, #b8473e)" }}>{err}</p>}
          <Button type="submit" disabled={loading} className="w-full" style={{ background: "var(--gold-deep)", color: "var(--bg)" }}>
            {loading ? "推演中…" : mode === "login" ? "登入" : "立牒注册"}
          </Button>
        </form>
        <button
          type="button"
          onClick={() => { setMode(mode === "login" ? "register" : "login"); setErr(""); }}
          className="mt-6 w-full text-center text-[14px] font-light underline-offset-4 hover:underline"
          style={{ color: "var(--ink-faint)" }}
        >
          {mode === "login" ? "尚无命牒？点此立牒注册" : "已有命牒？返回登入"}
        </button>
      </div>
    </div>
  );
}
