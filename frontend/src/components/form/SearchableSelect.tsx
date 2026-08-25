"use client";

import { useEffect, useId, useState } from "react";

export interface SearchableOption {
  value: string;
  label: string;
}

interface SearchableSelectProps {
  id: string;
  label: string;
  placeholder: string;
  value: string;
  options: SearchableOption[];
  disabled?: boolean;
  noResultsText: string;
  onChange: (value: string) => void;
}

export default function SearchableSelect({
  id,
  label,
  placeholder,
  value,
  options,
  disabled = false,
  noResultsText,
  onChange,
}: SearchableSelectProps) {
  const listId = useId().replace(/:/g, "");
  const selectedLabel = options.find((option) => option.value === value)?.label ?? "";
  const [query, setQuery] = useState(selectedLabel);

  useEffect(() => {
    setQuery(selectedLabel);
  }, [selectedLabel]);

  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-[12px] tracking-[.3px]" style={{ color: "var(--ink-faint)" }}>
        {label}
      </label>
      <input
        id={id}
        type="search"
        list={listId}
        value={query}
        disabled={disabled}
        placeholder={placeholder}
        autoComplete="off"
        onChange={(event) => {
          const nextQuery = event.target.value;
          setQuery(nextQuery);
          const input = nextQuery.trim().toLocaleLowerCase();
          const match = options.find((option) => option.label.toLocaleLowerCase() === input);
          onChange(match?.value ?? "");
        }}
        onBlur={() => {
          if (!value) setQuery("");
        }}
        aria-describedby={`${id}-hint`}
        className="w-full rounded-xl border px-4 py-2.5 text-[14px] outline-none transition-all focus:ring-2 disabled:cursor-not-allowed disabled:opacity-55"
        style={{ background: "var(--bg)", borderColor: "var(--line)", color: "var(--ink)" }}
      />
      <datalist id={listId}>
        {options.map((option) => <option key={option.value} value={option.label} />)}
      </datalist>
      <span id={`${id}-hint`} className="sr-only">{options.length ? placeholder : noResultsText}</span>
    </div>
  );
}
