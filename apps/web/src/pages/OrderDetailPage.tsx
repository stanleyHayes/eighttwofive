import type { ReactNode } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Divider from "@mui/material/Divider";
import Link from "@mui/material/Link";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Link as RouterLink, useParams } from "react-router";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import EmailOutlinedIcon from "@mui/icons-material/EmailOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { formatPesewas } from "@/features/catalog/money";
import {
  customerStageLabel,
  effectivePricePesewas,
  hasVisit,
  paymentStatus,
  type Order,
} from "@/features/orders/api";
import { useOrder } from "@/features/orders/hooks";
import {
  DETAIL_TRANSFORM,
  photoUrl,
  type PublicSettings,
} from "@/features/storefront/api";
import { usePublicSettings } from "@/features/storefront/hooks";
import { ApiError } from "@/lib/api";
import { clayDeep, sandDeep, stone } from "@/theme";

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof ApiError ? error.message : fallback;
}

function DeliveryLine({ order, t }: { order: Order; t: TFunction }) {
  const { mode, area, ratePesewas } = order.delivery;

  if (mode === "pickup") {
    return (
      <Typography sx={{ color: "text.secondary" }}>
        {t("orderDetail.deliveryPickup")}
      </Typography>
    );
  }

  if (ratePesewas == null) {
    return (
      <Typography sx={{ color: "text.secondary" }}>
        {t("orderDetail.deliveryArranged", {
          area: area || t("orderDetail.yourArea"),
        })}
      </Typography>
    );
  }

  return (
    <Typography sx={{ color: "text.secondary" }}>
      {t("orderDetail.deliveryDispatch", {
        area,
        rate: formatPesewas(ratePesewas),
      })}
    </Typography>
  );
}

function VisitCard({
  whatsappNumber,
  t,
}: {
  whatsappNumber?: string;
  t: TFunction;
}) {
  return (
    <Card
      variant="outlined"
      sx={{ bgcolor: "background.paper", borderColor: "divider" }}
    >
      <CardContent>
        <Typography variant="h5" component="h2" sx={{ mb: 1 }}>
          {t("orderDetail.visitTitle")}
        </Typography>
        <Typography sx={{ color: "text.secondary", mb: 2 }}>
          {t("orderDetail.visitBody")}
        </Typography>
        <Button
          variant="contained"
          href={whatsappNumber ? `tel:${whatsappNumber}` : undefined}
          disabled={!whatsappNumber}
          sx={{ width: { xs: "100%", sm: "auto" } }}
        >
          {t("orderDetail.visitCall")}
        </Button>
      </CardContent>
    </Card>
  );
}

function isCustomerFacingStatus(status: Order["status"]): boolean {
  return (
    status === "booked" || status === "in_production" || status === "ready"
  );
}

function StatusAlert({ order, t }: { order: Order; t: TFunction }) {
  if (!isCustomerFacingStatus(order.status)) {
    return null;
  }

  const label = customerStageLabel(order.status);
  const timeframe = order.quote.timeline
    ? t("orderDetail.estimated", { timeline: order.quote.timeline })
    : t("orderDetail.timeframeFallback");

  return (
    <Alert
      severity="info"
      icon={<EmailOutlinedIcon fontSize="inherit" />}
      sx={{ bgcolor: "background.paper", borderColor: "divider" }}
    >
      <Stack spacing={0.5}>
        <Typography variant="subtitle2" component="p" sx={{ fontWeight: 600 }}>
          {label}
        </Typography>
        <Typography variant="body2" sx={{ color: "text.secondary" }}>
          {t("orderDetail.emailNote", { timeframe })}
        </Typography>
      </Stack>
    </Alert>
  );
}

