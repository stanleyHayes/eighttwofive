import { useEffect, useMemo, useState } from "react";
import Alert from "@mui/material/Alert";
import Autocomplete from "@mui/material/Autocomplete";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import Collapse from "@mui/material/Collapse";
import Divider from "@mui/material/Divider";
import FormControl from "@mui/material/FormControl";
import InputAdornment from "@mui/material/InputAdornment";
import FormControlLabel from "@mui/material/FormControlLabel";
import FormHelperText from "@mui/material/FormHelperText";
import Radio from "@mui/material/Radio";
import RadioGroup from "@mui/material/RadioGroup";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import Typography from "@mui/material/Typography";
import CheckOutlined from "@mui/icons-material/CheckOutlined";
import { Link as RouterLink, useNavigate, useParams } from "react-router";
import { ErrorState } from "@/components/EmptyState";
import { JsonLd } from "@/components/JsonLd";
import { MeasureRule } from "@/components/MeasureRule";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import type { Design } from "@/features/catalog/api";
import { errorMessage } from "@/features/catalog/api";
import { formatPesewas } from "@/features/catalog/money";
import {
  useCreateCustomRequest,
  useCreateStandardOrder,
} from "@/features/orders/hooks";
import {
  DETAIL_TRANSFORM,
  THUMB_TRANSFORM,
  photoUrl,
  sortedPhotos,
  type PublicSettings,
} from "@/features/storefront/api";
import { DesignGrid, PhotoPlaceholder } from "@/features/storefront/DesignCard";
import { RetiredPanel } from "@/features/storefront/RetiredPanel";
import {
  usePublicDesign,
  usePublicDesigns,
  usePublicSettings,
} from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import {
  formatGhanaPhone,
  isValidGhanaPhone,
  normalizeGhanaPhone,
} from "@/lib/phone";
import { ApiError } from "@/lib/api";
import { amber, clayDeep, noir, noirAlpha50, sandDeep, stone } from "@/theme";

/** Canonical public origin, used for structured-data URLs. */
const SITE_ORIGIN = "https://eighttwofive.vercel.app";

type MeasurementKey = "bust" | "waist" | "hips" | "length";

const defaultMeasurements: Record<MeasurementKey, string> = {
  bust: "",
  waist: "",
  hips: "",
  length: "",
};

function Gallery({ design, cloudName }: { design: Design; cloudName: string }) {
  const photos = sortedPhotos(design);
  const [selected, setSelected] = useState(0);
  const current = photos[selected] ?? photos[0];

  if (!current || !cloudName) {
    return <PhotoPlaceholder name={design.name} />;
  }

  return (
    <Box>
      <Box
        sx={{
          border: "1px solid",
          borderColor: "divider",
          bgcolor: sandDeep,
          overflow: "hidden",
        }}
      >
        <Box
          component="img"
          src={photoUrl(cloudName, current.publicId, DETAIL_TRANSFORM)}
          alt={`${design.name} — photo ${selected + 1} of ${photos.length}`}
          loading="lazy"
          decoding="async"
          sx={{ width: "100%", display: "block", bgcolor: sandDeep }}
        />
      </Box>
      {photos.length > 1 && (
        <Stack direction="row" spacing={1} sx={{ mt: 1.5, flexWrap: "wrap" }}>
          {photos.map((photo, index) => (
            <Box
              key={photo.publicId}
              component="button"
              type="button"
              aria-label={`Show photo ${index + 1}`}
              aria-pressed={index === selected}
              onClick={() => setSelected(index)}
              sx={{
                p: 0,
                cursor: "pointer",
                bgcolor: "transparent",
                border: "1px solid",
                borderColor: index === selected ? amber : "divider",
                opacity: index === selected ? 1 : 0.75,
                transition: "border-color 160ms ease, opacity 160ms ease",
                "&:hover": {
                  opacity: 1,
                  borderColor: index === selected ? amber : noirAlpha50,
                },
                "&:focus-visible": {
                  outline: `2px solid ${amber}`,
                  outlineOffset: "2px",
                },
              }}
            >
              <Box
                component="img"
                src={photoUrl(cloudName, photo.publicId, THUMB_TRANSFORM)}
                alt=""
                loading="lazy"
                decoding="async"
                sx={{
                  width: 60,
                  height: 78,
                  objectFit: "cover",
                  display: "block",
                }}
              />
            </Box>
          ))}
        </Stack>
      )}
    </Box>
  );
}

