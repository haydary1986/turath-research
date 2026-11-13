"use client";

import { useTranslations } from "next-intl";
import { LogIn, LayoutDashboard } from "lucide-react";
import { Link } from "@/i18n/routing";
import { Button } from "@/components/ui/button";
import { useHydrated } from "@/hooks/use-hydrated";
import { useAuth } from "@/lib/auth";
import { cn } from "@/lib/utils";

interface PortalEntryButtonProps {
  size?: "default" | "lg";
  /** "onDark" renders a white button for the navy hero. */
  tone?: "default" | "onDark";
  className?: string;
}

/** Sign in (Google Workspace on /auth/login) for visitors, "enter the portal" for signed-in users. */
export function PortalEntryButton({ size = "default", tone = "default", className }: PortalEntryButtonProps) {
  const t = useTranslations("site.nav");
  const token = useAuth((state) => state.token);
  // The session lives in localStorage, so the server renders the visitor's button.
  const signedIn = useHydrated() && !!token;
  const Icon = signedIn ? LayoutDashboard : LogIn;

  return (
    <Button
      size={size}
      className={cn(
        size === "lg" && "h-11 px-5 text-base",
        tone === "onDark" && "bg-white text-primary hover:bg-white/90 [a]:hover:bg-white/90",
        className
      )}
      render={<Link href={signedIn ? "/portal" : "/auth/login"} />}
      title={signedIn ? undefined : t("signInHint")}
    >
      <Icon className="size-4" aria-hidden />
      {signedIn ? t("enterPortal") : t("signIn")}
    </Button>
  );
}
