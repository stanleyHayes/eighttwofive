import { request } from "@/lib/api";
import type { Collection, Design } from "@/features/catalog/api";

/** Public, no-auth read API for the storefront (live catalog only). */

export interface DeliveryRate {
  area: string;
  ratePesewas: number;
}

export interface PublicSettings {
  depositPesewas: number;
  whatsappNumber: string;
  visitLocation: string;
  instagramHandle: string;
  contactEmail: string;
  cloudName: string;
  deliveryRates: DeliveryRate[];
}

export interface CollectionWithDesigns {
  collection: Collection;
  designs: Design[];
}

export interface PublicDesignParams {
  collectionId?: string;
  q?: string;
}

export function getPublicSettings(): Promise<PublicSettings> {
  return request<PublicSettings>("/api/v1/settings");
}

export function listPublicCollections(): Promise<Collection[]> {
  return request<Collection[]>("/api/v1/collections");
}

/** 404s (ApiError.status === 404) when the collection is retired or unknown. */
export function getPublicCollection(slug: string): Promise<CollectionWithDesigns> {
  return request<CollectionWithDesigns>(`/api/v1/collections/${encodeURIComponent(slug)}`);
}

export function listPublicDesigns(params: PublicDesignParams = {}): Promise<Design[]> {
  const search = new URLSearchParams();
  if (params.collectionId) search.set("collection", params.collectionId);
  if (params.q) search.set("q", params.q);
  const qs = search.toString();
  return request<Design[]>(`/api/v1/designs${qs ? `?${qs}` : ""}`);
}

/** 404s (ApiError.status === 404) when the design is retired or unknown. */
export function getPublicDesign(slug: string): Promise<Design> {
  return request<Design>(`/api/v1/designs/${encodeURIComponent(slug)}`);
}

// --- Cloudinary photo URLs ---

/** Card grids: cropped portrait. */
export const CARD_TRANSFORM = "c_fill,w_600,h_780";
/** Detail gallery main image: full width, no crop. */
export const DETAIL_TRANSFORM = "w_1200";
/** Detail gallery thumbnail strip. */
export const THUMB_TRANSFORM = "c_fill,w_120,h_156";

export function photoUrl(cloudName: string, publicId: string, transform: string): string {
  return `https://res.cloudinary.com/${cloudName}/image/upload/${transform}/${publicId}`;
}

/** Photos sorted by their explicit order. */
export function sortedPhotos(design: Design): Design["photos"] {
  return [...design.photos].sort((a, b) => a.order - b.order);
}

/** Lowest size-band price, or null when the design has no bands. */
export function minBandPesewas(design: Design): number | null {
  if (design.sizeBands.length === 0) return null;
  return Math.min(...design.sizeBands.map((band) => band.pricePesewas));
}
