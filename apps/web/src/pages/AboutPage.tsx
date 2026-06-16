import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import AutoAwesomeOutlined from "@mui/icons-material/AutoAwesomeOutlined";
import { useTranslation } from "react-i18next";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { useDocumentTitle } from "@/lib/useDocumentTitle";

export function AboutPage() {
  const { t } = useTranslation();
  const craftBlocks = [
    {
      title: t("about.craftAtelierTitle"),
      body: t("about.craftAtelierBody"),
    },
    {
      title: t("about.craftFabricTitle"),
      body: t("about.craftFabricBody"),
    },
  ];
  useDocumentTitle(t("about.documentTitle"), t("about.documentDescription"));

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 4, md: 6 } }}>
        <PageBanner
          tone="ink"
          icon={<AutoAwesomeOutlined />}
          breadcrumbs={[
            { label: t("about.breadcrumbHome"), to: "/" },
            { label: t("about.breadcrumbAtelier") },
          ]}
          title={t("about.bannerTitle")}
          description={t("about.bannerDescription")}
        />
      </Box>

      {/* Story */}
      <Box component="section" sx={{ mb: { xs: 6, md: 10 }, maxWidth: 640 }}>
        <Typography variant="overline" component="p" sx={{ color: "text.secondary", mb: 1 }}>
          {t("about.storyOverline")}
        </Typography>
        <Typography variant="subtitle1" sx={{ color: "text.secondary", maxWidth: "54ch" }}>
          {t("about.storyBody")}
        </Typography>
      </Box>

      {/* Craft tiles */}
      <Stack
        component="section"
        direction={{ xs: "column", md: "row" }}
        spacing={2}
        sx={{ mb: { xs: 8, md: 12 } }}
      >
        {craftBlocks.map((block, index) => (
          <Box
            key={block.title}
            sx={{
              flex: 1,
              bgcolor: "background.paper",
              border: "1px solid",
              borderColor: "divider",
              p: 4,
              minHeight: 200,
              display: "flex",
              flexDirection: "column",
              justifyContent: "space-between",
            }}
          >
            <Typography variant="overline" component="p" sx={{ color: "text.secondary", mb: 1 }}>
              {`Fig. 0${index + 1}`}
            </Typography>
            <Box>
              <Typography variant="h5" component="h2" sx={{ mb: 1.5 }}>
                {block.title}
              </Typography>
              <Typography variant="body2" sx={{ color: "text.secondary" }}>
                {block.body}
              </Typography>
            </Box>
          </Box>
        ))}
      </Stack>
    </StorefrontLayout>
  );
}
