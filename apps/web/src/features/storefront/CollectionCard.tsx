import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Link as RouterLink } from "react-router";
import type { Collection } from "@/features/catalog/api";
import { brass, cream, GRAIN_URL, ink, sandDeep } from "@/theme";
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
      sx={{ color: "text.primary", display: "block" }}
    >
      {coverPublicId && cloudName ? (
        <Box
          component="img"
          src={photoUrl(cloudName, coverPublicId, CARD_TRANSFORM)}
          alt={collection.name}
          loading="lazy"
          decoding="async"
          sx={{
            width: "100%",
            aspectRatio: "600 / 780",
            objectFit: "cover",
            display: "block",
            bgcolor: sandDeep,
          }}
        />
      ) : (
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
            {collection.name}
          </Typography>
        </Box>
      )}
      <Stack spacing={0.25} sx={{ pt: 1.5 }}>
        <Typography variant="body2" component="h3" sx={{ fontWeight: 500 }}>
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
