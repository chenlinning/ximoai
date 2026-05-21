package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/cespare/xxhash/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

const (
	// ChatGPT internal API for OAuth accounts
	chatgptCodexURL = "https://chatgpt.com/backend-api/codex/responses"
	// OpenAI Platform API for API Key accounts (fallback)
	openaiPlatformAPIURL   = "https://api.openai.com/v1/responses"
	openaiStickySessionTTL = time.Hour // 缂傚倸鍊搁崐椋庢閿熺姴绐楁俊銈呭閹冲矂姊绘笟鈧埀顒傚仜閼活垶宕濋鐐寸厱閻忕偛澧介惌宀勬煟濞戞牕鍔﹂柡宀嬬秮閹晠宕ｆ径濠庢П缂傚倷娴囨ご鎼佸煘瀵?
	codexCLIUserAgent      = "codex_cli_rs/0.125.0"
	// Maximum header value length included in Codex CLI debug logs.
	codexCLIOnlyHeaderValueMaxBytes = 256

	// OpenAIParsedRequestBodyKey 缂傚倸鍊搁崐鎼佸磹閹间礁纾圭憸鐗堝笒缁犱即鏌熼梻瀵稿妽闁?handler 濠电姷鏁搁崑鐐哄箰婵犳碍鍎嶆繝濠傜墕缁€鍐煏婵炲灝鍔滄い鎾炽偢濮婃椽鎳￠妶鍛€鹃梺鑽ゅ枙娴滎剙顕ユ繝鍐﹀亝闁告劏鏅涢埀顒勬涧闇夐柣妯烘▕閸庢劙鏌涙惔锛勑ч柡灞剧洴瀵挳濡搁妷锔惧蒋闂備焦鎮堕崐妤呭磻閻愬搫绠為柕濞炬櫆閸嬶繝鏌℃径濠勬皑闁稿鎸婚ˇ鐗堟償閿濆浂妲搁梺璇插嚱缂嶅棝宕板Δ鍛柧妞ゆ帒鍊甸崑鎾诲礂婢跺﹣澹曢梻浣告啞濞诧附绂嶉悙鍨潟婵鍩栭埛鎴犵磼鐎ｎ厽纭剁紒鐘冲缁辨帗寰勬繝鍕ㄩ悗瑙勬礃婵炲﹤鐣烽妸锔剧瘈闁稿本绮庨悰顕€鏌ｆ惔锛勭暛闁稿酣浜堕幃銏ゅ幢濞戞顔婂┑掳鍊愰崑鎾绘煕閹寸姴鈻堥柡宀€鍠栭獮宥夋惞椤愶絾绶梻?
	OpenAIParsedRequestBodyKey = "openai_parsed_request_body"
	// OpenAI WS Mode 濠电姷鏁告慨浼村垂濞差亜纾块柤娴嬫櫅閸ㄦ繈鏌涢幘妤€瀚弸鍌炴⒑閹稿孩绀€闁稿﹤鎽滈幉鎾晝閸屾氨顔愰柡澶婄墕婢х晫绮欐繝姘厽闁挎繂顦幉鐐叏婵犲啯銇濇鐐存崌楠炴帡寮┑鍕姢缂佽鲸甯炵槐鎺懳熺亸鏍潔闂備礁鐤囬褔鏌婇敐澶婃槬闁告洦鍨扮粈鍐煃閸濆嫬浜為柛鐔插亾濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸劍閺呮繃銇勮箛鎾愁伌闁诲氦鍩栫换娑橆啅椤旇崵鍑归梺鍝勬媼閸撶喖寮诲☉銏╂晝闁挎繂娲ㄩ悡鍌炴⒑閼姐倕鏋戠紒璇茬墦瀵鈽夊鍡樺兊闂佺粯鎸哥花濂稿窗婵犲偆娓婚柕鍫濆€瑰▍鍫熴亜閺囧棗娲ら悡婵嬪箹濞ｎ剙濡奸柣蹇撶－閳ь剝顫夊ú鏍洪敃鍌ゆ晪婵炴垯鍨洪悡鐔兼煟濡搫绾у璺哄缁绘稑霉鐎ｎ偅鐝栧┑鈥冲级閸旀洟锝炲┑鍫熷磯濞?	// 濠?Codex 闂傚倷娴囬褎顨ラ崫銉т笉鐎广儱顦崹鍌涚箾瀹割喕绨婚柡鍕╁劜缁绘盯骞嬮悙瀵告闂佸憡顨嗙喊宥夊Φ閸曨垰鍐€闁靛ě鍕珮缂傚倷鑳舵慨宕囧垝濞嗘挸钃熼柣鏂垮悑閳锋劙鏌熼柇锕€寮炬俊顖氬€荤槐鎾存媴鐟欏嫧鎷归梺鐟版啞婵炲﹪鐛繝鍌楁婵炲棙鍔楃粔鍫曟⒑閸涘﹥瀵欓柛娑卞灠閻愬﹥绻濋悽闈浶為柛銊︽そ瀹曟洟鎳栭埡浣哥亰闂佸憡鎸嗛崟顐ｇ€梻浣瑰濞叉牠宕愰崫銉﹀枂闁挎洖鍊哥粻瑙勭箾閿濆骸澧柣蹇ｄ簻闇夐柣妯跨М椤忓牃鈧箓宕稿Δ浣告疂闂傚倸鐗婄粙鎴﹀箺閻㈠憡鈷戦梻鍫氭櫇閸掍即鏌熷ù瀣у亾閹颁胶鍔?5 婵犵數濮烽弫鎼佸磻濞戙垹鍑犻柟缁㈠枛缁犮儵鏌涢幇顖氱处濠?
	openAIWSReconnectRetryLimit = 5
	// OpenAI WS Mode 闂傚倸鍊搁崐鐑芥倿閿曚降浜归柛鎰典簽閻挸顪冪€ｎ亜顒㈢€规洖寮剁换娑㈠箣濞嗗繒浠鹃梺缁樺笒椤兘寮诲鍫闂佸憡鎸鹃崰鏍ь嚕婵犳艾骞㈡繛鎴烇耿閺佹粌鈹戞幊閸婃洟宕銈囩當闁跨喓濮甸悡鐔兼煟濮椻偓娴滆泛顬婇悜鑺ョ厱濠电姴鍟粈瀣煕閳规儳浜炬俊鐐€栫敮鎺楀磹閸洖鐒垫い鎺嗗亾妞ゆ垵娲ゅ嵄闁归偊鍏橀弸搴ㄦ煙閹咃紞闁绘挻鍨甸—鍐Χ閸℃瑥顫х紓渚囧枛缁夌懓顕ｉ幎鑺ュ亹缂備焦菤閹锋椽姊洪崫鍕垫Ъ婵炲娲濋崐鎾⒒娴ｅ湱婀介柛濠冩礀铻炴繛鎴欏灩缁犵喎霉閸忓吋缍戞鐐灪缁绘盯宕卞Ο鍝勵潕闂佹寧绋掗崝娆忣潖濞差亜绠伴幖杈剧岛濡插牆顪冮妶蹇撶槣闁告瑥鍟村畷娲焵?
	openAIWSRetryBackoffInitialDefault = 120 * time.Millisecond
	openAIWSRetryBackoffMaxDefault     = 2 * time.Second
	openAIWSRetryJitterRatioDefault    = 0.2
	openAICompactSessionSeedKey        = "openai_compact_session_seed"
	codexCLIVersion                    = "0.125.0"
	// Codex 闂傚倸鍊搁崐鎼佸磹閸濄儮鍋撳鐓庡籍鐎规洘绻傞埢搴ㄥ箻閳ь剟鎮滈挊澶岋紲濠电娀娼ч悧蹇涘级缁嬫娓婚柕鍫濇婵倿鏌涙繝鍐╃妞ゆ洩缍侀獮鍡涒€栭垾宕囩Ш闁诡喒鏅犲畷锝嗗緞婵犲喚浠遍梻鍌欐祰濡椼劑姊藉澶婄９闁哄稁鍋嗛惌澶屸偓骞垮劚濡瑩宕曢悢鍏肩厓闁宠桨绀侀弳娆撴煃瑜滈崜娑㈠极婵犳艾钃熸繛鎴炃氶弸搴ㄦ煙闁箑骞橀柛鎴節濮婂搫煤鐠囪弓绨荤紓浣割槸婢у酣宕?闂傚倷娴囧畷鍨叏閺夋嚚娲Χ婢跺娅囬梺闈涱槴閺呮盯鎷戦悢鍏肩厱闁靛鍠栨晶顖炴煛閸滀礁澧撮柟顔斤耿閹瑧鎹勬潪鐗堢潖闂備浇宕甸崰鎰崲閸繍娼栧┑鐘宠壘閻愬﹪鏌ㄥ┑鍡樺櫢濠㈣娲熷娲嚒閵堝懏鐎惧┑鐐插级閸ㄥ潡骞冮悙鐑樻櫇闁稿本绋戦崜銊╂⒑閺傘儲娅呴柛鐔跺嵆閸┿垽寮埀顒勫Φ閸曨喚鐤€闁圭偓娼欏▍锝夋⒑鐠囪尙绠版繛宸弮瀵鈽夐姀鐘电潉闂佺鏈喊宥夋倶閸℃娓婚柕鍫濇婢跺嫰鏌涢幘瀵告噭闁哄懌鍎靛铏圭矙鐠恒劎浼囬梺绋款儐閻╊垰鐣烽姀鈶╁亾闂堟稒鎲哥痪鎯с偢閺岋繝宕掑☉鍗炲妼闂佺楠哥换鎺楀焵椤掑喚娼愰柟鍝ヮ焾铻炴繝闈涱儏缁狀垶鏌涢幇闈涙灍閻庣數濮撮…璺ㄦ崉閻戞ɑ鎷遍梺姹囧妽閸ㄥ灝顫?
	openAICodexSnapshotPersistMinInterval = 30 * time.Second
)

// OpenAI allowed headers whitelist (for non-passthrough).
var openaiAllowedHeaders = map[string]bool{
	"accept-language":       true,
	"content-type":          true,
	"conversation_id":       true,
	"user-agent":            true,
	"originator":            true,
	"session_id":            true,
	"x-codex-turn-state":    true,
	"x-codex-turn-metadata": true,
}

// OpenAI passthrough allowed headers whitelist.
// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ幗瀹曟煡鎮楅敐搴″妞ゎ偅娲熼幃瑙勬姜閹殿喚协闂佺粯姊婚崢褔鎮樺畷鍥ｅ亾鐟欏嫭绀€婵炲眰鍔戦妴渚€宕ㄩ鍓ь啎闂佺懓顕崑鐐典焊椤撶喆浜滈柟瀛樼箓椤忣偊鏌涢幒鎾崇瑲闁诡垱妫冩俊鎼佸Ψ瑜嶉幗瀣⒒娴ｄ警鏀版い顐㈩樀瀹曟繈鏁冮崒姘卞摋婵炲濮撮鍛棯瑜旈弻宥夊传閸曨偀鍋撴繝姘疇濞达綀娅ｇ壕濂告煏婵炑冨暙婵℃椽姊洪悷鏉挎Щ闁瑰啿绻橀獮鍡涘籍閸埃鍋撻敃鍌氱闁绘垵妫欓ˉ娑㈡⒒閸屾艾鈧悂宕愰崫銉㈠亾濮樷偓閸パ呮煣闂佺粯鍔掔拃锕傚极鐎ｎ偁浜滈柟鍝勬娴滈箖鏌℃担鍝ュⅵ闁哄本绋撴禒锕傚箚瑜庨幆娑欑箾鐎电孝闁绘鎹囧濠氬Ω閵夈垺顫嶅┑鈽嗗灥閸嬫劙宕濋娑氱闁哄鍨瑰В鐐烘煛閸涱垰孝闁伙絽鍢茶灃闁逞屽墴椤㈡ɑ绺界粙鍨獩濡炪倖鐗楁笟妤呭疮閸儲鈷掗柛灞捐壘閳ь剚鎮傞幃褑绠涘☉妯硷紱闂佺粯鍔楅崕銈夊磻婢舵劖鐓ユ繝闈涙－濡插摜绱掗埀?闂傚倸鍊烽懗鍓佸垝椤栨粌鍨濋柣妯款嚙閸ㄥ倸霉閸忚偐鏆橀柍褜鍓欓崐鎸庝繆閹间礁鐓涘〒姘ｅ亾婵☆偄鍢查—鍐Χ閸涱垳顔囬柣搴㈣壘閹芥粌危閹扮増鏅查柛婊€鑳堕崬鐢告⒑閸涘﹥灏柤鍐插瀵囧焵椤掑倻纾藉ù锝囶焾閼哥懓顭胯濞村嘲危閹版澘绠抽柟瀛樻⒐閻庡鏌熼懝鐗堝涧缂佹彃娼￠獮澶愵敂閸曘劍鏂€闂佺粯鍔曢悺銊т焊娴煎瓨鐓熼柍鈺佸暞缁€澶嬨亜閵堝懎鈧灝顫忓ú顏勫窛濠电姴瀚崳顓㈡⒑閹肩偛濡兼繛鎮搞倖锛傞梻浣筋潐瀹曟﹢顢氳缁鐣烽崶銊ユ瀾闁诲函缍嗛崰鏍倿?
var openaiPassthroughAllowedHeaders = map[string]bool{
	"accept":                true,
	"accept-language":       true,
	"content-type":          true,
	"conversation_id":       true,
	"openai-beta":           true,
	"user-agent":            true,
	"originator":            true,
	"session_id":            true,
	"x-codex-turn-state":    true,
	"x-codex-turn-metadata": true,
}

// codex_cli_only 闂傚倸鍊风粈浣虹礊婵犲洤缁╅弶鍫氭櫆閺嗘粓鏌ｉ幇鐗堟锭闁搞劍绻堥弻锝夊箣閿濆棭妫勭紓渚囧亜缁夊綊寮婚悢纰辨晬闁绘劘寮撻崰濠囨⒑閸濄儱浠滃褍閰ｉ獮澶愬箹娴ｇ懓浜遍梺鍓插亞閸犳捇藝椤撶偐鏀介柣鎰皺婢ф盯鏌涢妸銉хШ妤犵偛绻橀幃鈺冪磼濡厧寮抽梺璇插嚱缂嶅棝宕滃璺虹闁绘劦鍓涚弧鈧梺闈涢獜缁插墽娑甸悙顑句簻闁哄洢鍔岄獮鏍煃鐠囪尙啸妞わ箑婀遍埀顒冾潐濞插繘宕曢柆宥庢晣濠靛倻顭堝婵囥亜閹捐泛顎屾俊鎻掔秺濮婄粯鎷呴悷閭﹀殝缂備礁顑嗛崹鍧楀箖瑜斿畷銊╊敍濮橆剛浜伴梺鐟板悑閻ｎ亪宕濆澶婄；闁靛ň鏅滈悡蹇撯攽閻愯尙浠㈤柍褜鍓氶〃濠囧箠濠靛洢鍋呴柛鎰ㄦ櫅娴狀垶姊洪幖鐐插妧鐎广儱妫欓弳浼存⒒娴ｅ憡鎯堥柤娲诲灠鐓ゆい鎾跺剱閸ゆ洘銇勯弴妤€浜炬繝纰夌磿閸忔ɑ淇婇悿顖ｆЬ濡炪倖鏌ㄧ粔鐟邦潖濞差亜绠伴幖杈剧岛濡插牊绻濋棃娑樷偓鐟邦潖閼姐倕鍨濇い鎾跺枎缁剁偤鏌熼柇锕€澧伴柣鎾村灴濮婅櫣绮欓幐搴㈡嫳闂佽崵鍟欓崶銊ヤ画闂侀潧绻掓慨顓炍ｉ崼銉︾厵闁告挆灞藉闂佽　鍋撳┑鐘崇閻撶喖鏌″畵顔兼噹閻撶喖姊洪崫鍕闁硅櫕鎸剧划璇测槈濞嗘劕鍔呴梺鎸庣箓濞层倝鍩€椤掑倸浠辨慨?
var codexCLIOnlyDebugHeaderWhitelist = []string{
	"User-Agent",
	"Content-Type",
	"Accept",
	"Accept-Language",
	"OpenAI-Beta",
	"Originator",
	"Session_ID",
	"Conversation_ID",
	"X-Request-ID",
	"X-Client-Request-ID",
	"X-Forwarded-For",
	"X-Real-IP",
}

// OpenAICodexUsageSnapshot represents Codex API usage limits from response headers
type OpenAICodexUsageSnapshot struct {
	PrimaryUsedPercent          *float64 `json:"primary_used_percent,omitempty"`
	PrimaryResetAfterSeconds    *int     `json:"primary_reset_after_seconds,omitempty"`
	PrimaryWindowMinutes        *int     `json:"primary_window_minutes,omitempty"`
	SecondaryUsedPercent        *float64 `json:"secondary_used_percent,omitempty"`
	SecondaryResetAfterSeconds  *int     `json:"secondary_reset_after_seconds,omitempty"`
	SecondaryWindowMinutes      *int     `json:"secondary_window_minutes,omitempty"`
	PrimaryOverSecondaryPercent *float64 `json:"primary_over_secondary_percent,omitempty"`
	UpdatedAt                   string   `json:"updated_at,omitempty"`
}

// NormalizedCodexLimits contains normalized 5h/7d rate limit data
type NormalizedCodexLimits struct {
	Used5hPercent   *float64
	Reset5hSeconds  *int
	Window5hMinutes *int
	Used7dPercent   *float64
	Reset7dSeconds  *int
	Window7dMinutes *int
}

// Normalize converts primary/secondary fields to canonical 5h/7d fields.
// Strategy: Compare window_minutes to determine which is 5h vs 7d.
// Returns nil if snapshot is nil or has no useful data.
func (s *OpenAICodexUsageSnapshot) Normalize() *NormalizedCodexLimits {
	if s == nil {
		return nil
	}

	result := &NormalizedCodexLimits{}

	primaryMins := 0
	secondaryMins := 0
	hasPrimaryWindow := false
	hasSecondaryWindow := false

	if s.PrimaryWindowMinutes != nil {
		primaryMins = *s.PrimaryWindowMinutes
		hasPrimaryWindow = true
	}
	if s.SecondaryWindowMinutes != nil {
		secondaryMins = *s.SecondaryWindowMinutes
		hasSecondaryWindow = true
	}

	// Determine mapping based on window_minutes
	use5hFromPrimary := false
	use7dFromPrimary := false

	if hasPrimaryWindow && hasSecondaryWindow {
		// Both known: smaller window is 5h, larger is 7d
		if primaryMins < secondaryMins {
			use5hFromPrimary = true
		} else {
			use7dFromPrimary = true
		}
	} else if hasPrimaryWindow {
		// Only primary known: classify by threshold (<=360 min = 6h -> 5h window)
		if primaryMins <= 360 {
			use5hFromPrimary = true
		} else {
			use7dFromPrimary = true
		}
	} else if hasSecondaryWindow {
		// Only secondary known: classify by threshold
		if secondaryMins <= 360 {
			// 5h from secondary, so primary (if any data) is 7d
			use7dFromPrimary = true
		} else {
			// 7d from secondary, so primary (if any data) is 5h
			use5hFromPrimary = true
		}
	} else {
		// No window_minutes: fall back to legacy assumption (primary=7d, secondary=5h)
		use7dFromPrimary = true
	}

	// Assign values
	if use5hFromPrimary {
		result.Used5hPercent = s.PrimaryUsedPercent
		result.Reset5hSeconds = s.PrimaryResetAfterSeconds
		result.Window5hMinutes = s.PrimaryWindowMinutes
		result.Used7dPercent = s.SecondaryUsedPercent
		result.Reset7dSeconds = s.SecondaryResetAfterSeconds
		result.Window7dMinutes = s.SecondaryWindowMinutes
	} else if use7dFromPrimary {
		result.Used7dPercent = s.PrimaryUsedPercent
		result.Reset7dSeconds = s.PrimaryResetAfterSeconds
		result.Window7dMinutes = s.PrimaryWindowMinutes
		result.Used5hPercent = s.SecondaryUsedPercent
		result.Reset5hSeconds = s.SecondaryResetAfterSeconds
		result.Window5hMinutes = s.SecondaryWindowMinutes
	}

	return result
}

// OpenAIUsage represents OpenAI API response usage
type OpenAIUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	ImageOutputTokens        int `json:"image_output_tokens,omitempty"`
}

// OpenAIForwardResult represents the result of forwarding
type OpenAIForwardResult struct {
	RequestID  string
	ResponseID string
	Usage      OpenAIUsage
	Model      string // 闂傚倸鍊风粈渚€骞夐敓鐘偓锕傚炊椤掆偓缁愭骞栭幖顓犲帨缂傚秵鐗犻弻鐔兼焽閿曗偓閸旓箓鏌ｉ弮鍌氬付闁诲繐纾埀顒冾潐濞叉牕煤閻樿纾婚柟鍓х帛閸婂鏌ら幁鎺戝姕婵炲懏鐗犲娲礈閹绘帊绨撮梺鎼炲妽濡炶棄顕ｇ€圭姷鐤€闁瑰彞鑳堕幊鎾烩€﹂妸鈺佺妞ゆ洖鎳夐崑鎾澄旈崨顔惧幍婵犻潧鍊搁幉锟犲极闁秵鐓欐い鏍ㄧ⊕椤ュ牏鈧鍣崜鐔肩嵁閹邦厽鍎熼柍钘夋缁辨牠姊婚崒娆戭槮闁圭⒈鍋婂畷顖炲Ω閳轰胶锛熼柡澶婄墐閺呮粌鐣烽弻銉︾厸闁告劑鍔庢晶鏇㈡煕濞嗗骏韬柡灞剧洴楠炴ê顪冮悙顒夋▊缂備緡鍠氶弫璇差潖?	// BillingModel is the model used for cost calculation.
	// When non-empty, CalculateCost uses this instead of Model.
	// This is set by the Anthropic Messages conversion path where
	// the mapped upstream model differs from the client-facing model.
	BillingModel string
	// UpstreamModel is the actual model sent to the upstream provider after mapping.
	// Empty when no mapping was applied (requested model was used as-is).
	UpstreamModel string
	// ServiceTier records the OpenAI Responses API service tier, e.g. "priority" / "flex".
	// Nil means the request did not specify a recognized tier.
	ServiceTier *string
	// ReasoningEffort is extracted from request body (reasoning.effort) or derived from model suffix.
	// Stored for usage records display; nil means not provided / not applicable.
	ReasoningEffort    *string
	Stream             bool
	OpenAIWSMode       bool
	ResponseHeaders    http.Header
	Duration           time.Duration
	FirstTokenMs       *int
	VideoCount         int
	ImageCount         int
	ImageSize          string
	ImageInputSize     string
	ImageOutputSize    string
	ImageOutputSizes   []string
	ImageSizeSource    string
	ImageSizeBreakdown map[string]int
}

type OpenAIWSRetryMetricsSnapshot struct {
	RetryAttemptsTotal            int64 `json:"retry_attempts_total"`
	RetryBackoffMsTotal           int64 `json:"retry_backoff_ms_total"`
	RetryExhaustedTotal           int64 `json:"retry_exhausted_total"`
	NonRetryableFastFallbackTotal int64 `json:"non_retryable_fast_fallback_total"`
}

type OpenAICompatibilityFallbackMetricsSnapshot struct {
	SessionHashLegacyReadFallbackTotal int64   `json:"session_hash_legacy_read_fallback_total"`
	SessionHashLegacyReadFallbackHit   int64   `json:"session_hash_legacy_read_fallback_hit"`
	SessionHashLegacyDualWriteTotal    int64   `json:"session_hash_legacy_dual_write_total"`
	SessionHashLegacyReadHitRate       float64 `json:"session_hash_legacy_read_hit_rate"`

	MetadataLegacyFallbackIsMaxTokensOneHaikuTotal int64 `json:"metadata_legacy_fallback_is_max_tokens_one_haiku_total"`
	MetadataLegacyFallbackThinkingEnabledTotal     int64 `json:"metadata_legacy_fallback_thinking_enabled_total"`
	MetadataLegacyFallbackPrefetchedStickyAccount  int64 `json:"metadata_legacy_fallback_prefetched_sticky_account_total"`
	MetadataLegacyFallbackPrefetchedStickyGroup    int64 `json:"metadata_legacy_fallback_prefetched_sticky_group_total"`
	MetadataLegacyFallbackSingleAccountRetryTotal  int64 `json:"metadata_legacy_fallback_single_account_retry_total"`
	MetadataLegacyFallbackAccountSwitchCountTotal  int64 `json:"metadata_legacy_fallback_account_switch_count_total"`
	MetadataLegacyFallbackTotal                    int64 `json:"metadata_legacy_fallback_total"`
}

type openAIWSRetryMetrics struct {
	retryAttempts            atomic.Int64
	retryBackoffMs           atomic.Int64
	retryExhausted           atomic.Int64
	nonRetryableFastFallback atomic.Int64
}

type accountWriteThrottle struct {
	minInterval time.Duration
	mu          sync.Mutex
	lastByID    map[int64]time.Time
}

func newAccountWriteThrottle(minInterval time.Duration) *accountWriteThrottle {
	return &accountWriteThrottle{
		minInterval: minInterval,
		lastByID:    make(map[int64]time.Time),
	}
}

func (t *accountWriteThrottle) Allow(id int64, now time.Time) bool {
	if t == nil || id <= 0 || t.minInterval <= 0 {
		return true
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if last, ok := t.lastByID[id]; ok && now.Sub(last) < t.minInterval {
		return false
	}
	t.lastByID[id] = now

	if len(t.lastByID) > 4096 {
		cutoff := now.Add(-4 * t.minInterval)
		for accountID, writtenAt := range t.lastByID {
			if writtenAt.Before(cutoff) {
				delete(t.lastByID, accountID)
			}
		}
	}

	return true
}

var defaultOpenAICodexSnapshotPersistThrottle = newAccountWriteThrottle(openAICodexSnapshotPersistMinInterval)

// ErrNoAvailableCompactAccounts indicates the request needs /responses/compact
// support but no compatible account is available.
var ErrNoAvailableCompactAccounts = errors.New("no available OpenAI accounts support /responses/compact")

// OpenAIGatewayService handles OpenAI API gateway operations
type OpenAIGatewayService struct {
	accountRepo           AccountRepository
	usageLogRepo          UsageLogRepository
	usageBillingRepo      UsageBillingRepository
	userRepo              UserRepository
	userSubRepo           UserSubscriptionRepository
	cache                 GatewayCache
	cfg                   *config.Config
	codexDetector         CodexClientRestrictionDetector
	schedulerSnapshot     *SchedulerSnapshotService
	concurrencyService    *ConcurrencyService
	billingService        *BillingService
	rateLimitService      *RateLimitService
	billingCacheService   *BillingCacheService
	userGroupRateResolver *userGroupRateResolver
	httpUpstream          HTTPUpstream
	deferredService       *DeferredService
	openAITokenProvider   *OpenAITokenProvider
	toolCorrector         *CodexToolCorrector
	openaiWSResolver      OpenAIWSProtocolResolver
	resolver              *ModelPricingResolver
	channelService        *ChannelService
	balanceNotifyService  *BalanceNotifyService
	settingService        *SettingService
	videoJobRepo          OpenAIVideoJobRepository

	openaiWSPoolOnce              sync.Once
	openaiWSStateStoreOnce        sync.Once
	openaiSchedulerOnce           sync.Once
	openaiWSPassthroughDialerOnce sync.Once
	openaiWSPool                  *openAIWSConnPool
	openaiWSStateStore            OpenAIWSStateStore
	openaiScheduler               OpenAIAccountScheduler
	openaiWSPassthroughDialer     openAIWSClientDialer
	openaiAccountStats            *openAIAccountRuntimeStats

	openaiWSFallbackUntil               sync.Map // key: int64(accountID), value: time.Time
	openaiWSRetryMetrics                openAIWSRetryMetrics
	responseHeaderFilter                *responseheaders.CompiledHeaderFilter
	codexSnapshotThrottle               *accountWriteThrottle
	openaiCompatSessionResponses        sync.Map
	openaiCompatAnthropicDigestSessions sync.Map
}

// NewOpenAIGatewayService creates a new OpenAIGatewayService
func NewOpenAIGatewayService(
	accountRepo AccountRepository,
	usageLogRepo UsageLogRepository,
	usageBillingRepo UsageBillingRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	userGroupRateRepo UserGroupRateRepository,
	cache GatewayCache,
	cfg *config.Config,
	schedulerSnapshot *SchedulerSnapshotService,
	concurrencyService *ConcurrencyService,
	billingService *BillingService,
	rateLimitService *RateLimitService,
	billingCacheService *BillingCacheService,
	httpUpstream HTTPUpstream,
	deferredService *DeferredService,
	openAITokenProvider *OpenAITokenProvider,
	resolver *ModelPricingResolver,
	channelService *ChannelService,
	balanceNotifyService *BalanceNotifyService,
	settingService *SettingService,
	videoJobRepo ...OpenAIVideoJobRepository,
) *OpenAIGatewayService {
	var openAIVideoJobRepo OpenAIVideoJobRepository
	if len(videoJobRepo) > 0 {
		openAIVideoJobRepo = videoJobRepo[0]
	}
	svc := &OpenAIGatewayService{
		accountRepo:         accountRepo,
		usageLogRepo:        usageLogRepo,
		usageBillingRepo:    usageBillingRepo,
		userRepo:            userRepo,
		userSubRepo:         userSubRepo,
		cache:               cache,
		cfg:                 cfg,
		codexDetector:       NewOpenAICodexClientRestrictionDetector(cfg),
		schedulerSnapshot:   schedulerSnapshot,
		concurrencyService:  concurrencyService,
		billingService:      billingService,
		rateLimitService:    rateLimitService,
		billingCacheService: billingCacheService,
		userGroupRateResolver: newUserGroupRateResolver(
			userGroupRateRepo,
			nil,
			resolveUserGroupRateCacheTTL(cfg),
			nil,
			"service.openai_gateway",
		),
		httpUpstream:          httpUpstream,
		deferredService:       deferredService,
		openAITokenProvider:   openAITokenProvider,
		toolCorrector:         NewCodexToolCorrector(),
		openaiWSResolver:      NewOpenAIWSProtocolResolver(cfg),
		resolver:              resolver,
		channelService:        channelService,
		balanceNotifyService:  balanceNotifyService,
		settingService:        settingService,
		videoJobRepo:          openAIVideoJobRepo,
		responseHeaderFilter:  compileResponseHeaderFilter(cfg),
		codexSnapshotThrottle: newAccountWriteThrottle(openAICodexSnapshotPersistMinInterval),
	}
	svc.logOpenAIWSModeBootstrap()
	return svc
}

// ResolveChannelMapping 闂傚倷娴囧畷鐢稿窗閹扮増鍋￠弶鍫氭櫅缁躲倕螖閿濆懎鏆為柛濠勬暬閺岋箑螣娓氼垱肖闂侀€炲苯澧悽顖ょ節楠炲啴鍩￠崘鈺侇€涢梺鍛婂姇濡﹤螞濠婂懐纾介柛灞剧懆閸忓矂鎮楀顒傜鐎规洜鍠栭崺鈧い鎺嶈兌閻熷綊鏌嶈閸撴瑩鎮鹃悜钘夋嵍妞ゆ挻绋戞禍楣冩煥濠靛棝顎楀ù婊勭箘閹増绺介崨濠勫幗闂婎偄娲﹂幐鍓х不娴煎瓨鍊垫慨姗嗗墯閹插憡淇婇崣澶婂闁宠閰ｉ獮鍥敋閸涱喚銈梻鍌欑劍鐎笛呮崲閸屾娲晝閸屾氨锛欓梺缁樺姉閸庛倝鎮?ChannelService闂?
func (s *OpenAIGatewayService) ResolveChannelMapping(ctx context.Context, groupID int64, model string) ChannelMappingResult {
	if s.channelService == nil {
		return ChannelMappingResult{MappedModel: model}
	}
	return s.channelService.ResolveChannelMapping(ctx, groupID, model)
}

// IsModelRestricted 婵犵數濮烽。钘壩ｉ崨鏉戠；闁逞屽墴閺屾稓鈧綆鍋呭畷宀勬煛瀹€瀣？濞寸媴濡囬幏鐘诲箵閹烘埈娼ラ梻浣哄帶閻忓牓寮查悩璇茶摕婵炴垯鍨规儫闂侀潧顦介崰鏍礈閸楃偐鏀介柍銉ュ暱缁狙囨煙椤旂厧鈧悂鎮鹃悜钘壩ㄩ柍鍝勫€瑰▍鍥ㄧ箾閹剧澹樻繛璇х畵閹风儤寰勬繛鐐杸闂佺粯鍔曢悺銊т焊闂堟稈鏀介柣鎰ㄦ櫅娴滄儳鈹戦悙鑼憼缂侇喗鎸剧划濠氬冀瑜滃鏍归悩宸剰缂佺姵绋掗妵鍕箳閸℃ぞ澹曢梻浣虹帛閹稿宕归崸妤€钃熼柕濞垮劗濡插牊淇婇婊冨付妞わ絽缍婂娲传閵夈儱澹夐梺鍛婃尵閸犳牠鐛崘顔奸唶闁哄洨鍋涢懓鍨攽鎺抽崐鎰板磻閹剧粯鐓?ChannelService闂?
func (s *OpenAIGatewayService) IsModelRestricted(ctx context.Context, groupID int64, model string) bool {
	if s.channelService == nil {
		return false
	}
	return s.channelService.IsModelRestricted(ctx, groupID, model)
}

// ResolveChannelMappingAndRestrict 闂傚倷娴囧畷鐢稿窗閹扮増鍋￠弶鍫氭櫅缁躲倕螖閿濆懎鏆為柛濠勬暬閺岋箑螣娓氼垱肖闂侀€炲苯澧悽顖ょ節楠炲啴鍩￠崘鈺侇€涢梺鍛婂姇濡﹤螞濠婂牊鈷掑ù锝呮啞閹牓鎮楀鐓庢灍闁逛究鍔戦獮姗€顢欓挊澶婂闂備胶绮…鍫ヮ敋瑜戠换?// 婵犵數濮烽。钘壩ｉ崨鏉戝瀭妞ゅ繐鐗嗛悞鍨亜閹哄棗浜剧紒鍓ц檸閸欏啴宕洪埀顒併亜閹烘垵鈧悂宕㈢€涙ɑ鍙忓┑鐘辫兌閻瑦顨ラ悙鍙夊枠妞ゃ垺顨婇崺鈧い鎺戝閸嬪倿鏌熺€电浠掔紒璇叉閵囧嫰骞囬鍏肩€绘繛瀛樼矒缁犳牕顫忓ú顏勪紶闁告洦鍣鍫曟⒑缁嬪潡鍙勫ù婊嗘硾閻ｅ嘲鈻庨幘瀛樻珫闂佸憡娲﹂崢楣冩偂鐎ｎ喗鈷戦柛婵嗗椤忊晝绱掔紒妯哄闁崇粯鎸婚妶锝夊礃閳轰椒鍖栭梺璇插嚱缂嶅棙绂嶅鍫濇辈婵炲棙鎸婚崐鐢告偡濞嗗繐顏╅柍缁樻礃閹便劍绻濋崒妯轰划闂佺娅曠划宀勫煝閹捐鍨傛い鏃囧Г濠㈡垿姊婚崒娆戝妽閻庣瑳鍛煓闁硅揪瀵岄弫濠囨煙鏉堚晝鏆塼ricted 濠电姷鏁告慨鐢割敊閺嶎厼绐楅柡宥庡弾閺佸嫰鏌涢妷銏℃珔闁搞劍绻堥弻鐔煎箚瑜忛幗鐘裁瑰┃鍨偓婵嬪蓟閿熺姴纾兼繛鎴烇供閸ゅ绱撴担鎻掍壕闂?false闂?
func (s *OpenAIGatewayService) ResolveChannelMappingAndRestrict(ctx context.Context, groupID *int64, model string) (ChannelMappingResult, bool) {
	if s.channelService == nil {
		return ChannelMappingResult{MappedModel: model}, false
	}
	return s.channelService.ResolveChannelMappingAndRestrict(ctx, groupID, model)
}

func (s *OpenAIGatewayService) isCodexImageGenerationBridgeEnabled(ctx context.Context, account *Account, apiKey *APIKey) bool {
	if override := account.CodexImageGenerationBridgeOverride(); override != nil {
		return *override
	}
	if s != nil && s.channelService != nil && apiKey != nil && apiKey.GroupID != nil {
		ch, err := s.channelService.GetChannelForGroup(ctx, *apiKey.GroupID)
		if err != nil {
			slog.Warn("failed to resolve codex image generation bridge channel override", "group_id", *apiKey.GroupID, "error", err)
		} else if override := ch.CodexImageGenerationBridgeOverride(PlatformOpenAI); override != nil {
			return *override
		}
	}
	return s != nil && s.cfg != nil && s.cfg.Gateway.CodexImageGenerationBridgeEnabled
}

func (s *OpenAIGatewayService) checkChannelPricingRestriction(ctx context.Context, groupID *int64, requestedModel string) bool {
	if groupID == nil || s.channelService == nil || requestedModel == "" {
		return false
	}
	mapping := s.channelService.ResolveChannelMapping(ctx, *groupID, requestedModel)
	billingModel := billingModelForRestriction(mapping.BillingModelSource, requestedModel, mapping.MappedModel)
	if billingModel == "" {
		return false
	}
	return s.channelService.IsModelRestricted(ctx, *groupID, billingModel)
}

func (s *OpenAIGatewayService) isUpstreamModelRestrictedByChannel(ctx context.Context, groupID int64, account *Account, requestedModel string, requireCompact bool) bool {
	if s.channelService == nil {
		return false
	}
	upstreamModel := resolveOpenAIAccountUpstreamModelForRequest(account, requestedModel, requireCompact)
	if upstreamModel == "" {
		return false
	}
	return s.channelService.IsModelRestricted(ctx, groupID, upstreamModel)
}

func (s *OpenAIGatewayService) needsUpstreamChannelRestrictionCheck(ctx context.Context, groupID *int64) bool {
	if groupID == nil || s.channelService == nil {
		return false
	}
	ch, err := s.channelService.GetChannelForGroup(ctx, *groupID)
	if err != nil {
		slog.Warn("failed to check openai channel upstream restriction", "group_id", *groupID, "error", err)
		return false
	}
	if ch == nil || !ch.RestrictModels {
		return false
	}
	return ch.BillingModelSource == BillingModelSourceUpstream
}

// ReplaceModelInBody 闂傚倸鍊风粈渚€骞栭鈷氭椽鏁傞柨顖氫壕缂佹绋戦崯浼村汲閿曞倹鐓犵痪鏉垮船婢ь垶鎮楀顓炲摵婵﹥妞藉畷褰掝敋閸涱厼澹嬫繝鐢靛Л閸嬫捇鏌℃径瀣鐟滅増甯楅弲鏌ユ煕閳╁厾顏堟儗椤曗偓濮婃椽鎮℃惔锝勭驳闂佹悶鍔屽锟犵嵁?JSON model 闂傚倷娴囬褏鈧稈鏅濈划娆撳箳濡炲皷鍋撻崘顔煎耿婵炴垼椴搁弲鈺呮倵閸忓浜鹃梺鍛婃处閸撴艾鈻嶉弽顓熲拺闁告挻褰冩禍鏍煕鎼搭喖娅嶇€规洘鍨块獮姗€骞囨担鐟板厞闂佸搫顦悧鍕礉瀹ュ應鏋?gjson/sjson 闂傚倷娴囬褎顨ョ粙鍖¤€块梺顒€绉寸壕濠氭煟閺冨洤浜圭€规挷绶氶弻娑㈠Ψ椤旂厧顫梺鍝勬媼閸撴瑩鈥︾捄銊﹀磯闁惧繒鎳撻。鐢告⒑?
func (s *OpenAIGatewayService) ReplaceModelInBody(body []byte, newModel string) []byte {
	return ReplaceModelInBody(body, newModel)
}

func (s *OpenAIGatewayService) getCodexSnapshotThrottle() *accountWriteThrottle {
	if s != nil && s.codexSnapshotThrottle != nil {
		return s.codexSnapshotThrottle
	}
	return defaultOpenAICodexSnapshotPersistThrottle
}

func (s *OpenAIGatewayService) billingDeps() *billingDeps {
	return &billingDeps{
		accountRepo:          s.accountRepo,
		userRepo:             s.userRepo,
		userSubRepo:          s.userSubRepo,
		billingCacheService:  s.billingCacheService,
		deferredService:      s.deferredService,
		balanceNotifyService: s.balanceNotifyService,
	}
}

// CloseOpenAIWSPool 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁瑰鍋熺粻鎯р攽閻樿弓杩?OpenAI WebSocket 闂傚倷绀侀幖顐λ囬锕€鐤炬繝闈涱儏绾惧鏌ｉ幇顒備粵闁哄棙绮撻弻鈩冨緞鐎ｎ亝鐦庨悷婊勬楠炲啴鍩￠崨顓炵€銈嗗姂閸婃牕鈻撻幖浣光拻濞达絽鎲￠崯鐐烘煠閸︻厼浜炬い銊ｅ劚椤垻浠︾拋铏潖?worker 闂傚倸鍊风粈渚€骞夐敍鍕灊鐎光偓閸曨剙鍓舵繝闈涘€婚…鍫ユ偂濮椻偓閺屸€愁吋鎼粹€崇闂佹娊鏀辩敮锟犲蓟閵堝鍋傞幖绮规閺嗐垻绱撴担鍝勨枅缂佺姵鐗犲濠氭偄閻撳海鐣抽梺鍦劋閸ㄥ灚鎱ㄥ畝鍕拺?// 闂傚倷绀佸﹢閬嶅储瑜旈幃娲Ω閳轰浇鎽曢梺闈涱焾閸庡鎳撻幐搴涗簻闊洦鎸婚崳娲煕鎼达絽鏋涢柡宀嬬節瀹曟﹢濡搁妷銏犱壕鐟滅増甯掑Ч鏌ョ叓閸ャ劍濯肩憸鐗堝笒缁€鍌滅棯椤撶姵鐒块柍褜鍓氱€笛呮崲濞戞瑥绶為柛顐ｇ妇閸嬫挸鈹戦崱娆愭濠德板€曢幊搴ｇ不椤曗偓閺屻倝骞侀幒鎴濆Х缂備椒绌堕崝鎴濐潖濞差亜浼犻柛鏇ㄥ亝濞堟煡姊虹粙鍧楊€楅柕鍫熸倐楠炲棝宕橀钘変簻闂佺绻掗崢褎绂掑ú顏呪拺闂傚牊渚楅悡顓犵磼閸欏銇濈€?
func (s *OpenAIGatewayService) CloseOpenAIWSPool() {
	if s != nil && s.openaiWSPool != nil {
		s.openaiWSPool.Close()
	}
}

func (s *OpenAIGatewayService) logOpenAIWSModeBootstrap() {
	if s == nil || s.cfg == nil {
		return
	}
	wsCfg := s.cfg.Gateway.OpenAIWS
	logOpenAIWSModeInfo(
		"bootstrap enabled=%v oauth_enabled=%v apikey_enabled=%v force_http=%v responses_websockets_v2=%v responses_websockets=%v payload_log_sample_rate=%.3f event_flush_batch_size=%d event_flush_interval_ms=%d prewarm_cooldown_ms=%d retry_backoff_initial_ms=%d retry_backoff_max_ms=%d retry_jitter_ratio=%.3f retry_total_budget_ms=%d ws_read_limit_bytes=%d",
		wsCfg.Enabled,
		wsCfg.OAuthEnabled,
		wsCfg.APIKeyEnabled,
		wsCfg.ForceHTTP,
		wsCfg.ResponsesWebsocketsV2,
		wsCfg.ResponsesWebsockets,
		wsCfg.PayloadLogSampleRate,
		wsCfg.EventFlushBatchSize,
		wsCfg.EventFlushIntervalMS,
		wsCfg.PrewarmCooldownMS,
		wsCfg.RetryBackoffInitialMS,
		wsCfg.RetryBackoffMaxMS,
		wsCfg.RetryJitterRatio,
		wsCfg.RetryTotalBudgetMS,
		openAIWSMessageReadLimitBytes,
	)
}

func (s *OpenAIGatewayService) getCodexClientRestrictionDetector() CodexClientRestrictionDetector {
	if s != nil && s.codexDetector != nil {
		return s.codexDetector
	}
	var cfg *config.Config
	if s != nil {
		cfg = s.cfg
	}
	return NewOpenAICodexClientRestrictionDetector(cfg)
}

func (s *OpenAIGatewayService) getOpenAIWSProtocolResolver() OpenAIWSProtocolResolver {
	if s != nil && s.openaiWSResolver != nil {
		return s.openaiWSResolver
	}
	var cfg *config.Config
	if s != nil {
		cfg = s.cfg
	}
	return NewOpenAIWSProtocolResolver(cfg)
}

func classifyOpenAIWSReconnectReason(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	var fallbackErr *openAIWSFallbackError
	if !errors.As(err, &fallbackErr) || fallbackErr == nil {
		return "", false
	}
	reason := strings.TrimSpace(fallbackErr.Reason)
	if reason == "" {
		return "", false
	}

	baseReason := strings.TrimPrefix(reason, "prewarm_")

	switch baseReason {
	case "policy_violation",
		"message_too_big",
		"upgrade_required",
		"ws_unsupported",
		"auth_failed",
		"invalid_encrypted_content",
		"previous_response_not_found":
		return reason, false
	}

	switch baseReason {
	case "read_event",
		"write_request",
		"write",
		"acquire_timeout",
		"acquire_conn",
		"conn_queue_full",
		"dial_failed",
		"upstream_5xx",
		"event_error",
		"error_event",
		"upstream_error_event",
		"ws_connection_limit_reached",
		"missing_final_response":
		return reason, true
	default:
		return reason, false
	}
}

func resolveOpenAIWSFallbackErrorResponse(err error) (statusCode int, errType string, clientMessage string, upstreamMessage string, ok bool) {
	if err == nil {
		return 0, "", "", "", false
	}
	var fallbackErr *openAIWSFallbackError
	if !errors.As(err, &fallbackErr) || fallbackErr == nil {
		return 0, "", "", "", false
	}

	reason := strings.TrimSpace(fallbackErr.Reason)
	reason = strings.TrimPrefix(reason, "prewarm_")
	if reason == "" {
		return 0, "", "", "", false
	}

	var dialErr *openAIWSDialError
	if fallbackErr.Err != nil && errors.As(fallbackErr.Err, &dialErr) && dialErr != nil {
		if dialErr.StatusCode > 0 {
			statusCode = dialErr.StatusCode
		}
		if dialErr.Err != nil {
			upstreamMessage = sanitizeUpstreamErrorMessage(strings.TrimSpace(dialErr.Err.Error()))
		}
	}

	switch reason {
	case "invalid_encrypted_content":
		if statusCode == 0 {
			statusCode = http.StatusBadRequest
		}
		errType = "invalid_request_error"
		if upstreamMessage == "" {
			upstreamMessage = "encrypted content could not be verified"
		}
	case "previous_response_not_found":
		if statusCode == 0 {
			statusCode = http.StatusBadRequest
		}
		errType = "invalid_request_error"
		if upstreamMessage == "" {
			upstreamMessage = "previous response not found"
		}
	case "upgrade_required":
		if statusCode == 0 {
			statusCode = http.StatusUpgradeRequired
		}
	case "ws_unsupported":
		if statusCode == 0 {
			statusCode = http.StatusBadRequest
		}
	case "auth_failed":
		if statusCode == 0 {
			statusCode = http.StatusUnauthorized
		}
	case "upstream_rate_limited":
		if statusCode == 0 {
			statusCode = http.StatusTooManyRequests
		}
	default:
		if statusCode == 0 {
			return 0, "", "", "", false
		}
	}

	if upstreamMessage == "" && fallbackErr.Err != nil {
		upstreamMessage = sanitizeUpstreamErrorMessage(strings.TrimSpace(fallbackErr.Err.Error()))
	}
	if upstreamMessage == "" {
		switch reason {
		case "upgrade_required":
			upstreamMessage = "upstream websocket upgrade required"
		case "ws_unsupported":
			upstreamMessage = "upstream websocket not supported"
		case "auth_failed":
			upstreamMessage = "upstream authentication failed"
		case "upstream_rate_limited":
			upstreamMessage = "upstream rate limit exceeded, please retry later"
		default:
			upstreamMessage = "Upstream request failed"
		}
	}

	if errType == "" {
		if statusCode == http.StatusTooManyRequests {
			errType = "rate_limit_error"
		} else {
			errType = "upstream_error"
		}
	}
	clientMessage = upstreamMessage
	return statusCode, errType, clientMessage, upstreamMessage, true
}

func (s *OpenAIGatewayService) writeOpenAIWSFallbackErrorResponse(c *gin.Context, account *Account, wsErr error) bool {
	if c == nil || c.Writer == nil || c.Writer.Written() {
		return false
	}
	statusCode, errType, clientMessage, upstreamMessage, ok := resolveOpenAIWSFallbackErrorResponse(wsErr)
	if !ok {
		return false
	}
	if strings.TrimSpace(clientMessage) == "" {
		clientMessage = "Upstream request failed"
	}
	if strings.TrimSpace(upstreamMessage) == "" {
		upstreamMessage = clientMessage
	}

	setOpsUpstreamError(c, statusCode, upstreamMessage, "")
	if account != nil {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: statusCode,
			Kind:               "ws_error",
			Message:            upstreamMessage,
		})
	}
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": clientMessage,
		},
	})
	return true
}

func (s *OpenAIGatewayService) openAIWSRetryBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	initial := openAIWSRetryBackoffInitialDefault
	maxBackoff := openAIWSRetryBackoffMaxDefault
	jitterRatio := openAIWSRetryJitterRatioDefault
	if s != nil && s.cfg != nil {
		wsCfg := s.cfg.Gateway.OpenAIWS
		if wsCfg.RetryBackoffInitialMS > 0 {
			initial = time.Duration(wsCfg.RetryBackoffInitialMS) * time.Millisecond
		}
		if wsCfg.RetryBackoffMaxMS > 0 {
			maxBackoff = time.Duration(wsCfg.RetryBackoffMaxMS) * time.Millisecond
		}
		if wsCfg.RetryJitterRatio >= 0 {
			jitterRatio = wsCfg.RetryJitterRatio
		}
	}
	if initial <= 0 {
		return 0
	}
	if maxBackoff <= 0 {
		maxBackoff = initial
	}
	if maxBackoff < initial {
		maxBackoff = initial
	}
	if jitterRatio < 0 {
		jitterRatio = 0
	}
	if jitterRatio > 1 {
		jitterRatio = 1
	}

	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	backoff := initial
	if shift > 0 {
		backoff = initial * time.Duration(1<<shift)
	}
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	if jitterRatio <= 0 {
		return backoff
	}
	jitter := time.Duration(float64(backoff) * jitterRatio)
	if jitter <= 0 {
		return backoff
	}
	delta := time.Duration(rand.Int63n(int64(jitter)*2+1)) - jitter
	withJitter := backoff + delta
	if withJitter < 0 {
		return 0
	}
	return withJitter
}

func (s *OpenAIGatewayService) openAIWSRetryTotalBudget() time.Duration {
	if s != nil && s.cfg != nil {
		ms := s.cfg.Gateway.OpenAIWS.RetryTotalBudgetMS
		if ms <= 0 {
			return 0
		}
		return time.Duration(ms) * time.Millisecond
	}
	return 0
}

func (s *OpenAIGatewayService) recordOpenAIWSRetryAttempt(backoff time.Duration) {
	if s == nil {
		return
	}
	s.openaiWSRetryMetrics.retryAttempts.Add(1)
	if backoff > 0 {
		s.openaiWSRetryMetrics.retryBackoffMs.Add(backoff.Milliseconds())
	}
}

func (s *OpenAIGatewayService) recordOpenAIWSRetryExhausted() {
	if s == nil {
		return
	}
	s.openaiWSRetryMetrics.retryExhausted.Add(1)
}

func (s *OpenAIGatewayService) recordOpenAIWSNonRetryableFastFallback() {
	if s == nil {
		return
	}
	s.openaiWSRetryMetrics.nonRetryableFastFallback.Add(1)
}

func (s *OpenAIGatewayService) SnapshotOpenAIWSRetryMetrics() OpenAIWSRetryMetricsSnapshot {
	if s == nil {
		return OpenAIWSRetryMetricsSnapshot{}
	}
	return OpenAIWSRetryMetricsSnapshot{
		RetryAttemptsTotal:            s.openaiWSRetryMetrics.retryAttempts.Load(),
		RetryBackoffMsTotal:           s.openaiWSRetryMetrics.retryBackoffMs.Load(),
		RetryExhaustedTotal:           s.openaiWSRetryMetrics.retryExhausted.Load(),
		NonRetryableFastFallbackTotal: s.openaiWSRetryMetrics.nonRetryableFastFallback.Load(),
	}
}

func SnapshotOpenAICompatibilityFallbackMetrics() OpenAICompatibilityFallbackMetricsSnapshot {
	legacyReadFallbackTotal, legacyReadFallbackHit, legacyDualWriteTotal := openAIStickyCompatStats()
	isMaxTokensOneHaiku, thinkingEnabled, prefetchedStickyAccount, prefetchedStickyGroup, singleAccountRetry, accountSwitchCount := RequestMetadataFallbackStats()

	readHitRate := float64(0)
	if legacyReadFallbackTotal > 0 {
		readHitRate = float64(legacyReadFallbackHit) / float64(legacyReadFallbackTotal)
	}
	metadataFallbackTotal := isMaxTokensOneHaiku + thinkingEnabled + prefetchedStickyAccount + prefetchedStickyGroup + singleAccountRetry + accountSwitchCount

	return OpenAICompatibilityFallbackMetricsSnapshot{
		SessionHashLegacyReadFallbackTotal: legacyReadFallbackTotal,
		SessionHashLegacyReadFallbackHit:   legacyReadFallbackHit,
		SessionHashLegacyDualWriteTotal:    legacyDualWriteTotal,
		SessionHashLegacyReadHitRate:       readHitRate,

		MetadataLegacyFallbackIsMaxTokensOneHaikuTotal: isMaxTokensOneHaiku,
		MetadataLegacyFallbackThinkingEnabledTotal:     thinkingEnabled,
		MetadataLegacyFallbackPrefetchedStickyAccount:  prefetchedStickyAccount,
		MetadataLegacyFallbackPrefetchedStickyGroup:    prefetchedStickyGroup,
		MetadataLegacyFallbackSingleAccountRetryTotal:  singleAccountRetry,
		MetadataLegacyFallbackAccountSwitchCountTotal:  accountSwitchCount,
		MetadataLegacyFallbackTotal:                    metadataFallbackTotal,
	}
}

func (s *OpenAIGatewayService) detectCodexClientRestriction(c *gin.Context, account *Account) CodexClientRestrictionDetectionResult {
	return s.getCodexClientRestrictionDetector().Detect(c, account)
}

func getAPIKeyIDFromContext(c *gin.Context) int64 {
	if c == nil {
		return 0
	}
	v, exists := c.Get("api_key")
	if !exists {
		return 0
	}
	apiKey, ok := v.(*APIKey)
	if !ok || apiKey == nil {
		return 0
	}
	return apiKey.ID
}

// isolateOpenAISessionID 闂?apiKeyID 婵犵數濮烽弫鎼佸磿閹寸姴绶ゅù鐘差儏鎼村﹪鏌＄仦璇插姎闁?session 闂傚倸鍊风粈渚€骞栭銈囩煋闁哄鍤氬ú顏勭厸闁告侗鍠栭崜銊╂⒑閻熼偊鍤熷┑顕€顥撶划濠氬籍閸喓鍙嗛梺鍝勫暙濞层倖绂嶉崷顓犵＜闁?
// 缂傚倸鍊烽懗鍫曟惞鎼淬劌鐭楅幖娣妼缁愭鏌￠崶鈺佇ｇ€规洖寮堕幈銊ヮ潨閸℃绠婚梺宕囩帛濮婂鍩€椤掆偓缁犲秹宕曢柆宥呯疇闊洦绋戠壕?API Key 闂傚倸鍊烽悞锕傛儑瑜版帒绀夌€光偓閳ь剟鍩€椤掍礁鍤柛锝忕到椤曪綁顢曢敃鈧洿婵犮垼娉涢敃锕傤敇濞差亝鍊甸柣鐔告緲椤忣亝绻濋姀鈽嗙劷缂侇喖锕、娆徝规惔锛勭Ш闁轰焦鍔欏畷銊╊敃閿涘嫰鎸兼繝鐢靛Х閺佹悂宕戝☉妯忔椽顢橀姀鐘电暫婵炲濮撮鍛劔闁荤喐绮屽ù椋庡垝婵犲洤钃熼柕澶涘閸樺崬鈹戦鏂や緵闁告挻鐩幃妯绘綇閵娧咁啎闂佸壊鍋嗛崰搴ㄦ倶鐎电硶鍋撶憴鍕８闁搞劋绮欓獮鍐閿涘嫰妾繝銏ｆ硾椤戝洨绮?session_id/conversation_id闂?// 闂傚倸鍊风粈渚€骞夐敍鍕殰婵°倓绀侀崹鏂棵归悩宸剰缂佺姴鐏氶妵鍕疀閹炬潙娅ч梺宕囩帛閺屻劑鍩為幋锔藉亹闁圭粯甯楀▓鍫曟⒑缁洘娅嗘い銊ワ躬瀵鏁愭径濞⑩晠鏌曟径鍫濆姶濞寸娀绠栧铏圭磼濡厧顥夌紓浣瑰絻濞硷繝鐛箛娑欏亹閻犲洦褰冮懓鍨攽閻愬弶顥為柨姘舵煕閹惧崬濡界紒缁樼洴楠炲鎮欑划瑙勫闂備礁鎽滄慨瀵哥矓閸撲礁鍨濇い鎾跺枎缁剁偤鏌熼柇锕€澧茬悮婵嬫煟鎼粹€冲辅闁稿鎹囬幃妤呮晲鎼粹€愁潾濡炪倧瀵岄崳锝咁潖婵犳艾纾兼慨姗嗗墯濞堫參鎮楀▓鍨灈闁绘牕銈搁妴浣肝旈埀顒勫煡婢跺娼╂い鎾跺仧閿涘繘姊绘笟鈧褑鍣归梺鍛婃处閸撴岸顢樿ぐ鎺撯拻濞达絽鎲＄拹鈩冦亜閵夛箑濮囬柕鍡樺浮瀹曠兘顢橀悤浣圭稐闂備浇顫夐鏍窗濡ゅ啠鍋撳顓狀暡缂佺粯绻堝Λ鍐ㄢ槈濮橆兘鎷℃繝鐢靛仜閵堝摜绮婚弽顓炶摕闁绘梻鍘х粈鍫㈡喐瀹ュ棛顩插┑鍌氭啞閻?
func isolateOpenAISessionID(apiKeyID int64, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	h := xxhash.New()
	_, _ = fmt.Fprintf(h, "k%d:", apiKeyID)
	_, _ = h.WriteString(raw)
	return fmt.Sprintf("%016x", h.Sum64())
}

func logCodexCLIOnlyDetection(ctx context.Context, c *gin.Context, account *Account, apiKeyID int64, result CodexClientRestrictionDetectionResult, body []byte) {
	if !result.Enabled {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	fields := []zap.Field{
		zap.String("component", "service.openai_gateway"),
		zap.Int64("account_id", accountID),
		zap.Bool("codex_cli_only_enabled", result.Enabled),
		zap.Bool("codex_official_client_match", result.Matched),
		zap.String("reject_reason", result.Reason),
	}
	if apiKeyID > 0 {
		fields = append(fields, zap.Int64("api_key_id", apiKeyID))
	}
	if !result.Matched {
		fields = appendCodexCLIOnlyRejectedRequestFields(fields, c, body)
	}
	log := logger.FromContext(ctx).With(fields...)
	if result.Matched {
		return
	}
	log.Warn("OpenAI codex_cli_only 拒绝非官方客户端请求")
}

func appendCodexCLIOnlyRejectedRequestFields(fields []zap.Field, c *gin.Context, body []byte) []zap.Field {
	if c == nil || c.Request == nil {
		return fields
	}

	req := c.Request
	requestModel, requestStream, promptCacheKey := extractOpenAIRequestMetaFromBody(body)
	fields = append(fields,
		zap.String("request_method", strings.TrimSpace(req.Method)),
		zap.String("request_path", strings.TrimSpace(req.URL.Path)),
		zap.String("request_query", strings.TrimSpace(req.URL.RawQuery)),
		zap.String("request_host", strings.TrimSpace(req.Host)),
		zap.String("request_client_ip", strings.TrimSpace(c.ClientIP())),
		zap.String("request_remote_addr", strings.TrimSpace(req.RemoteAddr)),
		zap.String("request_user_agent", strings.TrimSpace(req.Header.Get("User-Agent"))),
		zap.String("request_content_type", strings.TrimSpace(req.Header.Get("Content-Type"))),
		zap.Int64("request_content_length", req.ContentLength),
		zap.Bool("request_stream", requestStream),
	)
	if requestModel != "" {
		fields = append(fields, zap.String("request_model", requestModel))
	}
	if promptCacheKey != "" {
		fields = append(fields, zap.String("request_prompt_cache_key_sha256", hashSensitiveValueForLog(promptCacheKey)))
	}

	if headers := snapshotCodexCLIOnlyHeaders(req.Header); len(headers) > 0 {
		fields = append(fields, zap.Any("request_headers", headers))
	}
	fields = append(fields, zap.Int("request_body_size", len(body)))
	return fields
}

func snapshotCodexCLIOnlyHeaders(header http.Header) map[string]string {
	if len(header) == 0 {
		return nil
	}
	result := make(map[string]string, len(codexCLIOnlyDebugHeaderWhitelist))
	for _, key := range codexCLIOnlyDebugHeaderWhitelist {
		value := strings.TrimSpace(header.Get(key))
		if value == "" {
			continue
		}
		result[strings.ToLower(key)] = truncateString(value, codexCLIOnlyHeaderValueMaxBytes)
	}
	return result
}

func hashSensitiveValueForLog(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func logOpenAIInstructionsRequiredDebug(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	upstreamStatusCode int,
	upstreamMsg string,
	requestBody []byte,
	upstreamBody []byte,
) {
	msg := strings.TrimSpace(upstreamMsg)
	if !isOpenAIInstructionsRequiredError(upstreamStatusCode, msg, upstreamBody) {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	accountID := int64(0)
	accountName := ""
	if account != nil {
		accountID = account.ID
		accountName = strings.TrimSpace(account.Name)
	}

	userAgent := ""
	originator := ""
	if c != nil {
		userAgent = strings.TrimSpace(c.GetHeader("User-Agent"))
		originator = strings.TrimSpace(c.GetHeader("originator"))
	}

	fields := []zap.Field{
		zap.String("component", "service.openai_gateway"),
		zap.Int64("account_id", accountID),
		zap.String("account_name", accountName),
		zap.Int("upstream_status_code", upstreamStatusCode),
		zap.String("upstream_error_message", msg),
		zap.String("request_user_agent", userAgent),
		zap.Bool("codex_official_client_match", openai.IsCodexOfficialClientByHeaders(userAgent, originator)),
	}
	fields = appendCodexCLIOnlyRejectedRequestFields(fields, c, requestBody)

	logger.FromContext(ctx).With(fields...).Warn("OpenAI 上游返回 Instructions are required，已记录请求详情用于排查")
}

func isOpenAIInstructionsRequiredError(upstreamStatusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if upstreamStatusCode != http.StatusBadRequest {
		return false
	}

	hasInstructionRequired := func(text string) bool {
		lower := strings.ToLower(strings.TrimSpace(text))
		if lower == "" {
			return false
		}
		if strings.Contains(lower, "instructions are required") {
			return true
		}
		if strings.Contains(lower, "required parameter: 'instructions'") {
			return true
		}
		if strings.Contains(lower, "required parameter: instructions") {
			return true
		}
		if strings.Contains(lower, "missing required parameter") && strings.Contains(lower, "instructions") {
			return true
		}
		return strings.Contains(lower, "instruction") && strings.Contains(lower, "required")
	}

	if hasInstructionRequired(upstreamMsg) {
		return true
	}
	if len(upstreamBody) == 0 {
		return false
	}

	errMsg := gjson.GetBytes(upstreamBody, "error.message").String()
	errMsgLower := strings.ToLower(strings.TrimSpace(errMsg))
	errCode := strings.ToLower(strings.TrimSpace(gjson.GetBytes(upstreamBody, "error.code").String()))
	errParam := strings.ToLower(strings.TrimSpace(gjson.GetBytes(upstreamBody, "error.param").String()))
	errType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(upstreamBody, "error.type").String()))

	if errParam == "instructions" {
		return true
	}
	if hasInstructionRequired(errMsg) {
		return true
	}
	if strings.Contains(errCode, "missing_required_parameter") && strings.Contains(errMsgLower, "instructions") {
		return true
	}
	if strings.Contains(errType, "invalid_request") && strings.Contains(errMsgLower, "instructions") && strings.Contains(errMsgLower, "required") {
		return true
	}

	return false
}

func isOpenAITransientProcessingError(upstreamStatusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if upstreamStatusCode != http.StatusBadRequest {
		return false
	}

	match := func(text string) bool {
		lower := strings.ToLower(strings.TrimSpace(text))
		if lower == "" {
			return false
		}
		if strings.Contains(lower, "an error occurred while processing your request") {
			return true
		}
		if strings.Contains(lower, "selected model is at capacity") {
			return true
		}
		return strings.Contains(lower, "you can retry your request") &&
			strings.Contains(lower, "help.openai.com") &&
			strings.Contains(lower, "request id")
	}

	if match(upstreamMsg) {
		return true
	}
	if len(upstreamBody) == 0 {
		return false
	}
	if match(gjson.GetBytes(upstreamBody, "error.message").String()) {
		return true
	}
	return match(string(upstreamBody))
}

// ExtractSessionID extracts the raw session ID from headers or body without hashing.
// Used by ForwardAsAnthropic to pass as prompt_cache_key for upstream cache.
func (s *OpenAIGatewayService) ExtractSessionID(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}
	sessionID := strings.TrimSpace(c.GetHeader("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.GetHeader("conversation_id"))
	}
	if sessionID == "" && len(body) > 0 {
		sessionID = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	return sessionID
}

func explicitOpenAISessionID(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}

	sessionID := strings.TrimSpace(c.GetHeader("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.GetHeader("conversation_id"))
	}
	if sessionID == "" && len(body) > 0 {
		sessionID = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	}
	return sessionID
}

// GenerateExplicitSessionHash generates a sticky-session hash only from explicit
// client session signals. It intentionally skips content-derived fallback and is
// used by stateless endpoints such as /v1/images.
func (s *OpenAIGatewayService) GenerateExplicitSessionHash(c *gin.Context, body []byte) string {
	sessionID := explicitOpenAISessionID(c, body)
	if sessionID == "" {
		return ""
	}

	currentHash, legacyHash := deriveOpenAISessionHashes(sessionID)
	attachOpenAILegacySessionHashToGin(c, legacyHash)
	return currentHash
}

// GenerateSessionHash generates a sticky-session hash for OpenAI requests.
//
// Priority:
//  1. Header: session_id
//  2. Header: conversation_id
//  3. Body:   prompt_cache_key (opencode)
//  4. Body:   content-based fallback (model + system + tools + first user message)
func (s *OpenAIGatewayService) GenerateSessionHash(c *gin.Context, body []byte) string {
	if c == nil {
		return ""
	}

	sessionID := explicitOpenAISessionID(c, body)
	if sessionID == "" && len(body) > 0 {
		sessionID = deriveOpenAIContentSessionSeed(body)
	}
	if sessionID == "" {
		return ""
	}

	currentHash, legacyHash := deriveOpenAISessionHashes(sessionID)
	attachOpenAILegacySessionHashToGin(c, legacyHash)
	return currentHash
}

// GenerateSessionHashWithFallback 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁规壆澧楅崑瀣攽閻樻彃顏ら柛瀣崌瀹曞綊顢曢妶鍥╁帓闂備浇妗ㄧ欢锟犲窗閺嶎叏缍栭柕蹇曞Х閺嗗棗鈹戦悩鎻掓殶缂佸鍢查埞鎴︽倻閸モ晛鍩屽┑鐐跺皺婵炩偓闁糕斁鍋撳銈嗗坊閸嬫捇鏌涢悢绋款嚋缂佺粯鐩畷鍫曨敆娴ｅ搫骞楅梻渚€娼х换鍫ュ垂瑜版帒鍑犻柟杈鹃檮閻撴洟鏌曟繝蹇擃洭妞わ絾鐓￠幃妤呮偨闂堟稑浠樺銈庡亝缁诲倿鎮鹃悜钘夌倞闁靛鍨洪悘渚€姊婚崒娆戭槮闁硅绱曢幑銏ゅ磼閻愭潙浜遍梺绯曞墲閻熝呯不閺冨牊鐓曢柕澶樺枛婢ь垶鏌?// 闂備浇宕甸崰鎰垝鎼淬垺娅犳俊銈呮噹缁犱即鎮归崶顏嶅殝闁肩増瀵ч妵鍕箻閸楃偟浠肩紓浣哄Т缂嶅﹪寮诲☉妯锋斀闁糕剝顨忔导鈧梻?session_id/conversation_id/prompt_cache_key 闂傚倸鍊风粈渚€骞栭锕€鐤柟鍓佺摂閺佸﹪鏌熼柇锕€鏋熸い顐ｆ礃缁绘繈妫冨☉娆忣槱婵犳鍨遍幐鎶藉箖瑜版帒绠掗柟鐑樺灥椤姊?fallbackSeed 闂傚倸鍊烽悞锕傛儑瑜版帒鍨傚┑鐘宠壘缁愭鏌熼悧鍫熺凡闁搞劌鍊归幈銊ノ熸径绋挎儓闂佹椿鍘介悷鈺呭蓟濞戙垹绠涙い鏍ㄦ皑濮ｃ垽姊洪崫鍕棛闁告濞婂璇测槈濡攱鏂€闂佺硶鍓濋悷銉┿€傞搹鍦＝濞达綀顕栧▓鏃€绻涘顔煎箺濞?// 闂傚倷娴囧畷鍨叏閺夋嚚娲閵堝懐锛熼梺鍝勮閸庨亶鎷戦悢鍏肩厽闁哄倽娉曞▓閬嶆煛鐎ｎ偆澧甸柡宀嬬節瀹曞爼鍩℃担鍥风秮閺岋繝宕卞Δ鍐唶闂?WS ingress闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸庢鏌涚仦鍓р槈妞ゆ洟浜堕弻宥夊传閸曨剙娅ｇ紓浣插亾闁稿本澹曢崑鎾荤嵁閸喖濮庨柣搴㈠嚬閸犳氨鍒掗敐鍛傛棃宕ㄩ闂村寲闂備焦鎮堕崕顖炲礉鎼达絿涓嶉柟瀛樺姴鎼淬劌鐐婄憸婵嬬叕椤掑倵鍋撶憴鍕┛缂佹煡绠栬棟闁绘鐗婇崕鐔兼煥濞戞ê顏柣锝呭暱閳规垿鎮╅鑲╀痪闂佺楠搁崥瀣Φ閺冨倻鐭欐繛鍡欏亾鐎靛矂姊虹粙璺ㄧ伇闁稿鐩幆灞轿旈崨顔惧幍闁诲海鏁搁…鍫熺墡闂備礁鎼幊蹇涘箖閸岀偛绠栫憸鐗堝笒缁犳帡鏌熼悜妯虹仴妞ゎ偄鎳忕换婵嬪閿濆棛銆愰柣搴㈢濠㈡﹢鎮鹃悜鑺ユ櫜濠㈣泛顑囬崣鍡涙煟鎼搭垳绉甸柛鎾寸洴閺佸秴鈹戦崼銏紳婵炶揪绲芥竟濠偽ｈぐ鎺撶厽妞ゅ繐瀚粔娲煕閳规儳浜?
func (s *OpenAIGatewayService) GenerateSessionHashWithFallback(c *gin.Context, body []byte, fallbackSeed string) string {
	sessionHash := s.GenerateSessionHash(c, body)
	if sessionHash != "" {
		return sessionHash
	}

	seed := strings.TrimSpace(fallbackSeed)
	if seed == "" {
		return ""
	}

	currentHash, legacyHash := deriveOpenAISessionHashes(seed)
	attachOpenAILegacySessionHashToGin(c, legacyHash)
	return currentHash
}

func resolveOpenAIUpstreamOriginator(c *gin.Context, isOfficialClient bool) string {
	if c != nil {
		if originator := strings.TrimSpace(c.GetHeader("originator")); originator != "" {
			return originator
		}
	}
	if isOfficialClient {
		return "codex_cli_rs"
	}
	return "opencode"
}

// BindStickySession sets session -> account binding with standard TTL.
func (s *OpenAIGatewayService) BindStickySession(ctx context.Context, groupID *int64, sessionHash string, accountID int64) error {
	if sessionHash == "" || accountID <= 0 {
		return nil
	}
	ttl := openaiStickySessionTTL
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.StickySessionTTLSeconds > 0 {
		ttl = time.Duration(s.cfg.Gateway.OpenAIWS.StickySessionTTLSeconds) * time.Second
	}
	return s.setStickySessionAccountID(ctx, groupID, sessionHash, accountID, ttl)
}

// SelectAccount selects an OpenAI account with sticky session support
func (s *OpenAIGatewayService) SelectAccount(ctx context.Context, groupID *int64, sessionHash string) (*Account, error) {
	return s.SelectAccountForModel(ctx, groupID, sessionHash, "")
}

// SelectAccountForModel selects an account supporting the requested model
func (s *OpenAIGatewayService) SelectAccountForModel(ctx context.Context, groupID *int64, sessionHash string, requestedModel string) (*Account, error) {
	return s.SelectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, nil)
}

// SelectAccountForModelWithExclusions selects an account supporting the requested model while excluding specified accounts.
// SelectAccountForModelWithExclusions 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ瀬閸ヮ剙绠ユい鏃傛嚀娴滅偓鎱ㄥΟ绋垮姎濠殿喖鍊婚埀顒侇問閸犳捇宕濋幋锔衡偓浣糕槈濡攱顫嶅┑鈽嗗灠閸氬鏆╅梻鍌氬€风粈浣革耿闁秮鈧箓宕煎婵囨そ婵¤埖寰勭€ｎ亙鎮ｉ梻浣筋嚃閸ㄥジ鎮樺☉銏″亜闁绘挸娴锋导瀣倵鐟欏嫭绀€婵炶尙濞€瀹曟垿骞樼紒妯衡偓濠氭煠閹帒鍔氶柛鎿冨弮濮婃椽宕ㄦ繝鍐槱闂佸摜濮甸悧鐘烘闂佸啿鎼幊蹇涙偂閺囩喓绠鹃柛鈩冾殘缁犵増绻涢崼姘珖缂佽鲸甯楀蹇涘Ω瑜忛悿鍕倵鐟欏嫭纾搁柛搴ゆ珪缁傛帡鏁冮崒娑樷偓閿嬨亜閹烘埈妲圭悮褔姊婚崒娆戭槮婵犫偓闁秴纾块柕鍫濇处閺嗘粓鏌嶉妷锔界伇闁哄绉归獮鏍庨鈧俊鑲╃磼閻樺崬宓嗛柡灞剧洴婵＄兘骞嬪┑鍥ф濠殿噯绲介柊锝咁潖閸濆嫅褔宕惰椤牓鏌ｆ惔銏犲毈闁搞劌宕銉╁礋椤栨氨顦板銈嗙墬濮樸劑鎮￠幋锔解拺闁告繂瀚烽崕娑㈡煕鐎Ｑ冧壕闂?
func (s *OpenAIGatewayService) SelectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*Account, error) {
	return s.selectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, excludedIDs, false, 0, PlatformOpenAI)
}

// noAvailableOpenAISelectionError builds the standard "no account available" error
// while preserving the compact-specific error when applicable.
func noAvailableOpenAISelectionError(requestedModel string, compactBlocked bool) error {
	if compactBlocked {
		return ErrNoAvailableCompactAccounts
	}
	if requestedModel != "" {
		return fmt.Errorf("no available OpenAI accounts supporting model: %s", requestedModel)
	}
	return errors.New("no available OpenAI accounts")
}

// openAICompactSupportTier classifies an OpenAI account by compact capability.
// 0 = explicitly unsupported, 1 = unknown / not yet probed, 2 = explicitly supported.
func openAICompactSupportTier(account *Account) int {
	if account == nil || !account.IsOpenAI() {
		return 0
	}
	supported, known := account.OpenAICompactSupportKnown()
	if !known {
		return 1
	}
	if supported {
		return 2
	}
	return 0
}

// isOpenAIAccountEligibleForRequest centralises the schedulable / OpenAI / model /
// compact-support checks used during account selection.
func accountMatchesPlatform(account *Account, platform string) bool {
	if account == nil {
		return false
	}
	platform = NormalizePlatformSlug(platform)
	if platform == "" {
		platform = PlatformOpenAI
	}
	return account.Platform == platform
}

func isOpenAIAccountEligibleForRequest(account *Account, requestedModel string, requireCompact bool, platform string) bool {
	if account == nil || !account.IsSchedulable() || !accountMatchesPlatform(account, platform) {
		return false
	}
	if requestedModel != "" && !account.IsModelSupported(requestedModel) {
		return false
	}
	if requireCompact && openAICompactSupportTier(account) == 0 {
		return false
	}
	return true
}

// prioritizeOpenAICompactAccounts re-orders a slice so that accounts with known
// compact support are tried first, followed by unknown, then explicitly unsupported.
// The relative order within each tier is preserved.
func prioritizeOpenAICompactAccounts(accounts []*Account) []*Account {
	if len(accounts) == 0 {
		return nil
	}
	supported := make([]*Account, 0, len(accounts))
	unknown := make([]*Account, 0, len(accounts))
	unsupported := make([]*Account, 0, len(accounts))
	for _, account := range accounts {
		switch openAICompactSupportTier(account) {
		case 2:
			supported = append(supported, account)
		case 1:
			unknown = append(unknown, account)
		default:
			unsupported = append(unsupported, account)
		}
	}
	out := make([]*Account, 0, len(accounts))
	out = append(out, supported...)
	out = append(out, unknown...)
	out = append(out, unsupported...)
	return out
}

// resolveOpenAIAccountUpstreamModelForRequest resolves the upstream model that
// would be sent for a given request, honouring compact-only mappings when the
// caller is on the /responses/compact path.
func resolveOpenAIAccountUpstreamModelForRequest(account *Account, requestedModel string, requireCompact bool) string {
	upstreamModel := resolveOpenAIForwardModel(account, requestedModel, "")
	if upstreamModel == "" {
		return ""
	}
	if requireCompact {
		return resolveOpenAICompactForwardModel(account, upstreamModel)
	}
	return upstreamModel
}

func (s *OpenAIGatewayService) selectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, stickyAccountID int64, platform string) (*Account, error) {
	platform = NormalizePlatformSlug(platform)
	if platform == "" {
		platform = PlatformOpenAI
	}
	if platform != PlatformOpenAI {
		requireCompact = false
	}
	if s.checkChannelPricingRestriction(ctx, groupID, requestedModel) {
		slog.Warn("channel pricing restriction blocked request",
			"group_id", derefGroupID(groupID),
			"model", requestedModel)
		return nil, fmt.Errorf("%w supporting model: %s (channel pricing restriction)", ErrNoAvailableAccounts, requestedModel)
	}

	// 1. 闂傚倷娴囬褏鎹㈤幇顔藉床闁瑰濮靛畷鏌ユ煕閳╁啰鈯曢柛搴★攻閵囧嫰寮介妸褏鐓€闂佹悶鍔嶇换鍐╃┍婵犲浂鏁嶆繝濠傚暙婵箓姊虹粙娆惧剱闁归€涚窔钘濋弶鍫氭櫇绾惧ジ寮堕崼娑樺妞ゃ儱顑夐弻鐔割槹鎼粹寬銏ゆ煃鐟欏嫬鐏寸€规洖宕灒闁绘垶锕╅崥瀣節?	// Try sticky session hit
	if account := s.tryStickySessionHit(ctx, groupID, sessionHash, requestedModel, excludedIDs, requireCompact, stickyAccountID, platform); account != nil {
		return account, nil
	}

	// 2. 闂傚倸鍊风粈渚€宕ョ€ｎ喖纾块柟鎯版鎼村﹪鏌ら懝鎵牚濞存粌缍婇弻娑㈠Ψ椤旂厧顫╅梺娲诲幗椤ㄥ棝濡甸崟顖氬唨闁靛ě鍛毉婵犵數鍋涢悧鍡涙偉婵傚摜宓侀柡宥冨妽缂嶅洭鏌熼鍡楁湰閸犳ɑ绻?OpenAI 闂傚倷娴囧畷鍨叏閻㈢绀夌憸蹇曞垝婵犳艾绠ｉ柨婵嗗暕濮?	// Get schedulable OpenAI accounts
	accounts, err := s.listSchedulableAccounts(ctx, groupID, platform)
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}

	// 3. 闂傚倸鍊风粈浣革耿闁秮鈧箓宕奸妷瀣喘椤㈡瑩鎮欓澶嬬稐婵＄偑鍊栭幐鍫曞垂閸︻厾鐭嗛柛宀€鍋為悡鏇熴亜閹邦喖孝闁诲繑鐓￠弻?+ LRU 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ瀬閸ヮ剙绠ユい鏃傛嚀娴滅偓鎱ㄥΟ绋垮姎濠殿喖鍊婚埀顒侇問閸犳岸寮繝姘槬闁逞屽墯閵囧嫰骞掑鍥у婵犳鍨遍幐鎶藉箖濡ゅ懎鎹舵い鎾跺€姀掳浜滈柨鏃囶潐濞呭﹪鏌?	// Select by priority + LRU
	selected, compactBlocked := s.selectBestAccount(ctx, groupID, accounts, requestedModel, excludedIDs, requireCompact, platform)

	if selected == nil {
		return nil, noAvailableOpenAISelectionError(requestedModel, compactBlocked)
	}

	// 4. 闂傚倷娴囧畷鍨叏瀹曞洨鐭嗗ù锝堫潐濞呯姴霉閻樺樊鍎愰柛瀣典邯閺屾盯鍩勯崘顏呭櫑闂佹悶鍔嶇换鍐╃┍婵犲浂鏁嶆繝濠傚暙婵箓姊虹粙娆惧剱闁归€涚窔钘濋弶鍫氭櫇绾惧ジ寮堕崼娑樺妞ゃ儱顑夐弻鐔割槹鎼粹寬銏ゆ煃鐟欏嫬鐏寸€规洖銈告慨鈧柍銉﹀墯娴犻亶姊?	// Set sticky session binding
	if sessionHash != "" {
		_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, selected.ID, openaiStickySessionTTL)
	}

	return s.hydrateSelectedAccount(ctx, selected)
}

// tryStickySessionHit 闂傚倷娴囬褏鎹㈤幇顔藉床闁瑰濮靛畷鏌ユ煕閳╁啰鈯曢柛搴★攻閵囧嫰寮介妸褋鈧帗銇勯埡鍐ㄥ幋妤犵偞鐗楀蹇涘礈瑜庨崑褔姊虹紒妯诲鞍閻㈩垽绻濆濠氭偄绾拌鲸鏅炲銈嗗坊閸嬫捇鏌涙繝鍛惞缂侇噯绲借灃闁告侗鍠栨禒顖炴⒑閹肩偛鍔橀柛鏂挎捣缁顫濋褎顔旈梺缁樺姌鐏忔瑧绮婚懡銈傚亾鐟欏嫭绌跨紓宥勭椤曪絾绂掔€ｅ灚鏅㈤梺閫炲苯澧撮挊婵嬫煃閸濆嫭鍣洪柣鎾存礃缁绘盯宕卞Ο鍝勵潕婵炲濮甸惄顖炲蓟?// 濠电姷鏁告慨鐑姐€傛禒瀣劦妞ゆ巻鍋撻柛鐔锋健閸┾偓妞ゆ巻鍋撶紓宥咃躬楠炲啫螣鐠囪尙绐為梺褰掑亰閸撴瑧绮ｅ☉銏♀拺闁荤喐澹嗛幗鐘绘偨椤栨粌鏋涚€规洏鍨介、妤呭焵椤掑嫬鐓″鑸靛姇缁犲鎮楀☉娆欎緵婵﹥瀵х换婵嬪閿濆棛銆愰柣搴㈢濠㈡﹢鎮鹃悜鑺ユ櫜闁告侗鍨卞▓婵嬫⒑閸濆嫷妲奸柛搴☆煼椤㈡瑦寰勯幇顓涙嫼濠电偠灏欑划顖涚箾閸ヮ剚鐓曢柡鍌濇硶閻掑摜鈧娲栫紞濠囥€佸☉妯锋婵炲棗瀛╅崬澶愭⒒娴ｅ憡鎯堥柛濠傛贡閳ь剛鐟抽崨顖滅劶闂侀€炲苯澧撮柡宀嬬秮閹晠宕橀幓鎹胶绱撴担鍝勑ｉ柛銊ャ偢婵＄敻骞囬弶鍧楁暅濠德板€撻悞锕€鈻嶉弽褉鏀介柣鎰綑閻忥箓鏌ㄩ弴妤佹珔閻撱倝鏌曢崼婵愭Ч闁绘挻鐩幃妤呮晲鎼粹€茬敖闂佹椿鍘兼鎼佲€︾捄銊﹀磯闁告繂瀚锋导鍐倵鐟欏嫭绌跨紓宥勭窔楠炴鎯旈妸銉э紲濠电姴锕ら幊蹇涘窗閹烘鈷掑ù锝呮啞閸熺偤鏌ｉ悤浣哥仸鐎规洖缍婇獮搴ㄦ寠婢跺鈧剟姊洪棃娑氬妞わ富鍨崇划鍫ュ幢濡偐顔曢梺绯曞墲閿氬┑顔碱槹閵囧嫯绠涢弮鈧崐鎰版煛瀹€瀣М闁诡喗鐟╁畷锝嗗緞濡椿妫ょ紓鍌氬€峰ù鍥ㄣ仈閸涘﹦鐝堕柛鈩冪☉缁狀垶鏌ｅΟ鑽ゃ偞闁衡偓娴犲鐓曢柍鈺佸暢婢规﹢鏌ｅ┑鎰仸闁哄备鍓濆鍕幢濡崵褰嗛梻浣告惈鐞氼偊宕濆畝鈧崣?nil闂?//
// tryStickySessionHit attempts to get account from sticky session.
// Returns account if hit and usable; clears session and returns nil if account is unavailable.
func (s *OpenAIGatewayService) tryStickySessionHit(ctx context.Context, groupID *int64, sessionHash, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, stickyAccountID int64, platform string) *Account {
	if sessionHash == "" {
		return nil
	}

	accountID := stickyAccountID
	if accountID <= 0 {
		var err error
		accountID, err = s.getStickySessionAccountID(ctx, groupID, sessionHash)
		if err != nil || accountID <= 0 {
			return nil
		}
	}

	if _, excluded := excludedIDs[accountID]; excluded {
		return nil
	}

	account, err := s.getSchedulableAccount(ctx, accountID)
	if err != nil {
		return nil
	}

	// 婵犵數濮烽。钘壩ｉ崨鏉戠；闁逞屽墴閺屾稓鈧綆鍋呭畷宀勬煛瀹€瀣？濞寸媴濡囬幏鐘诲箵閹烘嚩鎾斥攽閻橆喖鐏辨繛澶嬬〒閳ь剚纰嶅姗€鎮鹃悜鑺ユ櫜濠㈣泛顑囬崣鍡涙煟鎼搭垳绉甸柛濠忕秮閺佸啴宕掑☉姘箞婵犳鍠楅敋濠⒀傜矙閹箖宕归銈囶啎闂佽鍎抽崯鍧椼€傞幎鑺ョ厵闁肩⒈鍓欓。濂告煙瀹勭増鍤囩€规洜鍏橀、妯款槺缂佹墎鏅犲缁樻媴閽樺鎯為梺绋款儐缁嬫垼鐏嬪┑鐐村灟閸ㄦ椽宕曞Δ浣瑰弿婵＄偠顕ф禍鎯旈悩闈涗汗闁稿鎹囬幃宄邦煥閸愵亞浼囧┑鈥冲级閸旀瑩鐛澶樻晩闁芥ê顦辫ぐ?	// Check if sticky session should be cleared
	if shouldClearStickySession(account, requestedModel) {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}

	// 濠电姴鐥夐弶搴撳亾濡や焦鍙忛柟缁㈠枟閸庢銆掑锝呬壕闂佽鍨悞锕€顕ラ崟顓濇勃闁诡垎灞肩穿闂傚倷鐒︾€笛兠哄澶婄；闁规崘鍩栭崰鎰版煛婢跺顕滈柛瀣ㄥ劤閳ь剚顔栭崰鏇㈠础閹跺鈧礁鈽夊鍡樺兊濡炪倖鎸鹃崑娑欐櫠妤ｅ啯鈷掑ù锝呮啞閸熺偤鏌ｉ悤浣哥仸鐎规洖缍婇獮搴ㄦ寠婢跺鈧剟姊洪棃娑氬婵☆偅鐩妴鍛村蓟閵夈儳顔愰柡澶婄墕婢х晫澹曠捄銊х＜濠㈣泛顭堥崑銏ゆ煛鐏炲墽娲存鐐疵悾鐑藉炊閸℃劕鍔滈柕鍥у瀵挳濡搁妷銉交闂?	// Verify account is usable for current request
	if !isOpenAIAccountEligibleForRequest(account, requestedModel, false, platform) {
		return nil
	}
	account = s.recheckSelectedOpenAIAccountFromDB(ctx, account, requestedModel, requireCompact, platform)
	if account == nil {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}
	if groupID != nil && s.needsUpstreamChannelRestrictionCheck(ctx, groupID) &&
		s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel, requireCompact) {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}

	// 闂傚倸鍊风粈渚€骞夐敍鍕殰闁跨喓濮寸紒鈺呮⒑椤掆偓缁夋挳鎷戦悢灏佹斀闁绘ɑ褰冮弳娆愩亜閺囷繝鍝洪柟渚垮妼铻ｉ柧蹇曟缁辩偛鈹?TTL 婵犲痉鏉库偓妤佹叏閻戣棄纾婚柣鎰惈绾惧潡鏌ゅù瀣珕鐎规洘鐓￠弻鐔告綇閸撗呮殸闂佺粯鍔曢敃顏堝蓟閿濆绠涙い鏍ㄧ〒閵嗘劕顪冮妶蹇氬悅闁哄懐濞€瀵?	// Refresh session TTL and return account
	_ = s.refreshStickySessionTTL(ctx, groupID, sessionHash, openaiStickySessionTTL)
	return account
}

// selectBestAccount 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忛悗娑欙供濞堜粙鏌熼梻瀵割槮闁绘帒鐏氶妵鍕箳瀹ュ牆鍘￠梺鐟板暱濞差參寮诲鍫闂佸憡鎸婚悷鈺勬闂佹儳娴氶崑鍕閻愮儤鐓曢柡鍥ュ妼閻忕娀鏌ｉ鐕佹疁闁哄睙鍐炬僵妞ゆ巻鍋撻柍褜鍏涚划娆忕暦閵忋値鏁嗛柛鏇ㄥ厴閹峰搫顪冮妶鍡楀闁糕晛瀚嵄鐟滅増甯楅崐鍨叏濡鍔氬┑顔煎€婚埀顒侇問閸犳岸寮繝姘槬闁逞屽墯閵囧嫰骞掑鍥у婵犳鍨遍幐鎶藉箖濡ゅ懎鎹舵い鎾跺€姀掳浜滈柨鏃囶潐濞呭﹪鏌＄仦鍓р槈闁宠姘︾粻娑㈠箻閹碱厼鏁ょ紓鍌氬€峰ù鍥ㄣ仈閹间焦鍋￠柨鏃堟暜閸嬫捇鎮介棃娑樹粯闁句紮缍侀弻娑樷攽閸℃浠奸悗娈垮枟椤ㄥ牏妲?+ LRU闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆¤娲、姗€濮€閻橀潧濮?// 闂傚倷绀侀幖顐λ囬锕€鐤炬繝濠傜墕閽冪喖鏌曟繛鍨壄?nil 闂傚倷娴囧畷鐢稿磻閻愮數鐭欓煫鍥ㄧ☉缁€澶愬箹濞ｎ剙濡煎鍛攽椤旂瓔鐒炬繛澶嬬〒閻氭儳顓兼径瀣幗濠碘槅鍨甸褏寰婃繝姘仯濞达絽婀遍崣鈧梺鍝勭焿缂嶄線鏁愰悙渚晢闁逞屽墯閹便劑鍩€椤掍胶绡€闁靛骏绲剧涵楣冩倵濮橆厽绶查柣锝囧厴閺佹劖寰勬繝鍌氭婵＄偑鍊栭崝妤呭窗鎼淬垻顩?//
// selectBestAccount selects the best account from candidates (priority + LRU).
// Returns nil if no available account. The second return reports whether at
// least one candidate was filtered out solely because it lacks compact support
// (only meaningful when requireCompact=true).
func (s *OpenAIGatewayService) selectBestAccount(ctx context.Context, groupID *int64, accounts []Account, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, platform string) (*Account, bool) {
	var selected *Account
	selectedCompactTier := -1
	compactBlocked := false
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, groupID)

	for i := range accounts {
		acc := &accounts[i]

		// 闂傚倷娴囧畷鍨叏閹绢喖绠规い鎰堕檮閸嬵亪鏌涢妷銏℃珕鐎规洘鐓￠弻娑㈠箛閸忓摜鍑归梺绋款儐閻楁濡甸崟顖氱疀闁告挷绀侀崺宀€绱撴担璇℃畷闁圭懓娲ら～蹇旂節濮橆剛顦伴梺瀹犳〃鐠佹煡宕戦幘婢勬棃鍩€椤掑嫬鐓濋柟鎹愵嚙缁狅綁鏌ｅΟ鍝勬毐妞ゅ繑鎮傚娲箰鎼淬垻锛曢梺绋款儐閹搁箖骞?		// Skip excluded accounts
		if _, excluded := excludedIDs[acc.ID]; excluded {
			continue
		}

		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel, false, platform)
		if fresh == nil {
			continue
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, false, platform)
		if fresh == nil {
			continue
		}
		if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, fresh, requestedModel, requireCompact) {
			continue
		}
		compactTier := 0
		if requireCompact {
			compactTier = openAICompactSupportTier(fresh)
			if compactTier == 0 {
				compactBlocked = true
				continue
			}
		}

		// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ瀬閸ヮ剙绠ユい鏃傛嚀娴滅偓鎱ㄥΟ绋垮姎濠碘€茬矙閹鎮介棃娑樹粯闁句紮缍侀弻娑樷攽閸℃浠奸悗娈垮枟椤ㄥ牏妲愰幘瀛樺闁告繂瀚ぐ娆撴⒑閸涘﹥澶勯柛蹇旂〒閸掓帡顢涘☉妤冪畾闂侀潧鐗嗛幊搴ㄥ焵椤掆偓濠€鍗烇耿娴ｈ櫣纾藉ù锝堫嚃濞堟洟鏌ｅΔ鍐ㄐ㈡い鏇秮閹晫绮欑捄銊ュЕ婵＄偑鍊栫敮濠囨嚄閸洖鐓濋柡鍥ュ灪閻撴洘绻涢幋婵嗚埞闁哄鍠撻埀顒侇問閸犳骞愰幖浣圭畳闂備焦瀵х换鍌炴偋婵犲嫭鍏滈柛顐ｆ礃閻撴稑霉閿濆娑ч柍褜鍓氱换鍫ョ嵁閸℃稑绀冩い鏃囧亹椤︽澘顪冮妶鍡樷拻闁冲嘲鐗嗗嵄闂侇剙绉甸埛?		// Select highest priority and least recently used
		if selected == nil {
			selected = fresh
			selectedCompactTier = compactTier
			continue
		}

		// compact 婵犵數濮烽。钘壩ｉ崨鏉戝瀭妞ゅ繐鐗嗛悞鍨亜閹哄棗浜剧紒鍓ц檸閸樻儳鈽夐悽绋跨劦妞ゆ帊鑳剁粻楣冩煙鐎电浠﹂悘蹇ｅ幗閵囧嫰骞嬪┑鎰枅闂?tier 濠电姷鏁搁崑鐐差焽濞嗘挸瑙﹂悗锝庡枟閺咁亪姊绘担鍛婂暈閽冮亶鏌ｉ埡濠傜仸鐎殿噮鍋勯濂稿川椤忓拋娼旀繝纰樻閸ㄧ敻顢氳閸┾偓?tier 闂傚倸鍊风粈渚€骞夐敓鐘茬闁哄洢鍨圭粻鐘荤叓閸ャ劍鈷愭繛鍛█閺岋絽螣閼姐倖娈梺鍦劋椤ㄥ懐鐚惧澶嬬厱闁靛鍨哄▍鍥煟?priority/LRU闂?
		if requireCompact && compactTier != selectedCompactTier {
			if compactTier > selectedCompactTier {
				selected = fresh
				selectedCompactTier = compactTier
			}
			continue
		}

		if s.isBetterAccount(fresh, selected) {
			selected = fresh
			selectedCompactTier = compactTier
		}
	}

	return selected, compactBlocked
}

// isBetterAccount 闂傚倸鍊风粈渚€骞夐敍鍕殰闁搞儺鍓欑壕褰掓煛瀹ュ骸骞栭柦?candidate 闂傚倸鍊风粈渚€骞栭銈傚亾濮樺崬鍘寸€规洝顫夌€靛ジ寮堕幋鐘垫毎濠电偠鎻徊楣冨礉?current 闂傚倸鍊风粈渚€骞栭鈷氭椽濡歌瀹曞弶绻濋棃娑氬妞ゎ偅娲橀妵鍕箻閸楃偟浠奸弶?// 闂傚倷娴囧畷鐢稿窗閹扮増鍋￠柕澹偓閸嬫挸顫濋悡搴♀拫閻庤娲栫紞濠囥€佸☉銏″€烽悗娑櫳戦悵顐ｇ節濞堝灝鏋熼柨鏇楁櫊瀹曟垿宕卞☉妯肩崶濠殿喗銇涢崑鎾绘煛鐏炲墽顬肩紒鐘崇洴楠炴鎹勭悰鈥冲綑婵犲痉鏉库偓妤佹叏閺夋嚚娲敇閻戝棗娈ㄩ梺鍦檸閸犳牜绮荤紒妯镐簻闁哄啠鍋撻柛搴㈠▕閹峰繒鈧綆鍠楅埛鎴︽煙缁嬫寧鎹ｉ柍钘夘樀閺岋絽螖閳ь剟鏁冮鍕靛殨闁规儼濮ら崑鍕煟閹捐櫕鎹ｉ柨娑欑懇濮婅櫣绮欑捄銊ь唶闂佸憡鑹鹃鍥儉椤忓牜鏁嶉柣鎰綑閳ь剙鐖奸悡顐﹀炊閵婏箑纾抽梺缁樼⊕缁海妲愰幒鎾寸秶闁靛ě灞炬闂佽娴烽弫鎼佸储瑜戦悘鎺楁⒑閸涘﹦绠撻悗姘煎墮椤曪絽螖閸涱喒鎷洪梺鍦焾濞寸兘鍩婇弴鐘电＜閻庯綆鍋勯悘瀛橆殽閻愭彃鏆ｆ鐐达耿瀵爼骞嬮鐐搭啌濠电姷顣藉Σ鍛村垂娴兼潙绠规い鎰╁€栭弳婊堟煏婢诡垰鎳愰敍婊堟⒑缂佹鎳勯柣鐔村劤閳ь剚鍑归崣鍐閻熸粍妫冨濠氬Ω閵夈垺顫嶅┑鈽嗗灣缁垶骞忛崡鐐╂斀闁绘﹩鍋勬禍鎯ь渻閵堝棙灏甸柛鐘冲姈閸掑﹪骞橀钘変化闂佽鍘界敮鎺撲繆娴犲鐓涢柛鈩冾殘缁犺崵鈧鍠栭悥濂稿春閳╁啯濯撮柣鐔稿閻涒晜绻濋悽闈涗粶闁绘锕畷褰掑礈娴ｆ彃浜鹃柣銏ゆ涧鐢爼鎽堕敐澶嬬厱婵犻潧妫楅鈺冣偓娈垮枟椤ㄥ﹤顫忓ú顏勭閹艰揪绲哄Σ鍫ユ⒑閸忓吋銇熼柛銊ㄦ硾閻ｇ兘鏁愭径濠勭暰閻熸粌顦靛鍛婃償閵婏妇鍘撻梺瀹犳〃缁€渚€寮抽悙鐢电當闁硅揪闄勯埛鎴︽煕濠靛棗顏褎鍔欓弻娑氣偓锝庡亝鐏忣參鏌嶉挊澶樻█鐎殿噮鍓熼獮鎰償閳╁喚浠┑鐘垫暩婵參宕戦幘娣簻闁规崘娉涢弸鎴濃攽椤栨稒灏﹂柟顔肩秺楠炰線骞掗幋婵愮€撮梻浣告惈濡鎹㈠鈧濠氭晲婢跺á鈺呮煏婢跺牆鍔村ù鐘虫尦濮?//
// isBetterAccount checks if candidate is better than current.
// Rules: higher priority (lower value) wins; same priority: never used > least recently used.
func (s *OpenAIGatewayService) isBetterAccount(candidate, current *Account) bool {
	// 濠电姷鏁搁崑鐐差焽濞嗘挸瑙﹂悗锝庡枟閺咁亪姊绘担鍛婂暈閽冭京绱掔€ｎ偅宕岄柟顖楀亾濡炪倕绻愬ù鍌毼ｉ悡搴樻斀闁绘劖褰冪痪褔鎮楃粭娑樻储閳ь剙鍟村畷鎯邦檨婵炵鍔戦弻娑㈩敃閻樿尙浼勫銈忚吂閺呯姴顫忛悜妯诲闁规鍠栨俊钘夆攽閻橆喖鐏柨鏇樺灲瀹曟椽鍩€椤掍降浜滈柟鍝勬娴滈箖姊洪崨濠呭妞ゆ垵鎳愰崣鍛存煟閻樺厖鑸柛鏂款儔閹偞绻濋崶鈺佸絼闂佹悶鍎崕閬嶅礉閵堝洨纾奸柍?
	// Higher priority (lower value)
	if candidate.Priority < current.Priority {
		return true
	}
	if candidate.Priority > current.Priority {
		return false
	}

	// 闂傚倸鍊风粈渚€骞夐敓鐘冲殞闁绘劦鍓涢梽鍕煛閸愩劌鈧綊锝為弴鐔翠簻闁规儳宕悘鈺冪磼閳ь剟宕掗悙瀵稿幈濡炪倖鍔楁慨鎾倶閺屻儲鐓熸繛鎴炵懄瀹曞矂鏌＄仦璇插闁宠棄顦灒缂備焦蓱鐎氭娊姊绘担椋庝覆閻庨潧鐭傚畷鏉课旈埀顒勨€﹂崶顏嗙杸婵炴垶顭囬ˇ銊╂⒑閸愬弶鎯堥柛濠囶棑閸掓帡顢涢悙绮规嫼闂佸憡绋戦敃銊︾珶濡偐纾奸柕濞垮劚閹垹绱掔紒妯兼创鐎规洖鐖奸、妤佹媴闂€鎰处闂傚倷绶氬褔鈥﹂銏♀挃闁告洦鍨侀崶顏嶆▌濠?	// Same priority, compare last used time
	switch {
	case candidate.LastUsedAt == nil && current.LastUsedAt != nil:
		// candidate 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忛悗鐢电《閸嬫挸鈽夐幒鎾寸彅闂佹寧娲忛崹浠嬨€佸鈧幃銏㈢矙濞嗛敮鍋撻幘缁樼厽闁绘ê鍘栭懜顏堟煕閺傝法效鐎殿喖鐤囩粻娑樷槈濞嗘垵寮虫繝鐢靛仦閸ㄥ爼鏁嬪銈冨妽閻熝呮閹烘搩娼欓柡宓啰鐫勯柣?
		return true
	case candidate.LastUsedAt != nil && current.LastUsedAt == nil:
		// current 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忛悗鐢电《閸嬫挸鈽夐幒鎾寸彅闂佹寧娲忛崹浠嬨€佸鈧幃銏㈢矙濞嗛敮鍋撻幘缁樼厽闁绘ê鍘栭懜顏堟煕閺傝法效鐎殿喖鐤囩粻娑樷槈濞嗘垵寮虫繝鐢靛仦閸ㄥ爼鏁嬪銈冨妽閻熝呮閹烘梻纾兼俊顖氭惈閹介潧螖?
		return false
	case candidate.LastUsedAt == nil && current.LastUsedAt == nil:
		// 闂傚倸鍊搁崐椋庢閿熺姴绐楁繛鎴炵懄閸欏繘鏌曢崼婵囧珔闁肩増瀵ч妵鍕箻鐠虹儤鐎炬繝娈垮灡閹告娊骞冭ぐ鎺戠畳闁圭儤鍨甸‖澶愭⒑閸濆嫭顥滅紒缁橈耿瀵濡搁妷銏☆潔濠碘槅鍨拃锕€鈻撻鐘电＝濞达綀娅ｇ敮娑氣偓鍏夊亾闁归棿鑳跺畵?
		return false
	default:
		// 闂傚倸鍊搁崐椋庢閿熺姴绐楁俊銈呮噹閸ㄥ倹绻濋棃娑欏窛缂佺娀绠栭弻娑㈠Ψ椤旂厧顫梺绋款儍閸旀垿寮婚弴鐔虹闁割煈鍠栭‖瀣磽娴ｅ搫校闁绘濞€瀵濡搁妷銏☆潔濠碘槅鍨甸崑鎰板礉椤栫偞鈷戦梺顐ゅ仜閼活垱鏅堕濮愪簻闁挎棁妫勯ˉ瀣煃瑜滈崜姘跺传鎼淬劌绀夐柟瀛樼箥閸ゆ洟鏌ｉ姀銏╃劸闁绘帒鐏氶妵鍕箳瀹ュ牆鍘￠梺宕囩帛濞茬喖寮诲☉妯锋闁告鍋為悘宥夋倵濞堝灝鏋熼柟鍛婃倐濠€渚€姊洪幐搴ｇ畵闁绘锕﹀☉鐢稿醇閺囩喓鍘告繛杈剧秮濞佳囧焵椤掍胶绠炴?
		return candidate.LastUsedAt.Before(*current.LastUsedAt)
	}
}

// SelectAccountWithLoadAwareness selects an account with load-awareness and wait plan.
func (s *OpenAIGatewayService) SelectAccountWithLoadAwareness(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*AccountSelectionResult, error) {
	return s.selectAccountWithLoadAwareness(ctx, groupID, sessionHash, requestedModel, excludedIDs, false, PlatformOpenAI)
}

func (s *OpenAIGatewayService) selectAccountWithLoadAwareness(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, platform string) (*AccountSelectionResult, error) {
	platform = NormalizePlatformSlug(platform)
	if platform == "" {
		platform = PlatformOpenAI
	}
	if platform != PlatformOpenAI {
		requireCompact = false
	}
	if s.checkChannelPricingRestriction(ctx, groupID, requestedModel) {
		slog.Warn("channel pricing restriction blocked request",
			"group_id", derefGroupID(groupID),
			"model", requestedModel)
		return nil, fmt.Errorf("%w supporting model: %s (channel pricing restriction)", ErrNoAvailableAccounts, requestedModel)
	}

	cfg := s.schedulingConfig()
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, groupID)
	var stickyAccountID int64
	if sessionHash != "" && s.cache != nil {
		if accountID, err := s.getStickySessionAccountID(ctx, groupID, sessionHash); err == nil {
			stickyAccountID = accountID
		}
	}
	if s.concurrencyService == nil || !cfg.LoadBatchEnabled {
		account, err := s.selectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, excludedIDs, requireCompact, stickyAccountID, platform)
		if err != nil {
			return nil, err
		}
		result, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
		if err == nil && result.Acquired {
			return s.newSelectionResult(ctx, account, true, result.ReleaseFunc, nil)
		}
		if stickyAccountID > 0 && stickyAccountID == account.ID && s.concurrencyService != nil {
			waitingCount, _ := s.concurrencyService.GetAccountWaitingCount(ctx, account.ID)
			if waitingCount < cfg.StickySessionMaxWaiting {
				return s.newSelectionResult(ctx, account, false, nil, &AccountWaitPlan{
					AccountID:      account.ID,
					MaxConcurrency: account.Concurrency,
					Timeout:        cfg.StickySessionWaitTimeout,
					MaxWaiting:     cfg.StickySessionMaxWaiting,
				})
			}
		}
		return s.newSelectionResult(ctx, account, false, nil, &AccountWaitPlan{
			AccountID:      account.ID,
			MaxConcurrency: account.Concurrency,
			Timeout:        cfg.FallbackWaitTimeout,
			MaxWaiting:     cfg.FallbackMaxWaiting,
		})
	}

	accounts, err := s.listSchedulableAccounts(ctx, groupID, platform)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, ErrNoAvailableAccounts
	}

	isExcluded := func(accountID int64) bool {
		if excludedIDs == nil {
			return false
		}
		_, excluded := excludedIDs[accountID]
		return excluded
	}

	// ============ Layer 1: Sticky session ============
	if sessionHash != "" {
		accountID := stickyAccountID
		if accountID > 0 && !isExcluded(accountID) {
			account, err := s.getSchedulableAccount(ctx, accountID)
			if err == nil {
				clearSticky := shouldClearStickySession(account, requestedModel)
				if clearSticky {
					_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
				}
				if !clearSticky && isOpenAIAccountEligibleForRequest(account, requestedModel, false, platform) {
					account = s.recheckSelectedOpenAIAccountFromDB(ctx, account, requestedModel, requireCompact, platform)
					if account == nil {
						_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
					} else if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel, requireCompact) {
						_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
					} else {
						result, err := s.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
						if err == nil && result.Acquired {
							_ = s.refreshStickySessionTTL(ctx, groupID, sessionHash, openaiStickySessionTTL)
							return s.newSelectionResult(ctx, account, true, result.ReleaseFunc, nil)
						}

						waitingCount, _ := s.concurrencyService.GetAccountWaitingCount(ctx, accountID)
						if waitingCount < cfg.StickySessionMaxWaiting {
							return s.newSelectionResult(ctx, account, false, nil, &AccountWaitPlan{
								AccountID:      accountID,
								MaxConcurrency: account.Concurrency,
								Timeout:        cfg.StickySessionWaitTimeout,
								MaxWaiting:     cfg.StickySessionMaxWaiting,
							})
						}
					}
				}
			}
		}
	}

	// ============ Layer 2: Load-aware selection ============
	baseCandidateCount := 0
	candidates := make([]*Account, 0, len(accounts))
	for i := range accounts {
		acc := &accounts[i]
		if isExcluded(acc.ID) {
			continue
		}
		// Scheduler snapshots can be temporarily stale (bucket rebuild is throttled);
		// re-check schedulability here so recently rate-limited/overloaded accounts
		// are not selected again before the bucket is rebuilt.
		if !acc.IsSchedulable() {
			continue
		}
		if requestedModel != "" && !acc.IsModelSupported(requestedModel) {
			continue
		}
		if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, acc, requestedModel, requireCompact) {
			continue
		}
		baseCandidateCount++
		candidates = append(candidates, acc)
	}

	if len(candidates) == 0 {
		return nil, ErrNoAvailableAccounts
	}

	accountLoads := make([]AccountWithConcurrency, 0, len(candidates))
	for _, acc := range candidates {
		accountLoads = append(accountLoads, AccountWithConcurrency{
			ID:             acc.ID,
			MaxConcurrency: acc.EffectiveLoadFactor(),
		})
	}

	loadMap, err := s.concurrencyService.GetAccountsLoadBatch(ctx, accountLoads)
	if err != nil {
		ordered := append([]*Account(nil), candidates...)
		sortAccountsByPriorityAndLastUsed(ordered, false)
		if requireCompact {
			ordered = prioritizeOpenAICompactAccounts(ordered)
		}
		for _, acc := range ordered {
			fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel, false, platform)
			if fresh == nil {
				continue
			}
			fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, requireCompact, platform)
			if fresh == nil {
				continue
			}
			if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, fresh, requestedModel, requireCompact) {
				continue
			}
			result, err := s.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
			if err == nil && result.Acquired {
				if sessionHash != "" {
					_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, fresh.ID, openaiStickySessionTTL)
				}
				return s.newSelectionResult(ctx, fresh, true, result.ReleaseFunc, nil)
			}
		}
	} else {
		var available []accountWithLoad
		for _, acc := range candidates {
			loadInfo := loadMap[acc.ID]
			if loadInfo == nil {
				loadInfo = &AccountLoadInfo{AccountID: acc.ID}
			}
			if loadInfo.LoadRate < 100 {
				available = append(available, accountWithLoad{
					account:  acc,
					loadInfo: loadInfo,
				})
			}
		}

		if len(available) > 0 {
			sort.SliceStable(available, func(i, j int) bool {
				a, b := available[i], available[j]
				if a.account.Priority != b.account.Priority {
					return a.account.Priority < b.account.Priority
				}
				if a.loadInfo.LoadRate != b.loadInfo.LoadRate {
					return a.loadInfo.LoadRate < b.loadInfo.LoadRate
				}
				switch {
				case a.account.LastUsedAt == nil && b.account.LastUsedAt != nil:
					return true
				case a.account.LastUsedAt != nil && b.account.LastUsedAt == nil:
					return false
				case a.account.LastUsedAt == nil && b.account.LastUsedAt == nil:
					return false
				default:
					return a.account.LastUsedAt.Before(*b.account.LastUsedAt)
				}
			})
			shuffleWithinSortGroups(available)

			selectionOrder := make([]accountWithLoad, 0, len(available))
			if requireCompact {
				appendTier := func(out []accountWithLoad, tier int) []accountWithLoad {
					for _, item := range available {
						if openAICompactSupportTier(item.account) == tier {
							out = append(out, item)
						}
					}
					return out
				}
				selectionOrder = appendTier(selectionOrder, 2)
				selectionOrder = appendTier(selectionOrder, 1)
				// tier 0 闂傚倸鍊烽懗鍫曗€﹂崼銉︽櫇闁靛鏅滈崑锟犳煃閸濆嫭鍣归柣鎺戠仛閵囧嫰骞掗崱妞惧婵＄偑鍊х€靛矂宕板杈潟闁绘劕鎼猾宥夋煃瑜滈崜娑氬垝濞嗘劕绶為柟閭﹀厸缁卞爼姊洪崨濠冨闁稿瀚划濠氬蓟閵夛妇鍘甸梺缁樻尭濞寸兘骞楅悩宕囩闁告瑥顦辩粻鐐搭殽閻愯尙绠婚柟顔界矒閹稿﹥寰勫畝濠冃ラ梻鍌欒兌缁垶鎮烽妷鈺佺疇闁归偊鍠氶悳缁樹繆濡ゅ啫鐝?recheck 闂傚倸鍊风粈渚€骞栭锕€鐤い鏍仜绾惧潡鏌ゅù瀣澒闁稿鎸搁～婵嬪Ψ閵壯傜礄闁诲氦顫夊ú鈺冪礊娓氣偓閻涱喚鈧綆鍠楅崑鎰版煠绾板崬澧伴柡?cache tier 0 闂傚倷娴囬褎顨ョ粙鍖¤€块梺顒€绉寸壕濠氭煏閸繃濯奸柣?				// 闂備浇顕уù鐑藉箠閹捐绠熼梽鍥Φ閹版澘绀冮柕濞у嫭顔曢柣搴″帨閸嬫捇鏌嶈閸撶喖骞嗛埀顒併亜韫囨挻绁╂俊鍙夋そ閺?1/2闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬪鈹戦悩鎻掝仾闁哄棙绮撻弻鈩冨緞鐎ｅ墎绌块梺闈╁瘜閸樻悂宕戦幘缁樻櫜閹煎瓨绻勯惄搴ㄦ⒑鐠囪尙绠冲┑鐐╁亾闂佽鍠栫紞濠囧箖濠婂吘鐔兼惞闁稒妯婇梻鍌欐缁鳖喚绱為崱娑樼闁绘梻鍘ч弸浣衡偓骞垮劚濞茬娀宕戦幘鑸靛枂闁告洦鍋勭紞姒梙e 闂傚倷娴囬褏鎹㈤幇顔藉床闁瑰濮撮弸鍫⑩偓骞垮劚閹锋垿鎳撻幐搴涗簻闁规儳宕悘鈺冪磼閳ь剟宕卞☉娆屾嫼闂佸壊鐓堥崳顕€宕曢幇鐗堢厽闁圭虎鍨版禍楣冩⒒閸屾瑧鍔嶉悗绗涘懐鐭欓柟娆″眰鍔戦崺鈧い鎺戝€荤壕濂稿级閸稑濡跨紒鐘崇墱缁辨帡宕掑☉妯肩懆濡炪倖鎸搁妶绋跨暦閵娾晩鏁囬柍钘夋閺佺厧鈹戦悩鎰佸晱闁哥姵顨堥幑銏ゅ箛椤旇偐绛忔繛瀵稿Т椤戝棝宕戞径鎰厱妞ゆ劧绲剧粈鈧悗鐟版啞缁诲牓寮婚悢琛″亾濞戞顏嗙箔閳哄懏鐓曟慨妤€妫楁晶鎾煛?
				selectionOrder = appendTier(selectionOrder, 0)
			} else {
				selectionOrder = append(selectionOrder, available...)
			}

			for _, item := range selectionOrder {
				fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, item.account, requestedModel, false, platform)
				if fresh == nil {
					continue
				}
				fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, requireCompact, platform)
				if fresh == nil {
					continue
				}
				if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, fresh, requestedModel, requireCompact) {
					continue
				}
				result, err := s.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
				if err == nil && result.Acquired {
					if sessionHash != "" {
						_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, fresh.ID, openaiStickySessionTTL)
					}
					return s.newSelectionResult(ctx, fresh, true, result.ReleaseFunc, nil)
				}
			}
		}
	}

	// ============ Layer 3: Fallback wait ============
	sortAccountsByPriorityAndLastUsed(candidates, false)
	if requireCompact {
		candidates = prioritizeOpenAICompactAccounts(candidates)
	}
	for _, acc := range candidates {
		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel, false, platform)
		if fresh == nil {
			continue
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, requireCompact, platform)
		if fresh == nil {
			continue
		}
		if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, fresh, requestedModel, requireCompact) {
			continue
		}
		return s.newSelectionResult(ctx, fresh, false, nil, &AccountWaitPlan{
			AccountID:      fresh.ID,
			MaxConcurrency: fresh.Concurrency,
			Timeout:        cfg.FallbackWaitTimeout,
			MaxWaiting:     cfg.FallbackMaxWaiting,
		})
	}

	if requireCompact && baseCandidateCount > 0 {
		return nil, ErrNoAvailableCompactAccounts
	}
	return nil, ErrNoAvailableAccounts
}

func (s *OpenAIGatewayService) listSchedulableAccounts(ctx context.Context, groupID *int64, platform string) ([]Account, error) {
	platform = NormalizePlatformSlug(platform)
	if platform == "" {
		platform = PlatformOpenAI
	}
	if s.schedulerSnapshot != nil {
		accounts, _, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, false)
		return accounts, err
	}
	var accounts []Account
	var err error
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		accounts, err = s.accountRepo.ListSchedulableByPlatform(ctx, platform)
	} else if groupID != nil {
		accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, *groupID, platform)
	} else {
		accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatform(ctx, platform)
	}
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}
	return accounts, nil
}

func (s *OpenAIGatewayService) tryAcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (*AcquireResult, error) {
	if s.concurrencyService == nil {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {}}, nil
	}
	return s.concurrencyService.AcquireAccountSlot(ctx, accountID, maxConcurrency)
}

func (s *OpenAIGatewayService) resolveFreshSchedulableOpenAIAccount(ctx context.Context, account *Account, requestedModel string, requireCompact bool, platform string) *Account {
	if account == nil {
		return nil
	}

	fresh := account
	if s.schedulerSnapshot != nil {
		current, err := s.getSchedulableAccount(ctx, account.ID)
		if err != nil || current == nil {
			return nil
		}
		fresh = current
	}

	if !isOpenAIAccountEligibleForRequest(fresh, requestedModel, requireCompact, platform) {
		return nil
	}
	return fresh
}

func (s *OpenAIGatewayService) recheckSelectedOpenAIAccountFromDB(ctx context.Context, account *Account, requestedModel string, requireCompact bool, platform string) *Account {
	if account == nil {
		return nil
	}
	if s.schedulerSnapshot == nil || s.accountRepo == nil {
		if !isOpenAIAccountEligibleForRequest(account, requestedModel, requireCompact, platform) {
			return nil
		}
		return account
	}

	latest, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil || latest == nil {
		return nil
	}
	if !isOpenAIAccountEligibleForRequest(latest, requestedModel, requireCompact, platform) {
		return nil
	}
	return latest
}

func (s *OpenAIGatewayService) getSchedulableAccount(ctx context.Context, accountID int64) (*Account, error) {
	var (
		account *Account
		err     error
	)
	if s.schedulerSnapshot != nil {
		account, err = s.schedulerSnapshot.GetAccount(ctx, accountID)
	} else {
		account, err = s.accountRepo.GetByID(ctx, accountID)
	}
	if err != nil || account == nil {
		return account, err
	}
	return account, nil
}

func (s *OpenAIGatewayService) hydrateSelectedAccount(ctx context.Context, account *Account) (*Account, error) {
	if account == nil || s.schedulerSnapshot == nil {
		return account, nil
	}
	hydrated, err := s.schedulerSnapshot.GetAccount(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	if hydrated == nil {
		return nil, fmt.Errorf("selected openai account %d not found during hydration", account.ID)
	}
	return hydrated, nil
}

func (s *OpenAIGatewayService) newSelectionResult(ctx context.Context, account *Account, acquired bool, release func(), waitPlan *AccountWaitPlan) (*AccountSelectionResult, error) {
	hydrated, err := s.hydrateSelectedAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	return &AccountSelectionResult{
		Account:     hydrated,
		Acquired:    acquired,
		ReleaseFunc: release,
		WaitPlan:    waitPlan,
	}, nil
}

func (s *OpenAIGatewayService) schedulingConfig() config.GatewaySchedulingConfig {
	if s.cfg != nil {
		return s.cfg.Gateway.Scheduling
	}
	return config.GatewaySchedulingConfig{
		StickySessionMaxWaiting:  3,
		StickySessionWaitTimeout: 45 * time.Second,
		FallbackWaitTimeout:      30 * time.Second,
		FallbackMaxWaiting:       100,
		LoadBatchEnabled:         true,
		SlotCleanupInterval:      30 * time.Second,
	}
}

// GetAccessToken gets the access token for an OpenAI account
func (s *OpenAIGatewayService) GetAccessToken(ctx context.Context, account *Account) (string, string, error) {
	switch account.Type {
	case AccountTypeOAuth:
		if !account.IsOpenAI() {
			return "", "", fmt.Errorf("unsupported OAuth account platform for OpenAI-compatible gateway: %s", account.Platform)
		}
		// 濠电姷鏁搁崑鐘诲箵椤忓棛绀婇柍褜鍓氱换娑欏緞鐎ｎ偆顦伴悗?TokenProvider 闂傚倸鍊风粈渚€宕ョ€ｎ喖纾块柟鎯版鎼村﹪鏌ら懝鎵牚濞存粌缍婇弻娑㈠Ψ閵忊剝鐝栫紒鎯у綖缁瑩寮婚悢鐓庣畾鐟滃繘鏁嶅澶嬬厵闁伙絽鑻埢鍫ユ煛?token
		if s.openAITokenProvider != nil {
			accessToken, err := s.openAITokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return "", "", err
			}
			return accessToken, "oauth", nil
		}
		// 闂傚倸鍊搁崐鎼佸磹閸濄儮鍋撳鐓庡缂侇喗妫冮幊婊冣枔閹稿寒鍟嶉梻浣规偠閸庢椽宕滃璺虹柧妞ゆ帊鐒﹂崣蹇旂節闂堟稒顥炴繝鈧悗鏀卬Provider 闂傚倸鍊风粈渚€骞栭锔藉亱婵犲﹤瀚々鍙夈亜韫囨挾澧曢柛灞诲姂閺屾洟宕煎┑鍥х獩缂佹儳澧介崑鎾诲Φ閸曨垰绫嶉柛銉仢閹炬番浜滈敎濠氬炊閵娧冨箺闂備礁鐤囧銊╂嚄閸洖绠洪柣妯肩帛閸嬨劍銇勯弽銊︾殤闁圭晫濮风槐鎺楀磼濞戞鐟查梺璇″灡濡啯淇婇崼鏇炴そ濞达絽澧ｉ悢鍏尖拻濞达絽鎲￠崯鐐烘煟閻旀繂娲ら拑鐔奉熆閼搁潧濮﹂柡浣割儏闇夐柛蹇撳悑缂嶆垿鏌?
		accessToken := account.GetOpenAIAccessToken()
		if accessToken == "" {
			return "", "", errors.New("access_token not found in credentials")
		}
		return accessToken, "oauth", nil
	case AccountTypeAPIKey:
		apiKey := account.GetOpenAIApiKey()
		if apiKey == "" {
			return "", "", errors.New("api_key not found in credentials")
		}
		return apiKey, "apikey", nil
	default:
		return "", "", fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

func (s *OpenAIGatewayService) shouldFailoverUpstreamError(statusCode int) bool {
	switch statusCode {
	case 401, 402, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}

func (s *OpenAIGatewayService) shouldFailoverOpenAIUpstreamResponse(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if s.shouldFailoverUpstreamError(statusCode) {
		return true
	}
	return isOpenAITransientProcessingError(statusCode, upstreamMsg, upstreamBody)
}

func (s *OpenAIGatewayService) handleFailoverSideEffects(ctx context.Context, resp *http.Response, account *Account) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
}

// Forward forwards request to OpenAI API
func (s *OpenAIGatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	restrictionResult := s.detectCodexClientRestriction(c, account)
	apiKeyID := getAPIKeyIDFromContext(c)
	logCodexCLIOnlyDetection(ctx, c, account, apiKeyID, restrictionResult, body)
	if restrictionResult.Enabled && !restrictionResult.Matched {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"type":    "forbidden_error",
				"message": "This account only allows Codex official clients",
			},
		})
		return nil, errors.New("codex_cli_only restriction: only codex official clients are allowed")
	}

	originalBody := body
	reqModel, reqStream, promptCacheKey := extractOpenAIRequestMetaFromBody(body)
	originalModel := reqModel

	if account.Type == AccountTypeAPIKey && !openai_compat.ShouldUseResponsesAPI(account.Extra) {
		return s.forwardResponsesViaRawChatCompletions(ctx, c, account, body)
	}

	compatMessagesBridge := isOpenAICompatMessagesBridgeBody(body)
	setOpenAICompatMessagesBridgeContext(c, compatMessagesBridge)

	isCodexCLI := openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) || (s.cfg != nil && s.cfg.Gateway.ForceCodexCLI)
	wsDecision := s.getOpenAIWSProtocolResolver().Resolve(account)
	clientTransport := GetOpenAIClientTransport(c)
	// 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忛柟缁㈠枛缁犵娀骞栧ǎ顒€濡奸柛灞诲姂閺岀喓鈧數顭堟牎婵?WS 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁割偁鍎辩壕鍧楁煙閹澘袚闁稿鍠愰妵鍕冀閵娧呯厑闁诲孩纰嶅畝绋款潖濞差亜鍨傛い鏇炴噹閸撳啿鈹戦悩顐壕闂佸湱铏庨崰妤呭磻?WS 濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸劍閺呮繈鏌曟径鍡樻珕闁稿顦甸弻娑㈩敃閿濆棛顦ㄩ梺鍝勬媼閸撴盯鍩€椤掆偓閸樻粓宕戦幘缁樼厱闁哄洢鍔嬬花鐣岀磼鏉堛劌鍝烘慨濠呮缁瑧鎹勯妸褜鍞剁紓鍌欑椤︿即骞愰崘鑼殾闁汇垹澹婇弫鍥煏韫囧ň鍋撻崗鍛版?HTTP -> WS 闂傚倸鍊风粈渚€骞夐敓鐘偓鍐川椤栨稑搴婂┑掳鍊曢幏瀣极婵犲洦鐓曟繛鎴濆船鐢酣骞栧ǎ顒€鐒烘俊鎻掔秺楠炴牕菐椤掆偓閻忣亝绻涢崼娑樼伈婵?
	wsDecision = resolveOpenAIWSDecisionByClientTransport(wsDecision, clientTransport)
	if c != nil {
		c.Set("openai_ws_transport_decision", string(wsDecision.Transport))
		c.Set("openai_ws_transport_reason", wsDecision.Reason)
	}
	if wsDecision.Transport == OpenAIUpstreamTransportResponsesWebsocketV2 {
		logOpenAIWSModeDebug(
			"selected account_id=%d account_type=%s transport=%s reason=%s model=%s stream=%v",
			account.ID,
			account.Type,
			normalizeOpenAIWSLogValue(string(wsDecision.Transport)),
			normalizeOpenAIWSLogValue(wsDecision.Reason),
			reqModel,
			reqStream,
		)
	}
	// 闂備浇宕甸崰鎰垝鎼淬垺娅犳俊銈呮噹缁犱即鏌涘☉姗堟敾婵炲懐濞€閺岋絽螣閾忕櫢绱炲銈傛櫇閸忔﹢寮诲☉妯锋闁告鍋為悘宥夋⒑閸濆嫭顥炵紒顔芥崌瀵?WSv2闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟鐑橆殕閸庢鏌涢弽顐ｎ吂1 闂傚倸鍊风粈渚€骞夐敍鍕煓闁圭儤顨呴崹鍌涚節闂堟侗鍎愰柛銈呯墦閺岀喐娼忛崜褏鏆犵紓渚囧亜缁夊綊寮诲☉銏犵闁肩⒈鍓欐俊鐣岀磽娴ｉ缚妾搁柛銊ョ埣瀵鎮㈤悡搴ｇ暢闂佸湱鍎ら崹鍨叏閵堝洨纾藉ù锝堟鐢盯鏌涢妸銉т虎闁伙絿鍏樺畷濂稿即閻斿皝鍋撻悜鑺ョ厽闁瑰鍊栭幋婵冩瀺闁瑰墽绮悡鐔兼煟濡搫绾х紒浣叉櫊閺屾盯濡歌濞呮洘淇婇崣澶婂闁宠鍨归埀顒婄秵閸嬪棝宕濋崼鏇熲拺闁告捁灏欓崢娑㈡煕閻樺磭澧遍柡渚囧櫍楠炲酣鎳為妷褍骞堥梻浣侯攰閹活亞寰婃ィ鍐炬晜闁绘ɑ鍓氬▓浠嬫煟閹邦剙绾ч柛鐘筹耿閺屾盯濡搁敃鈧崝銈嗐亜椤撴粌濮傜€规洜鍘ч埞鎴﹀幢濮樹究鍋炵紓鍌氬€搁崐鎼佸磹閹间礁绐楁慨妯挎硾缁愭鏌″搴″幍濞存粌缍婇弻鐔虹磼濡桨鍒婇梺鎼炲€栧ú鐔煎蓟瀹ュ牜妾ㄩ梺鍛婃尰瀹€鎼佺嵁韫囨稑宸濋柡澶嬪灥閼板灝鈹戦鐭亪宕ョ€ｎ喖妫橀柍褜鍓欓埞鎴︽倷閺夋垹浠搁梺褰掆偓娑氼槮妞も晛銈告俊姝岊槾缂佺姵鏌ㄩ…璺ㄦ崉閻戞ɑ鎷遍梺绋跨箲缁秹濡甸崟顖氱睄闁搞儜鍌涚潖闂備礁鎼Λ鏃堝础閹惰棄钃熺€广儱顦敮闂佹寧鏌ㄦ晶鐣屾娴煎瓨鈷?
	if wsDecision.Transport == OpenAIUpstreamTransportResponsesWebsocket {
		if c != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"type":    "invalid_request_error",
					"message": "OpenAI WSv1 is temporarily unsupported. Please enable responses_websockets_v2.",
				},
			})
		}
		return nil, errors.New("openai ws v1 is temporarily unsupported; use ws v2")
	}
	passthroughEnabled := account.IsOpenAIPassthroughEnabled()
	if passthroughEnabled {
		// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ幗瀹曟煡鎮楅敐搴″妞ゎ偅娲熼弻娑㈩敃閿濆棛顦ョ紓浣插亾闁糕剝绋掗悡鏇㈡煃閳轰礁褰侀柟杈剧畱濮瑰弶绻濇繝鍌滃闁绘挻娲樼换娑㈠幢濡繐銈搁弻鍥敍濞戞氨顔曢梺瑙勫劤閸熷潡銆傞幎鑺ョ厵闁肩⒈鍓欓。濂告煙瀹勭増鍤囨鐐差儔楠炴帡骞橀搹顐ｆ殬闂傚倸鍊搁崐鐑芥倿閿曚降浜归柛鎰典簼瀹曟煡鏌熸潏鍓хシ濞存粎鎳撻妴鎺戭潩閿濆懍澹曢柣搴ゎ潐濞测晝绱炴担鍓插殨闁稿﹦鍣ュΣ楣冩⒑閸︻厽娅曢柛鐘冲姉閹广垹鈽夐姀鐘诲敹闂侀潧绻嗛弲婊冣枔閹殿喚纾藉ù锝勭矙閸濊櫣绱掔拠鑼ⅵ鐎殿喖顭峰畷鍗炩槈濡櫣鈧厼顪冮妶鍡樷拻闁告鍛皫閻庯綆鍠楅埛鎴犵磽娴ｅ顏堟倿妤ｅ啯鐓曟繛鍡楄嫰娴滈箖姊绘担瑙勩仧闁告ü绮欏畷鏇㈠箮缁涘鏅梺闈涱槴閺呮稓绮堢€ｎ偁浜滈柟鎵虫櫅閻忊晛霉閻撳酣鍙勬慨?Unmarshal闂?
		reasoningEffort := extractOpenAIReasoningEffortFromBody(body, reqModel)
		return s.forwardOpenAIPassthrough(ctx, c, account, originalBody, reqModel, reasoningEffort, reqStream, startTime)
	}

	reqBody, err := getOpenAIRequestBodyMap(c, body)
	if err != nil {
		return nil, err
	}

	if v, ok := reqBody["model"].(string); ok {
		reqModel = v
		originalModel = reqModel
	}
	if v, ok := reqBody["stream"].(bool); ok {
		reqStream = v
	}
	if promptCacheKey == "" {
		if v, ok := reqBody["prompt_cache_key"].(string); ok {
			promptCacheKey = strings.TrimSpace(v)
		}
	}
	apiKey := getAPIKeyFromContext(c)
	imageGenerationAllowed := GroupAllowsImageGeneration(nil)
	if apiKey != nil {
		imageGenerationAllowed = GroupAllowsImageGeneration(apiKey.Group)
	}
	codexImageGenerationBridgeEnabled := isCodexCLI && imageGenerationAllowed && s.isCodexImageGenerationBridgeEnabled(ctx, account, apiKey)
	if IsImageGenerationIntentMap(openAIResponsesEndpoint, reqModel, reqBody) && !imageGenerationAllowed {
		setOpsUpstreamError(c, http.StatusForbidden, ImageGenerationPermissionMessage(), "")
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"type":    "permission_error",
				"message": ImageGenerationPermissionMessage(),
			},
		})
		return nil, errors.New("image generation disabled for group")
	}

	// Track if body needs re-serialization
	bodyModified := false
	// 闂傚倸鍊风粈渚€骞夐敓鐘偓鍐幢濡炴洘妞藉浠嬵敇閻旇渹妲愰柣鐔哥矊缁夊綊宕洪姀銈呯閻庢稒锚椤庢捇姊虹粙鎸庣闁搞劋鍗宠棟闁规鍠氶惌娆忣熆鐠轰警鍎愬┑顖涙尦閺屾稑鈽夊鍫濅紣濠电偞鎸婚崝娆忣潖缂佹鐟归柛銉戝倻鏁栭梻浣告憸婵潧煤閻旂厧绠氱€光偓閸曨偆鐣鹃悷婊勭矒瀹曢潧螖閸涱喚鍘遍梺鍝勬川閸嬫盯銆傚畷鍥╃＜闁归偊鍙庡▓婊堟煛鐏炲墽鈽夐柍璇查铻ｉ悷娆忓瀹曡櫕淇婇悙顏勨偓鎰板疾椤忓棛涓嶉柟鐐墯閸ゆ洘銇勯幇鈺佲偓妤冪矆閸曨垱鐓熸俊顖涱儥閸ゆ瑦绻涢崼鐔虹疄婵﹥妞藉畷銊︾節閸曨剙娅楅梺钘夊暢妞村摜鎹㈠☉銏犵闁圭儤鎸告禒鏉戔攽椤旂》宸ユ繛灏栤偓宕囨殾闁告鍊ｉ悢鐑樺鐎规洖娲ょ粭锛勭磽閸屾艾鈧悂宕愭搴ｇ焼濞撴埃鍋撻柟顔ㄥ洤骞㈡慨妯稿劙濮规姊洪懝鏉款棈缂傚牏澧楃粋宥堛亹閹烘挾鍘甸梺缁樺灦钃遍柣鎾卞劜閵囧嫰寮崠鈥充紣闂佸疇妫勯ˇ鐢哥嵁濮椻偓閹煎綊顢曢敐鍛闂傚倷娴囬鏍窗濡ゅ懎绠伴柤濮愬€楅惌娆忣熆閼搁潧濮囬柣鎺戠仛閵囧嫰骞掗幋婵冨亾閻熸壆鏆﹀璺侯儍娴滄粓鏌″鍐ㄥ闁汇劍鍨圭槐鎺楀箛椤旈棿妲愰梺?set/delete闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸ゆ劖銇勯弽銊ュ毈婵炲皷鏅犻弻鏇熺箾閻愵剚鐝曢梺缁樺笒閵堟悂寮诲☉鈶┾偓锕傚箣濠靛洨浜梺姹囧焺閸ㄨ鲸绻涙繝鍥ц摕婵炴垶绮庨悿鈧柣搴秵娴滄粓锝為崶顒佲拺?Marshal闂?
	patchDisabled := false
	patchHasOp := false
	patchDelete := false
	patchPath := ""
	var patchValue any
	markPatchSet := func(path string, value any) {
		if strings.TrimSpace(path) == "" {
			patchDisabled = true
			return
		}
		if patchDisabled {
			return
		}
		if !patchHasOp {
			patchHasOp = true
			patchDelete = false
			patchPath = path
			patchValue = value
			return
		}
		if patchDelete || patchPath != path {
			patchDisabled = true
			return
		}
		patchValue = value
	}
	markPatchDelete := func(path string) {
		if strings.TrimSpace(path) == "" {
			patchDisabled = true
			return
		}
		if patchDisabled {
			return
		}
		if !patchHasOp {
			patchHasOp = true
			patchDelete = true
			patchPath = path
			return
		}
		if !patchDelete || patchPath != path {
			patchDisabled = true
		}
	}
	disablePatch := func() {
		patchDisabled = true
	}

	// 闂傚倸鍊搁崐鎼佸磹閹间焦鍋嬮煫鍥ㄧ☉绾惧鏌曢崼婵愭Ц闁绘帒鐏氶妵鍕箳閸℃ぞ澹曠紓鍌欐祰椤曆囶敄閸ヮ灛锝夊箛椤旇棄纾繛杈剧稻濞叉粓宕伴弽顓炲瀭闁诡垎鍛闂佹悶鍎弲鈺呭绩閵娿儮鏀介柣鎰綑閻忥箓鏌熼崨濠冨€愭い銏＄墵楠炲鎮╅幇浣圭稐婵犵數濮靛ú锕傚蓟瀹告ⅸructions 濠电姷鏁搁崑鐐哄垂閸洖绠插ù锝囩《閺嬪秹鏌ㄥ┑鍡╂Ц闁绘挻锕㈤弻鈥愁吋鎼粹€崇缂備緡鍋勭粔褰掑蓟閵堝洨鐭欓悹鎭掑妺缁敻姊洪崫鍕棞闁绘鎹囧璇测槈濠婂懐鏉搁柣搴秵娴滄粍鎱ㄩ敂鍓х＝濞撴艾娲ゅ▍姗€鏌涢妸锕€鈻曟鐐茬箳閳ь剨缍嗛崰鏍ф纯闂備胶纭堕崜婵嬨€冮崼婢喖宕熼鍌滎啎闁诲海鏁搁…鍫ｂ叿闂備胶顭堥敃銉╁垂閸ф鏄ラ柍?
	if isInstructionsEmpty(reqBody) && !compatMessagesBridge {
		reqBody["instructions"] = "You are a helpful coding assistant."
		bodyModified = true
		markPatchSet("instructions", "You are a helpful coding assistant.")
	}

	if codexImageGenerationBridgeEnabled && ensureOpenAIResponsesImageGenerationTool(reqBody) {
		bodyModified = true
		disablePatch()
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Injected /responses image_generation tool for Codex client")
	}

	if normalizeOpenAIResponsesImageGenerationTools(reqBody) {
		bodyModified = true
		disablePatch()
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Normalized /responses image_generation tool payload")
	}
	if codexImageGenerationBridgeEnabled && applyCodexImageGenerationBridgeInstructions(reqBody) {
		bodyModified = true
		disablePatch()
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Added Codex image_generation bridge instructions")
	}

	// 闂傚倷娴囬褏鑺遍懖鈺佺筏濠电姵鐔紞鏍ь熆鐠轰警鍎愭繛鍛Х閳ь剙绠嶉崕閬嵥囬婧惧亾濮橆剦妲告い顓℃硶閹瑰嫰宕崟顓熜為梻浣规偠閸婃宕戦悙鍝勭疄闁靛ň鏅滈崑锟犳煛婢跺﹦姘ㄩ柛瀣尵閹叉挳宕熼浣哄娇闂備椒绱徊浠嬪嫉椤掑嫬纾绘慨妞诲亾闁诡喗锕㈤幃娆撳礂閸濄儳鈹涘┑鐐存尰閸╁啴宕戦幘鍨涘亾鐟欏嫭绀€婵犫偓闁秴鐒垫い鎺戯功缁夌敻鏌涚€ｎ亝鍤€闁靛棙鐗犲铏规嫚閼碱剛顔囩紓浣虹帛閸旀鍩€椤掍浇澹橀柟鑺ョ矊鍗遍柟閭﹀厴閺嬪酣鏌熼幆褏锛嶆い锕備憾濮婃椽宕ㄦ繝浣虹箒闂佸憡眉缁瑥鐣?Codex CLI闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆¤娲、姗€濮€閻橀潧濮?
	billingModel := account.GetMappedModel(reqModel)
	if billingModel != reqModel {
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Model mapping applied: %s -> %s (account: %s, isCodexCLI: %v)", reqModel, billingModel, account.Name, isCodexCLI)
		reqBody["model"] = billingModel
		bodyModified = true
		markPatchSet("model", billingModel)
	}
	upstreamModel := billingModel
	if imageGenerationAllowed && normalizeOpenAIResponsesImageOnlyModel(reqBody) {
		bodyModified = true
		disablePatch()
		if model, ok := reqBody["model"].(string); ok {
			upstreamModel = strings.TrimSpace(model)
		}
		logger.LegacyPrintf(
			"service.openai_gateway",
			"[OpenAI] Normalized /responses image-only model request inbound_model=%s image_model=%s upstream_model=%s",
			reqModel,
			billingModel,
			upstreamModel,
		)
	}
	if err := validateOpenAIResponsesImageModel(reqBody, upstreamModel); err != nil {
		setOpsUpstreamError(c, http.StatusBadRequest, err.Error(), "")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": err.Error(),
				"param":   "model",
			},
		})
		return nil, err
	}
	if hasOpenAIImageGenerationTool(reqBody) {
		logger.LegacyPrintf(
			"service.openai_gateway",
			"[OpenAI] /responses image_generation request inbound_model=%s mapped_model=%s account_type=%s",
			reqModel,
			upstreamModel,
			account.Type,
		)
	}
	if err := validateCodexSparkInput(reqBody, upstreamModel); err != nil {
		setOpsUpstreamError(c, http.StatusBadRequest, err.Error(), "")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": err.Error(),
				"param":   "input",
			},
		})
		return nil, err
	}

	// Compact-only model 闂傚倸鍊风粈渚€骞栭銈傚亾濮樼厧鏋熼柟渚垮姂楠炴﹢顢欓挊澶婂闂備胶绮…鍥╁垝椤栫偛鐤炬い鎺嶇劍閸欏繑鎱ㄥΔ鈧Λ妤佺閻愮儤鐓熸い鎾跺仧閻掓悂鏌?/responses/compact 闂傚倷娴囧畷鍨叏瀹曞洦濯伴柨鏇炲€搁崹鍌炴煙濞堝灝鏋熸い鎰矙閺岋綁骞囬鐓庡闂佺顑冮崝鎴﹀蓟閿濆妫橀柟绋块閺嗗牆鈹戦悙鍙夊櫣缂佺粯锕㈠濠氬Ω閵夈垺顫嶅┑鈽嗗灟鐠€锕€鈻撻鐘电＝濞达綀顕栧▓鏇㈡煟濡や緡娈橀柍褜鍓涢弫鎼佸储瑜戦悘鎺楁⒑閸涘﹦绠撻悗姘煎墮椤曪絽螖閸愵亞锛濇繛杈剧到閹碱偊鎮甸敓鐘崇厱闁靛鍨哄▍鍛存煢閸屾凹鍎愮紒?	// OAuth 婵犵數濮烽。钘壩ｉ崨鏉戝瀭妞ゅ繐鐗嗛悞鍨亜閹哄棗浜剧紒鍓ц檸閸欏啴宕洪埀顒併亜閹烘垵顏存俊顐ｅ灴閺岀喖宕ｆ径瀣攭閻庤娲滈崰鏍€佸▎鎾村殐闁冲搫鍊愰妸鈺傗拻濞达絽鎲￠崯鐐烘煟濡や緡娈滈柟顔ㄥ浂鏁囬柣妯诲墯濞肩喎鈹戦悙鍙夘棡闁圭顭烽幃鐐寸附閸涘﹦鍘卞銈庡幗閸ㄧ敻寮稿☉銏″仺?OAuth 闂傚倷娴囧畷鐢稿窗閹扮増鍋￠柕澹偓閸嬫挸顫濋悡搴ｄ化闂侀€炲苯澧柣蹇斿哺閹囨偐閼碱剚娈鹃梺缁橆焾椤曆囧礃閳ь剙顪冮妶鍡楃瑨妞わ缚鍗抽幊婊嗐亹閹烘挴鎷?compact-only 闂傚倸鍊烽懗鍫曞储瑜旈妴鍐╂償閵忋埄娲稿┑鐘诧工閻楀﹪宕戦埡鍛厽闁逛即娼ф晶浼存煃缂佹ɑ绀€妞ゎ叀娉曢幑鍕惞鐟欏嫬鏆┑鐐存尰閸╁啴宕戦幘鍨涘亾鐟欏嫭绀€婵犫偓闁秴鐒垫い鎺戯功閸掓澘顭胯濞撮鍒掗崼銉ュ耿婵炴垶鐟ч崢?
	isCompactRequest := isOpenAIResponsesCompactPath(c)
	compactMapped := false
	if isCompactRequest {
		compactMappedModel := resolveOpenAICompactForwardModel(account, billingModel)
		if compactMappedModel != "" && compactMappedModel != billingModel {
			compactMapped = true
			upstreamModel = compactMappedModel
			reqBody["model"] = compactMappedModel
			bodyModified = true
			markPatchSet("model", compactMappedModel)
			logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Compact model mapping applied: %s -> %s (account: %s, isCodexCLI: %v)", billingModel, compactMappedModel, account.Name, isCodexCLI)
		}
	}

	// OpenAI OAuth 闂傚倷娴囧畷鍨叏閻㈢绀夌憸蹇曞垝婵犳艾绠ｉ柨婵嗗暕濮规鏌ｉ悩鍏呰埅闁告柨绉舵禍?ChatGPT internal Codex endpoint闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸庢鏌涚仦鎯ф惛闁逞屽墯鐢€崇暦濠婂嫭濯撮柣鐔哄濮ｅ洤鈹戦悙鑸靛涧缂佽弓绮欓獮妤€顭ㄩ崘锝嗙亖闂佺粯鍨剁湁缂佽妫欓妵鍕冀閵婏絼绮堕梺绋款儐閹告悂鎮鹃悜钘夌倞闁挎繂鎳嶆竟鏇㈡⒑缂佹ê鐏辨俊顐㈠閹叉挳鏁傞柨顖氫壕妤犵偛鐏濋崝姘舵煟濡や胶鐭婃い顓炴喘楠炲洭鎮ч崼銏犲箻闂備胶绮幐鍛婎殽閹间礁姹叉繛鍡樻尰閸婄敻姊婚崼鐔衡槈缂佺嫏鍛＜?	// 濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸劍閺呮繈鏌曟径鍡樻珕闁稿顦甸弻娑㈩敃閿濆棛顦ラ梺娲诲幗椤ㄥ棝濡甸崟顖氬唨闁靛ě鍛毉婵犵鈧啿绾ф繛鑼枛瀵鈽夐姀鐘殿唺濠电娀娼уΛ鏃傝姳閽樺鏀?Codex/GPT 缂傚倸鍊搁崐椋庢閿熺姴鍨傞柣銏犲閺佸鎲搁弬璺ㄦ殾鐟滅増甯╅弫鍌炴煕椤愶絾鍎曢柕澶堝劘閳ь剚甯掗～婵嬪础閻愰潧袩I Key 闂傚倷娴囧畷鍨叏閻㈢绀夌憸蹇曞垝婵犳艾绠ｉ柨婵嗗暕濮规姊虹粔鍡楀濞堟梻绱掗埀顒勫幢濡偐顔曢梺绯曞墲椤ㄥ牏绮婚幎鑺ュ€甸柣鐔煎亰濡叉悂鎳ｉ幇鐗堢厱闁靛绲芥俊璺ㄧ磼閻樼鑰块柡宀嬬節瀹曡精绠涢幘鎻掝棜闂傚倸鍊风粈渚€骞夐敓鐘偓锕傚炊椤掆偓缁愭骞栭幖顓犲帨缂?闂傚倸鍊风粈渚€骞栭銈傚亾濮樼厧鏋熼柟渚垮姂楠炴﹢顢欓挊澶婂闂備胶绮…鍫ヮ敋瑜忛幉鎾晝閸屾氨顔愰柡澶婄墕婢х晫绮欐繝姘厽闁挎繂鐗撻崫鍝勄庨崶褝韬い銏＄☉椤啰鎷犻煫顓烆棜闁诲氦顫夊ú鏍洪悩璇茬；闁瑰墽绮崐濠氭煢濡警妲肩悮婵嬫煟鎼达紕鐣柛搴や含閹广垽骞掗弮鍌滅劶?
	// 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忓ù鍏兼綑绾惧潡姊洪鈧粔瀵告喆閿曞倹鍊甸柨婵嗛閸樻挳鏌涚€ｎ偅宕岀€规洜鍏橀、妯款槼闁轰緡鍨跺娲箹閻愭祴鍋撻弽顐㈠灊閹兼番鍔岄悞鍨亜閹烘垵鏋ゆ繛鍏煎姍閺岋綁顢樿閺嬨倗绱?base_url 闂?OpenAI-compatible 濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸劍閺呮繈鏌曟径鍡樻珕闁稿顦甸弻娑㈩敃閿濆棛顦ラ弶?
	if model, ok := reqBody["model"].(string); ok {
		if !compactMapped {
			upstreamModel = normalizeOpenAIModelForUpstream(account, model)
			if upstreamModel != "" && upstreamModel != model {
				logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Upstream model resolved: %s -> %s (account: %s, type: %s, isCodexCLI: %v)",
					model, upstreamModel, account.Name, account.Type, isCodexCLI)
				reqBody["model"] = upstreamModel
				bodyModified = true
				markPatchSet("model", upstreamModel)
			}
		}

		if !SupportsVerbosity(upstreamModel) {
			if text, ok := reqBody["text"].(map[string]any); ok {
				delete(text, "verbosity")
			}
		}
	}

	// 闂傚倷娴囧畷鐢稿窗閹扮増鍋￠柕澹偓閸嬫挸顫濋悡搴ｄ化闂侀€炲苯澧柣蹇斿哺閹囨偐閼碱剚娈?reasoning.effort 闂傚倸鍊风粈渚€骞夐敓鐘冲仭闁靛鏅涚壕鍦喐閻楀牆绗掓慨瑙勭叀閺岋綁寮崹顔藉€梺鍝勬媼閸撶喖寮诲☉婊庢闂佸搫鎳愬﹢寤糾al -> none闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆″眰鍔戦崺鈧い鎺戝€荤壕濂稿级閸稑濡跨紒鐘崇墱缁辨帞鈧綆浜跺Σ褰掓煙椤栨稒鐓ラ柍瑙勫灴瀹曞ジ濮€閳ヨ櫕娈舵繝鐢靛Х閺佹悂宕戦悙鍝勭濡わ絽鍟犻埀顒婄畵瀹曞ジ濡烽妷銉ヤ憾闂備胶鎳撻顓熸叏閸偄鍨旈柤娴嬫櫇绾惧ジ鎮楅敐搴濈敖婵ǜ鍔戦弻娑氣偓锝庡亝瀹曞矂鏌熼鐟板⒉闁诡垱妫冮、娆撳礂绾板崬鏅犲┑鐘愁問閸犳鏁悙闈涘灊妞ゆ牜鍋涚粈澶愭煙鏉堝墽鐣遍柣?
	if reasoning, ok := reqBody["reasoning"].(map[string]any); ok {
		if effort, ok := reasoning["effort"].(string); ok && effort == "minimal" {
			reasoning["effort"] = "none"
			bodyModified = true
			markPatchSet("reasoning.effort", "none")
			logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Normalized reasoning.effort: minimal -> none (account: %s)", account.Name)
		}
	}

	if account.Type == AccountTypeOAuth {
		codexResult := codexTransformResult{}
		if compatMessagesBridge {
			codexResult = applyCodexOAuthTransformWithOptions(reqBody, codexOAuthTransformOptions{
				IsCodexCLI:              isCodexCLI,
				IsCompact:               isCompactRequest,
				SkipDefaultInstructions: true,
				PreserveToolCallIDs:     true,
			})
			ensureCodexOAuthInstructionsField(reqBody)
			bodyModified = true
			disablePatch()
		} else {
			codexResult = applyCodexOAuthTransform(reqBody, isCodexCLI, isCompactRequest)
		}
		if codexResult.Modified {
			bodyModified = true
			disablePatch()
		}
		if codexResult.NormalizedModel != "" {
			upstreamModel = codexResult.NormalizedModel
		}
		if codexResult.PromptCacheKey != "" {
			promptCacheKey = codexResult.PromptCacheKey
		}
	}

	// Handle max_output_tokens based on platform and account type
	if !isCodexCLI {
		if maxOutputTokens, hasMaxOutputTokens := reqBody["max_output_tokens"]; hasMaxOutputTokens {
			switch account.Platform {
			case PlatformOpenAI:
				// For OpenAI API Key, remove max_output_tokens (not supported)
				// For OpenAI OAuth (Responses API), keep it (supported)
				if account.Type == AccountTypeAPIKey {
					delete(reqBody, "max_output_tokens")
					bodyModified = true
					markPatchDelete("max_output_tokens")
				}
			case PlatformAnthropic:
				// For Anthropic (Claude), convert to max_tokens
				delete(reqBody, "max_output_tokens")
				markPatchDelete("max_output_tokens")
				if _, hasMaxTokens := reqBody["max_tokens"]; !hasMaxTokens {
					reqBody["max_tokens"] = maxOutputTokens
					disablePatch()
				}
				bodyModified = true
			case PlatformGemini:
				// For Gemini, remove (will be handled by Gemini-specific transform)
				delete(reqBody, "max_output_tokens")
				bodyModified = true
				markPatchDelete("max_output_tokens")
			default:
				// For unknown platforms, remove to be safe
				delete(reqBody, "max_output_tokens")
				bodyModified = true
				markPatchDelete("max_output_tokens")
			}
		}

		// Also handle max_completion_tokens (similar logic)
		if _, hasMaxCompletionTokens := reqBody["max_completion_tokens"]; hasMaxCompletionTokens {
			if account.Type == AccountTypeAPIKey || account.Platform != PlatformOpenAI {
				delete(reqBody, "max_completion_tokens")
				bodyModified = true
				markPatchDelete("max_completion_tokens")
			}
		}

		// Remove unsupported fields (not supported by upstream OpenAI API)
		unsupportedFields := []string{"prompt_cache_retention", "safety_identifier"}
		for _, unsupportedField := range unsupportedFields {
			if _, has := reqBody[unsupportedField]; has {
				delete(reqBody, unsupportedField)
				bodyModified = true
				markPatchDelete(unsupportedField)
			}
		}
	}

	// 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忛柟缁㈠枛缁犵娀骞栫划瑙勵潐闁?WSv2 婵犵數濮烽。钘壩ｉ崨鏉戝瀭妞ゅ繐鐗嗛悞鍨亜閹哄棗浜剧紒鍓ц檸閸樻儳鈽夐悽绋跨劦妞ゆ帊鑳剁粻楣冩煥濠靛棙鍣哄璺哄缁辨帞鎷犻幓鎺濅純閻庢鍠楁繛濠囧春閳?previous_response_id闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸ゆ劖銇勯弽顐粶闁活厽顨婇弻锝夊箛闂堟稑顫銈傛櫇閸忔﹢寮婚垾鎰佸悑閹艰揪绲界敮鎾寸箾鏉堝墽鍒板鐟帮躬瀹曠敻寮撮姀銏犲絼闂佹悶鍎崕閬嶅礉閵堝洨纾奸柍褜鍓熷畷娆撀ㄥ宄?WSv1闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆¤娲﹀蹇涘煛娴ｅ摜浜伴梻浣虹帛閸ㄥ爼鈥﹂崶顒€鐓濋柡鍐ㄧ墛閻撳啰鎲稿鍫濈婵﹩鍓﹂悞钘夆攽閻樺疇澹橀柣鎰躬閺屾洘绻濊箛鏇犳殸閻炴碍锕㈠?	// 婵犵數濮烽弫鎼佸磻濞戔懞鍥敇閵忕姷顦悗骞垮劚椤︻垳绮堥崼婢濆綊鎮℃惔锝嗘喖闂佸搫鎷嬮崜姘跺箞閵娿儙鐔烘嫚閹绘帩娼介梻浣规偠閸婃洟藝閻㈢绠栫€瑰嫭澹嬮弸搴ㄧ叓閸ャ劍鎯勫ù鐘层偢濮婃椽宕滈懠顒€甯ラ梺鎼炲姀濞夋盯鎮鹃悜钘壩ㄧ憸灞解柦椤忓牊鐓曢柟鐐殔閸犳艾螞婵犲洦鈷掗柛灞剧懅鐠愪即鏌涚€ｎ亜顏€规洘鍔栫换婵嗩潩椤掍胶鈧剟姊洪棃娑氬婵☆偅鐩妴?Codex CLI 闂傚倷娴囧畷鍨叏閺夋嚚娲敇閵忕姷鍝楅梻渚囧墮缁夌敻宕曢幋婢濆綊鎮℃惔锝嗘喖闂佸搫鎷嬮崜娑㈠焵椤掆偓閸樻粓宕戦幘缁樼厱闁哄洢鍔嬬花鐣岀磼鏉堛劌鍝烘慨?WSv1 闂傚倸鍊风粈渚€骞夐敍鍕瀳鐎广儱顦崹鍌溾偓瑙勬礀濞层劎绮婚弮鍫熺厱閻忕偞鍎抽幖鎼佹煃瑜滈崜姘辨暜閿熺姷宓侀柡宥冨妽婵挳鏌熺紒妯虹瑐濠㈣娲熷Λ鍛搭敃閵忊剝鎮欏┑鐐差嚟閸忔﹢骞冨▎鎰瘈闁稿本顨嗛～宥夋⒑鐟欏嫬鍔ょ痪缁㈠幗閻楀酣姊绘担鑺ャ€冪紒璁圭節瀹曟娊鏁愭径灞界ウ闂佸綊鍋婇崢鑺ュ垔閹绢喗鐓曟繝闈涘閸旀粓鏌熼懠棰濆殭闁宠鍨块幃娆撴嚑椤掆偓閳亶姊虹粙娆惧劀缂傚秳绶氬畷娲焵?
	if wsDecision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
		if _, has := reqBody["previous_response_id"]; has {
			delete(reqBody, "previous_response_id")
			bodyModified = true
			markPatchDelete("previous_response_id")
		}
	}

	if sanitizeEmptyBase64InputImagesInOpenAIRequestBodyMap(reqBody) {
		bodyModified = true
		disablePatch()
	}

	// Apply OpenAI fast policy (闂傚倸鍊风粈渚€骞夐敓鐘冲仭闁靛鏅涚壕鐟扳攽閻樺疇澹橀柣?Claude BetaPolicy 闂?fast-mode 闂傚倷绀侀幖顐λ囬锕€鐤炬繛鎴炩棨濞差亝鏅濋柛灞炬皑椤?闂?	// 闂傚倸鍊搁崐宄懊归崶顬盯宕熼娑樹罕闂佺硶鍓濇笟妤呭极?body 闂?service_tier 闂傚倷娴囬褏鈧稈鏅濈划娆撳箳濡炲皷鍋撻崘顔煎耿婵炴垼椴搁弲鈺呮倵閸忓浜鹃梺鍛婃处閸撴艾鈻?priority" 闂?fast闂?flex"闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆″眰鍔戦崺鈧い鎺戝€荤壕濂稿级閸稑濡跨紒鐘靛仦椤ㄣ儵鎮欓弶鎴濐潔閻庡灚婢樼€氼剟鎮惧┑瀣劦妞ゆ帒瀚粻鐑樼節婵犲倻澧涢柣?	// 闂傚倸鍊风粈浣革耿闁秵鍋￠柟鎯版楠炪垽鏌嶉崫鍕偓褰掑级?filter闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬪鏌涢埄鍐槈闁搞劌鍊婚幉鎼佹偋閸繄鐟查梺鎶芥敱濡啴寮婚弴銏犻唶婵犻潧娲ゅ▍褔姊洪崷顓熸珪闁哥姵鍔楅幑銏犫槈閵忕娀鍞堕梺闈涚箚閺呮粌鈻撻幍顔剧＝濞达絼绮欓崫铏光偓鍏夊亾闁归棿鑳跺畵?block闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬪鈹戦悩鎻掝仱闁稿鎸搁～婵嬵敆娓氬洦锟ョ紓鍌欐祰妞村摜鏁幒妤€鐓橀柟瀵稿仧缁♀偓闁硅偐琛ラ崜婵嬵敊婢跺绡€婵炲牆鐏濋弸銈夋煙閾忣偅宕岀€规洘鍨剁换婵嬪礋椤掑缍楁繝娈垮枟椤牆鈻斿☉婊呯闂傚倷鑳堕…鍫ュ嫉椤掑嫬绀勭憸鐗堝笒閸?gpt-5.5 缂傚倸鍊搁崐鐑芥倿閿斿墽鐭欓柟娆¤娲獮蹇曚沪閼恒儲顔夐梻鍌氬€烽悞锕傚箖閸洖绀夌€光偓閸曞灚鏅為梺鍛婄☉閻°劑宕曢幘缁樼厱闁斥晛鍟伴埊鏇㈡倵?	// fast 闂傚倸鍊风粈渚€骞栭锕€鐤柛鎰ゴ閺嬫牗绻涢幋鐑囧叕闁肩増瀵ч妵鍕疀閹炬剚浼屽┑鐐烘？閸楁娊寮婚弴銏犻唶婵犻潧娲らˇ鈺呮⒑閸濆嫭鍣洪柟顔煎€垮濠氬焺閸愨晛顎撻梺鍛婄缚閸庨亶銆傞崼鏇熲拺?	//
	// 婵犵數濮烽弫鎼佸磻濞戔懞鍥敇閵忕姷顦悗骞垮劚椤︻垳绮堥崼婢濆綊鎮℃惔锝嗘喖闂?	//   1. 婵犵數濮甸鏍窗濡ゅ啰绱﹂柛褎顨呯壕褰掓煛閸ワ絾鍤嶉柛銉墮閻撴盯鏌涢幇鍏哥敖闁挎稒绮撳铏圭磼濡崵鍙嗘繝鈷€鍐弰妞ゃ垺妫冨畷锟犳倶鐠囪尙绉洪柡浣瑰姍瀹曞ジ鎮㈤幁鎺嗗亾婵犲洦鈷?upstreamModel闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬪鏌涢埄鍐槈缂佲偓閸曨垱鐓ユ繛鎴灻顐ょ棯閹佸仮闁绘搩鍋婂畷鍫曞Ω閿旂偓顓荤紓?GetMappedModel +
	//      normalizeOpenAIModelForUpstream + Codex OAuth normalize闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆″眰鍔戦崺鈧い鎺戝€荤壕濂稿级閸稑濡跨紒鐘崇墱缁?	//      chat-completions / messages 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁割偁鍎辩壕鍧楀级閸偄浜栧ù婊嗩潐缁绘盯骞嬮悙鍨櫘闂佸磭鎳撶粔鐢垫崲濠靛顥堟繛鎴炵懄閹瑩姊烘导娆忕槣闁哥姵鐗犻崺銉﹀緞閹邦剛顔掑銈嗙墬閻喗绔熼弴銏♀拻濞达綀妫勬禍褰掓煛鐏炵瓔妯€妤犵偞鎹囬、鏃堝幢濮楀棙缍楁繝鐢靛█濞佳囶敄閸℃稒鍊垮┑鍌氭啞閻撴洘銇勯鐔风仴闁哄鐩幃浠嬵敍濮橆剚姣愰梺瀹犳椤︾敻鐛Ο铏规殾闁搞儱妫庨崹浠嬪蓟閻旂⒈鏁婇柤鎭掑劚绾炬娊鎮楃憴鍕妞ゃ劌锕ユ穱濠囨倻閽樺）銊╂煏婢诡垰鎳夐弸鏍⒒閸屾瑧鍔嶉柟顔肩埣瀹曟洟鎼归锝呭伎閻庤娲栧ú锔锯偓姘煼閺岋綁寮崶褌妲愰梺缁樻⒒閸樠囨倶瀹曞洠鍋撶憴鍕婵炶尙濞€瀹?	//      缂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏇楀亾妞ゎ厼娲ら埢搴ㄥ箣閺傚じ澹曠紒缁㈠幖閹冲繘鎮甸鍡欑＜閻庯綆浜跺Σ鍛娿亜閹剧偨鍋㈢€规洏鍔戦、娆戜焊閺嵮傚闂備緡鍓欑粔鐢告偂閻樼粯鐓涘璺洪閸旂敻鎮楀顐㈠祮闁哄本鐩俊鑸垫償閳ュ磭鐫勯梻?whitelist 闂傚倸鍊风粈渚€骞夐敍鍕煓闁圭儤顨呴崹鍌涚節闂堟侗鍎愰柛銈呯墦閺岀喓鈧稒顭囩粻銉╂煕閻旈攱鍣洪柕鍥у瀵噣宕惰濮规绱掗幆褍缍栫紒顔界懇瀵?	//   2. action=pass 闂傚倸鍊风粈渚€骞栭锕€鐤い鏍ㄧ啲閼板潡鏌涢敂璇插箹濠殿噮鍓熼弻娑㈠箛閸忓摜鍑归梺鍝ュ枎閹碱偊婀侀梺鎸庣箓濞诧箓宕甸埀顒勬煙?raw "fast" 闂備浇宕甸崰鎰垝鎼淬垺娅犳俊銈呭暞閺嗘粌鈹戦悩鎻掝伀闁告宀搁弻鐔虹磼閵忕姵鐏堢紓浣插亾鐎光偓閸曨剛鍘搁悗骞垮劚缁绘帡鎮惧畷鍥╃＜闁?"priority" 闂傚倸鍊风粈渚€骞夐敓鐘茬闁哄稁鍘介崑锟犳煏婢跺牆鍓?body闂?	//      闂傚倸鍊风粈渚€骞夐敓鐘冲亜妞ゆ帒鍊绘稉宥夋煙鏉堝墽鐣遍柛?native /responses 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁割偁鍎辩壕鍧楀级閸偄浜栧ù婊嗩潐缁绘盯骞嬪▎蹇曚痪闂佺粯甯掗…鐑芥偂椤愶箑鐐婄憸搴ㄥ箲閿濆鐓?"fast" 缂傚倸鍊搁崐鎼佸磹閻戣姤鍊块柨鏇炲€归崑锟犳煏閸繃宸濈紒鐘虫閺屾稓浠﹂幆褎鎯涢梺閫炲苯澧悽顖椻偓鎰佹綎鐟滅増甯掔粻娑欍亜閺傚灝鈷旂€规洖鐖煎娲嚒閵堝懍娌梺鍛婂灥缂嶅﹤鐣烽幋锕€宸濇い鏍殔娴滅偓鎱ㄥ鍡楀妞ゃ儯鍨归埞鎴﹀灳瀹曞洦鎲肩紓浣哥焸娴滃爼濡撮崒姘煎晠闁?
	//      completions 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁割偁鍎辩壕鍧楀级閸偄浜栧ù婊嗩潐缁绘盯骞嬪▎蹇曚痪闂?normalizeResponsesBodyServiceTier 闂傚倷娴囬褍霉閻戣棄绠犻柟鎹愵嚙鐎氬銇勯幒鎴濐仼闁搞劌鍊归幈銊ノ熼幐搴ｃ€愰柣鐔哥懕缁犳捇骞冨鈧幃娆戞崉鏉炵増鐫忔俊?
	//      闂傚倷娴囧畷鐢稿磻閻愮數鐭欓柟瀵稿仧闂勫嫰鏌￠崘銊モ偓鍦偓姘煼閺岋綁寮崹顔藉€梺鍝勬媼閸撴盯鍩€椤掆偓閸樻粓宕戦幘缁樼厱闁规澘鍚€缁ㄨ姤銇勯弮鈧崝娆忣潖缂佹ɑ濯撮柣鐔稿缁佺兘姊洪崫鍕⒈闁告挻宀稿鑼崉娓氼垳鍙嗛梺鍛婂姈閸庢娊寮查悩缁樷拺闂傚牊渚楀Σ鍫曟煕鎼淬垹鈻曠€殿喛鍩栫粋鎺斺偓锝庡亞閸樿棄鈹戦悩璇у伐閻庢凹鍓熻棢閻庯綆鍠楅悡鐔兼煃閳轰礁鏆欓柣蹇擃嚟閳ь剚顔栭崳顕€宕戞繝鍌滄殾濠靛倻顭堝敮闂侀潧绻掓慨铏珶閺囥垺鈷戦柛婵嗗椤忊晝绱掔紒妯虹闁轰緡鍣ｅ畷鍗炩槈濞嗘垵甯?
	if rawTier, ok := reqBody["service_tier"].(string); ok {
		if normTier := normalizedOpenAIServiceTierValue(rawTier); normTier != "" {
			action, errMsg := s.evaluateOpenAIFastPolicy(ctx, account, upstreamModel, normTier)
			switch action {
			case BetaPolicyActionBlock:
				msg := errMsg
				if msg == "" {
					msg = fmt.Sprintf("openai service_tier=%s is not allowed for model %s", normTier, upstreamModel)
				}
				blocked := &OpenAIFastBlockedError{Message: msg}
				writeOpenAIFastPolicyBlockedResponse(c, blocked)
				return nil, blocked
			case BetaPolicyActionFilter:
				delete(reqBody, "service_tier")
				bodyModified = true
				disablePatch()
			default:
				// pass闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟鐑樻尵閳瑰秹鏌ц箛鎾冲辅闁稿鎸搁～婵嬪Ψ閵壯傜磿闂備線鈧偛鑻晶顔剧磼閻樿尙效鐎规洜鎳撶叅妞ゅ繐瀚Σ顒€鈹戦悙鏉戠仸婵ǜ鍔戝畷鈥愁潩椤撴粈绨婚梺鍝勭Р閸斿酣骞婇崨瀛樼厽闁归偊鍓ㄩ煬顒勬煛瀹€瀣М濠碘剝鎮傛俊鐑藉Ψ椤旇崵妫┑鐘殿暯閳ь剙鍟跨痪褔鏌熼鐓庘偓鎼佹偩瀹勯偊娼╁Λ鐗堢箖閺呮繈姊洪棃娑氱畾闁逞屽墯閺嬪ジ宕?"fast"闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸ゆ劖銇勯弽銊х煂缂佲偓婵犲倵鏀介柣妯诲絻閺嗙喖鏌嶇紒妯活棃闁哄苯绉烽¨渚€鏌涢幘璺烘瀻闁伙絿鍏橀幃鈺冩嫚閹绘帒鏁ら梻浣瑰濞叉垿鎳楅崼鏇€?"priority"
				// 闂傚倸鍊风粈渚€骞夐敓鐘冲殞闁绘劦鍓﹀▓浠嬫煙闂傚顦﹂柣銈庡櫍閺岀喖骞戦幇闈涙闂?body闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸庢柨鈹戦崒婊庣劸闁诲繗娅曟穱濠囶敍濠靛棗鎯為梺宕囨嚀缁夌數鎹㈠┑瀣棃婵炴垶鐟ュ▍褔姊洪懡銈呮瀾闁告梹鐗滈幑銏犫槈閵忕姷顔掗梺鎯ф禋閸嬪嫰寮抽崼銉︹拺閺夌偞澹嗛ˇ锔剧磽瀹ュ拑宸ラ柣锝呭槻椤繄鎹勯搹璇″悑婵＄偑鍊栧濠氬疾椤愶箑姹查柣妯肩帛閳锋垿鏌涘┑鍡楊仼妞ゅ繑鎸抽弻娑㈠Ω閵堝懎绁悗瑙勬穿缁查箖濡堕敐澶婄闁宠桨鐒﹂悾閬嶆⒒娴ｄ警鐒鹃柨鏇畵瀹曚即骞樺鍕剁秮瀵挳濮€閿涘嫬骞堥梻浣告贡閸庛倗鎹㈤崒娑氼洸闁绘劕顕粻楣冩煕椤愩倕鏋庨柣蹇婃櫊閺岀喖宕ｆ径瀣攭閻庤娲滈崰鏍€佸▎鎾村殐闁冲搫鍊愰妸鈺傗拻濞达絼璀﹂悞楣冩煥閺囶亞鐣垫い銏＄墵瀵挳濮€閻橀潧濮?
				if normTier != rawTier {
					reqBody["service_tier"] = normTier
					bodyModified = true
					markPatchSet("service_tier", normTier)
				}
			}
		}
	}

	if IsImageGenerationIntentMap(openAIResponsesEndpoint, reqModel, reqBody) && !imageGenerationAllowed {
		setOpsUpstreamError(c, http.StatusForbidden, ImageGenerationPermissionMessage(), "")
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"type":    "permission_error",
				"message": ImageGenerationPermissionMessage(),
			},
		})
		return nil, errors.New("image generation disabled for group")
	}
	imageBillingModel := ""
	imageSizeTier := ""
	imageInputSize := ""
	if IsImageGenerationIntentMap(openAIResponsesEndpoint, reqModel, reqBody) {
		var imageCfgErr error
		imageCfg, imageCfgErr := resolveOpenAIResponsesImageBillingConfigDetailed(reqBody, billingModel)
		if imageCfgErr != nil {
			setOpsUpstreamError(c, http.StatusBadRequest, imageCfgErr.Error(), "")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"type":    "invalid_request_error",
					"message": imageCfgErr.Error(),
					"param":   "size",
				},
			})
			return nil, imageCfgErr
		}
		imageBillingModel = imageCfg.Model
		imageSizeTier = imageCfg.SizeTier
		imageInputSize = imageCfg.InputSize
	}

	// Re-serialize body only if modified
	if bodyModified {
		serializedByPatch := false
		if !patchDisabled && patchHasOp {
			var patchErr error
			if patchDelete {
				body, patchErr = sjson.DeleteBytes(body, patchPath)
			} else {
				body, patchErr = sjson.SetBytes(body, patchPath, patchValue)
			}
			if patchErr == nil {
				serializedByPatch = true
			}
		}
		if !serializedByPatch {
			var marshalErr error
			body, marshalErr = json.Marshal(reqBody)
			if marshalErr != nil {
				return nil, fmt.Errorf("serialize request body: %w", marshalErr)
			}
		}
	}

	// Get access token
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	// Capture upstream request body for ops retry of this attempt.
	setOpsUpstreamRequestBody(c, body)

	// 命中 WS 时仅走 WebSocket Mode；不再自动回退 HTTP。
	if wsDecision.Transport == OpenAIUpstreamTransportResponsesWebsocketV2 {
		wsReqBody := reqBody
		if len(reqBody) > 0 {
			wsReqBody = make(map[string]any, len(reqBody))
			for k, v := range reqBody {
				wsReqBody[k] = v
			}
		}
		_, hasPreviousResponseID := wsReqBody["previous_response_id"]
		logOpenAIWSModeDebug(
			"forward_start account_id=%d account_type=%s model=%s stream=%v has_previous_response_id=%v",
			account.ID,
			account.Type,
			upstreamModel,
			reqStream,
			hasPreviousResponseID,
		)
		maxAttempts := openAIWSReconnectRetryLimit + 1
		wsAttempts := 0
		var wsResult *OpenAIForwardResult
		var wsErr error
		wsLastFailureReason := ""
		wsPrevResponseRecoveryTried := false
		wsInvalidEncryptedContentRecoveryTried := false
		recoverPrevResponseNotFound := func(attempt int) bool {
			if wsPrevResponseRecoveryTried {
				return false
			}
			previousResponseID := openAIWSPayloadString(wsReqBody, "previous_response_id")
			if previousResponseID == "" {
				logOpenAIWSModeInfo(
					"reconnect_prev_response_recovery_skip account_id=%d attempt=%d reason=missing_previous_response_id previous_response_id_present=false",
					account.ID,
					attempt,
				)
				return false
			}
			if HasFunctionCallOutput(wsReqBody) {
				logOpenAIWSModeInfo(
					"reconnect_prev_response_recovery_skip account_id=%d attempt=%d reason=has_function_call_output previous_response_id_present=true",
					account.ID,
					attempt,
				)
				return false
			}
			delete(wsReqBody, "previous_response_id")
			wsPrevResponseRecoveryTried = true
			logOpenAIWSModeInfo(
				"reconnect_prev_response_recovery account_id=%d attempt=%d action=drop_previous_response_id retry=1 previous_response_id=%s previous_response_id_kind=%s",
				account.ID,
				attempt,
				truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen),
				normalizeOpenAIWSLogValue(ClassifyOpenAIPreviousResponseIDKind(previousResponseID)),
			)
			return true
		}
		recoverInvalidEncryptedContent := func(attempt int) bool {
			if wsInvalidEncryptedContentRecoveryTried {
				return false
			}
			removedReasoningItems := trimOpenAIEncryptedReasoningItems(wsReqBody)
			if !removedReasoningItems {
				logOpenAIWSModeInfo(
					"reconnect_invalid_encrypted_content_recovery_skip account_id=%d attempt=%d reason=missing_encrypted_reasoning_items",
					account.ID,
					attempt,
				)
				return false
			}
			previousResponseID := openAIWSPayloadString(wsReqBody, "previous_response_id")
			hasFunctionCallOutput := HasFunctionCallOutput(wsReqBody)
			if previousResponseID != "" && !hasFunctionCallOutput {
				delete(wsReqBody, "previous_response_id")
			}
			wsInvalidEncryptedContentRecoveryTried = true
			logOpenAIWSModeInfo(
				"reconnect_invalid_encrypted_content_recovery account_id=%d attempt=%d action=drop_encrypted_reasoning_items retry=1 previous_response_id_present=%v previous_response_id=%s previous_response_id_kind=%s has_function_call_output=%v dropped_previous_response_id=%v",
				account.ID,
				attempt,
				previousResponseID != "",
				truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen),
				normalizeOpenAIWSLogValue(ClassifyOpenAIPreviousResponseIDKind(previousResponseID)),
				hasFunctionCallOutput,
				previousResponseID != "" && !hasFunctionCallOutput,
			)
			return true
		}
		retryBudget := s.openAIWSRetryTotalBudget()
		retryStartedAt := time.Now()
	wsRetryLoop:
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			wsAttempts = attempt
			wsResult, wsErr = s.forwardOpenAIWSV2(
				ctx,
				c,
				account,
				wsReqBody,
				token,
				wsDecision,
				isCodexCLI,
				reqStream,
				originalModel,
				upstreamModel,
				startTime,
				attempt,
				wsLastFailureReason,
			)
			if wsErr == nil {
				break
			}
			if c != nil && c.Writer != nil && c.Writer.Written() {
				break
			}

			reason, retryable := classifyOpenAIWSReconnectReason(wsErr)
			if reason != "" {
				wsLastFailureReason = reason
			}
			if reason == "previous_response_not_found" && recoverPrevResponseNotFound(attempt) {
				continue
			}
			if reason == "invalid_encrypted_content" && recoverInvalidEncryptedContent(attempt) {
				continue
			}
			if retryable && attempt < maxAttempts {
				backoff := s.openAIWSRetryBackoff(attempt)
				if retryBudget > 0 && time.Since(retryStartedAt)+backoff > retryBudget {
					s.recordOpenAIWSRetryExhausted()
					logOpenAIWSModeInfo(
						"reconnect_budget_exhausted account_id=%d attempts=%d max_retries=%d reason=%s elapsed_ms=%d budget_ms=%d",
						account.ID,
						attempt,
						openAIWSReconnectRetryLimit,
						normalizeOpenAIWSLogValue(reason),
						time.Since(retryStartedAt).Milliseconds(),
						retryBudget.Milliseconds(),
					)
					break
				}
				s.recordOpenAIWSRetryAttempt(backoff)
				logOpenAIWSModeInfo(
					"reconnect_retry account_id=%d retry=%d max_retries=%d reason=%s backoff_ms=%d",
					account.ID,
					attempt,
					openAIWSReconnectRetryLimit,
					normalizeOpenAIWSLogValue(reason),
					backoff.Milliseconds(),
				)
				if backoff > 0 {
					timer := time.NewTimer(backoff)
					select {
					case <-ctx.Done():
						if !timer.Stop() {
							<-timer.C
						}
						wsErr = wrapOpenAIWSFallback("retry_backoff_canceled", ctx.Err())
						break wsRetryLoop
					case <-timer.C:
					}
				}
				continue
			}
			if retryable {
				s.recordOpenAIWSRetryExhausted()
				logOpenAIWSModeInfo(
					"reconnect_exhausted account_id=%d attempts=%d max_retries=%d reason=%s",
					account.ID,
					attempt,
					openAIWSReconnectRetryLimit,
					normalizeOpenAIWSLogValue(reason),
				)
			} else if reason != "" {
				s.recordOpenAIWSNonRetryableFastFallback()
				logOpenAIWSModeInfo(
					"reconnect_stop account_id=%d attempt=%d reason=%s",
					account.ID,
					attempt,
					normalizeOpenAIWSLogValue(reason),
				)
			}
			break
		}
		if wsErr == nil {
			firstTokenMs := int64(0)
			hasFirstTokenMs := wsResult != nil && wsResult.FirstTokenMs != nil
			if hasFirstTokenMs {
				firstTokenMs = int64(*wsResult.FirstTokenMs)
			}
			requestID := ""
			if wsResult != nil {
				requestID = strings.TrimSpace(wsResult.RequestID)
			}
			logOpenAIWSModeDebug(
				"forward_succeeded account_id=%d request_id=%s stream=%v has_first_token_ms=%v first_token_ms=%d ws_attempts=%d",
				account.ID,
				requestID,
				reqStream,
				hasFirstTokenMs,
				firstTokenMs,
				wsAttempts,
			)
			wsResult.UpstreamModel = upstreamModel
			if wsResult.ImageCount > 0 {
				wsResult.ImageSize = imageSizeTier
				wsResult.ImageInputSize = imageInputSize
				wsResult.BillingModel = imageBillingModel
			}
			return wsResult, nil
		}
		s.writeOpenAIWSFallbackErrorResponse(c, account, wsErr)
		return nil, wsErr
	}

	httpInvalidEncryptedContentRetryTried := false
	for {
		// Build upstream request
		upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
		upstreamReq, err := s.buildUpstreamRequest(upstreamCtx, c, account, body, token, reqStream, promptCacheKey, isCodexCLI)
		releaseUpstreamCtx()
		if err != nil {
			return nil, err
		}

		// Get proxy URL
		proxyURL := ""
		if account.ProxyID != nil && account.Proxy != nil {
			proxyURL = account.Proxy.URL()
		}

		// Send request
		upstreamStart := time.Now()
		resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
		SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
		if err != nil {
			// Ensure the client receives an error response (handlers assume Forward writes on non-failover errors).
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamError(c, 0, safeErr, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: 0,
				Kind:               "request_error",
				Message:            safeErr,
			})
			c.JSON(http.StatusBadGateway, gin.H{
				"error": gin.H{
					"type":    "upstream_error",
					"message": "Upstream request failed",
				},
			})
			return nil, fmt.Errorf("upstream request failed: %s", safeErr)
		}

		// Handle error response
		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))

			upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
			upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
			upstreamCode := extractUpstreamErrorCode(respBody)
			if !httpInvalidEncryptedContentRetryTried && resp.StatusCode == http.StatusBadRequest && upstreamCode == "invalid_encrypted_content" {
				if trimOpenAIEncryptedReasoningItems(reqBody) {
					body, err = json.Marshal(reqBody)
					if err != nil {
						return nil, fmt.Errorf("serialize invalid_encrypted_content retry body: %w", err)
					}
					httpInvalidEncryptedContentRetryTried = true
					logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Retrying non-WSv2 request once after invalid_encrypted_content (account: %s)", account.Name)
					continue
				}
				logger.LegacyPrintf("service.openai_gateway", "[OpenAI] Skip non-WSv2 invalid_encrypted_content retry because encrypted reasoning items are missing (account: %s)", account.Name)
			}
			if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
				upstreamDetail := ""
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
					if maxBytes <= 0 {
						maxBytes = 2048
					}
					upstreamDetail = truncateString(string(respBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
					Platform:           account.Platform,
					AccountID:          account.ID,
					AccountName:        account.Name,
					UpstreamStatusCode: resp.StatusCode,
					UpstreamRequestID:  resp.Header.Get("x-request-id"),
					Kind:               "failover",
					Message:            upstreamMsg,
					Detail:             upstreamDetail,
				})

				s.handleFailoverSideEffects(ctx, resp, account)
				return nil, &UpstreamFailoverError{
					StatusCode:             resp.StatusCode,
					ResponseBody:           respBody,
					RetryableOnSameAccount: account.IsPoolMode() && (isPoolModeRetryableStatus(resp.StatusCode) || isOpenAITransientProcessingError(resp.StatusCode, upstreamMsg, respBody)),
				}
			}
			return s.handleErrorResponse(ctx, resp, c, account, body)
		}
		defer func() { _ = resp.Body.Close() }()

		reasoningEffort := extractOpenAIReasoningEffort(reqBody, originalModel)
		serviceTier := extractOpenAIServiceTier(reqBody)
		releaseOpenAIParsedRequestBody(c)

		// Handle normal response
		var usage *OpenAIUsage
		var firstTokenMs *int
		imageCount := 0
		var imageOutputSizes []string
		if reqStream {
			streamResult, err := s.handleStreamingResponse(ctx, resp, c, account, startTime, originalModel, upstreamModel)
			if err != nil {
				return nil, err
			}
			usage = streamResult.usage
			firstTokenMs = streamResult.firstTokenMs
			imageCount = streamResult.imageCount
			imageOutputSizes = streamResult.imageOutputSizes
		} else {
			nonStreamResult, err := s.handleNonStreamingResponse(ctx, resp, c, account, originalModel, upstreamModel)
			if err != nil {
				return nil, err
			}
			usage = nonStreamResult.usage
			imageCount = nonStreamResult.imageCount
			imageOutputSizes = nonStreamResult.imageOutputSizes
		}

		// Extract and save Codex usage snapshot from response headers (for OAuth accounts)
		if account.Type == AccountTypeOAuth {
			if snapshot := ParseCodexRateLimitHeaders(resp.Header); snapshot != nil {
				s.updateCodexUsageSnapshot(ctx, account.ID, snapshot)
			}
		}

		if usage == nil {
			usage = &OpenAIUsage{}
		}

		forwardResult := &OpenAIForwardResult{
			RequestID:       resp.Header.Get("x-request-id"),
			Usage:           *usage,
			Model:           originalModel,
			UpstreamModel:   upstreamModel,
			ServiceTier:     serviceTier,
			ReasoningEffort: reasoningEffort,
			Stream:          reqStream,
			OpenAIWSMode:    false,
			Duration:        time.Since(startTime),
			FirstTokenMs:    firstTokenMs,
		}
		if imageCount > 0 {
			forwardResult.ImageCount = imageCount
			forwardResult.ImageSize = imageSizeTier
			forwardResult.ImageInputSize = imageInputSize
			forwardResult.ImageOutputSizes = imageOutputSizes
			forwardResult.BillingModel = imageBillingModel
		}
		return forwardResult, nil
	}
}

func (s *OpenAIGatewayService) forwardOpenAIPassthrough(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	reqModel string,
	reasoningEffort *string,
	reqStream bool,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	upstreamPassthroughModel := ""
	if isOpenAIResponsesCompactPath(c) {
		compactMappedModel := resolveOpenAICompactForwardModel(account, reqModel)
		if compactMappedModel != "" && compactMappedModel != reqModel {
			nextBody, setErr := sjson.SetBytes(body, "model", compactMappedModel)
			if setErr != nil {
				return nil, fmt.Errorf("set compact passthrough model: %w", setErr)
			}
			body = nextBody
			upstreamPassthroughModel = compactMappedModel
		}
	}

	if account != nil && account.Type == AccountTypeOAuth {
		if rejectReason := detectOpenAIPassthroughInstructionsRejectReason(reqModel, body); rejectReason != "" {
			rejectMsg := "OpenAI codex passthrough requires a non-empty instructions field"
			setOpsUpstreamError(c, http.StatusForbidden, rejectMsg, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: http.StatusForbidden,
				Passthrough:        true,
				Kind:               "request_error",
				Message:            rejectMsg,
				Detail:             rejectReason,
			})
			logOpenAIPassthroughInstructionsRejected(ctx, c, account, reqModel, rejectReason, body)
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"type":    "forbidden_error",
					"message": rejectMsg,
				},
			})
			return nil, fmt.Errorf("openai passthrough rejected before upstream: %s", rejectReason)
		}

		normalizedBody, normalized, err := normalizeOpenAIPassthroughOAuthBody(body, isOpenAIResponsesCompactPath(c))
		if err != nil {
			return nil, err
		}
		if normalized {
			body = normalizedBody
		}
		reqStream = gjson.GetBytes(body, "stream").Bool()
	}

	sanitizedBody, sanitized, err := sanitizeEmptyBase64InputImagesInOpenAIBody(body)
	if err != nil {
		return nil, err
	}
	if sanitized {
		body = sanitizedBody
	}

	// Apply OpenAI fast policy to the passthrough body (filter/block by service_tier).
	// 缂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ礀缁愭鈧箍鍎卞ú銊╁础濮樿埖鍊垫繛鎴烆伆閹寸偛鍨旈柟缁㈠枟閸嬶綁鏌熼鐔风瑨濠碘€茬矙閺?upstream 闂傚倷娴囧畷鐢稿窗閹扮増鍋￠柕澶堝剻濞戞﹩鐓ラ柛顐墰缁嬪繐鈹戞幊閸婃洟骞婅箛娑樼？?model闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟鐑樺灩閺嗗棝鏌熼梻瀵割槮闁绘帒鐏氶妵鍕箳閸℃ぞ澹曠紓鍌欐祰椤曆囶敄閸ヮ灛锝夊箛閺夎法顦繛杈剧悼閹虫挾澹曢姘兼富闁靛牆妫涙晶顒佹叏濡濡跨紒顕嗙秮閹瑥霉鐎ｎ偒娼?body 闂備浇顕у锕傦綖婢舵劖鍋ら柡鍥╁С閻掑﹥绻涢崱妯虹仴闁搞劍绻堥弻宥夊煛娴ｅ憡娈叉繛?compact 闂傚倸鍊风粈渚€骞栭銈傚亾濮樼厧鏋熼柟渚垮姂楠炴﹢顢欓挊澶婂?+
	// OAuth normalize闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬧晠鏌ｉ妸銈夌反y 濠电姷鏁搁崑鐐哄垂閸洖绠归柍鍝勬噹閸屻劑鏌熼鍡忓亾闁?model 闂傚倷娴囬褏鈧稈鏅濈划娆撳箳濡炲皷鍋撻崘顔煎耿婵炴垼椴搁弲鈺呮倵閸忓浜鹃梺鍛婃处閸嬪棝銆侀崨瀛樷拺缂佸娉曠粻浼存煙閾忣偄濮嶉柟顔惧厴楠炲洭顢橀悩娈垮晭闂備浇顫夊畷妯衡枍閺囥垹绠熼柛娑橈功缁♀偓闂侀€炲苯澧撮柛銊╃畺楠炲洦鎷呯憴鍕┾偓鍐⒒娴ｇ懓顕滅紒璇插€婚埀顒佸嚬閸犳氨鍒掗敐鍛傛棃宕ㄩ鎯у箺婵＄偑鍊曠换鎰偓姘卞厴瀹曟洝绠涘☉娆戝幈闂佸疇妗ㄩ懗鍫曟倿閹间焦鐓?slug闂?	// 闂傚倷绀侀幖顐λ囬锕€鐤炬繝濠傜墛閸嬶繝鏌ㄩ弴鐐测偓褰掑磻鐎ｎ喗鈷戞い鎺嗗亾缂佸鎸抽幆灞轿旀担鍏哥盎闂佸搫绉查崝搴ㄥ箠娴ｈ櫣纾奸柕濠忛檮缁舵煡鏌?chat-completions / messages / native /responses 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁割偁鍎辩壕鍧楀级閸偄浜栧ù婊嗩潐缁绘盯骞嬪▎蹇曚痪闂?	// upstreamModel 濠电姷鏁搁崕鎴犲緤閽樺娲晜閻愵剙搴婇梺鍛婂姦娴滄牠宕戦幘璇插瀭妞ゆ劧绲洪幏鍦磽娴ｅ壊鍎忔い锔炬暬瀹曟椽鍩€椤掍降浜滈柟鐑樺灥椤忣亪鏌涢妶鍜冭含鐎殿喖鐖煎畷鐓庘攽閸″繑瀵栫紓鍌欒閸嬫挸顭块懜闈涘闁抽攱鍨堕妵鍕箳閸℃ぞ澹曢梻浣筋嚙缁绘垹鎹㈤崼婵堟殾?whitelist 闂傚倸鍊风粈渚€骞夐敍鍕煓闁圭儤顨呴崹鍌涚節闂堟侗鍎愰柛銈呯墦閺岀喓鈧稒顭囩粻銉╂煕閻旈攱鍣洪柕鍥у瀵噣宕惰濮规绱掗幆褍缍栫紒顔界懇瀵寮撮姀鐘茶€垮┑掳鍊愰崑鎾绘煃瑜滈崜娆撳箺濠婂懏顫?body 濠电姷鏁搁崑鐐哄垂閸洖绠归柍鍝勬噹閸屻劌霉閻樺樊鍎愰柛搴☆樀閺屾稑鈽夊▎鎰▏闁?	// model 闂傚倷娴囬褏鈧稈鏅濈划娆撳箳濡炲皷鍋撻崘顔煎耿婵炴垼椴搁弲鈺呮倵閸忓浜鹃梺鍛婃处閸嬪棝鎯堥崟顖涒拺闁告挻褰冩禍婵囩箾閹绘帞效鐎规洘鍨块獮妯兼嫚閼碱剙濮︽俊鐐€栫敮鎺斺偓姘煎墴閹?reqModel闂?
	policyModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if policyModel == "" {
		policyModel = reqModel
	}
	updatedBody, policyErr := s.applyOpenAIFastPolicyToBody(ctx, account, policyModel, body)
	if policyErr != nil {
		var blocked *OpenAIFastBlockedError
		if errors.As(policyErr, &blocked) {
			writeOpenAIFastPolicyBlockedResponse(c, blocked)
		}
		return nil, policyErr
	}
	body = updatedBody

	apiKey := getAPIKeyFromContext(c)
	if IsImageGenerationIntent(openAIResponsesEndpoint, reqModel, body) && !GroupAllowsImageGeneration(apiKeyGroup(apiKey)) {
		setOpsUpstreamError(c, http.StatusForbidden, ImageGenerationPermissionMessage(), "")
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"type":    "permission_error",
				"message": ImageGenerationPermissionMessage(),
			},
		})
		return nil, errors.New("image generation disabled for group")
	}
	imageBillingModel := ""
	imageSizeTier := ""
	imageInputSize := ""
	if IsImageGenerationIntent(openAIResponsesEndpoint, reqModel, body) {
		var imageCfgErr error
		imageCfg, imageCfgErr := resolveOpenAIResponsesImageBillingConfigDetailedFromBody(body, reqModel)
		if imageCfgErr != nil {
			setOpsUpstreamError(c, http.StatusBadRequest, imageCfgErr.Error(), "")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"type":    "invalid_request_error",
					"message": imageCfgErr.Error(),
					"param":   "size",
				},
			})
			return nil, imageCfgErr
		}
		imageBillingModel = imageCfg.Model
		imageSizeTier = imageCfg.SizeTier
		imageInputSize = imageCfg.InputSize
	}

	logger.LegacyPrintf("service.openai_gateway",
		"[OpenAI passthrough] forwarding request account=%d name=%s type=%s model=%s stream=%v",
		account.ID,
		account.Name,
		account.Type,
		reqModel,
		reqStream,
	)
	if reqStream && c != nil && c.Request != nil {
		if timeoutHeaders := collectOpenAIPassthroughTimeoutHeaders(c.Request.Header); len(timeoutHeaders) > 0 {
			streamWarnLogger := logger.FromContext(ctx).With(
				zap.String("component", "service.openai_gateway"),
				zap.Int64("account_id", account.ID),
				zap.Strings("timeout_headers", timeoutHeaders),
			)
			if s.isOpenAIPassthroughTimeoutHeadersAllowed() {
				streamWarnLogger.Warn("检测到超时相关请求头，当前配置会透传，可能增加断流风险")
			} else {
				streamWarnLogger.Warn("检测到超时相关请求头，将按配置过滤以降低断流风险")
			}
		}
	}

	// Get access token
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	upstreamReq, err := s.buildUpstreamRequestOpenAIPassthrough(upstreamCtx, c, account, body, token)
	releaseUpstreamCtx()
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	if c != nil {
		c.Set("openai_passthrough", true)
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Passthrough:        true,
			Kind:               "request_error",
			Message:            safeErr,
		})
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"type":    "upstream_error",
				"message": "Upstream request failed",
			},
		})
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ幗瀹曟煡鎮楅敐搴″妞ゎ偅娲熼幃瑙勬姜閹殿喚协闂佺粯姊婚崢褔鎮樺畷鍥ｅ亾鐟欏嫭绀€婵炲眰鍔戦妴渚€宕ㄩ钘夊伎闂佸湱鍎ら幐楣冨煀閺囩喆浜滄い鎾跺仧婢э附銇勯姀鈽嗙劷缂佽鲸甯掕灒閻忓繑鐗楅弫鍨節閻㈤潧鈻堟繛浣冲厾娲Χ婢跺鍓ㄥ銈嗘磵閸嬫捇鏌＄仦鍓р槈閾伙綁鏌ｉ幘鍐差劉闁诲酣绠栧铏圭磼濮楀棙鐣芥俊鐐存綑閹芥粎鍒掔€ｎ喖绠虫俊銈勭劍濞呮粓姊洪幖鐐插姌闁告柨鐭傞崺鈧い鎺戝€告禒閬嶆煛鐏炶濡奸柍钘夘槸閳诲酣骞嬮弮鈧敍澶岀磽?429/529 闂傚倷娴囬褏鎹㈤幒妤€纾婚柣鏃傚帶绾剧粯淇婇婵嗩棈濠殿喛娅曢妵鍕箳閹存績鍋撻懡銈咁棜濡わ絽鍟悡娆撴倵濞戞瑯鐒界紒鐘虫尵閹叉悂骞嶉灏栨瀰闂佺粯鎼╅崑濠傜暦閹偊妲奸梺鎼炲€曞ú顓㈠蓟閺囥垹骞㈡俊銈呭暟椤旀帡鎮楃憴鍕妞ゃ劌鎳橀垾锕傚Ω閳轰胶顦ㄥ銈呯箰閸熲晝鈧稈鏅犲?		// 濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸劍閺呮繈鏌曟径鍡樻珕闁稿顦甸弻娑㈠箻閼奸鍞归梺绋款儐閹瑰洭骞冮悜钘夌妞ゆ梹瀚庨妸鈺傗拺闂傚牃鏅濈粔顒佺箾閸滃啰绉柟顕呭櫍閹瑩顢楅崒婊呮婵犳鍠楅敃鈺呭储閽樺鏋嶉柟鍓х帛閻撶喖鏌ｅΟ鍝勭骇缂佷讲鏅犻弻娑㈠Ω瑜庡▍鏇熶繆閸欏濮囬柍瑙勫灴瀹曞爼濡搁妷銉ノら梻鍌欑閹碱偊宕愰幖浣瑰€舵繝闈涚墛閺嗘粓鏌曟径鍡樻珕闁稿鍓濈换娑㈠幢濡吋鍣梺浼欑秬娴滎剟骞夐幖浣哥骇闁割煈鍠掗崑鎾绘偡闁箑娈梺鍛婃处閸嬫帒顭囬埡鍛厱闁归偊鍓氶埢鏇熶繆閺屻儰鎲炬慨?failover 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忓ù鍏兼綑绾惧潡鏌涢幇顔藉▏閹兼惌鐓堥弫瀣煃瑜滈崜鐔兼偘椤曗偓楠炲洭顢橀悢濂夆偓鎾绘⒑閸涘﹦缂氶柛搴ㄤ憾瀵偊濡堕崶鈺冿紲缂傚倷闄嶉崹褰掔嵁閹扮増鐓?SLA闂?
		if shouldFailoverOpenAIPassthroughResponse(resp.StatusCode) {
			return nil, s.handleFailoverErrorResponsePassthrough(ctx, resp, c, account, body)
		}
		return nil, s.handleErrorResponsePassthrough(ctx, resp, c, account, body)
	}

	serviceTier := extractOpenAIServiceTierFromBody(body)

	var usage *OpenAIUsage
	var firstTokenMs *int
	imageCount := 0
	var imageOutputSizes []string
	if reqStream {
		result, err := s.handleStreamingResponsePassthrough(ctx, resp, c, account, startTime, reqModel, upstreamPassthroughModel)
		if err != nil {
			return nil, err
		}
		usage = result.usage
		firstTokenMs = result.firstTokenMs
		imageCount = result.imageCount
		imageOutputSizes = result.imageOutputSizes
	} else {
		result, err := s.handleNonStreamingResponsePassthrough(ctx, resp, c, reqModel, upstreamPassthroughModel)
		if err != nil {
			return nil, err
		}
		usage = result.usage
		imageCount = result.imageCount
		imageOutputSizes = result.imageOutputSizes
	}

	if snapshot := ParseCodexRateLimitHeaders(resp.Header); snapshot != nil {
		s.updateCodexUsageSnapshot(ctx, account.ID, snapshot)
	}

	if usage == nil {
		usage = &OpenAIUsage{}
	}

	forwardResult := &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		Usage:           *usage,
		Model:           reqModel,
		UpstreamModel:   upstreamPassthroughModel,
		ServiceTier:     serviceTier,
		ReasoningEffort: reasoningEffort,
		Stream:          reqStream,
		OpenAIWSMode:    false,
		Duration:        time.Since(startTime),
		FirstTokenMs:    firstTokenMs,
	}
	if imageCount > 0 {
		forwardResult.ImageCount = imageCount
		forwardResult.ImageSize = imageSizeTier
		forwardResult.ImageInputSize = imageInputSize
		forwardResult.ImageOutputSizes = imageOutputSizes
		forwardResult.BillingModel = imageBillingModel
	}
	return forwardResult, nil
}

func logOpenAIPassthroughInstructionsRejected(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	reqModel string,
	rejectReason string,
	body []byte,
) {
	if ctx == nil {
		ctx = context.Background()
	}
	accountID := int64(0)
	accountName := ""
	accountType := ""
	if account != nil {
		accountID = account.ID
		accountName = strings.TrimSpace(account.Name)
		accountType = strings.TrimSpace(string(account.Type))
	}
	fields := []zap.Field{
		zap.String("component", "service.openai_gateway"),
		zap.Int64("account_id", accountID),
		zap.String("account_name", accountName),
		zap.String("account_type", accountType),
		zap.String("request_model", strings.TrimSpace(reqModel)),
		zap.String("reject_reason", strings.TrimSpace(rejectReason)),
	}
	fields = appendCodexCLIOnlyRejectedRequestFields(fields, c, body)
	logger.FromContext(ctx).With(fields...).Warn("OpenAI passthrough 本地拦截：Codex 请求缺少有效 instructions")
	logger.FromContext(ctx).With(fields...).Warn("OpenAI passthrough 鏈湴鎷︽埅锛欳odex 璇锋眰缂哄皯鏈夋晥 instructions")
}

func (s *OpenAIGatewayService) buildUpstreamRequestOpenAIPassthrough(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
) (*http.Request, error) {
	targetURL := openaiPlatformAPIURL
	switch account.Type {
	case AccountTypeOAuth:
		targetURL = chatgptCodexURL
	case AccountTypeAPIKey:
		baseURL := account.GetOpenAIBaseURL()
		if baseURL != "" {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return nil, err
			}
			targetURL = buildOpenAIResponsesURL(validatedURL)
		}
	}
	targetURL = appendOpenAIResponsesRequestPathSuffix(targetURL, openAIResponsesRequestPathSuffix(c))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ幗瀹曟煡鎮楅敐搴″妞ゎ偅娲熼弻娑㈠箻閼奸鍞归梺绋款儐閹瑰洭鎮伴鈧畷褰掝敊閸欍儳妫梻鍌欒兌绾爼寮插鍛煓闁规崘娉涢崹婵嬫煃閸濆嫭濯奸柡浣哥У缁绘繈宕归銏狀潓濠电偛鐗婅ぐ鍐煘閹达富鏁婇柣鎰靛墮濞堝矂姊虹粙鍖℃敾闁诡喖鍊规穱濠囨偨缁嬭法鐫勯梺鍓插亞閸犳劕鈻嶉弽顓熲拺闁告挻褰冩禍婵堢磼鐠囨彃顏€殿喗濞婂顕€宕奸悢鍝勫箞闂傚鍋勫ú銈夘敄閸涱喕绻嗛柟绋挎捣缁犻箖鏌涘☉妯绘悙婵炲眰鍊楅幉鎾晜闁款垰浜炬鐐茬仢閸旀岸鏌熼崘鍙夋崳缂侇喖鐗忛埀顒婄秵閸犳鎮￠妷锔剧闁瑰鍎戞笟娑欎繆閹绘帞鍩ｉ柡?
	allowTimeoutHeaders := s.isOpenAIPassthroughTimeoutHeadersAllowed()
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lower := strings.ToLower(strings.TrimSpace(key))
			if !isOpenAIPassthroughAllowedRequestHeader(lower, allowTimeoutHeaders) {
				continue
			}
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}

	// 闂傚倷娴囧畷鐢稿窗閹邦優娲冀椤剚绋戦埥澶娢熻箛鏇炲妺缂佺粯绻堝畷鎯邦槾妞は佸洦鈷戦梻鍫熷喕缁憋繝鏌涘☉鍗炲绩妞ゆ柨顦靛濠氬磼濞嗘垵濡藉┑鐐插悑閻熲晠骞冨ú顏勎╅柕澶樺枤閸旂兘姊洪崜鎻掍簼缂佹椽绠栧鎼佸籍閸喎鈧敻鏌ㄥ┑鍡涱€楀褜鍨辩换娑㈠醇濠靛牏鍔梺鍝勮閸斿矂鍩ユ径濞㈢喖宕归鍛磾闂傚倷娴囨竟鍫濈暦閻㈢绐楅幖娣妼閻撴繈鏌￠崘銊у閹喖姊洪棃娑辨Ф闁稿骸宕晥闁告瑥顦辩粻楣冩煙鐎电浠﹂悘蹇ｅ弮閺屻劑寮村Ο鍝勫Е闂佽鍠撻崕闈涚暦濠婂嫭濯撮柣鐔煎亰閳ь剚鐩娲濞戞艾顣洪梺缁橆殔缁绘劕宓?
	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Del("x-goog-api-key")
	req.Header.Set("authorization", "Bearer "+token)

	// OAuth 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ幗瀹曟煡鎮楅敐搴″妞ゎ偅娲熼弻娑㈩敃閿濆棛顦ョ紓浣插亾?ChatGPT internal API 闂傚倸鍊风粈渚€骞栭锕€鐤い鏍仜绾惧潡鏌ら崷顓烆暭闁挎稑鍊搁埞鎴︻敊閽樺顑傜紓渚囧櫘閸ㄨ泛鐣峰┑瀣嵆闁绘ê鍟挎惔濠囨⒑闁偛鑻晶鎾煙椤旀儳鍘寸€殿喗娼欓～婵囶潙閺嶃剱婊堟⒒娓氣偓濞佳兠洪敃鈧灋闁告劏鏅欑换?
	if account.Type == AccountTypeOAuth {
		promptCacheKey := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
		req.Host = "chatgpt.com"
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
		apiKeyID := getAPIKeyIDFromContext(c)
		// 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁规壆澧楅崑瀣煕濞戝崬鐏＄€规洖寮堕幈銊ノ熺拠宸殺闂佺顑嗛幐鎼佸煡婢跺ň鏋嶆い鎾楀倿鍋楀銈冨灪椤牓濡甸幇鏉跨闁瑰濯崥鍛攽閿涘嫬浠滄い鎴濇噽閳ь剙鐏氱敮妤冣偓闈涖偢瀹曟帡鎮欑€电骞堟繝纰樻閸ㄦ娊宕㈣瀵憡鎯旈～顑跨盎濡炪倖鍔х徊鑺ユ叏瀹ュ洠鍋撶憴鍕闁告梹顨呴埥澶愬垂椤愵偅效闁硅壈鎻紞鍡椢ｉ悜鑺モ拻濞达絽鎲￠崯鐐烘煕閺冣偓濞叉粎鍒掗弮鍌楀亾濞戞瑯鐒芥い?compact 闂傚倷娴囧畷鐢稿磻閻愮數鐭欓煫鍥ㄧ☉绾惧潡姊洪鈧粔鎾磼閵娾晜鐓涚€广儱楠搁獮妤呮煛閸滀礁澧伴柍褜鍓欓崢婊堝磻閹剧粯鐓曢柡鍥ュ妺缁ㄧ晫绱掓潏銊ュ摵婵﹨娅ｇ划娆戞崉閵娧屽敹缂傚倷绀侀ˇ浼村箰閹剁晫宓侀柛鎰╁壆閺冨牆绀冩い蹇撴４缁卞弶淇婇悙顏勨偓鏍箰閻愵剚鍙忛柣鎴ｆ閺勩儵鏌ｅΟ鑽ゃ偞婵℃彃鐗撻弻鐔虹磼閵忕姷浠╂繛瀛樼矒缁犳牕顫忔繝姘＜婵﹩鍘煎▍銈呪攽椤旂》宸ユい顓犲厴閸ㄩ箖寮介鐐靛€炲銈嗗坊閸嬫挾绱掗銏⑿ч柡宀€鍠栭、娑㈡倷闊厼鏋堥梻浣规偠閸婃垿宕规禒瀣摕婵炴垯鍨圭粻鎶芥煙閹屽殶闁告埊绻濆娲川婵犲倻浜堕梺鍛婃尰缁诲倿顢氶妷鈺佺妞ゆ牗姘ㄩˇ鏉款渻閵堝棙灏甸柛鐘查叄閸┾偓妞ゆ帒鍊告禒閬嶆煛瀹€瀣М濠碘剝鎮傛俊鐑藉Ψ椤旂粯鍋呴梻鍌欑窔閳ь剛鍋涢懟顖涙櫠閻楀牄浜滈柟瀵稿仦缁€瀣煕閳规儳浜?
		clientSessionID := strings.TrimSpace(req.Header.Get("session_id"))
		clientConversationID := strings.TrimSpace(req.Header.Get("conversation_id"))
		if isOpenAIResponsesCompactPath(c) {
			req.Header.Set("accept", "application/json")
			if req.Header.Get("version") == "" {
				req.Header.Set("version", codexCLIVersion)
			}
			if clientSessionID == "" {
				clientSessionID = resolveOpenAICompactSessionID(c)
			}
		} else if req.Header.Get("accept") == "" {
			req.Header.Set("accept", "text/event-stream")
		}
		if req.Header.Get("OpenAI-Beta") == "" {
			req.Header.Set("OpenAI-Beta", "responses=experimental")
		}
		if req.Header.Get("originator") == "" {
			req.Header.Set("originator", "codex_cli_rs")
		}
		// 闂傚倸鍊烽悞锕€顪冮崹顕呯劷闁秆勵殔缁€澶愭煙鐎涙濡囬柡瀣⒒閹叉瓕绠涢弴鐐茬亰闁诲函缍嗛崜姘扁偓姘哺閺屾稑鈻庤箛锝喰﹂梺閫炲苯澧伴柡浣割煼瀵?session 闂傚倸鍊风粈渚€骞栭銈囩煋闁哄鍤氬ú顏勭厸闁告侗鍠栭崜銊╂⒑閻熼偊鍤熷┑顕€顥撶划濠氬籍閸喓鍙嗛梺鍝勬处濮樸劎鎷归悧鍫㈢鐎瑰壊鍠栨晶鎾煛瀹€瀣М闁糕晪绻濆畷鎺戭煥閸愶絽鎮呭┑锛勫亼閸婃牠骞愰崫銉х濠电姴娲ㄥ畵渚€鎮楅敐搴′航婵℃彃缍婇獮鏍箹椤撶偞鐏嗙紓浣介哺閹瑰洤顫忕紒妯肩懝闁搞儜鍌滅泿缂傚倷娴囬褔顢栭崶顬綁骞囬弶鍧楀敹闂佸搫娲ㄩ崑鐔兼晬濞嗘垹纾肩紓浣靛灩瀵喗銇勯妷锔藉暗缂侇喖鐗婂鍕箛椤撴稒瀚介梻浣告啞閸旀垿宕濆畝鈧惀顏囶槼闁靛洤瀚伴獮鎺楀箻闊祴鍋撻幒妤佺厵闁荤喐婢橀顓炩攽閳╁啰鎽冩い锔藉▕閺岋繝宕卞Δ鍐啋闂佸搫鐬奸崰鎾寸珶閺囩姭鍋撻敐搴′函闁告艾鍊荤槐鎾存媴娴犲鎽垫繝銏㈡嚀濡繈鐛箛娑欏亹婵炴潙顑嗛弬鈧梻浣稿閸嬫帡宕戦崨顕呮闁告洦鍨遍埛鎴︽煟閻斿憡绶查柍閿嬫閺屾稓鈧綆鍋勯悘鎾煕閳规儳浜?
		if clientSessionID == "" {
			clientSessionID = promptCacheKey
		}
		if clientConversationID == "" {
			clientConversationID = promptCacheKey
		}
		if clientSessionID != "" {
			req.Header.Set("session_id", isolateOpenAISessionID(apiKeyID, clientSessionID))
		}
		if clientConversationID != "" {
			req.Header.Set("conversation_id", isolateOpenAISessionID(apiKeyID, clientConversationID))
		}
	}

	// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ幗瀹曟煡鎮楅敐搴″妞ゎ偅娲熼幃瑙勬姜閹殿喚协闂佺粯姊婚崢褔鎮樺畷鍥ｅ亾鐟欏嫭绀€婵炲眰鍔戦妴渚€宕ㄩ鍓ь啎闂佺懓顕崑鐐哄煕閹烘鐓曢柣妯哄暱婵秶鈧鍣崑濠囥€佸璺虹劦妞ゆ帒瀚畵渚€鏌熼柇锕€骞楀┑顖涙尦閺岋箑螣娓氼垱鈻撳┑鈩冪叀娴滆泛顫忓ú顏呭仭闂侇叏绠戝▓鍫曟⒑閸濆嫯瀚版い锕傛涧閻ｇ兘妫冨☉杈╁弳闂佸憡渚楁禍婊勭妤ｅ啯鍋℃繛鍡楃箰椤忣偊鏌ｉ幙鍐ㄧ仯缂?User-Agent 濠?ForceCodexCLI 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁瑰濮撮崹婵堚偓鍏夊亾闁逞屽墴閹崇偤鏌嗗鍛唴闂佽姤锚閿?
	customUA := account.GetOpenAIUserAgent()
	if customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}
	// OAuth 闂傚倷娴囬褍霉閻戣棄绠犻柟鎯у殺閸ヮ剦鏁嶉柣鎰皺閻撴垿妫呴銏″缂佸甯￠幃鐐寸節濮樺崬褰勯梺鎼炲劀閸曨剙闂梻浣虹帛閸旀洟鏁冮鍫濊摕闁靛鍎Σ鍫熺箾閸℃ê绗掓い銉︾箓椤啴濡堕崱妯侯槱闂侀潻缍囩徊鍊熺亱?Codex UA 缂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ礀缁愭鈧箍鍎卞ú銊╁础濮樿埖鐓欑紓浣靛灩閺嬫稓绱掗埀顒勫磼濞戞氨鐦堟繝鐢靛Т閸婃悂寮抽悢闈炲酣宕堕妸锔轰虎闂佸搫琚崝宀勫煡婢跺á鐔兼嚃閳轰礁袩濠电姷顣换婵堢不閺嶎厼缁╅梺顒€绉寸粻顖炴煕濞戝崬鏋熼柛搴ｅ枛閺屾洟宕煎┑鍡╁妷闂佽姤鍨甸ˇ鎵崲濞戙垹绠ｉ柣鎰綑绾板秹姊洪幐搴㈢８闁稿海鏁婚獮鍐Ψ閳哄倸娈濆┑鐐茬墛鐎笛勬櫏闂備胶鎳撻崥瀣偩椤忓牆鍨傞悷娆忓椤╅攱鎱ㄥ璇蹭壕闂佸搫鐬奸崰鏍х暦濞嗘挸围濠㈣泛顑愰埀顒€绉瑰铏瑰寲閺囩喐鐝旈梺鐟板暱闁帮綁鏁愰悙鍙傛棃宕ㄩ鐙呯床婵犵數濮撮敃銈夊疮閾忣偅鍙忛柣銏犳啞閳?
	if account.Type == AccountTypeOAuth && !openai.IsCodexCLIRequest(req.Header.Get("user-agent")) {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}

	// 浏览器型 UA 兜底：仅 OAuth（ChatGPT 内部接口）账号生效，若最终 user-agent 仍为浏览器
	// （Chrome/Firefox/Safari/Edge 等），替换为后台配置的 Codex UA，避免 Cloudflare 触发 JS 质询。
	s.overrideBrowserUserAgent(ctx, account, req)

	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}

	return req, nil
}

func shouldFailoverOpenAIPassthroughResponse(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, 529:
		return true
	default:
		return false
	}
}

func (s *OpenAIGatewayService) handleFailoverErrorResponsePassthrough(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	requestBody []byte,
) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	logOpenAIInstructionsRequiredDebug(ctx, c, account, resp.StatusCode, upstreamMsg, requestBody, body)
	if s.rateLimitService != nil {
		_ = s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:             account.Platform,
		AccountID:            account.ID,
		AccountName:          account.Name,
		UpstreamStatusCode:   resp.StatusCode,
		UpstreamRequestID:    resp.Header.Get("x-request-id"),
		Passthrough:          true,
		Kind:                 "failover",
		Message:              upstreamMsg,
		Detail:               upstreamDetail,
		UpstreamResponseBody: upstreamDetail,
	})
	return &UpstreamFailoverError{
		StatusCode:      resp.StatusCode,
		ResponseBody:    body,
		ResponseHeaders: resp.Header.Clone(),
	}
}

func (s *OpenAIGatewayService) handleErrorResponsePassthrough(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	requestBody []byte,
) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	logOpenAIInstructionsRequiredDebug(ctx, c, account, resp.StatusCode, upstreamMsg, requestBody, body)
	if s.rateLimitService != nil {
		// Passthrough mode preserves the raw upstream error response, but runtime
		// account state still needs to be updated so sticky routing can stop
		// reusing a freshly rate-limited account.
		_ = s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:             account.Platform,
		AccountID:            account.ID,
		AccountName:          account.Name,
		UpstreamStatusCode:   resp.StatusCode,
		UpstreamRequestID:    resp.Header.Get("x-request-id"),
		Passthrough:          true,
		Kind:                 "http_error",
		Message:              upstreamMsg,
		Detail:               upstreamDetail,
		UpstreamResponseBody: upstreamDetail,
	})

	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)

	if upstreamMsg == "" {
		return fmt.Errorf("upstream error: %d", resp.StatusCode)
	}
	return fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
}

func isOpenAIPassthroughAllowedRequestHeader(lowerKey string, allowTimeoutHeaders bool) bool {
	if lowerKey == "" {
		return false
	}
	if isOpenAIPassthroughTimeoutHeader(lowerKey) {
		return allowTimeoutHeaders
	}
	return openaiPassthroughAllowedHeaders[lowerKey]
}

func isOpenAIPassthroughTimeoutHeader(lowerKey string) bool {
	switch lowerKey {
	case "x-stainless-timeout", "x-stainless-read-timeout", "x-stainless-connect-timeout", "x-request-timeout", "request-timeout", "grpc-timeout":
		return true
	default:
		return false
	}
}

func (s *OpenAIGatewayService) isOpenAIPassthroughTimeoutHeadersAllowed() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIPassthroughAllowTimeoutHeaders
}

func collectOpenAIPassthroughTimeoutHeaders(h http.Header) []string {
	if h == nil {
		return nil
	}
	var matched []string
	for key, values := range h {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if isOpenAIPassthroughTimeoutHeader(lowerKey) {
			entry := lowerKey
			if len(values) > 0 {
				entry = fmt.Sprintf("%s=%s", lowerKey, strings.Join(values, "|"))
			}
			matched = append(matched, entry)
		}
	}
	sort.Strings(matched)
	return matched
}

type openaiStreamingResultPassthrough struct {
	usage            *OpenAIUsage
	firstTokenMs     *int
	imageCount       int
	imageOutputSizes []string
}

type openaiNonStreamingResultPassthrough struct {
	*OpenAIUsage
	usage            *OpenAIUsage
	imageCount       int
	imageOutputSizes []string
}

func openAIStreamClientOutputStarted(c *gin.Context, localStarted bool) bool {
	if localStarted {
		return true
	}
	return c != nil && c.Writer != nil && c.Writer.Written()
}

func openAIStreamEventIsPreamble(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "response.created", "response.in_progress":
		return true
	default:
		return false
	}
}

func openAIStreamDataStartsClientOutput(data, eventType string) bool {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return false
	}
	if strings.TrimSpace(eventType) == "response.failed" {
		return false
	}
	return !openAIStreamEventIsPreamble(eventType)
}

func openAIStreamFailedEventShouldFailover(payload []byte, message string) bool {
	if isOpenAITransientProcessingError(http.StatusBadRequest, message, payload) {
		return true
	}
	code := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "response.error.code").String()))
	if code == "" {
		code = strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "error.code").String()))
	}
	errType := strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "response.error.type").String()))
	if errType == "" {
		errType = strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, "error.type").String()))
	}
	combined := strings.ToLower(strings.TrimSpace(message + " " + code + " " + errType))
	if combined == "" {
		return true
	}
	nonRetryableMarkers := []string{
		"invalid_request",
		"content_policy",
		"policy",
		"safety",
		"high-risk cyber",
		"not allowed",
		"violat",
	}
	for _, marker := range nonRetryableMarkers {
		if strings.Contains(combined, marker) {
			return false
		}
	}
	return true
}

func (s *OpenAIGatewayService) newOpenAIStreamFailoverError(
	c *gin.Context,
	account *Account,
	passthrough bool,
	upstreamRequestID string,
	payload []byte,
	message string,
) *UpstreamFailoverError {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "OpenAI stream disconnected before completion"
	}
	detail := ""
	if len(payload) > 0 && s != nil && s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		detail = truncateString(string(payload), maxBytes)
	}
	if c != nil {
		setOpsUpstreamError(c, http.StatusBadGateway, message, detail)
		event := OpsUpstreamErrorEvent{
			Platform:           PlatformOpenAI,
			UpstreamStatusCode: http.StatusBadGateway,
			UpstreamRequestID:  strings.TrimSpace(upstreamRequestID),
			Passthrough:        passthrough,
			Kind:               "failover",
			Message:            message,
			Detail:             detail,
		}
		if account != nil {
			event.Platform = account.Platform
			event.AccountID = account.ID
			event.AccountName = account.Name
		}
		appendOpsUpstreamError(c, event)
	}
	body, _ := json.Marshal(gin.H{
		"error": gin.H{
			"type":    "upstream_error",
			"message": message,
		},
	})
	return &UpstreamFailoverError{
		StatusCode:   http.StatusBadGateway,
		ResponseBody: body,
	}
}

func (s *OpenAIGatewayService) handleStreamingResponsePassthrough(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	startTime time.Time,
	originalModel string,
	mappedModel string,
) (*openaiStreamingResultPassthrough, error) {
	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	if v := resp.Header.Get("x-request-id"); v != "" {
		c.Header("x-request-id", v)
	}

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	usage := &OpenAIUsage{}
	imageCounter := newOpenAIImageOutputCounter()
	var firstTokenMs *int
	clientDisconnected := false
	sawDone := false
	sawTerminalEvent := false
	sawFailedEvent := false
	failedMessage := ""
	clientOutputStarted := false
	upstreamRequestID := strings.TrimSpace(resp.Header.Get("x-request-id"))
	pendingLines := make([]string, 0, 8)
	writePendingLines := func() bool {
		for _, pending := range pendingLines {
			if _, err := fmt.Fprintln(w, pending); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "[OpenAI passthrough] Client disconnected during streaming, continue draining upstream for usage: account=%d", account.ID)
				return false
			}
		}
		pendingLines = pendingLines[:0]
		return true
	}

	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
	defer putSSEScannerBuf64K(scanBuf)

	needModelReplace := strings.TrimSpace(originalModel) != "" && strings.TrimSpace(mappedModel) != "" && strings.TrimSpace(originalModel) != strings.TrimSpace(mappedModel)
	resultWithUsage := func() *openaiStreamingResultPassthrough {
		return &openaiStreamingResultPassthrough{
			usage:            usage,
			firstTokenMs:     firstTokenMs,
			imageCount:       imageCounter.Count(),
			imageOutputSizes: imageCounter.Sizes(),
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		lineStartsClientOutput := false
		forceFlushFailedEvent := false
		if data, ok := extractOpenAISSEDataLine(line); ok {
			dataBytes := []byte(data)
			trimmedData := strings.TrimSpace(data)
			if needModelReplace && strings.Contains(data, mappedModel) {
				line = s.replaceModelInSSELine(line, mappedModel, originalModel)
				if replacedData, replaced := extractOpenAISSEDataLine(line); replaced {
					dataBytes = []byte(replacedData)
					trimmedData = strings.TrimSpace(replacedData)
				}
			}
			eventType := strings.TrimSpace(gjson.Get(trimmedData, "type").String())
			if eventType == "response.failed" {
				failedMessage = extractOpenAISSEErrorMessage(dataBytes)
				if !openAIStreamClientOutputStarted(c, clientOutputStarted) && openAIStreamFailedEventShouldFailover(dataBytes, failedMessage) {
					return resultWithUsage(),
						s.newOpenAIStreamFailoverError(c, account, true, upstreamRequestID, dataBytes, failedMessage)
				}
				forceFlushFailedEvent = true
				sawFailedEvent = true
			}
			if trimmedData == "[DONE]" {
				sawDone = true
			}
			if openAIStreamEventIsTerminal(trimmedData) {
				sawTerminalEvent = true
			}
			imageCounter.AddSSEData(dataBytes)
			lineStartsClientOutput = forceFlushFailedEvent || openAIStreamDataStartsClientOutput(trimmedData, eventType)
			if firstTokenMs == nil && lineStartsClientOutput && trimmedData != "[DONE]" {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			s.parseSSEUsageBytes(dataBytes, usage)
		}

		if !clientDisconnected {
			if !clientOutputStarted && !lineStartsClientOutput {
				pendingLines = append(pendingLines, line)
				continue
			}
			if !clientOutputStarted && len(pendingLines) > 0 {
				if !writePendingLines() {
					continue
				}
			}
			if _, err := fmt.Fprintln(w, line); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "[OpenAI passthrough] Client disconnected during streaming, continue draining upstream for usage: account=%d", account.ID)
			} else {
				clientOutputStarted = true
				flusher.Flush()
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if sawTerminalEvent && !sawFailedEvent {
			return resultWithUsage(), nil
		}
		if sawFailedEvent {
			return resultWithUsage(), fmt.Errorf("upstream response failed: %s", failedMessage)
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return resultWithUsage(), fmt.Errorf("stream usage incomplete: %w", err)
		}
		if errors.Is(err, bufio.ErrTooLong) {
			logger.LegacyPrintf("service.openai_gateway", "[OpenAI passthrough] SSE line too long: account=%d max_size=%d error=%v", account.ID, maxLineSize, err)
			return resultWithUsage(), err
		}
		if !openAIStreamClientOutputStarted(c, clientOutputStarted) {
			msg := "OpenAI stream disconnected before completion"
			if errText := strings.TrimSpace(err.Error()); errText != "" {
				msg += ": " + errText
			}
			return resultWithUsage(),
				s.newOpenAIStreamFailoverError(c, account, true, upstreamRequestID, nil, msg)
		}
		if clientDisconnected {
			return resultWithUsage(), fmt.Errorf("stream usage incomplete after disconnect: %w", err)
		}
		logger.LegacyPrintf("service.openai_gateway",
			"[OpenAI passthrough] 婵犵數濮烽弫鎼佸磻閻旂儤宕叉繝闈涙灩瑜版帒鐐婃い鎺戭槹閺咁亝绻涢弶鎴濇倯婵炲吋鐟╅幆灞轿旈崨顔惧幐閻庤鎼╅崰鏍箠閹扮増鏅繝闈涱儐閻撶喖鏌曡箛濠冩珔闁诲骏绠撻弻锟犲川椤旀儳寮ㄩ梺纭呮珪閻熲晠鐛€ｎ喗鏅濋柍褜鍓欒灋? account=%d request_id=%s err=%v",
			account.ID,
			upstreamRequestID,
			err,
		)
		return resultWithUsage(), fmt.Errorf("stream read error: %w", err)
	}
	if sawFailedEvent {
		return resultWithUsage(), fmt.Errorf("upstream response failed: %s", failedMessage)
	}
	if !clientDisconnected && !sawDone && !sawTerminalEvent && ctx.Err() == nil {
		logger.FromContext(ctx).With(
			zap.String("component", "service.openai_gateway"),
			zap.Int64("account_id", account.ID),
			zap.String("upstream_request_id", upstreamRequestID),
		).Info("上游流在未收到 [DONE] 时结束，疑似断流")
		logger.FromContext(ctx).With(
			zap.String("component", "service.openai_gateway"),
			zap.Int64("account_id", account.ID),
			zap.String("upstream_request_id", upstreamRequestID),
		).Info("涓婃父娴佸湪鏈敹鍒?[DONE] 鏃剁粨鏉燂紝鐤戜技鏂祦")
		if !openAIStreamClientOutputStarted(c, clientOutputStarted) {
			return resultWithUsage(),
				s.newOpenAIStreamFailoverError(c, account, true, upstreamRequestID, nil, "OpenAI stream ended before a terminal event")
		}
		return resultWithUsage(), errors.New("stream usage incomplete: missing terminal event")
	}

	return resultWithUsage(), nil
}

func (s *OpenAIGatewayService) handleNonStreamingResponsePassthrough(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
) (*openaiNonStreamingResultPassthrough, error) {
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}

	// Detect SSE responses from upstream and convert to JSON.
	// Some upstreams (e.g. other sub2api instances) may return SSE even when
	// stream=false was requested. Without this conversion the client would
	// receive raw SSE text or a terminal event with empty output.
	if isEventStreamResponse(resp.Header) || looksLikeOpenAINonStreamingSSEBody(body) {
		return s.handlePassthroughSSEToJSON(resp, c, body, originalModel, mappedModel)
	}

	usage := &OpenAIUsage{}
	usageParsed := false
	if len(body) > 0 {
		if parsedUsage, ok := extractOpenAIUsageFromJSONBytes(body); ok {
			*usage = parsedUsage
			usageParsed = true
		}
	}
	if !usageParsed {
		// 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁瑰濮撮崹婵堚偓鍏夊亾闁逞屽墴閹崇偤鏌嗗鍛唴闂佸吋浜介崕鎻掆枍閺嵮€鏀芥い鏃€鏋绘笟娑㈡煕閹惧鈽夌悮娆撴煥閺囩偛鈧綊宕戦敐澶嬬厱闁靛绲芥俊鐣岀棯閹规劕鍚圭紒?SSE 闂傚倸鍊风粈渚€骞栭锕€纾圭紒瀣紩濞差亝鏅濋柍褜鍓熼弫鍐閵堝棗浜遍梺鍓插亐閹冲洭寮搁弽褜娓婚柕鍫濇鐏忋劑鎮楀▓鍨⒋妤犵偞甯掕灃闁逞屽墴瀹?usage
		usage = s.parseSSEUsageFromBody(string(body))
	}

	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	if originalModel != "" && mappedModel != "" && originalModel != mappedModel {
		body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
	}
	c.Data(resp.StatusCode, contentType, body)
	return &openaiNonStreamingResultPassthrough{
		OpenAIUsage:      usage,
		usage:            usage,
		imageCount:       countOpenAIResponseImageOutputsFromJSONBytes(body),
		imageOutputSizes: collectOpenAIResponseImageOutputSizesFromJSONBytes(body),
	}, nil
}

// handlePassthroughSSEToJSON converts an SSE response body into a JSON
// response for the passthrough path. It mirrors handleSSEToJSON while
// preserving passthrough payloads, except compact-only model remapping may
// rewrite model fields back to the original requested model.
func (s *OpenAIGatewayService) handlePassthroughSSEToJSON(resp *http.Response, c *gin.Context, body []byte, originalModel string, mappedModel string) (*openaiNonStreamingResultPassthrough, error) {
	bodyText := string(body)
	finalResponse, ok := extractCodexFinalResponse(bodyText)

	usage := &OpenAIUsage{}
	if ok {
		if parsedUsage, parsed := extractOpenAIUsageFromJSONBytes(finalResponse); parsed {
			*usage = parsedUsage
		}
		// When the terminal event has an empty output array, reconstruct
		// output from accumulated delta events so the client gets full content.
		if len(gjson.GetBytes(finalResponse, "output").Array()) == 0 {
			if outputJSON, reconstructed := reconstructResponseOutputFromSSE(bodyText); reconstructed {
				if patched, err := sjson.SetRawBytes(finalResponse, "output", outputJSON); err == nil {
					finalResponse = patched
				}
			}
		}
		body = finalResponse
		if originalModel != "" && mappedModel != "" && originalModel != mappedModel {
			body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
		}
		// Correct tool calls in final response
		body = s.correctToolCallsInResponseBody(body)
	} else {
		terminalType, terminalPayload, terminalOK := extractOpenAISSETerminalEvent(bodyText)
		if terminalOK && terminalType == "response.failed" {
			msg := extractOpenAISSEErrorMessage(terminalPayload)
			if msg == "" {
				msg = "Upstream compact response failed"
			}
			return nil, s.writeOpenAINonStreamingProtocolError(resp, c, msg)
		}
		usage = s.parseSSEUsageFromBody(bodyText)
		if originalModel != "" && mappedModel != "" && originalModel != mappedModel {
			bodyText = s.replaceModelInSSEBody(bodyText, mappedModel, originalModel)
		}
		body = []byte(bodyText)
	}

	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)

	contentType := "application/json; charset=utf-8"
	if !ok {
		contentType = resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "text/event-stream"
		}
	}
	c.Data(resp.StatusCode, contentType, body)

	return &openaiNonStreamingResultPassthrough{
		OpenAIUsage:      usage,
		usage:            usage,
		imageCount:       countOpenAIImageOutputsFromSSEBody(bodyText),
		imageOutputSizes: collectOpenAIImageOutputSizesFromSSEBody(bodyText),
	}, nil
}

func writeOpenAIPassthroughResponseHeaders(dst http.Header, src http.Header, filter *responseheaders.CompiledHeaderFilter) {
	if dst == nil || src == nil {
		return
	}
	if filter != nil {
		responseheaders.WriteFilteredHeaders(dst, src, filter)
	} else {
		// 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁瑰濮撮崹婵堚偓鍏夊亾闁逞屽墴閹崇偤鏌嗗鍛唴闂佸吋浜介崕鎻掆枍閺嵮€鏀芥い鏃€鏋绘笟娑㈡煕閹炬潙鍝虹€殿喖寮剁缓浠嬪川婵炵偓瀚奸梻浣告贡缁垳鏁悙鐢典笉闁挎繂娲ㄧ壕鑲╃磽娴ｅ顏堝传濞差亝鐓欐い鏃囨閸斻倝鏌嶇拠鏌ュ弰妤犵偛娲、妯衡攽閸繄顐奸梻鍌氬€烽懗鍓佹兜閸洖鐤鹃柣鎰ゴ閺嬪秹鏌ㄥ┑鍡╂Ф闁逞屽厸缁舵艾鐣烽妸褉鍋撳☉娅亪宕?content-type
		if v := strings.TrimSpace(src.Get("Content-Type")); v != "" {
			dst.Set("Content-Type", v)
		}
	}
	// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ幗瀹曟煡鎮楅敐搴″妞ゎ偅娲熼幃瑙勬姜閹殿喚协闂佺粯姊婚崢褔鎮樺畷鍥ｅ亾鐟欏嫭绀€婵炲眰鍔戦妴渚€宕ㄧ€涙ê鈧灚绻涢崼婵堜虎闁哄鐩弻鐔哄枈閸楃偘鍠婇悗瑙勬礀閹碱偉鐏冮梺鍛婁緱閸橀箖宕Δ鍛拺缁绢厼鎷嬪鎰版煕閺冩挾鐣电€?x-codex-* 闂傚倸鍊风粈渚€骞夐敍鍕床闁稿本绮庨惌鎾绘倵閸偆鎽冨┑顔藉▕閺屾稑鈻庤箛锝喰︾紓浣稿閸嬨倕顕ｉ崼鏇為唶婵犻潧妫岄幐鍐磽娴ｆ彃浜鹃梺鍛婃处閸ㄩ亶鎮￠悩铏弿闁荤喓澧楅幖鎰版煕閺冣偓閻熝呮閹烘嚦鏃堝焵椤掑嫬纾归柟闂寸杩濋梺鍦劋閸╁牆顭囬幍顔瑰亾閸忓浜鹃梺閫炲苯澧扮紒顔碱煼瀵濡烽敂鎯у箞闂備線娼ч敍蹇涘礃閵娿儺妫庣紓鍌氬€峰ù鍥ㄣ仈缁嬭法鏆嗛柛娑橈梗缁?	// 婵犵數濮烽弫鎼佸磻濞戔懞鍥敇閵忕姷顦悗骞垮劚椤︻垳绮堥崼婢濆綊鎮℃惔锝嗘喖闂佸搫鎷嬮崜姘跺箞閵娿儙鐔煎锤濡も偓閹界敻姊虹化鏇熸珔闁稿﹤娼￠獮?http.Response.Header 闂?key 濠电姷鏁搁崑鐐哄垂閸洖绠伴柟缁㈠枛绾惧鏌熼崜褏甯涢柣鎾跺█閺屾盯骞橀弶鎴犵シ闂佸憡锚閹芥粎妲愰幒鎾寸秶闁靛鍎哄Λ鍡涙⒑?canonicalize闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟鐑樻⒒閻瑩鏌熼柇锕€鏋涚紒韬插€濋弻锝夊箛闂堟稑顫梺宕囩帛濡啫顕ｉ崼鏇熷€烽柡澶嬪焾濡棝姊洪崫銉ユ瀻闁挎洦浜璇测槈濠婂懐鏉搁梺鍝勬川閸嬫稒绂掑Ο渚富闁靛牆妫欓埛鎰版煕閺冣偓濞茬喎锕㈡笟鈧弻锝嗘償椤栨粎校闂佸憡蓱閸庡啿宓?闂傚倸鍊烽懗鍫曞储瑜旈妴鍐╂償閵忋埄娲稿┑鐘诧工閹虫劗澹曟禒瀣厵闁硅鍔﹂崵娆徝归懖鈺佇ラ柍褜鍓欑粻宥夊磿闁秴绠犻柟閭﹀枓閸嬫捇鎮烽柇锔解枅闂?	// 闂傚倷绀侀幖顐λ囬锕€鐤炬繝濠傜墛閸嬶繝鏌嶉崫鍕櫣闂傚偆鍨堕弻锝夊箣閿濆憛鎾绘煕?EqualFold 闂傚倸鍊烽懗鍫曗€﹂崼銉晞闁糕剝绋掗崕妤併亜閺傚灝鈷旈柛妤佸哺閺屸剝寰勭€ｎ亞浜堕梺鍝勵儏闁帮綁寮诲澶娢ㄧ憸宥嗙閸濆娊鐟邦煥閹邦喚袦闂佽鍠楅…鍫㈢不濞戙垹绠婚柛鎾茬贰閸炲爼姊绘担鍛婃喐闁哥姵鍔欓獮鎰版嚒閵堝洨鐓撴繝銏ｆ硾椤剟宕ョ€ｎ喗鐓曢柍鈺佸彁閹达附鍋熼梺顒€绉甸埛鎴︽偡濞嗗繐顏╂い銉ヮ儔閺屾盯鎮╅搹顐㈢闂佸磭绮幑鍥嵁鐎ｎ喗鏅濋柍褜鍓熼幏鎴︽偄閸忚偐鍘繝銏ｅ煐缁嬫捇鎮炬潏銊ｄ簻闁靛濡囩粻鐐烘煛?
	getCaseInsensitiveValues := func(h http.Header, want string) []string {
		if h == nil {
			return nil
		}
		for k, vals := range h {
			if strings.EqualFold(k, want) {
				return vals
			}
		}
		return nil
	}

	for _, rawKey := range []string{
		"x-codex-primary-used-percent",
		"x-codex-primary-reset-after-seconds",
		"x-codex-primary-window-minutes",
		"x-codex-secondary-used-percent",
		"x-codex-secondary-reset-after-seconds",
		"x-codex-secondary-window-minutes",
		"x-codex-primary-over-secondary-limit-percent",
	} {
		vals := getCaseInsensitiveValues(src, rawKey)
		if len(vals) == 0 {
			continue
		}
		key := http.CanonicalHeaderKey(rawKey)
		dst.Del(key)
		for _, v := range vals {
			dst.Add(key, v)
		}
	}
}

func (s *OpenAIGatewayService) buildUpstreamRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token string, isStream bool, promptCacheKey string, isCodexCLI bool) (*http.Request, error) {
	// Determine target URL based on account type
	var targetURL string
	switch account.Type {
	case AccountTypeOAuth:
		if !account.IsOpenAI() {
			return nil, fmt.Errorf("unsupported OAuth account platform for OpenAI-compatible gateway: %s", account.Platform)
		}
		// OAuth accounts use ChatGPT internal API
		targetURL = chatgptCodexURL
	case AccountTypeAPIKey:
		// API Key accounts use Platform API or custom base URL
		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			targetURL = openaiPlatformAPIURL
		} else {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return nil, err
			}
			targetURL = buildOpenAIResponsesURL(validatedURL)
		}
	default:
		targetURL = openaiPlatformAPIURL
	}
	targetURL = appendOpenAIResponsesRequestPathSuffix(targetURL, openAIResponsesRequestPathSuffix(c))

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set authentication header
	req.Header.Set("authorization", "Bearer "+token)

	// Set headers specific to OAuth accounts (ChatGPT internal API)
	if account.Type == AccountTypeOAuth {
		// Required: set Host for ChatGPT API (must use req.Host, not Header.Set)
		req.Host = "chatgpt.com"
		// Required: set chatgpt-account-id header
		chatgptAccountID := account.GetChatGPTAccountID()
		if chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	// Whitelist passthrough headers
	for key, values := range c.Request.Header {
		lowerKey := strings.ToLower(key)
		if openaiAllowedHeaders[lowerKey] {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}
	if account.Type == AccountTypeOAuth {
		compatMessagesBridge := isOpenAICompatMessagesBridgeContext(c) || isOpenAICompatMessagesBridgeBody(body)
		clientConversationID := strings.TrimSpace(req.Header.Get("conversation_id"))
		// 婵犵數濮烽弫鎼佸磻閻愬搫绠伴柟闂寸缁犵姵淇婇婵勨偓鈧柡瀣Ч楠炴牗娼忛崜褍鐭濋梺绋款儐閹瑰洭鎮伴鈧畷褰掝敊閸欍儳妫梻鍌欒兌绾爼寮插鍛煓闁规崘娉涢崹婵嬫煃閸濆嫭濯奸柡浣哥Ч閺岋綁骞囬妸锔界彅婵炲瓨绮撶粻鏍偂椤愶箑鐐婄憸搴ㄥ箲閿濆鐓熼柟閭﹀墾闊剟鏌?session 濠电姷鏁告慨浼村垂濞差亜纾垮┑鍌滎焾缁犲灚銇勮箛鎾搭棡妞ゎ偅娲樼换婵嬫濞戝崬鍓伴柣鐔哥懕缁犳捇鐛弽銊︾秶闁告挆鍕还闂備浇妗ㄧ欢姘辩不閺嶎厼钃熼柨婵嗩檧缂嶆牗銇勯幒鍡椾壕婵犳鍠栭崐鍦崲濞戙垹閱囧ù锝呮啞閸ｄ粙鏌ｉ鐑嗘█闁诡喖缍婂畷鎯邦槻缂佺嫏鍥ㄧ厱濠电姴瀚弳顒勬煛瀹€瀣М濠碘剝鎮傛俊鐑藉Ψ椤旂粯鍋呴梻鍌欑窔閳ь剛鍋涢懟顖涙櫠椤曗偓閺屾洟宕奸悢绋库偓鎰偓瑙勬礃婵炲﹪鐛弽銊﹀闁告稑顭Λ鐔兼⒒娴ｇ儤鍤€闁搞垺鐓￠幆宀勫醇閵夈儳鏌堥柟鑹版彧缁茶法澹曟禒瀣厱闁归偊鍘奸崝銈嗙箾閸涱喚鎳呯紒杈ㄥ笚濞煎繘濡歌閻ゅ嫬鈹戦纭峰伐闁靛牏顭堥悾鐑芥晲閸℃绐炲┑鈽嗗灠閹碱偊藟濠靛鈷戦柤濮愬€曟牎婵炲瓨绮堢划娆忕暦濠靛洦鍎熼柕濞垮劜閻庮剟姊洪棃娑氬妞わ富鍨崇划濠氭晲婢跺鍙勫┑顔斤供閸撴瑩鍩€椤掑啫鍚圭紒顕嗙到铻栭柛娑卞枛娴狀垶姊洪幖鐐插姌闁告柨娴风划璇参熷Ч鍥︾盎濡炪倖鍔﹂崑鍕閻楀牜娈介柣鎰彧閼板潡鏌熼鐣屾噰妞ゃ垺姊归幏鍛存⒐閹邦剦妫?		clientConversationID := strings.TrimSpace(req.Header.Get("conversation_id"))
		req.Header.Del("conversation_id")
		req.Header.Del("session_id")

		if compatMessagesBridge {
			req.Header.Del("OpenAI-Beta")
			req.Header.Del("originator")
		} else {
			req.Header.Set("OpenAI-Beta", "responses=experimental")
			req.Header.Set("originator", resolveOpenAIUpstreamOriginator(c, isCodexCLI))
		}
		apiKeyID := getAPIKeyIDFromContext(c)
		if isOpenAIResponsesCompactPath(c) {
			req.Header.Set("accept", "application/json")
			if req.Header.Get("version") == "" {
				req.Header.Set("version", codexCLIVersion)
			}
			compactSession := resolveOpenAICompactSessionID(c)
			req.Header.Set("session_id", isolateOpenAISessionID(apiKeyID, compactSession))
		} else {
			req.Header.Set("accept", "text/event-stream")
		}
		if promptCacheKey != "" {
			isolated := isolateOpenAISessionID(apiKeyID, promptCacheKey)
			req.Header.Set("session_id", isolated)
			if !compatMessagesBridge || clientConversationID != "" {
				req.Header.Set("conversation_id", isolated)
			}
		}
	}

	// Apply custom User-Agent if configured
	customUA := account.GetOpenAIUserAgent()
	if customUA != "" {
		req.Header.Set("user-agent", customUA)
	}

	// 闂傚倸鍊风粈渚€宕ョ€ｎ喗鍎戠憸鐗堝笒绾惧潡寮堕崼姘珕妞ゎ偅娲熼弻鐔告綇閸撗呮殸闁?ForceCodexCLI闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸ゆ劖銇勯弽顐粶闁搞劌鍊归妵鍕冀閵娿儱姣堥梺鎼炲€栧ú鐔煎蓟濞戞ǚ鏀介柛鈩冾殢娴犲ジ姊虹紒妯诲鞍闁搞劌娼￠獮鍐ㄎ熺捄銊ф澑闂佸湱鍋撻幆灞轿涘☉姘辩＝濞达綀顕栧▓鏃堟煕婵犲啯鍊愬┑?User-Agent 濠电姷鏁搁崑娑㈩敋椤撱垹鐓曢柛顐ゅ枔椤╂煡鏌ㄥ┑鍡礂濠㈣泛顑囩弧鈧梺鍛婂姦娴滄牠寮?Codex CLI闂?	// 闂傚倸鍊烽悞锕€顪冮崹顕呯劷闁秆勵殔缁€澶愭倵閿濆骸澧插┑顔挎珪閵囧嫰骞掗幋婵冨亾閼姐倕顥氬Δ锝呭暞閻撴瑩鎮楀☉娆樼劷缂佺姵鎸鹃幉鎼佸箥椤斿皷鏋呴梺鍝勭焿缁绘繂鐣锋總鍓叉晝闁跨喓濮存禒鎰版⒒娓氣偓閳ь剛鍋涢懟顖涙櫠椤斿墽纾奸悹鍥ф▕閸庢垶淇?闂傚倸鍊峰ù鍥Υ閳ь剟鏌涚€ｎ偅宕岄柡宀嬬秮閹垽宕ㄦ繝鍕殥闂?User-Agent 闂傚倸鍊风粈渚€骞栭锕€鐤柟鍓佺摂閺佸﹪鏌熼柇锕€鏋熸い顐ｆ礃缁绘繈妫冨☉娆忕獩濡炪倐鏅犻ˉ鎾诲焵椤掆偓缁犲秹宕曢柆宥嗗亱闁糕剝绋戦崒銊╂煙缂併垹鏋熼柣鎾存礃缁绘盯骞嬮悙鍨櫑闂佽绻愰敃顏堝蓟?Codex 濠电姷鏁搁崑鐐哄箰婵犳碍鍎嶆繝濠傜墕楠炪垺绻涢崱妯哄Е婵炵鍎靛缁樻媴閸涘﹤鏆堥柣銏╁灙閺呯姴鐣烽幋鐐电瘈闁搞儜鍛Е婵＄偑鍊栫敮濠囨倿閿曞偊缍栨俊銈呭暟绾剧厧螞閻楀牏绠撳ù婊勬緲閳?
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}

	// 浏览器型 UA 兜底：仅 OAuth（ChatGPT 内部接口）账号生效，若最终 user-agent 仍为浏览器
	// （Chrome/Firefox/Safari/Edge 等），替换为后台配置的 Codex UA，避免 Cloudflare 触发 JS 质询。
	s.overrideBrowserUserAgent(ctx, account, req)

	// Ensure required headers exist
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}

	return req, nil
}

// overrideBrowserUserAgent 检查请求的最终 user-agent，若为浏览器 UA 则替换为后台配置的 Codex UA。
// 用于规避 Cloudflare 对浏览器型 UA 在 ChatGPT 内部接口上的访问质询。
// 影响范围严格限定：仅 OAuth（Codex/ChatGPT 内部接口）账号生效；API Key 等其他账号原样透传。
// 仅在识别为浏览器（Mozilla/...）时改写，其他 CLI/工具 UA 不动。
func (s *OpenAIGatewayService) overrideBrowserUserAgent(ctx context.Context, account *Account, req *http.Request) {
	if req == nil || account == nil {
		return
	}
	if account.Type != AccountTypeOAuth {
		return
	}
	currentUA := req.Header.Get("user-agent")
	if !openai.IsBrowserUserAgent(currentUA) {
		return
	}
	codexUA := DefaultOpenAICodexUserAgent
	if s != nil && s.settingService != nil {
		if v := strings.TrimSpace(s.settingService.GetOpenAICodexUserAgent(ctx)); v != "" {
			codexUA = v
		}
	}
	req.Header.Set("user-agent", codexUA)
}

func (s *OpenAIGatewayService) handleErrorResponse(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	requestBody []byte,
) (*OpenAIForwardResult, error) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	logOpenAIInstructionsRequiredDebug(ctx, c, account, resp.StatusCode, upstreamMsg, requestBody, body)

	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		logger.LegacyPrintf("service.openai_gateway",
			"OpenAI upstream error %d (account=%d platform=%s type=%s): %s",
			resp.StatusCode,
			account.ID,
			account.Platform,
			account.Type,
			truncateForLog(body, s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes),
		)
	}

	if status, errType, errMsg, matched := applyErrorPassthroughRule(
		c,
		PlatformOpenAI,
		resp.StatusCode,
		body,
		http.StatusBadGateway,
		"upstream_error",
		"Upstream request failed",
	); matched {
		c.JSON(status, gin.H{
			"error": gin.H{
				"type":    errType,
				"message": errMsg,
			},
		})
		if upstreamMsg == "" {
			upstreamMsg = errMsg
		}
		if upstreamMsg == "" {
			return nil, fmt.Errorf("upstream error: %d (passthrough rule matched)", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream error: %d (passthrough rule matched) message=%s", resp.StatusCode, upstreamMsg)
	}

	// Check custom error codes
	if !account.ShouldHandleErrorCode(resp.StatusCode) {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  resp.Header.Get("x-request-id"),
			Kind:               "http_error",
			Message:            upstreamMsg,
			Detail:             upstreamDetail,
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"type":    "upstream_error",
				"message": "Upstream gateway error",
			},
		})
		if upstreamMsg == "" {
			return nil, fmt.Errorf("upstream error: %d (not in custom error codes)", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream error: %d (not in custom error codes) message=%s", resp.StatusCode, upstreamMsg)
	}

	// Handle upstream error (mark account status)
	shouldDisable := false
	if s.rateLimitService != nil {
		shouldDisable = s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
	}
	kind := "http_error"
	if shouldDisable {
		kind = "failover"
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           account.Platform,
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  resp.Header.Get("x-request-id"),
		Kind:               kind,
		Message:            upstreamMsg,
		Detail:             upstreamDetail,
	})
	if shouldDisable {
		return nil, &UpstreamFailoverError{
			StatusCode:             resp.StatusCode,
			ResponseBody:           body,
			RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
		}
	}

	// Return appropriate error response
	var errType, errMsg string
	var statusCode int

	switch resp.StatusCode {
	case 401:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream authentication failed, please contact administrator"
	case 402:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream payment required: insufficient balance or billing issue"
	case 403:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream access forbidden, please contact administrator"
	case 429:
		statusCode = http.StatusTooManyRequests
		errType = "rate_limit_error"
		errMsg = "Upstream rate limit exceeded, please retry later"
	default:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream request failed"
	}

	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": errMsg,
		},
	})

	if upstreamMsg == "" {
		return nil, fmt.Errorf("upstream error: %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
}

// compatErrorWriter is the signature for format-specific error writers used by
// the compat paths (Chat Completions and Anthropic Messages).
type compatErrorWriter func(c *gin.Context, statusCode int, errType, message string)

// handleCompatErrorResponse is the shared non-failover error handler for the
// Chat Completions and Anthropic Messages compat paths. It mirrors the logic of
// handleErrorResponse (passthrough rules, ShouldHandleErrorCode, rate-limit
// tracking, secondary failover) but delegates the final error write to the
// format-specific writer function.
func (s *OpenAIGatewayService) handleCompatErrorResponse(
	resp *http.Response,
	c *gin.Context,
	account *Account,
	writeError compatErrorWriter,
) (*OpenAIForwardResult, error) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	if upstreamMsg == "" {
		upstreamMsg = fmt.Sprintf("Upstream error: %d", resp.StatusCode)
	}
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)

	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)

	// Apply error passthrough rules
	if status, errType, errMsg, matched := applyErrorPassthroughRule(
		c, account.Platform, resp.StatusCode, body,
		http.StatusBadGateway, "api_error", "Upstream request failed",
	); matched {
		writeError(c, status, errType, errMsg)
		if upstreamMsg == "" {
			upstreamMsg = errMsg
		}
		if upstreamMsg == "" {
			return nil, fmt.Errorf("upstream error: %d (passthrough rule matched)", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream error: %d (passthrough rule matched) message=%s", resp.StatusCode, upstreamMsg)
	}

	// Check custom error codes; if the account does not handle this status,
	// return a generic error without exposing upstream details.
	if !account.ShouldHandleErrorCode(resp.StatusCode) {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  resp.Header.Get("x-request-id"),
			Kind:               "http_error",
			Message:            upstreamMsg,
			Detail:             upstreamDetail,
		})
		writeError(c, http.StatusInternalServerError, "api_error", "Upstream gateway error")
		if upstreamMsg == "" {
			return nil, fmt.Errorf("upstream error: %d (not in custom error codes)", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream error: %d (not in custom error codes) message=%s", resp.StatusCode, upstreamMsg)
	}

	// Track rate limits and decide whether to trigger secondary failover.
	shouldDisable := false
	if s.rateLimitService != nil {
		shouldDisable = s.rateLimitService.HandleUpstreamError(
			c.Request.Context(), account, resp.StatusCode, resp.Header, body,
		)
	}
	kind := "http_error"
	if shouldDisable {
		kind = "failover"
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           account.Platform,
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  resp.Header.Get("x-request-id"),
		Kind:               kind,
		Message:            upstreamMsg,
		Detail:             upstreamDetail,
	})
	if shouldDisable {
		return nil, &UpstreamFailoverError{
			StatusCode:             resp.StatusCode,
			ResponseBody:           body,
			RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
		}
	}

	// Map status code to error type and write response
	errType := "api_error"
	switch {
	case resp.StatusCode == 400:
		errType = "invalid_request_error"
	case resp.StatusCode == 404:
		errType = "not_found_error"
	case resp.StatusCode == 429:
		errType = "rate_limit_error"
	case resp.StatusCode >= 500:
		errType = "api_error"
	}

	writeError(c, resp.StatusCode, errType, upstreamMsg)
	return nil, fmt.Errorf("upstream error: %d %s", resp.StatusCode, upstreamMsg)
}

// openaiStreamingResult streaming response result
type openaiStreamingResult struct {
	usage            *OpenAIUsage
	firstTokenMs     *int
	imageCount       int
	imageOutputSizes []string
}

type openaiNonStreamingResult struct {
	*OpenAIUsage
	usage            *OpenAIUsage
	imageCount       int
	imageOutputSizes []string
}

func (s *OpenAIGatewayService) handleStreamingResponse(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, startTime time.Time, originalModel, mappedModel string) (*openaiStreamingResult, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}

	// Set SSE response headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Pass through other headers
	if v := resp.Header.Get("x-request-id"); v != "" {
		c.Header("x-request-id", v)
	}

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	bufferedWriter := bufio.NewWriterSize(w, 4*1024)
	flushBuffered := func() error {
		if err := bufferedWriter.Flush(); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	usage := &OpenAIUsage{}
	imageCounter := newOpenAIImageOutputCounter()
	var firstTokenMs *int
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)

	streamInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	// 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忛柟缁㈠枛缁犵娀鏌熼柇锕€鍔掓繛宸簻缁€鍐┿亜閺冨洤浜规い鏂匡躬閹嘲顭ㄩ崘顏嗕紘缂備緡鍣崣鍐ㄧ暦閹烘埈娼╅柕澶堝灩娴滈箖鏌熼悜姗嗘當鐎瑰憡绻傞埞鎴︽偐閹绘帗娈插銈嗘煥缁夌懓顫忛搹瑙勫厹闁告粈绀佸▓鎰版⒑闂堚晝绉剁紒鎻掆偓鐔轰罕闂備胶顫嬮崟鍨暦闂佹娊鏀卞ú鐔煎蓟閿熺姴绀冮柕濠忕畱缁愭盯鎮楀☉娆戠疄婵﹥妞藉畷顐﹀礋椤愶絾顔戞俊鐐€栧褰掑礉濡も偓鍗遍柟閭﹀厴閺€浠嬫煕閵夈垺鏉归柡瀣墵閹鐛崹顔煎闂佺粯顨嗙划宀勨€﹂崶鈺傚磯閻炴稈鍓濋～宥夋⒑閸︻厼浜鹃柛鎾村哺瀹曟洟骞囬悧鍫㈠幗闂佸搫璇為崨顓犲幗闁诲氦顫夊ú姗€鏁嬮梺瀹狀嚙濮橈妇绮诲☉銏犵闁惧浚鍋夌欢銏ゆ⒒閸屾艾鈧悂宕愰幖浣哥柈闁瑰墎鐡旈弫瀣喐閺冨牆违濞撴埃鍋撻柡浣瑰姍瀹曠厧鈽夊Δ浣烘綎闂傚倷绀侀幉锟犮€冮崼銉ョ；闁绘劕鎼粻?
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}

	keepaliveInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamKeepaliveInterval > 0 {
		keepaliveInterval = time.Duration(s.cfg.Gateway.StreamKeepaliveInterval) * time.Second
	}
	var keepaliveTicker *time.Ticker
	if keepaliveInterval > 0 {
		keepaliveTicker = time.NewTicker(keepaliveInterval)
		defer keepaliveTicker.Stop()
	}
	var keepaliveCh <-chan time.Time
	if keepaliveTicker != nil {
		keepaliveCh = keepaliveTicker.C
	}
	// Track downstream writes separately from upstream reads: pre-output failover
	// can buffer response.created / response.in_progress, so keepalive must be
	// based on downstream idle time.
	lastDownstreamWriteAt := time.Now()

	// 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忛柟缁㈠枛缁犵娀骞栧ǎ顒€鈧垶绂嶈ぐ鎺撶厱鐎光偓閳ь剟宕戦悙鐑樺€块悹鍥梿瑜版帗鏅插鑸瞪戦ˉ鏍ь渻閵堝懐绠哄鏉戞憸閹广垹鈽夐姀鐘茶€垮銈嗘尵婵兘鎯侀崼銉︹拺閺夌偞澹嗛ˇ锕傛煥閺囶亞鐣垫鐐茬箻閹晛煤缂佹ɑ娅栨繝鐢靛仜濡瑩宕归悷鎵虫灁闁靛牆妫涚粻楣冩倵閻㈡鐒鹃棄瀣渻閵堝棙鈷愰柛鏃€顨呭嵄闁归偊鍏橀弨浠嬫倵閿濆骸浜為柛鏂跨埣濮婃椽宕滈懠顒€甯ラ梺鍝ュУ閻楁粓寮鈧畷妯侯啅椤旀儳鏁搁梻浣哄帶閹芥粓寮幖浣碘偓鍌炲矗婢跺牅绨婚梺褰掑亰閸忔瑩宕戦姀鈶╁亾鐟欏嫭绀冮柨姘舵煃鐠囧弶鍞夌紒鐘崇洴楠炴﹢鎼归銉у彂闂傚倷娴囬褏鑺遍懖鈺佺筏缂備焦顭囬悵鍫曟煕椤愮姴鍔氶柛姘贡閹叉悂寮捄銊︽闂傚嫬娲ㄧ划璇测槈濮楀棙鍍甸梺缁樻尭鐎涒晠寮抽妷锔剧瘈闁汇垽娼ч崜宕囩磽瀹ヤ礁浜炬繝鐢靛仦閹哥锕㈤柆宥呯鐟滅増甯楅弲鎼佹煟濡搫妫?	// 婵犵數濮烽弫鎼佸磻濞戔懞鍥敇閵忕姷顦悗骞垮劚椤︻垳绮堥崼婢濆綊鎮℃惔锝嗘喖闂佸搫鎷嬮崜姘跺箞閵娿儮鏀介柛鈩冾殘閸欏儱nAI `/v1/responses` streaming 濠电姷鏁搁崑娑㈡偤閵娧冨灊鐎光偓閸曞灚鏅為梺鍛婃处閸嬧偓闁哄閰ｉ弻鏇＄疀閵壯勫仹闂佹眹鍊愰崑鎾绘⒒娴ｅ憡鍟炵紒瀣浮閹箖顢旈崼婵堝姦濡炪倖宸婚崑鎾剁磼缂佹ê绗ч柛鎺撳浮椤㈡﹢濮€閳╁啯鐝栭梻渚€鈧偛鑻晶瀛橆殽?OpenAI Responses schema闂?	// 闂傚倸鍊风粈渚€骞夐敓鐘冲亜妞ゆ帒鍊绘稉宥夋煙鏉堝墽鐣遍柛銊ュ€归妵鍕冀閵娧佲偓鎺楁煃缂佹ɑ顥堥柟顔筋殜閺佹劖鎯旈垾鑼晼闂?SDK闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬪鏌涘☉鍗炵仭妞ゆ劒绮欓弻鏇熷緞濞戞氨绐楃紓?OpenCode闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆¤娲、娆撴倷椤掑缍楀┑鐐存綑閸氬顭囧▎鎾村€峰┑鐘叉处閻撶喐淇婇婵囩《缂併劏娉曠槐鎺楀焵椤掑倵鍋撻敐搴″缁炬儳鍚嬮妵鍕箛閳轰礁濮堕梺鎼炲€栭悷鈺呭蓟閿濆棙鍎熼柨婵嗘噹楠炲鎮楀▓鍨灆缂侇喗鐟╁顐﹀箻缂佹ê浜归梻鍌氱墛缁嬫捇鎮￠幋婵冩斀闁绘﹩鍠栭悘顏堟煕鎼淬倕浜归柛鎺撳浮瀹曟粏顦抽柛鐘叉閺屻劑寮撮悙娴嬪亾瑜版帒鍚归柍褜鍓熼弻锝堢疀閺囩偘娌梺绋块閻ゅ洭銆傞崸妤佲拻闁稿本鐟ч崝宥夋倵缁楁稑娲﹂崑锟犳煥濠靛棭妲搁柣?
	errorEventSent := false
	clientDisconnected := false // 闂傚倷娴囬褎顨ラ崫銉т笉鐎广儱顦崹鍌涚箾瀹割喕绨婚柡鍕╁劜缁绘盯骞嬮悙瀵告闂佸憡顨嗙喊宥夊Φ閸曨垰鍐€闁靛ě鈧慨鍥⒑缂佹ê绗ч柡鍜佸亰閸┾偓妞ゆ帊绶￠崯蹇涙煕閻樺啿娴€规洘鍨块獮妯肩磼濡厧骞堟繝娈垮枟閿曗晠宕滃璺虹闁挎繂顦伴悡娆撴煙缂併垹娅樻俊顐ｅ灩缁?drain 濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸劍閺呮繈鏌曟径鍡樻珕闁稿顦甸弻銈夊传閵夛附姣勫銈傛櫇閸忔﹢寮婚弴鐔风窞婵炴垯鍨洪宥夋⒑閸濆嫭锛嶅ù婊庝簻椤?usage
	sawTerminalEvent := false
	sawFailedEvent := false
	failedMessage := ""
	clientOutputStarted := false
	upstreamRequestID := strings.TrimSpace(resp.Header.Get("x-request-id"))
	var streamFailoverErr error
	sendErrorEvent := func(reason string) {
		if errorEventSent || clientDisconnected {
			return
		}
		errorEventSent = true
		payload := `{"type":"error","sequence_number":0,"error":{"type":"upstream_error","message":` + strconv.Quote(reason) + `,"code":` + strconv.Quote(reason) + `}}`
		if err := flushBuffered(); err != nil {
			clientDisconnected = true
			return
		}
		if _, err := bufferedWriter.WriteString("data: " + payload + "\n\n"); err != nil {
			clientDisconnected = true
			return
		}
		if err := flushBuffered(); err != nil {
			clientDisconnected = true
			return
		}
		clientOutputStarted = true
		lastDownstreamWriteAt = time.Now()
	}

	needModelReplace := originalModel != mappedModel
	resultWithUsage := func() *openaiStreamingResult {
		return &openaiStreamingResult{
			usage:            usage,
			firstTokenMs:     firstTokenMs,
			imageCount:       imageCounter.Count(),
			imageOutputSizes: imageCounter.Sizes(),
		}
	}
	finalizeStream := func() (*openaiStreamingResult, error) {
		if !sawTerminalEvent {
			if !openAIStreamClientOutputStarted(c, clientOutputStarted) {
				return resultWithUsage(), s.newOpenAIStreamFailoverError(
					c,
					account,
					false,
					upstreamRequestID,
					nil,
					"OpenAI stream ended before a terminal event",
				)
			}
			return resultWithUsage(), fmt.Errorf("stream usage incomplete: missing terminal event")
		}
		if sawFailedEvent {
			return resultWithUsage(), fmt.Errorf("upstream response failed: %s", failedMessage)
		}
		if !clientDisconnected {
			hadBufferedData := bufferedWriter.Buffered() > 0
			if err := flushBuffered(); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during final flush, returning collected usage")
			} else if hadBufferedData {
				clientOutputStarted = true
				lastDownstreamWriteAt = time.Now()
			}
		}
		return resultWithUsage(), nil
	}
	handleScanErr := func(scanErr error) (*openaiStreamingResult, error, bool) {
		if scanErr == nil {
			return nil, nil, false
		}
		if sawTerminalEvent && !sawFailedEvent {
			logger.LegacyPrintf("service.openai_gateway", "Upstream scan ended after terminal event: %v", scanErr)
			return resultWithUsage(), nil, true
		}
		if sawFailedEvent {
			return resultWithUsage(), fmt.Errorf("upstream response failed: %s", failedMessage), true
		}
		// 闂傚倷娴囬褎顨ラ崫銉т笉鐎广儱顦崹鍌涚箾瀹割喕绨婚柡鍕╁劜缁绘盯骞嬮悙瀵告闂佸憡顨嗙喊宥夊Φ閸曨垰鍐€闁靛ě鈧慨鍥⒑缂佹ê绗ч柡鍜佸亰閸┾偓妞ゆ帊绶￠崯蹇涙煕閻樺啿娴€?闂傚倸鍊风粈渚€骞夐敓鐘冲仭闁挎洖鍊归崑瀣繆閵堝懎鏆熼柣顓熸尭椤啰鈧綆浜滈銏ゆ倵濮橆厼鍝烘慨濠冩そ瀹曞綊顢氶崨顓炲婵犵數濮崑鎾绘煙缂併垹鏋熼柣鎾寸洴閺屾盯骞囬棃娑樺闂佸摜鍠庡ú銊ф閹烘挻缍囬柕濞у懐鏆紓鍌欑椤戝懘藝閹殿喗宕叉繝闈涱儏缁€鍐┿亜韫囨挸顏柛姗嗗墴濮婃椽鎳￠妶鍛呫垺绻涘ù瀣珚妞ゃ垺淇洪ˇ鍫曟煏閸ャ劌濮嶇€规洘鍎奸¨渚€鏌涢妶鍡╂疁闁哄苯绉烽¨渚€鏌涢幘瀵告噰妞ゃ垺宀告俊鐑藉煛娴ｅ搫濮︽俊鐐€栫敮濠勭矆娓氣偓椤㈡碍娼忛妸锕€寮垮┑鈽嗗灥濞咃綁鏁嶅鍥╃＜闁绘ê纾崣鈧梺?context canceled闂?		// /v1/responses 闂?SSE 濠电姷鏁搁崑娑㈡偤閵娧冨灊鐎光偓閸曞灚鏅為梺鍛婃处閸嬧偓闁哄閰ｉ弻鏇＄疀閵壯勫仹闂佹眹鍊愰崑鎾绘⒒娴ｅ憡鍟炵紒瀣浮閹箖顢旈崼婵堝姦濡炪倖宸婚崑鎾剁磼缂佹ê绗ч柛鎺撳浮椤㈡﹢濮€閳╁啯鐝栭梻渚€鈧偛鑻晶瀛橆殽?OpenAI 闂傚倸鍊风粈渚€骞夐敓鐘偓鍐川椤栨稑搴婂┑掳鍊曢幏瀣极婵犲洦鐓曟繛鎴烆焽閹界娀鏌￠崪浣稿缂佺粯鐩獮瀣倷閻㈢數鍘掔紓鍌欒兌婵绱炴繝鍌ゆ綎婵炲樊浜滅粻鐢告煙閻戞ê鐒炬繛鍫幘缁辨挻鎷呯拠鈩冪暥濡炪倧瀵岄崹鍫曞春閵忋倕绫嶉柛顐ｆ儕閳哄懏鐓忓璺虹墕閸旀碍鎱ㄧ憴鍕弨婵﹥妞介幃鈩冩償椤旂晫鏋冨┑鐐茬摠缁秶鍒掗幘璇茬畺濡わ絽鍟崐閿嬨亜閹烘垵鈧敻寮?error event闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸庢鏌涚仦鍓р槈妞ゆ洟浜堕弻宥夊传閸曨剙娅ｇ紓浣插亾闁稿本澹曢崑鎾荤嵁閸喖濮庨柣搴㈠嚬閸ｏ綁骞冮悜钘夌閻庨潧鎽滈?SDK 闂傚倷娴囧畷鐢稿窗閹扮増鍋￠弶鍫氭櫅缁躲倕螖閿濆懎鏆為柛濠勬暬閺屻倝骞侀幒鎴濆缂備礁澧庨崑銈咁嚕閸洖閱囨い鎰垫線閸戜粙鎮跺顓犵疄婵?
		if errors.Is(scanErr, context.Canceled) || errors.Is(scanErr, context.DeadlineExceeded) {
			return resultWithUsage(), fmt.Errorf("stream usage incomplete: %w", scanErr), true
		}
		if errors.Is(scanErr, bufio.ErrTooLong) {
			logger.LegacyPrintf("service.openai_gateway", "SSE line too long: account=%d max_size=%d error=%v", account.ID, maxLineSize, scanErr)
			sendErrorEvent("response_too_large")
			return resultWithUsage(), scanErr, true
		}
		if !openAIStreamClientOutputStarted(c, clientOutputStarted) {
			msg := "OpenAI stream disconnected before completion"
			if errText := strings.TrimSpace(scanErr.Error()); errText != "" {
				msg += ": " + errText
			}
			return resultWithUsage(), s.newOpenAIStreamFailoverError(c, account, false, upstreamRequestID, nil, msg), true
		}
		// 闂傚倷娴囬褎顨ラ崫銉т笉鐎广儱顦崹鍌涚箾瀹割喕绨婚柡鍕╁劜缁绘盯骞嬮悙瀵告闂佸憡顨嗙喊宥夊Φ閸曨垰鍐€闁靛鍔岄ˉ婵嬫⒑閹肩偛鈧牜鏁敓鐘茶摕闁跨喓濮寸壕濂告煏閸繃鍣圭紒鐘叉贡缁辨挻鎷呮禒瀣懙闂佹悶鍔岄悘婵嬵敋閿濆鏁婇柣鎰靛墮椤庢盯姊洪棃鈺佺槣闁告﹢绠栬棟妞ゆ梻鏅粻楣冩煙鐎电浠﹂悘蹇ｅ弮閺屻劑寮村Ο鍝勫Е闂佽鍠撻崕闈涚暦婵傜鍗抽柣鎰問濡茶鲸绻濋悽闈涗粶婵☆垰锕ョ粋宥呪攽鐎ｎ亞鏌ч梺瀹犳〃閹儴銇愰幒鎾充汗闂佸憡鐟ラˇ顖氣枍閿濆棛绡€闂傚牊绋戦埀顒€缍婇幃褔鎮╅懠顒佹濡炪倖鍔戦崺鍕触鐎ｎ喗鐓曢柡鍥ュ妼楠炴ê顫㈤崶褉鏀介幒鎶藉磹濡や焦鍙忛柟缁㈠枟閸庢顭跨捄鐑樻拱妞ゎ偅娲樼换婵嬫濞戞瑥绐涢梺宕囩帛濮婂鍩€椤掆偓缁犲秹宕曢柆宥呯疇閹兼番鍔婇埀顒€鍟埥澶愬閿涘嫬骞堟繝鐢靛█濞佳兾涘☉妯绘瘎濠碉紕鍋戦崐鏍箰閹间礁绠烘繝濠傜墕缁狀垳鎲搁悧鍫濈瑲闁稿骸鐭傞弻娑樷攽閸曨偄濮庨梺鍏兼緲濞硷繝寮婚埄鍐ㄧ窞闁糕剝蓱閻濇洟姊洪崫鍕潶闁告柨鐬奸崣鍛存⒑閸濆嫯鐧侀悘鐐跺Г濞呭﹪姊绘担鍛婂暈闁荤喆鍎佃棟濞村吋娼欏Ч鍙夈亜韫囨挾澧涢柍?usage
		if clientDisconnected {
			return resultWithUsage(), fmt.Errorf("stream usage incomplete after disconnect: %w", scanErr), true
		}
		sendErrorEvent("stream_read_error")
		return resultWithUsage(), fmt.Errorf("stream read error: %w", scanErr), true
	}
	processSSELine := func(line string, queueDrained bool) {
		if streamFailoverErr != nil {
			return
		}
		// Extract data from SSE line (supports both "data: " and "data:" formats)
		if data, ok := extractOpenAISSEDataLine(line); ok {

			// Replace model in response if needed.
			// Fast path: most events do not contain model field values.
			if needModelReplace && mappedModel != "" && strings.Contains(data, mappedModel) {
				line = s.replaceModelInSSELine(line, mappedModel, originalModel)
			}

			dataBytes := []byte(data)
			if openAIStreamEventIsTerminal(data) {
				sawTerminalEvent = true
			}
			eventType := strings.TrimSpace(gjson.GetBytes(dataBytes, "type").String())
			forceFlushFailedEvent := false
			if eventType == "response.failed" {
				failedMessage = extractOpenAISSEErrorMessage(dataBytes)
				if !openAIStreamClientOutputStarted(c, clientOutputStarted) && openAIStreamFailedEventShouldFailover(dataBytes, failedMessage) {
					sawFailedEvent = true
					streamFailoverErr = s.newOpenAIStreamFailoverError(c, account, false, upstreamRequestID, dataBytes, failedMessage)
					return
				}
				forceFlushFailedEvent = true
				sawFailedEvent = true
			}
			imageCounter.AddSSEData(dataBytes)

			// Correct Codex tool calls if needed (apply_patch -> edit, etc.)
			if correctedData, corrected := s.toolCorrector.CorrectToolCallsInSSEBytes(dataBytes); corrected {
				dataBytes = correctedData
				data = string(correctedData)
				line = "data: " + data
				eventType = strings.TrimSpace(gjson.GetBytes(dataBytes, "type").String())
			}
			startsClientOutput := forceFlushFailedEvent || openAIStreamDataStartsClientOutput(data, eventType)

			// 闂傚倸鍊风粈渚€骞夐敓鐘茬闁哄稁鍘介崑锟犳煏婢跺棙娅呴柣顓燁殜閺屾盯鍩勯崘顏呭枦闂佺顑嗛幑鍥偘椤曗偓瀹曞綊顢欓崣銉ф／闂傚倷鑳剁涵鍫曞疾濠婂懐鐭欓柟鎹愭硾閸ㄦ繈鏌嶉崫鍕闁轰礁绉归弻鏇㈠醇濠靛洤绐涘銈忚吂閺呯娀寮婚敐鍡樺劅闁挎稑瀚弳娑㈡⒑閹肩偛濡芥俊鐐舵閻ｇ兘宕崟銊︻潔濠殿喗锚瀹曨剙鐣烽妷銉富闁靛牆妫涙晶顒傜磼鐎ｎ偄娴柟顔惧仱閹墽浠︾粙澶稿濠电偛鐗嗛悘婵嬪几閹剧粯鐓曢悗锝庡亝瀹曞矂鏌＄仦鍓р槈妞ゎ偅绻堥、妤佹媴閻熶即妫烽梻鍌欐祰椤骞愰幎鍓垮骞橀懜娈挎綗?drain 濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸劍閺呮繈鏌曟径鍡樻珕闁稿顦甸弻娑㈩敃閿濆棛顦ㄩ梺?
			if !clientDisconnected {
				shouldFlush := queueDrained && (clientOutputStarted || startsClientOutput)
				if firstTokenMs == nil && startsClientOutput {
					// 濠电姷鏁搁崕鎴犲緤閽樺娲晜閻愵剙搴婇梺绯曞墲缁嬫帡宕曟繝鍌ょ唵閻犺桨璀﹂崕婊呯磼鏉堛劍顥堥柡灞糕偓鎰佸悑闊洦娲滈弳銈夋⒑?token 濠电姷鏁搁崑娑㈡偤閵娧冨灊鐎光偓閸曞灚鏅為梺鍛婃处閸嬧偓闁哄閰ｉ弻鏇＄疀閿濆棛绐旈梺缁樺笒閿曨亪寮婚敃鈧灒濞撴凹鍨辨晥闁荤喐绮忛崺鍥磿閻㈢钃熸繛鎴欏灩缁狅綁鏌ㄥ┑鍡橈紞濞寸姭鏅犲铏圭磼濡偐顓奸梺鎼炲妼閻栫厧顕ｉ銏╁悑闁告侗鍨卞▓鏇㈡⒑閸涘﹥澶勫ù婊呭仧濡叉劕顫濋懜纰樻嫼缂備礁顑堝▔鏇犵不閹殿喚纾煎璺猴功閹界姷绱掓潪鎵煓鐎殿喕绮欓、姗€鎮欓幍顔惧€?TTFT闂?
					shouldFlush = true
				}
				if _, err := bufferedWriter.WriteString(line); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
				} else if _, err := bufferedWriter.WriteString("\n"); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
				} else if shouldFlush {
					if err := flushBuffered(); err != nil {
						clientDisconnected = true
						logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming flush, continuing to drain upstream for billing")
					} else {
						clientOutputStarted = true
						lastDownstreamWriteAt = time.Now()
					}
				}
			}

			// Record first token time
			if firstTokenMs == nil && startsClientOutput {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			s.parseSSEUsageBytes(dataBytes, usage)
			return
		}

		// Forward non-data lines as-is
		if !clientDisconnected {
			if _, err := bufferedWriter.WriteString(line); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
			} else if _, err := bufferedWriter.WriteString("\n"); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
			} else if queueDrained && clientOutputStarted {
				if err := flushBuffered(); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming flush, continuing to drain upstream for billing")
				} else {
					clientOutputStarted = true
					lastDownstreamWriteAt = time.Now()
				}
			}
		}
	}

	// 闂傚倸鍊风粈渚€骞栭锕€鐤柣妤€鐗婇崣蹇涙煕閹炬鎳愰悡瀣⒑閸︻叀妾搁柛鐘崇墱閻?闂?keepalive 闂傚倸鍊烽悞锕傛儑瑜版帒绀夌€光偓閳ь剟鍩€椤掍礁鍤柛鎾跺枛楠炲啴濡烽埡浣侯槶婵炶揪绲块幊鎾诲几閸屾稓绠鹃弶鍫濆⒔閹ジ鏌￠崪浣镐喊闁诡喓鍨藉畷銊╊敍濡も偓娴滈箖鏌涢…鎴濅簼缂佽埖鐓￠幃妤€顫濋悡搴ｄ桓闂佺妫勯崐濠氬焵椤掑﹦绉甸柛鐘崇墱閹叉挳鏁冮崒娑樷偓鍫曟煟閹邦亞绁锋俊鐐倐閺屾盯寮崹顕呬純闂佸搫鐬奸崰搴ㄥ煝閹捐鍨傛い鏃囧吹閸戝綊鏌ｆ惔銏╁晱闁哥姵鐗曢…鍥晸閻樺弶鐎悗骞垮劚濞诧絽鈻介鍫熺參婵☆垯璀﹀Σ瑙勩亜韫囥儳绡€闁?goroutine 濠?channel 闂備浇顕х€涒晠顢欓弽顓炵獥闁圭儤顨呯壕濠氭煙閸撗呭笡闁抽攱鍨块獮鏍偓娑櫳戠亸顓灻瑰鍫㈢暫婵?
	if streamInterval <= 0 && keepaliveInterval <= 0 {
		defer putSSEScannerBuf64K(scanBuf)
		for scanner.Scan() {
			processSSELine(scanner.Text(), true)
			if streamFailoverErr != nil {
				return resultWithUsage(), streamFailoverErr
			}
		}
		if result, err, done := handleScanErr(scanner.Err()); done {
			return result, err
		}
		return finalizeStream()
	}

	type scanEvent struct {
		line string
		err  error
	}
	// 闂傚倸鍊烽懗鍓佸垝椤栫偞鍋℃い鏍仜缁€鍫熺箾閹存瑥鐏柛?goroutine 闂傚倷娴囧畷鍨叏閺夋嚚娲煛閸滀焦鏅悷婊勫灴婵＄敻骞囬弶璺ㄥ€為梺闈浤涢埀顒勫几閺嶃劎绡€闁汇垽娼у瓭闂佸摜鍠嶉崡鍐茬暦閸濆嫀鏃堝川椤旀儳寮虫繝鐢靛仦閸ㄥ爼鈥﹂崶顒佸€跨紒瀣紩閸︻厺鐒婇柡宥冨妽閻濇岸鎮楃憴鍕妞ゃ劌鎳橀、妯荤附缁嬪潡鍞跺銈嗗姂閸ㄦ椽顢欓幋锔解拻濞达絽鎲￠崯鐐烘煟閻曞倻鐣甸柟顔ㄥ洦鍋愰柛娆忓€婚崰鏍х暦閸楃倣鐔兼嚃閳哄啯姣岄梻鍌欒兌缁垶宕濋弽顑句汗闁告劦鍠撻埀顒€鍟埥澶愬閿涘嫬骞?keepalive/闂傚倷鑳堕崕鐢稿礈濠靛牊鏆滈柟鐑橆殔缁犵娀鐓崶銊︽儎婵炴挸顭烽弻娑氫沪閸撗€妲堢紓浣稿閸嬨倝寮诲☉銏犲嵆闁靛鍎辩粻娲⒑?
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}(scanBuf)
	defer close(done)

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return finalizeStream()
			}
			if result, err, done := handleScanErr(ev.err); done {
				return result, err
			}
			processSSELine(ev.line, len(events) == 0)
			if streamFailoverErr != nil {
				return resultWithUsage(), streamFailoverErr
			}

		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if clientDisconnected {
				return resultWithUsage(), fmt.Errorf("stream usage incomplete after timeout")
			}
			logger.LegacyPrintf("service.openai_gateway", "Stream data interval timeout: account=%d model=%s interval=%s", account.ID, originalModel, streamInterval)
			// 濠电姷鏁告慨浼村垂閻撳簶鏋栨繛鎴炲焹閸嬫挸顫濋悡搴㈢彎濡ょ姷鍋涢崯顖滄崲濠靛纾绘い鏍ㄧ箥濞兼劙鏌熷畡鐗堝殗闁诡喖澧芥禒锕€鈹戦崟顐や紙闂佸搫鐭夌换婵嗙暦濮椻偓椤㈡顦查柡鍜冪悼缁辨挻鎷呮禒瀣懙缂備浇顕ч悧鍡涙偩閻戣姤鏅插鑸瞪戦弲顒€鈹戦悩缁樻锭婵☆偅顨婇悰顔锯偓锝庡枟閳锋垿鏌涘┑鍡楊伀鐎涙繂鈹戦悙瀵搞偞闁哄懐濞€閵嗕礁鈻庨幘宕囧€為梺鍐叉惈閸婃悂顢旈幖浣光拺闁圭瀛╅ˉ鍡樹繆椤愩垹顏柟顕嗙節瀵挳鎮欓埡鍌ゆ綌闂備浇顫夊畷姗€锝炴径鎰ラ柛鎰电厛閻斿棝鎮规ウ鎸庮仩閺佸牓鎮楀▓鍨珮闁革綇绲介锝夋偨閸涘﹤浠洪梺姹囧灩閻忔岸宕伴幒妤佲拻濞达絽鎲￠崯鐐烘煟閻曚礁鐏︾€规洖缍婇幃浠嬪川婵犲倸澹勯梻浣虹帛閺屻劑骞夐敓鐘茬？闁绘柨鍚嬮悡銉╂煛閸愩劍澶勬い銉ヮ樀閺岋綀绠涢弮鎴炵杹濠殿喖锕ュ浠嬬嵁閹邦厽鍎熼柨婵嗘濞呮瑦淇婇悙顏勨偓鏍鸿箛娑樼９婵犻潧顑呯粻鏍煃閸濆嫭鍣洪柛銈咁儑閻ヮ亪鎮ч崺顐簼缁?
			if s.rateLimitService != nil {
				s.rateLimitService.HandleStreamTimeout(ctx, account, originalModel)
			}
			sendErrorEvent("stream_timeout")
			return resultWithUsage(), fmt.Errorf("stream data interval timeout")

		case <-keepaliveCh:
			if clientDisconnected {
				continue
			}
			if time.Since(lastDownstreamWriteAt) < keepaliveInterval {
				continue
			}
			if _, err := bufferedWriter.WriteString(":\n\n"); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
				continue
			}
			if err := flushBuffered(); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during keepalive flush, continuing to drain upstream for billing")
			} else {
				lastDownstreamWriteAt = time.Now()
			}
		}
	}

}

// extractOpenAISSEDataLine 濠电姷鏁搁崑鐘诲箵椤忓棗绶ら柤鎭掑労濞堜粙鏌涘┑鍕姕妞ゎ偅娲熼弻鐔告綇妤ｅ啯顎嶉梺缁樻尪閸庣敻寮诲鍫闂佸憡鎸鹃崰鏍偘椤曗偓楠炴鎷犻懠顒傛澑婵＄偑鍊栧濠氬磻閹惧墎纾?SSE `data:` 闂傚倷娴囧畷鐢稿磻閻愮數鐭欓柟杈鹃檮閸ゆ劖銇勯弽顐粶闁肩缍婇幃妤呮偨閻㈢偣鈧﹪鏌涚€ｎ偅宕岄柟顔惧厴楠炲秹顢欓悙顒侇唵闂?// 闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁圭儤顨呴崒銊ф喐閺冨牄鈧?`data: xxx` 濠?`data:xxx` 濠电姷鏁搁崑鐐哄垂閸洖绠伴柛顐ｆ礀绾捐淇婇妶鍛珔缂傚秵鐗犻弻鐔兼倻濮楀棙鐣堕梺鍛婂姀閸嬫捇姊绘担铏瑰笡闁瑰摜顭堥湁濡炲瀛╅崗婊堟煃瑜滈崜鐔奉潖?
func extractOpenAISSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	start := len("data:")
	for start < len(line) {
		if line[start] != ' ' && line[start] != '	' {
			break
		}
		start++
	}
	return line[start:], true
}

func extractOpenAISSEEventLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "event:") {
		return "", false
	}
	start := len("event:")
	for start < len(line) {
		if line[start] != ' ' && line[start] != '	' {
			break
		}
		start++
	}
	return strings.TrimSpace(line[start:]), true
}

type openAICompatSSEFrame struct {
	EventType string
	Data      string
}

type openAICompatSSEFrameParser struct {
	eventType string
	dataLines []string
}

func (p *openAICompatSSEFrameParser) AddLine(line string) (openAICompatSSEFrame, bool) {
	if line == "" {
		return p.dispatch()
	}
	if strings.HasPrefix(line, ":") {
		return openAICompatSSEFrame{}, false
	}
	if eventType, ok := extractOpenAISSEEventLine(line); ok {
		p.eventType = eventType
		return openAICompatSSEFrame{}, false
	}
	if data, ok := extractOpenAISSEDataLine(line); ok {
		p.dataLines = append(p.dataLines, data)
	}
	return openAICompatSSEFrame{}, false
}

func (p *openAICompatSSEFrameParser) Finish() (openAICompatSSEFrame, bool) {
	return p.dispatch()
}

func (p *openAICompatSSEFrameParser) dispatch() (openAICompatSSEFrame, bool) {
	frame := openAICompatSSEFrame{
		EventType: p.eventType,
		Data:      strings.Join(p.dataLines, "\n"),
	}
	p.eventType = ""
	p.dataLines = nil
	return frame, frame.Data != ""
}

func openAICompatPayloadWithEventType(payload, eventType string) string {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" || strings.TrimSpace(payload) == "" || strings.TrimSpace(payload) == "[DONE]" {
		return payload
	}
	if gjson.Get(payload, "type").Exists() {
		return payload
	}
	patched, err := sjson.Set(payload, "type", eventType)
	if err != nil {
		return payload
	}
	return patched
}

func (s *OpenAIGatewayService) replaceModelInSSELine(line, fromModel, toModel string) string {
	data, ok := extractOpenAISSEDataLine(line)
	if !ok {
		return line
	}
	if data == "" || data == "[DONE]" {
		return line
	}

	if m := gjson.Get(data, "model"); m.Exists() && m.Str == fromModel {
		newData, err := sjson.Set(data, "model", toModel)
		if err != nil {
			return line
		}
		return "data: " + newData
	}

	if m := gjson.Get(data, "response.model"); m.Exists() && m.Str == fromModel {
		newData, err := sjson.Set(data, "response.model", toModel)
		if err != nil {
			return line
		}
		return "data: " + newData
	}

	return line
}

// correctToolCallsInResponseBody 濠电姷鏁搁崕鎴犲緤閽樺褰掑磼閻愯尙鐛ュ┑掳鍊曢幏瀣极閸℃绡€濠电姴鍊绘晶鏇灻归懖鈺佇ラ柍褜鍓欑粻宥夊磿闁秴绠犻柟閭﹀枓閸嬫捇鎮烽柇锔叫﹂梻鍥ь樀閺屻劌鈹戦崱姗嗘！闁诲繐娴氶崑濠囧蓟閵堝鍨傛い鎰╁灮娴煎矂姊虹拠鈥崇仩闁哥喐娼欓悾鐑芥偄绾拌鲸鏅濋梺鎸庣☉鐎氼噣鎮￠幋锔解拻濞达絼璀﹂悞鍓х磼缂佹ê娴€殿喚绮鍕箛椤撶偛澹勯梻浣虹帛閺屻劑宕ョ€ｎ喖纾?
func (s *OpenAIGatewayService) correctToolCallsInResponseBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	corrected, changed := s.toolCorrector.CorrectToolCallsInSSEBytes(body)
	if changed {
		return corrected
	}
	return body
}

func (s *OpenAIGatewayService) parseSSEUsage(data string, usage *OpenAIUsage) {
	s.parseSSEUsageBytes([]byte(data), usage)
}

func (s *OpenAIGatewayService) parseSSEUsageBytes(data []byte, usage *OpenAIUsage) {
	if usage == nil || len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
		return
	}
	// 闂傚倸鍊搁崐椋庢閿熺姴纾婚柛鏇ㄥ瀬閸ヮ剙绠ユい鏃傛嚀娴滅偓鎱ㄥΟ绋垮姎濠殿喖鍊归〃銉╂倷閳轰椒澹曢梺鑽ゅ枑缁秹宕规總鎼炩偓鍌涖偅閸愨斁鎷洪梺鍛婄箓鐎氱兘宕曢幇鐗堢厱闁哄倻鍋涢幊蹇涳綖閺囥垺鐓熸俊銈傚亾闁绘绻橀妴鍛存倻閼恒儳鍘卞┑鐐村灥瀹曨剟鐛Δ鍛棅妞ゆ帒顦晶鎾煛鐏炵澧查柟宄版嚇瀹曨偊宕熼崹顐ｇ様闂佽姘﹂～澶愬箰閻戣姤鍊块柨鏂垮⒔閻瑥顭块懜闈涘Е闁轰礁娲弻锝夊箛椤栨氨姣㈠┑鈽嗗灟缁舵艾顫忓ú顏勫窛濠电姴鍊稿В鍫ユ⒑缁嬫鍎忔俊顐ｇ洴瀹曟岸骞掑Δ鈧粻濠氭煙妫颁胶鍔嶆俊妞煎妼椤啴濡舵惔鈥茶埅闂佸憡锕㈢粻鏍箚閸愵喖绠ｆ繝銏＄箓缂嶅﹪骞冮埡鍛紶闁告洦鍘肩敮鍧楁⒒娴ｈ櫣甯涢柣妤婂幘瀵板﹪宕稿Δ鈧粻顖炴煟濡鍤欐鐐灪娣囧﹪顢涘☉娆嶄户濠碘€虫▕閸ㄨ泛顫忔ウ瑁や汗闁圭儤鎸婚柨顓犵磽娴ｅ壊妯€妞ゃ儲鎹囬獮鎴﹀閻欌偓濞笺劑鏌嶈閸撴瑩鎮惧畡鎷旀棃宕ㄩ鍥ュ姂閺屾洘绔熼姘仾闁哥姵娲樼换婵嬫偨闂堟稐绮堕梺缁橆殔濡繈骞冨Ο渚僵闁绘劖鍨濆Ч妤佺箾閹炬潙鐒归柛瀣尵閳ь剝顫夊ú鈺冪礊娴ｅ壊鍤曞ù鐘差儏鍞梺闈涚箳婵櫕绔?
	if len(data) < 72 {
		return
	}
	eventType := gjson.GetBytes(data, "type").String()
	if eventType != "response.completed" && eventType != "response.done" &&
		eventType != "response.incomplete" && eventType != "response.cancelled" && eventType != "response.canceled" {
		return
	}

	if parsedUsage, ok := extractOpenAIUsageFromJSONBytes(data); ok {
		*usage = parsedUsage
	}
}

func extractOpenAIUsageFromJSONBytes(body []byte) (OpenAIUsage, bool) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return OpenAIUsage{}, false
	}
	if usage, ok := openAIUsageFromGJSON(gjson.GetBytes(body, "usage")); ok {
		return usage, true
	}
	return openAIUsageFromGJSON(gjson.GetBytes(body, "response.usage"))
}

func openAIUsageFromGJSON(value gjson.Result) (OpenAIUsage, bool) {
	if !value.Exists() || !value.IsObject() {
		return OpenAIUsage{}, false
	}
	inputTokens := value.Get("input_tokens").Int()
	if inputTokens == 0 {
		inputTokens = value.Get("prompt_tokens").Int()
	}
	outputTokens := value.Get("output_tokens").Int()
	if outputTokens == 0 {
		outputTokens = value.Get("completion_tokens").Int()
	}
	cacheReadTokens := value.Get("input_tokens_details.cached_tokens").Int()
	if cacheReadTokens == 0 {
		cacheReadTokens = value.Get("prompt_tokens_details.cached_tokens").Int()
	}
	imageOutputTokens := value.Get("output_tokens_details.image_tokens").Int()
	if imageOutputTokens == 0 {
		imageOutputTokens = value.Get("completion_tokens_details.image_tokens").Int()
	}
	return OpenAIUsage{
		InputTokens:              int(inputTokens),
		OutputTokens:             int(outputTokens),
		CacheCreationInputTokens: int(value.Get("cache_creation_input_tokens").Int()),
		CacheReadInputTokens:     int(cacheReadTokens),
		ImageOutputTokens:        int(imageOutputTokens),
	}, true
}

func (s *OpenAIGatewayService) handleNonStreamingResponse(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, originalModel, mappedModel string) (*openaiNonStreamingResult, error) {
	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}

	// Detect SSE responses for ALL account types via Content-Type header.
	// Some OpenAI-compatible upstreams (including other sub2api instances)
	// may return SSE even when stream=false was requested.
	if isEventStreamResponse(resp.Header) || looksLikeOpenAINonStreamingSSEBody(body) {
		return s.handleSSEToJSON(resp, c, body, originalModel, mappedModel)
	}

	usageValue, usageOK := extractOpenAIUsageFromJSONBytes(body)
	if !usageOK {
		return nil, fmt.Errorf("parse response: invalid json response")
	}
	usage := &usageValue

	// Replace model in response if needed
	if originalModel != mappedModel {
		body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
	}

	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)

	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}

	c.Data(resp.StatusCode, contentType, body)

	return &openaiNonStreamingResult{
		OpenAIUsage:      usage,
		usage:            usage,
		imageCount:       countOpenAIResponseImageOutputsFromJSONBytes(body),
		imageOutputSizes: collectOpenAIResponseImageOutputSizesFromJSONBytes(body),
	}, nil
}

func isEventStreamResponse(header http.Header) bool {
	contentType := strings.ToLower(header.Get("Content-Type"))
	return strings.Contains(contentType, "text/event-stream")
}

func looksLikeOpenAINonStreamingSSEBody(body []byte) bool {
	if len(body) == 0 || !bytes.Contains(body, []byte("data:")) {
		return false
	}
	bodyText := string(body)
	if _, ok := extractCodexFinalResponse(bodyText); ok {
		return true
	}
	if _, _, ok := extractOpenAISSETerminalEvent(bodyText); ok {
		return true
	}
	for _, line := range strings.Split(bodyText, "\n") {
		if data, ok := extractOpenAISSEDataLine(line); ok && strings.TrimSpace(data) == "[DONE]" {
			return true
		}
	}
	return false
}

func (s *OpenAIGatewayService) handleSSEToJSON(resp *http.Response, c *gin.Context, body []byte, originalModel, mappedModel string) (*openaiNonStreamingResult, error) {
	bodyText := string(body)
	finalResponse, ok := extractCodexFinalResponse(bodyText)

	usage := &OpenAIUsage{}
	if ok {
		if parsedUsage, parsed := extractOpenAIUsageFromJSONBytes(finalResponse); parsed {
			*usage = parsedUsage
		}
		// When the terminal event has an empty output array, reconstruct
		// output from accumulated delta events so the client gets full content.
		// gjson Array() returns empty slice for null, missing, or empty arrays.
		if len(gjson.GetBytes(finalResponse, "output").Array()) == 0 {
			if outputJSON, reconstructed := reconstructResponseOutputFromSSE(bodyText); reconstructed {
				if patched, err := sjson.SetRawBytes(finalResponse, "output", outputJSON); err == nil {
					finalResponse = patched
				}
			}
		}
		body = finalResponse
		if originalModel != mappedModel {
			body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
		}
		// Correct tool calls in final response
		body = s.correctToolCallsInResponseBody(body)
	} else {
		terminalType, terminalPayload, terminalOK := extractOpenAISSETerminalEvent(bodyText)
		if terminalOK && terminalType == "response.failed" {
			msg := extractOpenAISSEErrorMessage(terminalPayload)
			if msg == "" {
				msg = "Upstream compact response failed"
			}
			return nil, s.writeOpenAINonStreamingProtocolError(resp, c, msg)
		}
		usage = s.parseSSEUsageFromBody(bodyText)
		if originalModel != mappedModel {
			bodyText = s.replaceModelInSSEBody(bodyText, mappedModel, originalModel)
		}
		body = []byte(bodyText)
	}

	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)

	contentType := "application/json; charset=utf-8"
	if !ok {
		contentType = resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "text/event-stream"
		}
	}
	c.Data(resp.StatusCode, contentType, body)

	return &openaiNonStreamingResult{
		OpenAIUsage:      usage,
		usage:            usage,
		imageCount:       countOpenAIImageOutputsFromSSEBody(bodyText),
		imageOutputSizes: collectOpenAIImageOutputSizesFromSSEBody(bodyText),
	}, nil
}

func extractOpenAISSETerminalEvent(body string) (string, []byte, bool) {
	var terminalType string
	var terminalPayload []byte
	forEachOpenAISSEDataPayload(body, func(data []byte) {
		if terminalPayload != nil {
			return
		}
		eventType := strings.TrimSpace(gjson.GetBytes(data, "type").String())
		switch eventType {
		case "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
			terminalType = eventType
			terminalPayload = append([]byte(nil), data...)
		}
	})
	if terminalPayload != nil {
		return terminalType, terminalPayload, true
	}
	return "", nil, false
}

func extractOpenAISSEErrorMessage(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	for _, path := range []string{"response.error.message", "error.message", "message"} {
		if msg := strings.TrimSpace(gjson.GetBytes(payload, path).String()); msg != "" {
			return sanitizeUpstreamErrorMessage(msg)
		}
	}
	return sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(payload)))
}

func (s *OpenAIGatewayService) writeOpenAINonStreamingProtocolError(resp *http.Response, c *gin.Context, message string) error {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "Upstream returned an invalid non-streaming response"
	}
	setOpsUpstreamError(c, http.StatusBadGateway, message, "")
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusBadGateway, gin.H{
		"error": gin.H{
			"type":    "upstream_error",
			"message": message,
		},
	})
	return fmt.Errorf("non-streaming openai protocol error: %s", message)
}

func extractCodexFinalResponse(body string) ([]byte, bool) {
	var finalResponse []byte
	forEachOpenAISSEDataPayload(body, func(data []byte) {
		if finalResponse != nil {
			return
		}
		eventType := gjson.GetBytes(data, "type").String()
		if eventType == "response.done" || eventType == "response.completed" {
			if response := gjson.GetBytes(data, "response"); response.Exists() && response.Type == gjson.JSON && response.Raw != "" {
				finalResponse = []byte(response.Raw)
			}
		}
	})
	if finalResponse != nil {
		return finalResponse, true
	}
	return nil, false
}

// reconstructResponseOutputFromSSE scans raw SSE body text for delta events and
// returns a JSON-encoded output array reconstructed from accumulated deltas.
// Returns (nil, false) if no content was found in deltas.
func reconstructResponseOutputFromSSE(bodyText string) ([]byte, bool) {
	acc := apicompat.NewBufferedResponseAccumulator()
	imageOutputs := make([]json.RawMessage, 0, 1)
	seenImages := make(map[string]struct{})
	forEachOpenAISSEDataPayload(bodyText, func(data []byte) {
		if imageOutput, ok := extractImageGenerationOutputFromSSEData(data, seenImages); ok {
			imageOutputs = append(imageOutputs, imageOutput)
		}
		var event apicompat.ResponsesStreamEvent
		if err := json.Unmarshal(data, &event); err == nil {
			acc.ProcessEvent(&event)
		}
	})
	if !acc.HasContent() && len(imageOutputs) == 0 {
		return nil, false
	}

	var output []json.RawMessage
	if acc.HasContent() {
		outputJSON, err := json.Marshal(acc.BuildOutput())
		if err == nil {
			_ = json.Unmarshal(outputJSON, &output)
		}
	}
	output = append(output, imageOutputs...)
	if len(output) == 0 {
		return nil, false
	}

	outputJSON, err := json.Marshal(output)
	if err != nil {
		return nil, false
	}
	return outputJSON, true
}

func extractImageGenerationOutputFromSSEData(data []byte, seen map[string]struct{}) (json.RawMessage, bool) {
	if len(data) == 0 || !gjson.ValidBytes(data) {
		return nil, false
	}
	if gjson.GetBytes(data, "type").String() != "response.output_item.done" {
		return nil, false
	}
	item := gjson.GetBytes(data, "item")
	if !item.Exists() || !item.IsObject() || item.Get("type").String() != "image_generation_call" {
		return nil, false
	}
	if strings.TrimSpace(item.Get("result").String()) == "" {
		return nil, false
	}
	key := strings.TrimSpace(item.Get("id").String())
	if key == "" {
		key = strings.TrimSpace(item.Get("output_format").String()) + "|" + strings.TrimSpace(item.Get("result").String())
	}
	if key != "" && seen != nil {
		if _, exists := seen[key]; exists {
			return nil, false
		}
		seen[key] = struct{}{}
	}
	return json.RawMessage(item.Raw), true
}

func (s *OpenAIGatewayService) parseSSEUsageFromBody(body string) *OpenAIUsage {
	usage := &OpenAIUsage{}
	forEachOpenAISSEDataPayload(body, func(data []byte) {
		s.parseSSEUsageBytes(data, usage)
	})
	return usage
}

func (s *OpenAIGatewayService) replaceModelInSSEBody(body, fromModel, toModel string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if _, ok := extractOpenAISSEDataLine(line); !ok {
			continue
		}
		lines[i] = s.replaceModelInSSELine(line, fromModel, toModel)
	}
	return strings.Join(lines, "\n")
}

func (s *OpenAIGatewayService) validateUpstreamBaseURL(raw string) (string, error) {
	if s.cfg != nil && !s.cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateURLFormat(raw, s.cfg.Security.URLAllowlist.AllowInsecureHTTP)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     s.cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     s.cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

// buildOpenAIResponsesURL builds the OpenAI Responses endpoint URL.
func buildOpenAIResponsesURL(base string) string {
	return buildOpenAIEndpointURL(base, "/v1/responses")
}

func trimOpenAIEncryptedReasoningItems(reqBody map[string]any) bool {
	if len(reqBody) == 0 {
		return false
	}

	inputValue, has := reqBody["input"]
	if !has {
		return false
	}

	switch input := inputValue.(type) {
	case []any:
		filtered := input[:0]
		changed := false
		for _, item := range input {
			nextItem, itemChanged, keep := sanitizeEncryptedReasoningInputItem(item)
			if itemChanged {
				changed = true
			}
			if !keep {
				continue
			}
			filtered = append(filtered, nextItem)
		}
		if !changed {
			return false
		}
		if len(filtered) == 0 {
			delete(reqBody, "input")
			return true
		}
		reqBody["input"] = filtered
		return true
	case []map[string]any:
		filtered := input[:0]
		changed := false
		for _, item := range input {
			nextItem, itemChanged, keep := sanitizeEncryptedReasoningInputItem(item)
			if itemChanged {
				changed = true
			}
			if !keep {
				continue
			}
			nextMap, ok := nextItem.(map[string]any)
			if !ok {
				filtered = append(filtered, item)
				continue
			}
			filtered = append(filtered, nextMap)
		}
		if !changed {
			return false
		}
		if len(filtered) == 0 {
			delete(reqBody, "input")
			return true
		}
		reqBody["input"] = filtered
		return true
	case map[string]any:
		nextItem, changed, keep := sanitizeEncryptedReasoningInputItem(input)
		if !changed {
			return false
		}
		if !keep {
			delete(reqBody, "input")
			return true
		}
		nextMap, ok := nextItem.(map[string]any)
		if !ok {
			return false
		}
		reqBody["input"] = nextMap
		return true
	default:
		return false
	}
}

func sanitizeEncryptedReasoningInputItem(item any) (next any, changed bool, keep bool) {
	inputItem, ok := item.(map[string]any)
	if !ok {
		return item, false, true
	}

	itemType, _ := inputItem["type"].(string)
	if strings.TrimSpace(itemType) != "reasoning" {
		return item, false, true
	}

	_, hasEncryptedContent := inputItem["encrypted_content"]
	if !hasEncryptedContent {
		return item, false, true
	}

	delete(inputItem, "encrypted_content")
	if len(inputItem) == 1 {
		return nil, true, false
	}
	return inputItem, true, true
}

func IsOpenAIResponsesCompactPathForTest(c *gin.Context) bool {
	return isOpenAIResponsesCompactPath(c)
}

func OpenAICompactSessionSeedKeyForTest() string {
	return openAICompactSessionSeedKey
}

func NormalizeOpenAICompactRequestBodyForTest(body []byte) ([]byte, bool, error) {
	return normalizeOpenAICompactRequestBody(body)
}

func isOpenAIResponsesCompactPath(c *gin.Context) bool {
	suffix := strings.TrimSpace(openAIResponsesRequestPathSuffix(c))
	return suffix == "/compact" || strings.HasPrefix(suffix, "/compact/")
}

func normalizeOpenAICompactRequestBody(body []byte) ([]byte, bool, error) {
	if len(body) == 0 {
		return body, false, nil
	}

	normalized := []byte(`{}`)
	// Keep the current Codex /compact schema while still dropping request-scoped
	// fields such as prompt_cache_key, store, and stream.
	for _, field := range []string{
		"model",
		"input",
		"instructions",
		"tools",
		"parallel_tool_calls",
		"reasoning",
		"text",
		"previous_response_id",
	} {
		value := gjson.GetBytes(body, field)
		if !value.Exists() {
			continue
		}
		next, err := sjson.SetRawBytes(normalized, field, []byte(value.Raw))
		if err != nil {
			return body, false, fmt.Errorf("normalize compact body %s: %w", field, err)
		}
		normalized = next
	}

	if bytes.Equal(bytes.TrimSpace(body), bytes.TrimSpace(normalized)) {
		return body, false, nil
	}
	return normalized, true, nil
}

func resolveOpenAICompactSessionID(c *gin.Context) string {
	if c != nil {
		if sessionID := strings.TrimSpace(c.GetHeader("session_id")); sessionID != "" {
			return sessionID
		}
		if conversationID := strings.TrimSpace(c.GetHeader("conversation_id")); conversationID != "" {
			return conversationID
		}
		if seed, ok := c.Get(openAICompactSessionSeedKey); ok {
			if seedStr, ok := seed.(string); ok && strings.TrimSpace(seedStr) != "" {
				return strings.TrimSpace(seedStr)
			}
		}
	}
	return uuid.NewString()
}

func openAIResponsesRequestPathSuffix(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	normalizedPath := strings.TrimRight(strings.TrimSpace(c.Request.URL.Path), "/")
	if normalizedPath == "" {
		return ""
	}
	idx := strings.LastIndex(normalizedPath, "/responses")
	if idx < 0 {
		return ""
	}
	suffix := normalizedPath[idx+len("/responses"):]
	if suffix == "" || suffix == "/" {
		return ""
	}
	if !strings.HasPrefix(suffix, "/") {
		return ""
	}
	return suffix
}

func appendOpenAIResponsesRequestPathSuffix(baseURL, suffix string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedSuffix := strings.TrimSpace(suffix)
	if trimmedBase == "" || trimmedSuffix == "" {
		return trimmedBase
	}
	return trimmedBase + trimmedSuffix
}

func (s *OpenAIGatewayService) replaceModelInResponseBody(body []byte, fromModel, toModel string) []byte {
	if m := gjson.GetBytes(body, "model"); m.Exists() && m.Str == fromModel {
		newBody, err := sjson.SetBytes(body, "model", toModel)
		if err != nil {
			return body
		}
		return newBody
	}
	return body
}

// OpenAIRecordUsageInput input for recording usage
type OpenAIRecordUsageInput struct {
	Result             *OpenAIForwardResult
	APIKey             *APIKey
	User               *User
	Account            *Account
	Subscription       *UserSubscription
	InboundEndpoint    string
	UpstreamEndpoint   string
	UserAgent          string // 闂傚倷娴囧畷鍨叏閺夋嚚娲敇閵忕姷鍝楅梻渚囧墮缁夌敻宕曢幋婢濆綊鎮℃惔锝嗘喖闂?User-Agent
	IPAddress          string // Client IP address.
	RequestPayloadHash string
	APIKeyService      APIKeyQuotaUpdater
	ChannelUsageFields
}

// RecordUsage records usage and deducts balance
func (s *OpenAIGatewayService) RecordUsage(ctx context.Context, input *OpenAIRecordUsageInput) error {
	if input == nil {
		return errors.New("openai usage input is nil")
	}
	result := input.Result
	if result == nil {
		return errors.New("openai usage result is nil")
	}
	if s.rateLimitService != nil && input.Account != nil && input.Account.Platform == PlatformOpenAI {
		s.rateLimitService.ResetOpenAI403Counter(ctx, input.Account.ID)
	}

	apiKey := input.APIKey
	user := input.User
	account := input.Account
	subscription := input.Subscription
	ApplyOpenAIImageBillingResolution(result)

	// 闂傚倷娴囧畷鍨叏瀹曞洦顐介柕鍫濇处椤洟鏌￠崶銉ョ仾闁稿鏅犻弻銈嗘叏閹邦兘鍋撻幇鏉跨；闁规崘顕ч幑鑸点亜閹捐泛浠掓俊鍙夊姇閳规垿鏁嶉崟顔筋€楅梺鎼炲妼閻栧ジ鐛崱娑樼妞ゆ棁鍋愰ˇ鏉款渻閵堝棗绗傞柤鍐茬埣瀹曘垽骞囬悧鍫㈠幐婵犮垼娉涢敃锔芥櫠濞戙垺鐓欓柛娑橈攻閸婃劗鈧娲╃紞浣哥暦椤掆偓椤繘鎮ラ崳鈧琻闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬪鏌涢埄鍐槈闂傚偆鍨辩换娑橆啅椤旇崵鐩庨梺绋胯閸旀垿寮婚妶鍡樺弿闁归偊鍏橀崑鎾澄旀担鍝ョ獮闂佸憡娲﹂崹閬嶅磹閸偆绠鹃柟瀵稿仧閹虫劙鏌ｉ幒鏂夸壕闁靛洤瀚板鏉戭潩椤掆偓缁侇噣鎮楃憴鍕┛缂傚秳绀侀锝嗙鐎ｅ灚鏅濋梺鎸庣箓濞诧箑鈻撶粩顪祂en闂?	// 闂傚倸鍊烽悞锕傚箖閸洖纾块柧蹇ｅ亝閸欏繒鈧娲栧ú锔锯偓?input_tokens 闂傚倸鍊风粈渚€骞夐敓鐘冲亱闁哄洢鍨圭粻鐘诲箹濞ｎ剙濡肩紒鐘冲哺閺屾盯顢曢妶鍛亖濡?cache_read_tokens闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸庢銆掑锝呬壕闂佺硶鏂侀崑鎾愁渻閵堝棗鍧婇柛瀣崌閺岋綁顢楅崘銊ゆ睏濠碘€冲级閸旀瑩寮崒鐐寸叆妞ゆ牗鐭竟鏇炩攽閻愭潙鐏﹂柣鐕佸灦閹偤鎮欓鍙ョ盎闂佸搫娲﹂懝鎯х暤閸℃瑢鍋撶憴鍕┛缂傚秳绀侀锝嗙鐎ｅ灚鏅濋梺鎸庣箓濞诧箑鈻撶粩顪祂en濠电姷鏁搁崑鐐哄垂閸洖绠伴柛婵勫劤閻捇鎮楅崹顐ゆ憙濠殿喗濞婇弻锛勪沪鐠囨彃濮庣紓浣哄У濠㈡﹢鈥︾捄銊﹀磯闁绘垶蓱瀹曞磭绱撻崒姘卞妞ゎ厾鍏樺璇测槈濠婂懐鏉搁柣搴秵娴滄粍鎱ㄥ澶嬧拺缂備焦顭囬惌銈夋煕閵娿劍纭炬い鏇稻缁傛帞鈧綆鍋勫▓鐐烘⒒娓氬洤寮跨紒鐘冲灴楠炲啯绗熼埀顒勫蓟?
	actualInputTokens := result.Usage.InputTokens - result.Usage.CacheReadInputTokens
	if actualInputTokens < 0 {
		actualInputTokens = 0
	}

	// Calculate cost
	tokens := UsageTokens{
		InputTokens:         actualInputTokens,
		OutputTokens:        result.Usage.OutputTokens,
		CacheCreationTokens: result.Usage.CacheCreationInputTokens,
		CacheReadTokens:     result.Usage.CacheReadInputTokens,
		ImageOutputTokens:   result.Usage.ImageOutputTokens,
	}

	// Get rate multiplier
	multiplier := 1.0
	if s.cfg != nil {
		multiplier = s.cfg.Default.RateMultiplier
	}
	if apiKey.GroupID != nil && apiKey.Group != nil {
		resolver := s.userGroupRateResolver
		if resolver == nil {
			resolver = newUserGroupRateResolver(nil, nil, resolveUserGroupRateCacheTTL(s.cfg), nil, "service.openai_gateway")
		}
		multiplier = resolver.Resolve(ctx, user.ID, *apiKey.GroupID, apiKey.Group.RateMultiplier)
	}
	imageMultiplier := resolveImageRateMultiplier(apiKey, multiplier)

	var cost *CostBreakdown
	var err error
	billingModel := forwardResultBillingModel(result.Model, result.UpstreamModel)
	if result.BillingModel != "" {
		billingModel = strings.TrimSpace(result.BillingModel)
	}
	if input.BillingModelSource == BillingModelSourceChannelMapped && input.ChannelMappedModel != "" && input.ChannelMappedModel != input.OriginalModel {
		billingModel = input.ChannelMappedModel
	}
	if input.BillingModelSource == BillingModelSourceRequested && input.OriginalModel != "" {
		billingModel = input.OriginalModel
	}
	billingModels := usageBillingModelCandidates(
		billingModel,
		result.BillingModel,
		input.ChannelMappedModel,
		input.OriginalModel,
		result.UpstreamModel,
		result.Model,
	)
	serviceTier := ""
	if result.ServiceTier != nil {
		serviceTier = strings.TrimSpace(*result.ServiceTier)
	}
	cost, err = s.calculateOpenAIRecordUsageCost(ctx, result, apiKey, billingModels, multiplier, imageMultiplier, tokens, serviceTier)
	if err != nil {
		if !isUsagePricingUnavailableError(err) {
			return err
		}
		logger.L().With(
			zap.String("component", "service.openai_gateway"),
			zap.Strings("billing_models", billingModels),
			zap.String("requested_model", input.OriginalModel),
			zap.String("mapped_model", input.ChannelMappedModel),
			zap.String("upstream_model", result.UpstreamModel),
			zap.Int64("api_key_id", apiKey.ID),
			zap.Int64("account_id", account.ID),
		).Warn("openai_usage.pricing_missing_record_zero_cost", zap.Error(err))
		cost = &CostBreakdown{BillingMode: string(BillingModeToken)}
	}

	// Determine billing type
	isSubscriptionBilling := subscription != nil && apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	billingType := BillingTypeBalance
	if isSubscriptionBilling {
		billingType = BillingTypeSubscription
	}

	// Create usage log
	durationMs := int(result.Duration.Milliseconds())
	accountRateMultiplier := account.BillingRateMultiplier()
	requestID := resolveUsageBillingRequestID(ctx, result.RequestID)

	// Prefer the original requested model for billing display when it differs from the upstream model.
	requestedModel := result.Model
	if input.OriginalModel != "" {
		requestedModel = input.OriginalModel
	}

	usageLog := &UsageLog{
		UserID:              user.ID,
		APIKeyID:            apiKey.ID,
		AccountID:           account.ID,
		RequestID:           requestID,
		Model:               result.Model,
		RequestedModel:      requestedModel,
		UpstreamModel:       optionalNonEqualStringPtr(result.UpstreamModel, result.Model),
		ServiceTier:         result.ServiceTier,
		ReasoningEffort:     result.ReasoningEffort,
		InboundEndpoint:     optionalTrimmedStringPtr(input.InboundEndpoint),
		UpstreamEndpoint:    optionalTrimmedStringPtr(input.UpstreamEndpoint),
		InputTokens:         actualInputTokens,
		OutputTokens:        result.Usage.OutputTokens,
		CacheCreationTokens: result.Usage.CacheCreationInputTokens,
		CacheReadTokens:     result.Usage.CacheReadInputTokens,
		ImageOutputTokens:   result.Usage.ImageOutputTokens,
		ImageCount:          result.ImageCount,
		ImageSize:           optionalTrimmedStringPtr(result.ImageSize),
		ImageInputSize:      optionalTrimmedStringPtr(result.ImageInputSize),
		ImageOutputSize:     optionalTrimmedStringPtr(result.ImageOutputSize),
		ImageSizeSource:     optionalTrimmedStringPtr(result.ImageSizeSource),
		ImageSizeBreakdown:  result.ImageSizeBreakdown,
		VideoCount:          result.VideoCount,
	}
	if cost != nil {
		usageLog.InputCost = cost.InputCost
		usageLog.OutputCost = cost.OutputCost
		usageLog.ImageOutputCost = cost.ImageOutputCost
		usageLog.CacheCreationCost = cost.CacheCreationCost
		usageLog.CacheReadCost = cost.CacheReadCost
		usageLog.TotalCost = cost.TotalCost
		usageLog.ActualCost = cost.ActualCost
	}
	if result.ImageCount > 0 {
		usageLog.RateMultiplier = imageMultiplier
	} else {
		usageLog.RateMultiplier = multiplier
	}
	usageLog.AccountRateMultiplier = &accountRateMultiplier
	usageLog.BillingType = billingType
	usageLog.Stream = result.Stream
	usageLog.OpenAIWSMode = result.OpenAIWSMode
	usageLog.DurationMs = &durationMs
	usageLog.FirstTokenMs = result.FirstTokenMs
	usageLog.CreatedAt = time.Now()
	// 闂傚倷娴囧畷鍨叏瀹曞洨鐭嗗ù锝堫潐濞呯姴霉閻樺樊鍎愰柛瀣典邯閺屾盯鍩勯崘锔跨不闂侀€炲苯澧悽顖ょ節楠炲啴鍩￠崘鈺侇€涢梺鍛婂姇濡﹤螞濠婂啠鏀介柣鎴濇川閸掔増绻涢懠顒€鈻堢€规洘绮岄埞鎴﹀炊瑜滈崵?	usageLog.ChannelID = optionalInt64Ptr(input.ChannelID)
	usageLog.ModelMappingChain = optionalTrimmedStringPtr(input.ModelMappingChain)
	// 闂傚倷娴囧畷鍨叏瀹曞洨鐭嗗ù锝堫潐濞呯姴霉閻樺樊鍎愰柛瀣典邯閺屾盯鍩勯崘顏呭櫗婵犳鍨伴妶绋款潖娴犲鐐婃い鎺嗗亾闁搞劌鍊块弻娑氫沪缂併垹娈奸梺缁樻⒒閸樠囨倶瀹曞洠鍋撶憴鍕婵炲眰鍔戦妴?
	if cost != nil && cost.BillingMode != "" {
		billingMode := cost.BillingMode
		usageLog.BillingMode = &billingMode
	} else if result.ImageCount > 0 {
		billingMode := string(BillingModeImage)
		usageLog.BillingMode = &billingMode
	} else {
		billingMode := string(BillingModeToken)
		usageLog.BillingMode = &billingMode
	}
	// 婵犵數濮烽弫鎼佸磿閹寸姷绀婇柍褜鍓氶妵鍕即閸℃顏柛?UserAgent
	if input.UserAgent != "" {
		usageLog.UserAgent = &input.UserAgent
	}

	// 婵犵數濮烽弫鎼佸磿閹寸姷绀婇柍褜鍓氶妵鍕即閸℃顏柛?IPAddress
	if input.IPAddress != "" {
		usageLog.IPAddress = &input.IPAddress
	}

	if apiKey.GroupID != nil {
		usageLog.GroupID = apiKey.GroupID
	}
	if subscription != nil {
		usageLog.SubscriptionID = &subscription.ID
	}

	// 闂傚倷娴囧畷鍨叏瀹曞洦顐介柕鍫濇处椤洟鏌￠崶銉ョ仾闁稿鏅犻弻銈嗘叏閹邦兘鍋撻弽顐熷亾濮橆剦鐓奸柡灞诲姂瀵潙螖閳ь剚绂嶆ィ鍐╁仭婵犲﹤瀚婊勭箾婢跺绀堟俊鍙夊姍楠炴帒螖娴ｉ晲绨绘繝鐢靛仩鐏忣亪顢氳楠炲啯绗熼埀顒勫蓟閿濆棙鍎熼柕鍫濆缂嶅牓姊哄ú璇插季闁革綇缍侀獮鍐箮缁涘鏅┑鐘诧工鐎氼剟顢旈幖浣光拺闂傚牃鏅涢惁婊堟煕濡湱鐭欑€殿喖鐤囩粻娑樷槈濞嗘垵寮虫繝鐢靛仦閸ㄥ爼鈥﹂崶銊︽珷闁哄倸绨遍弨浠嬫煟閹邦剙绾ч柛鐘筹耿閺岀喖顢欓懡銈囩厯婵犵鍓濋幃鍌炲极閸岀偛绠ｉ柟鐑樻煥缁楋紕绱撻崒姘偓鎼佸磹妞嬪海鐭嗗〒姘ｅ亾闁诡啫鍥у窛妞ゆ牗绮庨悾鍫曟⒑缁嬫寧婀伴柛婵嗛叄閸┾偓妞ゆ帊绀佸ù顔锯偓瑙勬磸閸斿秶鎹㈠☉銏犳闁告挆鍛床闁诲氦顫夊ú鏍э耿闁秴鐒垫い鎺戯功缁夐潧霉濠婂嫮鐭掗柟顔哄劦閺屽棗顓奸崱蹇斿闂備礁鎼ˇ鍐测枖閺囩偞姣勯梻鍌欑劍閹爼宕愰弽顐㈠灊閹兼番鍔岄悞鍨亜閹烘垵鏋ゆ繛鍏煎姍閺岋綁顢樿閺嬨倗绱掗纰辩吋闁诡喓鍨介幃婊兾熼悜妯尖偓顓㈡⒒娴ｅ憡鍟炴繛璇х畵瀹曟垿宕熼娑樷偓鍫曟煟閺傚灝鎮戦柣?
	if apiKey.GroupID != nil {
		applyAccountStatsCost(ctx, usageLog, s.channelService, s.billingService,
			account.ID, *apiKey.GroupID, result.UpstreamModel, result.Model,
			tokens, cost.TotalCost,
		)
	}

	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")
		logger.LegacyPrintf("service.openai_gateway", "[SIMPLE MODE] Usage recorded (not billed): user=%d, tokens=%d", usageLog.UserID, usageLog.TotalTokens())
		s.deferredService.ScheduleLastUsedUpdate(account.ID)
		return nil
	}

	billingErr := func() error {
		_, err := applyUsageBilling(ctx, requestID, usageLog, &postUsageBillingParams{
			Cost:                  cost,
			User:                  user,
			APIKey:                apiKey,
			Account:               account,
			Subscription:          subscription,
			RequestPayloadHash:    resolveUsageBillingPayloadFingerprint(ctx, input.RequestPayloadHash),
			IsSubscriptionBill:    isSubscriptionBilling,
			AccountRateMultiplier: accountRateMultiplier,
			APIKeyService:         input.APIKeyService,
		}, s.billingDeps(), s.usageBillingRepo)
		return err
	}()

	if billingErr != nil {
		return billingErr
	}
	writeUsageLogBestEffort(ctx, s.usageLogRepo, usageLog, "service.openai_gateway")

	return nil
}

func (s *OpenAIGatewayService) calculateOpenAIRecordUsageCost(
	ctx context.Context,
	result *OpenAIForwardResult,
	apiKey *APIKey,
	billingModels []string,
	multiplier float64,
	imageMultiplier float64,
	tokens UsageTokens,
	serviceTier string,
) (*CostBreakdown, error) {
	billingModel := firstUsageBillingModel(billingModels)
	if result != nil && result.ImageCount > 0 {
		return s.calculateOpenAIImageCost(ctx, billingModel, apiKey, result, imageMultiplier), nil
	}
	if result != nil && result.VideoCount > 0 {
		return s.calculateOpenAIVideoCost(ctx, billingModel, apiKey, result, multiplier), nil
	}
	if len(billingModels) == 0 || billingModel == "" {
		return nil, errors.New("openai usage billing model is empty")
	}
	var lastErr error
	for _, candidate := range billingModels {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		cost, err := s.calculateOpenAIRecordUsageTokenCost(ctx, apiKey, candidate, multiplier, tokens, serviceTier)
		if err == nil {
			return cost, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no non-empty billing model candidates")
	}
	return nil, fmt.Errorf("calculate OpenAI usage cost failed for billing models %s: %w", strings.Join(billingModels, ","), lastErr)
}

func isUsagePricingUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrModelPricingUnavailable) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no pricing available") || strings.Contains(msg, "pricing not found")
}

func (s *OpenAIGatewayService) calculateOpenAIVideoCost(
	ctx context.Context,
	billingModel string,
	apiKey *APIKey,
	result *OpenAIForwardResult,
	multiplier float64,
) *CostBreakdown {
	videoCount := 1
	if result != nil && result.VideoCount > 0 {
		videoCount = result.VideoCount
	}
	if resolved := s.resolveOpenAIChannelPricing(ctx, billingModel, apiKey); resolved != nil &&
		(resolved.Mode == BillingModePerRequest || resolved.Mode == BillingModeVideo) {
		gid := apiKey.Group.ID
		cost, err := s.billingService.CalculateCostUnified(CostInput{
			Ctx:            ctx,
			Model:          billingModel,
			GroupID:        &gid,
			RequestCount:   videoCount,
			RateMultiplier: multiplier,
			Resolver:       s.resolver,
			Resolved:       resolved,
		})
		if err == nil {
			return cost
		}
		logger.LegacyPrintf("service.openai_gateway", "Calculate video channel cost failed: %v", err)
	}
	return &CostBreakdown{BillingMode: string(BillingModeVideo)}
}

func (s *OpenAIGatewayService) calculateOpenAIRecordUsageTokenCost(
	ctx context.Context,
	apiKey *APIKey,
	billingModel string,
	multiplier float64,
	tokens UsageTokens,
	serviceTier string,
) (*CostBreakdown, error) {
	if s.resolver != nil && apiKey.Group != nil {
		gid := apiKey.Group.ID
		return s.billingService.CalculateCostUnified(CostInput{
			Ctx:            ctx,
			Model:          billingModel,
			GroupID:        &gid,
			Tokens:         tokens,
			RequestCount:   1,
			RateMultiplier: multiplier,
			ServiceTier:    serviceTier,
			Resolver:       s.resolver,
		})
	}
	return s.billingService.CalculateCostWithServiceTier(billingModel, tokens, multiplier, serviceTier)
}

func (s *OpenAIGatewayService) calculateOpenAIImageCost(
	ctx context.Context,
	billingModel string,
	apiKey *APIKey,
	result *OpenAIForwardResult,
	multiplier float64,
) *CostBreakdown {
	sizeTier := NormalizeImageBillingTierOrDefault(result.ImageSize)
	if resolved := s.resolveOpenAIChannelPricing(ctx, billingModel, apiKey); resolved != nil &&
		(resolved.Mode == BillingModePerRequest || resolved.Mode == BillingModeImage) {
		gid := apiKey.Group.ID
		cost, err := s.billingService.CalculateCostUnified(CostInput{
			Ctx:            ctx,
			Model:          billingModel,
			GroupID:        &gid,
			RequestCount:   result.ImageCount,
			SizeTier:       sizeTier,
			RateMultiplier: multiplier,
			Resolver:       s.resolver,
			Resolved:       resolved,
		})
		if err == nil {
			return cost
		}
		logger.LegacyPrintf("service.openai_gateway", "Calculate image channel cost failed: %v", err)
	}

	var groupConfig *ImagePriceConfig
	if apiKey != nil && apiKey.Group != nil {
		groupConfig = &ImagePriceConfig{
			Price1K: apiKey.Group.ImagePrice1K,
			Price2K: apiKey.Group.ImagePrice2K,
			Price4K: apiKey.Group.ImagePrice4K,
		}
	}
	return s.billingService.CalculateImageCost(billingModel, sizeTier, result.ImageCount, groupConfig, multiplier)
}

func (s *OpenAIGatewayService) resolveOpenAIChannelPricing(ctx context.Context, billingModel string, apiKey *APIKey) *ResolvedPricing {
	if s.resolver == nil || apiKey == nil || apiKey.Group == nil {
		return nil
	}
	gid := apiKey.Group.ID
	resolved := s.resolver.Resolve(ctx, PricingInput{Model: billingModel, GroupID: &gid})
	if resolved.Source == PricingSourceChannel {
		return resolved
	}
	return nil
}

// ParseCodexRateLimitHeaders extracts Codex usage limits from response headers.
// Exported for use in ratelimit_service when handling OpenAI 429 responses.
func ParseCodexRateLimitHeaders(headers http.Header) *OpenAICodexUsageSnapshot {
	snapshot := &OpenAICodexUsageSnapshot{}
	hasData := false

	// Helper to parse float64 from header
	parseFloat := func(key string) *float64 {
		if v := headers.Get(key); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return &f
			}
		}
		return nil
	}

	// Helper to parse int from header
	parseInt := func(key string) *int {
		if v := headers.Get(key); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return &i
			}
		}
		return nil
	}

	// Primary (weekly) limits
	if v := parseFloat("x-codex-primary-used-percent"); v != nil {
		snapshot.PrimaryUsedPercent = v
		hasData = true
	}
	if v := parseInt("x-codex-primary-reset-after-seconds"); v != nil {
		snapshot.PrimaryResetAfterSeconds = v
		hasData = true
	}
	if v := parseInt("x-codex-primary-window-minutes"); v != nil {
		snapshot.PrimaryWindowMinutes = v
		hasData = true
	}

	// Secondary (5h) limits
	if v := parseFloat("x-codex-secondary-used-percent"); v != nil {
		snapshot.SecondaryUsedPercent = v
		hasData = true
	}
	if v := parseInt("x-codex-secondary-reset-after-seconds"); v != nil {
		snapshot.SecondaryResetAfterSeconds = v
		hasData = true
	}
	if v := parseInt("x-codex-secondary-window-minutes"); v != nil {
		snapshot.SecondaryWindowMinutes = v
		hasData = true
	}

	// Overflow ratio
	if v := parseFloat("x-codex-primary-over-secondary-limit-percent"); v != nil {
		snapshot.PrimaryOverSecondaryPercent = v
		hasData = true
	}

	if !hasData {
		return nil
	}

	snapshot.UpdatedAt = time.Now().Format(time.RFC3339)
	return snapshot
}

func codexSnapshotBaseTime(snapshot *OpenAICodexUsageSnapshot, fallback time.Time) time.Time {
	if snapshot == nil {
		return fallback
	}
	if snapshot.UpdatedAt == "" {
		return fallback
	}
	base, err := time.Parse(time.RFC3339, snapshot.UpdatedAt)
	if err != nil {
		return fallback
	}
	return base
}

func codexResetAtRFC3339(base time.Time, resetAfterSeconds *int) *string {
	if resetAfterSeconds == nil {
		return nil
	}
	sec := *resetAfterSeconds
	if sec < 0 {
		sec = 0
	}
	resetAt := base.Add(time.Duration(sec) * time.Second).Format(time.RFC3339)
	return &resetAt
}

func buildCodexUsageExtraUpdates(snapshot *OpenAICodexUsageSnapshot, fallbackNow time.Time) map[string]any {
	if snapshot == nil {
		return nil
	}

	baseTime := codexSnapshotBaseTime(snapshot, fallbackNow)
	updates := make(map[string]any)

	// 濠电姷鏁搁崕鎴犲緤閽樺娲晜閻愵剙搴婇梺绋跨灱閸嬬偤宕戦妶澶嬬厪濠电偟鍋撳▍鍛存煕鐏炶濮傞柡宀嬬秮瀵剙鈻庨悙顒傛瀮婵?primary/secondary 闂傚倷娴囬褏鈧稈鏅濈划娆撳箳濡炲皷鍋撻崘顔煎耿婵炴垼椴搁弲鈺呮倵閸忓浜鹃梺鍛婃处閸撴艾鈻嶉弽顓熺厽闊洦娲栨禒鈺呮煕鐎ｎ偄濮嶉柛鈺冨仱楠炴帡骞婇妸褎鍤€妞ゎ厹鍔戝畷姗€鈥﹂幋婵單ㄩ梻浣圭湽閸╁嫰宕归柆宥呯闁搞儜灞芥婵犵數濮甸懝鎯ф暜闂備礁鍟块幖顐﹀磻婵犲嫭顫曟い蹇撴噸缁?
	if snapshot.PrimaryUsedPercent != nil {
		updates["codex_primary_used_percent"] = *snapshot.PrimaryUsedPercent
	}
	if snapshot.PrimaryResetAfterSeconds != nil {
		updates["codex_primary_reset_after_seconds"] = *snapshot.PrimaryResetAfterSeconds
	}
	if snapshot.PrimaryWindowMinutes != nil {
		updates["codex_primary_window_minutes"] = *snapshot.PrimaryWindowMinutes
	}
	if snapshot.SecondaryUsedPercent != nil {
		updates["codex_secondary_used_percent"] = *snapshot.SecondaryUsedPercent
	}
	if snapshot.SecondaryResetAfterSeconds != nil {
		updates["codex_secondary_reset_after_seconds"] = *snapshot.SecondaryResetAfterSeconds
	}
	if snapshot.SecondaryWindowMinutes != nil {
		updates["codex_secondary_window_minutes"] = *snapshot.SecondaryWindowMinutes
	}
	if snapshot.PrimaryOverSecondaryPercent != nil {
		updates["codex_primary_over_secondary_percent"] = *snapshot.PrimaryOverSecondaryPercent
	}
	updates["codex_usage_updated_at"] = baseTime.Format(time.RFC3339)

	if normalized := snapshot.Normalize(); normalized != nil {
		if normalized.Used5hPercent != nil {
			updates["codex_5h_used_percent"] = *normalized.Used5hPercent
		}
		if normalized.Reset5hSeconds != nil {
			updates["codex_5h_reset_after_seconds"] = *normalized.Reset5hSeconds
		}
		if normalized.Window5hMinutes != nil {
			updates["codex_5h_window_minutes"] = *normalized.Window5hMinutes
		}
		if normalized.Used7dPercent != nil {
			updates["codex_7d_used_percent"] = *normalized.Used7dPercent
		}
		if normalized.Reset7dSeconds != nil {
			updates["codex_7d_reset_after_seconds"] = *normalized.Reset7dSeconds
		}
		if normalized.Window7dMinutes != nil {
			updates["codex_7d_window_minutes"] = *normalized.Window7dMinutes
		}
		if reset5hAt := codexResetAtRFC3339(baseTime, normalized.Reset5hSeconds); reset5hAt != nil {
			updates["codex_5h_reset_at"] = *reset5hAt
		}
		if reset7dAt := codexResetAtRFC3339(baseTime, normalized.Reset7dSeconds); reset7dAt != nil {
			updates["codex_7d_reset_at"] = *reset7dAt
		}
	}

	return updates
}

// updateCodexUsageSnapshot saves the Codex usage snapshot to account's Extra field
func (s *OpenAIGatewayService) updateCodexUsageSnapshot(ctx context.Context, accountID int64, snapshot *OpenAICodexUsageSnapshot) {
	if snapshot == nil {
		return
	}
	if s == nil || s.accountRepo == nil {
		return
	}

	now := time.Now()
	updates := buildCodexUsageExtraUpdates(snapshot, now)
	if len(updates) == 0 {
		return
	}
	if !s.getCodexSnapshotThrottle().Allow(accountID, now) {
		return
	}

	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.accountRepo.UpdateExtra(updateCtx, accountID, updates)
	}()
}

func (s *OpenAIGatewayService) UpdateCodexUsageSnapshotFromHeaders(ctx context.Context, accountID int64, headers http.Header) {
	if accountID <= 0 || headers == nil {
		return
	}
	if snapshot := ParseCodexRateLimitHeaders(headers); snapshot != nil {
		s.updateCodexUsageSnapshot(ctx, accountID, snapshot)
	}
}

func getOpenAIReasoningEffortFromReqBody(reqBody map[string]any) (value string, present bool) {
	if reqBody == nil {
		return "", false
	}

	// Primary: reasoning.effort
	if reasoning, ok := reqBody["reasoning"].(map[string]any); ok {
		if effort, ok := reasoning["effort"].(string); ok {
			return normalizeOpenAIReasoningEffort(effort), true
		}
	}

	// Fallback: some clients may use a flat field.
	if effort, ok := reqBody["reasoning_effort"].(string); ok {
		return normalizeOpenAIReasoningEffort(effort), true
	}

	return "", false
}

func deriveOpenAIReasoningEffortFromModel(model string) string {
	if strings.TrimSpace(model) == "" {
		return ""
	}

	modelID := strings.TrimSpace(model)
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		modelID = parts[len(parts)-1]
	}

	parts := strings.FieldsFunc(strings.ToLower(modelID), func(r rune) bool {
		switch r {
		case '-', '_', ' ':
			return true
		default:
			return false
		}
	})
	if len(parts) == 0 {
		return ""
	}

	return normalizeOpenAIReasoningEffort(parts[len(parts)-1])
}

func extractOpenAIRequestMetaFromBody(body []byte) (model string, stream bool, promptCacheKey string) {
	if len(body) == 0 {
		return "", false, ""
	}

	model = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	stream = gjson.GetBytes(body, "stream").Bool()
	promptCacheKey = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	return model, stream, promptCacheKey
}

// normalizeOpenAIPassthroughOAuthBody 闂傚倷娴囬褏鎹㈤幇顔藉床闁归偊鍎靛☉妯锋斀閻庯綆浜為崝锕€顪冮妶鍡楀潑闁稿鎸剧槐鎺旀嫚瀹割喖鍓冲┑?OAuth 闂傚倷娴囧畷鍨叏閺夋嚚娲敇閵忕姷鍝楅梻渚囧墮缁夌敻宕曢幋锔界厽婵°倐鍋撻柣妤€锕ラ崚濠囧箻椤旂晫鍘甸梺璇″瀻閸涱喗鍠栭梻浣告惈濡厽绂嶉鍫濊摕闁斥晛鍟刊鎾煕閿旇骞愭俊顐熸櫊閺岋綁鎮╅崘鎻掝潕闂佸摜濮甸悧鏇㈩敋閿濆鏁婃繛鍡欏亾閳诲矂姊绘担鐑樺殌闁逞屽墰閸犲孩绂嶈ぐ鎺撶厵闁荤喐婢橀顓熴亜閵忥紕鈽夋い顐ｇ箞椤㈡鍩€椤掑倻绀婃慨妞诲亾婵﹦绮幏鍛嫚閳ュ啿濡奸梻浣告憸婵敻鎯勯鐐参ュ〒姘ｅ亾妤犵偛顑夐幃娆戝枈濡崵浜欓梻鍌欒兌缁垶鎳濇ィ鍐炬晪妞ゆ挾濮烽悳?
// 1) 闂傚倸鍊风粈渚€骞夐敍鍕殰闁绘劕顕粻楣冩煃瑜滈崜姘辨崲?ChatGPT internal API 濠电姷鏁搁崑鐐哄垂閸洖绠伴柛婵勫劤閻捇鏌ｉ姀銏╃劸闁哄鐒﹂妵鍕即濡も偓娴滄儳螖閻橀潧浠﹂柛鏃€鐗犻獮蹇涘川鐎涙ê鈧粯淇婇姘倯婵炲牏顭堥埞鎴︽晬閸曨偂鏉梺绋匡攻缁嬫垿鍩㈤弬搴撴闁靛繆鈧啿澹?Responses 闂傚倸鍊风粈渚€骞夐敓鐘冲仭闁靛鏅涚壕鍦喐閻楀牆绗掓慨?// 2) store=false 3) 闂?compact 濠电姷鏁搁崕鎴犲緤閽樺娲晜閻愵剙搴婇梺鍛婂姦娴滄牠宕?stream=true闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟鐑橆殕閸庢鏌ｆ惔鈶芥攽act 闂備浇顕х€涒晠顢欓弽顓炵獥闁哄稁鍘肩粻瑙勩亜閹板墎鐣遍柡?stream=false
func normalizeOpenAIPassthroughOAuthBody(body []byte, compact bool) ([]byte, bool, error) {
	if len(body) == 0 {
		return body, false, nil
	}

	normalized := body
	changed := false

	for _, field := range openAIChatGPTInternalUnsupportedFields {
		if value := gjson.GetBytes(normalized, field); !value.Exists() {
			continue
		}
		next, err := sjson.DeleteBytes(normalized, field)
		if err != nil {
			return body, false, fmt.Errorf("normalize passthrough body delete %s: %w", field, err)
		}
		normalized = next
		changed = true
	}

	if compact {
		if store := gjson.GetBytes(normalized, "store"); store.Exists() {
			next, err := sjson.DeleteBytes(normalized, "store")
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body delete store: %w", err)
			}
			normalized = next
			changed = true
		}
		if stream := gjson.GetBytes(normalized, "stream"); stream.Exists() {
			next, err := sjson.DeleteBytes(normalized, "stream")
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body delete stream: %w", err)
			}
			normalized = next
			changed = true
		}
	} else {
		if store := gjson.GetBytes(normalized, "store"); !store.Exists() || store.Type != gjson.False {
			next, err := sjson.SetBytes(normalized, "store", false)
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body store=false: %w", err)
			}
			normalized = next
			changed = true
		}
		if stream := gjson.GetBytes(normalized, "stream"); !stream.Exists() || stream.Type != gjson.True {
			next, err := sjson.SetBytes(normalized, "stream", true)
			if err != nil {
				return body, false, fmt.Errorf("normalize passthrough body stream=true: %w", err)
			}
			normalized = next
			changed = true
		}
	}

	return normalized, changed, nil
}

func detectOpenAIPassthroughInstructionsRejectReason(reqModel string, body []byte) string {
	model := strings.ToLower(strings.TrimSpace(reqModel))
	if !strings.Contains(model, "codex") {
		return ""
	}

	instructions := gjson.GetBytes(body, "instructions")
	if !instructions.Exists() {
		return "instructions_missing"
	}
	if instructions.Type != gjson.String {
		return "instructions_not_string"
	}
	if strings.TrimSpace(instructions.String()) == "" {
		return "instructions_empty"
	}
	return ""
}

func extractOpenAIReasoningEffortFromBody(body []byte, requestedModel string) *string {
	reasoningEffort := strings.TrimSpace(gjson.GetBytes(body, "reasoning.effort").String())
	if reasoningEffort == "" {
		reasoningEffort = strings.TrimSpace(gjson.GetBytes(body, "reasoning_effort").String())
	}
	if reasoningEffort != "" {
		normalized := normalizeOpenAIReasoningEffort(reasoningEffort)
		if normalized == "" {
			return nil
		}
		return &normalized
	}

	value := deriveOpenAIReasoningEffortFromModel(requestedModel)
	if value == "" {
		return nil
	}
	return &value
}

func extractOpenAIServiceTier(reqBody map[string]any) *string {
	if reqBody == nil {
		return nil
	}
	raw, ok := reqBody["service_tier"].(string)
	if !ok {
		return nil
	}
	return normalizeOpenAIServiceTier(raw)
}

func extractOpenAIServiceTierFromBody(body []byte) *string {
	if len(body) == 0 {
		return nil
	}
	return normalizeOpenAIServiceTier(gjson.GetBytes(body, "service_tier").String())
}

func normalizeOpenAIServiceTier(raw string) *string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return nil
	}
	if value == "fast" {
		value = "priority"
	}
	// 闂傚倸鍊峰ù鍥Υ閳ь剟鏌涚€ｎ偅宕岄柡宀€鍠撻幉鎾礋椤愩埄娼曠紓?OpenAI 闂傚倷娴囬褍顫濋敃鍌︾稏濠㈣泛瀛╅幊宀勬⒒娴ｅ憡鎯堥柣顒€銈稿畷浼村冀瑜滈崵鏇熴亜閹烘垵顏╅柣鎰躬閺屾洘绻濊箛鏇炵煗闂佺顑嗛幑鍥х暦閹烘垟鏀﹂柟顖嗗倸顥氶梺鑽ゅ枑閻熴儳鈧凹鍣ｉ幃锟狀敍濠婂懐锛滈梺缁橈耿濞佳囧几濞戙垺鐓欐い鏍ㄨ壘閺嗭絿鈧娲滈崰鏍€佸Δ鍛劦妞ゆ帒瀚ㄩ埀顒€鍊垮濠氬Ψ閿旀儳骞愰梻浣稿閸嬪懐鎹㈠鍛傦綁骞栨担鍦幍濡炪倖鏌ㄩ妶鎼侇敁濡ゅ懏鐓?tier 闂傚倸鍊烽懗鍫曗€﹂崼銉︽櫇闁挎洖鍊歌繚闁诲函缍嗛崜姘讹綖閺囥垺鐓熼柕濞垮劚閸樼帵ority/flex/auto/default/scale闂?	// 闂?Codex 闂傚倷娴囬褎顨ラ崫銉т笉鐎广儱顦崹鍌涚箾瀹割喕绨婚柡鍕╁劜缁绘盯骞嬮悙瀵告闂佸憡顨嗙喊宥夊Φ閸曨垰鍐€妞ゆ劦婢€濞岊亞绱撴担鐤唹闁哄懏绻堥獮澶愬箹娴ｅ摜楠囬梺鍓茬厛閸犳盯宕洪悙鐑樷拺缁绢厼鎳庢禍褰掓煕韫囨棑鑰跨€殿噮鍋勯鍏煎緞婵犲啫绁堕梻浣告惈鐎氼喚绮嬬€规敍 闂傚倸鍊风粈渚€骞夐敓鐘冲仭妞ゆ牗绋撻々鍙夌節婵犲倹顥撳ù?priority 闂?flex闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸庢銆掑锝呬壕闂?codex-rs/core/src/client.rs闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆″眰鍔戦崺鈧い鎺戝€荤壕?	// 濠电姷鏁搁崑鐘诲箵椤忓棗绶ら柦妯猴級濞戞﹩鐓ラ柛顐ｇ箘椤ρ冣攽閳藉棗鐏犻柛姘儐閸掑﹥绺介崨濠勫帗闂佸憡绻傜€氼剟寮抽悙鐢电＜缂備降鍨归悘锔锯偓?OpenAI SDK 闂傚倸鍊烽悞锕傛儑瑜版帒绀夌€光偓閳ь剟鍩€椤掍礁鍤柛锝忕到椤曪綁顢曢敃鈧洿婵犮垼娉涢敃锕傤敇濞差亝鈷戠紓浣姑悘杈╂偖濞嗘挻鐓曢悗锝庡亝瀹曞嫮绱掗悩宕団姇闁诡垱妫冮弫鎰板炊瑜夐崑?auto/default/scale 濠电姷鏁搁崑娑㈩敋椤撶喐鍙忓ù鍏兼綑绾惧潡鏌熷▓鍨灓闁活厽宀搁弻娑㈩敃閿濆棛顦ョ紓浣哄閸ｏ綁寮婚悢鐓庣畾鐟滃秹寮抽柆宥嗙厽?闂傚倷娴囧畷鍨叏閹绢噮鏁勯柛娑欐綑閻ゎ噣鏌熼幆鏉啃撻柛搴★攻閵囧嫰寮介顫捕閺?	// 闂傚倸鍊烽悞锕€顪冮幐搴ｎ洸闁绘劕鎼粣妤呮煙闁箑澧鹃柤鐗堝閵囧嫰骞橀崡鐐典痪闂佺瀛╅〃濠囧蓟閺囩喎绶為柛顐ｇ箓婵姊虹粙娆惧剱闁圭懓娲ら～蹇涙嚒閵堝倸浜鹃梻鍫熺⊕閹叉悂鏌ｉ幘鎼疁闁哄备鍓濆鍕幢濡崵褰嗛梻浣告惈鐞氼偊宕濆畝鈧崣?nil闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸庢柨鈹戦崒婊庣劸闁?normalizeResponsesBodyServiceTier 濠?body 濠电姷鏁搁崑鐐哄垂閸洖绠归柍鍝勬噹閸屻劌鈹戦崒婊庣劸闁搞劌鍊婚幉鎼佹偋閸繄鐟查梺鎶芥敱濡啴寮婚弴銏犻唶婵犻潧妫欏▓婊堟⒑?
	switch value {
	case "priority", "flex", "auto", "default", "scale":
		return &value
	default:
		return nil
	}
}

// OpenAIFastBlockedError indicates a request was rejected by the OpenAI fast
// policy (action=block). Mirrors BetaBlockedError on the Claude side.
type OpenAIFastBlockedError struct {
	Message string
}

func (e *OpenAIFastBlockedError) Error() string { return e.Message }

// evaluateOpenAIFastPolicy returns the action and error message that should be
// applied for a request with the given account/model/service_tier. When the
// policy service is unavailable or no rule matches, it returns
// (BetaPolicyActionPass, "") so callers can short-circuit safely.
//
// Matching rules:
//   - Scope filters by account type (all / oauth / apikey / bedrock)
//   - ServiceTier must be empty (= any), "all", or equal the normalized tier
//   - ModelWhitelist narrows the rule to specific models; FallbackAction
//     handles the non-matching case (default: pass)
//
// Difference from Claude BetaPolicy (keeps first-match short-circuit):
//   - BetaPolicy works on a token set from anthropic-beta; filter can accumulate
//     a set and block uses first-match.
//   - OpenAI fast policy works on a single service_tier field; filter deletes it.
//     A request only carries one service_tier, so tier matching is mutually exclusive.
//     When model whitelists overlap, admins can use rule order to express intent.
//     Therefore this uses first-match semantics.
func (s *OpenAIGatewayService) evaluateOpenAIFastPolicy(ctx context.Context, account *Account, model, serviceTier string) (action, errMsg string) {
	if s == nil || s.settingService == nil {
		return BetaPolicyActionPass, ""
	}
	tier := strings.ToLower(strings.TrimSpace(serviceTier))
	if tier == "" {
		return BetaPolicyActionPass, ""
	}
	settings := openAIFastPolicySettingsFromContext(ctx)
	if settings == nil {
		fetched, err := s.settingService.GetOpenAIFastPolicySettings(ctx)
		if err != nil || fetched == nil {
			return BetaPolicyActionPass, ""
		}
		settings = fetched
	}
	return evaluateOpenAIFastPolicyWithSettings(settings, account, model, tier)
}

// evaluateOpenAIFastPolicyWithSettings is the pure-function core extracted so
// long-lived sessions (e.g. WS) can prefetch settings once and avoid hitting
// the settingService on every frame. See WSSession entry and
// openAIFastPolicySettingsFromContext for the caching glue.
func evaluateOpenAIFastPolicyWithSettings(settings *OpenAIFastPolicySettings, account *Account, model, tier string) (action, errMsg string) {
	if settings == nil {
		return BetaPolicyActionPass, ""
	}
	isOAuth := account != nil && account.IsOAuth()
	isBedrock := account != nil && account.IsBedrock()
	for _, rule := range settings.Rules {
		if !betaPolicyScopeMatches(rule.Scope, isOAuth, isBedrock) {
			continue
		}
		ruleTier := strings.ToLower(strings.TrimSpace(rule.ServiceTier))
		if ruleTier != "" && ruleTier != OpenAIFastTierAny && ruleTier != tier {
			continue
		}
		eff := BetaPolicyRule{
			Action:               rule.Action,
			ErrorMessage:         rule.ErrorMessage,
			ModelWhitelist:       rule.ModelWhitelist,
			FallbackAction:       rule.FallbackAction,
			FallbackErrorMessage: rule.FallbackErrorMessage,
		}
		return resolveRuleAction(eff, model)
	}
	return BetaPolicyActionPass, ""
}

// openAIFastPolicyCtxKey 闂?context 濠电姷鏁搁崑鐐哄垂閸洖绠归柍鍝勬噹閻鏌涢幇闈涙灁闁逞屽墮閸婂潡宕洪敓鐘插窛妞ゆ挾濮烽崢顒勬⒒娴ｅ憡鎯堢紒瀣╃窔瀹曟垿宕ㄧ€涙ê浠?OpenAIFastPolicySettings 缂傚倸鍊搁崐鎼佸磹閹间礁纾圭憸鐗堝笒缁犱即鏌熼梻瀵稿妽闁?// 闂傚倸鍊搁崐鐑芥倿閿曞偆鏁勬繛鍡樻尭闂傤垱銇勯弽銊х煁妞ゎ偅娲樼换婵嬫濞戞瑥绐涘銈傛櫇閸忔﹢寮诲☉妯锋闁告鍋熸禒顓㈡⒑閸濆嫭顥滅紒缁樺姍濠€?WebSocket 闂傚倸鍊搁崐鎼佸磹閻㈢纾婚柟鐐墯閻斿棝鎮归搹鐟板妺闁规彃鎽滅槐鎺楀箛椤撗勭杹闂佽鍠栫紞濠傜暦閸洖惟鐟滃繘鎮鹃悽鍛娾拺闁告縿鍎辨禒婊堟煟鎺抽崝宀勵敋閵夆晛绀嬫い鎰靛亝閸嶉潧顪冮妶鍡楃瑐缂佲偓娓氣偓閿濈偤骞樼€靛摜鐦堥梺闈涢獜缁插墽娑甸崜褏纾煎璺猴功閸╋絿鈧鍣崑濠冧繆閼搁潧绶炲┑鐘叉啗閺囥垺鐓熼煫鍥ㄦ礀娴犫晠鏌涚€ｎ偄濮嶆い銏℃瀹曪繝鎮樼拠鑼Ш闁诡喒鏅犲畷褰掝敃閿濆洤娑ч梻鍌欑椤撲粙寮堕崹顔夹曢梻浣筋嚃閸ｏ絿绮婚弽銊ょ箚闁绘垼妫勫敮闂佹寧娲嶉崑鎾寸箾閹炬潙濮嶆慨濠呮缁辨帒螣鐠囪尙妯傞柣搴＄畭閸庡崬煤閵娾敡鍥敋閳ь剙顫忕紒妯肩懝闁搞儜鍐炬交闂備浇顕х换鎴犳崲閸繄鏆﹂柕蹇嬪€曞洿闂佸憡绋戦幊蹇曟暜閻旇櫣涓嶆繛鎴欏灩缁犲鏌℃径瀣劸婵?DB 闂傚倸鍊风粈渚€骞夐敍鍕煓闁圭儤顨呴崹鍌涚節闂堟侗鍎愰柛銈呯墦閺岀喐娼忛崜褏鏆犻弶?//
// Trade-off闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟鐑樺焾濞尖晠鏌曟繛鐐珕闁稿鍊濋弻鏇熷緞閸℃ɑ鐝曢梺绋款儌閺呯娀寮婚弴鐔风窞闁割偅绻傛慨搴ｇ磽閸屾氨小缂佽埖宀稿濠氭晸閻樿尙鍊為梺鍐叉惈閸熶即鎯侀崼銏㈢＝濞达綀顕栧▓鏃€銇勯敂钘夘棆闁逞屽墰閺佹悂宕㈣閿濈偛鈹戠€ｅ灚鏅㈤梺閫炲苯澧撮柕鍡楀暙閳藉濮€閿涘嫬骞堟繝鐢靛█濞佳兾涘☉妯兼懃缂傚倸鍊搁崐鍝ョ矓閹绢噮鏁勫璺猴功閺?WS session闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟杈鹃檮閸嬪鏌涢埄鍐ㄦ惛濞存粌婀遍幉鎼佸箣閿曗偓妗呴梺鍦濠㈡娑甸埀顒勬⒑缂佹ê濮囬柣掳鍔戝畷鏇㈠Ψ閳哄倵鎷?session闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟娆¤娲、姗€濮€閻橀潧濮︽俊鐐€栧濠氬磻閹剧粯鐓曞┑鐘插濞呮洟鏌熼獮鍨伈鐎殿喕绮欓、姗€鎮欓幓鎺撳皨?// 闂傚倸鍊风粈渚€骞栭锔藉亱闁糕剝铔嬮崶銊ヮ嚤闁哄鍨归ˇ顓㈡⒑缂佹﹩娈旈柣妤€绻橀崺銏ゅ籍閸繄楠囬梺缁樺姈濞兼瑩藟鐎ｎ偁浜?闂傚倸鍊烽懗鍫曞磻閵娾晛纾块柡灞诲劚閽冪喖鏌ㄩ悢鍝勑㈤柣?闂傚倷娴囬褔鏌婇敐鍜佺劷鐟滃繒妲愰悙瀵哥瘈闁搞儜鍕发闂佸搫顦悧鍕礉瀹€鍕垫晪閺夊牄鍔嶉崣蹇斾繆椤栨繃顏犻柨娑樼У娣囧﹪顢涘鍗炩叺闂佸搫澶囬崜婵嗩嚗閸曨垰閱囨繝闈涙琚氬┑锛勫亼閸婃牕煤瀹ュ鐤鹃柣妯款嚙閺?缂傚倸鍊搁崐鐑芥倿閿斿墽鐭欓柟鐑橆殕閸嬪绻濇繝鍌滃婵鐓￠弻鐔兼焽閿曗偓楠炴﹢鏌嶇紒妯活棃闁哄苯绉烽¨渚€鏌涢幘璺烘灈妤犵偛锕ら埥澶婎潩閿濆懍澹曞┑鐐村灦椤忣喖顫濈捄铏癸紱?婵?缂傚倸鍊搁崐鐑芥倿閿曞倸钃熼柕濞炬櫓閺佸嫰鏌涘☉娆愮稇闁哄嫨鍎甸弻鏇＄疀鐎ｎ亖鍋撻弽顓炵；闁靛ň鏅滈悡鐔兼煛閸屾稑顕滈柛鐔哄仦缁?闂傚倸鍊风粈渚€骞栭鈷氭椽鏁冮崒姘鳖唶闂佽鍎煎Λ鍕⒒椤栫偞鐓曟繝闈涘閸旀粓鏌涢悢閿嬫儓闁靛棙甯掗～婵嬵敆婵犲倸顫犵紓鍌欒閸嬫挸顭跨捄鍙峰牆危?Claude
// BetaPolicy 闂?gin.Context 缂傚倸鍊搁崐鎼佸磹閹间礁纾圭憸鐗堝笒缁犱即鏌熼梻瀵稿妽闁稿鍊濋弻鏇熺節韫囨搩娲梺宕囩帛濞茬喖寮婚敐澶婃闁圭楠搁弳鍫㈢磼濡や礁鐏存慨濠冩そ瀹曘劍绻濋崨顓ф闂備礁鎼悧婊堝礈閻旂厧绠栫憸鏃堝蓟閸℃鍚嬮柛娑卞灣閸橆剟姊绘担鍛婃儓缂佸绶氬畷婊冣枎閹炬潙鈧爼鏌涢弴銊ョ仭闁绘挸绻橀弻娑㈠焺閸愮偓鐣堕梺閫炲苯澧柣蹇旂箞婵℃挳骞掑Δ鈧粈鍫澝归敐鍥ㄥ殌闁?hot-reload 闂傚倸鍊风粈渚€骞栭锕€鐤い鎰剁稻濞呯娀骞栭幖顓犲帥闁轰礁娲弻鐔兼⒒鐎电濡介梺绋款儑婵炩偓闁哄本鐩崺鍕礃閻愵剛鏆﹂梻?
// 闂傚倸鍊风粈渚€骞夐敓鐘冲仭妞ゆ牜鍋涢崹鍌涖亜閺嶃劎銆掓い鈺傜叀閺岀喖鎮滃鍡樼暦闂佺粯甯炵划顖炲箞閵娿儙鐔兼惞閻у摜绀婄紓鍌欒兌婵敻鎮ч悩鍙傛盯宕ㄩ幖顓熸櫆闂佺鏈銊︾妤ｅ啯鈷?session 闂備浇顕х€涒晠顢欓弽顓炵獥闁哄稁鍘肩粻瑙勩亜閹板墎鐣遍柡鍕╁劜娣囧﹪濡堕崨顔兼缂備讲鍋撻柛鈩冪⊕閳锋垿鏌涢…鎴濇珮闁稿孩鍔欓弻锝夊箻椤栨矮澹曢梻?
type openAIFastPolicyCtxKeyType struct{}

var openAIFastPolicyCtxKey = openAIFastPolicyCtxKeyType{}

// withOpenAIFastPolicyContext 闂傚倷娴囬褏鎹㈤幇顔藉床闁归偊鍎靛☉銏犵睄闁稿本绮屽畷銉╂煟鎼淬垻鈯曟い顓炴喘閵?settings 闂傚倸鍊搁…顒勫磻閸曨個褰掑磼閻愯尙锛涢梺绯曞墲缁嬫垿鎯屽Δ鍛閺夊牆澧介幃鑲╃棯閹佸仮闁哄瞼鍠撻埀顒佺⊕椤洨绮诲鈧弻锟犲幢閺囩偛绁梺?context闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟瀵稿仧闂勫嫰鏌￠崘銊モ偓褰掝敃娴犲鐓熼柕蹇曞Х娴犳盯鎮?ctx
// 闂傚倷娴囧畷鐢稿磻閻愮數鐭欓柟瀵稿Х閻捇鏌涢锝嗙闁?goroutine 濠电姷鏁搁崑鐐哄垂閸洖绠归柍鍝勬噹閸屻劑鏌熼鍡忓亾闁?evaluateOpenAIFastPolicy 濠电姷鏁告慨浼村垂閻撳簶鏋栨繛鎴炴皑閻捇鏌涢锝嗙闁哄绶氶弻鏇㈠醇濠垫劖笑閺?
func withOpenAIFastPolicyContext(ctx context.Context, settings *OpenAIFastPolicySettings) context.Context {
	if ctx == nil || settings == nil {
		return ctx
	}
	return context.WithValue(ctx, openAIFastPolicyCtxKey, settings)
}

func openAIFastPolicySettingsFromContext(ctx context.Context) *OpenAIFastPolicySettings {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(openAIFastPolicyCtxKey).(*OpenAIFastPolicySettings); ok {
		return v
	}
	return nil
}

// applyOpenAIFastPolicyToBody applies the OpenAI fast policy to a raw request
// body. When action=filter it removes the service_tier field; when
// action=block it returns (body, *OpenAIFastBlockedError). On pass it
// Rationale for normalize-on-pass: chat-completions / messages already normalize
// service_tier before calling this function. Passthrough and native /responses
// do not have that earlier step, so aliases like "fast" must be normalized here
// before reaching upstream.
func (s *OpenAIGatewayService) applyOpenAIFastPolicyToBody(ctx context.Context, account *Account, model string, body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	rawTier := gjson.GetBytes(body, "service_tier").String()
	if rawTier == "" {
		return body, nil
	}
	normTier := normalizedOpenAIServiceTierValue(rawTier)
	if normTier == "" {
		return body, nil
	}
	action, errMsg := s.evaluateOpenAIFastPolicy(ctx, account, model, normTier)
	switch action {
	case BetaPolicyActionBlock:
		msg := errMsg
		if msg == "" {
			msg = fmt.Sprintf("openai service_tier=%s is not allowed for model %s", normTier, model)
		}
		return body, &OpenAIFastBlockedError{Message: msg}
	case BetaPolicyActionFilter:
		trimmed, err := sjson.DeleteBytes(body, "service_tier")
		if err != nil {
			return body, fmt.Errorf("strip service_tier from body: %w", err)
		}
		return trimmed, nil
	default:
		// On pass, write aliases such as "fast" back as canonical values such as "priority".
		if normTier == rawTier {
			return body, nil
		}
		updated, err := sjson.SetBytes(body, "service_tier", normTier)
		if err != nil {
			return body, fmt.Errorf("normalize service_tier on pass: %w", err)
		}
		return updated, nil
	}
}

// writeOpenAIFastPolicyBlockedResponse writes a 403 JSON response for a
// request blocked by the OpenAI fast policy.
func writeOpenAIFastPolicyBlockedResponse(c *gin.Context, err *OpenAIFastBlockedError) {
	if c == nil || err == nil {
		return
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"type":    "permission_error",
			"message": err.Message,
		},
	})
}

// applyOpenAIFastPolicyToWSResponseCreate evaluates the OpenAI fast policy
// against a single client闂傚倸鍊烽悞锕傚礈濮樿泛纾婚柛娑卞枟閸欏繒鐥悙顒€袩tream WebSocket frame whose top-level
// "type"=="response.create". It mirrors the HTTP-side
// applyOpenAIFastPolicyToBody contract but operates on a Realtime/Responses
// WS payload:
//
//   - pass: keeps service_tier, normalizing aliases such as "fast" to "priority"
//   - filter: returns a copy with top-level service_tier removed
//   - block: returns (frame, *OpenAIFastBlockedError)
//
// Only frames whose "type" field strictly equals "response.create" are
// inspected/mutated. Any other frame type 闂?including the empty string 闂?// passes through untouched. The OpenAI Realtime client-event spec requires
// "type" to be set, so an empty type is treated as a malformed frame we do
// not police; the upstream is the source of truth for rejecting it.
//
// service_tier lives at the top level of response.create 闂?same as the
// Responses HTTP body shape (see openai_gateway_chat_completions.go:304 +
// extractOpenAIServiceTierFromBody at line 5593, and the test fixture at
// openai_ws_forwarder_ingress_session_test.go:402). We therefore only need
// to inspect / strip the top-level field; there is no nested form in the
// schema today.
//
// The caller is responsible for choosing the upstream model passed in 闂?// this helper does not re-derive it.
func (s *OpenAIGatewayService) applyOpenAIFastPolicyToWSResponseCreate(
	ctx context.Context,
	account *Account,
	model string,
	frame []byte,
) ([]byte, *OpenAIFastBlockedError, error) {
	if len(frame) == 0 {
		return frame, nil, nil
	}
	if !gjson.ValidBytes(frame) {
		return frame, nil, nil
	}
	frameType := strings.TrimSpace(gjson.GetBytes(frame, "type").String())
	// Strict match: only response.create is policy-checked. Empty / other
	// types pass through untouched so we never accidentally strip fields
	// from response.cancel, conversation.item.create, or any future
	// client-event the spec adds. The Realtime spec requires "type" on
	// every client event, so an empty type is malformed input 闂?let the
	// upstream reject it rather than guessing at our layer.
	if frameType != "response.create" {
		return frame, nil, nil
	}
	rawTier := gjson.GetBytes(frame, "service_tier").String()
	if rawTier == "" {
		return frame, nil, nil
	}
	normTier := normalizedOpenAIServiceTierValue(rawTier)
	if normTier == "" {
		return frame, nil, nil
	}
	action, errMsg := s.evaluateOpenAIFastPolicy(ctx, account, model, normTier)
	switch action {
	case BetaPolicyActionBlock:
		msg := errMsg
		if msg == "" {
			msg = fmt.Sprintf("openai service_tier=%s is not allowed for model %s", normTier, model)
		}
		return frame, &OpenAIFastBlockedError{Message: msg}, nil
	case BetaPolicyActionFilter:
		trimmed, err := sjson.DeleteBytes(frame, "service_tier")
		if err != nil {
			return frame, nil, fmt.Errorf("strip service_tier from ws frame: %w", err)
		}
		return trimmed, nil, nil
	default:
		if normTier == rawTier {
			return frame, nil, nil
		}
		updated, err := sjson.SetBytes(frame, "service_tier", normTier)
		if err != nil {
			return frame, nil, fmt.Errorf("normalize service_tier in ws frame: %w", err)
		}
		return updated, nil, nil
	}
}

// newOpenAIFastPolicyWSEventID returns a Realtime-style event_id for a
// server-emitted error event. Matches the loose "evt_<rand>" convention used
// by upstream Realtime servers; the exact value is not load-bearing and is
// only required for client-side log correlation. We reuse the existing
// google/uuid dependency rather than pulling a new one.
func newOpenAIFastPolicyWSEventID() string {
	id, err := uuid.NewRandom()
	if err != nil {
		// Extremely unlikely; fall back to a fixed prefix so the field is
		// still non-empty and the schema stays self-consistent.
		return "evt_openai_fast_policy"
	}
	// Strip dashes so it visually matches "evt_<hex>" rather than UUID v4
	// canonical form, mirroring what real Realtime traces look like.
	return "evt_" + strings.ReplaceAll(id.String(), "-", "")
}

// buildOpenAIFastPolicyBlockedWSEvent renders an OpenAI Realtime/Responses
// style "error" event payload for a request blocked by the OpenAI fast
// policy. The shape mirrors Realtime error events as observed in upstream
// traces and per the spec's server "error" event:
//
//	{
//	  "event_id": "evt_<random>",
//	  "type": "error",
//	  "error": {
//	    "type": "invalid_request_error",
//	    "code": "policy_violation",
//	    "message": "..."
//	  }
//	}
//
// event_id lets clients correlate the rejection in their logs; "code" gives
// programmatic clients a stable identifier (HTTP-side equivalent is the
// 403 permission_error JSON body).
func buildOpenAIFastPolicyBlockedWSEvent(err *OpenAIFastBlockedError) []byte {
	if err == nil {
		return nil
	}
	eventID := newOpenAIFastPolicyWSEventID()
	payload, mErr := json.Marshal(map[string]any{
		"event_id": eventID,
		"type":     "error",
		"error": map[string]any{
			"type":    "invalid_request_error",
			"code":    "policy_violation",
			"message": err.Message,
		},
	})
	if mErr != nil {
		// Fallback to a minimal hand-rolled payload; Marshal of the literal
		// shape above should never fail in practice.
		return []byte(`{"event_id":"` + eventID + `","type":"error","error":{"type":"invalid_request_error","code":"policy_violation","message":"openai fast policy blocked this request"}}`)
	}
	return payload
}

func sanitizeEmptyBase64InputImagesInOpenAIBody(body []byte) ([]byte, bool, error) {
	if len(body) == 0 || !bytes.Contains(body, []byte(`"image_url"`)) || !bytes.Contains(body, []byte(`base64,`)) {
		return body, false, nil
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return body, false, fmt.Errorf("sanitize request body: %w", err)
	}
	if !sanitizeEmptyBase64InputImagesInOpenAIRequestBodyMap(reqBody) {
		return body, false, nil
	}
	normalized, err := json.Marshal(reqBody)
	if err != nil {
		return body, false, fmt.Errorf("serialize sanitized request body: %w", err)
	}
	return normalized, true, nil
}

func sanitizeEmptyBase64InputImagesInOpenAIRequestBodyMap(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	input, ok := reqBody["input"]
	if !ok {
		return false
	}
	normalizedInput, changed := sanitizeEmptyBase64InputImagesInOpenAIInput(input)
	if !changed {
		return false
	}
	reqBody["input"] = normalizedInput
	return true
}

func sanitizeEmptyBase64InputImagesInOpenAIInput(input any) (any, bool) {
	items, ok := input.([]any)
	if !ok {
		return input, false
	}

	normalizedItems := make([]any, 0, len(items))
	changed := false
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			normalizedItems = append(normalizedItems, item)
			continue
		}
		if shouldDropEmptyBase64InputImagePart(itemMap) {
			changed = true
			continue
		}
		content, ok := itemMap["content"]
		if !ok {
			normalizedItems = append(normalizedItems, itemMap)
			continue
		}
		parts, ok := content.([]any)
		if !ok {
			normalizedItems = append(normalizedItems, itemMap)
			continue
		}

		normalizedParts := make([]any, 0, len(parts))
		itemChanged := false
		for _, part := range parts {
			if shouldDropEmptyBase64InputImagePart(part) {
				changed = true
				itemChanged = true
				continue
			}
			normalizedParts = append(normalizedParts, part)
		}
		if itemChanged {
			if len(normalizedParts) == 0 {
				continue
			}
			itemMap["content"] = normalizedParts
		}
		normalizedItems = append(normalizedItems, itemMap)
	}
	if !changed {
		return input, false
	}
	return normalizedItems, true
}

func shouldDropEmptyBase64InputImagePart(part any) bool {
	partMap, ok := part.(map[string]any)
	if !ok {
		return false
	}
	typeValue, _ := partMap["type"].(string)
	if strings.TrimSpace(typeValue) != "input_image" {
		return false
	}
	imageURL, _ := partMap["image_url"].(string)
	return isEmptyBase64DataURI(imageURL)
}

func isEmptyBase64DataURI(raw string) bool {
	if !strings.HasPrefix(raw, "data:") {
		return false
	}
	rest := strings.TrimPrefix(raw, "data:")
	semicolonIdx := strings.Index(rest, ";")
	if semicolonIdx < 0 {
		return false
	}
	rest = rest[semicolonIdx+1:]
	if !strings.HasPrefix(rest, "base64,") {
		return false
	}
	return strings.TrimSpace(strings.TrimPrefix(rest, "base64,")) == ""
}

func getOpenAIRequestBodyMap(c *gin.Context, body []byte) (map[string]any, error) {
	if c != nil {
		if cached, ok := c.Get(OpenAIParsedRequestBodyKey); ok {
			if reqBody, ok := cached.(map[string]any); ok && reqBody != nil {
				return reqBody, nil
			}
		}
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	if c != nil {
		c.Set(OpenAIParsedRequestBodyKey, reqBody)
	}
	return reqBody, nil
}

func releaseOpenAIParsedRequestBody(c *gin.Context) {
	if c == nil {
		return
	}
	delete(c.Keys, OpenAIParsedRequestBodyKey)
}

func extractOpenAIReasoningEffort(reqBody map[string]any, requestedModel string) *string {
	if value, present := getOpenAIReasoningEffortFromReqBody(reqBody); present {
		if value == "" {
			return nil
		}
		return &value
	}

	value := deriveOpenAIReasoningEffortFromModel(requestedModel)
	if value == "" {
		return nil
	}
	return &value
}

func normalizeOpenAIReasoningEffort(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}

	// Normalize separators for "x-high"/"x_high" variants.
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)

	switch value {
	case "none", "minimal":
		return ""
	case "low", "medium", "high":
		return value
	case "xhigh", "extrahigh":
		return "xhigh"
	default:
		// Only store known effort levels for now to keep UI consistent.
		return ""
	}
}
