import { useState } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import Pagination from "@mui/material/Pagination";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import ArchiveOutlinedIcon from "@mui/icons-material/ArchiveOutlined";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutlined";
import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import UnarchiveOutlinedIcon from "@mui/icons-material/UnarchiveOutlined";
import CollectionsOutlined from "@mui/icons-material/CollectionsOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { hideUntilMd, tableMinWidth } from "@/components/tableResponsive";
import { DEFAULT_PAGE_SIZE, errorMessage, type Collection } from "@/features/catalog/api";
import {
  useCollectionsPaged,
  useDeleteCollection,
  useRestoreCollection,
  useRetireCollection,
} from "@/features/catalog/hooks";
import { CollectionFormDialog } from "@/features/catalog/CollectionFormDialog";
import { ConfirmDeleteDialog } from "@/features/catalog/ConfirmDeleteDialog";
import { StatusChip } from "@/features/catalog/StatusChip";
import { useCan } from "@/features/auth/permissions";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

type DialogState = { mode: "create" } | { mode: "edit"; collection: Collection } | null;

export function AdminCollectionsPage() {
  const canWrite = useCan("catalogue:write");
  const canDelete = useCan("catalogue:delete");
  const [page, setPage] = useState(1);
  const collections = useCollectionsPaged({ page, pageSize: DEFAULT_PAGE_SIZE });
  const retire = useRetireCollection();
  const restore = useRestoreCollection();
  const remove = useDeleteCollection();

  const rows = collections.data?.items ?? [];
  const total = collections.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / DEFAULT_PAGE_SIZE));

  const [dialog, setDialog] = useState<DialogState>(null);
  const [deleteTarget, setDeleteTarget] = useState<Collection | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const toggleRetired = (collection: Collection) => {
    setActionError(null);
    const mutation = collection.status === "live" ? retire : restore;
    mutation.mutate(collection.id, {
      onError: (error) => setActionError(errorMessage(error)),
    });
  };

  const rowBusy = (id: string) =>
    (retire.isPending && retire.variables === id) ||
    (restore.isPending && restore.variables === id);

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<CollectionsOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Collections" }]}
        title="Collections"
        description="Each collection is a limited, themed run of around ten designs. Create, edit, retire, or delete them here."
      />

      {canWrite && (
        <Box sx={{ mb: 4, mt: 3 }}>
          <Button variant="contained" onClick={() => setDialog({ mode: "create" })}>
            New collection
          </Button>
        </Box>
      )}

      {actionError && (
        <Alert severity="error" onClose={() => setActionError(null)} sx={{ mb: 3 }}>
          {actionError}
        </Alert>
      )}

      {collections.isPending && (
        <Stack spacing={1.5}>
          {[0, 1, 2, 3].map((row) => (
            <Skeleton key={row} variant="rectangular" height={48} />
          ))}
        </Stack>
      )}

      {collections.isError && (
        <ErrorState
          message={errorMessage(collections.error)}
          onRetry={() => collections.refetch()}
        />
      )}

      {collections.isSuccess && rows.length === 0 && (
        <EmptyState
          label="Empty catalog"
          title="No collections yet."
          body="Create the first collection to start the catalog — each one is a limited, themed run of around ten designs."
        />
      )}

      {collections.isSuccess && rows.length > 0 && (
        <Box sx={{ overflowX: "auto" }}>
          <Table aria-label="Collections" size="small" sx={tableMinWidth(560)}>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell sx={hideUntilMd}>Slug</TableCell>
              <TableCell>Status</TableCell>
              <TableCell sx={hideUntilMd}>Created</TableCell>
              {(canWrite || canDelete) && <TableCell align="right">Actions</TableCell>}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((collection) => (
              <TableRow key={collection.id} hover>
                <TableCell sx={{ fontWeight: 500 }}>{collection.name}</TableCell>
                <TableCell sx={{ color: "text.secondary", ...hideUntilMd }}>{collection.slug}</TableCell>
                <TableCell>
                  <StatusChip status={collection.status} />
                </TableCell>
                <TableCell sx={{ color: "text.secondary", ...hideUntilMd }}>
                  {formatDate(collection.createdAt)}
                </TableCell>
                {(canWrite || canDelete) && (
                  <TableCell align="right" sx={{ whiteSpace: "nowrap" }}>
                    {canWrite && (
                      <IconButton
                        size="small"
                        aria-label={`Edit ${collection.name}`}
                        onClick={() => setDialog({ mode: "edit", collection })}
                      >
                        <EditOutlinedIcon fontSize="small" />
                      </IconButton>
                    )}
                    {canWrite && (
                      <IconButton
                        size="small"
                        aria-label={
                          collection.status === "live"
                            ? `Retire ${collection.name}`
                            : `Restore ${collection.name}`
                        }
                        disabled={rowBusy(collection.id)}
                        onClick={() => toggleRetired(collection)}
                      >
                        {collection.status === "live" ? (
                          <ArchiveOutlinedIcon fontSize="small" />
                        ) : (
                          <UnarchiveOutlinedIcon fontSize="small" />
                        )}
                      </IconButton>
                    )}
                    {canDelete && (
                      <IconButton
                        size="small"
                        aria-label={`Delete ${collection.name}`}
                        onClick={() => {
                          setDeleteError(null);
                          setDeleteTarget(collection);
                        }}
                      >
                        <DeleteOutlineIcon fontSize="small" />
                      </IconButton>
                    )}
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
        </Box>
      )}

      {collections.isSuccess && rows.length > 0 && pageCount > 1 && (
        <Stack direction="row" sx={{ mt: 3, justifyContent: "center" }}>
          <Pagination
            count={pageCount}
            page={page}
            onChange={(_event, value) => setPage(value)}
            color="primary"
          />
        </Stack>
      )}

      {dialog && (
        <CollectionFormDialog
          initial={dialog.mode === "edit" ? dialog.collection : undefined}
          onClose={() => setDialog(null)}
        />
      )}

      {deleteTarget && (
        <ConfirmDeleteDialog
          title={`Delete ${deleteTarget.name}?`}
          name={deleteTarget.name}
          description={
            <>
              This is PERMANENT. The collection "{deleteTarget.name}" and every design
              inside it will be deleted for good — there is no way to restore them. If
              you only want to hide it from the storefront, retire it instead.
            </>
          }
          confirming={remove.isPending}
          error={deleteError}
          onClose={() => setDeleteTarget(null)}
          onConfirm={() => {
            setDeleteError(null);
            remove.mutate(deleteTarget.id, {
              onSuccess: () => setDeleteTarget(null),
              onError: (error) => setDeleteError(errorMessage(error)),
            });
          }}
        />
      )}
    </Box>
  );
}
