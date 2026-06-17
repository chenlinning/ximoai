const MEMBERSHIP_UPDATED_EVENT = 'ximoai:membership-updated'

export function notifyMembershipUpdated(): void {
  window.dispatchEvent(new CustomEvent(MEMBERSHIP_UPDATED_EVENT))
}

export function onMembershipUpdated(callback: () => void): () => void {
  const handler = () => callback()
  window.addEventListener(MEMBERSHIP_UPDATED_EVENT, handler)
  return () => window.removeEventListener(MEMBERSHIP_UPDATED_EVENT, handler)
}