function OrderContent({
  order,
  settings,
  t,
}: {
  order: Order;
  settings?: PublicSettings;
  t: TFunction;
}) {
  const imageUrl =
    settings?.cloudName && order.designSnapshot.photoPublicId
      ? photoUrl(
          settings.cloudName,
          order.designSnapshot.photoPublicId,
          DETAIL_TRANSFORM,
        )
      : null;
  const stage = customerStageLabel(order.status);
  const timeframe = order.quote.timeline
    ? t("orderDetail.estimated", { timeline: order.quote.timeline })
    : null;
  const price = effectivePricePesewas(order);

  return (
    <Stack spacing={4}>
      <StatusAlert order={order} t={t} />

      <Box>
        <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
          {t("orderDetail.orderRef", { ref: order.ref })}
        </Typography>
        <Typography variant="h2" component="h1" sx={{ mt: 1.5 }}>
          {stage}
        </Typography>
        {timeframe && (
          <Typography
            variant="subtitle1"
            sx={{ color: "text.secondary", mt: 1 }}
          >
            {timeframe}
          </Typography>
        )}
      </Box>

      <Card
        variant="outlined"
        sx={{ bgcolor: "background.paper", borderColor: "divider" }}
      >
        <Stack direction={{ xs: "column", md: "row" }}>
          {imageUrl ? (
            <Box
              component="img"
              src={imageUrl}
              alt={order.designSnapshot.name}
              loading="lazy"
              decoding="async"
              sx={{
                width: { xs: "100%", md: 320 },
                height: { xs: 280, md: 320 },
                objectFit: "cover",
                display: "block",
                bgcolor: sandDeep,
                borderRight: { md: "1px solid" },
                borderColor: { md: "divider" },
                flexShrink: 0,
              }}
            />
          ) : (
            <Box
              sx={{
                width: { xs: "100%", md: 320 },
                height: { xs: 280, md: 320 },
                bgcolor: sandDeep,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                p: 3,
                borderRight: { md: "1px solid" },
                borderColor: { md: "divider" },
                flexShrink: 0,
              }}
            >
              <Typography variant="h5" sx={{ textAlign: "center" }}>
                {order.designSnapshot.name}
              </Typography>
            </Box>
          )}
          <CardContent sx={{ flex: 1, p: { xs: 3, md: 4 } }}>
            <Stack spacing={2}>
              <Typography variant="h5" component="h2">
                {order.designSnapshot.name}
              </Typography>
              {price > 0 && (
                <Typography
                  variant="subtitle1"
                  sx={{ color: "text.secondary" }}
                >
                  {formatPesewas(price)}
                </Typography>
              )}
              <Divider />
              <Box
                sx={{
                  display: "grid",
                  gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)" },
                  gap: { xs: 2.25, sm: 3 },
                }}
              >
                <DetailItem
                  label={t("orderDetail.detailSize")}
                  value={
                    order.customisation.bandLabel ||
                    order.customisation.sizeMode
                  }
                />
                <DetailItem
                  label={t("orderDetail.detailPayment")}
                  value={paymentStatus(order)}
                />
                <DetailItem
                  label={t("orderDetail.detailPhone")}
                  value={order.customerPhone}
                />
                <DetailItem
                  label={t("orderDetail.detailDelivery")}
                  value={<DeliveryLine order={order} t={t} />}
                />
              </Box>
            </Stack>
          </CardContent>
        </Stack>
      </Card>

      {hasVisit(order) && (
        <VisitCard whatsappNumber={settings?.whatsappNumber} t={t} />
      )}
    </Stack>
  );
}

function DetailItem({ label, value }: { label: string; value: ReactNode }) {
  return (
    <Box>
      <Typography
        variant="overline"
        component="span"
        sx={{ color: stone, display: "block" }}
      >
        {label}
      </Typography>
      {typeof value === "string" ? (
        <Typography sx={{ color: "text.primary" }}>{value}</Typography>
      ) : (
        value
      )}
    </Box>
  );
}

export function OrderDetailPage() {
  const { t } = useTranslation();
  const { ref } = useParams<{ ref: string }>();
  const order = useOrder(ref ?? "");
  const settings = usePublicSettings();

  const notFound =
    order.error instanceof ApiError && order.error.status === 404;

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 8, md: 13 }, maxWidth: 920 }}>
        <Link
          component={RouterLink}
          to="/account"
          underline="none"
          variant="overline"
          sx={{
            color: stone,
            display: "inline-flex",
            alignItems: "center",
            gap: 0.75,
            mb: 3,
            "&:hover": { color: "text.primary" },
          }}
        >
          <ArrowBackIcon sx={{ fontSize: 15 }} />
          {t("orderDetail.backToAccount")}
        </Link>

        {order.isLoading || settings.isLoading ? (
          <Stack spacing={4}>
            <Box>
              <Skeleton variant="text" width={120} />
              <Skeleton variant="text" width={240} height={56} sx={{ mt: 1 }} />
              <Skeleton variant="text" width={180} sx={{ mt: 1 }} />
            </Box>
            <Skeleton variant="rectangular" height={320} />
          </Stack>
        ) : notFound ? (
          <EmptyState
            label={t("orderDetail.notFoundLabel")}
            title={t("orderDetail.notFoundTitle")}
            body={t("orderDetail.notFoundBody")}
            action={{ label: t("orderDetail.backToAccount"), to: "/account" }}
          />
        ) : order.error ? (
          <ErrorState
            message={errorMessage(order.error, t("orderDetail.loadError"))}
            onRetry={() => order.refetch()}
          />
        ) : order.data ? (
          <OrderContent order={order.data} settings={settings.data} t={t} />
        ) : null}
      </Box>
    </StorefrontLayout>
  );
}
