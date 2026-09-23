import { Josefin_Sans, JetBrains_Mono, Manrope } from "next/font/google";

// Three faces, three jobs (DESIGN.md D3, ratified 2026-09-22).
//
// Josefin Sans is the display voice and stays exactly as it was -- the wordmark,
// the property deed, headings, anything large and characterful. It is loaded as
// three static weights because 95 call sites interpolate one of them by name.
//
// Manrope is the body/UI face. D3(b) is argued on one thing: Josefin's very low
// x-height is what makes the 12px toast and the event log hard to read. Manrope
// is geometric enough to sit beside Josefin without imitating it and has a far
// taller x-height at the sizes that were failing. It is set on `<body>`
// (`app/layout.tsx`) so the system stack stops being the default.
//
// JetBrains Mono is the numeral face. Every dollar amount in this game is a
// number that changes while someone is looking at it, and proportional digits
// reflow as it changes -- `$1500 -> $1750` shifts the glyphs, which is the
// defect D3(a) exists to fix and the thing Phase 5's count-up would otherwise
// animate on top of.
//
// Both new faces are variable on Google Fonts
// (`next/dist/compiled/@next/font/dist/google/font-data.json`, read 2026-09-22)
// and they are loaded differently on purpose, measured 2026-09-22 against this
// build's basic-latin subset -- the only one an English UI ever fetches:
//
//   Manrope, no `weight`  -> one variable file, 24,576 B, every weight
//   Manrope, weight 400+600+700 -> 24,576 B *each*, 73,728 B total
//   JetBrains Mono, no `weight` -> one variable file, 40,480 B
//   JetBrains Mono, weight 500  -> one static instance, 21,876 B
//
// So Manrope stays variable (the body face is set at three weights across the
// app and the variable file is the cheapest way to have all of them), and the
// numeral face is pinned, because it needs exactly one weight and pinning it
// halves the largest asset on the page. 500 rather than 400: a mono at 400 next
// to Manrope's body copy reads thin, and the balance is the headline number on
// the card. Tailwind weight utilities work on Manrope and do NOT move the
// numeral face -- its class sets `font-weight: 500` outright, which is what
// keeps every amount in the app the same weight wherever it is nested.
const josephinBold = Josefin_Sans({
  weight: ["700"],
  subsets: ["latin"],
});
const josephinNormal = Josefin_Sans({
  weight: ["400"],
  subsets: ["latin"],
});
const josephinLight = Josefin_Sans({
  weight: ["100"],
  subsets: ["latin"],
});
const manrope = Manrope({
  subsets: ["latin"],
});
const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["500"],
});

// The numeral face as one string rather than a bare `.className`, because the
// ratified fallback if the weight measurement ever comes back bad (DESIGN.md D3
// "The argument it beat", PLAN.md section 3 dial "Numeral face") is Josefin +
// `tabular-nums` + one mono. Bundling the face and the figure setting here makes
// that dial one edit rather than one per call site. `tabular-nums` is redundant
// against a monospace face and deliberate: it is what survives the swap.
const numeralFace = `${jetbrainsMono.className} tabular-nums`;

export {
  josephinLight,
  josephinNormal,
  josephinBold,
  manrope,
  jetbrainsMono,
  numeralFace,
};
