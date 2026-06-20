import { describe, expect, it } from "vitest";
import {
  OG_TRANSFORM,
  buildOgHtml,
  escapeHtml,
  firstDesignPhotoPublicId,
  isCrawler,
  isSocialCrawler,
  photoUrl,
  sortedPhotos,
} from "./og";

describe("isCrawler", () => {
  it.each([
    "facebookexternalhit/1.1",
    "WhatsApp/2.21.4.22 A",
    "Twitterbot/1.0",
    "LinkedInBot/1.0",
    "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)",
    "Discordbot/2.0",
    "Mozilla/5.0 (compatible; Googlebot/2.1)",
    "Mozilla/5.0 (compatible; bingbot/2.0)",
    "Instagram 219.0.0.12.117 Android",
  ])("detects %s", (ua) => {
    expect(isCrawler(ua)).toBe(true);
  });

  it("returns false for a normal browser user agent", () => {
    expect(
      isCrawler(
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
      ),
    ).toBe(false);
  });
});

describe("isSocialCrawler", () => {
  it.each([
    "facebookexternalhit/1.1",
    "WhatsApp/2.21.4.22 A",
    "Twitterbot/1.0",
    "LinkedInBot/1.0",
    "Discordbot/2.0",
    "Instagram 219.0.0.12.117 Android",
  ])("detects social crawler %s", (ua) => {
    expect(isSocialCrawler(ua)).toBe(true);
  });

  it.each([
    "Mozilla/5.0 (compatible; Googlebot/2.1)",
    "Mozilla/5.0 (compatible; bingbot/2.0)",
  ])("excludes JS-rendering search engine %s (they get the real SPA)", (ua) => {
    expect(isSocialCrawler(ua)).toBe(false);
  });

  it("returns false for a normal browser user agent", () => {
    expect(
      isSocialCrawler(
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
      ),
    ).toBe(false);
  });
});

describe("buildOgHtml", () => {
  it("renders title, description, url, image and a redirect script", () => {
    const html = buildOgHtml({
      title: "Velvet Set — Eight Two Five",
      description: "A limited-run collection of made-to-measure pieces.",
      url: "https://eightfivetwo.vercel.app/collections/velvet-set",
      image: "https://res.cloudinary.com/demo/image/upload/c_fill,w_1200,h_630/photo1",
      type: "product.group",
    });

    expect(html).toContain("<title>Velvet Set — Eight Two Five</title>");
    expect(html).toContain('property="og:title" content="Velvet Set — Eight Two Five"');
    expect(html).toContain(
      'property="og:description" content="A limited-run collection of made-to-measure pieces."',
    );
    expect(html).toContain(
      'property="og:url" content="https://eightfivetwo.vercel.app/collections/velvet-set"',
    );
    expect(html).toContain(
      'property="og:image" content="https://res.cloudinary.com/demo/image/upload/c_fill,w_1200,h_630/photo1"',
    );
    expect(html).toContain('property="og:image:width" content="1200"');
    expect(html).toContain('property="og:image:height" content="630"');
    expect(html).toContain('name="twitter:card" content="summary_large_image"');
    expect(html).toContain('window.location.replace("/")');
  });

  it("escapes HTML in meta values", () => {
    const html = buildOgHtml({
      title: 'A <script> alert("x") </script>',
      description: 'Quote "me" & friend',
      url: "https://example.com/?a=1&b=2",
    });

    expect(html).toContain(
      "<title>A &lt;script&gt; alert(&quot;x&quot;) &lt;/script&gt;</title>",
    );
    expect(html).toContain('content="Quote &quot;me&quot; &amp; friend"');
    expect(html).toContain('content="https://example.com/?a=1&amp;b=2"');
  });

  it("omits image tags when no image is provided", () => {
    const html = buildOgHtml({
      title: "Eight Two Five",
      description: "Made-to-measure womenswear from Accra.",
      url: "https://example.com/",
    });

    expect(html).not.toContain("og:image");
    expect(html).toContain('name="twitter:card" content="summary"');
  });
});

describe("photoUrl", () => {
  it("builds a Cloudinary delivery URL", () => {
    expect(photoUrl("demo", "my-folder/photo", OG_TRANSFORM)).toBe(
      "https://res.cloudinary.com/demo/image/upload/f_jpg,q_auto,c_fill,w_1200,h_630/my-folder/photo",
    );
  });
});

describe("sortedPhotos", () => {
  it("sorts photos by their order field", () => {
    const photos = [
      { publicId: "b", order: 2 },
      { publicId: "a", order: 1 },
      { publicId: "c", order: 3 },
    ];
    expect(sortedPhotos(photos).map((p) => p.publicId)).toEqual(["a", "b", "c"]);
  });
});

describe("firstDesignPhotoPublicId", () => {
  it("returns the first ordered photo across designs", () => {
    const designs = [
      { photos: [] },
      {
        photos: [
          { publicId: "second", order: 2 },
          { publicId: "first", order: 1 },
        ],
      },
    ];
    expect(firstDesignPhotoPublicId(designs)).toBe("first");
  });

  it("returns undefined when no design has photos", () => {
    expect(firstDesignPhotoPublicId([{ photos: [] }])).toBeUndefined();
    expect(firstDesignPhotoPublicId([])).toBeUndefined();
  });
});

describe("escapeHtml", () => {
  it("escapes the five HTML-special characters", () => {
    expect(escapeHtml(`"a" & 'b' <c>`)).toBe("&quot;a&quot; &amp; &#39;b&#39; &lt;c&gt;");
  });
});
