import { josephinBold, numeralFace } from "../ui/fonts";
import { InputHTMLAttributes } from "react";

interface BaseInputProps extends InputHTMLAttributes<HTMLInputElement> {
  className?: string;
  // One input component serves both jobs: a room code and a starting cash
  // figure are numerals, a room name and a player name are words. Two font
  // classes on one element is a specificity tie, so the caller says which
  // rather than appending one through `className` (DESIGN.md D3(a)).
  numeral?: boolean;
}

export const BaseRoomInput = ({
  className = "",
  numeral = false,
  ...props
}: BaseInputProps) => {
  return (
    <input
      {...props}
      className={`${
        numeral ? numeralFace : josephinBold.className
      } border p-4 bg-inherit rounded text-2xl w-64 mt-4 ${className}`}
    />
  );
};
