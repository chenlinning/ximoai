package service

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BillingMode identifies how a model is billed.
type BillingMode string

const (
	BillingModeToken      BillingMode = "token"
	BillingModePerRequest BillingMode = "per_request"
	BillingModeImage      BillingMode = "image"
	BillingModeVideo      BillingMode = "video"
)

// IsValid checks whether BillingMode is supported.
func (m BillingMode) IsValid() bool {
	switch m {
	case BillingModeToken, BillingModePerRequest, BillingModeImage, BillingModeVideo, "":
		return true
	}
	return false
}

const (
	BillingModelSourceRequested     = "requested"
	BillingModelSourceUpstream      = "upstream"
	BillingModelSourceChannelMapped = "channel_mapped"
)

// Channel is an upstream channel entity.
type Channel struct {
	ID                 int64
	Name               string
	Description        string
	Status             string
	BillingModelSource string
	RestrictModels     bool
	Features           string
	FeaturesConfig     map[string]any
	CreatedAt          time.Time
	UpdatedAt          time.Time

	GroupIDs []int64

	ModelPricing []ChannelModelPricing
	ModelMapping map[string]map[string]string

	ApplyPricingToAccountStats bool
	AccountStatsPricingRules   []AccountStatsPricingRule
}

// AccountStatsPricingRule defines account-stat pricing overrides.
type AccountStatsPricingRule struct {
	ID         int64
	ChannelID  int64
	Name       string
	GroupIDs   []int64
	AccountIDs []int64
	SortOrder  int
	Pricing    []ChannelModelPricing // 閻熸瑥瀚崹顖炲礃閸涱垱鐣辨俊顖椻偓宕団偓椋庘偓瑙勭煯閻滎垶鏁嶉崼婵愭Щ闁活潿鍔庨獮鍥嫉婢跺﹦鏆板ù鐘活棑缁劑寮搁崟鍓佺
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ChannelModelPricing defines pricing rules for models in a channel.
type ChannelModelPricing struct {
	ID               int64
	ChannelID        int64
	Platform         string
	Models           []string
	BillingMode      BillingMode
	InputPrice       *float64
	OutputPrice      *float64
	CacheWritePrice  *float64
	CacheReadPrice   *float64
	ImageOutputPrice *float64
	PerRequestPrice  *float64
	Intervals        []PricingInterval
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PricingInterval is a tier for token, per-request, image, or video pricing.
type PricingInterval struct {
	ID              int64
	PricingID       int64
	MinTokens       int
	MaxTokens       *int
	TierLabel       string
	InputPrice      *float64
	OutputPrice     *float64
	CacheWritePrice *float64
	CacheReadPrice  *float64
	PerRequestPrice *float64
	SortOrder       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IsActive reports whether the channel is enabled.
func (c *Channel) IsActive() bool {
	return c.Status == StatusActive
}

// normalizeBillingModelSource fills the default source for older rows.
func (c *Channel) normalizeBillingModelSource() {
	if c == nil {
		return
	}
	if c.BillingModelSource == "" {
		c.BillingModelSource = BillingModelSourceChannelMapped
	}
}

// GetModelPricing finds pricing by model name, case-insensitively.
func (c *Channel) GetModelPricing(model string) *ChannelModelPricing {
	modelLower := strings.ToLower(model)

	for i := range c.ModelPricing {
		for _, m := range c.ModelPricing[i].Models {
			if strings.ToLower(m) == modelLower {
				cp := c.ModelPricing[i].Clone()
				return &cp
			}
		}
	}

	return nil
}

// FindMatchingInterval finds the interval matching totalTokens using (min, max].
func FindMatchingInterval(intervals []PricingInterval, totalTokens int) *PricingInterval {
	for i := range intervals {
		iv := &intervals[i]
		if totalTokens > iv.MinTokens && (iv.MaxTokens == nil || totalTokens <= *iv.MaxTokens) {
			return iv
		}
	}
	return nil
}

// GetIntervalForContext finds the pricing tier for a context size.
func (p *ChannelModelPricing) GetIntervalForContext(totalTokens int) *PricingInterval {
	return FindMatchingInterval(p.Intervals, totalTokens)
}

// GetTierByLabel finds a per-request/image/video tier by label.
func (p *ChannelModelPricing) GetTierByLabel(label string) *PricingInterval {
	labelLower := strings.ToLower(label)
	for i := range p.Intervals {
		if strings.ToLower(p.Intervals[i].TierLabel) == labelLower {
			return &p.Intervals[i]
		}
	}
	return nil
}

// Clone returns a deep copy of ChannelModelPricing.
func (p ChannelModelPricing) Clone() ChannelModelPricing {
	cp := p
	if p.Models != nil {
		cp.Models = make([]string, len(p.Models))
		copy(cp.Models, p.Models)
	}
	if p.Intervals != nil {
		cp.Intervals = make([]PricingInterval, len(p.Intervals))
		copy(cp.Intervals, p.Intervals)
	}
	return cp
}

// Clone returns a deep copy of Channel.
func (c *Channel) Clone() *Channel {
	if c == nil {
		return nil
	}
	cp := *c
	if c.GroupIDs != nil {
		cp.GroupIDs = make([]int64, len(c.GroupIDs))
		copy(cp.GroupIDs, c.GroupIDs)
	}
	if c.ModelPricing != nil {
		cp.ModelPricing = make([]ChannelModelPricing, len(c.ModelPricing))
		for i := range c.ModelPricing {
			cp.ModelPricing[i] = c.ModelPricing[i].Clone()
		}
	}
	if c.ModelMapping != nil {
		cp.ModelMapping = make(map[string]map[string]string, len(c.ModelMapping))
		for platform, mapping := range c.ModelMapping {
			inner := make(map[string]string, len(mapping))
			for k, v := range mapping {
				inner[k] = v
			}
			cp.ModelMapping[platform] = inner
		}
	}
	if c.FeaturesConfig != nil {
		cp.FeaturesConfig = deepCopyFeaturesConfig(c.FeaturesConfig)
	}
	if c.AccountStatsPricingRules != nil {
		cp.AccountStatsPricingRules = make([]AccountStatsPricingRule, len(c.AccountStatsPricingRules))
		for i, rule := range c.AccountStatsPricingRules {
			cp.AccountStatsPricingRules[i] = rule
			if rule.GroupIDs != nil {
				cp.AccountStatsPricingRules[i].GroupIDs = make([]int64, len(rule.GroupIDs))
				copy(cp.AccountStatsPricingRules[i].GroupIDs, rule.GroupIDs)
			}
			if rule.AccountIDs != nil {
				cp.AccountStatsPricingRules[i].AccountIDs = make([]int64, len(rule.AccountIDs))
				copy(cp.AccountStatsPricingRules[i].AccountIDs, rule.AccountIDs)
			}
			if rule.Pricing != nil {
				cp.AccountStatsPricingRules[i].Pricing = make([]ChannelModelPricing, len(rule.Pricing))
				for j := range rule.Pricing {
					cp.AccountStatsPricingRules[i].Pricing[j] = rule.Pricing[j].Clone()
				}
			}
		}
	}
	return &cp
}

// IsWebSearchEmulationEnabled reports whether web-search emulation is enabled.
func (c *Channel) IsWebSearchEmulationEnabled(platform string) bool {
	if c == nil || c.FeaturesConfig == nil {
		return false
	}
	wse, ok := c.FeaturesConfig[featureKeyWebSearchEmulation].(map[string]any)
	if !ok {
		return false
	}
	enabled, ok := wse[platform].(bool)
	return ok && enabled
}

// deepCopyFeaturesConfig creates a deep copy of FeaturesConfig to prevent cache pollution.
func deepCopyFeaturesConfig(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		if inner, ok := v.(map[string]any); ok {
			dst[k] = deepCopyFeaturesConfig(inner)
		} else {
			dst[k] = v
		}
	}
	return dst
}

// ValidateIntervals validates pricing tiers for the given billing mode.
func ValidateIntervals(intervals []PricingInterval, mode BillingMode) error {
	if len(intervals) == 0 {
		return nil
	}
	sorted := make([]PricingInterval, len(intervals))
	copy(sorted, intervals)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].MinTokens < sorted[j].MinTokens
	})

	for i := range sorted {
		if err := validateSingleInterval(&sorted[i], i); err != nil {
			return err
		}
	}

	if mode == BillingModePerRequest || mode == BillingModeImage || mode == BillingModeVideo {
		return nil
	}
	return validateIntervalOverlap(sorted)
}

