interface BrandMarkProps {
  size?: number;
}

export function BrandMark({ size = 36 }: BrandMarkProps) {
  return (
    <span
      className="brand-mark"
      style={{ width: size, height: size, borderRadius: Math.round(size * 0.28) }}
      aria-hidden="true"
    >
      <svg
        viewBox="0 0 24 24"
        width={size * 0.62}
        height={size * 0.62}
        fill="none"
        stroke="currentColor"
        strokeWidth="2.6"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M4.5 8.2 8 16l4-6 4 6 3.5-7.8" />
      </svg>
    </span>
  );
}
