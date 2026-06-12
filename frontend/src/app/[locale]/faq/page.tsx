"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useReveal } from "@/components/landing/useReveal";

export default function FaqPage() {
  const t = useTranslations("faqPage");
  const [search, setSearch] = useState("");
  const [activeCat, setActiveCat] = useState("basics");
  const [openMap, setOpenMap] = useState<Record<string, boolean>>({});
  const headRef = useReveal();

  const cats = [
    { key: "basics", label: t("categories.basics") },
    { key: "accuracy", label: t("categories.accuracy") },
    { key: "life", label: t("categories.life") },
    { key: "cycles", label: t("categories.cycles") },
    { key: "relationships", label: t("categories.relationships") },
    { key: "philosophy", label: t("categories.philosophy") },
    { key: "comparison", label: t("categories.comparison") },
    { key: "practice", label: t("categories.practice") },
    { key: "depth", label: t("categories.depth") },
    { key: "service", label: t("categories.service") },
  ];

  const toggle = (k: string) => setOpenMap((p) => ({ ...p, [k]: !p[k] }));

  return (
    <div className="py-20 max-md:py-14" style={{ background: "var(--bg)" }}>
      <div className="mx-auto max-w-[var(--maxw)] px-7">
        <div ref={headRef} className="reveal mb-12 text-center">
          <h1 className="mb-4 font-[var(--serif)] text-[44px] font-medium tracking-[-.3px] max-md:text-[32px]" style={{ color: "var(--ink)" }}>{t("title")}</h1>
          <p className="text-[17px] font-light" style={{ color: "var(--ink-soft)" }}>{t("sub")}</p>
        </div>

        <input
          type="text"
          placeholder={t("search")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="mb-8 w-full rounded-lg border px-4 py-3 text-sm"
          style={{ background: "var(--bg-card)", borderColor: "var(--line)", color: "var(--ink)" }}
        />

        <div className="flex gap-8 max-md:flex-col">
          <nav className="w-48 shrink-0 max-md:w-full">
            <ul className="list-none space-y-0.5" style={{ borderColor: "var(--line)" }}>
              {cats.map((c) => (
                <li key={c.key}>
                  <button
                    onClick={() => setActiveCat(c.key)}
                    className="w-full rounded px-3 py-2 text-left text-sm transition-colors"
                    style={{
                      background: activeCat === c.key ? "var(--gold-soft)" : "transparent",
                      color: activeCat === c.key ? "var(--gold-deep)" : "var(--ink-soft)",
                      fontWeight: activeCat === c.key ? 600 : 400,
                    }}
                  >
                    {c.label}
                  </button>
                </li>
              ))}
            </ul>
          </nav>

          <div className="flex-1 border-t" style={{ borderColor: "var(--line)" }}>
            {[1, 2, 3, 4, 5].map((n) => {
              const key = `${activeCat}-${n}`;
              return (
                <div key={key} className="border-b" style={{ borderColor: "var(--line)" }}>
                  <button
                    onClick={() => toggle(key)}
                    className="flex w-full items-center justify-between gap-4 px-0 py-5 text-left font-[var(--serif)] text-[17px] font-medium bg-transparent border-none cursor-pointer"
                    style={{ color: "var(--ink)" }}
                  >
                    {activeCat === "basics" ? [`What is Bazi?`, `What are the Four Pillars?`, `What is a Day Master?`, `What are the Five Elements?`, `What are Heavenly Stems and Earthly Branches?`][n - 1] : `${cats.find((c) => c.key === activeCat)?.label} — Question ${n}`}
                    <span className="shrink-0 font-sans text-[var(--ink-faint)]" style={{ transform: openMap[key] ? "rotate(45deg)" : "" }}>＋</span>
                  </button>
                  <div className="overflow-hidden transition-[max-height] duration-[280ms]" style={{ maxHeight: openMap[key] ? "200px" : "0" }}>
                    <p className="max-w-[92%] pb-5 text-[14px] font-light" style={{ color: "var(--ink-soft)" }}>
                      {activeCat === "basics" ? `A detailed explanation about ${{ 0: "Bazi fundamentals", 1: "the Four Pillars structure", 2: "Day Master concepts", 3: "Five Elements theory", 4: "Stems and Branches" }[n - 1]}.` : "This question explores the topic in plain, accessible language. A comprehensive answer will be added here."}
                    </p>
                  </div>
                </div>
              );
            })}
            <p className="mt-6 text-xs italic" style={{ color: "var(--ink-faint)" }}>
              {/* TODO: future: fetch FAQ from /content/faq API */}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
