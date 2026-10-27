---
name: Wayposts Lab
description: A live social feed and a view of the architecture under load.
colors:
  canvas: "oklch(0.972 0.002 25)"
  surface: "oklch(1 0 0)"
  surface-low: "oklch(0.985 0.003 25)"
  hover: "oklch(0.965 0.004 25)"
  hairline: "oklch(0.905 0.005 25)"
  hairline-strong: "oklch(0.84 0.008 25)"
  ink: "oklch(0.19 0.012 25)"
  ink-muted: "oklch(0.46 0.014 25)"
  ink-faint: "oklch(0.55 0.012 25)"
  signal: "oklch(0.55 0.17 35)"
  signal-deep: "oklch(0.49 0.17 35)"
  signal-ink: "oklch(0.46 0.16 35)"
  signal-wash: "oklch(0.96 0.025 35)"
  signal-line: "oklch(0.86 0.06 35)"
  live: "oklch(0.46 0.12 155)"
  live-wash: "oklch(0.95 0.03 155)"
  danger: "oklch(0.47 0.19 22)"
  danger-wash: "oklch(0.96 0.03 22)"
  danger-line: "oklch(0.86 0.06 22)"
  graphite: "oklch(0.19 0.008 25)"
  graphite-raised: "oklch(0.235 0.009 25)"
  graphite-ink: "oklch(0.95 0.004 25)"
  graphite-muted: "oklch(0.72 0.008 25)"
  graphite-faint: "oklch(0.62 0.008 25)"
  graphite-line: "oklch(0.31 0.008 25)"
  graphite-signal: "oklch(0.74 0.15 38)"
  graphite-live: "oklch(0.75 0.14 155)"
typography:
  display:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "22px"
    fontWeight: 700
    lineHeight: 1.15
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "20px"
    fontWeight: 700
    lineHeight: 1.15
    letterSpacing: "-0.02em"
  title:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "14px"
    fontWeight: 600
    lineHeight: 1.25
    letterSpacing: "normal"
  body:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.55
    letterSpacing: "normal"
  label:
    fontFamily: "Archivo, ui-sans-serif, system-ui, sans-serif"
    fontSize: "11px"
    fontWeight: 600
    lineHeight: 1.4
    letterSpacing: "normal"
  data:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, monospace"
    fontSize: "13px"
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: "normal"
rounded:
  sm: "6px"
  md: "10px"
  lg: "12px"
  xl: "14px"
  sheet: "16px"
  pill: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
  xl: "32px"
components:
  button-primary:
    backgroundColor: "{colors.signal}"
    textColor: "{colors.surface}"
    rounded: "{rounded.pill}"
    padding: "0 14px"
    height: "36px"
  button-primary-hover:
    backgroundColor: "{colors.signal-deep}"
    textColor: "{colors.surface}"
    rounded: "{rounded.pill}"
    padding: "0 14px"
    height: "36px"
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.pill}"
    padding: "0 14px"
    height: "36px"
  nav-item-active:
    backgroundColor: "{colors.signal-wash}"
    textColor: "{colors.signal-ink}"
    rounded: "{rounded.md}"
    padding: "0 12px"
    height: "44px"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "0 11px"
    height: "38px"
  live-pill:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink-muted}"
    rounded: "{rounded.pill}"
    padding: "0 10px"
    height: "30px"
  traffic-panel:
    backgroundColor: "{colors.graphite-raised}"
    textColor: "{colors.graphite-ink}"
    rounded: "{rounded.lg}"
    padding: "14px"
  post:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    padding: "16px 20px 12px"
---

# Design System: Wayposts Lab

## 1. Overview

**Creative North Star: "The Feed and the Instrument"**

Wayposts Lab is two materials in one frame. The center is a real social feed: paper-white, quiet, familiar, built for reading and posting. The right column is the instrument: graphite, dense, exact, built for watching a live system. The separation is the idea. A visitor should be able to enjoy the feed without thinking about infrastructure, then look right and see the same moment measured in req/s, p95, errors, and in-flight requests.

The interface is deliberately *not* a dashboard with a feed bolted on. The feed is the product; the instrument is the evidence. That is why the timeline has no cards, no glass, no glow, and no decorative charts. Metrics are real numbers from the gateway's `/metrics` endpoint. If the generator is not running, the panel says so and gives the command to start it.

