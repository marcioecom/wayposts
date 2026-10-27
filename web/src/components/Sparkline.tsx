interface SparklineProps {
  values: number[];
  label: string;
}

const WIDTH = 260;
const HEIGHT = 56;
const PAD = 4;

export function Sparkline({ values, label }: SparklineProps) {
  const series = values.length > 0 ? values : [0];
  const plotted = series.length === 1 ? [series[0], series[0]] : series;
  const max = Math.max(1, ...plotted);
  const stepX = (WIDTH - PAD * 2) / Math.max(1, plotted.length - 1);

  const points = plotted.map((value, index) => {
    const x = PAD + index * stepX;
    const y = HEIGHT - PAD - (value / max) * (HEIGHT - PAD * 2);
    return [x, y] as const;
  });

  const line = points
    .map(([x, y], index) => `${index === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`)
    .join(" ");
  const first = points[0];
  const last = points[points.length - 1];
  const area = `${line} L${last[0].toFixed(1)} ${HEIGHT} L${first[0].toFixed(1)} ${HEIGHT} Z`;

  return (
    <svg
      className="sparkline"
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      preserveAspectRatio="none"
      role="img"
      aria-label={label}
    >
      <path className="sparkline-area" d={area} />
      <path className="sparkline-line" d={line} />
      <circle
        className="sparkline-dot"
        cx={last[0]}
        cy={last[1]}
        r={2.6}
      />
    </svg>
  );
}
