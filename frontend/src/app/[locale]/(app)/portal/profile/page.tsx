"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { FileText, Loader2, Quote, RefreshCw, TrendingUp } from "lucide-react";
import { api } from "@/lib/api";
import { PROFILE_ENDPOINT, type ResearchProfile } from "@/lib/portal/research-profile";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RequireAuth } from "@/components/portal/require-auth";
import { ResearchLinksForm, type LinksFormValues } from "@/components/portal/research-links-form";
import { ResearchPublications } from "@/components/portal/research-publications";
import { usePortalErrors } from "@/components/portal/use-portal-errors";

const POLL_INTERVAL_MS = 3000;

function formFrom(profile: ResearchProfile): LinksFormValues {
  return {
    orcid: profile.orcid ?? "",
    google_scholar_url: profile.google_scholar_url ?? "",
    scopus_url: profile.scopus_url ?? "",
    researchgate_url: profile.researchgate_url ?? "",
    linkedin_url: profile.linkedin_url ?? "",
    web_of_science_url: profile.web_of_science_url ?? "",
    phone: profile.phone ?? "",
    phone_secondary: profile.phone_secondary ?? "",
    recovery_email: profile.recovery_email ?? "",
  };
}

function ProfilePage() {
  const t = useTranslations("portal.profile");
  const locale = useLocale();
  const toMessage = usePortalErrors();

  const [profile, setProfile] = useState<ResearchProfile | null>(null);
  const [form, setForm] = useState<LinksFormValues | null>(null);
  const [loadFailed, setLoadFailed] = useState(false);
  const [saving, setSaving] = useState(false);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const wasSyncing = useRef(false);

  const load = useCallback(() => {
    api
      .get<ResearchProfile>(PROFILE_ENDPOINT)
      .then((res) => {
        const data = res.data ?? null;
        setProfile(data);
        if (data) setForm(formFrom(data));
      })
      .catch(() => setLoadFailed(true));
  }, []);

  useEffect(load, [load]);

  const retry = () => {
    setLoadFailed(false);
    load();
  };

  const syncing = profile?.syncing ?? false;

  // Polling replaces only the read-only data, so text being typed in the form survives.
  useEffect(() => {
    if (!syncing) return;
    wasSyncing.current = true;
    const timer = setInterval(() => {
      api
        .get<ResearchProfile>(PROFILE_ENDPOINT)
        .then((res) => {
          if (!res.data) return;
          setProfile(res.data);
          if (!res.data.syncing && wasSyncing.current) {
            wasSyncing.current = false;
            if (!res.data.last_sync_error) setNotice(t("syncDone"));
          }
        })
        .catch((err) => setError(toMessage(err)));
    }, POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [syncing, t, toMessage]);

  const save = () => {
    if (!form) return;
    setSaving(true);
    setError("");
    setNotice("");
    api
      .put<ResearchProfile>(PROFILE_ENDPOINT, form)
      .then((res) => {
        if (!res.data) return;
        setProfile(res.data);
        setForm(formFrom(res.data));
        setNotice(res.data.syncing ? t("savedSyncing") : t("saved"));
      })
      .catch((err) => setError(toMessage(err)))
      .finally(() => setSaving(false));
  };

  const refresh = () => {
    setStarting(true);
    setError("");
    setNotice("");
    api
      .post(`${PROFILE_ENDPOINT}/sync`, {})
      .then(() => setProfile((current) => current && { ...current, syncing: true, last_sync_error: null }))
      .catch((err) => setError(toMessage(err)))
      .finally(() => setStarting(false));
  };

  if (loadFailed) {
    return (
      <Alert variant="danger" className="flex flex-wrap items-center justify-between gap-3">
        <span>{t("loadError")}</span>
        <Button variant="outline" size="sm" onClick={retry}>{t("retry")}</Button>
      </Alert>
    );
  }
  if (!profile || !form) {
    return <Loader2 className="mx-auto mt-10 size-6 animate-spin text-muted-foreground" aria-hidden />;
  }

  const stats = [
    { label: t("hIndex"), value: profile.h_index, icon: TrendingUp },
    { label: t("citations"), value: profile.total_citations, icon: Quote },
    { label: t("publications"), value: profile.total_publications, icon: FileText },
  ];
  const syncErrorKey = `syncErrors.${profile.last_sync_error}`;
  const hasOrcid = Boolean(profile.orcid);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("subtitle")}</p>
      </div>

      {error && <Alert variant="danger">{error}</Alert>}
      {notice && !error && <Alert variant="success">{notice}</Alert>}

      <ResearchLinksForm
        values={form}
        saving={saving}
        onChange={(key, value) => setForm((current) => current && { ...current, [key]: value })}
        onSubmit={save}
      />

      <Card>
        <CardHeader className="flex-row flex-wrap items-center justify-between gap-3">
          <div className="min-w-0">
            <CardTitle>{t("statsTitle")}</CardTitle>
            <p className="mt-1 text-xs text-muted-foreground">
              {profile.last_synced_at
                ? t("lastSync", { time: new Date(profile.last_synced_at).toLocaleString(locale) })
                : t("neverSynced")}
            </p>
          </div>
          {hasOrcid && (
            <Button variant="outline" size="sm" disabled={syncing || starting} onClick={refresh}>
              <RefreshCw className={`size-4 ${syncing || starting ? "animate-spin" : ""}`} aria-hidden />
              <span>{t("refresh")}</span>
            </Button>
          )}
        </CardHeader>
        <CardContent className="space-y-4">
          {syncing && (
            <Alert variant="progress" className="flex items-center gap-2">
              <Loader2 className="size-4 shrink-0 animate-spin" aria-hidden />
              <span>{t("syncing")}</span>
            </Alert>
          )}
          {!syncing && profile.last_sync_error && (
            <Alert variant="warning">
              {t.has(syncErrorKey) ? t(syncErrorKey) : t("syncErrors.generic")}
            </Alert>
          )}
          <div className="grid gap-4 sm:grid-cols-3">
            {stats.map(({ label, value, icon: Icon }) => (
              <div key={label} className="rounded-xl border p-4">
                <div className="flex items-center justify-between text-sm text-muted-foreground">
                  <span>{label}</span>
                  <Icon className="size-4" aria-hidden />
                </div>
                <p className="mt-2 text-3xl font-bold tabular-nums">{value}</p>
              </div>
            ))}
          </div>
          {hasOrcid && profile.last_synced_at && (
            <p className="text-xs text-muted-foreground">
              {profile.h_index_source ? t("statsSource") : t("statsNoSource")}
            </p>
          )}
        </CardContent>
      </Card>

      <ResearchPublications
        publications={profile.publications}
        total={profile.total_publications}
        hasOrcid={hasOrcid}
        syncing={syncing}
      />
    </div>
  );
}

export default function ResearchProfileRoute() {
  return (
    <RequireAuth>
      <ProfilePage />
    </RequireAuth>
  );
}
