package service

type MembershipLevelDefinition struct {
	Code        string
	Name        string
	Color       string
	Description string
	SortOrder   int
	IsDefault   bool
}

const (
	MembershipLevelCodeBronze   = "bronze"
	MembershipLevelCodeSilver   = "silver"
	MembershipLevelCodeGold     = "gold"
	MembershipLevelCodePlatinum = "platinum"
	MembershipLevelCodeDiamond  = "diamond"
)

var fixedMembershipLevelDefinitions = []MembershipLevelDefinition{
	{
		Code:        MembershipLevelCodeBronze,
		Name:        "黄铜会员",
		Color:       "#a15a2b",
		Description: "系统固定黄铜会员等级",
		SortOrder:   10,
		IsDefault:   true,
	},
	{
		Code:        MembershipLevelCodeSilver,
		Name:        "白银会员",
		Color:       "#94a3b8",
		Description: "系统固定白银会员等级",
		SortOrder:   20,
	},
	{
		Code:        MembershipLevelCodeGold,
		Name:        "黄金会员",
		Color:       "#d99a00",
		Description: "系统固定黄金会员等级",
		SortOrder:   30,
	},
	{
		Code:        MembershipLevelCodePlatinum,
		Name:        "铂金会员",
		Color:       "#b89a56",
		Description: "系统固定铂金会员等级",
		SortOrder:   40,
	},
	{
		Code:        MembershipLevelCodeDiamond,
		Name:        "钻石会员",
		Color:       "#0ea5e9",
		Description: "系统固定钻石会员等级",
		SortOrder:   50,
	},
}

func FixedMembershipLevelDefinitions() []MembershipLevelDefinition {
	out := make([]MembershipLevelDefinition, len(fixedMembershipLevelDefinitions))
	copy(out, fixedMembershipLevelDefinitions)
	return out
}

func fixedMembershipLevelByCode(code string) (MembershipLevelDefinition, bool) {
	for _, def := range fixedMembershipLevelDefinitions {
		if def.Code == code {
			return def, true
		}
	}
	return MembershipLevelDefinition{}, false
}

func applyFixedMembershipLevelDefinition(input *MembershipLevelInput, def MembershipLevelDefinition) {
	input.Name = def.Name
	input.Code = def.Code
	input.Color = def.Color
	input.Enabled = true
	input.IsDefault = def.IsDefault
	input.SortOrder = def.SortOrder
	input.Description = def.Description
}
