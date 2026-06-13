import { ApiError, request, type Subscriber } from "@/lib/api";

export type { Subscriber };

/** One page of a paginated admin listing, mirroring the API envelope. */
export interface PagedResult<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface PageParams {
  page: number;
  pageSize: number;
}

export const DEFAULT_PAGE_SIZE = 20;

export function errorMessage(
  error: unknown,
  fallback = "Something went wrong. Try again in a moment.",
): string {
  return error instanceof ApiError ? error.message : fallback;
}

function pageQuery({ page, pageSize }: PageParams): string {
  const search = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  });

  return search.toString();
}

/** Lists newsletter subscribers (admin only), newest first, paginated. */
export function listSubscribers(params: PageParams): Promise<PagedResult<Subscriber>> {
  return request<PagedResult<Subscriber>>(`/api/v1/admin/waitlist?${pageQuery(params)}`);
}

/**
 * Serializes subscribers to a CSV string with a header row. Fields are quoted
 * and embedded quotes are doubled, so commas and quotes in names survive.
 */
export function subscribersToCsv(subscribers: Subscriber[]): string {
  const header = ["Email", "Name", "Joined"];
  const rows = subscribers.map((s) => [s.email, s.name, s.createdAt]);

  return [header, ...rows]
    .map((row) => row.map(csvCell).join(","))
    .join("\r\n");
}

function csvCell(value: string): string {
  return `"${value.replace(/"/g, '""')}"`;
}
