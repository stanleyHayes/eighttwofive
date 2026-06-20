import { useState, type FormEvent } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import FormControlLabel from "@mui/material/FormControlLabel";
import FormGroup from "@mui/material/FormGroup";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { brass, monoFamily } from "@/theme";
import { errorMessage, type PermissionMeta, type RoleDef } from "./api";
import { useCreateRole, useUpdateRole } from "./hooks";

interface RoleFormDialogProps {
  /** When set, the dialog edits this role; otherwise it creates one. */
  initial?: RoleDef;
  permissions: PermissionMeta[];
  onClose: () => void;
}

/** Preserve the catalogue's order while bucketing permissions by their group. */
function groupPermissions(permissions: PermissionMeta[]): [string, PermissionMeta[]][] {
  const groups: [string, PermissionMeta[]][] = [];
  for (const perm of permissions) {
    const existing = groups.find(([name]) => name === perm.group);
    if (existing) existing[1].push(perm);
    else groups.push([perm.group, [perm]]);
  }
  return groups;
}

/** Render conditionally (mount when needed) so the fields reset per target. */
export function RoleFormDialog({ initial, permissions, onClose }: RoleFormDialogProps) {
  // The admin role is the recovery path: its permissions and dashboard access
  // are fixed, so the form shows them locked.
  const isAdminRole = initial?.key === "admin";
  const isSystem = initial?.system ?? false;

  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [adminArea, setAdminArea] = useState(initial?.adminArea ?? true);
  const [selected, setSelected] = useState<Set<string>>(
    () => new Set(initial?.permissions ?? []),
  );
  const [nameError, setNameError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);

  const create = useCreateRole();
  const update = useUpdateRole();
  const saving = create.isPending || update.isPending;

  const togglePerm = (key: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!name.trim()) {
      setNameError("Name is required.");
      return;
    }
    setNameError(null);
    setServerError(null);

    const input = {
      name: name.trim(),
      description: description.trim(),
      permissions: [...selected],
      adminArea,
    };
    const callbacks = {
      onSuccess: () => onClose(),
      onError: (error: unknown) => setServerError(errorMessage(error)),
    };
    if (initial) update.mutate({ key: initial.key, input }, callbacks);
    else create.mutate(input, callbacks);
  };

  return (
    <Dialog open onClose={saving ? undefined : onClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle sx={{ typography: "h5" }}>
          {initial ? `Edit ${initial.name}` : "New role"}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2.5} sx={{ mt: 0.5 }}>
            {isAdminRole && (
              <Alert severity="info">
                The Admin role always keeps every permission and dashboard access — only its
                name and description can change.
              </Alert>
            )}

            <TextField
              label="Name"
              value={name}
              onChange={(event) => {
                setName(event.target.value);
                if (nameError) setNameError(null);
              }}
              error={Boolean(nameError)}
              helperText={nameError ?? " "}
              fullWidth
              required
              autoFocus
            />
            <TextField
              label="Description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              multiline
              minRows={2}
              fullWidth
            />

            <FormControlLabel
              control={
                <Switch
                  checked={adminArea}
                  onChange={(event) => setAdminArea(event.target.checked)}
                  disabled={isSystem}
                />
              }
              label="Can access the admin dashboard"
            />

            <Box>
              <Typography
                variant="overline"
                component="p"
                sx={{ color: brass, fontFamily: monoFamily, mb: 0.5 }}
              >
                Permissions
              </Typography>
              <Stack spacing={2} sx={{ mt: 1 }}>
                {groupPermissions(permissions).map(([group, perms]) => (
                  <Box key={group}>
                    <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
                      {group}
                    </Typography>
                    <FormGroup>
                      {perms.map((perm) => (
                        <FormControlLabel
                          key={perm.key}
                          control={
                            <Checkbox
                              size="small"
                              checked={isAdminRole || selected.has(perm.key)}
                              disabled={isAdminRole}
                              onChange={() => togglePerm(perm.key)}
                            />
                          }
                          label={
                            <Box>
                              <Typography variant="body2">{perm.label}</Typography>
                              <Typography variant="caption" sx={{ color: "text.secondary" }}>
                                {perm.description}
                              </Typography>
                            </Box>
                          }
                          sx={{ alignItems: "flex-start", mb: 0.5 }}
                        />
                      ))}
                    </FormGroup>
                  </Box>
                ))}
              </Stack>
            </Box>

            {serverError && <Alert severity="error">{serverError}</Alert>}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3 }}>
          <Button variant="text" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" loading={saving}>
            {initial ? "Save changes" : "Create role"}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}
