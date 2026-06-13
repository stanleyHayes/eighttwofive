import { useMe } from "./useMe";

/** Capability strings, mirroring the API's domain.Permission values. */
export const PERMISSIONS = {
  analyticsRead: "analytics:read",
  ordersRead: "orders:read",
  ordersWrite: "orders:write",
  slotsRead: "slots:read",
  slotsWrite: "slots:write",
  catalogueRead: "catalogue:read",
  catalogueWrite: "catalogue:write",
  subscribersRead: "subscribers:read",
  settingsWrite: "settings:write",
  teamRead: "team:read",
  teamWrite: "team:write",
} as const;

export type Permission = (typeof PERMISSIONS)[keyof typeof PERMISSIONS];

/** True when the signed-in user's role grants the given permission. */
export function useCan(permission: Permission): boolean {
  const me = useMe();

  return me.data?.permissions?.includes(permission) ?? false;
}
