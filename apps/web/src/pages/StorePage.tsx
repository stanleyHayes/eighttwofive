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
import { useTranslation } from "react-i18next";
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

// How many designs the all-designs grid reveals per "Load more" press.
const STORE_PAGE_SIZE = 12;

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
  const { t } = useTranslation();
  useDocumentTitle(
    t("store.documentTitle"),
    t("store.documentDescription"),
  );
  const [searchParams, setSearchParams] = useSearchParams();
  const [term, setTerm] = useState(() => searchParams.get("q") ?? "");
  const q = useDebouncedValue(term.trim(), 300);
  const [visibleCount, setVisibleCount] = useState(STORE_PAGE_SIZE);

  // A new search collapses the grid back to the first page.
  useEffect(() => {
    setVisibleCount(STORE_PAGE_SIZE);
  }, [q]);

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
    ? t("store.designCount", {
        count: designs.data.length,
        formattedCount: designs.data.length.toLocaleString("en-GH"),
      })
    : q
      ? t("store.searching")
      : t("store.liveCatalog");

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
          breadcrumbs={[
            { label: t("store.breadcrumbHome"), to: "/" },
            { label: t("store.breadcrumbStore") },
          ]}
          title={t("store.bannerTitle")}
          description={t("store.bannerDescription")}
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
            {t("store.collectionsHeading")}
          </Typography>
          {collections.isSuccess && (
            <Typography
              variant="overline"
              sx={{
                color: "text.secondary",
                fontVariantNumeric: "tabular-nums",
              }}
            >
              {t("store.collectionCount", {
                count: collections.data.length,
                formattedCount: collections.data.length.toLocaleString("en-GH"),
              })}
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
            label={t("store.collectionsEmptyLabel")}
            title={t("store.collectionsEmptyTitle")}
            body={t("store.collectionsEmptyBody")}
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
              {t("store.allDesignsHeading")}
            </Typography>
            <Typography
              variant="overline"
              component="p"
              sx={{
                color: "text.secondary",
                fontVariantNumeric: "tabular-nums",
              }}
            >
              {q
                ? t("store.countMatching", { label: designCountLabel, query: q })
                : designCountLabel}
            </Typography>
          </Box>
          <TextField
            value={term}
            onChange={(event) => setTerm(event.target.value)}
            placeholder={t("store.searchPlaceholder")}
            size="small"
            slotProps={{
              htmlInput: { "aria-label": t("store.searchAriaLabel") },
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
                label={t("store.noMatchLabel")}
                title={t("store.noMatchTitle", { query: q })}
                body={t("store.noMatchBody")}
              />
              <Button variant="outlined" onClick={() => setTerm("")}>
                {t("store.clearSearch")}
              </Button>
            </Stack>
          ) : (
            <EmptyState
              label={t("store.designsEmptyLabel")}
              title={t("store.designsEmptyTitle")}
              body={t("store.designsEmptyBody")}
            />
          )
        ) : (
          <>
            <DesignGrid
              designs={designs.data.slice(0, visibleCount)}
              cloudName={cloudName}
            />
            {designs.data.length > visibleCount && (
              <Box
                sx={{ display: "flex", justifyContent: "center", mt: { xs: 4, md: 6 } }}
              >
                <Button
                  variant="outlined"
                  size="large"
                  onClick={() => setVisibleCount((v) => v + STORE_PAGE_SIZE)}
                >
                  {t("store.loadMore", {
                    remaining: designs.data.length - visibleCount,
                  })}
                </Button>
              </Box>
            )}
          </>
        )}
      </Box>
    </StorefrontLayout>
  );
}
