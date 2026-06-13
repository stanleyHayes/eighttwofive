import type { QueryClient } from "@tanstack/react-query";
import type { User } from "@/lib/api";

/** A fully-privileged admin, for tests that exercise write controls. */
export const ADMIN_ME: User = {
  id: "admin-1",
  email: "admin@example.com",
  name: "Admin",
  role: "admin",
  permissions: [
    "analytics:read",
    "orders:read",
    "orders:write",
    "slots:read",
    "slots:write",
    "catalogue:read",
    "catalogue:write",
    "subscribers:read",
    "settings:write",
    "team:read",
    "team:write",
  ],
  isSuperAdmin: true,
};

/** Pre-populates the `me` query so permission-gated UI renders in tests. */
export function seedAdmin(client: QueryClient): void {
  client.setQueryData(["me"], ADMIN_ME);
}
