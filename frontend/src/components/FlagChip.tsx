import { useEffect, useRef } from "react";
import { Globe } from "lucide-react";
import * as Flags from "country-flag-icons/react/3x2";
import { countryCodeFor } from "../flags";

interface Props {
  name: string;
  className?: string;
}

export default function FlagChip({ name, className }: Props) {
  const cc = countryCodeFor(name);
  const Flag = cc
    ? (Flags as Record<string, React.ComponentType<React.SVGProps<SVGSVGElement>>>)[cc]
    : null;
  const ref = useRef<SVGSVGElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    // WebView2 paints the library's <title>/<desc> as text over the flag.
    el.querySelectorAll("title, desc").forEach((n) => n.remove());
  }, [cc]);

  if (!Flag) {
    return <Globe size={16} className={className ?? "text-text-faint"} />;
  }
  return (
    <Flag
      ref={ref}
      aria-hidden
      preserveAspectRatio="xMidYMid slice"
      className={className ?? "h-full w-full"}
    />
  );
}
