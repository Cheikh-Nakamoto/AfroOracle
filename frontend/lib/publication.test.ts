import { describe, expect, it } from "vitest";
import { latestPublication, publicationView } from "./publication";

const pub = { median: 187, confidence: 100, validated: true, tx: "0xabc", chain: "simulated", timestamp: 1700000000 };

describe("publication view", () => {
  it("picks the last publication", () => {
    expect(latestPublication([{ ...pub, median: 1 }, pub])?.median).toBe(187);
    expect(latestPublication([])).toBeNull();
    expect(latestPublication(undefined)).toBeNull();
  });
  it("marks validated as published", () => {
    const v = publicationView(pub);
    expect(v.published).toBe(true);
    expect(v.title).toBe("Publié");
    expect(v.tx).toBe("0xabc");
  });
  it("marks unvalidated as not published and hides tx", () => {
    const v = publicationView({ ...pub, validated: false, tx: "" });
    expect(v.published).toBe(false);
    expect(v.title).toMatch(/Non publié/);
    expect(v.tx).toBe("—");
  });
  it("treats a missing validated flag as not published", () => {
    expect(publicationView({ ...pub, validated: undefined }).published).toBe(false);
  });
});
