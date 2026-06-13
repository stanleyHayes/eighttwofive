import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import EventAvailableOutlined from "@mui/icons-material/EventAvailableOutlined";
import StraightenOutlined from "@mui/icons-material/StraightenOutlined";
import { Link as RouterLink } from "react-router";
import { MeasureRule } from "@/components/MeasureRule";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { formatPesewas } from "@/features/catalog/money";
import { usePublicSettings } from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import {
  amber,
  brass,
  cream,
  creamText,
  GRAIN_URL,
  ink,
  monoFamily,
} from "@/theme";

const FIT_PATHS = [
  {
    n: "01",
    title: "Order a size band",
    body: "Choose the closest band on the design page. The atelier still confirms the final fit before cutting.",
    action: { label: "Browse designs", to: "/store" },
  },
  {
    n: "02",
    title: "Send measurements",
    body: "Enter your bust, waist, hips, and desired length in centimetres when your band is not listed.",
    action: { label: "Start in store", to: "/store" },
  },
  {
    n: "03",
    title: "Book a visit",
    body: "Choose a home visit when you want the measuring handled in person before the garment is made.",
    action: { label: "Book a visit", to: "/slots" },
  },
];

const MEASURES = [
  {
    label: "Bust",
    body: "Measure around the fullest part, with the tape level and relaxed.",
  },
  {
    label: "Waist",
    body: "Measure the natural waist, usually the narrowest point of the torso.",
  },
  {
    label: "Hips",
    body: "Measure the fullest part of the hips, standing naturally.",
  },
  {
    label: "Length",
    body: "Measure from the shoulder or waist point to where you want the hem to sit.",
  },
];

const TIMELINE = [
  { label: "Select", body: "Pick a design and fit path." },
  { label: "Confirm", body: "The atelier checks details before cutting." },
  { label: "Cut", body: "Your piece is made to order." },
  { label: "Finish", body: "Pickup or dispatch when ready." },
];

function FitPathCard({
  n,
  title,
  body,
  action,
}: {
  n: string;
  title: string;
  body: string;
  action: { label: string; to: string };
}) {
  return (
    <Box
      sx={{
        bgcolor: "background.paper",
        border: "1px solid",
        borderColor: "divider",
        p: { xs: 3, md: 4 },
        minHeight: 320,
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
      }}
    >
      <Box>
        <Box
          component="span"
          sx={{ fontFamily: monoFamily, color: brass, fontSize: "0.75rem" }}
        >
          {n}
        </Box>
        <Typography variant="h4" component="h2" sx={{ mt: 2, mb: 1.5 }}>
          {title}
        </Typography>
        <Typography variant="body2" sx={{ color: "text.secondary" }}>
          {body}
        </Typography>
      </Box>
      <Button
        component={RouterLink}
        to={action.to}
        variant="outlined"
        endIcon={<ArrowOutwardIcon sx={{ fontSize: 16 }} />}
        sx={{ mt: 3, width: { xs: "100%", sm: "fit-content" } }}
      >
        {action.label}
      </Button>
    </Box>
  );
}

