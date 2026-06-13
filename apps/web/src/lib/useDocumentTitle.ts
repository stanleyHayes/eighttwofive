import { useEffect } from "react";

const SUFFIX = "Eight Two Five";

/**
 * Sets the document title (and optionally the meta description) for the current
 * route — lightweight client-side SEO for the SPA. Restores nothing on unmount;
 * the next route sets its own.
 */
export function useDocumentTitle(title?: string, description?: string) {
  useEffect(() => {
    document.title = title ? `${title} · ${SUFFIX}` : SUFFIX;

    if (description) {
      let meta = document.querySelector('meta[name="description"]');
      if (!meta) {
        meta = document.createElement("meta");
        meta.setAttribute("name", "description");
        document.head.appendChild(meta);
      }
      meta.setAttribute("content", description);
    }
  }, [title, description]);
}