func validateSingleInterval(iv *PricingInterval, idx int) error {
	if iv.MinTokens < 0 {
		return fmt.Errorf("interval #%d: min_tokens (%d) must be >= 0", idx+1, iv.MinTokens)
	}
	if iv.MaxTokens != nil {
		if *iv.MaxTokens <= 0 {
			return fmt.Errorf("interval #%d: max_tokens (%d) must be > 0", idx+1, *iv.MaxTokens)
		}
		if *iv.MaxTokens <= iv.MinTokens {
			return fmt.Errorf("interval #%d: max_tokens (%d) must be > min_tokens (%d)",
				idx+1, *iv.MaxTokens, iv.MinTokens)
		}
	}
	return validateIntervalPrices(iv, idx)
}

func validateIntervalPrices(iv *PricingInterval, idx int) error {
	prices := []struct {
		name string
		val  *float64
	}{
		{"input_price", iv.InputPrice},
		{"output_price", iv.OutputPrice},
		{"cache_write_price", iv.CacheWritePrice},
		{"cache_read_price", iv.CacheReadPrice},
		{"per_request_price", iv.PerRequestPrice},
	}
	for _, p := range prices {
		if p.val != nil && *p.val < 0 {
			return fmt.Errorf("interval #%d: %s must be >= 0", idx+1, p.name)
		}
	}
	return nil
}

