import axios from "axios";
export type PublicContent={id:number;slug:string;title:string;summary:string;tags:string[];markdown:string;locale:string};
const base=process.env.NEXT_PUBLIC_API_BASE_URL;
export async function getPublicContent(type:string,locale:string){const r=await axios.get(`${base}/public/content/${type}`,{params:{locale}});return (r.data.data??r.data) as PublicContent[]}
export async function getPublicContentDetail(type:string,slug:string,locale:string){const r=await axios.get(`${base}/public/content/${type}/${encodeURIComponent(slug)}`,{params:{locale}});return (r.data.data??r.data) as PublicContent}
export async function getPublicPricing(locale:string){const r=await axios.get(`${base}/public/pricing`,{params:{locale,public:1}});return r.data.data??r.data}
