"use client";
import { useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Calculator, Languages, Search } from "lucide-react";
import { PromptWorkspace } from "../../prompt-studio/PromptWorkspace";
import { initialRecords } from "../../prompt-studio/data";

const locales = [
  { value: "zh", label: "中文 · zh" },
  { value: "en", label: "English · en" },
  { value: "ja", label: "日本語 · ja" },
  { value: "ko", label: "한국어 · ko" },
];

export default function PromptComposerPage() {
  const params = useSearchParams();
  const initialId = Number(params.get("calculation_id")) || initialRecords[0].id;
  const [selectedId, setSelectedId] = useState(initialId);
  const [query, setQuery] = useState("");
  const [locale, setLocale] = useState("zh");
  const records = useMemo(() => initialRecords.filter((item) => `${item.name}${item.birth}${item.place}`.toLowerCase().includes(query.toLowerCase())), [query]);
  const record = initialRecords.find((item) => item.id === selectedId) ?? initialRecords[0];
  return <section>
    <header className="mb-5"><h2 className="text-2xl">Prompt 编排</h2><p className="mt-1 text-sm" style={{color:"var(--ink-soft)"}}>选择计算档案、版本与目标语言，再配置十个章节使用的前置数据并生成完整调用指令。</p></header>
    <div className="mb-5 border p-4" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(14rem,1fr)_minmax(18rem,2fr)_12rem_11rem]">
        <label className="flex items-center gap-2 border px-3 py-2" style={{borderColor:"var(--line)"}}><Search size={15}/><span className="sr-only">搜索档案</span><input value={query} onChange={e=>setQuery(e.target.value)} className="min-w-0 flex-1 bg-transparent text-sm outline-none" placeholder="搜索计算档案"/></label>
        <label><span className="sr-only">选择计算档案</span><select value={selectedId} onChange={e=>setSelectedId(Number(e.target.value))} className="h-full w-full border px-3 py-2 text-sm" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}>{records.map(item=><option value={item.id} key={item.id}>{item.name} · {item.birth} · {item.place}</option>)}</select></label>
        <label><span className="sr-only">选择计算版本</span><select className="h-full w-full border px-3 py-2 text-sm" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}>{Array.from({length:record.versions},(_,i)=>record.versions-i).map(v=><option key={v}>计算版本 v{v}</option>)}</select></label>
        <label><span className="sr-only">目标语言</span><select value={locale} onChange={e=>setLocale(e.target.value)} className="h-full w-full border px-3 py-2 text-sm" style={{borderColor:"var(--line)",background:"var(--bg-card)"}}>{locales.map(item=><option value={item.value} key={item.value}>{item.label}</option>)}</select></label>
      </div>
      <div className="mt-3 flex flex-wrap items-center gap-x-6 gap-y-2 text-xs" style={{color:"var(--ink-faint)"}}><p className="flex items-center gap-2"><Calculator size={14}/>当前事实哈希 {record.factsHash}；仅消费该版本已保存的确定性计算结果。</p><p className="flex items-center gap-2"><Languages size={14}/><strong style={{color:"var(--ink-soft)"}}>统一生成方式：</strong>同一套章节 Prompt + locale 指令，直接生成目标语言，不经过二次翻译。</p></div>
    </div>
    <PromptWorkspace record={record} locale={locale}/>
  </section>;
}
