import Image from "next/image";
import { cn } from "@/lib/utils";

/** Official Al-Turath University artwork in public/brand (never edited). */
const ARTWORK = {
  /** Crest + navy wordmark, for light backgrounds. */
  color: { src: "/brand/logo-color.png", width: 1000, height: 500 },
  /** Crest + white wordmark, for navy / dark backgrounds. */
  onDark: { src: "/brand/logo-on-dark.png", width: 1000, height: 500 },
  /** The shield crest alone. */
  crest: { src: "/brand/crest.png", width: 634, height: 844 },
} as const;

export type LogoVariant = keyof typeof ARTWORK;

interface UniversityLogoProps {
  variant: LogoVariant;
  /** Rendered height in CSS pixels; the width follows the artwork's ratio. */
  height: number;
  alt: string;
  className?: string;
  priority?: boolean;
}

export function UniversityLogo({ variant, height, alt, className, priority }: UniversityLogoProps) {
  const art = ARTWORK[variant];
  const width = Math.round((art.width / art.height) * height);
  return (
    <Image
      src={art.src}
      width={width}
      height={height}
      alt={alt}
      priority={priority}
      className={cn("h-auto max-w-full select-none", className)}
      style={{ height, width: "auto" }}
    />
  );
}
