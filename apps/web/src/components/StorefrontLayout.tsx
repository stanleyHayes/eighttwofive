import { useState, type ReactNode } from "react";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Drawer from "@mui/material/Drawer";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import useScrollTrigger from "@mui/material/useScrollTrigger";
import MenuIcon from "@mui/icons-material/Menu";
import CloseIcon from "@mui/icons-material/Close";
import PersonIcon from "@mui/icons-material/PersonOutlined";
import InstagramIcon from "@mui/icons-material/Instagram";
import WhatsAppIcon from "@mui/icons-material/WhatsApp";
import MailOutlined from "@mui/icons-material/MailOutlined";
import { Link as RouterLink, useLocation } from "react-router";
import { useTranslation } from "react-i18next";
import { BrandMark } from "@/components/BrandMark";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { MeasureRule } from "@/components/MeasureRule";
import { ThemeToggle } from "@/components/ThemeToggle";
import { usePublicSettings } from "@/features/storefront/hooks";
import {
  amber,
  brass,
  cream,
  creamMuted,
  creamText,
  displayFamily,
  ink,
  monoFamily,
} from "@/theme";

const WORDMARK = "Eight Two Five";

const CONTACT_EMAIL = "hello@eighttwofive.com";

interface NavItem {
  tKey: string;
  to: string;
  also: string[];
}

const NAV_ITEMS: NavItem[] = [
  { tKey: "nav.store", to: "/store", also: ["/collections", "/designs"] },
  { tKey: "nav.fit", to: "/fit-guide", also: ["/slots"] },
  { tKey: "nav.atelier", to: "/about", also: [] },
  { tKey: "nav.contact", to: "/contact", also: [] },
];

function isNavActive(item: NavItem, pathname: string): boolean {
  return [item.to, ...item.also].some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
  );
}

function UtilityLink({
  href,
  icon,
  children,
  hideOnXs,
}: {
  href: string;
  icon: ReactNode;
  children: ReactNode;
  hideOnXs?: boolean;
}) {
  const external = href.startsWith("http");
  return (
    <Link
      href={href}
      {...(external ? { target: "_blank", rel: "noopener" } : {})}
      underline="none"
      sx={{
        display: hideOnXs ? { xs: "none", md: "inline-flex" } : "inline-flex",
        alignItems: "center",
        gap: 0.5,
        color: creamMuted,
        fontFamily: monoFamily,
        fontSize: "0.6875rem",
        letterSpacing: "0.04em",
        transition: "color 160ms ease",
        "&:hover": { color: amber },
      }}
    >
      {icon}
      <Box component="span" sx={{ whiteSpace: "nowrap" }}>
        {children}
      </Box>
    </Link>
  );
}

/** Slim top bar: contact + email on the left, social + language on the right. */
export function UtilityBar() {
  const settings = usePublicSettings();
  const whatsappDisplay = settings.data?.whatsappNumber ?? "";
  const whatsapp = whatsappDisplay.replace(/\D/g, "");

  return (
    <Box
      sx={{
        bgcolor: ink,
        color: creamMuted,
        borderBottom: "1px solid rgba(232,222,203,0.1)",
      }}
    >
      <Container maxWidth="lg">
        <Stack
          direction="row"
          sx={{
            minHeight: 38,
            alignItems: "center",
            justifyContent: "space-between",
            gap: 1,
          }}
        >
          <Stack
            direction="row"
            spacing={{ xs: 1.5, sm: 3 }}
            sx={{ alignItems: "center", minWidth: 0 }}
          >
            {whatsappDisplay && (
              <UtilityLink
                href={`https://wa.me/${whatsapp}`}
                icon={<WhatsAppIcon sx={{ fontSize: 15 }} />}
              >
                {whatsappDisplay}
              </UtilityLink>
            )}
            <UtilityLink
              href={`mailto:${CONTACT_EMAIL}`}
              icon={<MailOutlined sx={{ fontSize: 15 }} />}
              hideOnXs
            >
              {CONTACT_EMAIL}
            </UtilityLink>
          </Stack>

          <Stack direction="row" spacing={0.25} sx={{ alignItems: "center" }}>
            <IconButton
              component="a"
              href="https://instagram.com"
              target="_blank"
              rel="noopener"
              aria-label="Instagram"
              size="small"
              sx={{ color: creamText, "&:hover": { color: amber } }}
            >
              <InstagramIcon sx={{ fontSize: 17 }} />
            </IconButton>
            {whatsapp && (
              <IconButton
                component="a"
                href={`https://wa.me/${whatsapp}`}
                target="_blank"
                rel="noopener"
                aria-label="WhatsApp"
                size="small"
                sx={{ color: creamText, "&:hover": { color: amber } }}
              >
                <WhatsAppIcon sx={{ fontSize: 17 }} />
              </IconButton>
            )}
            <Box
              sx={{
                width: "1px",
                height: 16,
                bgcolor: "rgba(232,222,203,0.2)",
                mx: 0.5,
              }}
            />
            <LanguageSwitcher color={creamText} />
          </Stack>
        </Stack>
      </Container>
    </Box>
  );
}

