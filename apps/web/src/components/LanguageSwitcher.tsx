import Button from "@mui/material/Button";
import { useTranslation } from "react-i18next";
import { LANG_STORAGE_KEY } from "@/i18n";
import { monoFamily } from "@/theme";

/** Toggles between English and French; shows the current code. */
export function LanguageSwitcher({ color = "inherit" }: { color?: string }) {
  const { i18n } = useTranslation();
  const isFrench = i18n.language?.startsWith("fr");
  const next = isFrench ? "en" : "fr";

  return (
    <Button
      onClick={() => {
        void i18n.changeLanguage(next);
        window.localStorage.setItem(LANG_STORAGE_KEY, next);
      }}
      aria-label={`Switch language to ${next.toUpperCase()}`}
      sx={{
        color,
        minWidth: 0,
        px: 1,
        py: 0.5,
        fontFamily: monoFamily,
        fontSize: "0.6875rem",
        letterSpacing: "0.14em",
      }}
    >
      {isFrench ? "FR" : "EN"}
    </Button>
  );
}
