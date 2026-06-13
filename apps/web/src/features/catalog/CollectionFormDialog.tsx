import { useState, type FormEvent } from "react";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import { errorMessage, type Collection } from "./api";
import { useCreateCollection, useUpdateCollection } from "./hooks";

interface CollectionFormDialogProps {
  /** When set, the dialog edits this collection; otherwise it creates one. */
  initial?: Collection;
  onClose: () => void;
}

/** Render conditionally (mount when needed) so the fields reset per target. */
export function CollectionFormDialog({ initial, onClose }: CollectionFormDialogProps) {
  const [name, setName] = useState(initial?.name ?? "");
  const [note, setNote] = useState(initial?.note ?? "");
  const [nameError, setNameError] = useState<string | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);

  const create = useCreateCollection();
  const update = useUpdateCollection();
  const saving = create.isPending || update.isPending;

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!name.trim()) {
      setNameError("Name is required.");
      return;
    }
    setNameError(null);
    setServerError(null);

    const input = { name: name.trim(), note: note.trim() };
    const callbacks = {
      onSuccess: () => onClose(),
      onError: (error: unknown) => setServerError(errorMessage(error)),
    };
    if (initial) {
      update.mutate({ id: initial.id, input }, callbacks);
    } else {
      create.mutate(input, callbacks);
    }
  };

  return (
    <Dialog open onClose={saving ? undefined : onClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle sx={{ typography: "h5" }}>
          {initial ? "Edit collection" : "New collection"}
        </DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 0.5 }}>
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
              label="Note"
              value={note}
              onChange={(event) => setNote(event.target.value)}
              multiline
              minRows={3}
              fullWidth
            />
            {serverError && <Alert severity="error">{serverError}</Alert>}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 3 }}>
          <Button variant="text" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" loading={saving}>
            {initial ? "Save changes" : "Create collection"}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}
