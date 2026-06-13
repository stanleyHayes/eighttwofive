import { useMemo } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Container from "@mui/material/Container";
import Link from "@mui/material/Link";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ArrowOutwardIcon from "@mui/icons-material/ArrowOutward";
import { Link as RouterLink } from "react-router";
import { MeasureRule } from "@/components/MeasureRule";
import { Reveal } from "@/components/Reveal";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import type { Collection, Design } from "@/features/catalog/api";
import { errorMessage } from "@/features/catalog/api";
import { DesignGrid } from "@/features/storefront/DesignCard";
import { CARD_TRANSFORM, photoUrl, sortedPhotos } from "@/features/storefront/api";
import { usePublicCollections, usePublicDesigns, usePublicSettings } from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { WaitlistForm } from "@/features/waitlist/WaitlistForm";
import { amber, brass, cream, creamMuted, creamText, displayFamily, GRAIN_URL, ink, inkSoft, monoFamily } from "@/theme";

const HERO_PUBLIC_ID = "eightfivetwo/hero-atelier";

function byCreatedAtDesc<T extends { createdAt: string }>(a: T, b: T): number {
  return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
}

const STEPS = [
  { n: "01", title: "Choose a design", body: "Browse the live collections, each priced in Ghana Cedis." },
  { n: "02", title: "Measured your way", body: "A standard band, your own measurements, or a home visit." },
  { n: "03", title: "Cut & delivered", body: "Made to order, tracked, then dispatched or picked up." },
];

// --- Hero (full-bleed image) ------------------------------------------------

function Hero({ cloudName }: { cloudName: string }) {
  const bg = cloudName ? photoUrl(cloudName, HERO_PUBLIC_ID, "f_auto,q_auto,w_2000") : "";

  return (
    <Box
      component="section"
      sx={{
        position: "relative",
        bgcolor: ink,
        color: cream,
        minHeight: { xs: "78vh", md: "min(88vh, 820px)" },
        display: "flex",
        alignItems: { xs: "flex-end", md: "center" },
        overflow: "hidden",
      }}
    >
      {/* Photograph */}
      {bg && (
        <Box
          aria-hidden
          sx={{
            position: "absolute",
            inset: 0,
            backgroundImage: `url("${bg}")`,
            backgroundSize: "cover",
            backgroundPosition: { xs: "70% center", md: "right center" },
          }}
        />
      )}
      {/* Scrim for legibility — heavier on the left where the text sits */}
      <Box
        aria-hidden
        sx={{
          position: "absolute",
          inset: 0,
          background: {
            xs: `linear-gradient(to top, ${ink} 8%, rgba(22,18,13,0.45) 55%, rgba(22,18,13,0.25) 100%)`,
            md: `linear-gradient(to right, ${ink} 22%, rgba(22,18,13,0.7) 42%, transparent 72%)`,
          },
        }}
      />
      <Box aria-hidden sx={{ position: "absolute", inset: 0, backgroundImage: GRAIN_URL, opacity: 0.05 }} />

      <Container maxWidth="lg" sx={{ position: "relative", zIndex: 1, py: { xs: 6, md: 0 } }}>
        <Box sx={{ maxWidth: { xs: "100%", md: 620 } }}>
          <Reveal delay={0}>
            <Box component="span" sx={{ fontFamily: monoFamily, fontSize: "0.6875rem", letterSpacing: "0.24em", color: brass, textTransform: "uppercase" }}>
              The 852 Atelier · Est. Accra
            </Box>
          </Reveal>
          <Reveal delay={90}>
            <Typography variant="h1" sx={{ mt: 2, mb: 3 }}>
              Made-to-measure{" "}
              <Box component="br" sx={{ display: { xs: "none", sm: "block" } }} />
              womenswear,{" "}
              <Box component="span" sx={{ color: amber, fontStyle: "italic" }}>
                cut to you.
              </Box>
            </Typography>
          </Reveal>
          <Reveal delay={180}>
            <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5} sx={{ alignItems: { sm: "center" } }}>
              <Button
                component={RouterLink}
                to="/store"
                size="large"
                endIcon={<ArrowOutwardIcon sx={{ fontSize: 18 }} />}
                sx={{ bgcolor: amber, color: ink, "&:hover": { bgcolor: brass }, width: { xs: "100%", sm: "auto" } }}
              >
                Shop the store
              </Button>
              <Button
                href="#how-its-made"
                variant="outlined"
                size="large"
                sx={{ color: cream, borderColor: "rgba(232,222,203,0.45)", "&:hover": { borderColor: cream }, width: { xs: "100%", sm: "auto" } }}
              >
                How it's made
              </Button>
            </Stack>
          </Reveal>
        </Box>
      </Container>
    </Box>
  );
}

// --- Section heading --------------------------------------------------------

