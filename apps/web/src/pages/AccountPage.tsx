import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Container from "@mui/material/Container";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import PersonOutlined from "@mui/icons-material/PersonOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { OrderSummaryCard } from "@/features/orders/OrderSummaryCard";
import { useOrders } from "@/features/orders/hooks";
import { usePublicSettings } from "@/features/storefront/hooks";
import { ApiError, logout } from "@/lib/api";
import { useMe } from "@/features/auth/useMe";

function errorMessage(error: unknown, fallback = "Could not load your orders. Try again in a moment."): string {
  return error instanceof ApiError ? error.message : fallback;
}

export function AccountPage() {
  const me = useMe();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const orders = useOrders();
  const settings = usePublicSettings();

  const signOut = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      navigate("/", { replace: true });
      queryClient.setQueryData(["me"], null);
      void queryClient.invalidateQueries({ queryKey: ["me"] });
    },
  });

  return (
    <StorefrontLayout>
      <Container component="main" maxWidth="lg">
        <Box sx={{ py: { xs: 5, md: 8 } }}>
          <PageBanner
            tone="ink"
            icon={<PersonOutlined />}
            breadcrumbs={[{ label: "Home", to: "/" }, { label: "Account" }]}
            title="Your account"
            description={me.data?.name ? `Welcome back, ${me.data.name}.` : undefined}
          />
        </Box>
        <Box sx={{ pb: { xs: 8, md: 13 }, maxWidth: 720 }}>
          <Typography variant="overline" component="h2" sx={{ color: "text.secondary" }}>
            your orders
          </Typography>

          {orders.isLoading && (
            <Stack spacing={2} sx={{ mt: 3 }}>
              <Skeleton variant="rectangular" height={160} />
              <Skeleton variant="rectangular" height={160} />
            </Stack>
          )}

          {orders.error && !orders.isLoading && (
            <Box sx={{ mt: 3 }}>
              <ErrorState
                message={errorMessage(orders.error)}
                onRetry={() => orders.refetch()}
              />
            </Box>
          )}

          {!orders.isLoading && !orders.error && orders.data && orders.data.length === 0 && (
            <Box sx={{ mt: 3 }}>
              <EmptyState
                label="No orders yet"
                title="Nothing in your wardrobe yet."
                body="Orders will appear here once you place one. Browse the live collections to find your first piece."
                action={{ label: "Browse the store", to: "/store" }}
              />
            </Box>
          )}

          {!orders.isLoading && !orders.error && orders.data && orders.data.length > 0 && (
            <Stack spacing={3} sx={{ mt: 3 }}>
              {orders.data.map((order) => (
                <OrderSummaryCard key={order.id} order={order} cloudName={settings.data?.cloudName} />
              ))}
            </Stack>
          )}

          <Button
            variant="outlined"
            color="primary"
            loading={signOut.isPending}
            onClick={() => signOut.mutate()}
            sx={{ mt: 6 }}
          >
            Log out
          </Button>
        </Box>
      </Container>
    </StorefrontLayout>
  );
}
