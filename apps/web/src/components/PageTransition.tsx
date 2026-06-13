import type { ReactNode } from "react";
import Box from "@mui/material/Box";
import { useLocation } from "react-router";

/**
 * Plays a refined entrance whenever the route changes — keyed on pathname so the
 * subtree remounts and the animation re-runs. The content fades and rises with a
 * gentle settle (an "editorial" ease-out), while a hairline amber rule sweeps
 * across the top as a signature beat. Everything is gated behind
 * prefers-reduced-motion so it stays still for those who ask for it.
 */
export function PageTransition({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();

  return (
    <Box
      key={pathname}
      sx={{
        position: "relative",
        "@keyframes pageRise": {
          from: { opacity: 0, transform: "translate3d(0, 16px, 0) scale(0.994)" },
          to: { opacity: 1, transform: "none" },
        },
        "@keyframes ruleSweep": {
          from: { transform: "scaleX(0)" },
          to: { transform: "scaleX(1)" },
        },
        "@media (prefers-reduced-motion: no-preference)": {
          willChange: "opacity, transform",
          animation: "pageRise 520ms cubic-bezier(0.16, 1, 0.3, 1) both",
          "&::before": {
            content: '""',
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            height: "2px",
            background: "linear-gradient(90deg, transparent, #e0a44a, transparent)",
            transformOrigin: "left",
            animation: "ruleSweep 620ms cubic-bezier(0.16, 1, 0.3, 1) both",
            pointerEvents: "none",
          },
        },
      }}
    >
      {children}
    </Box>
  );
}
