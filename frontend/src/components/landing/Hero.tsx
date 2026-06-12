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
    <section className="relative px-0 pb-24 pt-28 max-md:pb-16 max-md:pt-20">
      <div className="mx-auto max-w-[1200px] px-5 md:px-7">
        <div className="grid items-center gap-12 md:grid-cols-2">
          {/* Left: text */}
          <div>
            <span className="mb-6 inline-flex items-center gap-2 text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">
              <span className="h-px w-6 bg-[var(--line)]" />
              {t("eyebrow")}
            </span>
            <h1 className="mb-6 font-[var(--serif)] font-medium leading-[1.12] tracking-[-.3px] text-[clamp(36px,5vw,64px)]">
              {t("title")}{" "}
              <em className="gold-embossed italic" style={{ color: "var(--gold-deep)" }}>
                {t("titleEm")}
              </em>{" "}
              {t("titleEnd")}
            </h1>
            <p className="mb-8 max-w-[56ch] text-[17px] leading-relaxed font-light text-[var(--ink-soft)]">
              {t("sub")}
            </p>
            <div className="flex gap-3 flex-wrap">
              <Link
                href="/login"
                className="btn-gold inline-flex h-12 items-center gap-2 rounded-lg px-6 text-sm font-medium transition-all w-full sm:w-auto justify-center sm:justify-start"
                style={{ minHeight: "44px" }}
              >
                {t("cta")}
              </Link>
              <Link
                href="#how"
                className="btn-ghost inline-flex h-12 items-center gap-2 rounded-lg border px-6 text-sm font-medium transition-all w-full sm:w-auto justify-center sm:justify-start"
                style={{ minHeight: "44px" }}
              >
                {t("secondary")}
              </Link>
            </div>
            <p className="mt-5 text-xs tracking-[.3px] text-[var(--ink-faint)]">{t("note")}</p>
          </div>

          {/* Right: letterpress chart */}
          <div ref={chartRef} className="reveal relative">
            {/* Seal stamp */}
            <div
              className="absolute -right-1 -top-2 z-10 select-none rounded-sm border px-1.5 py-0.5 text-[13px] font-bold leading-none tracking-wider opacity-85"
              style={{
                borderColor: "var(--seal-red)",
                color: "var(--seal-red)",
                fontFamily: "var(--serif)",
                transform: "rotate(6deg)",
              }}
            >
              命
            </div>
            <div
              className="relative rounded-lg p-8 px-6"
              style={{
                background: "var(--bg-card)",
                border: "2px solid var(--line)",
                boxShadow: "inset 0 0 0 4px var(--bg-card), inset 0 0 0 5px var(--gold-soft)",
              }}
            >
              <div
                className="mb-6 text-center font-[var(--serif)] text-xs italic tracking-[.8px] text-[var(--ink-faint)]"
              >
                {t("chartTitle")}
              </div>
              <div className="grid grid-cols-4 gap-3">
                {pillars.map((p) => (
                  <div key={p.lab} className="text-center">
                    <div className="mb-3 font-[var(--serif)] text-[9px] tracking-[4px] uppercase text-[var(--ink-faint)]">
                      {p.lab}
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <span
                        className="flex h-14 items-center justify-center rounded-md font-[var(--serif)] text-[32px] font-medium tracking-wider max-md:text-[26px]"
                        style={{
                          color: "var(--ink)",
                          textShadow: "0 1px 0 oklch(98% 0.01 80 / 0.4)",
                        }}
                      >
                        {p.stem}
                      </span>
                      <span
                        className="flex h-14 items-center justify-center rounded-md font-[var(--serif)] text-[32px] font-medium tracking-wider max-md:text-[26px]"
                        style={{
                          color: "var(--ink-soft)",
                          textShadow: "0 1px 0 oklch(98% 0.01 80 / 0.4)",
                        }}
                      >
                        {p.branch}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
              <div
                className="mt-6 flex justify-center gap-5 border-t pt-5"
                style={{ borderColor: "var(--line-soft)" }}
              >
                {["wood", "fire", "earth", "metal", "water"].map((wx) => (
                  <div key={wx} className="flex flex-col items-center gap-1.5 text-xs tracking-wide text-[var(--ink-soft)]" style={{ fontFamily: "var(--body)" }}>
                    <span
                      className="flex h-4 w-4 rounded-full"
                      style={{ background: `var(--wuxing-${wx})` }}
                    />
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
