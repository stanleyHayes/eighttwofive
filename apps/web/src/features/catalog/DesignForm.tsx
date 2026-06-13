import { useState, type FormEvent } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import LinearProgress from "@mui/material/LinearProgress";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import AddIcon from "@mui/icons-material/Add";
import ArrowDownwardIcon from "@mui/icons-material/ArrowDownward";
import ArrowUpwardIcon from "@mui/icons-material/ArrowUpward";
import CloseIcon from "@mui/icons-material/Close";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutlined";
import StarIcon from "@mui/icons-material/Star";
import StarBorderIcon from "@mui/icons-material/StarBorder";
import { Link as RouterLink, useNavigate } from "react-router";
import { amber, clayDeep, ink, monoFamily, sandDeep } from "@/theme";
import {
  cloudinaryPreviewUrl,
  errorMessage,
  getUploadSignature,
  uploadToCloudinary,
  type Collection,
  type Design,
  type DesignInput,
  type DesignPhoto,
  type UploadSignature,
} from "./api";
import { useCreateDesign, useUpdateDesign, useUploadConfig } from "./hooks";
import { ghsInputToPesewas, pesewasToGhsInput } from "./money";

let uidSeq = 0;
function nextUid(): string {
  uidSeq += 1;
  return `u${uidSeq}`;
}

interface ChartEntryDraft {
  uid: string;
  key: string;
  value: string;
}

interface BandDraft {
  uid: string;
  label: string;
  price: string;
  chart: ChartEntryDraft[];
}

interface UploadItem {
  key: string;
  fileName: string;
  progress: number;
  error?: string;
}

interface BandFieldErrors {
  label?: string;
  price?: string;
}

interface FormErrors {
  collectionId?: string;
  name?: string;
  bands?: string;
  bandFields: Record<string, BandFieldErrors>;
}

const NO_ERRORS: FormErrors = { bandFields: {} };

function emptyBand(): BandDraft {
  return { uid: nextUid(), label: "", price: "", chart: [] };
}

function bandsFromDesign(design: Design): BandDraft[] {
  return design.sizeBands.map((band) => ({
    uid: nextUid(),
    label: band.label,
    price: pesewasToGhsInput(band.pricePesewas),
    chart: Object.entries(band.chart).map(([key, value]) => ({
      uid: nextUid(),
      key,
      value,
    })),
  }));
}

function hasErrors(errors: FormErrors): boolean {
  return (
    Boolean(errors.collectionId || errors.name || errors.bands) ||
    Object.keys(errors.bandFields).length > 0
  );
}

interface DesignFormProps {
  collections: Collection[];
  /** When set, the form edits this design; otherwise it creates one. */
  initial?: Design;
}

