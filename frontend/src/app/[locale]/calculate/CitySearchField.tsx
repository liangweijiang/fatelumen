"use client";

import { useState } from "react";
import { ChevronDown } from "lucide-react";
import type { GeoCity } from "@/types/api";

type Props = {
  value: string;
  disabled: boolean;
  loading: boolean;
  loadingMore: boolean;
  options: GeoCity[];
  total: number;
  selectedID?: number;
  placeholder: string;
  ariaLabel: string;
  loadingText: string;
  emptyText: string;
  statusText: string;
  getName: (city: GeoCity) => string;
  onChange: (value: string) => void;
  onSelect: (city: GeoCity) => void;
  onLoadMore: () => void;
};

export function CitySearchField({ value, disabled, loading, loadingMore, options, total, selectedID, placeholder, ariaLabel, loadingText, emptyText, statusText, getName, onChange, onSelect, onLoadMore }: Props) {
  const [open, setOpen] = useState(false);
  const canShow = open && !disabled;
  const inputClass = "w-full rounded-xl border px-4 py-2.5 pr-10 text-[14px] outline-none transition-all focus:ring-2 disabled:cursor-not-allowed disabled:opacity-55";

  return <div className="relative">
    <input role="combobox" aria-label={ariaLabel} aria-autocomplete="list" aria-expanded={canShow} aria-controls="calculate-city-results" value={value} disabled={disabled} placeholder={placeholder} autoComplete="off" onFocus={() => setOpen(true)} onBlur={() => window.setTimeout(() => setOpen(false), 120)} onChange={(event) => { setOpen(true); onChange(event.target.value); }} className={inputClass} style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink)" }} />
    <button type="button" disabled={disabled} aria-label={ariaLabel} aria-expanded={canShow} onMouseDown={(event) => event.preventDefault()} onClick={() => setOpen((value) => !value)} className="absolute right-0 top-0 flex h-full w-10 items-center justify-center disabled:opacity-40" style={{ color: "var(--ink-faint)" }}><ChevronDown className={`h-4 w-4 transition ${canShow ? "rotate-180" : ""}`} aria-hidden="true" /></button>
    {canShow && <div id="calculate-city-results" role="listbox" onScroll={(event) => { const node = event.currentTarget; if (node.scrollHeight - node.scrollTop - node.clientHeight < 48 && options.length < total && !loadingMore) onLoadMore(); }} className="absolute z-30 mt-1 max-h-72 w-full overflow-y-auto rounded-xl border shadow-xl" style={{ background: "var(--bg-card)", borderColor: "var(--line)" }}>
      {loading ? <p role="status" className="px-3 py-3 text-sm" style={{ color: "var(--ink-faint)" }}>{loadingText}</p>
        : options.length === 0 ? <p className="px-3 py-3 text-sm" style={{ color: "var(--ink-faint)" }}>{emptyText}</p>
        : options.map((city) => <button key={city.id} type="button" role="option" aria-selected={selectedID === city.id} onMouseDown={(event) => event.preventDefault()} onClick={() => { onSelect(city); setOpen(false); }} className="block w-full border-b px-3 py-2.5 text-left last:border-b-0 hover:bg-[var(--bg-soft)] focus:bg-[var(--bg-soft)] focus:outline-none" style={{ borderColor: "var(--line-soft)" }}><span className="block text-sm" style={{ color: "var(--ink)" }}>{getName(city)}</span><span className="mt-0.5 block text-xs" style={{ color: "var(--ink-faint)" }}>{city.admin1_name || city.country_code}</span></button>)}
      {options.length > 0 && <div className="sticky bottom-0 flex items-center justify-between border-t px-3 py-2 text-xs" style={{ background: "var(--bg-card)", borderColor: "var(--line)", color: "var(--ink-faint)" }}><span>{statusText}</span>{options.length < total && <button type="button" disabled={loadingMore} onMouseDown={(event) => event.preventDefault()} onClick={onLoadMore} className="text-[var(--gold-deep)] disabled:opacity-50">{loadingMore ? loadingText : "+"}</button>}</div>}
    </div>}
  </div>;
}
