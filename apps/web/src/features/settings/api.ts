import { ApiError, request } from "@/lib/api";

export interface DeliveryRate {
  area: string;
  ratePesewas: number;
}

export interface Settings {
  depositPesewas: number;
  whatsappNumber: string;
  visitLocation: string;
  instagramHandle: string;
  deliveryRates: DeliveryRate[];
}

export type SettingsInput = Settings;

export function getSettings(): Promise<Settings> {
  return request<Settings>("/api/v1/settings");
}

export function updateSettings(input: SettingsInput): Promise<Settings> {
  return request<Settings>("/api/v1/admin/settings", {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export function errorMessage(
  error: unknown,
  fallback = "Something went wrong. Try again in a moment.",
): string {
  return error instanceof ApiError ? error.message : fallback;
}
