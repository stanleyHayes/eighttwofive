import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import { Link as RouterLink } from "react-router";
import type { Design } from "@/features/catalog/api";
import { formatPesewas } from "@/features/catalog/money";
import {
  amber,
  brass,
  cream,
  GRAIN_URL,
  ink,
  monoFamily,
  sandDeep,
} from "@/theme";
import { CARD_TRANSFORM, minBandPesewas, photoUrl, sortedPhotos } from "./api";

/**
 * Intentional stand-in when a design has no photos (or Cloudinary is
 * unconfigured): a warm ink field with a mono micro-label and the name set in
 * the display face, so the absence reads as deliberate, not broken.
 */
export function PhotoPlaceholder({ name }: { name: string }) {
  return (
    <Box
      sx={{
        position: "relative",
        aspectRatio: "600 / 780",
        bgcolor: ink,
        backgroundImage: GRAIN_URL,
        backgroundBlendMode: "overlay",
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        p: 2.5,
        overflow: "hidden",
      }}
    >
      <Typography
        variant="overline"
        component="span"
        sx={{ color: brass, position: "relative" }}
      >
        No image yet
      </Typography>
      <Typography
        variant="h5"
        component="span"
        sx={{ color: cream, position: "relative" }}
      >
        {name}
      </Typography>
    </Box>
  );
}

export function DesignCard({
  design,
  cloudName,
}: {
  design: Design;
  cloudName: string;
}) {
  const photos = sortedPhotos(design);
  const cover = photos[0];
  const min = minBandPesewas(design);

  return (
    <Link
      component={RouterLink}
      to={`/designs/${design.slug}`}
      underline="none"
      sx={{
        color: "text.primary",
        display: "block",
        height: "100%",
        "&:hover .e25-view, &:focus-visible .e25-view": {
          opacity: 1,
          transform: "none",
        },
        "&:hover .e25-frame, &:focus-visible .e25-frame": {
          borderColor: amber,
        },
        "&:hover .e25-name, &:focus-visible .e25-name": { color: amber },
        "@media (prefers-reduced-motion: no-preference)": {
          "&:hover .e25-cover, &:focus-visible .e25-cover": {
            transform: "scale(1.045)",
          },
        },
      }}
    >
      <Box
        sx={{
          position: "relative",
          overflow: "hidden",
          aspectRatio: "600 / 780",
          bgcolor: sandDeep,
          border: "1px solid",
          borderColor: "divider",
        }}
      >
        {cover && cloudName ? (
          <Box
            className="e25-cover"
            component="img"
            src={photoUrl(cloudName, cover.publicId, CARD_TRANSFORM)}
            alt={design.name}
            loading="lazy"
            decoding="async"
            sx={{
              position: "absolute",
              inset: 0,
              width: "100%",
              height: "100%",
              objectFit: "cover",
              display: "block",
              bgcolor: sandDeep,
              transition: "transform 600ms cubic-bezier(0.22, 1, 0.36, 1)",
            }}
          />
        ) : (
          <Box
            className="e25-cover"
            sx={{
              position: "absolute",
              inset: 0,
              transition: "transform 600ms",
            }}
          >
            <PhotoPlaceholder name={design.name} />
          </Box>
        )}

        <Box
          className="e25-frame"
          aria-hidden
          sx={{
            position: "absolute",
            inset: 8,
            border: "1px solid transparent",
            transition: "border-color 240ms ease",
            pointerEvents: "none",
          }}
        />

        {/* Hover reveal */}
        <Box
          className="e25-view"
          aria-hidden
          sx={{
            position: "absolute",
            insetInline: 0,
            bottom: 0,
            p: 1.75,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            color: cream,
            background:
              "linear-gradient(to top, rgba(22,18,13,0.78), transparent)",
            opacity: 0,
            transform: "translateY(8px)",
            transition: "opacity 280ms ease, transform 280ms ease",
          }}
        >
          <Box
            component="span"
            sx={{
              fontFamily: monoFamily,
              fontSize: "0.6875rem",
              letterSpacing: "0.18em",
              textTransform: "uppercase",
            }}
          >
            View
          </Box>
          <ArrowOutwardIcon sx={{ fontSize: 16 }} />
        </Box>
      </Box>

      <Stack spacing={0.25} sx={{ pt: 1.5, minHeight: 58 }}>
        <Typography
          className="e25-name"
          variant="body2"
          component="h3"
          sx={{ fontWeight: 500, transition: "color 200ms ease" }}
        >
          {design.name}
        </Typography>
        {min !== null && (
          <Typography
            variant="body2"
            sx={{ color: "text.secondary", fontVariantNumeric: "tabular-nums" }}
          >
            {design.sizeBands.length > 1
              ? `From ${formatPesewas(min)}`
              : formatPesewas(min)}
          </Typography>
        )}
      </Stack>
    </Link>
  );
}

/** Responsive storefront card grid: 2-up on phones, 4-up on desktop. */
export function DesignGrid({
  designs,
  cloudName,
}: {
  designs: Design[];
  cloudName: string;
}) {
  return (
    <Box
      sx={{
        display: "grid",
        gridTemplateColumns: {
          xs: "repeat(2, 1fr)",
          sm: "repeat(3, 1fr)",
          md: "repeat(4, 1fr)",
        },
        gap: { xs: 2, md: 3 },
        rowGap: { xs: 3.5, md: 5 },
      }}
    >
      {designs.map((design) => (
        <DesignCard key={design.id} design={design} cloudName={cloudName} />
      ))}
    </Box>
  );
}
