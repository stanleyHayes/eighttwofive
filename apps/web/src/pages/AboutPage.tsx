import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import AutoAwesomeOutlined from "@mui/icons-material/AutoAwesomeOutlined";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { useDocumentTitle } from "@/lib/useDocumentTitle";

const CRAFT_BLOCKS = [
  {
    title: "The atelier",
    body: "A small Accra workroom where every garment is cut one at a time. No racks, no leftover stock — a design exists only when someone orders it.",
  },
  {
    title: "The fabric",
    body: "Each collection is built around a limited bolt of fabric. Around ten designs share it; when the bolt runs out, the collection retires for good.",
  },
];

export function AboutPage() {
  useDocumentTitle(
    "The Atelier",
    "Eight Two Five makes made-to-measure womenswear in limited, themed collections, cut one garment at a time in Accra.",
  );

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 4, md: 6 } }}>
        <PageBanner
          tone="ink"
          icon={<AutoAwesomeOutlined />}
          breadcrumbs={[{ label: "Home", to: "/" }, { label: "Atelier" }]}
          title="Cut to you, in Accra."
          description="A small atelier making made-to-measure womenswear in limited, themed collections — nothing mass-produced, nothing left on a shelf."
        />
      </Box>

      {/* Story */}
      <Box component="section" sx={{ mb: { xs: 6, md: 10 }, maxWidth: 640 }}>
        <Typography variant="overline" component="p" sx={{ color: "text.secondary", mb: 1 }}>
          Our story
        </Typography>
        <Typography variant="subtitle1" sx={{ color: "text.secondary", maxWidth: "54ch" }}>
          Eight Two Five makes made-to-measure womenswear in limited, themed collections.
          Nothing is mass-produced and nothing sits on a shelf — every piece is cut after you
          order it, to your measurements, and retired when its fabric runs out.
        </Typography>
      </Box>

      {/* Craft tiles */}
      <Stack
        component="section"
        direction={{ xs: "column", md: "row" }}
        spacing={2}
        sx={{ mb: { xs: 8, md: 12 } }}
      >
        {CRAFT_BLOCKS.map((block, index) => (
          <Box
            key={block.title}
            sx={{
              flex: 1,
              bgcolor: "background.paper",
              border: "1px solid",
              borderColor: "divider",
              p: 4,
              minHeight: 200,
              display: "flex",
              flexDirection: "column",
              justifyContent: "space-between",
            }}
          >
            <Typography variant="overline" component="p" sx={{ color: "text.secondary", mb: 1 }}>
              {`Fig. 0${index + 1}`}
            </Typography>
            <Box>
              <Typography variant="h5" component="h2" sx={{ mb: 1.5 }}>
                {block.title}
              </Typography>
              <Typography variant="body2" sx={{ color: "text.secondary" }}>
                {block.body}
              </Typography>
            </Box>
          </Box>
        ))}
      </Stack>
    </StorefrontLayout>
  );
}
