import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { DEFAULT_PAGE_SIZE, deleteSubscriber, listSubscribers, type PageParams } from "./api";

const subscribersKey = ["admin", "subscribers"] as const;

/**
 * Fetches one page of newsletter subscribers. Previous page data is kept while
 * the next page loads so the table doesn't flash empty when paging.
 */
export function useSubscribers(params: PageParams) {
  return useQuery({
    queryKey: [...subscribersKey, params.page, params.pageSize],
    queryFn: () => listSubscribers(params),
    placeholderData: keepPreviousData,
  });
}

export function useDeleteSubscriber() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteSubscriber(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: subscribersKey });
    },
  });
}

export { DEFAULT_PAGE_SIZE };
