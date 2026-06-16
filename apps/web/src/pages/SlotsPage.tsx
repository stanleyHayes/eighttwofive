import { useState, type ChangeEvent, type FormEvent } from "react";
import { useSearchParams } from "react-router";
import { useTranslation } from "react-i18next";
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
import {
  formatGhanaPhone,
  isValidGhanaPhone,
  normalizeGhanaPhone,
} from "@/lib/phone";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { amber, brass, monoFamily } from "@/theme";

interface BookingFormProps {
  slot: Slot;
  designId: string | null;
  onClose: () => void;
}

function BookingForm({ slot, designId, onClose }: BookingFormProps) {
  const { t } = useTranslation();
  const book = useBookSlot();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);

    if (!email.trim() || !name.trim() || !phone.trim()) {
      setFormError(t("slots.errorAllFields"));
      return;
    }

    if (!isValidGhanaPhone(phone)) {
      setFormError(t("slots.errorInvalidPhone"));
      return;
    }

    book.mutate(
      {
        slotId: slot.id,
        input: {
          designId: designId ?? undefined,
          email: email.trim(),
          name: name.trim(),
          phone: normalizeGhanaPhone(phone),
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
              {t("slots.dialogEyebrow")}
            </Typography>
            <Typography variant="h4" component="p" sx={{ mt: 1 }}>
              {t("slots.dialogTitle")}
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
                {t("slots.selectedSlot")}
              </Typography>
              <Typography sx={{ mt: 0.5 }}>
                {formatSlotTime(slot.start, slot.end)}
              </Typography>
            </Box>
            <TextField
              label={t("slots.fullName")}
              value={name}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                setName(event.target.value)
              }
              fullWidth
              required
            />
            <TextField
              label={t("slots.email")}
              type="email"
              value={email}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                setEmail(event.target.value)
              }
              fullWidth
              required
            />
            <TextField
              label={t("slots.phoneNumber")}
              value={phone}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                setPhone(formatGhanaPhone(event.target.value))
              }
              fullWidth
              required
              inputMode="tel"
              placeholder="024 123 4567"
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
            {t("slots.cancel")}
          </Button>
          <Button type="submit" variant="contained" loading={book.isPending}>
            {t("slots.payDeposit")}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
}

export function SlotsPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("slots.docTitle"), t("slots.docDescription"));
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
          breadcrumbs={[
            { label: t("slots.breadcrumbHome"), to: "/" },
            { label: t("slots.breadcrumbBookVisit") },
          ]}
          title={t("slots.bannerTitle")}
          description={t("slots.bannerDescription")}
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
            message={errorMessage(error, t("slots.errorLoad"))}
            onRetry={() => refetch()}
          />
        )}

        {!isLoading && !error && openSlots.length === 0 && (
          <EmptyState
            label={t("slots.emptyLabel")}
            title={t("slots.emptyTitle")}
            body={t("slots.emptyBody")}
            action={{ label: t("slots.emptyAction"), to: "/store" }}
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
                {t("slots.chooseTime")}
              </Typography>
              <Typography
                variant="overline"
                sx={{
                  color: "text.secondary",
                  fontVariantNumeric: "tabular-nums",
                }}
              >
                {t("slots.openCount", {
                  count: openSlots.length,
                  formatted: openSlots.length.toLocaleString("en-GH"),
                })}
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
                              {t("slots.slotLabel", {
                                number: String(index + 1).padStart(2, "0"),
                              })}
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
              {t("slots.continue")}
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
