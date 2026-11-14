"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Alert } from "@/components/ui/alert";
import { RequireAuth } from "@/components/portal/require-auth";
import { usePortalErrors } from "@/components/portal/use-portal-errors";
import { ChangesTable } from "@/components/portal/ris-sync/changes-table";
import { ReviewsList } from "@/components/portal/ris-sync/reviews-list";
import { RunsPanel } from "@/components/portal/ris-sync/runs-panel";
import { RIS_SYNC_API, type RISSyncReview, type RISSyncRun, type RISSyncStatus } from "@/lib/portal/ris-sync";
import type { Org } from "@/lib/portal/types";

const POLL_MS = 3000;

/** «المزامنة مع RIS»: run the two-way profile sync, see what it changed, decide conflicts. */
function RISSyncPage() {
  const t = useTranslations("portal.risSync");
  const toMessage = usePortalErrors();
  const [status, setStatus] = useState<RISSyncStatus | null>(null);
  const [reviews, setReviews] = useState<RISSyncReview[]>([]);
  const [departments, setDepartments] = useState<Map<string, string>>(new Map());
  const [selected, setSelected] = useState<RISSyncRun | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);
  const refresh = () => setTick((n) => n + 1);

  useEffect(() => {
    let active = true;
    Promise.all([api.get<RISSyncStatus>(RIS_SYNC_API.status), api.get<RISSyncReview[]>(RIS_SYNC_API.reviews)])
      .then(([s, r]) => {
        if (!active) return;
        setStatus(s.data ?? null);
        setReviews(r.data ?? []);
      })
      .catch((err) => active && setError(toMessage(err)));
    return () => {
      active = false;
    };
  }, [tick, toMessage]);

  useEffect(() => {
    let active = true;
    api.get<Org[]>("/api/v1/departments")
      .then((res) => active && setDepartments(new Map((res.data ?? []).map((d) => [d.id, d.name_ar]))))
      .catch(() => undefined);
    return () => {
      active = false;
    };
  }, []);

  // While a run is in progress, poll until it finishes.
  const running = status?.runs.some((r) => !r.finished_at) ?? false;
  useEffect(() => {
    if (!running) return;
    const timer = setInterval(refresh, POLL_MS);
    return () => clearInterval(timer);
  }, [running]);

  async function start(mode: "dry_run" | "apply") {
    if (mode === "apply" && !window.confirm(t("confirmApply"))) return;
    setBusy(true);
    setError("");
    try {
      await api.post(RIS_SYNC_API.runs, { mode });
      setSelected(null);
      refresh();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }

  const selectedRun = selected && status?.runs.find((r) => r.id === selected.id);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("title")}</h1>
        <p className="text-sm text-muted-foreground">{t("intro")}</p>
      </div>
      {error && <Alert variant="danger">{error}</Alert>}
      {status && <RunsPanel status={status} busy={busy} selectedRun={selected?.id ?? null} onStart={start} onSelect={setSelected} />}
      {status?.runs[0]?.stats.unmapped_departments?.length ? (
        <Alert variant="warning">
          {t("unmapped", { names: status.runs[0].stats.unmapped_departments.join("، ") })}
        </Alert>
      ) : null}
      {selectedRun?.finished_at && <ChangesTable run={selectedRun} departments={departments} />}
      <ReviewsList reviews={reviews} departments={departments} onResolved={refresh} />
    </div>
  );
}

export default function Page() {
  const role = useAuth((s) => s.user?.role);
  return (
    <RequireAuth>
      {role === "admin" ? <RISSyncPage /> : null}
    </RequireAuth>
  );
}
