# Product

## Register

product

## Users

Engineers, recruiters, and peers who open the Wayposts link. Some are evaluating the
architecture; some are evaluating the author; some are just curious about a social feed
that claims to survive load. They arrive with a browser, no setup, and roughly two
minutes of patience. The author also uses the same surface daily to switch seeded
identities, follow accounts, publish, like, and watch fanout and nudges behave.

## Product Purpose

Wayposts Lab is a working social feed and a live demonstration of the architecture behind
it. A visitor should be able to use the product like a normal feed — read, post, like,
follow, switch identity — and, in the same view, see the machine doing its job: request
volume, latency, error rate, feed reads, post writes, WebSocket state, and fanout
freshness. Success is a visitor understanding both "this feels like a real feed" and
"this is engineered to carry volume" without reading a single paragraph of documentation.

## Brand Personality

Precise, alive, unpretentious. The voice is that of a good engineer showing you the
thing working, not pitching it. Social warmth in the feed; instrument-grade clarity in
the telemetry. Confidence comes from real data and clean interaction, never from
adjectives.

## Anti-references

- SaaS purple with glow: violet and cyan gradients, glass cards, diffuse shadows,
  gradient logos.
- Corporate blue dashboard: gray-blue panels, identical KPI cards, decorative charts.
- Matrix terminal: black background, neon green monospace everywhere, hacker-film
  theatrics.
- AI landing page: oversized hero, gradient text, uppercase eyebrows over every section,
  three icon cards.

## Design Principles

1. **The feed is the product.** A visitor must be able to use it as a feed first. The
   architecture story is told through the same surface, not a separate marketing page.
2. **Make the machine visible without stealing the stage.** Telemetry lives in a dedicated
   instrument rail with a distinct material; it never competes with the timeline for
   attention.
3. **Real data or nothing.** Metrics come from the gateway's live `/metrics`. No invented
   traffic, no decorative sparkline, no fake p95.
4. **Familiar social grammar, lab-grade precision.** Standard feed affordances lower the
   cost of entry; mono type and exact IDs signal that this is a measuring instrument.
5. **Calm under load.** High volume should look composed: stable layout, quiet motion,
   clear state. Pressure is shown in numbers, not in visual noise.

## Accessibility & Inclusion

WCAG 2.2 AA. Body text at least 4.5:1, large text at least 3:1. Full keyboard path for
every action, visible focus rings, native dialogs for identity and mobile panels. Status
is never carried by color alone: live/offline states pair a dot with a label. All motion
has a `prefers-reduced-motion` alternative. The interface is available in English and
Brazilian Portuguese, persisted per visitor.
