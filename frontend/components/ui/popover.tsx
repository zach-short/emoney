"use client";

import * as React from "react";
import * as PopoverPrimitive from "@radix-ui/react-popover";

import { cn } from "@/lib/utils";

const Popover = PopoverPrimitive.Root;

const PopoverTrigger = PopoverPrimitive.Trigger;

const PopoverAnchor = PopoverPrimitive.Anchor;

const PopoverContent = React.forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>
>(({ className, align = "center", sideOffset = 8, ...props }, ref) => (
  <PopoverPrimitive.Portal>
    <PopoverPrimitive.Content
      ref={ref}
      align={align}
      sideOffset={sideOffset}
      // The third of D4's four glass surfaces: "popover and menu surfaces".
      // Same treatment as the scrim (`ui/drawer.tsx`) and for the same reason
      // -- on a black ground an opaque panel reads as a page rather than as
      // something floating over the room -- and the `overlay` step of the
      // elevation scale, which is what separates it from a drawer's `modal`.
      //
      // The stock shadcn entrance and exit, added in Phase 6 behind the
      // reduced-motion gate (PLAN.md BD-16, which held them back until the gate
      // existed). Only `animate-in` / `animate-out` start an animation, and
      // both are `motion-safe:`, so under reduced motion the popover appears
      // and goes at once; the rest only set the values those two read. 180 ms
      // is the "Standard transition" dial (PLAN.md section 3).
      className={cn(
        "z-50 w-72 rounded-md border bg-popover/55 p-4 text-popover-foreground shadow-overlay outline-none backdrop-blur-[12px]",
        "motion-safe:data-[state=open]:animate-in motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=closed]:zoom-out-95 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[side=bottom]:slide-in-from-top-2 motion-safe:data-[side=left]:slide-in-from-right-2 motion-safe:data-[side=right]:slide-in-from-left-2 motion-safe:data-[side=top]:slide-in-from-bottom-2 motion-safe:data-[state=open]:duration-standard motion-safe:data-[state=closed]:duration-standard motion-safe:ease-out",
        className,
      )}
      collisionPadding={12}
      {...props}
    />
  </PopoverPrimitive.Portal>
));
PopoverContent.displayName = PopoverPrimitive.Content.displayName;

export { Popover, PopoverTrigger, PopoverAnchor, PopoverContent };
