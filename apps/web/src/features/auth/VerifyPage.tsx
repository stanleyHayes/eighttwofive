import { useEffect, useRef } from "react";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import CircularProgress from "@mui/material/CircularProgress";
import Typography from "@mui/material/Typography";
import Link from "@mui/material/Link";
import { Link as RouterLink, useNavigate, useSearchParams } from "react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { verifyLogin } from "@/lib/api";
import { clayDeep } from "@/theme";

export function VerifyPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token");
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const fired = useRef(false);

  const verify = useMutation({
    mutationFn: verifyLogin,
    onSuccess: async ({ user }) => {
      queryClient.setQueryData(["me"], user);
      await queryClient.invalidateQueries({ queryKey: ["me"] });
      navigate(user.role === "admin" ? "/admin" : "/account", { replace: true });
    },
  });
  const { mutate } = verify;

  useEffect(() => {
    if (fired.current || !token) return;
    fired.current = true;
    mutate(token);
  }, [token, mutate]);

  const failed = !token || verify.isError;

  return (
    <Container component="main" maxWidth="lg">
      <Box sx={{ py: { xs: 8, md: 13 }, maxWidth: 480 }}>
        <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
          sign in
        </Typography>
        <Typography variant="h2" component="h1" sx={{ mt: 1.5, mb: 4 }}>
          {failed ? "That link didn't work" : "Signing you in"}
        </Typography>

        {failed ? (
          <Box
            role="alert"
            sx={{ border: "1px solid", borderColor: "error.main", p: 4 }}
          >
            <Typography variant="overline" component="p" sx={{ color: "error.main" }}>
              link expired or invalid
            </Typography>
            <Typography sx={{ mt: 1.5 }}>
              Sign-in links only work once and expire quickly.{" "}
              <Link component={RouterLink} to="/login">
                Request a new sign-in link
              </Link>
              .
            </Typography>
          </Box>
        ) : (
          <Stack direction="row" spacing={2} sx={{ alignItems: "center" }}>
            <CircularProgress size={24} aria-label="Verifying" />
            <Typography variant="overline" sx={{ color: "text.secondary" }}>
              verifying your link
            </Typography>
          </Stack>
        )}
      </Box>
    </Container>
  );
}
