// Every DrawerContent in the app picked its own height ad hoc -- h-[80vh],
// h-[75vh], h-[70vh], h-[600px], h-[90vh] and min-h-[90vh], with no
// relationship to each other. Three sizes cover every real case; pick the
// closest fit rather than adding a fourth. These are full Tailwind arbitrary-
// value class strings, not raw magnitudes, because the JIT scanner needs the
// literal class name somewhere in a scanned file -- a template literal built
// from a bare number would not generate any CSS.
export const DRAWER_HEIGHT_COMPACT = "h-[70vh]";
export const DRAWER_HEIGHT_STANDARD = "h-[80vh]";
export const DRAWER_HEIGHT_TALL = "h-[90vh]";
export const DRAWER_MIN_HEIGHT_TALL = "min-h-[90vh]";
