import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { ThemeProvider } from "@mui/material/styles";
import CssBaseline from "@mui/material/CssBaseline";
import GlobalStyles from "@mui/material/GlobalStyles";
import { createAppTheme, cream, darkBg, type ThemeModeName } from "@/theme";
import { ThemeModeContext, type ThemeModeContextValue } from "@/components/themeMode";

const STORAGE_KEY = "e25-theme";
const THEME_TRANSITION_MS = 480;

// While `.theme-transition` is on <html> (only for the toggle window), surface
// colours ease between light and dark instead of snapping. Scoped to colour
// properties so it never touches hover/transform interactions, and disabled
// under reduced-motion.
const themeCrossfade = (
  <GlobalStyles
    styles={{
      "@media (prefers-reduced-motion: no-preference)": {
        ".theme-transition, .theme-transition *, .theme-transition *::before, .theme-transition *::after":
          {
            transitionProperty:
              "background-color, color, border-color, fill, stroke",
            transitionDuration: `${THEME_TRANSITION_MS}ms`,
            transitionTimingFunction: "ease",
          },
      },
    }}
  />
);

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
  const timerRef = useRef(0);

  useEffect(() => {
    window.localStorage.setItem(STORAGE_KEY, mode);
    document.documentElement.style.backgroundColor = mode === "dark" ? darkBg : cream;
    const meta = document.querySelector('meta[name="theme-color"]');
    meta?.setAttribute("content", mode === "dark" ? darkBg : "#16120d");
  }, [mode]);

  useEffect(() => () => window.clearTimeout(timerRef.current), []);

  const toggle = useCallback(() => {
    // Arm the colour crossfade just before the theme flips, then clear it so
    // the transition rule isn't left applying to every later interaction.
    const root = document.documentElement;
    root.classList.add("theme-transition");
    window.clearTimeout(timerRef.current);
    timerRef.current = window.setTimeout(
      () => root.classList.remove("theme-transition"),
      THEME_TRANSITION_MS,
    );
    setMode((current) => (current === "dark" ? "light" : "dark"));
  }, []);

  const value = useMemo<ThemeModeContextValue>(() => ({ mode, toggle }), [mode, toggle]);

  return (
    <ThemeModeContext.Provider value={value}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {themeCrossfade}
        {children}
      </ThemeProvider>
    </ThemeModeContext.Provider>
  );
}
