import Chip from "@mui/material/Chip";
import { moss, stone } from "@/theme";
import type { CatalogStatus } from "./api";

export function StatusChip({ status }: { status: CatalogStatus }) {
  const color = status === "live" ? moss : stone;
  return (
    <Chip
      label={status}
      size="small"
      variant="outlined"
      sx={{
        color,
        borderColor: color,
        borderRadius: 0,
        textTransform: "uppercase",
        letterSpacing: "0.12em",
        fontSize: "0.6875rem",
        height: 22,
      }}
    />
  );
}
