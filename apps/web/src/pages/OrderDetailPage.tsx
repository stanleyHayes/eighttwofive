import type { ReactNode } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Container from "@mui/material/Container";
import Divider from "@mui/material/Divider";
import Link from "@mui/material/Link";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Link as RouterLink, useParams } from "react-router";
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
import { DETAIL_TRANSFORM, photoUrl, type PublicSettings } from "@/features/storefront/api";
import { usePublicSettings } from "@/features/storefront/hooks";
import { ApiError } from "@/lib/api";
import { clayDeep, sandDeep, stone } from "@/theme";

function errorMessage(error: unknown, fallback = "Could not load this order. Try again in a moment."): string {
  return error instanceof ApiError ? error.message : fallback;
}

function DeliveryLine({ order }: { order: Order }) {
  const { mode, area, ratePesewas } = order.delivery;

  if (mode === "pickup") {
    return (
      <Typography sx={{ color: "text.secondary" }}>
        Pickup — free, collect when your order is ready.
      </Typography>
    );
  }

  if (ratePesewas == null) {
    return (
      <Typography sx={{ color: "text.secondary" }}>
        Dispatch to {area || "your area"} — delivery arranged directly with Eight Two Five.
      </Typography>
    );
  }

  return (
    <Typography sx={{ color: "text.secondary" }}>
      Dispatch to {area} — {formatPesewas(ratePesewas)}
    </Typography>
  );
}

function VisitCard({ whatsappNumber }: { whatsappNumber?: string }) {
  return (
    <Card variant="outlined" sx={{ bgcolor: "background.paper", borderColor: "divider" }}>
      <CardContent>
        <Typography variant="h5" component="h2" sx={{ mb: 1 }}>
          Booked visit
        </Typography>
        <Typography sx={{ color: "text.secondary", mb: 2 }}>
          Your home-visit slot will appear here once the calendar is live.
        </Typography>
        <Button
          variant="contained"
          href={whatsappNumber ? `tel:${whatsappNumber}` : undefined}
          disabled={!whatsappNumber}
          sx={{ width: { xs: "100%", sm: "auto" } }}
        >
          Call Eight Two Five
        </Button>
      </CardContent>
    </Card>
  );
}

function isCustomerFacingStatus(status: Order["status"]): boolean {
  return status === "booked" || status === "in_production" || status === "ready";
}

function StatusAlert({ order }: { order: Order }) {
  if (!isCustomerFacingStatus(order.status)) {
    return null;
  }

  const label = customerStageLabel(order.status);
  const timeframe = order.quote.timeline
    ? `Estimated: ${order.quote.timeline}`
    : "Roughly two weeks, depending on current bookings";

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
          {timeframe}. We&apos;ll email you at each step.
        </Typography>
      </Stack>
    </Alert>
  );
}

function OrderContent({ order, settings }: { order: Order; settings?: PublicSettings }) {
  const imageUrl =
    settings?.cloudName && order.designSnapshot.photoPublicId
      ? photoUrl(settings.cloudName, order.designSnapshot.photoPublicId, DETAIL_TRANSFORM)
      : null;
  const stage = customerStageLabel(order.status);
  const timeframe = order.quote.timeline ? `Estimated: ${order.quote.timeline}` : null;
  const price = effectivePricePesewas(order);

  return (
    <Stack spacing={4}>
      <StatusAlert order={order} />

      <Box>
        <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
          order {order.ref}
        </Typography>
        <Typography variant="h2" component="h1" sx={{ mt: 1.5 }}>
          {stage}
        </Typography>
        {timeframe && (
          <Typography variant="subtitle1" sx={{ color: "text.secondary", mt: 1 }}>
            {timeframe}
          </Typography>
        )}
      </Box>

      <Card variant="outlined">
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
                flexShrink: 0,
              }}
            >
              <Typography variant="h5" sx={{ textAlign: "center" }}>
                {order.designSnapshot.name}
              </Typography>
            </Box>
          )}
          <CardContent sx={{ flex: 1 }}>
            <Stack spacing={2}>
              <Typography variant="h5" component="h2">
                {order.designSnapshot.name}
              </Typography>
              {price > 0 && (
                <Typography variant="subtitle1" sx={{ color: "text.secondary" }}>
                  {formatPesewas(price)}
                </Typography>
              )}
              <Divider />
              <DetailItem label="Size" value={order.customisation.bandLabel || order.customisation.sizeMode} />
              <DetailItem label="Delivery" value={<DeliveryLine order={order} />} />
              <DetailItem label="Payment" value={paymentStatus(order)} />
              <DetailItem label="Phone" value={order.customerPhone} />
            </Stack>
          </CardContent>
        </Stack>
      </Card>

      {hasVisit(order) && <VisitCard whatsappNumber={settings?.whatsappNumber} />}
    </Stack>
  );
}

function DetailItem({ label, value }: { label: string; value: ReactNode }) {
  return (
    <Box>
      <Typography variant="overline" component="span" sx={{ color: stone, display: "block" }}>
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
  const { ref } = useParams<{ ref: string }>();
  const order = useOrder(ref ?? "");
  const settings = usePublicSettings();

  const notFound =
    order.error instanceof ApiError && order.error.status === 404;

  return (
    <StorefrontLayout>
      <Container component="main" maxWidth="lg">
        <Box sx={{ py: { xs: 8, md: 13 }, maxWidth: 720 }}>
          <Link component={RouterLink} to="/account" underline="hover" sx={{ color: stone, display: "inline-block", mb: 3 }}>
            ← Back to account
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
              label="Order not found"
              title="We couldn't find that order."
              body="The reference may be mistyped, or the order belongs to a different account. Your orders are listed on your account page."
              action={{ label: "Back to account", to: "/account" }}
            />
          ) : order.error ? (
            <ErrorState message={errorMessage(order.error)} onRetry={() => order.refetch()} />
          ) : order.data ? (
            <OrderContent order={order.data} settings={settings.data} />
          ) : null}
        </Box>
      </Container>
    </StorefrontLayout>
  );
}
