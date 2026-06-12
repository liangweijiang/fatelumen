import { useTranslations } from "next-intl";
import Link from "next/link";
import { useReveal } from "./useReveal";

export function Hero() {
  const t = useTranslations("hero");
  const chartRef = useReveal();
  return (
    <section className="relative px-0 pb-20 pt-[120px] text-center max-md:pt-20 max-md:pb-14">
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <span className="mb-7 inline-flex items-center gap-3 text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">
          <span className="h-px w-8 bg-[var(--line)]" />
          {t("eyebrow")}
          <span className="h-px w-8 bg-[var(--line)]" />
        </span>
        <h1 className="mx-auto mb-[26px] max-w-[880px] font-[var(--serif)] text-[68px] font-medium leading-[1.06] tracking-[-.5px] max-md:text-[42px]">
          {t("title")} <br /><em className="italic text-[var(--gold-deep)]">{t("titleEm")}</em> {t("titleEnd")}
        </h1>
        <p className="mx-auto mb-[38px] max-w-[600px] text-[19px] font-light text-[var(--ink-soft)] max-md:text-[17px]">{t("sub")}</p>
        <div className="flex gap-[14px] justify-center flex-wrap">
          <Link href="/login" className="inline-flex gap-2 h-[50px] px-7 text-[15px] items-center justify-center rounded-lg font-medium transition-all" style={{ background: "var(--gold)", color: "#fff", boxShadow: "0 1px 2px rgba(168,133,26,.3)" }}>{t("cta")}</Link>
          <Link href="#how" className="inline-flex gap-2 h-[50px] px-7 text-[15px] items-center justify-center rounded-lg border font-medium transition-all" style={{ borderColor: "var(--line)", color: "var(--ink)" }}>{t("secondary")}</Link>
        </div>
        <p className="mt-[22px] text-[13px] tracking-[.3px] text-[var(--ink-faint)]">{t("note")}</p>

        <div ref={chartRef} className="reveal mx-auto mt-16 max-w-[560px]">
          <div className="relative overflow-hidden rounded-2xl border p-9 px-7" style={{ background: "var(--bg-card)", borderColor: "var(--line)", boxShadow: "0 20px 50px -28px rgba(26,23,21,.4)" }}>
            <div className="absolute inset-0 pointer-events-none" style={{ background: "radial-gradient(circle at 50% 0%, rgba(201,162,39,.08), transparent 60%)" }} />
            <div className="relative">
              <div className="mb-[22px] text-center font-[var(--serif)] text-[15px] italic tracking-[.5px] text-[var(--ink-faint)]">{t("chartTitle")}</div>
              <div className="grid grid-cols-4 gap-[14px]">
                {[{ lab: t("hour"), stem: "丙", branch: "寅" }, { lab: t("day"), stem: "甲", branch: "子" }, { lab: t("month"), stem: "辛", branch: "卯" }, { lab: t("year"), stem: "戊", branch: "午" }].map((p) => (
                  <div key={p.lab} className="text-center">
                    <div className="mb-2.5 text-[10px] tracking-[2px] uppercase text-[var(--ink-faint)]">{p.lab}</div>
                    <div className="flex flex-col gap-1.5">
                      <span className="flex h-12 items-center justify-center rounded-lg border text-2xl max-md:h-10 max-md:text-xl" style={{ borderColor: "var(--line-soft)", background: "var(--bg)", color: "var(--gold-deep)" }}>{p.stem}</span>
                      <span className="flex h-12 items-center justify-center rounded-lg border text-2xl max-md:h-10 max-md:text-xl" style={{ borderColor: "var(--line-soft)", background: "var(--bg)", color: "var(--ink)" }}>{p.branch}</span>
                    </div>
                  </div>
                ))}
              </div>
              <div className="mt-6 flex justify-center gap-[18px] border-t pt-[22px]" style={{ borderColor: "var(--line-soft)" }}>
                {["wood", "fire", "earth", "metal", "water"].map((wx) => (
                  <div key={wx} className="flex flex-col items-center gap-1.5 text-[11px] text-[var(--ink-faint)]">
                    <span className="flex h-[30px] w-[30px] items-center justify-center rounded-full font-[var(--serif)] text-[15px] text-white" style={{ background: `var(--wuxing-${wx})` }}>{t(wx as "wood")}</span>
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
