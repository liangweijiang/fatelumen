"use client";

import { useEffect } from "react";
import { useThemeStore, rehydrateTheme } from "@/lib/theme/useThemeStore";

export default function ThemeProvider({ children }: { children: React.ReactNode }) {
  const theme = useThemeStore((s) => s.theme);

  useEffect(() => {
    rehydrateTheme();
  }, []);

  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  return <>{children}</>;
}