function NavLink({ item, active }: { item: NavItem; active: boolean }) {
  const { t } = useTranslation();
  return (
    <Link
      component={RouterLink}
      to={item.to}
      underline="none"
      variant="overline"
      aria-current={active ? "page" : undefined}
      sx={{
        position: "relative",
        color: active ? cream : creamMuted,
        py: 0.5,
        "&::after": {
          content: '""',
          position: "absolute",
          left: 0,
          bottom: 0,
          height: "1px",
          width: "100%",
          backgroundColor: amber,
          transformOrigin: "left",
          transform: active ? "scaleX(1)" : "scaleX(0)",
          transition: "transform 280ms cubic-bezier(0.22, 1, 0.36, 1)",
        },
        "&:hover": { color: cream },
        "&:hover::after": { transform: "scaleX(1)" },
      }}
    >
      {t(item.tKey)}
    </Link>
  );
}

function Wordmark() {
  const { t } = useTranslation();
  return (
    <Link
      component={RouterLink}
      to="/"
      underline="none"
      aria-label={t("layout.wordmarkAria", { brand: WORDMARK })}
      sx={{
        color: cream,
        fontFamily: displayFamily,
        fontWeight: 700,
        fontSize: { xs: "1.1rem", sm: "1.3rem", md: "1.55rem" },
        letterSpacing: "-0.02em",
        lineHeight: 1,
        whiteSpace: "nowrap",
        display: "inline-flex",
        alignItems: "center",
        gap: { xs: 0.75, md: 1.25 },
      }}
    >
      <BrandMark size={22} sx={{ display: { xs: "none", sm: "block" } }} />
      {WORDMARK}
    </Link>
  );
}

function MobileNav({
  open,
  onClose,
  pathname,
}: {
  open: boolean;
  onClose: () => void;
  pathname: string;
}) {
  const { t } = useTranslation();
  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          sx: {
            width: { xs: "84vw", sm: 380 },
            bgcolor: ink,
            backgroundImage: "none",
            color: cream,
          },
        },
      }}
    >
      <Stack sx={{ height: "100%", p: 3 }}>
        <Stack
          direction="row"
          sx={{ justifyContent: "space-between", alignItems: "center", mb: 5 }}
        >
          <Box
            component="span"
            sx={{
              fontFamily: monoFamily,
              fontSize: "0.6875rem",
              letterSpacing: "0.2em",
              color: brass,
              textTransform: "uppercase",
            }}
          >
            {t("nav.menu")}
          </Box>
          <IconButton
            onClick={onClose}
            aria-label={t("nav.menu")}
            sx={{ color: cream }}
          >
            <CloseIcon />
          </IconButton>
        </Stack>

        <Stack spacing={1} sx={{ flex: 1 }}>
          {NAV_ITEMS.map((item, index) => {
            const active = isNavActive(item, pathname);
            return (
              <Link
                key={item.to}
                component={RouterLink}
                to={item.to}
                onClick={onClose}
                underline="none"
                aria-current={active ? "page" : undefined}
                sx={{
                  fontFamily: displayFamily,
                  fontWeight: 600,
                  fontSize: "2.2rem",
                  letterSpacing: "-0.02em",
                  color: active ? amber : cream,
                  py: 1,
                  display: "flex",
                  alignItems: "baseline",
                  gap: 1.5,
                }}
              >
                <Box
                  component="span"
                  sx={{
                    fontFamily: monoFamily,
                    fontSize: "0.75rem",
                    color: brass,
                  }}
                >
                  0{index + 1}
                </Box>
                {t(item.tKey)}
              </Link>
            );
          })}
        </Stack>

        <MeasureRule variant="light" sx={{ mb: 3 }} />
        <Link
          component={RouterLink}
          to="/account"
          onClick={onClose}
          underline="none"
          variant="overline"
          sx={{
            color: creamText,
            display: "inline-flex",
            alignItems: "center",
            gap: 1,
          }}
        >
          <PersonIcon sx={{ fontSize: 18 }} /> {t("nav.account")}
        </Link>
      </Stack>
    </Drawer>
  );
}

