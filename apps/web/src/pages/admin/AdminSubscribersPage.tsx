import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
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
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { hideUntilMd, tableMinWidth } from "@/components/tableResponsive";
import {
  DEFAULT_PAGE_SIZE,
  errorMessage,
  subscribersToCsv,
  type Subscriber,
} from "@/features/subscribers/api";
import { useSubscribers } from "@/features/subscribers/hooks";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
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

export function AdminSubscribersPage() {
  const [page, setPage] = useState(1);
  const query = useSubscribers({ page, pageSize: DEFAULT_PAGE_SIZE });

  const subscribers = query.data?.items ?? [];
  const total = query.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / DEFAULT_PAGE_SIZE));

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<MailOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Subscribers" }]}
        title="Newsletter"
        description="Everyone who joined the waitlist. Export the list or browse it page by page."
      />

      <Stack
        direction={{ xs: "column", sm: "row" }}
        spacing={2}
        sx={{ mt: 3, mb: 4, alignItems: { sm: "center" }, justifyContent: "space-between" }}
      >
        <Typography variant="body2" sx={{ color: "text.secondary" }}>
          {total === 0
            ? "No subscribers yet"
            : `${total} subscriber${total === 1 ? "" : "s"}`}
        </Typography>
        <Button
          variant="outlined"
          size="small"
          startIcon={<FileDownloadOutlined />}
          disabled={subscribers.length === 0}
          onClick={() => downloadCsv(subscribers)}
        >
          Export CSV
        </Button>
      </Stack>

      {query.isPending && (
        <Stack spacing={1.5}>
          {[0, 1, 2, 3, 4].map((row) => (
            <Skeleton key={row} variant="rectangular" height={48} />
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
                  <TableCell>Email</TableCell>
                  <TableCell>Name</TableCell>
                  <TableCell sx={hideUntilMd}>Joined</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {subscribers.map((subscriber) => (
                  <TableRow key={subscriber.id} hover>
                    <TableCell sx={{ fontWeight: 500, wordBreak: "break-all" }}>
                      {subscriber.email}
                    </TableCell>
                    <TableCell>{subscriber.name}</TableCell>
                    <TableCell sx={{ color: "text.secondary", ...hideUntilMd }}>
                      {formatDate(subscriber.createdAt)}
                    </TableCell>
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
    </Box>
  );
}
