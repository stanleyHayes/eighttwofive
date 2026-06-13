import Box from "@mui/material/Box";
import Skeleton from "@mui/material/Skeleton";
import { useParams } from "react-router";
import CollectionsOutlined from "@mui/icons-material/CollectionsOutlined";
import { EmptyState, ErrorState } from "@/components/EmptyState";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { errorMessage } from "@/features/catalog/api";
import { DesignGrid } from "@/features/storefront/DesignCard";
import { RetiredPanel } from "@/features/storefront/RetiredPanel";
import { usePublicCollection, usePublicSettings } from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";
import { ApiError } from "@/lib/api";

export function CollectionPage() {
  const { slug = "" } = useParams();
  const settings = usePublicSettings();
  const collection = usePublicCollection(slug);

  useDocumentTitle(
    collection.data?.collection.name,
    collection.data?.collection.note || undefined,
  );

  const cloudName = settings.data?.cloudName ?? "";
  const notFound =
    collection.isError && collection.error instanceof ApiError && collection.error.status === 404;

  if (notFound) {
    return (
      <StorefrontLayout>
        <RetiredPanel
          overline="collection retired"
          title="This collection has been retired"
          body="Every collection is a limited run — when the fabric is gone, it's gone. The live collections are still open in the store."
        />
      </StorefrontLayout>
    );
  }

  return (
    <StorefrontLayout>
      {collection.isPending ? (
        <Box sx={{ py: { xs: 6, md: 9 } }}>
          <Skeleton width={120} />
          <Skeleton width={360} height={72} sx={{ mt: 1 }} />
          <Box
            sx={{
              mt: 5,
              display: "grid",
              gridTemplateColumns: {
                xs: "repeat(2, 1fr)",
                sm: "repeat(3, 1fr)",
                md: "repeat(4, 1fr)",
              },
              gap: { xs: 2, md: 3 },
            }}
          >
            {Array.from({ length: 4 }, (_, i) => (
              <Skeleton
                key={i}
                variant="rectangular"
                sx={{ aspectRatio: "600 / 780", height: "auto" }}
              />
            ))}
          </Box>
        </Box>
      ) : collection.isError ? (
        <Box sx={{ py: { xs: 6, md: 9 } }}>
          <ErrorState
            message={errorMessage(collection.error)}
            onRetry={() => collection.refetch()}
          />
        </Box>
      ) : (
        <>
          <Box sx={{ py: { xs: 4, md: 6 } }}>
            <PageBanner
              tone="ink"
              icon={<CollectionsOutlined />}
              breadcrumbs={[
                { label: "Home", to: "/" },
                { label: "Store", to: "/store" },
                { label: collection.data.collection.name },
              ]}
              title={collection.data.collection.name}
              description={collection.data.collection.note || undefined}
            />
          </Box>

          <Box component="section" sx={{ mb: { xs: 8, md: 12 } }}>
            {collection.data.designs.length === 0 ? (
              <EmptyState
                label="In the darkroom"
                title="Designs are on their way."
                body="This collection is live but its designs are still being photographed. Check back shortly."
              />
            ) : (
              <DesignGrid designs={collection.data.designs} cloudName={cloudName} />
            )}
          </Box>
        </>
      )}
    </StorefrontLayout>
  );
}
