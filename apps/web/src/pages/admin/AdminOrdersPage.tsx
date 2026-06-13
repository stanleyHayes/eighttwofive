import { useMemo, useState, type ChangeEvent, type FormEvent } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Pagination from "@mui/material/Pagination";
import Paper from "@mui/material/Paper";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import ReceiptLongOutlined from "@mui/icons-material/ReceiptLongOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { clayDeep } from "@/theme";
import {
  DEFAULT_PAGE_SIZE,
  errorMessage,
  type Order,
  type OrderStatus,
} from "@/features/orders/api";
import {
  bucketOrders,
  formatPesewas,
  orderTypeLabel,
  parseGhs,
  useAdminOrdersPaged,
  useAdminOrder,
  useMarkPaidManually,
  useSendPaymentLink,
  useUpdateOrderStatus,
  useUpdateQuote,
  whatsappLink,
} from "@/features/orders/hooks";
import { useCan } from "@/features/auth/permissions";
import { hideUntilMd, tableMinWidth } from "@/components/tableResponsive";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function StatusBadge({ status }: { status: Order["status"] }) {
  const color: Record<Order["status"], "default" | "primary" | "success" | "error" | "warning"> = {
    pending_payment: "warning",
    requested: "default",
    quoted: "primary",
    payment_link_sent: "primary",
    booked: "success",
    in_production: "primary",
    ready: "success",
    fulfilled: "success",
    cancelled: "error",
  };

  return <Chip size="small" label={status.replace(/_/g, " ")} color={color[status] ?? "default"} />;
}

function TypeBadge({ type }: { type: Order["type"] }) {
  return (
    <Typography
      component="span"
      sx={{
        display: "inline-block",
        px: 1,
        py: 0.5,
        bgcolor: "rgba(22, 20, 18, 0.08)",
        fontSize: "0.75rem",
        textTransform: "uppercase",
        letterSpacing: "0.08em",
      }}
    >
      {orderTypeLabel(type)}
    </Typography>
  );
}

const TRANSITIONS: OrderStatus[] = [
  "pending_payment",
  "requested",
  "quoted",
  "payment_link_sent",
  "booked",
  "in_production",
  "ready",
  "fulfilled",
  "cancelled",
];

// The theme gives buttons a tall (paddingBlock: 16) "atelier" height; inside the
// dense action rows we override that so each button matches the 40px small field
// beside it. height + box-sizing keeps the total at 40px regardless of label.
const compactActionSx = {
  flexShrink: 0,
  whiteSpace: "nowrap",
  width: { xs: "100%", sm: "auto" },
  minHeight: 0,
  height: 40,
  py: 0,
  px: 2.5,
} as const;

interface OrderDetailPanelProps {
  order: Order;
  onClose: () => void;
}

