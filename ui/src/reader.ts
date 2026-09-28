import { computed, getCurrentScope, onScopeDispose, ref, watch, type Ref } from 'vue'

// An Operation is a section's busy lock with a run that resolves to whether
// the work succeeded: useOperation from the shell, or a cloud task adapted
// with taskOperation.
export interface Operation { busy: Readonly<Ref<boolean>>; run(work: () => Promise<void>): Promise<boolean> }

// useReader is a section's automatic read: the same read runs on mount, on a
// role change and on a changed filter, never concurrently. A request made
// while the section is busy (this read or any other operation) is kept and
// replayed once after it finishes, so a burst of changes ends in exactly one
// more read with the inputs of that moment. A delay debounces typed filters.
// The read receives stale(), true once a newer request or a cancel superseded
// it, so a late result for an earlier role or filter is dropped, not shown.
// pending is true from a request until its read starts; loading while it runs;
// failed after a read that threw, until the next operation starts.
export function useReader(operation: Operation, read: (stale: () => boolean) => Promise<void>) {
  const pending = ref(false), loading = ref(false), failed = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined, queued = false, disposed = false, generation = 0
  async function flush() {
    clearTimeout(timer); timer = undefined
    if (disposed) return
    if (operation.busy.value) { queued = true; return }
    queued = false; pending.value = false; loading.value = true
    const current = generation
    try { failed.value = !(await operation.run(() => read(() => current !== generation))) }
    finally { if (!disposed) loading.value = false }
  }
  watch(operation.busy, busy => {
    if (busy) { if (!loading.value) failed.value = false }
    else if (queued && !disposed) void flush()
  })
  function request(delay = 0) {
    generation++; pending.value = true
    clearTimeout(timer)
    if (delay > 0) timer = setTimeout(() => void flush(), delay)
    else void flush()
  }
  function cancel() { generation++; clearTimeout(timer); timer = undefined; queued = false; pending.value = false }
  if (getCurrentScope()) onScopeDispose(() => { disposed = true; cancel() })
  return { request, cancel, pending, loading, failed }
}

// taskOperation lets a reader drive a cloud task: the task counts as busy
// until its job has finished, so the next request waits for that result.
export function taskOperation(task: { busy: Ref<boolean>; job: Ref<{ state: string } | null> }): Operation {
  return {
    busy: computed(() => task.busy.value || task.job.value?.state === 'running'),
    async run(work) { try { await work(); return true } catch { return false } },
  }
}
