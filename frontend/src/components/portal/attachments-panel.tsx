"use client";

import { useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { CheckCircle2, CircleDashed, FileText, Trash2, Upload } from "lucide-react";
import { Button } from "@/components/ui/button";
import { api, fetchObjectUrl } from "@/lib/api";
import type { Attachment } from "@/lib/portal/types";
import { usePortalErrors } from "./use-portal-errors";
import { Alert } from "@/components/ui/alert";

interface AttachmentsPanelProps {
  requestId: string;
  attachments: Attachment[];
  /** Roles shown with an upload button. */
  uploadRoles: string[];
  /** Roles that must be present before the request can be submitted. */
  requiredRoles: string[];
  /** Whether the current user may remove this attachment. */
  canRemove: (file: Attachment) => boolean;
  onChanged: () => void;
}

export async function openProtectedFile(fileId: string): Promise<void> {
  // Open the tab synchronously so popup blockers allow it, then point it at the blob.
  const tab = window.open("", "_blank");
  try {
    const url = await fetchObjectUrl(`/api/v1/portal/files/${fileId}`);
    if (tab) tab.location.href = url;
  } catch (error) {
    tab?.close();
    throw error;
  }
}

function formatSize(bytes: number): string {
  return bytes < 1024 * 1024
    ? `${Math.max(1, Math.round(bytes / 1024))} KB`
    : `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function AttachmentsPanel({
  requestId,
  attachments,
  uploadRoles,
  requiredRoles,
  canRemove,
  onChanged,
}: AttachmentsPanelProps) {
  const t = useTranslations("portal");
  const toMessage = usePortalErrors();
  const [busyRole, setBusyRole] = useState<string | null>(null);
  const [error, setError] = useState("");
  const inputs = useRef<Record<string, HTMLInputElement | null>>({});

  async function handleFile(role: string, file: File | undefined) {
    if (!file) return;
    setError("");
    setBusyRole(role);
    try {
      await api.upload(`/api/v1/portal/requests/${requestId}/files?role=${role}`, file);
      onChanged();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusyRole(null);
      const input = inputs.current[role];
      if (input) input.value = "";
    }
  }

  async function handleRemove(id: string) {
    if (!window.confirm(t("ui.deleteConfirm"))) return;
    setError("");
    try {
      await api.delete(`/api/v1/portal/files/${id}`);
      onChanged();
    } catch (err) {
      setError(toMessage(err));
    }
  }

  async function handleOpen(id: string) {
    setError("");
    try {
      await openProtectedFile(id);
    } catch (err) {
      setError(toMessage(err));
    }
  }

  const roles = [...new Set([...requiredRoles, ...uploadRoles, ...attachments.map((a) => a.role)])];

  return (
    <div className="space-y-4">
      {error && (
        <Alert variant="danger">{error}</Alert>
      )}
      {roles.map((role) => {
        const files = attachments.filter((a) => a.role === role);
        const required = requiredRoles.includes(role);
        const canUpload = uploadRoles.includes(role);
        return (
          <div key={role} className="rounded-lg border p-3">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 text-sm font-medium">
                {files.length > 0 ? (
                  <CheckCircle2 className="size-4 text-status-success" />
                ) : (
                  <CircleDashed className="size-4 text-muted-foreground" />
                )}
                <span>{t(`roles.${role}`)}</span>
                {required && (
                  <span className="text-xs text-destructive">* {t("ui.required")}</span>
                )}
              </div>
              {canUpload && (
                <>
                  <input
                    ref={(el) => {
                      inputs.current[role] = el;
                    }}
                    type="file"
                    accept="application/pdf,image/png,image/jpeg"
                    className="hidden"
                    onChange={(e) => handleFile(role, e.target.files?.[0])}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={busyRole === role}
                    onClick={() => inputs.current[role]?.click()}
                  >
                    <Upload className="size-3.5" />
                    <span>{busyRole === role ? t("ui.uploading") : t("ui.upload")}</span>
                  </Button>
                </>
              )}
            </div>
            {files.map((file) => (
              <div key={file.id} className="mt-2 flex items-center justify-between gap-2 text-sm">
                <button
                  type="button"
                  onClick={() => handleOpen(file.id)}
                  className="flex min-w-0 items-center gap-2 text-primary hover:underline"
                >
                  <FileText className="size-4 shrink-0" />
                  <span className="truncate">{file.original_name}</span>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {formatSize(file.size_bytes)}
                  </span>
                </button>
                {canRemove(file) && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t("ui.remove")}
                    onClick={() => handleRemove(file.id)}
                  >
                    <Trash2 className="size-3.5 text-destructive" />
                  </Button>
                )}
              </div>
            ))}
          </div>
        );
      })}
      <p className="text-xs text-muted-foreground">{t("ui.fileHint")}</p>
    </div>
  );
}