func validateIntervalOverlap(sorted []PricingInterval) error {
	for i, iv := range sorted {
		if iv.MaxTokens == nil && i < len(sorted)-1 {
			return fmt.Errorf("interval #%d: unbounded interval (max_tokens=null) must be the last one",
				i+1)
		}
		if i == 0 {
			continue
		}
		prev := sorted[i-1]
		if prev.MaxTokens == nil || *prev.MaxTokens > iv.MinTokens {
			return fmt.Errorf("interval #%d and #%d overlap: prev max=%s > cur min=%d",
				i, i+1, formatMaxTokensLabel(prev.MaxTokens), iv.MinTokens)
		}
	}
	return nil
}

func formatMaxTokensLabel(max *int) string {
	if max == nil {
		return "unbounded"
	}
	return fmt.Sprintf("%d", *max)
}

// ChannelUsageFields are channel fields embedded in usage records.
type ChannelUsageFields struct {
	ChannelID          int64
	OriginalModel      string
	ChannelMappedModel string
	BillingModelSource string
	ModelMappingChain  string
}

// SupportedModel is a concrete model exposed by a channel.
type SupportedModel struct {
	Name     string
	Platform string
	Pricing  *ChannelModelPricing
}

// wildcardSuffix 闁哄嫷鍨辫啯闁搞劌顑嗚啯鐎殿喖绻嬮懙鎴︽儍閸曨垪鍋撳璺哄赋缂佹绠戦幃妤冪磽閳ь剟寮介崶顏嶅敹闁挎稑鐗呯划搴ㄥ绩椤栨稑鐦悘蹇涚畺閸庢挳宕犺ぐ鎺戝赋闁挎稑顦埀?
const wildcardSuffix = "*"

