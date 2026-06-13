import { useEffect, useMemo, useState, type ReactNode } from "react";
import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import { createAppTheme, cream, darkBg, type ThemeModeName } from "@/theme";
import { ThemeModeContext, type ThemeModeContextValue } from "@/components/themeMode";

const STORAGE_KEY = "e25-theme";

function initialMode(): ThemeModeName {
  if (typeof window === "undefined") return "light";

  const saved = window.localStorage.getItem(STORAGE_KEY);
  if (saved === "light" || saved === "dark") return saved;

  const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches;

  return prefersDark ? "dark" : "light";
}

export function ThemeModeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<ThemeModeName>(initialMode);
  const theme = useMemo(() => createAppTheme(mode), [mode]);

  useEffect(() => {
    window.localStorage.setItem(STORAGE_KEY, mode);
    document.documentElement.style.backgroundColor = mode === "dark" ? darkBg : cream;
    const meta = document.querySelector('meta[name="theme-color"]');
    meta?.setAttribute("content", mode === "dark" ? darkBg : "#16120d");
  }, [mode]);

  const value = useMemo<ThemeModeContextValue>(
    () => ({ mode, toggle: () => setMode((current) => (current === "dark" ? "light" : "dark")) }),
    [mode],
  );

  return (
    <ThemeModeContext.Provider value={value}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ThemeModeContext.Provider>
  );
}
