import type { ReactNode } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import MailOutlined from "@mui/icons-material/MailOutlined";
import WhatsAppIcon from "@mui/icons-material/WhatsApp";
import { Link as RouterLink } from "react-router";
import { useTranslation } from "react-i18next";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { usePublicSettings } from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { amber, brass, ink } from "@/theme";

function waHref(whatsappNumber: string): string {
  return `https://wa.me/${whatsappNumber.replace(/\D/g, "")}`;
}

function ContactRow({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <Box sx={{ borderTop: "1px solid", borderColor: "divider", py: 3 }}>
      <Typography
        variant="overline"
        component="p"
        sx={{ color: "text.secondary", mb: 0.75 }}
      >
        {label}
      </Typography>
      {children}
    </Box>
  );
}

function ActionPanel({
  label,
  title,
  body,
  action,
  icon,
}: {
  label: string;
  title: string;
  body: string;
  action: ReactNode;
  icon: ReactNode;
}) {
  return (
    <Box
      sx={{
        position: "relative",
        overflow: "hidden",
        bgcolor: "background.paper",
        border: "1px solid",
        borderColor: "divider",
        p: { xs: 3, md: 4 },
        minHeight: 280,
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
      }}
    >
      <Box
        aria-hidden
        sx={{
          position: "absolute",
          right: -20,
          bottom: -24,
          color: brass,
          opacity: 0.08,
          lineHeight: 0,
          "& svg": { fontSize: 180 },
        }}
      >
        {icon}
      </Box>
      <Box sx={{ position: "relative" }}>
        <Typography
          variant="overline"
          component="p"
          sx={{ color: brass, mb: 1.5 }}
        >
          {label}
        </Typography>
        <Typography variant="h4" component="h2" sx={{ mb: 1.5 }}>
          {title}
        </Typography>
        <Typography
          variant="body2"
          sx={{ color: "text.secondary", maxWidth: "42ch" }}
        >
          {body}
        </Typography>
      </Box>
      <Box sx={{ position: "relative", mt: 3 }}>{action}</Box>
    </Box>
  );
}

export function ContactPage() {
  const { t } = useTranslation();
  useDocumentTitle(
    t("contact.docTitle"),
    t("contact.docDescription"),
  );
  const settings = usePublicSettings();
  const whatsapp = settings.data?.whatsappNumber ?? "";
  const visitLocation = settings.data?.visitLocation ?? "";

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 4, md: 6 } }}>
        <PageBanner
          tone="ink"
          icon={<MailOutlined />}
          breadcrumbs={[
            { label: t("contact.breadcrumbHome"), to: "/" },
            { label: t("contact.breadcrumbContact") },
          ]}
          title={t("contact.bannerTitle")}
          description={t("contact.bannerDescription")}
        />
      </Box>

      <Box
        component="section"
        sx={{
          mb: { xs: 6, md: 8 },
          display: "grid",
          gridTemplateColumns: { xs: "1fr", md: "1fr 1fr" },
          gap: { xs: 2.5, md: 3 },
        }}
      >
        <ActionPanel
          label={t("contact.messageLabel")}
          title={t("contact.messageTitle")}
          body={t("contact.messageBody")}
          icon={<WhatsAppIcon />}
          action={
            whatsapp ? (
              <Button
                href={waHref(whatsapp)}
                target="_blank"
                rel="noreferrer"
                variant="contained"
                endIcon={<ArrowOutwardIcon />}
                sx={{
                  bgcolor: amber,
                  color: ink,
                  width: { xs: "100%", sm: "auto" },
                }}
              >
                {t("contact.openWhatsApp")}
              </Button>
            ) : (
              <Button
                disabled
                variant="contained"
                endIcon={<ArrowOutwardIcon />}
                sx={{ width: { xs: "100%", sm: "auto" } }}
              >
                {t("contact.openWhatsApp")}
              </Button>
            )
          }
        />
        <ActionPanel
          label={t("contact.fittingLabel")}
          title={t("contact.fittingTitle")}
          body={t("contact.fittingBody")}
          icon={<MailOutlined />}
          action={
            <Button
              component={RouterLink}
              to="/slots"
              variant="outlined"
              endIcon={<ArrowOutwardIcon />}
              sx={{ width: { xs: "100%", sm: "auto" } }}
            >
              {t("contact.bookVisit")}
            </Button>
          }
        />
      </Box>

      <Box component="section" sx={{ mb: { xs: 8, md: 12 }, maxWidth: 720 }}>
        <Typography
          variant="subtitle1"
          sx={{ color: "text.secondary", maxWidth: "54ch", mb: 5 }}
        >
          {t("contact.intro")}
        </Typography>

        <Stack>
          <ContactRow label={t("contact.whatsappLabel")}>
            {whatsapp ? (
              <Link
                href={waHref(whatsapp)}
                target="_blank"
                rel="noreferrer"
                underline="hover"
                sx={{ color: "text.primary" }}
              >
                <Typography
                  variant="h5"
                  component="span"
                  sx={{ letterSpacing: "0.06em" }}
                >
                  {whatsapp}
                </Typography>
              </Link>
            ) : (
              <Typography variant="body2" sx={{ color: "text.secondary" }}>
                {t("contact.whatsappPending")}
              </Typography>
            )}
          </ContactRow>

          <ContactRow label={t("contact.visitLabel")}>
            {visitLocation ? (
              <Typography variant="h5" component="p">
                {visitLocation}
              </Typography>
            ) : (
              <Typography variant="body2" sx={{ color: "text.secondary" }}>
                {t("contact.visitPending")}
              </Typography>
            )}
          </ContactRow>

          <ContactRow label={t("contact.emailLabel")}>
            <Link
              href="mailto:hello@eighttwofive.com"
              underline="hover"
              sx={{ color: "text.primary" }}
            >
              <Typography
                variant="h5"
                component="span"
                sx={{ letterSpacing: "0.04em" }}
              >
                hello@eighttwofive.com
              </Typography>
            </Link>
          </ContactRow>
        </Stack>
      </Box>
    </StorefrontLayout>
  );
}
