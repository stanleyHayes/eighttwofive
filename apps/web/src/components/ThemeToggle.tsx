import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import DarkModeIcon from "@mui/icons-material/DarkModeOutlined";
import LightModeIcon from "@mui/icons-material/LightModeOutlined";
import { useThemeMode } from "@/components/themeMode";

/**
 * Light/dark mode toggle. The two glyphs are stacked and cross-fade with a
 * rotating, slightly over-shooting settle — the moon swings out as the sun
 * swings in — so the switch reads as one continuous motion. `color` lets it sit
 * on the dark chrome or the light canvas. Gated behind prefers-reduced-motion
 * (the glyphs simply swap with no spin).
 */
export function ThemeToggle({ color = "inherit" }: { color?: string }) {
  const { mode, toggle } = useThemeMode();
  const isDark = mode === "dark";

  const glyphSx = {
    position: "absolute",
    inset: 0,
    fontSize: 20,
    transition:
      "opacity 320ms ease, transform 520ms cubic-bezier(0.34, 1.56, 0.64, 1)",
    "@media (prefers-reduced-motion: reduce)": { transition: "opacity 1ms linear" },
  } as const;

  return (
    <IconButton
      onClick={toggle}
      aria-label={isDark ? "Switch to light theme" : "Switch to dark theme"}
      sx={{ color }}
    >
      <Box
        aria-hidden
        sx={{ position: "relative", width: 20, height: 20, display: "inline-flex" }}
      >
        <LightModeIcon
          sx={{
            ...glyphSx,
            opacity: isDark ? 1 : 0,
            transform: isDark ? "rotate(0) scale(1)" : "rotate(-90deg) scale(0.3)",
          }}
        />
        <DarkModeIcon
          sx={{
            ...glyphSx,
            opacity: isDark ? 0 : 1,
            transform: isDark ? "rotate(90deg) scale(0.3)" : "rotate(0) scale(1)",
          }}
        />
      </Box>
    </IconButton>
  );
}
