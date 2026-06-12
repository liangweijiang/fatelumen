"use client";

import { useTranslations } from "next-intl";

export function TrustBar() {
  const t = useTranslations("trust");
  return (
    <div className="border-y" style={{ background: "var(--bg-soft)", borderColor: "var(--line)" }}>
      <div className="mx-auto flex max-w-[var(--maxw)] items-center justify-center gap-12 px-7 py-7 flex-wrap max-md:gap-6">
        <div className="text-center">
          <div className="font-[var(--serif)] text-[30px] font-medium leading-none text-[var(--ink)]">48,200+</div>
          <div className="mt-1.5 text-xs tracking-[1px] uppercase text-[var(--ink-faint)]">{t("readings")}</div>
        </div>
        <span className="h-9 w-px bg-[var(--line)] max-md:hidden" />
        <div className="text-center">
          <div className="font-[var(--serif)] text-[30px] font-medium leading-none text-[var(--ink)]">12</div>
          <div className="mt-1.5 text-xs tracking-[1px] uppercase text-[var(--ink-faint)]">{t("categories")}</div>
        </div>
        <span className="h-9 w-px bg-[var(--line)] max-md:hidden" />
        <div className="text-center">
          <div className="font-[var(--serif)] text-[30px] font-medium leading-none text-[var(--ink)]">Stripe</div>
          <div className="mt-1.5 text-xs tracking-[1px] uppercase text-[var(--ink-faint)]">{t("payment")}</div>
        </div>
        <span className="h-9 w-px bg-[var(--line)] max-md:hidden" />
        <div className="text-center">
          <div className="font-[var(--serif)] text-[30px] font-medium leading-none text-[var(--ink)]">4.8★</div>
          <div className="mt-1.5 text-xs tracking-[1px] uppercase text-[var(--ink-faint)]">{t("rating")}</div>
        </div>
      </div>
    </div>
  );
}
