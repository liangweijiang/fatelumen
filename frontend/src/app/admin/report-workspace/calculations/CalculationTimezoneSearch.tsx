"use client";

import {useMemo,useState} from "react";
import {ChevronDown} from "lucide-react";
import type {TimeZoneOption} from "@/lib/timezone-options";

type Props={value:string;options:TimeZoneOption[];onChange:(value:string)=>void;className:string};

export function CalculationTimezoneSearch({value,options,onChange,className}:Props){
 const[open,setOpen]=useState(false);
 const visible=useMemo(()=>{const q=value.trim().toLowerCase();return(q?options.filter(x=>x.value.toLowerCase().includes(q)||x.label.toLowerCase().includes(q)):options).slice(0,80)},[options,value]);
 return <div className="relative">
  <input role="combobox" aria-label="出生记录时区" aria-autocomplete="list" aria-expanded={open} aria-controls="calculation-timezone-results" value={value} placeholder="搜索或下拉选择时区" autoComplete="off" onFocus={()=>setOpen(true)} onBlur={()=>window.setTimeout(()=>setOpen(false),120)} onChange={e=>{onChange(e.target.value);setOpen(true)}} className={`${className} pr-10`}/>
  <button type="button" aria-label="展开时区" onMouseDown={e=>e.preventDefault()} onClick={()=>setOpen(v=>!v)} className="absolute right-0 top-0 flex h-full w-10 items-center justify-center text-[var(--ink-faint)]"><ChevronDown className={`h-4 w-4 transition ${open?"rotate-180":""}`}/></button>
  {open&&<div id="calculation-timezone-results" role="listbox" className="absolute left-0 top-full z-[90] mt-1 max-h-72 w-full overflow-y-auto border border-[var(--line)] bg-[var(--bg-card)] shadow-xl">
   {visible.length===0?<p className="px-3 py-3 text-sm text-[var(--ink-faint)]">没有匹配的时区</p>:visible.map(zone=><button key={zone.value} type="button" role="option" aria-selected={zone.value===value} onMouseDown={e=>e.preventDefault()} onClick={()=>{onChange(zone.value);setOpen(false)}} className="block w-full border-b border-[var(--line-soft)] bg-[var(--bg-card)] px-3 py-2.5 text-left hover:bg-[var(--bg-soft)] focus:bg-[var(--bg-soft)] focus:outline-none"><span className="block text-sm">{zone.value}</span><span className="block text-xs text-[var(--ink-faint)]">{zone.label.replace(`${zone.value} · `,"")}</span></button>)}
  </div>}
 </div>
}
