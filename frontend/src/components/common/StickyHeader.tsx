"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Menu } from "lucide-react";
import ThemeSwitcher from "@/components/theme/ThemeSwitcher";
import { LanguageSwitcher } from "@/components/common/LanguageSwitcher";
import MobileDrawer from "@/components/common/MobileDrawer";
import BrandMark from "@/components/common/BrandMark";
import Link from "next/link";

export default function StickyHeader({ locale }: { locale: string }) {
  const t = useTranslations("nav");
  const [scrolled, setScrolled] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <>
      <header
        className="sticky top-0 z-50 h-16 transition-all duration-300"
        style={{
          paddingTop: "env(safe-area-inset-top)",
          background: scrolled
            ? "oklch(92% 0.012 70 / 0.82)"
            : "var(--bg)",
          backdropFilter: scrolled ? "saturate(160%) blur(14px)" : "none",
          borderBottom: scrolled ? "1px solid var(--line-soft)" : "1px solid transparent",
        }}
      >
        <div className="mx-auto flex h-full max-w-[1200px] items-center px-5 md:px-7">
          {/* Logo */}
          <Link
            href={`/${locale}`}
            className="flex items-center gap-2.5 font-[var(--serif)] text-[21px] font-semibold tracking-[.3px] text-[var(--ink)] shrink-0"
          >
            <BrandMark size={36} />
            FateLumen
          </Link>

          {/* Desktop nav links - centered */}
          <nav className="hidden md:flex flex-1 items-center justify-center gap-9">
            <Link href="#how" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("howItWorks")}</Link>
            <Link href="#report" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("sample")}</Link>
            <Link href="#pricing" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("pricing")}</Link>
            <Link href="#faq" className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("faq")}</Link>
            <Link href={`/${locale}/learn`} className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("learn")}</Link>
            <Link href={`/${locale}/cases`} className="text-[13px] tracking-[.3px] text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("cases")}</Link>
          </nav>

          {/* Desktop right cluster: lang capsule + theme capsule + divider + login */}
          <div className="hidden md:flex items-center gap-3 shrink-0">
            <LanguageSwitcher currentLocale={locale} />
            <ThemeSwitcher />
            <span
              className="h-[18px] w-px"
              style={{ background: "var(--line-soft)" }}
            />
            <Link
              href={`/login?lang=${locale}`}
              className="inline-flex items-center justify-center font-[var(--serif)] text-[13px] font-semibold transition-all"
              style={{
                background: "var(--gold-deep)",
                color: "var(--bg-card)",
                borderRadius: "9999px",
                height: "37px",
                padding: "0 22px",
                display: "inline-flex",
                alignItems: "center",
                border: "none",
                boxShadow: "0 4px 12px rgba(0,0,0,0.1)",
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = "var(--gold)";
                e.currentTarget.style.transform = "translateY(-2px)";
                e.currentTarget.style.boxShadow = "0 8px 20px rgba(0,0,0,0.15)";
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = "var(--gold-deep)";
                e.currentTarget.style.transform = "translateY(0)";
                e.currentTarget.style.boxShadow = "0 4px 12px rgba(0,0,0,0.1)";
              }}
              onMouseDown={(e) => {
                e.currentTarget.style.transform = "translateY(0)";
              }}
            >
              {t("signIn")}
            </Link>
          </div>

          {/* Mobile hamburger */}
          <button
            onClick={() => setDrawerOpen(true)}
            className="md:hidden ml-auto flex items-center justify-center rounded-lg transition-colors"
            style={{
              width: "44px",
              height: "44px",
              minWidth: "44px",
              minHeight: "44px",
              touchAction: "manipulation",
            }}
            aria-label="Open menu"
          >
            <Menu size={22} color="var(--ink)" />
          </button>
        </div>
      </header>

      {/* Mobile drawer */}
      <MobileDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        locale={locale}
      />
    </>
  );
}
