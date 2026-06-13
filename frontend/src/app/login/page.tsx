"use client";

import { useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { login, register } from "@/lib/auth-api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Lang = "en" | "zh" | "ja" | "ko";

const LANGS: { code: Lang; label: string }[] = [
  { code: "en", label: "English" },
  { code: "zh", label: "中文" },
  { code: "ja", label: "日本語" },
  { code: "ko", label: "한국어" },
];

const DICT: Record<Lang, Record<string, string>> = {
  en: {
    subLogin: "Enter the Hall of Destiny and begin your divination",
    subRegister: "Inscribe your tablet and chart your fortunes",
    name: "How shall we address you",
    nameLabel: "Name",
    email: "Email",
    password: "Password",
    pwHint: "At least 8 characters",
    signIn: "Enter",
    signUp: "Inscribe",
    loading: "Divining…",
    toRegister: "No tablet yet? Inscribe one",
    toLogin: "Already have a tablet? Return to enter",
    fail: "Something went wrong, please try again",
  },
  zh: {
    subLogin: "登入命理阁，开启您的推演之旅",
    subRegister: "立此命牒，记录您的命盘流年",
    name: "如何称呼您",
    nameLabel: "称谓",
    email: "邮箱",
    password: "密码",
    pwHint: "至少 8 位",
    signIn: "登入",
    signUp: "立牒注册",
    loading: "推演中…",
    toRegister: "尚无命牒？点此立牒注册",
    toLogin: "已有命牒？返回登入",
    fail: "操作失败，请稍后再试",
  },
  ja: {
    subLogin: "命理閣へ入り、推演の旅を始めましょう",
    subRegister: "命牒を記し、運命の流れを刻みます",
    name: "お名前をお聞かせください",
    nameLabel: "お名前",
    email: "メール",
    password: "パスワード",
    pwHint: "8 文字以上",
    signIn: "入る",
    signUp: "命牒を記す",
    loading: "推演中…",
    toRegister: "命牒をお持ちでない方はこちら",
    toLogin: "既に命牒をお持ちの方は入る",
    fail: "操作に失敗しました。後でお試しください",
  },
  ko: {
    subLogin: "명리각에 들어 추연의 여정을 시작하세요",
    subRegister: "명첩을 새겨 운명의 흐름을 기록합니다",
    name: "어떻게 불러드릴까요",
    nameLabel: "호칭",
    email: "이메일",
    password: "비밀번호",
    pwHint: "8자 이상",
    signIn: "입장",
    signUp: "명첩 등록",
    loading: "추연 중…",
    toRegister: "명첩이 없으신가요? 등록하기",
    toLogin: "이미 명첩이 있으신가요? 입장",
    fail: "작업에 실패했습니다. 잠시 후 다시 시도하세요",
  },
};

function readCookieLocale(): Lang | null {
  if (typeof document === "undefined") return null;
  const m = document.cookie.match(/(?:^|; )NEXT_LOCALE=([^;]+)/);
  const v = m?.[1] as Lang | undefined;
  return v && ["en", "zh", "ja", "ko"].includes(v) ? v : null;
}

export default function LoginPage() {
  const router = useRouter();
  const params = useSearchParams();
  const [lang, setLang] = useState<Lang>("en");
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const q = params.get("lang") as Lang | null;
    const fromQuery = q && ["en", "zh", "ja", "ko"].includes(q) ? q : null;
    setLang(fromQuery || readCookieLocale() || "en");
  }, [params]);

  const t = DICT[lang];

  function changeLang(next: Lang) {
    setLang(next);
    document.cookie = `NEXT_LOCALE=${next}; path=/; max-age=31536000`;
  }

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
      setErr((e as { message?: string })?.message || t.fail);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-6" style={{ background: "var(--bg)" }}>
      <div className="w-full max-w-[420px] rounded-xl border p-9" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
        <div className="mb-6 flex justify-end">
          <select
            value={lang}
            onChange={(e) => changeLang(e.target.value as Lang)}
            className="rounded-md border px-2 py-1 text-[13px]"
            style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink-soft)" }}
          >
            {LANGS.map((l) => (
              <option key={l.code} value={l.code}>{l.label}</option>
            ))}
          </select>
        </div>
        <h1 className="mb-2 text-center font-[var(--serif)] text-[32px] font-medium" style={{ color: "var(--ink)" }}>
          FateLumen
        </h1>
        <p className="mb-8 text-center text-[15px] font-light" style={{ color: "var(--ink-soft)" }}>
          {mode === "login" ? t.subLogin : t.subRegister}
        </p>
        <form onSubmit={handleSubmit} className="space-y-5">
          {mode === "register" && (
            <div className="space-y-2">
              <Label htmlFor="name" style={{ color: "var(--ink-soft)" }}>{t.nameLabel}</Label>
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} placeholder={t.name} />
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="email" style={{ color: "var(--ink-soft)" }}>{t.email}</Label>
            <Input id="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password" style={{ color: "var(--ink-soft)" }}>{t.password}</Label>
            <Input id="password" type="password" required value={password} onChange={(e) => setPassword(e.target.value)} placeholder={t.pwHint} />
          </div>
          {err && <p className="text-[14px]" style={{ color: "var(--fire, #b8473e)" }}>{err}</p>}
          <Button type="submit" disabled={loading} className="w-full" style={{ background: "var(--gold-deep)", color: "var(--bg)" }}>
            {loading ? t.loading : mode === "login" ? t.signIn : t.signUp}
          </Button>
        </form>
        <button
          type="button"
          onClick={() => { setMode(mode === "login" ? "register" : "login"); setErr(""); }}
          className="mt-6 w-full text-center text-[14px] font-light underline-offset-4 hover:underline"
          style={{ color: "var(--ink-faint)" }}
        >
          {mode === "login" ? t.toRegister : t.toLogin}
        </button>
      </div>
    </div>
  );
}
