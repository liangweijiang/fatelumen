"use client";

import { useState } from "react";
import { THEMES } from "@/lib/theme/themes";
import { useThemeStore } from "@/lib/theme/useThemeStore";
import { useTranslations } from "next-intl";
import { Palette, ChevronDown, Check, Lock } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";

export default function ThemeSwitcher() {
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const t = useTranslations();
  const [open, setOpen] = useState(false);

  const current = THEMES.find((th) => th.id === theme);

  return (
    <DropdownMenu onOpenChange={setOpen}>
      <DropdownMenuTrigger className="header-capsule theme">
        <Palette size={14} className="capsule-prefix-icon" />
        {current ? t(current.nameKey) : theme}
        <ChevronDown
          size={13}
          className={`capsule-chevron ${open ? "open" : ""}`}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {THEMES.map((th) => (
          <DropdownMenuItem
            key={th.id}
            disabled={!th.available}
            onClick={() => th.available && setTheme(th.id)}
            className={theme === th.id ? "font-semibold" : ""}
          >
            <span className="flex-1">
              {t(th.nameKey)}
              {!th.available && (
                <span className="ml-2 text-xs" style={{ color: "var(--ink-faint)" }}>
                  Coming soon
                </span>
              )}
            </span>
            {theme === th.id ? (
              <Check size={14} color="var(--gold-deep)" />
            ) : !th.available ? (
              <Lock size={12} color="var(--ink-faint)" />
            ) : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
