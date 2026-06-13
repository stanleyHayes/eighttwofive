import { useEffect, useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import InputAdornment from "@mui/material/InputAdornment";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import SearchOutlined from "@mui/icons-material/SearchOutlined";
import StorefrontOutlined from "@mui/icons-material/StorefrontOutlined";
import { useSearchParams } from "react-router";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import type { Design } from "@/features/catalog/api";
import { errorMessage } from "@/features/catalog/api";
import { useDebouncedValue } from "@/features/catalog/useDebouncedValue";
import { CollectionCard } from "@/features/storefront/CollectionCard";
import { sortedPhotos } from "@/features/storefront/api";
import { DesignGrid } from "@/features/storefront/DesignCard";
import {
  usePublicCollections,
  usePublicDesigns,
  usePublicSettings,
} from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { sand, sandDeep } from "@/theme";

function CardGridSkeleton({ count }: { count: number }) {
  return (
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
      {Array.from({ length: count }, (_, i) => (
        <Skeleton
          key={i}
          variant="rectangular"
          sx={{ aspectRatio: "600 / 780", height: "auto" }}
        />
      ))}
    </Box>
  );
}

export function StorePage() {
  useDocumentTitle(
    "The store",
    "Browse Eight Two Five's limited, themed collections and made-to-measure designs, cut to order in Accra.",
  );
  const [searchParams, setSearchParams] = useSearchParams();
  const [term, setTerm] = useState(() => searchParams.get("q") ?? "");
  const q = useDebouncedValue(term.trim(), 300);

  // Keep the page URL shareable: /store?q=… follows the debounced search.
  useEffect(() => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (q) next.set("q", q);
        else next.delete("q");
        return next;
      },
      { replace: true },
    );
  }, [q, setSearchParams]);

  const settings = usePublicSettings();
  const collections = usePublicCollections();
  const designs = usePublicDesigns(q ? { q } : {});
  // Same query as the unfiltered grid, so collection covers cost no extra fetch.
  const allDesigns = usePublicDesigns({});

  const cloudName = settings.data?.cloudName ?? "";
  const designCountLabel = designs.isSuccess
    ? `${designs.data.length.toLocaleString("en-GH")} ${designs.data.length === 1 ? "design" : "designs"}`
    : q
      ? "Searching"
      : "Live catalog";

  const coverByCollection = useMemo(() => {
    const covers = new Map<string, string>();
    for (const design of (allDesigns.data ?? []) as Design[]) {
      if (covers.has(design.collectionId)) continue;
      const photo = sortedPhotos(design)[0];
      if (photo) covers.set(design.collectionId, photo.publicId);
    }
    return covers;
  }, [allDesigns.data]);

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 4, md: 6 } }}>
        <PageBanner
          tone="ink"
          icon={<StorefrontOutlined />}
          breadcrumbs={[{ label: "Home", to: "/" }, { label: "Store" }]}
          title="The Store"
          description="Live collections and limited runs, cut to order. Browse the themed collections below, or search every design that's currently open."
        />
      </Box>

      {/* Collections */}
      <Box component="section" sx={{ mb: { xs: 8, md: 10 } }}>
        <Stack
          direction={{ xs: "column", sm: "row" }}
          spacing={1}
          sx={{
            mb: 3,
            justifyContent: "space-between",
            alignItems: { sm: "baseline" },
          }}
        >
          <Typography variant="h2" component="h2">
            Collections
          </Typography>
          {collections.isSuccess && (
            <Typography
              variant="overline"
              sx={{
                color: "text.secondary",
                fontVariantNumeric: "tabular-nums",
              }}
            >
              {collections.data.length.toLocaleString("en-GH")}{" "}
              {collections.data.length === 1 ? "collection" : "collections"}
            </Typography>
          )}
        </Stack>
        {collections.isPending ? (
          <CardGridSkeleton count={3} />
        ) : collections.isError ? (
          <ErrorState
            message={errorMessage(collections.error)}
            onRetry={() => collections.refetch()}
          />
        ) : collections.data.length === 0 ? (
          <EmptyState
            tone="ink"
            label="On the cutting table"
            title="The first collection is on the cutting table."
            body="Collections are limited and themed — around ten designs each. The first one hasn't dropped yet; join the list on the homepage and we'll write the moment it does."
          />
        ) : (
          <Box
            sx={{
              display: "grid",
              gridTemplateColumns: {
                xs: "repeat(2, 1fr)",
                sm: "repeat(3, 1fr)",
                md: "repeat(4, 1fr)",
              },
              gap: { xs: 2, md: 3 },
              rowGap: { xs: 3.5, md: 5 },
            }}
          >
            {collections.data.map((collection, index) => (
              <CollectionCard
                key={collection.id}
                collection={collection}
                coverPublicId={coverByCollection.get(collection.id) ?? null}
                cloudName={cloudName}
                tone={index % 2 === 0 ? sand : sandDeep}
              />
            ))}
          </Box>
        )}
      </Box>

      {/* All designs */}
      <Box component="section" sx={{ mb: { xs: 8, md: 12 } }}>
        <Stack
          direction={{ xs: "column", sm: "row" }}
          spacing={2}
          sx={{
            mb: 3,
            justifyContent: "space-between",
            alignItems: { sm: "flex-end" },
          }}
        >
          <Box>
            <Typography variant="h2" component="h2">
              All designs
            </Typography>
            <Typography
              variant="overline"
              component="p"
              sx={{
                color: "text.secondary",
                fontVariantNumeric: "tabular-nums",
              }}
            >
              {q ? `${designCountLabel} matching "${q}"` : designCountLabel}
            </Typography>
          </Box>
          <TextField
            value={term}
            onChange={(event) => setTerm(event.target.value)}
            placeholder="Search designs"
            size="small"
            slotProps={{
              htmlInput: { "aria-label": "Search designs" },
              input: {
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchOutlined fontSize="small" />
                  </InputAdornment>
                ),
              },
            }}
            sx={{ width: { xs: "100%", sm: 280 } }}
          />
        </Stack>
        {designs.isPending ? (
          <CardGridSkeleton count={4} />
        ) : designs.isError ? (
          <ErrorState
            message={errorMessage(designs.error)}
            onRetry={() => designs.refetch()}
          />
        ) : designs.data.length === 0 ? (
          q ? (
            <Stack spacing={2.5} sx={{ alignItems: "flex-start" }}>
              <EmptyState
                label="No match"
                title={`Nothing matches “${q}”.`}
                body="Try a shorter word — design names are short — or clear the search to see everything that's live."
              />
              <Button variant="outlined" onClick={() => setTerm("")}>
                Clear search
              </Button>
            </Stack>
          ) : (
            <EmptyState
              label="In the darkroom"
              title="Designs are being photographed."
              body="The live designs will appear here shortly. In the meantime, the collections above show what's coming."
            />
          )
        ) : (
          <DesignGrid designs={designs.data} cloudName={cloudName} />
        )}
      </Box>
    </StorefrontLayout>
  );
}
