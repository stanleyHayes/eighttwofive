import { useEffect } from "react";
import { SITE_ORIGIN } from "@/lib/site";

const SUFFIX = "Eight Two Five";

function upsertMeta(attr: "name" | "property", key: string, content: string) {
  let el = document.head.querySelector(`meta[${attr}="${key}"]`);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  el.setAttribute("content", content);
}

function upsertCanonical(href: string) {
  let el = document.head.querySelector('link[rel="canonical"]');
  if (!el) {
    el = document.createElement("link");
    el.setAttribute("rel", "canonical");
    document.head.appendChild(el);
  }
  el.setAttribute("href", href);
}

/**
 * Sets the document title and the route's SEO meta (description, canonical, and
 * Open Graph title/url/description) for the current page. Search engines that
 * render the SPA pick these up per route; non-JS social crawlers get their tags
 * from the edge middleware instead.
 */
export function useDocumentTitle(title?: string, description?: string) {
  useEffect(() => {
    const full = title ? `${title} · ${SUFFIX}` : SUFFIX;
    document.title = full;

    const canonical = SITE_ORIGIN + window.location.pathname;
    upsertCanonical(canonical);
    upsertMeta("property", "og:title", full);
    upsertMeta("property", "og:url", canonical);

    if (description) {
      upsertMeta("name", "description", description);
      upsertMeta("property", "og:description", description);
    }
  }, [title, description]);
}