// splitWildcardSuffix 閻忓繐妫欒啯闁搞劌顑嗚啯鐎殿喖绻戞刊鍫曞礆閸℃洝绀?(prefix, isWildcard)闁?//
//
//	"claude-opus-*"  闁?("claude-opus-", true)
//	"claude-opus-4"  闁?("claude-opus-4", false)
//	"*"              闁?("", true)
//
// 婵炲鍔嶉崜浼存晬濮樺磭绠查柛銉у仧濞?prefix 濞ｅ洦绻冪€垫棃宕㈤悢濂夋綏濠㈠爢鍐瘓闁告劖鐟辩槐婵嬫偨鏉堫偆娈堕柣顫妽閺岀喖骞愭径鎰粯 ToLower闁?
func splitWildcardSuffix(pattern string) (prefix string, isWildcard bool) {
	if strings.HasSuffix(pattern, wildcardSuffix) {
		return strings.TrimSuffix(pattern, wildcardSuffix), true
	}
	return pattern, false
}

// GetModelPricingByPlatform 闁革负鍔嶇€垫氨鈧鑹鹃柦鈺呭矗妫颁胶鐟撻柡灞诲劜婢规鍒掗崜褉鈧ê螣閳ュ磭鈧兘鎯冮崟顐ゆ毎濞寸姴鍤栫槐婵嬪嫉椤忓懎顥濋柛鎺撳缁绘垿宕?nil闁?// 濞?GetModelPricing 闁汇劌瀚亸顖炲礆椤愵剛绐楅柟?Platform 闂傚懏姊婚‖鍥晬瀹€鍕級闁稿繐绉峰▔鏇㈢嵁閸愭彃閰遍柛姘嫰閹洖螣閳ュ磭鈧鎷犻姘埍闂佹澘绉查埀?
func (c *Channel) GetModelPricingByPlatform(platform, model string) *ChannelModelPricing {
	if c == nil {
		return nil
	}
	modelLower := strings.ToLower(model)
	for i := range c.ModelPricing {
		if c.ModelPricing[i].Platform != platform {
			continue
		}
		for _, m := range c.ModelPricing[i].Models {
			if strings.ToLower(m) == modelLower {
				cp := c.ModelPricing[i].Clone()
				return &cp
			}
		}
	}
	return nil
}

// platformPricingIndex 闁哄嫷鍨板畷鐔哥▔椤忓嫰鎸柛娆擃暒缁楀懐鈧鐭悳顖涚┍閳╁啩绱栭柣銊ュ椤︽煡宕ラ崼銏犲亶鐎殿喗娲忛埀?// 濞戞挴鍋撴繛鍡忓墲婢瑰倿骞撹箛鎾崇ギ闁告瑯鍨伴幃鎾诲籍閼稿灚鏆滈柟闀愯兌缁ㄨ法娑甸鐣屽弨闁瑰灚鎷濈槐妾坸act 闁告帒妫欓弫顕€鏁嶆径澶岀憿闁哄牆顦花顓㈡焼瀹ュ懎鍧婇柨娑樻緥ildcard 闁告帒妫欓弫顕€鏁嶆径娑氱
// 闂侇剙鐏濋崢?SupportedModels 閻庝絻顫夐惁鈩冪▔椤忓嫰鎸柛娆庡嵆閸ｅ憡寰勫鍡楊棁闁硅绻愰悾鐐瀹勬澘鐏欓悶娑栧妸閳?//
// byLower 濞?names/originalCase 闁稿繐褰夐棅鈺呭触鐏炶偐顏卞┑鍌涱殔楠炴捇鏌屽鍫綈闁告帗鐟辩槐鐗堢?lower-case 婵☆垪鈧磭鈧兘宕ュ鍕 key闁?// 濡絾鐗旈柌婊堝川閹存帟鍘ǎ鍥ㄧ箘閺嗏偓闁稿繗娉涚敮顐ｆ叏鐎ｎ亗浜ｉ悘蹇撶箰閸熸捇濡存穱绌塵es 缂備礁鐡ㄧ€垫棃骞愭径濠勬毎濞寸姷鏌夐、鎴﹀箥椤愶絽浼庡銈呮惈缁參鎯冮崟顓€鏃傗偓瑙勪亢閸戭垱绂掗敐鍐ｅ亾
type platformPricingIndex struct {
	byLower      map[string]*ChannelModelPricing // lowercased model name 闁?pricing (Clone'd)
	originalCase map[string]string               // lowercased model name 闁?original-case model name
	names        []string                        // priced model names in their ORIGINAL case, insertion-ordered, deduped case-insensitively (first wins)
}

