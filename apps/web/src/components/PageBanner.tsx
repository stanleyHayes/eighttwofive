import { cloneElement, isValidElement, type ReactElement, type ReactNode } from "react";
import Box from "@mui/material/Box";
import Breadcrumbs from "@mui/material/Breadcrumbs";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import Typography from "@mui/material/Typography";
import { Link as RouterLink } from "react-router";
import { brass, cream, creamMuted, creamText, GRAIN_URL, ink } from "@/theme";

export interface PageBannerCrumb {
  label: string;
  /** Omit on the current (last) page — it renders as plain text. */
  to?: string;
}

export interface PageBannerAction {
  to?: string;
  href?: string;
  label: string;
}

export interface PageBannerProps {
  title: string;
  description?: string;
  /** Trail of crumbs; the last item is the current page and is never a link. */
  breadcrumbs: PageBannerCrumb[];
  /** A MUI icon element, rendered large and faint as a corner watermark. */
  icon: ReactElement;
  /** paper = bone/sand panel on the light canvas; ink = dramatic near-black panel. */
  tone?: "paper" | "ink";
  action?: PageBannerAction;
}

/**
 * A refined, reusable page header banner for the storefront and admin.
 *
 * Squared panel, hairline border, a faint oversized icon watermark in the
 * bottom-right corner, breadcrumbs in monospace, and a display-font title.
 * The "ink" tone is a warm near-black field (with a whisper of film grain)
 * for dramatic headers; "paper" sits quietly on the light canvas.
 */
export function PageBanner({
  title,
  description,
  breadcrumbs,
  icon,
  tone = "paper",
  action,
}: PageBannerProps) {
  const isInk = tone === "ink";

  // On ink we pin the constant dark palette; on paper we lean on the
  // theme's semantic colors so the surface adapts with a future dark mode.
  const surfaceSx = isInk
    ? { bgcolor: ink, color: creamText, borderColor: "rgba(244, 239, 230, 0.16)" }
    : { bgcolor: "background.paper", color: "text.primary", borderColor: "divider" };

  const titleColor = isInk ? cream : "text.primary";
  const descriptionColor = isInk ? creamMuted : "text.secondary";
  const separatorColor = brass;

  const watermark = isValidElement<{ sx?: object }>(icon)
    ? cloneElement(icon, {
        sx: {
          fontSize: { xs: 180, md: 240 },
          ...((icon.props as { sx?: object }).sx ?? {}),
        },
      })
    : icon;

  return (
    <Box
      sx={{
        position: "relative",
        overflow: "hidden",
        border: "1px solid",
        borderRadius: 0,
        py: { xs: 4, md: 6 },
        px: { xs: 3, md: 5 },
        ...surfaceSx,
      }}
    >
      {/* Whisper of film grain on the ink tone */}
      {isInk && (
        <Box
          aria-hidden
          sx={{
            position: "absolute",
            inset: 0,
            backgroundImage: GRAIN_URL,
            opacity: 0.06,
            pointerEvents: "none",
          }}
        />
      )}

      {/* Oversized icon watermark, clipped by the panel's overflow */}
      <Box
        aria-hidden
        sx={{
          position: "absolute",
          right: { xs: -28, md: -20 },
          bottom: { xs: -36, md: -28 },
          color: brass,
          opacity: isInk ? 0.1 : 0.06,
          transform: "rotate(-12deg)",
          pointerEvents: "none",
          lineHeight: 0,
        }}
      >
        {watermark}
      </Box>

      {/* Foreground content sits above the watermark */}
      <Box sx={{ position: "relative", zIndex: 1 }}>
        <Breadcrumbs
          aria-label="Breadcrumb"
          separator={
            <Box component="span" aria-hidden sx={{ color: separatorColor, mx: 0.25 }}>
              /
            </Box>
          }
          sx={{
            mb: { xs: 2, md: 2.5 },
            "& .MuiBreadcrumbs-ol": { alignItems: "center" },
          }}
        >
          {breadcrumbs.map((crumb, index) => {
            const isLast = index === breadcrumbs.length - 1;
            if (crumb.to && !isLast) {
              return (
                <Link
                  key={`${crumb.label}-${index}`}
                  component={RouterLink}
                  to={crumb.to}
                  variant="overline"
                  underline="hover"
                  sx={{
                    color: isInk ? creamMuted : "text.secondary",
                    "&:hover": { color: isInk ? cream : "text.primary" },
                  }}
                >
                  {crumb.label}
                </Link>
              );
            }
            return (
              <Typography
                key={`${crumb.label}-${index}`}
                variant="overline"
                component="span"
                sx={{ color: isInk ? cream : "text.primary" }}
              >
                {crumb.label}
              </Typography>
            );
          })}
        </Breadcrumbs>

        <Box
          sx={{
            display: "flex",
            flexDirection: { xs: "column", sm: "row" },
            alignItems: { sm: "flex-end" },
            justifyContent: "space-between",
            gap: { xs: 2.5, sm: 3 },
          }}
        >
          <Box sx={{ minWidth: 0 }}>
            <Typography variant="h2" component="h1" sx={{ color: titleColor }}>
              {title}
            </Typography>
            {description && (
              <Typography
                variant="subtitle1"
                component="p"
                sx={{ mt: { xs: 1.5, md: 2 }, maxWidth: "60ch", color: descriptionColor }}
              >
                {description}
              </Typography>
            )}
          </Box>

          {action && <PageBannerActionButton action={action} isInk={isInk} />}
        </Box>
      </Box>
    </Box>
  );
}

function PageBannerActionButton({
  action,
  isInk,
}: {
  action: PageBannerAction;
  isInk: boolean;
}): ReactNode {
  const sx = isInk
    ? {
        flexShrink: 0,
        color: cream,
        borderColor: "rgba(244, 239, 230, 0.4)",
        "&:hover": { borderColor: cream, bgcolor: "transparent" },
      }
    : { flexShrink: 0 };

  if (action.to) {
    return (
      <Button variant="outlined" component={RouterLink} to={action.to} sx={sx}>
        {action.label}
      </Button>
    );
  }
  if (action.href) {
    return (
      <Button
        variant="outlined"
        href={action.href}
        target="_blank"
        rel="noopener noreferrer"
        sx={sx}
      >
        {action.label}
      </Button>
    );
  }
  return (
    <Button variant="outlined" sx={sx}>
      {action.label}
    </Button>
  );
}
