import { Modal } from '@arco-design/web-vue'

/**
 * Promise-based wrapper for Arco Design Modal.confirm
 * Compatible with the old ElMessageBox.confirm await/catch pattern
 * Rejects with 'cancel' when user cancels, so existing catch blocks work
 */
export function confirmModal(
  content: string,
  title = '确认',
  opts: Record<string, any> = {}
): Promise<void> {
  return new Promise((resolve, reject) => {
    Modal.confirm({
      title,
      content,
      okText: opts.confirmButtonText || '确定',
      cancelText: opts.cancelButtonText || '取消',
      ...opts,
      onOk: () => resolve(),
      onCancel: () => reject('cancel'),
    })
  })
}