function SectionHead({ fig, title, action }: { fig: string; title: string; action?: { to: string; label: string } }) {
  return (
    <Box sx={{ mb: { xs: 3.5, md: 5 } }}>
      <MeasureRule label={fig} sx={{ mb: 2.5 }} />
      <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5} sx={{ justifyContent: "space-between", alignItems: { sm: "flex-end" } }}>
        <Typography variant="h2" component="h2">
          {title}
        </Typography>
        {action && (
          <Link
            component={RouterLink}
            to={action.to}
            underline="none"
            variant="overline"
            sx={{ color: "text.primary", display: "inline-flex", alignItems: "center", gap: 0.75, whiteSpace: "nowrap", "&:hover": { color: amber } }}
          >
            {action.label} <ArrowOutwardIcon sx={{ fontSize: 15 }} />
          </Link>
        )}
      </Stack>
    </Box>
  );
}

// --- Collection tile --------------------------------------------------------

function CollectionTile({ collection, cover, cloudName, index }: { collection: Collection; cover: string | null; cloudName: string; index: number }) {
  return (
    <Reveal delay={index * 70}>
      <Link
        component={RouterLink}
        to={`/collections/${collection.slug}`}
        underline="none"
        sx={{
          display: "block",
          position: "relative",
          aspectRatio: "4 / 5",
          bgcolor: ink,
          color: cream,
          overflow: "hidden",
          "&:hover .e25-cover": { transform: "scale(1.04)" },
          "&:hover .e25-frame": { borderColor: amber },
        }}
      >
        {cover && cloudName ? (
          <Box
            className="e25-cover"
            component="img"
            src={photoUrl(cloudName, cover, CARD_TRANSFORM)}
            alt=""
            loading="lazy"
            sx={{ position: "absolute", inset: 0, width: "100%", height: "100%", objectFit: "cover", transition: "transform 600ms cubic-bezier(0.22,1,0.36,1)" }}
          />
        ) : (
          <Box className="e25-cover" sx={{ position: "absolute", inset: 0, background: `linear-gradient(150deg, ${inkSoft}, ${ink})`, transition: "transform 600ms" }} />
        )}
        <Box aria-hidden sx={{ position: "absolute", inset: 0, background: "linear-gradient(to top, rgba(22,18,13,0.85) 0%, rgba(22,18,13,0.05) 55%, rgba(22,18,13,0.3) 100%)" }} />
        <Box className="e25-frame" aria-hidden sx={{ position: "absolute", inset: 10, border: "1px solid transparent", transition: "border-color 280ms ease" }} />
        <Box sx={{ position: "absolute", inset: 0, p: { xs: 2.5, md: 3 }, display: "flex", flexDirection: "column", justifyContent: "space-between" }}>
          <Box component="span" sx={{ fontFamily: monoFamily, fontSize: "0.6875rem", letterSpacing: "0.18em", color: brass }}>
            {`0${index + 1}`} / Collection
          </Box>
          <Typography variant="h4" component="h3" sx={{ color: cream }}>
            {collection.name}
          </Typography>
        </Box>
      </Link>
    </Reveal>
  );
}

// --- Loading / error --------------------------------------------------------

function GridSkeleton({ count, ratio }: { count: number; ratio: string }) {
  return (
    <Box sx={{ display: "grid", gridTemplateColumns: { xs: "repeat(2, 1fr)", sm: "repeat(3, 1fr)", md: "repeat(4, 1fr)" }, gap: { xs: 2, md: 3 }, rowGap: { xs: 3, md: 4 } }}>
      {Array.from({ length: count }, (_, i) => (
        <Skeleton key={i} variant="rectangular" sx={{ aspectRatio: ratio, height: "auto", bgcolor: "rgba(22,18,13,0.06)" }} />
      ))}
    </Box>
  );
}

function ErrorPanel({ error }: { error: unknown }) {
  return (
    <Box sx={{ bgcolor: ink, color: cream, p: { xs: 4, md: 6 } }}>
      <Typography variant="h4" component="p" sx={{ mb: 1 }}>
        The store is catching its breath.
      </Typography>
      <Typography variant="body2" sx={{ color: creamMuted }}>
        {errorMessage(error)}
      </Typography>
    </Box>
  );
}

// --- Page -------------------------------------------------------------------

