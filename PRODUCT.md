# Product

## Register

product

## Users

Engineers, recruiters, and peers who open the Wayposts link. Some evaluate the
architecture. Some evaluate the author. Some are curious about a social feed that claims
to survive load. They open a link in a browser with no setup. The author also uses the
same screen daily to switch seeded identities, follow accounts, publish, like, and watch
fanout and nudge behavior.

## Product purpose

Wayposts Lab is a working social feed and a live demonstration of the architecture behind
it. A visitor should be able to use the product like a normal feed: read, post, like,
follow, and switch identity. In the same view, they see request volume, latency, error
rate, feed reads, post writes, WebSocket state, and fanout freshness. Success means a
visitor can use the feed normally and read the numbers that show it carrying volume,
without opening the docs.

## Brand personality

Direct, precise, unpretentious. The voice explains what the system does and shows the
numbers. The feed uses normal social language. The telemetry uses exact labels and
values. The interface shows real data and uses predictable controls.

## Anti-references

- SaaS purple with glow: violet and cyan gradients, glass cards, diffuse shadows,
  gradient logos.
- Corporate blue dashboard: gray-blue panels, identical KPI cards, decorative charts.
- Matrix terminal: black background, neon green monospace everywhere, hacker-film look.
- AI landing page: oversized hero, gradient text, uppercase eyebrows over every section,
  three icon cards.

## Design principles

1. **The feed is the product.** A visitor must be able to use it as a feed first. The
   architecture story is shown on the same screen, not on a separate marketing page.
2. **Make the machine visible without competing with the timeline.** Telemetry has its own
   instrument rail with a distinct material. It stays out of the feed's way.
3. **Real data or nothing.** Metrics come from the gateway's live `/metrics`. No invented
   traffic, no decorative sparkline, no fake p95.
4. **Use familiar feed controls.** Reserve mono type and exact IDs for machine data.
   Standard affordances lower the cost of entry; the data still shows this is a lab.
5. **Calm under load.** High volume should not change the layout. Show pressure in the
   numbers, not in visual noise.

## Accessibility and inclusion

WCAG 2.2 AA. Body text at least 4.5:1, large text at least 3:1. Full keyboard path for
every action, visible focus rings, native dialogs for identity and mobile panels. Status
is never carried by color alone. Live and offline states pair a dot with a label. All
motion has a `prefers-reduced-motion` alternative. The interface is available in English
and Brazilian Portuguese, persisted per visitor.
