"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";

export function HowItWorks() {
  const t = useTranslations("how");
  const headRef = useReveal();
  const stepsRef = useReveal();
  const steps = [
    { num: "01", title: t("step1title"), desc: t("step1desc") },
    { num: "02", title: t("step2title"), desc: t("step2desc") },
    { num: "03", title: t("step3title"), desc: t("step3desc") },
  ];
  return (
    <section id="how" className="px-0 py-[120px] max-md:py-[80px]">
      <div className="mx-auto max-w-[var(--maxw)] px-5 md:px-10">
        <div ref={headRef} className="reveal mx-auto mb-[72px] max-w-[620px] text-center">
          <span className="mb-[18px] block text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">{t("eyebrow")}</span>
          <h2 className="mb-5 font-[var(--serif)] text-[44px] font-medium leading-[1.18] tracking-[-.3px] max-md:text-[32px]">{t("title")}</h2>
          <p className="text-[17px] leading-relaxed font-light text-[var(--ink-soft)]">{t("sub")}</p>
        </div>
        <div ref={stepsRef} className="reveal mx-auto max-w-[720px]">
          {steps.map((s) => (
            <div key={s.num}>
              <div className="flex flex-col gap-4 py-10 md:flex-row md:items-start md:gap-10">
                <div
                  className="shrink-0 font-[var(--serif)] text-[64px] font-medium leading-none tracking-[-1px] max-md:text-[48px]"
                  style={{ color: "var(--gold-deep)" }}
                >
                  {s.num}
                </div>
                <div>
                  <h3 className="mb-3 font-[var(--serif)] text-[24px] font-medium">{s.title}</h3>
                  <p className="text-[15px] leading-relaxed font-light text-[var(--ink-soft)] max-w-[56ch]">{s.desc}</p>
                </div>
              </div>
              <div className="gold-divider !mx-0 !max-w-full !px-0" />
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
