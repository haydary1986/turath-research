"use client";

import { useLocale, useTranslations } from "next-intl";
import { Check, Undo2, X } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  formatDate,
  type RequestEvent,
  type RequestKind,
  type RequestStatus,
} from "@/lib/portal/types";

type NodeState = "done" | "current" | "future" | "danger" | "warning";

interface Stage {
  key: string;
  /** to_status values whose event marks this stage as passed. */
  doneOn: string[];
}

const COMMON: Stage[] = [
  { key: "submission", doneOn: ["pending_dean"] },
  { key: "dean", doneOn: ["pending"] },
  { key: "sa", doneOn: ["pending_vp"] },
  { key: "vp", doneOn: ["approved", "rejected", "verified", "not_eligible"] },
];

const STAGES: Record<RequestKind, Stage[]> = {
  research_plan: [...COMMON, { key: "decision_research_plan", doneOn: ["approved", "rejected"] }],
  journal_verification: [
    ...COMMON,
    { key: "decision_journal_verification", doneOn: ["verified", "not_eligible"] },
    { key: "signatures", doneOn: ["completed"] },
    { key: "completed", doneOn: ["completed"] },
  ],
  appreciation_letter: [
    ...COMMON,
    { key: "decision_appreciation_letter", doneOn: ["verified", "not_eligible"] },
    { key: "letter", doneOn: ["issued"] },
  ],
  // Decided by scientific affairs alone: no dean or VP stage.
  title_change: [
    { key: "submission", doneOn: ["pending"] },
    { key: "sa", doneOn: ["approved", "rejected"] },
    { key: "decision_title_change", doneOn: ["approved", "rejected"] },
  ],
};

/** Stage index per status for the short title-change chain. */
const TITLE_CHANGE_STAGE: Partial<Record<RequestStatus, number>> = {
  draft: 0,
  returned: 0,
  pending: 1,
  approved: 2,
  rejected: 2,
};

/** Index of the last stage, where a rejection is shown. */
function decisionIndex(kind: RequestKind): number {
  return kind === "title_change" ? 2 : 4;
}

function currentStage(kind: RequestKind, status: RequestStatus): number {
  if (kind === "title_change") return TITLE_CHANGE_STAGE[status] ?? 0;
  return CURRENT_STAGE[status] ?? 0;
}

/** Index of the stage the request currently sits at, per status. */
const CURRENT_STAGE: Record<RequestStatus, number> = {
  draft: 0,
  returned: 0,
  pending_dean: 1,
  pending: 2,
  pending_vp: 3,
  approved: 4,
  rejected: 4,
  verified: 5,
  not_eligible: 4,
  awaiting_signatures: 5,
  completed: 6,
  issued: 5,
};

const TERMINAL_DANGER: RequestStatus[] = ["rejected", "not_eligible"];
const TERMINAL_DONE: RequestStatus[] = ["approved", "completed", "issued"];

function nodeState(index: number, status: RequestStatus, kind: RequestKind): NodeState {
  const current = currentStage(kind, status);
  if (TERMINAL_DANGER.includes(status)) {
    if (index === current) return "danger";
    return index < current ? "done" : "future";
  }
  if (TERMINAL_DONE.includes(status) && index <= current) return "done";
  if (status === "verified" && index <= 4) return "done";
  if (index < current) return "done";
  if (index > current) return "future";
  return status === "returned" ? "warning" : "current";
}

const CIRCLE: Record<NodeState, string> = {
  done: "border-primary bg-primary text-primary-foreground",
  current: "border-primary bg-background text-primary ring-2 ring-primary/30",
  future: "border-border bg-background text-muted-foreground",
  danger: "border-status-danger bg-status-danger text-white",
  warning: "border-status-warning bg-status-warning text-white",
};

const LABEL: Record<NodeState, string> = {
  done: "text-foreground",
  current: "font-bold text-foreground",
  future: "text-muted-foreground",
  danger: "font-bold text-status-danger-fg",
  warning: "font-bold text-status-warning-fg",
};

interface RequestStepperProps {
  kind: RequestKind;
  status: RequestStatus;
  events: RequestEvent[];
}

/** Horizontal map of the approval chain: where the request stands and what already passed. */
export function RequestStepper({ kind, status, events }: RequestStepperProps) {
  const t = useTranslations("portal.stages");
  const locale = useLocale();
  const stages = STAGES[kind];

  function stageDate(stage: Stage): string | null {
    // Events come oldest first. A request can pass a stage twice after being sent back, and
    // an event that keeps the status (a registry-number edit) does not complete a stage.
    const event = events
      .filter((e) => stage.doneOn.includes(e.to_status) && e.from_status !== e.to_status)
      .at(-1);
    return event ? formatDate(event.created_at, locale) : null;
  }

  return (
    <ol className="flex items-start overflow-x-auto py-2" aria-label={t("title")}>
      {stages.map((stage, index) => {
        const state = nodeState(index, status, kind);
        const date = state === "done" || state === "danger" ? stageDate(stage) : null;
        const label =
          state === "danger" && index === decisionIndex(kind)
            ? t(status === "rejected" ? "rejectedLabel" : "notEligibleLabel")
            : t(stage.key);
        return (
          <li
            key={stage.key}
            aria-current={state === "current" ? "step" : undefined}
            className="relative flex min-w-20 flex-1 flex-col items-center gap-1.5 px-1 text-center"
          >
            {index > 0 && (
              <span
                aria-hidden
                className={cn(
                  "absolute top-4 h-0.5 w-full",
                  "start-[-50%]",
                  state === "future" ? "bg-border" : "bg-primary/60"
                )}
              />
            )}
            <span
              className={cn(
                "relative z-10 flex size-8 shrink-0 items-center justify-center rounded-full border-2 text-xs font-bold",
                CIRCLE[state]
              )}
            >
              {state === "done" ? (
                <Check className="size-4" aria-hidden />
              ) : state === "danger" ? (
                <X className="size-4" aria-hidden />
              ) : state === "warning" ? (
                <Undo2 className="size-4" aria-hidden />
              ) : (
                index + 1
              )}
            </span>
            <span className={cn("text-xs leading-tight", LABEL[state])}>{label}</span>
            {date && <span className="text-[10px] text-muted-foreground tabular-nums">{date}</span>}
          </li>
        );
      })}
    </ol>
  );
}
