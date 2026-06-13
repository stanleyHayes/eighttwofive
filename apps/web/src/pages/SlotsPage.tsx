import { useState, type ChangeEvent, type FormEvent } from "react";
import { useSearchParams } from "react-router";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
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
import {
  useOpenSlots,
  useBookSlot,
  formatSlotTime,
} from "@/features/slots/hooks";
import { errorMessage, type Slot } from "@/features/slots/api";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { amber, brass, monoFamily } from "@/theme";

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
        <DialogTitle sx={{ p: 0 }}>
          <Box
            sx={{
              px: { xs: 3, sm: 4 },
              pt: 3.5,
              pb: 2.5,
              borderBottom: "1px solid",
              borderColor: "divider",
            }}
          >
            <Typography variant="overline" component="p" sx={{ color: brass }}>
              home visit
            </Typography>
            <Typography variant="h4" component="p" sx={{ mt: 1 }}>
              Book your fitting
            </Typography>
          </Box>
        </DialogTitle>
        <DialogContent sx={{ px: { xs: 3, sm: 4 }, pt: "24px !important" }}>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <Box
              sx={{
                border: "1px solid",
                borderColor: "divider",
                p: 2.5,
                bgcolor: "background.default",
              }}
            >
              <Typography
                variant="overline"
                component="p"
                sx={{ color: "text.secondary" }}
              >
                selected slot
              </Typography>
              <Typography sx={{ mt: 0.5 }}>
                {formatSlotTime(slot.start, slot.end)}
              </Typography>
            </Box>
            <TextField
              label="Full name"
              value={name}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                setName(event.target.value)
              }
              fullWidth
              required
            />
            <TextField
              label="Email"
              type="email"
              value={email}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                setEmail(event.target.value)
              }
              fullWidth
              required
            />
            <TextField
              label="Phone number"
              value={phone}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                setPhone(event.target.value)
              }
              fullWidth
              required
            />
            {(formError || book.isError) && (
              <Alert severity="error">
                {formError || errorMessage(book.error)}
              </Alert>
            )}
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: { xs: 3, sm: 4 }, pb: 3.5 }}>
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
  const [bookingSlot, setBookingSlot] = useState<Slot | null>(null);
  const openSlots = slots ?? [];

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 5, md: 8 }, maxWidth: 820 }}>
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

        {!isLoading && !error && openSlots.length === 0 && (
          <EmptyState
            label="No open slots"
            title="No visit slots are open right now."
            body="Fittings are added as the calendar opens up. Check back soon, or order online and add a custom size note instead."
            action={{ label: "Browse the store", to: "/store" }}
          />
        )}

        {!isLoading && !error && openSlots.length > 0 && (
          <FormControl component="fieldset" fullWidth>
            <Stack
              direction={{ xs: "column", sm: "row" }}
              spacing={1}
              sx={{
                justifyContent: "space-between",
                alignItems: { sm: "baseline" },
                mb: 2,
              }}
            >
              <Typography
                variant="overline"
                component="legend"
                sx={{ color: "text.secondary" }}
              >
                choose a time
              </Typography>
              <Typography
                variant="overline"
                sx={{
                  color: "text.secondary",
                  fontVariantNumeric: "tabular-nums",
                }}
              >
                {openSlots.length.toLocaleString("en-GH")} open{" "}
                {openSlots.length === 1 ? "slot" : "slots"}
              </Typography>
            </Stack>
            <RadioGroup
              value={selectedSlot?.id ?? ""}
              onChange={(event) => {
                const next = openSlots.find(
                  (slot) => slot.id === event.target.value,
                );
                if (next) setSelectedSlot(next);
              }}
            >
              <Stack spacing={2}>
                {openSlots.map((slot, index) => {
                  const selected = selectedSlot?.id === slot.id;
                  return (
                    <Card
                      key={slot.id}
                      variant="outlined"
                      sx={{
                        cursor: "pointer",
                        bgcolor: selected ? "background.paper" : "transparent",
                        borderColor: selected ? amber : "divider",
                        transition:
                          "border-color 180ms ease, background-color 180ms ease",
                        "&:hover": {
                          borderColor: selected ? amber : "text.primary",
                        },
                      }}
                      onClick={() => setSelectedSlot(slot)}
                    >
                      <CardContent>
                        <Stack
                          direction="row"
                          spacing={2}
                          sx={{ alignItems: "center" }}
                        >
                          <Radio
                            value={slot.id}
                            checked={selectedSlot?.id === slot.id}
                          />
                          <Box sx={{ flex: 1 }}>
                            <Typography
                              variant="overline"
                              component="p"
                              sx={{
                                color: selected ? brass : "text.secondary",
                                fontFamily: monoFamily,
                              }}
                            >
                              Slot {String(index + 1).padStart(2, "0")}
                            </Typography>
                            <Typography
                              variant="body1"
                              sx={{ fontWeight: "medium" }}
                            >
                              {formatSlotTime(slot.start, slot.end)}
                            </Typography>
                          </Box>
                        </Stack>
                      </CardContent>
                    </Card>
                  );
                })}
              </Stack>
            </RadioGroup>
          </FormControl>
        )}

        {!isLoading && !error && openSlots.length > 0 && (
          <Box sx={{ mt: 4 }}>
            <Button
              variant="contained"
              disabled={!selectedSlot}
              onClick={() => {
                if (selectedSlot) setBookingSlot(selectedSlot);
              }}
              sx={{ width: { xs: "100%", sm: "auto" } }}
            >
              Continue
            </Button>
          </Box>
        )}

        {bookingSlot && (
          <BookingForm
            slot={bookingSlot}
            designId={designId}
            onClose={() => setBookingSlot(null)}
          />
        )}
      </Box>
    </StorefrontLayout>
  );
}
