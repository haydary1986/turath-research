import { cn } from "@/lib/utils";

/** "used / max" hint under a text control; turns red past the limit. */
export function CharCount({ value, max, id }: { value: string; max: number; id?: string }) {
  const used = [...value].length;
  return (
    <p id={id} className={cn("mt-1 text-end text-xs tabular-nums", used > max ? "text-destructive" : "text-muted-foreground")} dir="ltr">
      {used} / {max}
    </p>
  );
}
