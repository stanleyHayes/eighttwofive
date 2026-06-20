import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createRole,
  deleteRole,
  invitePartner,
  listPermissions,
  listRoles,
  updateRole,
  type InviteInput,
  type RoleInput,
} from "./api";

export function useRoles() {
  return useQuery({ queryKey: ["admin", "roles"], queryFn: listRoles });
}

export function usePermissions() {
  // The catalogue only changes when the code adds an enforcement point.
  return useQuery({ queryKey: ["admin", "permissions"], queryFn: listPermissions, staleTime: Infinity });
}

// A role edit can change the signed-in user's own permissions and the labels in
// the team table, so refresh those alongside the role list.
function invalidateRoles(queryClient: ReturnType<typeof useQueryClient>) {
  void queryClient.invalidateQueries({ queryKey: ["admin", "roles"] });
  void queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
  void queryClient.invalidateQueries({ queryKey: ["me"] });
}

export function useCreateRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: RoleInput) => createRole(input),
    onSuccess: () => invalidateRoles(queryClient),
  });
}

export function useUpdateRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (vars: { key: string; input: RoleInput }) => updateRole(vars.key, vars.input),
    onSuccess: () => invalidateRoles(queryClient),
  });
}

export function useDeleteRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (key: string) => deleteRole(key),
    onSuccess: () => invalidateRoles(queryClient),
  });
}

export function useInvitePartner() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: InviteInput) => invitePartner(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
    },
  });
}