export function FitGuidePage() {
  useDocumentTitle(
    "Fit guide",
    "Choose a size band, send measurements, or book a fitting visit before your Eight Two Five piece is cut.",
  );
  const settings = usePublicSettings();
  const deposit = settings.data?.depositPesewas ?? 500_00;
  const location =
    settings.data?.visitLocation || "the Eight Two Five workspace";

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 4, md: 6 } }}>
        <PageBanner
          tone="ink"
          icon={<StraightenOutlined />}
          breadcrumbs={[{ label: "Home", to: "/" }, { label: "Fit guide" }]}
          title="Fit, before fabric."
          description="Choose a standard band, send your measurements, or book a visit. Every piece is confirmed before it is cut."
          action={{ to: "/store", label: "Browse designs" }}
        />
      </Box>

      <Box component="section" sx={{ mb: { xs: 7, md: 10 } }}>
        <MeasureRule
          label="Fig. 01 — Fit paths"
          sx={{ mb: { xs: 3.5, md: 5 } }}
        />
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" },
            gap: { xs: 2, md: 3 },
          }}
        >
          {FIT_PATHS.map((path) => (
            <FitPathCard key={path.n} {...path} />
          ))}
        </Box>
      </Box>

      <Box
        component="section"
        sx={{
          mb: { xs: 7, md: 10 },
          display: "grid",
          gridTemplateColumns: { xs: "1fr", md: "0.85fr 1.15fr" },
          gap: { xs: 4, md: 7 },
          alignItems: "start",
        }}
      >
        <Box>
          <Typography variant="overline" component="p" sx={{ color: brass }}>
            Measure yourself
          </Typography>
          <Typography variant="h2" component="h2" sx={{ mt: 1.5, mb: 2 }}>
            Four numbers are enough to start.
          </Typography>
          <Typography
            variant="subtitle1"
            sx={{ color: "text.secondary", maxWidth: "42ch" }}
          >
            Use a soft tape and measure in centimetres. The atelier reviews the
            numbers with you before cutting, so the form is a starting point,
            not a trap.
          </Typography>
        </Box>

        <Box sx={{ borderTop: "1px solid", borderColor: "divider" }}>
          {MEASURES.map((measure, index) => (
            <Box
              key={measure.label}
              sx={{
                display: "grid",
                gridTemplateColumns: { xs: "auto 1fr", sm: "96px 1fr" },
                gap: { xs: 2, sm: 3 },
                py: 3,
                borderBottom: "1px solid",
                borderColor: "divider",
              }}
            >
              <Typography
                variant="overline"
                component="p"
                sx={{ color: brass }}
              >
                {String(index + 1).padStart(2, "0")}
              </Typography>
              <Box>
                <Typography variant="h5" component="h3">
                  {measure.label}
                </Typography>
                <Typography
                  variant="body2"
                  sx={{ color: "text.secondary", mt: 0.5 }}
                >
                  {measure.body}
                </Typography>
              </Box>
            </Box>
          ))}
        </Box>
      </Box>

      <Box
        component="section"
        sx={{
          bgcolor: ink,
          color: cream,
          mx: { xs: -2, sm: -3 },
          px: { xs: 2, sm: 3 },
          py: { xs: 7, md: 10 },
          mb: { xs: 7, md: 10 },
          backgroundImage: GRAIN_URL,
          backgroundBlendMode: "overlay",
        }}
      >
        <MeasureRule
          variant="light"
          label="Fig. 02 — Visit"
          caption="measured in person"
          sx={{ mb: { xs: 4, md: 5 } }}
        />
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: { xs: "1fr", md: "1fr 1fr" },
            gap: { xs: 4, md: 7 },
            alignItems: "center",
          }}
        >
          <Box>
            <Typography
              variant="h2"
              component="h2"
              sx={{ color: cream, maxWidth: "12ch" }}
            >
              Want the tape handled for you?
            </Typography>
            <Typography sx={{ color: creamText, mt: 2.5, maxWidth: "44ch" }}>
              Book a home visit, or arrange a fitting at {location}. The visit
              deposit is {formatPesewas(deposit)} and counts toward your
              garment.
            </Typography>
          </Box>
          <Box
            sx={{
              border: "1px solid rgba(232,222,203,0.22)",
              p: { xs: 3, md: 4 },
              minHeight: 260,
              display: "flex",
              flexDirection: "column",
              justifyContent: "space-between",
            }}
          >
            <EventAvailableOutlined sx={{ color: amber, fontSize: 40 }} />
            <Box>
              <Typography
                variant="h5"
                component="h3"
                sx={{ color: cream, mb: 1 }}
              >
                Visit deposit
              </Typography>
              <Typography variant="body2" sx={{ color: creamText }}>
                The deposit holds your slot and is credited to the final garment
                payment.
              </Typography>
            </Box>
            <Button
              component={RouterLink}
              to="/slots"
              variant="contained"
              endIcon={<ArrowOutwardIcon sx={{ fontSize: 16 }} />}
              sx={{
                mt: 3,
                bgcolor: amber,
                color: ink,
                width: { xs: "100%", sm: "fit-content" },
              }}
            >
              Book a visit
            </Button>
          </Box>
        </Box>
      </Box>

      <Box component="section" sx={{ mb: { xs: 8, md: 12 } }}>
        <MeasureRule
          label="Fig. 03 — Order rhythm"
          sx={{ mb: { xs: 3.5, md: 5 } }}
        />
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: {
              xs: "1fr",
              sm: "repeat(2, 1fr)",
              md: "repeat(4, 1fr)",
            },
            borderTop: "1px solid",
            borderLeft: "1px solid",
            borderColor: "divider",
          }}
        >
          {TIMELINE.map((item, index) => (
            <Box
              key={item.label}
              sx={{
                minHeight: 190,
                p: 3,
                borderRight: "1px solid",
                borderBottom: "1px solid",
                borderColor: "divider",
              }}
            >
              <Typography
                variant="overline"
                component="p"
                sx={{ color: brass }}
              >
                Step {String(index + 1).padStart(2, "0")}
              </Typography>
              <Typography variant="h5" component="h3" sx={{ mt: 1.5 }}>
                {item.label}
              </Typography>
              <Typography
                variant="body2"
                sx={{ color: "text.secondary", mt: 1 }}
              >
                {item.body}
              </Typography>
            </Box>
          ))}
        </Box>
      </Box>
    </StorefrontLayout>
  );
}
