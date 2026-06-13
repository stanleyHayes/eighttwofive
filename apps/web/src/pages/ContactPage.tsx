import type { ReactNode } from "react";
import Box from "@mui/material/Box";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import MailOutlined from "@mui/icons-material/MailOutlined";
import { PageBanner } from "@/components/PageBanner";
import { StorefrontLayout } from "@/components/StorefrontLayout";
import { usePublicSettings } from "@/features/storefront/hooks";
import { useDocumentTitle } from "@/lib/useDocumentTitle";

function waHref(whatsappNumber: string): string {
  return `https://wa.me/${whatsappNumber.replace(/\D/g, "")}`;
}

function ContactRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Box sx={{ borderTop: "1px solid", borderColor: "divider", py: 3 }}>
      <Typography variant="overline" component="p" sx={{ color: "text.secondary", mb: 0.75 }}>
        {label}
      </Typography>
      {children}
    </Box>
  );
}

export function ContactPage() {
  useDocumentTitle(
    "Contact",
    "Talk to the Eight Two Five atelier — WhatsApp, email, or book a visit to be measured in person in Accra.",
  );
  const settings = usePublicSettings();
  const whatsapp = settings.data?.whatsappNumber ?? "";
  const visitLocation = settings.data?.visitLocation ?? "";

  return (
    <StorefrontLayout>
      <Box sx={{ py: { xs: 4, md: 6 } }}>
        <PageBanner
          tone="ink"
          icon={<MailOutlined />}
          breadcrumbs={[{ label: "Home", to: "/" }, { label: "Contact" }]}
          title="Talk to the atelier."
          description="Questions about a design, your measurements, or a visit to be measured in person — WhatsApp is the fastest way to reach us."
        />
      </Box>

      <Box component="section" sx={{ mb: { xs: 8, md: 12 }, maxWidth: 640 }}>
        <Typography
          variant="subtitle1"
          sx={{ color: "text.secondary", maxWidth: "54ch", mb: 5 }}
        >
          Questions about a design, your measurements, or a visit to be measured in person —
          WhatsApp is the fastest way to reach us.
        </Typography>

        <Stack>
          <ContactRow label="whatsapp">
            {whatsapp ? (
              <Link
                href={waHref(whatsapp)}
                target="_blank"
                rel="noreferrer"
                underline="hover"
                sx={{ color: "text.primary" }}
              >
                <Typography variant="h5" component="span" sx={{ letterSpacing: "0.06em" }}>
                  {whatsapp}
                </Typography>
              </Link>
            ) : (
              <Typography variant="body2" sx={{ color: "text.secondary" }}>
                The WhatsApp line opens with the storefront.
              </Typography>
            )}
          </ContactRow>

          <ContactRow label="visit">
            {visitLocation ? (
              <Typography variant="h5" component="p">
                {visitLocation}
              </Typography>
            ) : (
              <Typography variant="body2" sx={{ color: "text.secondary" }}>
                In-person measuring visits are by appointment — location coming soon.
              </Typography>
            )}
          </ContactRow>

          <ContactRow label="email">
            <Link href="mailto:hello@eighttwofive.com" underline="hover" sx={{ color: "text.primary" }}>
              <Typography variant="h5" component="span" sx={{ letterSpacing: "0.04em" }}>
                hello@eighttwofive.com
              </Typography>
            </Link>
          </ContactRow>
        </Stack>
      </Box>
    </StorefrontLayout>
  );
}
