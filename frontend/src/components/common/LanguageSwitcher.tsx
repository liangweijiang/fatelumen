"use client";

import { useRouter, usePathname } from "next/navigation";

const langs: { code: string; label: string }[] = [
  { code: "en", label: "EN" },
  { code: "zh", label: "中文" },
  { code: "ja", label: "日本語" },
  { code: "ko", label: "한국어" },
];

export function LanguageSwitcher({ currentLocale }: { currentLocale: string }) {
  const router = useRouter();
  const pathname = usePathname();

  const switchLang = (newLocale: string) => {
    const path = pathname.replace(/^\/[a-z]{2}/, `/${newLocale}`);
    router.push(path);
  };

  return (
    <select
      value={currentLocale}
      onChange={(e) => switchLang(e.target.value)}
      className="rounded border border-[var(--line)] bg-[var(--bg-card)] px-2 py-1 text-xs text-[var(--ink-soft)]"
      style={{ fontFamily: "var(--sans)" }}
    >
      {langs.map((l) => (
        <option key={l.code} value={l.code}>
          {l.label}
        </option>
      ))}
    </select>
  );
}
