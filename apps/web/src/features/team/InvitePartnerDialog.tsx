import { useMemo, useState, type FormEvent } from "react";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import { errorMessage } from "@/features/roles/api";
import { useInvitePartner, useRoles } from "@/features/roles/hooks";

/** Render conditionally (mount when needed) so the fields reset per open. */
export function InvitePartnerDialog({ onClose }: { onClose: () => void }) {
  const rolesQuery = useRoles();
  // Only dashboard roles can be invited as partners.
  const roles = useMemo(
    () => (rolesQuery.data ?? []).filter((role) => role.adminArea),
    [rolesQuery.data],
  );

  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState("");
  const [serverError, setServerError] = useState<string | null>(null);

  const invite = useInvitePartner();
  // Default the select to staff (or the first dashboard role) once roles load.
  const selectedRole = role || roles.find((r) => r.key === "staff")?.key || roles[0]?.key || "";

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setServerError(null);
    invite.mutate(
      { email: email.trim(), name: name.trim(), role: selectedRole },
      {
        onSuccess: () => onClose(),
        onError: (error) => setServerError(errorMessage(error)),
      },
    );
  };

  return (
    <Dialog open onClose={invite.isPending ? undefined : onClose} fullWidth maxWidth="xs">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle sx={{ typography: "h5" }}>Invite a partner</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 0.5 }}>
            <DialogContentText variant="body2">
              We'll email them a sign-in link. Following it signs them in with the role you pick.
            </DialogContentText>
            <TextField
              label="Email"
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              fullWidth
              required
              autoFocus
            />
            <TextField
              label="Name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              fullWidth
              required
            />
            <TextField
              select
              label="Role"
              value={selectedRole}
              onChange={(event) => setRole(event.target.value)}
              slotProps={{ select: { native: true } }}
              fullWidth
              disabled={roles.length === 0}
            >
              {roles.map((option) => (
                <option key={option.key} value={option.key}>
                  {option.name}
                </option>
              ))}
            </TextField>
            {serverError && <Alert severity="error">{serverError}</Alert>}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3 }}>
          <Button variant="text" onClick={onClose} disabled={invite.isPending}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="contained"
            loading={invite.isPending}
            disabled={!selectedRole}
          >
            Send invite
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}
