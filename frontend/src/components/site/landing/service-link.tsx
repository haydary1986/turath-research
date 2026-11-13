"use client";

import type { ReactNode } from "react";
import { Link } from "@/i18n/routing";
import { useHydrated } from "@/hooks/use-hydrated";
import { useAuth } from "@/lib/auth";
import { requiresSignIn } from "@/lib/site/types";

interface ServiceLinkProps {
  href: string;
  className?: string;
  children: ReactNode;
}

/**
 * Links a service card to its page. Portal pages need a session, so signed-out
 * visitors are sent to the sign-in page instead (it has no return-path support).
 */
export function ServiceLink({ href, className, children }: ServiceLinkProps) {
  const token = useAuth((state) => state.token);
  const signedIn = useHydrated() && !!token;
  const target = requiresSignIn(href) && !signedIn ? "/auth/login" : href;
  return (
    <Link href={target} className={className}>
      {children}
    </Link>
  );
}
