"use client";

import { THEMES } from "@/lib/theme/themes";
import { useThemeStore } from "@/lib/theme/useThemeStore";
import { useTranslations } from "next-intl";

export default function ThemeSwitcher() {
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const t = useTranslations();

  return (
    <select
      value={theme}
      onChange={(e) => setTheme(e.target.value)}
      className="rounded border border-[var(--line)] bg-[var(--bg-card)] px-2 py-1 text-xs text-[var(--ink-soft)]"
    >
      {THEMES.map((th) => (
        <option key={th.id} value={th.id} disabled={!th.available}>
          {t(th.nameKey)}{th.available ? "" : ` (${t("theme.comingSoon")})`}
        </option>
      ))}
    </select>
  );
}
