import type { ReactNode } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import type { SxProps, Theme } from "@mui/material/styles";
import { Link as RouterLink } from "react-router";
import { MeasureRule } from "@/components/MeasureRule";
import { useThemeMode } from "@/components/themeMode";
import { amber, brass, clay, cream, creamMuted, creamText, GRAIN_URL, ink } from "@/theme";

export interface EmptyStateAction {
  /** Internal route — rendered with react-router's Link. */
  to?: string;
  /** External / protocol link — rendered as an anchor. */
  href?: string;
  label: string;
}

export interface EmptyStateProps {
  /** Mono eyebrow, e.g. "Nothing here yet". Rendered small-caps. */
  label?: string;
  /** Headline in the display face. */
  title: string;
  /** Secondary supporting copy. */
  body?: ReactNode;
  /** Optional call to action, rendered as a Button. */
  action?: EmptyStateAction;
  /**
   * "paper" sits on the warm light canvas with a hairline border and a
   * MeasureRule accent. "ink" is a dramatic warm near-black panel with cream
   * text, for the rarer "nothing here at all" moments.
   */
  tone?: "paper" | "ink";
  sx?: SxProps<Theme>;
}

function actionProps(action: EmptyStateAction) {
  if (action.to) return { component: RouterLink, to: action.to } as const;
  if (action.href) return { href: action.href } as const;
  return {};
}

/**
 * Reusable, editorial empty / placeholder panel in the atelier aesthetic:
 * squared corners, hairline border, mono eyebrow, a display headline and a
 * signature MeasureRule. Two tones — light paper and dramatic ink.
 */
export function EmptyState({
  label,
  title,
  body,
  action,
  tone = "paper",
  sx,
}: EmptyStateProps) {
  const isInk = tone === "ink";
  const { mode } = useThemeMode();
  // On the paper tone the rule must read against whichever canvas is active.
  const ruleVariant = isInk || mode === "dark" ? "light" : "dark";

  return (
    <Box
      sx={[
        {
          position: "relative",
          maxWidth: 640,
          px: { xs: 4, md: 6 },
          py: { xs: 5, md: 7 },
          overflow: "hidden",
          bgcolor: isInk ? ink : "background.paper",
          color: isInk ? creamText : "text.primary",
          border: "1px solid",
          borderColor: isInk ? "rgba(232, 222, 203, 0.16)" : "divider",
          ...(isInk && {
            backgroundImage: `${GRAIN_URL}`,
            backgroundBlendMode: "overlay",
          }),
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Stack spacing={2.5} sx={{ position: "relative" }}>
        <MeasureRule variant={ruleVariant} label="FIG." caption="Made to measure" />

        <Box>
          {label && (
            <Typography
              variant="overline"
              component="p"
              sx={{ color: isInk ? brass : clay, mb: 1.5 }}
            >
              {label}
            </Typography>
          )}
          <Typography
            variant="h3"
            component="p"
            sx={{ color: isInk ? cream : "text.primary" }}
          >
            {title}
          </Typography>
          {body && (
            <Typography
              variant="body2"
              sx={{
                mt: 2,
                maxWidth: "52ch",
                color: isInk ? creamMuted : "text.secondary",
              }}
            >
              {body}
            </Typography>
          )}
        </Box>

        {action && (
          <Box>
            <Button
              variant={isInk ? "outlined" : "contained"}
              {...actionProps(action)}
              sx={
                isInk
                  ? { color: cream, borderColor: "rgba(232, 222, 203, 0.5)" }
                  : undefined
              }
            >
              {action.label}
            </Button>
          </Box>
        )}
      </Stack>
    </Box>
  );
}

export interface ErrorStateProps {
  /** The error copy to display. */
  message: string;
  /** Optional retry handler — renders a "Try again" button. */
  onRetry?: () => void;
  /** Optional override for the eyebrow label. */
  label?: string;
  sx?: SxProps<Theme>;
}

/**
 * Sibling of EmptyState for failure branches: same squared, bordered panel,
 * but with a clay/amber accent and an optional retry Button. Light tone only —
 * errors should read as a calm note, not an alarm.
 */
export function ErrorState({ message, onRetry, label, sx }: ErrorStateProps) {
  const { mode } = useThemeMode();

  return (
    <Box
      role="alert"
      sx={[
        {
          maxWidth: 640,
          px: { xs: 4, md: 6 },
          py: { xs: 5, md: 7 },
          bgcolor: "background.paper",
          border: "1px solid",
          borderColor: clay,
          borderLeftWidth: 3,
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Stack spacing={2.5}>
        <MeasureRule variant={mode === "dark" ? "light" : "dark"} label="ERR." caption="Out of true" />
        <Box>
          <Typography variant="overline" component="p" sx={{ color: clay, mb: 1.5 }}>
            {label ?? "Something slipped"}
          </Typography>
          <Typography variant="h4" component="p">
            We couldn&apos;t load this.
          </Typography>
          <Typography variant="body2" sx={{ mt: 2, maxWidth: "52ch", color: "text.secondary" }}>
            {message}
          </Typography>
        </Box>
        {onRetry && (
          <Box>
            <Button
              variant="outlined"
              onClick={onRetry}
              sx={{ borderColor: amber, color: "text.primary", "&:hover": { borderColor: amber } }}
            >
              Try again
            </Button>
          </Box>
        )}
      </Stack>
    </Box>
  );
}
