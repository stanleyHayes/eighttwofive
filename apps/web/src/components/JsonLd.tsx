import { useEffect } from "react";

/**
 * Injects a JSON-LD structured-data script into <head> while mounted. Search
 * engines that render the SPA read it for rich results (Product, Breadcrumb,
 * …). Re-injects only when the serialized data changes.
 */
export function JsonLd({ data }: { data: unknown }) {
  const json = JSON.stringify(data);

  useEffect(() => {
    const script = document.createElement("script");
    script.type = "application/ld+json";
    script.text = json;
    document.head.appendChild(script);

    return () => {
      script.remove();
    };
  }, [json]);

  return null;
}
