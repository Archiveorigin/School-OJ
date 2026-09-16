/** Run independent reads with bounded concurrency, preserving result order. */
export async function mapConcurrent<T, R>(items: readonly T[], limit: number, read: (item: T) => Promise<R>): Promise<R[]> {
  const results = new Array<R>(items.length)
  let cursor = 0
  await Promise.all(Array.from({ length: Math.min(Math.max(1, limit), items.length) }, async () => {
    while (cursor < items.length) {
      const index = cursor++
      results[index] = await read(items[index])
    }
  }))
  return results
}
