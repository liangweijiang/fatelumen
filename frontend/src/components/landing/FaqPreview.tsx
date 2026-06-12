"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";
import Link from "next/link";

export function FaqPreview({ locale }: { locale: string }) {
  const t = useTranslations("faqPreview");
  const [openIdx, setOpenIdx] = useState<number | null>(null);
  const headRef = useReveal();
  const faqRef = useReveal();

  const items = [
    { q: t("q1"), a: t("a1") },
    { q: t("q2"), a: t("a2") },
    { q: t("q3"), a: t("a3") },
    { q: t("q4"), a: t("a4") },
    { q: t("q5"), a: t("a5") },
    { q: t("q6"), a: t("a6") },
    { q: t("q7"), a: t("a7") },
    { q: t("q8"), a: t("a8") },
  ];

  return (
    <section id="faq" className="px-0 py-24 max-md:py-[68px]">
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <div ref={headRef} className="reveal mx-auto mb-[60px] max-w-[620px] text-center">
          <span className="mb-[18px] block text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">{t("eyebrow")}</span>
          <h2 className="mb-4 font-[var(--serif)] text-[44px] font-medium leading-[1.14] tracking-[-.3px] max-md:text-[32px]">{t("title")}</h2>
        </div>
        <div ref={faqRef} className="reveal mx-auto max-w-[740px] border-t" style={{ borderColor: "var(--line)" }}>
          {items.map((item, i) => (
            <div key={i} className={`border-b ${openIdx === i ? "open" : ""}`} style={{ borderColor: "var(--line)" }}>
              <button
                onClick={() => setOpenIdx(openIdx === i ? null : i)}
                className="flex w-full items-center justify-between gap-4 px-0 py-6 text-left font-[var(--serif)] text-[19px] font-medium text-[var(--ink)] bg-transparent border-none cursor-pointer"
              >
                {item.q}
                <span className="shrink-0 font-sans text-[var(--ink-faint)] transition-transform duration-250" style={{ transform: openIdx === i ? "rotate(45deg)" : "" }}>＋</span>
              </button>
              <div className="overflow-hidden transition-[max-height] duration-300" style={{ maxHeight: openIdx === i ? "200px" : "0" }}>
                <p className="max-w-[92%] pb-6 text-[15px] font-light text-[var(--ink-soft)]">{item.a}</p>
              </div>
            </div>
          ))}
          <div className="pt-6 text-center">
            <Link href={`/${locale}/faq`} className="text-sm text-[var(--ink-soft)] hover:text-[var(--ink)]">{t("viewAll")}</Link>
          </div>
        </div>
      </div>
    </section>
  );
}
