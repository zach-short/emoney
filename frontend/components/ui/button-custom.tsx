"use client";
import { ButtonHTMLAttributes } from "react";
import { type Tone, toneClasses } from "./link";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  className?: string;
  tone?: Tone;
};

// The primary action on `/create` and `/join`, which is why `identity` is the
// default (D2). `/install` and the error fallback also use it and are not
// among the three allowed places, so they pass `tone="plain"`.
const Button = ({ className = "", tone = "identity", ...props }: ButtonProps) => {
  return (
    <button
      className={`border ${toneClasses(tone)} rounded-sm px-4 py-5 w-64 text-2xl font-semibold ${className}`}
      {...props}
    />
  );
};

export default Button;
