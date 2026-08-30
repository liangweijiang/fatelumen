"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { AnnualCalendarYear, fetchAnnualCalendar } from "@/lib/admin-api";

const PAGE_SIZE=20;

export default function AnnualCalendarPage(){
  const now=new Date().getFullYear();
  const [start,setStart]=useState(now);const [end,setEnd]=useState(now+59);const [activeRange,setActiveRange]=useState([now,now+59]);
  const [rows,setRows]=useState<AnnualCalendarYear[]>([]);const [total,setTotal]=useState(0);const [page,setPage]=useState(1);const [version,setVersion]=useState("");const [loading,setLoading]=useState(true);const [error,setError]=useState("");
  const load=useCallback(async()=>{setLoading(true);setError("");try{const data=await fetchAnnualCalendar({start_year:activeRange[0],end_year:activeRange[1],page,page_size:PAGE_SIZE});setRows(data.items??[]);setTotal(data.total??0);setVersion(data.version??"");}catch{setError("干支日历读取失败，请确认后端服务和基础数据状态。");}finally{setLoading(false);}},[activeRange,page]);
  useEffect(()=>{void load();},[load]);
  const submit=(event:FormEvent)=>{event.preventDefault();if(start>end){setError("起始年份不能大于结束年份。");return;}setPage(1);setActiveRange([start,end]);};
  const pages=Math.max(1,Math.ceil(total/PAGE_SIZE));
  return <section>
    <header className="mb-6 flex flex-wrap items-end justify-between gap-4"><div><h2 className="text-2xl">干支日历</h2><p className="mt-1 text-sm" style={{color:"var(--ink-soft)"}}>公共年份数据与个人命盘分离保存；报告只截取计算当年起连续10年，并将结果固化到计算快照。</p></div><div className="border px-4 py-3 text-sm" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}><span className="font-medium">{version||"读取中"}</span><span className="ml-3" style={{color:"var(--gold-deep)"}}>共 {total} 年</span></div></header>
    <form onSubmit={submit} className="mb-4 grid gap-3 border p-4 sm:grid-cols-[1fr_1fr_auto]" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}><label><span className="mb-1 block text-xs">起始年份</span><input aria-label="起始年份" type="number" min={1} value={start} onChange={e=>setStart(Number(e.target.value))} className="w-full border px-3 py-2" style={{borderColor:"var(--line)",background:"var(--bg)"}}/></label><label><span className="mb-1 block text-xs">结束年份</span><input aria-label="结束年份" type="number" min={1} value={end} onChange={e=>setEnd(Number(e.target.value))} className="w-full border px-3 py-2" style={{borderColor:"var(--line)",background:"var(--bg)"}}/></label><button type="submit" className="btn-gold self-end px-5 py-2">查询</button></form>
    {error&&<p role="alert" className="mb-4 border px-4 py-3 text-sm text-red-800" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}>{error}</p>}
    <div className="overflow-x-auto border" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}><table className="w-full min-w-[920px] text-left text-sm"><thead><tr className="border-b" style={{borderColor:"var(--line)"}}>{["年份","干支","天干","地支","天干五行","地支五行","阴阳","生肖","甲子序号","数据版本"].map(x=><th key={x} className="px-4 py-3 font-medium">{x}</th>)}</tr></thead><tbody>{loading?<tr><td colSpan={10} className="px-4 py-12 text-center" aria-live="polite">正在读取干支日历…</td></tr>:rows.length===0?<tr><td colSpan={10} className="px-4 py-12 text-center">当前范围没有基础数据</td></tr>:rows.map(row=><tr key={row.year} className="border-b last:border-0" style={{borderColor:"var(--line-soft)"}}><td className="px-4 py-3 font-medium">{row.year}</td><td className="px-4 py-3 text-base" style={{color:"var(--gold-deep)"}}>{row.ganzhi}</td><td className="px-4 py-3">{row.stem}</td><td className="px-4 py-3">{row.branch}</td><td className="px-4 py-3">{row.stem_element}</td><td className="px-4 py-3">{row.branch_element}</td><td className="px-4 py-3">{row.stem_yin_yang}干 · {row.branch_yin_yang}支</td><td className="px-4 py-3">{row.zodiac}</td><td className="px-4 py-3">{row.cycle_index}/60</td><td className="px-4 py-3 font-mono text-xs">{row.data_version}</td></tr>)}</tbody></table></div>
    <footer className="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm"><span style={{color:"var(--ink-soft)"}}>查询范围 {activeRange[0]}—{activeRange[1]} · 第 {page}/{pages} 页</span><div className="flex gap-2"><button type="button" disabled={page<=1||loading} onClick={()=>setPage(v=>v-1)} className="border px-4 py-2 disabled:opacity-40" style={{borderColor:"var(--line)"}}>上一页</button><button type="button" disabled={page>=pages||loading} onClick={()=>setPage(v=>v+1)} className="border px-4 py-2 disabled:opacity-40" style={{borderColor:"var(--line)"}}>下一页</button></div></footer>
  </section>;
}
