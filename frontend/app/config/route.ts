import { NextResponse } from "next/server";

// Read at request time so the API URL can change without rebuilding the image.
export const dynamic = "force-dynamic";

export function GET() {
  const token = process.env.DASHBOARD_TOKEN ?? (process.env.NODE_ENV === "production" ? "" : "dev-token-FAO");
  return NextResponse.json({
    apiUrl: process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL ?? "",
    token,
  });
}
