import { ApiError, request } from "@/lib/api";

export type OrderType = "standard" | "custom_size" | "design_change" | "visit";
export type OrderStatus =
  | "pending_payment"
  | "requested"
  | "quoted"
  | "payment_link_sent"
  | "booked"
  | "in_production"
  | "ready"
  | "fulfilled"
  | "cancelled";

export interface DesignSnapshot {
  name: string;
  photoPublicId: string;
  pricePesewas: number;
}

export interface Customisation {
  sizeMode: string;
  bandLabel?: string;
  measurements?: Record<string, string>;
  designChange?: string;
}

export interface Quote {
  pricePesewas: number;
  timeline: string;
  notes: string;
}

export interface Delivery {
  mode: string;
  area?: string;
  ratePesewas?: number;
}

export interface Payment {
  providerRef: string;
  amountPesewas: number;
  status: string;
  method: string;
  paidAt?: string;
}

export interface StatusChange {
  status: OrderStatus;
  at: string;
  by: string;
}

export interface Order {
  id: string;
  ref: string;
  customerId: string;
  designId: string;
  designSnapshot: DesignSnapshot;
  type: OrderType;
  customisation: Customisation;
  quote: Quote;
  delivery: Delivery;
  payments: Payment[];
  status: OrderStatus;
  statusHistory: StatusChange[];
  customerPhone: string;
  totalPesewas: number;
  createdAt: string;
  updatedAt: string;
}

export function errorMessage(
  error: unknown,
  fallback = "Something went wrong. Try again in a moment.",
): string {
  return error instanceof ApiError ? error.message : fallback;
}

export function customerStageLabel(status: OrderStatus): string {
  switch (status) {
    case "pending_payment":
      return "awaiting payment";
    case "requested":
      return "request received";
    case "quoted":
      return "quote ready";
    case "payment_link_sent":
      return "payment link sent";
    case "booked":
      return "order confirmed";
    case "in_production":
      return "in production";
    case "ready":
      return "ready";
    case "fulfilled":
      return "ready";
    case "cancelled":
      return "cancelled";
    default:
      return status;
  }
}

export function effectivePricePesewas(order: Order): number {
  return order.quote.pricePesewas > 0
    ? order.quote.pricePesewas
    : order.designSnapshot.pricePesewas;
}

export function hasVisit(order: Order): boolean {
  return order.type === "visit" || order.customisation.sizeMode === "home_visit";
}

export function paymentStatus(order: Order): string {
  const successful = order.payments.some((payment) => payment.status === "success");
  if (successful) return "Paid";

  const pending = order.payments.some((payment) => payment.status === "pending");
  if (pending) return "Payment pending";

  return "Unpaid";
}

/**
 * Returns a payment whose charged amount didn't match the order total. The
 * webhook never books these, so the admin needs to see it and reconcile the
 * transaction in Paystack by reference.
 */
export function mismatchedPayment(order: Order): Payment | undefined {
  return order.payments.find((payment) => payment.status === "mismatch");
}

export function listOrders(): Promise<Order[]> {
  return request<Order[]>("/api/v1/orders");
}

export function getOrder(ref: string): Promise<Order> {
  return request<Order>(`/api/v1/orders/${ref}`);
}

export interface CreateStandardOrderInput {
  designId: string;
  bandLabel: string;
  delivery: string;
  customerPhone: string;
  email: string;
  name: string;
}

export interface CreateStandardOrderResponse {
  order: Order;
  paymentUrl: string;
}

export function createStandardOrder(input: CreateStandardOrderInput): Promise<CreateStandardOrderResponse> {
  return request<CreateStandardOrderResponse>("/api/v1/orders", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function listAdminOrders(): Promise<Order[]> {
  // The admin orders endpoint is paginated; this unpaged helper pulls the
  // largest page and unwraps it so callers still get a plain array.
  const result = await listAdminOrdersPaged({ page: 1, pageSize: 100 });
  return result.items;
}

/** One page of a paginated admin listing, mirroring the API envelope. */
export interface PagedResult<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface PageParams {
  page: number;
  pageSize: number;
}

export const DEFAULT_PAGE_SIZE = 20;

export function listAdminOrdersPaged(params: PageParams): Promise<PagedResult<Order>> {
  const search = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
  });

  return request<PagedResult<Order>>(`/api/v1/admin/orders?${search.toString()}`);
}

export function getAdminOrder(ref: string): Promise<Order> {
  return request<Order>(`/api/v1/admin/orders/${ref}`);
}

export interface QuoteInput {
  pricePesewas: number;
  timeline: string;
  notes: string;
}

export function updateQuote(ref: string, input: QuoteInput): Promise<void> {
  return request<void>(`/api/v1/admin/orders/${ref}/quote`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export interface PaymentLinkResponse {
  paymentUrl: string;
}

export function sendPaymentLink(ref: string): Promise<PaymentLinkResponse> {
  return request<PaymentLinkResponse>(`/api/v1/admin/orders/${ref}/payment-link`, {
    method: "POST",
  });
}

export interface CreateCustomRequestInput {
  designId: string;
  sizeMode: string;
  measurements?: Record<string, string>;
  bandLabel?: string;
  designChange?: string;
  delivery: string;
  customerPhone: string;
  email: string;
  name: string;
}

export interface CreateCustomRequestResponse {
  order: Order;
}

export function createCustomRequest(input: CreateCustomRequestInput): Promise<CreateCustomRequestResponse> {
  return request<CreateCustomRequestResponse>("/api/v1/orders/request", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export interface MarkPaidInput {
  note: string;
}

export function markPaidManually(ref: string, input: MarkPaidInput): Promise<void> {
  return request<void>(`/api/v1/admin/orders/${ref}/mark-paid`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export interface UpdateStatusInput {
  status: OrderStatus;
}

export function updateOrderStatus(ref: string, input: UpdateStatusInput): Promise<void> {
  return request<void>(`/api/v1/admin/orders/${ref}/status`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}
