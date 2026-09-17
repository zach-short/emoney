import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "E-Money",
    short_name: "E-Money",
    description: `Money isn't real, but it doesn't have to be`,
    start_url: "/",
    display: "standalone",
    background_color: "#000",
    theme_color: "#000000",
    icons: [
      {
        src: "/logo192.png",
        sizes: "192x192",
        type: "image/png",
        purpose: "any",
      },
      {
        src: "/logo384.png",
        sizes: "384x384",
        type: "image/png",
        purpose: "any",
      },
      {
        src: "/logo512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "any",
      },
      // Android crops these to its own adaptive shape, so the mark is sized to
      // fit the 80% safe circle and the black ground runs to every edge. They
      // stay separate from the "any" set above: a single icon declared for both
      // purposes has to use the padded artwork everywhere.
      {
        src: "/logo-maskable-192.png",
        sizes: "192x192",
        type: "image/png",
        purpose: "maskable",
      },
      {
        src: "/logo-maskable-512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "maskable",
      },
    ],
  };
}
