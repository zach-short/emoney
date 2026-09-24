import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";
import { manrope } from "@/components/ui/fonts";

export const metadata: Metadata = {
  title: "E-Money",
  description: "Money isn't real, but it doesn't have to be",
};
export const viewport: Viewport = {
  themeColor: "#000",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <>
      <html lang="en">
        {/* The body/UI face (DESIGN.md D3(b)). Everything that does not name
            a face of its own inherits Manrope from here; before this the default
            was the system stack. Josefin stays the display face and is still
            interpolated explicitly where it belongs. */}
        <body className={manrope.className}>
          <main>{children}</main>
          <Toaster />
        </body>
      </html>
    </>
  );
}
