"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";
import Link from "next/link";

export function SampleReport() {
  const t = useTranslations("sample");
  const textRef = useReveal();
  const docRef = useReveal();
  return (
    <div className="border-y" style={{ background: "var(--bg-soft)", borderColor: "var(--line)" }}>
      <section id="report" className="px-0 py-24 max-md:py-[68px]">
        <div className="mx-auto max-w-[var(--maxw)] px-7">
          <div className="grid grid-cols-[1fr_1.1fr] items-center gap-14 max-md:grid-cols-1 max-md:gap-9">
            <div ref={textRef} className="reveal">
              <span className="text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">{t("eyebrow")}</span>
              <h2 className="my-4 font-[var(--serif)] text-[40px] font-medium leading-[1.16] tracking-[-.3px] max-md:text-[30px]">{t("title")}</h2>
              <p className="mb-6 text-base font-light text-[var(--ink-soft)]">{t("desc")}</p>
              <ul className="list-none">
                {[t("item1"), t("item2"), t("item3"), t("item4")].map((item) => (
                  <li key={item} className="flex gap-3 items-start py-[9px] text-[15px] font-light text-[var(--ink-soft)]">
                    <svg className="shrink-0 mt-1 text-[var(--gold-deep)]" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2"><path d="M20 6 9 17l-5-5" /></svg>
                    {item}
                  </li>
                ))}
              </ul>
              <Link href="#pricing" className="inline-flex mt-4 h-11 items-center gap-2 rounded-lg px-[22px] text-sm font-medium text-white transition-all" style={{ background: "var(--gold)", boxShadow: "0 1px 2px rgba(168,133,26,.3)" }}>{t("cta")}</Link>
            </div>
            <div ref={docRef} className="reveal rounded-2xl border p-10 px-9" style={{ background: "var(--bg-card)", borderColor: "var(--line)", boxShadow: "0 24px 60px -34px rgba(26,23,21,.5)" }}>
              <div className="border-b pb-4 mb-5 text-center font-[var(--serif)] text-sm italic tracking-[.5px] text-[var(--ink-faint)]" style={{ borderColor: "var(--line-soft)" }}>{t("docH")}</div>
              {[
                { title: t("sec1title"), desc: t("sec1desc"), bars: [true, true, true, false, false] },
                { title: t("sec2title"), desc: t("sec2desc") },
                { title: t("sec3title"), desc: t("sec3desc") },
              ].map((sec, i) => (
                <div key={i} className="mb-[18px]">
                  <h4 className="mb-1.5 flex items-center gap-2 font-[var(--serif)] text-[17px] font-medium">
                    <span className="italic text-[var(--gold-deep)]">{["I.", "IV.", "VIII."][i]}</span> {sec.title}
                  </h4>
                  <p className="text-[13px] font-light leading-[1.6] text-[var(--ink-soft)]">{sec.desc}</p>
                  {sec.bars && (
                    <div className="flex gap-2 mt-2">
                      {sec.bars.map((f, j) => (
                        <span key={j} className="flex-1 h-[5px] rounded-sm" style={{ background: f ? "var(--gold)" : "var(--line-soft)" }} />
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