function OrderDetailPanel({ order, onClose }: OrderDetailPanelProps) {
  const canWrite = useCan("orders:write");
  const updateQuote = useUpdateQuote(order.ref);
  const sendLink = useSendPaymentLink(order.ref);
  const markPaid = useMarkPaidManually(order.ref);
  const updateStatus = useUpdateOrderStatus(order.ref);

  const [priceGhs, setPriceGhs] = useState(() =>
    order.quote.pricePesewas > 0 ? (order.quote.pricePesewas / 100).toFixed(2) : "",
  );
  const [timeline, setTimeline] = useState(order.quote.timeline);
  const [quoteNotes, setQuoteNotes] = useState(order.quote.notes);
  const [manualNote, setManualNote] = useState("");
  const [selectedStatus, setSelectedStatus] = useState<OrderStatus>(order.status);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  const isCustomRequest = order.type === "custom_size" || order.type === "design_change";
  const canEditQuote = isCustomRequest && (order.status === "requested" || order.status === "quoted");

  const showSuccess = (message: string) => {
    setActionSuccess(message);
    setActionError(null);
  };

  const showError = (message: string) => {
    setActionError(message);
    setActionSuccess(null);
  };

  const handleSaveQuote = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setActionError(null);
    setActionSuccess(null);

    const pricePesewas = parseGhs(priceGhs);
    if (pricePesewas === null || pricePesewas < 0) {
      showError("Enter a valid price in GHS.");
      return;
    }

    updateQuote.mutate(
      { pricePesewas, timeline: timeline.trim(), notes: quoteNotes.trim() },
      {
        onSuccess: () => showSuccess("Quote saved."),
        onError: (err) => showError(errorMessage(err)),
      },
    );
  };

  const handleSendPaymentLink = () => {
    setActionError(null);
    setActionSuccess(null);

    sendLink.mutate(undefined, {
      onSuccess: (data) => showSuccess(`Payment link created: ${data.paymentUrl}`),
      onError: (err) => showError(errorMessage(err)),
    });
  };

  const handleMarkPaid = () => {
    setActionError(null);
    setActionSuccess(null);

    markPaid.mutate(
      { note: manualNote.trim() || "Manual payment" },
      {
        onSuccess: () => {
          showSuccess("Order marked as paid.");
          setManualNote("");
        },
        onError: (err) => showError(errorMessage(err)),
      },
    );
  };

  const handleStatusChange = () => {
    setActionError(null);
    setActionSuccess(null);

    updateStatus.mutate(selectedStatus, {
      onSuccess: () => showSuccess(`Status updated to ${selectedStatus.replace(/_/g, " ")}.`),
      onError: (err) => showError(errorMessage(err)),
    });
  };

  return (
    <Paper elevation={0} sx={{ p: { xs: 2, md: 3 }, border: "1px solid", borderColor: "divider" }}>
      <Stack spacing={4}>
        <Stack direction="row" spacing={2} sx={{ justifyContent: "space-between", alignItems: "flex-start" }}>
          <Box>
            <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
              {order.ref}
            </Typography>
            <Typography variant="h2" component="h2" sx={{ mt: 0.5 }}>
              {order.designSnapshot.name}
            </Typography>
            <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
              <TypeBadge type={order.type} />
              <StatusBadge status={order.status} />
            </Stack>
          </Box>
          <Button variant="text" size="small" onClick={onClose}>
            Close
          </Button>
        </Stack>

        <Stack spacing={1}>
          <Typography variant="body2" sx={{ color: "text.secondary" }}>
            Customer phone
          </Typography>
          <Typography>{order.customerPhone}</Typography>
          <Box>
            <Button
              variant="outlined"
              size="small"
              href={whatsappLink(order.customerPhone, order.ref)}
              target="_blank"
              rel="noopener noreferrer"
            >
              Open WhatsApp
            </Button>
          </Box>
        </Stack>

        <Divider />

        <Stack spacing={2}>
          <Typography variant="h5" component="h3">
            Measurements & choices
          </Typography>
          <Typography>
            <strong>Size mode:</strong> {order.customisation.sizeMode}
          </Typography>
          {order.customisation.bandLabel && (
            <Typography>
              <strong>Band:</strong> {order.customisation.bandLabel}
            </Typography>
          )}
          {order.customisation.measurements &&
            Object.entries(order.customisation.measurements).length > 0 && (
              <Box>
                <Typography sx={{ mb: 0.5 }}>
                  <strong>Measurements:</strong>
                </Typography>
                <Stack component="ul" spacing={0.5} sx={{ pl: 2, m: 0 }}>
                  {Object.entries(order.customisation.measurements).map(([key, value]) => (
                    <Box component="li" key={key}>
                      <Typography variant="body2">
                        {key}: {value}
                      </Typography>
                    </Box>
                  ))}
                </Stack>
              </Box>
            )}
          {order.customisation.designChange && (
            <Typography>
              <strong>Design change:</strong> {order.customisation.designChange}
            </Typography>
          )}
          <Typography>
            <strong>Delivery:</strong> {order.delivery.mode}
            {order.delivery.area ? ` — ${order.delivery.area}` : ""}
            {order.delivery.ratePesewas !== undefined
              ? ` (${formatPesewas(order.delivery.ratePesewas)})`
              : ""}
          </Typography>
        </Stack>

        <Divider />

        {canWrite && canEditQuote && (
          <Stack component="form" spacing={2} onSubmit={handleSaveQuote}>
            <Typography variant="h5" component="h3">
              Quote
            </Typography>
            <TextField
              label="Price (GHS)"
              type="number"
              slotProps={{ htmlInput: { min: 0, step: "0.01" } }}
              value={priceGhs}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setPriceGhs(event.target.value)}
              fullWidth
            />
            <TextField
              label="Timeline"
              value={timeline}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setTimeline(event.target.value)}
              placeholder="e.g. 2 weeks"
              fullWidth
            />
            <TextField
              label="Notes"
              value={quoteNotes}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setQuoteNotes(event.target.value)}
              multiline
              minRows={2}
              fullWidth
            />
            <Box>
              <Button type="submit" variant="contained" loading={updateQuote.isPending}>
                Save quote
              </Button>
            </Box>
          </Stack>
        )}

        {canWrite && canEditQuote && <Divider />}

        <Stack spacing={2}>
          <Typography variant="h5" component="h3">
            Payment
          </Typography>
          <Typography>
            <strong>Total:</strong> {formatPesewas(order.totalPesewas)}
          </Typography>
          {canWrite && (
            <>
              <Box>
                <Button
                  variant="outlined"
                  size="small"
                  onClick={handleSendPaymentLink}
                  loading={sendLink.isPending}
                  disabled={!isCustomRequest || order.status === "payment_link_sent"}
                  sx={compactActionSx}
                >
                  Send payment link
                </Button>
              </Box>
              <Stack
                direction={{ xs: "column", sm: "row" }}
                spacing={1.5}
                sx={{ alignItems: { sm: "center" } }}
              >
                <TextField
                  label="Manual payment note"
                  value={manualNote}
                  onChange={(event: ChangeEvent<HTMLInputElement>) => setManualNote(event.target.value)}
                  placeholder="e.g. Cash on pickup"
                  size="small"
                  sx={{ flex: 1, minWidth: { sm: 200 } }}
                />
                <Button
                  variant="contained"
                  size="small"
                  onClick={handleMarkPaid}
                  loading={markPaid.isPending}
                  sx={compactActionSx}
                >
                  Mark paid manually
                </Button>
              </Stack>
            </>
          )}
        </Stack>

        {canWrite && (
          <>
            <Divider />

            <Stack spacing={2}>
              <Typography variant="h5" component="h3">
                Status
              </Typography>
              <Stack
                direction={{ xs: "column", sm: "row" }}
                spacing={1.5}
                sx={{ alignItems: { sm: "center" } }}
              >
                <TextField
                  select
                  label="New status"
                  value={selectedStatus}
                  onChange={(event: ChangeEvent<HTMLInputElement>) =>
                    setSelectedStatus(event.target.value as OrderStatus)
                  }
                  slotProps={{
                    select: { native: true },
                  }}
                  size="small"
                  sx={{ flex: 1, minWidth: { sm: 200 } }}
                >
                  {TRANSITIONS.map((status) => (
                    <option key={status} value={status}>
                      {status.replace(/_/g, " ")}
                    </option>
                  ))}
                </TextField>
                <Button
                  variant="outlined"
                  size="small"
                  onClick={handleStatusChange}
                  loading={updateStatus.isPending}
                  sx={compactActionSx}
                >
                  Update status
                </Button>
              </Stack>
            </Stack>
          </>
        )}

        {actionError && <Alert severity="error">{actionError}</Alert>}
        {actionSuccess && <Alert severity="success">{actionSuccess}</Alert>}

        <Divider />

        <Stack spacing={1}>
          <Typography variant="body2" sx={{ color: "text.secondary" }}>
            History
          </Typography>
          <Stack component="ul" spacing={0.5} sx={{ pl: 2, m: 0 }}>
            {order.statusHistory.map((change) => (
              <Box component="li" key={`${change.status}-${change.at}`}>
                <Typography variant="body2">
                  {change.status.replace(/_/g, " ")} — {formatDate(change.at)} by {change.by || "system"}
                </Typography>
              </Box>
            ))}
          </Stack>
        </Stack>
      </Stack>
    </Paper>
  );
}

