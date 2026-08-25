"use client";
import {useEffect,useState} from "react";
import {useParams} from "next/navigation";
import {getPublicContentDetail,type PublicContent} from "@/lib/public-content";
export default function PublicContentDetail({type}:{type:"knowledge"|"faq"|"case"}){const p=useParams<{locale:string;slug:string}>();const [item,setItem]=useState<PublicContent|null>(null);const [error,setError]=useState(false);useEffect(()=>{getPublicContentDetail(type,p.slug,p.locale||"en").then(setItem).catch(()=>setError(true))},[p.locale,p.slug,type]);if(error)return <main className="mx-auto max-w-3xl px-7 py-20">Content not found.</main>;if(!item)return <main className="mx-auto max-w-3xl px-7 py-20">Loading…</main>;return <main className="mx-auto max-w-3xl px-7 py-20"><h1 className="font-[var(--serif)] text-4xl">{item.title}</h1><p className="mt-4 text-[var(--ink-soft)]">{item.summary}</p><article className="mt-10 whitespace-pre-wrap leading-7">{item.markdown}</article></main>}
