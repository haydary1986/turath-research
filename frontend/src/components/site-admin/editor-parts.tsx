"use client";

import type { ReactNode } from "react";
import { useTranslations } from "next-intl";
import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";

/** Returns a copy of `list` with the item at `from` moved to `to`. */
export function move<T>(list: T[], from: number, to: number): T[] {
  if (to < 0 || to >= list.length) return list;
  const next = [...list];
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

export function replaceAt<T>(list: T[], index: number, value: T): T[] {
  return list.map((item, i) => (i === index ? value : item));
}

export function removeAt<T>(list: T[], index: number): T[] {
  return list.filter((_, i) => i !== index);
}

export function Field({ id, label, children, hint }: { id: string; label: string; children: ReactNode; hint?: string }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}

interface RowControlsProps {
  index: number;
  count: number;
  onMove: (to: number) => void;
  onRemove: () => void;
}

export function RowControls({ index, count, onMove, onRemove }: RowControlsProps) {
  const t = useTranslations("portal.site.content");
  return (
    <div className="flex gap-1">
      <Button type="button" variant="ghost" size="icon-sm" aria-label={t("moveUp")} disabled={index === 0} onClick={() => onMove(index - 1)}>
        <ArrowUp aria-hidden />
      </Button>
      <Button type="button" variant="ghost" size="icon-sm" aria-label={t("moveDown")} disabled={index === count - 1} onClick={() => onMove(index + 1)}>
        <ArrowDown aria-hidden />
      </Button>
      <Button type="button" variant="ghost" size="icon-sm" aria-label={t("remove")} onClick={onRemove}>
        <Trash2 className="text-destructive" aria-hidden />
      </Button>
    </div>
  );
}

interface ListHeaderProps {
  title: string;
  hint?: string;
  addLabel: string;
  canAdd: boolean;
  onAdd: () => void;
}

/** Card header of an editable list: title, optional hint and an "add" button. */
export function ListHeader({ title, hint, addLabel, canAdd, onAdd }: ListHeaderProps) {
  return (
    <CardHeader className="flex-row flex-wrap items-start justify-between gap-2">
      <div>
        <CardTitle>{title}</CardTitle>
        {hint && <p className="mt-1 text-sm text-muted-foreground">{hint}</p>}
      </div>
      <Button type="button" variant="outline" size="sm" disabled={!canAdd} onClick={onAdd}>
        <Plus aria-hidden />
        {addLabel}
      </Button>
    </CardHeader>
  );
}
