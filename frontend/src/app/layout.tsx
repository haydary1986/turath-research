import type { ReactNode } from "react";

// Root layout — minimal, no <html> tag (locale layout handles it)
export default function RootLayout({ children }: { children: ReactNode }) {
  return children;
}
