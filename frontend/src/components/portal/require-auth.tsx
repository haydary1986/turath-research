"use client";

import type { ReactNode } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/routing";
import { useHydrated } from "@/hooks/use-hydrated";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

/** Renders children only for signed-in users; waits for hydration so server and client markup match. */
export function RequireAuth({ children }: { children: ReactNode }) {
  const t = useTranslations("portal.ui");
  const token = useAuth((state) => state.token);
  const hydrated = useHydrated();

  if (!hydrated) return null;
  if (!token) {
    return (
      <Card className="mx-auto max-w-md">
        <CardContent className="space-y-4 py-8 text-center">
          <p>{t("signInRequired")}</p>
          <Button render={<Link href="/auth/login" />}>{t("signIn")}</Button>
        </CardContent>
      </Card>
    );
  }
  return <>{children}</>;
}
