/** Number locale of the public site: Arabic text with Western digits, matching the portal's dates. */
export function numberLocale(locale: string): string {
  return locale === "ar" ? "ar-IQ-u-nu-latn" : "en-GB";
}

export function formatNumber(value: number, locale: string, grouping = true): string {
  return new Intl.NumberFormat(numberLocale(locale), { useGrouping: grouping }).format(value);
}
