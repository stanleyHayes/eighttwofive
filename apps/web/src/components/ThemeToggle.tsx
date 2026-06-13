import IconButton from "@mui/material/IconButton";
import DarkModeIcon from "@mui/icons-material/DarkModeOutlined";
import LightModeIcon from "@mui/icons-material/LightModeOutlined";
import { useThemeMode } from "@/components/themeMode";

/** Light/dark mode toggle. `color` lets it sit on dark chrome or light canvas. */
export function ThemeToggle({ color = "inherit" }: { color?: string }) {
  const { mode, toggle } = useThemeMode();
  const isDark = mode === "dark";

  return (
    <IconButton
      onClick={toggle}
      aria-label={isDark ? "Switch to light theme" : "Switch to dark theme"}
      sx={{ color }}
    >
      {isDark ? <LightModeIcon sx={{ fontSize: 20 }} /> : <DarkModeIcon sx={{ fontSize: 20 }} />}
    </IconButton>
  );
}