function CopyLinkButton() {
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!copied && !error) return;
    const timer = setTimeout(() => {
      setCopied(false);
      setError(false);
    }, 2000);
    return () => clearTimeout(timer);
  }, [copied, error]);

  return (
    <Button
      variant="outlined"
      startIcon={copied ? <CheckOutlined /> : undefined}
      color={error ? "error" : "primary"}
      onClick={() => {
        navigator.clipboard
          .writeText(window.location.href)
          .then(() => setCopied(true))
          .catch(() => {
            setError(true);
          });
      }}
    >
      {error ? "Could not copy link" : copied ? "Link copied" : "Copy link"}
    </Button>
  );
}

const MEASUREMENT_LABELS: Record<MeasurementKey, string> = {
  bust: "Bust",
  waist: "Waist",
  hips: "Hips",
  length: "Length",
};

/** Keeps only digits and a single decimal point — measurements are numbers in cm. */
function sanitizeCm(raw: string): string {
  const cleaned = raw.replace(/[^\d.]/g, "");
  const [whole, ...rest] = cleaned.split(".");
  return rest.length > 0 ? `${whole}.${rest.join("")}` : whole;
}

function MeasurementForm({
  values,
  onChange,
}: {
  values: Record<MeasurementKey, string>;
  onChange: (values: Record<MeasurementKey, string>) => void;
}) {
  return (
    <Box>
      <Typography variant="body2" sx={{ color: "text.secondary", mb: 1.5 }}>
        Enter your measurements in centimetres — the atelier confirms the final
        fit with you before cutting.
      </Typography>
      <Box
        sx={{
          display: "grid",
          gridTemplateColumns: { xs: "1fr 1fr", sm: "repeat(4, 1fr)" },
          gap: 1.5,
        }}
      >
        {(Object.keys(defaultMeasurements) as MeasurementKey[]).map((key) => (
          <TextField
            key={key}
            label={MEASUREMENT_LABELS[key]}
            value={values[key] ?? ""}
            onChange={(event) =>
              onChange({ ...values, [key]: sanitizeCm(event.target.value) })
            }
            size="small"
            slotProps={{
              htmlInput: { inputMode: "decimal" },
              input: {
                endAdornment: (
                  <InputAdornment position="end">cm</InputAdornment>
                ),
              },
            }}
          />
        ))}
      </Box>
    </Box>
  );
}

function DeliverySelector({
  mode,
  area,
  rates,
  onModeChange,
  onAreaChange,
}: {
  mode: "pickup" | "dispatch";
  area: string;
  rates: { area: string; ratePesewas: number }[];
  onModeChange: (mode: "pickup" | "dispatch") => void;
  onAreaChange: (area: string) => void;
}) {
  return (
    <FormControl component="fieldset" fullWidth>
      <RadioGroup
        value={mode}
        onChange={(event) =>
          onModeChange(event.target.value as "pickup" | "dispatch")
        }
      >
        <Stack spacing={1}>
          <FormControlLabel
            value="pickup"
            control={<Radio />}
            label="Pickup — free, collect when ready"
          />
          <FormControlLabel
            value="dispatch"
            control={<Radio />}
            label="Dispatch to my area"
          />
          {mode === "dispatch" && (
            <Box sx={{ pl: 4 }}>
              <Autocomplete
                freeSolo
                autoHighlight
                options={rates.map((rate) => rate.area)}
                value={area}
                onChange={(_event, value) => onAreaChange(value ?? "")}
                onInputChange={(_event, value) => onAreaChange(value)}
                renderInput={(params) => (
                  <TextField
                    {...params}
                    label="Area / neighbourhood"
                    placeholder="Start typing your area"
                    fullWidth
                    helperText={
                      rates.length > 0
                        ? "Pick a listed area for a set rate, or type your own — we'll arrange delivery."
                        : "Enter your area; delivery will be arranged directly if no rate is set."
                    }
                  />
                )}
              />
            </Box>
          )}
        </Stack>
      </RadioGroup>
    </FormControl>
  );
}

