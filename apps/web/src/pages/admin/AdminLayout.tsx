import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { NavLink, Outlet, useNavigate } from "react-router";
import { logout } from "@/lib/api";
import { ThemeToggle } from "@/components/ThemeToggle";
import { useMe } from "@/features/auth/useMe";
import { PERMISSIONS, type Permission } from "@/features/auth/permissions";
import { amber, brass, monoFamily } from "@/theme";

const NAV_ITEMS: { label: string; to: string; permission: Permission }[] = [
  { label: "Designs", to: "/admin/designs", permission: PERMISSIONS.catalogueRead },
  { label: "Collections", to: "/admin/collections", permission: PERMISSIONS.catalogueRead },
  { label: "Orders", to: "/admin/orders", permission: PERMISSIONS.ordersRead },
  { label: "Slots", to: "/admin/slots", permission: PERMISSIONS.slotsRead },
  { label: "Subscribers", to: "/admin/subscribers", permission: PERMISSIONS.subscribersRead },
  { label: "Analytics", to: "/admin/analytics", permission: PERMISSIONS.analyticsRead },
  { label: "Team", to: "/admin/team", permission: PERMISSIONS.teamRead },
  { label: "Settings", to: "/admin/settings", permission: PERMISSIONS.settingsWrite },
];

export function AdminLayout() {
  const me = useMe();
  const navItems = NAV_ITEMS.filter((item) => me.data?.permissions.includes(item.permission));
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const signOut = useMutation({
    mutationFn: logout,
    onSuccess: () => {
      navigate("/", { replace: true });
      queryClient.setQueryData(["me"], null);
      void queryClient.invalidateQueries({ queryKey: ["me"] });
    },
  });

  return (
    <Container component="main" maxWidth="lg">
      <Box sx={{ py: { xs: 5, md: 8 } }}>
        <Stack
          direction={{ xs: "column", sm: "row" }}
          spacing={2}
          sx={{ justifyContent: "space-between", alignItems: { sm: "center" } }}
        >
          <Typography variant="overline" component="p" sx={{ color: brass }}>
            eight two five — admin
          </Typography>
          <Stack direction="row" spacing={0.5} sx={{ alignItems: "center" }}>
            <ThemeToggle color="text.primary" />
            <Button
              variant="text"
              size="small"
              loading={signOut.isPending}
              onClick={() => signOut.mutate()}
            >
              Log out
            </Button>
          </Stack>
        </Stack>
        <Box
          component="nav"
          aria-label="Admin sections"
          sx={{
            mt: 2,
            borderBottom: "1px solid",
            borderColor: "divider",
            display: "flex",
            gap: { xs: 3, md: 4 },
            // Many tabs: let them scroll horizontally on small screens rather
            // than overflow off the edge.
            overflowX: "auto",
            flexWrap: "nowrap",
            scrollbarWidth: "none",
            "&::-webkit-scrollbar": { display: "none" },
          }}
        >
          {navItems.map((item) => (
            <Box
              key={item.to}
              component={NavLink}
              to={item.to}
              sx={{
                flexShrink: 0,
                display: "inline-block",
                pb: 1.5,
                textDecoration: "none",
                textTransform: "uppercase",
                fontFamily: monoFamily,
                letterSpacing: "0.16em",
                fontSize: "0.75rem",
                fontWeight: 500,
                whiteSpace: "nowrap",
                color: "text.secondary",
                borderBottom: "2px solid transparent",
                marginBottom: "-1px",
                transition: "color 160ms ease",
                "&.active": { color: "text.primary", borderBottomColor: amber },
                "&:hover": { color: "text.primary" },
                "&:focus-visible": { outline: `2px solid ${amber}`, outlineOffset: "2px" },
              }}
            >
              {item.label}
            </Box>
          ))}
        </Box>
        <Box sx={{ pt: { xs: 4, md: 6 } }}>
          <Outlet />
        </Box>
      </Box>
    </Container>
  );
}
