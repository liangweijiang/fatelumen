export interface ThemeDef {
  id: string;
  nameKey: string;
  available: boolean;
}

export const THEMES: ThemeDef[] = [
  { id: "kraft", nameKey: "theme.kraft", available: true },
  { id: "theme2", nameKey: "theme.slot2", available: false },
  { id: "theme3", nameKey: "theme.slot3", available: false },
  { id: "theme4", nameKey: "theme.slot4", available: false },
  { id: "theme5", nameKey: "theme.slot5", available: false },
  { id: "theme6", nameKey: "theme.slot6", available: false },
];
