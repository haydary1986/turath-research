This is a [Next.js](https://nextjs.org) project bootstrapped with [`create-next-app`](https://nextjs.org/docs/app/api-reference/cli/create-next-app).

## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

This project uses [`next/font`](https://nextjs.org/docs/app/building-your-application/optimizing/fonts) to automatically optimize and load [Geist](https://vercel.com/font), a new font family for Vercel.

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.

## Languages and the i18n browser check

The site is Arabic-first (`/ar`, RTL) with an English locale (`/en`, LTR). UI strings
live in `src/messages/{ar,en}.json` and `src/messages/public/{ar,en}.json`;
`npm test` fails if a key exists in one locale but not the other.

Text that comes from people or the database (names, titles, notes, journal names...)
is rendered with `<Bidi>` / `<Ltr>` from `src/components/i18n/bidi.tsx`, which set
`lang`/`dir` (bidi isolation) and mark the element with `data-bidi`. Bilingual fields of
the public site document use `<Localized>` (English with an Arabic fallback).

`npm run test:i18n` (`e2e/i18n-check.mjs`) opens every public and portal page in both
locales with Playwright and the system Chrome, and fails on crashes, console errors,
`MISSING_MESSAGE`, untranslated keys, wrong `<html lang/dir>`, Arabic letters in English
UI text, or Latin words (outside a short allowlist) in Arabic UI text. Elements marked
with `data-bidi` or a `lang` attribute are treated as data and skipped.

```bash
I18N_BASE_URL=http://localhost:3001 \
I18N_ADMIN_EMAIL=admin@example.iq I18N_ADMIN_PASSWORD='...' \
npm run test:i18n
```

Optional: `CHROME_PATH` (default: macOS Google Chrome), `I18N_ONLY=/portal` to check a
subset, `I18N_HEADED=1` to watch the browser.
