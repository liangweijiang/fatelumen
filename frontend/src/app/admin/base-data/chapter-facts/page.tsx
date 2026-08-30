"use client";
import {useEffect,useState} from "react";
import {LoaderCircle} from "lucide-react";
import {fetchPromptConfigs,fetchPromptRegistry,type PromptRegistryResponse} from "@/lib/admin-api";
import {ChapterFactConfig} from "../../prompt-studio/ChapterFactConfig";

export default function ChapterFactsPage(){
 const [registry,setRegistry]=useState<PromptRegistryResponse|null>(null);const [active,setActive]=useState("");const [configured,setConfigured]=useState<Record<string,string[]>>({});const [error,setError]=useState("");
 useEffect(()=>{Promise.all([fetchPromptRegistry(),fetchPromptConfigs()]).then(([reg,cfg])=>{setRegistry(reg);setConfigured(cfg.configured);setActive(reg.chapters[0]?.key??"")}).catch(()=>setError("章节事实配置读取失败，请稍后重试。"))},[]);
 if(error)return <p role="alert" className="border p-4 text-red-800" style={{borderColor:"var(--line)"}}>{error}</p>;
 if(!registry||!active)return <div className="flex min-h-64 items-center justify-center"><LoaderCircle className="mr-2 animate-spin"/>正在读取章节事实配置</div>;
 const chapter=registry.chapters.find(c=>c.key===active)!;const value=configured[active]??chapter.default_facts;
 return <section><header className="mb-5"><h2 className="text-2xl">章节事实配置</h2><p className="mt-1 text-sm" style={{color:"var(--ink-soft)"}}>十个章节共用这一份配置；Prompt 编排弹窗会读取并修改相同数据。</p></header><div className="grid border lg:grid-cols-[18rem_1fr]" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}><nav aria-label="章节列表" className="border-b lg:border-b-0 lg:border-r" style={{borderColor:"var(--line)"}}>{registry.chapters.map(c=><button key={c.key} type="button" onClick={()=>setActive(c.key)} className="flex w-full items-center gap-3 border-b px-4 py-3 text-left text-sm" style={{borderColor:"var(--line-soft)",background:active===c.key?"var(--gold-soft)":"transparent"}}><span style={{color:"var(--gold-deep)"}}>{String(c.no).padStart(2,"0")}</span><span><strong className="block font-medium">{c.name}</strong><small style={{color:"var(--ink-faint)"}}>{c.key}</small></span></button>)}</nav><div className="p-5"><ChapterFactConfig registry={registry} activeKey={active} value={value} onChange={keys=>setConfigured(s=>({...s,[active]:keys}))}/></div></div></section>
}
