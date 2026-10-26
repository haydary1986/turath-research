"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Printer } from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { LetterView } from "@/components/portal/letter-view";
import { RequireAuth } from "@/components/portal/require-auth";
import type { IssuedLetter } from "@/lib/portal/types";

function LetterPage() {
  const params = useParams<{ id: string }>();
  const t = useTranslations("portal");
  const locale = useLocale();
  const [letter, setLetter] = useState<IssuedLetter | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "missing">("loading");

  useEffect(() => {
    let active = true;
    api
      .get<IssuedLetter>(`/api/v1/portal/letters/${params.id}`)
      .then((res) => {
        if (!active) return;
        setLetter(res.data ?? null);
        setState(res.data ? "ready" : "missing");
      })
      .catch(() => active && setState("missing"));
    return () => {
      active = false;
    };
  }, [params.id]);

  if (state === "loading") return <p className="text-center text-muted-foreground">{t("ui.loading")}</p>;
  if (state === "missing" || !letter) {
    return <p className="text-center text-muted-foreground">{t("ui.notFound")}</p>;
  }

  const verifyUrl = `${window.location.origin}/${locale}/verify/${letter.id}`;

  return (
    <div className="space-y-4">
      <div className="flex justify-end print:hidden">
        <Button onClick={() => window.print()}>
          <Printer className="size-4" />
          <span>{t("ui.print")}</span>
        </Button>
      </div>
      <LetterView letter={letter} verifyUrl={verifyUrl} />
    </div>
  );
}

export default function IssuedLetterPage() {
  return (
    <RequireAuth>
      <LetterPage />
    </RequireAuth>
  );
}
