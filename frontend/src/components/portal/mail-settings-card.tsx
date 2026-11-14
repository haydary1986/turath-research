"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Mail, Send } from "lucide-react";
import { api } from "@/lib/api";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Ltr } from "@/components/i18n/bidi";
import { usePortalErrors } from "./use-portal-errors";

interface MailStatus {
  enabled: boolean;
  sender: string;
}

/** Whether notification email is configured, and a button that sends a test message. */
export function MailSettingsCard() {
  const t = useTranslations("portal.mail");
  const toMessage = usePortalErrors();
  const [status, setStatus] = useState<MailStatus | null>(null);
  const [to, setTo] = useState("");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; text: string } | null>(null);

  useEffect(() => {
    let active = true;
    api.get<MailStatus>("/api/v1/admin/mail")
      .then((res) => active && setStatus(res.data ?? null))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  async function sendTest(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setResult(null);
    try {
      await api.post("/api/v1/admin/mail/test", { to: to.trim() });
      setResult({ ok: true, text: t("sent", { to: to.trim() }) });
    } catch (err) {
      setResult({ ok: false, text: toMessage(err) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card>
      <CardHeader className="flex-row flex-wrap items-center justify-between gap-2">
        <CardTitle className="flex items-center gap-2">
          <Mail className="size-5" aria-hidden />
          {t("title")}
        </CardTitle>
        {status && (
          <Badge variant="secondary" className={status.enabled ? "bg-status-success-bg text-status-success-fg" : "bg-status-warning-bg text-status-warning-fg"}>
            {status.enabled ? t("enabled") : t("disabled")}
          </Badge>
        )}
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">
          {status?.enabled ? <>{t("sender")}: <Ltr>{status.sender}</Ltr></> : t("disabledHint")}
        </p>
        {status?.enabled && (
          <form onSubmit={sendTest} className="flex flex-wrap items-end gap-2">
            <div className="min-w-64 flex-1 space-y-1.5">
              <Label htmlFor="mail-test-to">{t("testTo")}</Label>
              <Input id="mail-test-to" type="email" dir="ltr" required value={to} onChange={(e) => setTo(e.target.value)} />
            </div>
            <Button type="submit" disabled={busy || !to.trim()}>
              <Send className="size-4" aria-hidden />
              {busy ? t("sending") : t("sendTest")}
            </Button>
          </form>
        )}
        {result && <Alert variant={result.ok ? "success" : "danger"}>{result.text}</Alert>}
      </CardContent>
    </Card>
  );
}
