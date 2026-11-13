import { REVALIDATE, backendGetOptional } from "./server-api";
import {
  PUBLIC_API, type AnnouncementType, type PublicAnnouncement, type PublicPublication, type PublicStat, type SiteContent,
} from "./types";

/** Server-side loaders for the public landing page and site shell. */

const LATEST_ANNOUNCEMENTS = 4;

/** The landing-page document, or null when the backend is unreachable. */
export async function loadSiteContent(): Promise<SiteContent | null> {
  return (await backendGetOptional<SiteContent>(PUBLIC_API.site, REVALIDATE.content))?.data ?? null;
}

export async function loadStats(): Promise<PublicStat[]> {
  return (await backendGetOptional<PublicStat[]>(PUBLIC_API.stats, REVALIDATE.content))?.data ?? [];
}

export async function loadLatestPublications(): Promise<PublicPublication[]> {
  return (await backendGetOptional<PublicPublication[]>(PUBLIC_API.latestPublications, REVALIDATE.content))?.data ?? [];
}

export async function loadLatestAnnouncements(type: AnnouncementType): Promise<PublicAnnouncement[]> {
  const path = `${PUBLIC_API.announcements}?type=${type}&limit=${LATEST_ANNOUNCEMENTS}`;
  return (await backendGetOptional<PublicAnnouncement[]>(path, REVALIDATE.content))?.data ?? [];
}

export interface LandingData {
  content: SiteContent | null;
  stats: PublicStat[];
  publications: PublicPublication[];
  circulars: PublicAnnouncement[];
  activities: PublicAnnouncement[];
}

export async function loadLanding(): Promise<LandingData> {
  const [content, stats, publications, circulars, activities] = await Promise.all([
    loadSiteContent(),
    loadStats(),
    loadLatestPublications(),
    loadLatestAnnouncements("circular"),
    loadLatestAnnouncements("activity"),
  ]);
  return { content, stats, publications, circulars, activities };
}
