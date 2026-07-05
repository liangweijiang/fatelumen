"use client";

import { useEffect, useState } from "react";
import { useLocale } from "next-intl";
import { getToken } from "@/lib/auth-storage";

export function useReadingEntryHref(explicitLocale?: string) {
  const currentLocale = useLocale();
  const locale = explicitLocale || currentLocale;
  const calculatePath = `/${locale}/calculate`;
  const loginPath = `/login?lang=${locale}&next=${encodeURIComponent(calculatePath)}`;
  const [href, setHref] = useState(loginPath);

  useEffect(() => {
    setHref(getToken() ? calculatePath : loginPath);
  }, [calculatePath, loginPath]);

  return href;
}
