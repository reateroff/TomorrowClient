// Maps server/location names to an ISO 3166-1 alpha-2 country code, which the
// UI renders as an SVG flag (country-flag-icons). Server names from
// subscriptions usually contain a country name, an ISO code, or a city, e.g.
// "🇩🇪 Germany 01", "DE-Frankfurt", "Netherlands #3". We try, in order: an
// existing flag emoji, a keyword, then a standalone two-letter ISO code.

// Regional-indicator flag emoji → ISO code (reverse of the codepoint math).
function emojiToCode(emoji: string): string {
  const cps = Array.from(emoji).map((c) => c.codePointAt(0) ?? 0);
  if (cps.length !== 2) return "";
  const base = 0x1f1e6;
  const a = cps[0] - base + 65;
  const b = cps[1] - base + 65;
  if (a < 65 || a > 90 || b < 65 || b > 90) return "";
  return String.fromCharCode(a) + String.fromCharCode(b);
}

// Keyword → ISO code. Covers common VPN locations plus a few big cities.
const KEYWORDS: Record<string, string> = {
  germany: "DE", deutschland: "DE", frankfurt: "DE", munich: "DE",
  netherlands: "NL", holland: "NL", amsterdam: "NL",
  france: "FR", paris: "FR",
  "united kingdom": "GB", britain: "GB", england: "GB", london: "GB", uk: "GB",
  "united states": "US", usa: "US", america: "US", "new york": "US", miami: "US", "los angeles": "US",
  finland: "FI", helsinki: "FI",
  sweden: "SE", stockholm: "SE",
  russia: "RU", moscow: "RU",
  japan: "JP", tokyo: "JP",
  singapore: "SG",
  "hong kong": "HK",
  turkey: "TR", istanbul: "TR",
  poland: "PL", warsaw: "PL",
  spain: "ES", madrid: "ES",
  italy: "IT", milan: "IT",
  canada: "CA", toronto: "CA",
  switzerland: "CH", zurich: "CH",
  austria: "AT", vienna: "AT",
  ukraine: "UA", kyiv: "UA",
  "united arab emirates": "AE", dubai: "AE", emirates: "AE",
  india: "IN", mumbai: "IN",
  australia: "AU", sydney: "AU",
  korea: "KR", seoul: "KR",
  china: "CN",
  brazil: "BR",
  norway: "NO",
  denmark: "DK",
  ireland: "IE",
  luxembourg: "LU",
  latvia: "LV",
  estonia: "EE",
  czech: "CZ",
  romania: "RO",
  bulgaria: "BG",
  serbia: "RS",
  kazakhstan: "KZ",
  argentina: "AR",
  mexico: "MX",
};

// Standalone two-letter tokens that are ISO codes on their own.
const CODES = new Set(
  Object.values(KEYWORDS).concat([
    "DE", "NL", "FR", "GB", "US", "FI", "SE", "RU", "JP", "SG", "HK", "TR",
    "PL", "ES", "IT", "CA", "CH", "AT", "UA", "AE", "IN", "AU", "KR", "CN",
    "BR", "NO", "DK", "IE", "LU", "LV", "EE", "CZ", "RO", "BG", "RS", "KZ",
    "AR", "MX",
  ])
);

// countryCodeFor returns an ISO alpha-2 code for a server name, or "" when none
// is detected.
export function countryCodeFor(name: string): string {
  if (!name) return "";

  // 1. Already contains a flag emoji (regional indicator pair) — decode it.
  const existing = name.match(/[\u{1F1E6}-\u{1F1FF}]{2}/u);
  if (existing) {
    const cc = emojiToCode(existing[0]);
    if (cc) return cc;
  }

  const lower = name.toLowerCase();

  // 2. Keyword match (longest keyword first for multi-word names).
  const kw = Object.keys(KEYWORDS).sort((a, b) => b.length - a.length);
  for (const k of kw) {
    if (lower.includes(k)) return KEYWORDS[k];
  }

  // 3. Standalone two-letter token, e.g. "DE-01" or "US 3".
  for (const tok of name.toUpperCase().split(/[^A-Z]+/)) {
    if (CODES.has(tok)) return tok;
  }

  return "";
}
