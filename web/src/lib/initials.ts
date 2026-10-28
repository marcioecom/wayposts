type Named = { handle?: string };

export function initials(user: Named | null | undefined, id: number): string {
  const handle = user?.handle?.trim();
  if (handle) {
    const seeded = /^u(\d+)$/i.exec(handle);
    if (seeded) {
      return seeded[1].length > 4 ? seeded[1].slice(-4) : seeded[1];
    }

    const cleaned = handle.replace(/^@/, "");
    if (cleaned.length <= 3) return cleaned.toUpperCase();
    return cleaned.slice(0, 2).toUpperCase();
  }
  const fallback = String(id);
  return fallback.length > 4 ? fallback.slice(-4) : fallback;
}

export function avatarTone(id: number): number {
  return ((Math.abs(Math.floor(id)) - 1) % 6) + 1;
}
