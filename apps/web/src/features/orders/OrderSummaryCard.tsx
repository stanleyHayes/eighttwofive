import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { Link as RouterLink } from "react-router";
import { formatPesewas } from "@/features/catalog/money";
import { CARD_TRANSFORM, photoUrl } from "@/features/storefront/api";
import { sandDeep } from "@/theme";
import type { Order } from "./api";
import { customerStageLabel } from "./api";

export interface OrderSummaryCardProps {
  order: Order;
  cloudName?: string;
}

export function OrderSummaryCard({ order, cloudName }: OrderSummaryCardProps) {
  const date = new Date(order.createdAt).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
  const imageUrl =
    cloudName && order.designSnapshot.photoPublicId
      ? photoUrl(cloudName, order.designSnapshot.photoPublicId, CARD_TRANSFORM)
      : null;

  return (
    <Card variant="outlined" sx={{ display: "flex", flexDirection: { xs: "column", sm: "row" } }}>
      {imageUrl ? (
        <Box
          component="img"
          src={imageUrl}
          alt={order.designSnapshot.name}
          loading="lazy"
          decoding="async"
          sx={{
            width: { xs: "100%", sm: 160 },
            height: { xs: 200, sm: 160 },
            objectFit: "cover",
            display: "block",
            bgcolor: sandDeep,
            flexShrink: 0,
          }}
        />
      ) : (
        <Box
          sx={{
            width: { xs: "100%", sm: 160 },
            height: { xs: 200, sm: 160 },
            bgcolor: sandDeep,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            p: 2,
            flexShrink: 0,
          }}
        >
          <Typography variant="body2" sx={{ textAlign: "center" }}>
            {order.designSnapshot.name}
          </Typography>
        </Box>
      )}
      <CardContent sx={{ flex: 1, display: "flex", flexDirection: "column", gap: 1 }}>
        <Stack direction="row" spacing={1} sx={{ justifyContent: "space-between", alignItems: "flex-start" }}>
          <Typography variant="body2" sx={{ color: "text.secondary" }}>
            {order.ref}
          </Typography>
          <Typography variant="body2" sx={{ color: "text.secondary" }}>
            {date}
          </Typography>
        </Stack>
        <Typography variant="h5" component="h3" sx={{ fontWeight: 500 }}>
          {order.designSnapshot.name}
        </Typography>
        {order.designSnapshot.pricePesewas > 0 && (
          <Typography variant="body2" sx={{ color: "text.secondary" }}>
            {formatPesewas(order.designSnapshot.pricePesewas)}
          </Typography>
        )}
        <Stack direction="row" spacing={1} sx={{ mt: "auto", pt: 1, alignItems: "center" }}>
          <Typography
            variant="overline"
            component="span"
            sx={{ color: "primary.main", fontWeight: 600, letterSpacing: "0.08em" }}
          >
            {customerStageLabel(order.status)}
          </Typography>
          <Box sx={{ flex: 1 }} />
          <Link component={RouterLink} to={`/account/orders/${order.ref}`} underline="hover">
            View order
          </Link>
        </Stack>
      </CardContent>
    </Card>
  );
}