export function HomePage() {
  useDocumentTitle(
    undefined,
    "Made-to-measure womenswear, cut to you in Accra. Browse limited, themed collections and have each piece made to your measurements.",
  );
  const settings = usePublicSettings();
  const collections = usePublicCollections();
  const designs = usePublicDesigns({});

  const cloudName = settings.data?.cloudName ?? "";
  const featuredCollections = collections.data?.slice().sort(byCreatedAtDesc).slice(0, 3) ?? [];
  const featuredDesigns = designs.data?.slice().sort(byCreatedAtDesc).slice(0, 4) ?? [];

  const coverByCollection = useMemo(() => {
    const covers = new Map<string, string>();
    for (const design of (designs.data ?? []) as Design[]) {
      if (covers.has(design.collectionId)) continue;
      const photo = sortedPhotos(design)[0];
      if (photo) covers.set(design.collectionId, photo.publicId);
    }
    return covers;
  }, [designs.data]);

  const isPending = collections.isPending || designs.isPending;
  const isError = collections.isError || designs.isError;

  return (
    <StorefrontLayout bleedHero={<Hero cloudName={cloudName} />}>
      {/* Collections */}
      <Box component="section" sx={{ pt: { xs: 7, md: 11 }, pb: { xs: 6, md: 9 } }}>
        <SectionHead fig="Fig. 01 — Collections" title="Named, limited runs." action={{ to: "/store", label: "All collections" }} />
        {isPending ? (
          <GridSkeleton count={3} ratio="4 / 5" />
        ) : isError ? (
          <ErrorPanel error={collections.error ?? designs.error} />
        ) : featuredCollections.length > 0 ? (
          <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" }, gap: { xs: 2.5, md: 3 } }}>
            {featuredCollections.map((collection, index) => (
              <CollectionTile key={collection.id} collection={collection} cover={coverByCollection.get(collection.id) ?? null} cloudName={cloudName} index={index} />
            ))}
          </Box>
        ) : (
          <Typography sx={{ color: "text.secondary" }}>New collections are on the way.</Typography>
        )}
      </Box>

      {/* How it's made */}
      <Box component="section" id="how-its-made" sx={{ scrollMarginTop: 80, bgcolor: ink, color: cream, mx: { xs: -2, sm: -3 }, px: { xs: 2, sm: 3 }, py: { xs: 7, md: 11 } }}>
        <Container maxWidth="lg" disableGutters>
          <MeasureRule variant="light" label="Fig. 02 — Method" caption="Made to measure" sx={{ mb: { xs: 4, md: 6 } }} />
          <Typography variant="h2" component="h2" sx={{ mb: { xs: 4, md: 6 }, maxWidth: "16ch" }}>
            Design to doorstep, in three.
          </Typography>
          <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", md: "repeat(3, 1fr)" }, gap: { xs: 3.5, md: 5 } }}>
            {STEPS.map((step, index) => (
              <Reveal key={step.n} delay={index * 90}>
                <Stack spacing={1.5} sx={{ borderTop: "1px solid rgba(232,222,203,0.22)", pt: 3 }}>
                  <Box component="span" sx={{ fontFamily: displayFamily, fontWeight: 700, fontSize: "2.8rem", lineHeight: 1, color: amber }}>
                    {step.n}
                  </Box>
                  <Typography variant="h5" component="h3">
                    {step.title}
                  </Typography>
                  <Typography variant="body2" sx={{ color: creamText }}>
                    {step.body}
                  </Typography>
                </Stack>
              </Reveal>
            ))}
          </Box>
        </Container>
      </Box>

      {/* New designs */}
      <Box component="section" sx={{ pt: { xs: 7, md: 11 }, pb: { xs: 6, md: 9 } }}>
        <SectionHead fig="Fig. 03 — New In" title="Fresh off the table." action={{ to: "/store", label: "Shop now" }} />
        {isPending ? (
          <GridSkeleton count={4} ratio="600 / 780" />
        ) : isError ? (
          <ErrorPanel error={designs.error} />
        ) : featuredDesigns.length > 0 ? (
          <DesignGrid designs={featuredDesigns} cloudName={cloudName} />
        ) : (
          <Typography sx={{ color: "text.secondary" }}>New designs are on the way.</Typography>
        )}
      </Box>

      {/* Waitlist */}
      <Box component="section" sx={{ bgcolor: ink, color: cream, mx: { xs: -2, sm: -3 }, px: { xs: 2, sm: 3 }, py: { xs: 7, md: 11 } }}>
        <Container maxWidth="lg" disableGutters>
          <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", md: "1fr 1fr" }, gap: { xs: 4, md: 8 }, alignItems: "center" }}>
            <Box>
              <Box component="span" sx={{ fontFamily: monoFamily, fontSize: "0.6875rem", letterSpacing: "0.22em", color: brass, textTransform: "uppercase" }}>
                Fig. 04 — The list
              </Box>
              <Typography variant="h2" component="h2" sx={{ mt: 2, mb: 2 }}>
                Be the first to know.
              </Typography>
              <Typography sx={{ color: creamText, maxWidth: "38ch" }}>
                One note when a new collection drops. Limited runs sell through.
              </Typography>
            </Box>
            <Box>
              <WaitlistForm />
            </Box>
          </Box>
        </Container>
      </Box>
    </StorefrontLayout>
  );
}
