// acceptance/pending.txt, read as by acceptance/pending.go (ADR 0028, point 4): one key per
// line, "<file>.feature: <scenario title>", blank lines and # comments ignored.

/** Reads the keys of the pending list. */
export function parsePending(text: string): Set<string> {
  const keys = new Set<string>();
  text.split("\n").forEach((raw, i) => {
    const line = raw.trim();
    if (line === "" || line.startsWith("#")) {
      return;
    }
    if (!line.includes(".feature: ")) {
      throw new Error(`pending.txt:${i + 1}: "${line}" is not "<file>.feature: <scenario title>"`);
    }
    if (keys.has(line)) {
      throw new Error(`pending.txt:${i + 1}: "${line}" listed twice`);
    }
    keys.add(line);
  });
  return keys;
}
