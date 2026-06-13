import { request } from "@/lib/api";
import type { User, UserRole } from "@/lib/api";
import type { PagedResult, PageParams } from "@/features/catalog/api";

export { DEFAULT_PAGE_SIZE, errorMessage } from "@/features/catalog/api";
export type { PageParams } from "@/features/catalog/api";

export function listUsers(params: PageParams): Promise<PagedResult<User>> {
  const search = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
  });

  return request<PagedResult<User>>(`/api/v1/admin/users?${search.toString()}`);
}

export function setUserRole(id: string, role: UserRole): Promise<{ user: User }> {
  return request<{ user: User }>(`/api/v1/admin/users/${id}/role`, {
    method: "PUT",
    body: JSON.stringify({ role }),
  });
}
