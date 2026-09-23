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
      // No entry animation, deliberately: the stock shadcn popover ships four
      // `data-[state=*]:animate-*` classes, and unguarded motion here would
      // land before Phase 5 establishes the reduced-motion gate this work
      // ratified (PLAN.md section 3; nothing in the tree handles
      // `prefers-reduced-motion` as of 2026-09-23). Panel transitions are
      // Phase 6's (PLAN.md BD-16).
      className={cn(
        "z-50 w-72 rounded-md border bg-popover/55 p-4 text-popover-foreground shadow-overlay outline-none backdrop-blur-[12px]",
        className,
      )}
      collisionPadding={12}
      {...props}
    />
  </PopoverPrimitive.Portal>
));
PopoverContent.displayName = PopoverPrimitive.Content.displayName;

export { Popover, PopoverTrigger, PopoverAnchor, PopoverContent };
