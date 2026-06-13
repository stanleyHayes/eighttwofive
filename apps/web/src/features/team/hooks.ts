import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { UserRole } from "@/lib/api";
import type { PageParams } from "@/features/catalog/api";
import { listUsers, setUserRole } from "./api";

export function useUsers(params: PageParams) {
  return useQuery({
    queryKey: ["admin", "users", params.page, params.pageSize],
    queryFn: () => listUsers(params),
    placeholderData: keepPreviousData,
  });
}

export function useSetUserRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (vars: { id: string; role: UserRole }) => setUserRole(vars.id, vars.role),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
      void queryClient.invalidateQueries({ queryKey: ["me"] });
    },
  });
}
