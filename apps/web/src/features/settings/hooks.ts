import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getSettings, updateSettings, type Settings, type SettingsInput } from "./api";

const settingsKey = ["admin", "settings"] as const;

export function useSettings() {
  return useQuery({
    queryKey: settingsKey,
    queryFn: getSettings,
  });
}

export function useUpdateSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: SettingsInput) => updateSettings(input),
    onSuccess: (data) => {
      queryClient.setQueryData(settingsKey, data);
    },
  });
}

export function formatPesewas(pesewas: number): string {
  return `GH₵ ${(pesewas / 100).toFixed(2)}`;
}

export function parseGhs(value: string): number | null {
  const trimmed = value.trim();
  if (trimmed === "") return null;

  const parsed = Number.parseFloat(trimmed);
  if (Number.isNaN(parsed)) return null;

  return Math.round(parsed * 100);
}

export function validateSettings(settings: Settings): string | null {
  if (settings.depositPesewas < 0) {
    return "Deposit amount cannot be negative.";
  }

  const seen = new Set<string>();
  for (const rate of settings.deliveryRates) {
    const area = rate.area.trim();
    if (area === "") {
      return "Delivery area cannot be empty.";
    }
    if (rate.ratePesewas < 0) {
      return `Delivery rate for ${area} cannot be negative.`;
    }
    if (seen.has(area.toLowerCase())) {
      return `Area "${area}" is listed more than once.`;
    }
    seen.add(area.toLowerCase());
  }

  return null;
}
