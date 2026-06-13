import type { ReactNode } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Link as RouterLink } from "react-router";
import { amber } from "@/theme";

export interface TableEmptyStateAction {
  to?: string;
  onClick?: () => void;
  label: string;
}

/**
 * Compact, centered empty state sized to live inside a table section — a faint
 * icon watermark over a short headline and helper line, with an optional action.
 * Lighter than the full-page EmptyState panel, so it can repeat across the
 * admin tables without shouting.
 */
export function TableEmptyState({
  icon,
  title,
  body,
  action,
}: {
  icon: ReactNode;
  title: string;
  body?: ReactNode;
  action?: TableEmptyStateAction;
}) {
  return (
    <Stack spacing={1.25} sx={{ alignItems: "center", textAlign: "center", py: { xs: 5, md: 7 }, px: 2 }}>
      <Box
        aria-hidden
        sx={{
          display: "grid",
          placeItems: "center",
          width: 56,
          height: 56,
          mb: 0.5,
          color: amber,
          border: "1px solid",
          borderColor: "divider",
          borderRadius: "50%",
          opacity: 0.9,
          "& svg": { fontSize: 26 },
        }}
      >
        {icon}
      </Box>
      <Typography variant="h6" component="p" sx={{ color: "text.primary" }}>
        {title}
      </Typography>
      {body && (
        <Typography variant="body2" sx={{ color: "text.secondary", maxWidth: "44ch" }}>
          {body}
        </Typography>
      )}
      {action && (
        <Button
          size="small"
          variant="outlined"
          onClick={action.onClick}
          {...(action.to ? { component: RouterLink, to: action.to } : {})}
          sx={{ mt: 1 }}
        >
          {action.label}
        </Button>
      )}
    </Stack>
  );
}
