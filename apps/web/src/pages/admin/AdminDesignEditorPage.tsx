import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import CircularProgress from "@mui/material/CircularProgress";
import { Link as RouterLink, useParams } from "react-router";
import EditOutlined from "@mui/icons-material/EditOutlined";
import { PageBanner } from "@/components/PageBanner";
import { errorMessage } from "@/features/catalog/api";
import { useCollections, useDesigns } from "@/features/catalog/hooks";
import { DesignForm } from "@/features/catalog/DesignForm";

export function AdminDesignEditorPage() {
  const { id } = useParams<{ id: string }>();
  const isEdit = Boolean(id);

  const collections = useCollections();
  // The API exposes no single-design GET; load the unfiltered list and pick.
  const designs = useDesigns({}, { enabled: isEdit });

  if (collections.isPending || (isEdit && designs.isPending)) {
    return <CircularProgress aria-label="Loading design editor" />;
  }
  if (collections.isError) {
    return <Alert severity="error">{errorMessage(collections.error)}</Alert>;
  }
  if (isEdit && designs.isError) {
    return <Alert severity="error">{errorMessage(designs.error)}</Alert>;
  }

  const design = isEdit ? designs.data?.find((d) => d.id === id) : undefined;
  if (isEdit && !design) {
    return (
      <Box>
        <Alert severity="error">Design not found.</Alert>
        <Button component={RouterLink} to="/admin/designs" variant="text" sx={{ mt: 2, px: 1 }}>
          Back to designs
        </Button>
      </Box>
    );
  }

  return (
    <Box>
      <PageBanner
        tone="ink"
        icon={<EditOutlined />}
        breadcrumbs={[
          { label: "Admin", to: "/admin" },
          { label: "Designs", to: "/admin/designs" },
          { label: isEdit ? "Edit" : "New" },
        ]}
        title={isEdit ? "Edit design" : "New design"}
        description={
          isEdit
            ? "Update this design's details, size bands, and photography."
            : "Add a new design to the catalog — its collection, size bands, and photos."
        }
      />
      <Box sx={{ maxWidth: 720, mt: { xs: 4, md: 5 } }}>
        <DesignForm collections={collections.data} initial={design} />
      </Box>
    </Box>
  );
}
