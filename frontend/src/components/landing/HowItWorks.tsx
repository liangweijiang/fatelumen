import { useTranslations } from "next-intl";
import { useReveal } from "./useReveal";

export function HowItWorks() {
  const t = useTranslations("how");
  const headRef = useReveal();
  const stepsRef = useReveal();
  const steps = [
    { num: t("step1num"), title: t("step1title"), desc: t("step1desc") },
    { num: t("step2num"), title: t("step2title"), desc: t("step2desc") },
    { num: t("step3num"), title: t("step3title"), desc: t("step3desc") },
  ];
  return (
    <section id="how" className="px-0 py-24 max-md:py-[68px]">
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <div ref={headRef} className="reveal mx-auto mb-[60px] max-w-[620px] text-center">
          <span className="mb-[18px] block text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">{t("eyebrow")}</span>
          <h2 className="mb-4 font-[var(--serif)] text-[44px] font-medium leading-[1.14] tracking-[-.3px] max-md:text-[32px]">{t("title")}</h2>
          <p className="text-[17px] font-light text-[var(--ink-soft)]">{t("sub")}</p>
        </div>
        <div ref={stepsRef} className="reveal grid grid-cols-3 border-t max-md:grid-cols-1" style={{ borderColor: "var(--line)" }}>
          {steps.map((s) => (
            <div key={s.num} className="border-b border-r p-[42px] px-8 transition-colors hover:bg-[var(--bg-card)] max-md:border-r-0 last:border-r-0" style={{ borderColor: "var(--line)" }}>
              <div className="mb-[18px] font-[var(--serif)] text-[36px] italic leading-none text-[var(--gold-deep)]">{s.num}</div>
              <h3 className="mb-3 font-[var(--serif)] text-[23px] font-medium">{s.title}</h3>
              <p className="text-[15px] font-light text-[var(--ink-soft)]">{s.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
