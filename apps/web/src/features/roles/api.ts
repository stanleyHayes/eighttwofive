import { request } from "@/lib/api";
import type { User } from "@/lib/api";

export { errorMessage } from "@/features/catalog/api";

/** A named, editable bundle of permissions (built-in or custom). */
export interface RoleDef {
  key: string;
  name: string;
  description: string;
  permissions: string[];
  /** Built-in role: cannot be deleted and its key is fixed. */
  system: boolean;
  /** Whether the role may enter the admin dashboard at all. */
  adminArea: boolean;
}

/** A single capability a role can grant, with UI labelling. */
export interface PermissionMeta {
  key: string;
  label: string;
  description: string;
  group: string;
}

export interface RoleInput {
  name: string;
  description: string;
  permissions: string[];
  adminArea: boolean;
}

export interface InviteInput {
  email: string;
  name: string;
  role: string;
}

export function listRoles(): Promise<RoleDef[]> {
  return request<RoleDef[]>("/api/v1/admin/roles");
}

export function listPermissions(): Promise<PermissionMeta[]> {
  return request<PermissionMeta[]>("/api/v1/admin/permissions");
}

export function createRole(input: RoleInput): Promise<RoleDef> {
  return request<RoleDef>("/api/v1/admin/roles", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateRole(key: string, input: RoleInput): Promise<RoleDef> {
  return request<RoleDef>(`/api/v1/admin/roles/${encodeURIComponent(key)}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function deleteRole(key: string): Promise<void> {
  return request<void>(`/api/v1/admin/roles/${encodeURIComponent(key)}`, {
    method: "DELETE",
  });
}

export function invitePartner(input: InviteInput): Promise<{ user: User }> {
  return request<{ user: User }>("/api/v1/admin/invitations", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
