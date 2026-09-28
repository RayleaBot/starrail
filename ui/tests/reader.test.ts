import { afterEach, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { useReader, type Operation } from '../src/reader'
afterEach(() => vi.useRealTimers())
function deferred() { let resolve!: () => void; const promise = new Promise<void>(r => { resolve = r }); return { promise, resolve } }
// operation mirrors useOperation: one busy lock, run resolves to success.
function operation(): Operation & { busy: ReturnType<typeof ref<boolean>> } {
  const busy = ref(false)
  return { busy, async run(work) { if (busy.value) return false; busy.value = true; try { await work(); return true } catch { return false } finally { busy.value = false } } }
}
async function settle() { for (let i = 0; i < 4; i++) { await nextTick(); await Promise.resolve() } }

it('replays requests made during a read exactly once, after it finishes', async () => {
  const op = operation(), gate = deferred(); let runs = 0
  const reader = useReader(op, async () => { runs++; if (runs === 1) await gate.promise })
  reader.request(); await settle()
  expect(runs).toBe(1); expect(reader.loading.value).toBe(true)
  reader.request(); reader.request(); await settle()
  expect(runs).toBe(1)
  gate.resolve(); await settle()
  expect(runs).toBe(2); expect(reader.loading.value).toBe(false)
})

it('debounces delayed requests into one read', async () => {
  vi.useFakeTimers(); const op = operation(); let runs = 0
  const reader = useReader(op, async () => { runs++ })
  reader.request(250); vi.advanceTimersByTime(100); reader.request(250); vi.advanceTimersByTime(100); reader.request(250)
  vi.advanceTimersByTime(249); await settle(); expect(runs).toBe(0)
  vi.advanceTimersByTime(1); await settle(); expect(runs).toBe(1)
})

it('waits for another operation of the section before reading', async () => {
  const op = operation(), gate = deferred(); let runs = 0
  const reader = useReader(op, async () => { runs++ })
  const write = op.run(() => gate.promise)
  reader.request(); await settle(); expect(runs).toBe(0)
  gate.resolve(); await write; await settle(); expect(runs).toBe(1)
})

it('marks a superseded read stale and reports a failed read', async () => {
  const op = operation(), gate = deferred(); const seen: boolean[] = []; let runs = 0
  const reader = useReader(op, async stale => { runs++; if (runs === 1) { await gate.promise; seen.push(stale()) } else throw new Error('no') })
  reader.request(); await settle(); reader.request(); gate.resolve(); await settle()
  expect(seen).toEqual([true]); expect(runs).toBe(2); expect(reader.failed.value).toBe(true)
})

it('reports a scheduled read as pending and drops it on cancel', async () => {
  vi.useFakeTimers(); const op = operation(); let runs = 0
  const reader = useReader(op, async () => { runs++ })
  reader.request(250); expect(reader.pending.value).toBe(true)
  reader.cancel(); vi.advanceTimersByTime(300); await settle()
  expect(runs).toBe(0); expect(reader.pending.value).toBe(false)
})

it('drops a scheduled read and a late result once its scope is disposed', async () => {
  vi.useFakeTimers(); const op = operation(), gate = deferred(); let runs = 0; const seen: boolean[] = []
  const scope = effectScope()
  const reader = scope.run(() => useReader(op, async stale => { runs++; await gate.promise; seen.push(stale()) }))!
  reader.request(); await settle(); reader.request(250)
  scope.stop(); vi.advanceTimersByTime(300); await settle()
  expect(runs).toBe(1); expect(reader.pending.value).toBe(false)
  gate.resolve(); await settle()
  expect(seen).toEqual([true])
})
