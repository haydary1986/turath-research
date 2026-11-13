import { getTranslations } from "next-intl/server";
import { ArrowLeft, ArrowRight } from "lucide-react";
import { Link } from "@/i18n/routing";
import type { AnnouncementType, PublicAnnouncement } from "@/lib/site/types";
import { cn } from "@/lib/utils";
import { AnnouncementCard } from "../announcement-card";
import { Section } from "./section";

interface LatestAnnouncementsProps {
  locale: string;
  circulars: PublicAnnouncement[];
  activities: PublicAnnouncement[];
}

/** Latest circulars and activities; each column renders only when it has published items. */
export async function LatestAnnouncements({ locale, circulars, activities }: LatestAnnouncementsProps) {
  const columns = [
    { type: "circular" as AnnouncementType, items: circulars, titleKey: "latestCirculars" },
    { type: "activity" as AnnouncementType, items: activities, titleKey: "latestActivities" },
  ].filter((column) => column.items.length > 0);
  if (columns.length === 0) return null;

  const t = await getTranslations({ locale, namespace: "site.home" });
  const Forward = locale === "ar" ? ArrowLeft : ArrowRight;

  return (
    <div className={cn("mx-auto grid max-w-6xl", columns.length === 2 && "lg:grid-cols-2 lg:gap-8")}>
      {columns.map(({ type, items, titleKey }) => (
        <Section
          key={type}
          id={type === "circular" ? "circulars" : "activities"}
          title={t(titleKey)}
          className="w-full"
          action={
            <Link
              href={`/news?type=${type}`}
              className="inline-flex items-center gap-1 text-sm font-semibold text-primary hover:text-accent"
            >
              {t("viewAll")}
              <Forward className="size-4" aria-hidden />
            </Link>
          }
        >
          <div className={cn("grid gap-3", columns.length === 1 && "md:grid-cols-2")}>
            {items.map((item) => (
              <AnnouncementCard key={item.id} item={item} compact />
            ))}
          </div>
        </Section>
      ))}
    </div>
  );
}
