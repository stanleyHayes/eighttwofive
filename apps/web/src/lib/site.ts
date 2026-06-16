/**
 * The canonical public origin for the storefront. Used for canonical links,
 * Open Graph URLs, JSON-LD, and sitemap entries regardless of where the SPA
 * actually runs (preview deploys, localhost). Defined once so every layer —
 * the SPA, the SEO helpers, and the edge middleware — agrees on one domain.
 */
export const SITE_ORIGIN = "https://eighttwofive.vercel.app";
