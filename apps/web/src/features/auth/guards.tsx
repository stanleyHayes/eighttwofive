import type { ReactNode } from "react";
import Box from "@mui/material/Box";
import Skeleton from "@mui/material/Skeleton";
import { Navigate } from "react-router";
import { useMe } from "./useMe";

/**
 * Minimal full-height placeholder during the brief session check — a quiet
 * skeleton rather than a spinner, in keeping with the rest of the app.
 */
function CheckingSession() {
  return (
    <Box
      aria-hidden
      sx={{
        minHeight: "100dvh",
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        gap: 2,
        px: { xs: 3, md: 8 },
        maxWidth: 720,
        mx: "auto",
      }}
    >
      <Skeleton variant="text" width={140} />
      <Skeleton variant="text" width="60%" height={56} />
      <Skeleton variant="rectangular" height={160} sx={{ mt: 2 }} />
    </Box>
  );
}

export function AuthGuard({ children }: { children: ReactNode }) {
  const me = useMe();

  if (me.isPending) return <CheckingSession />;
  if (!me.data) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export function AdminGuard({ children }: { children: ReactNode }) {
  const me = useMe();

  if (me.isPending) return <CheckingSession />;
  if (!me.data) return <Navigate to="/login" replace />;
  if (me.data.role !== "admin") return <Navigate to="/" replace />;
  return <>{children}</>;
}
