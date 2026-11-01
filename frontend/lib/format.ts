const LOCALE = "fr-FR";

export function formatNumber(n: number | undefined | null): string {
  return typeof n === "number" ? n.toLocaleString(LOCALE) : "—";
}

export function formatDate(ts: number | undefined | null): string {
  return ts ? new Date(ts * 1000).toLocaleString(LOCALE) : "—";
}

/** Human freshness of a publication (unix seconds), as text. */
export function freshness(ts: number | undefined | null, nowSec: number = Date.now() / 1000): string {
  if (!ts) return "";
  const d = Math.max(0, nowSec - ts);
  if (d < 60) return "à l'instant";
  if (d < 3600) return `il y a ${Math.floor(d / 60)} min`;
  if (d < 86400) return `il y a ${Math.floor(d / 3600)} h`;
  return `il y a ${Math.floor(d / 86400)} j`;
}
