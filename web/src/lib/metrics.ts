import { useCallback, useEffect, useRef, useState } from "react";

export interface GatewayRaw {
  totalRequests: number;
  errorRequests: number;
  timelineReads: number;
  postWrites: number;
  likes: number;
  inFlight: number;
  histogramCount: number;
  buckets: Map<number, number>;
}

export interface TrafficSample {
  at: number;
  reqPerSec: number;
  p95Ms: number | null;
  errorPct: number;
  inFlight: number;
  timelinePerSec: number;
  writesPerSec: number;
  likesPerSec: number;
}

export type TrafficStatus = "loading" | "ok" | "error";

export interface TrafficState {
  status: TrafficStatus;
  sample: TrafficSample | null;
  history: number[];
  lastSuccessAt: number | null;
  error: string | null;
}

const HISTORY_LENGTH = 60;
const POLL_MS = 5000;

const LINE_RE =
  /^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{([^}]*)\})?\s+([-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)$/;

const LABEL_RE = /([a-zA-Z_][a-zA-Z0-9_]*)="((?:[^"\\]|\\.)*)"/g;

function parseLabels(raw: string | undefined): Record<string, string> {
  const labels: Record<string, string> = {};
  if (!raw) return labels;
  let match: RegExpExecArray | null;
  LABEL_RE.lastIndex = 0;
  while ((match = LABEL_RE.exec(raw)) !== null) {
    labels[match[1]] = match[2]
      .replace(/\\"/g, '"')
      .replace(/\\\\/g, "\\");
  }
  return labels;
}

function isExcludedRoute(route: string): boolean {
  return route === "/metrics" || route === "/healthz";
}

export function parseGatewayMetrics(text: string): GatewayRaw {
  const raw: GatewayRaw = {
    totalRequests: 0,
    errorRequests: 0,
    timelineReads: 0,
    postWrites: 0,
    likes: 0,
    inFlight: 0,
    histogramCount: 0,
    buckets: new Map<number, number>(),
  };

  for (const line of text.split("\n")) {
    if (!line || line.charCodeAt(0) === 35) continue; // "#": HELP / TYPE
    const match = LINE_RE.exec(line);
    if (!match) continue;

    const name = match[1];
    const value = Number(match[3]);
    if (!Number.isFinite(value)) continue;

    if (name === "wayposts_http_requests_total") {
      const labels = parseLabels(match[2]);
      const route = labels.route ?? "";
      if (isExcludedRoute(route)) continue;
      raw.totalRequests += value;
      if ((labels.status ?? "").startsWith("5")) raw.errorRequests += value;
      if (labels.method === "GET" && route.startsWith("/v1/timeline")) {
        raw.timelineReads += value;
      }
      if (labels.method === "POST" && route === "/v1/posts") {
        raw.postWrites += value;
      }
      if (labels.method === "POST" && route.startsWith("/v1/posts/")) {
        raw.likes += value;
      }
      continue;
    }

    if (name === "wayposts_http_request_duration_seconds_bucket") {
      const labels = parseLabels(match[2]);
      if (isExcludedRoute(labels.route ?? "")) continue;
      const le = Number(labels.le);
      if (!Number.isFinite(le)) continue;
      raw.buckets.set(le, (raw.buckets.get(le) ?? 0) + value);
      continue;
    }

    if (name === "wayposts_http_request_duration_seconds_count") {
      const labels = parseLabels(match[2]);
      if (isExcludedRoute(labels.route ?? "")) continue;
      raw.histogramCount += value;
      continue;
    }

    if (name === "wayposts_http_in_flight") {
      raw.inFlight += value;
    }
  }

  return raw;
}

function percentileFromBuckets(
  buckets: Map<number, number>,
  total: number,
): number | null {
  if (total <= 0) return null;
  const target = total * 0.95;
  const sorted = [...buckets.entries()].sort((a, b) => a[0] - b[0]);
  for (const [le, cumulative] of sorted) {
    if (cumulative >= target) return le * 1000;
  }
  const last = sorted.at(-1);
  return last ? last[0] * 1000 : null;
}

function percentile95ForInterval(
  previous: GatewayRaw | null,
  next: GatewayRaw,
): number | null {
  if (!previous) return null;

  const buckets = new Map<number, number>();
  const les = new Set([...previous.buckets.keys(), ...next.buckets.keys()]);
  for (const le of les) {
    const after = next.buckets.get(le) ?? 0;
    const before = previous.buckets.get(le) ?? 0;
    const delta = Math.max(0, after - before);
    if (delta > 0) buckets.set(le, delta);
  }

  const total = Math.max(0, next.histogramCount - previous.histogramCount);
  return percentileFromBuckets(buckets, total);
}

export function deriveSample(
  previous: { at: number; raw: GatewayRaw } | null,
  next: GatewayRaw,
  at: number,
): TrafficSample {
  const seconds = previous ? Math.max(0.5, (at - previous.at) / 1000) : 0;
  const delta = (before: number, after: number): number =>
    Math.max(0, after - before);

  const requests = previous
    ? delta(previous.raw.totalRequests, next.totalRequests)
    : 0;
  const errors = previous
    ? delta(previous.raw.errorRequests, next.errorRequests)
    : 0;

  return {
    at,
    reqPerSec: seconds > 0 ? requests / seconds : 0,
    p95Ms: percentile95ForInterval(previous?.raw ?? null, next),
    errorPct: requests > 0 ? (errors / requests) * 100 : 0,
    inFlight: next.inFlight,
    timelinePerSec: seconds > 0
      ? delta(previous?.raw.timelineReads ?? 0, next.timelineReads) / seconds
      : 0,
    writesPerSec: seconds > 0
      ? delta(previous?.raw.postWrites ?? 0, next.postWrites) / seconds
      : 0,
    likesPerSec: seconds > 0
      ? delta(previous?.raw.likes ?? 0, next.likes) / seconds
      : 0,
  };
}

export async function fetchGatewayMetrics(
  apiBase: string,
  signal: AbortSignal,
): Promise<GatewayRaw> {
  const response = await fetch(`${apiBase}/metrics`, {
    signal,
    headers: { Accept: "text/plain" },
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}`);
  }
  return parseGatewayMetrics(await response.text());
}

const initialState: TrafficState = {
  status: "loading",
  sample: null,
  history: [],
  lastSuccessAt: null,
  error: null,
};

export function useTraffic(apiBase: string): TrafficState & {
  retry: () => void;
} {
  const [state, setState] = useState<TrafficState>(initialState);
  const previous = useRef<{ at: number; raw: GatewayRaw } | null>(null);
  const busy = useRef(false);
  const mounted = useRef(true);
  const controller = useRef<AbortController | null>(null);

  const load = useCallback(async () => {
    if (busy.current) return;
    busy.current = true;
    const at = Date.now();
    controller.current?.abort();
    const nextController = new AbortController();
    controller.current = nextController;
    const timeout = window.setTimeout(() => nextController.abort(), 8000);

    try {
      const raw = await fetchGatewayMetrics(apiBase, nextController.signal);
      if (!mounted.current) return;
      const sample = deriveSample(previous.current, raw, at);
      previous.current = { at, raw };
      setState((before) => ({
        status: "ok",
        sample,
        history: [...before.history, sample.reqPerSec].slice(-HISTORY_LENGTH),
        lastSuccessAt: at,
        error: null,
      }));
    } catch (cause) {
      if (!mounted.current) return;
      setState((before) => ({
        ...before,
        status: "error",
        error: cause instanceof Error ? cause.message : "Metrics request failed",
      }));
    } finally {
      window.clearTimeout(timeout);
      busy.current = false;
    }
  }, [apiBase]);

  useEffect(() => {
    mounted.current = true;
    void load();
    const timer = window.setInterval(() => void load(), POLL_MS);
    return () => {
      mounted.current = false;
      window.clearInterval(timer);
      controller.current?.abort();
    };
  }, [load]);

  const retry = useCallback(() => {
    setState((before) => ({
      ...before,
      status: before.sample ? "ok" : "loading",
      error: null,
    }));
    void load();
  }, [load]);

  return { ...state, retry };
}
