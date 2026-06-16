import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import CheckCircleOutlined from "@mui/icons-material/CheckCircleOutlined";
import ReceiptLongOutlined from "@mui/icons-material/ReceiptLongOutlined";
import { Link as RouterLink, useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { useMe } from "@/features/auth/useMe";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { amber, brass, ink, monoFamily } from "@/theme";

/**
 * Where Paystack returns the customer after checkout. The order is confirmed
 * server-side by the webhook, so this page is purely a reassuring landing: it
 * surfaces the reference Paystack hands back and points the customer onward.
 */
export function PaymentCallbackPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("payment.docTitle"));

  const [searchParams] = useSearchParams();
  // Paystack echoes the order ref as both `reference` and `trxref`.
  const reference =
    searchParams.get("reference") ?? searchParams.get("trxref") ?? "";

  const me = useMe();
  const signedIn = Boolean(me.data);

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 5, md: 8 }, maxWidth: 820 }}>
        <PageBanner
          tone="ink"
          icon={<CheckCircleOutlined />}
          breadcrumbs={[
            { label: t("payment.breadcrumbHome"), to: "/" },
            { label: t("payment.breadcrumbPayment") },
          ]}
          title={t("payment.bannerTitle")}
          description={t("payment.bannerDescription")}
        />

        <Box
          sx={{
            mt: { xs: 4, md: 5 },
            border: "1px solid",
            borderColor: "divider",
            p: { xs: 3, sm: 4 },
            bgcolor: "background.default",
          }}
        >
          {reference ? (
            <>
              <Typography
                variant="overline"
                component="p"
                sx={{ color: "text.secondary" }}
              >
                {t("payment.referenceLabel")}
              </Typography>
              <Typography
                sx={{
                  mt: 0.5,
                  fontFamily: monoFamily,
                  fontSize: "1.25rem",
                  letterSpacing: "0.04em",
                }}
              >
                {reference}
              </Typography>
            </>
          ) : (
            <Typography sx={{ color: "text.secondary" }}>
              {t("payment.noReference")}
            </Typography>
          )}

          <Typography sx={{ mt: 3, color: "text.secondary", maxWidth: "60ch" }}>
            {t("payment.reassurance")}
          </Typography>

          <Stack
            direction={{ xs: "column", sm: "row" }}
            spacing={1.5}
            sx={{ mt: 4 }}
          >
            {signedIn && (
              <Button
                component={RouterLink}
                to="/account"
                variant="contained"
                size="large"
                startIcon={<ReceiptLongOutlined sx={{ fontSize: 18 }} />}
                sx={{
                  bgcolor: amber,
                  color: ink,
                  "&:hover": { bgcolor: brass },
                  width: { xs: "100%", sm: "auto" },
                }}
              >
                {t("payment.viewOrders")}
              </Button>
            )}
            <Button
              component={RouterLink}
              to="/store"
              variant={signedIn ? "outlined" : "contained"}
              size="large"
              endIcon={<ArrowOutwardIcon sx={{ fontSize: 18 }} />}
              sx={
                signedIn
                  ? { width: { xs: "100%", sm: "auto" } }
                  : {
                      bgcolor: amber,
                      color: ink,
                      "&:hover": { bgcolor: brass },
                      width: { xs: "100%", sm: "auto" },
                    }
              }
            >
              {t("payment.continueBrowsing")}
            </Button>
          </Stack>
        </Box>
      </Box>
    </StorefrontLayout>
  );
}
