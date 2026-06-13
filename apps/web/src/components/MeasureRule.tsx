import Box from "@mui/material/Box";
import type { SxProps, Theme } from "@mui/material/styles";
import { brass, monoFamily, noirAlpha50 } from "@/theme";

interface MeasureRuleProps {
  /** Small mono label sitting on the rule (e.g. "FIG. 01"). */
  label?: string;
  /** Right-aligned mono caption (e.g. "MADE TO MEASURE"). */
  caption?: string;
  /** Light rule on dark fields, dark rule on light fields. */
  variant?: "light" | "dark";
  sx?: SxProps<Theme>;
}

/**
 * Signature tailor's measuring-tape divider: a baseline with evenly spaced
 * tick marks, optional FIG label and caption. Decorative; aria-hidden.
 */
export function MeasureRule({ label, caption, variant = "dark", sx }: MeasureRuleProps) {
  const lineColor = variant === "light" ? "rgba(232, 222, 203, 0.45)" : noirAlpha50;
  const textColor = variant === "light" ? "rgba(232, 222, 203, 0.8)" : "text.secondary";

  return (
    <Box
      sx={[
        { display: "flex", alignItems: "center", gap: 2, width: "100%" },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {label && (
        <Box
          component="span"
          sx={{
            fontFamily: monoFamily,
            fontSize: "0.6875rem",
            letterSpacing: "0.18em",
            color: brass,
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}
        >
          {label}
        </Box>
      )}

      <Box
        aria-hidden
        sx={{
          flex: 1,
          height: 9,
          borderBottom: "1px solid",
          borderColor: lineColor,
          // Tick marks: short vertical lines hanging from the baseline.
          backgroundImage: `repeating-linear-gradient(to right, ${
            variant === "light" ? "rgba(232,222,203,0.45)" : "rgba(22,18,13,0.5)"
          } 0 1px, transparent 1px 14px)`,
          backgroundPosition: "bottom",
          backgroundSize: "14px 9px",
          backgroundRepeat: "repeat-x",
        }}
      />

      {caption && (
        <Box
          component="span"
          sx={{
            fontFamily: monoFamily,
            fontSize: "0.6875rem",
            letterSpacing: "0.18em",
            textTransform: "uppercase",
            color: textColor,
            whiteSpace: "nowrap",
            flexShrink: 0,
          }}
        >
          {caption}
        </Box>
      )}
    </Box>
  );
}
