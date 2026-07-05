"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";

const learnItems = [
  "whatIsBazi",
  "dayMaster",
  "fiveElements",
  "stemsBranches",
  "tenGods",
  "luckCycles",
];

export default function LearnPage() {
  const t = useTranslations("learnPage");
  const headRef = useReveal();
  return (
    <div className="py-20 max-md:py-14" style={{ background: "var(--bg)" }}>
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <div ref={headRef} className="reveal mb-12 text-center">
          <h1 className="mb-4 font-[var(--serif)] text-[44px] font-medium tracking-[-.3px] max-md:text-[32px]" style={{ color: "var(--ink)" }}>{t("title")}</h1>
          <p className="text-[17px] font-light" style={{ color: "var(--ink-soft)" }}>{t("sub")}</p>
        </div>
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {learnItems.map((key) => (
            <div
              key={key}
              className="rounded-xl border p-6 transition-all hover:-translate-y-[2px] hover:shadow-md"
              style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}
            >
              <h3 className="mb-2 font-[var(--serif)] text-lg font-medium" style={{ color: "var(--ink)" }}>{t(`items.${key}.title`)}</h3>
              <p className="text-sm font-light" style={{ color: "var(--ink-soft)" }}>
                {t(`items.${key}.desc`)}
              </p>
            </div>
          ))}
        </div>
        <p className="mt-10 text-center text-xs italic" style={{ color: "var(--ink-faint)" }}>
          {/* TODO: future: fetch from /content/articles API */}
        </p>
      </div>
    </div>
  );
}
