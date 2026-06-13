"use client";

import { useTranslations } from "next-intl";

export function TrustBar() {
  const t = useTranslations("trust");
  return (
    <div
      className="border-y"
      style={{ borderColor: "var(--line-soft)" }}
    >
      <div className="mx-auto grid max-w-[var(--maxw)] grid-cols-4 max-md:grid-cols-2 px-5 md:px-10">
        <div className="text-center py-[34px] px-3" style={{ borderRight: "1px solid var(--line-soft)" }}>
          <div
            className="font-[var(--serif)] text-[32px] text-[var(--ink)]"
          >
            48,200+
          </div>
          <div className="mt-[7px] text-xs tracking-[1.5px] text-[var(--ink-faint)]">
            {t("readings")}
          </div>
        </div>
        <div className="text-center py-[34px] px-3 max-md:border-r-0" style={{ borderRight: "1px solid var(--line-soft)" }}>
          <div
            className="font-[var(--serif)] text-[32px] text-[var(--ink)]"
          >
            12
          </div>
          <div className="mt-[7px] text-xs tracking-[1.5px] text-[var(--ink-faint)]">
            {t("categories")}
          </div>
        </div>
        <div className="text-center py-[34px] px-3" style={{ borderRight: "1px solid var(--line-soft)" }}>
          <div
            className="font-[var(--serif)] text-[32px] text-[var(--ink)]"
          >
            Stripe
          </div>
          <div className="mt-[7px] text-xs tracking-[1.5px] text-[var(--ink-faint)]">
            {t("payment")}
          </div>
        </div>
        <div className="text-center py-[34px] px-3">
          <div
            className="font-[var(--serif)] text-[32px] text-[var(--ink)]"
          >
            4.8★
          </div>
          <div className="mt-[7px] text-xs tracking-[1.5px] text-[var(--ink-faint)]">
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
