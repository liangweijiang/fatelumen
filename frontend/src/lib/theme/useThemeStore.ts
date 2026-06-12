import { create } from "zustand";
import { persist } from "zustand/middleware";

interface ThemeState {
  theme: string;
  setTheme: (id: string) => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: "kraft",
      setTheme: (id: string) => {
        if (typeof document !== "undefined") {
          document.documentElement.setAttribute("data-theme", id);
        }
        set({ theme: id });
      },
    }),
    { name: "fatelumen-theme" }
  )
);
