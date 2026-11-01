import { formatDate } from "./format";
export type Publication = {
  median: number;
  confidence: number;
  validated?: boolean;
  tx?: string;
  chain?: string;
  aiReportHash?: string;
  timestamp?: number;
};

export function latestPublication(pubs: Publication[] | undefined | null): Publication | null {
  return pubs && pubs.length > 0 ? pubs[pubs.length - 1] : null;
}

/** View model: a result is "published" only when validated === true. */
export function publicationView(p: Publication) {
  const published = p.validated === true;
  return {
    published,
    title: published ? "Publié" : "Non publié (consensus non validé)",
    tx: published && p.tx ? p.tx : "—",
    chain: published && p.chain ? p.chain : "—",
    when: formatDate(p.timestamp),
    color: published ? "#1b5e20" : "#b71c1c",
    background: published ? "#e8f5e9" : "#fdecea",
  };
}
