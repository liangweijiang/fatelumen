"use client";

import { useTranslations } from "next-intl";
import Link from "next/link";
import { useReveal } from "@/hooks/useReveal";

export function Hero() {
  const t = useTranslations("hero");
  const chartRef = useReveal();

  const pillars = [
    { lab: t("hour"), stem: "丙", branch: "寅" },
    { lab: t("day"), stem: "甲", branch: "子" },
    { lab: t("month"), stem: "辛", branch: "卯" },
    { lab: t("year"), stem: "戊", branch: "午" },
  ];

  return (
    <section className="relative px-0 pb-32 pt-[160px] text-center max-md:pb-20 max-md:pt-28">
      {/* Seal stamp */}
      <div
        className="absolute right-8 top-24 z-10 select-none rounded-sm border px-1.5 py-0.5 text-[13px] font-bold leading-none tracking-wider opacity-20 max-md:right-4 max-md:top-20"
        style={{
          borderColor: "var(--seal-red)",
          color: "var(--seal-red)",
          fontFamily: "var(--serif)",
          transform: "rotate(8deg)",
        }}
      >
        命
      </div>

      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <span className="mb-8 inline-flex items-center gap-3 text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">
          <span className="h-px w-8 bg-[var(--line)]" />
          {t("eyebrow")}
          <span className="h-px w-8 bg-[var(--line)]" />
        </span>
        <h1 className="mx-auto mb-8 max-w-[880px] font-[var(--serif)] text-[68px] font-medium leading-[1.1] tracking-[-.3px] max-md:text-[42px]">
          {t("title")}{" "}
          <br />
          <em className="italic gold-embossed" style={{ color: "var(--gold-deep)" }}>{t("titleEm")}</em>{" "}
          {t("titleEnd")}
        </h1>
        <p className="mx-auto mb-10 max-w-[620px] text-[19px] leading-relaxed font-light text-[var(--ink-soft)] max-md:text-[17px]">
          {t("sub")}
        </p>
        <div className="flex gap-[14px] justify-center flex-wrap">
          <Link
            href="/login"
            className="btn-gold inline-flex h-[52px] items-center gap-2 rounded-lg px-7 text-[15px] font-medium transition-all"
          >
            {t("cta")}
          </Link>
          <Link
            href="#how"
            className="btn-ghost inline-flex h-[52px] items-center gap-2 rounded-lg border px-7 text-[15px] font-medium transition-all"
          >
            {t("secondary")}
          </Link>
        </div>
        <p className="mt-6 text-[13px] tracking-[.3px] text-[var(--ink-faint)]">{t("note")}</p>

        {/* Chart card */}
        <div ref={chartRef} className="reveal mx-auto mt-20 max-w-[560px]">
          <div
            className="ancient-frame relative overflow-hidden rounded-xl p-9 px-7"
            style={{
              background: "var(--bg-card)",
            }}
          >
            <div
              className="pointer-events-none absolute inset-0"
              style={{
                background: "radial-gradient(circle at 50% 0%, rgba(201,162,39,.1), transparent 60%)",
              }}
            />
            <div className="relative">
              <div
                className="mb-6 text-center font-[var(--serif)] text-[14px] italic tracking-[.8px] text-[var(--ink-faint)]"
              >
                {t("chartTitle")}
              </div>
              <div className="grid grid-cols-4 gap-4">
                {pillars.map((p) => (
                  <div key={p.lab} className="text-center">
                    <div className="mb-3 font-[var(--serif)] text-[9px] tracking-[3px] uppercase text-[var(--ink-faint)]">
                      {p.lab}
                    </div>
                    <div className="flex flex-col gap-2">
                      <span
                        className="flex h-[52px] items-center justify-center rounded-md border text-[28px] font-medium tracking-[.08em] max-md:h-10 max-md:text-[22px]"
                        style={{
                          borderColor: "var(--line-soft)",
                          background: "var(--bg)",
                          color: "var(--gold-deep)",
                        }}
                      >
                        {p.stem}
                      </span>
                      <span
                        className="flex h-[52px] items-center justify-center rounded-md border text-[28px] font-medium tracking-[.08em] max-md:h-10 max-md:text-[22px]"
                        style={{
                          borderColor: "var(--line-soft)",
                          background: "var(--bg)",
                          color: "var(--ink)",
                        }}
                      >
                        {p.branch}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
              <div
                className="mt-6 flex justify-center gap-[20px] border-t pt-[22px]"
                style={{ borderColor: "var(--line-soft)" }}
              >
                {["wood", "fire", "earth", "metal", "water"].map((wx) => (
                  <div key={wx} className="flex flex-col items-center gap-1.5 text-[11px] text-[var(--ink-faint)]">
                    <span
                      className="flex h-[30px] w-[30px] items-center justify-center rounded-full font-[var(--serif)] text-[15px] text-white"
                      style={{ background: `var(--wuxing-${wx})` }}
                    >
                      {t(wx as "wood")}
                    </span>
                    {t(wx as "wood")}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
