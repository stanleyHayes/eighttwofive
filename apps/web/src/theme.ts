import {
  createTheme,
  type PaletteOptions,
  type Theme,
} from "@mui/material/styles";

export type ThemeModeName = "light" | "dark";

// --- "The Pattern Room" — dark atelier ----------------------------------------
// Warm near-black chrome, a single amber/brass accent, a contemporary grotesque
// display, and monospace tailor's-spec labels. The constants below are the
// fixed brand accents + the always-dark chrome; surfaces/body text that adapt
// between light and dark modes use MUI semantic colors (background/text/divider).

export const noir = "#16120d"; // ink — chrome, hero (constant in both modes)
export const ink = noir;
export const inkSoft = "#221b13";
export const cream = "#f4efe6";
export const ivory = cream;
export const creamText = "#e8decb";
export const creamMuted = "#b3a78f";
export const bone = "#ece2d0";
export const sand = "#efe7d6";
export const sandDeep = "#e4d8c2";
export const stone = "#6f675c";
export const brass = "#cf9a3f";
export const amber = "#e0a44a";
export const amberDeep = "#c4863a";
export const clay = "#bf6a3f";
export const clayDeep = "#9c5230";
export const threadRed = "#bf4129";
export const moss = "#3a7a5b";
export const noirMuted = creamMuted;
export const noirText = creamText;
export const noirAlpha50 = "rgba(22, 18, 13, 0.5)";
export const noirAlpha70 = "rgba(22, 18, 13, 0.72)";

// Dark-mode surfaces (warm charcoals, kept lighter than the ink chrome so the
// always-dark nav/footer/hero still read as distinct layers).
export const darkBg = "#1b1611";
export const darkPaper = "#241d15";

export const displayFamily = '"Bricolage Grotesque", "Archivo", sans-serif';
export const monoFamily = '"Spline Sans Mono", ui-monospace, monospace';
const bodyFamily = '"Archivo", system-ui, sans-serif';

function buildPalette(mode: ThemeModeName): PaletteOptions {
  if (mode === "dark") {
    return {
      mode: "dark",
      primary: { main: cream, contrastText: ink },
      secondary: { main: amber, contrastText: ink },
      error: { main: "#d9745f" },
      warning: { main: amber },
      success: { main: "#5fae86" },
      background: { default: darkBg, paper: darkPaper },
      text: { primary: cream, secondary: creamMuted },
      divider: "rgba(232, 222, 203, 0.16)",
    };
  }

  return {
    mode: "light",
    primary: { main: noir, contrastText: cream },
    secondary: { main: amber, contrastText: noir },
    error: { main: threadRed },
    warning: { main: clay },
    success: { main: moss },
    background: { default: cream, paper: "#fffdf8" },
    text: { primary: noir, secondary: stone },
    divider: "rgba(22, 18, 13, 0.14)",
  };
}

