import { onUnmounted, ref } from 'vue'

// useOperation is one section's busy lock with its error and notice lines.
// Errors are the plugin's message; the fallback names the section's task.
export function useOperation(fallback = '操作未完成，请重试。') {
  const busy = ref(false), error = ref(''), notice = ref('')
  let disposed = false
  onUnmounted(() => { disposed = true })
  async function run(work: () => Promise<void>, success = ''): Promise<boolean> {
    if (busy.value) return false
    busy.value = true; error.value = ''; notice.value = ''
    try {
      await work()
      if (!disposed && success) notice.value = success
      return true
    } catch (cause) {
      if (!disposed) error.value = cause instanceof Error && cause.message ? cause.message : fallback
      return false
    } finally {
      if (!disposed) busy.value = false
    }
  }
  function clear() { error.value = ''; notice.value = '' }
  return { busy, error, notice, run, clear }
}

// download saves a text file through the browser, as an export.
export function download(name: string, content: string, type = 'application/json'): void {
  const url = URL.createObjectURL(new Blob([content], { type }))
  const link = document.createElement('a')
  link.href = url; link.download = name; link.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

const beijing = new Intl.DateTimeFormat('zh-CN', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
// formatTime writes a Unix millisecond time in Beijing time, or an em dash.
export function formatTime(ms?: number | null): string {
  if (!ms || ms <= 0) return '—'
  return beijing.format(new Date(ms))
}
