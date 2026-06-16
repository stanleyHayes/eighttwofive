import { useEffect, useRef, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import CheckIcon from "@mui/icons-material/Check";
import Link from "@mui/material/Link";
import { Link as RouterLink } from "react-router";
import { useMutation } from "@tanstack/react-query";
import { BrandMark } from "@/components/BrandMark";
import { MeasureRule } from "@/components/MeasureRule";
import { requestLoginLink } from "@/lib/api";
import { amber, brass, cream, creamMuted, GRAIN_URL, ink } from "@/theme";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function LoginPage() {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [emailError, setEmailError] = useState<string | undefined>(undefined);
  const [nameError, setNameError] = useState<string | undefined>(undefined);
  const emailRef = useRef<HTMLInputElement>(null);
  const successRef = useRef<HTMLDivElement>(null);
  const sendLink = useMutation({ mutationFn: requestLoginLink });

  // Live regions inserted with their content are often not announced;
  // moving focus to the region forces the announcement and gives keyboard
  // users a sensible position after the form unmounts.
  useEffect(() => {
    if (sendLink.isSuccess) successRef.current?.focus();
  }, [sendLink.isSuccess]);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (name.trim() === "") {
      setNameError(t("login.nameError"));
      return;
    }
    if (!EMAIL_RE.test(email.trim())) {
      setEmailError(t("login.emailError"));
      emailRef.current?.focus();
      return;
    }
    setNameError(undefined);
    setEmailError(undefined);
    sendLink.mutate({ email: email.trim().toLowerCase(), name: name.trim() });
  };

  return (
    <Container
      component="main"
      maxWidth={false}
      disableGutters
      sx={{
        minHeight: "100dvh",
        bgcolor: "background.default",
        display: "grid",
        gridTemplateColumns: { xs: "1fr", md: "minmax(360px, 0.92fr) 1fr" },
      }}
    >
      <Box
        sx={{
          display: { xs: "none", md: "flex" },
          flexDirection: "column",
          justifyContent: "space-between",
          bgcolor: ink,
          color: cream,
          p: { md: 6, lg: 8 },
          minHeight: "100dvh",
          backgroundImage: GRAIN_URL,
          backgroundBlendMode: "overlay",
        }}
      >
        <Link
          component={RouterLink}
          to="/"
          underline="none"
          sx={{
            color: cream,
            display: "inline-flex",
            alignItems: "center",
            gap: 1.25,
            width: "fit-content",
          }}
        >
          <BrandMark size={26} />
          <Typography sx={{ fontFamily: "inherit", fontWeight: 700 }}>
            Eight Two Five
          </Typography>
        </Link>

        <Box>
          <MeasureRule
            variant="light"
            label={t("login.asideLabel")}
            caption={t("login.asideCaption")}
            sx={{ mb: 4 }}
          />
          <Typography
            variant="h2"
            component="p"
            sx={{ color: cream, maxWidth: "10ch" }}
          >
            {t("login.asideHeading")}
          </Typography>
          <Typography sx={{ color: creamMuted, mt: 2.5, maxWidth: "34ch" }}>
            {t("login.asideBody")}
          </Typography>
        </Box>

        <Stack direction="row" spacing={2} sx={{ color: creamMuted }}>
          {[
            t("login.featureNoPassword"),
            t("login.featureOrderUpdates"),
            t("login.featureSecureLink"),
          ].map((item) => (
            <Typography key={item} variant="overline" component="span">
              {item}
            </Typography>
          ))}
        </Stack>
      </Box>

      <Box
        sx={{
          minHeight: { xs: "100dvh", md: "auto" },
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          px: { xs: 2.5, sm: 4, md: 6 },
          py: { xs: 6, md: 8 },
        }}
      >
        <Box sx={{ width: "100%", maxWidth: 500 }}>
          <Link
            component={RouterLink}
            to="/"
            underline="none"
            sx={{
              color: "text.primary",
              display: { xs: "inline-flex", md: "none" },
              alignItems: "center",
              gap: 1,
              mb: 5,
            }}
          >
            <BrandMark size={24} />
            <Typography variant="overline" component="span">
              Eight Two Five
            </Typography>
          </Link>

          <Typography variant="overline" component="p" sx={{ color: brass }}>
            {t("login.eyebrow")}
          </Typography>
          <Typography variant="h2" component="h1" sx={{ mt: 1.5, mb: 2 }}>
            {t("login.heading")}
          </Typography>
          <Typography sx={{ color: "text.secondary", mb: 4, maxWidth: "42ch" }}>
            {t("login.intro")}
          </Typography>

          {sendLink.isSuccess ? (
            <Box
              ref={successRef}
              tabIndex={-1}
              role="status"
              sx={{
                outline: "none",
                bgcolor: "background.paper",
                border: "1px solid",
                borderColor: "success.main",
                p: 4,
              }}
            >
              <Stack
                direction="row"
                spacing={0.75}
                sx={{ alignItems: "center" }}
              >
                <CheckIcon
                  aria-hidden
                  sx={{ fontSize: 16, color: "success.main" }}
                />
                <Typography variant="overline" sx={{ color: "success.main" }}>
                  {t("login.successLabel")}
                </Typography>
              </Stack>
              <Typography sx={{ mt: 1.5 }}>
                {t("login.successBodyBefore")}{" "}
                <Box component="span" sx={{ fontWeight: 600 }}>
                  {email.trim().toLowerCase()}
                </Box>
                {t("login.successBodyAfter")}
              </Typography>
            </Box>
          ) : (
            <Box
              component="form"
              onSubmit={handleSubmit}
              noValidate
              sx={{
                bgcolor: "background.paper",
                border: "1px solid",
                borderColor: "divider",
                p: { xs: 3, sm: 4 },
              }}
            >
              <Stack spacing={1.5}>
                <TextField
                  label={t("login.nameLabel")}
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value);
                    if (nameError) setNameError(undefined);
                  }}
                  error={Boolean(nameError)}
                  helperText={nameError ?? " "}
                  autoComplete="name"
                  fullWidth
                  required
                />
                <TextField
                  label={t("login.emailLabel")}
                  type="email"
                  value={email}
                  onChange={(e) => {
                    setEmail(e.target.value);
                    if (emailError) setEmailError(undefined);
                  }}
                  inputRef={emailRef}
                  error={Boolean(emailError)}
                  helperText={emailError ?? " "}
                  autoComplete="email"
                  fullWidth
                  required
                />
                <Button
                  type="submit"
                  variant="contained"
                  size="large"
                  loading={sendLink.isPending}
                  sx={{
                    alignSelf: "flex-start",
                    minWidth: 232,
                    bgcolor: amber,
                    color: ink,
                    "&:hover": { bgcolor: brass },
                  }}
                >
                  {t("login.submit")}
                </Button>
                {sendLink.isError && (
                  <Alert severity="error">{t("login.submitError")}</Alert>
                )}
              </Stack>
            </Box>
          )}

          <Typography sx={{ mt: 5 }}>
            <Link
              component={RouterLink}
              to="/"
              variant="overline"
              sx={{ color: "text.secondary" }}
            >
              {t("login.backHome")}
            </Link>
          </Typography>
        </Box>
      </Box>
    </Container>
  );
}
