"use client";

import { useTranslations } from "next-intl";

export function TrustBar() {
  const t = useTranslations("trust");
  return (
    <div
      className="border-y"
      style={{ background: "var(--bg-soft)", borderColor: "var(--line)" }}
    >
      <div className="mx-auto flex max-w-[var(--maxw)] items-center justify-center gap-14 px-7 py-9 flex-wrap max-md:gap-6 max-md:py-6">
        <div className="text-center">
          <div
            className="gold-embossed font-[var(--serif)] text-[38px] font-medium leading-none tracking-[-.5px] max-md:text-[30px]"
            style={{ color: "var(--ink)" }}
          >
            48,200+
          </div>
          <div className="mt-2 text-[11px] tracking-[2px] uppercase text-[var(--ink-faint)]">
            {t("readings")}
          </div>
        </div>
        <span className="h-10 w-px bg-[var(--line)] max-md:hidden" />
        <div className="text-center">
          <div
            className="gold-embossed font-[var(--serif)] text-[38px] font-medium leading-none tracking-[-.5px] max-md:text-[30px]"
            style={{ color: "var(--ink)" }}
          >
            12
          </div>
          <div className="mt-2 text-[11px] tracking-[2px] uppercase text-[var(--ink-faint)]">
            {t("categories")}
          </div>
        </div>
        <span className="h-10 w-px bg-[var(--line)] max-md:hidden" />
        <div className="text-center">
          <div
            className="gold-embossed font-[var(--serif)] text-[38px] font-medium leading-none tracking-[-.5px] max-md:text-[30px]"
            style={{ color: "var(--ink)" }}
          >
            Stripe
          </div>
          <div className="mt-2 text-[11px] tracking-[2px] uppercase text-[var(--ink-faint)]">
            {t("payment")}
          </div>
        </div>
        <span className="h-10 w-px bg-[var(--line)] max-md:hidden" />
        <div className="text-center">
          <div
            className="gold-embossed font-[var(--serif)] text-[38px] font-medium leading-none tracking-[-.5px] max-md:text-[30px]"
            style={{ color: "var(--ink)" }}
          >
            4.8★
          </div>
          <div className="mt-2 text-[11px] tracking-[2px] uppercase text-[var(--ink-faint)]">
            {t("rating")}
          </div>
        </div>
      </div>
    </div>
  );
}

export function GoldDivider() {
  return (
    <div className="gold-divider py-1" />
  );
}
