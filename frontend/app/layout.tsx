import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "AfriOracle — Prix du mil à Kaolack",
  description: "Observations locales, prix consensuel et publication vérifiable pour le marché du mil à Kaolack.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="fr">
      <body>{children}</body>
    </html>
  );
}
