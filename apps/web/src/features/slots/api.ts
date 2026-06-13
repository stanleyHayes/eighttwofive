import { ApiError, request } from "@/lib/api";

export interface Slot {
  id: string;
  start: string;
  end: string;
  status: "open" | "booked" | "closed";
  createdAt: string;
  updatedAt: string;
}

export interface Visit {
  id: string;
  orderId: string;
  slotId: string;
  depositPaymentId: string;
  status: "booked" | "done" | "cancelled";
  createdAt: string;
  updatedAt: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: "customer" | "admin";
}

export interface BookSlotInput {
  designId?: string;
  email: string;
  name: string;
  phone: string;
}

export interface BookSlotResult {
  visit: Visit;
  order: {
    id: string;
    ref: string;
    type: string;
    status: string;
    customerPhone: string;
    createdAt: string;
    updatedAt: string;
  };
  paymentUrl: string;
  user: User;
}

export interface CreateSlotInput {
  start: string;
  end: string;
}

export interface RescheduleVisitInput {
  newSlotId: string;
}

export function getOpenSlots(): Promise<Slot[]> {
  return request<Slot[]>("/api/v1/slots");
}

export function bookSlot(slotId: string, input: BookSlotInput): Promise<BookSlotResult> {
  return request<BookSlotResult>(`/api/v1/slots/${slotId}/book`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function getAdminSlots(): Promise<Slot[]> {
  return request<Slot[]>("/api/v1/admin/slots");
}

export function createSlot(input: CreateSlotInput): Promise<Slot> {
  return request<Slot>("/api/v1/admin/slots", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function closeSlot(slotId: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/v1/admin/slots/${slotId}/close`, {
    method: "POST",
  });
}

export function reopenSlot(slotId: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/api/v1/admin/slots/${slotId}/reopen`, {
    method: "POST",
  });
}

export function getAdminVisits(): Promise<Visit[]> {
  return request<Visit[]>("/api/v1/admin/visits");
}

export function rescheduleVisit(visitId: string, input: RescheduleVisitInput): Promise<Visit> {
  return request<Visit>(`/api/v1/admin/visits/${visitId}/reschedule`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function cancelVisit(visitId: string): Promise<Visit> {
  return request<Visit>(`/api/v1/admin/visits/${visitId}/cancel`, {
    method: "POST",
  });
}

export function errorMessage(
  error: unknown,
  fallback = "Something went wrong. Try again in a moment.",
): string {
  return error instanceof ApiError ? error.message : fallback;
}
