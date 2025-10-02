/**
 * Calculate new position for card placement.
 * Uses simple integer positioning with gaps.
 */
export function calculatePosition(
  cards: Array<{ position: number }>,
  insertIndex: number
): number {
  if (cards.length === 0) {
    return 1000;
  }

  if (insertIndex === 0) {
    return cards[0].position / 2;
  }

  if (insertIndex >= cards.length) {
    return cards[cards.length - 1].position + 1000;
  }

  const before = cards[insertIndex - 1].position;
  const after = cards[insertIndex].position;

  return (before + after) / 2;
}
