import { useMemo, useState, type ChangeEvent, type FormEvent } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import EventOutlined from "@mui/icons-material/EventOutlined";
import EventAvailableOutlined from "@mui/icons-material/EventAvailableOutlined";
import { PageBanner } from "@/components/PageBanner";
import { TableEmptyState } from "@/components/TableEmptyState";
import { hideUntilMd, tableMinWidth } from "@/components/tableResponsive";
import { moss, noir, stone } from "@/theme";
import {
  useAdminSlots,
  useCreateSlot,
  useCloseSlot,
  useReopenSlot,
  useAdminVisits,
  useRescheduleVisit,
  useCancelVisit,
  formatSlotTime,
} from "@/features/slots/hooks";
import { errorMessage, type Slot, type Visit } from "@/features/slots/api";

function fromLocalInputValue(value: string): string {
  return new Date(value).toISOString();
}

function StatusChip({ status }: { status: Slot["status"] | Visit["status"] }) {
  const color =
    status === "open"
      ? moss
      : status === "booked"
        ? noir
        : status === "closed"
          ? stone
          : status === "cancelled"
            ? "#8c3a2b"
            : moss;

  return <Chip label={status} size="small" sx={{ borderRadius: 0, color, borderColor: color }} variant="outlined" />;
}

interface SlotFormDialogProps {
  open: boolean;
  onClose: () => void;
}

function SlotFormDialog({ open, onClose }: SlotFormDialogProps) {
  const create = useCreateSlot();
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const handleClose = () => {
    onClose();
    setStart("");
    setEnd("");
    setFormError(null);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);

    if (!start || !end) {
      setFormError("Start and end times are required.");
      return;
    }

    const startISO = fromLocalInputValue(start);
    const endISO = fromLocalInputValue(end);

    if (new Date(endISO) <= new Date(startISO)) {
      setFormError("End time must be after start time.");
      return;
    }

    create.mutate(
      { start: startISO, end: endISO },
      {
        onSuccess: handleClose,
        onError: (err) => setFormError(errorMessage(err)),
      },
    );
  };

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle>Add a home-visit slot</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <TextField
              label="Start"
              type="datetime-local"
              value={start}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setStart(event.target.value)}
              slotProps={{ htmlInput: { step: 60 } }}
              fullWidth
              required
            />
            <TextField
              label="End"
              type="datetime-local"
              value={end}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setEnd(event.target.value)}
              slotProps={{ htmlInput: { step: 60 } }}
              fullWidth
              required
            />
            {(formError || create.isError) && <Alert severity="error">{formError || errorMessage(create.error)}</Alert>}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button type="button" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" loading={create.isPending}>
            Add slot
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}

interface RescheduleDialogProps {
  visit: Visit;
  open: boolean;
  slots: Slot[];
  onClose: () => void;
}

