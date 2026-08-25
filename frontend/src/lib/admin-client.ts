import axios from "axios";
import { getAdminToken, removeAdminToken } from "@/lib/admin-auth-storage";
const adminApi=axios.create({baseURL:process.env.NEXT_PUBLIC_API_BASE_URL,timeout:30000});
adminApi.interceptors.request.use(c=>{const t=getAdminToken();if(t)c.headers.Authorization=`Bearer ${t}`;return c});
adminApi.interceptors.response.use(r=>r,e=>{if(typeof window!=="undefined"&&e.response?.status===401){removeAdminToken();window.location.href="/admin/login"}return Promise.reject(e)});
export default adminApi;