function StorefrontHeader() {
  const { t } = useTranslation();
  const { pathname } = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);
  const scrolled = useScrollTrigger({ disableHysteresis: true, threshold: 24 });

  return (
    <Box
      component="header"
      sx={{
        position: "sticky",
        top: 0,
        zIndex: (t) => t.zIndex.appBar,
        bgcolor: scrolled ? "rgba(22, 18, 13, 0.86)" : ink,
        backdropFilter: scrolled ? "saturate(140%) blur(10px)" : "none",
        borderBottom: "1px solid",
        borderColor: scrolled ? "rgba(232,222,203,0.14)" : "transparent",
        transition: "background-color 240ms ease, border-color 240ms ease",
      }}
    >
      <Container maxWidth="lg">
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: "1fr auto 1fr",
            alignItems: "center",
            py: scrolled ? { xs: 1.25, md: 1.5 } : { xs: 1.75, md: 2.5 },
            transition: "padding 240ms ease",
          }}
        >
          <Box sx={{ justifySelf: "start", minWidth: 0 }}>
            <Stack
              component="nav"
              aria-label={t("layout.primaryNavAria")}
              direction="row"
              spacing={4}
              sx={{ display: { xs: "none", md: "flex" }, alignItems: "center" }}
            >
              {NAV_ITEMS.map((item) => (
                <NavLink
                  key={item.to}
                  item={item}
                  active={isNavActive(item, pathname)}
                />
              ))}
            </Stack>
            <IconButton
              onClick={() => setMenuOpen(true)}
              aria-label={t("layout.openMenu")}
              edge="start"
              sx={{ display: { xs: "inline-flex", md: "none" }, color: cream }}
            >
              <MenuIcon />
            </IconButton>
          </Box>

          <Box sx={{ justifySelf: "center", textAlign: "center", minWidth: 0 }}>
            <Wordmark />
          </Box>

          <Box sx={{ justifySelf: "end" }}>
            <Stack direction="row" spacing={0.5} sx={{ alignItems: "center" }}>
              <Box
                component="span"
                sx={{
                  display: { xs: "none", md: "inline-flex" },
                  alignItems: "center",
                  px: 1.1,
                  py: 0.35,
                  mr: 0.75,
                  border: "1px solid",
                  borderColor: brass,
                  color: brass,
                  fontFamily: monoFamily,
                  fontSize: "0.625rem",
                  letterSpacing: "0.16em",
                }}
              >
                825
              </Box>
              <ThemeToggle color={cream} />
              <IconButton
                component={RouterLink}
                to="/account"
                aria-label={t("nav.account")}
                sx={{ color: cream }}
              >
                <PersonIcon sx={{ fontSize: 21 }} />
              </IconButton>
            </Stack>
          </Box>
        </Box>
      </Container>

      <MobileNav
        open={menuOpen}
        onClose={() => setMenuOpen(false)}
        pathname={pathname}
      />
    </Box>
  );
}

