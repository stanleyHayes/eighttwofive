import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  createCollection,
  createDesign,
  deleteCollection,
  deleteDesign,
  getUploadSignature,
  listCollections,
  listCollectionsPaged,
  listDesigns,
  listDesignsPaged,
  restoreCollection,
  restoreDesigns,
  retireCollection,
  retireDesigns,
  updateCollection,
  updateDesign,
  type CollectionInput,
  type DesignInput,
  type DesignListParams,
  type PageParams,
} from "./api";

const collectionsKey = ["admin", "collections"] as const;
const designsKey = ["admin", "designs"] as const;

export function useCollections() {
  return useQuery({ queryKey: collectionsKey, queryFn: listCollections });
}

/** Paginated collections for the admin table. */
export function useCollectionsPaged(params: PageParams) {
  return useQuery({
    queryKey: [...collectionsKey, "paged", params.page, params.pageSize],
    queryFn: () => listCollectionsPaged(params),
    placeholderData: keepPreviousData,
  });
}

export function useDesigns(params: DesignListParams = {}, opts: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: [...designsKey, params.collectionId ?? "", params.q ?? ""],
    queryFn: () => listDesigns(params),
    enabled: opts.enabled ?? true,
  });
}

/** Paginated designs for the admin table, with filters preserved. */
export function useDesignsPaged(params: DesignListParams & PageParams) {
  return useQuery({
    queryKey: [
      ...designsKey,
      "paged",
      params.collectionId ?? "",
      params.q ?? "",
      params.page,
      params.pageSize,
    ],
    queryFn: () => listDesignsPaged(params),
    placeholderData: keepPreviousData,
  });
}

/**
 * Cloudinary config probe; data is null when uploads aren't configured.
 * Used for rendering previews (cloudName) and for the inline notice.
 */
export function useUploadConfig() {
  return useQuery({
    queryKey: ["admin", "uploadConfig"],
    queryFn: getUploadSignature,
    staleTime: Infinity,
    retry: false,
  });
}

function useInvalidateCatalog() {
  const queryClient = useQueryClient();
  return {
    collections: () => {
      void queryClient.invalidateQueries({ queryKey: collectionsKey });
    },
    designs: () => {
      void queryClient.invalidateQueries({ queryKey: designsKey });
    },
  };
}

// --- Collection mutations ---

export function useCreateCollection() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: createCollection,
    onSuccess: () => invalidate.collections(),
  });
}

export function useUpdateCollection() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: CollectionInput }) =>
      updateCollection(id, input),
    onSuccess: () => {
      invalidate.collections();
      invalidate.designs();
    },
  });
}

export function useRetireCollection() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: retireCollection,
    // Cascades to the collection's designs.
    onSuccess: () => {
      invalidate.collections();
      invalidate.designs();
    },
  });
}

export function useRestoreCollection() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: restoreCollection,
    onSuccess: () => {
      invalidate.collections();
      invalidate.designs();
    },
  });
}

export function useDeleteCollection() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: deleteCollection,
    // PERMANENT; also deletes the collection's designs.
    onSuccess: () => {
      invalidate.collections();
      invalidate.designs();
    },
  });
}

// --- Design mutations ---

export function useCreateDesign() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: createDesign,
    onSuccess: () => invalidate.designs(),
  });
}

export function useUpdateDesign() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: DesignInput }) => updateDesign(id, input),
    onSuccess: () => invalidate.designs(),
  });
}

export function useRetireDesigns() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: retireDesigns,
    onSuccess: () => invalidate.designs(),
  });
}

export function useRestoreDesigns() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: restoreDesigns,
    onSuccess: () => invalidate.designs(),
  });
}

export function useDeleteDesign() {
  const invalidate = useInvalidateCatalog();
  return useMutation({
    mutationFn: deleteDesign,
    onSuccess: () => invalidate.designs(),
  });
}
