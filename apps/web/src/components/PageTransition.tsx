import type { ReactNode } from "react";
import Box from "@mui/material/Box";
import { useLocation } from "react-router";

/**
 * Plays a refined entrance whenever the route changes — keyed on pathname so the
 * subtree remounts and the animation re-runs. A hairline amber rule sweeps the
 * top edge and fades, leading the content, which rises and resolves from a
 * whisper of blur (a "developing print" beat that fits the made-to-order story).
 * Everything is gated behind prefers-reduced-motion so it stays still for those
 * who ask for it.
 */
export function PageTransition({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();

  return (
    <Box
      key={pathname}
      sx={{
        position: "relative",
        "@keyframes e25PageRise": {
          "0%": {
            opacity: 0,
            transform: "translate3d(0, 20px, 0) scale(0.992)",
            filter: "blur(5px)",
          },
          "55%": { filter: "blur(0)" },
          "100%": {
            opacity: 1,
            transform: "none",
            filter: "blur(0)",
          },
        },
        "@keyframes e25RuleSweep": {
          "0%": { transform: "scaleX(0)", opacity: 1 },
          "62%": { transform: "scaleX(1)", opacity: 1 },
          "100%": { transform: "scaleX(1)", opacity: 0 },
        },
        "@media (prefers-reduced-motion: no-preference)": {
          willChange: "opacity, transform, filter",
          animation: "e25PageRise 600ms cubic-bezier(0.16, 1, 0.3, 1) 60ms both",
          "&::before": {
            content: '""',
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            height: "2px",
            background:
              "linear-gradient(90deg, transparent, #e0a44a 35%, #e0a44a 65%, transparent)",
            transformOrigin: "left",
            animation: "e25RuleSweep 760ms cubic-bezier(0.16, 1, 0.3, 1) both",
            pointerEvents: "none",
            zIndex: 5,
          },
        },
      }}
    >
      {children}
    </Box>
  );
}