const BENEFIT_KEYS = [
  "benefits.measurements",
  "benefits.payment",
  "benefits.limited",
  "benefits.fulfilment",
] as const;

export function BenefitsRow() {
  const { t } = useTranslation();
  return (
    <Box
      sx={{
        borderTop: "1px solid",
        borderColor: "divider",
        bgcolor: "background.default",
      }}
    >
      <Container maxWidth="lg">
        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: { xs: "1fr 1fr", md: "repeat(4, 1fr)" },
            gap: 0,
            py: { xs: 3, md: 4 },
          }}
        >
          {BENEFIT_KEYS.map((key, index) => (
            <Box
              key={key}
              sx={{
                textAlign: "center",
                px: 2,
                py: { xs: 1.5, md: 0 },
                borderLeft: { md: index === 0 ? "none" : "1px solid" },
                borderColor: { md: "divider" },
              }}
            >
              <Typography
                variant="overline"
                component="p"
                sx={{ color: "text.secondary" }}
              >
                {t(key)}
              </Typography>
            </Box>
          ))}
        </Box>
      </Container>
    </Box>
  );
}

function FooterColumn({
  heading,
  children,
}: {
  heading: string;
  children: ReactNode;
}) {
  return (
    <Stack spacing={1.75}>
      <Box
        component="span"
        sx={{
          fontFamily: monoFamily,
          fontSize: "0.6875rem",
          letterSpacing: "0.18em",
          color: brass,
          textTransform: "uppercase",
        }}
      >
        {heading}
      </Box>
      {children}
    </Stack>
  );
}

function FooterLink({
  to,
  href,
  children,
}: {
  to?: string;
  href?: string;
  children: ReactNode;
}) {
  const { pathname } = useLocation();
  const active = to ? pathname === to || pathname.startsWith(`${to}/`) : false;
  const sx = {
    color: active ? amber : creamText,
    fontSize: "0.9rem",
    width: "fit-content",
    transition: "color 160ms ease",
    "&:hover": { color: amber },
  } as const;
  if (href) {
    return (
      <Link href={href} target="_blank" rel="noopener" underline="none" sx={sx}>
        {children}
      </Link>
    );
  }
  return (
    <Link
      component={RouterLink}
      to={to ?? "/"}
      underline="none"
      aria-current={active ? "page" : undefined}
      sx={sx}
    >
      {children}
    </Link>
  );
}

