import type { MetadataRoute } from "next";
import { BRAND_NAVY, BRAND_PAGE } from "@/lib/site/brand";

export default function manifest(): MetadataRoute.Manifest {
  return {
    id: "/ar",
    name: "بوابة الشؤون العلمية — جامعة التراث",
    short_name: "الشؤون العلمية",
    description:
      "البوابة الإلكترونية لقسم الشؤون العلمية في جامعة التراث: الاعمامات والنشاطات العلمية وخدمات التدريسيين والباحثين.",
    lang: "ar",
    dir: "rtl",
    start_url: "/ar",
    scope: "/",
    display: "standalone",
    orientation: "any",
    background_color: BRAND_PAGE,
    theme_color: BRAND_NAVY,
    categories: ["education"],
    icons: [
      { src: "/icons/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any" },
      { src: "/icons/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any" },
      { src: "/icons/maskable-192.png", sizes: "192x192", type: "image/png", purpose: "maskable" },
      { src: "/icons/maskable-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
    ],
  };
}
