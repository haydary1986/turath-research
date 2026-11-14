/** Public site: landing-page content and announcements (activities and circulars). */

export const ANNOUNCEMENT_TYPES = ["circular", "activity"] as const;
export type AnnouncementType = (typeof ANNOUNCEMENT_TYPES)[number];

export const SERVICE_ICONS = ["research", "verify", "award", "calendar", "search", "shield", "file", "users", "book", "chart"] as const;
export type ServiceIcon = (typeof SERVICE_ICONS)[number];

/**
 * Every visible text of the landing-page document has an optional English
 * counterpart (the *_en fields). Arabic is the primary language: pages fall back to
 * it when the English value is empty (see lib/i18n/text-lang.ts pickLocalized).
 */
export interface SiteService {
  title: string;
  title_en: string;
  description: string;
  description_en: string;
  /** A key of SERVICE_ICONS, or "" for the default icon. */
  icon: string;
  /** A site-relative path under /portal, /journals or /news, or "". */
  link: string;
}

export const STAT_KEYS = ["academics", "publications", "indexed", "colleges", "activities"] as const;
export type StatKey = (typeof STAT_KEYS)[number];

/** How one live statistic is shown; a non-empty value replaces the computed number. */
export interface StatSetting {
  key: StatKey;
  label: string;
  label_en: string;
  value: string;
  hidden: boolean;
}

export interface SiteDirector {
  title: string;
  title_en: string;
  name: string;
  name_en: string;
  position: string;
  position_en: string;
  message: string;
  message_en: string;
  /** Managed by the photo endpoints; "" when there is no photo. */
  photo_id: string;
}

export const SOCIAL_PLATFORMS = ["facebook", "instagram", "x", "youtube", "linkedin", "telegram", "tiktok", "whatsapp", "website"] as const;
export type SocialPlatform = (typeof SOCIAL_PLATFORMS)[number];

export interface SiteSocial {
  platform: SocialPlatform;
  url: string;
}

export interface SiteFAQItem {
  question: string;
  question_en: string;
  answer: string;
  answer_en: string;
}

export interface SiteContact {
  email: string;
  phone: string;
  location: string;
  location_en: string;
  hours: string;
  hours_en: string;
}

export interface SiteLink {
  label: string;
  label_en: string;
  url: string;
}

export interface SiteContent {
  hero_title: string;
  hero_title_en: string;
  hero_subtitle: string;
  hero_subtitle_en: string;
  about: string;
  about_en: string;
  services: SiteService[];
  contact: SiteContact;
  links: SiteLink[];
  stats: StatSetting[];
  director: SiteDirector;
  colleges: SiteLink[];
  socials: SiteSocial[];
  faq: SiteFAQItem[];
  hide_publications: boolean;
  updated_at?: string;
}

/** A statistic as visitors see it: number is null when staff set a manual value. */
export interface PublicStat {
  key: StatKey;
  label: string;
  label_en: string;
  value: string;
  number: number | null;
}

/** The editor's view of a statistic: its setting plus the computed number. */
export interface StaffStat extends StatSetting {
  computed: number;
}

export interface LocalizedName {
  ar: string;
  en: string | null;
}

export interface PublicPublication {
  id: string;
  title_ar: string | null;
  title_en: string | null;
  journal: string | null;
  year: number | null;
  doi: string | null;
  author: LocalizedName;
  college: LocalizedName | null;
}

export interface AnnouncementAttachment {
  id: string;
  name: string;
  mime_type: string;
  size_bytes: number;
}

export const ANNOUNCEMENT_LANGS = ["ar", "en"] as const;
export type AnnouncementLang = (typeof ANNOUNCEMENT_LANGS)[number];

export interface PublicAnnouncement {
  id: string;
  type: AnnouncementType;
  /** The language the announcement is written in (never machine-translated). */
  lang: AnnouncementLang;
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
  lang: AnnouncementLang;
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
  statLabel: 80,
  statValue: 20,
  directorTitle: 150,
  directorName: 150,
  directorPosition: 200,
  directorMessage: 5000,
  colleges: 30,
  socials: 10,
  faq: 20,
  faqQuestion: 300,
  faqAnswer: 3000,
  serviceLink: 200,
} as const;

export const PUBLIC_API = {
  site: "/api/v1/public/site",
  announcements: "/api/v1/public/announcements",
  announcement: (id: string): string => `/api/v1/public/announcements/${encodeURIComponent(id)}`,
  file: (id: string): string => `/api/v1/public/files/${encodeURIComponent(id)}`,
  directorPhoto: (id: string): string => `/api/v1/public/site/director-photo/${encodeURIComponent(id)}`,
  stats: "/api/v1/public/stats",
  latestPublications: "/api/v1/public/publications/latest",
} as const;

/** Staff endpoints of the site editor. */
export const SITE_ADMIN_API = {
  content: "/api/v1/site",
  stats: "/api/v1/site/stats",
  directorPhoto: "/api/v1/site/director-photo",
} as const;

const SERVICE_LINK_ROOTS = ["/portal", "/journals", "/news"];

/** Mirrors publicsite.IsSafeServiceLink: a site path under the portal, the journal pages or the news page. */
export function isSafeServiceLink(link: string): boolean {
  if (link.length > SITE_LIMITS.serviceLink || !/^\/[a-z0-9_\-/]*$/.test(link) || link.includes("//") || link.includes("..")) {
    return false;
  }
  return SERVICE_LINK_ROOTS.some((root) => link === root || link.startsWith(`${root}/`));
}

/** Portal pages need a session; visitors are sent to sign in instead. */
export function requiresSignIn(link: string): boolean {
  return link === "/portal" || link.startsWith("/portal/");
}

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
