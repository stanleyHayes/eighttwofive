import { useEffect, useRef, useState, type ElementType, type ReactNode } from "react";
import Box from "@mui/material/Box";
import type { SxProps, Theme } from "@mui/material/styles";

interface RevealProps {
  children: ReactNode;
  /** Delay in ms before this element rises in (for staggered groups). */
  delay?: number;
  /** Translate distance in px before settling. */
  rise?: number;
  component?: ElementType;
  sx?: SxProps<Theme>;
}

/**
 * Fades and rises its children into view the first time they enter the
 * viewport. Honors prefers-reduced-motion (content is simply shown). Uses a
 * single IntersectionObserver per element and disconnects after revealing.
 */
export function Reveal({ children, delay = 0, rise = 22, component = "div", sx }: RevealProps) {
  const ref = useRef<HTMLDivElement | null>(null);
  // Without IntersectionObserver (SSR, jsdom, old browsers) content is shown
  // immediately rather than staying hidden.
  const [shown, setShown] = useState(() => typeof IntersectionObserver === "undefined");

  useEffect(() => {
    if (shown) return;

    const node = ref.current;
    if (!node) return;

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setShown(true);
            observer.disconnect();
          }
        }
      },
      { threshold: 0.12, rootMargin: "0px 0px -8% 0px" },
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [shown]);

  return (
    <Box
      ref={ref}
      component={component}
      sx={[
        {
          "@media (prefers-reduced-motion: no-preference)": {
            opacity: shown ? 1 : 0,
            transform: shown ? "none" : `translateY(${rise}px)`,
            transition: "opacity 720ms cubic-bezier(0.22, 1, 0.36, 1), transform 720ms cubic-bezier(0.22, 1, 0.36, 1)",
            transitionDelay: `${delay}ms`,
            willChange: "opacity, transform",
          },
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {children}
    </Box>
  );
}
