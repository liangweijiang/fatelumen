"use client";

import { useEffect, useRef } from "react";
import type { BirthProfile } from "@/types/api";
import type { ReportProfileSaveAction } from "@/lib/api/endpoints";

interface ProfileLinkDialogProps {
  open: boolean;
  profiles: BirthProfile[];
  selectedProfileId: number | null;
  busy: boolean;
  labels: {
    title: string;
    description: string;
    onlyReport: string;
    saveNew: string;
    updateSelected: string;
    updateWarning: string;
    cancel: string;
  };
  onChoose: (action: ReportProfileSaveAction) => void;
  onClose: () => void;
}

export default function ProfileLinkDialog({
  open,
  profiles,
  selectedProfileId,
  busy,
  labels,
  onChoose,
  onClose,
}: ProfileLinkDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const selected = profiles.find((profile) => profile.id === selectedProfileId);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby="profile-link-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      className="m-auto w-[min(92vw,520px)] rounded-2xl border p-0 backdrop:bg-black/45"
      style={{ background: "var(--bg-card)", borderColor: "var(--line)", color: "var(--ink)" }}
    >
      <div className="p-6 md:p-8">
        <h2 id="profile-link-title" className="text-xl font-semibold" style={{ fontFamily: "var(--serif-d)" }}>
          {labels.title}
        </h2>
        <p className="mt-2 text-sm leading-6" style={{ color: "var(--ink-soft)" }}>
          {labels.description}
        </p>

        <div className="mt-6 grid gap-3">
          <button
            type="button"
            disabled={busy}
            onClick={() => onChoose("none")}
            className="rounded-xl border px-4 py-3 text-left text-sm font-semibold transition-colors hover:bg-black/5 disabled:opacity-50"
            style={{ borderColor: "var(--line)" }}
          >
            {labels.onlyReport}
          </button>
          {selected && (
            <button
              type="button"
              disabled={busy}
              onClick={() => onChoose("update")}
              className="rounded-xl border px-4 py-3 text-left text-sm font-semibold transition-colors hover:bg-black/5 disabled:opacity-50"
              style={{ borderColor: "var(--gold)" }}
            >
              <span className="block">{labels.updateSelected.replace("{name}", selected.display_name || `#${selected.id}`)}</span>
              <span className="mt-1 block text-xs font-normal" style={{ color: "var(--ink-faint)" }}>
                {labels.updateWarning}
              </span>
            </button>
          )}
          <button
            type="button"
            disabled={busy}
            onClick={() => onChoose("new")}
            className="rounded-xl px-4 py-3 text-left text-sm font-semibold disabled:opacity-50"
            style={{ background: "var(--gold-deep)", color: "var(--bg-card)" }}
          >
            {labels.saveNew}
          </button>
        </div>

        <button
          type="button"
          disabled={busy}
          onClick={onClose}
          className="mt-5 w-full py-2 text-sm disabled:opacity-50"
          style={{ color: "var(--ink-soft)" }}
        >
          {labels.cancel}
        </button>
      </div>
    </dialog>
  );
}