export function SiteFooter() {
  const { t } = useTranslation();
  const settings = usePublicSettings();
  const whatsapp = settings.data?.whatsappNumber?.replace(/\D/g, "") ?? "";
  const location = settings.data?.visitLocation || t("layout.defaultLocation");

  return (
    <Box
      component="footer"
      sx={{ bgcolor: ink, color: cream, pt: { xs: 6, md: 9 }, pb: 4 }}
    >
      <Container maxWidth="lg">
        <MeasureRule
          variant="light"
          label="FIG. 05"
          caption={t("footer.theHouse")}
          sx={{ mb: { xs: 5, md: 7 } }}
        />

        <Box
          sx={{
            display: "grid",
            gridTemplateColumns: {
              xs: "1fr",
              sm: "1fr 1fr",
              md: "1.5fr 1fr 1fr 1fr",
            },
            gap: { xs: 4.5, md: 4 },
            mb: { xs: 5, md: 8 },
          }}
        >
          <Stack
            spacing={2.5}
            sx={{ gridColumn: { sm: "1 / -1", md: "auto" }, maxWidth: 380 }}
          >
            <Typography
              sx={{
                fontFamily: displayFamily,
                fontWeight: 700,
                fontSize: { xs: "2.4rem", md: "2.9rem" },
                lineHeight: 0.92,
                letterSpacing: "-0.03em",
              }}
            >
              Eight
              <br />
              Two Five
            </Typography>
            <Typography
              variant="body2"
              sx={{ color: creamMuted, maxWidth: "34ch" }}
            >
              {t("footer.tagline", { location })}
            </Typography>
          </Stack>

          <FooterColumn heading={t("footer.shop")}>
            <FooterLink to="/store">{t("footer.theStore")}</FooterLink>
            <FooterLink to="/account">{t("footer.yourAccount")}</FooterLink>
            <FooterLink to="/login">{t("footer.signIn")}</FooterLink>
          </FooterColumn>

          <FooterColumn heading={t("footer.theHouse")}>
            <FooterLink to="/about">{t("footer.ourStory")}</FooterLink>
            <FooterLink to="/fit-guide">{t("footer.fitGuide")}</FooterLink>
            <FooterLink to="/slots">{t("footer.bookVisit")}</FooterLink>
            <FooterLink to="/contact">{t("footer.contact")}</FooterLink>
          </FooterColumn>

          <FooterColumn heading={t("footer.connect")}>
            <FooterLink href="https://instagram.com">
              {t("footer.instagram")}
            </FooterLink>
            {whatsapp && (
              <FooterLink href={`https://wa.me/${whatsapp}`}>
                {t("footer.whatsapp")}
              </FooterLink>
            )}
            <FooterLink href="mailto:hello@eighttwofive.com">
              {t("footer.email")}
            </FooterLink>
          </FooterColumn>
        </Box>

        <Box sx={{ height: "1px", bgcolor: "rgba(232,222,203,0.16)", mb: 3 }} />

        <Stack
          direction={{ xs: "column", sm: "row" }}
          spacing={{ xs: 1.5, sm: 2 }}
          sx={{
            justifyContent: "space-between",
            alignItems: { xs: "flex-start", sm: "center" },
          }}
        >
          <Box
            component="span"
            sx={{
              fontFamily: monoFamily,
              fontSize: "0.6875rem",
              letterSpacing: "0.16em",
              color: creamMuted,
              textTransform: "uppercase",
            }}
          >
            {t("layout.copyright", { brand: WORDMARK, location })}
          </Box>
          <Stack
            direction="row"
            spacing={2.5}
            sx={{ alignItems: "center", flexWrap: "wrap" }}
          >
            <Box
              component="span"
              sx={{
                fontFamily: monoFamily,
                fontSize: "0.6875rem",
                letterSpacing: "0.16em",
                color: creamMuted,
                textTransform: "uppercase",
              }}
            >
              {t("footer.pricesIn")}
            </Box>
            <Box
              component="span"
              sx={{
                fontFamily: monoFamily,
                fontSize: "0.6875rem",
                letterSpacing: "0.16em",
                color: creamMuted,
                textTransform: "uppercase",
              }}
            >
              {t("footer.payments")}
            </Box>
          </Stack>
        </Stack>
      </Container>
    </Box>
  );
}

function SkipToContent() {
  const { t } = useTranslation();
  return (
    <Link
      href="#main-content"
      sx={{
        position: "absolute",
        top: -999,
        left: -999,
        zIndex: 9999,
        px: 2,
        py: 1,
        bgcolor: "background.paper",
        color: "text.primary",
        textDecoration: "none",
        "&:focus": { top: 8, left: 8 },
      }}
    >
      {t("layout.skipToContent")}
    </Link>
  );
}

/**
 * Shared shell for the public store pages. `bleedHero` renders a full-bleed
 * hero directly under the sticky header, outside the centered container.
 */
export function StorefrontLayout({
  children,
  bleedHero,
}: {
  children: ReactNode;
  bleedHero?: ReactNode;
}) {
  return (
    <Box
      sx={{
        minHeight: "100dvh",
        display: "flex",
        flexDirection: "column",
        bgcolor: "background.default",
      }}
    >
      <SkipToContent />
      <UtilityBar />
      <StorefrontHeader />
      {bleedHero}
      <Container
        id="main-content"
        component="main"
        maxWidth="lg"
        sx={{ flex: 1 }}
        tabIndex={-1}
      >
        {children}
      </Container>
      <BenefitsRow />
      <SiteFooter />
    </Box>
  );
}