function RescheduleDialog({ visit, open, slots, onClose }: RescheduleDialogProps) {
  const reschedule = useRescheduleVisit();
  const [newSlotId, setNewSlotId] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const availableSlots = useMemo(
    () => slots.filter((slot) => slot.status === "open" && slot.id !== visit.slotId),
    [slots, visit.slotId],
  );

  const handleClose = () => {
    onClose();
    setNewSlotId("");
    setFormError(null);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);

    if (!newSlotId) {
      setFormError("Choose a new open slot.");
      return;
    }

    reschedule.mutate(
      { visitId: visit.id, input: { newSlotId } },
      {
        onSuccess: handleClose,
        onError: (err) => setFormError(errorMessage(err)),
      },
    );
  };

  return (
    <Dialog open={open} onClose={handleClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle>Reschedule visit</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <FormControl fullWidth required>
              <InputLabel id="new-slot-label">New slot</InputLabel>
              <Select
                labelId="new-slot-label"
                value={newSlotId}
                label="New slot"
                onChange={(event) => setNewSlotId(event.target.value)}
              >
                {availableSlots.map((slot) => (
                  <MenuItem key={slot.id} value={slot.id}>
                    {formatSlotTime(slot.start, slot.end)}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            {(formError || reschedule.isError) && (
              <Alert severity="error">{formError || errorMessage(reschedule.error)}</Alert>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button type="button" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" loading={reschedule.isPending}>
            Reschedule
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}

export function AdminSlotsPage() {
  const { data: slots, isLoading: slotsLoading, error: slotsError } = useAdminSlots();
  const { data: visits, isLoading: visitsLoading, error: visitsError } = useAdminVisits();
  const close = useCloseSlot();
  const reopen = useReopenSlot();
  const cancel = useCancelVisit();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [rescheduleVisit, setRescheduleVisit] = useState<Visit | null>(null);

  const isLoading = slotsLoading || visitsLoading;
  const error = slotsError || visitsError;

  if (isLoading) {
    return (
      <Box sx={{ maxWidth: 960 }}>
        <Skeleton variant="text" width={160} sx={{ mb: 1 }} />
        <Skeleton variant="rectangular" height={200} />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ maxWidth: 960 }}>
        <Alert severity="error">{errorMessage(error, "Could not load calendar.")}</Alert>
      </Box>
    );
  }

  const slotList = slots ?? [];
  const visitList = visits ?? [];

  return (
    <Box sx={{ maxWidth: 960 }}>
      <PageBanner
        tone="ink"
        icon={<EventOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Visits" }]}
        title="Home-visit slots"
        description="Open windows for in-person measuring visits, and the visits customers have booked against them."
      />

      <Box sx={{ mb: 3 }} />

      <Stack spacing={4}>
        <Box>
          <Stack direction="row" spacing={2} sx={{ justifyContent: "space-between", alignItems: "center", mb: 2 }}>
            <Typography variant="h5" component="h2">
              Slots
            </Typography>
            <Button variant="contained" onClick={() => setDialogOpen(true)}>
              Add slot
            </Button>
          </Stack>

          <TableContainer sx={{ overflowX: "auto" }}>
            <Table sx={tableMinWidth(640)}>
              <TableHead>
                <TableRow>
                  <TableCell>Date / time</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {slotList.map((slot) => (
                  <TableRow key={slot.id}>
                    <TableCell>{formatSlotTime(slot.start, slot.end)}</TableCell>
                    <TableCell>
                      <StatusChip status={slot.status} />
                    </TableCell>
                    <TableCell align="right">
                      {slot.status === "open" && (
                        <Button size="small" onClick={() => close.mutate(slot.id)}>
                          Close
                        </Button>
                      )}
                      {slot.status === "closed" && (
                        <Button size="small" onClick={() => reopen.mutate(slot.id)}>
                          Reopen
                        </Button>
                      )}
                      {slot.status === "booked" && <Typography variant="body2">Booked</Typography>}
                    </TableCell>
                  </TableRow>
                ))}
                {slotList.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} sx={{ borderBottom: "none", p: 0 }}>
                      <TableEmptyState
                        icon={<EventOutlined />}
                        title="No slots yet"
                        body="Open your first home-visit window and clients can book a fitting."
                        action={{ label: "Add a slot", onClick: () => setDialogOpen(true) }}
                      />
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>

        <Box>
          <Typography variant="h5" component="h2" sx={{ mb: 2 }}>
            Visits
          </Typography>

          <TableContainer sx={{ overflowX: "auto" }}>
            <Table sx={tableMinWidth(640)}>
              <TableHead>
                <TableRow>
                  <TableCell>Order</TableCell>
                  <TableCell sx={hideUntilMd}>Slot</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {visitList.map((visit) => {
                  const slot = slotList.find((s) => s.id === visit.slotId);

                  return (
                    <TableRow key={visit.id}>
                      <TableCell>{visit.orderId}</TableCell>
                      <TableCell sx={hideUntilMd}>
                        {slot ? formatSlotTime(slot.start, slot.end) : "Unknown slot"}
                      </TableCell>
                      <TableCell>
                        <StatusChip status={visit.status} />
                      </TableCell>
                      <TableCell align="right">
                        <Stack direction="row" spacing={1} sx={{ justifyContent: "flex-end" }}>
                          {visit.status === "booked" && (
                            <>
                              <Button size="small" onClick={() => setRescheduleVisit(visit)}>
                                Reschedule
                              </Button>
                              <Button size="small" onClick={() => cancel.mutate(visit.id)}>
                                Cancel
                              </Button>
                            </>
                          )}
                        </Stack>
                      </TableCell>
                    </TableRow>
                  );
                })}
                {visitList.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} sx={{ borderBottom: "none", p: 0 }}>
                      <TableEmptyState
                        icon={<EventAvailableOutlined />}
                        title="No visits booked yet"
                        body="When a client books one of your open slots, their fitting shows up here."
                      />
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Box>
      </Stack>

      <SlotFormDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />

      {rescheduleVisit && (
        <RescheduleDialog
          visit={rescheduleVisit}
          open
          slots={slotList}
          onClose={() => setRescheduleVisit(null)}
        />
      )}
    </Box>
  );
}