function CustomerFields({
  name,
  email,
  phone,
  onNameChange,
  onEmailChange,
  onPhoneChange,
}: {
  name: string;
  email: string;
  phone: string;
  onNameChange: (value: string) => void;
  onEmailChange: (value: string) => void;
  onPhoneChange: (value: string) => void;
}) {
  return (
    <Stack spacing={2}>
      <TextField
        label="Full name"
        value={name}
        onChange={(event) => onNameChange(event.target.value)}
        fullWidth
        required
      />
      <TextField
        label="Email"
        type="email"
        value={email}
        onChange={(event) => onEmailChange(event.target.value)}
        fullWidth
        required
      />
      <TextField
        label="Phone number"
        type="tel"
        value={phone}
        onChange={(event) =>
          onPhoneChange(formatGhanaPhone(event.target.value))
        }
        fullWidth
        required
        error={phone.trim() !== "" && !isValidGhanaPhone(phone)}
        helperText={
          phone.trim() !== "" && !isValidGhanaPhone(phone)
            ? "Enter a Ghana number like 024 123 4567 or +233 24 123 4567."
            : "Mobile or landline — e.g. 024 123 4567. We reach you here and on WhatsApp."
        }
        slotProps={{ htmlInput: { inputMode: "tel" } }}
      />
    </Stack>
  );
}