export function DesignForm({ collections, initial }: DesignFormProps) {
  const navigate = useNavigate();
  const uploadConfig = useUploadConfig();
  const cloudName = uploadConfig.data?.cloudName ?? null;
  const uploadsUnavailable = uploadConfig.data === null;

  const [collectionId, setCollectionId] = useState(initial?.collectionId ?? "");
  const [name, setName] = useState(initial?.name ?? "");
  const [note, setNote] = useState(initial?.note ?? "");
  const [photos, setPhotos] = useState<DesignPhoto[]>(() =>
    [...(initial?.photos ?? [])].sort((a, b) => a.order - b.order),
  );
  const [uploads, setUploads] = useState<UploadItem[]>([]);
  const [uploadFlowError, setUploadFlowError] = useState<string | null>(null);
  const [bands, setBands] = useState<BandDraft[]>(() =>
    initial && initial.sizeBands.length > 0 ? bandsFromDesign(initial) : [emptyBand()],
  );
  const [errors, setErrors] = useState<FormErrors>(NO_ERRORS);
  const [serverError, setServerError] = useState<string | null>(null);

  const create = useCreateDesign();
  const update = useUpdateDesign();
  const saving = create.isPending || update.isPending;
  const uploading = uploads.some((upload) => !upload.error);

  // --- Photos ---

  const startUpload = (file: File, signature: UploadSignature) => {
    const key = nextUid();
    setUploads((current) => [...current, { key, fileName: file.name, progress: 0 }]);
    uploadToCloudinary(file, signature, (fraction) => {
      setUploads((current) =>
        current.map((upload) =>
          upload.key === key ? { ...upload, progress: fraction } : upload,
        ),
      );
    })
      .then((publicId) => {
        setUploads((current) => current.filter((upload) => upload.key !== key));
        setPhotos((current) => [...current, { publicId, order: current.length }]);
      })
      .catch((error: unknown) => {
        setUploads((current) =>
          current.map((upload) =>
            upload.key === key
              ? {
                  ...upload,
                  error: error instanceof Error ? error.message : "Upload failed.",
                }
              : upload,
          ),
        );
      });
  };

  const handleFiles = async (fileList: FileList | null) => {
    if (!fileList || fileList.length === 0) return;
    const files = Array.from(fileList);
    setUploadFlowError(null);

    let signature: UploadSignature | null;
    try {
      signature = await getUploadSignature();
    } catch (error) {
      setUploadFlowError(errorMessage(error, "Could not start the upload. Try again."));
      return;
    }
    if (!signature) {
      setUploadFlowError(
        "Photo uploads aren't configured yet — save the design without photos and add them later.",
      );
      return;
    }
    for (const file of files) startUpload(file, signature);
  };

  const movePhoto = (index: number, delta: number) => {
    setPhotos((current) => {
      const target = index + delta;
      if (target < 0 || target >= current.length) return current;
      const next = [...current];
      const [item] = next.splice(index, 1);
      next.splice(target, 0, item);
      return next;
    });
  };

  const removePhoto = (index: number) => {
    setPhotos((current) => current.filter((_, i) => i !== index));
  };

  // The first photo is the design's main image everywhere in the store, so
  // "set as main" simply moves the chosen photo to the front of the list.
  const makeCover = (index: number) => {
    setPhotos((current) => {
      if (index <= 0 || index >= current.length) return current;
      const next = [...current];
      const [item] = next.splice(index, 1);
      next.unshift(item);
      return next;
    });
  };

  const dismissUpload = (key: string) => {
    setUploads((current) => current.filter((upload) => upload.key !== key));
  };

  // --- Size bands ---

  const updateBand = (uid: string, patch: Partial<Pick<BandDraft, "label" | "price">>) => {
    setBands((current) =>
      current.map((band) => (band.uid === uid ? { ...band, ...patch } : band)),
    );
  };

  const addBand = () => setBands((current) => [...current, emptyBand()]);

  const removeBand = (uid: string) => {
    setBands((current) => current.filter((band) => band.uid !== uid));
  };

  const addChartEntry = (bandUid: string) => {
    setBands((current) =>
      current.map((band) =>
        band.uid === bandUid
          ? { ...band, chart: [...band.chart, { uid: nextUid(), key: "", value: "" }] }
          : band,
      ),
    );
  };

  const updateChartEntry = (
    bandUid: string,
    entryUid: string,
    patch: Partial<Pick<ChartEntryDraft, "key" | "value">>,
  ) => {
    setBands((current) =>
      current.map((band) =>
        band.uid === bandUid
          ? {
              ...band,
              chart: band.chart.map((entry) =>
                entry.uid === entryUid ? { ...entry, ...patch } : entry,
              ),
            }
          : band,
      ),
    );
  };

  const removeChartEntry = (bandUid: string, entryUid: string) => {
    setBands((current) =>
      current.map((band) =>
        band.uid === bandUid
          ? { ...band, chart: band.chart.filter((entry) => entry.uid !== entryUid) }
          : band,
      ),
    );
  };

  // --- Validation & submit ---

  const validate = (): FormErrors => {
    const next: FormErrors = { bandFields: {} };
    if (!collectionId) next.collectionId = "Choose a collection.";
    if (!name.trim()) next.name = "Name is required.";
    if (bands.length === 0) next.bands = "Add at least one size band.";

    const labelCounts = new Map<string, number>();
    for (const band of bands) {
      const label = band.label.trim().toLowerCase();
      if (label) labelCounts.set(label, (labelCounts.get(label) ?? 0) + 1);
    }
    for (const band of bands) {
      const fieldErrors: BandFieldErrors = {};
      const label = band.label.trim();
      if (!label) {
        fieldErrors.label = "Label is required.";
      } else if ((labelCounts.get(label.toLowerCase()) ?? 0) > 1) {
        fieldErrors.label = "Band labels must be unique.";
      }
      const pesewas = ghsInputToPesewas(band.price);
      if (pesewas === null || pesewas <= 0) {
        fieldErrors.price = "Enter a price greater than 0.";
      }
      if (fieldErrors.label || fieldErrors.price) next.bandFields[band.uid] = fieldErrors;
    }
    return next;
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setServerError(null);
    const validation = validate();
    setErrors(validation);
    if (hasErrors(validation)) return;

    const input: DesignInput = {
      collectionId,
      name: name.trim(),
      note: note.trim(),
      photos: photos.map((photo, index) => ({ publicId: photo.publicId, order: index })),
      sizeBands: bands.map((band) => ({
        label: band.label.trim(),
        // validate() guarantees a positive integer here.
        pricePesewas: ghsInputToPesewas(band.price) ?? 0,
        chart: Object.fromEntries(
          band.chart
            .filter((entry) => entry.key.trim() !== "")
            .map((entry) => [entry.key.trim(), entry.value.trim()]),
        ),
      })),
    };

    const callbacks = {
      onSuccess: () => navigate("/admin/designs"),
      onError: (error: unknown) => setServerError(errorMessage(error)),
    };
    if (initial) {
      update.mutate({ id: initial.id, input }, callbacks);
    } else {
      create.mutate(input, callbacks);
    }
  };

  return (
    <Box component="form" onSubmit={handleSubmit} noValidate>
      <Stack spacing={5}>
        {collections.length === 0 && (
          <Alert severity="info">
            Create a collection first — every design belongs to one.
          </Alert>
        )}

        <Stack spacing={2}>
          <TextField
            select
            label="Collection"
            value={collectionId}
            onChange={(event) => {
              setCollectionId(event.target.value);
              if (errors.collectionId) {
                setErrors((current) => ({ ...current, collectionId: undefined }));
              }
            }}
            error={Boolean(errors.collectionId)}
            helperText={errors.collectionId ?? " "}
            fullWidth
          >
            {collections.map((collection) => (
              <MenuItem key={collection.id} value={collection.id}>
                {collection.name}
                {collection.status === "retired" ? " (retired)" : ""}
              </MenuItem>
            ))}
          </TextField>
          <TextField
            label="Name"
            value={name}
            onChange={(event) => {
              setName(event.target.value);
              if (errors.name) setErrors((current) => ({ ...current, name: undefined }));
            }}
            error={Boolean(errors.name)}
            helperText={errors.name ?? " "}
            fullWidth
            required
          />
          <TextField
            label="Note"
            value={note}
            onChange={(event) => setNote(event.target.value)}
            multiline
            minRows={3}
            fullWidth
          />
        </Stack>

        <Box>
          <Typography variant="overline" component="h2" sx={{ color: clayDeep }}>
            photos
          </Typography>
          <Stack spacing={2} sx={{ mt: 1.5 }}>
            {uploadsUnavailable && (
              <Alert severity="info">
                Photo uploads aren't configured yet. You can still save this design
                without photos and add them later.
              </Alert>
            )}
            {uploadFlowError && <Alert severity="warning">{uploadFlowError}</Alert>}

            {photos.length > 0 && (
              <Typography variant="caption" sx={{ color: "text.secondary", display: "block" }}>
                The first photo is the design&apos;s main image across the store. Use the star to
                make any photo the main one.
              </Typography>
            )}
            {photos.length > 0 && (
              <Stack direction="row" spacing={2} sx={{ flexWrap: "wrap", rowGap: 2 }}>
                {photos.map((photo, index) => (
                  <Box key={photo.publicId} sx={{ width: 104 }}>
                    <Box sx={{ position: "relative" }}>
                      {cloudName ? (
                        <Box
                          component="img"
                          src={cloudinaryPreviewUrl(cloudName, photo.publicId)}
                          alt={index === 0 ? `Main photo` : `Photo ${index + 1}`}
                          loading="lazy"
                          decoding="async"
                          sx={{
                            width: 100,
                            height: 130,
                            objectFit: "cover",
                            display: "block",
                            bgcolor: sandDeep,
                            outline: index === 0 ? `2px solid ${amber}` : "none",
                          }}
                        />
                      ) : (
                        <Box
                          role="img"
                          aria-label={`Photo ${index + 1} (preview unavailable)`}
                          sx={{
                            width: 100,
                            height: 130,
                            bgcolor: sandDeep,
                            outline: index === 0 ? `2px solid ${amber}` : "none",
                          }}
                        />
                      )}
                      {index === 0 && (
                        <Box
                          sx={{
                            position: "absolute",
                            top: 6,
                            left: 6,
                            px: 0.75,
                            py: 0.25,
                            bgcolor: amber,
                            color: ink,
                            fontFamily: monoFamily,
                            fontSize: "0.5625rem",
                            fontWeight: 600,
                            letterSpacing: "0.1em",
                            textTransform: "uppercase",
                            display: "inline-flex",
                            alignItems: "center",
                            gap: 0.25,
                          }}
                        >
                          <StarIcon sx={{ fontSize: 11 }} /> Main
                        </Box>
                      )}
                    </Box>
                    <Stack direction="row" sx={{ mt: 0.25 }}>
                      <IconButton
                        size="small"
                        aria-label={`Set photo ${index + 1} as main`}
                        title="Set as main image"
                        disabled={index === 0}
                        onClick={() => makeCover(index)}
                      >
                        {index === 0 ? (
                          <StarIcon fontSize="inherit" sx={{ color: amber }} />
                        ) : (
                          <StarBorderIcon fontSize="inherit" />
                        )}
                      </IconButton>
                      <IconButton
                        size="small"
                        aria-label={`Move photo ${index + 1} up`}
                        disabled={index === 0}
                        onClick={() => movePhoto(index, -1)}
                      >
                        <ArrowUpwardIcon fontSize="inherit" />
                      </IconButton>
                      <IconButton
                        size="small"
                        aria-label={`Move photo ${index + 1} down`}
                        disabled={index === photos.length - 1}
                        onClick={() => movePhoto(index, 1)}
                      >
                        <ArrowDownwardIcon fontSize="inherit" />
                      </IconButton>
                      <IconButton
                        size="small"
                        aria-label={`Remove photo ${index + 1}`}
                        onClick={() => removePhoto(index)}
                      >
                        <CloseIcon fontSize="inherit" />
                      </IconButton>
                    </Stack>
                  </Box>
                ))}
              </Stack>
            )}

            {uploads.map((upload) => (
              <Stack
                key={upload.key}
                direction="row"
                spacing={2}
                sx={{ alignItems: "center", maxWidth: 480 }}
              >
                <Typography variant="body2" noWrap sx={{ minWidth: 140, maxWidth: 200 }}>
                  {upload.fileName}
                </Typography>
                {upload.error ? (
                  <>
                    <Typography variant="body2" sx={{ color: "error.main", flexGrow: 1 }}>
                      {upload.error}
                    </Typography>
                    <IconButton
                      size="small"
                      aria-label={`Dismiss failed upload ${upload.fileName}`}
                      onClick={() => dismissUpload(upload.key)}
                    >
                      <CloseIcon fontSize="inherit" />
                    </IconButton>
                  </>
                ) : (
                  <LinearProgress
                    variant="determinate"
                    value={Math.round(upload.progress * 100)}
                    aria-label={`Uploading ${upload.fileName}`}
                    sx={{ flexGrow: 1 }}
                  />
                )}
              </Stack>
            ))}

            <Box>
              <Button variant="outlined" component="label" disabled={uploadsUnavailable}>
                Upload photos
                <input
                  hidden
                  type="file"
                  accept="image/*"
                  multiple
                  onChange={(event) => {
                    void handleFiles(event.target.files);
                    event.target.value = "";
                  }}
                />
              </Button>
            </Box>
          </Stack>
        </Box>

        <Box>
          <Typography variant="overline" component="h2" sx={{ color: clayDeep }}>
            size bands
          </Typography>
          <Stack spacing={2} sx={{ mt: 1.5 }}>
            {errors.bands && (
              <Typography variant="body2" sx={{ color: "error.main" }} role="alert">
                {errors.bands}
              </Typography>
            )}
            {bands.map((band, index) => {
              const bandErrors = errors.bandFields[band.uid] ?? {};
              return (
                <Box
                  key={band.uid}
                  sx={{ border: "1px solid", borderColor: "divider", p: 2.5 }}
                >
                  <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5}>
                    <TextField
                      label="Label"
                      placeholder="8"
                      value={band.label}
                      onChange={(event) => updateBand(band.uid, { label: event.target.value })}
                      error={Boolean(bandErrors.label)}
                      helperText={bandErrors.label ?? " "}
                      sx={{ width: { sm: 140 } }}
                    />
                    <TextField
                      label="Price (GHS)"
                      placeholder="500.00"
                      value={band.price}
                      onChange={(event) => updateBand(band.uid, { price: event.target.value })}
                      error={Boolean(bandErrors.price)}
                      helperText={bandErrors.price ?? " "}
                      slotProps={{
                        input: {
                          startAdornment: (
                            <InputAdornment position="start">GH₵</InputAdornment>
                          ),
                        },
                        htmlInput: { inputMode: "decimal" },
                      }}
                      sx={{ width: { sm: 220 } }}
                    />
                    <Box sx={{ flexGrow: 1 }} />
                    <IconButton
                      aria-label={`Remove size band ${index + 1}`}
                      onClick={() => removeBand(band.uid)}
                      sx={{ alignSelf: { xs: "flex-end", sm: "flex-start" } }}
                    >
                      <DeleteOutlineIcon />
                    </IconButton>
                  </Stack>

                  <Typography
                    variant="overline"
                    component="p"
                    sx={{ color: "text.secondary", mt: 1 }}
                  >
                    size chart
                  </Typography>
                  <Stack spacing={1.5} sx={{ mt: 1 }}>
                    {band.chart.map((entry, entryIndex) => (
                      <Stack
                        key={entry.uid}
                        direction="row"
                        spacing={1.5}
                        sx={{ alignItems: "center" }}
                      >
                        <TextField
                          label="Measurement"
                          placeholder="bust"
                          size="small"
                          value={entry.key}
                          onChange={(event) =>
                            updateChartEntry(band.uid, entry.uid, {
                              key: event.target.value,
                            })
                          }
                        />
                        <TextField
                          label="Value"
                          placeholder={'34"'}
                          size="small"
                          value={entry.value}
                          onChange={(event) =>
                            updateChartEntry(band.uid, entry.uid, {
                              value: event.target.value,
                            })
                          }
                        />
                        <IconButton
                          size="small"
                          aria-label={`Remove measurement ${entryIndex + 1} from band ${index + 1}`}
                          onClick={() => removeChartEntry(band.uid, entry.uid)}
                        >
                          <CloseIcon fontSize="inherit" />
                        </IconButton>
                      </Stack>
                    ))}
                    <Box>
                      <Button
                        size="small"
                        variant="text"
                        startIcon={<AddIcon />}
                        onClick={() => addChartEntry(band.uid)}
                        sx={{ px: 1, py: 0.5 }}
                      >
                        Add measurement
                      </Button>
                    </Box>
                  </Stack>
                </Box>
              );
            })}
            <Box>
              <Button
                variant="outlined"
                startIcon={<AddIcon />}
                onClick={addBand}
                sx={{ px: 3, py: 1 }}
              >
                Add size band
              </Button>
            </Box>
          </Stack>
        </Box>

        {serverError && <Alert severity="error">{serverError}</Alert>}

        <Stack direction="row" spacing={2} sx={{ alignItems: "center" }}>
          <Button type="submit" variant="contained" loading={saving} disabled={uploading}>
            Save design
          </Button>
          <Button component={RouterLink} to="/admin/designs" variant="text" disabled={saving}>
            Cancel
          </Button>
          {uploading && (
            <Typography variant="body2" sx={{ color: "text.secondary" }}>
              Waiting for photo uploads to finish…
            </Typography>
          )}
        </Stack>
      </Stack>
    </Box>
  );
}
