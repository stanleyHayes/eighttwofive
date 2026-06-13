import { next } from "@vercel/edge";
import {
  OG_TRANSFORM,
  buildOgHtml,
  firstDesignPhotoPublicId,
  isSocialCrawler,
  photoUrl,
  sortedPhotos,
  type OgData,
} from "./src/lib/og";

/**
 * Run this middleware on the dynamic sitemap and on shareable storefront paths.
 * Other routes continue straight to the Vite SPA via vercel.json rewrites.
 */
export const config = {
  matcher: [
    "/sitemap.xml",
    "/collections/:slug",
    "/collections/:slug/",
    "/designs/:slug",
    "/designs/:slug/",
  ],
};

const DEFAULT_TITLE = "Eight Two Five";
const DEFAULT_DESCRIPTION = "Made-to-measure womenswear from Accra.";
/** Canonical public origin used in sitemap URLs. */
const SITE_ORIGIN = "https://eighttwofive.vercel.app";
const STATIC_PATHS = ["/", "/store", "/about", "/contact", "/slots"];

interface SettingsEnvelope {
  data: { cloudName: string };
}

interface CollectionEnvelope {
  data: {
    collection: { name: string; note: string };
    designs: Array<{ photos: Array<{ publicId: string; order: number }> }>;
  };
}

interface DesignEnvelope {
  data: {
    name: string;
    note: string;
    photos: Array<{ publicId: string; order: number }>;
  };
}

/** API base URL. Prefers VITE_API_URL; falls back to the request origin. */
function apiBase(requestUrl: URL): string {
  const env = process.env.VITE_API_URL?.trim() ?? "";
  if (env) return env.replace(/\/+$/, "");
  return requestUrl.origin;
}

function makeOgResponse(data: OgData): Response {
  return new Response(buildOgHtml(data), {
    headers: { "Content-Type": "text/html; charset=utf-8" },
  });
}

function fallbackResponse(requestUrl: string): Response {
  return makeOgResponse({
    title: DEFAULT_TITLE,
    description: DEFAULT_DESCRIPTION,
    url: requestUrl,
  });
}

function imageUrl(cloudName: string, publicId?: string): string | undefined {
  if (!cloudName || !publicId) return undefined;
  return photoUrl(cloudName, publicId, OG_TRANSFORM);
}

/** Build an XML sitemap of the static routes plus every live collection and design. */
async function sitemapResponse(requestUrl: URL): Promise<Response> {
  const base = apiBase(requestUrl);
  const urls = new Set(STATIC_PATHS.map((path) => SITE_ORIGIN + path));

  try {
    const [collectionsRes, designsRes] = await Promise.all([
      fetch(`${base}/api/v1/collections`),
      fetch(`${base}/api/v1/designs`),
    ]);

    if (collectionsRes.ok) {
      const body = (await collectionsRes.json()) as { data?: Array<{ slug: string }> };
      for (const item of body.data ?? []) urls.add(`${SITE_ORIGIN}/collections/${item.slug}`);
    }

    if (designsRes.ok) {
      const body = (await designsRes.json()) as { data?: Array<{ slug: string }> };
      for (const item of body.data ?? []) urls.add(`${SITE_ORIGIN}/designs/${item.slug}`);
    }
  } catch {
    // Network/API failure — fall back to the static routes already in the set.
  }

  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${[...urls].map((loc) => `  <url><loc>${loc}</loc></url>`).join("\n")}
</urlset>`;

  return new Response(xml, {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Cache-Control": "public, max-age=3600",
    },
  });
}

export default async function middleware(request: Request): Promise<Response> {
  const url = new URL(request.url);

  if (url.pathname === "/sitemap.xml") {
    return sitemapResponse(url);
  }

  const userAgent = request.headers.get("user-agent") ?? "";
  if (!isSocialCrawler(userAgent)) {
    return next();
  }

  const match = /^\/(collections|designs)\/([^/]+)\/?$/.exec(url.pathname);
  if (!match) {
    return next();
  }

  const [, type, slug] = match;
  const base = apiBase(url);

  try {
    const [settingsRes, resourceRes] = await Promise.all([
      fetch(`${base}/api/v1/settings`),
      fetch(`${base}/api/v1/${type}/${encodeURIComponent(slug)}`),
    ]);

    if (!settingsRes.ok || !resourceRes.ok) {
      return fallbackResponse(request.url);
    }

    const settings = (await settingsRes.json()) as SettingsEnvelope;
    const resource = (await resourceRes.json()) as CollectionEnvelope | DesignEnvelope;
    const cloudName = settings.data?.cloudName ?? "";

    let data: OgData;

    if (type === "collections") {
      const payload = resource as CollectionEnvelope;
      const { collection, designs } = payload.data;
      data = {
        title: `${collection.name} — ${DEFAULT_TITLE}`,
        description: collection.note || DEFAULT_DESCRIPTION,
        url: request.url,
        type: "product.group",
        image: imageUrl(cloudName, firstDesignPhotoPublicId(designs)),
      };
    } else {
      const payload = resource as DesignEnvelope;
      const design = payload.data;
      const photos = sortedPhotos(design.photos);
      data = {
        title: `${design.name} — ${DEFAULT_TITLE}`,
        description: design.note || DEFAULT_DESCRIPTION,
        url: request.url,
        type: "product",
        image: imageUrl(cloudName, photos[0]?.publicId),
      };
    }

    return makeOgResponse(data);
  } catch {
    return fallbackResponse(request.url);
  }
}
