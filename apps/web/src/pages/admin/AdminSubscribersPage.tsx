import { useState } from "react";
import Alert from "@mui/material/Alert";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import IconButton from "@mui/material/IconButton";
import Pagination from "@mui/material/Pagination";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import MailOutlined from "@mui/icons-material/MailOutlined";
import FileDownloadOutlined from "@mui/icons-material/FileDownloadOutlined";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { hideUntilMd, tableMinWidth } from "@/components/tableResponsive";
import { useCan } from "@/features/auth/permissions";
import {
  DEFAULT_PAGE_SIZE,
  errorMessage,
  subscribersToCsv,
  type Subscriber,
} from "@/features/subscribers/api";
import { useDeleteSubscriber, useSubscribers } from "@/features/subscribers/hooks";
import { brass, monoFamily } from "@/theme";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function initials(name: string, email: string): string {
  const parts = (name.trim() || email).split(/[\s@.]+/).filter(Boolean);
  return ((parts[0]?.[0] ?? "?") + (parts[1]?.[0] ?? "")).toUpperCase();
}

/** Triggers a client-side download of the subscriber list as a CSV file. */
function downloadCsv(subscribers: Subscriber[]): void {
  const blob = new Blob([subscribersToCsv(subscribers)], {
    type: "text/csv;charset=utf-8;",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");

  link.href = url;
  link.download = `subscribers-page-${new Date().toISOString().slice(0, 10)}.csv`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <Box sx={{ flex: 1, minWidth: 140 }}>
      <Typography
        variant="overline"
        component="p"
        sx={{ color: brass, fontFamily: monoFamily, letterSpacing: "0.16em" }}
      >
        {label}
      </Typography>
      <Typography variant="h5" component="p" sx={{ mt: 0.25 }}>
        {value}
      </Typography>
    </Box>
  );
}

export function AdminSubscribersPage() {
  const canWrite = useCan("subscribers:write");
  const [page, setPage] = useState(1);
  const query = useSubscribers({ page, pageSize: DEFAULT_PAGE_SIZE });
  const remove = useDeleteSubscriber();

  const subscribers = query.data?.items ?? [];
  const total = query.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / DEFAULT_PAGE_SIZE));
  const latest = subscribers[0]?.createdAt;

  const [deleteTarget, setDeleteTarget] = useState<Subscriber | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<MailOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Subscribers" }]}
        title="Newsletter"
        description="Everyone who joined the waitlist. Export the list, browse it page by page, or remove anyone who asks to be taken off."
      />

      <Stack
        direction="row"
        spacing={2}
        sx={{
          mt: 3,
          mb: 4,
          p: { xs: 2.5, md: 3 },
          border: "1px solid",
          borderColor: "divider",
          bgcolor: "background.paper",
          flexWrap: "wrap",
          rowGap: 2,
          alignItems: "center",
        }}
      >
        <Stat label="Subscribers" value={total.toLocaleString("en-GH")} />
        <Stat label="Newest signup" value={latest ? formatDate(latest) : "—"} />
        <Button
          variant="outlined"
          size="small"
          startIcon={<FileDownloadOutlined />}
          disabled={subscribers.length === 0}
          onClick={() => downloadCsv(subscribers)}
          sx={{ ml: "auto" }}
        >
          Export CSV
        </Button>
      </Stack>

      {remove.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {errorMessage(remove.error)}
        </Alert>
      )}

      {query.isPending && (
        <Stack spacing={1.5}>
          {[0, 1, 2, 3, 4].map((row) => (
            <Skeleton key={row} variant="rectangular" height={56} />
          ))}
        </Stack>
      )}

      {query.isError && (
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      )}

      {query.isSuccess && subscribers.length === 0 && (
        <EmptyState
          label="Empty list"
          title="No subscribers yet."
          body="When people join the waitlist on the storefront, they'll show up here."
        />
      )}

      {query.isSuccess && subscribers.length > 0 && (
        <>
          <Box sx={{ overflowX: "auto" }}>
            <Table aria-label="Subscribers" size="small" sx={tableMinWidth(560)}>
              <TableHead>
                <TableRow>
                  <TableCell>Subscriber</TableCell>
                  <TableCell sx={hideUntilMd}>Joined</TableCell>
                  {canWrite && <TableCell align="right">Actions</TableCell>}
                </TableRow>
              </TableHead>
              <TableBody>
                {subscribers.map((subscriber) => (
                  <TableRow key={subscriber.id} hover>
                    <TableCell>
                      <Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
                        <Avatar
                          sx={{
                            width: 36,
                            height: 36,
                            bgcolor: "rgba(224,164,74,0.16)",
                            color: brass,
                            fontFamily: monoFamily,
                            fontSize: "0.8rem",
                          }}
                        >
                          {initials(subscriber.name, subscriber.email)}
                        </Avatar>
                        <Box sx={{ minWidth: 0 }}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {subscriber.name || "—"}
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{ color: "text.secondary", wordBreak: "break-all" }}
                          >
                            {subscriber.email}
                          </Typography>
                        </Box>
                      </Stack>
                    </TableCell>
                    <TableCell sx={{ color: "text.secondary", ...hideUntilMd }}>
                      {formatDate(subscriber.createdAt)}
                    </TableCell>
                    {canWrite && (
                      <TableCell align="right">
                        <IconButton
                          size="small"
                          aria-label={`Remove ${subscriber.email}`}
                          onClick={() => {
                            setDeleteError(null);
                            setDeleteTarget(subscriber);
                          }}
                        >
                          <DeleteOutlineIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Box>

          {pageCount > 1 && (
            <Stack direction="row" sx={{ mt: 3, justifyContent: "center" }}>
              <Pagination
                count={pageCount}
                page={page}
                onChange={(_event, value) => setPage(value)}
                color="primary"
              />
            </Stack>
          )}
        </>
      )}

      {deleteTarget && (
        <Dialog
          open
          onClose={remove.isPending ? undefined : () => setDeleteTarget(null)}
          fullWidth
          maxWidth="xs"
        >
          <DialogTitle sx={{ typography: "h5" }}>Remove subscriber?</DialogTitle>
          <DialogContent>
            <DialogContentText variant="body2">
              {deleteTarget.email} will be removed from the waitlist. They can re-join from the
              storefront at any time.
            </DialogContentText>
            {deleteError && (
              <Alert severity="error" sx={{ mt: 2 }}>
                {deleteError}
              </Alert>
            )}
          </DialogContent>
          <DialogActions sx={{ px: 3, pb: 3 }}>
            <Button variant="text" onClick={() => setDeleteTarget(null)} disabled={remove.isPending}>
              Cancel
            </Button>
            <Button
              variant="contained"
              color="error"
              loading={remove.isPending}
              onClick={() => {
                setDeleteError(null);
                remove.mutate(deleteTarget.id, {
                  onSuccess: () => setDeleteTarget(null),
                  onError: (error) => setDeleteError(errorMessage(error)),
                });
              }}
            >
              Remove
            </Button>
          </DialogActions>
        </Dialog>
      )}
    </Box>
  );
}
