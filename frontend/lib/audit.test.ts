import { describe, expect, it } from "vitest";
import { askPath, outcomeRows, truncateHash } from "./audit";

describe("audit", () => {
  it("truncates long hashes only", () => {
    expect(truncateHash("0x1234567890abcdef1234567890abcdef")).toBe("0x123456…abcdef");
    expect(truncateHash("abc")).toBe("abc");
    expect(truncateHash(undefined)).toBe("—");
  });
  it("encodes the typed question", () => {
    expect(askPath("  prix & mil ? ")).toBe("/api/ask?q=" + encodeURIComponent("prix & mil ?"));
    expect(askPath("   ")).toBeNull();
  });
  it("builds rows with text status, reason and channel", () => {
    const rows = outcomeRows(
      [
        { NodeID: "D", Status: "excluded", Note: "preuve rejetée (score < 50)", Reward: 0, ScoreDelta: -2, Deviation: 0, Weight: 1 },
        { NodeID: "FAO", Status: "accepted", Note: "", Reward: 0.5, ScoreDelta: 0, Deviation: 1.2, Weight: 3 },
      ],
      [
        { node_id: "D", value: 187, proof_score: 20, status: "x", reward: 0, channel: "web" },
        { node_id: "FAO", value: 190, proof_score: null, status: "x", reward: 0, channel: "api" },
      ],
    );
    expect(rows[0].badge).toBe("EXCLU");
    expect(rows[0].reason).toContain("rejetée");
    expect(rows[0].delta).toBe("-2");
    expect(rows[0].channel).toBe("web");
    expect(rows[1].badge).toBe("ACCEPTÉ");
    expect(rows[1].reason).toContain("automatique");
  });
});
