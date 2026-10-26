"use client";

import { useEffect, useRef, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth, type AuthUser } from "@/lib/auth";

interface GoogleCredentialResponse {
  credential?: string;
}

interface GoogleIdentity {
  accounts: {
    id: {
      initialize: (config: {
        client_id: string;
        callback: (response: GoogleCredentialResponse) => void;
        hd?: string;
        auto_select?: boolean;
        use_fedcm_for_prompt?: boolean;
      }) => void;
      renderButton: (
        parent: HTMLElement,
        options: Record<string, string | number>,
      ) => void;
    };
  };
}

declare global {
  interface Window {
    google?: GoogleIdentity;
  }
}

const GSI_SRC = "https://accounts.google.com/gsi/client";

interface AuthConfig {
  google_client_id: string;
  allowed_domain: string;
}

function loadGoogleScript(): Promise<void> {
  if (window.google) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      `script[src="${GSI_SRC}"]`,
    );
    const script = existing ?? document.createElement("script");
    script.addEventListener("load", () => resolve(), { once: true });
    script.addEventListener(
      "error",
      () => reject(new Error("google script failed")),
      { once: true },
    );
    if (!existing) {
      script.src = GSI_SRC;
      script.async = true;
      document.head.appendChild(script);
    }
  });
}

interface LoginResponse {
  token: string;
  user: AuthUser;
}

/** Google Sign-In button limited to the university domain. Renders nothing until the server reports a client id. */
export function GoogleSignIn({
  onError,
}: {
  onError: (message: string) => void;
}) {
  const t = useTranslations("auth");
  const locale = useLocale();
  const router = useRouter();
  const setAuth = useAuth((state) => state.setAuth);
  const buttonRef = useRef<HTMLDivElement>(null);
  const [config, setConfig] = useState<AuthConfig | null>(null);

  useEffect(() => {
    let active = true;
    api
      .get<AuthConfig>("/api/v1/auth/config")
      .then((res) => active && setConfig(res.data ?? null))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!config?.google_client_id) return;
    let active = true;

    async function handleCredential(response: GoogleCredentialResponse) {
      if (!response.credential) return;
      try {
        const res = await api.post<LoginResponse>("/api/v1/auth/google", {
          credential: response.credential,
        });
        if (res.data) {
          setAuth(res.data.token, res.data.user);
          router.push("/portal");
        }
      } catch (err) {
        const code = err instanceof Error ? err.message : "";
        onError(
          t.has(`errors.${code}`) ? t(`errors.${code}`) : t("errors.generic"),
        );
      }
    }

    loadGoogleScript()
      .then(() => {
        if (!active || !window.google || !buttonRef.current || !config) return;
        window.google.accounts.id.initialize({
          client_id: config.google_client_id,
          callback: handleCredential,
          hd: config.allowed_domain,
          auto_select: false,
        });
        window.google.accounts.id.renderButton(buttonRef.current, {
          theme: "outline",
          size: "large",
          shape: "rectangular",
          text: "signin_with",
          width: 320,
          locale,
        });
      })
      .catch(() => active && onError(t("errors.generic")));

    return () => {
      active = false;
    };
  }, [config, locale, onError, router, setAuth, t]);

  if (!config) return null;

  if (!config.google_client_id) {
    return (
      <p className="text-center text-sm text-muted-foreground">
        {t("errors.google_not_configured")}
      </p>
    );
  }

  return (
    <div className="space-y-3">
      <p className="text-center text-sm font-medium">{t("signInWithGoogle")}</p>
      <div ref={buttonRef} className="flex justify-center" />
      <p className="text-center text-xs text-muted-foreground" dir="ltr">
        {t("universityOnly")} (@{config.allowed_domain})
      </p>
    </div>
  );
}
