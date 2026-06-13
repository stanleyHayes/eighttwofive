import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import { Link as RouterLink } from "react-router";
import { formatPesewas } from "@/features/catalog/money";
import { CARD_TRANSFORM, photoUrl } from "@/features/storefront/api";
import {
  amber,
  brass,
  cream,
  GRAIN_URL,
  ink,
  monoFamily,
  sandDeep,
} from "@/theme";
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
    <Card
      variant="outlined"
      sx={{
        display: "flex",
        flexDirection: { xs: "column", sm: "row" },
        bgcolor: "background.paper",
        borderColor: "divider",
        transition: "border-color 180ms ease, transform 180ms ease",
        "@media (prefers-reduced-motion: no-preference)": {
          "&:hover": { transform: "translateY(-2px)" },
        },
        "&:hover": { borderColor: amber },
      }}
    >
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
            borderRight: { sm: "1px solid" },
            borderColor: { sm: "divider" },
            flexShrink: 0,
          }}
        />
      ) : (
        <Box
          sx={{
            width: { xs: "100%", sm: 160 },
            height: { xs: 200, sm: 160 },
            bgcolor: ink,
            color: cream,
            backgroundImage: GRAIN_URL,
            backgroundBlendMode: "overlay",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            p: 2,
            borderRight: { sm: "1px solid" },
            borderColor: { sm: "divider" },
            flexShrink: 0,
          }}
        >
          <Typography
            variant="body2"
            sx={{ textAlign: "center", maxWidth: "14ch" }}
          >
            {order.designSnapshot.name}
          </Typography>
        </Box>
      )}
      <CardContent
        sx={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          gap: 1.25,
          p: { xs: 2.5, sm: 3 },
        }}
      >
        <Stack
          direction="row"
          spacing={1}
          sx={{ justifyContent: "space-between", alignItems: "flex-start" }}
        >
          <Typography
            variant="overline"
            sx={{ color: brass, fontVariantNumeric: "tabular-nums" }}
          >
            {order.ref}
          </Typography>
          <Typography
            variant="body2"
            sx={{ color: "text.secondary", fontVariantNumeric: "tabular-nums" }}
          >
            {date}
          </Typography>
        </Stack>
        <Typography variant="h5" component="h3" sx={{ fontWeight: 500 }}>
          {order.designSnapshot.name}
        </Typography>
        {order.designSnapshot.pricePesewas > 0 && (
          <Typography
            variant="body2"
            sx={{ color: "text.secondary", fontVariantNumeric: "tabular-nums" }}
          >
            {formatPesewas(order.designSnapshot.pricePesewas)}
          </Typography>
        )}
        <Stack
          direction="row"
          spacing={1.5}
          sx={{ mt: "auto", pt: 1.5, alignItems: "center" }}
        >
          <Box
            component="span"
            sx={{
              border: "1px solid",
              borderColor: "divider",
              px: 1,
              py: 0.5,
              color: "text.primary",
              fontFamily: monoFamily,
              fontSize: "0.6875rem",
              textTransform: "uppercase",
            }}
          >
            {customerStageLabel(order.status)}
          </Box>
          <Box sx={{ flex: 1 }} />
          <Link
            component={RouterLink}
            to={`/account/orders/${order.ref}`}
            underline="none"
            variant="overline"
            sx={{
              color: "text.primary",
              display: "inline-flex",
              alignItems: "center",
              gap: 0.75,
              "&:hover": { color: amber },
            }}
          >
            View order <ArrowOutwardIcon sx={{ fontSize: 15 }} />
          </Link>
        </Stack>
      </CardContent>
    </Card>
  );
}
