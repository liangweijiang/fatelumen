"use client";

import axios from "axios";
import Image from "next/image";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { setAdminToken } from "@/lib/admin-auth-storage";

type Captcha = { captcha_id: string; image: string };
const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL;

export default function AdminLoginPage() {
  const router = useRouter();
  const [captcha, setCaptcha] = useState<Captcha>({ captcha_id: "", image: "" });
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [answer, setAnswer] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const refreshCaptcha = async () => {
    const response = await axios.get(`${apiBase}/admin/auth/captcha`);
    setCaptcha(response.data.data ?? response.data);
    setAnswer("");
  };

  useEffect(() => {
    void refreshCaptcha().catch(() => setError("验证码暂不可用，请稍后重试。"));
  }, []);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const response = await axios.post(`${apiBase}/admin/auth/login`, {
        username,
        password,
        captcha_id: captcha.captcha_id,
        captcha_answer: answer,
      });
      setAdminToken((response.data.data ?? response.data).token);
      router.replace("/admin");
    } catch {
      setError("账号、密码或验证码错误。请重试。");
      void refreshCaptcha();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="flex min-h-screen items-center justify-center p-6" style={{ background: "var(--bg)" }}>
      <form onSubmit={submit} className="w-full max-w-md rounded-xl border p-8" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
        <h1 className="font-[var(--serif)] text-3xl">FateLumen Admin</h1>
        <p className="mt-2 text-sm" style={{ color: "var(--ink-soft)" }}>仅限内部管理员登录；不提供注册或第三方登录。</p>
        <label className="mt-6 block text-sm">用户名<input value={username} onChange={(e) => setUsername(e.target.value)} className="mt-1 w-full rounded border p-2" required /></label>
        <label className="mt-4 block text-sm">密码<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="mt-1 w-full rounded border p-2" required /></label>
        <label className="mt-4 block text-sm">图形验证码
          <span className="mt-1 flex gap-3">
            <input value={answer} onChange={(e) => setAnswer(e.target.value)} className="w-full rounded border p-2" required />
            <button type="button" onClick={() => void refreshCaptcha()} className="overflow-hidden rounded border" aria-label="刷新验证码">
              {captcha.image && <Image src={captcha.image} alt="点击刷新验证码" width={144} height={48} className="h-10 w-32" unoptimized />}
            </button>
          </span>
        </label>
        {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
        <button disabled={submitting} className="mt-6 w-full rounded py-2 text-white disabled:opacity-60" style={{ background: "var(--gold-deep)" }}>{submitting ? "登录中…" : "安全登录"}</button>
      </form>
    </main>
  );
}
