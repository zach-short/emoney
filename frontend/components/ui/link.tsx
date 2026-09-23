import NextLink, {LinkProps} from "next/link"

import { AnchorHTMLAttributes } from "react";

type ButtonLinkProps = LinkProps &
  AnchorHTMLAttributes<HTMLAnchorElement> & {
    className?: string;
    children?: React.ReactNode;
    tone?: Tone;
  };

// D2 keeps #ffff00 as identity only: the wordmark, and the primary action on
// `/`, `/create` and `/join`. This component is that primary action on `/`,
// so `identity` stays its default -- but it is also used for the two links on
// `/my-rooms`, which is not one of the three. Rather than collapse the button
// languages (explicitly NOT this phase's job -- "Rules that survive
// unchanged" #9), the off-brand call sites opt out with `tone="plain"`.
export type Tone = "identity" | "plain";

export const toneClasses = (tone: Tone) =>
  tone === "identity" ? "font border-yellow-200" : "border-border";

const Link = ({
  className = "",
  children,
  tone = "identity",
  ...props
}: ButtonLinkProps) => {
  return (
    <NextLink
      {...props}
      className={`border ${toneClasses(tone)} rounded-lg px-4 py-5 w-64 text-2xl font-semibold ${className}`}
    >
      {children}
    </NextLink>
  );
};

export default Link;
