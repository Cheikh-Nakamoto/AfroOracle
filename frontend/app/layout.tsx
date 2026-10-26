import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "AfriOracle",
  description: "Prix du mil à Kaolack — démo",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="fr">
      <body style={{ margin: 0 }}>{children}</body>
    </html>
  );
}
