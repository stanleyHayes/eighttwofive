import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  createCustomRequest,
  createStandardOrder,
  getAdminOrder,
  getOrder,
  listAdminOrders,
  listAdminOrdersPaged,
  listOrders,
  markPaidManually,
  sendPaymentLink,
  updateOrderStatus,
  updateQuote,
  type CreateCustomRequestInput,
  type CreateStandardOrderInput,
  type MarkPaidInput,
  type Order,
  type OrderStatus,
  type PageParams,
  type QuoteInput,
} from "./api";

const ordersKey = ["admin", "orders"] as const;
const customerOrdersKey = ["orders"] as const;

export function useOrders() {
  return useQuery({
    queryKey: customerOrdersKey,
    queryFn: listOrders,
  });
}

export function useOrder(ref: string) {
  return useQuery({
    queryKey: [...customerOrdersKey, ref],
    queryFn: () => getOrder(ref),
    enabled: ref !== "",
  });
}

export function useAdminOrders() {
  return useQuery({
    queryKey: ordersKey,
    queryFn: listAdminOrders,
  });
}

/** Paginated admin orders for the inbox table. */
export function useAdminOrdersPaged(params: PageParams) {
  return useQuery({
    queryKey: [...ordersKey, "paged", params.page, params.pageSize],
    queryFn: () => listAdminOrdersPaged(params),
    placeholderData: keepPreviousData,
  });
}

export function useAdminOrder(ref: string | null) {
  return useQuery({
    queryKey: [...ordersKey, ref],
    queryFn: () => getAdminOrder(ref!),
    enabled: ref !== null && ref !== "",
  });
}

export function useCreateStandardOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateStandardOrderInput) => createStandardOrder(input),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: customerOrdersKey });
      void queryClient.invalidateQueries({ queryKey: [...customerOrdersKey, result.order.ref] });
    },
  });
}

export function useCreateCustomRequest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateCustomRequestInput) => createCustomRequest(input),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: customerOrdersKey });
      void queryClient.invalidateQueries({ queryKey: [...customerOrdersKey, result.order.ref] });
    },
  });
}

export function useUpdateQuote(ref: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: QuoteInput) => updateQuote(ref!, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ordersKey });
      if (ref) {
        void queryClient.invalidateQueries({ queryKey: [...ordersKey, ref] });
      }
    },
  });
}

export function useSendPaymentLink(ref: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => sendPaymentLink(ref!),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ordersKey });
      if (ref) {
        void queryClient.invalidateQueries({ queryKey: [...ordersKey, ref] });
      }
    },
  });
}

export function useMarkPaidManually(ref: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: MarkPaidInput) => markPaidManually(ref!, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ordersKey });
      if (ref) {
        void queryClient.invalidateQueries({ queryKey: [...ordersKey, ref] });
      }
    },
  });
}

export function useUpdateOrderStatus(ref: string | null) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (status: OrderStatus) => updateOrderStatus(ref!, { status }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ordersKey });
      if (ref) {
        void queryClient.invalidateQueries({ queryKey: [...ordersKey, ref] });
      }
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

export function whatsappLink(phone: string, ref: string): string {
  const normalized = phone.replace(/\D/g, "");
  const text = encodeURIComponent(`Order ${ref}`);

  return `https://wa.me/${normalized}?text=${text}`;
}

export function orderTypeLabel(type: Order["type"]): string {
  switch (type) {
    case "standard":
      return "Standard";
    case "custom_size":
      return "Custom size";
    case "design_change":
      return "Design change";
    case "visit":
      return "Visit booking";
    default:
      return type;
  }
}

export function bucketOrders(orders: Order[]): {
  standard: Order[];
  custom: Order[];
  visits: Order[];
} {
  return {
    standard: orders.filter((o) => o.type === "standard"),
    custom: orders.filter((o) => o.type === "custom_size" || o.type === "design_change"),
    visits: orders.filter((o) => o.type === "visit"),
  };
}
