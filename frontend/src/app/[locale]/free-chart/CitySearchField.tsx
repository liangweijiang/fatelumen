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
  searchHint: string;
  loadingText: string;
  emptyText: string;
  statusText: string;
  getName: (city: GeoCity) => string;
  onChange: (value: string) => void;
  onSelect: (city: GeoCity) => void;
  onLoadMore: () => void;
  className: string;
};

export function CitySearchField({ value, disabled, loading, loadingMore, options, total, selectedID, placeholder, ariaLabel, searchHint, loadingText, emptyText, statusText, getName, onChange, onSelect, onLoadMore, className }: Props) {
  const [open, setOpen] = useState(false);
  const canShow = open && !disabled;

  return <div className="relative">
    <input
      role="combobox"
      aria-label={ariaLabel}
      aria-autocomplete="list"
      aria-expanded={canShow}
      aria-controls="free-chart-city-results"
      value={value}
      disabled={disabled}
      placeholder={placeholder}
      onFocus={() => setOpen(true)}
      onBlur={() => window.setTimeout(() => setOpen(false), 120)}
      onChange={(event) => { setOpen(true); onChange(event.target.value); }}
      className={`${className} pr-10`}
    />
    <button type="button" disabled={disabled} aria-label={ariaLabel} aria-expanded={canShow} onMouseDown={(event)=>event.preventDefault()} onClick={()=>setOpen((value)=>!value)} className="absolute right-0 top-0 flex h-full w-10 items-center justify-center text-[var(--ink-faint)] disabled:opacity-40"><ChevronDown className={`h-4 w-4 transition ${canShow?"rotate-180":""}`} aria-hidden="true" /></button>
    {canShow && <div id="free-chart-city-results" role="listbox" onScroll={(event)=>{const node=event.currentTarget;if(node.scrollHeight-node.scrollTop-node.clientHeight<48&&options.length<total&&!loadingMore)onLoadMore()}} className="absolute z-30 mt-1 max-h-72 w-full overflow-y-auto border border-[var(--line)] bg-[var(--bg-card)] shadow-xl">
      {loading ? <p role="status" className="px-3 py-3 text-sm text-[var(--ink-faint)]">{loadingText}</p>
        : options.length === 0 ? <p className="px-3 py-3 text-sm text-[var(--ink-faint)]">{value.trim().length < 1 ? searchHint : emptyText}</p>
        : options.map((city) => <button key={city.id} type="button" role="option" aria-selected={selectedID === city.id} onMouseDown={(event) => event.preventDefault()} onClick={() => { onSelect(city); setOpen(false); }} className="block w-full border-b border-[var(--line-soft)] px-3 py-2.5 text-left last:border-b-0 hover:bg-[var(--bg-soft)] focus:bg-[var(--bg-soft)] focus:outline-none"><span className="block text-sm text-[var(--ink)]">{getName(city)}</span><span className="mt-0.5 block text-xs text-[var(--ink-faint)]">{city.admin1_name || city.country_code}</span></button>)}
      {options.length>0&&<div className="sticky bottom-0 flex items-center justify-between border-t border-[var(--line)] bg-[var(--bg-card)] px-3 py-2 text-xs text-[var(--ink-faint)]"><span>{statusText}</span>{options.length<total&&<button type="button" disabled={loadingMore} onMouseDown={(event)=>event.preventDefault()} onClick={onLoadMore} className="text-[var(--gold-deep)] disabled:opacity-50">{loadingMore?loadingText:"+"}</button>}</div>}
    </div>}
  </div>;
}
