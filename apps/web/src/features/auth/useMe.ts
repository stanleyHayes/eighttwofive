import { useQuery } from "@tanstack/react-query";
import { getMe } from "@/lib/api";

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: getMe,
    staleTime: 60_000,
  });
}
