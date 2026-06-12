"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";
import Link from "next/link";

export function CtaBand() {
  const t = useTranslations("cta");
  const ref = useReveal();
  return (
    <section className="px-0 py-[104px] max-md:py-[72px]">
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <div ref={ref} className="reveal relative mx-auto max-w-[var(--maxw)] overflow-hidden rounded-[20px] px-8 py-20 text-center" style={{ background: "var(--bg-dark)", color: "var(--bg)" }}>
          <span className="absolute font-[var(--serif)] text-[340px] leading-none pointer-events-none select-none" style={{ color: "rgba(201,162,39,.06)", right: "-30px", top: "50%", transform: "translateY(-50%)" }}>八</span>
          <span className="relative mb-[18px] block text-xs font-medium tracking-[3px] uppercase" style={{ color: "var(--gold)" }}>{t("eyebrow")}</span>
          <h2 className="relative mb-[14px] font-[var(--serif)] text-[44px] font-medium tracking-[-.3px] max-md:text-[30px]">{t("title")}</h2>
          <p className="relative mb-8 text-[17px] font-light" style={{ color: "#b8b0a4" }}>{t("desc")}</p>
          <Link href="/login" className="relative inline-flex h-[50px] items-center gap-2 rounded-lg px-7 text-[15px] font-medium text-white transition-all" style={{ background: "var(--gold)", boxShadow: "0 1px 2px rgba(168,133,26,.3)" }}>{t("btn")}</Link>
        </div>
      </div>
    </section>
  );
}
