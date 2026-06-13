import { useMemo, useState, type ChangeEvent, type FormEvent } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import IconButton from "@mui/material/IconButton";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import SettingsOutlined from "@mui/icons-material/SettingsOutlined";
import { PageBanner } from "@/components/PageBanner";
import { useSettings, useUpdateSettings, validateSettings, parseGhs, formatPesewas } from "@/features/settings/hooks";
import { errorMessage } from "@/features/settings/api";
import type { DeliveryRate, Settings, SettingsInput } from "@/features/settings/api";

function pesewasToGhsInput(pesewas: number): string {
  return (pesewas / 100).toFixed(2);
}

interface SettingsFormProps {
  initial: Settings;
}

function SettingsForm({ initial }: SettingsFormProps) {
  const update = useUpdateSettings();

  const [depositGhs, setDepositGhs] = useState(pesewasToGhsInput(initial.depositPesewas));
  const [whatsapp, setWhatsapp] = useState(initial.whatsappNumber);
  const [visitLocation, setVisitLocation] = useState(initial.visitLocation);
  const [rates, setRates] = useState<DeliveryRate[]>(
    initial.deliveryRates.length > 0 ? initial.deliveryRates : [{ area: "", ratePesewas: 0 }],
  );
  const [formError, setFormError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const workingSettings: SettingsInput = useMemo(
    () => ({
      depositPesewas: parseGhs(depositGhs) ?? 0,
      whatsappNumber: whatsapp.trim(),
      visitLocation: visitLocation.trim(),
      deliveryRates: rates,
    }),
    [depositGhs, whatsapp, visitLocation, rates],
  );

  const handleRateAreaChange = (index: number, value: string) => {
    setRates((prev) => prev.map((r, i) => (i === index ? { ...r, area: value } : r)));
    setSaved(false);
  };

  const handleRateAmountChange = (index: number, value: string) => {
    const pesewas = parseGhs(value) ?? 0;
    setRates((prev) => prev.map((r, i) => (i === index ? { ...r, ratePesewas: pesewas } : r)));
    setSaved(false);
  };

  const handleAddRate = () => {
    setRates((prev) => [...prev, { area: "", ratePesewas: 0 }]);
    setSaved(false);
  };

  const handleRemoveRate = (index: number) => {
    setRates((prev) => prev.filter((_, i) => i !== index));
    setSaved(false);
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    setSaved(false);

    const validation = validateSettings(workingSettings);
    if (validation) {
      setFormError(validation);
      return;
    }

    update.mutate(workingSettings, {
      onSuccess: () => setSaved(true),
      onError: (err) => setFormError(errorMessage(err)),
    });
  };

  return (
    <form onSubmit={handleSubmit} noValidate>
      <Stack spacing={4}>
        <Stack spacing={2}>
          <Typography variant="h5" component="h2">
            Deposit
          </Typography>
          <TextField
            label="Home-visit deposit (GHS)"
            type="number"
            slotProps={{ htmlInput: { min: 0, step: "0.01" } }}
            value={depositGhs}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setDepositGhs(event.target.value);
              setSaved(false);
            }}
            helperText="This is the amount customers pay to book a home measurement visit."
            fullWidth
          />
        </Stack>

        <Divider />

        <Stack spacing={2}>
          <Typography variant="h5" component="h2">
            Contact
          </Typography>
          <TextField
            label="WhatsApp number"
            value={whatsapp}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setWhatsapp(event.target.value);
              setSaved(false);
            }}
            helperText="Shown to customers on the contact page and order details."
            fullWidth
          />
          <TextField
            label="Visit location / address"
            value={visitLocation}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              setVisitLocation(event.target.value);
              setSaved(false);
            }}
            helperText="Where customers can come for workplace appointments."
            multiline
            minRows={2}
            fullWidth
          />
        </Stack>

        <Divider />

        <Stack spacing={2}>
          <Typography variant="h5" component="h2">
            Delivery rates
          </Typography>
          <Typography sx={{ color: "text.secondary" }}>
            Add a flat dispatch rate for each area. Areas not listed here will be arranged directly
            with the customer.
          </Typography>

          <Stack spacing={1}>
            {rates.map((rate, index) => (
              <Stack key={index} direction="row" spacing={1} sx={{ alignItems: "flex-start" }}>
                <TextField
                  label="Area"
                  value={rate.area}
                  onChange={(event: ChangeEvent<HTMLInputElement>) =>
                    handleRateAreaChange(index, event.target.value)
                  }
                  fullWidth
                  sx={{ flex: 2 }}
                />
                <TextField
                  label="Rate (GHS)"
                  type="number"
                  slotProps={{ htmlInput: { min: 0, step: "0.01" } }}
                  value={rate.ratePesewas === 0 ? "" : pesewasToGhsInput(rate.ratePesewas)}
                  onChange={(event: ChangeEvent<HTMLInputElement>) =>
                    handleRateAmountChange(index, event.target.value)
                  }
                  sx={{ flex: 1 }}
                />
                <IconButton
                  type="button"
                  onClick={() => handleRemoveRate(index)}
                  aria-label={`Remove ${rate.area || "rate"} row`}
                  sx={{ mt: 1 }}
                >
                  <DeleteIcon />
                </IconButton>
              </Stack>
            ))}
          </Stack>

          <Box>
            <Button
              type="button"
              variant="outlined"
              startIcon={<AddIcon />}
              onClick={handleAddRate}
              disabled={update.isPending}
            >
              Add area
            </Button>
          </Box>
        </Stack>

        {(formError || update.isError) && (
          <Alert severity="error">{formError || errorMessage(update.error)}</Alert>
        )}
        {saved && <Alert severity="success">Settings saved.</Alert>}

        <Box>
          <Button type="submit" variant="contained" loading={update.isPending}>
            Save settings
          </Button>
        </Box>
      </Stack>

      <Box sx={{ mt: 4, pt: 2, borderTop: "1px solid", borderColor: "divider" }}>
        <Typography variant="body2" sx={{ color: "text.secondary" }}>
          Preview: deposit = {formatPesewas(workingSettings.depositPesewas)},{" "}
          {workingSettings.deliveryRates.length} delivery rate
          {workingSettings.deliveryRates.length === 1 ? "" : "s"} configured.
        </Typography>
      </Box>
    </form>
  );
}

export function AdminSettingsPage() {
  const { data, isLoading, error: loadError } = useSettings();

  if (isLoading) {
    return (
      <Box sx={{ maxWidth: 720 }}>
        <Skeleton variant="text" width={120} sx={{ mb: 1 }} />
        <Skeleton variant="rectangular" height={48} sx={{ mb: 2 }} />
        <Skeleton variant="rectangular" height={120} />
      </Box>
    );
  }

  if (loadError) {
    return (
      <Box sx={{ maxWidth: 720 }}>
        <Alert severity="error">{errorMessage(loadError, "Could not load settings.")}</Alert>
      </Box>
    );
  }

  const settings = data;
  if (!settings) {
    return (
      <Box sx={{ maxWidth: 720 }}>
        <Alert severity="error">Could not load settings.</Alert>
      </Box>
    );
  }

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<SettingsOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Settings" }]}
        title="Settings"
        description="Store-wide configuration — the home-visit deposit, contact details, and delivery rates."
      />
      <Box sx={{ maxWidth: 720, mt: { xs: 4, md: 5 } }}>
        <SettingsForm initial={settings} />
      </Box>
    </Box>
  );
}