interface OrderTableProps {
  title: string;
  orders: Order[];
  selectedRef: string | null;
  onSelect: (ref: string) => void;
}

function OrderTable({ title, orders, selectedRef, onSelect }: OrderTableProps) {
  return (
    <Box>
      <Typography variant="overline" component="h3" sx={{ color: clayDeep, mb: 1 }}>
        {title}
      </Typography>
      {orders.length === 0 ? (
        <EmptyState
          title="Nothing in this bucket yet."
          body="Orders land here as customers place them. Nothing to action right now."
          sx={{ maxWidth: "none" }}
        />
      ) : (
        <Box sx={{ overflowX: "auto" }}>
          <Table size="small" sx={tableMinWidth(560)}>
            <TableHead>
              <TableRow>
                <TableCell>Ref</TableCell>
                <TableCell sx={hideUntilMd}>Type</TableCell>
                <TableCell sx={hideUntilMd}>Customer</TableCell>
                <TableCell>Design</TableCell>
                <TableCell>Status</TableCell>
                <TableCell sx={hideUntilMd}>Created</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {orders.map((order) => (
                <TableRow
                  key={order.ref}
                  hover
                  selected={order.ref === selectedRef}
                  onClick={() => onSelect(order.ref)}
                  sx={{ cursor: "pointer" }}
                >
                  <TableCell sx={{ fontWeight: 500 }}>{order.ref}</TableCell>
                  <TableCell sx={hideUntilMd}>
                    <TypeBadge type={order.type} />
                  </TableCell>
                  <TableCell sx={hideUntilMd}>{order.customerPhone}</TableCell>
                  <TableCell>{order.designSnapshot.name}</TableCell>
                  <TableCell>
                    <StatusBadge status={order.status} />
                  </TableCell>
                  <TableCell sx={{ color: "text.secondary", ...hideUntilMd }}>
                    {formatDate(order.createdAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Box>
      )}
    </Box>
  );
}

export function AdminOrdersPage() {
  const [page, setPage] = useState(1);
  const ordersQuery = useAdminOrdersPaged({ page, pageSize: DEFAULT_PAGE_SIZE });
  const [selectedRef, setSelectedRef] = useState<string | null>(null);
  const detailQuery = useAdminOrder(selectedRef);

  const total = ordersQuery.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / DEFAULT_PAGE_SIZE));

  const { standard, custom, visits } = useMemo(
    () => bucketOrders(ordersQuery.data?.items ?? []),
    [ordersQuery.data],
  );

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<ReceiptLongOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Orders" }]}
        title="Orders"
        description="Fulfilment. Quote custom requests, send payment links, and move orders through production."
      />

      <Box sx={{ mb: 4 }} />

      {ordersQuery.isPending && (
        <Stack spacing={4}>
          {[0, 1, 2].map((bucket) => (
            <Box key={bucket}>
              <Skeleton variant="text" width={160} sx={{ mb: 1 }} />
              <Stack spacing={1}>
                {[0, 1, 2].map((row) => (
                  <Skeleton key={row} variant="rectangular" height={44} />
                ))}
              </Stack>
            </Box>
          ))}
        </Stack>
      )}

      {ordersQuery.isError && (
        <ErrorState
          message={errorMessage(ordersQuery.error)}
          onRetry={() => ordersQuery.refetch()}
        />
      )}

      {ordersQuery.isSuccess && (
        <Grid container spacing={{ xs: 3, md: 4 }}>
          <Grid size={{ xs: 12, md: selectedRef ? 7 : 12 }}>
            <Stack spacing={4}>
              <OrderTable
                title="Standard bookings"
                orders={standard}
                selectedRef={selectedRef}
                onSelect={setSelectedRef}
              />
              <OrderTable
                title="Custom requests"
                orders={custom}
                selectedRef={selectedRef}
                onSelect={setSelectedRef}
              />
              <OrderTable
                title="Visit bookings"
                orders={visits}
                selectedRef={selectedRef}
                onSelect={setSelectedRef}
              />

              {pageCount > 1 && (
                <Stack direction="row" sx={{ mt: 1, justifyContent: "center" }}>
                  <Pagination
                    count={pageCount}
                    page={page}
                    onChange={(_event, value) => {
                      // The selected order may not be on the new page; clear it.
                      setSelectedRef(null);
                      setPage(value);
                    }}
                    color="primary"
                  />
                </Stack>
              )}
            </Stack>
          </Grid>

          {selectedRef && (
            <Grid size={{ xs: 12, md: 5 }}>
              {detailQuery.isPending && (
                <Paper
                  elevation={0}
                  sx={{ p: { xs: 2, md: 3 }, border: "1px solid", borderColor: "divider" }}
                >
                  <Stack spacing={2}>
                    <Skeleton variant="text" width={100} />
                    <Skeleton variant="text" width={220} height={40} />
                    <Skeleton variant="rectangular" height={120} />
                    <Skeleton variant="rectangular" height={88} />
                  </Stack>
                </Paper>
              )}
              {detailQuery.isError && (
                <ErrorState
                  message={errorMessage(detailQuery.error)}
                  onRetry={() => detailQuery.refetch()}
                />
              )}
              {detailQuery.isSuccess && detailQuery.data && (
                <OrderDetailPanel order={detailQuery.data} onClose={() => setSelectedRef(null)} />
              )}
            </Grid>
          )}
        </Grid>
      )}
    </Box>
  );
}
