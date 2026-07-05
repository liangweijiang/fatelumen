"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";
import { useReadingEntryLink } from "@/hooks/useReadingEntryHref";
import Link from "next/link";

const caseItems = ["case1", "case2", "case3", "case4"];

export default function CasesPage() {
  const t = useTranslations("casesPage");
  const headRef = useReveal();
  const readingLink = useReadingEntryLink();

  return (
    <div className="py-20 max-md:py-14" style={{ background: "var(--bg)" }}>
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <div ref={headRef} className="reveal mb-12 text-center">
          <h1 className="mb-4 font-[var(--serif)] text-[44px] font-medium tracking-[-.3px] max-md:text-[32px]" style={{ color: "var(--ink)" }}>{t("title")}</h1>
          <p className="text-[17px] font-light" style={{ color: "var(--ink-soft)" }}>{t("sub")}</p>
        </div>
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
          {caseItems.map((key) => (
            <div
              key={key}
              className="rounded-xl border p-6 transition-all hover:-translate-y-[2px] hover:shadow-md"
              style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}
            >
              <div className="flex items-start gap-4">
                <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full text-lg font-semibold text-white" style={{ background: "var(--gold-deep)" }}>{t(`items.${key}.initial`)}</span>
                <div>
                  <h3 className="font-[var(--serif)] text-lg font-medium" style={{ color: "var(--ink)" }}>{t(`items.${key}.name`)}</h3>
                  <p className="mt-2 text-sm font-light italic" style={{ color: "var(--ink-soft)" }}>&ldquo;{t(`items.${key}.quote`)}&rdquo;</p>
                </div>
              </div>
            </div>
          ))}
        </div>
        <div className="mt-12 text-center">
          <Link href={readingLink.href} onClick={readingLink.onClick} className="inline-flex h-[50px] items-center gap-2 rounded-lg px-7 text-[15px] font-medium text-white transition-all" style={{ background: "var(--gold)", boxShadow: "0 1px 2px rgba(168,133,26,.3)" }}>{t("cta")}</Link>
        </div>
        <p className="mt-8 text-center text-xs italic" style={{ color: "var(--ink-faint)" }}>
          {/* TODO: future: fetch from /content/cases API */}
        </p>
      </div>
    </div>
  );
}
