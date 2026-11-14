import type { MetadataRoute } from "next";
import { SITE_URL } from "@/lib/seo/site";

// Read SITE_URL at request time, not when the image is built.
export const dynamic = "force-dynamic";

/** The public site and journal pages are crawlable; the portal, sign-in and API are not. */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        // The signed-in app and letter-verification pages carry personal data and stay out of search.
        disallow: [
          "/*/portal", "/*/auth", "/api/", "/*/dashboard", "/*/researchers", "/*/publications",
          "/*/insights", "/*/search", "/*/verify",
        ],
      },
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
    host: SITE_URL,
  };
}
