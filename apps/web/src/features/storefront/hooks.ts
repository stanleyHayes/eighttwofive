import { useQuery } from "@tanstack/react-query";
import { ApiError } from "@/lib/api";
import {
  getPublicCollection,
  getPublicDesign,
  getPublicSettings,
  listPublicCollections,
  listPublicDesigns,
  type PublicDesignParams,
} from "./api";

const storeKey = ["store"] as const;

/** A 404 is a real answer (retired/unknown) — don't retry it. */
function retryUnless404(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status === 404) return false;
  return failureCount < 1;
}

export function usePublicSettings() {
  return useQuery({
    queryKey: [...storeKey, "settings"],
    queryFn: getPublicSettings,
    staleTime: 5 * 60_000,
  });
}

export function usePublicCollections() {
  return useQuery({
    queryKey: [...storeKey, "collections"],
    queryFn: listPublicCollections,
  });
}

export function usePublicCollection(slug: string) {
  return useQuery({
    queryKey: [...storeKey, "collections", slug],
    queryFn: () => getPublicCollection(slug),
    retry: retryUnless404,
  });
}

export function usePublicDesigns(params: PublicDesignParams = {}) {
  return useQuery({
    queryKey: [...storeKey, "designs", params.collectionId ?? "", params.q ?? ""],
    queryFn: () => listPublicDesigns(params),
  });
}

export function usePublicDesign(slug: string) {
  return useQuery({
    queryKey: [...storeKey, "designs", "by-slug", slug],
    queryFn: () => getPublicDesign(slug),
    retry: retryUnless404,
  });
}
