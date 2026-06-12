"use client";

import { useState } from "react";
import { usePathname, useRouter } from "@/i18n/navigation";
import { Globe, ChevronDown, Check } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";

const langs: { code: string; label: string }[] = [
  { code: "en", label: "English" },
  { code: "zh", label: "中文" },
  { code: "ja", label: "日本語" },
  { code: "ko", label: "한국어" },
];

export function LanguageSwitcher({ currentLocale }: { currentLocale: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);

  const current = langs.find((l) => l.code === currentLocale);

  return (
    <DropdownMenu onOpenChange={setOpen}>
      <DropdownMenuTrigger className="header-capsule lang">
        <Globe size={14} className="capsule-prefix-icon" />
        {current?.label ?? currentLocale}
        <ChevronDown
          size={13}
          className={`capsule-chevron ${open ? "open" : ""}`}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {langs.map((l) => (
          <DropdownMenuItem
            key={l.code}
            onClick={() => router.replace(pathname, { locale: l.code })}
            className={currentLocale === l.code ? "font-semibold" : ""}
          >
            <span className="flex-1">{l.label}</span>
            {currentLocale === l.code && (
              <Check size={14} color="var(--gold-deep)" />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
