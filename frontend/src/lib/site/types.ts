/** Public site: landing-page content and announcements (activities and circulars). */

export const ANNOUNCEMENT_TYPES = ["circular", "activity"] as const;
export type AnnouncementType = (typeof ANNOUNCEMENT_TYPES)[number];

export interface SiteService {
  title: string;
  description: string;
}

export interface SiteContact {
  email: string;
  phone: string;
  location: string;
  hours: string;
}

export interface SiteLink {
  label: string;
  url: string;
}

export interface SiteContent {
  hero_title: string;
  hero_subtitle: string;
  about: string;
  services: SiteService[];
  contact: SiteContact;
  links: SiteLink[];
  updated_at?: string;
}

export interface AnnouncementAttachment {
  id: string;
  name: string;
  mime_type: string;
  size_bytes: number;
}

export interface PublicAnnouncement {
  id: string;
  type: AnnouncementType;
  title: string;
  body: string;
  event_date: string | null;
  reference_no: string | null;
  attachment: AnnouncementAttachment | null;
  pinned: boolean;
  published_at: string | null;
}

export interface StaffAnnouncement extends PublicAnnouncement {
  published: boolean;
  activity_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface AnnouncementInput {
  type: AnnouncementType;
  title: string;
  body: string;
  event_date: string;
  reference_no: string;
}

/** Mirrors backend/internal/service/publicsite limits. */
export const SITE_LIMITS = {
  heroTitle: 200,
  heroSubtitle: 500,
  about: 5000,
  services: 12,
  serviceTitle: 150,
  serviceDescription: 600,
  links: 8,
  linkLabel: 80,
  linkUrl: 500,
  email: 254,
  phone: 50,
  location: 300,
  hours: 200,
  title: 300,
  body: 20000,
  referenceNo: 100,
} as const;

export const PUBLIC_API = {
  site: "/api/v1/public/site",
  announcements: "/api/v1/public/announcements",
  announcement: (id: string): string => `/api/v1/public/announcements/${encodeURIComponent(id)}`,
  file: (id: string): string => `/api/v1/public/files/${encodeURIComponent(id)}`,
} as const;

export function isAnnouncementType(value: string | null | undefined): value is AnnouncementType {
  return !!value && (ANNOUNCEMENT_TYPES as readonly string[]).includes(value);
}

/** Roles that edit the public site and publish announcements. */
export function canManageSite(role?: string | null): boolean {
  return role === "admin" || role === "scientific_affairs";
}

export function formatFileSize(bytes: number, locale: string): string {
  const kb = bytes / 1024;
  const fmt = new Intl.NumberFormat(locale === "ar" ? "ar-IQ" : "en-GB", { maximumFractionDigits: 1 });
  return kb < 1024 ? `${fmt.format(kb)} KB` : `${fmt.format(kb / 1024)} MB`;
}
