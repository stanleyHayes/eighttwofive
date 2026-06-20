import { useEffect, useRef, useState, type FormEvent } from "react";
import Box from "@mui/material/Box";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Button from "@mui/material/Button";
import Alert from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import { ThemeProvider } from "@mui/material/styles";
import CheckIcon from "@mui/icons-material/Check";
import { ApiError } from "@/lib/api";
import { createAppTheme } from "@/theme";
import { useJoinWaitlist } from "./useJoinWaitlist";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// The waitlist lives on a permanently dark (ink) section, so its form renders
// in the dark theme regardless of the app's light/dark setting — otherwise the
// inputs pick up the light-mode white fill and clash with the dark surface.
const darkFormTheme = createAppTheme("dark");

interface FieldErrors {
  name?: string;
  email?: string;
}

export function WaitlistForm() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const nameRef = useRef<HTMLInputElement>(null);
  const emailRef = useRef<HTMLInputElement>(null);
  const successRef = useRef<HTMLDivElement>(null);
  const join = useJoinWaitlist();

  // Live regions inserted with their content are often not announced;
  // moving focus to the region forces the announcement and gives keyboard
  // users a sensible position after the form unmounts.
  useEffect(() => {
    if (join.isSuccess) successRef.current?.focus();
  }, [join.isSuccess]);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const errors: FieldErrors = {};
    if (!name.trim()) {
      errors.name = "Tell us your name.";
    }
    if (!EMAIL_RE.test(email.trim())) {
      errors.email = "Enter a valid email address, like ama@example.com.";
    }
    setFieldErrors(errors);
    if (errors.name) {
      nameRef.current?.focus();
      return;
    }
    if (errors.email) {
      emailRef.current?.focus();
      return;
    }
    join.mutate({ name: name.trim(), email: email.trim().toLowerCase() });
  };

  if (join.isSuccess) {
    return (
      <ThemeProvider theme={darkFormTheme}>
        <Box
          ref={successRef}
          tabIndex={-1}
          role="status"
          sx={{
            outline: "none",
            border: "1px solid",
            borderColor: "success.main",
            color: "text.primary",
            p: 4,
            maxWidth: 480,
          }}
        >
          <Stack direction="row" spacing={0.75} sx={{ alignItems: "center" }}>
            <CheckIcon
              aria-hidden
              sx={{ fontSize: 16, color: "success.main" }}
            />
            <Typography variant="overline" sx={{ color: "success.main" }}>
              you're on the list
            </Typography>
          </Stack>
          <Typography sx={{ mt: 1.5 }}>
            Thanks, {join.data.name}. We'll write to{" "}
            <Box component="span" sx={{ fontWeight: 600 }}>
              {join.data.email}
            </Box>{" "}
            the moment doors open.
          </Typography>
        </Box>
      </ThemeProvider>
    );
  }

  const isDuplicate =
    join.error instanceof ApiError && join.error.status === 409;
  const serverError = join.isError
    ? isDuplicate
      ? "You're already on the list — see you at launch."
      : "Something went wrong on our end. Try again in a moment."
    : null;

  return (
    <ThemeProvider theme={darkFormTheme}>
      <Box
        component="form"
        onSubmit={handleSubmit}
        noValidate
        sx={{ maxWidth: 480 }}
      >
        <Stack spacing={1.5}>
          <TextField
            label="Name"
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              if (fieldErrors.name)
                setFieldErrors((prev) => ({ ...prev, name: undefined }));
            }}
            inputRef={nameRef}
            error={Boolean(fieldErrors.name)}
            helperText={fieldErrors.name ?? " "}
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
              if (fieldErrors.email)
                setFieldErrors((prev) => ({ ...prev, email: undefined }));
            }}
            inputRef={emailRef}
            error={Boolean(fieldErrors.email)}
            helperText={fieldErrors.email ?? " "}
            autoComplete="email"
            fullWidth
            required
          />
          <Button
            type="submit"
            variant="contained"
            size="large"
            loading={join.isPending}
            sx={{ alignSelf: "flex-start", minWidth: 232 }}
          >
            Join the waitlist
          </Button>
          {serverError && (
            <Alert severity={isDuplicate ? "warning" : "error"}>
              {serverError}
            </Alert>
          )}
        </Stack>
      </Box>
    </ThemeProvider>
  );
}
