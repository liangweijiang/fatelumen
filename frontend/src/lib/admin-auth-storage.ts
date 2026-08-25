const KEY = "fatelumen_admin_token";
export const getAdminToken = () => typeof window === "undefined" ? null : localStorage.getItem(KEY);
export const setAdminToken = (token: string) => localStorage.setItem(KEY, token);
export const removeAdminToken = () => localStorage.removeItem(KEY);
