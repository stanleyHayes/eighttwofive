import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import { Link as RouterLink } from "react-router";
import type { Collection } from "@/features/catalog/api";
import {
  amber,
  brass,
  cream,
  GRAIN_URL,
  ink,
  monoFamily,
  sandDeep,
} from "@/theme";
import { CARD_TRANSFORM, photoUrl } from "./api";

export function CollectionCard({
  collection,
  coverPublicId,
  cloudName,
}: {
  collection: Collection;
  coverPublicId: string | null;
  cloudName: string;
  /** Retained for grid alternation; the no-cover fallback is now ink. */
  tone?: string;
}) {
  return (
    <Link
      component={RouterLink}
      to={`/collections/${collection.slug}`}
      underline="none"
      sx={{
        color: "text.primary",
        display: "block",
        height: "100%",
        "&:hover .e25-frame, &:focus-visible .e25-frame": {
          borderColor: amber,
        },
        "&:hover .e25-view, &:focus-visible .e25-view": {
          opacity: 1,
          transform: "none",
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
          bgcolor: coverPublicId && cloudName ? sandDeep : ink,
          border: "1px solid",
          borderColor: "divider",
        }}
      >
        {coverPublicId && cloudName ? (
          <Box
            className="e25-cover"
            component="img"
            src={photoUrl(cloudName, coverPublicId, CARD_TRANSFORM)}
            alt={collection.name}
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
              bgcolor: ink,
              backgroundImage: GRAIN_URL,
              backgroundBlendMode: "overlay",
              display: "flex",
              flexDirection: "column",
              justifyContent: "space-between",
              p: 2.5,
              transition: "transform 600ms cubic-bezier(0.22, 1, 0.36, 1)",
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
              {collection.name}
            </Typography>
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
              textTransform: "uppercase",
            }}
          >
            View collection
          </Box>
          <ArrowOutwardIcon sx={{ fontSize: 16 }} />
        </Box>
      </Box>
      <Stack spacing={0.25} sx={{ pt: 1.5, minHeight: 76 }}>
        <Typography
          className="e25-name"
          variant="body2"
          component="h3"
          sx={{ fontWeight: 500, transition: "color 200ms ease" }}
        >
          {collection.name}
        </Typography>
        {collection.note && (
          <Typography variant="body2" sx={{ color: "text.secondary" }}>
            {collection.note}
          </Typography>
        )}
      </Stack>
    </Link>
  );
}
