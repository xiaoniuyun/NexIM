import { ref } from 'vue'
export const toasts = ref([])
let seq = 0
export function toast(msg, type = '') {
  const id = ++seq
  toasts.value.push({ id, msg, type })
  setTimeout(() => { toasts.value = toasts.value.filter(x => x.id !== id) }, 2600)
}
