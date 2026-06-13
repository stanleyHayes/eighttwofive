import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getOpenSlots,
  bookSlot,
  getAdminSlots,
  createSlot,
  closeSlot,
  reopenSlot,
  getAdminVisits,
  rescheduleVisit,
  cancelVisit,
  type BookSlotInput,
  type CreateSlotInput,
  type RescheduleVisitInput,
} from "./api";

const slotsKey = ["slots"] as const;
const adminSlotsKey = ["admin", "slots"] as const;
const adminVisitsKey = ["admin", "visits"] as const;

export function useOpenSlots() {
  return useQuery({
    queryKey: slotsKey,
    queryFn: getOpenSlots,
  });
}

export function useBookSlot() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ slotId, input }: { slotId: string; input: BookSlotInput }) => bookSlot(slotId, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: slotsKey });
    },
  });
}

export function useAdminSlots() {
  return useQuery({
    queryKey: adminSlotsKey,
    queryFn: getAdminSlots,
  });
}

export function useCreateSlot() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateSlotInput) => createSlot(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminSlotsKey });
    },
  });
}

export function useCloseSlot() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (slotId: string) => closeSlot(slotId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminSlotsKey });
    },
  });
}

export function useReopenSlot() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (slotId: string) => reopenSlot(slotId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminSlotsKey });
    },
  });
}

export function useAdminVisits() {
  return useQuery({
    queryKey: adminVisitsKey,
    queryFn: getAdminVisits,
  });
}

export function useRescheduleVisit() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ visitId, input }: { visitId: string; input: RescheduleVisitInput }) =>
      rescheduleVisit(visitId, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminVisitsKey });
      void queryClient.invalidateQueries({ queryKey: adminSlotsKey });
    },
  });
}

export function useCancelVisit() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (visitId: string) => cancelVisit(visitId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminVisitsKey });
      void queryClient.invalidateQueries({ queryKey: adminSlotsKey });
    },
  });
}

export function formatSlotTime(start: string, end: string): string {
  const startDate = new Date(start);
  const endDate = new Date(end);

  const date = startDate.toLocaleDateString("en-GB", {
    year: "numeric",
    month: "short",
    day: "2-digit",
  });

  const startTime = startDate.toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
  });

  const endTime = endDate.toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
  });

  return `${date}, ${startTime}–${endTime}`;
}
