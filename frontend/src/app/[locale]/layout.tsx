import type { ReactNode } from "react";
import { NextIntlClientProvider } from "next-intl";
import { getMessages } from "next-intl/server";
import { notFound } from "next/navigation";
import { Cairo, Inter } from "next/font/google";
import { routing } from "@/i18n/routing";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/layout/app-sidebar";
import { AppHeader } from "@/components/layout/app-header";
import "../globals.css";

const cairo = Cairo({
  subsets: ["arabic", "latin"],
  variable: "--font-cairo",
  display: "swap",
});

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

type Props = {
  children: ReactNode;
  params: Promise<{ locale: string }>;
};

export default async function LocaleLayout({ children, params }: Props) {
  const { locale } = await params;

  if (!routing.locales.includes(locale as "ar" | "en")) {
    notFound();
  }

  const messages = await getMessages();
  const dir = locale === "ar" ? "rtl" : "ltr";
  const fontClass = locale === "ar" ? cairo.variable : inter.variable;

  return (
    <html lang={locale} dir={dir} suppressHydrationWarning>
      <body
        className={`${cairo.variable} ${inter.variable} ${fontClass} min-h-screen bg-background antialiased`}
        style={{ fontFamily: locale === "ar" ? "var(--font-cairo), sans-serif" : "var(--font-inter), sans-serif" }}
      >
        <NextIntlClientProvider messages={messages}>
          <TooltipProvider>
            <SidebarProvider>
              <div className="flex min-h-screen w-full">
                <AppSidebar />
                <div className="flex flex-1 flex-col overflow-hidden">
                  <AppHeader />
                  <main className="flex-1 overflow-auto p-4 md:p-6">
                    {children}
                  </main>
                </div>
              </div>
            </SidebarProvider>
          </TooltipProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
