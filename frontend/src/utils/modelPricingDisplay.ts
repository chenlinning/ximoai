import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_VIDEO,
  type BillingMode,
} from '@/constants/channel'

export interface UnitPriceLike {
  billing_mode: BillingMode
  image_output_price: number | null
  per_request_price: number | null
}

export function displayUnitPrice(pricing: UnitPriceLike | null | undefined): number | null {
  if (!pricing) return null
  switch (pricing.billing_mode) {
    case BILLING_MODE_IMAGE:
      return pricing.per_request_price ?? pricing.image_output_price
    case BILLING_MODE_PER_REQUEST:
    case BILLING_MODE_VIDEO:
      return pricing.per_request_price
    default:
      return null
  }
}
