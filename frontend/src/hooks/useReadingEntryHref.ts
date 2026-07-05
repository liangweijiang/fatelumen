"use client";

import type { MouseEvent } from "react";
import { useEffect, useState } from "react";
import { useLocale } from "next-intl";
import { getToken } from "@/lib/auth-storage";

function buildReadingEntry(locale: string) {
  const calculatePath = `/${locale}/calculate`;
  return {
    calculatePath,
    loginPath: `/login?lang=${locale}&next=${encodeURIComponent(calculatePath)}`,
  };
}

export function useReadingEntryHref(explicitLocale?: string) {
  const currentLocale = useLocale();
  const locale = explicitLocale || currentLocale;
  const { calculatePath, loginPath } = buildReadingEntry(locale);
  const [href, setHref] = useState(loginPath);

  useEffect(() => {
    setHref(getToken() ? calculatePath : loginPath);
  }, [calculatePath, loginPath]);

  return href;
}

export function useReadingEntryLink(explicitLocale?: string) {
  const currentLocale = useLocale();
  const locale = explicitLocale || currentLocale;
  const { calculatePath, loginPath } = buildReadingEntry(locale);
  const [href, setHref] = useState(loginPath);

  useEffect(() => {
    setHref(getToken() ? calculatePath : loginPath);
  }, [calculatePath, loginPath]);

  function onClick(event: MouseEvent<HTMLAnchorElement>) {
    if (!getToken()) return;
    event.preventDefault();
    window.location.href = calculatePath;
  }

  return { href, onClick };
}
