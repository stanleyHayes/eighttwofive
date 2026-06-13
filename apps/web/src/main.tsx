import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { ThemeModeProvider } from "@/components/ThemeModeProvider";
import { queryClient } from "./lib/queryClient";
import "@/i18n";
import App from "./App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeModeProvider>
        <App />
      </ThemeModeProvider>
    </QueryClientProvider>
  </StrictMode>,
);

// Fade out the boot splash once React has mounted.
const splash = document.getElementById("splash");
if (splash) {
  requestAnimationFrame(() => {
    window.setTimeout(() => {
      splash.classList.add("is-hidden");
      splash.addEventListener("transitionend", () => splash.remove(), { once: true });
    }, 350);
  });
}
