import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { useTranslation } from "react-i18next";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { WaitlistForm } from "@/features/waitlist/WaitlistForm";
import { clayDeep } from "@/theme";

export function WaitlistPage() {
  const { t } = useTranslation();

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 8, md: 13 }, maxWidth: 560 }}>
        <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
          {t("waitlist.eyebrow")}
        </Typography>
        <Typography variant="h1" sx={{ mt: 2, mb: 3 }}>
          {t("waitlist.title")}
        </Typography>
        <Typography sx={{ color: "text.secondary", mb: 4 }}>
          {t("waitlist.body")}
        </Typography>
        <WaitlistForm />
      </Box>
    </StorefrontLayout>
  );
}
