"use client";
import Link from "next/link";
import {
  MdOutlineInstallDesktop,
  MdOutlineInstallMobile,
} from "react-icons/md";
import { useIsStandalone } from "@/hooks/use-browser-env";

const FloatingInstallButton = ({ className }: { className?: string }) => {
  const isStandalone = useIsStandalone();

  if (isStandalone) {
    return null;
  }

  return (
    <Link href="/install" className={`${className}`}>
      <MdOutlineInstallMobile className={`block sm:hidden text-2xl`} />
      <MdOutlineInstallDesktop className={`hidden sm:block text-2xl`} />
    </Link>
  );
};

export default FloatingInstallButton;