// buildPricingIndex builds a platform/model lookup for pricing rules.
func buildPricingIndex(pricings []ChannelModelPricing) map[string]*platformPricingIndex {
	idx := make(map[string]*platformPricingIndex)
	for i := range pricings {
		p := pricings[i]
		pidx, ok := idx[p.Platform]
		if !ok {
			pidx = &platformPricingIndex{
				byLower:      make(map[string]*ChannelModelPricing),
				originalCase: make(map[string]string),
				names:        make([]string, 0),
			}
			idx[p.Platform] = pidx
		}
		for _, m := range p.Models {
			if _, wild := splitWildcardSuffix(m); wild {
				continue
			}
			lower := strings.ToLower(m)
			if _, exists := pidx.byLower[lower]; exists {
				continue // 濡絾鐗旈柌婊堝川閹存帟鍘柤铏矊閸ゎ參鏁嶉崸鍧卻e-insensitive 闁告ê顭烽崳鎼佸触鎼达綆鍎戝☉鎾亾濞戞搩浜滈悾鐐?/ 缂佹鍏涚粩瀛樼▔椤忓嫬鏂у┑顔碱儏閵囧洨浜歌箛鎾虫櫢闁?
			}
			cp := pricings[i].Clone()
			pidx.byLower[lower] = &cp
			pidx.originalCase[lower] = m
			pidx.names = append(pidx.names, m)
		}
	}
	return idx
}

