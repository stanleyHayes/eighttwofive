import { useState, type ChangeEvent, type FormEvent } from "react";
import { useSearchParams } from "react-router";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Container from "@mui/material/Container";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Radio from "@mui/material/Radio";
import RadioGroup from "@mui/material/RadioGroup";
import FormControl from "@mui/material/FormControl";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import EventOutlined from "@mui/icons-material/EventOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { useOpenSlots, useBookSlot, formatSlotTime } from "@/features/slots/hooks";
import { errorMessage, type Slot } from "@/features/slots/api";
import { useDocumentTitle } from "@/lib/useDocumentTitle";

interface BookingFormProps {
  slot: Slot;
  designId: string | null;
  onClose: () => void;
}

function BookingForm({ slot, designId, onClose }: BookingFormProps) {
  const book = useBookSlot();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);

    if (!email.trim() || !name.trim() || !phone.trim()) {
      setFormError("Please fill in all fields.");
      return;
    }

    book.mutate(
      {
        slotId: slot.id,
        input: {
          designId: designId ?? undefined,
          email: email.trim(),
          name: name.trim(),
          phone: phone.trim(),
        },
      },
      {
        onSuccess: (result) => {
          window.location.href = result.paymentUrl;
        },
        onError: (err) => setFormError(errorMessage(err)),
      },
    );
  };

  return (
    <Dialog open onClose={onClose} fullWidth maxWidth="sm">
      <form onSubmit={handleSubmit} noValidate>
        <DialogTitle>Book your home visit</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <Typography sx={{ color: "text.secondary" }}>
              {formatSlotTime(slot.start, slot.end)}
            </Typography>
            <TextField
              label="Full name"
              value={name}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setName(event.target.value)}
              fullWidth
              required
            />
            <TextField
              label="Email"
              type="email"
              value={email}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setEmail(event.target.value)}
              fullWidth
              required
            />
            <TextField
              label="Phone number"
              value={phone}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setPhone(event.target.value)}
              fullWidth
              required
            />
            {(formError || book.isError) && <Alert severity="error">{formError || errorMessage(book.error)}</Alert>}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" loading={book.isPending}>
            Pay deposit
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}

export function SlotsPage() {
  useDocumentTitle(
    "Book a visit",
    "Book a home or atelier fitting with Eight Two Five to be measured in person before your piece is cut.",
  );
  const [searchParams] = useSearchParams();
  const designId = searchParams.get("designId");
  const { data: slots, isLoading, error, refetch } = useOpenSlots();
  const [selectedSlot, setSelectedSlot] = useState<Slot | null>(null);

  return (
    <StorefrontLayout>
      <Container component="main" maxWidth="md" sx={{ py: { xs: 5, md: 8 } }}>
        <PageBanner
          tone="ink"
          icon={<EventOutlined />}
          breadcrumbs={[{ label: "Home", to: "/" }, { label: "Book a visit" }]}
          title="Book a home visit"
          description="Choose an open slot below. A GHS 500 deposit confirms your booking and counts toward your garment."
        />
        <Box sx={{ mt: { xs: 4, md: 5 } }} />

        {isLoading && (
          <Stack spacing={2}>
            <Skeleton variant="rectangular" height={80} />
            <Skeleton variant="rectangular" height={80} />
            <Skeleton variant="rectangular" height={80} />
          </Stack>
        )}

        {error && (
          <ErrorState
            message={errorMessage(error, "Could not load available slots.")}
            onRetry={() => refetch()}
          />
        )}

        {!isLoading && !error && (slots ?? []).length === 0 && (
          <EmptyState
            label="No open slots"
            title="No visit slots are open right now."
            body="Fittings are added as the calendar opens up. Check back soon, or order online and add a custom size note instead."
            action={{ label: "Browse the store", to: "/store" }}
          />
        )}

        {!isLoading && !error && (slots ?? []).length > 0 && (
          <FormControl component="fieldset" fullWidth>
            <RadioGroup value={selectedSlot?.id ?? ""}>
              <Stack spacing={2}>
                {(slots ?? []).map((slot) => (
                  <Card
                    key={slot.id}
                    variant="outlined"
                    sx={{
                      cursor: "pointer",
                      transition: "border-color 0.2s",
                      "&:hover": { borderColor: "text.primary" },
                    }}
                    onClick={() => setSelectedSlot(slot)}
                  >
                    <CardContent>
                      <Stack direction="row" spacing={2} sx={{ alignItems: "center" }}>
                        <Radio value={slot.id} checked={selectedSlot?.id === slot.id} />
                        <Box sx={{ flex: 1 }}>
                          <Typography variant="body1" sx={{ fontWeight: "medium" }}>
                            {formatSlotTime(slot.start, slot.end)}
                          </Typography>
                        </Box>
                      </Stack>
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            </RadioGroup>
          </FormControl>
        )}

        <Box sx={{ mt: 4 }}>
          <Button
            variant="contained"
            disabled={!selectedSlot}
            onClick={() => selectedSlot && setSelectedSlot(selectedSlot)}
          >
            Continue
          </Button>
        </Box>

        {selectedSlot && (
          <BookingForm slot={selectedSlot} designId={designId} onClose={() => setSelectedSlot(null)} />
        )}
      </Container>
    </StorefrontLayout>
  );
}
