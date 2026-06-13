import { useQuery } from "@tanstack/react-query";
import { getStoreAnalytics } from "./api";

const analyticsKey = ["admin", "analytics"] as const;

export function useStoreAnalytics() {
  return useQuery({
    queryKey: analyticsKey,
    queryFn: getStoreAnalytics,
  });
}
