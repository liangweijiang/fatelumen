"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { exchangeGoogleLogin } from "@/lib/auth-api";

function safeNextPath(next: string | null): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.includes("\\")) {
    return "/en/dashboard";
  }
  return next;
}

export default function GoogleLoginCallbackPage() {
  const router = useRouter();
  const params = useSearchParams();
  const [error, setError] = useState("");
  const exchangeStarted = useRef(false);

  useEffect(() => {
	// React development mode can re-run effects. The backend code is
	// intentionally one-time, so the browser must never submit it twice.
	if (exchangeStarted.current) return;
	exchangeStarted.current = true;

    const code = params.get("code");
    if (!code) {
      setError("The login session is missing. Please return and try again.");
      return;
    }
    const next = safeNextPath(sessionStorage.getItem("fatelumen_login_next"));
    sessionStorage.removeItem("fatelumen_login_next");
    exchangeGoogleLogin(code)
      .then(() => {
        window.history.replaceState({}, document.title, "/login/callback");
        router.replace(next);
      })
      .catch(() => setError("Google sign-in could not be completed. Please try again."));
  }, [params, router]);

  return (
    <main className="flex min-h-screen items-center justify-center px-6" style={{ background: "var(--bg)" }}>
      <section className="w-full max-w-md rounded-xl border p-8 text-center" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
        <h1 className="font-[var(--serif)] text-2xl" style={{ color: "var(--ink)" }}>FateLumen</h1>
        <p className="mt-4 text-sm" style={{ color: error ? "var(--fire, #b8473e)" : "var(--ink-soft)" }}>
          {error || "Completing your sign-in…"}
        </p>
        {error && <a className="mt-5 inline-block underline" href="/login">Return to sign in</a>}
      </section>
    </main>
  );
}
