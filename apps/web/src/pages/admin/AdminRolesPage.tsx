import { useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import IconButton from "@mui/material/IconButton";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import AdminPanelSettingsOutlined from "@mui/icons-material/AdminPanelSettingsOutlined";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutlined";
import EditOutlinedIcon from "@mui/icons-material/EditOutlined";
import LockOutlinedIcon from "@mui/icons-material/LockOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { useCan } from "@/features/auth/permissions";
import { ConfirmDeleteDialog } from "@/features/catalog/ConfirmDeleteDialog";
import { errorMessage, type RoleDef } from "@/features/roles/api";
import { useDeleteRole, usePermissions, useRoles } from "@/features/roles/hooks";
import { RoleFormDialog } from "@/features/roles/RoleFormDialog";
import { brass, monoFamily } from "@/theme";

type DialogState = { mode: "create" } | { mode: "edit"; role: RoleDef } | null;

function RoleCard({
  role,
  canWrite,
  onEdit,
  onDelete,
}: {
  role: RoleDef;
  canWrite: boolean;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <Box
      sx={{
        border: "1px solid",
        borderColor: "divider",
        bgcolor: "background.paper",
        p: { xs: 2.5, md: 3 },
      }}
    >
      <Stack
        direction="row"
        spacing={2}
        sx={{ justifyContent: "space-between", alignItems: "flex-start" }}
      >
        <Box sx={{ minWidth: 0 }}>
          <Stack direction="row" spacing={1.25} sx={{ alignItems: "center", flexWrap: "wrap" }}>
            <Typography variant="h6" component="h2">
              {role.name}
            </Typography>
            <Box component="span" sx={{ fontFamily: monoFamily, fontSize: "0.7rem", color: brass }}>
              {role.key}
            </Box>
          </Stack>
          {role.description && (
            <Typography variant="body2" sx={{ color: "text.secondary", mt: 0.75, maxWidth: "60ch" }}>
              {role.description}
            </Typography>
          )}
          <Stack direction="row" spacing={1} sx={{ mt: 1.5, flexWrap: "wrap", gap: 1 }}>
            {role.system && (
              <Chip
                size="small"
                variant="outlined"
                color="warning"
                icon={<LockOutlinedIcon sx={{ fontSize: 14 }} />}
                label="Built-in"
              />
            )}
            <Chip
              size="small"
              variant="outlined"
              label={role.adminArea ? "Dashboard access" : "Storefront only"}
            />
            <Chip
              size="small"
              variant="outlined"
              label={`${role.permissions.length} permission${role.permissions.length === 1 ? "" : "s"}`}
            />
          </Stack>
        </Box>

        {canWrite && (
          <Stack direction="row" spacing={0.5} sx={{ flexShrink: 0 }}>
            <IconButton size="small" aria-label={`Edit ${role.name}`} onClick={onEdit}>
              <EditOutlinedIcon fontSize="small" />
            </IconButton>
            <IconButton
              size="small"
              aria-label={`Delete ${role.name}`}
              onClick={onDelete}
              disabled={role.system}
              title={role.system ? "Built-in roles can't be deleted" : undefined}
            >
              <DeleteOutlineIcon fontSize="small" />
            </IconButton>
          </Stack>
        )}
      </Stack>
    </Box>
  );
}

export function AdminRolesPage() {
  const canWrite = useCan("team:write");
  const roles = useRoles();
  const permissions = usePermissions();
  const remove = useDeleteRole();

  const [dialog, setDialog] = useState<DialogState>(null);
  const [deleteTarget, setDeleteTarget] = useState<RoleDef | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const rows = roles.data ?? [];

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<AdminPanelSettingsOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Roles" }]}
        title="Roles & permissions"
        description="Define what each role can do. Built-in roles can be retuned but not deleted; create custom roles to fit your team. Changes take effect immediately."
      />

      {canWrite && (
        <Box sx={{ mb: 4, mt: 3 }}>
          <Button
            variant="contained"
            onClick={() => setDialog({ mode: "create" })}
            disabled={permissions.isPending}
          >
            New role
          </Button>
        </Box>
      )}

      {(roles.isPending || permissions.isPending) && (
        <Stack spacing={1.5}>
          {[0, 1, 2, 3].map((row) => (
            <Skeleton key={row} variant="rectangular" height={96} />
          ))}
        </Stack>
      )}

      {roles.isError && (
        <ErrorState message={errorMessage(roles.error)} onRetry={() => roles.refetch()} />
      )}

      {roles.isSuccess && rows.length === 0 && (
        <EmptyState label="Empty" title="No roles yet." body="Create a role to start assigning access." />
      )}

      {roles.isSuccess && permissions.isSuccess && rows.length > 0 && (
        <Stack spacing={2}>
          {rows.map((role) => (
            <RoleCard
              key={role.key}
              role={role}
              canWrite={canWrite}
              onEdit={() => setDialog({ mode: "edit", role })}
              onDelete={() => {
                setDeleteError(null);
                setDeleteTarget(role);
              }}
            />
          ))}
        </Stack>
      )}

      {dialog && permissions.isSuccess && (
        <RoleFormDialog
          initial={dialog.mode === "edit" ? dialog.role : undefined}
          permissions={permissions.data}
          onClose={() => setDialog(null)}
        />
      )}

      {deleteTarget && (
        <ConfirmDeleteDialog
          title={`Delete ${deleteTarget.name}?`}
          name={deleteTarget.name}
          description={
            <>
              This deletes the "{deleteTarget.name}" role. Anyone currently assigned to it loses
              dashboard access until you give them another role. This can't be undone.
            </>
          }
          confirming={remove.isPending}
          error={deleteError}
          onClose={() => setDeleteTarget(null)}
          onConfirm={() => {
            setDeleteError(null);
            remove.mutate(deleteTarget.key, {
              onSuccess: () => setDeleteTarget(null),
              onError: (error) => setDeleteError(errorMessage(error)),
            });
          }}
        />
      )}
    </Box>
  );
}
