package service

import (
	"errors"
	"fmt"
)

var ErrBillablePricingRequired = errors.New("billable usage requires positive pricing")

func channelPricingHasPositiveConfiguredPrice(p *ChannelModelPricing) bool {
	if p == nil {
		return false
	}
	if positiveFloat64Ptr(p.InputPrice) || positiveFloat64Ptr(p.OutputPrice) ||
		positiveFloat64Ptr(p.CacheWritePrice) || positiveFloat64Ptr(p.CacheReadPrice) ||
		positiveFloat64Ptr(p.ImageOutputPrice) || positiveFloat64Ptr(p.PerRequestPrice) {
		return true
	}
	for _, iv := range p.Intervals {
		if positiveFloat64Ptr(iv.InputPrice) || positiveFloat64Ptr(iv.OutputPrice) ||
			positiveFloat64Ptr(iv.CacheWritePrice) || positiveFloat64Ptr(iv.CacheReadPrice) ||
			positiveFloat64Ptr(iv.PerRequestPrice) {
			return true
		}
	}
	return false
}

func channelPricingHasPositivePerRequestPrice(p ChannelModelPricing) bool {
	if positiveFloat64Ptr(p.PerRequestPrice) {
		return true
	}
	for _, iv := range p.Intervals {
		if positiveFloat64Ptr(iv.PerRequestPrice) {
			return true
		}
	}
	return false
}

func positiveFloat64Ptr(v *float64) bool {
	return v != nil && *v > 0
}

func validateBillableUsageCost(cost *CostBreakdown, imageCount, videoCount int) error {
	mode := resolveUsageBillingMode(cost, imageCount, videoCount)
	switch mode {
	case BillingModePerRequest, BillingModeImage, BillingModeVideo:
		if cost == nil || cost.ActualCost <= 0 {
			return fmt.Errorf("%w: billing_mode=%s", ErrBillablePricingRequired, mode)
		}
	default:
		return nil
	}
	return nil
}

func resolveUsageBillingMode(cost *CostBreakdown, imageCount, videoCount int) BillingMode {
	if cost != nil && cost.BillingMode != "" {
		return BillingMode(cost.BillingMode)
	}
	if videoCount > 0 {
		return BillingModeVideo
	}
	if imageCount > 0 {
		return BillingModeImage
	}
	return BillingModeToken
}
