import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import en from "./locales/en.json";
import fr from "./locales/fr.json";

export const LANG_STORAGE_KEY = "e25-lang";
export const SUPPORTED_LANGS = ["en", "fr"] as const;
export type LangCode = (typeof SUPPORTED_LANGS)[number];

function initialLang(): LangCode {
  if (typeof window === "undefined") return "en";

  const saved = window.localStorage.getItem(LANG_STORAGE_KEY);
  if (saved === "en" || saved === "fr") return saved;

  return navigator.language?.startsWith("fr") ? "fr" : "en";
}

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    fr: { translation: fr },
  },
  lng: initialLang(),
  fallbackLng: "en",
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
});

export default i18n;
