import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import { useTranslation } from "react-i18next";
import {
  Link as RouterLink,
  isRouteErrorResponse,
  useRouteError,
} from "react-router";
import { BrandMark } from "@/components/BrandMark";
import { MeasureRule } from "@/components/MeasureRule";
import { amber, brass, cream, creamText, GRAIN_URL, ink, monoFamily } from "@/theme";

/**
 * Root errorElement: catches render and loader errors anywhere in the tree so a
 * crash shows a branded recovery page instead of a blank white screen. A 404
 * route response is forwarded to the dedicated not-found copy; everything else
 * is treated as an unexpected failure.
 */
export function ErrorPage() {
  const { t } = useTranslation();
  const error = useRouteError();

  const is404 = isRouteErrorResponse(error) && error.status === 404;
  const label = is404 ? t("errorPage.label404") : t("errorPage.label");
  const heading = is404 ? t("errorPage.heading404") : t("errorPage.heading");
  const body = is404 ? t("errorPage.body404") : t("errorPage.body");

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
        "&::before": {
          content: '""',
          position: "absolute",
          inset: 0,
          backgroundImage: GRAIN_URL,
          opacity: 0.05,
        },
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
          <Box component={RouterLink} to="/" aria-label={t("errorPage.homeAria")} sx={{ display: "inline-block" }}>
            <BrandMark size={40} />
          </Box>

          <Box>
            <Box
              component="span"
              sx={{
                fontFamily: monoFamily,
                fontSize: "0.6875rem",
                letterSpacing: "0.24em",
                color: brass,
                textTransform: "uppercase",
              }}
            >
              {label}
            </Box>
            <Typography variant="h1" sx={{ mt: 2 }}>
              {heading}
            </Typography>
          </Box>

          <Typography sx={{ color: creamText, maxWidth: "52ch" }}>{body}</Typography>

          <MeasureRule variant="light" caption={t("errorPage.caption")} />

          <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5}>
            {!is404 && (
              <Button
                onClick={() => window.location.reload()}
                size="large"
                sx={{ bgcolor: amber, color: ink, "&:hover": { bgcolor: brass }, width: { xs: "100%", sm: "auto" } }}
              >
                {t("errorPage.tryAgain")}
              </Button>
            )}
            <Button
              component={RouterLink}
              to="/"
              size="large"
              variant={is404 ? "contained" : "outlined"}
              endIcon={is404 ? <ArrowOutwardIcon sx={{ fontSize: 18 }} /> : undefined}
              sx={
                is404
                  ? { bgcolor: amber, color: ink, "&:hover": { bgcolor: brass }, width: { xs: "100%", sm: "auto" } }
                  : { color: cream, borderColor: "rgba(232,222,203,0.45)", "&:hover": { borderColor: cream }, width: { xs: "100%", sm: "auto" } }
              }
            >
              {t("errorPage.backHome")}
            </Button>
          </Stack>
        </Stack>
      </Container>
    </Box>
  );
}
