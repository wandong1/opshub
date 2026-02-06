import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'

export const vPermission: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding<string>) {
    const store = usePermissionStore()
    if (binding.value && !store.hasPermission(binding.value)) {
      el.parentNode?.removeChild(el)
    }
  }
}
