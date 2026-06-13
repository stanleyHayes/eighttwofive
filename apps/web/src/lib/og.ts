// Shared helpers for Open Graph meta tags. Used by the Vercel Edge Middleware
// (apps/web/middleware.ts) and by unit tests. Must stay runtime-agnostic.

export const OG_TRANSFORM = "c_fill,w_1200,h_630";

export interface OgData {
  title: string;
  description: string;
  image?: string;
  url: string;
  type?: string;
}

const CRAWLER_AGENTS = [
  "facebookexternalhit",
  "whatsapp",
  "twitterbot",
  "linkedinbot",
  "slackbot",
  "discordbot",
  "googlebot",
  "bingbot",
  "instagram",
];

/** Identify social-media / messaging / search crawlers (kept broad for callers/tests). */
export function isCrawler(userAgent: string): boolean {
  const ua = userAgent.toLowerCase();
  return CRAWLER_AGENTS.some((bot) => ua.includes(bot));
}

// Non-JS social/messaging crawlers that need the server-rendered OG shell.
// Search engines (Googlebot, Bingbot) render JavaScript, so they are
// deliberately excluded — they should receive the real SPA and index each
// route's own content rather than the shell's redirect to "/".
const SOCIAL_CRAWLER_AGENTS = [
  "facebookexternalhit",
  "whatsapp",
  "twitterbot",
  "linkedinbot",
  "slackbot",
  "discordbot",
  "instagram",
  "pinterest",
  "telegrambot",
];

/** True for non-JS social crawlers that need the pre-rendered OG meta shell. */
export function isSocialCrawler(userAgent: string): boolean {
  const ua = userAgent.toLowerCase();
  return SOCIAL_CRAWLER_AGENTS.some((bot) => ua.includes(bot));
}

/** Minimal HTML escaping for values injected into meta tag attributes. */
export function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/** Keep OG descriptions within the range most platforms render cleanly. */
export function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.slice(0, max - 1).trimEnd() + "…";
}

/** Build a Cloudinary delivery URL for a public ID and transformation string. */
export function photoUrl(cloudName: string, publicId: string, transform: string): string {
  return `https://res.cloudinary.com/${cloudName}/image/upload/${transform}/${publicId}`;
}

/** Sort photos by their explicit display order. */
export function sortedPhotos<T extends { order: number }>(photos: T[]): T[] {
  return [...photos].sort((a, b) => a.order - b.order);
}

/** Pick the first ordered photo from the first design that has photos. */
export function firstDesignPhotoPublicId(
  designs: Array<{ photos: Array<{ publicId: string; order: number }> }>,
): string | undefined {
  for (const design of designs) {
    const photos = sortedPhotos(design.photos);
    if (photos[0]?.publicId) return photos[0].publicId;
  }
  return undefined;
}

/**
 * Build a tiny HTML shell with OG/Twitter meta tags and a client-side redirect
 * to the SPA root. Social crawlers read the meta tags; real browsers execute
 * the redirect.
 */
export function buildOgHtml(data: OgData): string {
  const title = escapeHtml(data.title);
  const description = escapeHtml(truncate(data.description, 200));
  const url = escapeHtml(data.url);
  const type = data.type ? escapeHtml(data.type) : "website";
  const imageMeta = data.image
    ? `    <meta property="og:image" content="${escapeHtml(data.image)}" />
    <meta property="og:image:width" content="1200" />
    <meta property="og:image:height" content="630" />
    <meta name="twitter:card" content="summary_large_image" />
    <meta name="twitter:image" content="${escapeHtml(data.image)}" />`
    : '    <meta name="twitter:card" content="summary" />';

  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>${title}</title>
    <meta name="description" content="${description}" />
    <meta property="og:title" content="${title}" />
    <meta property="og:description" content="${description}" />
    <meta property="og:type" content="${type}" />
    <meta property="og:url" content="${url}" />
${imageMeta}
    <meta name="twitter:title" content="${title}" />
    <meta name="twitter:description" content="${description}" />
    <script>
      window.location.replace("/");
    </script>
  </head>
  <body>
    <noscript>
      <a href="/">Eight Two Five</a>
    </noscript>
  </body>
</html>`;
}
