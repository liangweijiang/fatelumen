"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";
import Link from "next/link";

export function CtaBand() {
  const t = useTranslations("cta");
  const ref = useReveal();
  return (
    <section className="px-0 py-[120px] max-md:py-[80px]">
      <div className="mx-auto max-w-[var(--maxw)] px-5 md:px-10">
        <div
          ref={ref}
          className="reveal relative mx-auto max-w-[var(--maxw)] overflow-hidden rounded-[24px] px-8 py-24 text-center max-md:py-16"
          style={{ background: "var(--bg-dark)", color: "var(--bg)" }}
        >
          <span
            className="pointer-events-none absolute select-none font-[var(--serif)] text-[360px] leading-none"
            style={{
              color: "rgba(201,162,39,.055)",
              right: "-40px",
              top: "50%",
              transform: "translateY(-50%)",
            }}
          >
            八
          </span>
          <span
            className="relative mb-5 block text-xs font-medium tracking-[3px] uppercase"
            style={{ color: "var(--gold)" }}
          >
            {t("eyebrow")}
          </span>
          <h2 className="relative mb-5 font-[var(--serif)] text-[44px] font-medium tracking-[-.3px] max-md:text-[30px]">
            {t("title")}
          </h2>
          <p
            className="relative mb-9 text-[17px] leading-relaxed font-light"
            style={{ color: "#b8b0a4" }}
          >
            {t("desc")}
          </p>
          <Link
            href="/login"
            className="btn-gold relative inline-flex h-[52px] items-center gap-2 rounded-lg px-7 text-[15px] font-medium transition-all"
          >
            {t("btn")}
          </Link>
        </div>
      </div>
    </section>
  );
}
