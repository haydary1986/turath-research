import type { ReactNode } from "react";

/** Minimal chrome-free shell for anonymous pages (login, public letter verification). */
export default function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <main className="flex min-h-svh items-center justify-center p-4">
      <div className="w-full">{children}</div>
    </main>
  );
}
