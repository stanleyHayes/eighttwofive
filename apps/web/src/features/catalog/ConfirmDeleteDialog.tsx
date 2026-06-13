import { useState, type ReactNode } from "react";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";

interface ConfirmDeleteDialogProps {
  title: string;
  /** Exact name the admin has to type before the delete button unlocks. */
  name: string;
  description: ReactNode;
  confirming: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: () => void;
}

/**
 * Type-the-name confirmation for permanent deletes. Render it conditionally
 * (mount when needed) so the typed value resets between targets.
 */
export function ConfirmDeleteDialog({
  title,
  name,
  description,
  confirming,
  error,
  onClose,
  onConfirm,
}: ConfirmDeleteDialogProps) {
  const [typed, setTyped] = useState("");
  const matches = typed === name;

  return (
    <Dialog open onClose={confirming ? undefined : onClose} fullWidth maxWidth="sm">
      <DialogTitle sx={{ typography: "h5" }}>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2.5} sx={{ mt: 0.5 }}>
          <Typography component="div" variant="body2" sx={{ color: "text.secondary" }}>
            {description}
          </Typography>
          <TextField
            label={`Type "${name}" to confirm`}
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
            fullWidth
            autoFocus
          />
          {error && <Alert severity="error">{error}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 3 }}>
        <Button variant="text" onClick={onClose} disabled={confirming}>
          Cancel
        </Button>
        <Button
          variant="contained"
          color="error"
          disabled={!matches}
          loading={confirming}
          onClick={onConfirm}
        >
          Delete permanently
        </Button>
      </DialogActions>
    </Dialog>
  );
}
