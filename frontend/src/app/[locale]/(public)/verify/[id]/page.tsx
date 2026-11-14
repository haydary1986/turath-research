"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { CheckCircle2, XCircle } from "lucide-react";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Bidi, Ltr } from "@/components/i18n/bidi";

interface Verification {
  letter_no: string;
  diwan_no?: string | null;
  issued_date: string;
  recipient_name: string;
  signer_title: string;
}

/** Public page: confirms that a printed letter's code is genuine. */
export default function VerifyLetterPage() {
  const params = useParams<{ id: string }>();
  const t = useTranslations("portal");
  const [result, setResult] = useState<Verification | null>(null);
  const [state, setState] = useState<"loading" | "valid" | "invalid">("loading");

  useEffect(() => {
    let active = true;
    api
      .get<Verification>(`/api/v1/verify/letter/${params.id}`)
      .then((res) => {
        if (!active) return;
        setResult(res.data ?? null);
        setState(res.data ? "valid" : "invalid");
      })
      .catch(() => active && setState("invalid"));
    return () => {
      active = false;
    };
  }, [params.id]);

  return (
    <Card className="mx-auto max-w-lg">
      <CardHeader>
        <CardTitle className="text-center">{t("letter.verifyTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4 text-center">
        {state === "loading" && <p className="text-muted-foreground">{t("ui.loading")}</p>}
        {state === "invalid" && (
          <div className="space-y-2 text-status-danger-fg">
            <XCircle className="mx-auto size-10" />
            <p>{t("letter.invalid")}</p>
          </div>
        )}
        {state === "valid" && result && (
          <div className="space-y-3">
            <CheckCircle2 className="mx-auto size-10 text-status-success" />
            <p className="font-medium text-status-success-fg">{t("letter.valid")}</p>
            <dl className="space-y-1 text-sm">
              <div>
                <dt className="inline text-muted-foreground">{t("letter.number")}: </dt>
                <dd className="inline font-medium"><Ltr>{result.diwan_no || result.letter_no}</Ltr></dd>
              </div>
              {result.diwan_no && (
                <div>
                  <dt className="inline text-muted-foreground">{t("letter.internalNo")}: </dt>
                  <dd className="inline font-medium"><Ltr>{result.letter_no}</Ltr></dd>
                </div>
              )}
              <div>
                <dt className="inline text-muted-foreground">{t("letter.issuedTo")}: </dt>
                <dd className="inline font-medium"><Bidi text={result.recipient_name} /></dd>
              </div>
              <div>
                <dt className="inline text-muted-foreground">{t("letter.issuedOn")}: </dt>
                <dd className="inline font-medium"><Ltr>{result.issued_date}</Ltr></dd>
              </div>
              <Bidi as="p" block text={result.signer_title} className="text-center text-muted-foreground" />
            </dl>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
