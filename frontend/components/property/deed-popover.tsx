"use client";

import { Property } from "@/types/schema";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import PropertyCard from "./cards/card";

// F1 (DESIGN.md D6): a deed's terms read in place, with the room still visible
// behind it. Both surfaces here are reference-only -- no deed is given an
// `onClick`, so `PropertyCardContainer` takes its read-only `<div>` branch and
// nothing lands in the tab order that does not do something.
//
// Both are gated to `lg` and above by their call sites, not here: D6 fixes the
// breakpoint and the drawer path below it is left completely unchanged.

// One deed, and the deed is the whole surface -- PAPER, not glass (Zach's
// call, chat 2026-09-23; PLAN.md BD-15). D4 glasses "popover and menu
// surfaces" but exempts the title deed by name, and the content here *is* the
// deed, so `PopoverContent`'s glass skin -- background, blur, border, padding
// -- is stripped off. That keeps the count at four glass surfaces rather than
// introducing a fifth.
//
// The `overlay` elevation is deliberately LEFT on the popover surface rather
// than moved onto the deed. With no padding and no border the surface's border
// box is exactly the deed's box -- measured identical at [343,300,256,371] --
// so one shadow lands in the right place. Putting it on both drew the step
// twice and read darker than an `overlay`: `shadow-none` here does NOT cancel
// `shadow-overlay`, because `cn()`'s `twMerge` does not recognise a custom
// `boxShadow` key as a shadow utility and keeps both classes.
const DeedPopover = ({
  property,
  children,
}: {
  property: Property;
  children: React.ReactNode;
}) => (
  <Popover>
    <PopoverTrigger asChild>{children}</PopoverTrigger>
    <PopoverContent
      className={`w-auto border-0 bg-transparent p-0 backdrop-blur-none`}
    >
      <PropertyCard property={property} className={``} />
    </PopoverContent>
  </Popover>
);

// A player's whole holding. Here the glass panel IS the popover surface and the
// deeds sit on it as paper, which is the same rule read the other way round:
// the deed is never glassed, the surface under it is. Taller than the gap to the
// viewport edge and it scrolls, using the height Radix measures for us rather
// than a guessed `max-h`.
//
// The width is two deeds plus the scrollbar, measured rather than reasoned:
// a deed is `w-64` (256px) and the gap is 16px, so the row needs 528px, but
// `overflow-y-auto` takes ~7px off the content box and `w-[35rem]` (560px, less
// 32px of padding) left only 521px -- seven short, so every deed wrapped onto
// its own row. 37rem leaves 560px of content box and fits the pair with room
// for a wider scrollbar.
const DeedListPopover = ({
  properties,
  children,
}: {
  properties?: Property[];
  children: React.ReactNode;
}) => (
  <Popover>
    <PopoverTrigger asChild>{children}</PopoverTrigger>
    <PopoverContent
      className={`max-h-[var(--radix-popover-content-available-height)] w-[37rem] overflow-y-auto`}
    >
      {properties && properties.length > 0 ? (
        <div className={`flex flex-wrap justify-center gap-4`}>
          {properties.map((property: Property) => (
            // Keyed on `id`, never on index: every refetch replaces the room
            // object whole, so each derived array is a new identity on every
            // socket message (PLAN.md section 5).
            <PropertyCard key={property.id} property={property} />
          ))}
        </div>
      ) : (
        // The string `manage-properties.tsx:79` already uses for this state,
        // reused rather than reinvented.
        <p className={`text-center font-semibold`}>No Properties Found</p>
      )}
    </PopoverContent>
  </Popover>
);

export { DeedPopover, DeedListPopover };
