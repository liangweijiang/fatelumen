"use client";

import { useTranslations } from "next-intl";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getPublicPricing } from "@/lib/public-content";
import { useReveal } from "@/hooks/useReveal";
import Link from "next/link";
import { useReadingEntryLink } from "@/hooks/useReadingEntryHref";

export function Pricing() {
  const t = useTranslations("pricing");
  const headRef = useReveal();
  const plansRef = useReveal();
  const readingLink = useReadingEntryLink();
  const params = useParams<{ locale: string }>();
  const [managedPlans, setManagedPlans] = useState<Array<{ id: number; title: string; summary: string; amount_cents: number; original_amount_cents: number; report_count: number; credits_granted: number; currency: string; benefits: string[] }>>([]);
  useEffect(() => { getPublicPricing(params.locale || "en").then(setManagedPlans).catch(() => setManagedPlans([])); }, [params.locale]);

  const CheckIcon = () => (
    <svg className="shrink-0 mt-[3px] text-[var(--gold-deep)]" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2"><path d="M20 6 9 17l-5-5" /></svg>
  );

  return (
    <section id="pricing" className="px-0 py-[120px] max-md:py-[80px]">
      <div className="mx-auto max-w-[var(--maxw)] px-5 md:px-10">
        <div ref={headRef} className="reveal mx-auto mb-[72px] max-w-[620px] text-center">
          <span className="mb-[18px] block text-xs font-medium tracking-[3px] uppercase text-[var(--ink-faint)]">{t("eyebrow")}</span>
          <h2 className="mb-5 font-[var(--serif)] text-[44px] font-medium leading-[1.16] tracking-[-.3px] max-md:text-[32px]">{t("title")}</h2>
          <p className="text-[17px] leading-relaxed font-light text-[var(--ink-soft)]">{t("sub")}</p>
        </div>
        {managedPlans.length > 0 ? <div ref={plansRef} className="reveal mx-auto grid max-w-[800px] grid-cols-2 gap-6 max-md:grid-cols-1">
          {managedPlans.map((plan) => <div key={plan.id} className="flex flex-col rounded-xl border p-[36px]" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
            <div className="font-[var(--serif)] text-[23px] font-medium">{plan.title}</div>{plan.original_amount_cents > plan.amount_cents && <div className="mt-2 text-sm line-through text-[var(--ink-faint)]">{(plan.original_amount_cents / 100).toFixed(2)} {plan.currency.toUpperCase()}</div>}<div className="gold-embossed mt-3 font-[var(--serif)] text-[48px] font-medium">{(plan.amount_cents / 100).toFixed(2)} <small className="font-sans text-sm">{plan.currency.toUpperCase()}</small></div><p className="mt-2 mb-6 text-sm text-[var(--ink-faint)]">{plan.summary}</p><ul className="mb-7 flex-1">{plan.report_count > 0 && <li className="flex gap-2 py-1 text-sm"><CheckIcon />{plan.report_count} report{plan.report_count > 1 ? "s" : ""}</li>}{plan.credits_granted > 0 && <li className="flex gap-2 py-1 text-sm"><CheckIcon />{plan.credits_granted} credits</li>}{(plan.benefits || []).map((item) => <li key={item} className="flex gap-2 py-1 text-sm"><CheckIcon />{item}</li>)}</ul><Link href={readingLink.href} onClick={readingLink.onClick} className="btn-ghost flex h-11 items-center justify-center rounded-lg">{t("quickCta")}</Link>
          </div>)}
        </div> : <div ref={plansRef} className="reveal mx-auto grid max-w-[800px] grid-cols-2 gap-6 max-md:grid-cols-1">
          <div className="flex flex-col rounded-xl border p-[36px] transition-colors" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
            <span className="self-start rounded-full px-3 py-[5px] text-[11px] font-semibold tracking-[2px] uppercase" style={{ background: "var(--gold-soft)", color: "var(--gold-deep)" }}>{t("free")}</span>
            <div className="mt-5 font-[var(--serif)] text-[23px] font-medium">{t("quickName")}</div>
            <div className="gold-embossed mt-3 font-[var(--serif)] text-[48px] font-medium leading-none tracking-[-1px]">{t("quickPrice")}</div>
            <div className="mt-2 mb-6 text-sm text-[var(--ink-faint)]">{t("quickDesc")}</div>
            <div className="mb-[22px] h-px bg-[var(--line-soft)]" />
            <ul className="mb-[30px] flex-1 list-none">
              {[t("quick1"), t("quick2"), t("quick3"), t("quick4")].map((item, i) => (
                <li key={i} className="flex items-start gap-[11px] py-2 text-sm font-light text-[var(--ink-soft)]"><CheckIcon />{item}</li>
              ))}
            </ul>
            <Link href={readingLink.href} onClick={readingLink.onClick} className="btn-ghost flex h-11 w-full items-center justify-center gap-2 rounded-lg border text-sm font-medium transition-all">{t("quickCta")}</Link>
          </div>
          <div className="flex flex-col rounded-xl border p-[36px] transition-colors" style={{ background: "var(--bg-card)", borderColor: "var(--gold)" }}>
            <span className="self-start rounded-full px-3 py-[5px] text-[11px] font-semibold tracking-[2px] uppercase" style={{ background: "var(--gold-soft)", color: "var(--gold-deep)" }}>{t("fullBadge")}</span>
            <div className="mt-5 font-[var(--serif)] text-[23px] font-medium">{t("fullName")}</div>
            <div className="gold-embossed mt-3 font-[var(--serif)] text-[48px] font-medium leading-none tracking-[-1px]">{t("fullPrice")}<small className="font-sans text-[15px] font-normal text-[var(--ink-faint)]">{t("fullPer")}</small></div>
            <div className="mt-2 mb-6 text-sm text-[var(--ink-faint)]">{t("fullDesc")}</div>
            <div className="mb-[22px] h-px bg-[var(--line-soft)]" />
            <ul className="mb-[30px] flex-1 list-none">
              {[t("full1"), t("full2"), t("full3"), t("full4"), t("full5")].map((item, i) => (
                <li key={i} className="flex items-start gap-[11px] py-2 text-sm font-light text-[var(--ink-soft)]"><CheckIcon />{item}</li>
              ))}
            </ul>
            <Link href={readingLink.href} onClick={readingLink.onClick} className="btn-gold flex h-11 w-full items-center justify-center gap-2 rounded-lg text-sm font-medium transition-all">{t("fullCta")}</Link>
          </div>
        </div>}
        <p className="mt-[34px] text-center text-xs tracking-[.5px] text-[var(--ink-faint)]">{t("payNote")}</p>
      </div>
    </section>
  );
}
