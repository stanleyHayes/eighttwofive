import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Link as RouterLink } from "react-router";
import { MeasureRule } from "@/components/MeasureRule";
import { brass, cream, creamMuted, GRAIN_URL, ink } from "@/theme";

/**
 * Friendly 404 panel for retired/unknown catalog pages — every collection is
 * limited, so a missing slug is part of the story, not an error screen.
 * Dressed in the dark-atelier aesthetic: a warm ink field, mono eyebrow, a
 * signature MeasureRule and a display headline.
 */
export function RetiredPanel({
  overline,
  title,
  body,
}: {
  overline: string;
  title: string;
  body: string;
}) {
  return (
    <Box sx={{ py: { xs: 8, md: 12 } }}>
      <Box
        sx={{
          position: "relative",
          maxWidth: 640,
          px: { xs: 4, md: 6 },
          py: { xs: 6, md: 8 },
          overflow: "hidden",
          bgcolor: ink,
          color: "common.white",
          border: "1px solid",
          borderColor: "rgba(232, 222, 203, 0.16)",
          backgroundImage: GRAIN_URL,
          backgroundBlendMode: "overlay",
        }}
      >
        <Stack spacing={3} sx={{ position: "relative" }}>
          <MeasureRule variant="light" label="FIG." caption="Made to order" />
          <Box>
            <Typography variant="overline" component="p" sx={{ color: brass }}>
              {overline}
            </Typography>
            <Typography
              variant="h2"
              component="h1"
              sx={{ mt: 1.5, mb: 2, color: cream }}
            >
              {title}
            </Typography>
            <Typography sx={{ color: creamMuted, maxWidth: "48ch" }}>{body}</Typography>
          </Box>
          <Box>
            <Button
              component={RouterLink}
              to="/store"
              variant="outlined"
              sx={{ color: cream, borderColor: "rgba(232, 222, 203, 0.5)" }}
            >
              Back to the store
            </Button>
          </Box>
        </Stack>
      </Box>
    </Box>
  );
}
