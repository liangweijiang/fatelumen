"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ChartRecord } from "./types";

type Translate = (key: string) => string;

export function ChartResultDialog({ open, onOpenChange, record, t }: { open: boolean; onOpenChange: (open: boolean) => void; record: ChartRecord | null; t: Translate }) {
  if (!record) return null;
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-[#21170d]/55 backdrop-blur-[2px] data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out data-[state=open]:fade-in" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[calc(100%-2rem)] max-w-[920px] -translate-x-1/2 -translate-y-1/2 overflow-y-auto border border-[var(--line)] bg-[var(--bg-card)] shadow-2xl focus:outline-none">
          <div className="flex items-start justify-between border-b border-[var(--line)] px-5 py-5 sm:px-8">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[.2em] text-[var(--gold-deep)]">{t("resultEyebrow")}</p>
              <Dialog.Title className="mt-2 font-[var(--serif)] text-2xl font-semibold text-[var(--ink)]">{t("resultTitle")}</Dialog.Title>
              <Dialog.Description className="mt-2 text-xs text-[var(--ink-faint)]">{record.place} · {record.coordinates}</Dialog.Description>
            </div>
            <Dialog.Close className="grid h-10 w-10 place-items-center border border-[var(--line)] text-[var(--ink-soft)] transition hover:border-[var(--gold-deep)] hover:text-[var(--gold-deep)]" aria-label={t("close")}><X className="h-4 w-4" /></Dialog.Close>
          </div>
          <div className="grid grid-cols-2 border-b border-[var(--line)] sm:grid-cols-4">
            {record.pillars.map((pillar, index) => (
              <article key={pillar.key} className="relative border-[var(--line)] px-3 py-6 text-center odd:border-r sm:border-r sm:last:border-r-0">
                <p className="mb-4 text-xs tracking-[.12em] text-[var(--ink-faint)]">{t(pillar.key)}</p>
                <div className="mx-auto mb-4 flex h-24 w-14 flex-col items-center justify-center border border-[var(--gold)] bg-[var(--bg)] font-[var(--serif)] text-3xl font-semibold leading-none text-[var(--ink)]"><span>{pillar.stem}</span><span className="my-1 h-px w-6 bg-[var(--line)]" /><span>{pillar.branch}</span></div>
                <p className="text-xs text-[var(--ink-soft)]">{pillar.stemElement} · {pillar.branchElement}</p>
                <span className="absolute right-2 top-2 text-[10px] text-[var(--line)]">{String(index + 1).padStart(2, "0")}</span>
              </article>
            ))}
          </div>
          <div className="grid sm:grid-cols-2">
            <dl className="space-y-4 border-b border-[var(--line)] p-5 sm:border-b-0 sm:border-r sm:p-8">
              <div className="flex justify-between gap-4"><dt className="text-xs text-[var(--ink-faint)]">{t("solarDate")}</dt><dd className="text-sm text-[var(--ink)]">{record.solarDate}</dd></div>
              <div className="flex justify-between gap-4"><dt className="text-xs text-[var(--ink-faint)]">{t("lunarDate")}</dt><dd className="text-sm text-[var(--ink)]">{record.lunarDate}</dd></div>
              <div className="flex justify-between gap-4"><dt className="text-xs text-[var(--ink-faint)]">{t("timezone")}</dt><dd className="text-right text-sm text-[var(--ink)]">{record.timezone}</dd></div>
              <div className="flex justify-between gap-4"><dt className="text-xs text-[var(--ink-faint)]">{t("trueSolarTime")}</dt><dd className="text-right text-sm text-[var(--ink)]">{record.trueSolarTime}</dd></div>
              <div className="flex justify-between gap-4"><dt className="text-xs text-[var(--ink-faint)]">{t("dayMaster")}</dt><dd className="font-[var(--serif)] text-lg font-semibold text-[var(--gold-deep)]">{record.dayMaster}</dd></div>
            </dl>
            <div className="p-5 sm:p-8">
              <p className="mb-4 text-xs text-[var(--ink-faint)]">{t("elementBalance")}</p>
              <div className="flex h-2 overflow-hidden" aria-label={t("elementBalance")}><span className="w-[22%] bg-[#64785e]" /><span className="w-[17%] bg-[#a95643]" /><span className="w-[19%] bg-[#a78350]" /><span className="w-[20%] bg-[#8b8980]" /><span className="w-[22%] bg-[#536c7d]" /></div>
              <div className="mt-3 flex justify-between text-xs text-[var(--ink-soft)]"><span>木 22</span><span>火 17</span><span>土 19</span><span>金 20</span><span>水 22</span></div>
            </div>
          </div>
          <div className="border-t border-[var(--line)] px-5 py-4 text-xs leading-5 text-[var(--ink-faint)] sm:px-8">{t("resultNote")}</div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
