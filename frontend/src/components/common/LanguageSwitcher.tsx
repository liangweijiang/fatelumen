"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { usePathname, useRouter } from "@/i18n/navigation";
import { Globe, ChevronDown, Check } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";

const langs = ["en", "zh", "ja", "ko"] as const;

export function LanguageSwitcher({ currentLocale }: { currentLocale: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const t = useTranslations("lang");
  const [open, setOpen] = useState(false);

  return (
    <DropdownMenu onOpenChange={setOpen}>
      <DropdownMenuTrigger className="header-capsule lang">
        <Globe size={14} className="capsule-prefix-icon" />
        {t(currentLocale)}
        <ChevronDown
          size={13}
          className={`capsule-chevron ${open ? "open" : ""}`}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {langs.map((code) => (
          <DropdownMenuItem
            key={code}
            onClick={() => router.replace(pathname, { locale: code })}
            className={currentLocale === code ? "font-semibold" : ""}
          >
            <span className="flex-1">{t(code)}</span>
            {currentLocale === code && (
              <Check size={14} color="var(--gold-deep)" />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
