import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import { Link as RouterLink } from "react-router";
import { BrandMark } from "@/components/BrandMark";
import { MeasureRule } from "@/components/MeasureRule";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { amber, brass, cream, creamText, GRAIN_URL, ink, monoFamily } from "@/theme";

export function NotFoundPage() {
  useDocumentTitle("Page not found");

  return (
    <Box
      component="main"
      sx={{
        minHeight: "100dvh",
        bgcolor: ink,
        color: cream,
        display: "flex",
        alignItems: "center",
        position: "relative",
        overflow: "hidden",
        "&::before": { content: '""', position: "absolute", inset: 0, backgroundImage: GRAIN_URL, opacity: 0.05 },
        "&::after": {
          content: '""',
          position: "absolute",
          top: "-20%",
          right: "-10%",
          width: "55%",
          height: "120%",
          background: `radial-gradient(closest-side, ${amber}26, transparent 70%)`,
          pointerEvents: "none",
        },
      }}
    >
      <Container maxWidth="md" sx={{ position: "relative", zIndex: 1, py: { xs: 8, md: 0 } }}>
        <Stack spacing={4} sx={{ maxWidth: 680 }}>
          <Box component={RouterLink} to="/" aria-label="Eight Two Five — home" sx={{ display: "inline-block" }}>
            <BrandMark size={40} />
          </Box>

          <Box>
            <Box component="span" sx={{ fontFamily: monoFamily, fontSize: "0.6875rem", letterSpacing: "0.24em", color: brass, textTransform: "uppercase" }}>
              Error 404 — Page not found
            </Box>
            <Typography variant="h1" sx={{ mt: 2 }}>
              Lost thread.
            </Typography>
          </Box>

          <Typography sx={{ color: creamText, maxWidth: "52ch" }}>
            The page you're after has been retired, or it never existed. Every
            thread leads back home.
          </Typography>

          <MeasureRule variant="light" caption="Return to the store" />

          <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5}>
            <Button
              component={RouterLink}
              to="/"
              size="large"
              endIcon={<ArrowOutwardIcon sx={{ fontSize: 18 }} />}
              sx={{ bgcolor: amber, color: ink, "&:hover": { bgcolor: brass }, width: { xs: "100%", sm: "auto" } }}
            >
              Back to home
            </Button>
            <Button
              component={RouterLink}
              to="/store"
              variant="outlined"
              size="large"
              sx={{ color: cream, borderColor: "rgba(232,222,203,0.45)", "&:hover": { borderColor: cream }, width: { xs: "100%", sm: "auto" } }}
            >
              Browse the store
            </Button>
          </Stack>
        </Stack>
      </Container>
    </Box>
  );
}
