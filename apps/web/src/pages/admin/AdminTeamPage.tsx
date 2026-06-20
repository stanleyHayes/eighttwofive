import { useState } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Pagination from "@mui/material/Pagination";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import GroupOutlined from "@mui/icons-material/GroupOutlined";
import PersonAddAltOutlined from "@mui/icons-material/PersonAddAltOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { hideUntilMd, tableMinWidth } from "@/components/tableResponsive";
import { useMe } from "@/features/auth/useMe";
import { useCan } from "@/features/auth/permissions";
import { DEFAULT_PAGE_SIZE, errorMessage } from "@/features/team/api";
import { useSetUserRole, useUsers } from "@/features/team/hooks";
import { InvitePartnerDialog } from "@/features/team/InvitePartnerDialog";
import { useRoles } from "@/features/roles/hooks";
import type { RoleDef } from "@/features/roles/api";
import type { User, UserRole } from "@/lib/api";

function RoleCell({
  user,
  roles,
  disabled,
  onChange,
}: {
  user: User;
  roles: RoleDef[];
  disabled: boolean;
  onChange: (role: UserRole) => void;
}) {
  if (user.isSuperAdmin) {
    return <Chip size="small" label="Super-admin" color="warning" variant="outlined" />;
  }

  return (
    <TextField
      select
      size="small"
      value={user.role}
      onChange={(event) => onChange(event.target.value)}
      disabled={disabled || roles.length === 0}
      slotProps={{ select: { native: true } }}
      sx={{ minWidth: 220 }}
      aria-label={`Role for ${user.email}`}
    >
      {/* A user can hold a role no longer in the list (e.g. just deleted); keep
          it selectable so the dropdown reflects reality until reassigned. */}
      {roles.every((role) => role.key !== user.role) && (
        <option value={user.role}>{user.role}</option>
      )}
      {roles.map((role) => (
        <option key={role.key} value={role.key}>
          {role.name}
          {role.adminArea ? "" : " — storefront only"}
        </option>
      ))}
    </TextField>
  );
}

export function AdminTeamPage() {
  const me = useMe();
  const canWrite = useCan("team:write");
  const [page, setPage] = useState(1);
  const [inviteOpen, setInviteOpen] = useState(false);
  const query = useUsers({ page, pageSize: DEFAULT_PAGE_SIZE });
  const rolesQuery = useRoles();
  const setRole = useSetUserRole();

  const users = query.data?.items ?? [];
  const roles = rolesQuery.data ?? [];
  const total = query.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / DEFAULT_PAGE_SIZE));

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<GroupOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Team" }]}
        title="Team & access"
        description="Invite partners and assign roles to control who can do what. Manage the roles themselves under Roles."
      />

      <Stack
        direction={{ xs: "column", sm: "row" }}
        spacing={2}
        sx={{ mt: 3, mb: 4, justifyContent: "space-between", alignItems: { sm: "center" } }}
      >
        <Typography variant="body2" sx={{ color: "text.secondary" }}>
          {total === 0 ? "No users yet" : `${total} user${total === 1 ? "" : "s"}`} · super-admins
          (from server config) always keep full access.
        </Typography>
        {canWrite && (
          <Button
            variant="contained"
            startIcon={<PersonAddAltOutlined />}
            onClick={() => setInviteOpen(true)}
            sx={{ flexShrink: 0 }}
          >
            Invite partner
          </Button>
        )}
      </Stack>

      {setRole.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {errorMessage(setRole.error)}
        </Alert>
      )}

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

      {query.isSuccess && users.length === 0 && (
        <EmptyState
          label="Empty"
          title="No users yet."
          body="Invite a partner, or people appear here once they sign in or check out on the storefront."
        />
      )}

      {query.isSuccess && users.length > 0 && (
        <>
          <Box sx={{ overflowX: "auto" }}>
            <Table aria-label="Team members" size="small" sx={tableMinWidth(640)}>
              <TableHead>
                <TableRow>
                  <TableCell>Email</TableCell>
                  <TableCell sx={hideUntilMd}>Name</TableCell>
                  <TableCell>Role</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {users.map((user) => {
                  const isSelf = me.data?.id === user.id;

                  return (
                    <TableRow key={user.id} hover>
                      <TableCell sx={{ fontWeight: 500, wordBreak: "break-all" }}>
                        {user.email}
                        {isSelf && (
                          <Chip size="small" label="You" sx={{ ml: 1 }} variant="outlined" />
                        )}
                      </TableCell>
                      <TableCell sx={hideUntilMd}>{user.name}</TableCell>
                      <TableCell>
                        <RoleCell
                          user={user}
                          roles={roles}
                          disabled={isSelf || setRole.isPending || !canWrite}
                          onChange={(role) => setRole.mutate({ id: user.id, role })}
                        />
                      </TableCell>
                    </TableRow>
                  );
                })}
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

      {inviteOpen && <InvitePartnerDialog onClose={() => setInviteOpen(false)} />}
    </Box>
  );
}
