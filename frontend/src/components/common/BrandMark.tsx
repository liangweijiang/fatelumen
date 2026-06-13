"use client";

export default function BrandMark({ size = 36 }: { size?: number }) {
  return (
    <span style={{ display:"inline-flex", alignItems:"center", justifyContent:"center", width:size, height:size, borderRadius:size*0.278, background:"var(--ink)", flexShrink:0 }} aria-hidden="true">
      <svg width={size*0.64} height={size*0.64} viewBox="0 0 32 32" fill="none">
        <path d="M21.5 5.5a11 11 0 1 0 0 21 8.5 8.5 0 0 1 0-21z" fill="var(--gold)" />
        <circle cx="22.5" cy="13" r="2.4" fill="var(--gold)" />
        <g stroke="var(--gold)" strokeWidth="1.4" strokeLinecap="round">
          <path d="M22.5 7.2v2M22.5 16.8v2M16.7 13h2M26.3 13h2M18.6 9.1l1.4 1.4M25 9.1l-1.4 1.4" />
        </g>
      </svg>
    </span>
  );
}
