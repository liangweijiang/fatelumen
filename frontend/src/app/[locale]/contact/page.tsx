"use client";

import { useTranslations } from "next-intl";
import { useReveal } from "@/hooks/useReveal";

export default function ContactPage() {
  const t = useTranslations("contactPage");
  const head = useReveal();
  return (
    <div className="py-20 max-md:py-14" style={{ background: "var(--bg)" }}>
      <div className="mx-auto max-w-[640px] px-7">
        <div ref={head} className="reveal mb-10">
          <h1 className="mb-3 font-[var(--serif)] text-[44px] font-medium tracking-[-.3px] max-md:text-[32px]" style={{ color: "var(--ink)" }}>{t("title")}</h1>
          <p className="text-[17px] font-light" style={{ color: "var(--ink-soft)" }}>{t("sub")}</p>
        </div>
        <div className="rounded-xl border p-7" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
          <p className="mb-1 text-xs uppercase tracking-[2px]" style={{ color: "var(--ink-faint)" }}>{t("emailLabel")}</p>
          <a href={`mailto:${t("email")}`} className="mb-6 inline-block text-[18px] font-medium" style={{ color: "var(--gold-deep)" }}>{t("email")}</a>
          <p className="mb-1 mt-4 text-xs uppercase tracking-[2px]" style={{ color: "var(--ink-faint)" }}>{t("hoursLabel")}</p>
          <p className="text-[16px] font-light" style={{ color: "var(--ink-soft)" }}>{t("hours")}</p>
        </div>
        <p className="mt-6 text-[14px] font-light italic" style={{ color: "var(--ink-faint)" }}>{t("note")}</p>
      </div>
    </div>
  );
}
