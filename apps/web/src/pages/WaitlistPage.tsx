import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { WaitlistForm } from "@/features/waitlist/WaitlistForm";
import { clayDeep } from "@/theme";

export function WaitlistPage() {
  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 8, md: 13 }, maxWidth: 560 }}>
        <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
          stay in the loop
        </Typography>
        <Typography variant="h1" sx={{ mt: 2, mb: 3 }}>
          Be the first to know
        </Typography>
        <Typography sx={{ color: "text.secondary", mb: 4 }}>
          Join the list for new collection drops and restocks. No spam.
        </Typography>
        <WaitlistForm />
      </Box>
    </StorefrontLayout>
  );
}
