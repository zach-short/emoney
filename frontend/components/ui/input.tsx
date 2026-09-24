import * as React from "react"

import { cn } from "@/lib/utils"

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<"input">>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        // What a field is, decided in Phase 3 (PLAN.md item 8; Phase 1 left
        // `bg-transparent` untouched and could not verify how it painted).
        // A field is a RECESSED well, not a raised control: it is the one
        // place the page is asking for something back, so it reads as a hole
        // in the surface rather than a plane on top of it. On a near-black
        // ground that is a faint fill -- transparent gave the field no extent
        // at all and left the hairline border doing the whole job -- plus the
        // `flat` elevation step, because a drop shadow would say "raised".
        // No fifth token: the recess is carried by the fill, which keeps D4's
        // scale at the four steps its dial fixes.
        className={cn(
          "flex h-9 w-full rounded-md border border-input bg-white/[0.04] px-3 py-1 text-base shadow-flat transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 md:text-sm",
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Input.displayName = "Input"

export { Input }
