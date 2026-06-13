import { createContext, useContext } from "react";
import type { ThemeModeName } from "@/theme";

export interface ThemeModeContextValue {
  mode: ThemeModeName;
  toggle: () => void;
}

export const ThemeModeContext = createContext<ThemeModeContextValue | null>(null);

// Safe default when rendered outside a provider (e.g. unit tests) — light
// mode, no-op toggle — so components using the toggle never crash.
export function useThemeMode(): ThemeModeContextValue {
  return useContext(ThemeModeContext) ?? { mode: "light", toggle: () => {} };
}
