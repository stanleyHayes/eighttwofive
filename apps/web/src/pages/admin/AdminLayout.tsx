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
  {
    label: "Designs",
    to: "/admin/designs",
    permission: PERMISSIONS.catalogueRead,
  },
  {
    label: "Collections",
    to: "/admin/collections",
    permission: PERMISSIONS.catalogueRead,
  },
  { label: "Orders", to: "/admin/orders", permission: PERMISSIONS.ordersRead },
  { label: "Slots", to: "/admin/slots", permission: PERMISSIONS.slotsRead },
  {
    label: "Subscribers",
    to: "/admin/subscribers",
    permission: PERMISSIONS.subscribersRead,
  },
  {
    label: "Analytics",
    to: "/admin/analytics",
    permission: PERMISSIONS.analyticsRead,
  },
  { label: "Team", to: "/admin/team", permission: PERMISSIONS.teamRead },
  {
    label: "Settings",
    to: "/admin/settings",
    permission: PERMISSIONS.settingsWrite,
  },
];

export function AdminLayout() {
  const me = useMe();
  const navItems = NAV_ITEMS.filter((item) =>
    me.data?.permissions.includes(item.permission),
  );
  const displayName = me.data?.name || me.data?.email || "Signed in";
  const roleLabel = me.data?.isSuperAdmin
    ? "Super-admin"
    : me.data?.role
      ? me.data.role.replace(/_/g, " ")
      : "Admin";
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
        <Stack spacing={2.5}>
          <Stack
            direction={{ xs: "column", sm: "row" }}
            spacing={2}
            sx={{
              justifyContent: "space-between",
              alignItems: { sm: "center" },
              bgcolor: "background.paper",
              border: "1px solid",
              borderColor: "divider",
              px: { xs: 2.5, md: 3 },
              py: { xs: 2, md: 2.5 },
            }}
          >
            <Box>
              <Typography
                variant="overline"
                component="p"
                sx={{ color: brass }}
              >
                eight two five — admin
              </Typography>
              <Typography
                variant="body2"
                sx={{ color: "text.secondary", mt: 0.5 }}
              >
                {displayName} · {roleLabel}
              </Typography>
            </Box>
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
              borderBottom: "1px solid",
              borderColor: "divider",
              display: "flex",
              gap: { xs: 2.5, md: 4 },
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
                  px: 0.25,
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
                  transition: "color 160ms ease, border-color 160ms ease",
                  "&.active": {
                    color: "text.primary",
                    borderBottomColor: amber,
                  },
                  "&:hover": { color: "text.primary" },
                  "&:focus-visible": {
                    outline: `2px solid ${amber}`,
                    outlineOffset: "2px",
                  },
                }}
              >
                {item.label}
              </Box>
            ))}
          </Box>
        </Stack>
        <Box sx={{ pt: { xs: 4, md: 6 } }}>
          <Outlet />
        </Box>
      </Box>
    </Container>
  );
}
