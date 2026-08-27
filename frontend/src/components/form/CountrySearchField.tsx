"use client";

import { useEffect, useId, useMemo, useState } from "react";
import { ChevronDown } from "lucide-react";

export type CountrySearchOption = {
  code: string;
  label: string;
};

type Props = {
  id: string;
  value: string;
  selectedCode?: string;
  options: CountrySearchOption[];
  placeholder: string;
  ariaLabel: string;
  emptyText: string;
  className: string;
  onChange: (value: string) => void;
  onSelect: (option: CountrySearchOption) => void;
};

export function CountrySearchField({ id, value, selectedCode, options, placeholder, ariaLabel, emptyText, className, onChange, onSelect }: Props) {
  const generatedID = useId().replace(/:/g, "");
  const listID = `${id}-${generatedID}-results`;
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const filteredOptions = useMemo(() => {
    const query = value.trim().toLocaleLowerCase();
    if (!query) return options;
    return options.filter((option) => option.label.toLocaleLowerCase().includes(query) || option.code.toLocaleLowerCase().includes(query));
  }, [options, value]);

  useEffect(() => { setActiveIndex(-1); }, [value]);

  function choose(option: CountrySearchOption) {
    onSelect(option);
    setOpen(false);
    setActiveIndex(-1);
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Escape") {
      setOpen(false);
      setActiveIndex(-1);
      return;
    }
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      setOpen(true);
      if (!filteredOptions.length) return;
      setActiveIndex((current) => {
        if (event.key === "ArrowDown") return current >= filteredOptions.length - 1 ? 0 : current + 1;
        return current <= 0 ? filteredOptions.length - 1 : current - 1;
      });
      return;
    }
    if (event.key === "Enter" && open && activeIndex >= 0 && filteredOptions[activeIndex]) {
      event.preventDefault();
      choose(filteredOptions[activeIndex]);
    }
  }

  return <div className="relative">
    <input
      id={id}
      role="combobox"
      aria-label={ariaLabel}
      aria-autocomplete="list"
      aria-expanded={open}
      aria-controls={listID}
      aria-activedescendant={activeIndex >= 0 ? `${listID}-${activeIndex}` : undefined}
      value={value}
      placeholder={placeholder}
      autoComplete="off"
      onFocus={() => setOpen(true)}
      onBlur={() => window.setTimeout(() => { setOpen(false); setActiveIndex(-1); }, 120)}
      onChange={(event) => { setOpen(true); onChange(event.target.value); }}
      onKeyDown={handleKeyDown}
      className={`${className} pr-10`}
      style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink)" }}
    />
    <button type="button" aria-label={ariaLabel} aria-expanded={open} onMouseDown={(event) => event.preventDefault()} onClick={() => setOpen((current) => !current)} className="absolute right-0 top-0 flex h-full w-10 items-center justify-center text-[var(--ink-faint)]">
      <ChevronDown className={`h-4 w-4 transition ${open ? "rotate-180" : ""}`} aria-hidden="true" />
    </button>
    {open && <div id={listID} role="listbox" className="absolute left-0 top-full z-40 mt-1 max-h-72 w-full overflow-y-auto rounded-xl border border-[var(--line)] bg-[var(--bg-card)] shadow-xl">
      {filteredOptions.length === 0
        ? <p className="px-3 py-3 text-sm text-[var(--ink-faint)]">{emptyText}</p>
        : filteredOptions.map((option, index) => <button
            id={`${listID}-${index}`}
            key={option.code}
            type="button"
            role="option"
            aria-selected={selectedCode === option.code}
            onMouseDown={(event) => event.preventDefault()}
            onMouseEnter={() => setActiveIndex(index)}
            onClick={() => choose(option)}
            className={`flex w-full items-center justify-between border-b border-[var(--line-soft)] px-3 py-2.5 text-left last:border-b-0 focus:outline-none ${activeIndex === index || selectedCode === option.code ? "bg-[var(--bg-soft)]" : "hover:bg-[var(--bg-soft)]"}`}
          >
            <span className="text-sm text-[var(--ink)]">{option.label}</span>
            <span className="ml-3 text-xs uppercase tracking-wide text-[var(--ink-faint)]">{option.code}</span>
          </button>)}
    </div>}
  </div>;
}