The voice is precise, alive, and unpretentious: a good engineer showing the thing work. The system explicitly rejects the saturated AI defaults — **SaaS purple with glow**, **corporate blue dashboards**, **Matrix terminals**, and **AI landing pages**. Identity comes from material contrast, typography, and one confident accent, not from effects.

**Key Characteristics:**
- Three-column social shell: navigation, feed, instrument rail.
- Hairlines instead of cards; one white feed against one graphite rail.
- One accent, Signal Vermilion, used for action and selection only.
- Archivo for the product; IBM Plex Mono for machine data only.
- Flat at rest; depth comes from material and modal backdrop, never from drop shadows.
- Real telemetry, honest empty states, complete error and loading coverage.
- English and Brazilian Portuguese, persisted per visitor.

## 2. Colors

A chalk-white product surface, an ink-dark instrument, and a single vermilion signal. The palette is restrained by design: the accent should feel like a mark made by hand, not a theme poured over the page.

### Primary
- **Signal Vermilion** (`oklch(0.55 0.17 35)`): the only brand accent. Primary buttons, the active navigation pill, the active tab underline, links, likes, the sparkline, and the focus state. White text on it reaches 5.26:1; it reaches 5.26:1 on white as text.
- **Signal Deep** (`oklch(0.49 0.17 35)`): hover and pressed state for primary actions.
- **Signal Ink** (`oklch(0.46 0.16 35)`): accent-colored text on white, where a darker value is needed for 4.5:1.
- **Signal Wash** (`oklch(0.96 0.025 35)`): selected navigation, like hover, and the new-post flash. Never used as a page background.
- **Signal Line** (`oklch(0.86 0.06 35)`): the 1px edge on the "new posts" banner and the live pill.

### Secondary
- **Instrument Graphite** (`oklch(0.19 0.008 25)`): the right rail, the sheet, and the toast. It is a material, not a dark mode.
- **Graphite Raised** (`oklch(0.235 0.009 25)`): the traffic panel inside the rail.
- **Graphite Signal** (`oklch(0.74 0.15 38)`): data accent on graphite; sparkline, metric values, links, and the stale state. 7.58:1 on the rail.

### Tertiary
- **Live Green** (`oklch(0.46 0.12 155)` light / `oklch(0.75 0.14 155)` on graphite): live WebSocket and live traffic. Always paired with a text label; the dot is never the only signal.
- **Danger Ember** (`oklch(0.47 0.19 22)`): destructive affordance, timeline failures. Its wash is used for the error block; it is never used as a decorative accent.

### Neutral
- **Chalk Canvas** (`oklch(0.972 0.002 25)`): the page behind the three columns.
- **True White** (`oklch(1 0 0)`): the feed surface and fields.
- **Hairline** (`oklch(0.905 0.005 25)`): every separator. This is the primary structural device.
- **Ink** (`oklch(0.19 0.012 25)`): body and headings. 18.5:1 on white.
- **Muted Ink** (`oklch(0.46 0.014 25)`): secondary copy, handles, labels. 7.17:1 on white.
- **Faint Ink** (`oklch(0.55 0.012 25)`): tertiary metadata, IDs, counters. 4.88:1 on white, still AA for small text.

### Named Rules
**The Two Materials Rule.** The feed is white. The instrument is graphite. They never blend, glass over each other, or borrow each other's text colors. The seam is the story.

**The One Accent Rule.** Signal Vermilion covers less than 10% of any view. Its rarity is why it reads as action.

**The No Glow Rule.** No gradients on text, no colored glows, no glassmorphism, no decorative shadow. If an element needs attention, it gets weight, space, or a solid wash.

## 3. Typography

**Display Font:** Archivo (with ui-sans-serif, system-ui fallbacks)
**Body Font:** Archivo (same family, multiple weights)
**Label/Mono Font:** IBM Plex Mono (with ui-monospace, SFMono-Regular fallbacks)

**Character:** Archivo has the slightly condensed, utilitarian confidence of a technical publication; it can carry a feed without feeling like a default UI font. IBM Plex Mono is reserved for the machine's voice: IDs, timestamps, metrics, commands, and cursor values. The pairing is one humanist-grotesque family plus one mono, not two competing sans-serifs.

