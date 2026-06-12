import { useTranslations } from "next-intl";
import { useReveal } from "./useReveal";
import Link from "next/link";

export function Pricing() {
  const t = useTranslations("pricing");
  const headRef = useReveal();
  const plansRef = useReveal();

  const CheckIcon = () => (
    <svg className="shrink-0 mt-[3px] text-[var(--gold-deep)]" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2"><path d="M20 6 9 17l-5-5" /></svg>
  );

  return (
    <section id="pricing" className="px-0 py-24 max-md:py-[68px]">
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <div ref={headRef} className="reveal mx-auto mb-[60px] max-w-[620px] text-center">
          <span className="mb-[18px] block text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">{t("eyebrow")}</span>
          <h2 className="mb-4 font-[var(--serif)] text-[44px] font-medium leading-[1.14] tracking-[-.3px] max-md:text-[32px]">{t("title")}</h2>
          <p className="text-[17px] font-light text-[var(--ink-soft)]">{t("sub")}</p>
        </div>
        <div ref={plansRef} className="reveal mx-auto grid max-w-[800px] grid-cols-2 gap-6 max-md:grid-cols-1">
          {/* Free plan */}
          <div className="flex flex-col rounded-2xl border p-[42px] px-[34px] transition-all hover:-translate-y-[3px] hover:shadow-lg" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
            <span className="self-start rounded-full px-3 py-[5px] text-[11px] font-semibold tracking-[2px] uppercase" style={{ background: "var(--gold-soft)", color: "var(--gold-deep)" }}>{t("free")}</span>
            <div className="mt-5 font-[var(--serif)] text-[23px] font-medium">{t("quickName")}</div>
            <div className="font-[var(--serif)] text-[48px] font-medium tracking-[-1px] leading-none mt-2.5">{t("quickPrice")}</div>
            <div className="text-sm text-[var(--ink-faint)] mt-1.5 mb-6">{t("quickDesc")}</div>
            <div className="h-px bg-[var(--line-soft)] mb-[22px]" />
            <ul className="flex-1 list-none mb-[30px]">
              {[t("quick1"), t("quick2"), t("quick3"), t("quick4")].map((item, i) => (
                <li key={i} className="flex items-start gap-[11px] text-sm text-[var(--ink-soft)] py-2 font-light"><CheckIcon />{item}</li>
              ))}
            </ul>
            <Link href="/login" className="flex h-11 w-full items-center justify-center gap-2 rounded-lg border text-sm font-medium transition-all" style={{ borderColor: "var(--line)", color: "var(--ink)" }}>{t("quickCta")}</Link>
          </div>
          {/* Paid plan */}
          <div className="flex flex-col rounded-2xl border p-[42px] px-[34px] transition-all hover:-translate-y-[3px] hover:shadow-lg featured-plan" style={{ background: "var(--bg-card)", borderColor: "var(--gold)", boxShadow: "0 0 0 1px var(--gold)" }}>
            <span className="self-start rounded-full px-3 py-[5px] text-[11px] font-semibold tracking-[2px] uppercase" style={{ background: "var(--gold-soft)", color: "var(--gold-deep)" }}>{t("fullBadge")}</span>
            <div className="mt-5 font-[var(--serif)] text-[23px] font-medium">{t("fullName")}</div>
            <div className="font-[var(--serif)] text-[48px] font-medium tracking-[-1px] leading-none mt-2.5">{t("fullPrice")}<small className="font-sans text-[15px] font-normal text-[var(--ink-faint)]">{t("fullPer")}</small></div>
            <div className="text-sm text-[var(--ink-faint)] mt-1.5 mb-6">{t("fullDesc")}</div>
            <div className="h-px bg-[var(--line-soft)] mb-[22px]" />
            <ul className="flex-1 list-none mb-[30px]">
              {[t("full1"), t("full2"), t("full3"), t("full4"), t("full5")].map((item, i) => (
                <li key={i} className="flex items-start gap-[11px] text-sm text-[var(--ink-soft)] py-2 font-light"><CheckIcon />{item}</li>
              ))}
            </ul>
            <Link href="/login" className="flex h-11 w-full items-center justify-center gap-2 rounded-lg text-sm font-medium text-white transition-all" style={{ background: "var(--gold)", boxShadow: "0 1px 2px rgba(168,133,26,.3)" }}>{t("fullCta")}</Link>
          </div>
        </div>
        <p className="mt-[30px] text-center text-xs tracking-[.5px] text-[var(--ink-faint)]">{t("payNote")}</p>
      </div>
    </section>
  );
}