// SupportedModels 閻犱緤绱曢悾璇层€掗悩璁冲闁汇劌瀚弫顕€骞愭担榧撲線宕圭€ｎ亜鐏欓悶娑辩厜缁辨繄绱掗幘瀵镐函濞ｅ洦绻嗛惁澶嬬▔瀹ュ懏鍎撻梺顐ｅ哺閸樸倗绮敂琛″亾?//
// 缂佺姵顨嗙涵鍫曟晬閸ь晣pping 闁?pricing 妤犵偞鍎兼禒鍫ユ晬婢舵稓绐?//
//   - Pass A闁挎稑娼癮pping闁挎稑顧€缁变即鏌嗗鍛潑 ModelMapping
//   - 缂侇喖澧介垾?src 闁?target闁挎稒纰嶅Ο澶岀矆閸濆嫭鍊?= src闁挎稑鐗忛弫銈夊箣閻ゎ垼娼掗悷娆愬釜缁辨岸鏁嶇仦鐣屾毎濞寸娀顥撻弫?target 闁革负鍔岄幃?platform 閻庤鐭悳顖炴煂鐏炲墽鍙€
//     闁挎稑娼癮pping 闁衡偓閻熸澘鏅搁柛姘閻ゅ嫰姊介崨閭﹀悁閻犳劕婀卞▓鎴﹀及?target闁挎稒绋栫换鏍及椤栨粍鏆忛柟鎾敱閸斿懘鎯岄妷褎鐣?閻庡湱鍋ゅ顖炴嚍鏉堫偄鐎?闁挎稑顦埀?//     target 濞戞捁娅ｉ埞鏍箣閺嶏箒绀嬮梺顐ｅ哺閸樸倗绮敂鑺ヮ槯闂侇偀鍋撻柛鏍ㄧ墧鐠愮喖骞?src 闁煎浜濋悡锟犲Υ?//   - 闂侇偅宀搁崢銈囩箔?src闁挎稑鐗嗛々?"claude-3-*"闁挎稑顧€缁变即鎮介妸銉﹀€?platform 閻庤鐭悳顖炴煂鐏炶棄顤呯紓鍌楀亾闁告牕缍婇崢銈夋儍閸曨儫渚€宕圭€ｂ晝绋婂☉鎾虫惈閳ь剚鐟╅埀顒€顦惈宥咁嚕閳ь剟鏁?//     婵絽绻嬮柌婊堝磹濞嗘挴鍋撴径灞炬殢闁煎浜ｉ棅鈺冣偓瑙勭煯閻滎垶鏁嶉崼銉㈠亾濮樿泛甯崇紒妤嬬畱濠р偓闁哄拋鍨粩鎾嚋椤掍焦笑 passthrough闁挎稑顔揳rget 闂侇偅鑹鹃悥鑸电▕閻斿憡笑闂侇偅宀搁崢銈囩箔閿旇偐绀嗛柕?//   - "*" 闁告娲滅€?mapping key 閻犙傚嵆閳ь剚宀搁崢銈囩箔閿曗偓閸ㄥ酣寮ㄩ銈囩闁告挸绉剁槐鎴炵▔閾忓厜鏁?闁?闁稿繈鍔岄惈宥咁嚕閳ь剟鏁嶆径鍫氬亾?//   - Pass B闁挎稑娼穜icing-only闁挎稑顧€缁变即鏌嗗鍛潑 ModelPricing 濞戞搩鍘芥晶宥夊嫉婢舵劖濮滈梺顐ｅ哺閸樸倗绮敂渚ヤ線宕圭€ｅ墎绀夐悗浣冾潐濠€顓㈠捶?Pass A 婵烇綀顕ф慨鐐存交閸モ晜鐣?//     閻炴稏鍎电紞鍫ュ灳閺傝　鍋撻弮鈧Ο澶岀矆閸濆嫭鍊?= 閻庤鐭悳顖毼熼垾宕団偓鐑藉触瀹ュ繒绀夐悗瑙勭煯閻?= 闁煎浜ｉ棅鈺呮晬閸絿绠归柡鍕靛灠閸櫻囨煥椤旂粯鍙忓璺虹▌缁辨壆鈧鐭悳顖溾偓娑櫭﹢顏堝础閸忓懎鏁╅悶娑栧妽缁楊參鏌嗛幘瀛樻殰闁归晲娴囬姘熼垾宕団偓鐑芥晬?//     闁告鍘栨繛鍥р柦閿熺姴甯抽柡鍕Т閻ㄧ娀鏁嶆径鍫氬亾?//
//
// 闁哄嫬澧介妵姘跺触瀹ュ懏鍤掑☉鎿冨幖閻ｇ偓绂掗柨瀣槯濞达綀娉曢弫?*閻庤鐭悳顖炴儍閸曨偄鏂у┑顔碱儏閵囧洨浜歌箛鎾虫櫢**闁挎稑鐗嗛悾鐐闁垮笑婵☆垪鈧磭鈧兘鐓鈥虫暅闁汇劌瀚花銊р偓鍦仦濞奸潧鈹冮幇鍓佺闁?// 闁?(Platform, Name) 缂佸鍟块悾楣冨箳閹烘垹纰嶉柨娑樻湰鐎?(Platform, lowercase(Name)) 闁告ê顭烽崳鎼佹晬鐏炶棄甯ラ柛鎺撳閳ь剙鎳撻崕銊╁礄閹巻鍋?//
// 婵炲鍔嶉崜浼存晬濮橆剛鏆板ù鐘烘腹缁酣宕?channel.ModelPricing 闁告劕鎳忛悡锟犲箥闁款垪鍋撻弬琛″亾閺傚灝寮块悘鐐╁亾 LiteLLM 闁搞儳鍋犻幆銈夋偨鏉堫偆娈堕柣顫妽閺?
func (c *Channel) SupportedModels() []SupportedModel {
	if c == nil {
		return nil
	}
	if len(c.ModelMapping) == 0 && len(c.ModelPricing) == 0 {
		return nil
	}

	idx := buildPricingIndex(c.ModelPricing)

	type dedupKey struct {
		platform string
		name     string
	}
	seen := make(map[dedupKey]struct{})
	result := make([]SupportedModel, 0)

	lookup := func(pidx *platformPricingIndex, name string) (display string, pricing *ChannelModelPricing) {
		if pidx == nil || name == "" {
			return name, nil
		}
		lower := strings.ToLower(name)
		if p, ok := pidx.byLower[lower]; ok {
			return pidx.originalCase[lower], p
		}
		return name, nil
	}

	add := func(platform, displayName string, pricing *ChannelModelPricing) {
		key := dedupKey{platform: platform, name: strings.ToLower(displayName)}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, SupportedModel{
			Name:     displayName,
			Platform: platform,
			Pricing:  pricing,
		})
	}

	// Pass A闁挎稒鐭划?mapping 閻忕偞娲栫槐?
	for platform, mapping := range c.ModelMapping {
		if len(mapping) == 0 {
			continue
		}
		pidx := idx[platform]
		for src, target := range mapping {
			prefix, isWild := splitWildcardSuffix(src)
			if isWild {
				if pidx == nil {
					continue
				}
				prefixLower := strings.ToLower(prefix)
				for _, candidate := range pidx.names {
					if strings.HasPrefix(strings.ToLower(candidate), prefixLower) {
						display, pricing := lookup(pidx, candidate)
						add(platform, display, pricing)
					}
				}
				continue
			}
			// 缂侇喖澧介垾?mapping闁挎稒鑹鹃悾鐐闁垮鐦?target 闁哄被鍎荤槐鐪沘rget 缂傚倸鎼妵?闂侇偅宀搁崢銈夊礆濞嗘挴鍋撻埀顒勫礌閺嶃劌鐦?src 闁?
			pricingKey := target
			if pricingKey == "" {
				pricingKey = src
			}
			if _, targetWild := splitWildcardSuffix(pricingKey); targetWild {
				pricingKey = src
			}
			_, pricing := lookup(pidx, pricingKey)
			// 闁哄嫬澧介妵姘跺触瀹ュ嫮鍠橀柛蹇撶墢閺?src 闁革负鍔岄悾鐐閻戣棄娅￠柣銊ュ鐢偅鎱ㄧ€ｎ亗浜ｉ悘蹇撶箰閸熸捇鏁嶉崼锝咁仧 src 闁哄牜鍓濋棅鈺呭及椤栨瑩鍤嬮悗瑙勭煯閻滎垰螣閳ュ磭鈧兘宕ュ蹇曠
			displayName, _ := lookup(pidx, src)
			add(platform, displayName, pricing)
		}
	}

	// Pass B闁挎稒鐭划?pricing 閻炴稏鍎电紞?mapping 闁哄牜浜ｉ々顐︽儎閺嶎偅鐣遍柛蹇氭腹缂嶅螣閳ュ磭鈧兘鏁嶉崼婊勫弿濠?閻庤鐭悳顖溾偓娑櫭﹢顏呮媴閸℃姊鹃梺鏉跨У濡惭呬焊?闁?濞戞挸绉靛Ο澶岀矆?闁?
	for platform, pidx := range idx {
		for _, name := range pidx.names {
			display, pricing := lookup(pidx, name)
			add(platform, display, pricing)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Platform != result[j].Platform {
			return result[i].Platform < result[j].Platform
		}
		return result[i].Name < result[j].Name
	})
	return result
}
