import { createElement } from "react";
import {
  Award, BadgeCheck, BookOpen, CalendarDays, ChartColumn, FileText, LayoutGrid, Microscope, SearchCheck, ShieldAlert, Users,
  type LucideIcon,
} from "lucide-react";
import type { ServiceIcon as ServiceIconKey } from "@/lib/site/types";

const ICONS: Record<ServiceIconKey, LucideIcon> = {
  research: Microscope,
  verify: BadgeCheck,
  award: Award,
  calendar: CalendarDays,
  search: SearchCheck,
  shield: ShieldAlert,
  file: FileText,
  users: Users,
  book: BookOpen,
  chart: ChartColumn,
};

/** The glyph of a service card; unknown or empty keys get a neutral grid icon. */
export function ServiceIcon({ name, className }: { name: string; className?: string }) {
  return createElement(ICONS[name as ServiceIconKey] ?? LayoutGrid, { className, "aria-hidden": true });
}
