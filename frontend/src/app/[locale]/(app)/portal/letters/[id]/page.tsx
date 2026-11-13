"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ArrowRight, Printer } from "lucide-react";
import { Link } from "@/i18n/routing";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { LetterView } from "@/components/portal/letter-view";
import { RequireAuth } from "@/components/portal/require-auth";
import type { IssuedLetter } from "@/lib/portal/types";

const DIWAN_EDITOR_ROLES = ["scientific_affairs", "admin"];

interface DiwanNoEditorProps {
  letter: IssuedLetter;
  onSaved: (letter: IssuedLetter) => void;
}

/** The registry may assign the number after printing, so it stays editable; nothing else does. */
function DiwanNoEditor({ letter, onSaved }: DiwanNoEditorProps) {
  const t = useTranslations("portal");
  const toMessage = usePortalErrors();
  const [value, setValue] = useState(letter.diwan_no ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save() {
    setError("");
    setBusy(true);
    try {
      const res = await api.put<IssuedLetter>(`/api/v1/portal/letters/${letter.id}/diwan-no`, {
        diwan_no: value,
      });
      if (res.data) onSaved(res.data);
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-[210mm] space-y-2 rounded-lg border p-3 print:hidden">
      {error && <Alert variant="danger">{error}</Alert>}
      <Label htmlFor="letter-diwan-no">{t("ui.diwanNoLabel")}</Label>
      <div className="flex flex-wrap items-center gap-2">
        <Input
          id="letter-diwan-no"
          dir="ltr"
          className="w-56"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={t("ui.diwanNoPlaceholder")}
        />
        <Button
          type="button"
          variant="outline"
          disabled={busy || value.trim() === (letter.diwan_no ?? "")}
          onClick={save}
        >
          {busy ? t("ui.saving") : t("ui.saveChanges")}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">{t("ui.diwanNoHint")}</p>
    </div>
  );
}

function LetterPage() {
  const params = useParams<{ id: string }>();
  const t = useTranslations("portal");
  const locale = useLocale();
  const role = useAuth((auth) => auth.user?.role);
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
      <div className="flex items-center justify-between print:hidden">
        <Button variant="outline" render={<Link href={`/portal/requests/${letter.request_id}`} />}>
          <ArrowRight className="size-4 ltr:-scale-x-100" />
          <span>{t("ui.backToRequest")}</span>
        </Button>
        <Button onClick={() => window.print()}>
          <Printer className="size-4" />
          <span>{t("ui.print")}</span>
        </Button>
      </div>
      {role && DIWAN_EDITOR_ROLES.includes(role) && <DiwanNoEditor letter={letter} onSaved={setLetter} />}
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
