import { useMutation } from "@tanstack/react-query";
import { joinWaitlist } from "@/lib/api";

export function useJoinWaitlist() {
  return useMutation({ mutationFn: joinWaitlist });
}
