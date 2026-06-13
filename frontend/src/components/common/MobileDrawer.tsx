"use client";

import { useEffect, useRef } from "react";
import { X, Check, Lock } from "lucide-react";
import { useTranslations } from "next-intl";
import { usePathname, useRouter } from "@/i18n/navigation";
import { THEMES } from "@/lib/theme/themes";
import { useThemeStore } from "@/lib/theme/useThemeStore";
import Link from "next/link";

const langs: { code: string; label: string }[] = [
  { code: "en", label: "English" },
  { code: "zh", label: "中文" },
  { code: "ja", label: "日本語" },
  { code: "ko", label: "한국어" },
];

export default function MobileDrawer({
  open,
  onClose,
  locale,
}: {
  open: boolean;
  onClose: () => void;
  locale: string;
}) {
  const t = useTranslations("nav");
  const tt = useTranslations();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const router = useRouter();
  const pathname = usePathname();
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  const navLinks = [
    { href: "#how", label: t("howItWorks") },
    { href: "#report", label: t("sample") },
    { href: "#pricing", label: t("pricing") },
    { href: "#faq", label: t("faq") },
    { href: `/${locale}/learn`, label: t("learn") },
    { href: `/${locale}/cases`, label: t("cases") },
  ];

  const handleNavClick = (href: string) => {
    onClose();
    if (href.startsWith("#")) {
      setTimeout(() => {
        const el = document.querySelector(href);
        if (el) el.scrollIntoView({ behavior: "smooth" });
      }, 100);
    }
  };

  return (
    <div className="mobile-drawer-overlay">
      <div className="mobile-drawer-backdrop" onClick={onClose} />
      <div ref={panelRef} className="mobile-drawer-panel">
        <div className="flex items-center justify-between px-5 py-3 border-b border-[var(--line-soft)]">
          <span
            className="font-[var(--serif)] text-[17px] font-semibold"
            style={{ color: "var(--ink)" }}
          >
            Menu
          </span>
          <button onClick={onClose} className="drawer-close-btn" aria-label="Close menu">
            <X size={22} color="var(--ink)" />
          </button>
        </div>

        <nav className="px-5 pt-2 pb-1">
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              onClick={() => handleNavClick(link.href)}
              className="drawer-link"
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div className="mx-5 h-px" style={{ background: "var(--line)" }} />

        <div className="px-5 pt-3 pb-2">
          <span className="drawer-section-label">Language</span>
          {langs.map((l) => (
            <button
              key={l.code}
              className="drawer-item-btn"
              onClick={() => {
                router.replace(pathname, { locale: l.code });
                onClose();
              }}
            >
              <span>{l.label}</span>
              {locale === l.code && <Check size={14} color="var(--gold-deep)" />}
            </button>
          ))}
        </div>

        <div className="mx-5 h-px" style={{ background: "var(--line)" }} />

        <div className="px-5 pt-3 pb-2">
          <span className="drawer-section-label">Theme</span>
          {THEMES.map((th) => (
            <button
              key={th.id}
              className="drawer-item-btn"
              disabled={!th.available}
              onClick={() => {
                if (th.available) setTheme(th.id);
              }}
            >
              <span>
                {tt(th.nameKey)}
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
            </button>
          ))}
        </div>

        <div className="mx-5 h-px" style={{ background: "var(--line)" }} />

        <div className="px-5 py-5">
          <Link
            href={`/login?lang=${locale}`}
            onClick={onClose}
            className="flex items-center justify-center w-full py-3.5 rounded-full text-[15px] font-[var(--serif)] font-medium transition-all hover:translate-y-[-1px] active:translate-y-0"
            style={{
              background: "var(--gold-deep)",
              color: "var(--bg-card)",
              minHeight: "44px",
            }}
          >
            {t("signIn")}
          </Link>
        </div>
      </div>
    </div>
  );
}
