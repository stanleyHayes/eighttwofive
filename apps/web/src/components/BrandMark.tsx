import Box from "@mui/material/Box";
import type { SxProps, Theme } from "@mui/material/styles";
import { amber, brass, ink } from "@/theme";

/**
 * The Eight Two Five emblem: three amber tailor's ticks on a measuring
 * baseline, set in an ink square. Matches public/favicon.svg. Decorative —
 * pair with the wordmark for the accessible name.
 */
export function BrandMark({ size = 28, sx }: { size?: number; sx?: SxProps<Theme> }) {
  return (
    <Box
      component="svg"
      viewBox="0 0 64 64"
      aria-hidden
      sx={[{ width: size, height: size, display: "block", flexShrink: 0 }, ...(Array.isArray(sx) ? sx : [sx])]}
    >
      <rect width="64" height="64" fill={ink} />
      <rect x="15" y="18" width="7" height="30" fill={amber} />
      <rect x="28.5" y="26" width="7" height="22" fill={amber} />
      <rect x="42" y="22" width="7" height="26" fill={amber} />
      <rect x="12" y="49" width="40" height="2.4" fill={brass} />
    </Box>
  );
}
