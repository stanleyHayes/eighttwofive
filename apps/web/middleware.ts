import { next } from "@vercel/edge";
import {
  OG_TRANSFORM,
  buildOgHtml,
  firstDesignPhotoPublicId,
  isCrawler,
  photoUrl,
  sortedPhotos,
  type OgData,
} from "./src/lib/og";

/**
 * Run this middleware only on shareable storefront paths. Other routes continue
 * straight to the Vite SPA via vercel.json rewrites.
 */
export const config = {
  matcher: [
    "/collections/:slug",
    "/collections/:slug/",
    "/designs/:slug",
    "/designs/:slug/",
  ],
};

const DEFAULT_TITLE = "Eight Two Five";
const DEFAULT_DESCRIPTION = "Made-to-measure womenswear from Accra.";

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

export default async function middleware(request: Request): Promise<Response> {
  const userAgent = request.headers.get("user-agent") ?? "";
  if (!isCrawler(userAgent)) {
    return next();
  }

  const url = new URL(request.url);
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
