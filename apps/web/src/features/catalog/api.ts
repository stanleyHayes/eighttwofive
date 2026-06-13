import { ApiError, request } from "@/lib/api";

export type CatalogStatus = "live" | "retired";

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
/** Mirrors the API's domain.MaxPageSize — the largest page a request may pull. */
export const MAX_PAGE_SIZE = 100;

export interface Collection {
  id: string;
  name: string;
  slug: string;
  note: string;
  status: CatalogStatus;
  createdAt: string;
  retiredAt?: string;
}

export interface DesignPhoto {
  publicId: string;
  order: number;
}

export interface SizeBand {
  label: string;
  pricePesewas: number;
  chart: Record<string, string>;
}

export interface Design {
  id: string;
  collectionId: string;
  name: string;
  slug: string;
  note: string;
  photos: DesignPhoto[];
  sizeBands: SizeBand[];
  status: CatalogStatus;
  createdAt: string;
  retiredAt?: string;
}

export interface CollectionInput {
  name: string;
  note: string;
}

export interface DesignInput {
  collectionId: string;
  name: string;
  note: string;
  photos: DesignPhoto[];
  sizeBands: SizeBand[];
}

export function errorMessage(
  error: unknown,
  fallback = "Something went wrong. Try again in a moment.",
): string {
  return error instanceof ApiError ? error.message : fallback;
}

// --- Collections ---

export async function listCollections(): Promise<Collection[]> {
  // The admin collections endpoint is paginated, but the design filter and the
  // design editor's collection picker need the full list (including retired
  // ones), so pull the largest page the API allows and unwrap it.
  const result = await request<PagedResult<Collection>>(
    `/api/v1/admin/collections?page=1&pageSize=${MAX_PAGE_SIZE}`,
  );
  return result.items;
}

export function listCollectionsPaged(params: PageParams): Promise<PagedResult<Collection>> {
  const search = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
  });

  return request<PagedResult<Collection>>(`/api/v1/admin/collections?${search.toString()}`);
}

export function createCollection(input: CollectionInput): Promise<Collection> {
  return request<Collection>("/api/v1/admin/collections", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateCollection(id: string, input: CollectionInput): Promise<void> {
  return request<void>(`/api/v1/admin/collections/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function retireCollection(id: string): Promise<void> {
  return request<void>(`/api/v1/admin/collections/${id}/retire`, { method: "POST" });
}

export function restoreCollection(id: string): Promise<void> {
  return request<void>(`/api/v1/admin/collections/${id}/restore`, { method: "POST" });
}

/** PERMANENT: also deletes every design in the collection. */
export function deleteCollection(id: string): Promise<void> {
  return request<void>(`/api/v1/admin/collections/${id}`, { method: "DELETE" });
}

// --- Designs ---

export interface DesignListParams {
  collectionId?: string;
  q?: string;
}

export async function listDesigns(params: DesignListParams = {}): Promise<Design[]> {
  // The admin designs endpoint is paginated; this unpaged helper (used by the
  // editor's "find by id" lookup) pulls the largest page and unwraps it.
  const search = new URLSearchParams({ page: "1", pageSize: String(MAX_PAGE_SIZE) });
  if (params.collectionId) search.set("collection", params.collectionId);
  if (params.q) search.set("q", params.q);
  const result = await request<PagedResult<Design>>(`/api/v1/admin/designs?${search.toString()}`);
  return result.items;
}

export function listDesignsPaged(
  params: DesignListParams & PageParams,
): Promise<PagedResult<Design>> {
  const search = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
  });
  if (params.collectionId) search.set("collection", params.collectionId);
  if (params.q) search.set("q", params.q);

  return request<PagedResult<Design>>(`/api/v1/admin/designs?${search.toString()}`);
}

export function createDesign(input: DesignInput): Promise<Design> {
  return request<Design>("/api/v1/admin/designs", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function updateDesign(id: string, input: DesignInput): Promise<Design> {
  return request<Design>(`/api/v1/admin/designs/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function retireDesigns(ids: string[]): Promise<void> {
  return request<void>("/api/v1/admin/designs/retire", {
    method: "POST",
    body: JSON.stringify({ ids }),
  });
}

export function restoreDesigns(ids: string[]): Promise<void> {
  return request<void>("/api/v1/admin/designs/restore", {
    method: "POST",
    body: JSON.stringify({ ids }),
  });
}

/** PERMANENT. */
export function deleteDesign(id: string): Promise<void> {
  return request<void>(`/api/v1/admin/designs/${id}`, { method: "DELETE" });
}

// --- Uploads (Cloudinary) ---

export interface UploadSignature {
  cloudName: string;
  apiKey: string;
  timestamp: number;
  folder: string;
  signature: string;
}

export function signUpload(): Promise<UploadSignature> {
  return request<UploadSignature>("/api/v1/admin/uploads/sign", {
    method: "POST",
    body: JSON.stringify({}),
  });
}

/** Returns null when Cloudinary is not configured on the server (503 not_configured). */
export async function getUploadSignature(): Promise<UploadSignature | null> {
  try {
    return await signUpload();
  } catch (error) {
    if (error instanceof ApiError && error.code === "not_configured") return null;
    throw error;
  }
}

export function cloudinaryPreviewUrl(cloudName: string, publicId: string): string {
  return `https://res.cloudinary.com/${cloudName}/image/upload/c_fill,w_200,h_260/${publicId}`;
}

/**
 * Direct multipart upload to Cloudinary using a server-issued signature.
 * Uses XMLHttpRequest so we can report upload progress.
 * Resolves with the Cloudinary public_id.
 */
export function uploadToCloudinary(
  file: File,
  signature: UploadSignature,
  onProgress: (fraction: number) => void,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const form = new FormData();
    form.append("file", file);
    form.append("api_key", signature.apiKey);
    form.append("timestamp", String(signature.timestamp));
    form.append("folder", signature.folder);
    form.append("signature", signature.signature);

    const xhr = new XMLHttpRequest();
    xhr.open(
      "POST",
      `https://api.cloudinary.com/v1_1/${signature.cloudName}/image/upload`,
    );
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) onProgress(event.loaded / event.total);
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const body = JSON.parse(xhr.responseText) as { public_id: string };
          resolve(body.public_id);
        } catch {
          reject(new Error("Unexpected response from the image service."));
        }
      } else {
        reject(new Error(`Image upload failed (status ${xhr.status}).`));
      }
    };
    xhr.onerror = () => {
      reject(new Error("Image upload failed. Check your connection and try again."));
    };
    xhr.send(form);
  });
}
