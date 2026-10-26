"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { CheckCircle2, XCircle } from "lucide-react";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface Verification {
  letter_no: string;
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
          <div className="space-y-2 text-red-600">
            <XCircle className="mx-auto size-10" />
            <p>{t("letter.invalid")}</p>
          </div>
        )}
        {state === "valid" && result && (
          <div className="space-y-3">
            <CheckCircle2 className="mx-auto size-10 text-green-600" />
            <p className="font-medium text-green-700">{t("letter.valid")}</p>
            <dl className="space-y-1 text-sm">
              <div>
                <dt className="inline text-muted-foreground">{t("letter.number")}: </dt>
                <dd className="inline font-medium" dir="ltr">{result.letter_no}</dd>
              </div>
              <div>
                <dt className="inline text-muted-foreground">{t("letter.issuedTo")}: </dt>
                <dd className="inline font-medium">{result.recipient_name}</dd>
              </div>
              <div>
                <dt className="inline text-muted-foreground">{t("letter.issuedOn")}: </dt>
                <dd className="inline font-medium" dir="ltr">{result.issued_date}</dd>
              </div>
              {result.signer_title && <p className="text-muted-foreground">{result.signer_title}</p>}
            </dl>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
