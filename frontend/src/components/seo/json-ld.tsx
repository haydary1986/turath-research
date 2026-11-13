import { serializeJsonLd, type JsonLdObject } from "@/lib/seo/json-ld";

/** Structured data for search engines, rendered server-side into the page HTML. */
export function JsonLd({ data }: { data: JsonLdObject | JsonLdObject[] }) {
  return <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: serializeJsonLd(data) }} />;
}
