// One money format, everywhere (DESIGN.md D3(a)).
//
// Before this there were two. The player card rendered `$1875`
// (`components/players/player-card-content.tsx:161`, a bare template literal)
// and the offer panel rendered `1,500` (`components/players/make-offer/amount.tsx:61`,
// `balance.toLocaleString()`) -- the same kind of number, two shapes, on two
// screens a player moves between while deciding what to send.
//
// Grouped and prefixed is the union of the two rather than a third option: the
// separator is what makes a five-figure balance readable at a glance, and the
// `$` is what says money rather than a count. Nothing is lost from either side.
//
// The locale is pinned. `toLocaleString()` with no argument resolves against the
// runtime's locale, which is the server's during SSR and the browser's after
// hydration -- a real mismatch, not a theoretical one, since `amount.tsx` is a
// client component that server-renders.
//
// Amounts are whole dollars. The Go side refuses a fraction rather than
// truncating it (`parseTradeSide`), which is why `amount.tsx` already floors
// every percent share, so there are no cents to show.
export const formatMoney = (amount?: number | null): string => {
  const whole = Math.trunc(amount ?? 0);
  const sign = whole < 0 ? "-" : "";
  return `${sign}$${Math.abs(whole).toLocaleString("en-US")}`;
};