### Hierarchy
- **Display** (700, 22px, 1.15, -0.02em): profile names and the largest identity moments. Never a marketing hero.
- **Headline** (700, 20px, 1.15, -0.02em): page title and dialog titles.
- **Title** (600, 14px, 1.25): post author names, rail section labels, button labels.
- **Body** (400, 15px, 1.55): post copy, prose, composer. Post body is capped at 62ch.
- **Label** (600, 11px, 1.4): field labels, hints, small actions. Sentence case, never a tracked uppercase eyebrow.
- **Data** (IBM Plex Mono, 500, 13px, 1.4): metrics, IDs, timestamps, commands. The large rate uses 38px mono.

### Named Rules
**The Mono-Is-Machine Rule.** If a human wrote it, it is Archivo. If the system measured it, identified it, or produced it, it is IBM Plex Mono. Mono never appears in a heading, button, or paragraph.

**The Quiet Scale Rule.** Product type has a tight ratio. No clamp-driven display sizes, no oversized hero numbers except the one live rate that is the instrument's primary reading.

## 4. Elevation

The system is flat by default. Structure comes from hairlines, white and graphite materials, and the viewport's own scroll. There is no shadow vocabulary on cards, buttons, posts, or the rail. The language toggle's 1px inset is the only near-shadow, and it exists to distinguish a selected segment, not to lift a surface.

Depth is reserved for the top layer. The identity dialog and the mobile/tablet sheets are native `<dialog>` elements with a solid, unblurred backdrop. The dialog panel uses a 1px hairline and a 14px radius; it does not use a drop shadow. The sheet's graphite background is the depth cue. The toast is a graphite slab with no shadow.

### Named Rules
**The Hairline Rule.** If two regions need separation, draw a 1px line. Do not reach for a card, a shadow, or a darker panel unless the region is the instrument rail itself.

**The Top-Layer Rule.** Only true modal surfaces (identity, live panels, you panels) get to float. Everything else stays in the document flow.

## 5. Components

### Buttons
- **Shape:** full pill (`999px`), 36px tall; small variant 30px.
- **Primary:** Signal Vermilion background, white text, 14px horizontal padding. Hover deepens to Signal Deep; disabled drops to 45% opacity.
- **Secondary:** True White with a Hairline Strong border and Ink text. Hover fills with Hover chalk.
- **Rail secondary:** transparent with Graphite Line border and Graphite Ink text. Used inside the instrument rail only.
- **Icon buttons:** 36px circle, transparent at rest, Hover chalk on hover, 2px accent focus ring.

### Navigation
- **Desktop rail:** 44px rows, 10px radius, icon plus label. Active row is Signal Wash with Signal Ink text. The active state is a filled pill, not a left stripe.
- **Feed tabs:** text buttons with a 2px Signal Vermilion underline on the current view. On mobile the underline is a centered 28px marker.
- **Mobile chrome:** 54px sticky top bar with the brand, refresh, live rate pill, and avatar. A 56px bottom navigation carries Home, Profile, Live, and You. Safe-area padding is respected.

### Composer
A borderless writing surface inside the feed: 40px avatar, "Publishing as @handle", a 16px textarea with no box, a mono character counter, and a pill Publish button. Focus is shown by the composer's bottom hairline turning Signal Line. `⌘↵` / `Ctrl↵` publishes.

### Post
- **Shape:** no card. 16px 20px 12px padding, 1px Hairline bottom border, white background. Hover shifts to Chalk Hover.
- **Author:** 40px avatar, 14px name, 12px handle and relative time. The whole author block opens the profile.
- **Body:** 15px, 1.55, max 62ch, `overflow-wrap: anywhere` and `pre-wrap`.
- **Actions:** 32px pill buttons for like, profile, and delete. Like uses Signal Vermilion when active and a 220ms pop. The full post ID sits at the far right in 10px mono, truncated with a title tooltip.
- **New post:** a 2.6s wash animation from Signal Wash back to white. No layout animation.

