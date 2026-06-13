"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";

export default function TermsPage() {
  const t = useTranslations("termsPage");
  const head = useReveal();
  const sections = [1, 2, 3, 4, 5, 6];
  return (
    <div className="py-20 max-md:py-14" style={{ background: "var(--bg)" }}>
      <div className="mx-auto max-w-[760px] px-7">
        <div ref={head} className="reveal mb-10">
          <h1 className="mb-2 font-[var(--serif)] text-[44px] font-medium tracking-[-.3px] max-md:text-[32px]" style={{ color: "var(--ink)" }}>{t("title")}</h1>
          <p className="text-xs" style={{ color: "var(--ink-faint)" }}>{t("updated")}</p>
        </div>
        <p className="mb-10 text-[16px] font-light leading-relaxed" style={{ color: "var(--ink-soft)" }}>{t("intro")}</p>
        {sections.map((n) => (
          <section key={n} className="mb-8">
            <h2 className="mb-3 font-[var(--serif)] text-[22px] font-medium" style={{ color: "var(--ink)" }}>{t(`s${n}Title`)}</h2>
            <p className="text-[15px] font-light leading-relaxed" style={{ color: "var(--ink-soft)" }}>{t(`s${n}Body`)}</p>
          </section>
        ))}
      </div>
    </div>
  );
}