export function createAppTheme(mode: ThemeModeName): Theme {
  const isDark = mode === "dark";
  const inputBorder = isDark ? "rgba(232, 222, 203, 0.32)" : noirAlpha50;
  const inputBorderHover = isDark ? "rgba(232, 222, 203, 0.5)" : noirAlpha70;
  const inputFocus = isDark ? cream : noir;

  return createTheme({
    palette: buildPalette(mode),
    shape: { borderRadius: 0 },
    typography: {
      fontFamily: bodyFamily,
      h1: {
        fontFamily: displayFamily,
        fontWeight: 700,
        letterSpacing: "-0.03em",
        lineHeight: 0.95,
        fontSize: "clamp(3.2rem, 10vw, 8rem)",
      },
      h2: {
        fontFamily: displayFamily,
        fontWeight: 700,
        letterSpacing: "-0.025em",
        lineHeight: 0.98,
        fontSize: "clamp(2.2rem, 5.2vw, 3.8rem)",
      },
      h3: {
        fontFamily: displayFamily,
        fontWeight: 600,
        letterSpacing: "-0.02em",
        lineHeight: 1.04,
        fontSize: "clamp(1.7rem, 3.2vw, 2.5rem)",
      },
      h4: {
        fontFamily: displayFamily,
        fontWeight: 600,
        fontSize: "1.6rem",
        lineHeight: 1.1,
        letterSpacing: "-0.015em",
      },
      h5: {
        fontFamily: displayFamily,
        fontWeight: 600,
        fontSize: "1.35rem",
        lineHeight: 1.15,
        letterSpacing: "-0.01em",
      },
      h6: { fontFamily: displayFamily, fontWeight: 600, fontSize: "1.1rem" },
      subtitle1: { fontSize: "1.0625rem", lineHeight: 1.65 },
      body1: { fontSize: "1rem", lineHeight: 1.7 },
      body2: { fontSize: "0.875rem", lineHeight: 1.65 },
      overline: {
        fontFamily: monoFamily,
        fontWeight: 500,
        letterSpacing: "0.18em",
        fontSize: "0.6875rem",
        lineHeight: 1.6,
      },
      button: {
        fontFamily: monoFamily,
        textTransform: "uppercase",
        letterSpacing: "0.16em",
        fontWeight: 500,
        fontSize: "0.75rem",
      },
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          body: {
            textRendering: "optimizeLegibility",
            WebkitFontSmoothing: "antialiased",
            MozOsxFontSmoothing: "grayscale",
          },
          "::selection": {
            backgroundColor: isDark
              ? "rgba(224, 164, 74, 0.32)"
              : "rgba(207, 154, 63, 0.28)",
            color: isDark ? cream : ink,
          },
        },
      },
      MuiButton: {
        defaultProps: { disableElevation: true },
        styleOverrides: {
          root: {
            borderRadius: 0,
            paddingInline: 32,
            paddingBlock: 16,
            boxShadow: "none",
            transition:
              "background-color 200ms ease, color 200ms ease, border-color 200ms ease",
            "&:hover": { boxShadow: "none" },
            "&.Mui-focusVisible": {
              outline: `2px solid ${amber}`,
              outlineOffset: "3px",
            },
          },
          outlined: {
            borderColor: "currentColor",
            "&:hover": { backgroundColor: "transparent" },
          },
        },
      },
      MuiIconButton: {
        styleOverrides: {
          root: {
            borderRadius: 0,
            transition: "background-color 180ms ease, color 180ms ease",
            "&:hover": {
              backgroundColor: isDark
                ? "rgba(232, 222, 203, 0.08)"
                : "rgba(22, 18, 13, 0.06)",
            },
            "&.Mui-focusVisible": {
              outline: `2px solid ${amber}`,
              outlineOffset: "3px",
            },
          },
        },
      },
      MuiLink: {
        styleOverrides: {
          root: {
            textUnderlineOffset: "0.18em",
            "&.Mui-focusVisible, &:focus-visible": {
              outline: `2px solid ${amber}`,
              outlineOffset: "3px",
            },
          },
        },
      },
      MuiTextField: { defaultProps: { variant: "outlined" } },
      MuiOutlinedInput: {
        styleOverrides: {
          root: {
            borderRadius: 0,
            backgroundColor: isDark ? darkPaper : "#fffdf8",
            "& .MuiOutlinedInput-notchedOutline": { borderColor: inputBorder },
            "&:hover .MuiOutlinedInput-notchedOutline": {
              borderColor: inputBorderHover,
            },
            "&.Mui-focused .MuiOutlinedInput-notchedOutline": {
              borderColor: inputFocus,
              borderWidth: 1,
            },
          },
        },
      },
      MuiFormHelperText: {
        styleOverrides: { root: { marginLeft: 0, fontFamily: monoFamily } },
      },
      MuiAlert: {
        defaultProps: { variant: "outlined" },
        styleOverrides: { root: { borderRadius: 0 } },
      },
      MuiPaper: { styleOverrides: { root: { backgroundImage: "none" } } },
      MuiTableCell: {
        styleOverrides: {
          root: {
            borderBottomColor: isDark
              ? "rgba(232, 222, 203, 0.13)"
              : "rgba(22, 18, 13, 0.1)",
          },
          head: {
            color: isDark ? creamMuted : stone,
            fontFamily: monoFamily,
            fontSize: "0.6875rem",
            fontWeight: 500,
            textTransform: "uppercase",
            backgroundColor: isDark
              ? "rgba(232, 222, 203, 0.035)"
              : "rgba(22, 18, 13, 0.035)",
          },
        },
      },
      MuiToggleButton: {
        styleOverrides: {
          root: {
            borderRadius: 0,
            fontFamily: monoFamily,
            textTransform: "uppercase",
            "&.Mui-focusVisible": {
              outline: `2px solid ${amber}`,
              outlineOffset: "2px",
            },
          },
        },
      },
      MuiChip: {
        styleOverrides: {
          root: {
            borderRadius: 0,
            fontFamily: monoFamily,
          },
        },
      },
    },
  });
}

// Default (light) theme for any non-provider importer.
export const theme = createAppTheme("light");

// Faint film-grain texture as a data-URI, for layering over ink fields.
export const GRAIN_URL =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='140' height='140'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.82' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E\")";
