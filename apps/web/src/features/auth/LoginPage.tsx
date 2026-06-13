import { useEffect, useRef, useState, type FormEvent } from "react";
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
import { requestLoginLink } from "@/lib/api";
import { clayDeep } from "@/theme";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function LoginPage() {
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
      setNameError("Enter your name so we know how to address you.");
      return;
    }
    if (!EMAIL_RE.test(email.trim())) {
      setEmailError("Enter a valid email address, like ama@example.com.");
      emailRef.current?.focus();
      return;
    }
    setNameError(undefined);
    setEmailError(undefined);
    sendLink.mutate({ email: email.trim().toLowerCase(), name: name.trim() });
  };

  return (
    <Container component="main" maxWidth="lg">
      <Box sx={{ py: { xs: 8, md: 13 }, maxWidth: 480 }}>
        <Typography variant="overline" component="p" sx={{ color: clayDeep }}>
          sign in
        </Typography>
        <Typography variant="h2" component="h1" sx={{ mt: 1.5, mb: 2 }}>
          Welcome back
        </Typography>
        <Typography sx={{ color: "text.secondary", mb: 4 }}>
          Enter your email and we'll send you a sign-in link. No passwords to
          remember.
        </Typography>

        {sendLink.isSuccess ? (
          <Box
            ref={successRef}
            tabIndex={-1}
            role="status"
            sx={{
              outline: "none",
              border: "1px solid",
              borderColor: "success.main",
              p: 4,
            }}
          >
            <Stack direction="row" spacing={0.75} sx={{ alignItems: "center" }}>
              <CheckIcon aria-hidden sx={{ fontSize: 16, color: "success.main" }} />
              <Typography variant="overline" sx={{ color: "success.main" }}>
                check your email
              </Typography>
            </Stack>
            <Typography sx={{ mt: 1.5 }}>
              We sent a sign-in link to{" "}
              <Box component="span" sx={{ fontWeight: 600 }}>
                {email.trim().toLowerCase()}
              </Box>
              . Open it on this device to continue.
            </Typography>
          </Box>
        ) : (
          <Box component="form" onSubmit={handleSubmit} noValidate>
            <Stack spacing={1.5}>
              <TextField
                label="Name"
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
                label="Email"
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
                sx={{ alignSelf: "flex-start", minWidth: 232 }}
              >
                Email me a link
              </Button>
              {sendLink.isError && (
                <Alert severity="error">
                  Something went wrong on our end. Try again in a moment.
                </Alert>
              )}
            </Stack>
          </Box>
        )}

        <Typography sx={{ mt: 5 }}>
          <Link component={RouterLink} to="/" variant="overline" sx={{ color: "text.secondary" }}>
            back to the homepage
          </Link>
        </Typography>
      </Box>
    </Container>
  );
}
