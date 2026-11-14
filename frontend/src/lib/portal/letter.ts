export const LETTER_PLACEHOLDERS = [
  "researcher_name",
  "college",
  "department",
  "research_title",
  "journal_name",
  "index_db",
  "quartile",
  "publication_date",
] as const;

export type LetterPlaceholder = (typeof LETTER_PLACEHOLDERS)[number];

const PLACEHOLDER_PATTERN = /\{\{\s*([a-z_]+)\s*\}\}/g;

/** Mirrors the backend renderer: one pass, unknown placeholders kept, values never re-expanded. */
export function renderBody(template: string, vars: Record<string, string>): string {
  return template.replace(PLACEHOLDER_PATTERN, (match, name: string) =>
    Object.prototype.hasOwnProperty.call(vars, name) ? vars[name] : match
  );
}

export const SAMPLE_LETTER_VARS: Record<LetterPlaceholder, string> = {
  researcher_name: "د. أحمد محمد علي",
  college: "كلية الهندسة",
  department: "هندسة الحاسوب",
  research_title: "AI-based Optimization for Smart Energy Systems",
  journal_name: "Energy and Buildings",
  index_db: "Scopus",
  quartile: "Q1",
  publication_date: "15/03/2024",
};

/**
 * Labels printed on the appreciation letter. The letter is an official Arabic
 * document laid out right-to-left on the university letterhead, so it reads the same
 * whatever language the portal is shown in.
 */
export const LETTER_LABELS_AR = {
  number: "العدد",
  date: "التاريخ",
  to: "الى",
  college: "الكلية",
  department: "القسم",
  verifyAt: "للتحقق من صحة الكتاب",
  internalNo: "رقم التحقق الداخلي",
} as const;
