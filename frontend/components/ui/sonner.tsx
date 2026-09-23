"use client"

import { Toaster as Sonner } from "sonner"
import { manrope } from "@/components/ui/fonts"

type ToasterProps = React.ComponentProps<typeof Sonner>

const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <Sonner
      // The app is dark-only (D1); there is no theme to resolve.
      theme="dark"
      // The body face has to be named here, not inherited. sonner's own stylesheet
      // sets `font-family` directly on the toaster element
      // (`:where([data-sonner-toaster])`, `node_modules/sonner/dist/styles.css:27-30`),
      // and a rule on the element always beats inheritance from `<body>` however low
      // its specificity. Measured 2026-09-22: without this the 12px room toast
      // computed `ui-sans-serif` -- the system stack -- which is the exact surface
      // D3(b) was argued on. Any class here outranks `:where()`.
      className={`toaster group ${manrope.className}`}
      // The room's notifications are top-center and were landing squarely on
      // the room name in the sticky header (h-16). Clear it.
      offset={76}
      // A burst of socket frames stacked without limit -- every refetch in
      // `app/room/[code]/page.tsx` raises one. Three is the dial (PLAN.md
      // section 3). `duration: 4000` is set per-toast at the call site and is
      // deliberately left alone. The mobile half of the offset dial is a rule
      // in `app/globals.css`, not a prop: `mobileOffset` is sonner 2.x and
      // this is 1.7.1 -- the comment there has the whole argument.
      visibleToasts={3}
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-overlay",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
      {...props}
    />
  )
}

export { Toaster }