### Inputs / Fields
- **Style:** 38px, True White, 1px Hairline Strong, 10px radius, 13px text (16px on mobile to avoid iOS zoom).
- **Focus:** Signal Vermilion border plus a 3px Signal Wash ring.
- **Dark sheet:** Graphite background, Graphite Line border, Graphite Ink text, Graphite Signal focus ring.
- **Errors:** Danger Ember text below the field; invalid state is also `aria-invalid`.

### Chips and Avatars
- **Chips:** pill, 30px, Hairline border. Active is Signal Wash with Signal Ink. Used for quick identities.
- **Avatars:** 40px circle (32px in the top bar, 22px in the bottom nav, 64px on profile) with a 1px dark hairline. Six restrained tones — rust, moss, denim, ochre, plum, teal — assigned by user ID. No gradients.

### Instrument Rail (signature)
- **Material:** Instrument Graphite column, 344px on desktop, with Graphite Raised traffic panel, 12px radius, 14px padding.
- **Reading:** 38px IBM Plex Mono rate, a 56px coral sparkline, then a 2-column mono grid: p95, 5xx, in flight, timeline reads/s, post writes/s, likes/s.
- **Honesty:** metrics exclude `/metrics` and `/healthz`; the footer states that fanout lag lives on the fanout service. When idle, the panel names the command to start the generator. When stale or unreachable, it says so and offers retry.
- **State:** live = green dot plus label; idle = gray; stale = coral. Color is never the only cue.

### Sheets and Dialogs
- **Desktop identity:** centered native dialog, 460px max, white panel, 14px radius, solid ink backdrop. The ID input receives focus and select-all on open.
- **Tablet/mobile live:** native dialog sheet from the right on tablet, bottom sheet on mobile (max 90dvh), Graphite background, sticky header, scrollable body.
- **Motion:** 180–260ms `cubic-bezier(0.16, 1, 0.3, 1)`. Reduced-motion users get an instant state change.
- **Focus:** visible 2px accent ring; the sheet close button uses a 1px inset ring rather than a floating halo.

### Empty and Error States
- Empty states are centered, icon-led, and teach the next action: "Follow by ID" on an empty home, "Switch identity" on an empty profile. No dashed card, no illustration.
- Timeline errors fill a Danger Wash block with a clear title, the technical detail, and a retry. Profile 404s and network failures are distinct: a missing user is not an empty timeline, and a failed load is not a missing user.

## 6. Do's and Don'ts

### Do:
- **Do** keep the feed white and the instrument rail graphite. The material contrast is the identity.
- **Do** separate feed items with 1px hairlines, not cards.
- **Do** use Signal Vermilion for action, selection, and the one sparkline — under 10% of any view.
- **Do** set metrics, IDs, timestamps, and commands in IBM Plex Mono.
- **Do** read real data from `/metrics`; label the view as gateway-only; give the start command when idle.
- **Do** show a text label with every status dot, and cover loading, empty, error, stale, and reduced-motion states.
- **Do** use 12–16px panel radii and pill buttons; keep the toast and dialogs flat.
- **Do** preserve the 62ch post measure and the 65–75ch prose measure.
- **Do** make the identity switch and the live panel available on mobile through native sheets.

### Don't:
- **Don't** use "SaaS purple with glow": violet/cyan gradients, glass cards, diffuse glows, gradient logos.
- **Don't** build a "corporate blue dashboard": gray-blue panels, identical KPI cards, decorative charts.
- **Don't** build a "Matrix terminal": black background with neon green mono everywhere. Graphite is a material, not a costume.
- **Don't** build an "AI landing page": oversized hero, gradient text, uppercase eyebrows above every section, three icon cards.
- **Don't** pair a 1px border with a wide decorative shadow on buttons or cards.
- **Don't** use card radii above 16px or nested cards.
- **Don't** use `border-left`/`border-right` thicker than 1px as a colored accent stripe.
- **Don't** use gradient text, glassmorphism, repeating stripe backgrounds, or hand-drawn SVG illustrations.
- **Don't** use tiny uppercase tracked eyebrows as section scaffolding, or numbered section markers on every section.
- **Don't** invent traffic, fake a p95, or animate a metric that did not happen.
- **Don't** use mono for paragraphs, headings, or button labels; the machine voice is data only.
- **Don't** make the live rate a hero-metric template with supporting stat chips; it is one reading in a working instrument.