function DesignDetail({
  design,
  settings,
}: {
  design: Design;
  settings: PublicSettings | undefined;
}) {
  const navigate = useNavigate();
  const bands = design.sizeBands;
  const [bandLabel, setBandLabel] = useState(bands[0]?.label ?? "");
  const band = bands.find((entry) => entry.label === bandLabel) ?? bands[0];
  const chartEntries = band ? Object.entries(band.chart) : [];
  const cloudName = settings?.cloudName ?? "";
  const relatedDesignsQuery = usePublicDesigns({
    collectionId: design.collectionId,
  });
  const relatedDesigns = useMemo(
    () =>
      (relatedDesignsQuery.data ?? [])
        .filter((entry) => entry.id !== design.id)
        .slice()
        .sort(
          (a, b) =>
            new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
        )
        .slice(0, 4),
    [design.id, relatedDesignsQuery.data],
  );

  // Structured data for search engines that render the SPA: a Product (with a
  // made-to-order price range) and a breadcrumb trail for rich results.
  const structuredData = useMemo(() => {
    const prices = design.sizeBands
      .map((entry) => entry.pricePesewas)
      .filter((price) => price > 0);
    const images = cloudName
      ? sortedPhotos(design).map((photo) =>
          photoUrl(cloudName, photo.publicId, DETAIL_TRANSFORM),
        )
      : [];

    const product: Record<string, unknown> = {
      "@context": "https://schema.org",
      "@type": "Product",
      name: design.name,
      description:
        design.note || `${design.name} — made-to-measure by Eight Two Five.`,
      brand: { "@type": "Brand", name: "Eight Two Five" },
    };
    if (images.length > 0) product.image = images;
    if (prices.length > 0) {
      product.offers = {
        "@type": "AggregateOffer",
        priceCurrency: "GHS",
        lowPrice: Math.min(...prices) / 100,
        highPrice: Math.max(...prices) / 100,
        availability: "https://schema.org/MadeToOrder",
      };
    }

    const breadcrumb = {
      "@context": "https://schema.org",
      "@type": "BreadcrumbList",
      itemListElement: [
        {
          "@type": "ListItem",
          position: 1,
          name: "Home",
          item: `${SITE_ORIGIN}/`,
        },
        {
          "@type": "ListItem",
          position: 2,
          name: "Store",
          item: `${SITE_ORIGIN}/store`,
        },
        {
          "@type": "ListItem",
          position: 3,
          name: design.name,
          item: `${SITE_ORIGIN}/designs/${design.slug}`,
        },
      ],
    };

    return [product, breadcrumb];
  }, [design, cloudName]);

  const [customSizeOpen, setCustomSizeOpen] = useState(false);
  const [sizeMode, setSizeMode] = useState<"self" | "home_visit" | "workplace">(
    "self",
  );
  const [measurements, setMeasurements] =
    useState<Record<MeasurementKey, string>>(defaultMeasurements);
  const [designChangeOpen, setDesignChangeOpen] = useState(false);
  const [designChange, setDesignChange] = useState("");

  const [customerName, setCustomerName] = useState("");
  const [customerEmail, setCustomerEmail] = useState("");
  const [customerPhone, setCustomerPhone] = useState("");

  const [deliveryMode, setDeliveryMode] = useState<"pickup" | "dispatch">(
    "pickup",
  );
  const [deliveryArea, setDeliveryArea] = useState("");

  const [formError, setFormError] = useState<string | null>(null);
  const [submittedRef, setSubmittedRef] = useState<string | null>(null);

  const standardOrder = useCreateStandardOrder();
  const customRequest = useCreateCustomRequest();

  const isCustom =
    customSizeOpen || designChangeOpen || designChange.trim() !== "";
  const isHomeVisit = customSizeOpen && sizeMode === "home_visit";
  const requestSizeMode = customSizeOpen ? sizeMode : "band";

  const deliveryValue = useMemo(() => {
    if (deliveryMode === "pickup") return "pickup";
    return `dispatch:${deliveryArea.trim() || "unknown"}`;
  }, [deliveryMode, deliveryArea]);

  const resetForm = () => {
    setFormError(null);
  };

  const validateCommon = () => {
    if (!customerName.trim()) return "Please enter your full name.";
    if (!customerEmail.trim()) return "Please enter your email.";
    if (!customerPhone.trim()) return "Please enter your phone number.";
    if (!isValidGhanaPhone(customerPhone)) {
      return "Please enter a valid Ghana phone number, e.g. 024 123 4567.";
    }
    if (deliveryMode === "dispatch" && !deliveryArea.trim()) {
      return "Please enter a delivery area.";
    }
    return null;
  };

  const handleStandardOrder = () => {
    resetForm();
    const error = validateCommon();
    if (error) {
      setFormError(error);
      return;
    }
    if (!band) {
      setFormError("Please select a size band.");
      return;
    }

    standardOrder.mutate(
      {
        designId: design.id,
        bandLabel: band.label,
        delivery: deliveryValue,
        customerPhone: normalizeGhanaPhone(customerPhone),
        email: customerEmail.trim(),
        name: customerName.trim(),
      },
      {
        onSuccess: (result) => {
          window.location.href = result.paymentUrl;
        },
        onError: (err) => setFormError(errorMessage(err)),
      },
    );
  };

  const handleCustomRequest = () => {
    resetForm();
    const error = validateCommon();
    if (error) {
      setFormError(error);
      return;
    }

    if (requestSizeMode === "self") {
      const missing = (
        Object.keys(defaultMeasurements) as MeasurementKey[]
      ).filter((key) => !measurements[key]?.trim());
      if (missing.length > 0) {
        setFormError(`Please fill in all measurements: ${missing.join(", ")}.`);
        return;
      }
    }

    customRequest.mutate(
      {
        designId: design.id,
        sizeMode: requestSizeMode,
        measurements: requestSizeMode === "self" ? measurements : undefined,
        bandLabel: band && requestSizeMode === "band" ? band.label : undefined,
        designChange: designChangeOpen ? designChange.trim() : undefined,
        delivery: deliveryValue,
        customerPhone: normalizeGhanaPhone(customerPhone),
        email: customerEmail.trim(),
        name: customerName.trim(),
      },
      {
        onSuccess: (result) => {
          // Checkout is anonymous (no session); confirm inline and tell the
          // customer to use the emailed link rather than bouncing to the
          // auth-gated account page.
          setSubmittedRef(result.order.ref);
        },
        onError: (err) => setFormError(errorMessage(err)),
      },
    );
  };

  const handleHomeVisit = () => {
    resetForm();
    const error = validateCommon();
    if (error) {
      setFormError(error);
      return;
    }
    navigate(`/slots?designId=${encodeURIComponent(design.id)}`);
  };

  const submitLabel = isHomeVisit
    ? "Book a home visit"
    : isCustom
      ? "Send request"
      : "Order this design";

  const isSubmitting = standardOrder.isPending || customRequest.isPending;

  return (
    <>
      <Box
        sx={{
          py: { xs: 4, md: 8 },
          mb: { xs: 6, md: 10 },
          display: "grid",
          gridTemplateColumns: { xs: "1fr", md: "7fr 5fr" },
          gap: { xs: 3.5, md: 7 },
          alignItems: "start",
        }}
      >
        {structuredData.map((data, index) => (
          <JsonLd key={index} data={data} />
        ))}
        <Gallery design={design} cloudName={cloudName} />

        <Box
          sx={{
            bgcolor: "background.paper",
            border: "1px solid",
            borderColor: "divider",
            p: { xs: 3, sm: 4 },
          }}
        >
          <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
            made to measure
          </Typography>
          <Typography variant="h2" component="h1" sx={{ mt: 1.5 }}>
            {design.name}
          </Typography>
          {design.note && (
            <Typography
              sx={{ color: "text.secondary", mt: 2, maxWidth: "48ch" }}
            >
              {design.note}
            </Typography>
          )}

          {bands.length > 0 && band ? (
            <>
              <Typography
                variant="overline"
                component="p"
                sx={{ color: "text.secondary", mt: 4 }}
              >
                size band
              </Typography>
              <ToggleButtonGroup
                exclusive
                value={band.label}
                onChange={(_, next: string | null) => {
                  if (next !== null) setBandLabel(next);
                }}
                aria-label="size band"
                sx={{ mt: 1, flexWrap: "wrap", gap: 1 }}
              >
                {bands.map((entry) => (
                  <ToggleButton
                    key={entry.label}
                    value={entry.label}
                    sx={{
                      px: 2.5,
                      py: 1,
                      border: "1px solid",
                      borderColor: noirAlpha50,
                      color: "text.primary",
                      "&.Mui-selected": {
                        bgcolor: noir,
                        color: "common.white",
                        "&:hover": { bgcolor: noir },
                      },
                    }}
                  >
                    {entry.label}
                  </ToggleButton>
                ))}
              </ToggleButtonGroup>

              {!isCustom && (
                <>
                  <Typography
                    variant="h5"
                    component="p"
                    sx={{
                      mt: 3,
                      pt: 2,
                      borderTop: "1px solid",
                      borderColor: "divider",
                      fontVariantNumeric: "tabular-nums",
                    }}
                  >
                    {formatPesewas(band.pricePesewas)}
                  </Typography>
                </>
              )}

              {isCustom && (
                <Alert severity="info" sx={{ mt: 3 }}>
                  Custom requests are quoted directly — no price shown until the
                  merchant replies.
                </Alert>
              )}

              <Typography
                variant="overline"
                component="p"
                sx={{ color: "text.secondary", mt: 4 }}
              >
                size chart — band {band.label}
              </Typography>
              {chartEntries.length > 0 ? (
                <Table
                  size="small"
                  sx={{
                    mt: 1,
                    maxWidth: 360,
                    borderTop: "1px solid",
                    borderBottom: "1px solid",
                    borderColor: "divider",
                  }}
                >
                  <TableBody>
                    {chartEntries.map(([measure, value]) => (
                      <TableRow key={measure}>
                        <TableCell
                          sx={{
                            color: "text.secondary",
                            textTransform: "capitalize",
                            pl: 0,
                          }}
                        >
                          {measure}
                        </TableCell>
                        <TableCell align="right" sx={{ pr: 0 }}>
                          {value}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <Typography
                  variant="body2"
                  sx={{ color: "text.secondary", mt: 1 }}
                >
                  The chart for this band is being finalised — every piece is
                  still cut to your measurements.
                </Typography>
              )}
            </>
          ) : (
            <Typography variant="body2" sx={{ color: "text.secondary", mt: 4 }}>
              Pricing for this design is being finalised.
            </Typography>
          )}

          <Stack
            spacing={2.5}
            sx={{
              mt: 4,
              pt: 4,
              borderTop: "1px solid",
              borderColor: "divider",
            }}
          >
            <FormControlLabel
              control={
                <Checkbox
                  checked={customSizeOpen}
                  onChange={(event) => {
                    setCustomSizeOpen(event.target.checked);
                    if (!event.target.checked) setSizeMode("self");
                  }}
                />
              }
              label="My size isn't listed"
            />

            <Collapse in={customSizeOpen}>
              <FormControl component="fieldset" fullWidth>
                <RadioGroup
                  value={sizeMode}
                  onChange={(event) =>
                    setSizeMode(event.target.value as typeof sizeMode)
                  }
                >
                  <Stack spacing={1}>
                    <FormControlLabel
                      value="self"
                      control={<Radio />}
                      label="Measure yourself"
                    />
                    <Collapse in={sizeMode === "self"}>
                      <Box sx={{ pl: 4 }}>
                        <MeasurementForm
                          values={measurements}
                          onChange={setMeasurements}
                        />
                      </Box>
                    </Collapse>

                    <FormControlLabel
                      value="home_visit"
                      control={<Radio />}
                      label="Book a home visit (GHS 500 deposit)"
                    />
                    <Collapse in={sizeMode === "home_visit"}>
                      <Box sx={{ pl: 4 }}>
                        <Typography
                          variant="body2"
                          sx={{ color: "text.secondary" }}
                        >
                          Choose an open slot and pay a deposit that counts
                          toward your garment.
                        </Typography>
                      </Box>
                    </Collapse>

                    <FormControlLabel
                      value="workplace"
                      control={<Radio />}
                      label="Come to the workplace"
                    />
                    <Collapse in={sizeMode === "workplace"}>
                      <Box sx={{ pl: 4 }}>
                        <Typography
                          variant="body2"
                          sx={{ color: "text.secondary" }}
                        >
                          Visit the Eight Two Five workspace — no booking or
                          deposit needed.
                        </Typography>
                      </Box>
                    </Collapse>
                  </Stack>
                </RadioGroup>
              </FormControl>
            </Collapse>

            <Divider />

            <FormControlLabel
              control={
                <Checkbox
                  checked={designChangeOpen}
                  onChange={(event) => {
                    setDesignChangeOpen(event.target.checked);
                    if (!event.target.checked) setDesignChange("");
                  }}
                />
              }
              label="Request a design change"
            />

            <Collapse in={designChangeOpen}>
              <TextField
                label="Describe the change you want"
                value={designChange}
                onChange={(event) => setDesignChange(event.target.value)}
                multiline
                rows={3}
                fullWidth
                placeholder="e.g. sleeveless, longer length"
              />
            </Collapse>

            <Divider />

            <Box>
              <Typography
                variant="overline"
                component="p"
                sx={{ color: "text.secondary", mb: 1 }}
              >
                your details
              </Typography>
              <CustomerFields
                name={customerName}
                email={customerEmail}
                phone={customerPhone}
                onNameChange={setCustomerName}
                onEmailChange={setCustomerEmail}
                onPhoneChange={setCustomerPhone}
              />
            </Box>

            <Box>
              <Typography
                variant="overline"
                component="p"
                sx={{ color: "text.secondary", mb: 1 }}
              >
                delivery
              </Typography>
              <DeliverySelector
                mode={deliveryMode}
                area={deliveryArea}
                rates={settings?.deliveryRates ?? []}
                onModeChange={setDeliveryMode}
                onAreaChange={setDeliveryArea}
              />
            </Box>

            {formError && <Alert severity="error">{formError}</Alert>}

            {submittedRef && (
              <Alert severity="success">
                Request received — reference <strong>{submittedRef}</strong>. We&apos;ve emailed a
                sign-in link to {customerEmail.trim()} so you can track it; the atelier will follow
                up with a quote.
              </Alert>
            )}

            <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5}>
              <Button
                variant="contained"
                loading={isSubmitting}
                disabled={submittedRef !== null}
                onClick={
                  isHomeVisit
                    ? handleHomeVisit
                    : isCustom
                      ? handleCustomRequest
                      : handleStandardOrder
                }
                sx={{ flex: { sm: 1 } }}
              >
                {submitLabel}
              </Button>
              <Box sx={{ flex: { sm: "0 0 auto" } }}>
                <CopyLinkButton />
              </Box>
            </Stack>

            <FormHelperText sx={{ color: stone, mt: 0 }}>
              {isCustom
                ? "You will receive a quote by email and in your account before any payment."
                : "Payment confirms your order and books it into production."}
            </FormHelperText>
          </Stack>
        </Box>
      </Box>

      {(relatedDesignsQuery.isPending || relatedDesigns.length > 0) && (
        <Box component="section" sx={{ mb: { xs: 8, md: 12 } }}>
          <MeasureRule
            label="Fig. — Same collection"
            sx={{ mb: { xs: 3, md: 4 } }}
          />
          <Stack
            direction={{ xs: "column", sm: "row" }}
            spacing={1.5}
            sx={{
              justifyContent: "space-between",
              alignItems: { sm: "flex-end" },
              mb: 3,
            }}
          >
            <Box>
              <Typography variant="h2" component="h2">
                More from this run.
              </Typography>
              <Typography
                variant="body2"
                sx={{ color: "text.secondary", mt: 1, maxWidth: "48ch" }}
              >
                Same fabric story, different cut. Every piece is made only when
                ordered.
              </Typography>
            </Box>
            <Button
              component={RouterLink}
              to="/store"
              variant="outlined"
              sx={{ width: { xs: "100%", sm: "auto" } }}
            >
              Shop all designs
            </Button>
          </Stack>

          {relatedDesignsQuery.isPending ? (
            <Box
              sx={{
                display: "grid",
                gridTemplateColumns: {
                  xs: "repeat(2, 1fr)",
                  sm: "repeat(3, 1fr)",
                  md: "repeat(4, 1fr)",
                },
                gap: { xs: 2, md: 3 },
              }}
            >
              {Array.from({ length: 4 }, (_, index) => (
                <Skeleton
                  key={index}
                  variant="rectangular"
                  sx={{ aspectRatio: "600 / 780", height: "auto" }}
                />
              ))}
            </Box>
          ) : (
            <DesignGrid designs={relatedDesigns} cloudName={cloudName} />
          )}
        </Box>
      )}
    </>
  );
}

export function DesignPage() {
  const { slug = "" } = useParams();
  const settings = usePublicSettings();
  const design = usePublicDesign(slug);

  useDocumentTitle(
    design.data?.name,
    design.data?.note ||
      `${design.data?.name ?? "A design"} — made-to-measure by Eight Two Five.`,
  );

  const notFound =
    design.isError &&
    design.error instanceof ApiError &&
    design.error.status === 404;

  return (
    <StorefrontLayout>
      {notFound ? (
        <RetiredPanel
          overline="design retired"
          title="This design has been retired"
          body="Each design lives only as long as its fabric. This one has sold through — the live designs are waiting in the store."
        />
      ) : design.isPending ? (
        <Box
          sx={{
            py: { xs: 4, md: 8 },
            display: "grid",
            gridTemplateColumns: { xs: "1fr", md: "7fr 5fr" },
            gap: { xs: 4, md: 8 },
          }}
        >
          <Skeleton
            variant="rectangular"
            sx={{ aspectRatio: "600 / 780", height: "auto" }}
          />
          <Box>
            <Skeleton width={120} />
            <Skeleton width={280} height={56} sx={{ mt: 1 }} />
            <Skeleton width={200} sx={{ mt: 3 }} />
          </Box>
        </Box>
      ) : design.isError ? (
        <Box sx={{ py: { xs: 6, md: 9 } }}>
          <ErrorState
            message={errorMessage(design.error)}
            onRetry={() => design.refetch()}
          />
        </Box>
      ) : (
        <DesignDetail
          key={design.data.id}
          design={design.data}
          settings={settings.data}
        />
      )}
    </StorefrontLayout>
  );
}
