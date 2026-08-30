"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";

export type SecondaryNavItem={href:string;label:string;description:string};
export function SecondaryNav({title,items}:{title:string;items:SecondaryNavItem[]}){const pathname=usePathname();return <div className="mb-6 border-b" style={{borderColor:"var(--line)"}}><div className="mb-3 flex items-baseline justify-between gap-4"><h1 className="text-2xl font-medium">{title}</h1><span className="text-xs" style={{color:"var(--ink-faint)"}}>原型模式 · 暂未接入后端</span></div><nav aria-label={`${title}二级导航`} className="flex gap-1 overflow-x-auto">{items.map(item=>{const active=pathname===item.href||pathname.startsWith(`${item.href}/`);return <Link key={item.href} href={item.href} aria-current={active?"page":undefined} className="min-w-36 border-b-2 px-4 py-3" style={{borderColor:active?"var(--gold-deep)":"transparent",background:active?"var(--gold-soft)":"transparent"}}><span className="block text-sm font-medium" style={{color:active?"var(--gold-deep)":"var(--ink)"}}>{item.label}</span><span className="block text-[11px]" style={{color:"var(--ink-faint)"}}>{item.description}</span></Link>})}</nav></div>}
