/**
 * sx for a table column (header + body cell) that collapses below the `md`
 * breakpoint, so dense admin tables drop secondary columns and fit a phone
 * instead of clipping off the right edge.
 */
export const hideUntilMd = { display: { xs: "none", md: "table-cell" } } as const;

/**
 * Responsive `minWidth` for an admin table: lets it shrink to the viewport on
 * phones (no forced horizontal scroll) while keeping a comfortable width on
 * desktop.
 */
export function tableMinWidth(desktop: number) {
  return { minWidth: { xs: "auto", md: desktop } } as const;
}
