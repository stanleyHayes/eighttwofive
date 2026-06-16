import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import EventAvailableOutlined from "@mui/icons-material/EventAvailableOutlined";
import StraightenOutlined from "@mui/icons-material/StraightenOutlined";
import { useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router";
import { MeasureRule } from "@/components/MeasureRule";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { formatPesewas } from "@/features/catalog/money";
import { usePublicSettings } from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import {
  amber,
  brass,
  cream,
  creamText,
  GRAIN_URL,
  ink,
  monoFamily,
} from "@/theme";

function FitPathCard({
  n,
  title,
  body,
  action,
}: {
  n: string;
  title: string;
  body: string;
  action: { label: string; to: string };
}) {
  return (
    <Box
      sx={{
        bgcolor: "background.paper",
        border: "1px solid",
        borderColor: "divider",
        p: { xs: 3, md: 4 },
        minHeight: 320,
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
      }}
    >
      <Box>
        <Box
          component="span"
          sx={{ fontFamily: monoFamily, color: brass, fontSize: "0.75rem" }}
        >
          {n}
        </Box>
        <Typography variant="h4" component="h2" sx={{ mt: 2, mb: 1.5 }}>
          {title}
        </Typography>
        <Typography variant="body2" sx={{ color: "text.secondary" }}>
          {body}
        </Typography>
      </Box>
      <Button
        component={RouterLink}
        to={action.to}
        variant="outlined"
        endIcon={<ArrowOutwardIcon sx={{ fontSize: 16 }} />}
        sx={{ mt: 3, width: { xs: "100%", sm: "fit-content" } }}
      >
        {action.label}
      </Button>
    </Box>
  );
}

export function FitGuidePage() {
  const { t } = useTranslation();
  useDocumentTitle(t("fit.documentTitle"), t("fit.documentDescription"));
  const settings = usePublicSettings();
  const deposit = settings.data?.depositPesewas ?? 500_00;
  const location = settings.data?.visitLocation || t("fit.defaultLocation");

  const fitPaths = [
    {
      n: "01",
      title: t("fit.path1Title"),
      body: t("fit.path1Body"),
      action: { label: t("fit.path1ActionLabel"), to: "/store" },
    },
    {
      n: "02",
      title: t("fit.path2Title"),
      body: t("fit.path2Body"),
      action: { label: t("fit.path2ActionLabel"), to: "/store" },
    },
    {
      n: "03",
      title: t("fit.path3Title"),
      body: t("fit.path3Body"),
      action: { label: t("fit.path3ActionLabel"), to: "/slots" },
    },
  ];

  const measures = [
    {
      label: t("fit.measureBustLabel"),
      body: t("fit.measureBustBody"),
    },
    {
      label: t("fit.measureWaistLabel"),
      body: t("fit.measureWaistBody"),
    },
    {
      label: t("fit.measureHipsLabel"),
      body: t("fit.measureHipsBody"),
    },
    {
      label: t("fit.measureLengthLabel"),
      body: t("fit.measureLengthBody"),
    },
  ];

  const timeline = [
    { label: t("fit.timelineSelectLabel"), body: t("fit.timelineSelectBody") },
    {
      label: t("fit.timelineConfirmLabel"),
      body: t("fit.timelineConfirmBody"),
    },
    { label: t("fit.timelineCutLabel"), body: t("fit.timelineCutBody") },
    {
      label: t("fit.timelineFinishLabel"),
      body: t("fit.timelineFinishBody"),
    },
  ];

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 4, md: 6 } }}>
        <PageBanner
          tone="ink"
          icon={<StraightenOutlined />}
          breadcrumbs={[
            { label: t("fit.breadcrumbHome"), to: "/" },
            { label: t("fit.breadcrumbFitGuide") },
          ]}
          title={t("fit.bannerTitle")}
          description={t("fit.bannerDescription")}
          action={{ to: "/store", label: t("fit.bannerActionLabel") }}
        />
      </Box>

      <Box component="section" sx={{ mb: { xs: 7, md: 10 } }}>
        <MeasureRule
          label={t("fit.fig01Label")}
          sx={{ mb: { xs: 3.5, md: 5 } }}
        />
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" },
            gap: { xs: 2, md: 3 },
          }}
        >
          {fitPaths.map((path) => (
            <FitPathCard key={path.n} {...path} />
          ))}
        </Box>
      </Box>

      <Box
        component="section"
        sx={{
          mb: { xs: 7, md: 10 },
          display: "grid",
          gridTemplateColumns: { xs: "1fr", md: "0.85fr 1.15fr" },
          gap: { xs: 4, md: 7 },
          alignItems: "start",
        }}
      >
        <Box>
          <Typography variant="overline" component="p" sx={{ color: brass }}>
            {t("fit.measureYourselfOverline")}
          </Typography>
          <Typography variant="h2" component="h2" sx={{ mt: 1.5, mb: 2 }}>
            {t("fit.measureYourselfTitle")}
          </Typography>
          <Typography
            variant="subtitle1"
            sx={{ color: "text.secondary", maxWidth: "42ch" }}
          >
            {t("fit.measureYourselfBody")}
          </Typography>
        </Box>

        <Box sx={{ borderTop: "1px solid", borderColor: "divider" }}>
          {measures.map((measure, index) => (
            <Box
              key={measure.label}
              sx={{
                display: "grid",
                gridTemplateColumns: { xs: "auto 1fr", sm: "96px 1fr" },
                gap: { xs: 2, sm: 3 },
                py: 3,
                borderBottom: "1px solid",
                borderColor: "divider",
              }}
            >
              <Typography
                variant="overline"
                component="p"
                sx={{ color: brass }}
              >
                {String(index + 1).padStart(2, "0")}
              </Typography>
              <Box>
                <Typography variant="h5" component="h3">
                  {measure.label}
                </Typography>
                <Typography
                  variant="body2"
                  sx={{ color: "text.secondary", mt: 0.5 }}
                >
                  {measure.body}
                </Typography>
              </Box>
            </Box>
          ))}
        </Box>
      </Box>

      <Box
        component="section"
        sx={{
          bgcolor: ink,
          color: cream,
          mx: { xs: -2, sm: -3 },
          px: { xs: 2, sm: 3 },
          py: { xs: 7, md: 10 },
          mb: { xs: 7, md: 10 },
          backgroundImage: GRAIN_URL,
          backgroundBlendMode: "overlay",
        }}
      >
        <MeasureRule
          variant="light"
          label={t("fit.fig02Label")}
          caption={t("fit.fig02Caption")}
          sx={{ mb: { xs: 4, md: 5 } }}
        />
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: { xs: "1fr", md: "1fr 1fr" },
            gap: { xs: 4, md: 7 },
            alignItems: "center",
          }}
        >
          <Box>
            <Typography
              variant="h2"
              component="h2"
              sx={{ color: cream, maxWidth: "12ch" }}
            >
              {t("fit.visitTitle")}
            </Typography>
            <Typography sx={{ color: creamText, mt: 2.5, maxWidth: "44ch" }}>
              {t("fit.visitBody", {
                location,
                deposit: formatPesewas(deposit),
              })}
            </Typography>
          </Box>
          <Box
            sx={{
              border: "1px solid rgba(232,222,203,0.22)",
              p: { xs: 3, md: 4 },
              minHeight: 260,
              display: "flex",
              flexDirection: "column",
              justifyContent: "space-between",
            }}
          >
            <EventAvailableOutlined sx={{ color: amber, fontSize: 40 }} />
            <Box>
              <Typography
                variant="h5"
                component="h3"
                sx={{ color: cream, mb: 1 }}
              >
                {t("fit.visitDepositTitle")}
              </Typography>
              <Typography variant="body2" sx={{ color: creamText }}>
                {t("fit.visitDepositBody")}
              </Typography>
            </Box>
            <Button
              component={RouterLink}
              to="/slots"
              variant="contained"
              endIcon={<ArrowOutwardIcon sx={{ fontSize: 16 }} />}
              sx={{
                mt: 3,
                bgcolor: amber,
                color: ink,
                width: { xs: "100%", sm: "fit-content" },
              }}
            >
              {t("fit.visitButtonLabel")}
            </Button>
          </Box>
        </Box>
      </Box>

      <Box component="section" sx={{ mb: { xs: 8, md: 12 } }}>
        <MeasureRule
          label={t("fit.fig03Label")}
          sx={{ mb: { xs: 3.5, md: 5 } }}
        />
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: {
              xs: "1fr",
              sm: "repeat(2, 1fr)",
              md: "repeat(4, 1fr)",
            },
            borderTop: "1px solid",
            borderLeft: "1px solid",
            borderColor: "divider",
          }}
        >
          {timeline.map((item, index) => (
            <Box
              key={item.label}
              sx={{
                minHeight: 190,
                p: 3,
                borderRight: "1px solid",
                borderBottom: "1px solid",
                borderColor: "divider",
              }}
            >
              <Typography
                variant="overline"
                component="p"
                sx={{ color: brass }}
              >
                {t("fit.timelineStep", {
                  n: String(index + 1).padStart(2, "0"),
                })}
              </Typography>
              <Typography variant="h5" component="h3" sx={{ mt: 1.5 }}>
                {item.label}
              </Typography>
              <Typography
                variant="body2"
                sx={{ color: "text.secondary", mt: 1 }}
              >
                {item.body}
              </Typography>
            </Box>
          ))}
        </Box>
      </Box>
    </StorefrontLayout>
  );
}
