import { dirOf, pickName } from "@/lib/i18n/text-lang";

interface DataOptionProps {
  value: string;
  ar: string | null | undefined;
  en?: string | null;
  locale: string;
}

/**
 * A <select> option whose label is database text (a college or department name):
 * the name in the page language when the record has it, laid out in its own direction.
 */
export function DataOption({ value, ar, en, locale }: DataOptionProps) {
  const name = pickName(ar, en, locale);
  return (
    <option value={value} lang={name?.lang} dir={name ? dirOf(name.lang) : undefined} data-bidi="">
      {name?.text ?? value}
    </option>
  );
}
