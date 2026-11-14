import { notFound } from "next/navigation";

/**
 * Any path under a locale that matches no page: hands over to [locale]/not-found.tsx
 * so the 404 is rendered inside the locale layout (right language, lang and dir)
 * instead of the bare default page.
 */
export default function UnknownPage() {
  notFound();
}
