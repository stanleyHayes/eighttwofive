import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { NavLink, Outlet, useNavigate } from "react-router";
import { logout } from "@/lib/api";
import { ThemeToggle } from "@/components/ThemeToggle";
import { amber, brass, monoFamily } from "@/theme";

const NAV_ITEMS = [
  { label: "Designs", to: "/admin/designs" },
  { label: "Collections", to: "/admin/collections" },
  { label: "Orders", to: "/admin/orders" },
  { label: "Slots", to: "/admin/slots" },
  { label: "Subscribers", to: "/admin/subscribers" },
  { label: "Analytics", to: "/admin/analytics" },
  { label: "Settings", to: "/admin/settings" },
];

export function AdminLayout() {
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
          {NAV_ITEMS.map((item) => (
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
