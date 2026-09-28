<script setup lang="ts">
// Authored 20px stroke icons, one weight, for the rail, states and row actions.
const paths: Record<string, string> = {
  search: '<circle cx="8.5" cy="8.5" r="5.5"/><path d="m12.5 12.5 4.5 4.5"/>',
  user: '<circle cx="10" cy="7" r="3.5"/><path d="M4 17c0-3.3 2.7-5.5 6-5.5s6 2.2 6 5.5"/>',
  clock: '<circle cx="10" cy="10" r="7"/><path d="M10 6v4l3 2"/>',
  growth: '<path d="m3 15 5-5 3 3 6-7"/><path d="M13 6h4v4"/>',
  book: '<path d="M4 4h5a2 2 0 0 1 2 2v11a1.5 1.5 0 0 0-1.5-1.5H4z"/><path d="M16 4h-5a2 2 0 0 0-2 2v11a1.5 1.5 0 0 1 1.5-1.5H16z"/>',
  cube: '<path d="m10 2.5 7 4v7l-7 4-7-4v-7z"/><path d="m3 6.5 7 4 7-4"/><path d="M10 10.5v7"/>',
  compass: '<circle cx="10" cy="10" r="7.5"/><path d="m13 7-1.5 4.5L7 13l1.5-4.5z"/>',
  megaphone: '<path d="M3 8.5v3a1 1 0 0 0 1 1h2l6 3.5v-12l-6 3.5H4a1 1 0 0 0-1 1z"/><path d="M15 8a3 3 0 0 1 0 4"/>',
  image: '<rect x="3" y="4" width="14" height="12" rx="2"/><circle cx="7.5" cy="8.5" r="1.5"/><path d="m17 13-4-3.5-6 5.5"/>',
  images: '<rect x="5" y="6" width="12" height="10" rx="2"/><path d="M3 13V5a2 2 0 0 1 2-2h8"/><path d="m17 13-3.5-3L8 15"/>',
  cards: '<rect x="4" y="5" width="9" height="12" rx="1.5"/><path d="M8 3h7a1.5 1.5 0 0 1 1.5 1.5V14"/>',
  sync: '<path d="M16.5 8A6.5 6.5 0 0 0 5 5.5L3.5 7"/><path d="M3.5 3.5V7H7"/><path d="M3.5 12a6.5 6.5 0 0 0 11.5 2.5l1.5-1.5"/><path d="M16.5 16.5V13H13"/>',
  tasks: '<rect x="5" y="4" width="10" height="13" rx="1.5"/><path d="M8 4a2 2 0 0 1 4 0"/><path d="m7.5 11 2 2 3.5-4"/>',
  dice: '<rect x="3.5" y="3.5" width="13" height="13" rx="2.5"/><circle cx="7" cy="7" r="1" fill="currentColor" stroke="none"/><circle cx="13" cy="7" r="1" fill="currentColor" stroke="none"/><circle cx="10" cy="10" r="1" fill="currentColor" stroke="none"/><circle cx="7" cy="13" r="1" fill="currentColor" stroke="none"/><circle cx="13" cy="13" r="1" fill="currentColor" stroke="none"/>',
  bell: '<path d="M6 8a4 4 0 0 1 8 0v3l1.5 2.5h-11L6 11z"/><path d="M8.5 16a1.5 1.5 0 0 0 3 0"/>',
  trophy: '<path d="M6 4h8v4a4 4 0 0 1-8 0z"/><path d="M6 5H3.5a2.5 2.5 0 0 0 2.5 3M14 5h2.5a2.5 2.5 0 0 1-2.5 3"/><path d="M10 12v3M7 16.5h6"/>',
  calendar: '<rect x="3.5" y="5" width="13" height="12" rx="2"/><path d="M3.5 9h13M7 3v3M13 3v3"/>',
  checkin: '<rect x="3.5" y="5" width="13" height="12" rx="2"/><path d="M3.5 9h13M7 3v3M13 3v3"/><path d="m7.5 13 2 2 3.5-3.5"/>',
  check: '<circle cx="10" cy="10" r="7.5"/><path d="m6.5 10 2.5 2.5 4.5-5"/>',
  people: '<circle cx="7.5" cy="7" r="3"/><path d="M2.5 17c0-3 2.2-5 5-5s5 2 5 5"/><circle cx="14" cy="8" r="2.5"/><path d="M14 12.5c2.3 0 4 1.8 4 4.5"/>',
  cloud: '<path d="M6 16a3.5 3.5 0 0 1-.5-7A5 5 0 0 1 15 8a3.5 3.5 0 0 1 0 8z"/>',
  sliders: '<path d="M4 6h12M4 10h12M4 14h12"/><circle cx="7" cy="6" r="1.5" fill="currentColor" stroke="none"/><circle cx="13" cy="10" r="1.5" fill="currentColor" stroke="none"/><circle cx="9" cy="14" r="1.5" fill="currentColor" stroke="none"/>',
  info: '<circle cx="10" cy="10" r="7.5"/><path d="M10 9v5"/><circle cx="10" cy="6.5" r="0.75" fill="currentColor" stroke="none"/>',
  alert: '<path d="M10 3.5 17.5 16.5h-15z"/><path d="M10 8v4"/><circle cx="10" cy="14.5" r="0.75" fill="currentColor" stroke="none"/>',
  error: '<circle cx="10" cy="10" r="7.5"/><path d="m7.5 7.5 5 5M12.5 7.5l-5 5"/>',
  wallet: '<rect x="3" y="6" width="14" height="11" rx="2"/><path d="M3 9h14"/><circle cx="13.5" cy="13" r="1" fill="currentColor" stroke="none"/>',
  ticket: '<path d="M3 8a2 2 0 0 0 0 4v3h14v-3a2 2 0 0 1 0-4V5H3z"/><path d="M12 5v10" stroke-dasharray="2 2"/>',
  chart: '<path d="M4 16h12"/><rect x="5.5" y="9" width="3" height="7"/><rect x="10" y="5" width="3" height="11"/><rect x="14.5" y="11" width="3" height="5"/>',
  plus: '<path d="M10 5v10M5 10h10"/>',
  download: '<path d="M10 3v9"/><path d="m6.5 8.5 3.5 3.5 3.5-3.5"/><path d="M4 15v1.5A1.5 1.5 0 0 0 5.5 18h9a1.5 1.5 0 0 0 1.5-1.5V15"/>',
  upload: '<path d="M10 12V3"/><path d="M6.5 6.5 10 3l3.5 3.5"/><path d="M4 15v1.5A1.5 1.5 0 0 0 5.5 18h9a1.5 1.5 0 0 0 1.5-1.5V15"/>',
  trash: '<path d="M4 6h12"/><path d="M7 6V4.5A1.5 1.5 0 0 1 8.5 3h3A1.5 1.5 0 0 1 13 4.5V6"/><path d="m5.5 6 .8 10a1.5 1.5 0 0 0 1.5 1.4h4.4a1.5 1.5 0 0 0 1.5-1.4l.8-10"/>',
  swap: '<path d="M4 7h10"/><path d="m11 4 3 3-3 3"/><path d="M16 13H6"/><path d="m9 16-3-3 3-3"/>',
  scan: '<path d="M3 7V5a2 2 0 0 1 2-2h2M13 3h2a2 2 0 0 1 2 2v2M17 13v2a2 2 0 0 1-2 2h-2M7 17H5a2 2 0 0 1-2-2v-2"/><path d="M5 10h10"/>',
  chat: '<path d="M3 5.5A1.5 1.5 0 0 1 4.5 4h8A1.5 1.5 0 0 1 14 5.5v5a1.5 1.5 0 0 1-1.5 1.5H8l-3.5 3v-3H4.5A1.5 1.5 0 0 1 3 10.5z"/><path d="M14 8h1.5A1.5 1.5 0 0 1 17 9.5v4a1.5 1.5 0 0 1-1.5 1.5H15v2l-2.5-2H10"/>',
  inbox: '<path d="M3 11h4l1.5 2h3L13 11h4"/><path d="M5 4h10l2 7v5a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 3 16v-5z"/>',
  history: '<path d="M4 10a6.5 6.5 0 1 1 2 4.7"/><path d="M3.5 9.5v4H7"/><path d="M10 7v3.5l2.5 1.5"/>',
}
const props = defineProps<{ name: string; label?: string }>()
</script>
<template>
  <svg class="icon" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" :aria-hidden="props.label ? undefined : 'true'" :role="props.label ? 'img' : undefined" :aria-label="props.label" v-html="paths[props.name] ?? paths.info" />
</template>
