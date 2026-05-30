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
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
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
	openaiStickySessionTTL = time.Hour // 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄨ崵妲愰幒妤€绠涙い鎾楀嫮鏆﹀┑鐑囩到濞层倝鏁冮鍛箚闁割偅娲栧婵囥亜閹炬鍊诲Λ顖炴⒒閸屾瑨鍏岄柟铏崌瀹曨垶宕稿Δ浣稿壒闂佺鏈崙鐟邦焽閺嶎偆纾藉ù锝堫嚃閻掍粙鏌ｉ悢椋庣Ш闁哄苯绉烽¨渚€鏌涢幘璺烘灈鐎规洘妞介崺鈧い鎺嶉檷娴滄粓鏌熼悜妯虹仴闁逞屽墮閹诧紕绮嬮幒鎴叆闁割偆鍠撻崢鎯р攽閻愯泛钄兼い鏇嗗洦鍊堕柨鏃堟暜閸嬫挾鎲撮崟顒傤槰缂備浇顕ч悧鎾诲Υ娴ｇ硶妲堥柕蹇娾偓鏂ュ亾閻戣姤鍊甸柣銏☆問閻掔偓顨ラ悙鑼ⅵ婵﹨娅ｉ幏鐘绘嚑椤掑偆鍞洪梻浣侯焾椤戝棛绮欓幒妤€鐤鹃柤鍝ユ暩椤╃兘鎮楅敐搴′簻闁告挸缍婇幃妤呭礂婢跺﹣澹曢梻浣告啞濞诧箓宕滃☉銏犲偍闁芥ê顦弨浠嬫煟閹邦厽缍戦柣蹇ョ畵閺岋絽螖娴ｅ湱顦伴悗瑙勬礃缁诲牓寮崘顔肩＜婵炴垶鑹鹃獮鎰版煟鎼粹€冲辅闁稿鎹囬幃妤呮晲鎼粹€愁潾閻炴熬绠撳缁樻媴缁涘娈愰梺鍝ュУ鐢€愁嚕閺屻儲鍋愰柤纰卞墯濞堥箖姊洪幐搴ｇ畵婵☆偅顨堝▎銏ゆ倷濞村鏂€闂佺粯蓱瑜板啴顢楅姀銏″仏闁冲搫鍟扮壕浠嬫煕鐏炴崘澹橀柍褜鍓氶幃鍌氱暦閹邦剛鏆嬮柟鍐诧工缂嶅﹤鐣烽崼鏇炍╅柨婵嗗閻帡姊绘担鍝ョШ婵☆偉娉曠划鍫熺瑹閳ь剟骞冮垾婢勭喓浜搁弽褌澹?
	codexCLIUserAgent      = "codex_cli_rs/0.125.0"
	// Maximum header value length included in Codex CLI debug logs.
	codexCLIOnlyHeaderValueMaxBytes = 256

	// OpenAIParsedRequestBodyKey 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鎮㈤崗灏栨嫽闁诲酣娼ф竟濠偽ｉ鍓х＜闁绘劦鍓欓崝銈嗙節閳ь剟鏌嗗鍛枀闂佸綊妫块悞锕傚磻鐎ｎ喗鐓曟い鎰剁悼缁犳﹢鏌ｉ悢鏉戝缂佽鲸鎸婚幏鍛村传閸曟埊绻濋弻娑樜旀担绯曟灆閻庢鍠栭…鐑藉箖閵忋倕绀傞悘蹇旂墬鐎氫粙姊虹拠鍙夋崳闁轰焦鎮傞垾锕傚醇閻斿墎绠氭繛瀵稿Т椤戝棝鍩?handler 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾剁磽娴ｅ搫校婵犮垺锕㈤崺鐐哄箣閿旇棄浜归悗瑙勬礀濞村倿寮抽敓鐘斥拺缂佸娼￠妤冪磼婢跺本鏆╅柟骞垮灩閳规垿宕堕埡鍐闂備胶顭堥張顒傚垝瀹€鍕┾偓鍌炴惞閸︻厾锛濇繛杈剧稻瑜板啯绂嶆ィ鍐┾拺闁告稑锕ゆ慨锕傛煕閻樻剚娈滈柟顔欍倗鐤€闁圭虎鍨遍弬鈧梻浣虹帛閸旀浜稿▎鎰浄闁靛繈鍊栭悡鏇㈡煛閸愶絽浜剧紓渚囧枛缁夎淇婇悽绋跨妞ゆ牗姘ㄩ鎺楁⒑缂佹ê濮屾俊顐㈠缁傛帡顢欑亸鏍ㄦ杸濡炪倖姊婚妴瀣绩缂佹ü绻嗛柣鎰閻瑦顨ラ悙璇ц含闁轰焦鎹囬幃鈺呭矗閸屽啫娲﹂悡鏇熺箾閹存繂鑸归柡瀣ㄥ€濋弻宥堫檨闁告挾鍠撶划濠氬冀瑜滃鏍ㄧ箾瀹割喕绨奸柛銈嗗浮閺屾洟宕煎┑鍥ф闂佽绻愬Λ娆戞崲濞戞埃鍋撳☉娆樼劷闁活厹鍊濋弻娑㈠箻鐎靛憡鍣板銈冨灪瀹€鍛婁繆閻戣姤鏅滈柛鎾楀懏顫岄梻鍌欑閹诧繝宕濊缁骞嬮悙娈挎锤濠电娀娼ч鍡涘煕閹达附鐓曟繛鎴烇公瀹搞儱鈹戦鐟颁壕闂傚倷绀侀幖顐﹀箠韫囨稒鍎庢い鏍仜閺勩儵鏌嶈閸撴岸濡甸崟顖氱闁瑰瓨绻嶆禒鎯р攽閻愭潙姣嗛柛銉厛濮婃寧绻濋姀锝呯厫闁告梹鐗犻幃锟犳偄闂€鎰畾濡炪倖鐗楃划宀勫礉鐎ｎ喗鐓冪紓浣股戠粈瀣煛鐏炲墽鈯曢柟顖涙婵偓闁绘ê妯婂姘舵⒒娴ｅ憡鎯堥柡鍫墰缁瑩骞掑Δ鈧粻姘舵煛閸愩劎澧涢柡鍜佸墴閺屾盯寮撮悢閿嬬槕闂佹悶鍎崝搴ㄥ储閹间焦鐓熼煫鍥ㄦ礀娴犳粌顭胯濡嫮鍙呴柣搴ｆ暩绾爼宕戦幘鏂ユ灁闁割煈鍠楅悘宥夋⒑鐟欏嫮鎽冩繛鍛礃缁岃鲸绻濋崶褏顔愭繛杈剧秬閸婁粙濡搁埡鍌滃弳闂佸搫鍟犻崑鎾绘煕鎼淬垹鈻曢柟宄扮箳閳ь剨缍嗛崰妤呭煕閹烘鐓曢悘鐐插⒔閹冲懘鏌涢弬璺ㄐч柡宀€鍠栭、娆撴嚃閳轰胶鍘介柣搴ゎ潐濞叉﹢宕濋弴锛勪簷闂備礁鎲℃笟妤呭储妤ｅ啯鏅繛鎴欏灪閳锋垹绱撴担濮戭亝鎱ㄩ崶顒佺厵缁炬澘宕禍鐐电磼椤旂⒈鐓奸柟顔界懇閹粌螣閻撳骸濡囨繝鐢靛Х閺佹悂宕戦悢鐓庣；闁圭偓鏋奸弸宥夋煕閳╁啰鈯曢柣鎾存礋閹鏁愰崨顓ф殺缂備讲妾ч崑鎾绘⒒娴ｅ憡鎯堝璺烘喘瀹曟粌鈹戦崱鈺佹闂佸憡娲﹂崜娑氬姬閳ь剟姊洪崨濠冨闁告挻绋戦埢鎾诲Χ婢跺鎷绘繛杈剧导鐠€锕傛倿閻愵兙浜滈柟瀛樼箖椤ャ垺顨ラ悙鏉戝缂佺粯绻傝缂佸瀵ч崰姗€鏌熼銊ユ搐闁卞洭鏌ｉ弬鍨Щ濠碘剝濞婂缁樻媴閻熼偊鍤嬬紓浣筋嚙閸婃悂婀侀梺绋挎湰椤曢亶鏁愭径濠勵啇婵炶揪绲介崢婊堝箯濞差亝鐓熼柣妯哄级婢跺嫮鎲搁弶鍨殻鐎规洖鐏氬蹇涘煛閸愵亷绱抽柣搴＄畭閸庨亶骞婅箛娴板濮€閳垛晛浜鹃悷娆忓缁€鍐╃節閵忊槅鐒鹃柣蹇撳暣濮婃椽宕ㄦ繝浣虹箒闂佸憡锕㈢粻鏍ь嚕閺勫浚妲诲銈庝簻閸熷瓨淇婇崼鏇炲耿婵°倕鍟伴幊鍡涙⒒娴ｄ警鐒鹃柨鏇畵瀵偆鎷犻懠顒佹闂佺鎻粻鎴犵不婵犳碍鍋ｉ柛銉簻閻ㄧ儤銇勮熁閸曨厾鐦堥梺闈涢獜缁插墽娑垫ィ鍐╃叆闁哄浂浜炵粙鑽ょ磼閺冨倸鏋涚€殿喗鎸虫慨鈧柍閿亾闁归绮换娑欐綇閸撗冨煂闂佺顕滅换婵嗙暦椤栫偞鍊烽悗闈涙憸椤旀洟鏌ｉ悩鍙夊巶闁搞儺鐓堥埀顒€绉剁槐鎾存媴缁嬪簱鍋撻崫銉х煋鐟滅増甯掔粻鏍ㄧ箾閸℃ɑ灏柛銊ュ€归妵鍕箛闂堟稐绨肩紓鍌氱У椤ㄥ棛鎹㈠┑鍡忔灁闁割煈鍠楅悘宥咁渻閵堝骸骞栭柣妤佹尭閻ｅ嘲煤椤忓嫬鍞ㄥ銈嗘尵閸嬬喖宕㈤柆宥嗏拺闁荤喖鍋婇崵鐔封攽椤曗偓椤ユ挻绔熼弴鐔洪檮闁告稑锕ら埀顒傛暬閺屻劌鈹戦崱娑扁偓妤侇殽閻愭惌娈橀柣銉邯椤㈡﹢鎮欏顔芥缂傚倷娴囨ご鍝ユ暜閿熺姴绠栭柍鍝勬噹缁€鍐偓鐟板閸犳藟鐎ｎ剛纾介柛灞剧懄缁佹澘顪冮弶鎴炴喐闁轰緡鍠栬灃闁逞屽墴椤㈡岸濡烽埡浣侯槹濡炪倖甯掗崐濠氭儊閸儲鈷戦柛娑橈攻婢跺嫰鏌涘Ο鑽ょ煉濠碘剝鎮傞崺锟犲磼濮橈絽浜鹃柣銏犳啞閸嬧剝绻涢崱妤冪妞ゅ浚浜炵槐鎺楀焵椤掑嫬绀冮柕濞垮灪閺傗偓闂備胶绮崝姗€顢氬鍐惧晠闁靛鏅滈悡鐔兼煥濠靛棙鎼愰柛妯绘綑閳规垿鍩勯崘鈺佲偓鎰版煛娴ｇ鈧灝鐣峰鍡╂Ь濠碘剝褰冨ú顓烆潖缂佹ɑ濯村〒姘煎灣閸旀悂姊洪崫鍕⒈闁告挻鐩畷姘跺箳閹炬潙鍔呴梺闈涱煭缁犳垶绂掗幒鎴富闁靛牆妫欑壕鐢告煕鐎ｎ偅宕岄柡灞糕偓宕囨殕閻庯綆鍓涜摫闂備浇顕栭崹鍗炍涢崘顔兼瀬鐎广儱顦粈瀣亜閹捐泛鏋戞い鏂挎濮婂宕掑▎鎺戝帯濡炪値鍘奸悧鎾诲蓟婵犲洦鏅查柛鈩冪懐濞叉悂姊洪崜鎻掍簼婵炲弶鐗滅划濠氬冀椤撶喓鍘棅顐㈡搐椤戝懘鍩€椤掍焦绀嬮柣娑卞櫍婵偓闁挎稑瀚鏇㈡⒑閻熼偊鍤熼柛搴㈠姍閹偤鎳為妷锝勭盎闂佸搫鍊哥亸鍛啅閵夆晜鐓冮悷娆忓閻忥附顨ラ悙璇ц含闁哄本绋掔换婵嬪礃閵娿儺娼氶柣搴ゎ潐濞叉﹢宕归崸妤冨祦婵☆垵鍋愮壕鍏间繆椤栨粌甯舵鐐茬墦濮婄粯鎷呴悜妯烘畬濡炪倖娲﹂崢浠嬪箞閵娾晛绠绘い鏃囧亹閸樻椽姊洪崜鎻掍簴闁稿孩鐓￠幃鈥斥槈濮橈絽浜鹃柛蹇擃槸娴滈箖姊洪柅鐐茶嫰婢у鈧娲戦崡鍐差嚕娴犲鏁囬柣妯垮皺閵堬箓鏌ｆ惔锛勭暛闁稿酣浜惰棟濞村吋娼欑粻姘舵煟閹邦厾鏆樺ù婊勭矒閺岀喖寮堕崹顕呮殺缂備礁顑堝畷鐢垫閹烘搫绱ｆ繝闈涙閳峰鈹?
	OpenAIParsedRequestBodyKey = "openai_parsed_request_body"
	// OpenAI WS Mode 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘劙婀侀梺绋跨箰閸氬绱為幋锔界厱闁靛鍎遍埀顒€婀遍幑銏犫槈濮楀棗鏅犲銈嗘瀹曠敻鎯勬惔锝囩＜闁绘劦鍓欓崝銈嗙箾绾绡€鐎殿喖顭锋俊鍫曞触閵堝懏璐￠柍褜鍓ㄧ紞鍡涘磻閸涘瓨鍋熼柡鍐ㄧ墛閳锋垿鏌熼懖鈺佷粶濠碉紕鏅槐鎺旀嫚閼碱剙鈪甸悗娈垮枦椤曆囧煡婢舵劕顫呴柣妯活問閸熷绻濆閿嬫緲閳ь剚鍔欏畷鎴﹀箻缂佹ê鈧灚鎱ㄥΟ鐓庡付濠⒀冾嚟閳ь剚顔栭崰妤呭箖閸屾氨鏆﹂柟鐑橆殕閸婇绱掗娆炬綈閻庢艾銈稿缁樻媴閸涘﹨纭€濡炪値鍘奸悧蹇涘焵椤掍胶鈻撻柡鍛箘閸掓帡宕奸妷銉у姦濡炪倖甯掔€氼參鍩涢幋鐘电＜閻庯綆浜滈惃锛勨偓瑙勬偠閸庣敻寮婚悢鍏煎殟闁靛／鍛帨婵°倗濮烽崑娑⑺囬悽鍝ュ祦婵せ鍋撴い銏＄懇閹崇偤濡烽敂鎯х悼闂傚倸鍊烽懗鍫曗€﹂崼銉ュ珘妞ゆ巻鍋撴い顐ｇ箞婵℃悂鍩℃繝鍐╂珦闂備浇顫夋竟鍡樻櫠濡ゅ懏鍋傛繛鎴烇供閻斿棝鎮归搹鐟扮殤闁告梻鍠庨…鑳槼婵炲弶鐗犻垾锕傛嚄椤栵絾鞋闂備礁鎼幏瀣磻閸℃稑绀嗛柟鐑樺灍閺嬪酣鏌熼柇锕€鏋ら柣锕€鐗嗛埞鎴︻敊閺傘倓绶甸梺鍛婏耿缁犳牕鐣烽姀锛勵浄閻庯綆鍋€閹锋椽姊洪崷顓х劸婵炴挳顥撶划濠氬箻濞ｎ兛绨婚梺鎸庢椤曆冣枍瀹ュ棭娈介柣鎰▕濡偓闂佽鍠楅悷鈺呫€佸Δ浣瑰缂佸銇樼槐鐔哥節閻㈤潧啸闁轰礁鎲￠幈銊╁箻椤旇偐鏌堝銈嗙墱閸嬫稒顢婃繝鐢靛█濞佳兠归崒姘ｆ灁濞寸姴顑嗛悡鐔兼煙闁箑澧伴柟鐣屽Х閳ь剝顫夊ú鏍嫉椤掑嫨鈧啴濡烽埡鍌氣偓鐑芥煛婢跺鐏﹂悹鍥╁仱閹鈻撻崹顔界彯闂佸憡鎸婚悷鈺呭春閻愬搫绠ｉ柨鏇楀亾缂佲偓鐎ｎ偁浜滈柟鎵虫櫅閻忊晝鎮Ο鑲╃＝闁稿本鑹鹃埀顒€鍢查湁闁搞儯鍔婃禒鍫ユ煕濠靛嫬鍔滈柡鍡樼矒閺岋綁骞囬鍌欑驳濡炪値鍋呭ú鐔煎蓟閻斿吋鍊绘俊顖滃劋椤旀洟姊虹紒妯活梿婵℃ぜ鍔戦幃鐐寸節閸ャ劎鍘搁悗骞垮劚濞寸兘宕㈠☉娆戠闁稿繐鍚嬮ˉ鍫熸叏婵犲偆鐓肩€规洘甯掗埢搴ㄥ箣椤撶啘婊堟⒒娴ｅ湱婀介柛濠冩礀鐓ゆい鎿冩娇閳ь兛绀侀～婵嬪箛娴ｅ厜鏋岄梻鍌欑閹碱偊鎳熼鐐存櫇妞ゅ繐鐗嗛弸渚€鏌熼柇锕€骞栫紒鍓佸仜閳规垿鎮╅幓鎺撴閻炴熬缍佸濠氬磼濞嗘埈妲梺瑙勭ゴ閳ь剝绉ú顏呮櫇闁稿本鑹鹃悗顓㈡⒑鐟欏嫷鍟忛柛鐘冲哺閹矂宕卞☉娆戝幈闂侀€涘嵆濞佳囧几閻斿吋鐓熼柟鎯у暱閺嗭綁鏌＄仦鍓р槈闁宠鍨垮畷鍗炍旀繝浣瑰亝婵犵數鍋涢悺銊у垝鎼淬劌纾婚柕鍫濐槸閽冪喐绻涢幋娆忕仾闁稿瀚伴弻鐔虹磼濡櫣鐟愭繝銏ｎ潐濞叉牠鍩為幋锔藉亹闁割煈鍋呭В鍕節濞堝灝鏋ら柛蹇斆锝夘敃閿曗偓缁犵懓霉閿濆懏鎲搁柨娑欑洴濮婅櫣绱掑Ο鍝勵潓闂佸湱鈷堥崑濠傜暦瑜版帒纾奸柣鎰嚟閸樺崬顪冮妶鍡楀Ё缂佹彃澧界划鍫ュ焵椤掑倻纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼达絾鏆╂い顓炵墛娣囧﹥绂掔€ｎ偀鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜閻犲洦褰冮弳锝呪攽閿涘嫭鏆€规洜鍠栭、娆撳礈瑜庡鎴︽⒒娴ｅ憡璐￠柛瀣崌瀹曟粌顫濋鐑嗗殼閻庡厜鍋撻柛鏇ㄥ厴閹锋椽鏌ｉ悩鍙夋悙鐎殿喖鐖煎畷顒佸緞閹邦厾鍘遍梺鍐叉惈閸婄粯鏅堕柆宥嗙厵妤犵偛鐏濋悘鈺呮煃閽樺妲告い顐ｇ矒瀹曞崬螖娴ｅ湱褰嬫繝纰夌磿閸嬫垿宕愰弽顬稒鎷呴崫銉︽闂佹眹鍨婚…鍫㈢矆閸曨厽鍠愰柣妤€鐗嗙粭姘舵煃闁垮鐏撮柡灞剧☉閳规垿宕卞Δ濠佺棯濠电姵顔栭崹浼村Χ閹间礁钃熸繛鎴欏灩缁犲鎮归搹鐟板妺闁诲繐鐗撻幃妤€鈻撻崹顔界仌濡炪倖娉﹂崶褏鍙€婵犮垼鍩栭崝鏇綖閸涘瓨鐓冮柛婵嗗閳ь剛鎳撻埢宥夊炊椤掍讲鎷绘繛杈剧悼閹虫捇顢氬鍛＜閻犲洦褰冮埀顒€鎽滈崣鍛存⒑闂堟单鍫ュ疾濠婂牆纾婚柛宀€鍋為悡鏇犫偓鍏夊亾闁逞屽墴瀹曟垿鎮欓崫鍕幈闂佺鎻梽鍕偂閻樻祴鏀芥い鏃囨婵洭鏌嶈閸撴艾煤閻斿鍤曢柟闂寸缁狅綁鏌ｅΟ鍏兼毄闁挎稒绮撻弻锝夋偐閸欏鈹涢柣蹇撶箲閻燂箓寮查崼鏇熸櫆闂佹鍨版禍鐐箾閸繄浠㈤柡瀣堕檮閵囧嫰寮撮崱妤佹悙闁绘挴鈧剚鐔嗛悹杞拌閻擃剚绻涚粭鍝勫闁哄本鐩崺鍕礂閻欌偓娴滎亜鐣峰Δ浣虹瘈闁搞儯鍔夐幏缁樼箾鏉堝墽绉繛鍜冪悼閺侇喖鈽夐姀锛勫幍缂備礁顑嗙€笛囧箟閹间焦鐓犲Δ锕€娼￠崫铏圭磼閾忚娅曠紒顔界懅閹瑰嫰濡歌閻愬﹥绻濋悽闈浶ラ柡浣告啞閹便劑骞橀鑲╋紱闂佺硶鍓濋妵婊堝焵椤掍礁绗氱紒缁樼箓椤繈顢橀悩鎻掑闂傚倷绀佹竟濠囧磻閸涱垱宕查柛宀€鍋涢悞鍨亜閹烘垵鈧綊宕甸埀顒勬⒑鐎圭媭鍤欑紒澶屾嚀閻ｉ攱绺介崨濠備簻缂傚倸鐗忔慨鐑芥儎鎼淬劍鈷掑ù锝呮啞閹牊绻涚仦鍌氣偓鏇＄亱濠德板€愰崑鎾绘偂閵堝鐓忛柛顐ｇ箥濡插綊鏌涚€ｎ亜鈧湱鎹㈠┑瀣棃婵炴垵宕崜鎵磽娴ｅ搫鞋妞ゎ偄顦遍幑銏犫槈閵忊剝娅滈柟鑲╄ˉ閳ь剚鍓氬璇测攽閻樻鏆柍褜鍓涢崑銊╁磻閵忋倖鐓涚€光偓鐎ｎ剛袣缂備胶濮甸惄顖炵嵁濡吋濯奸柛锔诲幘椤︻垶姊婚崒娆戠獢闁逞屽墰閸嬫盯鎳熼娑欐珷闁告瑥顦禍婊勩亜閹扳晛鐒烘俊鍙夋倐閹繝濡舵径瀣幐閻庡箍鍎卞ù閿嬬鐟欏嫮绠剧€瑰壊鍠曠花濂告煛閸涱喚绠為柡灞剧〒娴狅妇鎷犲ù瀣壕闁哄稁鍘奸悿鐐亜韫囨挸顏ら柡鈧禒瀣厽闁归偊鍘界紞鎴︽煟韫囨洖鏋涢柡灞剧洴婵℃悂鏁傛慨鎰檸闂備浇顕栭崳顖滄崲濠靛绠栭柕蹇嬪€曠粈鍌炴煠濞村娅呮鐐搭殕缁绘繄鍠婂Ο娲绘綉闂佹悶鍔嬮崡鎶界嵁閺嶎収鏁冮柕鍫濇处閿涘繘鎮峰鍐ч柍銉閹瑰嫰濡搁敃鈧壕顖炴⒑閸涘﹦绠撻悗姘舵敱缁傛帡宕滆绾捐棄霉閿濆牊顏犻悽顖涚〒缁辨帞鈧綆鍋勯悘瀛樼節閳ь剚绗熼埀顒勫蓟閿濆棙鍎熸い鏍ㄧ矌鏍￠梻浣侯焾椤戝懘骞婂鈧獮鍐偨缁嬪灝鍞ㄩ悷婊冾樀瀵劍绂掔€ｎ偆鍘撻梺闈涱槶閸庨亶寮虫潏鈺冪＜闁绘瑦鐟ュú锕傛偂閺囥垺鐓涢柛鎰剁到娴滄儳鈹戦悙璺虹毢闁哥姵鐗犲顐﹀箛閻楀牆鈧攱绻涢弶鎴剱闁哄倵鍋撻梻鍌欑婢瑰﹪宕戦崱娑樼獥閹艰揪绲介ˉ姘亜閺嶃劎銆掔紒?	// 婵?Codex 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟撮崣鍕煏閸℃鏆ｅ┑锛勫厴閸┾剝鎷呮搴ｅ€為梻鍌欑窔濞佳囨晬韫囨稑纾兼繝濠傛噺閸犳帡姊绘担绛嬪殭濡ょ姴鎽滅划璇差吋婢跺﹦锛熼梻渚囧墮閸楁洟宕堕澶嬫櫖闂佺粯鍔栬ぐ鍐倵椤撱垺鈷戠紒瀣濠€鎵磼鐎ｎ偅灏电紒顔碱煼瀹曟粏顦柛瀣崌閹兘寮跺▎鐐棏闂備礁鎽滄慨闈浢哄鍫熷殟閺夊牄鍔庣弧鈧┑顔斤供閸橀箖宕㈤幖浣光拺闁告稑锕ョ壕鐢告煛閸涱喚娲寸€规洘绻傞…銊╁川椤栨粣绱茬紓鍌氬€烽悞锕傗€﹂崶顬¤櫣鈧稒锕╁▓浠嬫煟閹邦剚鈻曢柛搴㈡閺岀喖顢欓悾灞惧櫘闂侀€炲苯澧存繛浣冲洤绠熼柨鐔哄閺佸洤鈹戦悩宕囶暡闁抽攱甯掗湁闁挎繂鎳忛崯鐐烘煕閻斿搫浠﹂柕鍥у婵℃悂濡烽敂缁橈骏闂備礁鎽滄慨鎾晝閵堝鏁囬柛蹇曞帶缁剁偛鈹戦悜鍡樼窙缂佺姵鎹囧璇测槈閵忕姴宓嗛梺闈涱焾閸庤櫕绂掗埡鍛拺闁告稑锕ゆ慨鈧梺绋款儐閹瑰洤顫忕紒妯肩懝闁逞屽墮椤洩顦堕柛锝呯秺濮婃椽宕ㄦ繝鍐槱闂佺顑呴幊姗€骞冮悽鍓叉晩闁告挆鍜冪闯闂備胶顭堥張顒勬嚌妤ｅ啫鐒垫い鎺戝€搁崢鎾煙閾忣偒娈滈柟铏矒瀹曞綊顢曢姀鐘辩礋闂佽姘﹂～澶娒洪弽顬℃椽濡舵径濠傜€┑鐘绘涧濞层劎绮绘ィ鍐╃厱闁斥晛鍙愰幋鐘辩剨妞ゆ挶鍨洪悡銏ゆ煕閹板吀绨婚柡瀣洴閺屸€崇暆鐎ｎ剛袦閻庢鍣崜鐔风暦瑜版帩鏁婇柦妯侯槴閸嬫捇顢涢悙绮规嫼闂佽鍎兼慨銈夊极闁秵鐓曢柡鍐ｅ亾闁荤啿鏅涢锝嗙節濮橆厼浜滈梺绋跨箺閸嬫劙宕濋悜鑺モ拺闁圭瀛╃壕鐢告煕鐎ｎ偅灏甸柍褜鍓氶鏍窗濮樿泛纾婚柟鎯ь嚟閳瑰秴鈹戦悩鍙夌ォ闁轰礁绉甸幈銊ヮ潨閸℃骞嬮梺绋款儐閹瑰洭骞婇敓鐘参ч柛鈾€鏅滅紞妤呮⒒娴ｇ瓔娼愰柛搴㈠▕閹椽濡搁敃鈧崹鏃堟煙缂併垹鏋熼柛瀣у墲缁绘盯宕卞Δ鍐唶濡炪倕绻堥崐婵嬪蓟閻旈鏆嬮柟娈垮枤妤旀繝娈垮枛閿曘儱顪冩禒瀣畺闁绘垼濮ら崑瀣煕椤愶絿绠栨繛鍫熋埞鎴︽偐閸偅姣勯梺绋款儐閻╊垶寮崘顕呮晜闁割偆鍠庢禒鎺戭渻閵堝棙鈷掗柡鍜佸亞瀵囧焵椤掑嫭鈷戠紒瀣濠€鐗堛亜閵娿儲顥㈤柨婵堝仜椤撳ジ宕ㄩ鍛澑闂備胶绮崝鏇烆嚕閸泙澶愭倷閻戞鍘遍梺鍝勫暊閸嬫挻銇勯妸銉уⅱ闁告帗甯″顕€宕煎┑瀣暪闂備礁鎼ú銊╁疮閸ф绠繛宸簼閳锋垿鏌涘☉姗堝姛闁宠棄顦甸弻銊╁即濡搫濮㈤梺閫炲苯澧柣蹇斿哺閹兘鍩￠崨顓℃憰濠电偞鍨惰彜婵℃彃鐗撻弻娑樜旈崘銊ゆ埛婵炲濮弲鐘差潖閾忓湱纾兼俊顖氭惈椤矂姊洪幐搴㈢８闁稿孩濞婇獮鎴﹀閻橆偅鏂€闁诲函缍嗛崑鍕濞差亝鈷掗柛灞炬皑婢ф盯鏌涢幒鍡椾壕闂備胶绮崝妤呭磿閵堝鍋傞柡鍥ュ灪閻撳繐鈹戦悩鑼婵＄虎鍣ｉ弻鏇㈠炊閵娧呯暭闂侀€炲苯澧叉い顐㈩槸鐓ゆ俊顖氥偨濞差亝鍋勯柣鎾虫捣閻ゅ洭姊洪崫鍕枆闁告ê銈搁幃鈥斥槈濡繐缍婇弫鎰板川椤旇棄鏋戝┑鐐茬摠缁牓宕￠崘宸綎缂備焦蓱婵挳鏌涘☉姗堝伐闁哄棗鐗撳娲捶椤撗呭姼濠碘槅鍋呯换鍌炴偩閻戣姤鍋勭痪鎷岄哺閺呮繈姊洪幐搴ｇ畵婵炶尙濞€瀹曟垿骞樺ú缁樻櫍闂侀潧绻嗗褔骞忕紒妯肩閺夊牆澧介幃濂告煙閾忣偅宕屾い銏¤壘椤劑宕熼鐙€鍟庨梻浣告啞閻熴儵藝鏉堛劍娅犻柣銏㈩暯閸嬫挾鎲撮崟顒傤槰闂佸憡姊归悷銉╂偩閻戠瓔鏁冮柨鏇楀亾閸烆垶姊洪幐搴㈢５闁稿鎹囬弻锛勨偓锝庝簻閳ь剙鐏濋～蹇撁洪鍕唶闁瑰吋鐣崹濠毸囬妷鈺傗拺缂佸灏呭銉╂煙閼恒儳鐭岄柛鎺撳浮閹筹繝濡堕崱妤佺€梻浣告啞濞诧箓宕㈡ィ鍐炬晩闁哄洢鍨洪埛鎴︽煙缁嬫寧鎹ｇ紒鐘虫尭铻栭柣妯哄级閸熺偤鏌￠崨顓犲煟妞ゃ垺宀搁崺鈧い鎺戝閽冪喖鏌ㄥ┑鍡欏ⅱ闁汇倐鍋撻梻浣瑰缁诲倻鈧凹鍣ｉ幆鍕償閵婏腹鎷婚梺鎼炲劀鐏為敮鏋呴梻浣告惈閻寰婇崐鐔轰航闂佽崵濮垫禍浠嬪礈閸楃儐鐓ラ柛鏇楁櫃缁ㄥ妫呴銏″闁瑰摜鍋撻弲鍫曞箛閻楀牏鍘撻悷婊勭矒瀹曟粓濡歌娑撳秹鏌熼幆褏锛嶉柡鍡缁辨帞鈧綆鍙庨崵锕傛煛閸愶絽浜鹃梺鐟板槻閹虫ê鐣烽妸锔剧瘈闁告洦鍓氱欢顓㈡⒒閸屾艾鈧悂宕愰幖浣哥９闁归棿绀佺壕褰掓煙闂傚鍔嶉柛瀣儏閳规垿鎮╅幓鎺濅紑闂佹椿鍘介悷鈺呭蓟閻斿憡缍囬柛鎾楀懏娈哥紓鍌欒兌婵潧顪冩禒瀣摕闁挎繂顦～鍛存煟濡搫鏆遍柡鍡楃墦濮婃椽骞庨懞銉︽殸闂佹悶鍔岄悘婵嬶綖韫囨梻绡€婵﹩鍓熼崬璺衡攽椤旀枻渚涢柛妯圭矙閹敻寮介鐔叉嫼闂佸憡绋戦敃銈囩箔閸岀偞鐓曟俊銈勭閳绘洜鈧鍠栭…鐑藉箖閵忋垺濯奸柛顭戝枟鐠愶繝鏌嶈閸撱劑骞€閵夆晛绀冪憸宥壦夊鑸碘拻濞达絽鎽滅粔鐑樸亜閿濆繒鐣甸柟铏殜瀹曞ジ寮撮悙娈挎Х?5 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃缂侇噮鍨抽幑銏犫槈閵忕姷顓哄┑鐐叉缁绘帗绂掓總鍛娾拺闁告稑锕ら悘鈺佲攽閻愨晛浜鹃梻浣告惈閻ジ宕伴幘鑸殿潟闁圭儤顨呴～鍛存煟濡櫣锛嶅ù婊庝簽缁辨捇宕掑▎鎺戝帯婵犳鍠曢崡鍐茬暦閺囥垺鍋ㄧ紒瀣劵閹芥洖鈹戦悙鏉戠仧闁搞劌婀辩划璇测槈濞嗗秳绨婚梺鍝勭Р閸斿秹鎯冮幋婵愮唵閻犲搫鎼顏嗙磼?
	openAIWSReconnectRetryLimit = 5
	// OpenAI WS Mode 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚敐澶婄闁挎繂鎲涢幘缁樼厱濠电姴鍊归崑銉╂煛鐏炶濮傜€殿喗娼欓～婵嬫嚋濞堝簱鍋撻弽銊х閻庢稒顭囬惌濠勭磽瀹ュ拑韬€殿喛顕ч埥澶愬閻樼數鏉搁梻浣侯焾缁绘劙骞楀鍫濇瀬闁哄稁鍘介埛鎴炵箾閼奸鍤欐鐐搭殜閺岀喖宕ㄦ繝鍐ㄢ偓鎰版煃閵夛附顥堢€规洘锕㈤、鏃€鎷呴梹鎰棜闂備焦瀵х换鍌滆姳閼测晞濮崇紓浣骨滄禍婊堟煙閸濆嫷鍎忛柣蹇ｅ櫍閺屽秷顧侀柛鎾寸箞閿濈偞寰勬繛鎺撴そ瀵€燁檨闁搞倖娲熼弻娑㈠箻閼碱剙濡介悗瑙勬礀瀵墎鎹㈠☉銏犵婵炲棗绻掓禒鑲╃磽娴ｅ搫顎撶紒鎻掑⒔閹广垹鈽夐姀鐘茶€垮┑掳鍊曢崯浼搭敊婵犲啰绠鹃悗鐢殿焾閳诲牏绱掗悩宕囧ⅹ妞ゎ偄绻橀幖褰掑捶椤撶媴绱叉繝纰樻閸ㄩ潧鈻嶉敐澶嬪仭闁靛鍎哄〒濠氭煏閸繈顎楁鐐达耿閺屾稖绠涢弬鍡╀邯椤㈡岸鏁愰崶銊ョ彴閻庣懓澹婇崰鏍箖閹达附鈷戦柛鎾村絻娴滀粙鏌涚€ｎ亝鍤囨い銏℃崌瀹曞爼顢楁担鍙夊濠电偠鎻徊浠嬪箟閿熺姴鐤柣鎰劋閻撴瑩鏌熼梻纾嬪厡闁伙綀娅ｉ埀顒冾潐濞叉垿宕￠崘宸殨闁稿﹦鍣ュ銊╂⒑閹肩偛鈧牕煤閺嶎厼鐓橀柟杈鹃檮閸嬫劙鎮楅崷顓炐㈡い銉︾箓铻栭柣姗€娼ф禍濂告煕閵娿劍顏犻柟骞垮灩閳藉濮€閻樻鍚呴梻浣虹帛閸ㄩ潧煤閳哄懎鐒垫い鎺嗗亾婵炵》绻濆濠氬焺閸愨晛顎撻梺闈╁瘜閸橀箖鎮￠埀顒勬⒒娴ｅ摜鏋冩い顐㈩樀瀹曞綊宕奸弴鐘茬ウ婵犵數濮村ú锕傛偂閺囩偐鏀介柣妯诲絻閺嗙偟绱掗埀顒傗偓锝庡厴閸嬫挾鎲撮崟顒€顦╅梺鎼炲妼閻栧ジ濡存笟鈧鎾閻樻爠鍥ㄧ厱闁靛鍨哄▍鍥煛閸℃劕鍔︽慨濠勭帛閹峰懘鎳為妷褋鈧﹪姊洪崫銉バｉ柟绋款煼楠炲牓濡搁埡鍌氫缓闂佸憡绋戦敃銈嗙濡ゅ懏鈷戠紓浣股戦悡銉╂煕濮橆剦鍎旈柟顕嗙節閹垽宕ㄦ繝鍌氫紟婵犵绱曢崑鐘活敋瑜庣粋宥夋倷椤掑倻顔曢柣搴㈢⊕椤洭鎯屾惔銏㈢濠㈣泛顑囧ú鎾煃閵夛妇澧紒缁樼箞瀹曞爼濡搁妷銏犱壕濠电姵纰嶉悡鐘绘煙椤撶喎绗掗柛鏂诲€濋弻娑樜熼崸妤€鎽电紓浣虹帛缁诲牓骞冩禒瀣棃婵炵缈伴崹浠嬪蓟濞戙垺鍋嗗ù锝呮憸娴狀垳绱撴笟鍥ф灈闁绘绻橀崺鈧い鎺戯功缁夌敻鏌涢悩鎰佹疁闁诡噯绻濆鎾閿涘嫬甯楅梺鑽ゅ枑閻熴儳鈧凹鍓熷畷婵嬪Χ閸氥倗鎳撻…銊╁川椤撴繂顥氬┑鐑囩到濞层倝鏁冮鍫濈畺婵炲棙鎼╅弫鍌炴煕閺囨ê濡煎ù婊堢畺閺屸€愁吋鎼粹€茬凹闂佸搫妫欑划鎾诲蓟閻斿吋鍊绘慨妤€妫欓悾鍓佺磼閻愵剙鍔ょ紓宥咃躬瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷鈺傗拺缂備焦锚缁楁帡鏌ｈ箛鏂垮摵濠碉紕鏁诲畷鐔碱敍濮橀硸鍞洪梻浣虹《閸撴繈濡甸悙瀵哥彾闁哄洨鍠嶇换鍡涙煟閹板吀绨婚柍褜鍓氶悧鐘茬暦瑜版帗顥堟繛娣灪濡炶姤淇婇崼鏇炲窛妞ゆ挾濮伴崑鎺楁⒒閸屾艾鈧绮堟担铏圭濠电姴娲ょ壕璇测攽閻樺弶鎼愮紒鈧崒娑欏弿婵＄偠顕ф禍楣冩倵濞堝灝鏋涢柟璇х節楠炲棝寮崼婢晠鏌ㄩ弮鈧崕鎶界嵁瀹ュ鈷掑ù锝囩摂閸ゅ啴鏌涢妸銉╁弰鐎规洦浜畷姗€顢旈崱鈺傂﹂梻鍌氬€搁崐鐑芥嚄閸撲礁鍨濇い鏍仜缁犳澘鈹戦悩瀹犲闁搞劌鍊块弻锝夊閵忣潿鈧﹪鏌嶈閸撴艾螞閸愩劎鏆︽慨妞诲亾妞ゃ垺鐟╁畷鍙夌珶椤栨碍澶勯柣鎾存礋閺屽秹鍩℃担鍛婃闂佺懓鍟块崯顖炲Φ閸曨垼鏁冮幖绮瑰墲閸ｇ晫鐥幑鎰棄闁靛棙甯掗～婵嬵敆娴ｈ鍊烽梻浣告惈椤戝棛绮欓幘鑸殿潟闁圭儤鎸哥欢鐐烘倶閻愭潙鍔ゆ繛鍫濊嫰椤啴濡堕崱妯尖敍缂傚倸绉崇欢姘舵偘椤旇姤鍎熼柕濠忕畱濞堢喖姊洪棃娑辨Ф闁稿孩婢樻晥婵°倕鍟扮壕浠嬫煕鐏炴崘澹橀柍褜鍓涢崗姗€骞冮悙鐑樻櫇濞达絽鎽滅粙濠囨⒒閸屾瑧顦︾紓宥咃躬瀵劑宕￠悜鍡樺瘜闂佽姤锚椤﹁棄顭囬弽銊х鐎瑰壊鍠曠花璇裁归懖鈺佲枅闁哄本鐩鎾倷閸忓摜椹冲┑顔界箓濞诧妇鎹㈠┑瀣仺闂傚牊绋愮划璺侯渻閵堝繘妾繝銏☆焽閸欏懎鈹戦悩缁樻锭妞ゆ垵鎳愭竟鏇°亹閹烘挾鍘电紓鍌欓檷閸ㄥ綊寮搁悢鍏肩厓闂佸灝顑呯粭鎺楁婢舵劖鐓ユ繝闈涙閸ｆ椽鎮归幇銊ュ⒉缂佺粯绻堥崺鈧い鎺嶈兌椤╃兘鎮楅敐搴′簽闁告椴哥换婵嬫偨闂堟刀銏ゆ煕閻曚礁鐏ｇ紒顔肩墦閸┾偓妞ゆ帒瀚悡锝夋煠婵劕鈧洘绂掗敂鍓х＜缂備焦顭囧ú瀛橆殽閻愬樊鍎忛柍璇叉唉缁犳盯寮村顓炰簼缂傚倸鍊搁崐鐑芥嚄閼稿灚鍙忛柣鎴ｆ缁愭骞栧ǎ顒€濡界紒鐘冲浮濮婄粯鎷呮笟顖滃姼缂備胶绮崝娆忕暦閵壯€鍋撻敐搴″缂佲偓婵犲洦鐓欓悗鐢殿焾閸撹鲸绻涢崼婊呯煓闁哄矉缍侀獮鍥礂閸濄儳娉块梻浣筋嚙濞存碍绂嶉悙鍨潟闁圭儤姊荤壕鍏间繆椤栨艾鎮戦柛鎺撶洴閹鎲撮崟顒傤槬閻庤娲﹂崜婵嬫倶閹烘鈷戦柛娑橈功缁犳捇鎮楀顒€妲绘い顓炴喘瀵挳濮€閳锯偓閹风粯绻涙潏鍓ф偧闁硅櫕鎹囬、姘煥閸涱垳锛滈柣搴秵閸嬪嫬霉椤旈敮鍋撶憴鍕缂佽鍊块崺銏℃償閵堝洨鏉搁梺鎸庣箓濡稓绮欐担鍦瘈闁汇垽娼ф禒婊勪繆椤栨熬鏀荤紒鍌氱Т椤劑宕ㄩ娆戠憹濠电偞娼欓崥瀣焽濞嗘垹涓嶇憸鐗堝笒缁狙囨煕椤愶絿绠撻柍閿嬫⒒閳ь剙鍘滈崑鎾寸箾閹寸偟鎳佺紒璇叉閺岀喓绱掑Ο铏诡儌婵犫拃鍕闁靛洤瀚伴、鏃堝礋椤愶絾顔掑┑鐘殿暯閸撴繈宕曢懖鈺冧笉婵炴垯鍨圭粻濠氭煠閸涘﹥娅曢柟鐑橆殕閳锋帒霉閿濆懏鍟為柛鐔哄仱閺岋綁鎮㈤弶鎴濆Е閻庤娲﹂崹璺侯嚕閸洖绠ｉ柣妯挎珪椤旀洘绻濋悽闈涗粶婵☆垰锕ョ粋宥咁煥閸喎鈧?
	openAIWSRetryBackoffInitialDefault = 120 * time.Millisecond
	openAIWSRetryBackoffMaxDefault     = 2 * time.Second
	openAIWSRetryJitterRatioDefault    = 0.2
	openAICompactSessionSeedKey        = "openai_compact_session_seed"
	codexCLIVersion                    = "0.125.0"
	// Codex 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鈽夊Ο閿嬫杸闂佸憡娲﹂崑鍕叏婢舵劖鈷戠紒瀣儥閸庢劙鏌熼崨濠冨€愰柨婵堝仜閳规垹鈧綆鍋勬禍妤呮煙閼圭増褰х紒杈ㄦ礋閹牓宕熼娑氬幗闁瑰吋鐣崐銈咁焽閹邦厾绠鹃柛娆忣樈閻掍粙鏌熼獮鍨伈鐎规洘甯￠幃娆撴嚑椤掆偓楠炴帡姊洪悷鏉挎倯闁伙綆浜畷婵嗙暆閳ь剛鍒掔紒妯肩瘈婵﹩鍘鹃崢鎾绘偡濠婂嫮鐭掔€规洘绮撻幃銏ゆ偂鎼达絿鏆繝鐢靛Т閿曘倝鎮ф繝鍕ㄥ亾閻㈤潧孝妞ゎ叀娉曢幑鍕惞閸︻厼濮界紓鍌欑瀹曨剙顕ｉ崜浣瑰床婵犻潧顑嗛崑銊╂⒒閸喓鈻撻柡瀣噹閳规垿鎮欓弶鍨殶闂佹悶鍎崝灞剧闁秵顥婃い鎰╁灪婢跺嫮绱掔€ｎ偄鐏╅柣銉海椤﹀绱掓潏銊ョ闁逞屽墾缂嶅棝宕戦幒妤€纾块柕澶涘缁犻箖鏌熼悙顒€澧柣鎾炽偢閺岋紕浠﹂悙顒傤槹閻庤娲滈崢褔鍩為幋锕€鐐婇柤鍛婎問濡勭節閻㈤潧校妞ゆ梹鐗犲畷浼村冀椤撶偟锛欓梺绉嗗嫷娈旈柦鍐枑缁绘盯骞嬮悙鍨櫘缂備讲妾ч崑鎾绘⒒娴ｅ憡鍟為柛鏂跨箻瀵彃鈹戠€ｅ骸娲俊鐑藉Ψ椤旇棄鐦滈梻渚€娼ч悧鍡椢涘▎鎴犵煔閻犳亽鍔庣壕鐣屸偓骞垮劚鐎氼喚绮ｉ弮鍫熺厽婵炴垵宕▍宥団偓瑙勬礀瀹曨剟鍩ユ径鎰闁糕剝銇炴竟鏇㈡⒑閸濆嫮鈻夐柛妯恒偢閹ê鐣烽崶锝呬壕閻熸瑥瀚粈鈧┑鐐茬湴閸斿秹銆冨▎鎾粹拻闁稿本鐟︾粊鐗堛亜閺囩喓澧电€规洑鍗抽獮鎺懳旈埀顒勬偪妤ｅ啯鐓熸俊顖氱仢閻ㄦ椽鏌￠崱顓犵暤闁哄被鍔岄埞鎴﹀幢濮楀棙顥ｇ紓鍌欑婢у酣宕戝☉銏犵厴闁硅揪闄勯崑鎰版煙缂佹ê绗氶柣蹇旀尰缁绘稓鈧數顭堥埢鍫澝瑰鍡樼【妞ゎ偄绻戠换婵嗩潩椤掑偊绱叉繝鐢靛仜濡瑩宕濆Δ鍛ч柡澶婄氨閺€浠嬫煃閳轰礁鏆斿ù鐘靛█閺屾盯寮埀顒勫垂閻㈤潻缍栭煫鍥ㄧ⊕鐎电姴顭块懜寰楊亪骞冮幋鐐电瘈闁靛骏绲剧涵楣冩煥閺囶亞鐣甸柡浣哥Т閳诲酣骞樼划瑙勫闂備礁鎲＄换鍌溾偓姘煎櫍閸┿垺寰勯幇顓犲幈闁诲函缍嗘禍婵嬎夐姀銈嗙厵妞ゆ牗纰嶅﹢鎵偓鍨緲鐎氼喗绂掗敂鍓х煓濠㈠墎顭堥ˉ姘繆閵堝洤啸闁稿鐩幃妯衡攽鐎ｎ亞顦┑顔斤供閸樿櫣鎹㈤崱妯镐簻闁规澘澧庨幃濂告煙閸愬弶宸濋柍褜鍓濋～澶娒哄Ο濂芥椽鎮㈤悡搴ｇ暫闂侀潧绻堥崐鏇犵矆閸岀偞鐓犳繛鏉戭儐濞呭懘鎮介娑辨疁婵﹦绮幏鍛村传閵夛妇鈧喖鈹戦埄鍐︿粻闁告柨娴烽崚鎺楀醇閻旇櫣鎳濋梺閫炲苯澧い鏇秮瀹曟ê霉鐎ｎ偆鈧姊虹憴鍕姢妞ゆ洦鍙冨畷銏ゅ箻缂佹ê浠┑鐘诧工鐎氼噣鎯岄幒鏂哄亾鐟欏嫭绀冪紒顔肩焸閸┿儲寰勯幇顒夋綂闂佺粯锚绾绢參鍩€椤掑澧紒缁樼箞閹粙妫冨ù璁圭秮閺屻倛銇愰幒鏃傛毇闂佸綊鏀遍崹鍧楀箖閵忋倕浼犻柕澹苯鏅梻鍌欒兌閹虫捇顢氶銏犵９闁瑰瓨绻嶅鈺呮偣閸濆嫭鎯堥柛銈嗗灴閺岀喖宕楅懖鈺傛闂佸憡鏌ㄧ粔褰掑箖閳ユ枼鏋庨柟鍓цˉ閹峰搫顪冮妶鍡樺蔼闁告柨閰ｉ幃鐐綇椤垶顔旈梺缁樺姌閹活亪鎳撻崸妤佺厸閻忕偛澧藉ú瀛橆殽閻愬樊鍎旀い銏＄☉椤劑宕ㄩ娑辨綋婵犵數濮甸鏍窗閺嶎厽鏅濋柕鍫濐槸缁犺銇勯幇鈺佲偓鏇烆嚕閺屻儲鈷戦柤濮愬€曞瓭濠电偞绁撮弲娑㈩敋閿濆绠婚柟顖涙緲灏忛梻浣哄帶椤洟宕愬Δ鍐棜闁芥ê顥㈣ぐ鎺撴櫜闁告侗鍙庡Λ灞解攽闄囩亸娆撯€﹂崼銉⑩偓锕傛嚄椤栵絾些闂傚倷绀佹惔婊呭緤娴犲鍋?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌＄仦绋垮⒉鐎垫澘瀚埀顒婄秵娴滄繈顢欓崨顓涙斀闁绘劕寮堕埢鏇灻瑰鍕煀婵炴彃鐏濋埞鎴︽偐椤旇偐浼囧┑鐐差槹閻╊垶銆侀弽顓炲耿婵＄偟绮悗娲⒑閹肩偛鍔撮柛鎾村哺閸╂盯骞嬮敂瑙ｆ嫽闂佸壊鍋嗛崰宥囨鏉堛劋绻嗛柛娆忣槹鐏忥箓鏌″畝瀣瘈鐎规洏鍔戦、娆撴倷椤掆偓閸ㄩ亶姊绘担鍝ョШ闁衡偓闁秴鍨傞柛褎顨呯粻鏍煏韫囧鈧洜绮堥崒鐐寸厾婵炴潙顑嗗▍鍛殽閻愯尙澧︽慨濠勭帛缁楃喖鍩€椤掆偓椤洩顦归挊婵喢归悩宸剰缂佹劖顨婇弻鈥愁吋閸愩劌顬夊┑鐐叉噽婵敻濡甸崟顖氬唨妞ゆ劦婢€濞岊亪姊虹紒妯诲鞍闁烩晩鍨跺璇测槈濮楀棛鍙嗛梺閫炲苯澧扮紒顔肩墛閹峰懐鎲撮崟顒傚娇闂備礁婀遍搹搴ㄥ窗濡ゅ懏鍋傛繛鍡樺姂娴滄粓鏌￠崘锝呬壕闂佸搫顑呴…鐑藉箠閹捐閿ゆ俊銈勮兌閸橀潧顪冮妶鍡樷拻闁哄拋鍋婇獮濠囧川鐎涙鍘甸梺鑽ゅ枑濠㈡﹢顢旈鍛缁炬澘褰夐柇顖炴煙椤斻劌娲ら柋鍥煏韫囥儳纾跨紒瀣╃窔濮婂宕掑顑藉亾閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞鐟滃繘寮抽敃鍌涚厽闁靛繈鍩勯悞鍓х磼閹邦収娈滈柡宀€鍠栭弻鍥晝閳ь剟寮搁悢琛″亾鐟欏嫭绀冮柣鎿勭節瀵鈽夐姀鈺傛櫇闂佹寧绻傚ú銊╊敇閻撳簶鏀介柣鎰絻閹垿鏌ｅΔ浣圭闁糕晝鍋ら獮瀣晜閽樺姹楅梺璇查濠€杈ㄦ叏閹绢啟澶娾攽鐎ｎ偀鎷虹紓鍌欑劍閿氬┑顔兼喘閺屻劑寮撮妸銈夊仐閻庢鍠栭…閿嬩繆閹间礁唯闁靛繆鍓濋弶鎼佹⒒娴ｈ櫣銆婇柛鎾寸箞閺佸鈹戦悙鍙夊櫡闁搞劏娅ｉ幑銏犫攽鐎ｎ亶娼婇梺鎸庣箓鐎涒晝绱為崼婵冩斀闁绘劘娉涢惃娲煕閻樻煡鍙勯柨婵堝仩缁犳盯骞樻担瑙勩仢妞ゃ垺妫冨畷銊╊敇閻曚礁鎮戦梻鍌氬€风欢姘焽瑜旈幃褔宕卞☉妯肩枃闂婎偄娲︾粙鎺楀磹閻㈠憡鐓欐い鏍ф閸嬫捇鏌￠埀顒佺鐎ｎ偆鍘介梺褰掑亰閸樿偐寰婇懡銈囩＜闁绘瑦鐟ュú锕傛偂閺囥垺鐓忓┑鐐茬仢閸旀粓鏌涚€ｃ劌鍔滄い銊ｅ劦閹瑩骞撻幒鎾搭啋闂備浇顕栭崰妤呮偡閳哄懎绠栨繝濠傚悩閻旇櫣纾兼俊顖滃帶缁插潡姊婚崒姘偓鐑芥嚄閸洍鈧箓宕奸妷銉ョ彉濡炪倖甯掗崐濠氭儗濞嗘挻鐓欓弶鍫濆⒔閻ｈ京绱掗埀顒勫醇閵夛妇鍘繝鐢靛仧閸嬫挸鈻嶉崱娑欑厓闂佸灝顑呴悘鎾煛瀹€瀣瘈鐎规洘甯掕灒闁告繂瀚～姘節閻㈤潧浠滈柟鍐查叄楠炲﹤顫滈埀顒€顕ｇ拠娴嬫闁靛繆鈧厖绨婚梺鑽ゅТ閹碱偊骞栭銈囩焾妞ゆ洍鍋撴慨濠呮閸栨牠寮撮悢鍛婄翻闂備焦鎮堕崝蹇撐涢崟顖ｆ晪闁挎繂顦介弫瀣煃瑜滈崜鐔风暦濞差亜鐒垫い鎺嶉檷娴滄粓鏌熼悜妯虹仴妞ゅ繆鏅涢々濂稿川鐎涙ǚ鎷洪梺鍛婄☉閿曘儲寰勯崟顖涚厱闁绘ê鍟块崫娲煙椤斿吋鍋ラ柛鈹惧亾濡炪倖甯掔€氼參鍩涢幋锔界厱闁挎棁顕ч獮鏍瑰鍛壕缂佺粯鐩畷妤呮嚃閳哄倸娅戦梻浣哥枃椤曆呯矓瑜版帒鏋侀柟閭﹀幖缁剁偤鎮楅敍鍗炲椤忓綊姊绘担钘夊惞濠殿喗娼欑叅闁冲搫鍟╅悞濠冪箾閸℃ê鐏╁☉鎾崇Ч閺岋絽螖閳ь剟鎮ц箛鏇犱笉闁告挆鈧崑鎾绘偡閺夋妫岄梺鍝ュУ閻楁洟顢氶敐澶樻晩闂佹鍨版禍鐐箾閸繄浠㈤柡瀣堕檮閵囧嫰寮撮崱妤佹悙闁绘挴鈧剚鐔嗛柤鎼佹涧婵洦銇勯銏″殗闁哄矉绲借灒闁稿繐鍚嬪В鍕攽閻愰潧甯舵い锕傛涧椤繒绱掑Ο璇差€撴繛鎾村嚬閸ㄦ娊宕濋幖浣光拺闁告繂瀚晶閬嶆煕閹惧娲寸€规洜澧楃换婵嬪磼閵堝懏鍊┑鐘灱濞夋盯顢栭崨鏉戠劦妞ゆ巻鍋撻柣鏍с偢瀵鈽夐姀鐘靛姶闂佸憡鍔楅崑鎾活敇閻熸壋鏀介柣鎰摠鐏忣厽銇勯鐘插幋鐎殿喖顭峰鎾閻樿鏁规繝鐢靛█濞佳兠归崒姣兼盯鍩勯崘顏嗙槇闂侀潧楠忕徊鍓ф兜妤ｅ啯鍊垫慨姗嗗亜瀹撳棛鈧鍠涢褔鍩ユ径鎰潊闁绘ɑ顔栭崯瀣⒑鐠囨煡鍙勬繛浣冲洤绠熼柨鐔哄Т闂傤垱銇勯幘鍗炵仾闁抽攱鍨块弻娑樷攽閸℃浠鹃梺闈╃悼閸忔﹢寮诲☉娆愬劅闁靛闄勯柨顓烆渻閵堝骸浜滅紒缁橈耿楠炴牞銇愰幒鎴炲祶濡炪倖鎸炬慨鐑藉储椤栫偞鈷戦柤濮愬€曟牎婵炲瓨绮堢划娆忕暦濠靛洦鍎熼柕蹇曞閸ゃ倝姊洪幖鐐插姶闁告挻宀搁崺娑㈠箣閻樼數锛滈柣搴秵閸嬫帡宕曢妷鈺傜厱閹兼番鍨规慨宥夋煛瀹€瀣М闁糕斁鍓濈换婵嬪磼濠娾偓閸濇姊绘担椋庝覆缂佹彃娼″畷妤€螣缂佹ê寮块梺閫炲苯澧撮柡灞界Ч瀹曠懓鈽夊▎鎰絽濠电偛鐡ㄧ划鎾剁不閺嵮屾綎濠电姵鑹鹃柋鍥煟閺冨洢鈧偓濠殿喖娲铏规喆閸曨偆绁烽梺纭呭Г缁捇骞嗛埀顒併亜韫囨挾澧曠紒鐘虫皑閹插摜鈧綆鍠栫壕濠氭煏韫囧鈧牠鎮″☉姘ｅ亾閸忓浜鹃柣搴秵閸撴盯鎯侀崼銉﹀€甸悷娆忓缁€鍐偨椤栨稑娴柛鈹垮劜瀵板嫰骞囬鍌ゆ敤闂備胶绮崝鏍亹閸愵喒鈧牠宕奸妷锔规嫽闂佺鏈悷銊╁礂鐏炰勘浜滄い鎾跺仧婢ф稒銇勯妸锝呭姦鐎规洦鍋婃俊鐑芥晝閳ь剚鎯旀繝鍥ㄢ拺闁革富鍘奸崝瀣亜閵娿儲鍤囬柟顔惧仱瀹曞綊顢曢悩杈╃泿闂備礁婀遍崕銈夊垂閻㈢鍑犻柕鍫濇缁犻箖鏌涜箛鏇炲付濠殿喖绉归弻鈥崇暆閳ь剟宕伴弽褏鏆︽い鎰剁畱鍞銈嗘⒐閸庤櫕鎱ㄦ總鍛娾拺闁硅鍔曢崥褰掓煕鐎ｎ剙浠ч柟骞垮灩閳规垹鈧綆鍋掑Λ鍐ㄢ攽閻愭潙鐏ョ€规洦鍓熷畷婊堝箥椤斿墽锛濇繛杈剧到閹碱偊顢撳畝鍕厱闁靛鍎抽崺锝団偓娈垮枦椤曆囧煡婢舵劕顫呴柣妯兼暩閺夋悂姊婚崒姘偓鎼佹偋婵犲嫮鐭欓柟鐑橆殕閸婂墎鈧箍鍎遍ˇ浼存偂閻旂厧绠归弶鍫濆⒔缁嬪鏌￠崱蹇旀珔闁宠鍨块、娆撴嚍閵夈儱鏀梻浣哄仺閸庣粯淇婇崶顒€绠查柛鏇ㄥ灠鎯熼梺鎸庢磵閸嬫挾鐥紒銏犵仸婵﹨娅ｇ槐鎺懳熼搹閫涙闂佽棄鍟存禍鍫曞蓟閻斿吋鍊婚柟瀛樺笚閸犳艾鈹戦纭峰伐闁圭⒈鍋呴弲銉╂⒑閹肩偛鍔€闁告劕澧介埀顒佹尦濮婄粯鎷呯憴鍕哗闂佸憡鏌ㄩ惌鍌炲箖濡　鏀介悗锝呭缁?
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
// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯闁靛／灞芥櫏闂備浇顕у锕傦綖婢跺⊕鍝勵潨閳ь剟骞冮敓鐘虫櫢闁绘灏欓悾鍝勨攽鎺抽崐鏇㈠箠韫囨稑鐤鹃柡灞诲劚缁犲湱绱掗鐓庡辅闁稿鎹囬獮鍥ㄦ媴鐟欏嫮銈板┑鐘殿暜缁辨洟宕戦幋锕€纾归柕鍫濐槸绾惧鏌涘☉鍗炲Ц婵炲樊浜濋崑鍕煟閹捐櫕鎹ｉ柣锝嗘そ閺岋綁鎮㈤崫銉﹀櫑闁诲孩鍑归崢浠嬪箞閵娿儺鐓ラ柛顐ゅ枔閸橀潧鈹戦悙鑼闁诲繑绻堝鎼佸Χ婢跺﹦顢呴棅顐㈡处缁嬫帡鍩涢幒鎳ㄥ綊鏁愰崼鐕佷哗闁汇埄鍨辩粙鎺楀箞閵婏妇绡€闁稿被鍊楅崥瀣倵鐟欏嫭纾搁柛銊ャ偢钘濈憸鐗堝笚閻撴瑦銇勯弮鍌滃彄妞ゅ繐鐗婇崑鈺呮煟閹达絾顥夐柣鎰躬閺屻劌鈹戦崱妯烘婵犮垼顫夊ú鐔煎蓟閿濆鍋勯柛婵勫劜閸Ｑ囨煟鎼淬垹鍤柛妯哄⒔閸掓帡宕奸妷銉у姦濡炪倖宸婚崑鎾绘煃鐟欏嫬鐏撮柟顔界懇閹崇娀顢楁担绋跨憥闂傚倷绀侀幉鈥愁潖瑜版帒鍨傞柟绋跨凹缁诲棝鎮楀☉娆樼劷闁荤喎缍婇弻宥堫檨闁告挻鐟╅幃楣冩倻閽樺）鈺呮煃閸濆嫸鏀婚柡鍛櫊濮婃椽宕ㄦ繝鍐ㄧ樂闁诲繒鍋犳慨銈呪枍濡ゅ懏鈷掗柛灞捐壘閳ь剛鍏橀幊妤呭醇閺囩偟鐤囬梺鍦亾濡炲潡寮€ｎ偁浜滈柟鎯у船閻忊晝鐥幆褜鐓奸柡宀嬬秮楠炲洭顢楅埀顒勫箖娓氣偓閺岋綁骞掗幋顖涙暰闂侀潧娲ょ€氫即鐛Ο鍏煎磯闁绘垶顭囬埀顒夊幗缁绘稓鈧稒顭囬惌瀣磼椤旇姤宕岀€殿喖顭烽幃銏㈡偘閳ュ厖澹曞┑鐐村灦閻燁垶鎮為悾宀€纾奸柣妯挎珪鐏忎即鏌曢崶褍顏紒鐘崇洴閺佹劙宕ㄩ鈧弸鍫ユ⒒娴ｅ憡鎯堥柡鍫墴閹嫰顢涢悙闈涚ウ闂佸湱鍎ら〃鍛不閼姐倐鍋撶憴鍕婵炲眰鍔戦獮濠囧冀椤撶啿鎷绘繛杈剧悼閻℃棃宕甸崘顔界厱闁靛鍎遍懜褰掓懚閻愮儤鐓曢柟鑸妽濞呭啰绱掔拠鍙夘棦闁哄瞼鍠栧鑽も偓闈涘濡差噣鏌涢悜鍡楃仸闁诡喖鍢查…銊╁礃椤庮垎鍕垫闁绘劖鎯屽▓姗€鏌嶈閸撱劎绱為崱娑樼獥婵°倕鎳庨悡鏇㈡煙閻戞ê娈憸鐗堝笚閺呮煡鏌涢弴銊ユ珮闁告柨鎳樺铏规偘閳ュ厖澹曢梻浣虹帛椤ㄥ懘鎮ч崟顒傤洸婵犲﹤瀚ㄦ禍婊堟煙閹佃櫕娅呴柣鎺斿劋娣囧﹪顢涘Δ鈧禍楣冩⒑鐠囧弶鍞夋い顐㈩槸鐓ゆ俊顖氬悑瀹曟煡鏌涢鐘插姎闁哄绶氶弻娑㈠箛闂堟稒鐏堢紒鐐劤閸氬骞堥妸銉庣喖宕崟顔肩厴闂備胶顢婂Λ鍕敄閸涙潙鐓橀柟杈鹃檮閸婇鈧懓澹婇崰鏍р枔閼哥數绡€婵炲牆鐏濋弸鐔兼煙閸涘﹤鈻曢柕鍡曠窔瀵粙顢橀悙鑼崺婵＄偑鍊栭悧妤冨垝瀹ュ鏅煫鍥ㄧ⊕閸嬧剝绻濇繝鍌氼仾妞ゃ儱鐗忛埀顒侇問閸犳盯顢氳椤㈡﹢宕楅悡搴ｇ獮婵犵數濮抽懗鍫曟倷婵犲洦鈷掑ù锝呮啞閸熺偞绻涚拠褏鐣电€规洘鍨块崺鈧い鎺戝閻撴洟鎮楅敐搴′簼閻忓繑澹嗙槐鎺旂磼濡偐鐣靛銈庡亝缁诲牓銆佸Δ鍛＜婵犲﹤瀚埀顒夊枟缁绘繈鎮介棃娑楀摋闂佽妞挎禍鐐垫閻愬搫鐒垫い鎺嶈兌缁犻箖鏌涘鍐ㄦ殶闁诡喚鍘ч…璺ㄦ喆閸曨剛顦ㄧ紓浣芥〃缁瑥鐣烽妸锔剧瘈闁告洦鍋勬俊鍥ㄧ節閻㈤潧啸闁轰焦鎮傚畷鎰邦敍閻愯尙顦ㄩ梺闈涱煭闂勫嫰顢欓崟顐熸斀闁绘ɑ顔栭弳顖炴煕閹惧绠撴い顏勫暣閹稿﹥寰勫Ο鑽ょ▉婵犵數鍋涘Ο濠冪濠靛纾婚柨鐔哄У閻撴稑顭跨捄渚剰妞ゆ洘绮嶇粋宥呪槈閵忊檧鎷绘繛杈剧悼閸庛倝宕甸埀顒勬⒑閸濄儱鏋欐繛澶嬫礋楠炴垿濮€閻欌偓濞笺劑鏌嶈閸撶喖骞冩导鎼晪闁逞屽墮閻ｇ兘宕￠悙鈺傤潔濠碘槅鍨甸崑鎰焽婢舵劖鈷掑ù锝囶焾椤ュ繘鏌涚€ｎ亝鍤囩€规洘妞藉畷姗€顢欓懖鈺嬬幢闂備胶鎳撴晶鐣屽垝椤栫偛鐤炬繝闈涱儐閻撴洟鎮橀悙闈涗壕闁汇劍鍨堕妵鍕箻鐎涙ǜ浠㈠┑顔硷龚濞咃絿妲愰幒鎳崇喖鎼归柅娑欐▕濠电姷顣藉Σ鍛村磻閸涘瓨鍋￠柍鍝勫閿涘倹绻濋悽闈涗沪闁割煈鍨跺畷鐟懊洪鍕幈闂佸湱鍎ら〃鍡涙偂閺囥垺鍊堕柣鎰綑缁€鍐熆鐟欏嫸鑰块柡灞界Х椤т線鏌涢幘鏉戝摵妤犵偛鍟村畷鎺戭潩鏉堛劍顔曢梻浣告贡婢ф顭垮Ο璇差嚤闁绘绮悡蹇擃熆鐠鸿櫣澧曢柣蹇婃櫅闇夋繝濠傛噹娴滈箖鏌熸笟鍨閾伙絿鈧懓瀚伴。锔界珶閺囥垺鈷掑ù锝堫潐閸嬬娀鏌涙惔銏㈢煉鐎规洏鍔戦、娆撳箚瑜嶉崣濠囨⒒閸屾艾鈧悂宕愰悜鑺ュ殑闁告挷绀侀崹婵囥亜閺嶎偄浠滅紒鈧径鎰厵闁绘垶蓱閳锋劕鈹戦娆忓祮闁哄被鍔戝鎾倷濞村浜鹃柟闂寸閸戠姴銆掑锝呬壕闂佽鍠楅〃濠囧极閹邦厽鍎熼柍銉ㄥ皺閹稿鈹戦悙鑼憼缂侇喖绉堕崚鎺楀箻鐠囪尪鎽曢梺缁樻⒒閸樠勫閻樼粯鐓曢柡鍥ュ妼婢х粯淇婇銈呬沪缂佺粯绋撻埀顒佺⊕椤洭鎯岄幒鏃傜＜闁绘ê纾晶鐢碘偓娈垮枛椤兘宕洪崟顖氱闁宠桨绶￠埀顒佹崌濮婃椽宕ㄦ繝鍕窗闂佺閰ｆ禍鍫曞春閸涘瓨鍋ㄩ柛娑樑堥幏娲⒑閸涘﹦绠撻悗姘煎幖閿曘垽骞嶉鍓э紲闁诲函缍嗛崑鍛焊閹殿喚纾奸柛灞剧☉濞搭噣鏌℃担绋挎殻鐎规洘甯掗～婵囨綇閳哄倹鐦為梻浣藉吹閸犳劗鍒掑畝鍕厐闁挎繂顦卞畵渚€鏌涢埄鍐︿粶婵℃彃鐗婄换娑㈠幢濡櫣浠煎銈嗘⒐濞茬喖寮婚敐鍡樺劅闁挎繂妫欏В鍕渻閵堝骸骞栭柣妤€妫濋幃娲敇閵忊檧鎷绘繛杈剧秬濞咃絿鏁☉娆嶄簻闁靛鍎查ˉ銏☆殽閻愭潙鐏寸€规洘鍎奸¨渚€鏌涢妶鍡樼闁宠鍨块幃鈺冣偓鍦Т椤ユ繈鏌熼婊冩灈婵﹥妞藉Λ鍐ㄢ槈鏉堛剱銈夋⒑閹肩偛濡芥俊鐐扮矙閺佹劙鎮欏顔兼倯闂佸憡渚楁禍婵嬪棘閳ь剟姊绘担鍝ユ瀮婵☆偄瀚灋婵°倕鎳忛崐鍫曟煏婢跺棙娅嗛柣鎾存礋閹鏁愭惔鈥茬盎婵犳鍠栫粔鍫曞焵椤掑喚娼愭繛鍙夘焽閹广垽宕奸妷锝傚亾娴ｇ硶妲堟俊顖炴敱椤秴鈹戦鏂よ€跨痪顓熸倐瀹曨垱鎯旈妸锔规嫽婵炶揪绲介幉锟犲箚閸喆浜滈柨鏂跨仢瀹撳棛鈧娲栫紞濠囧箖閻戣姤顥堥柤鎼佹涧濞搭噣鏌熼鐣屾噰闁诡喗绮岃灒闁绘挸楠哥粻浼存⒒閸屾瑧顦﹂柟娴嬪墲缁楃喎螖閸涱厼鐎梺鍝勵槼濞夋洟骞楅妷鈺傗拻闁稿本鐟﹂ˇ椋庣磼妲屾牕鏋ょ€垫澘锕ョ粋鎺斺偓锝庝簻缁ㄣ儵姊洪懖鈺佸付濠⒀冨缁傚秴鈻庨幘绮规嫽婵炴挻鍩冮崑鎾绘煃瑜滈崜娑㈠磻濞戙垺鍤愭い鏍ㄧ⊕濞呯娀鎮楀☉娅辨粍绂嶅鍫熺厪闊洤锕ら悞娲煙鐎电啸缂佹唻绠撻弻锝夋晲閸涱垳浼囬梺娲诲幗閻熲晠寮诲☉鈶┾偓锕傚箣濠靛洨浜鹃梻浣虹帛缁嬪牓寮插鍕攳濠电姴娲﹂崐閿嬨亜韫囨挸顏ら柛瀣崌瀵粙顢橀悢铚傜綍婵犲痉鏉库偓鎾绘嚄閸洘鍋￠悷娆忓缁诲棝鏌曢崼婵囧櫤闁革絽缍婇弻锝夊Χ鎼粹剝鐝濋梺鍝勬湰閻╊垶銆侀弴銏″亹闁圭粯甯掗～姘舵⒒娴ｇ懓鈻曢柡鈧潏鈺傛殰闁跨喓濮撮拑鐔哥箾閹寸偟鐓繛宀婁邯閺岀喓鎲撮崟顐㈩潓濡炪們鍎茬换鍫濐潖濞差亝鐒婚柣鎰蔼鐎氭澘顭胯閸ㄥ爼寮婚悢纰辨晩闁绘挸绨堕崑鎾诲锤濡ゅ啫绁﹂梺鍛婂姦閸欏骸螣娴ｈ櫣纾藉ù锝呮惈鏍＄紓浣割儐閸ㄥ潡宕洪妷锕€绶炲┑鐐灮閸犳挸鈽夐崹顐Ч閹肩话銈傚亾閸ф鈷掗柛灞捐壘閳ь剛鍏橀幊妤呭礈娴ｇ鐏婂銈嗙墱閸嬫稓绮堟径瀣╃箚闁靛牆鎳忛崳鍦棯閹勫仴闁哄本娲熷畷鐓庘攽閹邦厜锔剧磼閻愵剙绀冩い鏇嗗洠鈧箓宕稿Δ浣告疂闂傚倸鐗婄粙鎴︼綖瀹ュ鈷戠紓浣股戠亸顏堟煕鎼淬倕鐨洪柟骞垮灩閳规垹鈧綆鍋掑Λ鍐ㄢ攽閻愭潙鐏﹂柣鐔讳含濡叉劙鎮欏ù瀣杸闂佹枼鏅涢崯顐ゅ緤婵犳碍鐓欓柤鎭掑劤閻本淇婇崣澶婂妤犵偞甯掕灃闁逞屽墴閹?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴楠炴﹢鎳犻澶嬓滈梻浣规偠閸斿秶鎹㈤崘顔嘉﹂柛鏇ㄥ灠閸愨偓闂侀潧臎閸涱垰甯庨梻鍌欑劍閹爼宕濆鍥ㄥ床闁割偁鍎遍拑鐔兼煥濠靛棙鍟掗柣鏂跨殱閺嬪酣鏌熼鍡楀閸炴煡姊婚崒娆戝妽闁诡喖鐖煎畷婵嗙暆閸曨偆锛欓梺褰掓？閻掞妇绮婚幒妤佲拻濞达絼璀﹂悞鍓х磼缂佹﹩娈滅€规洘鍨块獮鍥级閹稿氦绶㈡繝娈垮枟閵囨盯宕戦幘缁樼厸閻忕偟鏅暩濡炪伇鍌滅獢闁哄本鐩獮妯尖偓闈涙啞閸ｄ即鎮楃憴鍕闁告梹鐟╅悰顕€骞囬鐔峰妳濠电娀娼ч悧蹇涙偩妤ｅ啯鈷掑ù锝呮憸缁夌儤淇婇銉︾《缂侇喖鐗婄粭鐔煎焵椤掑嫬绠栭柍銉︽灱濡插牊淇婇婊呫€婇柛瀣崌楠炴帡骞嬮鐔峰厞闂備焦瀵х换鍌炲箠瀹ュ棛鐝堕柡鍥╁枔缁♀偓缂佸墽澧楄摫妞ゎ偄锕弻娑氣偓锝庝簻椤忣厾鈧娲橀悡鈥愁嚕婵犳艾唯闁冲灈鏅涙禍楣冩煛閸愩劎澧曠紒鈧崘鈹夸簻闁哄倹瀵ч～宥囨喐閺冨牆钃熸繛鎴炃氬Σ鍫ユ煕濡ゅ啫浠уù鐙€鍨伴—鍐Χ閸℃鍙嗗┑鐐叉▕閸欏啫顕ｆ繝姘櫜闁糕剝锚閸斿懘姊洪棃娑㈢崪缂佽弓绮欒棢婵犻潧顑嗛埛鎴犵棯椤撶偞鍣圭悮姘辩磽娓氬洤鏋涢柤娲诲灠閻ｇ敻宕熼娑掓嫼闁荤喐鐟ョ€氼剟藟鐎ｎ亶鐔嗙憸搴ㄣ€冩繝鍌ゅ殨闁哄被鍎遍拑鐔兼煏婢舵稑顩柛妯哄船閳规垿鎮╅幍宥呯墦瀹曟垿骞樼紒妯煎幘闂佸憡鍔樼亸娆撳春閿濆洠鍋撶憴鍕闁告挾鍠栭獮鍐╃鐎ｎ亜绐涙繝鐢靛仧閸嬫挾鈧艾銈稿缁樻媴閸涘﹤鏆堢紓浣割儐閸ㄥ潡寮崘顔嘉ㄧ憸宀勬倿婵犲啨浜滈柟鏉跨埣濡绢噣鏌ｉ幘璺衡枅闁哄本鐩獮鍥敇閻愮數銈锋俊鐐€х徊鐣屽椤撱垹鐒垫い鎺嶇贰閸熷繘鏌涢悤浣镐喊鐎规洘鍎奸ˇ鎾煕閺冩挾鐣辨い顏勫暣婵″爼宕卞Δ鍐噯闂備胶顭堥敃銊ョ暦閸偆鐝堕柡鍥ュ灪鐎电姴顭跨憴鍕畵缂傚秴锕顐﹀箛閺夎法鍊為梺鍛婃尫缁€浣规叏婵傚憡鈷掑ù锝呮惈鐢爼鏌熼懞銉х煉妤犵偛锕獮姗€鎮滈埞鎯т壕闁挎洖鍊归崵鍕亜閺嶇數绋绘禍娑樷攽閻樺灚鏆╅柛瀣仜椤洤鈻庨幘鍐茬哎闂佸湱绮敮鎺旂矈椤愶附鈷掑ù锝呮啞鐠愨€愁熆瑜庨〃鍛粹€﹂崹顔ョ喖鎳栭埡鈧粭澶愭⒑鐟欏嫬鍔跺┑顔哄€濋幃鈥斥枎閹炬潙鈧灚绻涢幋鐑嗕紗闁硅揪绠戦悡鏇㈡煙閹殿喖顣奸柣鎾跺枛楠炴牜鍒掔憴鍕垫綉闂佺粯鎸搁妶鎼佸蓟閳ュ磭鏆嗛柍褜鍓熷畷浼村冀椤撶偟鐤囧┑鐘绘涧椤戝棝宕戠€ｎ喗鐓曟い鎰剁悼缁犳牠姊哄▎鎯у箻缂佽鲸鎹囧畷鎺戔枎閹存繂顬夐柣鐔哥矋濠㈡﹢宕锔光偓锕傚炊椤掍焦娅㈤梺缁橈耿濞佳呯矈閿旂晫绡€闁靛骏绲介悡鎰版煕閺冩挾鐣电€殿噮鍋婂畷鎺楁倷鐎电骞堥梻浣告惈濞诧箓鏁嬫繝纰樷偓鑼煓闁哄矉绻濆畷鎺懳旈埀顒佺妤ｅ啯鈷掗柛灞捐壘閳ь剛鍏橀幊妤呭礈娴ｇ鐏婂銈嗙墱閸嬫稓绮堟径鎰厸闁搞儯鍎遍悘顏堟煕鐎ｃ劌鈧繈寮婚弴鐔虹鐟滃秹宕愭繝姘闁靛濡囩粻楣冩倵閻㈡鐒鹃悽顖氱埣閺岋絾骞婇柛鏃€鍨块獮鍐晸閻樺啿浜滈梺绋跨箺閸嬫劙宕㈤幘缁樷拺閻庡湱濮甸妴鍐磼閳ь剚绗熼埀顒€顕ｉ幎鑺ュ€烽柣銏㈡暩閿涙粓姊洪柅鐐茶嫰婢ь垳鈧灚婢樼€氼噣鍩€椤掑﹦绁烽柛鏂款儑閼鸿鲸绻濆顓涙嫼闂傚倸鐗婄粙鎾存櫠閺囥垺鐓欓柛鎰叀閸欏嫰鏌涢埞鎯т壕婵＄偑鍊栧濠氬磻閹剧粯鐓熼煫鍥ㄦ⒒缁犵偤鏌ㄥ┑鍫濅哗缂佺姵鐩獮妯绘媴鐟欏嫨浠㈠銈冨灪濡啫鐣烽悢纰辨晝闁靛牆娲ら弰銉モ攽閻樺灚鏆╁┑顔惧厴瀵偊宕ㄦ繝鍐ㄥ伎闁诲海鏁哥涵鍫曞磻閹炬剚娼╅柣鎾抽缁秹鎮楃憴鍕濠电偛锕妴浣割潨閳ь剚鎱ㄩ埀顒勬煃闁款垰浜鹃梺褰掓敱濡炶棄顫忛搹瑙勫厹闁告侗鍘哄Ч妤呮⒑缁嬫鍎愮紒瀣笚缁岃鲸绻濋崶褏顦ч梺鍏肩ゴ閺呮繈顢欓弴銏♀拺閻犲洠鈧櫕鐏嶉梺璇″枛閻栫厧鐣烽幇鏉垮唨妞ゆ挾鍠愬▍鏍⒑缁嬭法鐏遍柛瀣仱閸╂盯骞掗幊銊ョ秺閺佹劙宕堕埡鍌涘晵婵犵數鍋涢崥瀣礉濡ゅ懎鐒垫い鎺戝枤濞兼劖绻涢崣澶涜€块柡浣稿暣婵偓闁靛牆鍟犻崑鎾存媴閸撳弶鍍甸梺褰掝暒閻掞箓鐎风紓鍌氬€搁崐椋庢閿熺姴绐楁繛鎴欏焺閺佸鎲搁弮鍫濈畺濡わ絽鍟崐濠氭煢濡警妲洪柛濠勫仱濮婃椽妫冨☉杈╁姼闂佺瀛╅悡锟犲箖濡や降鍋呴柛鎰ㄦ杹閹锋椽鏌ｉ悩鍙夊鐟滄澘鍟村畷鍨綇閳哄啰锛濋悗骞垮劚閹冲繘藟閵忊懇鍋撶憴鍕；闁告鍟块锝嗙鐎ｅ灚鏅ｉ梺缁樻煥閻ㄦ繈宕?
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

// codex_cli_only 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闂囧鏌ㄥ┑鍡欏妞ゅ繒濮风槐鎺楀焵椤掍胶绡€闁稿本顨嗛弬鈧梻浣虹帛閿氱€殿喖鐖奸獮鏍箛椤斿墽锛濇繛杈剧到瀵泛鈻嶆繝鍕ㄥ亾濞堝灝鏋涢柣鏍帶閻ｇ兘鎮℃惔顔藉兊濡炪倖鎸鹃崰鎾广亹閸℃稒鈷掑ù锝呮啞閹牊淇婇锝囨噮闁告帗甯￠獮妯兼嫚閼艰埖鎲伴梻浣瑰缁诲倿藝椤栨粎涓嶆繛鎴欏灪閻撶喐淇婇妶鍌氫壕闂佺粯顨呴敃顏堝蓟閵娾晩鏁囬柕蹇婃閹锋椽姊虹涵鍛汗闁稿绋掓穱濠囨偩瀹€鈧壕濂告偣閸ヮ亜鐨哄褝濡囬埀顒侇問閸犳牠鈥﹂悜钘夋瀬闁归偊鍘肩欢鐐测攽閻樻彃顏撮柛姘嚇濮婄粯鎷呴悷閭﹀殝缂備浇顕ч崐姝岀亱濡炪倖鎸鹃崐锝呪槈閵忕姷顦板銈嗘尵婵兘鏁嶅┑鍥╃闁瑰墽顥愭竟妯荤箾鐏炲倸鈧牜绮嬪鍜佺叆闁告侗鍨抽敍婊勭節閵忥絾纭鹃柨鏇缁棃鎼归崗澶婁壕婵炲牆鐏濋弸娑欍亜椤撶姴鍘存鐐插暣婵偓闁靛繒濮甸悗鎶芥煛婢跺﹦澧戦柛鏂跨Т閳诲秹寮介鐔叉嫽婵炶揪缍€濞咃絿鏁☉銏＄厱闁哄啠鍋撻柨姘亜閺囶亞绉鐐搭焽閹风娀鎳犻鍌氱倞婵犵數濮烽弫鍛婃叏閺夋嚚娲晝閸屾氨鍘遍梺纭呮彧闂勫嫰鎮￠弴鐔虹瘈闂傚牊绋戦鈺呮煕閺冣偓閼归箖婀侀梺缁樏壕顓㈡儗婵犲啨浜滄い鎾偓鍐插Х濡炪倧闄勯悡鈥愁潖閾忚鏆滄い鏂垮⒔琚﹂梻浣虹帛娓氭宕板Δ鈧銉╁礋椤栨氨鐤€濡炪倖甯掗崑鍡涘疮閸パ€鏀介柣鎰煐瑜把呯磼閹绘帗鍋ユ鐐诧躬楠炴鈧潧澹婂ú鎼佹⒒娓氬洤澧紒澶屾暬閸╂盯骞嬮敂鐣屽幈闂佸湱鍋撳娆戝緤鐠囪鐟邦煥閸涱厺妲愰梺鍝勬湰閻╊垶骞冮埡鍛優妞ゆ劑鍨婚敍鎾绘⒑閼恒儔鎴犳崲閸愵喖桅闁告洦鍨扮粻濠氭偣妤︽寧銆冩慨瑙勵殜濮婅櫣鎮伴垾鍏呭濠电偛顕慨鎾敄閸℃稒鍋傞柣鏂垮悑閻撴瑩姊洪銊х暠闁诲繒濞€閺岋綁濡搁妷锔藉創濡炪値鍙€閸庡藝閹惰姤鐓熼柍鍝勶工閻忥妇鈧鍠涢褔鍩ユ径鎰潊闁绘ɑ鐗戦弲鐘诲蓟閺囩喓鐝舵い鏍ㄧ閸熸鏌曟繛鍨壔闁哄啫鐗婇崑鎰版⒒閸喓鈼ョ紒顔煎缁辨挻鎷呴搹鐟扮闂佹寧娲忛崹浠嬫偘椤曗偓瀹曟﹢顢欓懖鈺佸箰闂備礁鎲￠崝锔界閻愮儤鏅繝濠傚暊閺€浠嬫煃閽樺顥滈柣蹇ョ秮閺屾稑螣閻樻彃鏋ら柛銈嗘礋閺岀喖骞嗛弶鍟冩捇鏌嶉柨瀣伌闁诡喖缍婂畷鍫曞Ω閵壯呫偡闂備焦鎮堕崐娑欑椤掑倹顫曢柟鐐墯閸氬鏌涘☉鍗炴珮婵☆偓绠撻幃妤冩喆閸曨剛顦ㄧ紓渚囧枛閻倿鏁愰悙娴嬫斀闁割偆鍠庣壕顖炴⒑閹呯妞ゆ洘鐗曞嵄鐟滅増甯楅埛鎺懨归敐鍫燁仩閻㈩垱鐩弻娑㈠籍閹惧墎鏆ら悗瑙勬礃缁矂鍩ユ径濠庢僵妞ゆ巻鍋撶悮锝夋⒒娓氣偓閳ь剛鍋涢懟顖涙櫠鐎涙ɑ鍙忓┑鐘插暞閵囨繃淇婇銏犳殭闁宠棄顦垫慨鈧柕蹇曞У閵囨繄绱撻崒姘偓鐑芥倿閿曚降浜归柛鎰ㄦ櫆濞呯娀鏌﹀Ο渚Т婵℃彃鐗撻弻锝夊閻樺啿鏆堥梺绋匡躬閺€閬嶅Φ閸曨垰绠ｆ繝闈涙缁嬪繒绱撴担闈涘妞ゎ厼鍢查～蹇撁洪鍕炊闂佸憡娲﹂崜娆撳磻瀹ュ鈷戦柛娑橈攻閻撱儵鏌ｉ鐐测偓鍧楀箖娴兼惌鏁嬮柍褜鍓欓锝嗙鐎ｅ灚鏅ｉ梺缁樻煥閹碱偆鏁Δ鍛拺闁煎鍊曞瓭濠电偞绁撮弲婵囩缁嬪簱鏋庨煫鍥ㄦ⒐閻﹀孩绻濈喊澶岀？闁稿鍨垮畷鎰槾缂侇喖顭烽獮瀣攽閹惧厜鍋撻崸妤佲拺妞ゆ巻鍋撶紒澶嬫尦閹偤宕归鐘辩盎闂佸湱鍎ら崹鐢稿几閵堝棛绠鹃柛蹇曞帶婵牏绱掔紒妯兼创妤犵偞鎹囬獮鎺楀籍閸屻倕鏅梺璇叉唉椤煤濡ソ娲偄閻撳氦鎽曢梺绯曞墲椤ㄥ繘宕ョ€ｎ喖绠规繛锝庡墮閻忣亝绻涢崨顓犘㈤柍瑙勫灴閹晝绱掑Ο濠氭暘闂備胶顭堥敃銈夆€﹂悜钘夌伋闁挎洖鍊搁柋鍥ㄧ節閸偄濮囬柡鍛Т閳规垿鎮╃紒妯婚敪濠电偛鐪伴崹铏圭矉瀹ュ拋鐓ラ柛顐ゅ枔閸樼敻姊虹涵鍜佹綈闁告梹鐗滃☉鐢告倷椤戝彞绨婚柟鑲╄ˉ閸撴繂锕㈤弶澹冲酣宕惰闊剚顨ラ悙瀛樺磳妤犵偞甯″顕€宕掗崒妯哄闁宠鍨块、娆戞兜瀹勬澘顫犵紓鍌欑贰閸ｎ噣宕归崼鏇犲祦闊洦绋戝婵嬫倵濞戞顏呯椤撱垺鈷掑〒姘搐瀵法绱掗悩鍐茬仴闁崇粯鎸搁埞鎴﹀醇閵忣澁绱抽梻浣侯焾閺堫剟鎮疯钘濋柨鏂款潟娴滄粓鏌熺€涙绠栨い銉ｅ灮閳ь剝顫夊ú鏍礊婵犲倻鏆︾紒瀣嚦閺冨牆鐒垫い鎺戝€绘稉宥夋煛瀹ュ骸浜濋柛鐘冲姍閺岋繝宕掑┑鍥┿€婇梺鍝勬４缁犳捇寮婚弴鐔虹闁绘劦鍓氶悵锕傛⒑鏉炴壆顦︾紒澶屾暬楠炲牓濡搁妷顔藉缓闂佺硶鍓濋〃鍛达綖椤愶絿绠鹃悗娑欘焽閻棝鏌涘Δ鈧崯鍧楋綖韫囨拋娲敂閸涱厺鐢婚梻浣告惈椤︽壆鈧瑳鍥х９妞ゆ牜鍋為埛鎴澝归崗鑲╂噮缂佸娼ч湁婵犲﹤绨肩花濠氭煟閿濆懎妲婚柍瑙勫灴瀹曢亶鍩℃担鐟扳偓顖氣攽閻橆喖鐏辨繛澶嬬洴閺佸啴鏁冮崒娑欐珫濠电偞鍨崹娲煕閹达附鈷掗柛顐ゅ枎瀹曠喖鏌涢鐘插姎闁汇倗鍋撶换婵囩節閸屾粌顤€闂佺顑呴崐鍧楁偂椤愶箑鐐婇柕濠忕畱閻ㄦ垿姊虹悰鈥充壕婵炲濮撮鍡涙偂閻斿吋鐓欓梺顓ㄧ畱閻忕娀鏌ｉ妸銉︽儓闂囧鏌ｅ鍡楁灈闁诲繑鐓￠弻锛勪沪閻ｅ睗銉︺亜瑜岀欢姘跺蓟濞戙垹绠婚柡澶嬪灥閹界敻姊洪棃娑欘棞闁哥噥鍨抽幑銏犫攽鐎ｎ亞鍊為悷婊冪箻閹潡宕惰閺€浠嬫煟濡偐甯涙繛鎳峰嫮绠鹃悘鐐插€搁悘鑼偓瑙勬礃閸ㄥ灝鐣烽妸褉鍋撳☉娆樼劷闁告﹢浜跺铏规兜閸涱厾鍔烽梺鍛婃煥缁夋挳鍩㈠澶婂窛妞ゆ帞鍘х紞濠囧箖椤忓牆鐒垫い鎺戝閸ㄥ倿鎮规潪鎷岊劅婵炲吋鐗楃换娑橆啅椤旇崵鐩庣紓浣哄Т缂嶅﹪寮婚敐澶婄疀妞ゆ挾鍋熺粊閿嬬節绾版ê顫掗柛銉ｅ妼閳ь剙鐏氶〃銉╂倷閼碱兛铏庨梺鍛婃⒐绾板秵绌辨繝鍥舵晝闁挎繂娲﹂崳浼存倵濞堝灝鏋︽繛鍛礈閳ь兛绲婚崑鎰板焵椤掍胶鈯曢懣銈夋煙妞嬪海甯涚紒缁樼⊕濞煎繘宕滆閸╁矂姊虹涵鍜佸殝缂佺粯绻堥悰顕€宕卞☉姗€鍞堕梺闈涱檧婵″洭宕㈤幆褉鏀介柣鎰硾閽勫吋銇勯弴鍡楁处閸婂爼鏌ｅΟ鑲╁笡闁稿鍊块弻鏇㈠醇濠靛浂妫戞繛瀛樼矋缁捇寮婚悢鐓庣闁归偊鍘鹃妴鎰版⒑閸涘﹥宕屽ù婊冪埣瀵鈽夐姀鐘栄囨煕閳╁喚鐒芥い锔哄姂濮婃椽妫冨☉娆愭倷闁诲孩纰嶅姗€顢氶敐澶婄骇闁瑰疇娅曞Λ鍐春閳ь剚銇勯幒宥堝厡妞も晜鐓￠弻锝夊箛闂堟稑顫紓浣叉閸嬫挾绱撴担鍝勪壕婵犮垺锕㈣棟閺夊牃鏅涢ˉ姘舵煕瑜庨〃鍡涙偂閺囥垻鍙撻柛銉ｅ妽閻庡ジ鏌熺€电浠ч柍鐟扮Т閳规垿鎮╅崣澶婎槱闂佺顑愭禍婵嬪Φ閸曨垰鍐€闁汇垻鏁歌摫闂傚倸顦鍛村煘閹达附鏅柛鏇ㄥ亗閺夘參姊虹粙鍖℃敾闁绘濮撮锝嗙節濮橆儵鈺冩喐韫囨稒鍎楅柟鍓х帛閻撶喖鏌ｉ弮鍌氬付濞存粎澧楃换娑㈠礂閼测晛顫х紓浣虹帛缁嬫垿顢欒箛娑辨晩闁煎鍊楀▔璺ㄧ磽閸屾瑨鍏屽┑顔藉▕瀹曪繝骞庨悾灞界ウ闂佸搫绉查崝宥囩礊閸ヮ剚鐓曢柟鐐殔閸熶即鎮″Δ浣虹瘈缁剧増锚婢ц尙鎲搁弶鍨殻闁诡啫鍕瘈闁告洦鍓欐惔濠傗攽閻樼粯娑фい鎴濇钘熼柛顐ゅ枔缁犻箖鏌熺€电鈧垵顫濈捄铏癸紱闂佺懓澧界划顖炲磻閿熺姵鈷戞い鎾卞姂濡绢喚绱撳鍛村弰婵﹥妞介幃鈩冩償椤旂晫绋愰梻浣侯焾閿曘儱煤閻旈鏆﹂柣鐔稿閺€浠嬫煕閵夛絽鍔欑紒銊ヮ煼濮婅櫣鎲撮崟顐ゎ槰濠电偛顦伴惄顖氼嚕閸涘﹥鍎熼柨婵嗘川閿涙粓姊洪崨濠冨矮缂佲偓娴ｅ湱顩烽梺顒€绉甸悡娑氣偓鍏夊亾闁逞屽墴瀹曚即寮介鐐舵憰闂佹悶鍎洪崜姘跺疾濠靛鐓冪憸婊堝礈濮樿泛鐤鹃柤鎼佹涧椤曢亶鎮楀☉娆樼劷闁告ü绮欏娲捶椤撶偘澹曞┑鐐插悑閻熲晠骞冨Ο灏栧亾濞戞鎴﹀矗韫囨挴鏀介柣妯哄级閸ｇ儤銇勮箛鏇炐ョ紒杈ㄥ浮椤㈡瑩鎮剧仦鎯ф珰婵°倗濮烽崑鐐哄礉閺嶎厼绠氶柡鍐ㄧ墕椤懘鏌嶉妷锕€澧ù鐓庢濮婂宕掑顑藉亾瀹勬噴褰掑炊瑜忛弳锕傛煟閵忋埄鐒鹃柣銈庡枟缁绘盯宕卞Ο璇茬闂佺粯鍔曢敃顏堝蓟閺囩喓绠鹃柛顭戝枛婵秹姊哄Ч鍥х仼妞ゎ厼鍢查～蹇曠磼濡偐鎳濋梺閫炲苯澧い顓炴穿椤︽挳鏌熼獮鍨伈妤犵偞甯￠獮姗€寮堕幋婵呯礋濠碉紕鍋戦崐鏍箰妤ｅ啫纾归柨婵嗩槹閺呮彃顭跨捄鐚磋含闁哥偛鐖煎娲濞戣鲸顎嗘繝纰樷偓铏枠鐎规洏鍨介幃浠嬪川婵炵偓瀚奸梻浣告啞缁嬫垿鏁冮妷褉鍋撻崹顐ゆ噰闁诡喗锕㈤幃娆撴嚋濞堟寧顥夋俊鐐€ら崑鍛暆缁嬫娼栫紓浣股戞刊鏉戔攽椤旇棄濮€闁稿鎹囬崺鈧い鎺戝閻撴洟鎮楅敐搴′簼閻忓浚鍙冮弻宥囨喆閸曨偆浼岄梺璇″枓閺呮盯鎮鹃悜钘夌倞闁肩鐏氶鎰版⒒閸屾瑨鍏岀紒顕呭灦楠炴劖銈ｉ崘銊х崶闂佸搫绋侀崢濂告嫅閻斿吋鐓冮柕澶堝劤閿涘秹鏌￠崱鎰偓鏍Φ閸曨垰妫橀柛顭戝枟閸婎垶姊洪棃鈺冨埌缂傚秴锕濠氭晲婢跺﹦顔掗柣鐘烘閸庛倝鎮橀崼婵冩斀妞ゆ梻銆嬮弨缁樹繆閻愯埖顥夐柣锝囧厴椤㈡洟鏁冮埀顒傜矆鐎ｎ偁浜滈柡鍐ㄥ€甸幏鈥趁归悡搴℃殻婵﹦绮幏鍛瑹椤栨稒鏆炴繝鐢靛仜閹冲繐煤閻旇偐宓侀柟鐗堟緲缁€鍫㈡喐瀹ュ鍨傛繝闈涱儐閸嬶綁鏌涢妷锝呭闁靛牆鐡ㄦ穱濠囧箵閹烘柨顤€缂備胶绮惄顖氱暦閸楃倣鐔煎传閸曞灚缍嬮梻鍌欑閹诧繝寮婚妸鈺佺疇婵☆垯璀﹀鏍ㄧ箾瀹割喕绨荤紒鐘卞嵆楠炴牗娼忛崜褎鍋ч梺浼欑秮濞佳囧煘閹达附鍊烽柤纰卞墯閸曢箖姊虹粙鍖℃敾缂佽鐗嗛悾宄懊洪鍕姦濡炪倖甯婇梽宥嗙濠婂牊鐓欓柣鎴灻悘銉╂煃瑜滈崜娆撳箟閿涘嫮鐭夐柟鐑樻尰缂嶅洭鏌曟繝蹇曠暠鐎?
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
	Model      string // 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇顓滀簻闁归偊鍠栭弸搴∶瑰鍫㈢暫闁哄被鍔戝鎾倷濞村浜鹃柟闂寸劍閸婂嘲鈹戦悩鎻掓殧濞存粍绮撻弻鐔煎传閸曨剦妫炴繛瀛樼矊婢х晫妲愰幘瀛樺闁荤喐婢橀～宥咁渻閵堝啫濡奸柨鏇ㄤ邯閹即顢氶埀顒€顕ｆ禒瀣垫晣闁绘劖顔栭崯鍥ㄤ繆閻愵亜鈧牠骞愰悙顒佸弿閻庨潧鎲￠弳婊堟煏婵炑冩噽閿涙繈姊虹粙鎸庢拱婵ǜ鍔嶉悧搴ㄦ⒒娴ｈ櫣甯涙い銊ョ墛缁绘盯鍩€椤掑倵鍋撳▓鍨灈妞ゎ厾鍏橀獮鍐閵堝懐顦ч梺鍏肩ゴ閺呮稒鎱ㄥú顏呪拻濞达絿鎳撻婊勭箾閹绘帞效鐎规洘鍨块獮妯肩磼濡厧骞堥梻浣告惈閸燁垶骞愭ィ鍐╁仼闁汇垹鎲￠悡娑氣偓鍏夊亾閻庯綆鍓涜ⅵ闁诲孩顔栭崳顕€宕抽敐鍛殾闁圭儤鍩堝鈺傘亜閹达絾纭堕悽顖涚洴濮婂宕掑▎鎰偘濡炪倖娉﹂崨顔煎簥闂佸綊鍋婇崰鏍夊顓滀簻闁规儳宕悘鈺呮煟閹烘垹浠涢柕鍥у楠炴帡骞嬪┑鍥╀壕婵犵數鍋涢崥瀣礉閺嶎偅宕叉繛鎴欏灩閻顭跨捄楦垮闁愁亜缍婇弻锝夊箻閸楃偐鍋撳┑瀣摕闁挎繂鎲橀悢灏佹瀻闁诡垎鍕靛敳缂傚倷鑳堕崑鎾诲磿閹惰В鈧箓鎮滈挊澶庢憰闂佺粯姊婚崢褏绮堥崼銏″枑闊洦娲橀弳婊勭箾閹存瑥鐏柣鎾存礀閳规垿鎮╅幓鎺嶇敖闂佺粯鎸搁妶鎼佸蓟閿涘嫪娌悹鍥ㄧゴ閸嬫挸鈹戠€ｎ剙绁︽繛鎾村焹閸嬫挻顨ラ悙鏉戞诞妤犵偛顑呴埞鎴﹀幢閳哄倹绗庡┑鐘垫暩婵兘寮幖浣哥；婵炴垯鍨圭粻鐘荤叓閸ャ劎鈯曢柛瀣儔閺岋絽螣閻戞ǚ鏋欏┑鐐插悑閻楁鎹㈠☉姗嗗晠妞ゆ棁宕甸惄搴ｇ磽娴ｆ彃浜鹃梺鍓插亖閸庢煡鎮″▎鎴犵＝濞达絽顫栭鍡欑當闁稿本绮庣壕濂告煏婵犲繘妾悘蹇ラ檮閹便劍绻濋崘鈹夸虎濡ょ姷鍋為悧鐘诲箖濞嗘挸绠甸柟鐑樼箘閳ь剙鐡ㄧ换婵堝枈婢跺瞼锛熼梺绋款儑婵鐏嬮梺鍛婃处閸ㄥジ寮€ｎ喗鐓ユ繝闈涙缁€宀勬煕鐎ｎ偅宕岀€规洜鍏橀、姗€鎮欓幇鈺佸姦闁哄矉缍佹俊鍫曞磼濮橆偄顥氶梻鍌氬€搁崐鐑芥嚄閸洖绠犻柟鎯х畭娴滃綊鏌ｉ幇顔煎妺闁搞倕瀚伴弻娑㈩敃閿濆棛顦ョ紓浣哄Х婵炩偓闁哄瞼鍠栧畷婊嗩槻闁告棑绠撻弻宥堫檨闁告挶鍔庣划濠氬箻缂堢姷绠氶梺褰掓？缁€渚€鎮″☉妯忓綊鏁愰崟顕呭妳濠德ゅ皺婢ф绌辨繝鍥ㄥ€锋い蹇撳閸嬫捇寮介‖鈩冩そ瀵粙顢橀悙鐢垫瀮濠电姰鍨奸崺鏍礉閺囩姴顥氬┑鐘崇閻撴瑩鏌ｉ幇闈涘缂佹劗鍋ら弻锟犲礃閵娧冾暫閻庣懓鎲＄换鍐Φ閸曨垰绫嶉柍褜鍓熷畷鏇㈠箮閻ｅ苯绁﹂悗骞垮劚閹冲寮ㄦ禒瀣厽婵☆垵鍋愮敮娑欑箾閹冲嘲娲﹂悡鏇炩攽閻樻彃鏆為柛濠冨姈椤ㄣ儵鎮欏顔解枅闂佽桨鐒﹂幑鍥箖閳哄懎鐭楀璺哄閸嬫捇顢楅崟顑芥嫽婵炴挻鑹惧ú銈嗘櫠椤斿墽纾煎璺烘湰閺嗩剟鏌熼鍏夊亾閺傘儲鐎婚梺瑙勫閺呮瑧鑺辨繝姘拺闁告繂瀚ⅹ闂佸憡鏌ㄩ柊锝夊春婵犲洤鍗抽柣鏃傜節缁ㄥ姊洪棃娑辨Ф闁稿氦娅曢弲鍫曞閵堝棛鍘撻悷婊勭矒瀹曟粌顫濇０婵囨櫓闂佸搫绋侀崢濂告⒒椤栨稏浜滈柡鍥殔娴滈箖鎮楃憴鍕缂侇喗鎹囬獮鍐閵堝棗娈愰梺瀹犳〃閼冲爼濡堕敃鍌涒拻濞达絽鎽滅粔娲煕鐎ｎ亷韬€规洏鍨虹粋鎺斺偓锝庝簽閿涚喖姊虹紒妯荤叆闁告艾顑夐幃锟犲礃椤旂晫鍙冨┑鈽嗗灟鐠€锕€危鐟欏嫨浜滈柕澶涘閹冲懐绱掓潏銊ョ瑲婵炵厧绻樻俊姝岊槻闁冲嘲鑻埞鎴︻敊绾嘲浼愬銈庡幖閸㈡煡鎮鹃悜钘夌闁瑰瓨姊归悗濠氭⒑鐟欏嫭鍎楅柛妯衡偓鐔插徍闂傚倸鍊搁崐椋庣矆娴ｉ潻鑰块梺顒€绉撮悡鏇㈡煕椤愮姴鍔氱痪鎹愵嚙閳规垿鎮╅幓鎺嗗亾婵犳艾姹查柨鏂款潟娴滄粓鏌″搴′簻濞寸姵绮撻獮鍡涙偄閸忓皷鎷洪柣搴℃贡婵敻濡撮崘顔界厾闁告劘灏欓崺锝夋煛娴ｅ摜效闁诡喓鍨藉畷妤呮嚃閳轰礁绠版繝鐢靛О閸ㄧ厧鈻斿☉銏℃櫇闁挎棁濮ゅ▍鐘绘煙閹殿喖顣奸柣鎾寸洴閺屾稑鈽夐崡鐐寸亾闂佹椿鍘奸敃顏堝蓟閿濆鏅查柛銉戝啫绠ｉ柣搴㈩問閸犳牠鈥﹂柨瀣╃箚闁绘垼濮ら弲婊堟煙椤栧棗鍟伴鍥⒒閸屾艾鈧绮堟笟鈧獮澶愭晸閻樿尙顦梺纭呮彧缁犳垹绮堟径鎰婵烇綆鍓欓悘顏呯箾閸涱垰鈻堥柡宀嬬節瀹曞爼濡烽妷褌鐥梻浣虹帛閹歌煤濡吋宕叉繛鎴欏灩楠炪垺淇婇妶鍛櫧鐎规洦浜缁樼節鎼粹€斥拻闂佸憡鎸鹃崰鏍ь嚕婵犳碍鏅柛鏇ㄤ簼閸曞啴姊洪崨濠冨瘷闁告洦鍋勯悘鈥斥攽鎺抽崐妤佹叏閻戣棄纾婚柣鎰劋閺咁亝淇婇悙顏勨偓銈嗙濠婂牆鐤柟缁㈠枛缁犳牗绻涢崱妯绘儎闁轰礁妫楅…璺ㄦ崉娓氼垰鍓梺鍝勵儏閻楀繒妲愰幘璇茬＜婵﹩鍏橀崑鎾搭槹鎼淬埄鍋ㄩ梺璺ㄥ枔婵绮ｅΔ浣瑰弿婵☆垱瀵х涵楣冩煛鐎ｂ晝绐旈柟顔肩秺瀹曟儼顦抽柍缁樻礃缁?	// BillingModel is the model used for cost calculation.
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
	userPlatformQuotaRepo UserPlatformQuotaRepository

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
	openaiAccountRuntimeBlockUntil      sync.Map // key: int64(accountID), value: time.Time
	openaiOAuth429WindowStartUnixNano   atomic.Int64
	openaiOAuth429WindowCount           atomic.Int64
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
	userPlatformQuotaRepo UserPlatformQuotaRepository,
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
		userPlatformQuotaRepo: userPlatformQuotaRepo,
		responseHeaderFilter:  compileResponseHeaderFilter(cfg),
		codexSnapshotThrottle: newAccountWriteThrottle(openAICodexSnapshotPersistMinInterval),
	}
	if rateLimitService != nil {
		rateLimitService.SetAccountRuntimeBlocker(svc)
	}
	if openAITokenProvider != nil {
		openAITokenProvider.SetAccountRuntimeBlocker(svc)
	}
	svc.logOpenAIWSModeBootstrap()
	return svc
}

// ResolveChannelMapping 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍嵁瀹ュ鏁婄痪鎷岄哺濮ｅ姊绘担渚劸妞ゆ垶鍨归幑銏犫攽閸♀晛娈ㄩ梺鍓插亝濞叉牠鏌嬮崶銊﹀弿婵妫楅獮妤呮煟濠靛洦鈷掔紒杈ㄦ尰閹峰懘鎮剧仦鐣屽闂備胶顭堥敃銉ッ哄┑瀣€堕柛鎰靛枟閳锋垿鏌熺粙鎸庢崳缂佺姵鎹囬弻鐔煎礃閺屻儱寮伴悗娈垮枟婵炲﹪骞冨▎鎾村€绘俊顖滃帶楠炲牆鈹戦悩鍨毄濠殿喖顕埀顒佸嚬閸欏啫顕ｉ幎绛嬫晢闁告洦鍓涢崢鎼佹倵閸忓浜鹃柣搴秵閸撴盯寮抽悩缁樼叄濞村吋鐟ч崚浼存煏閸℃ê绗掓い顐ｇ箞閹剝鎯旈敐鍕暰闂備胶鍘у鍫曟偋閻樺樊娼栫紓浣诡焽閻熷綊鏌嶈閸撶喖宕洪埀顒併亜閹烘垵鈧憡绂掑鍫熺厾婵炶尪顕ч悘锟犳煛閸涱厾鍩ｆい銏＄洴閹瑩寮堕幋婊呯处濠碉紕鍋戦崐鏍偋閹捐纾规俊銈呭暙婵剟鏌嶈閸撴瑨鐏冮梺缁橈耿濞佳勭濠婂牊鐓曢柣鏂挎啞鐏忥妇鈧娲樺浠嬪极閹剧粯鍋愰柤纰卞墻濡蹭即姊绘担鍝ユ瀮婵℃ぜ鍔庡▎銏ゆ晸閻樿尙鍔﹀銈嗗笂閼宠泛煤鐎涙ɑ鍙忓┑鐘插暞閵囨繄鈧娲忛崝宥囨崲濠靛纾兼慨妯哄悑缂嶅鈹戦悩娈挎毌婵℃彃鎳樺畷瑙勫鐎涙ɑ娅囬梺闈涚墕閻楀繑绂嶆潏銊х瘈闁汇垽娼у瓭闂佹寧娲忛崐婵嬬嵁婵犲洤绠涙い鏃傗拡濡粍绻涚€电孝妞ゆ垵妫濋幃锟犲即閵忊€斥偓鍫曟煟閹邦厼绲婚柍閿嬫閺岀喖宕橀崣澶樻＆闂佸搫鏈粙鏍不濞戙垹绠婚柛鎾茬婵亶姊绘担绛嬪殐闁哥姵顨婇妴鍐川鏉堝墽绋忓┑鐘绘涧椤戞劙寮崱娑欑厱閻忕偛澧介惌銈夋煕閻斿鍎旈柡宀嬬稻閹棃濡舵惔銏㈢Х婵犵數鍋涘鍓佸垝閹惧磭鏆﹀ù鍏兼綑閸愨偓濡炪倖鎸鹃崑鐐参ｉ鍕拺闂侇偆鍋涢懟顖涙櫠椤栫偞鐓忛柛銉戝喚浼冨Δ鐘靛仦鐢帡顢樻總绋块唶婵犻潧妫楅懙鎰版⒒閸屾瑦绁扮€规洜鏁诲畷鐗堟償閿濆洨顦繝鐢靛Т濞层倝鎷戦悢琛″亾楠炲灝鍔氶柣妤佺矊椤﹪濡搁埡鍌楁嫼闂佸憡绋戦敃銉т焊娴煎瓨鐓熼柣鏂垮级濞呭﹥顨ラ悙鎻掓殲闁靛牞缍佸畷姗€鍩℃担鎻掍壕濠电姵纰嶉悡銏′繆椤栨瑨顒熸俊鎻掓贡閳ь剝顫夐幐鍝ョ矓閹绢喒鈧棃宕橀鍢壯囨煕閳╁厾顏堟瀹ュ洨纾藉ù锝呭閸庢劙鏌涢妸銉ヮ暢缂侇喖顑呴濂稿幢閹邦兛绨奸梻浣告啞閸斿繘寮插┑瀣煑闁糕剝銇涢弨浠嬫煟濡偐甯涙繛鎳峰嫪绻嗘い鎰剁悼缁犵偞銇勯姀鈩冨碍閾绘牠鏌嶈閸撶喎锕㈡担绯曟斀闁绘顕滃銉╂倵濮樼厧鏋ょ紒顔款嚙椤粓鍩€椤掑嫬钃熼柣鏃傗拡閺佸﹪鏌熼鍡楁湰濮ｅ绱撻崒娆愮グ婵℃ぜ鍔庨崚鎺戔枎韫囨洘娈鹃梺闈涱煭婵″洨寮ч埀顒勬⒑閸涘﹥澶勯柛瀣缁煤椤忓應鎷婚梺绋挎湰閸戝綊鎮￠鍕厱閻庯綆浜滈鈺呮偂閵堝鐓ラ柡鍥╁仜閳ь剙鎽滅划缁樺鐎涙鍘遍梺鍦亾椤ㄥ懘骞婂鍥╃當濠㈣泛顑囩粻楣冩倵閻㈡鐒鹃悽顖氱埣閺岋絾骞婇柛鏃€鍨甸悾鐑藉箛閺夊灝鐎銈嗗姧缁茶棄顕ｉ崹顔规斀妞ゆ梻鐡斿▓鏃€淇婇锝庢畷闁哄懎鐖奸、娑㈡倷鐎电骞嶉梻浣告贡缁垳鏁Δ鍛柈闁绘劗娼挎惔銊ョ倞鐟滄繈鐓鍌楀亾鐟欏嫭绌跨紒鍙夊劤椤曘儵宕熼鍌滅槇闂佺琚崐鏍礊鐎ｎ喗鈷掗柛灞剧懅椤︼箓鏌ｉ妶鍛枠妞ゃ垺淇洪ˇ褰掓煛鐏炴枻韬柡浣瑰姈瀵板嫮浠﹂悾灞诲亰闂傚倷鐒﹂幃鍫曞磿濞差亜绀堥柨鏇炲€搁弸渚€鎮楅敐搴℃灍闁绘挻娲樼换娑㈠箣濠靛棜鍩為梺鍝勵儍閸婃繈寮婚弴銏犻唶妞ゆ劦婢€閸犲﹤鈹戦纭峰伐妞ゎ厼鍢查悾鐑藉箳閹存梹鐎婚梺鐟扮摠缁诲倿鈥栨径鎰拺閻犲洤寮堕幑锝夋煟閻旂鈻曠€规洏鍔戦、娆撴偂鎼达絽鎼搁梻鍌氬€烽懗鍫曗€﹂崼銉ュ珘妞ゆ帒瀚烽弫瀣煏韫囨洖违婵炲樊浜堕弫鍌炴煕濞戝崬鏋熼柣婵囩墵濮婄粯鎷呮笟顖滃姼闂佸搫鐗滈崜娆擄綖濠靛惟闁冲搫鍊瑰▍鏍р攽閻愬弶顥為柟绋款煼閸╂盯骞嬮悩鐢碉紳婵炶揪绲介～鏍敂閸涱喖寮挎繝銏ｅ煐閸旀牠鎮￠弴銏犵閻庢稒顭囬埥澶愭煃瑜滈崜姘辨崲閸儳宓?ChannelService闂?
func (s *OpenAIGatewayService) ResolveChannelMapping(ctx context.Context, groupID int64, model string) ChannelMappingResult {
	if s.channelService == nil {
		return ChannelMappingResult{MappedModel: model}
	}
	return s.channelService.ResolveChannelMapping(ctx, groupID, model)
}

// IsModelRestricted 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣濠垫劖缍楅梺閫炲苯澧柛濠傛健楠炴劖绻濋崘顏嗗骄闂佸啿鎼鍥╃矓椤旈敮鍋撶憴鍕８闁告梹鍨甸锝夊醇閺囩偟顓洪梺缁樼懃閹虫劙鐛姀銈嗏拻闁稿本鐟х粣鏃堟煃瑜滈崜娑㈠磻濞戙垺鍤愭い鏍ㄧ⊕濞呯娀鏌熺紒銏犳灍闁绘挻鐩幃姗€鎮欓幓鎺嗘寖闂侀潧妫欑敮锟犲蓟瀹ュ牜妾ㄩ梺鍛婃尪閸斿海妲愰悙鍝勫耿婵炴垶顭囬敍娑㈡⒑閸涘﹣绶遍柛姗€绠栧鎶芥晜闁款垰浜鹃柛蹇擃槸娴滈箖姊洪崨濠冨闁告挻鐩畷銏ゅ箹娴ｇ懓鈧敻鏌涜箛鎿冩Ц濞存粓绠栭弻锝嗘償椤栨粎校闂佸憡鎸婚惄顖炲极瀹ュ鍋勯柛婵勫劤椤旀洟鏌ｆ惔锝嗘毄妞ゎ厼鐗撻、鎾诲箻閺傘儲鏂€闂佺偨鍎村▍鏇㈠窗濡椿娈介柣鎰皺缁犲鏌熼瑙勬珖闁归濞€閹崇娀顢楁径濠冩澑闂傚倸鍊风粈浣革耿闁秴纾块柕鍫濐槸閺勩儵鏌涢锝囩闁绘帊绮欓弻鏇熷緞閸℃ɑ鐝曢梺鎶芥敱鐢帡婀侀梺鎸庣箓閹冲繘骞嗛崼銉﹀仺妞ゆ牗姘ㄩ崺锝夋煛瀹€瀣К缂佺姵鐩獮妯兼崉鐞涒€冲緧闂佽瀛╅鏍窗濡ゅ懏鍋傞柨鐔哄Т缁犳牠鏌嶉崫鍕殲閻庢碍宀搁弻銈夋嚌閺夎法鍘梺纭呭皺婢ф鎹㈠┑瀣仺闂傚牊绋愬▽顏堟⒑閸撹尙鍘涢柛銊ㄦ硾閻ｇ兘宕烽鐔锋瀭闂佸憡娲﹂崑鍕叏閸涘瓨鈷掗柛灞捐壘閳ь剟顥撳▎銏狀潩椤掑鍔烽悷婊勬濡喐绻涚€电孝妞ゆ垵鎳愮划锝呂旈崨顔惧幐閻庡箍鍎辨鎼佺嵁閺嶎偆纾奸柍褜鍓熷畷濂稿Ψ閿旀儳骞堟繝寰锋澘鈧洟宕锝囶洸鐟滅増甯楅悡娑㈡煃瑜滈崜姘辩矉閹烘柡鍋撻敐搴′簽闁告ɑ鎹囧娲濞戣鲸肖闂佺閰ｆ禍璺侯嚕閾忣偅缍囬柍鍝勫暟閿涙粓姊虹紒妯兼噭闁荤喆鍎辩叅闁挎洖鍊归崑鍌涚箾閸℃ê濮傚ù婊勭矒閺岋繝宕橀妸銉㈠亾閸濄儲鏆滈柛顐ｆ礃閻撳啰鎲稿鍫濈婵娉涚粻鏍煕瀹€鈧崑娑氱不閺嶎灛鏃堟晲閸涱厽娈查梺绋匡工椤兘寮婚敃鈧灒闁绘艾顕粈鍡涙⒑闂堟单鍫ュ疾濠婂牊鍋傞柛鎰靛枟閻撴洘绻濋棃娑欘棞妞ゅ繆鏅犻弻宥堫檨闁告挻鐩獮濠囧箻鐠囪尙鍘洪悗骞垮劚椤︻垶鎮欐繝鍥ㄧ厪濠电倯鍐╁闁哥喎閰ｅ缁樻媴閻戞ê娈屾繝鈷€鍌滅煓閽樼喖鏌熼柇锕€骞橀柣婵婂煐娣囧﹪顢涘┑鍡楁優缂備胶濞€缁犳牠骞冭ぐ鎺戠倞闁搞儜鍕闂佸搫妫庨崐婵嗩潖妤﹁￥浜归柟鐑樼箖椤ユ牠姊虹粙璺ㄦ槀闁稿﹥娲熼、姘跺Ψ閳轰胶顦板銈嗗笒閸婂顢欓弴銏♀拺缂備焦锚婵箓鏌涢幘鏉戝摵鐎殿喖鐖奸獮鏍ㄦ媴閸忓瀚藉┑鐐舵彧缂嶁偓婵炲拑绲块弫顔尖槈閵忥紕鍘遍梺鍝勫暊閸嬫挻绻涢懠顒€鏋涙鐐插暙鐓ゆい蹇撳珋閳哄啯鍠愮€光偓閸曨剙鈧灝鈹戦悩鍙夊闁抽攱甯￠弻娑㈩敃閵堝懏鐏侀梺宕囩帛閹瑰洭寮婚敓鐘茬劦妞ゆ帊鑳堕々鐑芥倵閿濆骸浜為柛妯圭矙濮婅櫣娑甸崨顓犲姺闂佸憡鏌ㄧ粔鎾煝瀹ュ宸濇い鎺曞亹閹虫捇鈥﹂妸鈺佺妞ゆ劑鍨硅闂傚倷绀侀悿鍥綖婢舵劕鍨傞柛褎顨呯粻鏍ㄧ箾閸℃ɑ灏伴柛銈嗗灦閵囧嫰骞掑鍥у闂佸摜濮甸悧婊呮閹捐纾兼繛鍡樺灱缁愭姊洪崫銉バｆい銊ワ躬閻涱噣骞囬弶璺槶閻熸粌绉瑰畷鐟扳攽閸″繑鏂€闂佺粯锚閻ゅ洦绔熷鈧弻娑㈠箛閵娿儰澹曢梻浣藉吹閸犳劗鍒掑鍥у灊鐎光偓閸曘劉鍋撻敃鍌氱倞妞ゆ巻鍋撻柛鎰ㄥ亾婵＄偑鍊ら崗姗€鍩€椤掆偓绾绢厾绮斿ú顏呯厵妞ゆ棁顫夊▍鍛亜閺傝法绠绘い銏＄懇閹剝鎯旈埥鍡橆棈缂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柤绋跨仛閸欏繘鏌ｉ姀銏℃毄闁活厽鐟╅弻鐔兼倻濮楀棙鐣舵俊妤€鎳樺娲川婵犲啫顦╅梺鎼炲妽婢瑰棛鍒掓繝姘闁兼亽鍎抽崢閬嶆⒑閺傘儲娅呴柛鐔村妽缁傛帡鏁冮埀顒勨€︾捄銊﹀枂闁告洦鍓涢ˇ鏉库攽椤旂》鍔熺紒顕呭灦楠炲繘宕ㄧ€涙ɑ鍎梺鑽ゅ枑婢瑰棝顢曟總鍛娾拻濞达絽鎲＄拹鈩冦亜椤撶偟澧﹂柍銉畱閻ｏ繝骞嶉鑺ヮ啎闁荤喐绮庢晶妤冩暜閹烘鐓ラ柕鍫濇缁诲棝鏌曢崼婵嗏偓鍛婄妤ｅ啯鈷戦柦妯侯槸閺嗙喖鏌涢悩宕囧⒌鐎殿喖顭峰鎾偄閾忚鍟庨梻浣稿閻撳牓宕伴弽銊﹀弿濠㈣泛艌閺€浠嬫煃閳轰礁鏆欑紒鎻掝煼閺岋絽螖閳ь剟鏁冮妷褏鐭氶弶鍫涘妿缁♀偓闂佹悶鍎撮崺鏍疾椤掆偓閳规垿鎮╃拠褍浼愰梺鍝ュ櫏閸嬪懐绮嬮幒鎳崇喐绗熼娑樼槣闂備線娼ч悧鍡欐崲閹烘绀嗛柟娈垮枤绾剧晫鈧箍鍎遍幏鎴濐啅閵夛负浜滄い鎰╁灮缁犱即鎮￠妶鍡愪簻闊洦鎸搁褏绱掗崡鐐靛煟婵﹥妞藉Λ鍐ㄢ槈鏉堛剱銈夋⒑缁嬫寧鍞夊鏉戞憸缁晠鎮㈢粙璺ㄧ獮闂佸綊鍋婇崢鎼佸箯濞差亝鈷戦柛娑橈功閳藉鏌ㄩ弴妯哄姢濞存粍鎮傞幃浠嬪川婵犲嫬骞堥梻浣虹帛閿氱痪缁㈠幗閺呭爼鎮介崨濠勫幗濠电偞鍨靛畷顒€鈻嶅鍥ｅ亾鐟欏嫭绀冮柨鏇樺灲閵嗕礁鈻庨幇顕€妾紓浣割儐鐎笛囧汲椤忓牊鈷掗柛灞剧懅椤︼箓鏌熺喊鍗炰簻閻撱倝鏌曟繛鐐珔缁惧墽绮换娑㈠箣濞嗗繒浠鹃梺鎼炲€愰崑鎾绘⒒娴ｇ顥忛柣鎾崇墦瀹曚即骞掑Δ鈧紒鈺伱归悩宸剰缂佺姵濞婇弻鐔煎箚閻楀牜妫勭紒鐐劤缂嶅﹪寮婚悢鍏煎亱闁割偆鍠撻崙锛勭磼閻愵剙鍔ゆい顓犲厴瀵鎮㈤悡搴ｎ槶閻熸粌绻掗弫顔尖槈閵忥紕鍘?ChannelService闂?
func (s *OpenAIGatewayService) IsModelRestricted(ctx context.Context, groupID int64, model string) bool {
	if s.channelService == nil {
		return false
	}
	return s.channelService.IsModelRestricted(ctx, groupID, model)
}

// ResolveChannelMappingAndRestrict 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍嵁瀹ュ鏁婄痪鎷岄哺濮ｅ姊绘担渚劸妞ゆ垶鍨归幑銏犫攽閸♀晛娈ㄩ梺鍓插亝濞叉牠鏌嬮崶銊﹀弿婵妫楅獮妤呮煟濠靛洦鈷掔紒杈ㄦ尰閹峰懘鎮剧仦鐣屽闂備胶顭堥敃銉ッ哄┑瀣€堕柛鎰靛枟閳锋垿鏌熺粙鎸庢崳缂佺姵鎹囬弻鐔煎礃閺屻儱寮伴悗娈垮枟婵炲﹪骞冨▎鎾村€绘俊顖滃帶楠炲牆鈹戦悩鍨毄濠殿喖顕埀顒佸嚬閸欏啫顕ｉ幎绛嬫晢闁告洦鍓涢崢鎼佹倵閸忓浜鹃柣搴秵閸撴盯寮抽悩缁樼叄濞村吋鐟ч崚浼存煏閸℃ê绗掓い顐ｇ箞閹剝鎯旈敐鍕暰闂備胶鍘у鍫曟偋閻樺樊娼栫紓浣诡焽閻熷綊鏌嶈閸撶喖宕洪埀顒併亜閹烘垵鈧憡绂掑鍫熺厾婵炶尪顕ч悘锟犳煛閸涱厾鍩ｆい銏＄洴閹瑩寮堕幋婊呯处濠碉紕鍋戦崐鏍偋閹捐纾规俊銈呭暙婵剟鏌嶈閸撴瑨鐏冮梺缁橈耿濞佳勭濠婂牊鐓曢柣鏂挎啞鐏忥妇鈧娲樺浠嬪极閹剧粯鍋愰柤纰卞墻濡蹭即姊绘担鍝ユ瀮婵℃ぜ鍔庡▎銏ゆ晸閻樿尙鍔﹀銈嗗笂閼宠泛煤鐎涙ɑ鍙忓┑鐘插暞閵囨繄鈧娲忛崝宥囨崲濠靛纾兼慨妯哄悑缂嶅鈹戦悩娈挎毌婵℃彃鎳樺畷瑙勫鐎涙ɑ娅囬梺闈涚墕閻楀繑绂嶆潏銊х瘈闁汇垽娼у瓭闂佹寧娲忛崐婵嬪箖瑜庣换婵嬪炊瑜忛崢閬嶆⒑閸︻厼鍔嬮柛銈嗕亢閵囨劙骞掗幘瀛樼彸闂備礁鎲℃笟妤呭窗閺嶎厼姹查柛顐犲劜閳锋垿鎮归崶锝傚亾瀹曞洣鍝楅梻浣虹帛椤ㄥ棝骞戦崶顒傚祦闁告劑鍓弮鍫濈劦妞ゆ帒瀚哥紞鏍ㄧ節闂堟侗鍎愰柛瀣€块獮鏍庨鈧悘顔济瑰鍐煟婵﹦绮粭鐔煎焵椤掑嫬鐒垫い鎺嶈兌閵嗘帡鏌嶇憴鍕诞闁哄本鐩顕€鍩€椤掑嫬鍨傞柛褎顨嗛崑妯汇亜閺冨倵鎷℃繛绗哄姂閺屽秷顧侀柛鎾跺枛婵″瓨鎷呯化鏇熺€婚梺鍦亾濞兼瑩鎯傞崟顒傜瘈闁靛骏绲剧涵楣冩煥閺囨ê鈧繆妫㈡繝銏ｅ煐閸旀牠鍩涢幒妤佺厱閻忕偛澧介幊鍡涙煕韫囨挾鐏辩紒杈ㄥ浮椤㈡岸宕ㄩ鐘辨闂備胶鍋ㄩ崕鑽ょ不閹惧磭鏆﹀┑鍌滎焾鍞銈嗘婵倕鈻嶉弮鍫熺厽闁绘柨鎽滈惌瀣煕閵娿儳绉烘?// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣濠垫劖缍楅梺閫炲苯澧柛濠傛健楠炴劖绻濋崘顏嗗骄闂佸啿鎼鍥╃矓椤旈敮鍋撶憴鍕８闁告梹鍨甸锝夊醇閺囩偟顓哄┑鐘绘涧閻楀啴宕戦幘娲绘晣闁绘垵妫欑€靛矂姊洪棃娑氬闁硅櫕鍔楃划缁樺鐎涙鍘藉┑掳鍊愰崑鎾翠繆椤愶絿绠炴鐐插暣閹瑩宕崟顐も偓顓烆渻閵堝棗濮夊┑顔肩－閼鸿鲸绻濆顓涙嫼闂佽崵鍠撴晶妤呭箚閸喍绻嗘い鎰剁秵濞堟洜绱掗崒姘毙х€规洘绮忛ˇ瀵哥棯閹佸仮闁哄本鐩獮妯何旈埀顒€螞濞嗘搩鏁佹俊銈呮噺閳锋垿鏌涘☉姗堝姛闁瑰啿鍟撮弻娑㈡偄閸涘﹦绋囬梺浼欑到閸㈣尙鍙呭銈呯箰鐎氼噣宕濋敃鈧—鍐Χ閸℃鐟愰梻鍌氬缁夌數绮嬪鍜佺叆闁割偆鍠撻崢鐢告⒑缂佹ê鐏﹂柨姘舵煟韫囧鍔﹂柡灞界Х椤т線鏌涢幘鏉戝摵妤犵偛鍟村畷鎺戭潩鏉堛劍顔曢梻渚€娼х换鍡涘箠閸ヮ剙纾婚柟鐐灱濡插牊绻涢崨鐗堢彧闁搞劌鐏濋悾鐑芥晲閸垻鏉搁梺鍦檸閸ㄩ亶寮埀顒勬⒒娴ｈ櫣甯涙い顓炵墢閸掓帗銈ｉ崘銊ь槯閻庡厜鍋撻柛鏇ㄥ墰閸樿棄顪冮妶鍡樺暗闁哥姵鎹囬獮濠囧礋椤戝彞绨婚梺闈涱煭缁蹭粙宕濋敃鍌涚厵妞ゆ梻鏅幊鍥┾偓瑙勬穿缁叉儳顕ラ崟顒傜瘈闁告劦浜欓幃宥嗙節绾板纾块柛瀣灴瀹曟劙骞嬮敃鈧崹鍌涚箾瀹割喕绨甸柍褜鍓欓崯顖滄崲濠靛鐐婄憸搴∥ｉ鍕拺闂侇偆鍋涢懟顖涙櫠椤栫偞鐓忛柛銉戝喚浼冨Δ鐘靛仦鐢繝鐛€ｎ亖鏀介柟閭︿簽濡垶姊婚崒娆掑厡妞ゎ厼鐗撻、鏍川閺夋垹锛欓梺绉嗗嫷娈旈柦鍐枛閺岋綁寮崶銉㈠亾閳ь剟鏌涚€ｎ偅宕岄柟顔挎閳绘挾鎹勯妸銉バ繝鐢靛仜椤曨厽鎱ㄩ悽鍨殰闁绘劕鐏氶～鏇㈡煙閻戞ɑ鈷愰悗姘哺閺屾稑鈻庤箛锝嗏枔濠碘槅鍋撶换婵嗩潖濞差亝顥堟繛鎴炵懄閹瑩鏌ｆ惔銏㈢叝闁告濞婇幃浼搭敋閳ь剙鐣烽崼鏇ㄦ晢濠㈣泛顑嗗▍鎾绘⒒娴ｄ警鐒剧紒缁橆殜瀹曟垿骞囬弶璺ㄥ姦濡炪倖甯掗崐鑽ゆ暜濞戞氨纾肩紓浣诡焽閵嗘帡鏌嶈閸撴氨绮欓幒妞烩偓锕傚炊椤掍礁鍓归梺鍦劋濮婅崵澹曟總鍛婄厽婵☆垵娅ｉ敍宥咁熆瑜滄禍婵嬪Φ閸曨垼鏁冮柕蹇嬪灮椤斿洭鏌ｉ幘鍗炩偓鏍Φ閸曨喚鐤€闁圭偓鍓氭禒鍦磽娴ｇ鑸归柣鏍帶椤繐煤椤忓嫬绐涙繝鐢靛Т閸燁偊藝閳哄懏鈷戦梻鍫氭櫇缁夌敻鏌涢悩宕囧⒌闁炽儻绠撳畷鍫曨敆閳ь剟鏌嬮崶顒佺厸闁搞儮鏅涢弳閬嶆煙閻ゎ垱顏犵紒杈ㄦ尰閹峰懘宕滈幓鎺戝婵犵數鍋熼妴瀣崲濠靛棛鏆﹂柨婵嗩槸缁€瀣亜閹般劌浜鹃梺鑽ゅ枛閸嬪﹪鎮￠妷鈺傜厱闁斥晛鍠氶悞浠嬫煏閸℃ê濮嶆慨濠勫劋鐎电厧鈻庨幋婵嗙厒闂備焦鎮堕崹娲偂閿熺姴鏄ラ柕蹇婂墲閸庣喖鏌曟繛鍨姢妞ゆ梹甯￠弻锝嗘償閵忊懇濮囬柦鍐憾閺岋絽鈹戦幇顑垮枈濠殿喖锕︾划顖炲箯閸涙潙宸濆┑鐘插€瑰▓妯荤節閻㈤潧浠╂い鏇熺矌缁骞樺畷鍥ㄦ闂侀潧绶甸弶澶哥凹闂備礁鎲￠崝蹇涘疾濞戞瑧顩叉俊銈呮噺閻撶喖骞栭幖顓炵仯缂佸鏁婚弻娑㈡偐閹颁焦鐤侀梺绯曟櫆閻╊垶鐛€ｎ喗鏅滈柣锝呰嫰楠炲牊绻濋悽闈涒枅婵炰匠鍏炬稑螖閸滀焦鏅滈梺鍐叉惈閸婅埖绂嶅鍫㈠彄闁搞儜宥嗘暰濠电偛鎳庡ú銊ф閹烘挻缍囬柕濞垮劤椤戝倻绱撴担浠嬪摵閻㈩垱甯熼悘鎺楁⒑閸涘﹦绠撻悗姘槻椤洭骞樼紒妯锋嫽婵炶揪绲介幗婊堟晬瀹ュ洨纾兼い鏇炴噹閻忥附顨ラ悙鏉戝缂佺粯绻傞～婵嬵敄閳诡厼娲﹂悡銉︾節闂堟稒锛嶆俊鎻掓憸缁辨帡鍩€椤掑嫬绀冩い鏃傛櫕閸樻捇鏌℃径灞戒户濠⒀勵殜钘濋柣鎴ｅГ閻撴洟鏌″鍐ㄥ濠⒀冪仛閹便劍绻濋崨顕呬哗闂佸綊顥撴繛鈧鐐存崌楠炴帡骞嬮悙鍨樼紓鍌氬€搁崐鎼佸磹閸濄儳鐭撻柡澶嬪殾濞戞ǚ鏋庨柟瀵稿Х閻掑潡鎮楅獮鍨姎闁瑰嘲顑夐幃锟犲Ψ閳哄倻鍘卞銈嗗姧缁茶法绮婚弽顓炵骇闁割偅绻勬晶鏇㈡煃鐟欏嫬鐏撮柟顔界懇瀹曪絾寰勫Ο浼欑磼闂傚倷绀佸﹢閬嶁€﹂崼銉⑩偓锕€鐣￠幍顔芥闂佽澹嗘晶妤呭磻閵娾晜鐓曟繛鎴烇公瀹搞儱霉閻欌偓閸樺ジ鍩為幋锔藉€烽柛娆忣樈濡偟绱撴担铏瑰笡闁告梹鐟╅妴渚€寮崼婵堝幐闂佸憡渚楅崰姘跺储閹惧墎纾介柛灞剧懅閸斿秹鎷戞潏鈺冪＜闁逞屽墴瀹曟﹢顢欓悾灞藉箥濠电偞鎸婚懝楣冩倶濠靛洣绻嗛柣锝呯灱绾捐偐绱撴担璇＄劷缂佺姷鍋熼埀顒冾潐濞叉鏁幒鏇犱航闂佸搫顦遍崑鐐差熆濮椻偓瀹曠懓鈹戠€ｎ偀鎷婚梺绋挎湰閻熴劑宕楃仦淇变簻闁靛繒濯村銉╂煃鐠囪尙效鐎殿喗鎸抽幃鈺呭箵閹烘柧鍠婇梺璇查閸樻粓宕戦幘缁樼厱闁哄洢鍔屾禍鐐烘煕濡粯銇濇慨濠呮閹叉挳宕熼銈庢О婵＄偑鍊栧▔锕傚炊瑜嶉悗顓㈡⒑缁嬫寧婀扮紒瀣箲缁傚秴顭ㄩ崼鐔哄幐闂佸憡鍔戦崝宥夊箹閹邦喗鍋栨慨姗嗗厴閺€浠嬫煟濮楀棗鏋涢柣蹇ｄ邯閺屾盯濡搁妷褌铏庡銈庡亜缁绘垹鎹㈠┑鍡╂僵妞ゆ帒鍋婄槐鍙夌節閻㈤潧浠滄俊顐ｇ懇瀹曞綊宕归锛勭畾婵炲濮撮鍡涙偂閻旂厧绠归弶鍫濆⒔閹ジ鏌熼搹顐疁闁哄本绋戦埥澶愬础閻愬吀鍖栭梻浣虹帛閹稿骞戦崶褜娼栭柧蹇氼潐閸忔粓鏌涘☉鍗炴灈婵炴嚪鍥ㄧ厽閹兼番鍔嶅☉褔鏌ｉ鐐测偓鎼侇敋閿濆鏁冮柨婵嗘川閻﹀牓姊洪幖鐐插姌闁告柨閰ｅ畷銏ゆ偨閸涘﹦鍘告繝銏ｆ硾閿曪附鏅堕幇鐗堢厸闁告侗鍠氱粻鐐碘偓娈垮枦濞呮洜鎹㈤崨顖涱潟闁稿秶甯猼ed 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘﹢寮婚敐澶婎潊闁绘ê妯婂Λ宀勬⒑鏉炴壆顦﹂柨鏇ㄤ邯瀵鍨鹃幇浣告倯闁硅偐琛ラ埀顒€纾鎰磽閸屾瑧鍔嶉柛鏃€鐗犻妴鍐╃節閸パ嗘憰闂佺偨鍎查崜姘跺触鐎ｎ喖绠圭紒顔煎帨閸嬫捇宕樺顔煎Η闂傚倸鍊峰ù鍥ь浖閵娧呯焼濞达綀銆€閸嬫挸顫濋銏犵ギ閻庢鍠涢褔鍩ユ径鎰潊闁绘ɑ鐗撻崝鎴﹀蓟閺囷紕鐤€濠电姴鍊搁埛澶愭⒑缂佹绠栨俊顐㈠暙椤繐煤椤忓嫮顔愰梺缁樺姈瑜板啴鈥栨径宀€纾藉ù锝囩摂閸ゆ瑩鏌ｅΔ浣瑰碍妞ゆ洩绲剧换婵嗩潩椤撶偘绨婚梻浣虹帛閹哥霉閻戣姤鍋╁Δ锝呭暞閸嬧剝绻濇繝鍌氭殶缂佺姷绮〃銉╂倷閹碱厽鐤侀梺璇″枓閺呮繈骞夊Δ浣瑰闁绘垶锚濞堝矂姊哄畷鍥ㄥ殌闁哥喐娼欓悾鐤亹閹烘繃鏅╅梺鑽ゅ枛椤ｏ附绔熼弴鐐╂斀闁绘绮☉褎銇勯幋婵囧殗闁归攱鍨块幃銏ゅ礂閼测晛寮抽梻浣虹帛閺屻劑骞栭銏㈡懃缂傚倷鑳堕崑鎾诲磿閹惰棄瑙﹂悗锝庡墯瀹曞弶绻涢幋娆忕仼缂佺媴缍侀弻锝夊箣濠靛洨鐓戠紓浣稿閺佽顫忓ú顏勭闁绘劖绁撮崑鎾诲即閵忊晜鏅為梺鎼炲劗閺呮粓锝為弴銏＄厵闁诡垎鍛喖婵犳鍨遍幐鎶藉蓟閻旇櫣纾兼俊顖濇閻熸劖绻濋姀锝庢綈婵炶尙鍠庨～?false闂?
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

// ReplaceModelInBody 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻橀潧骞堟繝娈垮枟閿曗晠宕㈡禒瀣︽繝闈涱儐閻撴稑霉閿濆浂鐒鹃柍褜鍓欏鈥愁嚕婵犳艾惟闁冲搫锕ラ弲顒€鈹戦鏂や緵闁稿瀚拌棢婵浜壕浠嬫煕鐏炲墽鎳呴柛鏂跨У閵囧嫰濡搁妷锔绘￥缂備緡鍠楅悷鈺呯嵁鐎ｎ喗鏅濋柍褜鍓熼幃鍧楊敊濞村骸缍婂畷妤呭礂閼测晝鈻忔繝娈垮枦椤鎮￠敓鐘茶摕闁靛ň鏅涢崡鎶芥煟閹邦厾銈撮柛瀣崄閵囨劙骞掗幘鍏呯暗闂備胶绮…鍥极缁嬭娑㈩敍閻愬鍘告繝銏ｆ硾閿曪附绂掗姀銈嗙厾闁告縿鍎查崵鈧銈庡弨閸庡篓娓氣偓閺屾盯濡搁妷褍鐓熷Δ鐘靛仜閸燁偉鐏冮梺閫炲苯澧撮柨婵堝仜閳规垹鈧絽鐏氶弲锝夋⒑缂佹ê濮囩€殿喖鐖奸獮濠傗枎閹邦喚鐦堝┑鐐茬墕閻忔繈寮稿☉銏＄叆闁哄洦锚閸旀碍銇勯鍕殻闁圭锕ュ鍕沪閻愵剦鍟庨梺鑽ゅ枑缁孩鏅跺Δ鍐╂殰闁冲搫鎳庨弸渚€鎮楅敐搴℃灍闁绘挻娲樼换娑㈠箣濠靛棜鍩炲Δ鐘靛仦閻楁洟鈥﹂崸妤佸仭闂侇叏绠戦崜杈╃磽娴ｈ櫣甯涚紒璇茬墦楠炲啯绂掔€ｎ偒妫冨┑鐐殿棎閸╂牠鎯勯鐐茶摕婵炴垶鍩冮崑鎾绘晲鎼粹€茬凹閻庤娲栭惌鍌炲蓟閿涘嫪娌紒瀣仢閳峰鎮楅崹顐ｇ凡閻庢凹鍣ｉ崺鈧い鎺戯功缁夐潧霉濠婂嫮绠炴い銏＄懇瀹曘劎鈧稒锚娴狀厼鈹戦悩璇у伐闁瑰啿閰ｉ妴鍌涚附閸涘﹤浠哄銈嗙墬娓氭鈻撳鍛亾濞堝灝鏋熼柣鎾偓鎰佸殨濠电姵鑹惧敮闂佹寧姊归崕鎶界嵁閸儲鈷掑ù锝堫潐閻忛亶鏌￠崨顔炬创鐎规洏鍨洪妶锝夊礃閻愵剚娅嗛梻浣稿閸嬪棝宕版惔銊ョ闁荤喖鍋婂〒濠氭煏閸繃顥炵痪鍓ф暬閺屾稓鈧綆鍋呯亸浼存煙娓氬灝濡界紒缁樼箞瀹曟﹢鍩炴径姝屾闂傚倷娴囬鏍窗閹烘绀堟繝闈涱儏缁犳岸鏌￠崘銊у闁哄懏鐓￠弻娑㈠即閵娿儱绫嶉梺宕囨嚀缁绘ê顫忔繝姘＜婵炲棙鍨垫俊浠嬫⒑鐠団€虫灈闁绘牜鍘ч悾宄扳攽閸粍鍕冪紓浣圭☉椤戝洭宕濋悜鑺モ拺閻犳亽鍔岄弸鏂库攽椤旂厧妲婚柣?JSON model 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻濠氭煕閳规儳浜炬俊鐐€栫敮濠囨嚄閸洖鐓濋柟鍓х帛閻撴盯鏌涘☉鍗炴灓缂佺姵锕㈤弻娑㈠箳閹惧磭鐟ㄩ梺瀹狀嚙闁帮綁鐛Ο铏规殾闁搞儴娉涢弫钘夆攽閻樻鏆滅紒杈ㄦ礋瀹曟垵鈽夐姀鈥冲壄闂佺粯鍨煎Λ鍕婵犳碍鐓欓柟瑙勫姦閸ゆ瑧绱掗埀顒勫礃閳瑰じ绨婚梺鍝勫暙閸婂摜鏁崼鏇熺厾闁哄娉曟禒銏ゆ煃鐟欏嫬鐏撮柟顔界懇瀵爼骞嬮悩杈╃婵犵绱曢崑娑㈡偤閵娾晛绠栭柛灞惧嚬閸ゆ洟鏌＄仦璇插姎闁绘挻鐩弻娑樷槈閸楃偞鐏堥梺閫炲苯澧伴柡浣割煼瀵鈽夊鍛澑闂佺懓鐏濋崯顖滅懅婵犵數鍋涢悺銊у垝閹惧墎涓嶉柡宓本缍庡┑鐐叉▕娴滄粌顔忓┑鍡忔斀闁绘ɑ褰冮弳娆愩亜閿旇娅婃慨濠冩そ瀹曘劍绻濋崘銊╃€洪梻浣哄帶缂嶅﹦绮婚弽顓炴槬闁靛繒濯崥瀣熆鐠虹尨宸ラ柛鐐妼椤啴濡堕崱妯烘殫闂佸摜鍠庡锟犵嵁韫囨拋娲敂閸涱亝瀚奸梻浣告啞缁嬫垿鏁冮妷褌鐒婇柟娈垮枟閸犳劙鏌℃径濠勪虎闁哄棛鍋熺槐鎺楀磼濮樻瘷褏鈧鍠曠划娆撱€佸鈧幃銏ゅ传閸曨偆鐤勯梻鍌欒兌閹虫捇宕ョ€ｎ喖绠氱€光偓閸曨偆鐛ラ梺鍝勮癁閳ь剟寮稿鍥ｅ亾楠炲灝鍔氭繛鑼█瀹曟垿骞橀懜闈涙瀭闂佸憡娲﹂崜娑⑺囬妸鈺傗拺闁硅偐鍋涙俊鍏肩箾绾绡€闁诡喕鍗抽、娆戝枈鏉堛劌缂撻梻渚€鈧偛鑻晶鎾煟濞戝崬娅嶇€规洖鐖兼俊鎼佹晝閳ь剟顢撻幘缁樷拺闁煎鍊曢弸鎴炵節閵忊槄鑰跨€规洏鍨介幃娆撴倻濡攱瀚藉┑鐐舵彧缁蹭粙骞夐敓鐘茬畾闁割偆鍠嗘禍婊堟煥閺冨浂鍤欓柣蹇ョ畵閺岀喖顢欓妸銉︽悙缂佲偓鐎ｎ偁浜滈柡宥冨妽閻ㄦ垶銇勯敂璇叉珝闁诡喗顨婇悰顕€宕归鐓庮潛闂備胶鎳撻崯鍧椝囬棃娑卞殨?gjson/sjson 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟撮崣鍕煏閸℃鏆ｅ┑锛勬焿椤︽挳鏌ｉ鐔烘噰闁哄本鐩濠氼敇濠婂啠鍋撻幇鏉跨；闁规崘顕ч柨銈嗕繆閵堝嫯顔夐柟宄版惈椤啴濡堕崱妤€娼戦梺绋款儐閹稿墽妲愰幒鎾崇窞濠电姴楠稿▓鑸电節閵忥綆娼愭繛鍙夘焽閹广垹鈹戦崱鈺傚兊濡炪倖鎸炬慨鎾嵁濡ゅ懏鈷掑ù锝呮啞閹牓鏌涢悤浣镐簻閻撱倝鏌曢崼婵囶棤妞も晜鐓￠弻娑㈡晜鐠囨彃绠伴梺绋款儐閹搁箖骞夐幘顔肩妞ゆ巻鍋撻柣顓у枤缁辨挻鎷呴挊澶屽帿閻庡厜鍋撶紒瀣儥閸ゆ洟鎮归崶銊с偞婵℃彃鐗撻弻鏇＄疀閺囩儐娼旈梺闈涱煭鐠愮喐绂嶅鍫熺厸闁告劑鍔岄埀顒€鎽滈弫顕€鎳滈悙閫涚盎濡炪倖鍔﹂崑鍌炴焽椤栨稒鍙忓┑鐘插暞閵囨繄鈧娲﹂崑濠傜暦閻旂⒈鏁嗛柍褜鍓熼、鎾斥槈閵忊檧鎷洪梺鍛婄☉閿曘儳浜告导瀛樼厽闁绘柨寮跺▍濠囨煕閵娾晝鐣洪柡浣稿€块幃娆擃敆閳ь剛澹曢鐐粹拺闂傚牊渚楅悡顓犵磼閻樺啿鐏╁瑙勬礋椤㈡盯鎮欑划瑙勫闂備浇顕栭崹搴ㄥ川椤栵絽鏁介梻鍌欒兌椤㈠﹤鈻嶉弴銏犵闁搞儺鍓欒繚闂佺鐬奸崑鐐哄磻閵娾晜鐓曟繛鎴炩槈閸儱绠?
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
		accountRepo:           s.accountRepo,
		userRepo:              s.userRepo,
		userSubRepo:           s.userSubRepo,
		billingCacheService:   s.billingCacheService,
		deferredService:       s.deferredService,
		balanceNotifyService:  s.balanceNotifyService,
		userPlatformQuotaRepo: s.userPlatformQuotaRepo,
	}
}

// CloseOpenAIWSPool 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆洤鍔嬮柛銈傚亾婵＄偑鍊栧ú锕傚矗閸愩劎鏆﹂柟杈鹃檮閸嬪嫰鏌ц箛娑掑亾濞戞氨鎲归梻鍌欐祰椤曟牠宕规惔銊ョ劦妞ゆ帒瀚紒鈺伱归悩宸剱闁绘挾鍠愭穱濠囶敍濮樺彉铏庣紓浣瑰敾缁犳挸顕?OpenAI WebSocket 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾剧懓顪冪€ｎ亝鎹ｉ柣顓炴閵嗘帒顫濋敐鍛婵°倗濮烽崑鐐烘偋閻樻眹鈧線寮村杈┬㈤梻浣规偠閸庢椽宕滈敃鍌氭瀬闁告劦鍠楅悡銉╂煛閸ヮ煈娈斿ù婊堢畺濮婂搫效閸パ€鍋撳Δ鍛；闁规崘鍩栧畷鍙夌節闂堟侗鍎愭い銉ョ墛缁绘盯骞嬪┑鍡氬煘闂佸憡鎸绘竟鍡欐閹烘鍊锋い鎺嗗亾闁告柣鍊栭妵鍕敇閻樻彃骞嬮悗娈垮枛椤兘寮幇顓炵窞濠电姴瀚弶鍛婁繆閻愵亜鈧牜鏁幒妤€纾瑰瀣閸ㄦ繈鏌ｉ姀鐘冲暈闁抽攱鍨块弻娑樷攽閸℃浠惧銈冨劤閸嬬喓妲愰幒妤婃晩闁兼亽鍎辩壕鎶芥倵濞堝灝鏋涙い顓犲厴瀹曟椽宕熼姘鳖槰闂侀潧臎閸涱垰绁﹂梻鍌欐祰椤曆囧礄閻ｅ瞼绀婇柛鈩冾焽椤╁弶绻濋棃娑卞剱闁稿骸锕獮鏍庨鈧俊浠嬫煕鐎Ｑ冨⒉缂佺粯绻冪换婵嬪磼濮橀棿鐥俊鐐€戦崕閬嶆偋婵犲洢鈧啴濡烽埡鍌氣偓鐑芥煙缂佹ê绗氭繛鍫濐煼濮婅櫣鍖栭弴鐔哥彣缂備胶绮换鍌炴偩閻戣棄惟闁冲搫锕ラ弲锝夋⒑缂佹ê濮夋い锝勭矙瀹曟垿骞樺ú缁樻櫍闂佺粯鍨靛Λ娆戔偓闈涚焸濮婃椽妫冨☉姘暫濠碘槅鍋呴〃鍡涘箞閵娾晛纾奸柣鎰嚟閸樿鲸绻濋悽闈浶㈤柛鐔哄閺呭爼顢旈崼鐔哄帗缂傚倷鐒﹁摫鐎规洖鏈〃銉╂倷鐎电顫ч梺鐟板槻閹虫ê鐣峰Δ浣哥窞閻庯綆浜舵导婊冣攽閻樺灚鏆╅柛瀣█楠炴捇顢旈崱妤冪瓘婵炲濮撮鍛不閻斿吋鐓ラ柣鏂挎惈瀛濋梺姹囧€ら崳锝夊蓟閿濆绠涙い鏃傚帶婵℃椽姊虹紒妯诲暗濠电偐鍋撻梺鍝勬湰閻╊垶寮崒婊勫珰闁圭粯甯為鎰攽閻愯尙鎽犵紒顔奸叄瀹曟垿骞樼拠鍙傦箓鏌涢弴銊ョ仩閹喖姊洪幐搴ｇ畵闁瑰啿瀛╅幈銊︽償閳藉棙瀵岄梺闈涚墕妤犲憡绂嶉弽顓熺厱闁靛ě鍛缂備礁鍊圭敮锟犲极閸岀偞鍊锋い鎺嗗亾妞ゅ孩鐩娲箚瑜庣粋瀣煕鐎ｎ亝顥滄い?worker 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鎵虫櫅閻忣喖霉濠娾偓閸楁娊寮婚敐鍡樺劅闁炽儱鍟挎潏鍛存⒑缁嬫鍎愰柟鐟版喘瀵鈽夐姀鐘插祮闂侀潧顭堥崕閬嶏綖閳哄懏鈷戦柛娑橈攻瀹曞嫰鏌涢埡鍌滃⒈闁瑰箍鍨归埞鎴犫偓锝庡亽濡啫鈹戦悙鏉戠仸閼裤倝鏌涚€ｎ偅灏扮紒缁樼箓椤繈鏁愰崒姘闂佹寧娲栭崐鎼佹煁閸ヮ剚鐓忓鑸电〒閻ｅ崬霉濠婂嫷娈旈柍瑙勫灴椤㈡瑦鎱ㄩ幇顏嗘崟闂備胶顭堥鍡涘箲閸ヮ剙钃熼柨娑樺濞岊亞绱掗娑欑濞存粓绠栭弻鐔革紣娴ｄ警妲梺璇″枟閸ㄥ潡寮婚悢鍏煎殐闁冲搫濯绘径鎰厓鐟滄粓宕滃▎鎴濐棜妞ゆ挶鍩勯弫濠囨煙鏉堥箖妾柍閿嬪笒闇夐柨婵嗘处閸も偓闂佸磭绮ú鐔煎蓟閿熺姴鐒垫い鎺嶇劍缂嶅洭鏌嶉崫鍕偓鐟扳枔娴煎瓨鈷戦悹鎭掑妼閺嬫柨鈹戦鑺ュ唉闁归攱鍨块幃銏ゅ礂閼测晛甯鹃梻浣稿閸嬪懐鎹㈠鍛傦綀銇愰幒鎾跺幈闁诲函缍嗘禍婊堝焵椤掆偓濞硷繝鎮伴鈧浠嬵敃閵忕姷浜伴梺鑽ゅ枑閻熴儳鈧艾鍢查…鍥箣閿旇В鎷洪梺鍛婄箓鐎氼參藟閹剧粯鐓曢柕濞垮妼閸氳淇婇崣澶婂妤犵偞顭囬埀顒佺⊕閿氭い搴㈡崌濮婃椽宕ㄦ繝鍕暤闁诲孩鐨滈崶褍鍤戦梺鍛婁緱閸ㄥ磭澹曢挊澹濆綊鏁愰崨顔藉創闁哄稄绻濆铏圭磼濮楀棙鐣兼繝娈垮枟閹告娊銆佸鑸垫櫜闁糕剝鐟ч惁鍫濃攽椤旀枻渚涢柛妯哄悑缁傚秴顭ㄩ崼鐔叉嫼缂傚倷鐒﹁摫閻忓浚鍘界换娑氣偓鐢殿焾閸樻挳鏌熼鈧粻鏍嵁鎼淬劍鍤嶉柕澶堝灪鐎氬ジ姊绘笟鈧鑽も偓闈涚焸瀹曘垺绺界粙璺槷闁诲函缍嗛崰妤呮偂閺囥垺鐓忓┑鐐茬仢閸斻倕霉閻樺磭娲撮柡宀€鍠愮粭鐔煎垂椤旂⒈鐎抽梻浣风串缁插墽鎹㈤崼婵堟殾闁绘梻鈷堥弫鍕煠閹帒鍔氭い?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾剧懓顪冪€ｎ亝鎹ｉ柣顓炴闇夐柨婵嗩槹娴溿倝鏌ら弶鎸庡仴婵﹥妞介、妤呭焵椤掑倻鐭撻柟缁㈠枛绾惧ジ鏌曟繛褍瀚弸鎴︽⒑閸濆嫬鏆欓柣妤€妫涚划濠氬冀閵娧咁啎閻庣懓澹婇崰鏇犺姳婵犳艾绠氶柣鏂垮悑閳锋垿鎮楅崷顓炐ｉ柕鍡楀暞缁绘盯寮堕幋鐐叉灎濡ょ姷鍋為〃濠傤嚕閹绢喖顫呴柍閿亾闁归攱妞藉濠氬磼濮樺崬顤€婵炴挻纰嶉〃濠囧箖閻愭番鍋呴柛鎰ㄦ櫇閸橀亶鏌熼懖鈺勊夋俊鎻掓嚇閹剝绺介崨濠勫幍闂佹儳娴氶崑鍛暦鐏炵虎娈介柣鎰綑婵牓鏌熼娑欑叆闁宠棄顦灃闁告劦浜濋弲銊╂⒒閸屾艾鈧悂鈥﹂鍕；闁告洦鍊嬪ú顏呮櫇闁稿本姘ㄩ鍥ㄧ節閻㈤潧校闁煎綊绠栧濠氼敍濞戞氨顔曢悗鐟板閸犳洜鑺辨總鍛婄厽闁规儳鐡ㄧ粈瀣殽閻愬澧繛鐓庣箻瀹曟粏顦查柛鈺佺焸濮婃椽宕崟闈涘壉闂佺儵鏅╅崹璺侯嚕婵犳艾鐏崇€规洖娲﹀▓鏇㈡煟鎼搭垳绉甸柛鎾寸洴閹線宕奸妷锕€鈧敻鎮峰▎蹇擃仾缂佲偓閸愵喗鐓ラ柡鍥悘顏勄庨崶褝韬鐐寸墬閹峰懘鎳栧┑鍕闁哄本娲濈粻娑㈠即閻愭劏鈧剚鐔嗛悷娆忓缁€瀣煙椤旂厧妲绘い顓滃姂瀹曘劑顢樿濮ｅ姊哄Ч鍥х労闁搞劏浜弫顕€骞掗幘瀛樼彙闂傚倷绀侀幖顐ゆ偖椤愶箑纾块柟鎯版閻鏌熼崜褏甯涢柣鎾存礋閺屾洘寰勫☉姘煂婵犲痉銈嗩棄闁宠鍨块、娑樷攽閸℃洘鐫忛梻浣烘嚀瀵爼骞冮崒鐐茬畺鐟滄柨鐣烽悡搴樻斀闁割偁鍨洪崳顖滅磽閸屾艾鈧鎷嬮弻銉ョ；闁瑰墽绮悡鏇㈡倶閻愭彃鈷旈柣顓炴湰娣囧﹪顢曢銏画闂侀潧娲ょ€氫即鐛Ο鍏煎磯闁烩晜甯紞渚€寮婚敐澶婄闁瑰鍋涙禒鎾⒑閸濆嫯瀚扮紒澶屽厴绡撳〒姘ｅ亾闁哄本鐩獮妯尖偓娑欘焽濞堛倝姊洪柅鐐茶嫰婢т即鏌ｉ悢绋库枙鐎规洏鍔戦、娆撴偂鎼达絽鎼告繝鐢靛Х閺佹悂宕戦悙鍝勫瀭闁割偅娲橀崑锟犳煏婢诡垰瀚烽崑銊╂⒑缂佹ê濮夐柛搴涘€濋幃锟犲即閻旂繝绨婚梺瑙勬緲婢у酣骞冮悡搴樻斀闁炽儴娅曢崰姗€鏌＄仦鍓с€掗柍褜鍓ㄧ紞鍡涘磻閸涱垯鐒婃い鎾卞灪閻撳啴鎮峰▎蹇擃仼闁诲繐顕埀顒冾潐濞叉牕鐣烽鍕厺閹兼番鍔岀粻娑欍亜閹炬鍟▍妤€鈹戦悩鍨毄濠殿喗鎸抽幆鈧鑸靛姇缁狙囧箹鐎涙ɑ灏ù婊堢畺閺岋繝宕堕妷銉т患缂備胶濮锋繛鈧鐐寸墬濞煎繘宕滆閸嬔勭箾閹寸偞灏紒澶屾暬婵＄敻宕熼姘祮濠德板€愰崑鎾趁瑰鍫㈢暫婵﹥妞藉畷顐﹀礋椤曞懏钑夐梻浣侯焾閿曘儳鎹㈤崼銉﹀仒妞ゆ梻鏅悷褰掓煃瑜滈崜鐔兼偘椤曗偓楠炴帒螖閳ь剛绮婚敐鍡欑瘈闂傚牊绋掗悡鈧梺闈涳紡閸涱垽绱抽梻浣侯焾閺堫剟鎮烽敃鍋瑰洦顦版惔锝囷紲闁诲繒鍋熼弲顐﹀春閿濆洠鍋撶憴鍕缂佽鐗撻悰顔锯偓锝庝簴閺€浠嬫煙闁箑骞樼紒瀣╃劍缁绘繈鎮介棃娑楃捕濠碘槅鍨伴敃銉х矉瀹ュ拋鐓ラ柛鏇ㄥ亽閸ゃ倝姊虹紒姗嗙劷缂侇噮鍨堕幃锟犲即閵忥紕鍘搁梺鎼炲劘閸庤鲸淇婇悡骞熺懓顭ㄩ崟顓犵暫缂備胶绮惄顖氱暦婵傚憡鍋勯柛婵嗗缂佲晜绻濆▓鍨灈闁挎洏鍔岄埢宥夋晲閸パ冪亰濠电偛妫欓幐鎼佹煥閵堝棔绻嗛柕鍫濇儑閸儱纾婚柟鐐窞閺冨牆宸濇い鏃€鍎崇敮楣冩⒒娴ｇ顥忛柛瀣瀹曚即骞囬弶璺紮闂佸綊鍋婇崣搴♀枔娴犲鐓熼柟閭﹀墮缁狙勩亜閵夛絽鐏柍褜鍓濋～澶娒哄鈧弫鍐閵堝啠鍋撴担绯曟瀻闁圭偓娼欓惂鍕節閵忥絾纭炬い锕備憾閹嫭鎯旈妸锔规嫽闂佺鏈悷銊╁礂鐏炰勘浜滄い鎾跺仧婢с垽鏌熼獮鍨伈妤犵偞甯掕灃闁逞屽墴瀹曟帡濡歌閸犳劙骞栨潏鍓ф偧闁活厼顦甸弻鐔兼倻濡儤顔呭┑鐐叉▕娴滃爼寮崒鐐寸厱婵炴垵褰夌花濂告倵濮橆兙鍋㈡慨濠傤煼瀹曟帒顫濋钘変壕闁归棿鐒﹂崑瀣攽閻樻彃鏆熼柣鐔活潐娣囧﹪濡堕崨顔兼闂佺顑呴崐褰掑Φ閸曨垰绠婚悹铏规磪閵壯呮／闁诡垎浣镐划闂佸搫鏈粙鎴﹀煝鎼淬倗鐤€闁挎繂鏌婅濮婃椽妫冨☉娆忣槱缂備浇顕х€氫即宕洪埀?
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
	var globalAllowedClients []string
	if account != nil && account.IsCodexCLIOnlyEnabled() && s != nil && s.settingService != nil {
		ctx := context.Background()
		if c != nil && c.Request != nil {
			ctx = c.Request.Context()
		}
		if s.settingService.IsOpenAIAllowClaudeCodeCodexPluginEnabled(ctx) {
			globalAllowedClients = []string{openai.AllowedClientClaudeCode}
		}
	}
	return s.getCodexClientRestrictionDetector().Detect(c, account, globalAllowedClients)
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

// isolateOpenAISessionID 闂?apiKeyID 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ゆ繛纭风節瀵鎮㈢悰鈥充壕闁汇垺顔栭悞鍓р偓娑欑箘缁辨挻鎷呴悾灞界墯闂佺锕ュú鐔凤耿娓氣偓濮婅櫣绱掑Ο蹇ｄ簻铻ｅ┑鐘叉搐绾惧潡鐓崶銊р槈缂佺姴缍婇弻鈩冨緞鐎ｎ亞鍘愰梻濠庡墻閸撶喖寮婚垾宕囨殕閻庯綆鍓涢敍鐔哥箾鐎电顎撳┑鈥虫喘楠炲繘鎮╃拠鑼唽闂佸湱鍎ら崺鍫濐焽濞戙垺鈷?session 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻樻爠鍥ㄧ厱闁靛鍨哄▍鍥煕濡厧鈻堟慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵＄偑鍊х紓姘跺础閸愯尙鏆﹂梻鍫熶緱濞尖晜銇勯幋鐘蹭沪婵＄偘绮欓妴渚€寮崼婵堫槹濡炪倖鎸鹃崑娑氱不娴煎瓨鈷掗柛灞剧懅椤︼箓鏌熷ù瀣⒉缂佹鍠庤灃闁告侗鍘鹃悰銉╂⒑閸濆嫮鈻夐柛妯垮亹缁牓宕奸妷锔惧帾婵犵數鍋熼崑鎾斥枍閸℃稒鐓冮梺鍨儏閻忔挳鏌″畝瀣М闁诡喓鍨介幃鈩冩償濠靛棙鐎抽梻鍌欑劍閹爼宕愬Δ鍛獥閹兼番鍔岄悡姗€鏌熸潏鎯х槣闁轰礁顑夐弻宥堫檨闁告挾鍠曞Λ銏ゆ⒑鐟欏嫬绀冩繛澶嬬洴瀹曠懓鈹戦崱蹇旀杸闂佺粯锚閻ゅ洦绔熷Ο鑲╃＜妞ゆ劑鍨绘晥闂佸搫鏈惄顖炪€侀弴銏犖ч柛娑卞枤娴滄牠姊绘担鍛婂暈闁荤喆鍎辫灋婵犻潧妫鏍ㄧ箾瀹割喕绨诲ù鑲╁█閺屾盯寮撮妸銉ょ凹濠电偛鐗滈崢濂稿煘閹达附鍊烽柤纰卞墯閸曢箖姊虹粙鍖℃敾闁绘绮庨崚鎺斺偓锝庝憾閸氬顭跨捄鐚存敾鐎规挸绉撮—鍐Χ閸℃ê鏆楁繝娈垮枤閸忔﹢寮鍜佺叆闁割偆鍟块幏?
// 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆樻О闂備浇顕栭崹鎶藉磻閵堝宓侀柡宥庣仈鎼搭煈鏁嗛柍褜鍓氭穱濠冪附閸涘﹦鍘藉銈嗘尵閸ｃ儱鈻撳鍕垫闁绘劕顕晶顏堟嚕閹邦厹浜滈柟鍝勬娴滅偓绻濆鏋€曟禍鍦磼鏉堛劌娴鐐叉喘椤㈡顦抽柣銈勭窔濮婃椽宕崟顓犲姽缂備胶绮换鍌炴偩閻戣棄閱囬柡鍥╁仧閸樺憡绻涙潏鍓у埌婵炴潙瀚濠囧锤濡も偓閻掑灚銇勯幒鎴濇灓婵炲吋鍔栫换娑㈠矗婢跺苯鈪瑰銈嗘穿缂嶄礁鐣锋總绋垮嵆闁绘劗鏁搁弶浠嬫⒒娓氣偓濞佳団€﹂崼銉ョ？闁告繂瀚烽悞浠嬫煏婵炵偓娅嗛柣鎾存礋閺屽秹鍩℃担鍛婄亾濠电偛鐗婂鑽ゆ閹烘鍋愮€瑰壊鍠氶崥瀣攽椤旂》宸ラ柟纰卞亰閹箖鎮滈挊澶屽€為梺瀹犳〃閼冲爼顢曢崗鑲╃瘈婵炲牆鐏濋弸娑㈡煥閺囨ê鈧繈鍨鹃敃鍌涘€婚柦妯侯槹濡差剟姊洪柅鐐茶嫰婢ь垶鏌曢崶褍顏鐐村浮瀹曞崬顪冮幆褜妫滅紓鍌氬€搁崐鐑芥嚄閼稿灚鍙忛悗闈涙憸閻捇鏌ｉ悢绋款棆闁哄棴绠撻弻锟犲炊閵夈儳浠鹃梺缁樺笩閸嬫劙鍩€椤掆偓缁犲秹宕曢柆宥呯疇闁规壆澧楅崑顏堟煕閵夘喖澧い銉ワ攻閵囧嫰骞囬埡浣轰痪婵犮垼娉涚€氼喚妲愰幒鏂哄亾閿濆骸浜滈柣蹇婃櫅椤?API Key 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵挳鎮欏ù瀣壕鐟滅増甯掔壕鍧楁煙鐎电校闁哥姵鍔欓弻锝呂旈埀顒勬偋閸℃瑧绠旈柟鐑樻⒒绾惧ジ鏌嶈閸撴艾顕ラ崟顓濇勃缂佸銇樻竟鏇㈡⒑缁嬭法绠版い锔诲灡缁傚秵銈ｉ崘鈹炬嫼闂備緡鍋嗛崑娑㈡嚐椤栨稒娅犻柟缁㈠枟閻撴洟鏌嶇憴鍕姢濞存粎鍋撴穱濠囨倷椤忓嫧鍋撻弽顐ｆ殰濠电姴瀚惌鍡椼€掑锝呬壕閻庤娲滈弫濠氥€佸Δ鍛妞ゆ帒鍊搁獮鍫ユ⒒娴ｇ懓顕滅紒璇插€婚幑銏ゅ箳濡も偓缁€鍡涙煟濡も偓閻楀嫭绂嶅鍫熺厸闁搞儲婀圭花鐣岀磼婢舵ê鏋熼柕鍥у婵偓闁斥晛鍟伴ˇ浼存⒑鏉炴壆顦﹂柛鐔告尦瀹曟椽鍩€椤掍降浜滈柟鍝勬娴滄儳鈹戦悙鏉戠祷闁诲繑绻堥崺鐐哄箣閿旇棄浜瑰銈嗘閸嬫劖瀵奸崶褉鏀介柣鎰▕濡插綊鏌ｉ埡濠傜仸鐎殿噮鍋婂畷姗€顢欓懞銉︾彇闂備胶顭堢悮顐﹀磹閺嶎厼鐤鹃柨婵嗘噳閺€浠嬫煟閹邦剙绾ч柍缁樻礀闇夋繝濠傚缁犵偟鈧娲橀崝娆撳箖娴犲宸濆┑鐘插楠炴姊绘担铏瑰笡闁哄被鍔戦獮澶愭晬閸曨剦鍋ㄩ梺鍛婄懃椤︻厽绂嶅鍫㈠彄闁搞儵顥撶€佃偐绱掗妸銊ヤ汗缂佽鲸甯炵槐鎺懳熼懖鈺冩澖濠电姵顔栭崰鎺楀磻閹剧粯鈷戦悗鍦У椤ュ淇婇锝囨创鐎规洘绻冮ˇ鐗堟償閿濆浄绱冲┑鐐存尰閼规儳煤閵堝鍑犳繛鎴欏灪閻撱儵鏌￠崶銉ュ闂侇収鍨堕弻鏇㈠幢閺囩媭妲梺瀹狀嚙妤犳悂顢氶妷銉㈡斀濠电姴瀛╃紞鍫ユ⒑鐠団€崇仭婵☆偄鍟村顐㈩吋婢跺﹦顦板銈嗘尭椤︻參宕濋幋婵愭綎闁绘垶蓱婵粓鏌熷▓鍨灈濠碘€茬矙濮婃椽宕ㄦ繝鍐ｆ嫻闂佽崵鍠嗛崕鐢稿箖妤ｅ啯鍊婚柦妯猴級閳哄懏鐓冮柛婵嗗閺€濠氭煛閸涱喚绠炴慨濠冩そ楠炴劖鎯旈敐鍌涱潔闂佽瀛╅惌顕€宕￠幎鐣屽祦闁圭増婢樼粈鍌炴煠濞村娅囬柣锕€鐗撳鍝勑ч崶褏浼堝┑鐐板尃閸愨晜鐦庨梻鍌氬€峰ù鍥ь浖閵娾晜鍊块柨鏇炲€哥粻鏍煕鐏炵偓鐨戦柡鍡畵閺岀喓鈧稒顭囨俊鍥ㄤ繆閹绘帞澧涘ǎ鍥э躬椤㈡稑顫濇潏鈺婂敼婵犵绱曢崑妯煎垝濞嗘挸违濞达絽澹婂銊╂煃瑜滈崜姘跺箞閵娾晛鐒垫い鎺戝閻撶喐淇婇姘倯闁哄棌鏅犻弻锟犲幢閳轰椒绨婚梺瀹狀潐閸ㄥ潡骞冨▎蹇ｅ晠妞ゆ柨鍚嬮宥呪攽閻樻剚鍟忛柛鐘愁殜楠炴劙骞庢慨鎰ㄥ亾娓氣偓瀵粙顢橀悙鑼崺婵＄偑鍊栧濠氬储瑜庨幈銊╁川鐎涙ǚ鎷绘繛杈剧秬濡嫰宕ヨぐ鎺撶厱闁绘ê鍟挎慨澶愭煕閹烘挸绗уù鐙呯畵閹矂顢曢妷顔界秷濡炪倧濡囨晶妤呭箚閺冨牆鐏崇€规洖娲ｉ崰濠冪節閻㈤潧啸闁轰礁鎲￠幈銊╁级閹炽劍妞芥俊鑸靛緞鐎ｎ亞褰呴梻浣虹帛閺屻劑宕ョ€ｎ喗鍋傞煫鍥ㄦ惄閻斿棝鏌ら崫銉︽毄闁宠棄顦伴妵鍕煛婵犲倸鈷嬮梺鍝勬湰缁嬫垼鐏掗梺鍛婄箓鐎氼垶宕洪悙鐑樷拺閻庡湱濯鎰版煕閵娿儲鍋ラ柕鍡曠閳诲酣骞橀搹顐闂備線娼ч悧鍡欌偓姘煎枟濞煎寮崼鐔叉嫽婵炶揪绲介幉锟犲疮閻愮儤鐓欑紒瀣儥閻撳ジ鏌熼姘伃妞ゃ垺鐩幃娆撴嚑椤掑倹姣囧┑鐘殿暯濡插懘宕瑰畷鍥у灊妞ゆ牗绮庣粈濠囨煕閵夘喖澧柣鎾崇箰閳规垿鎮欓幋婵嗘殲濠殿垰銈搁弻娑㈡偄閻戣棄寮板┑顔硷功缁垶骞忛崨顖滅煓婵炲棛鍋撻ˉ鎴︽⒒娴ｄ警鐒炬い鎴濇噹铻炴繝闈涙閺嗭箓鏌ｅΔ鈧悧濠囧磿閻斿吋鐓忓┑鐘茬箳閻ｉ亶鏌嶈閸撴瑩鎮ユ總绋跨畺婵°倕鎳忛崑銊╂煟閵忋垹浠柍褜鍓欓敃顏堝蓟濞戞埃鍋撻敐搴′簼閻忓繒鏁婚弻鐔煎矗婢跺鍞夐悗瑙勬礈閸犳牠銆佸Δ鍛＜闁挎梹鍎抽弸鐘绘⒒閸屾艾鈧兘鎮為敃鍌氱畺闁割偅娲栫粈澶愭倵閿濆骸澧柛銊︾箖缁绘盯宕卞Ο璇叉殫閻庤鎸风欢姘跺蓟濞戙垹绠涢梻鍫熺⊕閻忓牆顪冮妶搴′簻闁硅櫕锕㈠濠氬Ω閵夈垺顫嶅┑鈽嗗灟鐠€锔藉閸愩劉鏀介柨娑樺濡炬悂鏌涢悩鎰佹畼闁瑰箍鍨归埞鎴犫偓锝庝憾濞煎﹪姊洪幐搴ｇ畵婵☆偅鐩俊鎾箛椤掑瀵岄梺闈涚墕濡鎮橀妷锔剧閻忕偛鍊告俊鍧楁煕?session_id/conversation_id闂?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鍝勭Ф閸斿秵銇勯弬鎸庡缂佺粯绻傞銉╂煥鐎ｎ偆鍑￠梺閫炲苯澧柟绋垮⒔閸掓帡宕奸悢铏规嚌闂侀€炲苯澧柣锝囨焿閵囨劙骞掗幋鐑嗘婵犵妲呴崑濠傤焽瑜忕槐鐐寸節閸パ呯暫闂佸疇妗ㄧ欢锟犲疮閸涱喓浜滈柡鍐ㄥ€哥敮璺侯熆閻熸壆澧㈢紒杈ㄦ崌瀹曟帒鈻庨幒鎴濆腐濠电姵顔栭崰妤呭箰閹惰棄绠栨俊顖欒濞尖晠鎮规潪鎵Э闁挎洖鍊归悡鏇㈡煛閸ャ儱濡奸柡瀣ㄥ€濋弻锝夊Χ閸屾矮澹曢梻鍌氬€风粈浣革耿闁秴纾婚柟鐐墯閻掕姤绻涢崱妤冪畾闁哄绉堕幉鎼佸棘濞嗙偓缍庨梺鍛婄箓鐎氼噣寮抽敃鍌涚厱闁靛鍨哄▍鍛磼濡ゅ绡€婵﹥妞藉畷顐﹀礋椤曞懏钑夐梻浣告啞濮婂綊宕归崹顔炬殾婵炲樊浜濋崐鐑芥煕濠靛棗顏い鎾存そ濮婃椽骞愭惔銏╂⒖濠碘槅鍋勭€氼厾绮嬪鍫涗汗闁圭儤鎸撮幏娲⒑闂堚晛鐦滈柛妯哄⒔閺侇喖鈽夐姀鈥充缓濡炪倖鐗楁笟妤€鈻撻弮鍫熺厓缂備焦蓱瀹曞瞼鈧娲栭妶绋款嚕閹绢喗鍋勯柛娆忣槸椤忓湱绱撻崒姘偓椋庢閿熺姴鍨傞柛妤冧紳閸濆嫀鐔烘偘閳╁啰鈧椽姊虹化鏇炲⒉閼垦兠瑰鍕煉闁哄本娲樼换娑㈠垂椤旂厧啸闂傚倸顭崜锕傚礈濞戙垹鐒垫い鎺嶇贰閸熷繘鏌涢悩宕囧⒌闁炽儻绠撻幃婊堟寠婢跺鈧剟姊鸿ぐ鎺戜喊闁告鍋愬▎銏ゆ倷濞村鏂€闂佺粯鍔忛弲婊堝闯娴煎瓨鐓涢柛娑卞灠閳诲牏鈧鍠栭…宄邦嚕閹绢喗鍋勫瀣捣閻涱噣姊绘担绋款棌闁稿鎳愰幑銏ゅ磼濞戞瑥寮块梺鐓庢憸閺佸摜绮绘ィ鍐╁€甸柣銏☆問閻掗箖鏌嶇拠鑼ⅱ缂佽鲸甯￠幃鈺佺暦閸パ€鍚傛俊鐐€ら崑鍕崲濮椻偓楠炴牞銇愰幒鎴炲祶濡炪倖鎸炬刊瀵告閸欏绡€缁剧増蓱椤﹪鏌涢妸锕€鈻曠€规洏鍨奸ˇ褰掓寠閻斿鐔嗛悹楦挎閻忚京鐥幑鎰棄闂囧鏌ㄥ┑鍡樺闁搞倐鍋撶紓鍌欐祰鐏忔瑧鍒掗婊勫床婵炴垶纰嶉崗婊冾渻鐎ｎ亝鎹ｉ柣锕€鐗撳铏圭磼濡湱绻侀梺鍝ュУ缁嬫挾鍒掔拠娴嬫婵☆垶鏀遍～宥呪攽閻愬弶顥滅紒璇差儏鏁堟俊銈呮噺閳锋垿鏌ゆ慨鎰偓鏇炵摥婵犵數鍋犻婊呯不閹烘桅闁圭増婢樼粈鍐┿亜閺冨洦顥夋繛鍫濈焸濮婃椽宕滈懠顒€甯ラ梺鍝ュУ鐢€愁嚕閵婏妇顩烽悗锝庡亞閸樹粙姊鸿ぐ鎺戜喊闁告挻鐟ч惀顏囶槼闁靛洤瀚版俊鐤槻濞寸娀浜堕弻锛勪沪鐠囨彃顬堝銈庡亝缁诲牓骞婂鍫熷仺闂傚牊绋戠粻鐐烘⒒閸屾瑦绁版い顐㈩樀瀹曟洟宕橀懠顒佹濡炪倖甯婂鎺旀崲閸℃稒鐓熼柨婵嗘噽閻忚京鐥幑鎰汗缂佽鲸鎸婚幏鍛嫚閿涘嫬濮烘繝鐢靛仦瑜板啴鎮洪妸鈹库偓鍐Ψ閳哄倸鈧鈧懓澹婇崰鏍礈閸洘鈷戦悹鍥ｂ偓铏亶闂佽崵鍟欓崶褏顦梺鍦劋濮婂綊宕伴崱娑欑厱闁哄洢鍔屾禍钘壝归崗鑲╃煉婵﹤顭峰畷鎺戭潩椤戣棄浜惧瀣捣閻棗銆掑锝呬壕濡ょ姷鍋為〃鍛粹€﹂妸鈺佺妞ゆ巻鍋撶€殿喖娼￠弻锝嗘償閵婏附閿梺纭呭Г缁捇骞嗗畝鍕缂備焦顭囬崢閬嶆⒑鐟欏嫬绀冮悘蹇旂懇钘濋柨鏇炲€归悡鏇㈡煏婵犲繐顩紒鐘崇墵閺屾洟宕惰椤忣厽顨ラ悙鎼劷闁圭懓瀚伴幃婊兾熺亸鏍т壕闁煎鍊楃壕钘壝归敐鍥剁劸妞わ絾濞婇弻娑氣偓锝冨妼閳ь剚绻傞锝嗙節濮橆厼浜滈梺绋跨箺閸嬫劙宕濋悜鑺モ拺闁圭瀛╃壕鐢告煕鐎ｎ偅灏い顓″劵椤﹀爼鎮樿箛鏃傛噰妤犵偛鍟撮、娆戜焊閺嶃劍鏉搁梺璇插嚱缂嶅棝宕戦崱娑樺偍濞寸姴顑嗛悡娆撴煠濞村娅堥挊鐔兼⒑闁偛鑻晶顕€鏌涢悢绋款嚋婵炵厧绻樺畷銊р偓娑櫱氶幏铏圭磽娴ｅ壊鍎愭い鎴炵懇瀹曟洟骞囬悧鍫㈠幍闂佽崵鍠撴晶妤呭窗濡椿娈介柣鎰絻閺嗘瑩鎽堕弽顓熺厱婵炴垵宕弸鐔哥箾閸涱喚鐭掗柡宀€鍠栭幊鏍煛閸曞﹤顦甸弻宥堫檨闁告挻宀稿畷婊冾潩椤掑鍔烽梺缁樺姦閸忔稓鎹㈤崱娑欑厽闁规澘鍚€缁ㄥ鏌嶈閸庢悂宕ㄩ鍕闂傚倸鐗婄粙鎾绘倿閸濄儮鍋撶憴鍕濠电偛锕顐﹀箛閺夋寧銇濇繛杈剧到濠€杈╃矙娴ｇ硶鏀介柣妯虹仛閺嗏晛鈹戦鐐毈闁硅櫕绻堝畷婊嗩槻妞ゃ儲宀搁弻娑滅疀閹捐櫕鍊梺鍛婄懃缁绘垿骞堥妸銉庢棃鍩€椤掆偓铻炴俊銈勭劍濞呯姵銇勯弽銊с€掔紒鐘荤畺閺屾盯顢曢顫盎閻庢鍣崳锝夊蓟閻旂⒈鏁婄紒娑橆儐閻ｅ爼姊哄畷鍥╁笡闁圭懓娲ら悾鐤亹閹烘繃鏅╅梺浼欑到婵傛棃鏌囬鐐粹拻闁稿本鐟ㄩ崗宀€绱掗鍛仸闁诡啫鍥у嵆闁靛繒濮堣閺岀喖姊荤€靛壊妲弶鈺傜箖缁绘稒娼忛崜褏袣闂佺锕ｇ划娆忣嚕椤曗偓瀹曠厧鈹戦崼婵喰曞┑锛勫亼閸婃牜鏁幒鏂哄亾濮樼厧澧撮柟顕嗙節閸ㄩ箖骞囨担鍝勬暩闂佽崵濮惧銊ф媰閿曞倹鍋傛い鎺嶈兌缁犻箖鎮樿箛鏃傚婵炲懎妫濋弻鏇㈠炊瑜嶉顓燁殽閻愭惌鐒介柟鐟板濞碱亪鎮ч崼婵堝幈闂傚倸鍊烽悞锔锯偓绗涘懐鐭欓柟瀵稿У瀹曞弶淇婇娆掝劅婵炲吋鐗滅槐鎾存媴鐠囷紕鍔烽梺鑽ゅ枎缂嶅﹪寮诲鍫闂佸憡鎸婚悷鈺呭灳閿曞倸鐐婃い蹇撶У閳诲姊绘笟鈧鑽ゅ緤閻ｅ瞼绀婂┑鐘插亞濞兼牗绻涘顔荤盎鐎瑰憡绻傞埞鎴︽偐閹绘帗娈銈忚吂閺呯姴顫忓ú顏勫窛濠电姴鍟棄宥夋倵閸忓浜鹃梺褰掓？闂勫秹鍩€椤戣法顦﹂摶锝夋偡娴ｉ潧鈧挸效濡ゅ懏鈷戦柛锔诲幖閸斿鏌熼崨濠冨唉妤犵偛绻掔槐鎺懳熼懖鈺婂晭闂佸搫顦悧鍡樻櫠娴犲绀嗗ù鐓庣摠閻撴瑩鎮归崶鍥ф噽閿涚喖姊洪悷鏉挎Щ缂傚秴锕畷娲礋椤栨氨顦ㄩ梺鎸庢濞夋洟鎯勬惔銊︹拻濞撴埃鍋撴繛浣冲毝銊╁焵椤掑倻纾奸柣妯哄暱閻忥箓鏌熸笟鍨鐎规洖鐖奸、妤佹媴閸濆嫬濡囬梻鍌欒兌绾爼宕滃┑瀣櫔婵犵數鍋為幆宀勫窗濮樿泛鐒垫い鎺戝枤濞兼劖绻涢崣澶岀煉鐎规洘顨呴～婊堝焵椤掑嫬违濞达絽澹婂銊╂煃瑜滈崜鐔肩嵁閸愵喖围闁糕剝锚椤庢捇姊洪棃鈺佺槣闁告﹢绠栭崺锝夊Ψ閳哄倵鎷婚梺绋挎湰閼归箖鍩€椤掆偓閸㈡煡婀侀梺鎼炲労閸撱劎绱為弽褜鐔嗛柤鎼佹涧婵箓鏌℃担闈╄含闁哄备鈧剚鍚嬮幖绮光偓宕囶啇缂傚倷鐒﹂崝鏍€冮崱妯尖攳濠电姴娲ゅ洿闂佸憡渚楅崢钘夆枔濠婂牊鈷戠紒瀣儥閸庢劙鏌熼崨濠冨€愰柨婵堝仜閳规垹鈧絽鐏氶弲锝夋⒑缂佹鎲块柛瀣崌閺岋繝宕遍鐘碉紵缂備浇椴搁幐濠氬箯閸涘瓨鎯為柣鐔稿椤愯偐绱撻崒娆愮グ妞ゆ泦鍥ㄥ亱闁糕剝姘ㄦ禍閬嶆⒒娴ｅ憡鍟為柛鏃€鍨垮畷婵囧緞婵烆澁缍佸畷濂告偄缁嬪灝浼庢繝娈垮枟椤ㄥ懎螞濡ゅ懎鍌ㄩ柟缁㈠枟閻撴瑩鎮樿箛搴ｎ槮濠⒀嗗皺缁辨帞绱掑Ο鑲╃杽闂佽鍠曠划娆徫涢崘銊㈡婵°倕鍟畷鐔兼⒒閸屾瑧绐旀繛浣冲洦鍋嬮柛鈩冪☉缁犵儤绻濇繝鍌涘櫣闁搞劍绻傞埞鎴︽偐鐎圭姴顥濋梺鍛婂姀閸嬫挻淇婇悙顏勨偓鏍箰妤ｅ啫纾婚柣妯款嚙缁犳椽鏌￠崶銉ョ仾闁抽攱鍨圭槐鎾存媴閼测剝鍨块崺娑㈠箳濡や胶鍘卞┑鈽嗗灣婵挳骞婇幇鐗堝剹闁糕剝绋掗悡鏇熴亜閹邦喖孝闁诲浚浜弻娑㈡偐閸愭彃顫庨梺閫炲苯澧紒鐘茬Ч瀹曟洟宕￠悙宥嗙☉閳藉锝為钘変壕闁告稒娼欑粻鐢告煙閸濆嫭顥為柡鍌楀亾闂傚倷鑳堕…鍫ュ嫉椤掑倸鏋堢€广儱顦悿顕€鏌ｉ幇顔煎妺闁?
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
		zap.String("request_client_ip", strings.TrimSpace(ip.GetClientIP(c))),
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

// GenerateSessionHashWithFallback 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆洨鎽傛繛鍏煎姇椤潡鎮烽悧鍫！闂佸搫鎳撳▔娑滅亙闂佸憡渚楅崢楣冩晬濞戙垺鐓熼幖鎼灣缁夐潧霉濠婂嫮澧电€殿喓鍔嶇粋鎺斺偓锝庡亞閸樿棄鈹戦埥鍡楃仴妞ゆ泦鍛筏缂備焦菧娴滄粓鐓崶褜鍎忛柍褜鍓氱换鍫濐嚕鐠囨祴妲堟慨姗堢到娴滈箖鏌ㄥ┑鍡涱€楀ù婊勭箘閳ь剝顫夊ú鏍儔婵傜鐒垫い鎺戝枤濞兼劖绻涢崣澶屽⒈缂佽京鍋炵换婵嬪炊閵夈垹浜惧〒姘ｅ亾鐎殿喗鎸虫慨鈧柣妯荤垹閸ャ劎鍘遍梺闈涱槶閸ㄥ搫鈻嶉崨瀛樺仺妞ゆ牗绋掑畷灞炬叏婵犲偆鐓肩€规洘甯掗埢搴ㄥ箛椤斿搫浠掑┑鐘殿暯閸撴繈濡靛鍫濈閹肩补妾ч弸搴ㄦ煏韫囧鈧牠寮查幖浣圭厽婵☆垰鐏濋惃娲煟濞戞牕鍔︽慨濠冩そ瀹曨偊宕熼娑欑€遍梻浣告啞閻熴儳鎹㈤幒鏃€顫曢柣鎰惈閸愨偓濡炪倖鎸鹃崰鎾诲储椤忓牊顥婃い鎰╁灪婢跺嫭绻涢崣澶屽ⅹ闁哄棌鏅犲缁樻媴閸涘﹥鍎撳┑鈽嗗亝椤ㄥ懓鐏嬪┑掳鍊曢幊搴ㄦ偂濮椻偓閺岀喐娼忔ィ鍐╊€嶉梺绋款儐閸旀瑩寮婚悢铏圭＜婵☆垵娅ｉ悷鎻掆攽閻愯尙澧曞褍娴峰Σ鎰板箻鐎涙ê顎撻梺鍛婃尭瀵墎鐟ч梻鍌欑劍閹爼宕瑰ú顏呭亗闁跨喓濮撮弰銉╂煟閹邦剚鎯堢紒鐙呯秮閺屻劑寮崶顭戞闂侀€炲苯澧い顓犲厴瀵鈽夐姀鐘栥劑鏌曡箛濠傚⒉闁绘繃妫冨铏瑰寲閺囩喐鐝旈梺鍏兼た閸ㄥ爼宕洪悙鍝勭闁挎棁妫勯埀顒傚厴閹銈﹂幐搴哗濠殿噯绲婚崹铏规崲濠靛鍋ㄩ梻鍫熺◥濮规姊虹粙娆惧剱闁圭懓娲ら～蹇涙惞閻熸澘顕ч梺鍝勬川閸犳劕顭块幒妤佲拺缂佸顑欓崕鎰版煙閸涘﹥鍊愰柍銉︽瀹曟﹢顢欓崲澹洦鐓曢柍鈺佸枤濞堟梹绻涢崗鐓庣伌婵﹨娅ｇ划娆忊枎閹冨闂備礁婀遍幊鎾趁洪鐑嗗殨閻犲洦绁村Σ鍫ユ煏韫囧ň鍋撻崗鍛泿缂傚倸鍊风粈渚€顢栭崱娑辨晞婵炲棙鎸搁崥褰掓倵閿濆骸澧扮痪鎹愭闇夐柨婵嗙墛椤忕娀鎮介娑氭创闁哄矉缍侀崺鈩冪節閸屾粈鍝楅梻浣风串缁插潡宕楀Ο铏规殾濠靛倻顭堥崡鎶芥煏婵犲繘妾繛鍛嫰閳规垿鎮欓懜闈涙锭缂備浇寮撶划娆撶嵁閺嶎収鏁冮柨鏃€鍎崇粊锕€鈹戞幊閸婃洟骞婅箛娑樼厱闁圭儤鍨埀顒佸笒椤繈鏁愰崨顒€顥氬┑鐘垫暩閸嬫﹢宕犻悩璇插窛妞ゆ梻鍘х花銉╂⒒娴ｇ顥忛柛瀣╃窔瀹曟洟鏌嗗鍛€梺绋挎湰缁秹宕伴幇鐗堢厽婵°倐鍋撻柣妤€妫涚划顓㈠箳濡や胶鍘遍梺鐟邦嚟婵挳鍩㈤崼銉︾厸鐎光偓閳ь剟宕伴幇顒夌劷闊洦鏌ｉ崑鍛存煕閹般劍娅撻柍褜鍓欑粔鐟邦潖閾忓湱纾兼俊顖濐嚙闂夊秴鈹戦悙璺虹毢闁哥姵鐗曢锝嗙節濮橆剙宓嗛梺缁樻濞咃綁鎯侀崼銉︻棅妞ゆ劑鍨烘径鍕煙鐏忔牗娅呴悡銈嗐亜閹捐泛鍓辨繛鎾愁煼閺屾洟宕煎┑鍫㈩唺缂備礁顑呴…鐑藉蓟閿濆绠奸柛鏇ㄥ幘閻﹀牆螖閻橀潧浠滈柛鐕佸亯閻忓啴姊洪崨濠佺繁闁哥姵鐗楃粋鎺懨洪鍛嫽闂佺鏈銊︽櫠濞戞氨纾奸悗锝庡亜閻忥妇绱掗崒娑樼瑨閾伙絽銆掑鐓庣仭閻庨潧鐭傚娲濞戞艾顣烘俊銈囧Т閹诧紕绮嬪鍛斀闁割偁鍨婚敍婊堟煟閻樺弶澶勭憸鏉垮暣閸┾偓妞ゆ巻鍋撴繛纭风節閻涱噣宕橀埞鍨簼闂佸憡鍔忛弲娑㈠焵椤掆偓椤兘寮婚敃鈧灒濞撴凹鍨辨闂備胶顭堥敃锕傚磻閵堝拋娼栨繛宸簼椤ュ牊绻涢幋锝勫惈闁告梹鎮傚娲箹閻愭彃顬夊┑锛勫仒缁瑩鐛崘銊庣喖鎳￠妶鍥ㄥ殞闂備線鈧偛鑻晶顔姐亜椤愩垻绠茬紒缁樼箓椤繈顢楅崒锔惧簥濠电姷鏁搁崑娑樜涘▎鎾虫槬闁割偅鎯婇敐澶樻晪闁逞屽墮椤繘鎼圭憴鍕幑闂佸憡绮堢粈浣糕枔濠靛牏纾藉ù锝勭矙閸濈儤绻涢懠顒€鏋涢柣娑卞櫍楠炲鏁冮埀顒勶綖閸涘瓨鐓忛柛顐ｇ箖閸ｄ粙鏌ㄥ☉娆戠煉婵﹨娅ｇ槐鎺懳熼崫鍕垫綌婵犵數鍋涢崥瀣偡閿斿墽鐭夌€广儱顦伴悞鑲┾偓骞垮劚濡矂骞忛搹鍦＝濞撴艾娲ら悘鈩冪箾閸欏澧柡鍛埣瀵濡烽敂鎯у箺闂備胶绮鍦崲閸岀偛绠犻柟閭﹀幘閳绘梻鈧箍鍎遍ˇ浼存偂濞戙垺鐓曢柟鎵虫櫅婵″灝顭胯閻╊垶寮婚敐澶婄閻庢稒顭囬ˇ浼存⒑閸濆嫭婀版繛鍙壝銉╁礋椤愬鍠栭幃婊兾熼梻鎾仐濠电姷鏁告慨顓㈠箯閸愵亙娌柣锝呭濡楁捇姊?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞鐟滃繘寮抽敃鍌涚厽闁靛繈鍩勯悞鍓х磼閹邦収娈滈柡宀€鍠栭弻鍥晝閳ь剟寮稿☉銏＄厱闁靛濡囩粻鐐搭殽閻愬澧懣鎰亜閹哄棗浜鹃梺璇叉禋娴滄繄鎹㈠☉銏犲窛妞ゆ牗顕撮敐鍚冲酣宕惰闊剙鈹戦垾宕囧煟鐎规洏鍔戦、娆撳礂绾板彉鍝楃紓鍌氬€搁崐鐑芥嚄閼稿灚鍙忛柟鎯版绾偓闂佽鍎煎Λ鍕不閺嶎偅鍠愰柣妤€鐗嗙粭鎺楁煟閹邦剨鏀婚柕鍥у缁犳盯骞樼紓搴涘灪缁绘盯宕奸悢铏圭杽濠殿喖锕ュ浠嬪箠閺嶎厼鐓涘〒姘处濮ｅ姊虹拠鍙夋崳闁荤喐濞婂畷鍫曗€栭濠勭暤闁哄本鐩鎾Ω閵壯傚摋缂傚倷鑳舵慨鎾€﹂悜钘夎摕婵炴垶顭傞弮鍫濈闁靛鍊栭崺娑樷攽閻愬樊鍤熷┑顕€绠栧畷鎴﹀箛椤旂瓔娼熼梺瑙勫劤閻°劍鍒婇幘顔界厱婵犻潧妫楅悵鏃堟煏韫囧鐏痪鍓ф櫕閳ь剙绠嶉崕閬嶅箯閹达妇鍙曟い鎺戝€甸崑鎾斥枔閸喗鐏曞銈嗘肠閸パ呭弨婵犮垼娉涜癌闁绘柨鍚嬮悡銉╂倵閿濆骸浜濇繝銏″灴濮婂宕掑▎鎺戝帯缂佺虎鍘奸悥鐓庣暦濠婂啠鏀介悗锝冨妷閸嬫捇宕橀鐘垫澑闂佸搫鍊归娆撳吹閵堝鈷戦梺顐ゅ仜閼活垱鏅剁€涙ɑ鍙?session_id/conversation_id/prompt_cache_key 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鎾煙椤斿吋鍋ユい銏″哺閸┾偓妞ゆ帒瀚拑鐔兼煟閺冨倸甯剁紒鈧崼婢濆綊鏁愰崶褍濡洪梺鐟板级閿曘垹顫忛悜妯诲闁规鍣Σ顕€姊洪幐搴㈠濞存粠浜滈锝嗙節濮橆厼浜滈梺绋跨箺閸嬫劙宕濋悜鑺モ拺闁圭瀛╃壕鐢告煕鐎ｎ偅宕岄柡宀嬬磿閳ь剨缍嗛崜娆撳几閹达附鐓忛柛銉戝喚浼冨銈冨灪濞茬喖寮幇鏉垮耿婵炲棙蓱琚ㄧ紓鍌氬€搁崐鐑芥嚄閸撲礁鍨濇い鏍ㄧ矋瀹曟煡鏌涢锝嗗剷婵炴垯鍨圭粈鍐煏婵炲灝鐏い顐㈢Т閳规垿鎮欓崣澶樻濠电偛鐡ㄩ懝鎯у祫闂佸憡顨堥崑鎰板绩娴犲鐓熸俊顖濐嚙婢ь垶鏌涢悢椋庣闁哄本鐩俊鎼佸煛閳ь剟骞夐悙顒夋闁绘劖娼欐慨宥嗩殽閻愭煡鍙勯柟绋匡攻瀵板嫬鐣濋埀顒勫汲椤愶附鐓熼柣鏂挎憸閻苯顭胯椤ㄥ牓寮鈧獮鎺楀籍閸屾瑧鐟濋梻浣告贡閸庛倝銆冮崱娑欏亗婵炲棙鎸婚悡鐔兼煙鐎甸晲绱虫い蹇撶墛閸婂爼鏌曟径娑滅濞存粍绮嶉妵鍕箳閸℃ぞ澹曟俊鐐€х粻鎾愁焽瑜旈敐?fallbackSeed 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵挳鎮欏ù瀣壕鐟滅増甯掔壕鍧楁煙鐎电校闁哥姵鍔欓弻锝呂旈埀顒勬偋閸℃瑧绠旈柟鐑橆殕閻撴洟鏌曟繛鍨姎闁逞屽墯閹倿宕洪悙鍝勭闁挎棁妫勬禍褰掓煟鎼粹剝璐″┑顔芥尦钘熸繝闈涚墢绾捐棄霉閿濆洦鍤€濠殿喖鐗婇妵鍕Ω閵夘垵鍚悗娈垮枛椤兘骞冮姀銈呯闁绘挸绨肩純鏇㈡⒒娴ｇ顥忛柛瀣瀹曚即骞樼拠鑼槱闂佽法鍠撴慨鐢稿煕閹达附鐓欓柤娴嬫櫅娴犳粌鈹戦垾鐐藉仮闁哄本绋掔换婵嬪磼濠婂憛銊ノ旈悩闈涗粶闁绘鎸绘穱濠囧箹娴ｈ倽銊╁箹鏉堝墽鍒伴柡瀣灩閳ь剙鐏氬妯尖偓姘煎枤閸掓帒鈻庨幘宕囶唶闁硅偐琛ラ埀顒€鍟块ˉ鎺楁⒒閸屾艾鈧悂宕愰悜鑺ュ€块柨鏇楀亾妞ゎ亜鍟村畷绋课旈埀顒勫磼閵娿儍褰掓偐瀹割喖鍓遍梺绋款儜缁绘繈寮诲澶婁紶闁告洦鍓欏▍锝夋⒑閽樺鏆熼柛鐘冲姉閹广垹鈽夐姀鐘殿吅濠电偛妫楃换鎺撶閼测晝纾藉ù锝呮惈鏍＄紓浣割儐鐢剝淇婇悽绋跨妞ゆ牗鑹鹃崬銊╂⒑闂堟侗鐓┑鈥虫搐閳绘捇濡堕崨顏呮杸濡炪倖娲栧Λ娑氱矈閻戣姤鐓曢柕濞垮妽椤ュ銇勯銏㈢閻撱倖銇勮箛鎾愁仹缂佸崬鐖煎娲川婵犲啫顦╅梺鍛婃惈缁犳垼鐏嬪┑鐐村灟閸ㄦ椽鍩涢幋锔界厱婵炴垶锕╅悡顒佺箾閸喐绀€闁宠鍨块幃娆戞嫚瑜嶆导鎰渻閵堝棙绌跨紒鎻掓健楠炲繘鎮╃憗浣告贡閳ь剨缍嗛崑鎺戭焽閺冣偓缁绘繄鍠婂Ο娲绘綉闂佹悶鍔岄悥鐓庮嚕閵娾晜鐒肩€广儱鎳忛ˉ婵嬫⒑闁偛鑻晶瀛樻叏婵犲嫮甯涢柟宄版嚇閹煎綊鐛惔鎾充壕濠电姴娲﹂悡鏇㈡煙閹屽殶缂佺姷鍋ら弻鐔碱敊閹冨箣婵犵绱曢崗姗€宕洪悙瀛樺劅闁规儳鍘栨竟鏇㈡⒑缁嬭法鐏遍柛瀣仱楠炲棝鎮欓悜妯煎幈闂佹寧妫侀褔鐛鈧弻銊モ槈濞嗘垹鐣虹紓浣虹帛缁嬫帒顭囪箛娑樼鐟滃繗鈪插┑锛勫亼閸婃牕煤濮椻偓閹囧即閵忕姷鍘洪梺瑙勫礃椤曆呯尵瀹ュ鐓冪憸婊堝礈濞戙垹绠查柕蹇曞Л濡插牊鎱ㄥΔ鈧Λ妤呭疾閻樺樊娓婚柕鍫濇閳锋帡鏌涘Ο鐘叉噽娑撳秹鏌ｉ幋鐑嗙劷缂?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌＄仦绋垮⒉鐎垫澘瀚埀顒婄秵娴滄繈顢欓崨顓涙斀闁绘劕寮堕埢鏇灻瑰鍕煉闁挎繄鍋ゅ畷銊р偓娑欘焽閸橆亪姊洪崜鎻掍簼缂佽鍟村畷铏鐎ｎ偆鍙嗗┑鐐村灦閼归箖寮稿☉娆愬弿濠电姴鍟妵婵堚偓瑙勬处閸嬪﹤鐣烽悢纰辨晪闁告侗鍠涢弲褔姊婚崒娆掑厡妞ゎ厼鐗撻弻濠囨晲閸℃瑯娲搁梺鍓插亝濞叉牜绮婚鐐寸厵閺夊牓绠栧顕€鏌涚€ｎ亜顏柡灞剧缁犳稑顫濋鎸庣潖闂備礁鎲＄湁缂侇喗鐟ラ～蹇撁洪鍕炊闂佸憡娲﹂崑鈧柛瀣崌閹稿﹥寰勯崱妯间簷闂備礁鎼ú銊╁磻閹版澘鍑犻柡鍐ㄧ墛閳锋垶銇勯幒鍡椾壕缂備礁顦遍弫濠氬箖閳ユ枼妲堥柕蹇娾偓鏂ュ亾閻㈠憡鐓ユ繝闈涙椤庢霉濠婂懎浠滄い顓″劵椤︽挳鏌℃担鍓茬吋鐎殿喖顭烽崹鎯х暦閸ャ劍鐣烽梺璇插嚱缂嶅棝宕滃☉銏″剨闁割偁鍎查崐鐢告偡濞嗗繐顏紒鈧埀顒勬⒑缂佹澧柕鍫㈩焾閻ｅ嘲煤椤忓嫮鍔撮梺鍛婂姀閺呮盯顢撻幘缁樷拺闁硅偐鍋涢崝锔姐亜閵夛附灏电紒顔芥椤㈡岸鍩€椤掑嫬钃熼柍銉ョ－閺嗗棝鎮楅敐搴″闁伙箑鐗撻幃妤冩喆閸曨剛顦ラ悗瑙勬处閸撴繈鎮橀崘顔解拺闁告稑锕ゆ慨锕€霉濠婂啰鍩ｇ€规洦鍓熷畷濂稿即閻斿弶瀚?WS ingress闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灪閵囧嫰濡搁妷顖濆惈閻庢鍠涢褔鍩ユ径濠庢僵妞ゆ劧绲芥刊浼存⒒娴ｅ憡鍟為柟绋挎閸┾偓妞ゆ巻鍋撻崡閬嶆煕椤愶絿绠ユ繛鎾愁煼閺屾洟宕煎┑鍥舵！缂備讲鍋撻悗锝庘偓銏㈡嚀椤劑宕橀鍕幗闁诲孩顔栭崰娑㈩敋瑜旈、姗€宕楅悡搴ｇ獮婵犵數濮抽懗鍫曟倷婵犲洦鈷掑ù锝呮啞閸熺偞绻涚拠褏鐣电€规洘绮岄埢搴ょ疀婵犲啰鈧椽姊洪幐搴ｇ畵婵炲眰鍔庢竟鏇㈡寠婢规繂缍婇弫鎰板幢濡ゅ啰銈峰┑鐐茬摠缁挾绮婚弽褜娼栭柧蹇撴贡閻瑩鏌熺粙鍨劉闁圭柉椴哥换婵嬪閿濆孩缍堝┑鐐跺皺閸犲酣鎮鹃悜钘夌闁挎洍鍋撶紒鐘差煼閺屻倖鎱ㄩ幇顑藉亾閺囩姷鐭堥柨鏇炲€归埛鎴︽煕閹剧懓鐨洪柛妯荤洴閺屾盯鎮╁畷鍥ь潷闂侀€涚┒閸斿秹骞嗛弮鍫澪╅柨鏃€鍎抽獮妤呮⒑閻熸澘鎮戦柣锝庝邯瀹曠銇愰幒鎴濇優濡炪倖甯掔€氼參鎮￠弴銏＄厽婵☆垱瀵ч悵顏堟煟閵婏箑鐏撮柡灞剧洴楠炴帡骞橀搹顐ョ檨闂備浇妗ㄩ悞锕傚礉濞嗗繒鏆﹂柕濞炬櫅绾惧吋绻涢幋鐐垫噽婵☆偄鍟撮幃妤冩喆閸曨剛锛橀梺鍛婃⒐閸ㄥ潡濡存担绯曟瀻闁瑰墽琛ラ幏濠氭⒑閸濆嫷妲归柛銊ф暬椤㈡俺顦规慨濠傤煼瀹曟帒顫濋钘変壕闁绘垼濮ら崐鍧楁煥閺囩偛鈧摜绮婚弽顓熺厱妞ゆ劧绲剧粈鈧紒鍓у亾鐎笛囧Φ閸曨垰鍐€妞ゆ劦婢€缁墎绱撴担鎻掍壕婵犮垼鍩栭崝鏇犵不瑜版帒绾ч柛顐ｇ箓閳锋棃鏌熼鎯у付闁宠棄顦甸獮妯虹暦閸ュ柌鍥ㄧ厸鐎光偓閳ь剟宕伴幘璇茬劦妞ゆ帊鑳堕埊鏇㈡煥濮橆兘鏀芥い鏃囧Г鐏忥附顨ラ悙瀵稿ⅹ閼挎劖銇勯幒鍡椾壕婵犵鈧偨鍋㈤柡宀嬬秮楠炲洭顢楁繝鍌氼潬闂備胶顢婄亸娆撯€﹂崼銉ョ厴闁瑰濮崑鎾绘晲鎼粹€愁潻閻熸粍婢樺Λ妤呪€旈崘顔嘉ч柛鈩冾殘閻熴劑姊虹粙鍖″姛闁轰浇顕ч悾鐑藉箣閿曗偓缁犲鏌ら幖浣规锭闁哄鍊垮娲川婵犲啫顦╅梺鍛婃尰閻╊垶宕洪悙娴嬫婵浜敍婵囩箾鏉堝墽鍒版繝鈧柆宥呯煑闁逞屽墰缁辨挻鎷呴崫鍕戯綁鏌ｅΔ鍐ㄤ粶閾荤偤鏌ｉ弬鍨倯闁抽攱鍨圭槐鎾存媴閻ч晲绶靛┑鐐茬墛濡啴寮婚敐鍛傛棃鍩€椤掑嫭鏅濇い蹇撳閺嗭箓鏌￠崶銉ョ仾闁稿瀚伴弻娑滅疀濮橆兛姹楅梺鍛婄懃濡鍩為幋锔藉€烽柡澶嬪灩娴犳悂姊洪幐搴ｎ暡闁靛洤瀚粻娑㈡晲閸涱剙鏋堥梻浣告惈閻绱炴笟鈧顐﹀箛閺夊灝绐涘銈嗘尵閸犳劙鎯侀敐澶嬧拻濞达絽鎽滈弸鍐┿亜椤愩埄妯€鐎规洖缍婂畷绋课旈埀顒傜不閺嶎厽鐓冮柛婵嗗婵ジ鏌℃担闈╄含闁哄被鍊曢湁閻庯綆鍋呴悵鎺楁⒑濮瑰洤鍔村ù婊庝簻椤繒绱掑Ο璇差€撴繛鎾村嚬閸ㄦ娊宕濋崨濠佺箚闁靛牆娲ゆ牎闂佽鍠栭崐鎼佹偩閻戣棄唯鐟滃宕戦幘缁樻櫜閹肩补鍓濋悘鍫熶繆濡も偓閹虫ê顫忓ú顏勪紶闁告洦鍋呭▓顓㈡⒑缁嬪尅宸ユい顓犲厴楠炲棛浠︾憴锝嗙€婚梺褰掑亰閸犳岸顢欓弴銏♀拺缂侇垱娲栨晶鏌ユ煟閻旀潙鍔滅紒鍌氱У閵堬綁宕橀埡浣插亾閻㈠憡鈷掗柛顐ゅ枔閵嗘帡鏌ｉ幒鎴炲暈闁逛究鍔嶇换婵嬪川椤曞懍鍝楃紓鍌欑贰閸犳鎮疯楠炴垿宕熼姣尖晝鎲稿畝鈧Σ鎰潨閳ь剙顫忕紒妯诲濞撴凹鍨抽崝绋款渻閵堝棗鐏ユ俊顐ｇ箞楠炲棝宕熼锝嗘櫖闂佺粯鍔樼亸娆撴偩妤ｅ啯鐓熼煫鍥ㄦ礀娴犫晛鈹戦悙鍙夊枠鐎殿噮鍣ｅ畷鐓庘攽閸℃瑧宕哄┑锛勫亼閸婃牕顫忚ぐ鎺戠？闁瑰墎鏅畵浣衡偓骞垮劚椤︿即鍩涢幋锔藉仯闁搞儜鍐獓闂佸湱娅㈢紞渚€寮婚埄鍐╁閻熸瑥瀚埀顒佸姍閺屽秹濡烽婊呮殼閻庤娲栭妶鎼佸箖閵忋倕浼犻柛鏇樺妼瑜板繘姊婚崒姘偓鎼佸磹閹间礁纾瑰瀣捣閻棗銆掑锝呬壕濡ょ姷鍋為悧鐘汇€侀弴銏℃櫇闁逞屽墰缁鎮╅懡銈呭絼闂佹悶鍎崝搴ㄥ煡婢跺瞼纾奸柣妯虹－婢х敻鏌＄仦鍓р槈闁伙絾绻堥崺鈧い鎺戝绾剧粯绻涢幋鐐寸殤濞戞挸绉归弻鈥愁吋鎼粹€茬凹闂佸摜濮甸崝娆撳蓟閿濆憘鏃堝焵椤掑嫭鍋嬮柛鈩冪懅缁犳棃鏌熼悜妯烩拻缁炬儳銈搁弻锝呂熼崫鍕瘣濠电偞鎯岄崳锝夊蓟閳ュ磭鏆嗛柍褜鍓熷畷浼村冀椤撶偟鐣哄┑鐘诧工閸氭﹢鎮㈤崗鍏煎劒闂備緡鍋呯粙鎺懳熸繝鍌楁斀闁挎稑瀚禍濂告煕婵炲灝鈧繂鐣烽幋锕€绀嬫い鏍ㄦ皑閻ゅ洭妫呴銏″婵炲弶锚閻ｇ兘宕ｆ径宀€鐦堥梻鍌氱墛娓氭宕曡箛鏂讳簻妞ゆ劑鍨洪崵鍥煛鐏炶濡奸柍瑙勫灴瀹曞崬顫滈崱姗堥獜闂傚倷绶氬褍螞濞嗘挸绀夋繛鍡樻尭閽冪喖鏌ㄥ┑鍡橆棡闁稿海鍠栭弻鏇＄疀鐎ｎ亞鍔撮梺鍝勬閳ь剚鏋奸弨浠嬫煟濮楀棗鏋涢柣蹇ｄ邯閺屻劑寮村Ο铏逛紙濡ょ姷鍋涢崯顖炲Χ閿濆绀冮柍杞拌閸嬫挻绻濆顓犲幘闂佽鍘界敮鎺楀礉濠婂嫮绠鹃柛娑卞枤閻绱掔紒妯肩疄濠殿喒鍋撻梺鎸庣箓濡盯濡撮幇顒夋富闁靛牆妫楅悘銉︾箾瀹割喖骞栭柣锝囧厴閺佹劖寰勬繝鍌濃偓鍨攽閻愭潙鐏﹂柣鐔濆洤鍌ㄥù鐘差儐閻撴瑩鏌涜箛鎾虫倯闁稿孩顨婇弻娑㈠Ω閵壯嶇礊缂備緡鍠楀Λ鍐箖娴犲宸濆┑鐘插楠炲牓姊绘担绛嬫綈闁稿孩濞婇、姘额敇閻忕粯妞介獮姗€顢欓悾灞藉箰濠电偠鎻徊浠嬪箺濠婂啰鎽ラ梻鍌欑閻ゅ洭锝炴径鎰瀭闁割煈鍠氶弳锕€鈹戦崒婊庣劸妞ゎ偄鎳忛妵鍕敃椤愩垺鐏撻梻鍌氬亞閸ㄨ京鎹㈠┑瀣仺闂傚牊绋愰崫妤呮⒑鐟欏嫭銇熷ù婊呭仜椤曘儲绻濋崶銊ユ疅闂侀潧锛忛崨顓㈢崕婵犵數濮烽弫鍛婃叏鐎涙ê顕遍柨婵嗩槹閺呮煡鏌涢锝団棩婵顨婂娲捶椤撶偛濡洪梺瑙勬倐缁犳牕鐣烽姀銈嗗癄濠㈣埖蓱鐎靛矂姊洪棃娑氬闁硅櫕鍔楃划缁樺鐎涙ê鈧灚鎱ㄥΟ鐓庡付闁哄鍊楃槐鎺楊敊閼恒儳鍙嗛柣鎾卞€栭妵鍕疀閹炬潙娅ч梺鍛婏耿娴滆泛顫忛搹鍏夊亾閸︻厼顎屾繛鍏煎姍閺屾稒鎯旈妶鍡欏涧缂?
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
// SelectAccountForModelWithExclusions 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯鐟滃宕戦幘娲绘晢闁告洦鍓涢崢閬嶆⒑闂堟侗妯堥柛鐘崇墬閺呭爼顢欓崜褏锛滈梺缁橈供閸犳牠宕濆鍫熺厪闁搞儜鍐句純閻庢鍣崳锝呯暦閹烘垟妲堟俊顖濆吹閺嗘碍绻濋悽闈涗粶闁宦板妿閸掓帗鎯旈妸銉э紱闂佽宕橀褏绮婚悙鐑樼厪濠电偛鐏濋崜濠氭煟閺冨洦顏犻柣顓熺懇閺屾盯鈥﹂幋婵囩亪濡炪値鍋呴幐鎼佸煘閹达附鍋愰柟棰佺閺呴亶姊洪崫銉バｉ柣妤冨Т閻ｇ兘骞囬鐘电槇濠殿喗锕╅崜娑㈠礉閿曗偓椤啴濡堕崱妤冪憪缂備浇顔愮换婵嗙暦椤栨繄鐤€婵炴垶鐟ч崢閬嶆⒑缂佹﹩鐒炬繛鍛礀閻ｅ嘲鐣濋埀顒勫焵椤掑喚娼愭繛鍙夘焽閹广垽宕奸妷褍绁﹂柣搴秵閸犳寮插鍫熷仩婵炴垶宸婚崑鎾诲礂閸涱収妫滄繝鐢靛仩閹活亞寰婄捄銊﹀厹閻犺桨缍嶉敐澶婇唶闁绘柨澧庣粻姘渻閵堝棛澧遍柛瀣仱瀵偅娼忛妸锝勭盎濡炪倖鍔戦崹璇茬摥闂備礁鐤囬～澶愬垂閸ф鏄ラ柕澶嗘櫅楠炪垺淇婇妶鍛殨闁告繃顨婂缁樻媴閸涘﹤鏆堥梺鐓庣秺缁犳牠銆侀弽顓炲窛闁哄鍨舵潏鍫ユ⒑閸愬弶鎯堥柟鍐茬箻閸╂盯骞掑Δ浣哄幈闁诲繒鍋炲畷妯荤珶濮椻偓閺屽秷顧侀柛鎾跺枛钘熼柟鎹愭硾閸ㄦ繈鏌涢銈呮瀾濠殿垱鎸冲濠氬醇濮橆厽鐝楅梺閫炲苯澧繛灏栤偓鎰佹綎婵炲樊浜堕弫鍡涙煃瑜滈崜娑氬垝閺冨牜鏁嬮柍褜鍓熷畷娲焵椤掍降浜滈柟鐑樺煀閸旂喖鏌ｉ敐鍥ㄦ毄闁逞屽墲椤煤濠婂牆鍌ㄧ憸鏃堝灳閿曞倸鍨傛い鏃囶潐閺傗偓闂備焦鎮堕崕顖炲礉婵犲啰顩烽柕蹇ョ磿缁♀偓閻庣數澧楅悘姘跺磹閺囥垺鍋傞柟閭﹀枓閸嬫捇宕归锝囧嚒闁诲孩鍑归崳锝夊春閳ь剚銇勯幒鎴姛缂佸娼ч湁婵犲﹤瀚幊鍥殽閻愭彃鏆ｉ柡浣瑰姈瀵板嫮鈧急鍕伜婵犵數鍋犻幓顏嗗緤閸ф鍋ら柡鍐ㄧ墕閸氬綊鏌涢幇闈涙灍闁绘挻娲熼弻鏇熺箾閸喖濮岄梺绋款儍閸ㄤ粙寮婚悢纰辨晩闁活収鍋掓禍顏堝春閵夛箑绶為柟閭﹀墻濞煎﹪姊洪悙钘夊姎闁告ɑ鐗滈懞杈ㄧ節濮橆厸鎷绘繛杈剧秬濞咃絿鏁☉銏＄厵闁告縿鍎洪悞楣冩婢舵劖鈷戦柛顭戝櫘閸庡繑銇勯弴顫喚闁诡喗顨婇弫鎰償閳╁啰浜堕梻浣侯焾閿曪箓寮繝姘畺鐎瑰嫭澹嬮弸搴ㄧ叓閸ャ劍鎯勫ù鐘插⒔缁辨挻鎷呴幓鎺嶅闂備線鈧偛鑻晶顕€鏌嶇憴鍕伌闁诡喗鐟ч幑鍕惞鐟欏嫷鍤勬繝鐢靛Х閺佹悂宕戦幇鏉跨；闁瑰墽绮崐鐢告偡濞嗗繐顏紒鈧崘顔界厱闁靛鍎查崑銉╂煟濞戝崬鏋涢摶锝夋煠濞村娅囬柨娑欑矊閳规垿鍩ラ崱妤冧淮闂侀潧妫岄崑鎾绘⒑缁嬫鍎愰柟姝屽吹閹广垹鈹戦崱鈺傚兊濡炪倖鎸炬慨鎾嵁濮椻偓濮婄粯鎷呯粵瀣異闂佸憡顭囬弲顐﹀极椤曗偓楠炴帒螖閳ь剛绮堟径瀣弿婵☆垰鐏濋悡鎰版煟閹惧瓨绀嬮柡宀€鍠栧畷褰掝敊閸忓吋顔勯柣搴㈩問閸ｎ噣宕板Δ鍛疄闁靛濡囩弧鈧梺鍛婂姀閺傚倹绂掗姀銈嗗€甸悷娆忓绾炬悂鏌涢弬璺ㄐら柟骞垮灩閳规垹鈧綆浜為ˇ銊ヮ渻閵堝骸骞楅柛銊﹀劶閳悂姊婚崒姘偓鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞顨呴…銊╁礋椤掆偓瀵潡姊哄Ч鍥х仼闁硅绻濆畷鎴犫偓锝庡枟閻撶喐淇婇婊冨付闁告柨顑嗛妵鍕Ψ閿濆應鍋撳┑鍡╂綎缂備焦蓱婵绱掔€ｎ偄顕滄慨锝嗗姍濮婃椽宕烽娑欏珱闂佺顑呴敃顏堟偘椤旂晫绡€闁告侗鍨抽弶绋库攽閻愭潙鐏﹂柣鐔村劜缁傚秴螖閸涱喒鎷洪梺鍛婄箓鐎氼厼顔忓┑瀣厱闁绘娅曠亸鐢电磼椤斿墽甯涢柕鍫秮瀹曟﹢鍩￠崘銊ョ疄闂傚倷鐒﹂弸濂稿疾濞戙垹绐楁慨姗嗗厴閺嬫梹淇婇婵嗕汗缁炬儳銈搁弻锝咁潨閳ь剙顭囪閵嗗倿鎳栭埡鍐紲缂傚倷鐒﹂…鍥虹€电硶鍋撶憴鍕闁告鍥风稏婵犻潧顑愰弫鍥煟閺傛寧鎯堢紒鑼舵硶缁辨捇宕掑顑藉亾瀹勬噴褰掑炊閵婏絼绮撻梺鍛婄缚閸庢煡寮冲鍕箚闁靛牆鍊告禍鎯ь渻閵堝骸骞栨繛娴嬫櫇缁鈽夊Ο閿嬵潔濠碘槅鍨辨竟鏇㈠疾濠靛绠熸慨婵嗙灱閻も偓濠电偞鍨堕悷銉︾妤ｅ啯鈷戦柛娑橈攻婢跺嫰鏌涢妸鈺€鎲剧€规洘鍔欓幃浠嬪川婵犲倷鐢绘繝鐢靛仜濡鎹㈤幇閭︽晜妞ゅ繐妫涘Λ顖炴煟閹伴潧澧柛濠冨姍閺岋紕浠﹂崜褉濮囬梺璇″灡濡啯淇婇崼鏇炲耿婵炲棗绻戦柨顓犵磽閸屾艾鈧绮堟笟鈧畷鎰板捶椤撶喐娈伴梺璺ㄥ枔婵挳寮告笟鈧弻娑㈠箛闂堟稒鐏堢紒鐐劤閸氬鎹㈠☉銏犵闁绘劕鍟畝绋跨暦閹达箑绠荤紓浣诡焽閸欏棝鏌ｆ惔顖滅シ闁告柨顑囬懞杈ㄧ節濮橆厸鎷洪梺闈╁瘜閸樺ジ宕濈€ｎ喗鐓曢柍杞拌兌婢ф洟宕￠柆宥嗙厱闁挎棁顕ч獮妤呮煕鐎ｎ偆澧遍柟鍙夋倐瀵剙鈻庨悙顒傜▉濠电姷鏁告慨鐢告嚌閸撗冾棜闁稿繗鍋愮粻楣冩煕閳╁厾顏堟倿閻愵兛绻嗙€瑰壊鍠栭弸娑㈡煃鐟欏嫬鐏撮柟顔规櫊椤㈡洟锝為鐑嗘闂傚倸鍊搁崐椋庢閿熺姴绀堟繛鍡樺灩閻捇鎮楅悽娈跨劸妞ゃ儲宀搁弻娑滎槼妞ゃ劌妫濋幃陇绠涘☉娆戝幈濡炪倖鍔х徊璺ㄧ不閺嵮岀唵閻犲搫鎼顓㈡煛瀹€瀣瘈鐎规洖宕灒闁兼祴鏅濋崢婊堟⒒娴ｅ憡鎯堥柤娲诲灣缁梻鎲撮崟顏嗙畾闂佺粯鍨兼慨銈夊疾濠婂牊鐓熼柨婵嗘噹椤ㄦ瑧绱掓潏顭掕€挎慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵＄偑鍊栭崹鐢稿箠閹邦喖鍨濋柡鍐ㄦ储娴滃綊鏌熼悜妯荤叆缂佷緤绠撳娲传閸曨厸鏋嗛梺璇茬箲濮婂綊骞嗛弮鍫氣偓锕傚箣閻愬瓨鐝滈梻鍌欑窔閳ь剛鍋涢懟顖涙櫠椤栨搴ㄥ炊瑜濋煬顒勬煙閾忣偆澧甸柛鈺嬬節瀹曟﹢顢旀惔锝呯哎闂傚倸鍊烽懗鍓佸垝椤栫偞鏅柣搴ゎ潐濞诧箓宕滃▎鎾村仼闁汇垻顭堥獮銏＄箾閹寸偟鎳呴柛妯绘倐閺岋綀绠涢弴鐐扮捕婵犫拃鍡橆棄閻撱倝鏌℃径瀣亶闁衡偓娴犲鐓ユ繛鎴灻鈺呮煕濡櫣鎽犳い銊ｅ劦閹瑥顔忛鐓庡闂備礁鐤囬～澶愬垂閸喚鏆﹂柟顖炲亰濡茬厧顪冮妶鍛畾闁告挻绻勯幑銏犫攽閸″繑鐏侀梺鍓茬厛閸犳鎮甸婊呯＝濞达綀妫勭槐锔剧磼椤旂晫鎳囩€殿喚绮换婵嬪炊閵婏附鐝抽梻浣告啞缁牓鎮為敂鍓х閹兼番鍔嶉埛鎴︽煕濞戞﹫鍔熺紒鐘虫崌閹顫濋浣告畬濡炪倧绠戝﹢閬嶅焵椤掑喚娼愭繛鍙夛耿瀹曟洘绺介弶鍡楁喘瀹曪絾寰勯崼婊呯泿婵＄偑鍊栧濠氬磻閹剧粯鐓熸俊銈勭劍瀹曞瞼鈧鍠栭…鐑藉极閹版澘骞㈡俊顖氭惈婵℃娊姊绘笟鈧褏鎹㈤幒鎾村弿闁哄鍤╅幒妤€閱囬柕澶涜吂閹锋椽姊虹涵鍛汗闁稿绋掓穱濠囨嚃閳规儳浜鹃悷娆忓婢跺嫰鏌涢幘璺烘灈闁靛棔绀侀～婵堟崉娴ｆ洏鍔戦弻宥嗘姜閹峰苯鍘″銈忛檮濠㈡﹢鈥旈崘顔嘉ч柛鈩冪懃椤冣攽椤旇婊堝礉閹达絾顥ら梻浣告惈椤︿即顢栧▎蹇嬧偓鎺撶節濮橆厾鍘梺鍓插亝缁诲啴藟閻樺樊鐔嗙憸婊堝礈濮樿泛绠為柕濞垮剻閻旇櫣鐭欓柛顭戝櫘濞煎酣姊绘担绛嬪殐闁革綆鍨抽幑銏犫攽鐎ｎ剙绁﹂柣搴秵閸犳寮插鍫熷仭婵炲棙鐟х粙濠氭倵濮橆兙鍋㈡慨濠勭帛閹峰懘宕ㄦ繝鍌涙畼缂傚倷娴囬褔宕愭繝姘劦妞ゆ帊鐒﹂惃鎴︽煕韫囨枂顏堟偩閻戣棄鍗抽柣鎰ㄦ櫆椤秹姊洪棃娑㈢崪缂佽鲸娲熷畷銏ゆ焼瀹ュ棛鍘介柟鍏兼儗閸ㄩ亶鍩€椤戞儳鍔︾€规洘顭堥ˇ鍙夈亜閵夛箑鍝烘慨?
func (s *OpenAIGatewayService) SelectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*Account, error) {
	return s.selectAccountForModelWithExclusions(s.withOpenAIQuotaAutoPauseContext(ctx), groupID, sessionHash, requestedModel, excludedIDs, false, 0, PlatformOpenAI, "")
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

func isOpenAIAccountEligibleForRequest(ctx context.Context, account *Account, requestedModel string, requireCompact bool, platform string, requiredCapability OpenAIEndpointCapability) bool {
	if account == nil || !accountMatchesPlatform(account, platform) || !account.IsSchedulableForModelWithContext(ctx, requestedModel) {
		return false
	}
	if paused, reason := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
		// Debug level: this fires per-candidate on the scheduling hot path, so Info
		// would amplify into log spam once several accounts cross the threshold.
		slog.Debug("account_auto_paused_by_quota",
			"account_id", account.ID,
			"window", reason.window,
			"threshold", reason.threshold,
			"utilization", reason.utilization,
		)
		return false
	}
	if requestedModel != "" && !account.IsModelSupported(requestedModel) {
		return false
	}
	if !account.SupportsOpenAIEndpointCapability(requiredCapability) {
		return false
	}
	if requireCompact && openAICompactSupportTier(account) == 0 {
		return false
	}
	return true
}

type openAIQuotaAutoPauseDecision struct {
	window      string
	threshold   float64
	utilization float64
}

func shouldAutoPauseOpenAIAccountByQuota(ctx context.Context, account *Account) (bool, openAIQuotaAutoPauseDecision) {
	if account == nil || !account.IsOpenAI() {
		return false, openAIQuotaAutoPauseDecision{}
	}
	// Per-account explicit-disable flags must take precedence over the global default.
	// Without these, leaving the account threshold blank means "use global default",
	// so an admin has no way to exempt a single account from auto-pause once a global
	// default exists. The disable flag is per-window so an account can opt out of
	// only 5h or only 7d auto-pause.
	disabled5h := resolveAccountExtraBool(account.Extra, "auto_pause_5h_disabled")
	disabled7d := resolveAccountExtraBool(account.Extra, "auto_pause_7d_disabled")
	threshold5h, threshold7d := resolveOpenAIQuotaAutoPauseThresholds(ctx, account)
	now := time.Now()
	if !disabled5h && threshold5h > 0 {
		if utilization, ok := resolveOpenAIQuotaUtilization(account.Extra, "5h", now); ok && utilization >= threshold5h {
			return true, openAIQuotaAutoPauseDecision{window: "5h", threshold: threshold5h, utilization: utilization}
		}
	}
	if !disabled7d && threshold7d > 0 {
		if utilization, ok := resolveOpenAIQuotaUtilization(account.Extra, "7d", now); ok && utilization >= threshold7d {
			return true, openAIQuotaAutoPauseDecision{window: "7d", threshold: threshold7d, utilization: utilization}
		}
	}
	return false, openAIQuotaAutoPauseDecision{}
}

// resolveAccountExtraBool reads a bool-like value from account extra, tolerating
// the few shapes JSON unmarshalling may produce (real bool, "true"/"false"
// strings, 0/1 numbers).
func resolveAccountExtraBool(extra map[string]any, key string) bool {
	if len(extra) == 0 {
		return false
	}
	value, ok := extra[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		return err == nil && parsed
	case float64:
		return v != 0
	case float32:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i != 0
		}
	}
	return false
}

func resolveOpenAIQuotaAutoPauseThresholds(ctx context.Context, account *Account) (float64, float64) {
	threshold5h, _ := resolveAccountExtraNumber(account.Extra, "auto_pause_5h_threshold")
	threshold7d, _ := resolveAccountExtraNumber(account.Extra, "auto_pause_7d_threshold")
	threshold5h = clamp01(threshold5h)
	threshold7d = clamp01(threshold7d)
	if threshold5h > 0 && threshold7d > 0 {
		return threshold5h, threshold7d
	}
	settings := openAIQuotaAutoPauseSettingsFromContext(ctx)
	if threshold5h <= 0 {
		threshold5h = clamp01(settings.DefaultThreshold5h)
	}
	if threshold7d <= 0 {
		threshold7d = clamp01(settings.DefaultThreshold7d)
	}
	return threshold5h, threshold7d
}

func resolveAccountExtraNumber(extra map[string]any, keys ...string) (float64, bool) {
	if len(extra) == 0 {
		return 0, false
	}
	for _, key := range keys {
		value, ok := extra[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		case json.Number:
			parsed, err := v.Float64()
			if err == nil {
				return parsed, true
			}
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

// resolveOpenAIQuotaUtilization returns the current utilization ratio (0..1) for the
// given Codex usage window. ok=false means there is no usable signal to pause on:
// either no snapshot exists, or the window has already rolled over so the cached
// percentage is stale. The stale guard matters because a paused account stops
// receiving requests, so its snapshot is never refreshed from upstream headers 闂?// without this check an old used_percent would keep the account paused forever even
// after the real window reset.
func resolveOpenAIQuotaUtilization(extra map[string]any, window string, now time.Time) (float64, bool) {
	usedPercent := readOpenAIQuotaUsedPercent(extra, window)
	if usedPercent <= 0 {
		return 0, false
	}
	if openAIQuotaWindowReset(extra, window, now) {
		return 0, false
	}
	return usedPercent / 100, true
}

// openAIQuotaWindowReset reports whether the Codex usage window's reset time has
// already passed relative to now. It prefers the absolute codex_<window>_reset_at
// timestamp and falls back to codex_<window>_reset_after_seconds anchored at
// codex_usage_updated_at, mirroring AccountUsageService's window-progress logic.
func openAIQuotaWindowReset(extra map[string]any, window string, now time.Time) bool {
	if len(extra) == 0 {
		return false
	}
	if resetAtRaw, ok := extra["codex_"+window+"_reset_at"]; ok {
		if resetAt, err := parseTime(fmt.Sprint(resetAtRaw)); err == nil {
			return !now.Before(resetAt)
		}
	}
	resetAfter := parseExtraInt(extra["codex_"+window+"_reset_after_seconds"])
	if resetAfter <= 0 {
		return false
	}
	base := now
	if updatedRaw, ok := extra["codex_usage_updated_at"]; ok {
		if updatedAt, err := parseTime(fmt.Sprint(updatedRaw)); err == nil {
			base = updatedAt
		}
	}
	resetAt := base.Add(time.Duration(resetAfter) * time.Second)
	return !now.Before(resetAt)
}

func readOpenAIQuotaUsedPercent(extra map[string]any, window string) float64 {
	if len(extra) == 0 {
		return 0
	}
	if value, ok := resolveAccountExtraNumber(extra, "codex_"+window+"_used_percent"); ok {
		return value
	}
	return 0
}

type openAIQuotaAutoPauseCtxKey struct{}

func withOpenAIQuotaAutoPauseSettings(ctx context.Context, settings OpsOpenAIAccountQuotaAutoPauseSettings) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAIQuotaAutoPauseCtxKey{}, settings)
}

func openAIQuotaAutoPauseSettingsFromContext(ctx context.Context) OpsOpenAIAccountQuotaAutoPauseSettings {
	if ctx == nil {
		return OpsOpenAIAccountQuotaAutoPauseSettings{}
	}
	settings, _ := ctx.Value(openAIQuotaAutoPauseCtxKey{}).(OpsOpenAIAccountQuotaAutoPauseSettings)
	return settings
}

func (s *OpenAIGatewayService) withOpenAIQuotaAutoPauseContext(ctx context.Context) context.Context {
	if s == nil || s.settingService == nil {
		return ctx
	}
	return withOpenAIQuotaAutoPauseSettings(ctx, s.settingService.GetOpenAIQuotaAutoPauseSettings(ctx))
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

func (s *OpenAIGatewayService) selectAccountForModelWithExclusions(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, stickyAccountID int64, platform string, requiredCapability OpenAIEndpointCapability) (*Account, error) {
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

	// Try sticky session hit
	if account := s.tryStickySessionHit(ctx, groupID, sessionHash, requestedModel, excludedIDs, requireCompact, stickyAccountID, platform, requiredCapability); account != nil {
		return account, nil
	}

	// 2. 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氶梺璇叉唉椤煤韫囨稑纾块柟鎯版閻掑灚銇勯幒鎴姛缂佸鏁婚弻娑㈡偐瀹曞洤鈷堟繝銏ｎ潐濞茬喎鐣风粙璇炬梹鎷呴崫鍕闂傚倷娴囬鏍垂閾忣偅娅犳俊銈呮儰婢跺绶為柟閭﹀墰椤旀劙姊洪崫鍕垫Ч闁糕晛锕悡顒勵敆閸曨剛鍘搁柣蹇曞仩椤曆囧焵椤掍胶绠炴鐐诧工閳规垹鈧綆浜為鐓庮渻閵堝棙顥嗛柛瀣姈閺呭爼顢涘锝嗘杸闂佺粯鍔樼亸娆撳箺閻樼數纾兼い鏃囧亹閻忚京绱掓潏鈺佷沪缂佺粯绻堝畷鎯邦樁闁硅姤娲栭埞鎴︽倷閺夋垹浠ч梺鎼炲妿閹虫捇寮鎴掔箚闁绘劦浜滈埀顒佺墪椤繑绻濆顒€鍋嶉悷婊勬閺佹劙鎮欓崫鍕幐闂佸憡渚楅崰鎺楀箯閸濆嫧鏀介柣鎰硾閽勫吋銇勯弴鍡楁川瀹撲焦淇婇妶鍕濞存粍绮撻弻鏇熺箾閸喖濮曞銈冨劘閸ㄥ綊鍩為幋锔芥櫖闁告洦鍓氬В鍫ユ倵鐟欏嫭绀堥柛鐘崇墵閵嗕礁鈽夊鍡樺兊濡炪倖宸婚崑鎾绘煛娴ｉ潧鈧繂顫忕紒妯肩懝闁逞屽墮椤洩顦堕柛锝呯秺濮婃椽宕ㄦ繝浣虹箒闂佹悶鍔屽畷顒冾暰婵犮垼娉涜墝闁衡偓娴犲鐓熸俊顖濇娴犳盯鏌￠崱蹇旀珚闁哄本绋撻埀顒婄秵閸嬪棗煤閹绢喗鐓欐い鏂诲妼濞层倝鐛姀锛勭闁瑰鍋熼幊鍐瑰搴＄仭缂佺粯绻堝Λ鍐ㄢ槈濡嘲浜鹃柟闂寸缁犵儤绻濇繝鍌氭殶闁哄棎鍊曢妴鎺戭潩閿濆懍澹曢梻浣告惈婢跺洭鍩€椤掍礁澧柛姘儔閺屾盯骞囬埡浣割瀳濡炪値鍓欓ˇ杈╂閹捐纾兼慨妯荤樂閵忥紕绠剧€瑰壊鍠栧顔锯偓娈垮枛椤兘骞冮姀銏犳瀳閺夊牄鍔嶅▍鏍⒒娴ｈ櫣銆婇柛鎾寸箞閵嗗啴宕卞Δ瀣偓鍨归悡搴ｆ憼闁绘挻娲熼弻锝呂熺拠鑼海闂佺懓顕慨闈涚暦?OpenAI 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌″畝瀣М濠殿喒鍋撻梺闈涚箚閺呮繈宕濋崫銉х＝濞达絾褰冩禍鐐節閵忥絾纭鹃柡鍫墴瀹曚即骞囬幍顔煎絼闂佹悶鍎崝宥囩矆閳ь剟姊洪幖鐐插缂佽鍊块崺鐐哄箣閿旇棄浜归柣搴℃贡婵挳藟濠靛牏纾藉ù锝呮惈椤庢挾绱撳鍕獢鐎殿喖顭锋俊鎼佸Ψ閵忊剝鏉搁梻浣虹《閸撴繈銆冮崱妞绘灁闁靛鍎弨?	// Get schedulable OpenAI accounts
	accounts, err := s.listSchedulableAccounts(ctx, groupID, platform)
	if err != nil {
		return nil, fmt.Errorf("query accounts failed: %w", err)
	}

	// Select by priority + LRU
	selected, compactBlocked := s.selectBestAccount(ctx, groupID, accounts, requestedModel, excludedIDs, requireCompact, platform, requiredCapability)

	if selected == nil {
		return nil, noAvailableOpenAISelectionError(requestedModel, compactBlocked)
	}

	hydrated, err := s.hydrateSelectedAccount(ctx, selected)
	if err != nil {
		return nil, err
	}

	// Set sticky session binding
	if sessionHash != "" {
		_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, selected.ID, openaiStickySessionTTL)
	}

	return hydrated, nil
}

// tryStickySessionHit 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻缁橆殽閻愭潙鐏村┑顔瑰亾闂侀潧鐗嗛幊鎰版偪閳ь剚淇婇悙顏勨偓鏍涙担鑲濇盯宕熼浣稿妳婵犵數濮村ú锕傚煕閹达附鐓熼柣鏃傚帶娴滀即鏌涢妶鍥ф瀻闁宠鍨块、姘跺焵椤掆偓椤洩顦归柟顕€绠栭幃婊堟寠婢跺孩鎲伴梻渚€娼ч¨鈧┑鈥虫喘瀹曘垽鏌嗗鍡忔嫼闂佸搫鍊堕崕鎻掆枍閸涘瓨鐓曢柣鏃囨硾瀹撳棝鏌涢埡渚婅含鐎殿喗鎸虫慨鈧柨娑樺楠炲牓姊虹涵鍛汗閻炴稏鍎卞嵄闁告洦鍨扮紒鈺佲攽閻樺磭顣查柣鎾崇箻閺屾盯濡烽幋婵嗘灓濞寸厧鍟撮幃妤€鈻撻崹顔界亞缂備緡鍠楅悷锔界┍婵犲偆娼扮€光偓婵犲唭褔姊绘笟鈧埀顒傚仜閼活垱鏅堕鐐村仺妞ゆ牗渚楀▓婊冣攽閿涘嫭鏆€规洜鍠栭、娑橆潩閸楃偛绠伴梻鍌欑閹诧繝宕濋幋锕€绀夌€光偓閸曨厼绁﹂柣搴秵閳ь剦鍘藉Λ鍐箖閳哄懏顥堟繛鎴烆殕閸曞啴姊绘担铏瑰笡妞ゃ劌妫濋妴鍐川缁厜鍋撻敃鍌氱倞妞ゅ繐妫涢弶绋库攽閻愭潙鐏︽い顓炴喘钘濇い鎾卞灪閸嬧剝绻濇繝鍌氭殻闁告瑥瀚埀顒冾潐濞叉﹢宕规總鏉嗗洨鎲撮崟顒€寮挎繝鐢靛Т閹冲繘顢旈悩鐢电＜閺夊牄鍔岀粭鎺楁懚閿濆鍋ｉ柛銉ｅ妿閸欌偓闂佽瀛╁钘夘潖閸濆嫅褔宕惰閸旀挳姊洪幖鐐插闁哄拋鍋婇獮鎴﹀閻橆偅鏂€闂佺硶妾ч弲娑樷枔閵婏妇绡€闁汇垽娼ф牎缂佺偓婢樼粔鐟扮暦閹达箑绀嬫い鎾跺У閿涘繘姊洪悷鏉挎Щ闁活厼鍊垮畷顐⒚洪鍛幐闂佸憡渚楅崹鐗堢濠婂嫨浜滄い蹇撳閺嗭絽鈹戦垾宕囧煟鐎规洖宕灃闁逞屽墮椤洭骞嬮敂瑙ｆ嫼缂備礁顑嗛娆撳磿閹扮増鐓欓柣鐔哄閸犳鈧鍠涢褔鍩ユ径濞炬瀻闁瑰濮峰Σ鍥⒒娴ｅ憡鍟炲〒姘殜瀹曘垺绺界粙璺ㄧ暰闂佺粯鍔楅弫鍝ュ閽樺鈧帒顫濋浣规倷濠电偛鎳忛悷褏妲愰幒妤€绠甸柟鍝勬娴滈箖鏌ｉ弮鍥ㄣ€冪紒銊у厴濮婂宕掑▎鎴М闂佽绁撮崜婵堢箔閻旇偤鏃堝川椤撶姷宕堕梻浣告惈缁夋煡宕濇惔銊ｂ偓鍛存煥鐎ｃ劋绨婚梺鍝勭▉閸嬪嫭绂掗敃鍌涚厓闂佸灝顑呴悘鎾煛瀹€鈧崰鏍箠閺嶎厼鐓涘ù锝夘棑閹规洟姊绘担鍛婂暈婵﹤缍婇弫鍐閵堝懓鎽曞┑鐐村灟閸ㄧ懓螞濮椻偓閺岀喓绮欓崹顕呭妷閻庤娲栫壕顓犳閹惧鐟归柛銉戝嫮浠屾俊鐐€栧▔锕傚炊妞嬪海鈼ゆ繝鐢靛█濞佳囶敄閹版澘鏋侀柛銉㈡櫆閸犳劙骞栭幖顓犲帥闁轰礁锕弻锟犲礃閵娧冾杸闂佹娊鏀遍崹鍦閹惧瓨濯村┑顔藉焾娴滄繈骞堥妸銉ф殕闁告洦鍓欓埀顒€鐖奸悡顐﹀炊閵婏腹鎷婚梺鐟板暱閹虫劗妲愰幒妤婃晩缁炬媽浜崥瀣⒑閸︻収鍔滅紒缁樼箖娣囧﹪宕奸弴鐐碉紲濠殿喗锕╅崑鍕夊鑸碘拺闁煎鍊曢弸鎴︽煟閻旀潙鍔ら柍褜鍓氶崙褰掑储閸撗冨灊婵椴稿畷澶愭煏婵炑冩嫅缁辨娊鏌ｆ惔锛勭暛闁稿氦浜埀顒佸嚬閸ｏ綀妫熼梺鎸庢煥椤洘绂嶅鍫熺厸闁搞儲婀圭花鐣岀磼鐎ｎ厼鍚圭紒杈ㄥ浮瀹曟帒鈽夊Ο纰卞剬闂備線鈧偛鑻晶鍓х磼闊厾鐭欓柟顔矫～婵堟崉閾忚鐓ｉ梻渚€娼х换鍫ュ磹閺嶎厼鐓曢柟杈鹃檮閳锋垶銇勯幇鈺佲偓鏇熺濠婂牊鐓犳繛鑼额嚙閻忥繝鏌￠崨顓犲煟妤犵偞锕㈤、娆撴寠婢跺本顎嶅┑鐘垫暩婵炩偓婵炰匠鍥舵晞闁糕剝绋掗崑鍌炴煕閹伴潧鏋熼柣鎾存礃缁绘繈妫冨☉娆樻濡炪倕娴氭禍顏堝蓟濞戙垺鏅查柛鈩兦滄禒銏ゆ⒑閸濆嫮鐒跨紓宥勭窔閻涱喖螣鐏忔牕浜炬繛鎴炵懐閻掕姤銇勯敂鍝勫缂佽鲸鎸婚幏鍛存惞閻熸壆顐奸梻浣哄劦閺呪晠宕规导瀛樺仼闁绘垼妫勭涵鈧梺缁樺姇缁夊爼宕伴弽褏鏆︽い鎰剁畱缁€瀣煕椤垵浜炵紒澶婄仢閳规垿鎮╅崹顐ｆ瘎闂佺顑嗛惄顖炪€佸棰濇晣闁绘梻顭堝鍧楁⒑濮瑰洤鐏╅柟璇х節瀹曟洘鎯旈～顑跨盎闂佸搫绋侀崑鍕濠婂牊鐓?// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘﹢寮婚敐澶婄婵犲灚鍔栫紞妤呮⒑闁偛鑻晶顕€鏌涙繝鍌涜础缂侇喖顑夐獮鎺楀棘閸濆嫪澹曢梺鎸庣箓缁ㄨ偐鑺辨禒瀣厱闁哄啯鎸鹃悾杈ㄣ亜椤忓嫬鏆ｅ┑鈥崇埣瀹曞崬螖閳ь剙顭囬幋锔解拺缂佸顑欓崕鎰版煙閻熺増鍠樼€殿喛顕ч埥澶愬閳ュ厖绨婚梻鍌欑閻忔繈顢栭崨顔绢浄闁圭虎鍠楅埛鎴犵磼椤栨稒绀冮柡澶婄秺閺屾稓鈧綆鍋呯亸顓熴亜椤忓嫬鏆ｅ┑鈥崇埣瀹曞崬螖閳ь剙顭囬幋锔解拺缂佸顑欓崕鎰版煙缁嬪灝鈷旀俊鍙夊姍楠炴﹢骞囨担鍛婂€梻浣告啞缁矂宕幎钘夎Е妞ゆ劏鎳￠弮鍫熷亹闂傚牊绋愮划鍫曟⒑閸濄儱娅忛柛瀣樀閹﹢骞掑Δ浣哄幗闂佺粯锚瀵墎绮氶崸妤佸€堕煫鍥ㄦ⒒閹冲懐绱掗鍡欑М闁诡喗鐟╅幃婊兾熼柨瀣伖闂佽崵鍠愮划搴㈡櫠濡ゅ啯鏆滈柟鐑樻尵椤╂彃霉閻撳海鎽犻柣鎾存礋閺岀喖骞嗚閸ょ喖鏌熼崘鍙夊櫧缂佽鲸甯￠、姘跺幢濞嗗繐鐓傞梻浣哥枃濡嫰藝閸偅鍙忛柍褜鍓熼弻宥夋煥椤栨矮澹曢梻浣哥秺閺€鍗烆渻閽樺娼栨繛宸簼閸ゆ帡鏌曢崼婵囧櫤闁诲海鍋撶换婵嬪閵忊€虫畬濠碘槅鍋呯换鍫ユ偘椤斿槈鏃堝川椤旈棿姹楃紓鍌氬€烽悞锕傛晪婵炲濮撮幖顐︹€旈崘顔嘉ч柛鈩冪懃椤呯磽娓氬洤鏋涢柣顓炲€搁锝囩矙鎼存挻顫嶅┑顔斤公缁茶姤绂嶆ィ鍐╁仭婵炲棗绻愰顏嗙棯閻愵剚鍊愰柡灞剧洴婵℃悂濡堕崶鈺冨幆闂備線娼уΛ鏃傜矆娴ｇ晫浜欓梻浣告啞娓氭宕㈡ィ鍐ㄧ闁挎洍鍋撴い顏勫暣婵″爼宕卞Δ鍐噯闂佽瀛╅崙褰掑礈閻旂厧绠栭柨鐔哄Т閻忔娊鏌熸０浣藉厡闁哥偟鏁诲娲礂閸忕厧鈧劖绻涙担鍐插閸欏繘鏌涢妷銏℃珖缁炬儳銈搁弻锝呂熼悜妯锋灆闂佺粯鎸搁妶鎼佸蓟閻旂⒈鏁婄紒娑橆儐閻ｅ爼姊虹€圭姵顥夋い锔诲灦閸┿垺鎯旈埦鈧弸搴ㄥ箹鏉堝墽绉紒缁樺灥閳规垿鎮╅幇浣告櫛闂佸摜濮甸悧鐘诲极閸愵喖唯鐟滃宕戦幘鍓佺當婵炴垶蓱閸ｈ櫣鈧娲栧鍓佹崲濠靛顥堟繛鎴濆船閸撴澘顪冮妶搴′簻闁硅櫕锕㈠濠氬Ω閵夈垺鏂€闂佺硶鍓濋敃鈺佄涢妶澶嬧拺闂傚牊绋掗ˉ鐐烘煕閺冣偓閻熲晛顕ｆ繝姘櫜闁糕剝锚閸斿懘姊洪棃娑氱疄闁糕晛瀚板鎶藉灳閺傘儲鏂€闂佺粯锕╅崰鏍倶椤忓牊鐓ラ柡鍥悘鍙夘殽閻愭彃鏆為柕鍫秮瀹曟﹢鍩℃担鎻掍壕濠电姵纰嶉悡鐘绘煙椤撶喎绗掗柛鏃€绮嶇换娑㈠川椤斿墽鐓夊┑顔硷攻濡炶棄鐣烽妸锔剧瘈濞达綀娅ｇ粻鎺楁⒒娴ｈ鍋犻柛鏂胯嫰閻ｇ兘顢楅崟顐ゅ幒闂佸湱鍋撻弸濂稿绩娴犲鍊甸柨婵嗛娴滄劙鏌熼柨瀣仢婵﹥妞藉畷銊︾節閸曨厾鏆ら梺璇插閼归箖宕查幓鎺戠カ濠电娀娼ч崐鎼佸箟閿熺姵鍋傞柡鍥ュ灩缁犲湱绱掗娆炬綈妞ゎ偄锕弻锝夊箻閸愬弶鍊梺闈涙搐鐎氱増鎱ㄩ埀顒勬煥濞戞ê顏柛锝庡弮閹宕归锝囧嚒闁诲孩鐭崡鎶芥偘椤曗偓瀹曞爼顢楁径瀣珝婵犵數鍋為崹鍫曟偡閵夆晩鏁傞柣鐔稿閺€浠嬫煟濡櫣浠涢柡鍡忔櫊閺屾稓鈧綆鍓欏暩婵炲瓨绮庨…鍫ュ煝鎼淬劌绠ｆ繝闈涙娴滅増淇婇悙顏勨偓鏍偋濡ゅ啰鐭欓柟鎯у娑撳秴螖閿濆懎鏆為柣鎾存礋閺屾洘寰勫Ο鐑樼亪婵犫拃灞界仸闁哄矉缍侀獮妯尖偓娑欘焽椤︿即姊洪崫鍕効缂佺粯绻傞悾鐑藉箳閹存梹鏂€闂佹悶鍎弬鍌炲焵椤掆偓閿曨亜顫忛搹鍦＜婵☆垵娅ｉ悷銊╂⒑绾懎袚缂侇喗鎹囧畷娲焵椤掍降浜滈柟鐑樺灥閺嬨倖绻涢崗鐓庡缂佺粯鐩畷锝嗗緞鐏炶В鎷＄紓鍌欐祰娴滎剟宕戦悙鍨床婵犻潧顑呴悙濠囨煏婢跺牆鐏╁ù婊勫劤闇夐柨婵嗘川閵嗗﹥淇婇幓鎺斿濞ｅ洤锕、娑㈡倷鐎涙ɑ娈告俊鐐€ら崑鍕矓閹绢喖鐓橀柟杈鹃檮閸婄兘鏌涘▎蹇ｆ▓婵☆偆鍋ら弻锝嗘償閵忊懇鏋旈梺鍝勬噽婵挳鎮鹃悜绛嬫晢濞达絽鎽滅粣鐐烘⒑瑜版帒浜伴柛妯绘倐楠炲繒绱掑Ο鑲╊啎闁哄鐗嗘晶浠嬪礆娴煎瓨鐓欓悹鍥囧懐鐦堝Δ鐘靛仜濡繂鐣锋總绋课ㄩ柨鏃€鍎抽獮鍫濃攽閻樺灚鏆╁┑顔芥尦瀹曟劙宕烽鐔奉伕闂佽法鍠撴慨鐢告偂閸愵亝鍠愭繝濠傜墕缁€鍫熺箾閹存瑥鐏柛瀣剁節閺岀喖骞嗛悧鍫閻庣懓鎲＄换鍐Φ閸曨垰鍐€闁靛ě鍛帓闂備礁鎲￠弻銊╂偉閻撳寒娼栫紓浣诡焽閻熷綊鏌嶈閸撶喖宕洪埀顒併亜閹烘垵鈧憡绂掑鍫熺厾婵炶尪顕ч悘锟犳煛閸涱厾鍩ｆ鐐达耿椤㈡瑩鎮剧仦钘夌闂佽楠搁崢婊堝磻閹剧粯鍊甸柨婵嗛婢ф壆鎮敃鍌涒拻濞达絿鐡旈崵鍐煕閻樺啿娴€殿喗鐓￠幃鈺呮嚑椤掍焦顔曟繝娈垮枟閵囨盯宕戦幘鎼闁绘劕鐡ㄥ畷灞绢殽閻愭潙鐏存い銏℃礋濡鹃亶鏌涜箛鎾剁伇缂佽鲸甯楀蹇涘Ω閿曗偓闂夊秹姊洪悷鏉挎Щ闁硅櫕锚閻ｇ兘顢曢敃鈧粈瀣煙閹碱厼鐏ｇ紒澶樺櫍閺岋紕浠﹂崜褎鍒涙繝纰樺墲閹倹淇婇悜鑺ユ櫜闁告侗鍓欓崹婵囩節閻㈤潧啸妞わ綆鍠氬Σ鎰板即閵忕姵鐎繝鐢靛У濮樸劌鐣垫笟鈧弻娑㈠Ψ椤旂厧顫╃紓浣哄珡閸ャ劎鍘遍悷婊冮叄閵嗗啴宕卞☉妯昏緢闂佸憡渚楅崰姘卞姬閳ь剟鎮楅崗绋垮祮闁衡偓閸楃儐鐔嗘慨妞诲亾闁糕斁鍋撳銈嗗笂閻掞箑鐣峰畝鍕厵妞ゆ梹鍎虫禒閬嶆煛娴ｇ鏆ｉ柛鈹惧亾濡炪倖甯掔€氼剟鎮″┑鍫氬亾楠炲灝鍔氭い锔诲灦瀹曟﹢鍩€椤掑嫭鍋℃繝濠傚枤濡偓閻庢鍠撻崝宥囩矉閹烘柡鍋撻敐搴′簽闁告ü绮欏楦裤亹閹烘垳鍠婇梺鍛婎焽閺咁偆妲愰悙鍝勭闁挎梻鏅崢浠嬫椤愩垺鍌ㄩ柛搴㈠▕閹箖鎮介崨濠勫幐闁诲繒鍋犻褎鎱ㄩ崒婧惧亾濞堝灝鏋熸繛鍏肩懆閻忓啯绻涙潏鍓у埌婵犫偓鏉堫偁浜归柛娑樼摠閳锋垹绱撴担鑲℃垹浜搁鍫熺厱濠电姴鍟扮粻鐐碘偓娈垮枛椤嘲顕ｉ幘顔碱潊闁绘顕ч弫瑙勭節閻㈤潧孝闁诲繑宀稿畷婵嬪冀閵婏附鐝￠梻鍌氬€搁崐鐑芥嚄閸撲礁鍨濇い鏍仜缁犳澘鈹戦悩宕囶暡闁稿骸瀛╅妵鍕冀椤愵澀绮剁紓浣哄У閻楁绌辨繝鍥ч柛娑卞枛濞呫倝姊洪崫鍕櫤闁绘搫绻濋悰顕€寮介銏犵亰闂佺绻愰ˇ顖涚妤ｅ啯鐓犵痪鏉垮船婢ь垶鏌￠崨顓滃仮婵﹤顭峰畷鎺戔枎閹存繂顬夋繝纰夌磿閸嬫稑顭囬垾宕囨殾闁靛繈鍊曠粈鍌滅棯閹峰矂鍝烘い鏃€妫冨楦裤亹閹烘搫绱甸柣鐘辩劍閻擄繝宕洪埀顒併亜閹烘埊宸ユい鈺婂墴閺岀喖鎮烽弶娆句純婵犵鍓濋幃鍌炲极閸愵喖鐒垫い鎺戝€搁ˉ姘亜閺嶎偄浠﹂柍閿嬪灴閺屾稑鈽夊鍫濆缂備胶濮甸幑鍥箖濡も偓椤繈鎳滈崹顐ｆ闂佽閰ｅ褔濡剁粙璺ㄦ殾婵せ鍋撴い銏＄懇閹稿﹥寰勯崱妞惧闂佹眹鍨归幉锟犲磻閳╁啰绠鹃柛鈩冾殘缁犵増銇勮箛锝勭敖缂佽鲸甯掗悾婵嬪礃閳哄啨鈧﹦绱撴担浠嬪摵闁瑰憡鎮傞、妯荤附缁嬭法顦板銈嗘尵婵嘲鐣烽崣澶夌箚闁靛牆娲ゅ暩闂佺顑囬崑銈夊Υ閸愵喖骞㈡繛鎴烆焽閻ｉ箖姊洪崫鍕殭闁绘妫欓崕顐︽⒒娓氣偓濞佳囨晬韫囨搩鍚嬮柛銉戝懎鎼告繝鐢靛Х閺佸憡鎱ㄩ悜钘夋瀬闁告稑锕ラ崣蹇涙煙缂併垹鏋熼柡鍛箞閺屾洟宕煎┑鎰︾紓浣哄Х婵炩偓闁绘搩鍋婂畷鍫曞Ω瑜夊Σ鍫㈢磽娴ｇ懓濮堟い銊ワ躬瀵鎮㈤崗鐓庘偓缁樻叏濡も偓濡瑩鎮鹃悜鑺モ拺闁规儼濮ら弫閬嶆偨椤栨稑娴繝鈧笟鈧娲箰鎼达絿鐣甸梺鐟板暱缁绘ê鐣烽鐐村€烽柣鎴烆焽閸橀亶姊虹紒妯荤叆闁圭⒈鍋呯粋鎺楁焼瀹ュ棛鍘搁悗鍏夊亾閻庯綆鍓涜ⅵ闂備浇顕栭崰鎾诲磹閺嶎厼绠柛娑欐綑娴肩娀鏌曟径鍫濃偓鏍€佸鈧铏规嫚閸欏鏀銈庡亜椤︻垳鍙呴梺鍝勭▉閸嬪棛绮堟繝鍌楁斀闁绘ê寮跺婵堚偓瑙勬尫缁舵岸鐛弽顬ュ酣顢楅埀顒佷繆娴犲鍊甸柣銏ゆ涧瀛濆銈庡弨濞夋洟骞戦崟顒傜懝妞ゆ牗鑹炬竟瀣⒒娓氣偓閳ь剛鍋涢懟顖涙櫠婵犳碍鐓曢柟鎹愭硾閺嬪孩銇勯銏㈢閻撱倖銇勮箛鎾村婵☆偄鍟埞鎴︽倷閺夋垹浠搁梺鐓庣秺缁犳牠濡撮崒鐐蹭紶闁告洖鐏氱€靛矂姊洪棃娑氬缂佺粯鍔欓幆浣割煥閸喓鍘遍梺闈涱焾閸庢娊鏁嶅澶嬬厱闁圭儤鎸稿ù顔锯偓瑙勬礀閵堝憡淇婇悜钘壩ㄩ柕澶堝妷閸嬫捇宕归瑙勬杸闂佸疇妫勫Λ妤呮倶閵夆晜鐓曢悗锝庡亜婵秵銇勯姀锛勬噰鐎殿喗鎸虫慨鈧柍閿亾闁圭柉娅ｇ槐鎾寸瑹閸パ呬画濠电偛寮堕悧妤呭疾閸洖钃熼柕澶涘閸欏棗鈹戦鏂や緵闁稿繑锕㈠銊╂嚍閵夛絼绨婚梺鍝勫暙閸婄敻骞忛埄鍐х箚闁圭粯甯炵粔娲煛鐏炵偓绀嬬€规洘鍎奸ˇ鍙夈亜韫囷絽澧扮紒杈ㄥ浮閹晛鐣烽崶銊ュ灡闁诲孩顔栭崳顕€宕抽敐澶婃槬闁逞屽墯閵囧嫰骞掗幋婵愪患缂佺偓鍎崇紞濠囧蓟閻旂厧绠ユい鏃傗拡閺嗩參姊虹紒妯诲鞍缂佸鎹囬崺鈧い鎺嗗亾闁告ɑ绮撳畷鎴﹀箻缂佹ê鈧敻鏌ㄥ┑鍡涱€楀褎澹嗛幃顕€鏁冮崒娑掓嫽婵炶揪绲块悺鏃堝吹閸愵喗鐓曢柣妯诲墯濞堟粓鏌熼鎯у幋闁糕斁鍓濋幏鍛存倻濡椿鍟庨梻鍌欑劍鐎笛呮崲閸屾娑樜旈崘鈺婂仺闂佺粯鍔楃换婵堟崲閸℃ǜ浜滈柡宥冨妿閳洟鏌曢崶鈺佹瀾濞ｅ洤锕、鏇㈡偐濞村浜炬俊銈呭暞椤洟鏌熼幑鎰靛殭缁炬儳鍚嬮幈銊ヮ潨閸℃骞嬮梺绋款儐閹告悂鎮鹃敓鐘茬疇濠电姴鍊荤粔铏光偓瑙勬礀閻栧吋淇婇幖浣规櫜闁告洦鍘鹃悡澶愭⒒閸屾瑧顦﹂柟璇х磿缁瑩骞嬮敂鑺ユ珖闂佹寧娲栭崐褰掑磻閻斿吋鐓曟い鎰剁稻缁€鍐煟閹惧瓨绀嬮柡灞界Ч閸┾剝鎷呴崨濠冾啀闂備礁鐤囧Λ鍕囨潏鈺傤潟闁圭儤顨嗛崑鎴︽煃瑜滈崜鐔风暦瑜版帒閱囬柡鍥╁仩閹芥洟姊洪幐搴ｇ畵闁瑰弶锕㈤幃銏ゅ礂閻撳孩顓奸梻渚€娼ч悧鍡涘疮椤愶絿顩烽柨鏇炲€归埛鎺懨归敐鍜佹綗闁逞屽厴閸嬫捇姊虹粙娆惧剱闁瑰憡鎮傚﹢渚€姊虹紒妯忣亜螣婵犲洤纾块柟鎵閻撶喖鏌熼幆褏鎽犵紒鈧€ｎ喗鐓涢悘鐐垫櫕鏁堥梺绯曟杹閸撴繈骞忛崨鏉戝窛濠电姴鍠氶崯鍫熺節閻㈤潧袨闁搞劌銈搁敐鐐村緞閹邦厽娅囬梺闈涚箞閸婃洟鎷戦悢鍏肩叆婵犻潧妫欓崳鎶芥煛閳ь剚绂掔€ｎ偆鍘甸梻渚囧弿缁犳垶鏅堕弶妫靛綊鎮╅搹顐⑩偓鎰叏婵犲啯銇濈€规洦鍋婂畷鐔碱敃椤愶及锝夋⒒娴ｅ憡鍟為柟姝屽吹閹广垽宕掗悜鍡樻櫔闂侀潧顦弲娑氱矆鐎ｎ偁浜滈柡宥冨姀婢规﹢鏌ら弶璺ㄤ虎闁宠鍨块崺銉╁幢濡ゅ啩娣柣搴ゎ潐濞叉垹寰婃繝姘﹂柟鐗堟緲楠炪垺绻涢幋鐐垫喛闁归绮换娑欐綇閸撗冨煂闂佺顕滅换婵嬬嵁閸℃稑閱囬柕澶涚畱娴犫晛鈹戦绛嬬劷闁告鍕洸闁规鍠掗崑鎾舵喆閸曨剛顦ㄧ紓浣筋嚙閸婂潡骞冮幆褉鏀介悗锝庝簽閸旓箑顪冮妶鍡楃瑨閻庢凹鍓熼幆?nil闂?//
// tryStickySessionHit attempts to get account from sticky session.
// Returns account if hit and usable; clears session and returns nil if account is unavailable.
func (s *OpenAIGatewayService) tryStickySessionHit(ctx context.Context, groupID *int64, sessionHash, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, stickyAccountID int64, platform string, requiredCapability OpenAIEndpointCapability) *Account {
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

	// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣濠垫劖缍楅梺閫炲苯澧柛濠傛健楠炴劖绻濋崘顏嗗骄闂佸啿鎼鍥╃矓椤旈敮鍋撶憴鍕８闁告梹鍨甸锝夊醇閺囩偟顓洪梺缁樼懃閹虫劙鐛姀銈嗏拻闁稿本鐟х粣鏃堟煃瑜滈崜娑㈠磻濞戙垺鍤愭い鏍ㄧ⊕濞呯娀鏌熺紒銏犳灍闁绘挻鐩幃姗€鎮欓幓鎺嗘寖闂侀潧妫欑敮锟犲蓟瀹ュ牜妾ㄩ梺鍛婃尪閸斿海妲愰悙鍝勫耿婵炴垶顭囬敍娑㈡⒑閸涘﹣绶遍柛姗€绠栧鎶芥晜闁款垰浜鹃柛蹇擃槸娴滈箖姊洪崨濠冨闁告挻鐩畷銏ゅ箹娴ｇ懓鈧敻鏌涜箛鎿冩Ц濞存粓绠栭弻锝嗘償椤栨粎校闂佸憡鎸婚惄顖炲极瀹ュ鍋勯柛婵勫劤椤旀洟鏌ｆ惔锝嗘毄妞ゎ厼鐗撻、鎾诲箻閺傘儲鏂€闂佺偨鍎村▍鏇㈠窗濡椿娈介柣鎰皺缁犲鏌熼瑙勬珖闁归濞€閹崇娀顢楁径濠冩澑闂傚倸鍊风粈浣革耿闁秴纾块柕鍫濐槸閸氬綊鏌嶉崫鍕櫣缂佺姴顭烽弻锟犲磼濡搫濮曢梺鍝勫€甸崑鎾绘⒒閸屾瑨鍏岀紒顕呭灦閺佸啴鍩￠崨顓犵崶闂佸搫绋侀崢浠嬪磹閸洖绾ч柛顐ｇ☉婵¤法绱掗悩鍐插姢妞ゎ叀娉曢幑鍕瑹椤栨艾澹嬮梻渚€娼уΛ娆戞暜閿熺姴钃熺€广儱鐗滃銊╂⒑閸涘﹥灏版慨妯稿姂瀹曠増绻濋崒銈呮倯闂佸憡绮堥懗鍫曞船闂堟稈鏀芥い鏂款潟娴犳粓鏌涚€ｎ偅宕岄柡宀€鍠栭、姘跺焵椤掑倻涓嶉柡宥庡幖缁犳牗绻濇繝鍌滃闁搞倕绉归弻鏇熷緞濞戙垺顎嶉梺缁樼箓閸熷潡鍩為幋锔藉亹妞ゆ棁鍋愭导鍥р攽閻愬樊妲归柣鈺婂灦閵嗕線寮介鐐靛€炲銈嗗坊閸嬫捇鏌ｉ鐕佹疁闁哄本鐩崺鍕礃椤忎礁顫岄梻浣虹帛閹搁箖宕伴弽顓犲祦闁哄稁鍘肩粻瑙勩亜閹捐泛浠уù鐙€鍨崇槐鎾存媴閸撳弶楔闂佽桨绀侀…宄邦嚕鐠囨祴妲堟繛鍡樺灩閻﹀牓妫呴銏″婵炲弶绮忛埅鐢告⒒閸屾瑦绁版俊妞煎妿缁牊绗熼埀顒€鐣烽鐐茬骇闁瑰濮靛▓楣冩⒑閸︻厼鍔嬮柛銊у枎鍗遍柛顐ゅ枑閸欏繑鎱ㄥΔ鈧Λ妤佹櫠椤斿墽纾奸柣姗€娼ф禒婊堟煃鐟欏嫬鐏撮柟顔规櫊楠炲洦鎷呴崨濠冪彵闂傚倷绀侀幗婊勬叏閻㈡悶鈧啯绻濋崶褎鐎柣搴秵閸嬪棛寮ч埀顒勬⒑閸愯尙娈遍柛瀣崌閺屾稓浠﹂崜褏鐓傞梺缁樻尰閹瑰洤顫忛搹瑙勫珰闁炽儴娅曢悵婵堢磽娴ｅ搫校闁绘娲熼幃楣冩偪椤栨ü姹楅梺鍦劋缁诲啴寮插┑瀣拺闂傚牊绋撴晶鏇熺箾閺夋垵鈧灝鐣烽娑欏劅闁靛鑵归幏缁樼箾鏉堝墽鎮奸柣鈩冩煥椤洭骞囬悧鍫㈠幈闁瑰吋鐣崹褰掑煝閺囩姭鍋撶憴鍕闁搞劏娉涢悾椋庣矙濞嗙偓瀵岄柣蹇撶箲閻楁洘绂嶆ィ鍐╃厱鐎光偓閳ь剟宕戝☉姘变笉妞ゆ洍鍋撻柡灞诲€濋獮渚€骞掗幋婵喰戦梻浣告啞椤洭寮拠宸綎婵炲樊浜濋崵鍐煃閸︻厼浜鹃悗姘洴濮婃椽宕ㄦ繝鍐ㄩ瀺闂佽崵鍟块弲鐘充繆閹绢喖纾兼繛鎴炲哺濡绢噣姊洪崨濠勨槈闁挎洩绠撳畷銏ゆ偨閸涘﹤鈧敻鏌ｉ悢鍛婄凡妞ゅ浚鍋勯…鑳槼妞ゃ劌锕ら悾鐑芥偨缁嬭法鍊為梺瀹狀潐閸庤櫕绂嶆ィ鍐╁仭婵炲棗绻愰顏嗙磼閳ь剟宕奸妷锔惧幈闂婎偄娲﹀Λ鎴︽嚀閸ф鐓忛柛鈩冩礈椤︼箓鎽堕敐鍡欑闁糕剝鐟ユ禍鎯旈弮鍌滅＝闁稿本鑹鹃埀顒傚厴閹偤鏁傞懞銉︾彿闁瑰吋鐣崝宀勬偪妤ｅ啯鐓熸俊顖滃劋閳绘洘绻涢崗鑲╁⒈缂佽鲸鎸婚幏鍛存嚃閳╁啫鐏﹂柛鎺戯躬楠炴﹢顢欓悾灞藉籍婵犵妲呴崹顖滄媰閿曗偓鍗遍柟闂寸劍閻撴瑦銇勯弽銊х煀濞寸姾椴搁幈銊︾節閸涱噮浠╃紓渚囧枟閻熴儵鍩㈡惔銊ョ畾鐟滃秵绔熸径宀€纾介柛灞剧懅椤︼附銇勯幋婵囶棦鐎规洝顫夌缓浠嬪川婵犲倵鍋撻崼鏇熷€甸柨婵嗙凹缁ㄦ挳鏌￠埀顒佺鐎ｎ偆鍘介梺褰掑亰閸樼晫绱為幋锔界厽闊洦娲栭弸娑㈡煛鐏炲墽娲村┑鈩冩倐閺佸倹鎱ㄩ幇顏囨闂佽姘﹂～澶娒哄Ο濂芥椽寮介鍌欑胺婵犵數鍋犻幓顏嗗緤閸ф绠犻柟鐐墯閸ゆ洟鏌涢…鎴濇灀闁衡偓娴犲鐓ユ繛鎴灻鈺伱归悩顐ｆ珕闁靛洤瀚板鎾幢濡も偓椤忣偅銇勯埡鍌滃弨闁哄瞼鍠栭、娑㈡晲閸ワ妇鎹曢梻浣告惈閸燁偊鎮ф繝鍥х；闁规壆澧楅埛鎺楁煕椤愩倕鏋旈柍绗哄劜閹便劎鎲撮崟鍨杹濠殿喖锕ㄥ▍锝囧垝濞嗗繆鏋庨柟顖嗗啫顥庨梻鍌欑濠€閬嶅箠閹捐秮娲敇閵忋垹绁﹂梺鍛婂姂閸斿酣寮崇€ｎ喗鈷戞い鎾卞妼椤╊剟鏌涘▎蹇旑棦婵﹥妞藉畷銊︾節閸屾鏇熺箾鐎涙鐭岄柛瀣仧閳ь剟娼ч妶绋跨暦閹偊妲烽梺琛″亾濞寸姴顑嗛悡鍐煏婢跺牆鍔氶柡鍡氫含缁辨帡鎮▎蹇斿闁绘挻娲熼弻锟犲礃閿濆懍澹曢梻浣藉吹閸熷潡寮查悩璇茬畺濞村吋鎯岄弫濠勭棯閺夊灝鑸归柟顔藉灦缁绘繈濮€閿濆懐鍘柦鍐憾閺岋繝宕ㄩ鍛彋濠殿喖锕ュ浠嬪箠閻愬搫唯闁告劕褰為柇顖炴寠濠靛绾ч柛顐亜娴滄牕霉?	// Check if sticky session should be cleared
	if shouldClearStickySession(account, requestedModel) {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}

	// Verify account is usable for current request
	if !isOpenAIAccountEligibleForRequest(ctx, account, requestedModel, false, platform, requiredCapability) {
		return nil
	}
	if s.isOpenAIAccountRuntimeBlocked(account) {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}
	account = s.recheckSelectedOpenAIAccountFromDB(ctx, account, requestedModel, requireCompact, platform, requiredCapability)
	if account == nil {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}
	if groupID != nil && s.needsUpstreamChannelRestrictionCheck(ctx, groupID) &&
		s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel, requireCompact) {
		_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
		return nil
	}

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鍝勭Ф閸斿秵銇勯弬鎸庡枠婵﹦绮幏鍛存嚍閵壯佲偓濠囨⒑閸濄儱校闁圭顭烽獮鍫ュΩ瑜夐崑鎾绘濞戞瑦鍠愮紒鐐劤閵堟悂寮诲澶婁紶闁告洦鍓欏▍銈夋⒑閹惰姤鏁遍柛銊ョ埣婵＄敻宕熼姘鳖唺闂佺硶鍓濋妵鐐寸珶閺囩姷纾介柛灞剧懅椤︼附銇勯敂鍨祮妤犵偞鍔欓獮搴ㄦ寠婢跺矈鍞甸梻浣侯攰閹活亪姊介崟顖氱；闁告洦鍨遍崐鍫曠叓閸パ勬崳闁告柨绉归弻锟犲磼濡も偓娴滈箖姊婚崒姘偓鐑芥嚄閸撲礁鍨濇い鏍ㄧ箖閹冲矂鏌ｉ悢鍝ョ煁婵犮垺锕㈠畷顖炲级閹搭厼娈ㄩ梺鍛婂姈缁佹挳寮告惔銊︾厵闁逛絻娅曞▍鍐磼閵娿儺鐓兼慨濠冩そ瀹曨偊宕熼鍌樺亽缂傚倷鑳剁划顖滄崲閸繄鏆︽い鎰ㄦ寣濞差亶鏁傞柛娑卞灠楠炲秴鈹戦悙瀛樺鞍闁煎綊绠栭幃妯衡攽閸ワ妇绠氭繝闈涘€搁幉锟犲箠濮樿埖鐓ユ繝闈涙－濡插綊鏌ｉ幙鍕М闁绘搩鍋婂畷鍫曞Ω閿旂虎妲版俊鐐€曠换鎺楀窗閹捐埖顫曢柟鐑樻尰缂嶅洭鏌嶆潪鐗堫樂缂侇喖鐖煎?TTL 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柟缁㈠枟閸庡顭块懜闈涘缂佺嫏鍥х閻庢稒蓱鐏忣厼霉濠婂懎浜惧ǎ鍥э躬婵″爼宕熼鐐差瀴闂備礁鎲￠悷銉ф崲濮椻偓瀵鏁愭径濠勵吅闂佹寧绻傚Λ顓炍涢崟顓犵＜闁绘劦鍓欓崝銈嗐亜椤撶姴鍘寸€殿喖顭烽弫鎰緞婵犲嫮鏉告俊鐐€栫敮濠囨倿閿曞倸纾块柟鎯у绾惧ジ鏌ｉ幇闈涘闁告柣鍊栫换娑氭兜妞嬪海鐦堥悗娈垮枛椤攱淇婇崼鏇炶Е闁靛牆鎳忕拹锟犳煃瑜滈崜銊х礊閸℃稑纾婚柛娑樼摠閸嬬喖鏌￠崶銉ョ仾闁稿﹦鏁婚幃宄扳枎韫囨搩浠剧紓浣插亾闁告劦鍠楅悡鐔兼煙閸喖顏紒鈾€鍋撻柣搴㈩問閸犳牠鈥﹂悜钘夌畺闁靛繈鍊曞婵嗏攽閻樻彃顏懖鏍⒒閸屾瑧顦﹂柟璇х節楠炴劙宕卞☉妯虹獩濡炪倖姊婚悺鏃堝疮閸涘瓨鈷掗柛灞捐壘閳ь剛鍏橀幊妤呭礈娴ｇ鐏婂銈嗙墱閸嬫稓绮堟径鎰厸闁搞儯鍎遍悘顏堟煛閸涱喚绠栭柕鍥у缁犳盯骞樼捄渚毇闂備浇妫勯崯浼村窗閺嶎厼钃熼柕濞垮劗閺€浠嬫煕閳锯偓閺呮粎鐟х紓鍌氬€烽懗鑸垫叏閻㈢數鐭欓柟鐑橆殔妗呴梺鍛婃处閸ㄤ即宕橀埀顒勬⒑闂堟丹娑欐媴閹绘帊澹曢梺鍦劋椤ㄥ棝鎮¤箛娑欑厱闁斥晛鍘鹃鍛弿闁搞儯鍔婃禍婊勩亜韫囨挸顏╅柡鍡到閳规垿鍩勯崘銊ュЦ缂備胶濮电粙鎴︼綖濠靛鏁嗗ù锝囩《閸嬫捇寮撮姀鈾€鎷绘繛杈剧到閹诧繝骞嗛崼銉︾厵闁惧浚鍋勬慨鍫㈢磼缂佹娲撮柛鈹惧亾濡炪倖甯掗崐鐢稿磻?	// Refresh session TTL and
	_ = s.refreshStickySessionTTL(ctx, groupID, sessionHash, openaiStickySessionTTL)
	return account
}

// selectBestAccount 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€挎鐐插暙铻栭柍褜鍓熼崺銉﹀緞婵炵偓鐎诲┑鐐叉濞撮攱娼婚弮鈧换婵嬫偨闂堟稐绮堕梺缁橆殔閹虫劙宕氶幒鎴旀瀻闁规儳鐤囬幗鏇㈡⒑缂佹ɑ鐓ラ柛姘儔閸╂盯骞掑Δ浣糕偓鐢告⒒閸喓鈽夋い銉ョ箲娣囧﹤顔忛鍏肩彎濠殿喖锕ㄥ▍锝囨閹烘嚦鐔兼惞閸︻厽鍣梻鍌欐祰椤曆呮崲閹存繄鏆嗛柛娑橈梗缁诲棝鏌ｉ姀銏╃劸缂佲偓鐎ｎ偁浜滈柡宥冨妽閻ㄦ垶銇勯弬鍖¤含闁诡喗顨婇悰顕€宕归鐓庮潛闂備胶绮〃鍛涘┑鍡欐殾闁靛繈鍊栭弲鎼佹煟濡粯鐏遍柟閿嬫そ濮婃椽鎳￠妶鍛€惧┑鐘灪閿曘垹顕ｉ搹顐ｇ秶闁靛ě鍜佸晭闁诲海鎳撴竟濠囧窗閺嵮屽殨妞ゆ棃鏁崑鎾斥枔閸喗鐏曞銈嗘肠閸ヨ埖鏅ｆ繝闈涘€婚…鍫ユ煁閸ャ劊浜滈柟鏉垮缁夌敻鏌嶈閸撴瑥煤椤撶儐娼栫紓浣股戞刊鎾煕濞戞﹫宸ラ柡鍡楃墦濮婃椽宕烽褏鍔稿銈庡幘閸忔﹢鐛崘顔藉€婚柦妯侯槺閸樻悂姊洪崨濠冨闁告搫绠戣灋闁靛ň鏅滈埛鎺楁煕鐏炲墽鎳呴柛鏂跨Ч閺屾稒鎯旈垾铏瘣闂傚洤顦伴幈銊ノ熼幐搴ｃ€愮紒鐐劤椤兘寮诲☉銏犲嵆闁靛鍎辩粻鍝勵渻閵堝棗濮冪紒顔界懇瀵鏁愭径濠勭杸濡炪倖姊婚崢褎淇婂ú顏呪拺缂備焦蓱鐏忕増绻涢懠顒€鏋涚€殿喖顭烽弫鎾绘偐閼碱剙鈧偤姊洪棃娑辨Ф闁稿酣浜堕垾鏍醇閵夛腹鎷洪柣鐘叉穿鐏忔瑧绮婚崘娴嬫斀闁绘劏鏅涙禍楣冩⒒娴ｅ憡鎯堥柣顓烆槺缁辩偞绗熼埀顒勫Υ娴ｅ壊娼ㄩ柍褜鍓熼獮鍐Χ閸℃ê顎撻梺闈╁瘜閸樿棄顭块幒妤佲拻闁稿本鐟чˇ锕傛煙绾板崬浜伴柟顖氱墕閳诲酣骞樺畷鍥舵О闂備胶绮崝锕傚礈濞嗘挸绀傚┑鐘插暕缁诲棝鏌ｉ幇鍏哥盎闁逞屽墯閻楁洟顢欒箛鏃傜瘈婵﹩鍓涢敍娑㈡⒑閻熸澘鈷旂紒顕呭灦閹繝宕橀鍛瀾濠电姴锕ら悧鍡欑矆閸屾稓绠鹃柟瀵稿仩婢规﹢鏌涢妸锔剧畺缂佺粯鐩畷鍗炍旈崘顏嶅敽闂備礁鎼Λ瀵哥不閹捐钃熼柡鍥风磿閻も偓闁诲函缍嗘禍鏍磻閹捐围濠㈣泛锕ラ悗顒勬⒑缁洖澧茬紒瀣浮閹繝寮撮姀锛勫幐闂佹悶鍎崕杈ㄤ繆婵傚憡鐓曟俊顖滅帛鐏忥箓鏌″畝鈧崰搴ㄦ偩閿熺姵鍋嬮柛顐ゅ枎閸撳灚淇婇悙顏勨偓銈嗙濠婂牆鐤悗娑櫭肩换鍡涙煕椤愶絾绀€妤犵偑鍨烘穱濠囧Χ閸屾矮澹曟俊鐐€ら崑鍕洪鐑嗘綎闁惧繒鎳撶€垫煡鏌￠崶鈺佹瀾闁绘繃妫冮弻锝嗘償閵忕姴姣堥梺鍛婃尰閻熴儵鎮鹃悽绋跨妞ゆ牗绋戞禒顓炩攽閻樿宸ラ柟鍐查叄閵嗗倹绺介崨濠備缓濡炪倖鐗楁笟妤€鈻撳鍛亾鐟欏嫭绀冮柛鏃€鐟ラ悾鐤亹閹烘繃鏅濋梺鎸庣箓閹冲孩瀵兼惔銏㈢瘈缁剧増蓱椤﹪鏌涢妸锕€鈻曢柍銉畵瀹曞ジ濡烽妷褜妲锋繝娈垮枟閿曗晠宕楀鈧銊╂嚍閵夛絼绨婚梺鍝勫暙閸婂摜鏁崼鏇熺厓鐟滄粓宕滃▎鎾偓锕傛倻閽樺妲梺閫炲苯澧柕鍥у楠炴帒顓奸崶顏嗙崶闂備礁鎽滈崰搴ｆ崲濮椻偓瀵鈽夐姀鈥充汗闂佸憡鍔栬ぐ鍐箺閻㈠憡鍊垫繛鍫濈仢濞呮﹢鏌涚€ｎ亝鍣介柟骞垮灩閳规垿宕遍埡鍌氬厞婵＄偑鍊栫敮濠囨倿閿曚讲鍙㈤梻鍌氬€搁崐椋庢濮樿泛鐒垫い鎺戝€告禒婊堟煠濞茶鐏￠柡鍛埣椤㈡盯鎮欑€电甯鹃梻浣规偠閸庢粓宕橀崣銉х＞濠德板€楁慨鐑藉磻閻愬灚鏆滈柟鐑橆殢閺佸洤鈹戦崒婊庣劸闁绘劕锕﹂幉绋款吋婢跺棌鍋撻弽銊ョ窞濠电偟鍋撻弬鈧梻浣虹帛钃遍柛鎾村哺瀹曨垵绠涘☉娆戝幈闂侀潧艌閺呮瑩骞夐悙顒夋闁绘劖娼欐慨宥嗩殽閻愭煡鍙勯柟绋匡攻瀵板嫬鐣濋埀顒勫汲椤愶絿绡€鐎典即鏀卞姗€鍩€椤掍焦绀嬫鐐诧攻閹棃濡搁妷褜鍞归梻浣哄帶閹芥粓宕戦敐鍡欘洸婵犲﹤鐗婇悡娆撴煙鐟欏嫬濮堢痪顓㈢畺閺屽秷顧侀柛鎿勭畵瀹曟垶绻濋崘鈺佸伎闂侀€炲苯澧存鐐村浮閹煎綊顢曢妶搴㈤敜婵犵數濮撮敃銈夋偋婵犲洦鍋傞柕澶嗘櫆閻撴盯鏌涢幇鈺佸缂佷讲鏅滅换娑㈠礂閻撳骸顫嶇紓浣虹帛閻╊垰鐣烽妸鈺婃晣闁靛鍔岄幆鍫ユ⒒娴ｅ憡鎯堥柣顓烆槺濡叉劙寮撮悢渚祫闂佹寧娲栭崐鍝ョ矆閸垺鍠愬璺好￠敐澶婇唶闁靛鑵归幏娲煟鎼粹剝璐″┑顖ｅ弮瀹曢潧鈻庨幇顓炲伎濠殿喗顨呴悧鍡涖€呴鍌滅＜妞ゆ梹顑欓崵娆撴煃閽樺妯€濠殿喒鍋撻梺缁樕戦鏍疮鐎ｎ喗鈷掑ù锝呮啞鐠愶繝鏌嶅畡鎵ⅵ鐎规洏鍨虹缓鐣岀矙閹稿海鈧剟姊洪棃娑氬婵炲眰鍔庢竟鏇熺鐎ｎ偆鍘遍柣蹇曞仦瀹曟ɑ绔熷鈧弻宥堫檨闁告挻鐟╁畷鏌ュ蓟閵夛妇鐣抽梻鍌欑劍閹爼宕曢鐐茬濠电姴鍟伴々鐑芥煕椤愮姴鍔滈柣鎾寸懇濮婂宕掑鐓庢闂佸憡鏌ㄧ粔褰掑蓟濞戞埃鍋撻敐鍛暢缂佲檧鍋撻梻浣告惈鐞氼偊宕濋幋婵愬殨妞ゆ洍鍋撶€规洖銈搁幃銏ゅ川婵犲骸鏁归梻鍌氬€烽懗鍫曗€﹂崼銉晞闁糕剝绋戠粻鏌ユ煕閵夘喖澧紒鐘崇墪闇夐柣妯烘▕閸庢劖銇勯妷銉у缂佺粯鐩獮瀣倶閺勫繐浜归柛鎺撳浮椤㈡盯鎮欑划瑙勫闂備礁鎲￠悷锝夊磿閾忣偆鈻旈柤纰卞墰绾剧晫鈧箍鍎辩€氼喚绮ｉ弮鍌楀亾濞堝灝鏋︽い鏇嗗洤鐓″璺好￠悢鍏肩叆閻庯綆鍋呭鎴︽⒒閸屾瑨鍏岀痪顓炵埣瀹曟粌鈹戠€ｃ劉鍋撻崘顓犵杸闁哄啫鍋嗗ù鍕節闂堟稑鈧悂骞夐敓鐘茬９闁割煈鍋嗙粻楣冩煕椤愶絿绠樺ù鐘灲閺岋紕鈧綆鍓欓弸鎴︽煏閸パ冾伃濠碘剝鎮傛俊鐤槻闁愁亜宕埞?+ LRU闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊婚崟顐ｅ枠妞ゃ垺淇洪ˇ鏌ユ偂閵堝棎浜滈柟鍨暞婵炲洭鏌嶈閸忔稓绮堟笟鈧敐鐐差煥閸繄鍔﹀銈嗗笂閻掞箓宕ｈ箛娑欑厓鐟滄粓宕滈悢鐓庤摕闁挎繂鎷嬪銊╂煃瑜滈崜娆撯€﹂崶顏嶆Ъ闂?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾剧懓顪冪€ｎ亝鎹ｉ柣顓炴閵嗘帒顫濋敐鍛婵°倗濮烽崑鐐烘偋閻樻眹鈧線寮村杈┬㈤梻浣规偠閸庢椽宕滈敃鍌氭瀬闁告劦鍠楅悡銉╂煛閸ヮ煈娈斿ù婊堢畺濮婂搫效閸パ€鍋撳Δ鍛；闁规崘鍩栧畷鍙夌節闂堟稒宸濈紒鈾€鍋撻梻浣侯焾閺堫剛鍒掑畝鍕┾偓鍌毭洪鍛嫼闂佽姤锚椤︻垶寮抽悢鍏肩厱闁绘ê纾晶鐢碘偓娈垮枛椤嘲顕ｉ幘顔藉亜闁惧繗顕栭崯搴ㄦ⒒娴ｇ顥忛柣鎾崇墦瀹曚即寮介妸褏褰?nil 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呮／闁诡垎宀€鍚嬮梺鍝勭焿缂嶄線鐛崶顒夋晩闁兼亽鍎查惁搴ㄦ⒒娴ｈ銇熼柛妯圭矙閹兘鍩￠崨顔间簵濡炪倖鍔х粻鎴︽倷婵犲洦鐓忓┑鐐戔偓閸嬫挻淇婇幓鎺斿ⅱ缂佽鲸鎸婚幏鍛村箵閹哄秴顥氭繝鐢靛О閸ㄥジ宕洪弽顓炵闁哄洨濮锋稉宥夋偡濞嗗繐顏紒鐘荤畺閺屻劌鈹戦崱娑扁偓妤€顭胯閸犳牠鍩為幋锔芥櫖闁告洦鍋勬禒妯侯渻閵堝倹娅呴柕鍫㈩焾閻ｇ兘濡搁埡濠冩櫍濠电偞鍨堕悷锕傚级閸涘﹣绻嗛柣鎰典簻閳ь剚鐗曢～蹇旂節濮橆厼浠遍梺鍝勫暙閻楀﹪宕愰崸妤佺厽闁逛即娼ф晶鎵磼閻樺啿鍔ゆい顓℃硶閹瑰嫭绗熼姘闂備線娼уΛ娆戞暜閿熺姴钃熼柨婵嗘媼濞尖晜銇勯幘璺轰粶濠殿喓鍨归—鍐Χ閸℃ê纰嶉梺鍛婅壘椤戝懘顢氶妷鈺佺妞ゆ挻绋戞禍楣冩煥濠靛棝顎楀褜鍠楅〃銉╂倷閹碱厽鐣风紓浣虹帛缁诲嫰宕版繝鍋界喎煤缂佹绉归梻鍌欑劍閹爼宕濇惔銊ユ瀬濠电姵鍝庨埀顑跨椤繈骞囨担鍓愵剟鏌ｆ惔锛勪粵婵犮垺锕㈤弫鍐Ψ瑜庡畷鍙夌節闂堟稒锛嶆繛灏栨櫆閵囧嫰骞樼捄杞扮钵闂佸憡鐗楅悧鏇㈠煘閹达附鍊烽柤鎼佹涧濞懷呯磽娴ｈ棄钄兼俊顐㈠閺佸啴濮€閵堝棛鍔堕悗骞垮劚閹虫劙鎮￠幋锔解拺闂侇偆鍋涢懟顖涙櫠鐎涙ɑ鍙忓┑鐘插暞閵囨繄鈧娲﹂崑濠傜暦閻旂⒈鏁囬柣妯诲絻铦庣紓鍌氬€搁崐鎼佸磹閸濄儳鐭撶€规洖娲﹂鑺ユ叏濡寧纭鹃柡瀣╃窔閺岀喖骞嶉纰辨毉闂佺锕ら崲鏌ュΥ閹烘埈娼╅柣鎾虫捣娴狀參姊洪崫鍕櫡闁搞劏妫勯～蹇撁洪鍛姷闂佺粯鍔樼亸顏嗏偓姘緲椤儻顧侀柛銊ョ埣瀵鎮㈤崫銉ф嚌闂佸壊鐓堥崰鏍綖閸ヮ剚鈷戠紒顖涙礃閺夊綊鏌涚€ｎ偅灏い顏勫暣婵″爼宕卞Δ鍐啰闂備胶鍘ч崯鎸庢櫠鎼粹槅鍤曠憸鏃堝春閳ь剚銇勯幒鎴濐仾闁抽攱鍨垮濠氬醇閻旂儤鍒涢梺缁樼⊕缁海妲愰幒妤€绠甸柟鐑樺灍閹稿啫鈹戦悙鏉戠仴闁诲繑宀告俊鍫曟晲婢跺﹦顦ㄩ梺瀹犳〃鐠佹煡宕戦幘璇插瀭妞ゆ劧绲藉鍨攽椤旂瓔娈旀俊顐ｎ殜閻涱喖螖閸愵亞锛滈梺鐓庢憸閺佹悂宕ｉ埀顒勬⒑閸濆嫮鐒跨紓宥勭窔瀵偊骞囬弶璺ㄥ€為悷婊冪Ч閻涱喚鈧綆鍠楅埛鎴﹀级閻愭潙顥嬮柛鏂跨Ч閺屾盯寮埀顒勬偋閺囥埄鏁嬮柕澶嗘櫅缁€瀣亜閹烘垵鈧鎯侀崼銉︹拺缂佸瀵у﹢鎵棯閺夎法效妞ゃ垺顭囬幏鐘裁圭€ｎ偅鏉搁梻浣瑰缁嬫垹鈧凹鍓氱粋宥嗙附閸涘﹦鍘辨繝鐢靛Т閸熺増鏅堕悽纰樺亾鐟欏嫭绀冪紒璇插€介悘鍐⒑閸涘﹣绶遍柛姗€绠栭幃浼村Ψ閳哄倻鍘甸梺绋跨箺閸嬫劙寮冲鈧弻娑㈠Ω閵夘喚鍚嬮梺?//
// selectBestAccount selects the best account from candidates (priority + LRU).
// Returns nil if no available account. The second return reports whether at
// least one candidate was filtered out solely because it lacks compact support
// (only meaningful when requireCompact=true).
func (s *OpenAIGatewayService) selectBestAccount(ctx context.Context, groupID *int64, accounts []Account, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, platform string, requiredCapability OpenAIEndpointCapability) (*Account, bool) {
	var selected *Account
	selectedCompactTier := -1
	compactBlocked := false
	needsUpstreamCheck := s.needsUpstreamChannelRestrictionCheck(ctx, groupID)

	for i := range accounts {
		acc := &accounts[i]

		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌″畝鈧崰鎾舵閹烘顫呴柣妯虹－娴滎亞绱撻崒娆掑厡濠殿噣绠栭敐鐐村緞閹邦儵锕傛煕閺囥劌鐏犵紒鐘崇洴閺屾盯顢曢敐鍡欘槰濡炪倕楠哥粔鐟邦潖閾忓湱鐭欐繛鍡樺劤閸撶偓绻涚€涙鐭ゅù婊庝簻椤曪絿鎷犲ù瀣潔闂侀潧绻掓慨鍫ュΩ閳哄倻鍘棅顐㈡搐閿曘倖鏅堕弻銉︾厽婵犻潧娲︾粈瀣煙椤旂瓔娈橀柟鍙夋尦瀹曠喖顢楅崒銈喰為梻鍌欒兌缁垶骞栭銈嗗床婵犻潧妫崵鏇㈡偣閸ャ劎銈存俊鎻掔墦閺屾洝绠涢弴鐐愩儵鏌ｉ妶鍌氫壕闂傚倸鍊烽懗鍫曗€﹂崼銏″床闁圭増婢樼粻鐑樼節婵犲倻澧曠紒鈧崟顓熷枑闁绘鐗嗙粭姘舵煃闁垮鐏╃紒杈ㄥ笧閳ь剨缍嗛崑鎺楀磿閵夆晜鐓曢幖娣灩婵秹鏌″畝瀣埌閾绘牕銆掑顒佸闁汇倓鐒︾换婵堝枈婢跺瞼锛熼梺杞版祰椤曆囨偩閻戣姤鍋勭痪鎷岄哺閺咁剙鈹戦绛嬬劸濞存粠鍓熷鍫曞箹娴ｅ厜鎷绘繛杈剧到閹诧繝宕悙鐑樼厵闁归棿绶″Λ鎴犵磼椤旇偐澧涚紒妤冨枛閸┾偓妞ゆ巻鍋撻柣锝囧厴瀹曨偊宕熼鐔哥暦闂備線鈧偛鑻晶鐗堜繆閸欏濮嶆鐐搭焽閳ь剚绋掗敋妞ゅ孩鎹囬弻锝夋偐閸欏顦遍梺鍛婃尰缁诲牓骞冩ィ鍐╁€婚柤鎭掑劗閹锋椽姊洪棃鈺佺槣闁告ü绮欏畷鐢稿焵椤掆偓閳规垿鎮欓懠顒佸嬀闂佺锕ョ换鍫ュ极閹扮増鍊烽柛鎾茶兌閺夋悂姊洪崫鍕殭闁稿﹤顭烽幆渚€宕煎┑鍐╂杸濡炪倖姊归弸缁樼瑹濞戙垺鐓曢柟鎯ь嚟濞叉挳鎸婂┑鍠㈠綊鎮℃惔锝嗘喖闂佹娊鏀遍崹鍧楀箖濡ゅ懎鎹舵い鎾跺€敐澶嬬厪闁糕剝顨呴弳锝夋煙椤旀娼愰柟宄版嚇瀹曠兘顢橀悙鑼摋闂佽姘﹂～澶娒哄鈧畷褰掑锤濡ゅ啫绁﹀┑鈽嗗灟鐠€锕傛偄閸℃稒鐓曢柡鍥ュ妼婢ь垱銇勯妷銉уⅵ闁哄本鐩崺鐐哄箚瑜屾竟鏇炩攽閿涘嫬浜奸柛濠冪墱閺侇噣骞掗弬鍝勪壕婵鍘ф晶鎾煙椤斿厜鍋撻弬銉︽杸闁诲函缍嗛崑鍡涘储閻㈠憡鈷戠痪顓炴媼濞兼劙鏌涢弮鎾剁暤鐎规洟娼ч埢搴ㄥ箣閻樼绱查梻浣虹帛閻熴儵骞婇幇顔剧煋妞ゆ洍鍋撻柡灞糕偓宕囨殕閻庯綆鍓欓崺宀勬煣娴兼瑧绉柡灞剧☉閳规垿宕卞Δ濠佺棯婵犳鍣徊楣冨礉閺団懇鈧棃宕橀鍢壯囨煕濞戝崬鐏ｆい鏂款樀濮婅櫣鎷犻垾铏亪闂佺锕ラ幃鍌炴晲閻愬墎鐤€闁瑰彞鐒﹀浠嬨€侀弮鍫濈妞ゆ挆鍐╂毆闂傚倷鑳堕幊鎾诲触鐎ｎ喗鍋╂い蹇撶墕閸ㄥ倸鈹戦悩宕囶暡闁哄拋鍓熼弻锟犲炊閵夈儳浠鹃梺鎶芥敱閸ㄥ湱妲愰幒鏂哄亾閿濆骸浜介柛搴涘劦閺屾稒鎯旈姀鐘差潚闂佸搫鐬奸崰鏍嵁閺嶃劍濯撮柧蹇氼潐濮ｅ洦淇?		// Skip excluded accounts
		if _, excluded := excludedIDs[acc.ID]; excluded {
			continue
		}

		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel, false, platform, requiredCapability)
		if fresh == nil {
			continue
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, false, platform, requiredCapability)
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

		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯鐟滃宕戦幘娲绘晢闁告洦鍓涢崢閬嶆⒑闂堟侗妯堥柛鐘崇墬閺呭爼顢欓崜褏锛滈梺缁橈供閸犳牠宕濆鍫熺厪闁搞儜鍐句純閻庢鍣崳锝呯暦閹烘垟妲堟俊顖濆吹閺嗘碍绻濋悽闈涗粶闁宦板妿閸掓帗鎯旈妸銉э紱闂佽宕橀褏绮婚悙鐑樼厪濠电偛鐏濋崜濠氭煟閺冨洦顏犻柣顓熺懇閺屾盯鈥﹂幋婵囩亪濡炪値鍋呴幐鎼佸煘閹达附鍋愭繛鍫熷濮ｅ矂姊洪柅鐐茶嫰婢ф煡鎮樿箛鏃傛噰闁诡垰鑻埢搴ㄥ箻鐎电骞嶆俊鐐€栧濠氭偤閺傚簱鏋旀繝濠傜墛閻撴瑦銇勯弮鍥у惞闁活厽鐟︽穱濠囶敃閵忕姵娈梺瀹犳椤︻垵鐏掗梺闈╁瘜閸樿偐绮敓鐘斥拻闁稿本鐟чˇ锔炬喐閹殿喖浠ф俊鍙夊姍椤㈡瑩宕滆椤︻垱绻涢幘鏉戠劰闁稿鎸鹃埀顒侇問閸犳盯顢氳閸┿儲寰勬繛銏㈠枛閺屻劎鈧綆鍋呭鎴︽⒒閸屾瑨鍏岀痪顓炵埣瀹曟粌鈹戠€ｃ劉鍋撻崘顓犵杸闁哄啫鍋嗗ù鍕節闂堟稑鈧悂骞夐敓鐘茬９闁割煈鍋嗙粻楣冩煕椤愶絿绠樺ù鐘灲閺岋紕鈧綆鍓欓弸鎴︽煏閸パ冾伃濠碘剝鎮傛俊鐤槻闁愁亜宕埞鎴﹀灳閸愯尙楠囬梺鍛婃⒐閻熲晠鎮伴鍢夌喓浜搁弽褌澹曞┑鐐村灦椤忣亪顢旈崼顐ｆ櫅闂佺懓澧界划顖炲煕閹达附鐓曟繛鎴烇公閸旂喓绱掗悩铏磳闁诡喗顨呴～婵嬫偂鎼淬垻浠岄梻渚€娼уú锕傚礉閺囥垹鐓濋幖娣妼缁犺崵绱掗娆炬綈閻庢艾銈稿缁樻媴閸涘﹤鏆堢紓浣割儐閸ㄥ潡寮崘顔嘉ㄩ柨鏇楀亾缂佸墎鍋ら弻娑㈠即閵娿儳浠梺缁樻尰濞茬喖鎮￠锕€鐐婇柕濠忚吂閹峰姊洪棃娑欘棡閻㈩垽绻濆璇测槈閵忕姷顔婇梺瑙勫礃濞呮洜鎷犻悙宸富闁靛牆鍟悘顏嗙磼鐎ｎ偄鐏撮柛鈹垮劜瀵板嫭绻涢姀鈩冾棃鐎规洘锕㈤、鏃堝幢濡崵妲ラ梻鍌氬€搁崐鎼佸磹妞嬪孩顐芥慨姗嗗墻閻掔晫鎲搁弮鍫濈畺鐟滄柨鐣烽崡鐐╂婵☆垵鍋愰弸鍐⒑閻熸澘鎮戦柣锝庝邯瀹曟繂鐣濋崟顒€鈧潡鏌涢…鎴濅簴濞存粍绮撻弻鐔煎传閸曨剦妫炴繛瀛樼矋閸庢娊鍩為幋锔藉亹妞ゆ劦婢€婢规洟姊绘担鍛婂暈妞ゃ劌鐗撳畷鏇㈠箣閿旇棄娈岄梺鍓茬厛閸熸棁銇愰幒鎾存珳闂佸壊鍋嗛崳銉︾閳哄啰纾奸柣鎰靛墮缁€鍐╀繆椤愩垹顏繝鈧笟鈧娲箰鎼达絿鐣甸梺缁橆殕缁挸鐣烽悽绋跨闁冲搫鍟伴鏇㈡⒑閸撴彃浜為柛鐘冲姉缁牏鈧綆鍠楅悡娑氣偓鍏夊亾閻庯綆鍓欓崺宀勬煣缂佹澧甸柡灞剧洴楠炲洭鍩℃担鍓叉闂備線娼х欢鍨紣娴ｈ櫣娉块梻鍌欑閹碱偄霉閸屾稓顩查柣鎰暩閻挻銇勯弴妤€浜鹃梺鍝勭灱閸犳牠銆佸▎鎾村€锋い鎺嶈兌瑜把呯磽閸屾瑧璐伴柛鐘愁殜閹柉顦规鐐村灴瀹曠喖顢涘В灏栨櫊閺屾洘寰勯崼婵嗗闂佸搫娲ㄩ崑鎰板绩娴犲鐓ユ繛鎴灻鈺伱瑰鍐﹀仮闁哄本绋掔换婵嬪礃椤忓嫧鎷￠梻浣芥〃缁讹繝宕板Δ鍐╁床婵犻潧顑呴悙濠囨煏婵炲灝鍔滈柣锝変憾濮婄粯鎷呯粙娆炬闂佺顑呴幊搴ｅ弲闂佸搫绋侀崢浠嬪磻閸屾稓绡€闂傚牊渚楅崕鎰版煟閹惧娲撮柡灞剧洴婵＄兘鏁愰崨顓烆潛闂備浇顕уù姘椤忓牆钃熼柨婵嗩槸缁犲鎮楀☉娆樼劷妞わ负鍔庣槐鎾存媴閾忕懓绗￠梺鐑╂櫓閸ㄥ爼鎮伴閿亾閿濆簼绨撮柡鈧禒瀣厱闁斥晛鍟虫竟姗€鏌ｉ幘杈捐€挎慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵＄偑鍊栭崹闈浢洪鐐垫殾濞村吋娼欑粻濠氭偣閸ヮ亜鐨洪柛鏃撶畱椤啴濡堕崱妤冪憪缂備浇顔愮换婵嗙暦椤栨繄鐤€婵炴垶鐟ч崢閬嶆⒑缂佹〞鎴﹀礈濮橆兘鏋旀繝濠傚濞堜粙鏌ｉ幇顓熺稇濠殿喖绉甸〃銉╂倷鐎电顫ч梺鐟板槻閹虫ê鐣烽悜绛嬫晣闁绘娅曢鎴︽⒒閸屾艾鈧悂宕愰幖浣哥９闁绘垼濮ら崐鍧楁煥閺冨牊鏆滈柛瀣尵缁厼鈽夊Ο璇叉閻庤娲橀悡锟犲蓟濞戞鏆嗛柍褜鍓熷畷鎴︽倷閸濆嫮锛涢柣搴秵娴滄牠寮ㄦ禒瀣厽婵☆垵顕х徊缁樸亜韫囷絼閭柡灞剧缁犳盯寮崒婊呭帨闂備礁鎼張顒勬儎椤栫偑鈧線寮撮姀鈩冩珳闂佺硶鍓濋悷顖毼ｆ导瀛樷拻濞达綀娅ｇ敮娑㈡煙濮濆苯鍚圭紒顔界懇楠炲鏁傞悾宀€鏆㈤梻鍌氬€烽悞锔锯偓绗涘懏宕查柛宀€鍊涢崶顒€绾ч幖瀛樻尰椤秹鎮峰鍕棃鐎殿喛顕ч濂稿醇椤愶綆鈧洭姊绘担鍛婂暈闁规瓕顕ч～婵嬪Ω閳轰胶顔夐梺闈涚箞閸婃洟鏌嬮崶顒佺厪濠㈣鍨伴崯浼村Χ閿曞倹鈷掑ù锝堟閵嗗﹪鏌涢幘瀵哥畼缂侇喗鐟╅獮瀣偐閻㈢數鍔堕梻浣告啞閸斿繘寮崒娑氼洸婵犲﹤鐗婇悡娑㈡煕閹扳晛濡介柟鍏煎姇闇夋繝濠傛－濞兼劙鏌曢崶褍顏柡浣稿€块幊鐐哄Ψ瑜嶉崵鎺撲繆閻愵亜鈧倖绂嶅鍫濈柈閻庢稒眉缁诲棝鏌涢锝嗙妤犵偑鍨烘穱濠囶敍閻愬瓨鏆犻柣搴㈢煯缁瑥顫忕紒妯诲缂佸瀵уВ鎰版⒑閹肩偛鐏柣鎿勭節楠炲啳顦圭€规洖宕灃闁逞屽墰缁﹪顢曢敂瑙ｆ嫽闂佺鏈悷褏鎷规导瀛樼厱闁规儳顕幊鍛磼椤旇姤顥堥柟顔炬櫕缁瑧鎹勯妸銉ョ疄?		// Select highest priority and least recently used
		if selected == nil {
			selected = fresh
			selectedCompactTier = compactTier
			continue
		}

		// compact 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣濠垫劖缍楅梺閫炲苯澧柛濠傛健楠炴劖绻濋崘顏嗗骄闂佸啿鎼鍥╃矓椤旈敮鍋撶憴鍕８闁告梹鍨甸锝夊醇閺囩偟顓哄┑鐘绘涧閻楀啴宕戦幘娲绘晣闁绘垵妫欑€靛矂姊洪棃娑氬闁硅櫕鍔楃划缁樺鐎涙鍘藉┑掳鍊愰崑鎾翠繆椤愶絿绠炴鐐插暣閹瑩宕崟顐も偓顓烆渻閵堝棗濮夊┑顔肩－閼鸿鲸绻濆顓涙嫼闂佽崵鍠撴晶妤呭箚閸喍绻嗘い鎰剁秵濞堟洜绱掗崒姘毙х€规洘绮忛ˇ瀵哥棯閹佸仮闁哄本鐩獮妯何旈埀顒€螞濞嗘搩鏁佹俊銈呮噺閳锋垿鏌涘☉姗嗙劦闁硅揪绠戠壕鍧楁煙閹増顥夐柣鎾偓鎰佺唵闁兼悂娼ф慨鍫ユ煕鐎ｃ劌濡跨紒杈ㄥ笧閳ь剨缍嗛崢鐣屾兜閸洘鐓曢柡鍐╂尵閻ｈ鲸銇勯鍕殻濠碘€崇埣瀹曞崬螖閸愵亝鍟伴梻鍌欒兌閸樠囧疮閹稿孩娅犻幖杈剧到閸ㄦ繄绱撴担楠ㄦ粓宕戦崨瀛樼厱闁硅埇鍔嶅▍鍥煕濡吋鏆柡宀嬬稻閹棃鏁愰崱妯荤槑婵＄偑鍊ら崢濂告偋婵犲嫮鐭夐柟鐑橆殕閺呮繈鏌涚仦鍓р槈闁逞屽墮閻栧ジ鎮￠锕€鐐婂瀣閸╁苯螖閻橀潧浠︽い銊ワ躬瀵寮撮姀鐘靛€為悷婊冪Ч椤㈡棃顢橀悤浣诡啍闂佺粯鍔曞Ο濠囧磿韫囨稒鐓冮悷娆忓閻忓瓨顨ラ悙鍙夊枠妞ゃ垺锚閳藉鈻庨幋鏂夸壕濞寸厧鐡ㄩ埛?tier 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊璁查弸娆撴⒑缂佹ê绗╁┑顔哄€楅幑銏犫槈閵忕姴鑰垮┑鈽嗗灥椤曆呭枈瀹ュ鐓熼柣鏂挎憸閹虫洜绱掗悩铏磳妤犵偛鍟灃闁告侗鍠楀▍婊堟煙閼测晞藟闁告挻绻堥幃妯侯吋婢跺鎷洪梺鍛婄箓鐎氼厽鍒婇悡骞熺懓顭ㄦ惔婵堢泿濡炪値鍋勭换鎺旀閹烘嚦鐔烘嫚瀹割喒鍋撻幘缁樷拺闁告稑锕﹂埥澶愭煥閺囨ê鈧繂顕ｉ幎钘夐唶闁靛濡囬崣鍐⒑閸涘﹤濮﹂柛鐘虫礋楠炲銈ｉ崘鈺冨幐閻庡厜鍋撻悗锝庡墰琚﹂梻浣虹《閺備線宕滃┑鍫熷床婵犻潧顑呯壕鍏肩節婵炴儳浜惧┑鈩冾殕閹瑰洭寮婚敐鍡樺劅闁靛繆鏅涢弲閬嶆⒑闂堚晝绉甸柛锝忕到閻ｇ兘骞嬮敃鈧粈瀣亜閺嶇數绋婚柡鍜冪稻缁绘繈妫冨☉妯峰亾婵犳埃鈧箓宕煎┑鎰櫊濠电娀娼уú銏＄濠婂牏鍙撻柛銉ｅ妽鐏忔澘鈹戦姘倯缂佺粯鐩畷妤呭礂鐏忔牗瀚荤紓鍌欒兌缁垳鎹㈤崟顖氱獥濠电姴浼ｉ悢璁胯櫣绱掑Ο鑲┬ч梻鍌氬€烽悞锕傚箖閸洖绀夐幖娣妼閺嬩礁鈹戦悩鍐叉惛闁逞屽厸缁€渚€锝炲┑瀣疀濞达絽鍘滈弸宀勬⒒閸屾瑨鍏岀痪顓炵埣瀵剚绗熼埀顒€鐣?tier 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厾缁炬澘宕晶缁樹繆閼碱剙鍘存慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵犵數鍋涢惇浼村垂閽樺鏆︾憸鐗堝笒閹硅埖銇勯幘璺盒＄紒妤€顦靛铏圭磼濮楀棛鍔搁悗瑙勬礈閺佽鐣烽敐澶婄缂備焦顭囬崢閬嶆⒑闂堟侗妾х紒鑼跺Г娣囧﹥绂掔€ｎ偆鍘撻梻浣哥仢椤戝懏鎱ㄩ崼銏㈢＜缂備焦顭囧ú瀵糕偓瑙勬磸閸旀垿銆佸☉姗嗘僵闁挎繂妫涢埀顑垮嵆濮婄粯鎷呴崨濠冨創闂佹椿鍘奸ˇ鍗烆嚗婵犲洤纾归柣鏃囨〃閼割亪姊婚崒娆掑厡闁告鍥风稏闁哄洢鍨圭壕褰掓煛瀹擃喒鍋撻柡瀣⒐閵囧嫰骞橀崡鐐典痪闂佹娊鏀遍崹鍧楀蓟濞戙垺鏅滈悹鍥ㄥ絻缁犲綊姊洪崨濠冪叆妞ゆ垵妫濇俊鐢稿礋椤栨鈺呮煏婢舵稓鐣辨繛鍛囧洦鈷戠紓浣诡焽椤ｆ煡鏌涙惔銏犫枙妞ゃ垺锕㈠畷妤呮偂鎼达絿鐛梺璇插嚱缂嶅棝宕滃▎鎾跺祦闁割偁鍎查埛鎺戙€掑锝呬壕濠电偘鍖犻崶浣告喘椤㈡﹢濮€閻樿尙鈧參姊洪崨濠勭畵閻庢皜鍥у嚑鐎广儱顦伴悡鏇㈡煏婢舵ê鏋涘褜鍨堕弻?priority/LRU闂?
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

// isBetterAccount 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鍝勭Ф閸斿秵銇勯弬鎸庡枠婵﹦绮幏鍛喆閸曨偂鍝楅梻浣侯焾鐎涒晛顪冮挊澶屾殾婵犲﹤鍟犻弸搴ㄦ煙鐎涙绠ユ俊顐ｇ矒閹嘲顭ㄩ崨顓ф毉闁汇埄鍨遍〃濠囧箖閳ユ枼妲堟慨姗堢到娴滅偓顨ラ悙鑼虎闁告梹鑹捐灃闁绘娅曢崐鎰版煟濞戝崬娅嶇€殿喕绮欓、姗€鎮㈤崫鍕睄?candidate 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻樻爠鍥ㄧ厱閻忕偛澧介悡顖氼熆鐟欏嫭绀€闁宠鍨块、娆撴儗椤愵偂绨婚柣锝囧厴椤㈡宕熼銏犱憾闂佽娴烽弫鍝ユ兜閸洖纾婚柟鎹愬煐閸犲棝鏌涢弴銊ュ妞わ富鍠栭—鍐Χ鎼粹€茬盎濡炪倧绠撴禍鍫曞春閳ь剚銇勯幒鎴濐仾婵炴嚪鍥ㄧ厪闁割偁鍩勯悞鐐亜閺囶亞绉€规洖銈稿鎾倷濞堝灝鎮堥梻鍌欒兌缁垶鏁嬮梺璇茬箲閼规崘顣鹃柟鍏肩暘閸ㄨ崵寮ч埀顒勬⒑濮瑰洤鐏叉繛浣冲嫮顩烽柍鍝勬噺閻撴瑦绻涢懠棰濆敽缂併劎鏅埀顒€鐏氬姗€鏁冮妷鈺佄ч柨婵嗩槸缁€鍐煏婵炑冩湰鍟?current 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻橀潧骞堟繝娈垮枟閿曗晠宕㈡禒瀣︽繝闈涙閺€浠嬫煃閳轰礁鏆熼柟钘夊暞閵囧嫰鏁冮埀顒傚椤撱垹鐒垫い鎺戝枤濞兼劖绻涢崣澶屽ⅹ妞ゆ洩缍佸畷婊勬媴閻熸澘绨ユ繝鐢靛█濞佳囶敄閸℃蛋澶愬醇閻旇櫣顔曢梺鐟邦嚟閸嬬姵绔熷Ο姹囦簻闁靛牆鍊告禍鐐節绾板纾块柛瀣灴瀹曟劙濡舵径濠勶紱闂佸憡渚楅崝鍫曞传閸曞灚寤洪梺閫炲苯澧ǎ鍥э躬閹晫绮欑捄顭戞Ч婵＄偑鍊栭悧妤€顫濋妸鈺婃晩闁圭儤顨嗛埛鎴︽煕濞戞﹩鐓繛鍫燂耿閺屾稓鈧綆鍓欓弸鎴犵磼閸屾稑绗氱紒铏规櫕缁瑧鎹勯妸鈺冨礈?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍嵁瀹ュ鏁婄痪鎷岄哺濮ｅ姊绘担渚劸妞ゆ垶鍨归幑銏犫攽鐎ｎ亣鎽曢梺鍝勬川閸犳捇鎮甸懜鐐逛簻闁哄稁鐓堝▓鏂棵瑰鍫㈢暫婵﹨娅ｇ划娆忊枎閹冨闂備胶鎳撻幉锟犲箖閸岀偞鏅查柣鎰ゴ閺€浠嬫倵閿濆骸浜滃ù婊勵殜閺岀喖鎳栭埡鍕婂淇婇悪娆忔搐缁狀垱銇勯幇鍫曟闁绘挾鍠栭獮鏍庨鈧埀顑惧€曢…鍥箛椤撶姷顔曢梺鍛婄懃椤﹁鲸鏅舵潏鈺冪＜閺夊牃鏅涙禒锔剧磼缂佹绠炵€规洖鐖兼俊鐑藉閻樺崬顥氬┑鐐舵彧缁茶法娑电捄浣曪綁宕奸妷锔惧帾闂婎偄娲よ墝闁稿鎹囧顒勫Ψ閵夈倕顥氶梻浣虹帛閸ㄥ吋鎱ㄩ妶澶婄９闁割煈鍋嗙粻楣冩煙鐎电鍓卞ù鐓庢閺岀喐娼忔ィ鍐╊€嶉梺绋款儑閸犳劙濡甸崟顖氱疀闁告挷鐒﹂崑褏绱撴担鍙夘€嗛柛瀣尰缁绘繈鎮介棃娑楃捕闂佺粯顨呭Λ婵嬪箖濡　鏀介悗锝庝簻閸ゆ垿姊虹紒妯荤叆闁告艾顑夐幃锟犲Ψ閳哄倻鍘搁梺鎼炲労閻撳牆鈻撻弴鐔虹闁告侗鍙忛崝鐔兼煃瑜滈崜婵嬶綖婢跺⊕鍝勵潨閳ь剙鐣疯ぐ鎺戝瀭妞ゆ梻鍋撳▓楣冩⒑閸涘娈橀柛瀣姇鍗遍柛顐ゅ枍缁诲棙銇勯弽銊х畵濞存粌婀遍埀顒冾潐濞叉繈宕洪弽顐ｅ床婵犻潧妫岄弸鏃堟煕椤垵鏋熼柣蹇撶Ч濮婃椽妫冨☉娆忣槱闂佺儵鏅╅崹浼存偩閻戣棄绠ｉ柨鏇楀亾缂佺姾宕电槐鎾存媴鐠囷紕鍔烽梺鍛娚戦幃鍌炲蓟閿濆牏鐤€闁哄洨鍋樼划鑸电節閳封偓閸屾粎鐓撻梺鐐藉劜閻楃娀骞婇弽顓炵厸闁告劦浜风槐鏌ユ⒒娴ｈ櫣甯涢柨姘辩棯缂併垹骞栭悡銈夋倵濞戞瑱渚涙繛鍫滅矙閺岋綁骞囬鐔峰壒闂佺粯甯婄划娆撳蓟閻斿吋鍋￠柡澶嬵儥娴尖偓闂備浇顕栭崰鏇㈠础閹惰棄鍨傚Δ锝呭暙缁€鍐煠绾板崬澧伴柍璇茬箰閳规垿鎮╅崹顐ｆ瘎婵犳鍠楅幐鎶藉箚閳ь剙顭块懜闈涘缂佺嫏鍥х閻庢稒蓱鐏忣厼霉濠婂懎浜惧ǎ鍥э躬婵″爼宕熼鐐差瀴闂備礁鎲￠悷銉ф崲濮椻偓瀵鍩勯崘銊х獮闁诲函缍嗘禍婵嬵敊閸涱厸鏀介柣鎰级閳绘洖霉濠婂啰鍩ｇ€殿噮鍋婂畷鍫曨敆娴ｅ搫骞楅梻浣烘嚀閻°劎鎹㈤崘顔㈠顭ㄩ崟顓狀啎闂佸壊鍋呯粙鎴炴叏閸屾稒鍙忓┑鐘插暞閵囨繄鈧娲滈崗姗€銆佸鈧崺鍕礃闁款垰浜炬俊銈呮噺閳锋垿鏌涘┑鍕姎閺嶏繝姊虹紒姗嗘畷缂侇喖閰ｅ畷姘跺箳濡や礁娈ラ梺闈涚墕濞层劑鏁嶅鍐ｆ斀闁宠棄妫楅悘锕傛煛閸涙澘鐨洪柤楦块哺缁绘繂顫濋娑欏闂備礁鎲＄换鍌溾偓姘煎墴瀵櫕绻濋崶銊у幈闁诲函缍嗛崑鍛暦瀹€鍕厸閻忕偛澧介埊鏇㈡煙椤栨稒顥堝┑顔瑰亾闂佺粯锕╅崑鍛存倶閸℃稒鈷掑ù锝囧劋閸も偓闂佹悶鍨肩亸娆撳箲閵忋倕绠虫俊銈傚亾闁绘帒鐏氶妵鍕箳閹搭垰濮涚紓浣割槺閺佸寮诲☉銏″亹闁归鐒﹂悿渚€姊虹化鏇熸珕闁烩晩鍨堕悰顔锯偓锝庡枟閺呮粓鏌ｉ敐鍛板妤犵偛绉剁槐鎾诲磼濞嗘埈妲銈嗗灥濡盯鍩€椤掑倻鎳楅柛銉戝拋鍞归梻浣瑰缁诲倿藝娴煎瓨鍋傞柛鎰靛枟閻撱垺淇婇娆掝劅婵☆垰鐗婃穱濠囶敍濡も偓娴滈箖姊婚崒娆戭槮闁圭⒈鍋婇幆灞惧緞鐏炵晫绛忛梺绋匡功閸犳挻绂嶅▎鎾粹拻濞撴埃鍋撻柍褜鍓涢崑娑㈡嚐椤栨稒娅犻柟缁㈠枟閻撴稑霉閿濆浂鐒鹃柡鍡稻閵囧嫰濮€閳╁喚妫冮悗瑙勬礈閸犳牕螞閸愩劉妲堟繛鍡樏幃顏堟⒒閸屾艾鈧嘲霉閸ヮ剨缍栧鑸靛姇绾惧潡鏌涜箛鎾虫倯闁告瑥绻橀弻鏇㈠醇濠垫劖笑缂佺偓鍎抽…鐑藉蓟濞戙垹鍗抽柕濞垮劤娴狀參姊虹紒妯诲暗闁哥姵鐗犲濠氭偄閻撳海顔夐梺褰掑亰閸樻崘銇愯濮婃椽宕烽鐕佷户缂傚倸绉崇欢姘嚕婵犳艾惟闁靛鍨洪～宥呪攽閻愬弶顥為柛銊ョ埣瀹曟娊寮舵惔鎾存杸濡炪倖姊婚妴瀣涘顓犵闁告粌鍟伴幃濂告煕閹烘挸绗掗柍璇查叄楠炲鎮欓懠顒婄础闂傚倷绶氬褑鍣归柣蹇曞仩婵倝寮抽鍫熲拻闁稿本鑹鹃埀顒勵棑缁牊绗熼埀顒勭嵁婢舵劖鏅搁柣妯哄暱濞堢偞淇婇妶蹇曞埌闁哥姵宀稿顐﹀礋椤栨稓鍘遍梺闈涱樈閸犳牗鏅堕鐐寸厱閹肩补妲呭Σ褰掓煏閸パ冾伂缂佺姵鐩獮妯兼崉鐞涒剝瀚涢梻鍌欐祰濡椼劎绮堟担铏圭煋鐟滅増甯掗拑鐔兼煥濠靛棭妲哥紒鐘崇⊕閵囧嫰骞樼捄鐑樼亖缂備礁顑呭ú顓烆潖濞差亝鐒婚柣鎰蔼鐎氭澘顭胯閸ｏ綁寮婚敐澶婂唨鐟滃宕戦姀銈嗙厵妞ゆ洖妫涚弧鈧銈冨灪濞茬喖寮崘顔肩劦妞ゆ帒瀚崐宄扳攽閻樺弶澶勯柣鎾崇箰閳规垿鎮╅懠顒傤唺闂佷紮缍€娴滎剟鎯€椤忓牊鍊锋い鎺嗗亾闁崇粯娲橀幈銊︾節閸涱噮浠╃紓浣介哺鐢帟鐏掗柣蹇撶箲閻楁洟锝炲畝鈧槐鎾诲磼濞嗘垼绐楅梺鍝ュУ閻楁洟婀侀悗鐟板鐎氬牓鏁愭径濠勭杸闂佸搫顦冲▔鏇㈩敊婵犲洦鈷戦悷娆忓閸斻倖銇勯弴銊ュ箻缂侇喗妫冨畷濂稿即閻斿弶瀚奸梻鍌氬€搁悧濠冪▔閻熸壋妲堟慨姗嗗亝閸曞啴姊虹紒妯哄Е闁告搫绠撳畷顖炲炊椤掍讲鎷婚梺绋挎湰閻燂妇绮婇悧鍫涗簻闁哄洤妫楅崢鎹愩亹閹烘挸鈧鏌﹀Ο渚Ч闁诲寒鍓熷楦裤亹閹烘搫绱电紓浣插亾濞撴埃鍋撶€规洘甯℃俊姝岊槼闁哥姵鍔欓弻鐔告綇妤ｅ啯顎嶉梺绋匡工閻栧ジ寮婚悢鍏煎€绘慨妤€妫欓悾鐑芥⒑閹惰姤鏁遍柛銊ユ健瀵鈽夊Ο閿嬵潔濠殿喗顨呴悧鍡樻叏濞戞氨纾藉ù锝呮惈鏍￠梺鍦嚀濞差參鐛崘銊庢棃鍩€椤掑嫸缍栨繝闈涱儐閸嬪倿骞栫划瑙勵潑婵炶偐绮穱濠囨倷椤忓嫧鍋撻弽顬℃椽鍩￠崘銊х瓘闂佺锕﹂崰鎾寸濞嗘挻鈷掑ù锝呮啞閸熺偛銆掑顓ф疁鐎规洑鍗抽獮鎺懳旈埀顒傜不椤栨稓绠剧€瑰壊鍠曠花濂告煃闁垮鐏撮柡灞剧洴閺佸倻鎷犻幓鎺戭劀闂備胶绮敮鐐哄磻閹捐埖宕叉繛鎴炲焹閸嬫捇鎮藉▓璺ㄥ姼闂佸憡蓱閹告娊寮诲☉銏犵厸濞撴艾锕ㄩ崥顐︽倵濞堝灝鏋熼柟鍛婂▕楠炲啴鍩￠崨顔间缓缂佺偓婢橀ˇ杈ㄦ償婵犲洦鈷掑ù锝囩摂閸ゆ瑩鏌涢幋鐘虫珪缂佽京鍋ゅ畷鍗炩槈濡》绱遍梻浣告啞濞诧箓宕归柆宥呯９闁割偅娲橀崐鍨箾閹寸偟鎳曞〒姘⊕缁绘盯宕煎鍛厯闂佸搫鐭夌紞渚€鐛崶顒夋晣闁绘劗鏁搁妶鐑芥⒒娴ｅ憡鎯堟俊顐㈩嚟瀵板﹥銈ｉ崘锔瑰亾閸愵喖唯闁冲搫鍊搁埀顒傚厴瀵爼宕奸悢椋庮槰闂侀€炲苯澧叉繛澶嬫礋閸┾偓妞ゆ帊绶￠崯蹇涙煕閻樺磭澧甸柟顔哄劜缁轰粙宕妷銉с偊闂佽鍑界紞鍡涘闯椤曗偓瀵偊宕堕浣哄幗闂佸綊鍋婇崢浠嬪磿濡ゅ懏鐓曢柣鏃堟敱濠€鎵磼缂佹绠為柟顔荤矙濡啫霉闊彃鍔滈柕鍥у閺佹劙宕奸悤浣诡棄闂佸憡顨婃禍鍫曞蓟濞戞ǚ妲堟繛鍡樺姉缁嬪洭姊洪幖鐐插闁稿﹥鎮傚﹢渚€姊虹粙璺ㄧ闁告艾顑囩槐鐐哄箣閻樼數锛滈梺缁樏肩拃锕€顭囬幇鐗堢厪闁搞儜鍐句純濡ょ姷鍋涘ú顓㈠春閳╁啯濯撮柟缁樺笂婢规洟姊洪崫鍕枆闁告鍘ф晥闁告瑥顦辩弧鈧繝鐢靛Т閸婄粯鏅跺☉銏＄厽闁规儳纾粻鐗堛亜椤撯剝纭堕柟椋庡█閸ㄦ儳鐣烽崶锕€鎽嬮梻鍌欒兌椤㈠﹤鈻嶉弴銏犵婵炲棙鎸搁弸浣衡偓骞垮劚閹峰鎮￠妷鈺傜厱妞ゆ劑鍊曢弳閬嶆煙閻ゎ垱顏犵紒杈ㄦ崌瀹曟帒鈻庨幋婵嗩瀴婵＄偑鍊ら崢鐓幟洪銏㈠祦濠电姴娲ょ粈瀣亜閺嶃劎銆掗柛妯圭矙濮婅櫣绱掑Ο鐑╂嫻濠电偛鍚嬮悷鈺佺暦濠靛围濠㈣泛顑囬崢鎾绘偡濠婂嫮鐭掔€规洘绮岄～婵堟崉閾忚妲遍柣鐔哥矌婢ф鏁幒妤佸剹婵°倕鎳忛悡鏇㈡煙娴煎瓨娑ф鐐搭殕閵囧嫰濡烽敐鍛亾濠靛钃熼柨婵嗩槹閸嬪嫰鏌涘┑鍕姢闁绘挷绀侀埞鎴﹀煡閸℃ぞ绨婚梺鍝ュ櫏閸嬪﹪銆佸鑸垫櫜闁糕剝鐟ч惁鍫濃攽椤旀枻渚涢柛搴ｆ暬閸╋繝宕ㄩ鎯у妇濠电姰鍨煎▔娑㈡偋閸℃稒鍊舵い蹇撴噷娴滄粍銇勯幇鈺佺伄鐎涙繈姊虹€圭媭娼愰柛銊ユ健瀹曟椽濡烽埡浣歌€垮┑掳鍊曢崯鈺呭礌閺嶎偆纾介柛灞剧懅鐠愪即鏌涢敐搴℃珝鐎规洖缍婂畷濂稿即閻樿尙銈﹂梻濠庡亜濞诧妇绮欓幋鐘冲厹闁逞屽墴濮婅櫣绱掑Ο鍝勵潊闂佸搫鎳忕划鎾愁嚕椤掑嫬鐒垫い鎺戝閳锋帒霉閿濆牊顏犻悽顖涚洴閺屻劑寮村Δ浣圭彋閻庤娲橀崹鍨暦閻旂⒈鏁嗛柛灞诲€栭柨銈夋⒒娴ｇ瓔鍤冮柛銊ユ捣娴狅箓骞嗚濡诧綁姊婚崒娆戠獢婵炰匠鍥ㄥ亱闁糕剝銇傚☉妯锋瀻闁瑰瓨绮庨崜銊╂⒑濮瑰洤鐏╅柟璇х節閹繝寮撮姀锛勫幗濠碘槅鍨伴幖顐﹀汲閸楃偐鏀芥い鏃囧亹婢э箓鏌＄仦鍓ф创妤犵偞甯￠獮瀣敍濠靛棙鍎撳┑掳鍊楁慨鐑藉磻閻愮儤鏅濋柕蹇嬪€戦埀顑跨閳诲酣骞橀弶鎴犳濠电姰鍨煎▔娑⑺囬婧惧亾閸偆鍙€婵﹤顭峰畷鎺戔枎閹邦喓鍋樻俊鐐€栧ú姗€鎮ч悩鑼殾闁靛繈鍊栭崑鈺呮倶閻愰潧浜炬繛鍫滃嵆濮婃椽宕烽鐐插闂佽鎮傞ˉ鎾诲箲閵忋倕骞㈡俊鐐存礀缂嶅﹪骞冮埡渚囧晠妞ゆ梻鍘ф竟澶愭⒒娴ｈ櫣甯涢柟绋款煼閹嫰顢涢悙鑼舵憰濠电偞鍨崹褰掓倿濞差亝鐓曢柟閭﹀枛娴滈箖鏌ｉ敐鍫濅汗缂佽鲸鎸婚幏鍛村传閸曟埈鍓涢埀顒冾潐濞叉粓寮繝姘槬闁逞屽墯閵囧嫰骞掑鍥獥闂佸摜鍠庣换姗€寮诲☉銏″亹鐎规洖娲ら埛宀勬⒑鐠団€虫灁闁稿氦灏欑紓鎾绘偩瀹€鈧惌娆撴煙缁嬪灝顒㈢悮锝夋⒒閸屾瑧鍔嶉柡瀣偢瀵彃鈽夐姀鐘垫焾濡炪倖鐗楃粙鎴﹀垂閺冨牊鐓欑紓浣靛灩閺嬬喖鏌ｉ幘璺烘灈闁哄矉缍佸顕€宕掗妶鍥уЪ婵＄偑鍊戦崕閬嶎敄婢舵劕钃熼柨婵嗘噳濡插牓鏌熼悜妯诲鞍闁绘繃娲滅槐鎾存媴閾忕懓绗＄紓浣筋嚙閻楁捇鐛崘鈺冾浄閻庯綆鍋掑Λ鍐ㄢ攽閻愭潙鐏︽い銊ョ墦楠炴垿寮撮姀鈾€鎷绘繛杈剧秬濞咃絿鏁☉娆嶄簻闁靛鍎查崵鍥煛娴ｇ鏆ｆい銏℃瀹曠喖顢橀悢鍓插晭闂佽崵鍠愮划搴㈡櫠濡ゅ啯鏆滈柟鐑樻⒒閻棝鏌涢锝囪穿鐟滅増甯楅弲鏌ユ煕閳╁喚娈樼紒渚€鏀辩换娑氣偓娑欘焽閻帞绱掗悩宕囧⒌鐎殿喖顭烽弫鎰緞婵犲孩缍傞梻渚€娼ч悧鍡椢涘▎鎾寸叆妞ゆ挶鍨洪悡鐔兼煏韫囧﹥娅呴柣蹇曞█閺岋綁寮借閸嬨垺顨ラ悙鎼疁鐎规洖銈稿鎾偄閸涘﹦褰告繝鐢靛О閸ㄧ厧鈻斿☉銏╂晞闁搞儺鍓欓崒銊╂煕濡ゅ啫浜归柡鈧禒瀣厽婵☆垵鍋愮敮娑欑箾閹冲嘲瀚换鍡樸亜閹板墎绉垫繛鍫熺矋閵囧嫰濮€閳╁喚妫冮梺绯曟櫔缁绘繂鐣峰鈧弫鎰板川椤掆偓椤ユ碍绻濋悽闈涗粶闁绘妫濋幃妯衡攽鐎ｎ亜鍤戦梺缁樻⒐閹埖绂嶅鍫熺厪濠电偛鐏濋崝鐢碘偓瑙勬偠閸庢煡濡甸崟顖ｆ晝闁靛繈鍨婚鍥煟閹惧崬鈧牠濡甸崟顔剧杸闁圭偓鍓氭导鈧俊鐐€栭弻銊ッ洪銏犺摕闁绘柨鍚嬮崵宀勬煟濡も偓閻楀棗鈽夎缁辨挻鎷呴悷鎵シ闂佸湱鈷堥崑鍕暤閸曨垱鈷戦柛鎾村絻娴滄牠鏌涙惔銏㈠弨闁糕晜鐩獮瀣晜閻ｅ苯骞堥梻濠庡亜濞诧箓骞愰幖浣哥畺闁硅揪闄勯悡蹇涙煕閵夈垺娅呴柡瀣洴閺岋紕浠﹂崜褎鍒涙繝纰樺墲閹倹淇婇幖浣规櫆濞ｅ洦澧庨崑鎾诲箛閻楀牃鎷烘繛鏉戝悑閻熝囧箖婵傚憡鐓曢煫鍥ㄦ閼拌法鈧鍣崑濠囩嵁閸ヮ剦鏁囬柣鎰暩閻涱喖鈹戦悩鍨毄濠殿喖顕埀顒佸嚬閸ｏ絽顕ｉ搹顐㈩嚤閻庢稒顭囬崢浠嬫⒑缂佹ɑ鐓ラ柟鑺ョ矒閹本绻濋崘锔跨盎闂佹寧娲栭崐鐟扳槈瑜庨妵鍕即椤忓棛袦閻庤娲忛崝宥囨崲濠靛绀冮柕濞垮劚闊﹂梻鍌氬€风欢姘焽瑜旈垾锕傤敇閻斿墎绠氶梺鎼炲労閸撴瑩宕掗妸鈺傜厵闁告挆鍛闂佹娊鏀遍崹鍧楀箖濡ゅ懎鎹舵い鎾跺€敐澶嬬厪闁糕剝顨呴弳鐔虹磼鏉堛劌娴柛鈹惧亾濡炪倖甯婇懗鍫曟偡瑜版帗鐓冪憸婊堝礈濞嗘搩鏁嬮柨婵嗩槸缁狀噣鏌ら幁鎺戝姎闁逞屽墰閸忔﹢寮婚敐澶婎潊闁靛繆鍓濆В鍕⒑娴兼瑧鍒伴柣鏍帶椤繘鎼圭憴鍕幑闂佸憡渚楅崰鏍р槈瑜斿濠氬磼濮橆剦浠奸柣搴㈢煯閸楀啿鐣烽崫鍕ㄦ闁靛繒濮烽娲⒑閹稿孩鈷掗柡鍜佸亰瀹曘垽宕￠悙鈺傛杸闂佺粯顭囩划顖氣槈瑜庢穱濠囶敃閿濆孩鐤佸銈冨灪濡啴銆侀弴銏狀潊闁冲搫鍟弫鎼佹煟閻斿摜鐭婇梺甯到閻ｅ嘲鈹戦崱蹇旂€婚梺瑙勫劤閻ゅ洭骞楅弴鐐╂斀闁绘劖娼欓悘锕傛煕閳轰礁鏆ｇ€规洘鍨块獮妯肩磼濡粯鐝抽梺纭呭亹鐞涖儵鍩€椤掑啫鐨洪柛鎴濈秺濮婅櫣鎷犻懠顒傤唶濠电偛鐡ㄥ畝绋跨暦閿濆绀冩い鏃囧閹芥洟鎮楅獮鍨姎妞わ附澹嗛埀顒傛暩婵挳鈥﹂崸妤佸殝闁活剦浜濋崹鍧楀春閵忋倕閱囬柕澶涚畱閳ь剙鐏氱换娑㈠醇濠靛牅铏庡┑鐐叉噺閿曘垽寮诲☉銏犵閻犺櫣鍎ら悘鍫ユ⒑缂佹ü绶遍柛锝忕秮閻涱噣宕堕澶嬫櫍闂佺粯鏌ㄩ崲鍙変繆閹惰姤鈷掑ù锝堫潐閻忛亶鏌￠崨顔炬创鐎规洘婢橀～婵嬵敇閻欌偓濞村嫬顪冮妶鍡樺暗闁稿鍔欏銊︾鐎ｎ偆鍘藉┑鈽嗗灥濞咃絾绂掑☉銏＄厸闁糕€崇箲濞呭懘鏌嶇憴鍕伌妞ゃ垺宀稿畷銊╊敇閻愮數娼夐梺璇叉唉椤煤濮椻偓瀹曞綊宕稿Δ鍐ㄧウ濠碘槅鍨伴幊娆愭償椤垶鏅為柣鐘充航閸斿骸鈻旈崸妤佲拻闁稿本鐟︾粊鐗堛亜椤愩埄妲搁柣锝呭槻铻ｉ悶娑掑墲閻忓啫鈹戦悙鏉戠仧闁搞劌缍婂畷娆撴偐缂佹鍘甸柣搴㈢⊕椤洦淇婇崶顒佺厸闁逞屽墴閹崇偤濡烽敐鍕泿闂備礁鎼粔鏌ュ礉鎼淬劌鐓€闁哄洢鍨洪崐鍫曠叓閸ャ儱鍔ら柣锝囨暬閺屸€崇暆閳ь剟宕伴弽顓溾偓浣糕枎閹炬潙娈愰梺鍐叉惈閸烆參濡搁妷搴㈡閹晠妫冨☉妤冮挼缂傚倷娴囬崺鏍х暆閹间焦鍋樻い鏇楀亾妤犵偞甯掕灃闁逞屽墰缁鏁愰崱娆戠槇闂佸壊鐓堥崑鍕叏婢舵劖鐓冪憸婊堝礈濮樿泛绠伴柛鎰▕濞兼牜绱撴担鑲℃垶鍒婇幘顔界厱婵炴垶锕弨濠氭煕鎼淬垺灏柍瑙勫灴閸┿儵宕卞鍓у嚬婵＄偑鍊戦崝宀勬偋閹捐崵宓侀柟杈剧畱椤懘鏌ｅ▎灞戒壕濠电偟顑曢崝鎴﹀蓟瀹ュ牜妾ㄩ梺鍛婃尰閻熲晠銆佸鑸垫櫜闁糕剝鐟ч惁鍫濃攽椤旀枻渚涢柛妯挎閳诲秴顭ㄩ崟顓犵槇闂侀潧楠忕徊鍓ф兜閸撗勫闁冲搫鎳忛悡鍐煕濠靛棗顏柛锝嗘そ閺岋綁骞樺畷鍥╊唶濡炪値鍙€濞夋洟骞戦崟顖涙優闁荤喖鍋婇弳顐︽⒒娴ｅ憡鍟炴慨濠勬嚀椤洤鈻庨幘瀵哥暢闂傚倷鑳剁划顖炩€﹂崼婵冩瀺闁哄洨濮抽悞濠囨煥閺冨洤袚闁?//
// isBetterAccount checks if candidate is better than current.
// Rules: higher priority (lower value) wins; same priority: never used > least recently used.
func (s *OpenAIGatewayService) isBetterAccount(candidate, current *Account) bool {
	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊璁查弸娆撴⒑缂佹ê绗╁┑顔哄€楅幑銏犫槈閵忕姴鑰垮┑鈽嗗灥椤曆呭枈瀹ュ鐓熼柣鏂挎憸閹虫洜绱掗悩铏磳妤犵偛鍟灃闁告侗鍠楀▍婊堟煙閼测晞藟闁告挻绻堥幃妯侯吋婢跺鎷洪梺鍛婄箓鐎氼厽鍒婇悡骞熺懓顭ㄦ惔婵堢泿濡炪値鍋勭换鎺旀閹烘嚦鐔烘嫚瀹割喒鍋撻幘缁樷拺闁告稑锕﹂埥澶愭煥閺囨ê鈧繂顕ｉ幎钘夐唶闁靛濡囬崣鍐⒑閸涘﹤濮﹂柛妯绘そ瀹曘儵顢曢妶鍥╋紲闁哄鐗勯崝宀€绮閺屽秷顧侀柛鎾卞妿缁辩偤宕卞☉妯硷紱闂佸憡渚楅崢楣冨汲閿旈敮鍋撻崗澶婁壕闂佸憡娲﹂崜娑㈠储闂堟侗娓婚柕鍫濇婢ь剚銇勯妸銉﹀櫤缂佸倸绉归幃娆擃敄鐠恒劎鐣鹃梻浣虹帛閸旓附绂嶅鍫濈劦妞ゆ帊鐒︾粈鍫ユ煙楠炲灝鐏叉鐐叉喘椤㈡牠鎮欓懠顒傂ㄩ悗瑙勬礃鐢帟顣剧紒缁㈠弮椤ユ挾绮旈鈧弻鐔碱敊閻熸澘鈷夐梺璇″灡閺屻劏鐏掔紓鍌欑劍钃辨繝銏″灴濮婂宕掑▎鎺戝帯缂備緡鍣崹璺虹暦濠靛鍐€妞ゆ垵褰炲Ч妤呮⒑閸涘﹤濮﹀ù婊呭仜铻炴い鏍ㄧ矋閸犳劙鏌￠崘銊у缂佺姵鐗楁穱濠囧Χ閸涱厽娈剁紓浣哄У濮婂湱鎹㈠☉銏犵闁绘劕妫欓崹鍨暦閹烘惟闁冲搫鍊婚崢鎾绘偡濠婂嫮鐭掔€规洘绮岄埢搴ㄥ箻瀹曞洦鐒鹃梻浣告惈椤︿即宕硅ぐ鎺戞辈闁挎洖鍊归悡娆愩亜閺嶎偄浠滃ù婊呭娣囧﹥绌遍崡鐐差瀳闂佸疇顫夐崹鍧楀箖濞嗗繆鏋旈柛顭戝枛婵℃挳姊绘担鍛婂暈婵﹤缍婂畷褰掑垂椤曞懎娈ㄩ柣鐘叉处缁佹潙危閸儲鐓忛煫鍥э攻濞呭棝鏌￠崨顔剧疄婵﹨娅ｇ槐鎺戭潨閸℃瑥濮奸梺璇茬箰缁绘垿鎮烽敂閿亾闂堟稏鍋㈢€规洜鍠栭、鏇㈩敃閿濆懐妲ｉ梻鍌欑窔濞佳囨偋閸℃瑦宕叉俊銈呮噹鐟欙箓鏌涘畝鈧崑鐐烘偂濞戙垺鐓曟繛鎴濆船閻忕姷鈧稒绻傞—鍐Χ鎼粹€茬盎缂備胶绮敃銏ょ嵁閸愵煈鐓ラ柛蹇撳⒔閸犳捇骞忛悩宸晠妞ゆ柨鈧喐瀚梻鍌氬€搁崐宄懊归崶顒婄稏濠㈣埖鍔﹂弫濠囨煛瀹ュ骸骞栫紒鎰殜閺屸€愁吋閸愩劌顬嗙紓浣筋嚙濡繈寮婚敃鈧灒濞撴凹鍨卞瓭闂備浇銆€閸嬫挸霉閻樺樊鍎愰柣鎾跺枑閹便劌螖閳ь剙螞濡ゅ懎鍑犳繛鎴欏灪閻撶喖鐓崶椋庡闂侇収鍨堕弻锛勪沪鐠囨彃顫囬悗娈垮枟濞兼瑨鐏掗梺鎯х箻閳ь剚绋撶粈鍕⒑鐠囧弶鍞夋い顐㈩槸鐓ゆ俊顖欒濞撹霉閻樺樊鍎忛柡鍕╁劦閺屽秷顧侀柛鎾村哺婵＄敻宕熼姘鳖唺閻庡箍鍎遍ˇ浼搭敁閺嶃劎绠鹃悗娑欘焽閻绱掗鑺ュ磳鐎殿喖顭烽幃銏ゆ偂鎼存繄鐐婇梻浣告啞濞诧箓宕滈敃鈧灋闁绘劕顕粻楣冩倵濞戞瑯鐒介柣顓熷笧缁辨帡鎮╁畷鍥р吂濡炪値鍋勭换鎰弲濡炪倕绻愮€氼剛绮ｅ☉娆戠瘈闁汇垽娼у瓭闂佺懓鍟块柊锝堟＂濠电娀娼уΛ顓炪€掓繝姘厪闁割偅绻冮ˉ婊堟煟韫囧鍔﹂柡宀€鍠栭幖褰掝敃閵忕媭娼氶柣搴ゎ潐濞测晝绱炴担鍝ユ殾闁靛ě鈧崑鎾斥槈濞嗘鍔烽梺鍛娒肩划娆忣潖閾忓湱纾兼俊顖氭禋娴滎亜鐣烽姀銈呭唨妞ゆ挾鍠庡▓婵嗩渻閵堝懐绠伴柣妤€锕幃锟犲即閵忥紕鍘搁梺绋挎湰娓氭宕曢妷鈺傜厱閹兼番鍨洪ˉ鍫ユ煛瀹€鈧崰鏍€佸▎鎾崇畾鐟滃苯鈻介鍡欑＝濞达綀娅ｇ敮娑氱磼鐠囪尙澧曢柣锝囧厴瀹曞ジ寮撮悙闈涘箰濠电偠鎻徊浠嬪箺濠婂應鍋撻棃娑栧仮婵﹤顭峰畷鎺戔枎閹存繂顬夐梻浣筋嚃閸犳牠鎮ラ悡搴ｆ殾闁哄洢鍩勯弫宥夋煟閹邦垰鐨洪柨娑氬枛濮婄粯绗熼崶褍顫╃紓浣割槸椤曨厾鍒掗崼鐔风窞闁归偊鍘鹃崢顏堟⒑閸撴彃浜濈紒璇插暙椤斿繐鈹戦崼銏★紡闂佽鍨庢担闀愬垝闂?
	// Higher priority (lower value)
	if candidate.Priority < current.Priority {
		return true
	}
	if candidate.Priority > current.Priority {
		return false
	}

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍓欑痪褔鏌ｉ妶鍛伃婵﹦绮幏鍛存惞閻熸壆顐奸梻浣告啞濡垹绮婚幘宕囨殾婵犲﹤鍟犲Σ鍫ユ煏韫囧﹥顎嗛柟绋垮暣濮婃椽宕ㄦ繝鍐槱闂佹悶鍔戞禍鍫曞箖閳ユ枼妲堥柕蹇ョ磿閸橀亶姊虹憴鍕棎闁哄懏绋掓穱濠冪附閸涘﹦鍘撻悷婊勭矒瀹曟粌鈻庨幋鐘殿槸婵犵數濮村ú锕傚疾閺屻儲鐓熼柟閭﹀墯閹牓鏌涘Δ浣糕枙闁哄矉缍佸顕€宕掑鍛缂傚倷闄嶉崝宀勨€﹀畡閭︽綎闁惧繗顫夐崰鍡涙煕閺囥劌浜滃┑顔哄灲閹鎲撮崟顒€顦╅梺鍛婃尵閸犳牠鐛崘顭戞建闁逞屽墴瀹曟椽鏁撻悩鑼槰濡炪倕绻愬Λ瀵告閺屻儲鈷掑〒姘ｅ亾闁逞屽墰閸嬫盯鎳熼娑欐珷閻庣數纭堕崑鎾舵喆閸曨剛顦ㄩ柣銏╁灲缁绘繈鐛崘銊㈡瀻闁瑰瓨鏌ㄦ禍楣冩煟閵忊槅鍟忛柣鎺斿亾椤ㄣ儵鎮欓懠顒€顤€婵烇絽娲ら敃顏堝箖濞嗘搩鏁傞柛鏇樺妼娴滈箖鏌″搴″箹缂佲偓婢跺绻嗛柕鍫濆閸斿秹鏌涘▎蹇曠闁哄瞼鍠撶槐鎺楀閻樺磭浜堕梻浣侯焾閿曘倝鎮ユ總绋胯摕闁挎稑瀚▽顏堟偣閸ャ劌绲诲┑顔芥礋濮婅櫣绱掑Ο璇叉殫闂佸摜濮甸崝妤呭箲閵忕姭妲堥柕蹇曞Х椤撴椽姊虹紒妯哄闁诲繑宀稿畷鎶筋敇閵忊€斥偓鐢告偡濞嗗繐顏紒鈧埀顒勬⒑濞茶骞栭柛濠傜仢椤曪絾绻濆顓熸珫闂佸憡娲︽禍婵嬪礋閸愵喗鐓熼柣妯哄级婢跺嫮鎲搁弶鍨殻闁炽儲鐗犲畷濂稿Ψ閿旇瀚奸梺璇查濠€杈ㄦ叏閻㈡潌澶嬪緞鐎ｃ劋绨婚梺鎸庢椤曆囨倶閵夆晜鐓熼煫鍥ㄦ煥缁椦呯磼鏉堛劍宕岀€规洘甯掗埢搴ㄥ箣濠靛棭鐎撮梻浣芥閸熷瓨绂嶉崼鏇炵畺婵°倐鍋撴い顐ｇ箞椤㈡﹢鎮欓埡鍌滀簽濠电姵顔栭崰鏍晝閿旀儳鍨濇い鏍仜缁狀垶鏌涘☉鍗炴珮婵℃彃缍婇獮鏍偓娑欘焽缁犳碍銇勯埡濠備喊婵﹨娅ｉ幏鐘诲蓟閵夈儱鍙婃繝鐢靛仒閸栫娀宕堕妸銉ュ闂備胶顭堥張顒€顫濋妸鈺佹辈闁挎洖鍊归悡娑橆熆鐠哄搫鐦ㄩ柛銈忕畵閺岋繝宕橀妸褍顣洪梺缁樺笒閻忔岸濡甸崟顖氱闁瑰瓨绻嶆禒閬嶆⒑闁偛鑻晶鍓х磼閻樿櫕灏柣锝囧厴瀹曞ジ寮撮妸锔芥珕闂備胶纭堕崜婵嬫偡閿曗偓椤潡骞嬮悙鐢电槇闂佹眹鍨藉褎绂掗敃鍌涚厱闁靛鍎抽崺锝夋煃閵夘垳鐣电€规洖鐖奸、鏍晝閳ь剙煤椤撶喍绻嗛柟闂寸閻撴盯鏌涚仦缁㈡畷閻庢艾銈稿缁樻媴閸涘﹤鏆堥梺鍛婃⒐濞叉牠顢氶敐澶婇唶闁哄洨鍋熼悾楣冩⒑閸撴彃浜栭柛搴㈢叀閹繝寮撮悙鈺傛杸闂佺粯锚瀵墎绮婇埡鍌欑箚妞ゆ劧绲介悘鎾煛鐏炲墽娲存鐐村浮楠炴﹢鎼归銉ф濠碉紕鍋戦崐鎴﹀垂濞差亝鍎庢い鏍仜缁犳牗绻涢崱妯虹仴闁搞劍绻堥幃宄扳枎韫囨搩浠煎銈呯箰缂嶅﹤顫忔繝姘＜婵炲棙鍩堝Σ顕€姊虹涵鍜佸殝缂佽鲸娲滈崚鎺戔枎閹惧磭顓洪梺鎸庢婵倕鈻嶉崶顒佲拺闂傚牊渚楅悡顓炩攽閳ヨ櫕鍠橀柟顔炬暬閹虫粓宕楅崫銉х暰婵＄偑鍊栭悧妤冩崲閸屾稓顩查柛锔诲幘濡垶鏌熼鍡曠娴狀噣姊洪崫鍕拱婵炲弶顭囬幑銏犫槈閵忕姴鐎銈嗘⒒閸樠囷綖閹剧粯鈷掑ù锝囩摂閸ゅ啴鏌涢悩鎰佹疁鐎规洖缍婂畷鐑筋敇閻曚焦缍楅梻浣告贡閸庛倕顫忛懡銈咁棜闁稿繘妫跨换鍡樸亜閺嶃劏澹橀柟顔藉灴閺屾盯骞掗幘鑸靛垱闂佽鍠楅〃鍫ュ箟閹绢喖绀嬫い鎺戝亞濡差剟姊绘担铏瑰笡闁绘娲熸俊鍫曞箹娴ｈ倽銉ッ归敐鍫濃偓浠嬪籍閳ь剟骞忛崨瀛樺仭闂侇叏鑵归崑鎾诲箻缂佹ǚ鎷婚梺绋挎湰椤ㄥ懏绂嶆ィ鍐┾拺鐟滅増甯掓禍浼存煕閻樺磭顣茬€垫澘瀚板畷鐔碱敍濞戞帗瀚介梻浣侯焾閺堫剟鎳濇ィ鍐ㄧ劦妞ゆ帒瀚峰Λ鎴炵箾閸℃劕鐏╂い顐ｇ箞椤㈡寰勭仦钘壭ㄩ梺鑽ゅ枑缁秴螞閸愵喖鍨傚Δ锝呭暞閺呮繈鏌涚仦璇测偓鏍疾閿濆鈷戦梻鍫熺〒缁犳碍淇婇悪娆忔搐缁犱即鏌涢幇闈涙灍闁抽攱鍨块弻娑樷槈濮楀牊鏁炬繝銏ｆ硾鐎氫即寮诲☉銏犖╅柕鍫濇川閻涖垽鎮楃憴鍕闁绘牕銈搁妴渚€寮▎鎯ф倯闂佺硶鈧磭绠查柣搴℃啞缁?	// Same priority, compare last used time
	switch {
	case candidate.LastUsedAt == nil && current.LastUsedAt != nil:
		// candidate 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€挎鐐插暙铻栭柛娑卞枛娴犳椽姊哄Ч鍥х伄闁稿缍侀崺鈧い鎺嗗亾闁挎洦浜璇测槈濮橈絽浜鹃柨婵嗛娴滄繈鎮樿箛锝呭籍闁哄苯绉归幐濠冨緞濡儵鏋呮俊銈囧Х閸嬬偟鏁敓鐘靛祦婵☆垰鐨烽崑鎾绘偨濞堣法鍔哥紓浣藉紦缁瑥顫忔繝姘＜婵炲棙鍨垫俊浠嬫煟鎼达絿鎳楅柛蹇曞█缂傛氨绮诲☉妯锋婵☆垵娅ｉ埀顒夊幗缁绘稓鈧數顭堝瓭濡炪倖鍨甸崯鎾春閳ь剚銇勯幋锝嗙《妞わ讣闄勯妵鍕敃閿濆洨鐓夐梺绯曟杹閸嬫挸顪冮妶鍡楃瑨閻庢凹鍓涚划濠氬冀椤撶喓鍘棅顐㈡搐椤戝懘鎮橀鈧弻锝夘敇閻旂儤鍣紓浣虹帛閻╊垰鐣烽崡鐐╂婵炲棙鍔栭鍥⒒娴ｄ警鐒炬い鎴濇楠炴垿宕堕鍌氱ウ濠碘槅鍨甸崑鎰閸忛棿绻嗘い鏍ㄧ矊鐢埖顨ラ悙鎼疁婵﹦绮幏鍛存惞閻熸壆顐奸梻浣瑰瀹€鎼佸蓟濞戞鐔兼嚒閵堝應鎷伴梻浣侯攰濞呮洜鍒掗幘姹団偓渚€寮崼婵嬪敹闂佺粯鏌ㄩ幖顐︾嵁閸儲鈷掑ù锝呮啞閹牓鏌涙繝鍛棄闁崇粯妫冨鎾偄閸涘﹦褰欓梻鍌欐祰椤曆呪偓娑掓櫊椤㈡瑩寮介鐐电崶闂佸搫绋侀崢浠嬪磻椤忓牊鐓曢柕澶堝灪濞呭洤鈽夐幘宕囆ｇ紒缁樼洴楠炲鎮欓崘鈺勫閸楅亶鏌涢銈呮灁缂佺娀绠栭弻娑㈠焺閸忕媭浜幃姗€濡疯閸嬫挸鈻撻崹顔界仌濠电偛鎳忓ú婊堝箲閵忕姭鏀介悗锝庡亜娴犳椽姊婚崒姘卞缂佸鍔曢埢浠嬵敂閸啿鎷洪梺纭呭亹閸嬫稒淇婃總鍛婄厽闁哄诞浣镐划閻庢鍣崑鍡涘焵椤掑﹦绉靛ù婊嗘硾閵嗘帗绻濆顓犲帾闂佸壊鍋呯换鍌炲汲濞嗗繆鏀介柍銉ㄥ皺閻瑩鏌″畝瀣М闁诡喓鍨归埞鎴﹀幢濡儤顏℃俊鐐€ら崑鍛洪悢鐓庤摕闁绘柨鍚嬮崐缁樹繆椤栨粎甯涢柛搴￠閳规垿鎮欓幓鎺撳€梺鑽ゅ暱閺呯姴顕ｆ繝姘伋鐎规洖娲﹀▓鐓庮渻閵堝棙鈷掗柛妯犲洤姹查柣鏂垮悑閻撶喐銇勯幇鈺佺仼妞ゅ浚鍙冮弻?
		return true
	case candidate.LastUsedAt != nil && current.LastUsedAt == nil:
		// current 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€挎鐐插暙铻栭柛娑卞枛娴犳椽姊哄Ч鍥х伄闁稿缍侀崺鈧い鎺嗗亾闁挎洦浜璇测槈濮橈絽浜鹃柨婵嗛娴滄繈鎮樿箛锝呭籍闁哄苯绉归幐濠冨緞濡儵鏋呮俊銈囧Х閸嬬偟鏁敓鐘靛祦婵☆垰鐨烽崑鎾绘偨濞堣法鍔哥紓浣藉紦缁瑥顫忔繝姘＜婵炲棙鍨垫俊浠嬫煟鎼达絿鎳楅柛蹇曞█缂傛氨绮诲☉妯锋婵☆垵娅ｉ埀顒夊幗缁绘稓鈧數顭堝瓭濡炪倖鍨甸崯鎾春閳ь剚銇勯幋锝嗙《妞わ讣闄勯妵鍕敃閿濆洨鐓夐梺绯曟杹閸嬫挸顪冮妶鍡楃瑨閻庢凹鍓涚划濠氬冀椤撶喓鍘棅顐㈡搐椤戝懘鎮橀鈧弻锝夘敇閻旂儤鍣紓浣虹帛閻╊垰鐣烽崡鐐╂婵炲棙鍔栭鍥⒒娴ｄ警鐒炬い鎴濇楠炴垿宕堕鍌氱ウ濠碘槅鍨甸崑鎰閸忛棿绻嗘い鏍ㄧ矊鐢埖顨ラ悙鎼疁婵﹦绮幏鍛存惞閻熸壆顐奸梻浣瑰瀹€鎼佸蓟濞戞鐔兼嚒閵堝應鎷伴梻浣侯攰濞呮洜鍒掗幘姹団偓渚€寮崼婵嬪敹闂佺粯鏌ㄩ幖顐︾嵁閸儲鈷掑ù锝呮啞閹牓鏌涙繝鍛棄闁崇粯妫冨鎾偄閸涘﹦褰欓梻鍌欐祰椤曆呪偓娑掓櫊椤㈡瑩寮介鐐电崶闂佸搫绋侀崢浠嬪磻椤忓牊鐓曢柕澶堝灪濞呭洤鈽夐幘宕囆ｇ紒缁樼洴楠炲鎮欓崘鈺勫閸楅亶鏌涢銈呮灁缂佺娀绠栭弻娑㈠焺閸忕媭浜幃姗€濡疯閸嬫挸鈻撻崹顔界仌濠电偛鎳忓ú婊堝箲閵忕姭鏀介悗锝庡亜娴犳椽姊婚崒姘卞缂佸鍔曢埢浠嬵敂閸啿鎷洪梺纭呭亹閸嬫稒淇婃總鍛婄厽闁哄诞浣镐划閻庢鍣崑鍡涘焵椤掑﹦绉靛ù婊嗘硾閵嗘帗绻濆顓犲帾闂佸壊鍋呯换鍌炲汲濞嗗繆鏀介柍銉ㄥ皺閻瑩鏌″畝瀣М闁诡喓鍨归埞鎴﹀幢濡儤顏℃俊鐐€ら崑鍛洪悢鐓庤摕闁绘柨鍚嬮崐缁樹繆椤栨粌鍔﹂柟鐤吹缁辨帡鎮欓鈧崝銈夋煕濮橆剦鍎愮紒宀冮哺缁绘繈宕堕懜鍨珫婵犳鍠楅敃鈺呭储娴犲纾块柟鍓х帛閳锋垿姊洪锝囥€掗柣顓熺懄缁绘稒鎷呴崘鍙夘棤濞?
		return false
	case candidate.LastUsedAt == nil && current.LastUsedAt == nil:
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄨ崵妲愰幒妤€绠涙い鎾楀嫮鏆︾紓鍌欒兌缁垶鎯勯鐐靛祦閻庯綆鍠楅崐鐑芥煕椤垵浜滄繛鍛殜濮婄粯鎷呴崨濠傛殘闂佽崵鍠嗛崕鎶藉箲閵忕媭娼ㄩ柍褜鍓欓锝嗙節濮橆剙宓嗛梺闈涚箳婵炩偓闁哥偠娉涢埞鎴︽偐缂佹ɑ閿┑鐐茬湴閸婃繈骞冮悽绋跨睄闁割偆鍟块幏娲⒑閸忚偐銈撮柡鍛洴閵嗗倹绺介崨濠傗偓鐢电磼濡や胶鈽夋繛灞傚€栭、濠囨⒒娴ｅ憡鍟炴繛璇х畵瀹曞綊骞嶉鍙ョ瑝婵犵數濮电喊宥夊磻閿濆鐓ｉ煫鍥ㄦ礃閸も偓闂佸憡鏌ㄩ澶愬蓟閿濆棙鍎熼柨婵嗘噸婢规洜绱撴担铏瑰笡缂佽鍊块崺銏ゅ箻鐠囨彃鐎銈嗘煥瑜扮偤鎮芥繝姘拻濞达絿鍎ら崵鈧梺瑙勭ゴ閸撴繄鎹㈠☉娆戠瘈闁搞儮鏅涚粊锕傛⒑閸涘﹤濮﹂柛妯款潐缁傚秷銇愰幒鎾跺幍闂佺粯鍨堕敋闁诲繆鏅犻弻锝夋晲閸℃せ鏋欏┑顔硷攻濡炶棄鐣烽悜绛嬫晣闁绘劖褰冮‖鍡涙⒒娴ｈ鍋犻柛鏂块叄瀵偅绻濆顒備紜闂佸搫绉查崝宀€娆㈤悙鐑樼厵闂侇叏绠戦獮鎰版煙闁垮銇濇慨濠冩そ瀹曘劍绻濋崟顓犳殼闂佽瀛╅崙褰掑矗閸愵厽锛傛繝鐢靛Т閿曘倝骞婇幇顔碱棜闁兼祴鏅濈壕钘壝归敐鍥仧闁稿浚鍓熼弻銈夊箯鐏炲墽銆婇梺閫炲苯澧い鏃€鐗犲畷浼村冀椤撴稈鍋撻敃鍌涚叆閻庯絺鏅濈粻姘舵⒑瑜版帗锛熺紒鈧担鍛婃殰闂傚倷绶氬褏鎹㈤崱妞綁宕ㄩ褏鍔烽梺鍝勫暊閸嬫挾绱掔紒妯肩畼闁哥姴锕よ灒婵炶尙绮紞澶愭⒒娴ｇ顥忛柣鎾崇墦瀹曞綊骞庨挊澶岊唵闂佸憡鍔﹂崰妤呭疾濠靛鐓冪憸婊堝礈閻旂厧鏄ラ柕蹇嬪€曠粻缁樼箾閼碱剛甯涢柡鍜冪秮濮婅櫣绱掑鍡欏姺闂佽桨鐒﹂幃鍌炲极瀹ュ應鏀介柛銉㈡櫇椤旀洟鏌℃径濠勫濠⒀呮櫕缁棃顢楁担铏诡啎闂佸憡绮岄崯浼村箖婵傚憡鐓涘〒姘搐閺嬫盯鏌嶉挊澶樻Ц妞ゎ偅绻堥弫鎰板川椤掆偓椤ユ岸姊绘担渚劸缂佺粯鍔欒棟妞ゆ牗绋撻々鎻捨旈敐鍛殲闁抽攱鍨归幉鎼佹偋閸繄鐟查梺鎸庣〒閸犳牠寮婚妶鍥ｅ亾閸︻厼校缁绢參绠栭弻?
		return false
	default:
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄨ崵妲愰幒妤€绠涙い鎾楀嫮鏆﹀┑鐑囩到濞层倝鏁冮鍛箚闁割偅娲栧婵囥亜閺傚灝鎮戞い蹇曞█濮婄粯鎷呯憴鍕哗闂佸憡鏌ㄩ惌鍌氱暦閹版澘鎹舵い鎾跺枎鎼村﹤鈹戦悩缁樻锭妞ゆ垵妫濊棢闁割偆鍠撶粻楣冩煙鐎电浠ч柟鍐插暟缁辨帡骞囬鐘偓鎺旂磼鏉堛劍灏伴柟宄版嚇瀹曨偊宕熼妸锔锯偓鎵磽閸屾瑨鍏屽┑顔炬暬閹囨偐瀹割喖娈ㄩ柣鐘叉处缁佹潙危閸儲鐓忛煫鍥ㄦ礀椤秹鏌曟繝蹇氱濞存粍绮撻弻锟犲礃閵娿儮鍋撻崫銉︽殰闁煎摜鍋ｆ禍婊勩亜閹伴潧浜楅梺顓у灡閹便劍绻濋崨顕呬哗缂備緡鍠楅悷銉╁煝鎼淬劌绠氱憸宥嗙珶閸曨垱鈷掑ù锝呮啞閸熺偞鎱ㄦ繝鍌滅Ш鐎规洖缍婂畷褰掝敋閸涱厽顓垮┑鐘垫暩婵敻鎳濇ィ鍐ㄧ闁规儼濮ら悡鐔兼煛閸愩劍澶勬い蹇曞У閵囧嫰寮埀顒€煤閻斿娼栨繛宸簻缁€鍫ユ煠绾板崬澧悽顖樺劦濮婃椽宕妷銉愶綁鏌ｅΔ鍐ㄢ枅闁搞劑绠栧浠嬵敃閻旇渹澹曢梺鎸庣箓妤犲憡鏅堕幍顔炬／闁硅鍔栭ˉ澶愭婢舵劖鐓ユ繝闈涙閸ｆ椽鏌熼姘卞ⅵ鐎殿喕绮欓弫鎾绘偐閺傘儲瀚肩紓鍌氬€烽悞锕傛晝閳轰讲鏋旀俊顖濄€€閺€浠嬫煟閹般劍娅呭ù婊堢畺閺岋絾鎯旈姀鈺佹櫛闂佸摜濮甸悧鐘诲灳閿曞倹鐓ラ悗锝傛櫇缁犳岸姊鸿ぐ鎺擄紵缂佲偓娴ｅ憡鏆滈梻鍌欑窔濞佳呮崲閸℃あ锝夊川椤撗呭姺闂佸搫鍟犻崑鎾剁磼缂佹绠橀柛鐘诧工铻ｆ繛鑼帛缂嶅姊绘担瑙勫仩闁告柨閰ｅ顐ゆ嫚閼碱剚娈鹃梺纭呮彧缁犳垹绮诲☉銏＄厸濠㈣泛瀛╃涵鑸点亜閿旇鏋涙い顏勫暣婵″爼宕卞▎蹇婃嫛闂備胶顭堥鍥磻閵堝鏄ユ繛鎴欏灩缁狅綁鏌ㄩ弮鍥嗘帡骞忛崫鍕垫富闁靛牆妫楁慨鍌炴煕婵犲啯绀冪紒鍌涘笒椤劑宕奸悢鍝勫箻婵犵數鍋犵亸顏堫敋瑜旈幃姗€鏁撻悩宕囧幐闂佸憡渚楅崢楣冨春閿濆棎浜滈柕蹇ョ磿閹冲嫰鏌熸笟鍨妤犵偛娲、鏃堝幢濞嗘劖娅掗梻鍌氬€搁崐鐑芥倿閿旈敮鍋撶粭娑樺悩濞戞瑦濯撮柤鍙夌箖閻╊垰鐣烽悢纰辨晬闁逞屽墯瀵板嫮浠︾粙澶稿闂佹寧绻傛鎼佸几閻斿吋鐓熼柟鎯у暱閺嗙喖鏌熼懠顒夌劸妞ゎ厹鍔戝畷鐓庘攽閸偅肖濠电姵顔栭崰妤呪€﹂崼婵冩灃婵炴垶鐟ラ閬嶆煟濡鍤欑紒鐘侯潐缁绘盯鏁愭惔鈥愁潻婵犲痉銈呬汗缂佽鲸甯￠崺鈧い鎺嶇缁剁偤鏌熼柇锕€骞橀柛妯兼暬閺岋絾鎯旈姀鈶╁濡炪値鍘煎ú銊у垝婵犳艾唯闁冲搫鍊婚崢閬嶆⒑闂堟稓澧曟俊顐ｇ懅缁牏鈧綆鍠楅悡娑氣偓鍏夊亾閻庯綆鍓涜ⅵ濠电姵顔栭崰鎺楀磻閹剧粯鈷戦梻鍫熺〒缁犳岸鏌￠崨顔剧疄鐎规洘绻堥獮鏍ㄦ媴閸忓瀚肩紓鍌氬€烽悞锕傛晪缂備焦顨嗙敮锟犲蓟閿濆牏鐤€闁哄倸鐏濋幗鍨節绾板纾块柡浣筋嚙閻ｇ兘鎮㈢喊杈ㄦ櫖濠殿喗锕㈢涵鎼佸船濞差亝鐓熼幖杈剧磿閻ｎ參鏌涙惔銊ゆ喚闁诡啫鍥у耿婵炴垶锚閸嬪秹姊虹捄銊ユ灁濠殿喚鏁婚崺娑㈠箣閻橆偄浜鹃悷娆忓缁€鈧┑鐐茬湴閸斿酣寮灏栨婵炲棙鍨归鏇㈡⒑閼测斁鎷￠柛鎾寸懇瀵鈽夊锝呬壕婵炲牆鐏濋弸锔姐亜閺囧棗娲ら悡鈥愁熆鐠哄ソ锟犳偄閸忚偐鍙嗛柣搴到閻忔氨绱炲畝鍕拻闁稿本鐟чˇ锕傛煙鐠囇呯瘈闁靛棗鍟村畷濂稿Ψ閵壯嶇幢闂備胶绮崝妤呭磿閵堝纾归柛鎾茶閸嬫捇鐛崹顔煎濡炪倧缂氶崡鍐茬暦閹版澘鍨傛い鎰╁€楅鏇㈡⒑閸撴彃浜濈紒璇插缁傛帡濮€閵堝棛鍘搁柣搴秵閸撴瑩寮稿☉銏＄厸鐎光偓閳ь剟宕伴弽褏鏆﹂柕濠忓缁♀偓闂佸憡鍔戦崝澶愬磻閹捐绠涙い鎾跺Х閻﹀牓姊洪柅鐐茶嫰婢ь垳绱掑Δ鍐ㄦ灈闁糕斁鍋撳銈嗗坊閸嬫挻銇勯銏㈢閻撱倖銇勮箛鎾愁伀妞ゆ柨娲弻鐔兼嚌閻楀牆娑х紓浣瑰絻濞硷繝骞冩ィ鍐╁仺闁告稑艌閹疯櫣绱撻崒娆戝妽闁挎洍鏅涢埢鎾诲籍閸喓鍙嗛梺鍝勬祩娴滅偤鎮鹃柆宥嗙厓闁芥ê顦藉Ο鈧梺璇″枤閺咁偆鍒掑▎鎾抽敜婵°倐鍋撻柡浣哥秺濮婄粯鎷呴崨濠冨創濠电偛鐪伴崝鎴濈暦娴兼潙绠婚悹鍥ㄥ絻閸嬪秹姊洪崨濠勨槈闁挎洩濡囩划缁樸偅閸愩劎楠囬梺鍓插亝缁诲倿鍩涢弮鍌滅＜濠㈣埖锚閺嬬喓绱掔紒妯尖姇闁瑰嘲鎳樺畷姗€宕ｆ径瀣€烽梻浣虹帛鐢帡鎮樺璺何﹂柛鏇ㄥ灠缁犲磭鈧箍鍎遍悧鍡涘储閿涘嫮纾藉ù锝呮惈鍟搁梺绋款儑閸嬨倝濡?
		return candidate.LastUsedAt.Before(*current.LastUsedAt)
	}
}

// SelectAccountWithLoadAwareness selects an account with load-awareness and wait plan.
func (s *OpenAIGatewayService) SelectAccountWithLoadAwareness(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*AccountSelectionResult, error) {
	return s.selectAccountWithLoadAwareness(s.withOpenAIQuotaAutoPauseContext(ctx), groupID, sessionHash, requestedModel, excludedIDs, false, PlatformOpenAI, "")
}

func (s *OpenAIGatewayService) selectAccountWithLoadAwareness(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, platform string, requiredCapability OpenAIEndpointCapability) (*AccountSelectionResult, error) {
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
		account, err := s.selectAccountForModelWithExclusions(ctx, groupID, sessionHash, requestedModel, excludedIDs, requireCompact, stickyAccountID, platform, requiredCapability)
		if err != nil {
			return nil, err
		}
		result, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
		if err == nil && result != nil && result.Acquired {
			return s.newAcquiredSelectionResult(ctx, account, result.ReleaseFunc)
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
				upstreamRestricted := needsUpstreamCheck && groupID != nil &&
					s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel, requireCompact)
				if clearSticky || s.isOpenAIAccountRuntimeBlocked(account) || upstreamRestricted {
					_ = s.deleteStickySessionAccountID(ctx, groupID, sessionHash)
				}
				if !clearSticky &&
					isOpenAIAccountEligibleForRequest(ctx, account, requestedModel, false, platform, requiredCapability) &&
					!s.isOpenAIAccountRuntimeBlocked(account) &&
					!upstreamRestricted {
					result, err := s.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
					if err == nil && result != nil && result.Acquired {
						selection, selectErr := s.newAcquiredSelectionResult(ctx, account, result.ReleaseFunc)
						if selectErr != nil {
							return nil, selectErr
						}
						_ = s.refreshStickySessionTTL(ctx, groupID, sessionHash, openaiStickySessionTTL)
						return selection, nil
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
		if !isOpenAIAccountEligibleForRequest(ctx, acc, requestedModel, false, platform, requiredCapability) {
			continue
		}
		if s.isOpenAIAccountRuntimeBlocked(acc) {
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

	tryAcquireFromLoadMap := func(loadMap map[int64]*AccountLoadInfo) (*AccountSelectionResult, bool, error) {
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

		if len(available) == 0 {
			return nil, false, nil
		}

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
			// tier 0 闂備胶顭堥敃銉╂偡閳哄懎鐒垫い鎺嗗亾妞わ富鍠氱槐鐐哄籍閳ь剛绮欐径鎰紶闁告洦鍋勭粻鏌ユ煙閻撳海鎽犳繛鍙夌箞楠炲繘鎮滈挊澶岊槷闂佺粯鐟ラ幊鎰玻濡楃嫙 recheck 闂備礁鎼崯顖炲磿閼测斁鍋撳銉ョ伈鐎规洩缍侀獮瀣偐閼碱剛鏉?cache tier 0 闂佽楠稿﹢閬嶅磻閵堝拋鐎?			// 闁诲海鎳撻幉陇銇愰崘銊ф殾鐎光偓閳ь剟鎯€椤忓浂妲洪柣?1/2闂備焦瀵х粙鎴︽偋婵犲嫭鏆滈柡澶嬶紩閸︻厸鍋撻敐搴濈盎闁诲繒濞€閹綊鎮滃Ο铏逛淮闂佷紮缍嗛崜鐔煎极瀹ュ洠鍋撶憴鍕綘che 闂佽绻愮换鎴犲枈瀹ュ拑鑰挎い鎾卞灩缁€鍡涙⒑閸噮鍎愰柣鎾亾闂備焦瀵х粙鎴βㄩ埀顒傜磼鏉堛劎绠栫紒鍌涘笒椤撳ジ宕ㄩ鈽嗗敼婵犳鍠楃换鎰緤娴犲鍋夐柛顐ｆ礀瀹告繈鏌熺€涙ê绗╅柛姗嗗墴閺?			selectionOrder = appendTier(selectionOrder, 0)
		} else {
			selectionOrder = append(selectionOrder, available...)
		}

		for _, item := range selectionOrder {
			fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, item.account, requestedModel, false, platform, requiredCapability)
			if fresh == nil {
				continue
			}
			fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, requireCompact, platform, requiredCapability)
			if fresh == nil {
				continue
			}
			if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, fresh, requestedModel, requireCompact) {
				continue
			}
			result, err := s.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
			if err == nil && result != nil && result.Acquired {
				selection, selectErr := s.newAcquiredSelectionResult(ctx, fresh, result.ReleaseFunc)
				if selectErr != nil {
					return nil, true, selectErr
				}
				if sessionHash != "" {
					_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, fresh.ID, openaiStickySessionTTL)
				}
				return selection, true, nil
			}
		}
		return nil, true, nil
	}

	loadMap, err := s.concurrencyService.GetAccountsLoadBatch(ctx, accountLoads)
	if err != nil {
		ordered := append([]*Account(nil), candidates...)
		sortAccountsByPriorityAndLastUsed(ordered, false)
		if requireCompact {
			ordered = prioritizeOpenAICompactAccounts(ordered)
		}
		for _, acc := range ordered {
			fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel, false, platform, requiredCapability)
			if fresh == nil {
				continue
			}
			fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, requireCompact, platform, requiredCapability)
			if fresh == nil {
				continue
			}
			if needsUpstreamCheck && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, fresh, requestedModel, requireCompact) {
				continue
			}
			result, err := s.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
			if err == nil && result != nil && result.Acquired {
				selection, selectErr := s.newAcquiredSelectionResult(ctx, fresh, result.ReleaseFunc)
				if selectErr != nil {
					return nil, selectErr
				}
				if sessionHash != "" {
					_ = s.setStickySessionAccountID(ctx, groupID, sessionHash, fresh.ID, openaiStickySessionTTL)
				}
				return selection, nil
			}
		}
	} else {
		if selection, attempted, selectErr := tryAcquireFromLoadMap(loadMap); selectErr != nil {
			return nil, selectErr
		} else if selection != nil {
			return selection, nil
		} else if attempted {
			if freshLoadMap, loadErr := s.concurrencyService.GetAccountsLoadBatchFresh(ctx, accountLoads); loadErr == nil {
				if selection, _, selectErr := tryAcquireFromLoadMap(freshLoadMap); selectErr != nil {
					return nil, selectErr
				} else if selection != nil {
					return selection, nil
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
		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, requestedModel, false, platform, requiredCapability)
		if fresh == nil {
			continue
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, requestedModel, requireCompact, platform, requiredCapability)
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

func (s *OpenAIGatewayService) resolveFreshSchedulableOpenAIAccount(ctx context.Context, account *Account, requestedModel string, requireCompact bool, platform string, requiredCapability OpenAIEndpointCapability) *Account {
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

	if !isOpenAIAccountEligibleForRequest(ctx, fresh, requestedModel, requireCompact, platform, requiredCapability) {
		return nil
	}
	if s.isOpenAIAccountRuntimeBlocked(fresh) {
		return nil
	}
	return fresh
}

func (s *OpenAIGatewayService) recheckSelectedOpenAIAccountFromDB(ctx context.Context, account *Account, requestedModel string, requireCompact bool, platform string, requiredCapability OpenAIEndpointCapability) *Account {
	if account == nil {
		return nil
	}
	if s.schedulerSnapshot == nil || s.accountRepo == nil {
		if !isOpenAIAccountEligibleForRequest(ctx, account, requestedModel, requireCompact, platform, requiredCapability) {
			return nil
		}
		return account
	}

	latest, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil || latest == nil {
		return nil
	}
	if !isOpenAIAccountEligibleForRequest(ctx, latest, requestedModel, requireCompact, platform, requiredCapability) {
		return nil
	}
	if s.isOpenAIAccountRuntimeBlocked(latest) {
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

func (s *OpenAIGatewayService) newAcquiredSelectionResult(ctx context.Context, account *Account, release func()) (*AccountSelectionResult, error) {
	selection, err := s.newSelectionResult(ctx, account, true, release, nil)
	if err != nil && release != nil {
		release()
	}
	return selection, err
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
		// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐鍛傜喎鈻庨幆褎顔勭紓鍌欒兌婵挳鎮樺璺何﹂柛鏇ㄥ枤閻も偓闂佸湱鍋撻幆灞轿涢妶鍥╃＝濞达絾褰冩禍鐐節閻㈤潧孝婵炶绠撻幃锟犲礃椤忓懎鏋戝┑鐘诧工閻楀棛绮堥崼鐔稿弿婵☆垰娼￠崫铏光偓瑙勬礀瀵墎鎹㈠☉銏犵闁绘劕鐏氶崳褏绱撴担绋款暢闁稿鍊濋獮鍐ㄎ旈崨顔芥珳闁硅偐琛ラ埀顒冨皺閺佹牗淇婇悙顏勨偓褏绱撳璺虹闁规儼妫勭粻?TokenProvider 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氶梺璇叉唉椤煤韫囨稑纾块柟鎯版閻掑灚銇勯幒鎴姛缂佸鏁婚弻娑㈡偐瀹曞洤鈷堟繝銏ｎ潐濞茬喎鐣风粙璇炬梹鎷呴崫鍕闂傚倷娴囬鏍垂閾忣偅娅犳俊銈呮儰婢跺绶為柟閭﹀墰椤旀劙姊洪崫鍕垫Ч闁糕晛锕悡顒勵敆閸曨剛鍘搁柣蹇曞仩椤曆囧焵椤掍胶绠炴鐐诧工閳规垹鈧綆浜為鐓庮渻閵堝棙顥嗛柛瀣姈閺呭爼顢涘锝嗘杸闂佺粯鍔樼亸娆撳箺閻樼數纾兼い鏃囧亹閻忚京绱掓潏鈺佷沪缂佺粯绻堝畷鎯邦樁闁硅姤娲栭埞鎴︽倷閺夋垹浠ч梺鎼炲妿閹虫捇寮鈧缁樼瑹閳ь剙顭囪閹广垽宕卞☉妯碱槶濠电娀娼ч鍡涘磻閻斿吋鐓涚€广儱楠告禍鐐电棯閹佸仮闁哄瞼鍠栭、娑㈠幢濡も偓閺嗙喓绱掔€ｎ亞绠崇紒杈ㄦ尰缁楃喖宕惰閻忓秹姊洪懡銈呮毐闁哄懏鐩、姘舵晲閸℃瑧鐦堝┑顔斤供閸樺吋绂嶅鍫熲拺缂備焦蓱鐏忣厽绻涚€电鍘撮柟顔惧亾閵堬綁宕橀埡浣风敾婵犵數濮撮敃銈夊疮閹殿喚涓嶅┑鐘崇閻撴稑霉閿濆懏鎲哥€涙繂顪冮妶鍡樺碍闁告艾顑呴銉╁礋椤撴稑浜鹃柨婵嗛婢ь喗顨ラ悙鑼ф慨濠勭帛閹峰懏顦版惔婵婎洬缂傚倷娴囧鎾跺垝濞嗘挸绠犻柣鏃傗拡閺佸秹鏌ｉ幇顖氱毢闁告ɑ妞藉娲礈閹绘帊绨撮梺绋垮閻擄繝骞?token
		if s.openAITokenProvider != nil {
			accessToken, err := s.openAITokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return "", "", err
			}
			return accessToken, "oauth", nil
		}
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鈽夊Ο閿嬫杸闂佸憡娲﹂崑鍕叏婢舵劖鈷戠紒瀣儥閸庢劙鏌熼崨濠冨€愰柨婵堝仜閳规垹鈧綆鍋勬禍妤呮煙閼测晞藟婵℃彃鎳庨…鍥礈瑜忕壕浠嬫煕鐏炲墽鎳呴悹鎰嵆閺屾盯鎮╅幇浣圭暥闁捐崵鍋ら弻娑㈠箛闂堟稒鐏堢紓浣哄Х婵數鎹㈠┑鍥╃瘈闁稿本纰嶅▓婊堟⒑閸濆嫷鍎愭俊顐㈠暣瀵鎮㈤搹鍦厯闂佸壊鐓堥崰姘ｆ导瀛樷拺闁告縿鍎遍弸搴ｇ磼婢跺本鍤€妞ゎ偄绻掔槐鎺懳熺拠宸偓鎾绘煟閻斿摜鎳冮悗姘煎墯缁傛帡鍩￠崨顔规嫼缂備礁顑呴悘婵嬵敆閵忊€茬箚闁绘劘鍩栭ˉ澶愭煟閿濆懎妲绘い顓滃姂瀹曟﹢鎮欓鈧崗濠囨⒑閼姐倕鈻堢紓鍌涜壘閳诲秹鏁愭径濠呮憰閻熸粌绉归垾鏃堝礃椤斿槈褔鏌涢埄鍐炬畼闁荤喐鍔欏铏圭磼濡椽鍤嬬紓浣哄У閹告悂鎮鹃悜鑺ユ櫜闁割偁鍨婚弶鎼佹⒑閸濆嫬鏆欓柛濠傤煼閹線宕奸妷锔规嫽闂佺鏈銊︽櫠濞戞氨纾奸悗锝庡亜濞搭噣鎸婇悢鍏肩厽闁归偊鍓﹂崵鐔虹磼閳锯偓閸嬫捇姊绘笟鈧埀顒傚仜閼活垱鏅堕幘顔界厵妞ゆ梻鐡斿▓婊呪偓娈垮枔閸旀垵鐣烽敓鐘冲仩鐎瑰嫮顢噕ider 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝曢梻浣藉Г閿氭い锔诲枤缁辨棃寮撮悢铏圭槇闂佹眹鍨藉褍鐡梻浣瑰濞插繘宕愬Δ鍛劦妞ゆ帊绀侀崵顒勬煕閿濆繒绉┑鈩冩尦瀹曘劑寮堕幋鐘靛幀濠电姰鍨煎▔娑㈡偋閸涱垵濮冲┑鐘崇椤ュ﹥銇勯幇鈺佺仾濠㈣泛瀚伴弻鐔煎传閵夈儰澹曢梺鍝勬噹閻栫厧顕ｉ幘顔碱潊闁挎稑瀚獮鍫ユ⒑鐠囪尙绠抽柛瀣█椤㈡俺顦抽柟渚垮姂瀹曟帡鎮欑€电骞愰梺璇茬箳閸嬬喖宕戦幘鍓佺焼閻庯綆鍏橀崑鎾舵喆閸曨剛顦梺鍛婎焼閸パ呭幋闂佺鎻粻鎴︽倷婵犲嫭鍠愰煫鍥ㄧ⊕閸嬫﹢鏌嶉崫鍕殶缁炬崘妫勯湁闁挎繂娲﹂崵鈧梺鍛婃煟閸婃洟鈥﹂懗顖ｆШ缂備緡鍠楅悷銉╂偩閻戣棄绠ｉ柨鏇楀亾缂佺姴顭烽幃褰掑炊瑜嶅Λ姗€鏌ㄩ弴鐐测偓褰掓偂閺囥垺鐓涢柛銉ｅ劚婵″ジ鏌ｈ箛鏇炐ョ紒杈ㄥ浮椤㈡洟濮€閳跺灕鍥ㄧ厸閻忕偛澧藉ú鏉戔攽閿涘嫬鍘存い銏＄懇瀹曪綁濡烽妸锔烩偓妤呮⒒閸屾瑧顦︽繝鈧柆宥呯；闁瑰墽绮崑鈺傘亜韫囨挻鍣芥い鈺佸级缁绘繃绻濋崒婊冾杸闂佸搫鎷嬮崢濂稿煘閹达附鍋愰悗鍦Т椤ユ繈姊虹紒妯哄闁挎洦浜濠氬即閻旇櫣顔曢悷婊冪Ч瀹曨垶顢曢妶鍥︾瑝濠电偛妯婃禍婵嬪煕閹烘鐓曢悘鐐插⒔閹冲懏銇勯敂鑲╃暤闁哄矉缍佹俊鍫曞川椤旇姤鍊锋俊鐐€愰弲娑㈠Χ閸涘﹣绻嗛柟闂寸閻撴盯鏌涚仦鍓ь暡闁伙綁浜跺缁樻媴缁嬫妫岄梺绋款儏閹冲海鍙呴梺鍝勭▉閸嬪棙绋夊鍡欑鐎瑰壊鍠曠花濂告煟閹捐泛鏋戝ǎ鍥э躬椤㈡稑鈹戦崱鏇熺潖闂佹眹鍩勯崹閬嶆儎椤栫偛钃熸繛鎴炲焹閸嬫捇鏁愰崘銊ヮ瀳婵犵鈧尙鐭欓柡灞炬礋瀹曟儼顦叉い蹇ｅ幘閳ь剚顔栭崰鏇犲垝濞嗘劒绻嗛柟闂寸劍閺呮粓鏌ｉ幇闈涘⒒闁告艾娲濠氬磼濞嗘埈妲┑鐘亾闂侇剙绉撮悿鐐亜閹板墎绋婚柛娆忕箰椤啰鈧綆浜滈銏°亜椤愶絾绀嬮柡宀€鍠栭幃婊冾潨閸℃鏆ョ紓浣哄亾閸庢娊宕ョ€ｎ剚宕叉繛鎴欏灩缁狅綁鏌ｉ幋婵囶棤闁告梻鍏樺娲嚒閵堝懏鐎鹃梺缁樻尨閳ь剚鍓氬鏍煣韫囨凹娼愰悗姘哺閺屽秹濡烽妷褝绱炴繛瀵稿閸樺ジ鍩為幋锕€鐓￠柛鈩冦仦缁ㄥ姊洪崫銉ユ珡闁搞劏浜划姘綇閵娧呯槇闂佹悶鍎滈崶褎鏆梻鍌欑閹碱偄煤閵娾晛纾婚柣鎰惈妗呭┑鐘绘涧濞层劎绮绘ィ鍐ㄧ骇闁割偅绻傞埛鏃堟煕閹烘挻绶叉い顓″劵椤у倻绱撳鍕獢妤犵偛鍟存慨鈧柕鍫濇噽椤ρ囨煟韫囨挾绠查柣妤侇殘閳ь剚淇哄Λ鍕煘閹达附鍊烽柤鎼佹涧濞懷呯磽娴ｈ棄绱︾紒顔界懇閻涱喗寰勯幇顓熸闂佺粯顭堢亸娆撳蓟閸儲鈷戠紓浣姑慨澶愭煕鎼存稑鈧繈骞冮敓鐘冲亜闁稿繗鍋愰崢浠嬫⒑閸濆嫬鏆婇柛瀣尵缁辨帞鎷犻幓鎺嗗亾閸濄儱寮查梻渚€娼ч悧鍡浰囬幍顔瑰亾濮橆剚鍤囬柡宀嬬秮瀵剙鈻庨悙顒傛瀮闂備胶绮弻銊ノ涘┑瀣摕闁靛牆顦粻鐟懊归敐鍫綈闂佹鍘界换婵嗏枔閸喗鐝紓浣哄У閹瑰洤顕ｆ繝姘労闁告劏鏅涢鎾绘⒑閸涘﹥灏扮憸鏉垮暣瀹曟粍瀵肩€涙ǚ鎷婚梺鎼炲劀鐏為敮鏋呴梻浣告惈閺堫剛绮欓弽顐や笉婵炴垯鍨圭粻濠氭煣韫囷絽浜滈柍褜鍓﹂崹宕囨閹捐纾兼慨妯荤樂閿濆鐓曢柕濞垮劜閸嬨儳鈧?
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

func (s *OpenAIGatewayService) handleFailoverSideEffects(ctx context.Context, resp *http.Response, account *Account, requestedModel ...string) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if len(requestedModel) > 0 {
		s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body, requestedModel[0])
		return
	}
	s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
}

// Forward forwards request to OpenAI API
func (s *OpenAIGatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	restrictionResult := s.detectCodexClientRestriction(c, account)
	apiKeyID := getAPIKeyIDFromContext(c)
	logCodexCLIOnlyDetection(ctx, c, account, apiKeyID, restrictionResult, body)
	if restrictionResult.Enabled && !restrictionResult.Matched {
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
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
	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€跨€殿喖顭烽幃銏ゆ惞閸︻叏绱查梻渚€娼х换鎺撴叏閻㈠憡鍊堕柛顐犲灮绾捐棄霉閿濆懏鎯堥崯鍛婄節閻㈤潧浜归柛瀣尭铻栭柣姗€娼ф禒锕傛煟濡や焦绀夌憸棰佺椤啴濡堕崱妤€娼戦梺绋款儐閹告悂鍩為幋锕€鐏抽柣鎰娴狀噣姊洪崫鍕拱缂佸鎸荤粋鎺楁晜閻愵剙鐝伴梺鍦帛鐢晛顭囨惔銊︹拻濞达絽鎲￠幆鍫ユ煟椤掆偓閵堢鐣锋导鏉戠閻犲洩灏欓崝锕€顪冮妶鍡楃瑨闁稿﹤缍婂鎶藉煛閸屾ü绨诲銈嗘尵閸嬬喐鏅跺☉銏＄厽婵°倐鍋撳Δ鐘虫倐閸?WS 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆浂鐒炬い銉ョ箻閺屾稓鈧絺鏅濈粣鏃傗偓瑙勬礃濞叉ê顭囪箛娑樼厸濞撴艾娲﹂ˉ锟犳⒒娴ｄ警娼掗柛鏇炵仛閻ｇ兘姊虹紒妯诲鞍闁荤啿鏅犲濠氭偄绾拌鲸鏅┑鐐村灦閻熝囧礄閿熺姵鍋犳慨姗嗗幖閸濈儤鎱ㄦ繝鍛仩缂侇喗鐟ラ埢搴ㄥ箚瑜嶆竟瀣⒒娴ｅ憡鎲稿┑顔炬暬瀹曟繂鈻庨幇顏嗙畾闂佹眹鍨婚…鍫㈢矆鐎ｎ偁浜滈柡宥冨姀婢规﹢鏌涢悙顏勫婵﹥妞藉Λ鍐归妶鍡欐创鐎规洏鍔戦、娑橆潩椤掑倻鎳嗛梻鍌氬€搁崐宄懊归崶顒夋晪鐟滃繘鍩€椤掍胶鈻撻柡鍛箞瀹曠増绻濋崒銈呮倯闂佸憡渚楅崹鍗炩枔閺囩姷纾藉ù锝呭閸庢劖銇勯幋鐐垫噧妞ゎ厼娲浠嬪Ω瑜忛鏇㈡倵閻熸澘顥忛柛鐘虫礈閼鸿鲸绻濆顓犲幈闂侀潧顭堥崕閬嶅焵椤掆偓缂嶅﹥淇婇悽绋跨妞ゆ牗鑹惧畵鍡涙⒑缂佹ê濮堟繛鍏肩懃閳诲秹鏁愭径瀣ф嫼闂佸憡绋戦敃銉т焊椤撱垺鐓曢柣鏃堟敱閸嬨儵鏌涢埡鍐ㄤ槐妤犵偛顑夐弫鍌炴寠婢跺鐫忓┑锛勫亼閸婃牠宕濋敃鈧…鍧楀焵椤掆偓椤法鎲撮崟顒傤槹濠殿喖锕︾划顖炲箯閸涙潙宸濆┑鐘叉噽椤㈠懘姊绘担鐟板姢缂佺粯鍔欓弻濠囨晲閸涱垱娈鹃梺鍝勵槼椤曟娊寮崼婵嗙獩濡炪倖鎸炬刊瀵告?WS 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼达絾鏆╂い顓炵墛娣囧﹥绂掔€ｎ偀鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜閻犲洩灏欐晶锔锯偓娈垮枛椤嘲顕ｉ幘顔藉亜濡炲娴烽悰顕€姊绘担铏广€婇柛鎾寸箚閹筋偊姊虹紒妯肩畺婵炶尙鍠庨～蹇涙惞閸︻厾鐓撳┑鐐叉閸庢娊宕滈弶娆炬富闁靛牆绻愰々顒勬煛娴ｇ瓔鍤欐い鏇悼閹风姴霉鐎ｎ偒娼旈梻渚€娼х换鍡涘疾濠婂牆鐤炬繝闈涱儐閳锋垿鏌熺粙鎸庢崳缂佺姵鎸绘穱濠囶敃閿濆洦鍒涢柦妯荤箞閺屾洘绻涢悙顒佺彆闂佹娊鏀遍崹鍧楀蓟濞戞ǚ鏀介柛鈩冾殢娴犵偓绻濆▓鍨仩闁靛牊鎮傚璇测槈閵忕姷顔掗柣搴ㄦ涧閹诧繝宕氬☉銏♀拺缂侇垱娲橀弶褰掓煕鐎ｎ偅灏い顏勫暣婵″爼宕卞Δ鈧鎴︽⒑缁嬫鍎愰柟鐟版喘瀵鈽夊Ο婊呭枛閹筹繝濡堕崶鈺佸辅闂佽姘﹂～澶娒哄鈧畷褰掑锤濡ゅ啫绁﹀┑鈽嗗灥閸嬫劗澹曢崗闂寸箚妞ゆ牗绮岀敮鑸殿殽閻愯尙澧︽慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵犵數鍋涢惇浼村垂閽樺鏆︽繝闈涚墐閸嬫捇鏁愭惔鈥斥拤婵犳鍠楁繛濠囧蓟閿濆鏅查柛娑卞灣娴煎洨绱掗悙顒€鍔ら柕鍫㈩焾椤曪綁宕奸弴鐐哄敹濠电偞鍨堕敋妞ゎ剙鐗撳娲川婵犲嫮鐣奸梺绋跨昂閸婃繈鐛崼銉ノ╅柨鏂垮⒔閻﹀牓姊洪崨濠佺繁闁革綆鍠楃粋鎺楀煛閸愵亞锛濇繛鎾磋壘濞层倝寮搁悢鍏肩厽闁绘梹绻傚ú銈囩不閺屻儲鐓曢柡鍥ュ妼閻忕娀骞嗛悢鍏煎仭婵犲﹤瀚惌鎺斺偓瑙勬处閸撶喎鐣峰鍕闁惧繒娅㈢槐鎶芥⒒娴ｄ警鐒鹃柡鍫墴閹柉顦归挊婵嬫煥閺傛娼熷ù婊勭矒閺屻劑寮捄銊よ檸閻庤鎸稿Λ娆撱€冮妷鈺傚€烽柡澶嬪灥椤帡鎮楃憴鍕闁挎洏鍨介獮濠囨偐濞茬粯鏅㈡繛杈剧秬椤曟牠宕惔銊︹拻闁稿本鐟ч崝宥夋煟椤忓嫮绉虹€规洖婀遍幉鎾礋椤撶姴楠勫┑鐘垫暩婵潙煤閵堝鍑犲〒姘ｅ亾闁哄本鐩俊鐑筋敊閻撳寒娼介梻浣虹帛閹稿摜鎹㈠鈧幃锟狀敃閿曗偓閻愬﹥銇勯濠勫埌闁绘鎸搁悾鐑藉箣閿曗偓缁犲鎮归崶顏勭毢妞は佸洦鈷戦柛娑橈功閵嗘帒顭胯椤ㄥ﹪銆?HTTP -> WS 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇顓滀簻闁哄啫鍊归崵鈧繛瀛樼矒缁犳牠寮诲☉銏犵疀闂傚牊绋掗悘鍫ユ倵閻熺増鍟炵紒璇插暣婵＄敻宕熼姘敤闂侀潧臎閳ь剙危閸儲鐓欓柛蹇撳悑缂嶆垿鏌ㄩ弴妯衡偓婵嬪春閻愬搫绠ｉ柣鎰皺閻ゅ洭姊绘担渚劸闁挎洏鍎辩叅闁绘棁鍋愬畵渚€鐓崶銊︾５闁稿鎹囬弫鎰償閳╁啰浜梻浣告惈椤戝嫮绮堟笟鈧崺鐐哄箣閿旇棄浜归梺鍛婄懃椤︻垶藝閳哄懏鈷戠紓浣股戠亸鐗堢箾閸欏鐒介柟骞垮灩閳藉濮€閻樻鍚呮繝鐢靛█濞佳兾涘鍫濇槬闁绘劗鍎ら悡鐔兼煏韫囧﹥娅呴柣蹇曞█濮婅櫣鏁鍓滈梺缁樹緱閸ｏ絽顕ｆ禒瀣р偓鏍Ψ閵夆晛寮板銈冨灪椤ㄥ﹪宕洪埀顒併亜閹烘垵顏柛濠傛健閺岋綁骞嬮悘鏄忓亹閹广垽宕卞☉娆戝幍缂傚倷鐒﹂敋缂佹う鍛＜濠㈣泛鑻崝瀣亜閵婏絽鍔﹂柟顔界懅閳ь剛鏁搁…鍫ュ焻瑜版帗鐓犻柟绋块婵牓鏌曢崶褍顏鐐村浮瀹曞崬顪冮幆褜妫滈梻鍌氬€峰ù鍥敋瑜嶉湁婵﹩鍘鹃々鍙夌節闂堟稓澧涚€规洖寮剁换娑㈠箣濞嗗繒浠奸梻浣稿船濞诧妇鎹㈠☉銏犵闁绘垵妫涢崝閿嬬箾鐎电甯堕柣妤€妫濋崺?
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
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞鐟滃繘寮抽敃鍌涚厽闁靛繈鍩勯悞鍓х磼閹邦収娈滈柡宀€鍠栭弻鍥晝閳ь剟寮稿☉銏＄厱闁靛濡囩粻鐐搭殽閻愬澧懣鎰亜閹哄棗浜鹃梺璇叉禋娴滄繄鎹㈠☉銏犲窛妞ゆ牗顕撮敐鍚冲酣宕惰闊剙鈹戦垾宕囧煟鐎规洏鍔戦、娆撳礂绾板彉鍝楃紓鍌氬€搁崐鐑芥嚄閼稿灚鍙忛柟鎯版绾偓闂佽鍎煎Λ鍕嫅閻斿摜绠鹃柟瀵稿€戝顑╋綁宕奸悢绋垮伎濠德板€愰崑鎾绘煟濡も偓閿曨亜顕ｉ锕€绠氱憸澶愬绩娴犲鐓熼柟閭﹀墮缁狙囨煕閿濆牊顥夐柍瑙勫灴閹瑩鍩℃担宄邦棜闂傚倸鍊风粈渚€骞栭锔藉剹濠㈣泛鑻欢銈夋煕婵犲嫬鏋斿ù婊€绮欏缁樻媴閻熸澘濮㈢紓浣虹帛鐢帡鍩㈠澶婎潊闁靛繆妲呭鐔兼⒑缂佹ê濮囨俊顖氾工閵嗘帗绻濆顓犲帾闂佸壊鍋呯换鍌炲焵椤掑倹鍤€闁宠绉瑰畷鍫曨敆娴ｅ搫骞堥梻濠庡亜濞诧箑螞閹达附鍤€閻犳亽鍔夐崑鎾斥枔閸喗鐏曞銈嗘肠閸パ呭弨婵犮垼娉涜癌闁绘柨鍚嬮悡銉╂倵閿濆懐浠涚紓宥呯焸濮婂宕掑▎鎴М闂佺顕滅换婵嬪Υ閸愵喖閱囬柕澶堝劤閿涙盯姊虹紒妯哄闁稿簺鍊濆畷鎰板礈娴ｆ彃浜炬鐐茬仢閸旀碍銇勯敂鍨祮闁糕晜鐩獮瀣晜閻ｅ苯骞堟繝鐢靛█濞佳兾涘Δ鍜佹晜妞ゅ繐鎳屾禍婊堟煏婢诡垰鍟╅幋椋庣磽娴ｄ粙鍝洪悽顖ょ節閵嗕礁鈻庨幘鏉戞疅闂侀潧顦崕鎶芥晬閹剧粯鐓?WSv2闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ礂閻撳簶鍋撶紒妯诲弿婵°倐鍋撴俊顐ｇ懇閹箖鎮滈懞銉㈡嫼缂備礁顑呴悘婵嬵敆閵徛颁簻闁靛鍎婚煬顒傗偓娈垮枦椤曆囧煡婢舵劕顫呴柣妯诲絻閺併倖淇婇悙顏勨偓鏍礉瑜忕槐鐐哄幢濞戞袝? 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鎹愭硾鍟搁梺鍛婏供閸ㄨ泛顫忕紒妯诲闁告稑锕ㄧ涵鈧梻浣侯焾缁ㄦ椽宕愬┑瀣ラ柛鎰靛枛瀹告繈鏌℃径瀣仴闁诲寒鍙冨铏圭矙閹稿孩鎷辩紓浣割儐閹告儳顕ｈ閸┾偓妞ゆ帒瀚埛鎺楁煕鐏炵偓鐨戝褎绋戦妴鎺戭潩椤撗勭杹閻庤娲樺ú鐔肩嵁閸ヮ剚鍋嬮柛顐犲灩楠炲牓姊绘笟鈧褔鎮ч崱娑樼疇闁归偊鍘藉▍鐘绘煥閺囩偛鈧綊鎮″☉姘ｅ亾閸忓浜鹃梺閫炲苯澧寸€规洑鍗抽獮鍥敆婢跺苯濮洪梻濠庡亜濞诧妇绮欓幋鐘电焼闁割偁鍨洪崰鎰扮叓閸ャ劎鈽夋慨瑙勭叀閺岋絽螣閸忓吋姣勭紒鐐礃濡嫰濡撮幒鎴僵妞ゆ帊鐒﹂幃娆愮箾鐎涙鐭嬬紒顔兼捣濡叉劙骞掗幘宕囩獮婵犵數濮寸€氼喗顨欓梺璇插椤旀牠宕板☉銏╂晪鐟滄棃宕洪妷锕€绶為柟閭﹀墻濞煎﹪姊虹紒妯曟垿顢欓弽顓炵柈閻庯綆鍠楅埛鎺懨归敐鍫燁棄濞存粌缍婇弻宥呯暋閹殿喖鈪甸悗瑙勬礃缁矂鍩㈡惔銊ョ疀妞ゅ繐妫涢悾楣冩⒒娴ｉ涓茬紒韫矙閹鎳栭埡渚囨闂佽姤锚椤ャ垼銇愰幒鎾存珳婵犮垼娉涢惉鑲╃玻濡も偓閳规垿鏁嶉崟顐″闂佽鍠栭崐鍨嚕鐠囨祴妲堥柕蹇曞閳哄懏鐓忓璇″灠閸熶即宕㈡禒瀣厽閹兼番鍩勯崯蹇涙煕閻樺磭澧甸柍銉畵閹粓鎸婃径宀€鏆梻渚€娼х换鍫ュ磹閺嶎厼纾婚柛宀€鍋涚粻褰掑级閸繂鈷旈柟顔笺偢閺岋繝宕遍幇顒備紙濠殿喖锕︾划顖炲箯閸涙潙宸濆┑鐘叉噽椤㈠懘姊绘担渚劸缂併劍妞藉畷鎰攽閸℃瑦娈鹃悷婊呭鐢帡宕归崒娑栦簻闁规澘澧庨崚鎵偓娈垮枛濞尖€愁潖濞差亝顥堟繛鎴炶壘椤ｅ搫鈹戦悙鑼勾闁告柨瀛╃粩鐔煎即閵忊€崇檮婵犮垼顫夌换鍌滅礊婵犲洤鏋侀柟鐗堟緲闁卞洭鏌ｉ弴姘樂闁告繃顨婂鍝勑ч崶褏鍔撮梺鎼炲姂娴滃爼骞嗛崟顖ｆ晬闁绘劘灏幗鏇炩攽閻愭潙鐏熼柛銊潐閸庮偊姊绘笟鈧褔鏁嶈箛娑樼＜婵犻潧娴傞埀顒侇殜濮婂宕掑▎鎰垫▊缂備線纭搁崰姘嚗婵犲洤閿ゆ俊銈傚亾缂佲偓閸屾稐绻嗘い鏍ㄧ箖椤忕娀鏌￠崱顓犵暠闁宠鍨垮畷鎺戭煥鎼达絽濮奸梻浣告啞椤ㄥ棙绻涙繝鍥ц摕闁挎繂顦Λ姗€鎮归幁鎺戝妞ゆ柨鐗撳铏圭矙濞嗘儳鍓遍梺鍦嚀濞差參鐛崘顭戠叆闁割偆鍠庡▓鐔兼⒑闂堟冻绱￠柛鎰╁妿椤旀垿姊婚崒姘偓鐑芥嚄閸洖绠犻柟鐐た閺佸銇勯幘鍗炵仼缁炬儳缍婇弻鈥愁吋鎼粹€崇缂備胶濯寸徊鍓ф崲濠靛顥堟繛鎴炆戝▓鏌ユ⒑鐠囨彃顒㈡俊鐐舵椤繐煤椤忓懎浠梺鍝勵槸缁ㄩ亶骞愰崘顏嗙＝濞撴艾娲ゅ▍姗€鏌涢妸銉у煟妤犵偛鍟撮弫鎾绘偐閸愯弓绨婚梻浣侯焾缁绘宕戦幇鏉垮偍闁芥ê顦弨浠嬫煃閽樺顥滈柣蹇曞█閺岀喓鍠婇崡鐐扮敖闂佹眹鍎烘禍锝壦囩€靛摜纾奸弶鍫涘妼缁楁岸鏌熷畡鐗堝殗鐎规洩绲惧鍕偓锝庡墮缂嶅啴姊婚崒娆戭槮闁圭⒈鍋婂鐢割敆閸曨剙鍓銈嗙墬缁嬫劗鎹㈤崱妯诲弿婵犻潧妫涢悞鐐繆閼碱剙甯堕柍瑙勫灴閹瑩宕ｆ径濠冾仭婵犵數鍋涘鎯版懌缂備礁鐭佹ご鍝ユ崲濠靛鐐婄憸蹇涙偂閹寸偟绡€闁靛骏绲剧涵楣冩煥閺囨ê鈧繈鏁愰悙鑼殕闁告洦鍏橀幏娲煟鎼粹剝璐″┑顖ｅ幖椤洭骞囬悧鍫㈠幈闂侀潧顭堥崕宕囩矓濞差亝鐓曢柍瑙勫劤娴滅偓淇婇悙顏勨偓鏍暜閹烘鏅濋柨鏂垮⒔閻捇鏌ｉ姀鐘冲暈闁绘挻娲熼幃妤呮晲鎼存繄鍑瑰銈冨劘閸ㄥ鍩€椤掑喚娼愭繛鍙夘焽閹广垽宕煎┑鍫熸婵犻潧鍊婚…鍫ュ础閹惰姤鐓熼柡鍌氱仢缁狙囨倵濮橆兙鍋㈡慨濠勭帛閹峰懘宕ㄦ繝鍌涙畼闂備礁缍婇埀顒佺閸忓苯霉濠婂拋鐒鹃柍璇查叄楠炴﹢宕橀幓鎺嶆喚濠电姷鏁搁崑鐐哄垂椤栫偛鍨傛繛宸簼閸嬪倿鏌￠崶銉ョ仾闁绘挾鍠愭穱濠囶敍濠靛浂浠╅梺鎸庣☉缁夋挳鈥﹂懗顖ｆ▌婵炲瓨绮岄悥鐓庮嚕婵犳艾鐒洪柛鎰典簽閹虫繈姊洪幖鐐插妧闁告洦鍘肩紞鍡椻攽鎺抽崐妤佹叏閻戣棄纾绘繛鎴欏灪閻撯偓闂佹寧绻傞ˇ顖滅不閻樼粯鐓熼柟閭﹀墯閹牓宕鐐村仭婵犲﹤鍟版晥闂佺粯渚楅崳锝呯暦婵傜唯闁挎棁顫夌€氳棄鈹戦悙鑸靛涧缂佽尪鍋愰幏鍐晝閸屾氨鍝楁繛瀵稿帶閻°劑鎮″▎鎰闁煎ジ顤傞崬铏圭磼閵娧勬珪闁逞屽墮閻忔艾顭垮鈧弫鍐Ψ閳轰絼褔鏌ㄥ┑鍡╂Ц缂佲偓閸愵喗鐓熼柟浼存涧婢ь垱绻涢崨顓燁棦婵﹦绮幏鍛存惞閻熸壆顐奸梺钘夊暣娴滃爼寮诲☉銏犵缁炬儳顑呴ˉ婵嬫⒑瀹曞洨甯涢柟姝屽吹缁骞掗弬鍝勪壕闁挎繂楠告禍鐐烘煕濡湱鐭欐慨濠囩細閵囨劙骞掗幋婊冩瀳闂備礁鎲￠幐濠氭偡瑜旈幆鈧い蹇撴祩濡嫰姊洪崫鍕拱闁烩晩鍨堕獮鍐煛閸愵亪鈹忕紒缁㈠幖閹冲宕戦幘璇查敜婵°倓鑳堕崢鎼佹煟韫囨洖浠滃褑妫勭叅闁哄鍨熼弨浠嬫煃閳轰礁鏆為柛濠冨姍閺屸剝鎷呯粙鎸庢闂佺硶鏂侀崑鎾愁渻閵堝棗绗掗悗姘煎墰缁牊寰勯幇顓犲帾闂佸壊鍋呯换鍐夐幘瓒佺懓顭ㄩ崟顓犵厑闂侀潧娲ょ€氫即鐛Ο灏栧亾闂堟稒鎲搁柣鎾亾婵犵數濮甸鏍窗閺嶎厼纾瑰┑鐘宠壘閻掑灚銇勯幒鎴濇灓婵炲吋鍔栫换娑㈠矗婢跺瞼鐓夐悗瑙勬磸閸ㄥジ藝瑜版帗鐓曢柍鍝勫暙娴犺鲸顨ラ悙宸剶闁轰礁鍟撮崺鈧い鎺戝瀹撲線鏌曡箛濠傚⒉闁告瑥绻戞穱濠囶敍濮橆叀纭€闂佸疇顫夌敮锟犲蓟濞戞埃鍋撻敐搴″濞寸姍鍛＜閺夊牄鍔嶅畷宀€鈧娲樼敮鎺楋綖濠靛鏁勯柦妯侯槷婢规洟姊鸿ぐ鎺擄紵缂佲偓娴ｅ搫顥氱憸鐗堝笚閻撴瑩姊婚崒姘煎殶妞わ讣濡囬惀顏堝箚瑜忕粔娲煛瀹€鈧崰鏍ь潖閼姐倐鍋撻棃娑橆棌婵″樊鍠氱槐鎾存媴閻熸澘顫嶅銈冨妼濡繈鐛崼銉ノ╅柕澶婃捣閸犳牠鐛幇顓熷劅闁挎繂鍟犻崑鎾诲箛椤斿墽锛濇繛杈剧悼閹虫挻鎱ㄩ崼鐔翠簻闁靛鍎婚煬顒傗偓娈垮枛椤兘宕规ィ鍐ㄧ疀濞达絽鎲￠崐顖炴⒑绾懎浜归悶娑栧劦閸┾偓妞ゆ巻鍋撻柛鐔绘硶缁濡烽妷銏℃杸闂佺粯鍔樼亸娆撳箺閻樼數纾兼い鏃囧亹閻忚京绱掓潏鈺佷沪缂佺粯绻堝畷鎯邦樁闁硅姤娲熷铏圭磼濡椿姊垮┑鐐叉嫅缁叉寧绔熼弴銏犵闂傚倸顕粻姘渻閵堝棛澧紒璇插暣婵℃挳宕橀鐣屽幈闂佸湱鍎ら崺濠囩叕椤掍焦鍙忓┑鐘插暞閵囨繃顨ラ悙瀵稿⒌闁诡喗鐟ラ湁閻庯綆浜欐竟鏇㈡⒑閸濆嫮鈻夐柣蹇斿姇鐓ゆい蹇撴噹娴滄粓姊虹紒妯诲碍婵﹫绠撻獮妤呭即閵忊€斥偓鍨殽閻愯尙浠㈤柛鏃€宀搁弻锝呂旀担铏圭厐濡炪値鍓欓敃顏呬繆閹间礁鐓涢柛灞绢殕鐎氬ジ姊绘担鍛婂暈缂佸鍨块弫鍐Ψ瑜岄悞濠囨煙濞堝灝鏋撻柛瀣崌瀹曟寰勬繝浣割棜闂傚倷绀佺紞濠偽涢崸妤佸殑閻犻缚娅ｉ弳銈呫€掑锝呬壕闂佺粯鎼╅崑濠傜暦閸洖惟鐟滃危閸儲鍊甸柣鐔哄閸熺偟绱掔拠鑼ⅵ鐎殿喖顭烽崺鍕礃閵娧呯嵁闂佽鍑界紞鍡樼閻愬顩烽柟缁㈠枟閳锋垿鏌涜箛鎾虫倯缂佽埖鐓￠弻锝堢疀閺冨倻鐤勯梺绯曟櫇閸嬨倝鐛€ｎ喗鏅滅紓浣股戝▍鎾绘⒒娴ｈ銇熼柛娆忛叄瀹曟垿骞掗幘棰濇锤濡炪倕绻愮€氼噣寮抽敃鍌涚厪濠㈣鍨崑鎾绘煕鐎ｎ偅宕岄柡浣瑰姈閹棃鍩勯崘顏冮偗濠电姷顣藉Σ鍛村磻閸涘瓨鏅濋柕蹇嬪€曢拑鐔衡偓骞垮劚閻楁粌顬婇妸鈺傗拺闁告稑锕ョ亸浼存煟閻斿弶娅婄€规洘妞介幃娆撳传閸曨収鍚呴梻浣瑰濡礁螞閸曨垰鐒垫い鎺戝€搁崢鎾煛鐏炵澧茬€垫澘瀚埀顒婄秵娴滅偞绂掗崗鑲╃閻庣數顭堟牎闂佽鍠栭崐鎼侊綖韫囨柣鍋婇悷浣靛€撳Ч妤呮⒑閻熺増鎯堟俊顐ｎ殕缁傚秹鎮欓鍌滎啎闂佺懓顕崑娑㈠吹椤掍椒绻嗙€瑰壊鍠栭弸娑欍亜椤忓嫬鏆ｅ┑鈥崇埣瀹曟帒鈽夊▎鎴濈到闂傚倷绶氬褔鎮ч崱娑樼疅闁斥晛鍟伴埞宥呪攽閻樻彃浜炴繛鍏煎閳ь剙鍘滈崑鎾斥攽閻樻彃鏁い鎺戝€荤壕浠嬫煕鐏炲墽鎳嗛柛蹇撹嫰閳规垿顢欓悙顒佹瘓閻庢鍠栭…閿嬩繆閹间礁鐓橀梺娆惧灠娴滈箖鏌ㄩ弮鍌氫壕鐎规洖顦甸弻鏇熺節韫囨洜鏆犵紒鍓ц檸閸ㄨ泛顫忛搹鍦＜婵☆垵娅ｆ禒鎼佹煢閸愵喕鎲鹃柡宀€鍠栭幃婊堝箣閹烘挸鏋ゆ繝娈垮枛閿曘劌鈻嶉敐鍥у灊婵炲棙鍨跺畷澶愭煏婵炑冭嫰閺佽偐绱撻崒姘偓椋庢閿熺姴绐楅柡宥庡亞閻捇鎮峰▎蹇擃仹缂佽妫濋弻锝夊閵忊晝鍔哥紓浣插亾閻庯綆鍋佹禍婊堟煛瀹ュ啫濡介柣銊﹀灴閺岋綁顢曢姀鐙€浼冨┑顔硷攻濡炰粙鐛弽顓熷€烽柟缁樺笒铻氶梻鍌欒兌椤牓寮甸鍌滅煓闁圭偓鍓氶悞浠嬫煛瀹ュ骸骞楅柍閿嬪浮閺屾稓浠﹂崜褎鍣銈忚缁犳捇寮婚悢鍝勬瀳闁告鍋樼花濠氭煣閼姐倕浠遍柡宀嬬節瀹曟﹢濡歌椤ｈ櫣绱掔紒銏犲箰闁稿鎹囧缁樻媴缁涘缍堥梺绋挎捣閸樠嗙亱闂佸憡娲﹂崹杈┾偓姘嚇閺岋綁寮崶銉㈠亾閳ь剟鏌涚€ｎ偅宕岄柣娑卞櫍瀹曞綊顢欓悡搴經濠碉紕鍋戦崐褏鈧潧鐭傚畷瑙勫閺夋垶鐎銈嗘磵閸嬫挻鎱ㄦ繝鍕笡闁瑰嘲鎳橀幃鐑芥焽閿曗偓濞堟繈姊绘担鍛婃儓閻炴凹鍋婂畷婵嗩吋婢跺﹦鏌ч梺鍓插亝濞叉﹢宕戦埡鍛€堕柣鎰祷濡惧憡绻涢懖鈺佹瀾缂佺粯绋撻埀顒傛暩椤牏鏁崼鏇熺厽閹烘娊宕濋幋锕€鏄?
	if wsDecision.Transport == OpenAIUpstreamTransportResponsesWebsocket {
		if c != nil {
			MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalFeatureGate)
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
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯闁靛／灞芥櫏闂備浇顕у锕傦綖婢跺⊕鍝勵潨閳ь剟骞冮敓鐘虫櫢闁绘灏欓悾鍝勨攽鎺抽崐鏇㈠箠韫囨稑鐤鹃柡灞诲劚缁犲湱绱掗鐓庡辅闁稿鎹囬獮鍥ㄦ媴鐟欏嫮銈板┑鐘殿暜缁辨洟宕戦幋锕€纾归柕鍫濐槸绾惧鏌涘☉鍗炲Ц婵炲樊浜濋崑鍕煟閹捐櫕鍞夐柟鑺ユ礀閳规垿鎮欓弶鎴犱户闂佺硶鏅涚€氭澘顕ｉ锕€绀冩い鏃傛櫕閸欏棗鈹戦悩缁樻锭婵☆偅鐟╄棢闁绘鍋ㄦ禍婊堟煥閺傛寧鎯堥柛鏂诲€楃槐鎺撴綇閵婏箑闉嶉梺鐟板槻閹虫﹢鐛幘璇茬鐎广儱鎷嬪Λ婊堟⒒閸屾艾鈧兘鎳楅崜浣瑰厹闁割偅娲栫粈鍫熺節闂堟稓澧㈤柣顓熺懇閺岀喖鎮滃鍡樼暦闂佺顑呴崐鍧楀蓟閿熺姴鐐婇柕澶堝劤娴犻箖姊虹紒妯诲碍闁哥喐鎸冲璇差吋閸℃ê顫￠梺瑙勵問閸ｎ喖危椤曗偓閹嘲顭ㄩ崼顐ｆ缂備浇鍩栧畝绋款嚕婵犳碍鍋勯柛娑橈功缁夊爼姊洪崨濠冨瘷闁告侗鍠楅蹇撯攽閻樻剚鍟忛柛鐘冲哺楠炲﹪骞橀幇浣告闂佸憡鎸烽悞锕€鐣烽崣澶岀瘈闂傚牊绋掑婵堢磼閳锯偓閸嬫捇姊绘担渚劸闁哄牜鍓涢崚鎺戠暆閸旇偐鍏橀崺鈧い鎺戝閳锋帒霉閿濆牊顏犻悽顖涚洴閺岀喎顫㈠畝濠傛闁绘挶鍊栨穱濠囶敍濮橆剚鍊悗瑙勬礀瀵墎鎹㈠☉銏犵婵炲棗绻掓禒鐓幬旈悩闈涗杭闁搞劎鍎ょ粚杈ㄧ節閸ヨ埖鏅┑顔筋焾娴滎剟顢欓崶顒佲拺闂傚牊绋撴晶娑㈡煙椤旂厧鈧悂顢氶敐鍡欑瘈婵﹩鍓涢崐鐐差渻閵堝棗绗掓い锔垮嵆瀵煡顢楅埀顒勫煘閹达附鍊烽柡澶嬪灩娴犵鈹戦瑙掓粓宕濋幋锔衡偓浣糕枎閹惧啿宓嗛梺闈涚箚濡狙囧箯濞差亝鐓熼柣鏂挎憸閹冲啴鎮楀鐓庡⒋鐎规洘绻堟俊鑸靛緞鐎ｎ剙骞堥梻浣虹帛濡啴寮ㄩ柆宥呯；闁靛鏅滈悡蹇涙煕閳╁喚娈旈柍褜鍓欏锟犳偘椤旇姤鍎熼柕濠忕畱濞堢喖姊洪棃娑崇础闁告劑鍔庨濂告⒒閸屾艾鈧嘲霉閸ヮ剙纾婚柟鐗堟緲閻撴洟鏌涢鐘插姎缂佲偓閸喓绠鹃柛鈩兩戠亸顏堟煃瑜滈崜娆撳磹閸︻厾绱﹀ù鐘差儏瀹告繂鈹戦悩鎻掝仼妤犵偛绉归弻锝嗘償閿濆棗娈岄柣搴㈠嚬閸犳寮茬捄浣曟棃宕ㄩ鐐村劒闂備焦鎮堕崕娲偂閸績鏋斿ù鐘差儐閻撶喖鏌熼柇锕€澧柍缁樻礋閺屾稒鎯旈姀鈾€鎷瑰銈冨妸閸庣敻骞冨▎鎾崇骇闁瑰鍋犻惂浣逛繆閵堝洤啸闁稿鍋ら弫鍐閵堝懐顔愰悷婊呭鐢晠寮崘顔界叆婵犻潧妫欓ˉ鐐淬亜閺傚灝鏆ｆ慨濠傤煼瀹曟帒顫濋钘変壕闁归棿绀佺壕褰掓煙闂傚顦︾痪鎯х秺閺岀喖姊荤€靛壊妲紒鐐劤缂嶅﹪寮婚敐澶婄闁挎繂鎲涢幘缁樼厱濠电姴鍊归崑銉╂煛鐏炶濮傜€殿喗娼欓～婵嬫嚋濞堝簱鍋撻弽銊х閻庢稒顭囬惌濠勭磽瀹ュ拑韬€殿喛顕ч埥澶愬閻樼數鏉搁梻浣侯焾缁绘劙骞楀鍫濈鐟滅増甯楅崐鐢告偡濞嗗繐顏紒鈧崘顔界厽闁瑰灝瀚弧鈧悗娈垮枛椤兘骞冮姀銈呭窛濠电姴瀚倴闂傚倷绀侀幉锟犲箰閸℃稑宸濇い鎰垫線閻㈣鈹戦悩鍨毄闁稿鐩幃娲Ω瑜嶉崹婵嬪箹濞ｎ剙濡肩紒鐘靛█閺岀喖骞戦幇闈涙閺夆晜绻堝娲捶椤撶偛濡洪梺绯曟櫅閻楀棝鈥﹂崶顒€鐓涢柛娑卞枤閸欏棗鈹戦悩缁樻锭婵☆偅鐟╁畷宕囩矙濞嗗墽鍞甸柣鐔哥懃鐎氼厾绮堢€ｎ喗鐓涚€光偓鐎ｎ剛蓱闂佽鍨卞Λ鍐╀繆閼稿灚鍎熼柕蹇嬪灮鍟告繝鐢靛Х閺佹悂宕戦悙鐢电焾闁圭虎鍠栭悿鐐節闂堟稓澧涙い顐ｆ礋閺岋綁骞囬鍌涙喖婵犳鍨遍幐鎶藉蓟濞戙垹绠婚柡澶嬪灩缁佸嘲鈹戦悙鑼闁告梹鍨甸～蹇涙惞閸︻厾鐓撻梺鍛婄墤閳ь剙鍟块～鐘绘⒒娓氣偓濞艰崵寰婃繝姘？闁告鍋涚粊顐⑩攽鎺抽崐褏寰婃禒瀣柈妞ゆ牜鍋涢悡鏇㈡煙鏉堥箖妾柣鎾存礋閺屻劑寮崶璺烘濡ょ姷鍋為〃鍡欐崲濞戙垹宸濇い鏃傛櫕椤︿即姊洪崫鍕拱闁烩晩鍨堕獮鍐煛閸涱厾顦ㄩ梺鍦帛鐢晛顭囬弴銏♀拻濞达絿鍎ら崵鈧紓浣哄Т缁夌懓鐣风涵鍛汗闁圭儤鍨归崣鈧┑鐘灱閸╂牠宕濋弴顫稏闁告稑鐡ㄩ悡鐔镐繆椤栨侗鍎ラ柛銈嗙懇閺屸剝鎷呴崫銉ㄥ┑顔硷功缁垳绮悢鐓庣劦妞ゆ巻鍋撴い顓炴穿椤︽挳鏌熼獮鍨伈鐎规洖宕埥澶娢熼崗鑲╁弰濠电姷鏁告慨鎾晝閵堝鐤柣妯款嚙閸戠娀鏌￠崘銊у闁绘挻鐟︾换娑㈠醇濠靛牅铏庨梺鍝勵儍閸婃鎯€椤忓牆绠查柟閭﹀弾濡嫰姊婚崶褜妯€闁哄被鍔岄埞鎴﹀幢濡ゅ﹣绱戦梻浣规た閸樹粙鎮烽埡鍛摕婵炴垶菤閺€钘夆攽閻樻彃鏋ゅù鐘櫇缁辨挻鎷呮禒瀣懙闁汇埄鍨抽崑銈夌嵁韫囨稒鍋愮紓浣姑▓銈咁渻閵堝棗鍧婇柛瀣崌閺屽秹顢旈崱妯绘瘓闂佽鍠楅〃鍛村煝閹捐鍨傛い鏃傛櫕娴滎亝淇婇悙顏勨偓銈夊储婵傜搴婇柡灞诲劜閸嬧晠鏌ｉ幋锝嗩棄缁绢厸鍋撻梻浣虹帛閸斿繘寮插鍫稏鐎广儱鎳夐弨浠嬫煃閽樺顥滈柣蹇嬪劜缁绘盯宕崘顏喰滈梺绯曟杹閸嬫挸顪冮妶鍡楃瑨妞わ富鍨堕悰顕€寮介妸锝勭盎濡炪倕绻愮€氼剟寮抽敐鍛斀闁炽儱纾崺锝団偓瑙勬礀瀹曨剝鐏掗梻浣哥仢椤戝懘顢斿ú顏呪拻闁稿本鐟чˇ锕傛煙鐠囇呯瘈闁炽儻绠撳畷鍗炩槈濡櫣鐛╂俊鐐€栧Λ渚€宕戦幇顑帡宕堕浣叉嫼闁荤姴娲﹁ぐ鍐吹鏉堚晝纾界€广儱鎳忛ˉ銏⑩偓瑙勬礃濠㈡鐏冮梺鍛婁緱閸橀箖宕㈤柆宥嗏拺闁荤喖鍋婇崵鐔封攽椤曗偓椤ユ挻绔熼弴銏″癄濠㈣绻傜紞濠囧极閹版澘妞藉ù锝呮贡缁嬫垶淇婇悙顏勨偓鏇犳崲閹扮増鍋嬪┑鐘叉搐绾惧綊鏌涢…鎴濇殠闁哄啫鐗婇弲鏌ユ煕濞戝崬骞楁繛鍫熺叀濮婅櫣绱掑Ο璇查瀺濠电偛寮堕…鍥箲閵忕姭妲堥柕蹇曞Т閼板灝鈹戞幊閸婃洜鈧凹鍓熼、鏃堫敇閻旇櫣顔曢柣搴㈢⊕椤洭鎯岄幒鏃傜＜闁绘ê纾晶顏呫亜椤愩垻绠崇紒杈ㄥ笒铻ｉ悹鍥ф▕閳ь剚鎹囬弻锝夋偄閸濄儲鍣ч柣搴㈠搸閸斿秶绮嬮幒鏇ㄦ▌濠殿喖锕ュ钘夌暦閵婏妇绠鹃柟鐑樻礀娴滃綊鏌涢幒鎾崇瑨闁宠姘︾粻娑㈠即閻斿壊鍟庨梻鍌欑閹碱偄煤閵忋倕鍨傞柛锔诲幘娑撳秵銇勯弬鍨挃缁炬儳鍚嬬换娑㈠箣閻戝洣绶甸梺绋垮閸旀瑩寮婚敓鐘叉そ濞达絿鍏橀崑妤€鈹戦纭锋敾婵＄偠妫勮灋闁告劑鍔夊Σ鍫ュ级閸稑澹夐柕蹇嬪€栭埛鎴︽煕濠靛棗顏柛锝堟缁辨帞鈧綆鍋呯亸鐢告煕閹烘挸娴€规洖銈搁崺妤呭煛娴ｅ嘲顥氶梻浣瑰缁诲倹顨ラ崨濠勵洸闁哄稁鐏愰悷閭︾叆闁告洦鍘鹃悡澶愭⒑閸濆嫮鐏遍柛鐘崇墵閻涱喖顓兼径瀣劒濡炪倖鍔х徊鑳亹閸涘瓨鈷掑ù锝囩摂閸ゆ瑧绱掔拠鍙夘棦鐎殿喗褰冮埥澶愬閿涘嫮鏆㈤梻鍌氬€烽懗鍓佸垝椤栫偛绠伴柟闂寸劍閻撯偓闂佹寧绻傞ˇ顖炴儗濡ゅ懏鐓曢柡鍥ュ妼婢ь垶鏌?Unmarshal闂?
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
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalFeatureGate)
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
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇顓滀簻闁哄啫鍊归崵鈧繛瀛樼矒缁犳牠寮诲☉銏犵疀闂傚牊绋掗悘鍫澪旈悩闈涗杭闁搞劎鍎ょ粚杈ㄧ節閸ャ劌鈧兘鎮楀☉娆樼劷妞わ负鍔岄埞鎴︽晬閸曨偂鍝楀┑鈽嗗亜鐎氫即銆佸鑸垫櫜闁糕剝鐟ù鍕煟鎼搭垳鍒伴柣蹇斿哺瀵煡鏁愭径瀣ф嫼缂傚倷鐒﹁摫妞ゃ儱妫欑换娑㈠箻閼搁潧娈岄柛妤呬憾閺岀喖骞嶉纰辨毉闂佺粯鎸鹃崰鏍蓟閿濆妫橀柟绋垮閸犳劙姊哄ú璇插箹闁挎洩濡囧Σ鎰板箳閹惧磭绐炴繝鐢靛Т鐎氼喗顨欓梺璇叉唉椤煤濡警娓婚柤鎭掑劜閸欏繘鏌嶈閸撶喖寮婚弴銏犻唶婵犻潧娲ゅ▍褍顪冮妶鍡樷拹閻㈩垽绻濆濠氭晲閸℃ê鍔呴梺闈涚箚閹冲洦鎱ㄩ崶顒佲拺闁圭娴烽。鏌ユ煏閸パ冾仾闁诡垱妫冩慨鈧柍銉ョ－閿涙挻绻濆▓鍨灈闁挎洏鍔岄埢宥夋晲閸パ冪亰濠电偛妫欓幐鍝ョ不娴兼潙绠归弶鍫濆⒔缁夊灚绻涢幓鎺斿煟婵﹦绮幏鍛喆閸曨偂鍝楅梻浣告啞閺屻劑顢栨径濠勬殾闁挎稑瀚ч崑鎾诲捶椤撶倫锝嗐亜閵夛絽鈧繂顫忕紒妯诲缂佹稑顑嗙紞鍫濐渻閵堝棙鈷愰柣妤冨Т閻ｇ兘寮跺▎鐐兊闂佸吋鎮傚褔宕滈悷鎵虫斀闁绘劕寮堕ˉ婊勭箾鐎涙ê鍝洪柟顔哄灲瀹曞崬鈽夊▎蹇庡寲闂佸搫顦遍崑鐐村垔娴犲鏁嗛柣鏂垮悑閻撴洟骞栧ǎ顒€鐏╁┑顔兼喘閺屽秶鎲撮崟顐や紝濡炪們鍨虹粙鎴﹀煡婢跺ň鏋庨煫鍥ㄦ尰閺侇亪姊婚崒娆戭槮闁圭⒈鍋婂鐢割敆閸屾粎鐓撻梺纭呮彧缁犳垿鎮￠垾鎰佺唵閻犺桨璀﹂悡顒佺箾缁楀搫濮傞柡灞剧洴椤㈡洟鏁愰崶鈺冩殸缂傚倷鐒﹂崬鑽ゅ緤閸撗勫床婵犻潧顑嗛崑銊╂⒒閸喓鈼ユ繛宀婁邯濮婃椽宕烽褏鍔稿銈庡幖閸㈡煡鎮惧畡鎵虫斀闁糕€崇箲閻庡妫呴銏＄カ缂佹彃澧界槐鐐存償閿濆洨锛濋梺绋挎湰閻熝囧礉瀹ュ棎浜滄い鎾跺仦閸犳﹢鏌熼鐐効闁靛洦鍔欓獮鎺楀箻鐠哄搫绠為梻鍌欑窔濞佳囁囨导鏉戝瀭闁汇垻顭堢壕鐟扳攽閻樺疇澹橀柡瀣╃窔閺屸€愁吋鎼粹€崇闂佹娊鏀辩敮鎺楁箒闂佹寧绻傞幊蹇涘疮閻愮儤鐓欓柤纰卞墻閻掗箖鏌嶇憴鍕伌妞ゃ垺宀搁崺鈧い鎺嗗亾妞ゎ厼娲╅ˇ鎾煕濞嗗繑鍤囨慨濠呮缁辨帒螣閻戔晜瀚介梻浣告啞椤棝宕ㄩ娆戠憹婵犳鍠楄摫濠⒀冮叄瀹曟垿骞樼拠鑼槯婵犮垼娉涢敃锝嗙珶閺囥垺鈷掑ù锝呮啞閸熺偞绻涚拠褏鐣电€规洘鍨垮畷鍗炩槈濞嗗繋鎮ｅ┑掳鍊х徊浠嬪疮椤栫偛纾婚柍鈺佸暟缁♀偓婵犵數濮撮崐鎼侇敂椤愶附鐓熸い鎾跺枎缁椦囨煃瑜滈崜婵嬶綖婢跺⊕娲偄婵傚鍔烽悷婊勫灴閹﹢宕橀瑙ｆ嫼闂佸憡绋戦敃锝囨闁秵鐓曢柣妯哄暱閸濊櫣鈧娲忛崹浠嬪蓟閸℃鍚嬮柛鈥崇箲鐎氬ジ姊绘担鍛婂暈缂佽鍊婚埀顒佸嚬閸樺ジ顢欒箛鎾斀閻庯綆鍋嗛崢閬嶆煟鎼搭垳绉甸柛瀣鐓ら柟缁㈠枟閻撳繘鏌涢埄鍐炬當闁逞屽墯閹倿骞冩ィ鍐╁€婚柦妯侯槺閸婄偤姊洪崘鍙夋儓闁哥姵绋撳Σ鎰板蓟閵夛腹鎷绘繛杈剧到閹芥粎绮斿ú顏呯厱閻庯綆浜烽煬顒傗偓瑙勬穿缁查箖骞嗛弮鍫濈參闁逞屽墴瀵悂寮埀顒傛崲濠靛洨绡€闁稿本绮岄·鈧梻浣虹帛閹稿鎯勯鐐茬畺婵せ鍋撻柟顔界懇楠炴捇骞掗崱妯虹槺闂傚倷绀侀悿鍥涢崟顖€鍥偨缁嬭儻鎽曢悗骞垮劚濞诧妇鈧碍宀搁弻锛勪沪鐠囨彃濮堕梺閫炲苯澧い銊ワ躬楠炴宕奸弴鐔告珳婵犮垼娉涢鍛椤撶偐鏀介柣鎰级椤ユ粎绱掔紒妯哄妞ゃ垺妫冮、鏃堝幢韫囨挷澹曢柣鐔哥懃鐎氼厾绮堟径瀣闁告侗鍠楃粈鍐磼瀹€鍐摵缂佺粯绻堝畷鍫曞Ω閵夈垹浜鹃梺鍨儍娴滄粓鐓崶銊﹀碍妞ゅ繈鍊濋弻娑氣偓锝庡亝瀹曞本顨ラ悙鍙夊枠鐎殿喖澧庨幑鍕Ω閿旀儳寮版繝纰夌磿閸嬫垿宕愰弴鐘冲床闁归偊鍎靛☉妯锋婵炲棙鍨熼弸鏍倵楠炲灝鍔氭い锔垮嵆閹€斥枎閹惧鍘介梺閫涘嵆濞佳勬櫠椤旂⒈鐔嗙憸婊堝垂閸洖钃熸繛鎴欏灩鍥撮梺绯曞墲椤洭藝閵娾晜鈷戦梻鍫熺⊕閹兼劙鎮楀顐㈠祮闁绘侗鍣ｅ畷鍫曨敆閳ь剟鎮″☉妯忓綊鏁愰崱妤冪シ婵炲瓨绮庨崑鎾寸┍婵犲洤围闁告侗鍘藉▓鍫曟⒑濞茶骞栨俊顐ｇ箞瀵鈽夐姀鐘插祮闂侀潧顭堥崕铏椤撱垺鈷戠紓浣股戠亸鐢告煕閻樺啿濮堢紒宀冮哺缁绘繈宕堕懜鍨珫婵犵數鍋為崹闈涚暦椤掑嫬绀夌€光偓閸曨兘鎷洪梺纭呭亹閸嬫盯鍩€椤掍胶澧甸柟顔惧厴閺佸倿鎮欓鈧惔濠傗攽閻愭潙鐏熼柛銊ョ秺閺屽宕堕浣哄幗闂佸搫鍟ú锕傤敂閻樼粯鐓熼柕鍫濇噹椤忊晠鏌嶇憴鍕伌妞ゃ垺鐟у☉鐢告煥鐎ｎ偅姣庡┑鐘殿暜缁辨洟宕戦悩鍙傛盯宕熼娑樹患闂佺粯鍨煎Λ鍕劔闂備焦瀵ч弻銊︽櫠娴犲鍎婇柛顐犲劜閳锋垿鏌涘☉姗堟敾濠㈣泛瀚伴弻娑㈠箻鐎靛憡鍣ч梺瀹狀嚙缁夌鐏冮梺鍛婁緱閸犳帡骞忓ú顏呪拺闁兼亽鍎嶉鍩跺洭顢涢悙鍙夎緢闂侀潧绻掓刊顓炪€掓繝姘厸濠㈣泛顑呴悘锕傛煙閸欏鍊愰柡宀€鍠栭悰顕€宕归鍙ユ偅闂備礁鐤囧Λ鍕囬崹顐ｅ弿闁逞屽墴閺岋絽螣濞茶鏅遍梺鍝ュ仒缁瑥顫忕紒妯诲闁告稑锕ㄧ涵鈧梻浣侯焾缁ㄦ椽宕愬┑瀣祦闁圭増婢樺婵嗏攽閻樻彃顏存繛鏉戝濮婃椽宕崟鍨ч梺鎼炲妿閸犳牕顕ｉ妸鈺傚殟闁靛绲肩花濠氭⒑閸濆嫬鏆欓柛濠傜埣閸┾偓妞ゆ巻鍋撴い鎴濇嚇椤㈡瑨绠涘☉妯溿劑鏌ㄩ弬鍨挃妞ゆ梹娲熼弻锝堢疀閹惧墎顔夐梺缁橆殕閸ㄥ灝鐣烽幋锕€绠荤€规洖娲﹀▓楣冩⒑閹肩偛鍔橀柛鏂跨Ч椤㈡瑩寮撮姀鈾€鎷绘繛杈剧到閹诧繝宕悙娣簻闁靛鍎虫晶锔锯偓瑙勬礃閸旀瑩寮幇顓炵窞閻庯綆浜欑花濠氭⒒娴ｈ櫣甯涢柛銊ョ埣閺佸顪冮妶鍛劉闁烩晩鍨堕獮鍐ㄎ旈崘鈺佹瀭闂佸憡娲﹂崜娑⑺囬妷銉㈡斀闁绘劘灏欏﹢鎾煕婵犲喚娈橀柛鎺撳浮椤㈡﹢濮€閳╁啯鐝梻浣告啞濞诧箓宕㈡ィ鍐╂櫖婵犻潧顑嗛埛鎴犵磼鐎ｎ偄顕滄繝鈧导瀛樼厾鐟滅増甯為悾娲煕閳规儳浜炬俊鐐€栫敮濠囨倿閿曞倸纾归柟閭﹀枓閸嬫挾鎲撮崟顒傤槰闂佸憡姊归悧鐘烘闂佺粯顨呴悧濠囧磿閻斿吋鐓ユ繝闈涙瀹告繈鏌涢弮鍥ㄧ【闁宠鍨块幃娆戔偓娑櫭棄宥夋⒑缁洘娅呴柛鐔告綑閻ｇ兘骞嬮敃鈧粻鑽ょ磽娴ｈ鐒介柛姗€娼ч—鍐Χ閸℃鍘介梺鍛婃煥閼活垳鍙呴梺闈涚墕椤﹀崬鐣垫笟鈧弻鏇＄疀婵犲倸鈷夐梺鍛婄懃缁绘垶绌辨繝鍥舵晬婵犲﹤娴烽崝鎼佹⒑閸涘﹥鈷愰柣鐕傜畵楠炲牓濡歌閸犲棝鏌涢弴鐐典粵闁汇倓绀侀埞鎴︻敊绾攱鏁惧┑锛勫仒缁瑩鐛繝鍌楁斀閻庯綆浜炴鍥р攽閻愬弶顥犻柛瀣崌钘熺€广儱娲ㄧ壕浠嬫煕鐏炴崘澹橀柍褜鍓氶幃鍌炲箖瑜戠粻娑㈠籍閳ь剙鈻嶉悩鍏呯箚闁靛牆鎳庨弳鐔兼煟椤撶儐妲洪柍褜鍓欑粻宥夊磿闁秵鍋嬮柛鏇ㄥ墰椤╄尙鎲搁悧鍫濈瑲闁绘挻鐟╅弻锝夊箣閻愬棙鍨规禍鎼佹偨閸涘﹦鍘卞┑鈽嗗灠閸氬寮抽浣瑰弿濠电姴鎳忛鐘电磼鏉堛劌绗掗摶锝夋煣韫囨稈鍋撳☉娆樻晣闂傚倷娴囧Λ鍕疮鐎涙顩查柛顐ｆ礀閽冪喖鏌ㄥ┑鍡╂Ц缂佺姴顭烽弻娑樜旈崘銊ょ捕婵犳鍠楀ú鐔奉潖濞差亝顥堟繛鎴炵懄閹瑩鏌ｆ惔銏㈩暡婵犮垺锕㈤、姘舵晲婢舵ɑ鏅㈤梺鍛婃处閸嬪棝藝闁秵鈷戦柣鎰閸旀岸鏌涘Ο鑽ゅ⒈婵″弶鍔欓弫鎰緞鐎Ｑ勫濠电偠鎻徊浠嬪箺濠婂牆鍑犻柛鎰ㄦ櫃缁诲棙銇勯幇鈺佺仼妞ゅ浚鍙冮幐濠傗攽鐎ｎ偆鍘介梺闈涚箳婵敻骞夐崫銉㈠亾鐟欏嫭灏柣鎺炵畵楠炲牓濡搁妷顔藉瘜闁荤姴娲╁鎾寸珶閺囥垺鈷掑ù锝呮啞鐠愶繝鏌涘Ο鐘叉噽缁€濠傗攽閻樺弶鍌ㄩ柍褜鍏涚欢姘嚕閹绢喖顫呴柍鈺佸暞閻濇娊姊绘担鍛婂暈濞撴碍顨婂畷銏＄附閺夊棗顦甸獮鎺懳旀担鍙夊闂備胶顭堥張顒勬嚌妤ｅ啫鐒垫い鎺嗗亾闁搞垺鐓″﹢渚€姊洪幖鐐插姶闁告搫绠撳顐も偓锝庡枟閻撴稓鈧箍鍎辨鎼佺嵁閺嶎偆纾奸柟閭﹀弾濞堟洖菐閸パ嶈含濠碘€崇埣瀹曘劑顢欓崗纰变哗缂傚倸鍊烽懗鑸垫叏閹惰棄纭€闁规儼妫勯拑鐔兼煏閸繃顥為柛娆忕箻閺岀喖鏌囬敃鈧崢鎾煕鐎ｎ偅灏摶鏍煕濞戝崬鏋涢柛鎾舵嚀閳规垿鎮欓崣澶樻濠电偛鐡ㄥ畝鎼佸箖閵忋倕骞㈡繛鎴炵懅閸橀箖姊鸿ぐ鎺擄紵闁绘帪绠戦埥澶庮槻闁宠鍨块、娆撳礂閻撳孩鐏庨梻浣告惈閻绱炴笟鈧悰顕€骞掑Δ鈧粻锝夋煟濡じ鍚瑙勫▕濮婄粯绗熼埀顒€顭囪鐓ら柕鍫濇礌閸嬫挸顫濋銏犵ギ闂佺粯渚楅崳锝夌嵁閹烘嚦鏃堝焵椤掑倻涓嶉柨婵嗘缁♀偓闂傚倸鐗婄粙鎴﹀汲濞嗗緷鐟邦煥閸垻鏆梺鍝勭焿缂嶄線骞冮姀銏㈢煓婵炲棛鍋撻ˉ瀣⒒娴ｅ憡鎯堟俊顐㈩嚟缁骞嬮敂鍏夊亾閿旂偓宕夐柕濠忕畱绾绢垱绻涢幘鏉戝毈闁搞劋鍗冲畷婊堫敇閻旇櫣顔曢柣搴㈢⊕椤洭鎯岀€ｎ剛纾兼い鏃囧Г瀹曞瞼鈧鍠栭…鐑藉垂妤ｅ啫绠涘ù锝呮啞閸婎垶姊绘担鍛婂暈闁告梹鍨垮畷婵嗙暆閸曘劉鍋撻弽銊ョ窞闁归偊鍘搁幏鍝勨攽椤旂偓鍤€婵炲眰鍊栨穱濠冪鐎ｎ偆鍘遍梺闈涱焾閸庤尙鎷归埄鍐х箚闁告瑥顦慨宥嗩殽閻愭潙绗掗摶鏍煃瑜滈崜娑氬垝鐠囨祴妲堟繛鍡楃С缁ㄥ姊洪崫鍕殭闁绘锕弫宥夋偄闂€鎰畾闂佸憡鐟ラˇ顖涙叏瀹ュ棙鍙?set/delete闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴︽煙閼测晛浠滈柍褜鍓氶悧鐘茬暦濠靛鍐€妞ゆ挾鍊ｉ敃鍌涚厱闁哄洢鍔岄悘鐘绘煕閹般劌浜鹃梻鍌欑窔濞佳団€﹂崼銉ョ？闁告鍊幒妤€閱囨繝闈涘暞閺傗偓闂備胶绮崝妯间焊濞嗗警鎺楁倷閻戞鍘搁梺鍛婁緱閸ㄤ即鍩㈤崼鈶╁亾濞堝灝鏋涙い顓炲槻椤曪綁骞橀钘変簻闂佹儳绻楅鏍疮鎼淬劍鈷掑ù锝堟鐢盯鏌涢弮鎾剁暤鐎规洘绮岄～婵嬫嚋闂堟稐鎴烽梻浣告惈濞层垽宕瑰ú顏勭厱闁瑰濮风壕钘壝归敐鍜佹綘妞ゅ繐妫涚粻鏃堟煙閻戞﹩娈曢柣鎾崇箻閺屾盯顢曢妶鍛€鹃梺绋匡攻閸旀鍩€椤掍緡鍟忛柛鐘崇洴椤㈡俺顦归柛鈹垮劜瀵板嫰骞囬鍌氬箑闂備礁鐤囬～澶愬磿閾忣偆顩插Δ锝呭暞閻撱儵鏌￠崶鈺佷粶闁逞屽墮閹芥粎鍒掓繝姘櫜闁糕剝鐟ч惁鍫ユ⒒閸屾氨澧涚紒瀣笒椤斿繐鈹戦崰銏㈡嚀椤劍鎯斿┑瀣粣婵犳鍠栭敃銈夊箹椤愶絾娅忛梻浣规偠閸庢粓宕ㄩ鐐愩倝姊婚崒娆戝妽闁诡喖鐖煎畷婵婄疀閺傝绮撻梺鍛婄缚閸庢彃鐣烽崣澶岀闁瑰鍋熼幊鍛磼閳锯偓閸嬫捇姊绘担瑙勫仩闁稿孩绮撳畷鍗炍熺拠鍙夋瘒濠电姷鏁告慨鐑藉极閹间礁纾婚柣鎰惈閸ㄥ倿鏌涢幘鑼槮闁搞劍绻堥獮鏍庨鈧俊浠嬫煕鐎ｅ吀閭柡灞界Х椤т線鏌涢幘璺烘灈鐎殿喖顭烽弫鎰板幢濡搫濡虫俊鐐€栭悧妤冨垝鎼淬垻澧″┑鐘垫暩閸嬫盯鎮洪妸褍鍨濈€广儱娲ら崹婵嬫煙閸撗呭笡闁哄懏鐓￠弻锝夊箛椤掍焦鍎撻梺缁樺姇閿曘儵濡甸崟顖氱鐎广儱鐗滈崬褰掓⒑?Marshal闂?
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鎮㈤崗灏栨嫽闁诲海鏁搁…鍫熶繆娴犲鈷戠紒瀣儥閸庡繑銇勯幋婵愭█闁诡噯绻濋、鏇㈡晝閳ь剟鎮欐繝鍥ㄧ厪濠电倯鈧崑鎾翠繆閹绘帞澧㈢紒杈ㄥ浮閹瑩顢楅埀顒勫礉閵堝棎浜滄い鎾跺Т閸樺鈧鍠栭…宄邦嚕閹绢喖顫呴柣妯款嚙閺佽绻濋悽闈浶㈤柣蹇斿哺瀹曟繈寮介銈嗗櫘闂傚倸鍊搁崐鐑芥嚄閸撲礁鍨濇い鏍ㄧ矋閺嗘粓鏌熼悜姗嗘畷闁稿﹨鍩栭幈銊ノ熼幐搴ｃ€愭俊妤€鎳樺娲川婵犲啫顦╅梺鎼炲妽婢瑰棛鍒掓繝姘闁兼亽鍎抽崢閬嶆⒑閺傘儲娅呴柛鐔村妽缁傛帡鏁冮埀顒勨€︾捄銊﹀枂闁告洦鍓涢ˇ顓犵磽娴ｄ粙鍝洪柟鐟版搐閻ｇ兘骞掗幋鏃€鐎婚梺褰掑亰閸犳捇宕戝Ο璁崇箚闁绘劦浜滈埀顒佺墪閳绘棃鏁冮崒姘卞€為梺鑲╊焾閻忔艾鈻嶉崼銉︹拻濞达綀顫夐崑鐘绘煕鎼淬垺銇濋柟顔矫埥澶愬閳╁啯鐝冲┑鐘灱濞夋盯鏁冮敃鍌涘仾闁逞屽墯娣囧﹪鎮欓鍕ㄥ亾閺嶎灛娑欐媴閼叉繃鐩畷鐔碱敇閻戝棙顥℃俊鐐€栭悧妤冪矙閹惧墎涓嶅Δ锝呭暙缁狙囨煕椤愶絿绠撻柍閿嬫⒒缁辨帞鈧綆鍓涚敮娑氱磼缂佹娲寸€规洩绲惧鍕暆閳ь剟鎮℃径鎰€甸悷娆忓绾惧鏌涘Δ鈧崯鍧楊敋閿濆棛顩烽悗锝呯仛閺咃綁姊虹紒妯哄閻忓繑鐟╅崺鈧い鎺戭槸瀵喗鎱ㄦ繝鍛仩闁归濞€閸ㄦ儳鐣烽崶锝呬壕濠电姵纰嶉悡鏇熺箾閹存繂鑸归柡瀣枑閵囧嫰寮埀顒€煤閻斿娼栫紓浣股戞刊鎾偡濞嗗繐顏╁ù鐘櫊濮婅櫣绮欓崹顔炬О闂侀潻缍囩徊浠嬵敋閿濆钃熼柕澶堝劤閸樻悂姊洪崨濠佺繁闁告﹢绠栧畷顐⑽旈崨顔规嫼闂傚倸鐗婃笟妤呮倿妤ｅ啯鐓曢幖鎼枛濞呭秶鈧鍠撻崝宥囩矉閹烘柡鍋撻敐搴′簽闁告ü绮欏楦裤亹閹烘垳鍠婇梺鍛婎焽閺咁偆妲愰悙鍝勭闁挎梻鏅崢浠嬫椤愩垺鍌ㄩ柛搴㈠▕閹箖鎮介崨濠勫幐閻庡厜鍋撻柍褜鍓熷畷浼村冀瑜忛弳锕傛煏婵犲繐顩紒鈾€鍋撻梻浣告啞閸旀垿宕濇径鎰；闁规崘顕х粻娑欍亜閹惧鐭嗙紒銊ヮ煼濮婃椽妫冨☉銏㈠椽缂備浇椴稿ú姗€寮查崼鏇炲瀭妞ゆ劦鍋呴鏃堟⒑缂佹ê濮囨俊顖氾躬瀹曟洟寮崼鐔哄幍濡炪倖妫侀～澶娾枍婵犲喚娈介柣鎰级婢跺嫰鏌熷畡鐗堝殗鐎规洜鍏橀、姗€鎮╅锝忕礆濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻樺濠氬醇閻旂儵鍋撻挊澹濇椽顢旈崨顔界彇闂備胶顭堥張顒佺┍濞差亜绠洪柡鍥ュ灪閸婄敻鏌ｉ悢鍝勵暭闁哥喓鍋ら弻宥夘敂閸♀晙鎴穟ctions 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈鏍￠悷婊勬緲閸熸潙锕㈡笟鈧娲箰鎼达絿鐣靛┑鐐茬湴閸旀垶淇婄€涙绡€闁搞儯鍔庨崢鎼佹煟鎼搭垳绉靛ù婊呭仦閺呫儵姊绘担鍛婃儓閻炴凹鍋婂畷婵嗙暆閸曨偆鍘撮梺纭呮彧缁犳垿鐛姀銈嗙厓闁告繂瀚埀顒佸姍閺佸啴宕掑☉鎺撳缂傚倸鍊烽悞锕佹懌闁诲繐绻嬮崡鎶藉蓟閵娾晛鍗虫俊顖濇娴煎牓鎮楀▓鍨灈妞ゎ厾鍏樺畷瑙勩偅閸愩劎鐤€婵炶揪绲介幉锟犲磹椤栫偞鈷戠痪顓炴噹娴滃綊鎮跺☉鏍у姦闁糕斁鍋撳銈嗗坊閸嬫挾鐥紒銏犲籍妞ゃ垺鐟╅獮鍥敊閸撗嶇闯闂備胶顭堥張顒勬偡瑜忛幏瑙勫鐎涙鍘遍柣搴秵娴滄粓顢旈銈囩＜妞ゆ棁濮らˉ鍡涙煏閸ャ劌濮嶆鐐村浮楠炲鏁愰崱妯烘毇闂傚倸鍊风欢姘焽瑜旈幃褔宕卞鏇熸そ婵℃悂鍩￠崒姘婵犵數鍋涘Λ娆撳箰婵犳艾纾婚柨婵嗩槹閻撴瑦銇勯幘璺烘瀻缂佹甯掗埞鎴︽婵炲拑绲垮Σ鎰板箳濡ゅ﹥鏅╅梺缁樺姦閸撴艾袙閸績鏀芥い鏃傘€嬮弨缁樹繆閻愯埖顥夐柣锝囧厴椤㈡洟鏁冮埀顒傜矆鐎ｎ偁浜滈柟鐑樺焾濡茶銇勯妷顖滅М婵﹦绮幏鍛存惞閻熸壆顐兼俊鐐€戦崝灞轿涘┑瀣祦闁硅揪绠戦悙濠勬喐濠婂牆鍚归柡鍥ュ灪閸嬶綁鏌涢妷锝呭闁靛牆鐡ㄦ穱濠囧箵閹烘柨顤€缂備胶绮换鍐崲濠靛纾兼繛鎴炆戦銈夋⒒娴ｅ憡鎯堟い锔垮嵆楠炲啴宕掗悙鑼舵憰闂佹寧绻傚Λ娆撳磿閻旀悶浜滈柡宥冨妿閻矂寮介垾鏂ユ斀闁绘劘灏欓幗鐘电磼椤旇偐肖闁告帗甯掗鍏煎緞婵犲嫮鏉介梻渚€娼ц墝闁哄懏绮撳鎻掝煥閸喓鍘遍梺鍦亾椤ㄥ懘骞忛幋鐘愁潟闁告劖绁撮弨浠嬫煟閹邦剛鎽犻悘蹇庡嵆閺屻倗鎲撮崟顐㈠Б闁绘挶鍊濋弻鏇㈠醇濠靛洤娅濋梺鍝勵儏缁夌敻骞堥妸銉富閻犲洩寮撴竟鏇㈡⒒娴ｅ憡鎯堥柡鍫墴閹嫰顢涘☉妤冪畾闂佸綊妫块悞锕傚疾濠靛鐓冪憸婊堝礈閻旂厧鏄ラ柕蹇嬪€曢崡鎶芥煟閺囨氨鍔嶆い鏃€妫冨铏圭磼濡搫顫戦柣蹇撶箲閻熝呭垝婵犳艾绠柤鎭掑劤閸樻捇鎮峰鍕煉鐎规洘绮撴俊姝岊槾缂佲偓婵犲洦鐓曢柍鈺佸暟閳藉绱掗幇顓ф疁闁哄备鈧磭鏆ゆい鏃傚帶椤忣偊鏌涢幋婊呯煓婵﹤顭峰畷鎺戭潩椤戣棄浜鹃柣鎴ｅГ閸ゅ嫰鏌涢幘鑼妽闁稿繑绮撻弻娑㈩敃閿濆棛顦ョ紓浣插亾闁割偆鍠撶弧鈧梻鍌氱墛娓氭宕曞澶嬬厓鐟滄粓宕滃▎鎾崇柈闁哄鍨归弳锕傛煟閹惧啿鐦ㄩ柣鏂挎閵囧嫯绠涢幘鎼闂佸搫顑嗙粙鎾诲焵椤掑喚娼愭繛鍙夌矒瀹曚即寮介婧惧亾娴ｈ倽鏃堝川椤撶媴绱叉繝鐢靛Т閿曘倖顨ラ幖浣哥柧妞ゆ挶鍨洪埛鎺懨归敐鍥╂憘闁搞倖鐟︾换娑氣偓鐢殿焾閸樺鈧鍣崑濠囩嵁閺嶃劍濯存繛鏉戭儏娴滈箖鏌ㄩ弴鐐测偓鎼佹煁閸ヮ剚鐓ユ繝闈涙閳ь剛顭堥銉╁籍閸啿鎷婚梺绋挎湰閼归箖鍩€椤掑倸鍘撮柟铏殜瀹曞ジ寮村璇蹭壕闁挎洖鍊搁柋鍥煏婢舵盯妾繛鍛喘濮婃椽妫冨☉杈ㄐ㈤梺鍝勬噺閻╊垰鐣疯ぐ鎺戠＜闁绘劕顕崢閬嶆偡濠婂啴鍙勯柕鍡楀暣瀹曘劑寮堕幋婵嗘尋闂備線娼чˇ顐﹀疾濠婂牊鍋?
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻濠氭煙瀹曞洤浠遍柡灞芥椤撳ジ宕卞Δ浣烘殶闂傚倷绀侀悿鍥ь浖閵娾晜鍤勯柛鎾茶兌妞规娊鐓崶銊︹拻缂佲檧鍋撻梻浣圭湽閸ㄨ棄顭囪閻楀酣姊绘担铏瑰笡婵☆偄绻樺畷婊冣槈濮橆収娼熼梺缁樺姇閹碱偊宕橀埀顒勬偡濠婂嫮鐭掗柟顔哄灲瀹曞崬鈽夊▎蹇庡寲闂佸搫顦遍崑鐐村垔娴犲鏁嗛柣鏂垮悑閻撴洟骞栧ǎ顒€鐏╁┑顔肩墢缁辨帞绱掑Ο鍏煎垱閻庤娲忛崝鎴︺€佸☉妯炴帞鎲楅妶鍛Е闂佸搫鑻粔鐑铰ㄦ笟鈧弻娑㈠箻鐎靛憡鍣紓渚囧枦椤曆囶敇閸忕厧绶炲┑鐘插缁辩敻姊婚崒娆戣窗闁告挻鐟╁畷锟犲即閵忕姷鍊炲銈嗗笒椤︿即寮查鈧埞鎴︽偐濞嗗繐顏╅柛鏂诲€曢湁婵犲﹤鎳庢禍楣冩煙娓氬灝濡兼い顐ｇ矒瀹曞崬螖閳ь剟鍩涢幒鎴旀斀闁斥晛鍟徊濠氭煙鐠囇呯瘈濠碉紕鏁诲畷鐔碱敍閿濆棙娅嶉梻浣规灱閺呮盯宕导鏉懳ュ┑鐘叉处閳锋垿鏌涢敂璇插箺闁搞倐鍋撻梺璇插绾板秴顭垮鈧幃楣冩倻缁涘鏅濋梺鎸庢磵閸嬫挾绱掗埀顒傗偓锝庡亖娴滄粓鏌熼幑鎰【闁哄懎娼￠弻锝夊箛椤旂厧濡洪梺鎶芥敱鐢帡婀侀梺鎸庣箓閻楁粌顭囬幇鐗堢厱閻庯綆鍓欓埢鍫ユ煛鐏炲墽鈯曠紒缁樼箞瀹曟﹢鎮欓鍌涙緬闂佽姘﹂～澶娒哄鈧畷褰掑锤濡も偓缁犳牗绻涢崱妯诲碍濞磋偐濞€閺屾盯寮撮妸銉ョ闂佸搫顦划娆忣潖缂佹鐟归柍褜鍓欓…鍥樁濠⒀勭箞濮婃椽宕崟顒€娅ょ紓渚囧枟閹告悂鎮鹃悜钘夌闁挎棁濮ゅ▍銏ゆ⒑缂佹◤顏嗗椤撱垹鍌ㄩ柛娑橈功缁♀偓闂侀潧楠忕徊鍓ф兜妤ｅ啯鐓ラ柡鍥朵簽閻ｈ鲸銇勯锝囩疄濠碘剝鎮傞崺锟犲磼濠婂啫绠為梻浣筋嚙閸戠晫绱為崱娑樼；闁告侗鍨悞濠囨煟閵忕姵鍟為柣鎾寸懇閺屾稑鈻庤箛锝喰ч柣蹇撻獜婵″洭鍩€椤掑喚娼愭繛鍙夌矒瀹曚即寮介婧惧亾娴ｈ倽鏃€鎷呴悷閭︹偓鎾绘⒑閸涘﹦绠撻悗姘煎墴閸┿垼绠涘☉娆屾嫽闂佺鏈懝楣冨焵椤掆偓閸㈡煡鈥旈崘顔肩闁哄啠鍋撴い顐ｆ礃閵囧嫰寮埀顒勫磿瀹曞洦顐介柕鍫濇偪瑜版帗鍋愮€瑰壊鍠栭崜浼存煟鎼淬垼澹樻い锔垮嵆婵＄敻宕熼姘鳖唺闂佺懓鐡ㄧ换宥嗙閼测晝纾奸柟顖嗗啠鎸冪紓渚囧櫘閸ㄥ爼鐛崼銉ノ╅柕澶樺枟鐎靛矂鏌ｉ悩鍙夌┛鐎殿喗鎸荤粩鐔煎即閵忊檧鎷绘繛杈剧悼閻℃棃宕甸崘顔界厱闁绘ɑ鍓氬▓婊堟煛娴ｇ鏆ｅ┑顔瑰亾闂侀潧鐗嗛幊鎰版偩閸濆嫧鏀介柣鎰级椤ョ偤鏌熺粙鎸庢喐缂侇喖鐗撳畷鎺楁倷鐎电骞堟繝鐢靛█濞佳呪偓姘煎墴瀹曟繈濡舵径瀣帗闁荤喐鐟ョ€氼噣鍩ユ径鎰厓閻熸瑥瀚悘鎾煙椤旂晫鎳呴柍褜鍓氱粙鎺楁晪闂傚倸顦粔鐟邦潖閾忕懓瀵查柡鍥╁枑閻濇棃姊洪崫銉ユ瀾婵炲吋鐟╅幃楣冩倻閽樺顓洪梺鎸庢磵閸嬫挾绱掗悩宸吋闁哄本鐩俊鎼佹晜闁款垱顫嶅┑鐐茬摠缁挾绮婚弽顓炵畺鐎瑰嫭澹嬮弸搴ㄧ叓閸ャ劍鎯勫ù鐘插⒔缁辨挻鎷呴幓鎺嶅闂備線鈧偛鑻晶顕€鏌嶇憴鍕伌闁诡喒鏅犻、鏇綖椤撶儐妫滈梻鍌氬€搁崐椋庢閿熺姴绀堟繛鍡樺灩閻捇鏌熺紒銏犳灍闁稿﹤娼￠弻娑⑩€﹂幋婵呯按婵炲瓨绮嶇划鎾诲蓟閻斿吋鍊绘俊顖濇閸樻劙姊洪崨濠冣拻闁哥姵鎸惧Σ鎰板箳閹惧磭绐為柣蹇曞仧閸嬫挸袙閸儲鈷戦柛婵嗗濠€鎵磼鐎ｎ偅宕岄柛鈹惧亾濡炪倖甯掗敃锔剧矓闂堟耽鐟邦煥閸曨厾鐤勯悗瑙勬礈閺佸宕洪埀顒併亜閹烘垵顏柍閿嬪灴濮婂宕奸悢琛″濡炪們鍎抽崑銈夊蓟閿濆憘鏃堝礃閵婃劑鍨洪妵鍕敃閵忋垻顔掗梺鍦帶濠€閬嶅箟閹绢喖绀嬫い鎰╁€曢柊閬嶆⒒閸屾瑦绁伴柛瀣姍閸╂盯宕奸妷銉ь槶濠电偞鍨崺鍕极婵犲洦鐓曢柕澶堝灪濞呭洨鐥幑鎰棄闂囧鏌ㄥ┑鍡欏妞ゅ繒濞€閹粙顢涘☉姘垱闂佸搫鏈惄顖氼嚕椤曗偓閸┾偓妞ゆ帒瀚ㄩ埀顒€鍟换婵嬪炊瑜庡Σ顒勬⒑闁偛鑻晶顖炴煏閸パ冾伃妤犵偞甯掗濂稿幢濞嗗秴浠掓繝鐢靛О閸ㄥジ锝炴径鎰櫇闁靛繈鍊曢拑鐔兼煟閺傚灝鎮戦柛銈呯Ч閺屾洘绔熼姘櫧闁告ɑ鍔欏娲川婵犲孩鐤佹繛瀛樼矊閻栫厧顕ｆ繝姘亜闁稿繗鍋愰崝鐑芥⒑閹稿孩纾甸柛瀣崌閺屾稑螣缂佹ê鍞夐梺鍝勭焿缁辨洟鍩€椤掑﹦绉靛ù婊庝邯瀹曪綁宕熼娑氬幐閻庡厜鍋撻柍褜鍓熷畷浼村冀椤撶姴绁﹂梺绯曞墲閸戠懓危妤ｅ啯鈷戦柟绋挎捣閳藉绱掓径灞炬毈濠碉紕鏁诲畷鐔碱敍濞戞瑦鐝栭梻浣侯焾閺堫剟鎮烽敃鍌氱獥婵﹩鍏橀弨鑺ャ亜閺冨倶鈧寮ㄧ紒妯圭箚闁绘劘鍩栭ˉ澶愭煟閿濆洤鍘村┑鈩冩倐閺佸倿宕滆濡插洤鈹戦悙鑸靛涧缂傚秮鍋撳┑鐐叉嫅缁茶法鍒掓繝姘婵°倓璁查幏缁樼箾鏉堝墽鍒伴柟璇х節瀹曨垶鎮欓悜妯哄壋婵犮垼娉涢惉鑲╁閸忕浜滈柡鍐ㄥ€瑰▍鏇㈡煙閸愭彃鏆ｉ柡?Codex CLI闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊婚崟顐ｅ枠妞ゃ垺淇洪ˇ鏌ユ偂閵堝棎浜滈柟鍨暞婵炲洭鏌嶈閸忔稓绮堟笟鈧敐鐐差煥閸繄鍔﹀銈嗗笂閻掞箓宕ｈ箛娑欑厓鐟滄粓宕滈悢鐓庤摕闁挎繂鎷嬪銊╂煃瑜滈崜娆撯€﹂崶顏嶆Ъ闂?
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

	// Compact-only model 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻樻爠鍥ㄧ厱閻忕偛澧介悡顖氼熆鐟欏嫭绀€闁宠鍨块、娆戠驳鐎ｎ剙濮洪梻浣告啞椤棝宕惰閸ゆ垿姊虹紒妯荤叆闁告艾顑夐幃鈥斥枎韫囧鍋撻幒鎴僵妞ゆ帒鍊烽搹搴㈢節濞堝灝鏋涢柛濠傜秺閵嗗啴濡烽埡鍌氣偓鐑藉级閸喎绀冮柍褜鍓氱€笛囧Φ閸曨垰顫呴柨娑樺閸ｄ即姊洪崷顓х劸闁挎洏鍎遍銉╁礋椤掑倻鐦堥梺绋胯閸婃牠藟婢舵劖鈷掗柛灞捐壘閳ь剚鎮傚畷鎰版倻閼恒儱娈戦梺鍛婃尫缁€渚€宕瑰┑鍥ヤ簻闁哄稁鍋勬禍褰掓煃瑜滈崜銊х不閹惧磭鏆﹂柛顐ｆ礀閻撴稑霉閿濆懏鎲稿ù婊呭娣囧﹪鎮欓鍕ㄥ亾閺嶎厽鍋嬫繝濠傜墕绾剧粯绻涢幋娆忕仾闁稿浜弻锝夊箛闂堟稑骞嶆繛瀛樼矋缁捇寮婚悢鐓庝紶闁告洦鍘滈妶澶嬬厱闁哄倽娉曟晥闂佸搫鏈粙鎴﹀煝鎼淬倗鐤€闁哄洨濯崬褰掓⒒娓氣偓閳ь剚绋撻埞鎺楁煕閺傝法鐒烽柣蹇撳暣濮婃椽鏌呴悙鑼跺濠⒀屽櫍閹藉爼鎮欑紙鐘电畾闂侀潧鐗嗙€氼垶宕楃仦淇变簻闁冲搫鍊婚崣鈧梺鍝勭焿缂嶄線鐛崶顒夋晩闁绘挸楠搁‖鍡涙⒒娴ｈ櫣甯涢柟鎼佺畺瀹曚即骞囬弶璇撅箓鏌涢弴銊ョ仩缂佺姴顭烽幃瑙勩偊閹稿簺鈧啴鏌熺€电鍘存慨濠呮缁辨帒螣閾忓湱鎳栭梻浣筋嚃閸犳牠宕愰崹顕呭殨?/responses/compact 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犳娊鏌嶈閸撴繈锝炴径濞掓椽寮介‖鈩冩そ閺佸倿鎸婃径濠勪簴濠电偛顕崢褔鎮洪妸鈺傚亗闁靛鏅滈悡娑㈡煕閵夈垺娅呭ù鐘崇矒閺屽秷顧侀柛鎾村哺楠炲啴宕掑杈ㄦ閻熸粎澧楃敮鎺斿婵傚憡鐓熼柟閭﹀墻閸ょ喖鏌涘鈧褔鍩為幋锔藉€烽柤纰卞墮椤ｆ椽姊虹拠鑼缂佽鐗嗛锝夊箹娴ｇ懓浜滈梺纭呭亹閸嬫鑺辨繝姘拺鐟滅増甯掓禍鏉棵瑰鍛槐闁诡垰鑻埢搴ㄥ箻鐎电骞愰柣搴″帨閸嬫捇鎮楅敐搴″鐞氾附淇婇妶鍥ラ柛瀣☉鐓ゆい鎿冩娇閳ь兛绀侀埢搴ㄥ箻閺夋垳绨甸梺纭呭亹鐞涖儱危閸涘瓨鍊甸梺顒€绉甸埛鎺楁煕鐏炲墽鎳嗛柛蹇撴湰閵囧嫰顢橀悙闈涒叺濡炪們鍨洪悧鐘茬暦濮椻偓椤㈡瑩鎳栭埡濠冃ラ梻鍌欒兌椤㈠﹪骞撻鍡欎笉闁硅揪闄勭€氬懘鏌ｉ弬鍨倯闁绘挶鍎茬换婵嬫濞戞瑱绱炲┑鈩冨絻閹测剝绌辨繝鍥舵晝闁挎繂绻楅埀顒€娼￠弻鈥崇暆閳ь剟宕伴幘璺哄灊婵炲棙鎸搁柨銈嗕繆閵堝惇鍫ュ磻閹炬緞鏃堝川椤旀儳骞愰梻浣虹《閸撴繈銆冮崼鐔告珷闁哄洢鍨洪悡鍐偡濞嗗繐顏╅柣蹇旀尦閺岀喖顢欓悾灞惧櫚閻庤娲╃徊鎯ь嚗閸曨剛绡€闁告劕褰為崫妤冪磽閸屾艾鈧悂宕愰悜鑺ュ殑闁告挷绀侀崹婵囥亜閺嶎偄浠﹂柡鍛箞閺屾洝绠涢弴鐑嗏偓宀勬煕閵堝棙绀€闁宠鍨块幃鈺冣偓鍦Т椤ユ繈鏌熼婊冩灈婵﹥妞藉Λ鍐ㄢ槈鏉堛剱銈夋⒑閹肩偛濡芥俊鐐扮矙閺佹劙鎮欏顔兼倯闂佸憡渚楁禍婵嬪棘閳ь剟姊绘担鍝ユ瀮婵☆偄瀚灋婵°倕鎳忛崐鍫曟煟閺傚灝鎮戦柛瀣剁秮閺屽秷顧侀柛鎾跺枛瀵偊宕橀鑲╁姦濡炪倖甯掔€氼剟鎮″┑瀣厵闁绘劦鍓氶妵鐔兼煛娴ｅ憡鍠橀柡宀嬬到铻ｉ柛蹇撳悑濮ｅ嫰姊洪幐搴ｂ姇缂佽鍟伴幑銏犫槈濞嗘劗绉堕梺鍛婃寙閸涘懏鑹鹃—鍐Χ閸℃顦ㄩ梺缁橆殕濞茬喖宕洪姀銈呯閻犲洦褰冨畵鍡涙⒑闂堟盯鐛滅紒杈ㄦ礋瀹曘垽鎸婃径鍡樻杸闁圭儤濞婂畷鎰板箻閼告鍋ㄩ梺鐐藉劜閺嬪ジ寮搁弮鈧幈銊ヮ渻鐠囪弓澹曢梻浣告惈閼活垳绮旈悜閾般劍绗熼埀顒勫蓟濞戙垹绠婚悗闈涙啞閸掓盯鎮楀▓鍨珮闁稿锕悰顕€寮介妸锕€顎撶紓浣割儏閻忔繃绂嶉崼鏇熺厽闁绘柨鎽滈惌瀣煕閵娿儲鍋ユ鐐插暙椤粓鍩€椤掑嫮宓侀柟鐑樺殾閺冣偓閹峰懘宕ｆ径濠庝紪闂傚倸鍊风粈渚€骞夐敍鍕煓闁硅揪闄勯弲婵嬫煥閺傚灝鈷斿☉鎾崇Ч閺岀喖宕滆鐢盯鏌涙繝鍌滀粵闁逛究鍔岃灒闁绘挸楠告禒妯荤節閳封偓鐏炶姤鐏嶉梺闈涙搐鐎氭澘顕ｉ幘顕呮晜闁糕剝顨呴悡鍌炴⒑閸撗呭笡闁绘濞€瀵鈽夐姀鐘电杸闂佺绻愰幗婊堝极閺嶎厽鈷戦柟绋挎捣閳洜绱掔拠鎻掓殶闁瑰箍鍨归埥澶娾枎閹邦喚肖闂備礁鎲￠幐鍡涘礃閵娧冨笓闂傚倸鍊风粈浣革耿闁秴鐓曢柛顐犲劚绾捐鈹戦悩鍙夋悙缂佺姵鐗犻弻锝夊閻樺啿鏆堥梺鍝勬噺閹倿寮婚敐鍛傜喖鎼归惂鍝ョ闂備礁鎲￠敃銏＄鐠轰警娼栨繛宸簼椤ュ牊绻涢幋锝夊摵閼叉牠姊绘担瑙勫仩闁告柨鐭傞獮鏍敃閿曗偓閻撴繄鈧箍鍎遍ˇ顖氼啅濠靛鍊垫繛鎴炵懐閻掍粙鏌涘▎蹇撴殻婵﹨娅ｇ划娆撳箰鎼淬垺瀚抽梻浣告啞閸ㄧ數绱炴繝鍌滄殾闁哄洢鍨圭粻娑欍亜閺傚灝鈷旈柨?	// OAuth 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣濠垫劖缍楅梺閫炲苯澧柛濠傛健楠炴劖绻濋崘顏嗗骄闂佸啿鎼鍥╃矓椤旈敮鍋撶憴鍕８闁告梹鍨甸锝夊醇閺囩偟顓哄┑鐘绘涧閻楀啴宕戦幘娲绘晣闁绘垵妫欑€靛矂姊洪棃娑氬闁硅櫕鍔楃划缁樺鐎涙鍘藉┑掳鍊愰崑鎾翠繆椤愶絿绠炴鐐插暣閹瑩宕崟顐も偓顓烆渻閵堝棗濮夊┑顔肩－閼鸿鲸绻濆顓涙嫼闂佽崵鍠撴晶妤呭箚閸喍绻嗘い鎰剁秵濞堟洜绱掗崒姘毙х€规洘绮忛ˇ瀵哥棯閹佸仮闁哄本鐩獮妯何旈埀顒€螞濞嗘搩鏁佹俊銈呮噺閳锋垿鏌涘☉姗堝姛闁瑰啿鍟撮弻娑㈡偄閸涘﹦绋囬梺浼欑到閸㈣尙鍙呭銈呯箰鐎氼噣宕濋敃鈧—鍐Χ閸℃鐟愰梻鍌氬缁夌數绮嬪鍜佺叆闁割偆鍠撻崢鐢告⒑缂佹ê鐏﹂柨姘舵煟韫囧鍔滈柕鍥у缁犳盯骞橀懜鍨枛濠电儑绲藉ú銈夋晝椤忓牄鈧線寮撮姀鈩冩珳闂佸憡渚楅崹顖滄濠靛鈷掑ù锝呮啞閹牓鏌ｉ鈧妶绋跨暦娴兼潙鍐€鐟滃繘寮抽敃鍌涚叆婵犻潧妫欓ˉ婊呯磼閸撲礁浠遍柟顔筋殜閺佹劖鎯斿┑鍫㈡晨闂備浇銆€閸嬫挻銇勯幘鍗炵仾闁绘挾鍠栭獮鏍庨鈧埀顑惧€曢…鍥箛椤撶姷顔曢梺鍛婄懃椤﹂亶鎯岄幒鏂哄亾鐟欏嫭纾婚柛妤€鍟块锝嗙鐎ｅ灚鏅ｅ┑鐘欏嫬鍔ゅù婊勫劤闇夐柨婵嗘川閵嗗﹪鏌＄€ｎ亪鍙勯柡宀€鍠栭幃娆擃敆娴ｈ櫣鈻忔繝鐢靛仜閻楀﹪宕濆▎蹇ｆ綎婵炲樊浜滅粈鍐煕濞嗗浚妲归柛搴㈡崌濮婅櫣绮欓崠鈩冩暰闂佸憡姊归悷锔界┍婵犲洤绠瑰ù锝呮憸閸樻悂姊虹粙鎸庢拱妞ゃ劌妫涢埀顒佷亢濡嫰鍩為幋锔藉€烽柤鎼佹涧濞懷呯磽娴ｈ棄绱︾紒顔界懇閻涱喗寰勯幇顓熸闂佺粯顭堢亸娆撳蓟閸儲鈷戠紓浣姑慨澶愭煕鎼存稑鈧繈骞冮敓鐘冲亜闂傗偓閹邦喚鐣鹃梻渚€娼ч悧鍡欌偓姘煎枤閹峰綊鎮ч崼銏㈩啎闂佸壊鍋嗛崰鎾绘儗閹烘鐓涚€光偓閳ь剟宕伴弽顓溾偓浣糕枎閹炬緞鈺呮煏婢舵稑顩柕鍫氭櫊濮婄儤娼幍顕呮М濠电偛妯婇崣鍐嚕婵犳碍鏅查柛娑樺€婚崰鎾诲箯閻樿绠甸柟鐑樼箖濞村洤鈹戦悩鍨毄闁稿濞€瀹曟垿骞囬弶璺ㄧ崶闁瑰吋鐣崝宀勬偂濮椻偓閺岀喐娼忔ィ鍐╊€嶉梺绋匡功閸忔﹢寮诲☉妯锋瀻闊浄绲鹃埢鎾斥攽閳藉棗浜剧紒缁樼箓椤繐煤椤忓嫭宓嶅銈嗘寙閸涱厼歇濠碉紕鍋戦崐銈夊储妤ｅ啫绀傛慨妞诲亾闁绘侗鍣ｅ畷姗€顢欓挊澶嗗亾閻戣姤鍊甸柛顭戝亝閸婃劖淇婄紒銏犳珝婵﹥妞藉畷銊︾節閸曘劍顫嶉梻浣瑰濞测晝绮婚幘宕囨殾闁靛繈鍊曠涵鈧梺缁樺灥濡瑧鈧潧鐭傚娲濞戞艾顣烘俊銈囧У閹倿鎮伴鍢夋棃宕ㄩ鎯у箞闂備線娼ц噹闁告洦鍓氶惁鎾绘煟鎼淬埄鍟忛柛鐘冲哺閳ワ箓宕奸妷銉у弨婵犮垼鍩栭崝鏇綖閸涘瓨鐓冮柕澶堝劚鐢姷绱?OAuth 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍嵁瀹ュ鏁婄痪鎷岄哺濮ｅ姊绘担渚劸妞ゆ垶鍨归幑銏犫攽鐎ｎ亣鎽曢梺鍝勬川閸犳捇鎮甸懜鐐逛簻闁哄稁鐓堝▓鏂棵瑰鍫㈢暫婵﹨娅ｇ划娆忊枎閹冨闂備胶鎳撻幉锟犲箖閸岀偞鏅查柣鎰ゴ閺€浠嬫倵閿濆骸浜滃ù婊勵殜閺岀喖鎳濋悧鍫濇锭缂備焦褰冮妶绋跨暦閻樿鍐€妞ゆ挾鍟块幏缁樼箾閹炬潙鐒归柛瀣崌閺屽秷顧侀柛鎾寸洴瀹曟垵鈽夐姀鈥虫濡炪倖鐗楃粙鎺戔枍閻樼偨浜滈柡鍥殔娴滈箖姊洪崫鍕効缂傚秮鍋撶紓浣哄У閻╊垰顕ｉ鍕瀭妞ゆ棁濮ら崵鍫ユ⒒閸屾瑦绁版い鏇熺墵瀹曨垶顢曢敃鈧悙濠囨煏婵炲灝鍔欐慨瑙勵殜濮婄粯鎷呴挊澶夋睏闂佸啿鍢查悧鎾崇暦濠婂喚娼╂い鎴ｅГ閻忎焦淇婇妶蹇曞埌闁哥噥鍨堕崺娑㈠箣閻樼數锛濇繛杈剧悼濞呫垺绗熷☉銏＄厽闁圭儤鏌ㄦ禍楣冩煏閸パ冾伃鐎殿喗鎸冲畷鍗炍旀担瑙勫€风紓鍌欒閸嬫捇鏌涢幇闈涙灍闁绘挸鍟伴幉绋款煥閸繄顦┑鐐叉閹告挳鍩€椤掍焦顥堢€规洘锕㈤、娆撳床婢诡垰娲﹂悡鏇㈡煃閳轰礁鏋ゆ繛鍫燂耿閺岋綁鎮㈤弶鎴濐瀳濡炪値浜滈崯瀛樹繆閹壆鐤€闁哄洨濮烽悰銉╂⒒娴ｅ憡鍟炴い銊ユ缁绘稒绻濋崶鈺佺ウ婵犵數濮撮崑鍡涙偂閵夆晜鐓曢柍鈺佸暙婵啯鎱ㄧ憴鍕垫疁婵﹥妞藉畷鐑筋敇閻愭彃顬嗛梻浣烘嚀閹诧繝骞愰幎鐣屽祦?compact-only 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆樻К闂備胶顭堢悮顐﹀礉鎼淬劌绠熼柟闂寸缁秹鏌涢锝嗗剷闁靛鏅滈悡鏇㈡煙閼割剙濡芥繛鍛閺屾稒鎯旈敐鍛亪闂佸搫鏈ú婵堢不濞戞埃鍋撻敐搴濈敖闁告梹娼欓埞鎴︽倷鐠鸿櫣姣㈠銈庡幖閻楁捇宕洪悙鍝勭闁挎棁妫勬禍褰掓煟閻樺弶鍘傞柛鎰屽懐鐓戦梻鍌氬€烽懗鍓佸垝椤栫偑鈧啴宕ㄧ€涙ɑ娅囧銈呯箰鐎氼噣寮抽敃鍌涚厵閺夊牆澧介悾閬嶆煟閹惧娲撮柡灞剧☉閳藉宕￠悙瀵镐壕闂備礁鎲＄湁缂侇喗鐟ラ～蹇撁洪鍛姷濠电偞鍨堕敋缂佷緡鍠栭埞鎴︽倷閺夊灝鐨熼梺鍛婃处閸撴岸鎮甸柨瀣閻庣數顭堢敮鍫曟煟鎺抽崝鎴﹀箖閿熺姴绀冩い蹇撴閿涙繃绻涙潏鍓хК闁稿鍊块獮瀣偐閻㈢數鍔堕梻渚€鈧偛鑻晶顔姐亜椤忓嫬鏆ｅ┑陇鍩栭幆鏃堝灳瀹曞洤鎽嬪┑鐘垫暩閸嬫盯藝閺夋５娲偄閼测晛绁﹂梺纭呮彧缁犳垹绮堢€ｎ偁浜滈柡宥冨妿閳洟鏌涙惔顔婚偗闁哄矉缍侀幃銏ゅ传閵夛箑娅戦梺璇插閸戝綊宕滈悢濂夊殨闁瑰墎鐡旈弫鍌炴煕閺囨ê濡介柡鍌楀亾闂傚倷鑳剁划顖炲礉閺囥垺鍋ら柕濞у倻鍓ㄦ繛瀵稿帶閻°劑鎮￠弴銏＄厓閺夌偞澹嗛ˇ锕傛煛閸☆厾鎮奸柍褜鍓濋～澶娒哄鈧畷褰掑锤濡ゅ啫绁﹀┑顔姐仜閸嬫挾鈧娲滈幊鎾诲煡婢跺á鐔兼偡闁附顥￠梻鍌欐祰瀹曠敻宕伴幇鐗堝仭闁靛／鈧崑鎾愁潩閻撳骸绠荤紓渚囧枛閻楁捇宕洪埀顒併亜閹哄棗浜鹃梺瀹狀潐閸ㄥ潡骞冮埡鍜佹晝妞ゎ偒鍘奸ˉ姘舵⒒閸屾艾鈧娆㈤敓鐘茬婵炲棙鍨归惌鎾绘煙缂併垹鏋熼柛濠傛健閺屾盯鈥﹂幋婵呯按婵炲瓨绮嶇划鎾诲蓟閻斿吋鍊绘俊顖濇閸樻劙姊洪崨濠冣拻闁哥姵鐗犲璇测槈閵忕姷顔婇梺瑙勫劤椤曨參宕欓垾鎰佹富闁靛牆楠稿銊╂煕鎼粹槅鐓奸挊婵嬫煕濞嗗浚妲虹紒鐘荤畺閺岀喓绱掗姀鐘崇彯濠碘槅鍋呴崹鍧楀蓟濞戙垹绠抽柟鎹愭珪鐠囩偤鎮楃憴鍕闁靛牆鎲℃穱濠囨倻閽樺）銊╂煏婢跺牆濡烽柛瀣崌瀹曞綊顢欑憴鍕澑闂備胶绮崝鏍ь焽濞嗘挻鍊堕柨鏇炲€归悡鐔兼煟閺冣偓缁诲倸煤閵堝纾?
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

	// OpenAI OAuth 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌″畝瀣М濠殿喒鍋撻梺闈涚箚閺呮繈宕濋崫銉х＝濞达絾褰冩禍鐐節閵忥絾纭鹃柡鍫墴瀹曚即骞囬幍顔煎絼闂佹悶鍎崝宥囩矆閳ь剟姊洪幖鐐插缂佽鍊块崺鐐哄箣閿旇棄浜归柣搴℃贡婵挳藟濠靛牏纾藉ù锝呮惈椤庢挾绱撳鍕獢鐎殿喖顭锋俊鎼佸Ψ閵忊剝鏉搁梻浣虹《閸撴繈銆冮崱妞绘灁闁靛鍎弨鑺ャ亜閺冣偓椤戞瑥顭囬幇顓滀簻闁靛鍎婚煬顒傗偓娈垮枛椤兘寮幇顓炵窞閻庯綆浜欏Ч妤呮⒒娴ｄ警鐒剧紒缁樺姍楠炲﹪骞樼拠鍙夋К闂佸憡绋戦悺銊╁煕閹达附鐓曟繛鎴烇公閺€濠氭煟閹惧崬鍔ょ紒杈ㄥ笚瀵板嫭鎯旈敍鍕缚缂?ChatGPT internal Codex endpoint闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灪閵囧嫰濡搁妷顖濆惈閻庢鍠涢褔鍩ユ径濠庢僵妞ゆ劧绲芥刊浼存⒒娴ｇ瓔鍤冮柛銊ラ叄瀹曠喖顢楅埀顒勫礉鎼淬劍鈷掗柛灞剧懅缁愭棃鏌嶈閸撴盯宕戝☉銏″殣妞ゆ牗绋掑▍鐘炽亜閺嶎偄浠﹂柛瀣ㄥ妽閵囧嫰寮介妸锝勭捕闂佺顑嗛幐鎼佹偩閻戣棄鐐婇柕濠忓瘜閸熷骸鈹戦悩鍨毄濠殿喗鎸抽弫鍐Χ閸ワ絽浜炬慨妯煎帶楠炴鏌涢幒鎴含妤犵偞锕㈤、娆撴偩鐏炶棄绠洪梻鍌欒兌缁垶寮婚妸鈺佺妞ゆ劧璐熼埀顒€鍊垮畷妤呭礂閻撳骸浼庨梻浣瑰缁诲倿骞婅箛娑樼闁规壆澧楅悡鍐偡濞嗗繐顏╅柣蹇旀尦閺岀喖顢欓悾灞惧櫚闂佺娅曠划鎾澄涢崘銊㈡婵炲棙鎸婚弳妯肩磽閸屾艾鈧悂宕愬畡鎳婂綊宕惰閸ゆ洟鏌熼幑鎰【闁搞劍绻冪换娑㈠幢濡鏆楅悗瑙勬尫閻掞附绌辨繝鍥ч柛灞剧煯婢规洘淇婇悙顏勨偓銈夊储瑜版帒绀夐柟瀛樼箘閺嗭附鎱ㄥ璇蹭壕闂佽桨鐒﹂崝娆忕暦閸楃偐鏋庨幖杈剧稻鐎垫粓姊婚崒姘偓鎼佸磹閻戣姤鍤勯柛鎾茬閸ㄦ繃銇勯弽顐粶闁搞劌鍊块弻娑㈠箻閼碱剙濡界紓浣哄У鐢偟妲愰幘璇茬＜婵炲棙鍔楅妶鏉款渻閵堝棙鑲犻柛蹇曞亾缁嬫垿鍩㈡惔銊ョ婵犮垹瀚哥粻鎾诲蓟濞戙垹鍗抽柕濞垮劤娴犫晠姊洪崨濠傚闁稿鎹囧缁樼瑹閳ь剙顭囪閳ワ箓顢橀悩鑼瓘闂佸吋绁撮弲娑㈠垂濠靛鐓曟い鎰剁稻缁€鍐煃闁垮鐏╃紒杈ㄥ笧閳ь剨缍嗛崑鎺楀磿閵夆晜鐓曢幖娣灩婵秹鏌″畝鈧崰鏍х暦閵婏妇绡€闁告劑鍔夐崑鎾诲箛閻楀牏鍘靛銈嗘磵閸嬫挾绱掗悩宕囧⒌妤犵偛鍟…銊╁醇閻旈妾┑鐘灱濞夋盯鎳熼鐐茬劦妞ゆ帒鍊告禒杈ㄦ叏婵犲啯銇濇鐐村姈閹棃鏁愰崶鈺傛闂傚倷鑳堕幊鎾诲疮鐠恒劎鐭撻柣銏㈡暩閸楁岸鏌ｉ弮鍌氬付闁告濞婇弻鏇＄疀婵犲喚鈧棝鏌熼悿顖涱仩缂佽鲸鎹囧畷鎺戔枎閹存繂顬夐梻浣瑰瀹€鎼佸蓟閿濆牏鐤€婵娉涢幗鍨箾閿濆懏鎼愰柨鏇ㄤ邯閵嗕線寮撮姀鐙€娼婇梺缁樼憿閸嬫挸霉鐏忔牕浜鹃梻鍌氬€烽懗鍫曞箠閹炬剚鍤曢柛顐ｆ礀缁犱即鏌熼幆鐗堫棄闁哄绶氶弻娑樼暆閳ь剟宕戦悙鐑樺亗闁靛璐熸禍婊堟煛瀹ュ啫濡块柣鎿冨灠椤法鎲撮崟顒傤槬闁剧粯鐗犻弻锝咁潨閳ь剙顭囪缁傛帒顭ㄩ崼鐔哄幗闂婎偄娲﹀鑽ょ不閻愮鍋撶憴鍕缂佽鍊块敐鐐测攽鐎ｎ偄娈濋梺姹囧灲濞佳囩嵁濡や胶绡€鐎典即鏀卞姗€鍩€椤掍焦宕岄柟铏殜瀹曞ジ寮撮悢宄板濠电姷鏁告慨鎾疮閵婏妇顩叉繝濠傚娴滄粓鏌熼幑鎰【濞寸媴绠撻弻娑㈡偐閹存劖鍨块妴鍐Ψ閳哄倸鈧兘鏌涘▎蹇ｆЦ闁哄濞€濮婅櫣鎷犻垾铏闂佹悶鍎滈崶褎鏆梻鍌欑窔濞佳呮崲閹烘挻鍙忛悗闈涙憸娑撳秴鈹戦悩宕囶暡闁抽攱甯￠弻娑氫沪閸撗勫櫙闂佺绻愰惉鑲╂閹烘鏁嬮柛娑卞幘娴犵螖閻橀潧浠﹂柛鏃€鐟ラ悾鐑藉Ω閿斿墽鐦堥柟鍏肩暘閸╁嫰宕澶嬧拻濞达絽鎽滅粔鐑樹繆椤愩儲纭剁紒顔肩墛閹峰懘骞囬悢婵嗘搐閻顭跨捄铏圭伇妞ゆ梹娲熷铏瑰寲閺囩偛鈷夐柦鍐憾閹绠涙惔鈥崇ギ闂佸搫鏈粙鎺旀崲濠靛绀嬫い鎺嶇劍閻︽挻绻濆▓鍨灈闁挎洏鍊濋垾锕€鐣￠幍顔芥婵犻潧鍊搁幉锟犲磻閸曨垱鍋犳繛鎴炲笒婢у弶銇勯銏⑿ょ紒杈ㄦ崌瀹曟帒鈻庨幒鎴濆腐闂佽瀛╅懝鍓ф崲濡櫣鏆﹂柕濞炬櫓閺佸秵绻濇繝鍐ㄥ閹?	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼达絾鏆╂い顓炵墛娣囧﹥绂掔€ｎ偀鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜閻犲洩灏欐晶锔锯偓娈垮枛椤嘲顕ｉ幘顔藉亜濡炲娴烽悰顕€姊绘担铏广€婇柛鎾寸箚閹筋偊姊虹紒妯肩畺婵炶尙鍠庨～蹇涙惞閸︻厾鐓撳┑鐐叉閸庢娊宕滈弶娆炬富闁靛牆绻愰々顒勬煛娴ｇ瓔鍤欐い鏇悼閹风姴霉鐎ｎ偒娼旈梻渚€娼х换鍡涘疾濠婂牆鐤炬繝闈涱儐閳锋垿鏌熺粙鎸庢崳缂佺姵鎸绘穱濠囶敃閿濆洦鍒涢柦妯荤箞閺屾洘寰勯崱妯荤彆闂佹娊鏀遍崹鍓佹崲濞戙垺鍤戝Λ鐗堢箓濞堫厼螖閻橀潧浠︽い銊ユ婵＄敻宕熼姣尖晠鏌曟径娑樼槣婵☆偓闄勭换婵堝枈婢跺瞼锛熼梺杞版祰椤曆囨偩閻戣姤鍋勭痪鎷岄哺閺咁剙鈹戦鏂や緵闁告挻鐟╁顐﹀Χ婢跺鎷绘繛鎾村焹閸嬫挻绻涙担鍐叉濞咃綁姊绘担鍛婂暈濞撴碍顨婂畷褰掑础閻愭劖甯楀鍕箾閻愵剚鏉搁梻浣虹帛钃辨い鏃€鐗犲鎶筋敍閻愬鍘撻悷婊勭矒瀹曟粌鈻庨幘宕囨焾闁荤姵浜介崝搴∥涢婊勫枑閹艰揪绲惧畷鍙夌箾閹存瑥鐏柛銈嗗灦閵囧嫰骞掗崱妞惧闂備礁鎼鍡欑矙閹捐鐒垫い鎺嶇贰閸熷繘鏌涢悩宕囨创妞ゃ垺淇洪ˇ褰掓煕閳瑰灝鍔滅€垫澘瀚伴獮鍥敆婢跺绉遍梻鍌欒兌缁垵鎽銈嗘⒐閻楃姴鐣烽鍕€绘俊顖濆亹閻﹀牓姊哄Ч鍥х伈婵炰匠鍥х厺闁割偆鍠撶粻鎯归敐鍛毐闁告碍锚閳藉濮€閻樼數娼夐梻浣侯焾閺堫剛鎹㈤崒姘辨瘓闂傚倸鍊峰ù鍥ㄦ叏閵堝鏅俊鐐€х€靛矂宕戞繝鍌ゅ殨?Codex/GPT 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄤ粙寮诲☉銏犖╅柕澶堝劗閸嬫捇宕稿Δ鈧拑鐔兼煥濠靛棭妲告い顐㈡嚇閺岋絽螣閻戞ǚ鏋欏┑鈽嗗亜閻倸顫忛悜妯诲闁规鍣Σ顔碱渻閵堝棗鐏ユ繛宸弮閻涱喗寰勯幇顒傤啇婵炶揪绲介幉锟犓夊┑瀣厽闁绘ê鍘栭懜顏堟煕閺傝法孝闁崇粯鎸婚妶锝夊礃閳轰椒鐢绘繝鐢靛Т閿曘倝骞婃惔銊ｂ偓鍌涚附閸涘﹤浠哄銈嗙墬椤ㄥ棗鈻嶆繝鍕ㄥ亾濞堝灝娅橀柛瀣工閻ｇ兘骞掑Δ浣糕偓鐑芥倵閻㈡鐒炬鐐茬墛娣囧﹪鎮欓鍕ㄥ亾閺嶎厼绀夌憸蹇擃嚗婵犲啨鍋呴柛鎰╁妿閿涚喖姊洪崫鍕偍闁搞劍妞介幃陇绠涢幙鍐數闁荤娀缂氬▍锝嗘櫠閺囥垺鐓曢柡鍐ｅ亾闁挎洏鍨藉璇差吋閸偅顎囬梻浣告啞閹歌顫濋妸鈺佹瀬妞ゆ洍鍋撴鐐村笒铻栭悗锝庡墰閳笺倖绻濋悽闈涒枅婵炰匠鍥舵晞闁告侗鍨遍崗婊堟煃瑜滈崜鐔奉潖閾忓湱纾兼俊顖氭惈椤帒鈹戦悙鏉垮皟闁糕€崇箰閹浮 Key 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌″畝瀣М濠殿喒鍋撻梺闈涚箚閺呮繈宕濋崫銉х＝濞达絾褰冩禍鐐節閵忥絾纭鹃柡鍫墴瀹曚即骞囬幍顔煎絼闂佹悶鍎崝宥囩矆閳ь剟姊洪幖鐐插缂佽鍊块崺鐐哄箣閿旇棄浜归柣搴℃贡婵挳藟濠靛牏纾藉ù锝呮惈椤庢挾绱撳鍕獢鐎殿喖顭锋俊鎼佸Ψ閵忊剝鏉搁梻浣虹《閸撴繈銆冮崱妞绘灁闁靛鍎弨鑺ャ亜閺冣偓椤戞瑥顭囬幇顓滀簻闁靛鍎诲銉︺亜椤愩垻绠婚柟鐓庣秺瀹曠兘顢橀悩闈涘脯闂傚倷鑳剁涵鍫曞礈濠靛鈧啴宕ㄩ幖顓熸櫆闁诲孩绋掗…鍥╃不妤ｅ啯鐓曟い鎰╁€曢弸鎴︽煃闁垮绗х紒杈ㄥ笚濞煎繘濡搁妷锕佺檨闂備胶纭堕弬渚€宕戦幘鎰佹富闁靛牆妫楃粭鎺楁倵濮樼厧澧撮柣娑卞櫍婵偓闁炽儴灏欑粻姘渻閵堝棛澧紒璇插€圭粋宥囨喆閸曗晙绨婚梺鍝勫暊閸嬫挻绻涢懠顒€鏋庢い顐㈢箻閹煎綊鎮烽弶娆惧殭闂備礁鎼ú銊╁磻濞戙垹鐓曞璺侯煬濞撳鏌曢崼婵嗘殭濠碘€炽偢閺岋絽螖閳ь剛鎹㈤幒妤€绀嗛柟鐑樻尵缁♀偓濠殿喗锕╅崜娑㈩敊閸涘瓨鈷戦柛蹇涙？閼割亪鏌涙惔銊ゆ喚闁糕斁鍋撳銈嗗笒閸婂綊寮抽鍕厸鐎光偓鐎ｎ剛袦闂佽鍠撻崹浠嬪箖閳╁啯鍎熸繝闈涙閻庡啿鈹戦悩娈挎毌闁逞屽墮閸熻法鐥閺岀喖顢欓悡搴樺亾閸ф宓佸┑鐘叉处閺呮彃顭跨捄鐚村姛闁汇倐鍋撻梻鍌欒兌缁垶銆冮崨瀛樺亱闊洦绋戦崒銊╂⒑椤掆偓缁夌敻鍩涢幋锔解拻闁割偆鍠嶇欢杈ㄤ繆閻欐瑥鎳愮壕濂告煠閼圭増纭剧悮姘箾閿濆懏鎼愰柨鏇畵楠炴垿宕熼姣尖晝鎲歌箛娑欐櫖婵犲﹤鐗婇埛鎴犵磽娴ｇ櫢渚涢柣鎺曞Г閵囧嫰骞掗幘璺哄绩闂佺娅曢幑鍥х暦缁嬭鏃€鎷呴崫鍕闂佽楠搁崢婊堝磻閹剧粯鍊甸柨婵嗛婢т即鏌ｉ敃鈧悧鎾诲箖濡ゅ啯鍠嗛柛鏇ㄥ墰椤︻喚绱撴笟鍥ф灈濠⒀呮櫕閸掓帞鎷犲ù瀣潔闂侀潧绻掓慨鐑筋敊閹烘鈷戦柛锔诲幘鐢盯鎮介妤佹珔閾荤偞绻濇繝鍌滃闁抽攱甯￠弻娑氫沪閹规劕顥濋梺閫炲苯澧柟顔煎€搁悾鐑藉箛椤撗勑ч柟鑹版彧缁插潡鎮為崗鑲╃闁圭偓娼欓悞褰掓煕鐎ｎ偅灏版い銊ｅ劦閹瑥顔忛瑙ｆ瀰闂備浇妗ㄩ悞锕傚箲閸ヮ剙绠栭柍鍝勬媼閺佸啴鏌ｉ弮鍥ㄨ吂濠㈣娲熷娲箰鎼淬垻顦ラ梺绋匡攻閹倿骞冨▎鎰瘈闁告劧缂氱花濠氭⒑閻熺増鎯堟俊顐ｎ殕缁傚秹宕滆绾捐棄霉閿濆洦鍤€濠殿喖鐗婇妵鍕Ω閵夘垵鍚梺缁樹緱閸ｏ絽顕ｆ禒瀣垫晣闁绘劖顔栭崯鍥ㄤ繆閻愵亜鈧牠骞愰悙顒佸弿閻庨潧鎲￠弳婊堟煏婵炑冩噽閿?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻樻爠鍥ㄧ厱閻忕偛澧介悡顖氼熆鐟欏嫭绀€闁宠鍨块、娆戠驳鐎ｎ剙濮洪梻浣告啞椤棝宕惰閸ゆ垿姊虹紒妯荤叆闁告艾顑夐幃鈥斥枎韫囧鍋撻幒鎴僵妞ゆ帒鍊烽搹搴㈢節濞堝灝鏋涢柛濠傜秺閵嗗啴濡烽埡鍌氣偓鐑藉级閸喎绀冮柍褜鍓氱€笛囧Φ閸曨垰顫呴柨娑樺閸ｄ即姊洪崷顓х劸闁挎洏鍎遍銉╁礋椤掑倻鐦堥梺绋胯閸婃牠藟婢舵劖鈷掗柛灞捐壘閳ь剚鎮傚畷鎰版倻閼恒儱娈戦梺鍛婃尫缁€渚€宕瑰┑鍥ヤ簻闁哄稁鍋勬禍褰掓煃瑜滈崜銊х不閹惧磭鏆﹀┑鍌滎焾鍞銈嗘婵倕鈻嶉弮鍫熺厽闁绘柨鎽滈惌濠勭磼缂佹﹫鑰块柣娑卞枟瀵板嫰骞囬鍌ゅ晪婵＄偑鍊栧Λ浣规叏閵堝應鏋嶉柛銉墯閳锋垹绱掔€ｎ偄顕滄繝鈧悧鍫熷弿婵☆垳顭堟慨鍌涖亜閵忥紕鎳囨鐐叉喘閹囧醇閵忕姴绠版繝鐢靛О閸ㄧ厧鈻斿☉銏℃櫇闁挎棁濮ゅ▍鐘绘煛閸モ晛鍓遍柣鏂挎閹叉瓕绠涘☉妯兼煣濡炪倖鍔戦崐鏇㈠垂濠靛洨绠鹃柛鈩兠慨澶岀磼閳锯偓閸嬫挻绻濆▓鍨灍闁挎洍鏅犲畷銏°偅閸愩劌鍋嶆繛瀵稿Т椤戝棝鍩涢幋锔界厵缂佸瀵ч幑锝囩磼閻樿櫕宕岄柡宀嬬到铻栧ù锝囨嚀绾炬娊鎮楃憴鍕闁稿锕ら悾鐑筋敃閿曗偓缁€瀣煕閺囥劌浜伴柛娆忓閳ь剝顫夊ú姗€宕曟總鍢庛劑宕掗悙鎼濡炪倖甯掗ˇ顖涙櫠椤栫偞鐓忛柛銉戝喚浼冩繝娈垮枓閸嬫捇姊洪幐搴ｂ槈閻庢皜鍏撅綁宕煎┑鎰瘜闂侀潧鐗嗙花鑲╄姳婵犳碍鐓曢柣鏃囨硾瀹撳棙顨ラ悙鑼闁诡喒鏅犻幊锟犲Χ閸涱厾瀵煎┑锛勫亼閸婃牠骞愭ィ鍐ㄧ？闁归偊鍎靛☉姗嗙叆闁割偆鍟块幏娲煟閻樺弶鎼愮€殿喖鐖煎畷顒佸緞鐎ｃ劋绨诲銈嗗姂閸╁嫬危閹间焦鍋傞柕鍫濐槹閻撴稓鈧箍鍎卞ù閿嬬鐟欏嫮绠剧€瑰壊鍠曠花濂告煕鐎ｎ偄濮嶉柟顔肩秺瀹曞爼顢旈崟顓燁嚄闂備焦瀵х粙鎺楁儎椤栨凹娼栨繛宸簼閸嬶繝鏌℃径濠勬皑闁圭鍟扮槐鎾寸瑹閸パ勭彯闂佹悶鍔忓▔娑㈡偩閻戣棄绠涙い鎾跺Х閻﹀牆鈹戦鏂や緵闁告﹢绠栧畷锝夊礃閵娿垺鏂€闂佸疇妫勫Λ妤呮倶閻斿吋鍋ｉ柍褜鍓熼弫鍐焵椤掆偓瀹撳嫰姊洪崗鑲┿偞闁哄懏绮撳畷鎴﹀冀閵娧呯槇闂傚倸鐗婃笟妤呭磿閹扮増鐓熼柟鎹愭硾閺嬫稒顨ラ悙瀵稿婵炵厧绻樺畷婊嗩槾闁挎稓鍠栧鐑樺濞嗘垹校闂佸憡鐟ラ崯鏉戭嚕鐠囨祴妲堟繛鍡樺姇閸斿懘姊洪棃娑氬閻庢凹鍓熼獮鍡涘醇閵夛腹鎷洪梺鑽ゅ枑婢瑰棝鎮鹃銏＄厱闁靛鍊楅惌娆撴煟濞戝崬娅嶆鐐村笒铻栭柍褜鍓涢悮鎯ь吋婢跺鍘遍柣蹇曞仜婢т粙鎯屽畝鍕厱?
	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒妯哄婵犫偓娓氣偓濮婅櫣绮欑捄銊ь唶闂佸憡鑹鹃鍥╂閻愬搫绠ｉ柣妯虹仛閿涘繘姊虹拠鈥崇€婚柛鏇ㄥ亗濞ｎ喗绻濆▓鍨灈闁挎洏鍎遍—鍐╃鐎ｃ劉鍋撴担鑲濇棃宕ㄩ鐘插Е婵＄偑鍊栫敮鎺楀磹婵犳碍鍎楅柟鍓х帛閸婄敻姊婚崼鐔衡姇闁哥喓鍋ら弻娑㈡偐閸欏妫﹂梺鍝勮閸旀垵顕ｉ幘顔藉€锋繛鏉戭儏娴滃墽鎲搁悧鍫濈瑨缁炬儳缍婇弻锝夊閻樺啿鏆堥梺缁樻尪閸庤尙鎹㈠┑瀣棃婵炴垶鐟Λ锕€顪冮妶鍛劉妞ゃ劌锕璇测槈濡粎鍠撶槐鎺懳熼崫鍕暘闂傚倷绀侀幖顐﹀嫉椤掑倻鐭欓柟鎹愵嚙閻掑灚銇勯幒鎴姛缂佸鏁婚弻娑氣偓锝庝簼閸ｅ綊鏌ｉ敐鍛Щ闁伙絾绻堥崺鈧い鎺戝閻掑灚銇勯幒鎴濇灓婵炲吋鍔栫换娑㈠矗婢跺瞼鐓夐悗瑙勬礃椤ㄥ懘锝炲鍫濈劦妞ゆ帒瀚繚婵炶揪缍€閸婁粙鎮㈢亸浣圭€婚梺缁樺姦閸庣兘顢旈崼鐔叉嫽婵炶揪缍€婵倝濡撮崘顏嗙＜閻犱礁婀辩弧鈧悗瑙勬礀濠€閬嶅箲閸曨剛鐟规い鏍ㄧ〒缁嬩焦绻濋悽闈涗粶婵☆垰锕ョ粋宥咁煥閸涱垯绗夐悷婊呭鐢鎮￠悢鍏肩厵闂侇叏绠戦獮姗€鏌曢崼鐔稿唉闁哄本绋撻埀顒婄秵閸嬪懎鐣峰畝鈧埀顒侇問閸犳洜鍒掑▎鎾扁偓渚€寮撮姀鐙€娼婇梺缁樏崥鈧柛鐘诧躬濮婄粯鎷呴悜妯烘畬闂佸憡鑹鹃澶愬箖妤ｅ喚鏁傞柛顐ｇ箘椤︻噣鎮楅崗澶婁壕闂佸憡娲﹂崜娆撳焵椤掆偓濞硷繝寮诲☉鈶┾偓锕傚箣濠靛懐鎸夊┑鐐茬摠缁秶鍒掗幘璇茶摕闁绘柨鍚嬮崐缁樹繆椤栨繃顏犲ù鐘虫尦濮婃椽宕崟闈涘壄闂佺锕ラ悧婊堝箲閵忕姭妲堥柕蹇曞Х椤ρ囨⒑缂佹ɑ灏Δ鐘虫倐閿濈偤骞掑Δ浣叉嫼闂佸憡绻傜€氼噣鎮炵捄銊х＝鐎广儱瀚粣鏃堟煏閸℃洜顦﹂摶锝嗙箾閸℃瑥浜炬禍娑㈡⒒閸屾瑧顦﹂柟纰卞亰椤㈡牠宕橀鑲╋紮濠德板€曢崯顖烇綖?base_url 闂?OpenAI-compatible 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼达絾鏆╂い顓炵墛娣囧﹥绂掔€ｎ偀鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜閻犲洩灏欐晶锔锯偓娈垮枛椤嘲顕ｉ幘顔藉亜濡炲娴烽悰顕€姊绘担铏广€婇柛鎾寸箚閹筋偊姊虹紒妯肩畺婵炶尙鍠庨～蹇涙惞閸︻厾鐓撳┑鐐叉閸庢娊宕滈弶娆炬富闁靛牆绻愰々顒勬煛娴ｇ瓔鍤欐い鏇悼閹风姴霉鐎ｎ偒娼旈梻渚€娼х换鍡涘疾濠婂牆鐤炬繝闈涱儐閳锋垿鏌熺粙鎸庢崳缂佺姵鎸绘穱濠囶敃閿濆洦鍒涢柦妯荤箞閺屾洘寰勯崱妯荤彅缂?
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍嵁瀹ュ鏁婄痪鎷岄哺濮ｅ姊绘担渚劸妞ゆ垶鍨归幑銏犫攽鐎ｎ亣鎽曢梺鍝勬川閸犳捇鎮甸懜鐐逛簻闁哄稁鐓堝▓鏂棵瑰鍫㈢暫婵﹨娅ｇ划娆忊枎閹冨闂備胶鎳撻幉锟犲箖閸岀偞鏅查柣鎰ゴ閺€浠嬫倵閿濆骸浜滃ù婊勵殜閺岀喖鎳濋悧鍫濇锭缂備焦褰冮妶绋跨暦閻樿鍐€妞ゆ挾鍟块幏缁樼箾閹炬潙鐒归柛瀣崌閺屽秷顧侀柛鎾寸洴瀹曟垵鈽夐姀鈥虫濡炪倖鐗楃粙鎺戔枍閻樼偨浜滈柡鍥殔娴滈箖姊洪崫鍕効缂傚秮鍋撶紓浣哄У閻╊垰顕ｉ鍕瀭妞ゆ棁濮ら崵鍫ユ⒒閸屾瑦绁版い鏇熺墵瀹曨垶顢曢敃鈧悙濠囨煏婵炲灝鍔欐慨瑙勵殜濮婄粯鎷呴挊澶夋睏闂佸啿鍢查悧鎾崇暦濠婂喚娼╂い鎴ｅГ閻?reasoning.effort 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锕傛煟閵忕媭鐓兼慨濠勭帛缁楃喖鍩€椤掆偓椤洩顦归挊婵囥亜閹板墎鐣遍柣銈囧亾缁绘盯骞嬮悙璺侯棟濡炪們鍎插畝鎼佸蓟濞戙垺鏅滈悹鍥ㄥ絻缁犳椽姊洪崫銉バｉ柛鏃€鐟╁濠氭晲閸涘倹妫冮崺鈧い鎺戝閸嬪鏌涢埄鍐噮缂佺姵妫冮弻鐔兼倻濡闉嶉梺鍛婄懃缁绘﹢骞冨畡鎵虫瀻闊洦鎼╂导鈧梻浣告啞閻熴垽宕戦幘缁樷拻濞达絽鎲￠幆鍫ユ煟椤撶儐妲虹紒杈╁仦閹峰懘宕滈幓鎺擃吙婵＄偑鍊栫敮鎺斺偓姘煎墴瀹曞綊宕掑☉鏍︾盎闂佸搫鍟ú锕偹夋径鎰厓鐟滄粓宕滈敃鍌氬瀭閻犺桨璀﹀鏍ㄧ箾瀹割喕绨诲ù鑲╁█閺屾盯寮撮妸銉ヮ潻濡炪倕瀛╅惄顖氼潖濞差亜宸濆┑鐘插暙閺嗘姊洪崫銉バｉ柣妤佹礋椤㈡岸鏁愰崶銊ョ彴濠电偞娼欓鍡涳綖瀹ュ應鏀介柣妯款嚋瀹搞儵鎮楀☉鎺撴珚妞ゃ垺鎹囧畷鍫曨敆娴ｅ弶瀚藉┑鐐舵彧缁蹭粙骞夐敓鐘茬畾闁割偁鍎查悡娆撴煣韫囷絽浜滃┑顔兼喘閺屻劑寮村Ο铏逛患濡炪倖娲橀悷褔宕犻弽顐ょ缂?-> none闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊洪悙钘夊姎缁剧虎鍘奸敃銏ゅ箻椤旂晫鍘遍梺鍝勫暊閸嬫捇鏌涢妸銉т虎闁伙絽鍢茬叅妞ゅ繐瀚崝锕€顪冮妶鍡楃瑐闂傚嫬绉电粋宥咁煥閸喓鍘甸梺缁樺灦閿氶柣蹇嬪劦閺屽秷顧侀柛鎾寸箓閻ｇ兘宕归銈囧骄闂佸搫娲ㄩ崰鎾绘⒔閸曨厾纾奸悗锝庡幗绾泛螖閻樺弶鍠樻慨濠冩そ瀹曟粓骞撻幒宥囧嚬缂傚倷绀侀鍡涘垂閼哥數鈹嶅┑鐘插瀹曞鏌曟繛褍鎷戠槐鏌ユ⒒娴ｈ櫣甯涢柨姘辩棯缂併垹骞楅柡鍛板煐濞煎繘鍩￠崘顏庣床闂佸搫顦悧鍕礉瀹€鈧划顓☆樄闁哄苯绉烽¨渚€鏌涢幘鍗炲缂佽京鍋ゅ畷鍗炍熺喊杈ㄩ敜闂佽崵濮惧銊х礊瀹ュ鏅查柛顐ｇ◥濮规姊洪崷顓炲妺闁规悂绠栧畷銏ゆ濞戣鲸瀵岄梺闈涚墕濡稒鏅堕鍌滅＜閻庯綆鍋勫ù顕€鏌熼鍡欑瘈濠碉紕鍏橀崺锟犲磼濠婂啫绗氶梻浣藉吹閸犳劙鎮烽妷褉鍋撳鐓庡⒋闁诡喗顭囬埀顒傛暩绾爼宕戦幘鑸靛枂闁告洦鍓涢ˇ銉╂⒑闂堟稓澧曢柟璇х節楠炲牓濡搁埡浣哄姦濡炪倖甯掔€氼參鎮￠崘顔界厪濠㈣泛鐗嗘俊浠嬫煟濠垫劒绨肩紒缁樼洴瀹曞ジ鏁愰崨顖氬缚缂傚倷鑳剁划顖滄崲閸儱绠栧ù鐘差儐椤ュ牊绻涚壕瀣牚闁稿海鏁诲濠氬焺閸愨晛顎撻柣鐔哥懃鐎氼剟鍩€椤掍礁濮夐柍褜鍓濋～澶娒哄鈧畷褰掑锤濡も偓缁犳牗绻涢崱妯诲碍濞磋偐濞€閺屾盯寮撮妸銉ュ闂佸摜鍋犲▔娑㈠煘閹寸偛绠犻梺绋磕涢崨顓犵瓘婵炲濮撮鍛存偪椤曗偓閺岋絽螣濞嗘儳娈梺缁樺笒閻忔岸濡甸崟顖氱闁规惌鍨版慨娑㈡⒑娴兼瑧鎮奸柡浣规倐閸┾偓妞ゆ帒鍠氬鎰箾閸欏澧靛┑鈥崇埣瀹曘劍绻濇担铏圭暰闂備胶绮崹鐓幬涢崟顐ｆ殰闂傚倷绶氬褔藝椤撱垹纾垮ù鍏兼綑缁犵喎螖閿濆懎鏆為柍閿嬪浮閺屾稓浠﹂崜褎鍣梺绋跨箰閻倿寮婚悢鍏兼優妞ゆ劧绲界壕鍐差渻閵堝啫鐏€光偓閹间降鈧礁顫滈埀顒勫箖閵忋垻鐭欓柛顭戝枙缁辩喖姊婚崒娆戭槮濠㈢懓锕幃锟犲醇閵夈儳锛涢梺鍛婃处閸ㄤ即宕归崒鐐寸厸闁告劑鍔庢晶娑㈡煟閹捐泛校缂佺粯鐩鍫曞箣椤撶偛澹夋繝鐢靛仜閹冲骸煤閵娾晜鍎夋い蹇撶墕缁犳氨鎲哥€ｎ喖纾婚柕蹇嬪€栭悡娆愩亜閺冨倻鎽傛繛鍫熺矒閺屸剝鎷呴悷鏉款潔闂佽鍨遍弻銊╁煘閹达箑閱囬柣鏂挎啞椤忥絾绻濋悽闈浶ｆい鏃€鐗犲畷鏉课旈崟顐ょ☉闂傚倷绀侀幉鈥愁潖瑜版帒鍨傞柛顭戝櫘閸ゆ洟鎮归崶銊с偞婵℃彃鐗婇幈銊ヮ潨閸垹鍩屾繛瀛樼矒缁犳牠寮婚妸銉㈡斀闁糕剝渚楅埀顒侇殔闇夋繝濠傚缁犳﹢鏌嶈閸撴繈锝炴径濞掓椽寮介鐔峰壒闂佺鐬奸崑娑㈡嫅閻斿吋鐓熼柡鍐ㄥ€甸幏锟犳煛娴ｉ潻韬柡宀嬬秮閹垽宕ㄦ繝鍕殥闂備礁鍟块崲鏌ニ囬棃娑辨綎闁惧繗顫夌€氭岸鏌嶉妷銉э紞濞寸媭鍘奸埞鎴﹀煡閸℃ぞ绨婚梺鍝ュ枑閹歌櫕淇婄€涙ɑ濯寸紒顖涙礃閻庡姊虹憴鍕姢缁剧虎鍙冭棟鐟滃繒妲愰幒妤佸€锋い鎺嶈兌閸戯繝鎮楃憴鍕闁告挾鍠庨锝夊蓟閵夛箑浜瑰┑鐐存綑椤戝棝寮埀顒勬⒒娴ｈ櫣甯涢拑閬嶆煕閹炬潙鍝虹€规洦鍨电粻娑樷槈濞嗘垵骞堥梻浣虹帛钃遍柛鎾磋壘閳绘挸顫滈埀顒勫蓟閳╁啰鐟归柛銉戝嫮褰庨梻浣筋嚃閸犳鎮烽埡浣勬盯宕橀妸銏☆潔濠碘槅鍨跺Λ鍧楀窗婵犲倵鏀介柨娑樺娴滃ジ鏌涙繝鍐⒌闁诡啫鍐剧叆闁割偅绻勯敍娑樷攽閻愭潙鐏︽慨妯稿姂閹矂宕卞Δ濠勫數闂佸吋鎮傚褎鎱ㄩ崼銉︾厽闁规儳顕幊鍥┾偓娈垮枛閻栫厧鐣烽悡搴樻斀闁糕€崇箲鐎垫牠姊绘担椋庝覆缂傚秳鐒︾粋宥夊醇閺囩偠鎽?
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

	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€跨€殿喖顭烽幃銏ゆ惞閸︻叏绱查梻渚€娼х换鎺撴叏閻㈠憡鍊堕柛顐犲灮绾捐棄霉閿濆懏鎯堥崯鍛婄節閻㈤潧浜归柛瀣尭铻栭柣姗€娼ф禒锕傛煟濡や胶鐭掔€规洘宀搁獮鎺楀籍閳ь剟宕伴崱娑欑厱闁哄洨鍋熸禒娑氱磽瀹ュ懐澧︽慨?WSv2 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣濠垫劖缍楅梺閫炲苯澧柛濠傛健楠炴劖绻濋崘顏嗗骄闂佸啿鎼鍥╃矓椤旈敮鍋撶憴鍕８闁告梹鍨甸锝夊醇閺囩偟顓哄┑鐘绘涧閻楀啴宕戦幘娲绘晣闁绘垵妫欑€靛矂姊洪棃娑氬闁硅櫕鍔楃划缁樺鐎涙鍘藉┑掳鍊愰崑鎾翠繆椤愶絿绠炴鐐插暣閹瑩宕崟顐も偓顓烆渻閵堝棗濮夊┑顔肩－閼鸿鲸绻濆顓涙嫼闂佽崵鍠撴晶妤呭箚閸喍绻嗘い鎰剁秵濞堟洜绱掗崒姘毙х€规洘绮忛ˇ瀵哥棯閹佸仮闁哄本鐩獮妯何旈埀顒€螞濞嗘搩鏁佹俊銈呮噺閳锋垿鏌涘☉姗嗙劦闁硅揪绠戠壕鍧楁煙閹増顥夐柣鎾偓鎰佺唵闁兼悂娼ф慨鍫ユ煕鐎ｃ劌濡跨紒杈ㄥ笧閳ь剨缍嗛崢鐣屾兜閸洘鐓曢柡鍐╂尵閻ｈ鲸銇勯鍕殻濠碘€崇埣瀹曞崬螖閸愵亝鍟伴梻鍌欒兌閸樠囧疮閹稿孩娅犻幖杈剧到閸ㄦ繄绱撴担楠ㄦ粓宕戦崨瀛樼厱闁硅埇鍔嶅▍鍥煕濞嗗繑顥滈柍瑙勫灴閹晝绱掑Ο濠氭暘婵犵妲呴崑鍛存偡閳轰胶鏆﹂柟鎯版娴肩娀鏌涢弴鐘虫毄闁哥偟鏁婚弻锝夋偐閸欏鍎撻梺鍦拡閸嬪﹪銆侀弮鍫晝闁靛牆娲ㄩ敍婊堟煛婢跺﹦澧戦柛鏂跨灱缁叀顦归柡宀€鍠愰ˇ鐗堟償閳锯偓閺嬪懎螖閻橀潧浠﹂柟鐟版喘閻涱噣骞掗幋鏃€鏂€闂佸憡渚楅崹鎶芥晬閻愮儤鈷掑ù锝囩摂閸ゆ瑩鎮楀☉鎺撴珚妞ゃ垺鐟╁浠嬵敇閻愮數宕舵繝寰锋澘鈧捇鎳楅崜浣轰笉闁诡垎鈧弨浠嬫煟濡搫绾ч柟鍏煎姍閺岋繝宕熼埡浣稿Е闂?previous_response_id闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴︽煙閼测晛浠滈柍褜鍓氶悧鐘茬暦濠靛鍐€妞ゆ挾鍊ｉ敃鍌涚厱闁哄洢鍔岄悘鐘绘煕閹般劌浜惧┑锛勫亼閸婃牠宕濋敃鈧…鍧楀焵椤掑倻纾兼い鏃囧亹閸╋絾鎱ㄦ繝鍌涙儓閻撱倝鎮归崶銊ョ祷缂佺姴鎼—鍐Χ閸愩劌顬堥梺鎸庢处娴滄粓顢氶敐鍡欑瘈婵﹩鍘藉▍婊勭節閵忥絾纭鹃柨鏇畵閹潡鍩€椤掑嫭鈷掗柛灞捐壘閳ь剚鎮傞幃褎绻濋崟顓犵厯闂佺鎻拋锝囩礊閺嶃劊浜滈柟鎵虫櫅閻掔儤绻涢幘鎰佺吋闁哄本娲熷畷鐓庘攽閸″繐浜鹃柣鐔稿閺嬪秹鏌涢妷顔煎闁绘挻娲熼悡顐﹀炊閵婏富妫栭梺鍏煎濞夋洟鍩€椤掍緡鍟忛柛鐘崇墵閳ワ箓鎮滈挊澶婄€俊銈忕到閸燁偆绮诲☉妯忓綊鏁愰崨顔兼殘闂佺顭崹璺侯潖閾忚鍏滈柛娑卞弾濡牓姊虹憴鍕仧濞存粎鍋涢銉︾節閸ャ劌浠奸柣蹇曞仧閸嬫挸鈻撴导瀛樷拺閻熸瑥瀚崝銈嗐亜閺囥劌骞樼紒顔剧帛閵堬綁宕橀妸褔鐛撻梻浣稿閸嬪懐鎹㈤崒鐐茬厱闁哄啫鐗婇悡鏇㈡煙閻戞ɑ灏扮紒璺哄级閵囧嫰顢橀悙鏉戜淮闂佽鍣ｉˉ鎾诲极椤曗偓椤㈡瑩宕归钘夊壍闂備浇顕у锕傦綖婢跺⊕鍝勵煥閸繃鐎柣鐘叉搐濡﹪宕甸弴銏＄厵缂備降鍨归弸娑欍亜椤愩垺鍤囬柡灞炬礉缁犳盯寮撮悙鎰╁灮缁辨帡骞撻幒瀣划濠殿喖锕︾划顖炲箯閸涱垱鍠嗛柛鏇ㄥ亗閸濇姊绘担渚劸閻庡灚甯″畷鍦崉閾忚娈鹃梺鍝勬储閸ㄦ椽鎮為懖鈹惧亾楠炲灝鍔氶柟铏姍钘濇い鏇楀亾婵﹥妞藉Λ鍐ㄢ槈濮橆剦鏆繝鐢靛仜閻即宕濇惔銏㈢彾闁哄洨鍠撻梽鍕煕濞戞﹫鍔熼柛妯绘崌閹嘲顭ㄩ崟顓犵厜閻庤娲樼划鎾诲箖閵忊槅妲归幖娣灪椤旀洘绻濋悽闈涗粶婵☆偅鐟╅獮蹇涘礃椤斿槈鈺呮煏婢舵ê鏋熸俊妞煎姂閹?WSv1闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊婚崟顐ｅ枠妞ゃ垺淇洪ˇ鏌ユ偂閵堝鐓ラ柡鍥殔娴滄儳顪冮妶搴″箹婵炴祴鏅濈划璇测槈濡攱顫嶅┑鈽嗗灣閸樠囩嵁閹邦兘鏀介柣鎰煐瑜把呯磼闊厾鐭欐鐐搭殔椤劑宕熼褎閿ゅ┑鐐差嚟閸樠囨偤閵娾晛鐓曢柟鐑樺殮瑜版帗鏅查柛顐ゅ櫏娴犫晠鏌ｉ姀鈺佺仭闁烩晩鍨跺璇测槈閵忕姈鈺呮煏婢诡垰鍟伴崢浠嬫⒒娴ｅ搫浠洪柛搴ゅ吹缁骞樺畷鍥ㄦ闂佸壊鍋呭ú锕傚极閸℃稒鐓冪憸婊堝礈閻旂厧绠栭柍銉︽灱閺€浠嬫倵閿濆骸浜為柛妯绘倐濮婃椽宕ㄦ繝鍌氼潎闂佸憡鏌ㄧ€涒晠寮茬捄琛℃闁靛骏绱曢崢浠嬫⒑鐟欏嫬鍔ら柛鐔锋健瀵娊鎮㈤崗鑲╁幍闂佽偐鈷堥崜锕傛倿閻愵兙浜滄い鎾寸矊婵倻鈧娲滈崢褔鍩為幋锕€閱囨い鎰跺強閵堝應鏀介柣妯诲墯閸熷繘鏌涢悩鎰佹疁闁轰礁鍟撮崺锟犲川椤撶媭妲堕梻浣瑰濞叉牠宕愯ぐ鎺戠９闁秆勵殕閻撱垺淇婇娆掝劅婵″弶鎸抽弻锟犲焵椤掍胶顩烽悗锝庡亞閸樿棄鈹戦埥鍡楃仴婵炲拑缍佸鎼佸礃椤垹鍞甸柣鐔哥懃鐎氭悂鎳撻崸妤佺厸鐎光偓鐎ｎ剛袦濡ょ姷鍋涘ú顓€佸Δ浣虹瘈闁告侗鍘介崕鎾绘⒒閸屾瑧顦﹂柟纰卞亰瀵敻顢楅埀顒傚弲濠碘槅鍨伴崥瀣暦閸欏绡€闂傚牊鍐婚崝鐔兼煟閵堝倸浜鹃梻鍌欑閹碱偄煤閵婏附鍙忛梻鍫熺▓閺嬫棃鏌熼梻瀵稿妽闁绘挾鍠栭弻锝夊箛椤掑娈跺銈忕稻閻擄繝寮婚妸鈺佸嵆婵☆垵娅ｆ禒鐓庮渻?	// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃缂侇噮鍨抽幑銏犫槈閵忕姷顓洪梺鍝勫暊閸嬫捇鏌涢妶鍛ч柡灞剧洴婵＄兘顢欓悡搴樻嫽闂備浇妗ㄧ粈浣该洪銏犺摕闁哄浄绱曢悿鈧梺鍝勬川閸婎偊濡烽敂杞扮盎闂佹寧妫侀褍鈻嶅澶嬬厵妞ゆ梻鐡斿▓婊堟煟濞戝崬娅嶇€规洖缍婇、娆撴偂鎼搭喗缍撴繝纰夌磿閸嬫垿宕愯閳ь剟娼ч惌鍌氱暦閻熸壆鏆﹂柛銉戝啰浜伴梻浣稿閸嬩線宕曢柆宥嗙厑闁搞儯鍔庣弧鈧梺闈涢獜缁辨洜绮婚幘鍓佺＝鐎广儱鎷戦煬顒侇殽閻愭彃鏆ｉ柛鈺佸瀹曟﹢鍩℃担绋课ら梻鍌欑劍鐎笛呮崲閸屾娑樷枎閹惧磭鐛ラ梺鍝勭▉閸樹粙鍩涢幒鎳ㄥ綊鏁愰崨顔兼殘闂佽鍨伴悧鎾诲蓟閻旈鏆嬮梺顓ㄧ畱閸撳爼鎮楃憴鍕缂侇喖鐭傞敐鐐测攽閸喎纾梺鎯х箰濠€閬嶅级娴犲鈷掑〒姘ｅ亾婵炰匠鍥ｂ偓锕傚醇閵夈儳锛熷┑鐐叉閹稿宕戦崟顖涚厽闁瑰浼濋鍫熷仧濞寸厧鐡ㄩ埛鎴︽煕閿旇骞橀悽顖涚洴閹粙顢涘Δ浣圭彅濡炪値鍘煎ú锔剧矉閹烘柡鍋撻敐搴′簮闁归绮换娑欐綇閸撗呅氬銈庡亜椤﹂潧鐣烽幋锔藉亹缂備焦顭囬崢杈ㄧ節閻㈤潧孝闁哥噥鍨崇划鍫⑩偓锝庡枟鐎电姵绻濋棃娑卞剱闁绘挾鍠栭弻鏇＄疀鐎ｎ亞鍔撮梺缁樺笒婢х晫妲愰幒妤佸亹鐎规洖娲ら埛鍫ユ⒑闁偛鑻晶浼存煙閾忣偅灏甸柍褜鍓氶崙褰掑储閻ｅ瞼鍗氶柣鏃囥€€閸嬫捇鏁愭惔鈩冪亪闂佸憡鍨规繛鈧鐐寸墪鑿愭い鎺嗗亾濠碘€虫健閺屾稑鈻庨幇顓炵３闂佸搫鏈惄顖涗繆閻戣姤鏅查柛娑卞弾閻庢挳姊绘担绛嬪殐闁搞劍澹嗛埀顒佸嚬閸撶喎锕㈡笟鈧铏圭磼濡浚浜幆鍥ㄥ閺夋垹锛涢梺闈涚箚閸撴繈宕ｈ箛鎾斀闁绘ɑ褰冮弳鐐烘煏閸ャ劎绠橀柍褜鍓濋～澶娒哄Ο鐓庡灊闁规崘顕х粻鐘绘煟濡粯銇熼柡浣告閺屽秷顧侀柛鎾寸洴瀵偊顢氶埀顒佷繆閻戣棄鐓涢柛灞绢殕鐎氬ジ姊绘担渚敯闁稿鍔欏畷鎴濃槈濞嗘劕寮块梺閫炲苯澧柍瑙勫灴閹瑥顔忛鍙囨⒑閻戔晜娅撻柛銊ョ埣閻涱噣宕橀埞鍨簼闂佸憡鍔忛弲娑㈠焵椤掆偓椤兘寮婚敃鈧灒闁绘艾顕粈鍡涙⒑闂堟丹娑㈠礋椤愶絿鈧箖姊虹拠鑼闁稿濮鹃。楣冩⒑閸濆嫮鐒块梻鍕婵＄敻宕熼鍓ф澑闂佽宕樺▔娑⒙烽埀顒勬⒒娴ｈ櫣甯涢柟姝岊嚙鐓ら柣鏃傚帶閽冪喖鏌ｉ弬鍨倯闁稿﹪鏀遍妵鍕箳閹搭厽效闂佷紮绲介悥鐓庮潖濞差亜浼犻柕澶堝剾閿濆鐓犻悷娆忓缁€鍐煟韫囨搩鍎愮紒缁樼箞閹粙妫冨ù韬插灪缁绘稓鎷犺閻ｇ敻鏌涢埡鍌滄创妤犵偞甯掕灃濞达絽鎼獮鍫ユ⒑鐠囪尙绠抽柛瀣⊕閺呰埖绂掔€ｎ亞鐤囬梺鍛婄☉閻°劑宕戦敐澶嬬厵闂侇叏缂氱花鑽も偓瑙勬尭濡繈寮婚敍鍕勃闁告挆浣插亾閹烘鐓冪憸婊堝礈閵娧呯闁糕剝顭囬々鍙夌節婵犲倻澧遍柡浣稿€归妵鍕箻鐠虹洅娑㈡煕鐎ｎ偅灏甸柟鍙夋尦瀹曠喖顢楅崒銈喰為梻鍌欑閹测€愁潖瑜版帗鍋嬫繝濠傜墕缁犳煡鏌曡箛鏇炐涢柡鈧禒瀣厱闁抽敮鍋撻柡鍛矒瀹曟捇妫冨☉杈ㄥ瘜闂侀潧鐗嗗Λ妤冪箔閸岀偞鐓犻柛鎰皺閸╋綁鏌涢埞鎯т壕婵＄偑鍊栫敮鎺椝囬娑欐珷閻庢稒蓱閸欏繐鈹戦悩鎻掓殲闁靛洦绻冩穱濠囶敃閵忕姵娈梺瀹犳椤︻垶锝炲┑瀣垫晢濠㈣泛顑嗛惁鎺撶節閻㈤潧啸妞わ綀妫勫嵄闁告稒娼欑壕濠氭煕濞戞鎽犻柛搴″閵囧嫰寮介顫捕閺?Codex CLI 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌＄仦绋垮⒉鐎垫澘瀚埀顒婄秵娴滄繈顢欓崨顓涙斀闁绘劕寮堕埢鏇灻瑰鍐煟鐎殿噮鍋婂畷鍫曨敆娴ｅ搫甯鹃梻濠庡亜濞诧箑煤閺嵮勬瘎闂傚倷绀侀幉锛勬崲閸愵喓鈧啯绻濋崒銈嗙稁缂傚倷鐒﹂…鍥偡瑜版帗鐓曢柕澶嬪灥閸犳艾顭囬懡銈囩＝闁稿本鐟чˇ锔姐亜閿曞倷鎲剧€殿噮鍋嗛幏鐘绘嚑椤掍焦顔曢梻浣告惈濞层垽宕归崷顓犱笉闁挎繂妫涚弧鈧梺闈涢獜缁辨洜绮婚幘鍓佺＝鐎广儱鎷戦煬顒侇殽閻愭彃鏆ｉ柛鈺佸瀹曟﹢鍩℃担绋课ら梻鍌欑劍鐎笛呮崲閸屾娑樷枎閹惧磭鐛ラ梺鍝勭▉閸樹粙鍩涢幒鎳ㄥ綊鏁愰崨顔兼殘闂佽鍨伴悧鎾诲蓟閻旈鏆嬮梺顓ㄧ畱閸撳爼鎮楃憴鍕缂侇喖鐭傞崺銉﹀緞閹邦剦娼婇梺缁橈耿濞佳勬叏閿旀垝绻嗛柣鎰典簻閳ь剚鐗曢蹇旂節濮橆剛锛涢梺鐟板⒔缁垶鎮￠弴鐔剁箚妞ゆ牗绻傞崥褰掓煟椤撶喎绗ч柍褜鍓濋～澶娒哄鈧畷褰掑锤濡ゅ啫绁﹀┑鈽嗗灥閸嬫劗澹曢崗闂寸箚妞ゆ牗绮岀敮鑸殿殽閻愯尙澧︽慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵犵數鍋涢惇浼村垂閽樺鏆︽繝闈涚墐閸嬫捇鏁愭惔鈥斥拤婵犳鍠楁繛濠囧蓟閿濆鏅查柛娑卞灣娴煎洨绱掗悙顒€鍔ら柕鍫㈩焾椤曪綁宕奸弴鐐哄敹濠电偞鍨堕敋妞ゎ剙鐗撳娲川婵犲嫮鐣奸梺绋跨昂閸婃繈鐛?WSv1 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鏉垮閹ジ鏌嶈閸撴碍绻涙繝鍥х畺婵°倕鎳愬畵渚€鏌涢…鎴濇灈濠殿喖楠搁—鍐Χ韫囨挾妲ｉ梺鎼炲姀濞夋盯鎮鹃悿顖樹汗闁圭儤鍨归敍婊冣攽閻樻瑥瀚崝銈吤瑰鍫㈢暫闁诡喖鍢查埢搴ょ疀閹垮啩鐥紓鍌欒閸嬫捇鏌嶈閸撴瑩鍩為幋锔藉€烽柤纰卞墯閸曢箖姊洪崨濠冪叆婵炲鍏樺畷姘跺箳閹惧墎鐦堝┑顔斤供閸撴盯鎯堥崟顖涒拺闁告挻褰冩禍鐐烘煕閻樿櫕宕岀€规洏鍨介弻鍡楊吋閸℃瑥骞楅梻濠庡亜濞诧箑煤閺嶃劎顩烽柨鏇炲€归悡鏇㈠箹濞ｎ剙鐏╅柍缁樻礃椤ㄣ儵鎮欑€电鈷屽Δ鐘靛仦閻楁骞忛崨鏉戞嵍妞ゆ挾鍋涚粭姘舵⒑閼姐倕鏋戠紒顔肩Ф閸掓帡骞樺畷鍥ㄦ濠电姴锕ら崰姘焽閳哄懎绾ч柛顐ｇ☉婵¤姤绻涢崼婊冨祮婵﹨娅ｉ幑鍕Ω閵夛妇浜栧┑鐘愁問閸犳牠宕查弻銉﹀仼闁割煈鍋嗛悷褰掓煃瑜滈崜鐔奉嚕婵犳艾鐏抽柟棰佺閹垿姊洪崨濠傚闁告柨鐭傞垾鏍ㄥ緞鐎ｎ剛鐦堝┑鐐茬墕閻忔繈鎮橀敓鐘崇厵闁告稑锕ら埢鏇犫偓娈垮枛椤兘骞冮姀銈嗘優闁革富鍙忕槐鍙夌節绾版ɑ顫婇柛銊╂涧閳诲秹鏁愭径瀣画闂佸綊鍋婇崢鍓у姬閳ь剟姊洪棃娑㈢崪缂傚秴妫楅…鍥箛椤撶姷顔曢梺鑲┾拡閸撴瑩寮告惔銊︽嚉闁绘劗鍎ら悡鏇熺箾閹寸偟鎳勯柛搴㈩殜閺屸剝鎷呯粙鎸庢闂佸搫鏈ú婵堢不濞戞瑧绡€闁稿本鍩冮幏銈夋⒒娴ｇ瓔鍤冮柛鐘愁殜閹兘濡烽埡浣哄幋闂佺鎻梽鍕磹妞嬪簶鍋撻悷鏉款棌闁哥姵鐗曢埢鎾诲醇閺囩啿鎷虹紓浣割儓濞夋洜绮婚弻銉︾叆闁哄洦锚閻忔煡鏌ｅ☉鍗炴珝鐎规洘锕㈡俊鎼佸閿涘嫧鍋撴繝姘拺鐟滅増甯掓禍浼存煕閻樺啿鍝洪柟顖欑窔瀹曞ジ濡烽敂瑙勫缂傚倷绀侀鍡欌偓绗涘喛鑰块柣妤€鐗勬禍婊堟煏婵炲灝鍔滄い銉ｅ灲閺屻劌鈹戦崼婵呯捕濡炪倖鎸搁崥瀣嚗閸曨厸鍋撻敍鍗炲椤忓綊姊绘担钘夊惞闁哥姵鍔欓幃鐑藉Ψ瑜夐崑鎾愁潩閻撳骸鈷嬮悗瑙勬礃缁诲牊淇婇崼鏇炵倞闁冲搫鍋婄槐锛勭磽閸屾艾鈧兘姊藉澶婂瀭鐟滅増甯炲畵浣逛繆閵堝懏鍣洪柣鎾跺枑娣囧﹪濡堕崒姘闂傚倷绀佹惔婊呭緤娴犲缍栭煫鍥ㄦ礈绾惧吋淇婇婵愬殭妞ゅ孩鎹囧娲礂闂傜鍩呴梺绋垮瘨閸ㄥ爼宕洪埀顒併亜閹哄棗浜鹃梺鍝ュ枑婢瑰棗危閹版澘绠抽柡鍐ｅ亾濠殿垰銈搁弻娑㈡晜鐠囨彃绠婚梺浼欑畱閻楁捇骞冨Δ鍐╁枂闁告洦鍓涢ˇ銊︾節閻㈤潧浠滈柨鏇ㄤ簻椤曪綁顢曢敃鈧粻娑欍亜閹捐泛浠х紒鎰仱閺岋綀绠涢弴鐐版埛闂佸搫鎷嬮崑濠冧繆閸洘鏅濋柛灞剧▓閹风粯绻涙潏鍓у埌闁硅姤绮庣划鏂棵洪鍛幈闁诲函缍嗘禍鐑界叕椤掑倵鍋撶憴鍕８闁搞劋绮欓獮濠囨倷閸濆嫀銊╂煏婢跺牆鍔ゅù婊勭墵濮婄粯鎷呴崨濠呯闂佹儳绻愰柊锝呯暦閺夎鏃堝川椤旈棿绨甸梻浣告惈濞层劑宕伴幘璺哄К闁逞屽墴濮婂宕掑鍗烆杸缂備礁顑嗛崹鍧椼€侀弮鍫濈厸闁告侗鍠氶崢閬嶆⒑閸濆嫬鏆婇柛瀣尵缁辨帡顢欓懞銉ョ３閻庢鍠栭…鐑藉箖閵忋倕绀傞柣鎾冲椤撹鈹戦埥鍡椾簽濠⒀勵殘閹广垽宕掑锝嗙亖濡炪倖鎸堕崹娲煕閹达附鍊甸柛锔诲幖椤庡本绻涢崗鐓庡闁哄本鐩俊鎼佸Ψ閿曗偓娴犳潙螖閻橀潧浠滈柛鐕佸亰閸┿垺鎯旈妸銉ь啋闂佸搫顦伴崹鍫曨敇缂佹ü绻嗛柣鎰典簻閳ь剚鐗曢蹇旂節濮橆剛锛涢梺鐟板⒔缁垶鎮￠崘鈹夸簻闊洦鎸炬晶鏃堟煙閼碱剙浜鹃柟渚垮妽缁绘繈宕ㄩ鍛摋缂傚倷绶￠崰妤呮偡閵夆晛鐓濋幖娣妼缁犳氨鎲稿鍡曠箚鐟滃繒妲愰幘璇茬＜婵﹩鍏橀崑鎾诲传閵壯呯厠閻庤娲栧ú锕傤敃閼恒儲鍙忔慨妤€妫楁晶浼存煛閸☆參妾紒缁樼☉椤斿繘顢欓懡銈囨晨闂?
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

	// Apply OpenAI fast policy (闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锕傛煟閵忕媭鐓兼慨濠勭帛缁楃喖鍩€椤掆偓椤洩顦归挊婵囥亜閹板墎鐣遍柣銈囧亾缁绘盯骞嬮悙璺侯棟濡炪們鍎插畝鎼佸蓟閿濆鍋勭紒瀣儥濡酣姊虹悰鈥充壕婵炲濮撮鍡涙偂閻旈晲绻嗘い鏍ㄧ箖椤忕娀鏌＄€ｎ亜鏆欐い顓℃硶閹叉挳宕熼幆鎵冲亾閸ф鐓?Claude BetaPolicy 闂?fast-mode 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾剧懓顪冪€ｎ亝鎹ｉ柣顓炴閵嗘帒顫濋敐鍛婵°倗濮烽崑鐐烘偋閻樻眹鈧線寮村杈┬㈤梻浣规偠閸庢椽宕滈敃鍌氭瀬闁告劦鍠楅悡銉╂煛閸ヮ煈娈斿ù婊堢畺濮婂搫效閸パ€鍋撳Δ鍛；闁规崘鍩栧畷鍙夌箾閹存瑥鐏╃紒鐙呯秮閺岋綁骞囬妸锔界彆闂佹寧绋戦悧鍡涘煘閹达附鍊烽柛娆忣槴閺嬫瑦绻涚€涙鐭嬬紒璇茬墕椤曪綁骞撻幒鍡樻杸闁诲函缍嗛崑鍡涘储闁秵鐓熼煫鍥ㄦ礀娴犙囨煕鐎ｎ偅宕岄柟顖氬暣楠炲鎮欓鍐泿?闂?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤濠€閬嶅焵椤掑倹鍤€閻庢凹鍙冨畷宕囧鐎ｃ劋姹楅梺鍦劋閸ㄥ綊宕愰悙宸富闁靛牆楠告晶濠氭煕閹炬潙鍝洪柟顖氬椤㈡稑顫濇潏銊︻啎闂備胶绮濠氬煕閸儱鏋佺€广儱妫涚粻楣冩煙鐎甸晲绱抽柤娴嬫櫇绾鹃箖鏌￠崶銉ョ仾闁抽攱甯掗湁闁挎繂鐗婇鐘绘煏閸℃韬柡灞剧洴楠炴鈧潧鎽滈悾铏圭磽娴ｈ娈橀柛鐘叉唉閻忓啴姊洪崨濠佺繁闁告ü绮欓幃?body 闂?service_tier 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻濠氭煕閳规儳浜炬俊鐐€栫敮濠囨嚄閸洖鐓濋柟鍓х帛閻撴盯鏌涘☉鍗炴灓缂佺姵锕㈤弻娑㈠箳閹惧磭鐟ㄩ梺瀹狀嚙闁帮綁鐛Ο铏规殾闁搞儴娉涢弫钘夆攽閻樻鏆滅紒杈ㄦ礋瀹曟垵鈽夐姀鈥冲壄闂佺粯鍨煎Λ鍕婵犳碍鐓欓柟瑙勫姦閸ゆ瑧绱掗埀顒勫礃閳瑰じ绨婚梺鍝勫暙閸婂摜鏁崼鏇熺厾闁哄娉曟禒銏ゆ煃鐟欏嫬鐏撮柟顔界懇瀵爼骞嬮悩杈╃婵犵绱曢崑娑㈡偤閵娾晛绠栭柛灞惧嚬閸ゆ洟鏌＄仦璇插姎闁绘挻鐩弻娑樷槈閸楃偞鐏堥梺閫炲苯澧伴柡浣割煼瀵鈽夊鍛澑闂佺懓鐏濋崯顖滅懅婵犵數鍋涢悺銊у垝閹惧墎涓嶉柡宓本缍庡┑鐐叉▕娴滄粌顔忓┑鍡忔斀闁绘ɑ褰冮弳娆愩亜閿旇娅婃慨濠冩そ瀹曘劍绻濋崘銊╃€洪梻浣哄帶缂嶅﹦绮婚弽顓炴槬?priority" 闂?fast闂?flex"闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊洪悙钘夊姎缁剧虎鍘奸敃銏ゅ箻椤旂晫鍘遍梺鍝勫暊閸嬫捇鏌涢妸銉т虎闁伙絽鍢茬叅妞ゅ繐瀚崝锕€顪冮妶鍡楃瑐闂傚嫬绉电粋宥咁煥閸喓鍘甸梺缁樺灦閿氶柣蹇嬪劦閺屽秷顧侀柛鎾寸箓閻ｇ兘宕归銈囧骄闂佸搫娲ㄩ崰鎾绘⒔閸曨厾纾奸悗锝庡幗绾泛螖閻樺弶鍠樻慨濠冩そ瀹曟粓骞撻幒宥囧嚬缂傚倷绀侀鍡涘垂閼哥數鈹嶅┑鐘插瀹曞鏌曟繛褍鎷戠槐鏌ユ⒒娴ｈ櫣甯涢柨鏇樺灩椤洩顦崇紒鍌涘浮閺佸倹鎱ㄩ幇顏嗙泿闂備線娼ч…顓犲緤娴犲绀夐柨鏇炲€归悡娆愩亜閺冨倹娅曢柟鍐叉川閳ь剚顔栭崰鏍偉婵傚摜宓侀悗锝庝簴閺€浠嬫煙闁箑骞樼紒澶愮畺濮婄粯鎷呯粵瀣婵°倗濮甸幃鍌炲箖濡警娼╂い鎺戭槺閸為潧鈹戦埥鍡楃仩闁汇劎鍏樺畷鎴﹀箻閹颁焦鍍甸梺缁樻尭妤犳悂鍩涚€ｎ喗鈷戦悹鍥ｂ偓铏亶闂佺瀛╂繛濠囧春閻愬搫绠ｉ柣姗嗗亜娴滈箖鏌ㄥ┑鍡欏嚬缂併劋绮欓弻娑㈠籍閹惧墎鏆犲銈庝簻閸熷瓨淇婇崼鏇炲耿婵°倕鍟伴幊鍡涙⒑鐠囨彃顒㈤柛鎴濈秺瀹曟粌鈽夊顒€鐏婃繝鐢靛У绾板秹宕愮紒妯圭箚妞ゆ牗绻傛禍褰掓煟閿曗偓閻楀﹦鎹㈠┑瀣仺闂傚牊鍒€閵忋倖鐓曞┑鐘插鐢盯鏌￠崨鐗堢【闁宠棄顦垫慨鈧柨娑樺楠?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闂囧鏌ㄥ┑鍡╂Ч濞存嚎鍊濋弻銈夊级閹稿骸浠村┑顔硷攻濡炰粙銆佸Δ鍛劦妞ゆ帒鍊婚惌鎾绘煟閵忋埄鐒剧痪鎯ь煼閺屻劑鎮㈤崫鍕戯綁鏌ｉ幘瀵告噰闁哄瞼鍠栭、娑橆煥閸愮偓姣夋俊鐐€戦崕閬嵥囨导娣偓鍐Ψ閳哄倸鈧攱銇勮箛鎾愁伀濞寸厧鐗撳娲传閸曨喖顏紓浣割槺閹虫捇鎮鹃悜绛嬫晝闁挎洍鍋撶紒鈧€ｎ偁浜滈柟閭﹀枛閺嬪骸霉濠婂啫鈷旈柟鍙夋倐閹囧醇濠靛牏鎳嗙紓?filter闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ偒鍎ラ柛搴＄箲閵囧嫰骞嬪┑鎰枅閻庢鍠涢褔鍩ユ径鎰潊闁绘﹢娼ф慨鍫曟⒒娴ｅ憡鍟為柛鏃撻檮缁傚秹寮介‖顒婄秮瀹曞ジ濡烽敂瑙勫闂備胶顢婇崑鎰板磻濞戞瑤绻嗗┑鍌氭啞閻撴洖鈹戦悩鎻掍簽闁绘挸鍚嬮〃銉╂倷瀹割喗鈻堝Δ鐘靛仦閻楁骞忛崨瀛樺仭濡鑳剁紞宥夋⒒閸屾瑧顦﹀鐟帮躬瀹曟垿宕ㄩ娑樺簥闂佸憡娲﹂崹閬嶅磻閿熺姵鐓涢悘鐐额嚙閸旀粓鏌嶉柨瀣伌闁哄瞼鍠栭獮鍡涘级閸熷啯鎹囬弻鈩冩媴娴犲鎽垫繛锝呮搐閿曨亪銆侀弴銏狀潊闁绘娅曢鎺楁煟鎼淬埄鍟忛柛鐘崇墵閳ワ箑鐣￠柇锕€娈ㄩ梺鍦檸閸犳牠锝為崨瀛樼厽婵☆垱顑欓崵娆撴煛娴ｇ瓔娼愮紒缁樼箞閹粙妫冨☉鎺撶€版繝鐢靛仒閸栫娀宕楅悩铏仢濠碘€崇埣瀹曘劑顢旈崨顖楀亾濞差亝鍋℃繝濠傚閻撱儲銇勯銏㈢閻撱倖銇勮箛鎾愁仾鐎规挸绉撮—鍐Χ閸℃ê鏆楅梺鍝ュУ閸旀瑩骞冮悽鍓叉晜闁割偆鍟块幏娲⒑閸涘﹦缂氶柛搴ら哺閻楀酣姊绘担鍛婂暈婵﹤缍婇妴鍐╃節閸モ晛绁﹂梺纭呮彧缁犳垿锝為崨瀛樼厽婵妫楁禍婵囥亜椤愩垻效婵﹥妞藉Λ鍐ㄢ槈濮橀硸鍞哄┑鐘垫暩閸嬫劙宕戦幘缁樷拺闁告縿鍎辨禒婊堟煟閺嶎偄甯舵い顐㈢箰鐓ゆい蹇撴媼濡啫鈹戦悙鏉戠仸婵ǜ鍔戦幃鐐偅閸愨斁鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜妞ゆ棁鍋愰悞鎼佹煕閳瑰灝鐏叉鐐搭焽閹风娀鎳犻澶婃倯濠碉紕鍋戦崐鏍蓟閵婏附娅犲ù鐘差儐閺咁剚绻濋棃娑欏窛缂佺娀绠栧鍫曞醇濠靛棌鎸冮柣搴濈祷閸嬫劗妲愰幒妤婃晩闁伙絽鏈崳浼存倵鐟欏嫭澶勯柛瀣躬楠炴牞銇愰幒鎴狀槯婵犮垼娉涢敃锝嗙珶閺囥垺鈷戠紒瀣硶缁犵増銇勯敂璇茬仭缂佸倸绉甸妶锝夊礃閳圭偓瀚奸柣鐔哥矌婢ф鏁Δ鍛櫖闁绘柨鍚嬮悡鐘绘倵閸︻厼校缁绢參绠栭弻?block闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ偒鍎ラ柛搴＄箲閵囧嫰骞嬪┑鍥ф畻闂佺硶鏅濋崑銈夌嵁鐎ｎ喗鏅滈柦妯侯槷濮规姊绘担鍛婅础缂侇噮鍨抽弫顕€鍩￠崒婊庢祫闂備緡鍓欑粔鐢稿煕閹寸姷纾奸悗锝庡亽閸庛儵鏌涙惔銏犲闁哄瞼鍠栧畷銊︾節閸愩劉鍋撻幇鐗堢叆婵犻潧鐗嗘禒婊堟煃鐟欏嫬鐏╅柍褜鍓ㄧ徊浠嬫倶濮樿泛鐤鹃柕蹇ョ磿缁犻箖鏌熼崜褜妫庡瑙勆戠换娑氭嫚瑜忛悾鐢告煛娴ｈ宕屽┑锛勬焿椤﹀鐥崣銉х煓闁哄本绋撴禒锕傚礈瑜庨崳顓犵磽娴ｉ鍔嶆繝銏★耿閳ユ棃宕橀鑲╅獓闂佺懓顕崑鐐哄窗閹扮増鈷戦弶鐐村鐠愪即鏌涢敐蹇曠М闁绘侗鍣ｉ獮鎺楀箠閾忣偅顥堥柛鈹惧亾濡炪倖甯掔€氼參宕戦崒娑欏弿婵＄偠顕ф禍楣冩⒑閸濆嫮鐏遍柛鐘虫崌閸┾偓妞ゆ帒鍊归弳鈺傘亜椤撶偟澧涚紒鍌涘笩椤﹀绱掓潏銊ユ诞闁糕晪绠撻崺鈧い鎺戝绾惧鏌熼崜褏甯涢柍閿嬪灩閻ヮ亪顢橀悙鏉戞婵炲瓨绮岀紞濠囧箖鐠鸿　妲堥柡宓啫袝闁诲氦顫夊ú妯煎垝瀹€鍕厴闁瑰濮崑鎾绘晲閸涱垯绮甸梺鍝勬噳閺呮粎鎹㈠┑瀣潊闁挎繂妫涢妴鎰渻閵堝棗鐏ユ繛灞傚妼椤曪綀顦归柛鈹惧亾濡炪倖宸婚崑鎾绘煃鐟欏嫬鐏撮柟顔界懇閹崇娀顢楅埀顒勩€傚ú顏呪拺閻犲洩灏欑粻鑼磼鐠囪尙澧曟い鏇秮楠炴牗鎷呴懖婢喚鐔嗛悹杞拌閸庢垿鏌涘Ο鍏兼毈婵﹥妞介獮鍡氼槼缂佸倸顑夐弻娑氣偓锝庝簼閸ｅ綊鏌ｉ敐鍛Щ闁伙絾绻堥崺鈧い鎺戝閻掑灚銇勯幒鎴濇灓婵炲吋鍔栫换娑㈠矗婢舵稖鈧法鈧娲栫紞濠傜暦濠婂嫭濯撮柣鐔告緲缁ㄣ儲绻濋悽闈涒枅婵炰匠鍥舵晞闁告侗鍨抽惌鍡涙倵閿濆骸浜栧ù婊勭矒閺岀喖鎮滃Ο铏瑰姼濠碘剝褰冨﹢杈╂閹炬剚鍚嬮柛鈾€鏅滈悾椋庣磽娴ｈ櫣甯涚紒璇插€块崺銏ゅ箻鐠囨彃鐎銈嗘⒒閺咁偅绂嶉懜鍏哥箚闁绘劦浜滈埀顑惧€濆畷銏＄附閸涘﹤浜遍梺绯曞墲缁嬫垿鎮″┑瀣厸闁稿本姘ㄥ銊︿繆閹绘帞澧涚紒缁樼箖缁绘繈宕掑☉妯活仱婵＄偑鍊栧鐟拔涢崘顭戞綎濠电姵鑹剧壕鍏兼叏濮楀棗鍘撮柛瀣崌閹粓鎳為妷銉︾暠闂備礁澹婇崑鍛哄澶婄劦妞ゆ帒锕﹂悾鐢碘偓瑙勬礀閵堝憡淇婇悜钘壩ㄩ柕澹啰绉繝纰夌磿閸嬫垿宕愰弽顐ｆ殰闁圭儤鏌￠崑鎾愁潩閻撳骸顫紓渚囧枛閻楁挸鐣烽悢纰辨晣闁绘棃鏀遍悗楣冩⒒娴ｈ櫣甯涙い銊ユ嚇閹囧幢濞嗘垹鐣堕梺鍦劋椤ㄥ棝鎮?gpt-5.5 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚敐澶婄闁挎繂鎲涢幘缁樼厱濠电姴鍊归崑銉╂煛鐏炶濮傜€殿噮鍓熷畷褰掝敊鐟欏嫬鐦遍梻鍌欑劍濡炲潡宕㈡禒瀣仭闁冲搫鎳庨拑鐔兼煟閺傝法娈遍柡瀣叄濮婁粙宕堕澶嬫櫔閻熸粌绉堕崣鍛渻閵堝棙灏柛銊嚙閻ｅ灚绗熼埀顒勬偂椤愶箑鐐婇柕濠忓閿涙稑鈹戦悙鑼劸濞存粠浜濠氬Χ婢跺﹦鐣抽梺鍦劋閸ㄥ灚鎱ㄩ弴鐑嗘富闁靛牆妫欓悡銉︺亜閿曞倹娑фい顐㈢箲缁绘繂顫濋鍌︾床婵犳鍠楅敃鈺呭礂濮椻偓瀹曟垿骞樼紒妯衡偓濠氭煠閹帒鍔氶柍褜鍓欏锟犲蓟閵娾晛鍗抽柣鎰ゴ閸嬫捇宕妷褌绗夐梺鍝勭▉閸樹粙鎮￠弴鐔翠簻闁规澘澧庣粙鑽ょ磼閳ь剟鍩€椤掑倻纾藉ù锝嗗絻娴滅偓绻濋姀锝嗙【濠㈣泛娲畷鎴﹀箻鐠囪尙顦ф繝銏ｆ硾閿曪絾绔熼弴銏♀拻濞达絽鎲￠崯鐐寸箾閸欏澧甸柟顔矫～婵堟崉閾忚鐓ｉ梻浣虹帛閸旀宕曢妶澶婄厱闁硅揪闄勯悡鏇熺箾閹寸儑鍏柛鏃傚枛閺屽秹鎸婃径妯烩枅闂佸搫鐭夌徊楣冪嵁婵犲洦鐓曢柡鍐ｅ亾闁搞劎鏁婚幃楣冩倻閽樺宓嗛梺闈涚箳婵兘顢欓幒鏃傜＝闁稿本鐟ч崝宥嗐亜椤撶偞鍠樼€规洏鍨介弻鍡楊吋閸″繑瀚奸梻浣告惈閸婂爼宕曢幓鎺嗘瀺闁告稑鐡ㄩ悡鏇㈡煟閺冨洦纭剧€规挸妫濋弻娑㈠煛娓氬﹨鍚悗娈垮枟閹歌櫕鎱ㄩ埀顒勬煃閵夛附鐏遍柛?	// fast 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鎾煙椤斿吋鍋ユい銏″哺閸┾偓妞ゆ帒瀚拑鐔哥箾閹存瑥鐏╃紒鐘崇⊕閵囧嫰骞樼捄鐩掔偤鏌涚€ｎ剙浠辨慨濠冩そ瀹曨偊宕熼棃娑樺闂備胶绮〃鍡涖€冮崼銉ョ闁靛繒濮Σ鍫ユ煏韫囨洖啸妞ゆ挻妞藉铏圭磼濡櫣浠村┑鐐茬湴閸婃繂鐣烽敐澶婂嵆闁靛繆妾ч幏娲⒑閸忚偐銈撮柡鍛洴閵嗗倹绺介崨濠傗偓鐢电磼濡や胶鈽夋繛灞傚€栭、濠囨⒒娴ｅ憡鍟炴繛璇х畵瀹曟澘顫濋懜闈涗痪闂侀€炲苯澧存慨濠冩そ瀹曠兘顢橀悙鏉款棜闂備礁鎲￠幐鑽ゅ枈瀹ュ洠鍋撴担鍐ㄤ汗濞寸媴濡囬幉鎾晲閸℃ɑ婢戦梻鍌欒兌缁垶宕濆Δ鍛？闁靛牆顦伴弲顒勬煟閺傚灝鎮戦柣鎾存礃娣囧﹪濡堕崟顓фМ闂佸磭绮ú婊堝焵椤掍緡鍟忛柛鐘崇墵閳ワ箑鐣￠柇锕€娈ㄩ梺鍦檸閸犳牠锝為崨瀛樼厽婵☆垱顑欓崵娆撴煛娴ｇ瓔娼愮紒缁樼箞閹粙妫冨☉鎺撶€版繝鐢靛仒閸栫娀宕楅悩铏仢濠德ゅ煐缁旂喖鏁冮埀顒€煤椤撱垹鏄ラ柨鐔哄Т瀹告繃銇勯弮鍥舵綈閻庢艾銈稿缁樻媴閸涘﹤鏆堢紓浣筋嚙閸婂鍩€椤掍礁鍤柛娆忓暙閻ｇ兘宕樼憗浣规そ椤㈡棃宕ㄩ姘濠碉紕鍋戦崐鏍ь潖婵犳艾鍌ㄧ憸鏃堝春閳ь剚銇勯幒鍡椾壕闂佽绻戠换鍫ャ€佸鑸垫櫜闁糕剝鐟ч惁鍫濃攽椤旀枻渚涢柛鎾寸洴瀹曟繈骞橀瑙ｆ嫼闂佸憡绋戦敃銈嗘叏閳ь剟姊洪崫鍕櫤闁烩晩鍨堕妴渚€骞樼拠鑼啋缂傚倷鐒﹁彜闁归攱妞藉娲川婵犲嫧妲堥梺鎸庢穿婵″洨妲愰幘鎰佹僵闁煎摜鏁搁崢閬嶆煙閸忚偐鏆橀柛鏂款儔楠炲銈ｉ崘鈺冨帾闂佺硶鍓濋〃鍛村焵椤掆偓閻忔繈鎮鹃柨瀣檮缂佸鐏濆畵鍡涙⒑缂佹ɑ鐓ョ€殿喖澧庨埀?	//
	// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃缂侇噮鍨抽幑銏犫槈閵忕姷顓洪梺鍝勫暊閸嬫捇鏌涢妶鍛ч柡灞剧洴婵＄兘顢欓悡搴樻嫽闂備浇妗ㄧ粈浣该洪銏犺摕闁哄浄绱曢悿鈧梺鍝勬川閸婎偊濡烽敂杞扮盎闂佹寧妫侀褍鈻嶅澶嬬厵妞ゆ梻鐡斿▓婊堟煟濞戝崬娅嶇€规洖缍婇、娆撴偂鎼搭喗缍撴繝纰夌磿閸嬫垿宕愯閳ь剟娼ч惌鍌氱暦閻熸壆鏆﹂柛銉戝啰浜伴梻浣稿閸嬩線宕曢柆宥嗙厑闁搞儯鍔庣弧鈧梺闈涢獜缁辨洜绮婚幘鍓佺＝鐎广儱鎷戦煬顒侇殽閻愭彃鏆ｉ柛鈺佸瀹曟﹢鍩℃担绋课ら梻鍌欑劍鐎笛呮崲閸屾娑樷枎閹惧磭鐛ラ梺鍝勭▉閸樹粙鍩?	//   1. 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊閻樺樊妫岄梺杞扮閿曨亪寮婚垾鎰佸悑閹肩补鈧磭顔愮紓鍌欑劍閸旀牠銆冮崱妯尖攳濠电姴娲ゅ洿闂佸憡渚楅崢钘夆枔閸撲胶纾藉ù锝囨嚀閺佸墽绱掗悩铏磳鐎殿喛顕ч埥澶婎煥鎼粹懣顏呬繆閻愵亜鈧垿宕濆畝鍕疇闁归偊鍘鹃悵鍫曟煛閸ゅ爼顣﹀Ч妤呮⒑閸︻厼鍔嬮柟鎼佺畺瀹曘垽骞栨担鍏夋嫼闂佺鍋愰崑娑㈠礉閻㈢數纾奸柡灞诲劤閻ｈ櫣鈧娲熷褔顢橀崗鐓庣窞閻庯綆鍓欓獮鍫ユ⒒娓氣偓濞佳囁囨禒瀣獥闁哄秲鍔嶅▍鐘炽亜閺囨浜鹃梺鍝勭焿缂嶄線鐛Ο灏栧亾闂堟稒鍟為柛鎺撶洴濮婃椽宕崟顒佹嫳闂佺儵鏅╅崹鍫曟偘椤曗偓瀹曞爼顢楅埀顒傜矆閸岀偞鐓曟繝闈涘閸旀粓鏌￠崨顓滃仮婵﹦绮幏鍛存偡闁箑娈濈紓鍌欑椤戝棛鏁悙鍝勭闁圭儤顨呯粻濠氭煙妫颁胶鍔嶇紓宥呴叄濮婃椽骞嗚缁犵増绻濋埀顒佹綇閳轰緡妫滈梺绋跨箻濡法鎹㈤崱妯镐簻闁哄秲鍔庣粻鎾绘煕濮橆剛绉洪柡灞剧☉閳诲氦绠涢弮鎴烇紒缂傚倷鑳剁划顖滄崲閸儱鏄ユ繛鎴欏灩閻掑灚銇勯幒鎴濐仼缂佲偓閸愨斂浜滈柡鍐ㄥ€哥敮鍫曟煕濞嗗骏鏀诲ǎ鍥э躬閹瑩顢旈崟銊ヤ壕闁硅揪绠戦崹鍌炴煕濠靛棗鐝嬫繛鎴欏灩缁€鍐煏婵炑冨椤旀洟姊绘担鑺ョ《闁哥姵鎸婚幈銊╂偨缁嬭法锛欓梺鍓插亝濞叉﹢宕戦敐澶嬬厱闁靛鍠曠花濠氭煟閵娿劍顏犵紒杈ㄥ笚瀵板嫮鈧綆鍋佹禒銏ゆ⒑閸濆嫬顦柛鎾寸箞楠炲繘宕ㄧ€涙ê浠梺鍝勵槹椤戞瑥顭囧鑸电厽閹兼番鍊ゅ鎰箾閸欏澧靛┑鈥崇埣楠炴牗鎷呴崫銉ф毇闂備線娼х换鍫ュ磹閺囩姷涓嶉柟鐑橆殕閻撴瑩鏌涘┑鍡楊仾妞ゃ儲绮岄湁婵犲﹤瀚崝銈夋煃鐟欏嫬鐏撮柟顔规櫊瀹曪絾寰勬繝搴⑿熼梻?upstreamModel闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ偒鍎ラ柛搴＄箲閵囧嫰骞嬪┑鎰枅閻庢鍠涢褔鍩ユ径鎰潊闁绘﹢娼ф慨鍫曟⒒娴ｅ憡鍟為柛鏃撻檮缁傚秹寮介‖顒婄秮瀹曞ジ濮€閵忣澁绱冲┑鐐舵彧缁叉崘銇愰崘鈺冾洸濡わ絽鍟埛鎴︽煕濞戞﹫鏀诲璺哄閺屾盯濡搁埡鈧幉楣冩煙椤曞棛绡€濠碉紕鍏橀弫鍌炲礈瑜滈崯搴ㄦ⒒娴ｇ儤鍤€妞ゆ洦鍘界€电厧鈹戠€ｅ灚鏅滈梺鍓插亽閸嬪懘寮插鍐炬富闁靛牆妫楁慨鍌炴煕婵犲喚娈旈摶鐐淬亜閺嶎偄浠﹂柣鎾寸懄閵囧嫰寮借椤ユ瑧绱掗埀顒佹媴閸︻収娴勫銈嗘磵閸嬫挻鎱ㄦ繝鍛仩缂佽鲸甯掕灒缂備焦锚閸擃噣姊绘担渚劸妞ゆ垵鎳橀弫鍐Χ婢跺浠奸梺缁樺灱濡嫰鏌嬮崶顒佺厸闁搞儮鏅涢埀顒婄秮閸╋繝宕ㄩ鎯у汲闂備礁鎼崯顐﹀磹閸涘﹦顩查柣鎰靛厵娴滄粓鏌熼崜褎鍤€闁告艾婀辩槐?GetMappedModel +
	//      normalizeOpenAIModelForUpstream + Codex OAuth normalize闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊洪悙钘夊姎缁剧虎鍘奸敃銏ゅ箻椤旂晫鍘遍梺鍝勫暊閸嬫捇鏌涢妸銉т虎闁伙絽鍢茬叅妞ゅ繐瀚崝锕€顪冮妶鍡楃瑐闂傚嫬绉电粋宥咁煥閸喓鍘甸梺缁樺灦閿氶柣蹇嬪劦閺屽秷顧侀柛鎾寸箓閻ｇ兘宕归銈囧骄闂佸搫娲ㄩ崰鎾绘⒔閸曨厾纾奸悗锝庡幗绾泛螖閻樺弶鍠樻慨濠冩そ瀹曟粓骞撻幒宥囧嚬缂傚倷绀侀鍡涘垂閼哥數鈹嶅┑鐘插瀹曞鏌曟繛褍鎷戠槐鏌ユ⒒娴ｈ櫣甯涢柨姘辩棯缂併垹骞楅柡鍛板煐濞煎繘鍩￠崘顏庣床?	//      chat-completions / messages 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆浂鐒炬い銉ョ箻閺屾稓鈧絺鏅濈粣鏃傗偓瑙勬礃濞叉ê顭囪箛娑樼厸濞撴艾娲﹂ˉ锟犳⒒娴ｄ警娼掗柛鏇炵仛閻ｅ墎绱撴担鍝勭彙闁搞儯鍔庨崢鍗烆渻閵堝棗濮﹂柛瀣缁傚秴顭ㄩ崗鐘垫嚀椤劑宕熼鐐╁悅闂傚倸娲らˇ鎵崲濠靛洨绡€闁稿本鍐荤槐鐐测攽閻愭彃鎮戦柛鏃€顨堝Σ鎰板箳閺冨倻锛滃┑鈽嗗灠閸氬宕氬☉妯滄棃鎮╅棃娑楃捕濡炪倖鍨甸ˇ鐢哥嵁閸愩劉鏋庨柟鎯у暱閻庮厼顪冮妶鍡楀闁搞劍妞介幃鐢割敋閳ь剙顫忔繝姘＜婵炲棙鍩堝Σ顔剧磼閸撗嗘闁告瑥鍟撮悰顔界節閸パ咁啋闂佸憡鎸烽懗鍫曟偂鐎ｎ喗鈷戞慨鐟版搐閻忊晠鏌ｈ箛鏃囧闁伙絿鍏樺畷锝嗗緞瀹€鈧惁鍫ユ⒒閸屾氨澧涘〒姘殜瀹曟洟骞囬婊€绨婚梺闈涱槶閸庤櫕鏅跺☉姘辩＜缂備焦顭囧ú瀛橆殽閻愬樊鍎旈柟顔界懇瀹曞綊顢曢姀锛勫将闂傚倸鍊峰ù鍥敋閺嶎厼绐楁俊銈呮噺閸嬶繝鏌嶆潪鎵窗婵炲吋鐗犻弻锝夊箣閻忔椿浜、姘綇閵娧咁啎闂佺硶鍓濋〃鍫㈢不閾忣偂绻嗛悹鍥囧懐袦濠殿喖锕ュ钘夌暦椤愶箑唯闁靛鍔х紞渚€寮婚敐鍛傛棃宕橀妸銏＄€伴柣搴ゎ潐濞叉ê顪冮懞銉ょ箚闁绘垼濮ら弲婵嬫煃瑜滈崜娑樷枎閵忋倖鍊烽柣鎴烆焽閸樼敻姊绘笟鍥у伎缂佺姵鍨堕弲鍫曟偋閸稐绨婚梺鍝勫暊閸嬫捇鎮介娑樻诞闁炽儲妫冨畷姗€顢欓崲澹洦鐓曢柍鈺佸暟閹冲啯銇勯搴℃处閳锋垶绻涢懠棰濆殭鐎殿噣绠栭弻娑㈡偐閹颁焦鐣肩紓渚囧枓閺呯娀骞冮姀銈嗗亗閹肩补鈧磭銈梻鍌欑窔濞佳呮崲閸℃鐎剁憸鏃堢嵁韫囨洜纾兼俊顖濆亹椤旀洟鏌℃径濠勫濠⒀呮櫕缁棃顢楁担椋庣畾濡炪倖鍔戦崹褰掝敂椤忓棛纾奸柛灞炬皑鍟搁梺闈涙处閸旀瑩鐛幒妤€绠婚悹鍝勬惈缁犱即姊绘担绛嬪殭缂佺粯甯″畷鎴︽偄閸忕厧浠遍梺鍝勫€介鎶芥偄閻撳海鍔﹀銈嗗坊閸嬫捇鎽堕弽顓熺厽婵せ鍋撴繛浣冲嫮顩烽柨鏇炲€归悡娆撴煟閻斿搫顣奸柛鐘筹耿閺屾洟宕遍弴鐘电崲閻庢鍣崳锝呯暦閻撳簶鏀介柛鈩冨焹閸嬫捇宕归瑙勬杸濡炪倖姊婚悺鏂库枔閺冣偓娣囧﹪顢曢敐鍥ㄥ櫘缂備浇浜崑娑滅亙婵炶揪缍€濞咃綁鎯侀崼銉︹拺婵懓娲ら悘鍙夌箾娴ｅ啿娲ら悡婵嬫煕椤愩倕鏋旂紒鐘冲劤闇夐柨婵嗘噹閺嗚鲸绻涢弶鎴濃偓鍨嚕椤愶箑绀嬫い鏍ㄧ〒閸橀亶姊洪弬銉︽珔闁哥喍鍗抽崺濠囧即閵忥紕鍘辨繝鐢靛Т閸婄粯绂掗姀銈嗙厓閻熸瑥瀚悘瀵糕偓瑙勬礃鐢帡锝炲┑瀣垫晣闁绘劕鐡ㄩ悾濂告⒒閸屾瑨鍏岀紒顕呭灦楠炴劗鎷犵憗浣告惈椤粓鍩€椤掍椒绻嗛柣銏㈩焾缁€瀣亜閺嶇數绋婚柡鍜冪秮濮婅櫣绱掑Ο娲绘濡炪們鍎查幐鑽ょ矉閹烘绠绘い鏃囆掗幏娲⒑閸涘﹦绠撻悗姘卞厴瀹曟洟骞囬悧鍫㈠幗闂佸啿鎼敃銈夋倶閿旂瓔娈介柣鎰絻閺嗙偟绱掗崒娑樼闁逞屽墾缁蹭粙鎮樺璺虹柧妞ゆ巻鍋撻柍瑙勫灴椤㈡瑩寮妶鍕繑闂備礁鎲￠幐濠氭嚌閹规劦鏆伴梻浣筋潐婢瑰棙鏅跺Δ鍛厱闁硅揪闄勯崐鐢告煕閿旇骞栭弽鈥愁渻閵堝啫鍔氭い锔垮嵆婵＄敻宕熼娑欐珕闂佺粯鍔﹂崜姘掗崼銉︹拺缂備焦眉缁堕亶鏌涢悩宕囧⒊闁诲繑甯″娲箚瑜忕粻鐑樸亜椤愩埄妲搁柍缁樻尰閵堬綁宕橀埞鐐闂備胶顢婇崑鎰板磻濞戙垹绀夋慨姗嗗幒缁诲棙銇勯幇鈺佺伈闁告瑥瀚埀顒冾潐濞叉繈锝炴径宀€鐭夐柟鐑樻煛閸嬫捇鏁愭惔婵囨崳闂佺濮ゅú鐔奉潖閾忓湱纾兼俊顖滃劦閹峰姊洪崘鎻掓Щ闁绘鎸搁锝夘敃閿濆洨鐦堥梺鎼炲劘閸斿酣宕㈡ィ鍐┾拺閻犲洠鈧啿绠洪柣銏╁灡鐢€崇暦濠靛棭娼╅弶鍫氭櫆閿涘繘姊虹紒妯哄Е闁告挻鐟╅崺銏ゅ即閵忥紕鍘靛銈嗘⒒閻℃柨鈻撳鈧弻鐔煎矗婢跺鍞夐悗瑙勬礈閸犳牠銆佸Δ鍛＜闁靛牆鏌婇悙娴嬫斀闁挎稑瀚禍濂告煕婵犲啫鐏寸€规洘绻傞悾婵嬪礋椤掍焦鐝栭梻渚€娼ч¨鈧┑鈥虫喘閸╃偛顓奸崨顏呮杸闂佺粯锚瀵埖寰勯崟顖涚厱濠电姴鍋嗛悡濂告煛瀹€瀣暠閾伙綁鏌ｅ顒夊殶缂佲偓閸℃稒鈷戦梻鍫熷崟閸儱鐤炬繝濠傜墛閸嬪倿鐓崶銊︽儎闁绘柨妫濋幃褰掑炊閿濆倸浜剧€规洖娲ｉ崫妤呮⒒娴ｇ儤鍤€闁哥喎娼¤棟闂傚牊渚楅崵鏇㈡煙闂傚顦﹂柛鎰ㄥ亾婵＄偑鍊栭幐楣冨疮閸ф绠┑鐘崇閳锋垹绱掔€ｎ偄顕滄繝鈧导瀛樼厽闁绘梹绻傚ú銈囩矆婢跺备鍋撻獮鍨姎妞わ缚鍗抽幃鈥斥枎閹扳晙绨婚梺鍝勫暙濞层倖绂嶈ぐ鎺撶厱闁冲搫顑囬幃濂告煃瑜滈崜婵嬶綖婢跺⊕鍝勎熸總鑺ユそ閹垽鎮℃惔锝庡晣闁荤喐绮庢晶妤冩暜閳哄懎鏋侀柛鎰靛枟閻撱儲绻濋棃娑欘棡闁革絿鎳撻湁闁绘挸閰ｉ崣鍕煛瀹€瀣ɑ闁诡垱妫冩俊鑸垫償閵忋垻啸濠电姷鏁搁崑娑橆嚕閸洘鍋嬮柡鍥╁Т瀵煡姊绘担鐟邦嚋婵☆偄鍟村顐︻敋閳ь剙鐣烽幋锕€绠婚柟棰佺劍閸嶉潧顪冮妶鍡楃瑨妞わ箓浜跺畷銉р偓锝庡枟閳锋垿鏌涘┑鍡楊伀闁绘帟娉曠槐鎾愁吋閸曨収妲銈嗘穿缂嶄線銆佸璺虹劦妞ゆ巻鍋撻柣锝囧厴楠炲棝鏌囬敂鍙劍绻濈喊妯峰亾閸愯尙楠囬梺鍛婃⒐閻熴儵锝炶箛娑欐優闁革富鍘鹃敍婊冣攽閳藉棗鐏ｉ柛搴ｆ暬楠炲繑绻濆顓涙嫼闂佸憡绋戦…顒勬倿婵傚憡鐓曢柕澶涚到婵″ジ鏌嶈閸撴瑦鏅舵惔銊ョ劦妞ゆ帒鍠氬鎰箾閸欏澧悡銈夋煟濡鍤欑痪鎯ь煼閺岀喖骞戦幇顒傚帿闂佸摜濮村Λ婵嬪蓟濞戙垹鍗抽柕濞垮劚椤偄顪冮妶蹇氱闁稿酣浜堕崺鐐哄箣閿旇棄鈧兘鎮规ウ鎸庮仩婵絻鍨虹换婵嬫偨闂堟稐姹楅梺绋款儐閹瑰洭骞?	//      缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鏁愭径濠勵吅闂佹寧绻傞幉娑㈠箻缂佹鍘辨繝鐢靛Т閸婂綊宕戦妷鈺傜厸閻忕偠顕ф慨鍌溾偓娈垮枟濞兼瑨鐏冮梺閫炲苯澧紒鍌氱Ч楠炲棜顧佹繛鎾愁煼閺屾洟宕煎┑瀣碘偓妤侇殽閻愬澧紒缁樼〒娴狅箓宕掑锝呬壕婵犻潧顑呴弰銉╂煏韫囧﹤澧查柛搴ｅ枛閺屾洘绻涢崹顔煎Б濡炪倖鎸诲钘夘潖濞差亜浼犻柛鏇ㄥ亐閸嬫捇骞栨担鑲濓附淇婇婵嗗惞闁绘繂鐖奸弻锟犲炊閳轰絿銉х棯閹规劕浜圭紒杈ㄦ尰閹峰懘妫冨☉姘偅婵°倗濮烽崑鐐烘偋閻樿钃熼柣鏃傚帶缁€鍐煠绾板崬澧版い鏃€甯″铏规嫚閳ヨ櫕鐏撻梺杞扮椤兘濡存笟鈧顕€宕煎┑鍡氣偓鍨攽閻愬弶顥為柛鈺侊功濡叉劙寮婚妷锔规嫼闁荤姴娲﹁ぐ鍐吹鏉堚晝纾界€广儱鎳忛ˉ鐐电磼閸屾氨肖闁圭懓瀚幏鍛村即椤忓棛袦閻庤娲忛崝宥囨崲濞戞瑥绶為柛婵勫劤濞夊潡姊婚崒娆掑厡闁硅櫕鎸婚弲鑸电鐎ｎ亞锛涢梺闈浥堥弲娑氬婵犳碍鐓忛煫鍥э攻閸ｄ即鏌涚€ｎ偅灏甸柟鍙夋尦瀹曠喖顢楅崒锔惧枠闂傚倷绀侀幉鈥愁潖瑜版帒鍨傞柛褎顨呰繚婵炶揪绲跨涵鍫曞几鎼淬劍鐓欓悗娑欘焽閻矂鏌涢幘鍗炲婵﹥妞藉畷顐﹀礋椤旂偓顎囬梻浣侯焾閺堫剟宕欓悷鎷旀稑顭ㄩ崼鐔叉嫽闂佺鏈懝楣冨焵椤掑嫷妫戞繛鍡愬灲閺佹捇鎮╅懠顒夋Ф婵犵數鍋涘Λ娆撳垂瑜版帗鍎楅柟鍓х帛閻撶喖鏌曡箛鏇炐ｉ柛鐔哄仱閺屾稓鈧綆浜滈埀顒€娼″濠氭晲閸涘倻鍠栭幊鏍煛娴ｄ警鍋ч梻鍌欒兌缁垶骞愰懡銈囩煓闁硅揪鑵归埀顒婂閹瑰嫰濡搁敃鈧壕顖氣攽閻愬弶鈻曞ù婊冪埣閸┾偓妞ゆ帊绶″▓婊堟煛鐏炲墽娲寸€殿噮鍣ｅ畷鎺戔槈濞嗘劗妲楅梻鍌欐祰椤曟牠宕板Δ鍛偓鍐川鏉堝墽绋忓┑鐘绘涧椤戞劙寮崘顔界厪闊洦娲栧暩闂侀潧鐗滈崳锝咁潖缂佹ɑ濯撮柛娑橈攻閸庢捇姊洪崫鍕⒈闁告挾鍠栭獮鍡涘礋椤栵絾鏅┑鐐村灦椤ㄥ牏绮诲鑸碘拺闁兼祴鏅╅悞楣冩煟韫囨柨鍝虹€规洘娲熼幃銏ゅ礂閼测晛甯楅梻渚€娼чˇ顓㈠磿閹惰姤鏅€广儱顦伴悡鐔搞亜閹扳晛鐏╂い蹇ｅ幗閹?whitelist 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鎹愭硾鍟搁梺鍛婏供閸ㄨ泛顫忕紒妯诲闁告稑锕ㄧ涵鈧梻浣侯焾缁ㄦ椽宕愬┑瀣ラ柛鎰靛枛瀹告繈鏌℃径瀣仴闁诲寒鍙冨铏圭矙閹稿孩鎷辩紓浣割儐閹告儳顕ｈ閸┾偓妞ゆ帒瀚埛鎺楁煕鐏炵偓鐨戝褎绋戦妴鎺戭潩椤撗勭杹閻庤娲樺ú鐔肩嵁閸ヮ剚鍋嬮柛顐犲灩楠炲牓姊绘笟鈧褔鎮ч崱娑樼疇闁归偊鍘藉▍鐘绘煥閺囩偛鈧綊鎮″☉姘ｅ亾閸忓浜鹃梺閫炲苯澧寸€规洑鍗抽獮妯兼嫚閼碱剙濮︽俊鐐€栫敮濠囨嚄閸洖鐓€闁哄洨鍠嗘禍婊勩亜閹捐泛浠︾€瑰憡绻勭槐鎺楊敊绾拌京鍚嬫繝纰夌磿閸忔﹢宕洪敓鐘茬＜婵犲﹤鍟粻鐐烘⒒閸屾瑨鍏岀紒顕呭灠铻為柛鎰靛枛缂佲晠姊洪鈧粔鎾⒒椤栨稓绠剧€瑰壊鍠曠花濂告煟閹惧磭绠婚柡灞剧洴婵＄兘骞嬪┑鍡樼亾婵炲鍘ч崯鎾箖濡も偓閳绘捇宕归鐣屽蒋闂備線娼荤紞鍥╁緤娴犲鍋╅柣鎴ｆ缁犳岸姊洪銊╂濡ゆ柨鈹戦悩鎰佸晱闁哥姵鐩敐鐐村緞閹扳斁鍋撻崘顔奸唶闁靛繆妲呭鐔兼⒑閸︻厼鍔嬫い銊ユ噽缁顫濇潏銊ユ瀾閻庡箍鍎卞ú銊х矆婵犲洦鐓涚€广儱楠告禍鐐电棯閹冩倯闁靛洤瀚板顕€宕掑☉娆戝涧闂備胶鎳撻崯鍨洪妸鈺佺劦?	//   2. action=pass 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鎾煙椤斿吋鍋ユい銏＄懄閹便劑骞囬鍡欐晨闂傚倷绀侀幖顐ょ矓閺屻儱绀夐幖娣妼閻ゎ噣鏌＄仦璇插姕闁绘挾濞€閺屸剝寰勭€ｎ亶鍤嬮梺绋款儍閸庣敻寮婚敍鍕勃闁告挆鍕灡闂備浇妗ㄧ粈渚€宕愭繝姘闁绘顕х粻鐢告煙閻戞ɑ顥旈柛鐔锋处缁绘繈鎮介棃娴讹絾銇勯弮鈧悧鐘茬暦閹剁瓔鏁嬮柍褜鍓欓悾宄邦煥閸喎浜滈梺缁樻尭濮橈箓骞楅弴鐐╂斀闁绘劖娼欓悘鐔兼煕閵娿儲璐＄紒顔款嚙閳藉濮€閿涘嫬骞堥梻濠庡亜濞诧箓骞愭繝姘疅闁归棿鐒﹂悡鏇㈡煙鐎涙绠樼紒澶屽劋閹便劍绻濋崘鈹夸虎閻庤娲﹂崑濠冧繆閻戣棄唯闁挎棃鏁崑鎾寸節濮橆厸鎷洪梺闈╁瘜閸樻悂骞忛敓鐘崇厱閻庯綆浜峰銉╂煥閺囨ê鐏紒妤冨枛閸┾偓妞ゆ巻鍋撴い顐㈢箰鐓ゆい蹇撳椤斿洭鏌熼懝鐗堝涧缂佹彃娼￠幃楣冩濞戞帗鏂€闂佺粯鍔欓·鍌炲吹鐎ｎ剛纾奸柣妯挎珪鐏忣參鏌ｉ敐鍥у幋闁诡喚鏅划娆戞崉閵娿儱袝濠碉紕鍋戦崐鏍暜閹烘柡鍋撳鐓庡籍闁?raw "fast" 闂傚倸鍊搁崐鎼佸磹閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞鐟滃繘寮抽敃鍌涚厽闁靛繈鍩勯悞鍓х磼閹邦収娈滈柡宀€鍠栭弻鍥晝閳ь剟寮稿☉銏＄厱闁靛濡囩粻鐐搭殽閻愬澧懣鎰亜閹哄棗浜鹃梺璇叉禋娴滄繄鎹㈠☉銏犲窛妞ゆ牗顕撮敐鍚冲酣宕惰闊剙鈹戦垾宕囧煟鐎规洏鍔戦、姗€鎮㈤崜鎻掓暭闂傚倸鍊风粈渚€骞栭鈶芥稑鈻庨幋婵嗙亰閻庡厜鍋撻柛鏇ㄥ亞閸樻挳姊虹涵鍛涧闂傚嫬瀚板畷鎴﹀箛閻楀牏鍘电紓鍌欑劍閿氬ǎ鍥跺灠闇夐悘蹇旂墪娴滈箖姊婚崒姘偓椋庣矆娓氣偓楠炲鏁撻悩顔瑰亾閸愵喖骞㈡俊銈呭暞濞堟洟姊鸿ぐ鎺擄紵缂佲偓娓氣偓閹锋垿鎮㈤崗鑲╁幗闂佸搫鍟ú锕傤敂閻樼數妫柟顖嗕礁浠梺鍝勬湰濞叉繄绮诲☉銏犲嵆闁绘劖鍔х紞渚€寮婚敐鍫㈢杸闁规崘娉涢埅杈╃磽娴ｄ粙鍝洪柟绋款煼楠炲繘宕ㄩ弶鎴狀唽闂佺懓鎼粔鎾夊鑸碘拺閻犲洦鐓￠妤呮煕濡崵鐭掔€规洘鍨块獮妯肩磼濡厧骞堥梻浣告惈濞层垽宕濈仦鐐珷濞寸厧鐡ㄩ悡鏇熶繆椤栨粎甯涢柛濠冨姍閺岀喖顢欑粵瀣杹闂佺粯渚楅崳锝呯暦瑜版帩鏁婇柣鎾冲瘨濞兼稓绱撻崒姘偓鐑芥嚄閸撲礁鍨濇い鏍ㄧ矋閺嗘粓鏌ㄩ悢鍝勑㈢紒鐘崇墵閺岀喖顢涢崱妤佸櫤婵炲牓绠栧娲箹閻愭彃濮夐梺鍝勬噺缁诲牓寮鍜佺叆闁割偆鍟块幏?"priority" 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厾缁炬澘宕晶缁樹繆閼碱剙鍘存慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍缂傚倷绀侀鍕濮橆剛鏆﹂柕蹇ョ磿椤╃兘鎮楅敐搴濈敖闁挎稒鐩铏规崉閵娿儲鐏佹繝娈垮枤閺佸骞冮檱缁犳盯骞欓崘顏勬暩闂佽崵濞€缂傛艾鈻嶉敐鍡樻珷闁哄洢鍨洪悡?body闂?	//      闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锔剧磼閵娧呭笡濞ｅ洤锕幃娆擃敂閸曘劌浜鹃柡宥庡亝閺嗘粓鏌熼悜姗嗘當缁炬儳婀辩槐鎾存媴鐠囷紕鍔烽梺宕囩帛濮婂鍩€椤掆偓缁犲秹宕曢柆宓ュ洦瀵肩€涙ê浜楀┑鐐叉閹稿摜鐥閺屾盯顢曢敐鍥╃暤闂佹娊鏀卞Λ鍐蓟閿濆鏅插璺侯槹閸犳岸姊?native /responses 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆浂鐒炬い銉ョ箻閺屾稓鈧絺鏅濈粣鏃傗偓瑙勬礃濞叉ê顭囪箛娑樼厸濞撴艾娲﹂ˉ锟犳⒒娴ｄ警娼掗柛鏇炵仛閻ｅ墎绱撴担鍝勭彙闁搞儯鍔庨崢鍗烆渻閵堝棗濮﹂柛瀣缁傚秴顭ㄩ崗鐘垫嚀椤劑宕熼鐐╁悅闂傚倸娲らˇ鎵崲濠靛洨绡€闁稿本鍐荤槐鐐测攽閻愭彃鎮戦柛鏃€顨堝Σ鎰板箳閺冨倻锛滃┑鈽嗗灠閸氬宕氬☉妯滄棃鎮╅棃娑楃捕濡炪倖鍨甸幊姗€宕洪姀鈩冨劅闁靛牆娲ㄩ弶鎼佹⒑閸濆嫭澶勬慨妯稿妼铻炴い鏍仦閳锋帡鏌涚仦鍓ф噯闁稿繐鐬肩槐鎺楊敋閸涱厾浠梺杞扮贰閸ｏ綁鐛幒鎴富闁绘垟鏅涙禍楣冩煥閺囩偛鈧綊宕愭繝姘厾闁诡厽甯掗崝姘瑰鍕垫當妞ゎ亜鍟存俊鍫曞幢濡も偓琛肩紓鍌欒兌婵敻宕归崸妤€绠栨繛鍡樺姉缁♀偓闂佸憡娲﹂崜娆撳几閹达附鐓欓柛蹇氬亹閺嗘﹢鏌涢弬璺ㄧ伇缂侇喖顭峰濠氬Ψ閿旀儳寮虫繝鐢靛█濞佳兾涢鐐嶏綀銇愰幒鎾跺幗?"fast" 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鏁愭径濠勵吅闂佹寧绻傞幉娑㈠箻缂佹鍘辨繝鐢靛Т閸婂綊宕戦妷鈺傜厸閻忕偠顕ф慨鍌溾偓娈垮枟閹告娊骞冨▎寰濆湱鈧綆浜欐竟鏇㈡偡濠婂懎顣奸悽顖涘笧婢规洘绻濆顓犲弳闂佺粯鏌ㄩ幖顐ｇ墡闂備胶绮幐鍝ユ崲濮椻偓瀵鈽夐姀鈺傛櫇闂佹寧绻傚ú銊╂偩濞差亝鍊甸柣鐔哄閸熺偟绱掔拠鎻掝伀婵″弶鍔欓獮鎺懳旀担闀愭睏闂備焦鍎冲ù姘跺磻閹烘嚦娑㈠礃椤旇В鎷洪梺鍛婄箓鐎氼厼锕㈤幍顔剧＜閻庯綆鍋呯亸鎵磼閸屾稑娴柡浣稿暣瀹曟帒顫濇潏鈺傛瘞闂佽崵鍠愮划宀勬煀閿濆宓侀柛鈩冦仜濡插牓鏌曡箛濠冾€嗛柟閿嬫そ濮婄粯绗熼崶褌绨介梺绋款儐閻╊垶骞婇悢纰辨晬婵炴垶鐟﹂悵宄邦渻閵堝棛澧遍柛瀣仱瀹曟垿濡搁敂杞扮盎闂佸搫绉查崝濠冪濠婂牊鐓曢悗锝庡亝瀹曞本顨ラ悙鍙夊闁瑰嘲鎳樺畷鐑筋敇閻橆偅鐎遍梻鍌欐祰瀹曠敻宕伴幇顔煎灊閹兼番鍨哄▍鐘充繆閵堝懎鏆為柡鍡樼矒閺岀喖鎮滃Ο鐑╂嫽婵炴垶鎸哥粔鐢垫崲濞戙垹绠ｉ柣鎰仛閸ｏ絾绻涚€涙鐭嬬紒顔芥崌瀵鏁撻悩鑼紲濠殿喗锚瑜板鎼规惔銊︹拺闁规儼濮ら弫杈ㄦ叏婵犲偆鐓奸柛鈹惧亾濡炪倖甯掗崰姘焽閹邦厾绠鹃柛娆忣槺婢х敻鏌熼鍝勫姦闁诡喒鍓濋幆鏃堫敊閻ｅ瞼鈻夊┑鐘垫暩閸嬫稑螣婵犲啰顩叉繝闈涱儏閸氬綊鏌熼悜姗嗘畷闁绘挸绻橀弻娑㈩敃閿濆洨鐣洪梺闈╃到閹诧紕鎹㈠☉妯兼殕濠电姳绶氶崑妤€鈹戦纭锋敾婵＄偠妫勯悾鐑藉Ω閿斿墽鐦堥梺绋挎湰缁牆菐椤斿墽纾介柛灞捐壘閳ь剙鎽滅划鏃堝箻椤旇姤娅囬梺闈涚墕椤︿即宕戦埡鍛厽闁硅揪绲鹃ˉ澶岀磼閻欏懐绉柡灞诲姂瀵潙螖閳ь剚绂嶆ィ鍐╁€甸柣鐔哄閸熺偟绱掔拠鎻掓殻濠碉紕鏁诲畷鐔碱敍濮橆剙鏁ゆ俊鐐€ら崢浠嬪垂閸洘鍋╅柤鍝ユ暩缁犻箖鎮楀☉娆樼劷闁活厼鐭傞弻娑氣偓锝庡亝瀹曞本顨ラ悙鑼ⅵ濠碘剝鎮傛俊鐑筋敊閽樺绱﹂梻鍌欒兌绾爼宕滃┑瀣ㄢ偓鍐川鏉堝墽绋忓銈呯箰鐎氬嘲銆掓繝姘厪闁割偅绻傞弳娆撴煕閺冣偓閿曘垽寮诲☉銏犖╅柕澹憛銊╂⒑缁洘娅囬柛瀣ㄥ€濋悰顔锯偓锝庡枟閺呮繈鏌嶈閸撶喖骞冨Ο铏规殾闁搞儻绲芥禍楣冩偡濞嗗繐顏紒鈧埀顒€鈹戦悙棰濆殝缂佺姵鎹囬悰顔藉緞閹邦厼娈愰梺鍐叉惈閸熶即鏁嶅┑鍥╃閺夊牆澧介崚浼存煙閼恒儳鐭掗柟顔惧仧缁瑥螞閻㈠灚鍤€妞ゎ厹鍔戝畷姗€顢旈崟顓炲箺婵犵數濮烽。浠嬪礈濠靛绠伴柛鎰皺閺嗭箓鏌熼悜妯虹劸婵炲皷鏅犻弻锝夊箻閸愬樊鍔夊┑鐐叉噹閿曨亜顫?
	//      completions 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆浂鐒炬い銉ョ箻閺屾稓鈧絺鏅濈粣鏃傗偓瑙勬礃濞叉ê顭囪箛娑樼厸濞撴艾娲﹂ˉ锟犳⒒娴ｄ警娼掗柛鏇炵仛閻ｅ墎绱撴担鍝勭彙闁搞儯鍔庨崢鍗烆渻閵堝棗濮﹂柛瀣缁傚秴顭ㄩ崗鐘垫嚀椤劑宕熼鐐╁悅闂傚倸娲らˇ鎵崲濠靛洨绡€闁稿本鍐荤槐鐐测攽閻愭彃鎮戦柛鏃€顨堝Σ鎰板箳閺冨倻锛滃┑鈽嗗灠閸氬宕氬☉妯滄棃鎮╅棃娑楃捕濡炪倖鍨甸幊姗€宕洪姀鈩冨劅闁靛牆娲ㄩ弶鎼佹⒑閸濆嫭澶勬慨妯稿妼铻炴い鏍仦閳?normalizeResponsesBodyServiceTier 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟版晥濠电姭鍋撳〒姘ｅ亾婵﹨娅ｇ槐鎺懳熼搹閫涚礃婵犵妲呴崑鍕偓姘煎枤閸掓帗绻濆顓炰汗缂傚倷鐒﹂…鍥储閻㈠憡鈷戠痪顓炴媼濞兼劙鏌涢弮鎾剁暤鐎规洟娼ч埢搴ㄥ箻閺夋垟鍋撻崹顐ｅ弿婵妫楁晶缁樹繆閹绘帞绉洪柡灞炬礋瀹曟儼顦叉い蹇ｅ幗椤ㄣ儵鎮欓弶鎴炶癁濡ょ姷鍋涢鍛村煘閹达箑绠涙い蹇撴噹娴滄繈姊婚崒姘偓鐑芥倿閿曞倸绠栭柛顐ｆ礀缁€澶屸偓鍏夊亾闁告洦鍋嗛敍娆撴偡濠婂懎顣奸悽顖涘笧缁鎮欓悜妯煎帾婵犵數鍋涢悘婵嬪礈婵犳碍鐓熼柡鍐ㄥ€哥敮鍓佺磼閻樺崬宓嗘鐐寸墬濞煎繘宕滆閵堢兘姊洪柅鐐茶嫰婢ь垶鏌涢弮鈧悷鈺侇嚕婵犳碍鏅插璺侯儏娴滄粓姊洪崨濠勭細闁稿孩鐓″畷鐢稿磼濠婂懐锛濇繛杈剧到閹碱偅鐗庨梻浣告贡閹虫挸煤椤撱垺鍋樻い鏇楀亾鐎规洘锕㈤垾锕傚箣閻愯尙绱﹂梻鍌欑窔閳ь剛鍋涢懟顖涙櫠鐎涙﹩娈介柣鎰絻閺嗘瑩鏌嶇拠鏌ュ弰妤犵偛顑夐幃娆撴嚑椤掑﹦纭€闂傚倷绀侀幖顐λ囬柆宥呯；闁绘柨鎲″▍鐘充繆閵堝懏鍣洪柛搴㈡崌閻擃偊宕堕妸锔惧弰缂?
	//      闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呮／闁诡垎宀€鍚嬮梺鍝勭焿缂嶄線鐛崶顒夋晩闁兼亽鍎查惁搴ㄦ⒒娴ｈ銇熼柛妯圭矙閹兘鍩￠崨顓℃憰闂佺粯妫侀妴鈧柛瀣崌閹棄鈻撶捄銊ュЪ濠电偛顕慨宥夊炊閵娿垺瀚介梻浣告啞濞诧箓宕戦崟顖ｆ晜妞ゆ挶鍨洪悡娑氣偓鍏夊亾閻庯綆鍓涢惁鍫ユ倵鐟欏嫭绀冮柨鏇樺灪娣囧﹪骞栨担鑲濄劑鏌曡箛鏇炐″瑙勬礋濮婃椽妫冮埡浣烘В闂佸憡蓱閹倸鐣烽幋锕€绠婚柟棰佺劍閸嶉潧顪冮妶鍡楃瑨妞わ箓浜跺畷銉р偓锝庡枟閳锋垿鏌涘┑鍡楊伀闁绘帟娉曠槐鎾愁吋閸曨収妲銈嗘穿缂嶄線銆佸Δ鍛妞ゆ劑鍨婚埀顒夊幖椤啴濡堕崱姗嗘⒖濠碘槅鍋勭€氫即宕洪埀顒併亜閹烘埈妲搁柣蹇ラ檮閹便劍绻濋崘鈹夸虎閻庤娲﹂崑濠傜暦閻旂⒈鏁嗛柍褜鍓熼、鎾斥槈閵忊檧鎷洪梺鍛婄☉閿曘儳浜告导瀛樼厽闁冲搫锕ら悘锔锯偓瑙勬礃濡炰粙宕洪埀顒併亜閹哄秹妾峰ù婊勭矒閺岀喖宕崟顒夋婵炲瓨绮撶粻鏍ь潖濞差亜宸濆┑鐐寸閸ㄥ綊宕氶幒妤€绠荤€规洖娲﹀▓楣冩⒑绾懏褰х紒鐘冲灩缁鎳￠妶鍥╋紳婵炶揪缍€椤曟牠鎮炴禒瀣厱婵せ鍋撳ù婊嗘硾椤繘鎼圭憴鍕瀭闂佸憡娲﹂崜娑㈠礄閿熺姵鈷戦柛娑橈工閻掑綊鏌涚€ｎ偅灏电紒杈ㄦ尰閹峰懘妫冨☉姗嗘綂濠电姵顔栭崰鎾诲磹濠靛洣绻嗛柣銏㈩焾缁€瀣亜閺嶃劍鐨戦柣顐㈠濮婃椽鏌呴悙鑼跺濠⒀冾嚟閳ь剝顫夊ú妯兼崲閸岀偛鐓濋幖娣€楅悿鈧梺鎸庣箓濡稓绮欐担铏圭＝闁稿本鑹鹃埀顒傚厴閹偤鏁冩担瑙勫櫡婵犵數濮甸鏍垂闁秴绠伴柟鎯版閽冪喖鏌ㄥ┑鍡╂Ч闁稿瀚槐鎺斺偓锝庡亽閸庛儲淇婇銏☆棤缂佽鲸鎸婚幏鍛存偪椤栨艾甯梻浣侯焾缁绘垿鏁嬪銈庡亜缁绘劗鍙呭銈呯箰鐎氼亞妲愰崼鏇熲拺闁告稑锕ユ径鍕煕濡亽鍋㈤柛鈺傜洴瀹曞ジ濡烽敂瑙勫闂備礁鎲＄粙鎴︽晝閵壯傜剨闁规鍠掗崑鎾诲礂婢跺﹣澹曠紓鍌欑椤戝棝顢栧▎鎾冲惞闁哄洢鍨洪悡鐘电棯閺夊灝鑸瑰褜鍨抽埀顒冾潐濞叉牠藝椤栫偛违闁圭儤鍩堝鈺呮煟閹捐櫕鎹ｅù鐙€鍨跺娲川婵犲嫭鍣у┑鈽嗗亝缁诲倿锝炶箛鎾佹椽顢旈崟顐ょ崺濠电姷鏁告慨鎾磹閹间緤缍栫€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灩閳规垿鎮欑拠褍浼愬銈嗘穿缂嶄礁顕ｆ繝姘ㄩ柨鏇楀亾濞存粌婀辩槐鎾诲磼濞嗘垵濡介柤瑁ゅ€濋弻鐔兼煥鐎ｎ偁浠㈠┑顔硷工椤嘲鐣烽幒鎴僵闁告鍎愰弶绋库攽閻愬瓨灏扮痪鏉跨Ч閵嗗啴宕ㄩ鐐点偒闂傚倷绀佹竟濠囧磻閸涱劶鍝勵潨閳ь剟骞冮敓鐘插嵆闁靛繒濮烽鎰攽閻戝洨绉甸柛鎾寸懇閹﹢鍩￠崨顔惧帗缂傚倷鐒﹁摫缂佲偓閸儲鐓冪憸婊堝礈濮橀鏁婇柡宥庡幖缁愭绻涢幋娆忕仼闁哄嫨鍎甸弻鈥愁吋鎼粹€茬敖闂佹椿鍘奸ˇ鐢稿蓟閻斿吋鍊绘俊顖濆吹閸樻悂姊虹粙娆惧剱闁圭懓娲顐﹀箛椤撶喎鍔呴梺鐐藉劥鐏忔瑩寮弽顓熲拻濞达絽鎲￠崯鐐淬亜椤撶偟澧曢摶鐐烘煕閺囥劌鐏犻柣鎾达耿閺岀喐娼忔ィ鍐╊€嶉梺绋款儐閸旀瑩骞冭ぐ鎺戠倞闁搞儜鍐╂缂備焦鍞荤紞浣割潖閾忚瀚氶柟缁樺俯閸斿姊洪崨濠傜仴缂傚秴锕ら悾宄邦煥閸喎浜滅紓鍌欑劍閿曨偆妲愰幘缁樷拻濞达絿鐡旈崵娆撴煕閹寸姵娅曠紒杈╁仱瀹曞崬鈽夊Ο鑲╁炊婵犲痉鏉库偓鏇㈠箠韫囨稑纾婚柛宀€鍋為悡鐔兼煛閸屾氨浠㈤柟顔藉灴閺岋綁骞橀崘鍙夋闂佸搫鑻粔闈涱焽椤忓牆绠ユい鏇炴噺琚﹂梻鍌欑閹碱偄螞濞嗘挻鍋￠柍鍝勬噹閽冪喖鏌ㄥ┑鍡欏ⅱ闁汇倐鍋撻梻浣烘嚀閻忔繈宕鈶╂灁闁割偅娲橀埛鎴︽⒑椤愩倕浠滈柤娲诲灡閺呭爼顢涘☉鏍︾盎闂佸搫鍟犻崑鎾绘煟濡ゅ啫孝闁伙絿鍏橀獮搴ｇ驳鐎ｎ偅娅旈梻渚€鈧偛鑻晶顕€鏌ｉ敐鍥у幋妤犵偛顑夐幃娆撴偨閻㈠灚顫岄梻鍌欒兌椤牓寮甸鍌氬灊鐎广儱鎳夐弸鏃堟煟閹伴潧澧扮紒鈾€鍋撻梻鍌氬€搁悧濠勭矙閹达箑鐒垫い鎺嗗亾妞ゎ厾鍏橀崹楣冩晝閸屾岸鍞跺┑鐘绘涧濞诧箑鈻撴导瀛樷拻闁稿本鑹鹃埀顒勵棑濞嗐垹顫濋澶屽姺閻熸粌绻橀獮鎴﹀閵堝懐顔婇梺鍦亾濞兼瑥顕ｉ崸妤佲拺闁诡垎鍕洶闂佺顑呭Λ婵嬪箖閻㈢閱囬柡鍥╁枔閸樻悂姊洪幖鐐插姌闁稿酣浜堕幃姗€顢旈崼鐔哄帗闂備礁鐏濋鍛存倶閹绢喗鐓涢悘鐐插⒔閳藉鏌嶇憴鍕伌鐎规洖宕灃濠电姳鐒﹂崑鍛攽閿涘嫬浜奸柛濠冩礈閹广垽宕卞☉妯兼煣濠电娀娼ч悧濠囷綖閺囥垺鐓欓柣鎴炆戦埛鎺旂棯閹冩倯濞ｅ洤锕、娑樷攽閸ユ湹鍝楁俊鐐€栭崹鐢稿磹閸噮娼栭柣鎴炆戞慨婊堟煙濞堝灝鏋︾紒鎰殜濮婃椽妫冮埡鍐ょ紓浣藉紦缁瑩骞冩ィ鍐╁€婚柦妯侯槺椤㈠懘姊虹紒妯哄闁哄懏绮撻敐鐐差吋閸涱亝鏂€闂佺粯鍔曞鍫曀夐悙鐑樼厱闁靛ě鍕瘎闂?
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
				// pass闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ礂閻撳簶鍋撶紒妯圭箚妞ゆ牗绻嶉崵娆撴煛閸℃﹩娈滄慨濠傛惈閻ｇ兘宕堕妸銉︾暚缂傚倷绀侀ˇ顖滅礊婵犲偆鍤曢柡鍕禋濡胶绱撴担鍝勑ｉ柣鈺婂灦閻涱喖顫滈埀顒€鐣峰鈧幊鐘诲礈娴ｄ警鈧偓闂傚倸鍊搁崐鐑芥嚄閸洍鈧箓宕奸妷顔芥櫈闂佺硶鍓濈粙鎴犵不娴煎瓨鐓欓梻鍌氼嚟缁涘繒绱撳鍡欏缂佺粯绻堝Λ鍐ㄢ槈閸楃偛澹堥梺鍝ュ剱閸ㄨ泛顫忓ú顏咁棃婵炴垵纾涵鈧梻浣侯焾閺堫剛鍒掓惔銊︽櫖闁绘柨鍚嬮埛鎺楁煕鐏炴崘澹橀柍褜鍓熼ˉ鎾斥枎閵忕媭娼╅悹楦挎閸旓箑顪冮妶鍡楃瑨闁稿﹤鎲＄粋鎺戭煥閸喓鍘惧┑鐐跺蔼椤曆囨倶閿熺姵鐓涢柛娑卞幘閸╋絾銇勯姀锛勬噰鐎规洘绮忛ˇ鎾煥濞戞瑧鐭掓慨濠呮缁辨帒顫滈崱娆忓Ъ闂佽绻愮换鎴︽偡閿曞倸鐤鹃柕澶嗘櫆閻撶喖骞栧ǎ顒€鈧倕顭囬幇顓犵闁告瑥顦遍惌鎺擃殽閻愯韬鐐搭焽閹瑰嫰宕崟顓熺叄濠电姷顣槐鏇㈠磻閹达箑纾归柡鍥╁У瀹曟煡鏌熼柇锕€鏋撻柛瀣尭椤繈鎮℃惔锛勵啇闂佸憡顨夋ご鎼佸Φ閸曨垰绠抽柛鈩冦仦婢规洟姊绘担鍝ユ瀮妞ゎ偄顦靛畷褰掑锤濡も偓缁犳牗绻涢崱妯诲碍缂佺嫏鍥ㄧ厵閻庣數顭堝皬濠碘剝顨嗛幐鑽ゆ崲濠靛棌鏋旈柛顭戝枟閻忓秹鏌ら崹锕€娲﹂悡鏇㈡煛閸愶絽浜鹃梺鎼炲妿閺佸骞冩ィ鍐╁€婚柦妯侯槺娴煎姊鸿ぐ鎺濇殥闁绘帪绠撳畷鎾绘濞戣鲸瀵岄梺闈涚墕濡瑧浜搁幍顔剧＜妞ゆ棁鍋愭晶銏ゆ煠妤﹀潡鍝虹紒缁樼箓椤繈顢樺☉娆忣伖闂傚倷绀侀幉锛勬崲閸屾壕鍋撳鐓庢灕闁愁亜缍婂缁樻媴閸涘﹤鏆堥梺鍝勮閸旀垿寮绘繝鍥ㄦ櫜濠㈣泛鑻粊锔界節閻㈤潧孝婵炲眰鍊曢蹇撯攽鐎ｎ偄鈧灚绻涢幋鐐垫噽闁绘帊绮欓弻娑樜熼崷顓犵厯濠殿喖锕紓姘跺Φ閹版澘绠抽柟瀛樼矊閺嬪牓姊绘担鍛婂暈闁圭鐖煎畷婵嬪箣閿旇棄浜楀銈嗗笒鐎氱兘寮崱娑欑厱闁哄洢鍔屾晶浼存煕濡粯鍊愰柟顔筋殜瀹曟寰勬繝浣割棜闂備浇顕ч崙鐣岀礊閸℃稑绀堟繛鎴炲閸欏搫鈹戦悩鍨毄濠殿喚鍏橀妴鍌涚鐎ｎ亞顦┑鐘绘涧椤戝懐绮婚弽顓熺厱閻忕偛澧介埣銉х磼鐠囧弶顥㈤柡宀嬬秮楠炲鏁愰崱鈺傤棄闂佸摜鍎愰崹浼粹€旈崘顔嘉ч柛鈩冿供濮婂潡鎮楃憴鍕婵炰匠鍡欎罕婵＄偑鍊栭悧妤呭春閸愵喖缁╁ù鐘差儐閻撶喐淇婇娑欍仧闁哥喎绻橀弻锟犲幢閳轰胶浠搁梺鍝勮嫰缁夌兘篓娓氣偓閺屾盯骞樼€靛憡鍣伴悗瑙勬处閸ㄦ娊骞戦崟顖毼╃憸搴ㄦ晬韫囨稒鍋℃繝濠傚椤ュ牏鈧鍠栭…鐑藉箖閵忋垹鏋堥弶鍫涘妽濞呮捇姊绘担铏瑰笡闁圭鎽滈懞閬嶅醇閺囩偟锛涢梺瑙勫礃椤曆呯不閻熸噴褰掓晲閸ャ劌娈屾繛瀵稿Т閵堟悂骞冨Δ鍛仭闁哄顑欐导鍐⒑缁嬫鍎忛柨鏇樺€濋垾锕傚炊椤掆偓閻撴稑霉閿濆懎顏柣鈺婂灦楠炲啳顦圭€规洖銈告慨鈧柕蹇嬪灪濮ｅ洭姊婚崒娆戭槮闁圭⒈鍋婇獮濠呯疀閺囩喎搴婇梺鍓插亝缁诲秴顭囬弽銊х鐎瑰壊鍠曠花鍏笺亜閵夈儳澧涚紒缁樼洴楠炲鎮欑€靛憡顓婚梻浣风串缂嶄胶绮婚弽褜娼栨繛宸簼閻掑鏌ｉ幇顖氳敿閻庢碍婢橀…鑳檨闁搞劌鐖煎濠氭晬閸曘劌浜鹃柨婵嗙凹缁ㄧ敻鏌涚€ｃ劌鐏柍?"fast"闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴︽煙閼测晛浠滈柍褜鍓氶悧鐘茬暦濠靛鍐€妞ゆ挾鍊ｉ敃鍌涚厱闁哄洢鍔岄悘鐘绘煕閹般劌浜鹃梻鍌欑窔濞佳嗗櫣闂佸憡渚楅崹鎵暜閹惧墎纾介柛灞捐壘閳ь剟顥撻幏瀣蓟閵夈儳锛涢梺鍦亾閺嬪ジ寮ㄦ禒瀣厽婵☆垵顕х徊濠氭煃瑜滈崜娑㈠极鐠囧樊鍤曢柕濠忓椤╃兘鎮楅敐搴′簽闁告妫勯埞鎴﹀煡閸℃浠村銈嗘肠閸涱厾绛忔繝鐢靛У绾板秹鎮″☉銏＄厱闁斥晛鍟伴幊鍐煛鐎ｎ偆娲撮柡宀嬬磿娴狅箓鎮剧仦婵勫€楃槐鎺撴綇閵娿儳鐟ㄩ柧浼欑稻缁绘盯鎳犻鍌氱濡炪們鍎遍悧鎾愁潖缂佹ɑ濯撮柛娑橈攻閸庢捇姊洪崗鍏笺仧闁搞劌纾崚鎺楀籍閸喎鈧姊洪幑鎰劷闁告柨绉剁划顓㈡偄閻撳海鍔﹀銈嗗笒鐎氼剟鎷戦悢鍝ョ闁瑰瓨鐟ラ悘鈺冪磼閻樺樊鐓奸柟顔肩秺閹煎綊鎮烽弶鍨瀱闂備浇顕ф鎼佲€﹀畡閭︽綎闁绘垶锚椤曡鲸绻涢崱妯虹仸闁稿瑪鍥ㄢ拺缂佸娉曠粻鏌ユ煥閺囨ê鐏查柣娑卞櫍瀹曟﹢顢欓懖鈺佸箰闂備礁鎲￠崝蹇涘疾濞戙垺鍋熷ù鐓庣摠閳锋垿鏌涢敂璇插箻閻㈩垱鐩幃浠嬵敍濞戞ɑ璇為悗娈垮櫘閸嬪﹥淇婇懜闈涚窞閻庯急鍕伜婵犵數鍋犻幓顏嗗緤閸ф绠犻柟鐐た閺佸绻濇繝鍌氼伀缂佺娀绠栭弻娑樷枎韫囷絾笑闂佽绻戦幐鎶藉蓟閻斿吋鎯炴い鎰靛亝閻や線鎮楃憴鍕闁靛牏顭堥锝夊箻椤旇偐鍔?"priority"
				// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍓欑痪褔鏌ｉ妶鍛伃婵﹦绮幏鍛存惞閻熸壆顐奸梻浣告啞濡垹绮婚幘宕囨殾婵犲﹤鐗婇弲婵嬫煃瑜滈崜鐔煎春閵忋倕绠婚悗闈涘濞村嫰鏌ｆ惔顖滅У闁稿妫濆畷銏ゆ偨閸涘ň鎷婚梺绋挎湰閼归箖鍩€椤掍焦鍊愭い銏″哺椤㈡﹢鍩楅崫鍕枠闁轰礁鍟村畷鎺戔槈濮橆剙绠洪梻鍌欑窔濞佳囨偋閸℃﹩娈界紒瀣氨閺嬪秶鈧箍鍎遍ˇ浼存偂濞戞埃鍋撻崗澶婁壕闂侀€炲苯澧寸€规洑鍗冲浠嬵敇閻樿尙銈﹂梻浣侯攰閹活亞绮婚幋鐘典笉婵炴垯鍨洪埛鎺楁煕椤愩倕鏋旈柍顖涙礃閵囧嫰寮崹顕呬純濠?body闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灲閺岋紕浠︾拠鎻掝潎闂佺硶鏅濋崑銈夌嵁鐎ｎ喗鏅濋柍褜鍓涙竟鏇㈠礂閼测晝鐦堟繝鐢靛Т閸婂綊骞戦敐澶嬬厱闁哄啠鍋撻柟顔煎€搁～蹇涙惞鐟欏嫬鐝伴梺鑲┾拡閸撴盯顢欓幋婵冩斀闁绘劕寮堕崳鐑樼箾閸欏鐒界紒顔界濞煎繘濡歌閻﹀牓姊洪幖鐐插姤闁糕晜鐗犲鏌ヮ敆閳ь剟鍩為幋锔藉亹缂備焦蓱闁款厼鈹戦埥鍡椾簼妞ゃ劌锕悰顕€宕卞☉娆忊偓鐑芥煟閹寸們姘跺箯濞差亝鍊甸悷娆忓缁€鈧┑鐐额嚋缁犳挸鐣烽悽绋跨劦妞ゆ帒鍊荤壕钘壝归敐鍛棌婵″弶妞介弻鈩冩媴鐟欏嫬鈧劖顨ラ悙鏉戠伌濠殿喒鍋撻梺缁橈供閸嬪懘寮埀顒勬⒑鐠囨彃鍤辩紓宥呮瀹曟垿骞掗幊铏洴瀹曟﹢濡搁姀鈩冩澑闂備胶绮崝鏍ь焽濞嗘挻鍊堕柨鏇炲€归悡鐔兼煟閺冣偓濞兼瑩宕濆鑸电厓閻犲洩灏欏暩濡炪倧绠掗～澶愬箞閵婏妇绡€闁告劏鏂傛禒銏ゆ⒑閸︻収鍔滅紒缁樼箖娣囧﹪宕奸弴鐐茬獩濡炪倖妫冮弫顕€宕戦幘娣亝闁告劏鏂侀幏娲⒑閸涘﹦鈽夐柨鏇樺劦閸╂盯骞囬悧鍫㈠幗濠德板€撻懗鍫曟儗閹烘搩娈介柣鎰綑閻忔潙鈹戦鐟颁壕闂備胶绮敃鈺呭磻閸涜揪缍栫€广儱顦伴埛鎴︽⒒閸喓銆掔紒鐘插暱閳规垿顢欓幆褍骞嬪銈冨灪閻熲晠鐛幒鎳虫梹鎷呴崣澶婎伖闂傚倷娴囬鏍垂鎼淬劌绀嬫い鎾楀嫮鍊為梻鍌氬€烽懗鍫曗€﹂崼銉晞闁告侗鍨崑鎾愁潩椤撶喓鍑″銈嗘穿缂嶄線鐛惔銊﹀癄濠㈣埖鍔曢弫褰掓⒒娓氣偓濞佳嚶ㄩ埀顒勬煠闂堟稓绉烘鐐茬箰鐓ゆい蹇撴噽閸樺憡绻濋姀锝嗙【闁兼椿鍨辩粋鎺楁晝閳ь剟鈥﹂崸妤佸仭闁绘鐗滃Λ锕傛煢濡崵绠為柡灞诲姂瀵剟宕归瑙勫瘱缂備焦鍎虫晶浠嬪疾椤愶箑鐒垫い鎺嗗亾缂佺姴绉瑰畷鏇㈡焼瀹ュ懐顔嗛梺鐟扮摠缁诲秹宕崨瀛樼厪濠㈣泛妫欏▍鍡涙煟閹捐泛鏋涢柡灞诲妼閳规垿宕卞Ο鐑橆仩婵犲痉銈呯毢缂侇噮鍨舵俊鐢稿礋椤栵絾鏅濋梺鎸庣箓濞层劑鎮炬總鍛娾拺闁革富鍘介崵鈧柣搴㈢煯閸楁娊鐛弽顓炴嵍妞ゆ挾鍠庨悵浼存⒑閻愯棄鍔ユ繛鍛礋瀹曟劙顢涘☉姘辩槇缂佺偓婢橀ˇ杈╁閸ф鐓曢悗锝庡亜閻忓鈧娲橀崝娆忣嚕閸婄噥妾ㄥ┑鐐插悑閻楁洟鍩為幋锔藉亹閻庡湱濮撮ˉ婵嬫⒑濮瑰洤鈧宕戦幘璇参﹂柛鏇ㄥ灠缁犳盯鏌涢幘鑼跺厡闁哄棛濮撮埞鎴︻敊濞嗙偓缍堥梺缁樻惈缁绘繂顕ｆ繝姘櫜闁告稑鍊婚崰鏍箠閺嶎厼鐓涘ù锝勮閸嬔囨⒒閸屾瑧鍔嶅┑鐐诧躬瀵劑鏌嗗鍛€梺绉嗗嫷娈旈柦鍐枑缁绘盯骞嬮悙鍐╁哺瀵劍绂掔€ｎ偆鍘遍梺鏂ユ櫅閸犳艾鈻撻姀鐘嗗綊鎮╅棃娑樹粯濡炪値浜滈崯瀛樹繆閸洖宸濇い鏍ㄧ矤閸炲綊姊绘担鐑樺殌闁规祴鍓濈换娑欑節閸屻倕娈ㄩ柣鐘叉处缁佹潙危閸儲鐓忛煫鍥ㄦ礀椤秹鏌嶉崫鍕櫤闁绘挸绻橀弻娑㈩敃閿濆洨鐣洪梺闈╃到濠€杈╂閹烘挾鐟归柛銉戝嫮褰庨梻浣筋嚃閸犳銆冮崼銉ョ疅闁圭虎鍠栫粈瀣亜閹烘垵鈧敻寮鍥╃＝闁稿本鐟ч崝宥夋煟閹垮嫮纾跨紒顔碱煼瀵粙濡搁妶鍥╃暰闂備礁澹婇崑鍛哄鈧鏌ュ箵閹烘繄鍞甸柣鐘烘鐏忋劑宕濋悢鎼炰簻闁挎柨鎼慨鍌涙叏婵犲懏顏犻柍褜鍓欏﹢杈ㄦ叏閻㈢违闁告劦鍠楅悡鐔兼煙閾忕懓浠ч柣锝囨暬閺岀喖顢氶埀顒傜不閺嶎厼钃熼柛鈩冾殢閸氬鏌涢垾宕囩閻庢俺妫勯埞鎴︽倷閼搁潧娑х紓浣藉蔼閸嬫劙骞忛崘顔芥櫇闁稿本绋戦埀顒€鐏濊灃闁挎繂鎳庨弳鐐烘煟閹惧崬鍔﹂柡宀嬬節瀹曞爼寮甸悽鍨櫦闂備椒绱徊鍧楀极閹间礁鐒垫い鎺戝枤濞兼劖绻涢崣澶婄伌鐎规洩绻濋獮搴ㄦ偩鐏炵晫銈︽繝纰樻閸垳鎷冮敃鍌涘亜闁糕剝绋掗悡鏇㈡煛閸モ晛浠滄い锝呯－缁辨帗寰勫Ο鐑樼亾闂侀€炲苯澧い鏃€鐗犲畷鏉课旈崨顓狀唶闂佹儳娴氶崑鍛村矗韫囨稒鐓冪憸婊堝礈閻旂厧钃熼柕濞垮劗濡插牊淇婇娆掝劅濞寸姷鍘ц灃闁绘﹢娼ф禒婊堟煟濡や焦灏い顐㈢箳缁辨帒螣鐠囧樊鈧捇姊洪崨濠勨槈闁挎洏鍔庡☉鐢稿焵椤掑嫭鈷掑ù锝勮閻掑墽绱掗妸锔姐仢鐎规洘鍔曡灃闁告侗鍘鹃澶愭⒑闂堟稓绠為柛濠冩礈婢规洟宕楅懖鈺冾啎闂佺懓顕崑娑㈠吹椤掍胶绠鹃柛娆忣槹閸婃劖鎱ㄦ繝鍛仩缂佽鲸甯掕灒闁告繂瀚峰鎾翠繆閻愵亜鈧牕煤閿曞倸鍌ㄩ柛鎾楀啫鐏婄紓鍌欑劍鑿ч柛瀣嚇閺屾盯骞囬妸锔界彇闂佸憡锚瀹曨剟鈥旈崘顔嘉ч柛鈩冾殔椤洭姊虹粙鍖℃敾婵炶尙鍠庨锝夊川婵犲啫鍔呴梺闈涱煭婵″洭宕㈡禒瀣棅妞ゆ劑鍨洪幖鎰版煥閺囨ê鍔氶柍璇茬У缁绘繈宕堕妸褍骞愰柣搴″帨閸嬫捇鏌嶈閸撶喎鐣锋导鏉戝唨鐟滃繘寮抽敃鍌涚叆婵犻潧妫欓ˉ婊呯磼閸撲礁浠遍柟顔筋殜閺佹劖鎯斿┑鍫㈡晨闂備浇銆€閸嬫挻銇勯幘鍗炵仾闁绘挾鍠栭獮鏍庨鈧埀顑惧€曢…鍥箛椤撶姷顔曢梺鍛婄懃椤﹂亶鎯岄幒鏂哄亾鐟欏嫭纾婚柛妤€鍟块锝嗙鐎ｅ灚鏅ｅ┑鐘欏嫬鍔ゅù婊勫劤闇夐柨婵嗘川閵嗗﹪鏌＄€ｎ亪鍙勯柡宀€鍠栭幃娆擃敆娴ｈ櫣鈻忔繝鐢靛仜閻楀﹪宕濆▎蹇ｆ綎婵炲樊浜滅粈鍐煕濞嗗浚妲归柛搴㈡崌濮婅櫣绮欓崠鈩冩暰闂佸憡姊归悷锔界┍婵犲洤绠瑰ù锝呮憸閸樻悂姊虹粙鎸庢拱妞ゃ劌妫涢埀顒佷亢濡嫰鍩為幋锔藉€烽柤鎼佹涧濞懷呯磽娴ｈ棄钄奸柛瀣姍楠炲繘宕橀鑺ユ珖闂佺鏈粙鎴﹀焵椤掆偓閻忔繆鐏冮梺鎸庣箓閹冲酣寮抽悙鐑樼厽闁规儳宕崝锕傛煛瀹€瀣瘈鐎规洖鐖煎鐢告偨閸偅娅﹂梻鍌欐缁鳖喚寰婃禒瀣€舵慨妯挎硾妗呴梺鍛婃处閸ㄤ即锝為崨瀛樼叆婵炴垶锚椤忊晠鏌嶅畡鎵劯闁诡喗顨呴埢鎾诲垂椤旂晫浜堕梻浣筋潐閻℃洖鈻嶉弴銏犵疄闁靛ň鏅涢悞鍨亜閹烘垵顏柣鎾跺枑閹便劌顪冪拠韫婵犵數鍋橀崠鐘诲礋椤掆偓瀵?
				if normTier != rawTier {
					reqBody["service_tier"] = normTier
					bodyModified = true
					markPatchSet("service_tier", normTier)
				}
			}
		}
	}

	if IsImageGenerationIntentMap(openAIResponsesEndpoint, reqModel, reqBody) && !imageGenerationAllowed {
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalFeatureGate)
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

	// 闂備礁鎲＄粙鎺楀垂濠靛鍤?WS 闂備礁鎼崯顖氾耿閸︻厼鍨濋幖杈剧稻鐏?WebSocket Mode闂備焦瀵х粙鎺旂矙閹邦喚绠斿鑸靛姇缁€鍐偓骞垮劚閻楀棗鈻撴导瀛樼厱闁哄倽顕ф俊鍏肩箾閸欏澧甸柡灞芥嚇閸┾偓?HTTP闂?
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

				s.handleFailoverSideEffects(ctx, resp, account, upstreamModel)
				return nil, &UpstreamFailoverError{
					StatusCode:             resp.StatusCode,
					ResponseBody:           respBody,
					RetryableOnSameAccount: account.IsPoolMode() && (account.IsPoolModeRetryableStatus(resp.StatusCode) || isOpenAITransientProcessingError(resp.StatusCode, upstreamMsg, respBody)),
				}
			}
			return s.handleErrorResponse(ctx, resp, c, account, body, billingModel)
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
			MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
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
	// 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鏁愭径濠勵吅闂佹寧绻傞幉娑㈠箻缂佹鍘遍梺闈涚墕閹冲酣顢旈銏＄厸閻忕偛澧藉ú瀛樸亜閵忊剝绀嬮柡浣瑰姍瀹曞崬鈻庡Ο鎭嶆氨绱撻崒姘偓鐑芥倿閿曞倸绀夐柡宥庡幑閳ь剙鍟村畷濂稿Ψ閵壯冨Е婵＄偑鍊栧濠氬磻閹惧墎纾奸柣妯垮皺鏁堥悗瑙勬礃濞茬喎鐣烽敓鐘冲€剁紓浣股戦妵婵嗏攽閳╁啯鍊愰柛鈹惧墲閹峰懏绗熼娑欐殢婵犵數濮甸鏍窗濡ゅ應鈧箓宕奸妷銉︽К闂佸搫绋侀崢鑲╁瑜版帗鐓曟い顓熷灥娴滅偟绱掗悩鑽ょ暫闁哄瞼鍠撻埀顒傛暩椤牓宕滈崡鐏诲綊鎮℃惔銏╂＆闂佸搫鐬奸崰鎾诲焵椤掑倹鏆╂い顓炵墛缁傛帒顭ㄩ崼鐔哄幈闂侀潧顭梽鍕夐崼銉︾厸鐎光偓閳ь剟宕伴幘鑸殿潟闁圭儤顨呴～鍛存煟濡櫣锛嶅ù婊冪埣濮婄粯鎷呮笟顖滃姼濡炪倖鍨堕崹鍦閻愬鐟归柍褜鍓欓锝嗙節濮橆厼浜滅紒鐐妞存悂寮查敐澶嬧拺缂備焦蓱椤ュ牊銇勯妷锔藉磳闁诡喚鍏樻俊鎼佹晜閸撗呮缂備礁澧芥晶妤勫綔闂佺顑嗛幑鍥箠閻樻椿鏁嗛柛灞剧矊楠炲棝姊?upstream 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍嵁瀹ュ鏁婄痪鎷岄哺濮ｅ姊绘担渚劸妞ゆ垶鍨归幑銏犫攽鐎ｎ亣鎽曢梺鍝勬川閸犲海娆㈤悙鐑樼厱妞ゆ劧绲跨粻鎾愁熆鐟欏嫭鐓ラ柍瑙勫灴閹瑩寮堕幋鐘辨闂備焦瀵уú宥夊疾閻樿绠栭柨鐔哄Т鍞梺鍐叉惈閸婂宕㈤崡鐑嗘富闁靛牆妫楁慨锕傛煕鐎ｎ亜顏柡鍛埣楠炲秹顢欓崜褝绱查梺璇插嚱缂嶅棙绂嶉悙鐢典笉濞寸厧鐡ㄩ悡鍐偡濞嗗繐顏╅柣蹇ラ檮椤ㄣ儵鎮欑拠褑鍚梺鍝勬湰缁嬫帞鎹㈠┑瀣闁冲搫鍠氬Σ瑙勪繆閵堝洤啸闁稿鐩弫鍐Ω瑜忔稉宥嗙箾閹寸們姘ｉ崼鐔剁箚妞ゆ牗鍤庨埀顒€顑囧Σ?model闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ礂閻撳簶鍋撶紒妯圭箚妞ゆ牗绻冮鐘裁归悩灞傚仮婵﹥妞藉畷顐﹀礋椤掑顥ｆ繝纰樻閸嬪懐鎹㈤崼婵愬殨濠电姵纰嶉崑鍕煕韫囨挾姣為柟宄邦煼閺岋絾鎯旈妸锔介敪婵犮垻鎳撻敃銈呭祫濡炪倖娲嶉崑鎾存叏婵犲懏顏犵紒杈ㄥ笒铻ｉ柧蹇撴贡閹冲棝姊绘担绛嬪殭缂佺粯鍨甸悾婵嬪川婵犱胶绠氶梺姹囧灮椤牏绮堢€ｎ偁浜滈柡宥冨妽閻ㄦ垶銇勯弬鍖¤含婵﹨娅ｉ崠鏍即閻愭祴鎷ら梻渚€娼уú銊╁磻閻愮數鍗氶柣鏃傚帶閸楁娊鏌ｅΟ鍏兼毄闁挎稒绻堝铏圭矙閹稿孩鎷遍梺鑽ゅ櫐婵″洨鍒掓繝鍥ㄥ亱闁割偅绋愮花濠氭⒑閸濆嫭鎼愭俊顐ｎ殔闇夐柛宀€鍋涢弸渚€鏌涢弴銊ョ仭闁绘挻娲熼弻鏇熷緞濡儤鐏撴繛瀵稿У缁捇寮婚妸銉㈡斀闁糕檧鏅滅瑧缂傚倷鑳舵慨鐢告儎椤栫偛钃熼柍鈺佸暙缁剁偤骞栧ǎ顒€鐏柕鍡楋工椤啴濡惰箛鎾舵В闂佸憡顭囬弲顐﹀箲閵忕姭妲堟繛鍡樺姉缁夊爼姊洪崨濠冨瘷闁告洦鍋傛潻姗€姊婚崒娆掑厡妞ゃ垹锕ら埢宥夊即閵忕姷顔囬梺缁樺姦閸撴盯鎮甸崼鏇熺厸闁搞儯鍎遍悞娲煛娴ｅ摜校闁逛究鍔岃灒闁告繂瀚崐顖炴煟鎼淬劍娑ч柣顓炲€搁～蹇撁洪鍜佹濠电偞鍨堕懝楣冦€傞崫鍕ㄦ斀闁宠棄妫楁禍婵堢磼鐎ｎ偄娴€殿喗鐓″畷濂稿即閵婏附娅屽┑鐐舵彧缁插潡骞婇幘鎰佸殨闁割煈鍠掗弨浠嬫煃閽樺顥滈柣蹇ョ稻閵囧嫰顢橀悙闈涱杸婵烇絽娲ら敃锕傚箲閸曨垰惟鐟滃繘鏁嶅鍐炬富闁靛牆妫欑粈鈧┑鈽嗗亝缁嬫挾鍒掗弮鍫晪闁逞屽墴瀵鎮㈢喊杈ㄦ櫓闂佷紮绲介張顒勫闯瑜斿濠氬炊瑜滃Ο鈧梺璇″枟椤ㄥ﹪寮幇顓熷劅闁炽儴灏崺鍛節?body 闂傚倸鍊搁崐鎼佸磹閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞妞ゆ帒顦伴弲顏堟偡濠婂啰效闁挎繄鍋涢埞鎴犫偓锝庡亝濞呭洭姊虹粙璺ㄧ効濠碘€虫川缁瑨绠涢弮鍌滅槇闂侀潧楠忕徊浠嬫偂閹扮増鐓曢柡鍐ｅ亾闁绘濮撮悾鐑藉箣閿曗偓鍥存繝銏ｆ硾椤戝洭宕㈤幖浣光拺闁硅偐鍋涢崝妤呮煛閸涱喚鈽夐柣顭戝墴濮婄粯鎷呴搹鐟扮闁汇埄鍨辩敮锟犲极閸愵喖唯鐟滃骸鐣烽崣澶岀闁瑰瓨鐟ラ悘鈺傤殽閻愵亜鐏ǎ鍥э躬椤㈡稑鈹戦崶鏈靛摋濠电偛顕慨鐢稿箰閸愬樊娼栨繛宸簻缁犲綊鏌ｉ幇顓炵祷妞ゎ剙顦辩槐鎾存媴缁涘娈梺缁橆殕閹告悂顢氶敐鍥ㄥ珰鐎瑰壊鍠栭幃鎴炵節閵忥絾纭鹃柨鏇畵瀹曘垽宕ㄦ繝鍕啎闁哄鐗嗘晶浠嬪礆娴煎瓨鐓欐繛鏉戭儌閸嬫捇寮妷锔句簴闂備礁鎲￠悷銉┧囨潏鈺冧笉?compact 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻樻爠鍥ㄧ厱閻忕偛澧介悡顖氼熆鐟欏嫭绀€闁宠鍨块、娆戠驳鐎ｎ剙濮洪梻浣告啞椤棝宕惰閸ゆ垿姊虹紒妯荤叆闁告艾顑夐幃鈥斥枎韫囧鍋撻幒鎴僵妞ゆ帒鍊烽搹搴㈢節濞堝灝鏋涢柛濠傜秺閵嗗啴濡烽埡鍌氣偓鐑藉级閸喎绀冮柍褜鍓氱€笛囧Φ閸曨垰顫呴柨娑樺閸ｄ即姊洪崷顓х劸闁挎洏鍎遍銉╁礋椤掑倻鐦堥梺绋胯閸婃牠藟?+
	// OAuth normalize闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ偒鍎ラ柛搴＄焸閺岋繝宕ㄩ姘ｆ瀰閻庢鍠栭…鐑藉极閹邦厼绶炲┑鐐╂噰閺呯娀寮婚弴銏犻唶婵犲灚鍔栨闂備礁鎲￠悷銉ヮ熆?婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈瀛濈紓鍌氱Т閿曨亜顕ｇ拠宸悑濠㈣泛锕ｇ槐鍫曟⒑閸涘﹥澶勯柛鎾寸懃閳诲秹鏁愭径瀣ф嫼缂備礁顑堥崕濠氾綖閿曞倹鐓曢柡鍐ｅ亾闁搞劌鐏濋锝嗙節濮橆厼浜滅紒鐐妞存悂寮查鍕拺缂侇垱娲嶉崑鎾崇暦閸モ晩鍞跺┑鐐茬摠缁挾绮婚弽褜娼?model 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻濠氭煕閳规儳浜炬俊鐐€栫敮濠囨嚄閸洖鐓濋柟鍓х帛閻撴盯鏌涘☉鍗炴灓缂佺姵锕㈤弻娑㈠箳閹惧磭鐟ㄩ梺瀹狀嚙闁帮綁鐛Ο铏规殾闁搞儴娉涢弫钘夆攽閻樻鏆滅紒杈ㄦ礋瀹曟垵鈽夐姀鈥冲壄闂佺粯鍨煎Λ鍕婵犳碍鐓欓柟瑙勫姦閸ゆ瑧绱掗埀顒勫礃閳瑰じ绨婚梺鍝勫暙閸婂摜鏁崼鏇熺厾闁哄娉曟禒銏ゆ煃鐟欏嫬鐏撮柟顔界懇瀵爼骞嬮悩杈╃婵犵绱曢崑娑㈡偤閵娾晛绠栭柛灞惧嚬閸ゆ洟鏌＄仦璇插姎闁绘挻鐩弻娑樷槈閸楃偞鐏堥梺閫炲苯澧伴柡浣割煼瀵鈽夊鍛澑闂佺懓鐏濋崯顖滅懅婵犵數鍋涢悺銊у垝閹惧墎涓嶉柡宓本缍庡┑鐐叉▕娴滄粌顔忓┑鍡忔斀闁绘ɑ褰冮弳娆愩亜閿旇娅婃慨濠呮缁瑥鈻庨幆褍澹堟繝纰樻閸嬪懐鎹㈤崼鐔剁箚閻庢稒顭囬悷褰掓煃瑜滈崜娆撴偩閻戣棄惟鐟滃宕戦幘鏂ユ婵炲懓椴稿畝鎼佺嵁韫囨稒鎯為柛锔诲幘閿涙繃绻涙潏鍓у埌濠㈢懓锕畷鏇㈠箻鐎靛摜顔曟繝銏ｆ硾椤戝棛绮堥崼銏㈢＜妞ゆ柨澧界敮娑㈡倵娴ｅ啫浜归柍褜鍓氱粙鎺椻€﹂崶顒€鍌ㄩ柣銏犳啞閳锋垿鏌熼鍡楀椤╀即姊虹粙娆惧剰閻庢凹鍙冮獮鍫ュΩ閵夘喖鎮戞繝銏ｆ硾椤戝洭宕㈤棃娑辨富闁靛牆妫欓埛鎰版煕鎼淬垹鈻曠€规洏鍨婚埀顒佺⊕閿曗晛鈻撴禒瀣厽闁归偊鍓欑痪褔鏌涢悩鎴愭垿濡甸崟顖氼潊闁宠棄鎳撻埀顒€娼￠弻鐔碱敊鐟欏嫭鐝旈梺宕囩帛閹瑰洤鐣疯ぐ鎺濇晩闁兼亽鍎宠ぐ顐︽⒒閸屾艾鈧悂宕愰幖浣哥９鐎瑰嫭鍣磋ぐ鎺戠倞妞ゆ帒锕︾粙蹇旂節閵忥絾纭鹃柨鏇畵瀵娊鏁傞幋鎺旂畾濡炪倖鐗楃缓鍧楀焵椤戣棄浜鹃梻浣告惈椤戝懐绮旇ぐ鎺戣摕闁跨喓濮撮悙濠囨煏婢跺牆鍔ゅù鐘哄亹缁辨挻鎷呴崫鍕戙儵鏌涢悩宕囧⒌鐎殿喛顕ч埥澶娢熼柨瀣垫綌婵犳鍠楅〃鍛存偋閸℃ɑ鍙忔繛鎴炴皑绾捐棄霉閿濆懎绾фい搴℃閺屾稓鈧綆鍋呭畷灞炬叏婵犲嫮甯涚紒妤冨枛閸┾偓妞ゆ帒瀚悞鍨亜閹烘垵鈧憡绂掑鍫熺厾婵炶尪顕ч悘锟犳煛閸涱厾鍩ｆ鐐达耿椤㈡瑩鎮剧仦钘夌疄闂傚倷绶氬褑澧濋梺鍝勬噺缁诲牓骞冩ィ鍐╁€绘慨妤€妫欓鏃堟⒑缂佹ê濮囩€殿喛鍩栧鍕礋椤栨稓鍘甸悗鐟板婢ф宕虫禒瀣厵闁告瑥顦扮亸锔锯偓瑙勬礈閸犳牠宕洪悙鍝勭畾鐟滃本绔熼弴銏♀拺闁告稑锕ゆ慨锕傛煕閻樻剚娈滈柛鈺傜洴楠炴帡骞婇妸銉хШ闁轰焦鍔欏畷鍫曞Ω閵夛妇鏆氬┑锛勫亼閸婃牕煤濡厧鍨濋幖杈剧稻椤洟鏌熼悜妯烩拹閻庢碍宀搁弻鐔虹磼濡搫娼戦梺绋款儐閹歌崵鎹㈠┑鍡╂僵妞ゆ帒鍊告慨锔戒繆閻愵亜鈧牜鏁繝鍕焼濞撴埃鍋撶€规洜鏁婚、妤呭礋椤掑倸骞堥梻浣虹帛椤牆鈻嶉弴銏″剭闁硅揪闄勯悡鏇㈡煙閻戞ɑ灏繛鎼櫍閺屸剝鎷呴悷鏉款潚閻庤娲忛崝鎴濈暦閹烘垟妲堟繛鍡樕戦ˉ鍫ユ煟鎼淬値娼愭繛鎻掔箻瀹曟繈骞嬮敂琛″亾娴ｇ硶鏋庨柟鎯х－閻ｉ箖鎮峰鍐ょ紒顔碱煼瀹曨偊宕熼妸锔芥澑闂備焦瀵х粙鎴犫偓姘煎墯缁傚秵绺介崨濠勫幈婵犵數濮撮崯顖滅矆閸儲鐓欐鐐茬仢閻忓弶顨ラ悙鍙夊枠妞ゃ垺鐟╅幃閿嬶紣娴ｅ壊妫滃┑鐘愁問閸犳鏁嬮悗瑙勬处閸撶喎鐣烽姀鈶╁亾閻㈢數銆婇柛瀣尵閹叉挳宕熼鍌ゆО婵犵數鍋涘鍓佹崲閸曨厼鍨濋悹鍥ㄧゴ濡插牊淇婇鐐存暠妞ゎ偄绉撮埞鎴︽倷閸欏妫￠梺鎼炲妿閺佸鎮伴鈧畷濂稿Ψ閿旇瀚藉┑鐐舵彧缁蹭粙骞楀鍫濆嚑闁告劏鏅欑换鍡樹繆閻愰潧甯跺┑顔煎€块弻鐔兼惞椤愵偅鐤侀悗瑙勬礀閵堢顕ｉ幘顔藉亜閻忓繋绀佹禍楣冩煕瑜庨〃鍡涙偂濞嗘挻鈷掗柛灞惧嚬閸ょ喖鏌涢弬璺ㄐч柡?slug闂?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾剧懓顪冪€ｎ亝鎹ｉ柣顓炴閵嗘帒顫濋敐鍛婵°倗濮烽崑鐐烘偋閻樻眹鈧線寮村杈┬㈤梻浣规偠閸庢椽宕滈敃鍌氭瀬闁告劦鍠楅悡銉╂煛閸ヮ煈娈斿ù婊堢畺濮婂搫效閸パ€鍋撳Δ鍛；闁规崘鍩栧畷鍙夌節闂堟稒宸濈紒鈾€鍋撻梻浣侯焾閺堫剛鍒掑畝鍔肩兘鍩€椤掑嫭鈷掑ù锝勮閻掔偓銇勯幋鐐茬仼闁瑰箍鍨归埞鎴犫偓锝庝海閹芥洟姊洪棃娴ュ牓寮插☉銏犵闁规儼濮ら悡鐔兼煙闁箑骞橀柕鍫熸尦閺屾稓鈧綆鍋呭畷鍕煏閸ャ劌濮嶆鐐村浮楠炲鎮崨顖氳€块梻鍌欐祰椤曆囧礄閻ｅ瞼绀婇柛鈩冪☉缁愭淇婇妶鍛櫣闁绘挻娲熼弻鐔衡偓鐢殿焾娴滅厧霉濠婂嫮鐭掗柡宀€鍠栧畷顐﹀礋椤掑顥ｅ┑鐐茬摠缁本鏅堕悾灞绢潟闁规儳鐡ㄦ刊鎾煕閹惧啿绾ч弫鍫ユ⒒娴ｇ儤鍤€闁规祴鍓濈换娑欑節閸モ晛绁﹂梺绯曞墲閻熴倕鈻介鍫濈骇闁绘劖娼欓婊勬叏婵犲倻绉烘鐐茬箻瀹曘劑寮堕幋鐙€妲梻浣告啞缁哄潡宕曢崘娴嬫灃闁秆勵殕閳锋帡鏌涚仦鍓ф噮妞わ讣绠撻弻鐔哄枈閸楃偘绨界紓渚囧枟濡啫顕ｆ繝姘ㄩ柕澹本啸闂備胶鎳撻崥瀣偩椤忓牆绀夌€光偓閳ь剛鍒掓繝姘亹閻熶椒绀佺紞濠囧极閹版澘閱囬柣鏇氭閸濇绱撴担鍝勪壕闁稿孩濞婃俊鍫曞箹娴ｆ瓕鎽曢梺鍝勬川閸犳挾寮ч埀顒勬椤愩垺澶勭紒瀣浮椤㈡挸顓奸崶鈺冿紳婵炶揪缍€濡嫰鎮￠幇鐗堢厽闁瑰灝瀚弧鈧悗?chat-completions / messages / native /responses 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆浂鐒炬い銉ョ箻閺屾稓鈧絺鏅濈粣鏃傗偓瑙勬礃濞叉ê顭囪箛娑樼厸濞撴艾娲﹂ˉ锟犳⒒娴ｄ警娼掗柛鏇炵仛閻ｅ墎绱撴担鍝勭彙闁搞儯鍔庨崢鍗烆渻閵堝棗濮﹂柛瀣缁傚秴顭ㄩ崗鐘垫嚀椤劑宕熼鐐╁悅闂傚倸娲らˇ鎵崲濠靛洨绡€闁稿本鍐荤槐鐐测攽閻愭彃鎮戦柛鏃€顨堝Σ鎰板箳閺冨倻锛滃┑鈽嗗灠閸氬宕氬☉妯滄棃鎮╅棃娑楃捕濡炪倖鍨甸幊姗€宕洪姀鈩冨劅闁靛牆娲ㄩ弶鎼佹⒑閸濆嫭澶勬慨妯稿妼铻炴い鏍仦閳?	// upstreamModel 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤濞硷繝寮婚悢琛″亾閻㈡鐒剧€涙繄绱撴担鐣屽牚闁稿﹥绻堝濠氭晝閳ь剝鐏掓繛鎾村嚬閸ㄩ亶鏁嶉崱妞绘斀闁绘劕寮堕埢鏇灻瑰鍕疄鐎殿喗褰冮…銊╁醇閻斿搫骞楅梻渚€鈧稑宓嗘繛浣冲嫭娅犳い鏇楀亾妤犵偞鐗犲鍫曞箣椤栨繂鎯堟繝娈垮枛閿曘儱顪冮挊澶屾殾闁靛濡囩弧鈧梺绋挎湰椤曟挳寮撮悢铏诡啎闁诲孩绋掗…鍥儗鐎ｎ喗鐓熸俊銈呭暙瀛濋梺浼欑悼閸忔﹢鐛€ｎ喗鏅濋柍褜鍓涚划缁樼節濮橆厼浠梺鎼炲劘閸斿瞼寰婃繝姘厽閹兼惌鍠栭獮鏍ㄣ亜椤忓嫬鏆ｅ┑鈥崇埣瀹曞崬螖閳ь剝銆栫紓鍌氬€风粈浣割嚕鐠轰警娓诲ù鐘差儑瀹撲線鐓崶銊р槈闁圭鍩栭妵鍕箻鐠虹洅銉╂煥濞戞瑧绠栫紒缁樼⊕濞煎繘宕滆閸╁本绻濋姀銏″殌闁挎洦浜滈悾鐑藉即閿涘嫮鏉搁梺鍝勫€介濠勮姳婵犳碍鈷戦柟绋垮閳锋帡鏌涚€ｎ偅宕岀€殿喗濞婇、妤呭磼濡も偓娴滈箖鎮峰▎蹇擃仾缂佲偓閸愨晙绻嗛柣鎰閻瑧鈧娲樺浠嬪春閳ь剚銇勯幒宥夋濞存粍绮撻弻鐔煎传閸曨厜褎淇婇幆褍妲婚棁澶嬬節婵犲倸顏柣顓熷浮閺屸€崇暆閳ь剟宕伴弽顓炵畺婵犲﹤鍚橀悢鍏兼優闂侇偅绋掑Ο濠傗攽閿涘嫬浜奸柛濠冩礀闇夋慨姗嗗幘椤╁弶銇勮箛鎾跺闁藉啰鍠愮换娑㈠箣濞嗗繒浠奸柛銉︽尦濮婃椽宕ㄦ繝鍕厐闂佸摜鍠愭繛濠傜暦閵夆晩鏁冮柨鏃囨閳ь剙鐏氱换娑㈠醇濠靛牅铏庨梺鍝勵儐閻╊垶寮婚敐澶婂唨妞ゆ劑鍨规禒姗€姊烘导娆戞偧闁稿繑锕㈤獮鍐焺閸愨晛鍔呭┑鈽嗗灣缁垶寮堕幖浣光拻濞达綀娅ｉ妴濠囨煃瑜滈崜娆戝椤撶姷涓嶉柡灞诲劜閸婄敻鏌涢…鎴濅簽濠⒀嗕含缁辨帗娼忛妸锕€纾抽悗瑙勬礃鐢帡鍩㈡惔銊ョ婵犻潧妫欓弳顓㈡⒒閸屾瑨鍏屾い顓炵墦椤㈡牠宕卞☉妯碱唶闂佸綊妫跨拋鏌ュ焵椤掑﹦鐣电€规洜顭堣灃濞达絽鎲￠悿鍌炴⒒閸屾艾鈧悂鎮ф繝鍕煓闁硅揪璁ｇ紞鏍偓骞垮劚椤︿即鍩涢幋锔界厵闁诡垱婢樿闂佽桨绶氭禍鍫曞蓟濞戙垹惟闁靛／鍐幗濠电姷顣槐鏇㈠极鐠囪尙鏆﹂柣鏃傗拡閺佸秵鎱ㄥ鈧涵鎼佸船濞差亝鈷掑ù锝堟閵嗗﹪鏌涢幘瀵哥疄濠碉紕鏁婚幃娆戔偓闈涙憸瑜伴箖姊洪崫鍕偍闁搞劍妞介崺娑㈠箳閹炽劌缍婇弫鎰板炊閳哄倹鍟掗梻浣规偠閸婃牠鎮疯濡叉劙骞掗弮鍌滐紲濠碘槅鍨靛▍锝嗙婵傚憡鈷戦柣鐔稿閻ｎ參鏌涢妸銊︻棄闁伙絿鍏橀幃鈩冩償濡粯鏉搁梻浣稿閸嬪棝宕伴幇閭︽晩?whitelist 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鎹愭硾鍟搁梺鍛婏供閸ㄨ泛顫忕紒妯诲闁告稑锕ㄧ涵鈧梻浣侯焾缁ㄦ椽宕愬┑瀣ラ柛鎰靛枛瀹告繈鏌℃径瀣仴闁诲寒鍙冨铏圭矙閹稿孩鎷辩紓浣割儐閹告儳顕ｈ閸┾偓妞ゆ帒瀚埛鎺楁煕鐏炵偓鐨戝褎绋戦妴鎺戭潩椤撗勭杹閻庤娲樺ú鐔肩嵁閸ヮ剚鍋嬮柛顐犲灩楠炲牓姊绘笟鈧褔鎮ч崱娑樼疇闁归偊鍘藉▍鐘绘煥閺囩偛鈧綊鎮″☉姘ｅ亾閸忓浜鹃梺閫炲苯澧寸€规洑鍗抽獮妯兼嫚閼碱剙濮︽俊鐐€栫敮濠囨嚄閸洖鐓€闁哄洨鍠嗘禍婊勩亜閹捐泛浠︾€瑰憡绻勭槐鎺楊敊绾拌京鍚嬫繝纰夌磿閸忔﹢宕洪敓鐘茬＜婵犲﹤鍟粻鐐烘⒒閸屾瑨鍏岀紒顕呭灠铻為柛鎰靛枛缂佲晠姊洪鈧粔鎾⒒椤栨稓绠剧€瑰壊鍠曠花濂告煟閹惧磭绠婚柡灞剧洴婵＄兘骞嬪┑鍡樼亾婵炲鍘ч崯鎾箖濡も偓閳绘捇宕归鐣屽蒋闂備線娼荤紞鍥╁緤娴犲鍋╅柣鎴ｆ缁犳岸姊洪銊╂濡ゆ柨鈹戦悩鎰佸晱闁哥姵鐩敐鐐村緞閹扳斁鍋撻崘顔奸唶闁靛繆妲呭鐔兼⒑閸︻厼鍔嬫い銊ユ噽缁顫濇潏銊ユ瀾閻庡箍鍎卞ú銊х矆婵犲洦鐓涚€广儱楠告禍鐐电棯閹冩倯闁靛洤瀚板顕€宕掑☉娆戝涧闂備胶鎳撻崯鍨洪妸鈺佺劦妞ゆ帊绶￠崯蹇涙煕閻樺磭澧甸柍銉畱閻ｏ繝鏌囬敃鈧▓銊╂⒑閸︻叀妾搁柛鐘崇墵閿濈偤宕ㄧ€涙鍘藉┑鈽嗗灠閻忔繈鎷曟總鍛婄厓鐟滄粓宕滃▎鎾村€舵繝闈涱儏閻撴﹢鏌熺€电浠滅紒鐘靛█濮婅櫣绮欓崠鈩冩暰闂佸憡姊归悷銉╂偩閻戣棄绠ｉ柨鏇楀亾缂佺姾宕电槐鎾存媴鐠囷紕鍔烽梺鍛娚戦幑鍥箖瀹勯偊鐓ラ柛鏇ㄥ幘閻撳鎮楃憴鍕缂侇喖鐭傞崺銏℃償閵娿儳顔掗悗瑙勬礀濞村倿宕崼鐔虹瘈闁汇垽娼у瓭闂佹寧娲忛崐婵嬬嵁婵犲懐鐤€婵炴垵纾粙?body 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈瀛濈紓鍌氱Т閿曨亜顕ｇ拠宸悑濠㈣泛锕ｇ槐鍫曟⒑閸涘﹥澶勯柛鎾寸懃閳诲秹鏁愭径瀣ф嫼缂備礁顑堥崕濠氾綖閿曞倹鐓曢柡鍌濇硶閻掑憡绻濋埀顒佺瑹閳ь剙顫忛搹鍦＜婵☆垰娴氭禍婊嗙亽婵犵數濮村ú銈囧閸ф鐓欓柟娈垮枛椤ｅジ鏌ｉ幘瀛樼妤犵偞鐗滈崚鎺旀喆閸曞灚缍夋繝纰樻閸亪宕戦幘缁樷拻濞达絽鎲￠幆鍫ユ煛閸偄澧扮紒顔界懇楠炲鏁冮埀顒勬偂閳ユ剚鐔嗛悹鍝勫娇閸儱鍑犻幖娣妽閻撴瑩姊洪銊х暠闁哄鍊濋弻宥囨嫚閼碱剛顔掑┑?	// model 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻濠氭煕閳规儳浜炬俊鐐€栫敮濠囨嚄閸洖鐓濋柟鍓х帛閻撴盯鏌涘☉鍗炴灓缂佺姵锕㈤弻娑㈠箳閹惧磭鐟ㄩ梺瀹狀嚙闁帮綁鐛Ο铏规殾闁搞儴娉涢弫钘夆攽閻樻鏆滅紒杈ㄦ礋瀹曟垵鈽夐姀鈥冲壄闂佺粯鍨煎Λ鍕婵犳碍鐓欓柟瑙勫姦閸ゆ瑧绱掗埀顒勫礃閳瑰じ绨婚梺鍝勫暙閸婂摜鏁崼鏇熺厾闁哄娉曟禒銏ゆ煃鐟欏嫬鐏撮柟顔界懇瀵爼骞嬮悩杈╃婵犵绱曢崑娑㈡偤閵娾晛绠栭柛灞惧嚬閸ゆ洟鏌＄仦璇插姎闁绘挻鐩弻娑樷槈閸楃偞鐏堥梺閫炲苯澧伴柡浣割煼瀵鈽夊鍛澑闂佺懓鐏濋崯顖滅懅婵犵數鍋涢悺銊у垝閹惧墎涓嶉柡宓本缍庡┑鐐叉▕娴滄粌顔忓┑鍡忔斀闁绘ɑ褰冮弳娆愩亜閿旇娅婃慨濠呮缁瑥鈻庨幆褍澹堟繝纰樻閸嬪懐鎹㈤崼銉у祦闁糕剝绋戦柋鍥煏婢跺牆鍔ゆい锔芥緲椤啴濡堕崱娆忣潷缂備礁顑呴悧鎾荤嵁韫囨拋娲敂閸涱亝瀚奸梻浣告啞缁嬫垿鏁冮妷褌鐒婇柟娈垮枟閸犳劙鏌℃径濠勪虎闁哄棛鍋熺槐鎺楀磼濮樻瘷銏ゆ煃鐟欏嫬鐏寸€规洖鐖奸崺锟犲礃閹勬珬闂傚倸鍊风粈浣革耿鏉堚晛鍨濇い鏍ㄧ矋閺嗘粓鏌ｉ幇顒傛憼婵炲懎绻樺铏规嫚閸欏鏀銈庡亜椤︻垳鍙呭┑顔姐仜閸嬫挾鈧娲栫紞濠傜暦缁嬭鏃堝礃閵娧佸亰濠电姷顣藉Σ鍛村垂娴兼潙瑙﹂悗锝庡厴閸嬫挸顫濋鐐叉懙闂佸搫琚崐妤呭窗婵犲洤纭€闁绘劕妯婂璇测攽閻樻剚鍟忛柛锝庡灦濮婁粙宕熼鍌楁敵婵犵數濮村ú锕傚磹闁垮浜滈煫鍥ㄦ尭椤忋倝鏌涚€ｎ偅宕岀€殿喕绮欓、鏇㈡晲閸℃﹩鍞堕梻鍌欑濠€閬嶅磿閵堝钃熼柨鐔哄Т绾惧鏌熼悙顒€澧繛灏栨櫊閺岋綁骞橀幎绛嬧偓妤呮煃瀹勯偊鍎旀慨?reqModel闂?
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
		MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalFeatureGate)
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
				streamWarnLogger.Warn("检测到超时相关请求头，将按配置过滤以降低断流风险")
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
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯闁靛／灞芥櫏闂備浇顕у锕傦綖婢跺⊕鍝勵潨閳ь剟骞冮敓鐘虫櫢闁绘灏欓悾鍝勨攽鎺抽崐鏇㈠箠韫囨稑鐤鹃柡灞诲劚缁犲湱绱掗鐓庡辅闁稿鎹囬獮鍥ㄦ媴鐟欏嫮銈板┑鐘殿暜缁辨洟宕戦幋锕€纾归柕鍫濐槸绾惧鏌涘☉鍗炲Ц婵炲樊浜濋崑鍕煟閹捐櫕鎹ｉ柣锝嗘そ閺岋綁鎮㈤崫銉﹀櫑闁诲孩鍑归崢浠嬪箞閵娿儺鐓ラ柛顐ゅ枔閸橀潧鈹戦悙鑼闁诲繑绻堝鎼佸Χ婢跺﹦顢呴棅顐㈡处缁嬫帡鍩涢幒鎳ㄥ綊鏁愰崼鐕佷哗闁汇埄鍨辩粙鎺楀箞閵婏妇绡€闁稿被鍊楅崥瀣倵鐟欏嫭纾搁柛銊ャ偢钘濈憸鐗堝笚閻撴瑦銇勯弮鍌滃彄妞ゅ繐鐗婇崑鈺呮煟閹达絾顥夐柣鎰躬閺屻劌鈹戦崱妯烘婵犮垼顫夊ú鐔煎蓟閿濆鍋勯柛婵勫劜閸Ｑ囨煟鎼淬垹鍤柛妯哄⒔閸掓帡宕奸妷銉у姦濡炪倖宸婚崑鎾绘煃鐟欏嫬鐏撮柟顔界懇閹崇娀顢楁担绋跨憥闂傚倷绀侀幉鈥愁潖瑜版帒鍨傞柟绋跨凹缁诲棝鎮楀☉娆樼劷闁荤喎缍婇弻宥堫檨闁告挻鐟╅幃楣冩倻閽樺）鈺呮煃閸濆嫸鏀婚柡鍛櫊濮婃椽鎳為妷鍐句邯钘濇い鏍ㄧ☉椤曢亶骞栧ǎ顒€濡介柍閿嬪笒闇夐柨婵嗘噺閸熺偤鎮归幇鍓佺瘈闁哄本绋掗幆鏂库槈濡嘲浜炬繝闈涱儑瀹撲線鏌熼柇锕€骞戦柛瀣嚇閺屾盯骞囬埡浣割瀷闂佸憡蓱閸旀瑥顫忓ú顏勪紶闁告洦鍓欑粣娑㈡⒑閸濄儱孝婵☆偅鐟х划瀣吋閸滀胶鍙嗛梺鍛婃磵閺呮瑧鑺辨繝姘拺閻熸瑥瀚粈鍐╃箾婢跺鈯曠紒鍌涘笩椤﹀弶銇勯鈥冲姷妞ぱ冨€垮濠氬礋椤愩埄浼冩繝纰夌磿閺佽鐣烽悢纰辨晬婵ê鍚嬬紞鍌炴⒒娴ｅ摜鏋冩俊顐㈠铻炴繛鎴欏灩缁€澶婎渻鐎ｎ亝鎹ｇ痪鎹愭闇夐柨婵嗗閻瑩鏌涘┑鍥ㄣ仢闁诡喕绮欓、娑樷槈濡⒈鍎庨梻浣筋嚙缁绘垹鏁敓鐘茶摕闁挎繂鐗忛悿鈧梺瑙勫劤婢у酣顢欐径鎰拺缂備焦锕╁▓鏇熴亜閵娿儻宸ユい鏇秮椤㈡洟鏁冮埀顒勫垂閸屾稏浜滈柟鎵虫櫅閳ь剚顨婇幆渚€宕奸妷锔规嫼濠殿喚鎳撳ú銈夋倶閸欏绠惧ù锝呭暱濞层倝鎮″┑瀣厱妞ゆ劑鍊曢弸鏃傜磼閻樺啿鍔ら棁澶愭煥濠靛棙鍣归柡鍡涗憾閺屾稑螣鐠囨彃濮㈤柣鎾卞€栭妵鍕疀閹惧磭寮搁悷婊冪Ч閳ワ妇鎹勯妸锕€纾繛鎾村嚬閸ㄤ即宕滄导瀛樷拺闁告稑锕ョ亸顏堟煕閺傝法鐒搁柍銉︽瀹曟﹢顢欓崲澹洦鐓曢柍鈺佸枤閻掍粙鏌ㄥ☉娆愮婵﹨娅ｇ划娆忊枎閹冨闂備礁婀遍幊鎾趁洪鐑嗗殨濠电姵纰嶉弲顒勬煕閺囥劋绨奸柛鏇炲暣濮婃椽宕ㄦ繝鍐ㄧ樂闂侀€炲苯澧崡閬嶆煕椤愮姴鍔滈柣鎿勭到闇夐柛蹇氬亹閹冲懐绱掓径妯烘珝闁哄备鈧磭鏆嗛悗锝庡墰琚︽俊銈囧Х閸嬬偤鏁冮姀鐘垫殾婵☆垯璀﹂崯鍛亜閺冨倸甯堕崬顖炴⒒閸屾艾鈧嘲霉閸ヮ剦鏁嬬憸鏃堝蓟婵犲洦鏅查柛銉㈡櫃缁楀姊洪崫鍕檨閻忕偛澧界粙渚€姊绘担鐟板姢缂佺粯鍔曢敃銏℃綇閳轰緡妫滈梺绋跨箰閸氬宕ｈ箛鏂剧箚闁靛牆鍊告禍鎯р攽閳藉棗浜濋柣鐔叉櫊楠炲啯銈ｉ崘鈺佹疅闂侀潧顦崐妤冪不濮樿埖鈷戠紓浣姑慨鍥煟鎺抽崝搴ｆ閻愬搫绠ｉ柨鏃傛櫕閸樼敻姊洪崗闂磋埅闁稿孩濞婇幆灞界暆閸曨剛鍘遍梺鍦劋閹稿摜绮閺屽秷顧侀柛鎾卞妿缁辩偤宕卞☉妯肩崶闂佸搫绋侀崑鍡樼▔瀹ュ鐓ｉ煫鍥风到娴滄绱掔拠鍙夘棦闁哄本娲熷畷鐓庘攽閸♀晙绱戦梻浣告啞閺屻劎绮旈悽鍨床婵炴垯鍨瑰婵囥亜閺傚灝鈷旈柣鎾愁槸閳规垿顢欑涵閿嬫暰濠碉紕鍋樼划娆撴偘椤曗偓瀵粙顢橀悢灏佸亾閻戣姤鐓欑紓浣姑粭鎺撱亜椤愶絽鐏存慨濠勭帛閹峰懘宕ㄦ繝鍌涙畼闂備礁鎼悮顐﹀礉閹达箑绠氱€光偓閸曨偆锛滈梺缁樺姇瀵泛危椤掑嫭鈷戦梺顐ゅ仜閼活垱鏅堕鐐寸厪闁搞儜鍐句純濡ょ姷鍋炵敮锟犵嵁鐎ｎ亖鏀介柛鎰╁妺婢规洟姊洪崨濠勨槈闁挎洩绠撻妴鍛存倻閼恒儮鎷哄銈嗗坊閸嬫挾绱掓径灞炬毈闁诡喒鈧枼妲堥柕蹇娾偓鏂ュ亾閸洘鐓熼柟浼村亰閺夋椽鏌涢妶鍡欐噧闁宠鍨块崹楣冩倷閽樺鍒掗梻浣告惈閼活垳绮旇ぐ鎺戠濠电姴鍟欢鐐翠繆椤栨稐鎲鹃柨鏇炲€归埛鎴︽倵閸︻厼顎岄柛銈嗙懇濮婅櫣鏁鍓滈梺缁樹緱閸犳岸鍩€椤掑﹦绉甸柛鐘崇墱閻氭儳顓兼径瀣帗閻熸粍绮撳畷婊堟偄閻撳孩鐎悗骞垮劚濡娆㈤悙纰樺亾閸忓浜鹃梺閫炲苯澧板?429/529 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻缁橆殽閻愭潙鐏村┑顔瑰亾闂侀潧鐗嗛幊鎰邦敊婵犲倵鏀介幒鎶藉磹閹版澘纾婚柟鎯у濡垶鏌熼鍡楃灱閸氬姊洪崫鍕効缂傚秳绀侀锝夘敋閳ь剙鐣烽幒鎴僵妞ゆ垼娉曠敮娑氱磽閸屾瑧鍔嶉柛搴ゆ珪閺呭爼鎮剧仦钘夌亰濡炪倖鐗楃粙鎾诲煘瀹ュ應鏀介柣妯哄级缁佲晠鏌℃担鍝バｇ紒缁樼箞濡啫鈽夊▎娆戠婵犵妲呴崑鍕偋婵犲嫭宕叉繝闈涙閺嬫棃鏌涢…鎴濇灍闁诲繑娲栭埞鎴︽倷閸欏娅ゅ┑鐐跺皺閸犳劖绌辨繝鍥ㄥ仺缂佸娉曢ˇ鏉款渻閵堝棛澧慨妯稿姂椤㈡瑦绻濋崶銊㈡嫼闂佽崵鍠愭竟鍡涘箺閻樼數纾兼繛宸簽閻ｈ櫣鈧娲橀崹鍧楃嵁濡偐纾兼俊顖滃劋椤撳ジ姊绘笟鈧褔鎮ч崱娑樼閻庯綆鈧厽绋戦…銊╁礋椤忓棛鐣鹃梻渚€娼ч悧鍡欐崲閹烘绀嗗ù鐓庣摠閻撴洟鏌ｉ弴姘鳖槮闁诲骏绠撻弻鐔碱敊閸忓浜鹃柡鍌樺劜閻庡姊洪崷顓炰壕婵炲吋鐟╅崺鈧い鎺嗗亾闁诲繑姘ㄩ幑銏犫槈閵忕姷顓洪梺缁樺姇閻忔岸宕虫禒瀣拺缂備焦锚缁椦囨煛閸滀礁浜炴俊鍙夊姍楠炴帒螖娴ｉ晲姹楅梻浣瑰劤濞存岸宕戦崟顖氭辈婵°倕鎳忛埛鎴︽煟閻斿憡绶茬紒鐙欏洦鐓欐い鏃傚帶閳ь剙娼￠幃浼搭敊绾板崬鎮戞繝銏ｅ煐钃遍柡鍛櫊閺岋綀绠涢幘鍓侇唹闂佺粯顨嗛〃濠囧箖濡や礁顕遍悗娑櫱氶幏缁樼箾鏉堝墽绉繛鍜冪悼閺侇喖鈽夐姀锛勫幍闁诲繐绻戦悧妤€鈻嶆繝鍕ㄥ亾鐟欏嫭绀冮柛銊ュ閹广垹鈹戠€ｎ亞锛滃┑鐘诧工鐎氼參顢欓弴銏♀拻濞达絿鐡旈崵鍐煕閵娿儳鍩ｇ€规洘鍨剁换婵嬪磻閺傘倗绋荤紒杞扮矙瀹曘劍绻涢悙顒€顏归梻鍌欒兌椤㈠﹪宕戦幇鏉跨；婵炴垯鍨归悞鍨亜閹哄秶鍔嶇紒鈧埀顒勬煟閹惧崬鈧牠濡甸崟顖氱闁告鍋熸禒濂告⒑閽樺鏆熼柛鐘崇墵瀵鏁撻悩鑼€為梺闈涱槶閸庤櫕绂掗悡骞棃鎮╅棃娑楁澀闂佹悶鍔庨崕銈囩矚鏉堛劎绡€闁搞儺鐏涜閺屾稑鈽夐崡鐐茬濠电偛鐗愬▔娑⑩€旈崘顔嘉ч柛鈩兦氶幏褰掓煟閵忊晛鐏茬紒缁樼箞閻涱噣宕橀妸搴㈡瀹曟﹢鍩℃担鍦偓顓㈡⒒娴ｅ憡鍟炴繛璇х畵瀹曟粌鈽夐埗鍝勬喘婵℃悂濡烽钘夌槣闂備線娼ч悧鍡涘疮椤愶絼绻嗗┑鍌氭啞閻撴瑩鏌ц箛锝呬壕闁兼澘娼￠弻娑㈠Ω閳哄啰鏆梺杞扮閸熸潙鐣烽幒鎳舵帞浜搁弽銊︾彋闂佸搫鑻粔闈涱焽椤忓牆绠悘鐐舵鐢垱淇婇悙顏勨偓褏鎷归悢鐓庣鐎光偓閸曘劉鍋撻弮鍫濈妞ゆ柨妲堣閺屾稑鈽夐崡鐐典化濡炪倖鏌ㄥú顓烆潖濞差亜浼犻柕澶堝劜濮ｆ劙姊洪崫鍕櫤缂佽鐗撳畷娲焵椤掍降浜滈柟鍝勭Х閸忓矂鏌嶇紒妯诲磳闁哄矉绻濆畷銊╊敍濮ｈ￥鍨洪妵?		// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼达絾鏆╂い顓炵墛娣囧﹥绂掔€ｎ偀鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜閻犲洩灏欐晶锔锯偓娈垮枛椤嘲顕ｉ幘顔藉亜濡炲娴烽悰顕€姊绘担铏广€婇柛鎾寸箚閹筋偊姊虹紒妯肩畺婵炶尙鍠庨～蹇涙惞閸︻厾鐓撳┑鐐叉閸庢娊宕滈弶娆炬富闁靛牆绻愰々顒勬煛娴ｇ瓔鍤欐い鏇悼閹风姴霉鐎ｎ偒娼旈梻渚€娼х换鎺撴叏閻戠瓔鏁婇柟鐑橆殕閳锋垿鏌ｉ幘宕囨槀闁稿鍨洪妵鍕Χ閸涱厸鍋撻崹顔炬殾妞ゆ帒鍟ㄦ禍褰掓煙閻戞ɑ灏ㄩ柟鐤缁辨挻鎷呴崜鎻掑壉濡炪倖鍨堕悷鈺佺暦閺囥垹绠涢柣妤€鐗忛崢鐢告⒑閼姐倕鏋斿褎顨婂畷鏉课熷ú缁橆啍闂佺粯鍔栬ぐ鍐汲閿濆鐓欐い鏃傛櫕閻帡鏌熺粙鍖℃敾鐎垫澘瀚悾婵囩節閸屾稑鈧垱绻濈喊澶岀？闁稿鍨垮畷鎰板冀瑜滃鏍煕閿旇骞愰柛瀣尭椤繈顢楅崟顐㈠強濠电姷顣介崜婵嬪箖閸岀偛鏄ラ柨鐔哄Т绾惧吋淇婇妶鍕槮妞ゅ繐缍婂濠氬磼濮橆兘鍋撻幖浣哥９闁归棿鐒﹂崑瀣煕閹伴潧鏋涢柣銈囧亾缁绘繈妫冨☉姘叡闂佹椿鍘介幐濠氬Φ閸曨垰绠崇€广儱鐗嗛崢锛勭磽娴ｅ搫鈻堢紒鐘崇墵瀵鈽夊鍡欏弳闂佸憡鍔戦崝宥呪枔閸撲胶纾藉ù锝呭濡牓鏌涢敐蹇曠М鐎殿喖顭烽幃銏㈠枈鏉堛劍娅旈梻浣告啞娓氭宕㈡禒瀣仧婵犻潧顑嗛埛鎴︽偣閸ワ絺鍋撻搹顐や憾闂備浇宕甸崯鍧楀疾閻樿违濞达綀鍊介弮鍫濆窛妞ゆ挾濯寸槐鍙夌節閻㈤潧孝闁挎洏鍊濋獮濠呯疀濞戝磭绋忛柣蹇曞仩琚欓柡鈧禒瀣厽婵☆垵顕ф晶顖炴煕閻旈绠婚柡灞剧洴閹晠骞囨担鍦澒闂備浇妗ㄧ粈渚€宕幘顔兼槬闁跨喓濮村婵囥亜閹捐泛浠滃ù婊冪埣濮婄粯鎷呴挊澹捇鏌ㄥ顓滀簻闁挎棁顕ф禍鎵偓娈垮枤椤牓顢橀崗鐓庣窞閻庯綆鍓欓獮宥夋⒒娴ｅ憡鍟為柟绋挎瀹曘劑顢橀悪鈧崑褔姊婚崒娆掑厡缂侇噮鍨堕獮鎰偅閸愩劎鐛ラ梺鍝勭▉閸樺ジ鎷戦悢鍏肩叆婵犻潧妫欓崯鎺楁煟閺冨倸甯跺ù鑲╁█閺屾盯寮撮妸銉ョ睄闂佺粯绋掗幐鍓ф閹捐纾兼繛鍡樺姦濡倿鏌ｉ悩杈╁妽闁绘挴鈧剚鍤曢柡灞诲劜閸嬫劙鎮归崶顏勮敿闁硅姤娲栭埞鎴︽倷閺夋垹浠ч梺鎼炲姀椤宕曢锔界厽闁绘柨鎽滈惌濠偽旈悩铏€愰柛鈹垮灩椤撳吋寰勬繝鍌氱ギ闂備胶绮弻銊︽櫠鎼达絿涓嶆慨妯垮煐閳锋垿鏌涘☉姗堝姛闁瑰啿鍟妵鍕晜鐠囪尙浠梺閫涚┒閸旀垵鐣烽崼鏇ㄦ晢濞达絽鎼敮妤呮⒑閼姐倕鏋戦柣鐔村劤閳ь剚鍑归崜鐔煎箖濡皷鍋撻悽鐢点€婇柛瀣尵閹叉挳宕熼鍌ゆК闂備胶绮悧鏇㈠触鐎ｎ偆鈹嶅┑鐘叉搐缁犵懓霉閿濆牆鈧粙濡搁埡鍌滃帾婵犮垼娉涢悧鍡涘礈婵犳碍鐓忛柛顐ｇ箥濡插綊鏌嶉柨瀣诞闁哄本绋撴禒锕傚礈瑜庨崳顔碱渻閵堝繗顓虹紒鐘虫崌瀵鎮㈤崫鍕€抽梺鍛婎殘閸嬫﹢宕版繝鍥ㄥ€甸悷娆忓缁€鍐煕閺冣偓閻熲晠鎮伴鈧浠嬪Ω閿曗偓椤庢捇姊洪懡銈呮瀾濠㈢懓顑夊畷鎴﹀箻缂佹ê娈濋梺姹囧灮閺佹悂鎯侀崼銉︹拻闁稿本姘ㄦ晶娑氱磼鐎ｎ偅灏伴柡鍛版硾閳藉濮€閿涘嫬骞愰梻浣虹《閸撴繈鈥﹂崶顒佸剹闁圭儤顨嗛悡娑㈡倶閻愰潧浜剧紒鈧崘顏佸亾閸偅绶查悗姘緲閻ｇ兘宕￠悙宥囧枛閹筹繝濡堕崨顖溕戦梻鍌氬€搁崐鐑芥嚄閸洍鈧箓宕奸妷顔芥櫈闂佺鐬奸崑娑氱矆閸喓绡€闂傚牊绋撴晶娑氣偓瑙勬礀瀵墎鎹㈠☉銏犵婵炲棗绻掓禒鐓幬旈悩闈涗杭闁搞劎鍎ょ粚杈ㄧ節閸ヨ埖鏅┑顔斤供閸撴岸宕愰鐐粹拺闂傚牃鏅濈粔鐢告煕閿濆繒鍒版い顐㈢箻閹粌螣娓氼垰娈兼繝鐢靛仜濡瑩宕圭涵鍛彚濠电姷鏁搁崑娑㈡偤閵娧冨灊闁绘顕х粈鍫ユ煟閺冨洤浜圭€规挷绀侀…鍧楁嚋闂堟稑顫岀紓浣哄Х閹虫捇婀侀梺鎸庣箓閹冲繘骞夌粙璇炬棃鎮╅崣澶婃灎濠殿喖锕ュ钘夌暦濠婂牊鍤戞い鎺嗗亾閻㈩垬鍎靛娲传閵夈儛锝夋偨椤栨せ鍋撳畷鍥ㄦ闂佺鎻粻鎴犵不閼姐倗纾藉ù锝咁潠椤忓懐顩锋い鎾卞灪閳锋帒銆掑顒佹悙鐎殿喛娉曠槐鎺楁偐閸愯尙浠撮梺宕囩帛閹瑰洭銆侀弴銏℃櫆閻熸瑱绲剧€氬ジ姊绘担鍛婂暈缂佸鍨块弫鍐Ψ閿曗偓缁剁偤鏌涢弴銊ョ仭闁绘挻娲熼幃妤呮晲鎼粹€茬盎缂備焦顨嗙敮妤呭Φ閸曨垼鏁囬柣鎰綑閺嬬娀姊虹化鏇熸珨缂佺粯绻傞悾鐑藉Ω閳哄﹥鏅╅梺璇″瀻閸愵亞鏉介梻鍌氬€搁崐椋庣矆娴ｈ櫣绀婂┑鐘叉搐绾捐鈹戦悩鍙夋悙缂佲偓閸喐鍙忔俊顖涘绾箖鏌ｉ幘瀵哥疄闁哄矉绻濆畷鍫曞煛娴ｅ湱鈧櫣绱撴担绛嬪殭婵☆偅绻堝濠氭晬閸曨亝鍕冮柣鐘叉处瑜板啯鎱ㄥ澶嬧拺缁绢厼鎳庨惃娲煕鐎ｎ偅宕屾?failover 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒妯哄婵犫偓娓氣偓濮婅櫣绮欑捄銊ь唶闂佸憡鑹鹃鍥╂閻愬搫绠ｉ柣妯虹仛閿涘繘姊虹拠鈥崇€婚柛鏇ㄥ亗濞ｎ噣姊绘担鍛婃儓闁哄牜鍓熼幆鍕敍閻愰潧绁﹂梺鎼炲労閸擄箓寮繝鍥ㄧ參婵☆垯璀﹀Σ鎾煛鐎ｎ亞绠绘慨濠冩そ閹兘寮堕幐搴♀偓顖炴⒑鐠団€崇仩闁活厼鍊块獮鍐晸閻樻煡鍞堕梺闈涱槶閸庢煡鎮楅鍕厽閹兼惌鍨崇粔鐢告煕閻樻剚娈滈柟顕嗙節瀹曟﹢顢旈崨顓熺€炬繝鐢靛Т閿曘倝鎮ч崱娆戠焼闁割偁鍎查悡鐔兼煛閸屾氨浠㈤柟顔藉灴閺屾稓鈧綆鍊栭幋锕€桅闁告洦鍨伴崡铏繆閵堝倸浜炬繛瀛樼矋閸庡疇鐏冮梺缁橈耿濞佳勭濠婂嫮绠剧€瑰壊鍠栧顕€鏌曢崱鏇狀槮妞ゎ偅绮撻崺鈧い鎺戝缁犳牠鏌曡箛銉х？闂傚嫬瀚…璺ㄦ崉閻戞﹩妫炴繛瀛樼矒缁犳牠寮婚悢鐓庣鐟滃繒鏁☉銏＄厓闂佸灝顑呴悘鎾煛鐏炲墽鈽夐柍钘夘槸椤粓宕煎┑鍡╂浆缂傚倸鍊搁崐鎼佸磹閻熸壆鏆嗛柟闂寸閽冪喐绻涢幋鐐垫噭闁稿海鍠栭弻鏇熺箾閸喒鍋撻弽顓炵獥婵鍩栭崐鍨箾閸繄浠㈡繛鍛耿閺屾稓鈧綆浜烽弨璇睬庨崶褝韬€规洖銈稿鎾倷闂堟稈鍋撻悙鐑樷拺闁告劕寮堕幆鍫ユ煕閻樺磭銆掓俊鍙夊姍閹崇娀顢楅崒婊愮闯闂備胶顭堥張顒勬嚌妤ｅ啫鐒垫い鎺戝€搁崢瀛樻叏婵犲懏顥夋い鎾冲悑瀵板嫭绻濋崒婧惧亾椤撱垺鍋℃繝濠傛噹椤ｅジ鎮介婊冧粶闁伙絽鐏氱粭鐔煎焵椤掑嫬钃熼柣鏃傚帶缁犮儲銇勯弮鍌楁嫛闁哄棭鍙冨?SLA闂?
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
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯闁靛／灞芥櫏闂備浇顕у锕傦綖婢跺⊕鍝勵潨閳ь剟骞冮敓鐘虫櫢闁绘灏欓悾鍝勨攽鎺抽崐鏇㈠箠韫囨稑鐤鹃柡灞诲劚缁犲湱绱掗鐓庡辅闁稿鎹囬獮鍥ㄦ媴鐟欏嫮銈板┑鐘殿暜缁辨洟宕戦幋锕€纾归柕鍫濐槸绾惧鏌涘☉鍗炲Ц婵炲樊浜濋崑鍕煟閹捐櫕鍞夐柟鑺ユ礀閳规垿鎮欓弶鎴犱户闂佹悶鍔屽﹢杈╁垝缂佹绡€婵﹩鍘鹃崢鐐節闂堟稑鈧憡绔熸繝鍥х骇婵炲樊浜濋悡鏇㈡煟閹邦垰鐨虹紒澶屽劋閹便劍绻濋崨顕呬哗缂備緡鍠楅悷銉╁煝鎼淬劌绠氱憸宥嗙珶婢舵劖鈷掑ù锝呮啞鐠愶繝鏌熼搹顐ｅ碍閻撱倖銇勯幘鍗炵仼缂佺姵鐗曢湁闁稿繐鍚嬮崕妤呮煛娴ｅ壊鍎旈柡灞界Х椤т線鏌涢幘瀵告噰闁诡噣绠栭幃鍧楊敍濡鐫忛梻浣告贡閸庛倗鎹㈤崘顔肩柧闁冲搫鎳忛埛鎴︽煕濞戞﹫鍔熼柟鏂ュ亾闂備胶顭堟鎼佸疮閼愁垳浜辨俊鐐€栭悧婊堝磻閻愬搫鐓曢柟鐑橆殕閻撴洟鎮橀悙鎻掆挃闁瑰啿妫濋弻娑滅疀閹惧墎浼囬梺姹囧労娴滎亪銆佸鈧幃娆撴濞戞艾骞楅梺璇插椤旀牠宕板Δ鍕╀汗闁告劦鍠掗埀顒婄畵婵℃悂鍩℃担鍝ョ崺婵＄偑鍊栭悧妤冨垝鎼淬劌鍌ㄦい鏍仦閳锋帒霉閿濆洨鎽傛繛鍏煎姉閳ь剝顫夊ú姗€鏁嬮梺宕囩帛閺屻劑鍩ユ径鎰潊闁绘灏欓埀顒夊幖閳规垿鎮╃紒妯婚敪濡炪倖鍨靛Λ婵嬪箖閿熺姴绀冩い鏃傛櫕閸樺崬鈹戦悩缁樻锭婵☆偅顨婇、鏃堫敂閸曘劍鏂€濡炪倖鐗楅崫搴ㄥ磻閵忋倖鐓涢悘鐐殿焾婢ф煡鏌熷畡鐗堝殗鐎规洦鍋婃俊鐑藉箛閸撲焦鍋х紓浣介哺鐢偟妲愰幒鎳崇喖鎳栭埡鍐╂緰闂佽姘﹂～澶娒洪弽顐ょ濠电姴娲㈤埀顑跨窔瀵挳濮€閻欌偓濞煎﹪姊虹紒妯兼喛闁稿鎸荤换娑㈠礂閸忕厧闉嶇紓浣虹帛缁诲牓骞冩禒瀣棃婵炴垶顨堥幑鏇㈡⒒娴ｈ櫣甯涙い銊ユ嚇閺佸啴濡搁埡浣勶箓鏌熼悧鍫熺凡缂佲偓閸愨斂浜滈柡鍐ㄦ搐娴滃綊鏌涘Ο缁樺唉婵﹨娅ｅ☉鐢稿川椤斿吋閿梺璇查叄濞佳囨儗閸屾凹鍤曟い鎰剁悼缁♀偓闂佹悶鍎崝搴ㄥ储娴犲鈷戦梺顐ｇ☉瀹撳棙绻涙担鍐插濞呯姵銇勯弮鍌涙珪缂佺娀绠栭弻娑㈩敃閿濆洨鐣奸梺缁樺笒閹诧繝骞堥妸锔剧瘈闁告侗鍣禒鈺冪磽娓氬洤鏋熼柣鐔叉櫅閻ｇ兘鎮╃拠鑼姶闂佸憡鍔忛弲婊冃掓惔銊︹拻闁稿本鐟︾粊鐗堛亜閺囩喓澧电€规洑鍗冲浠嬵敇閻愮绱梺鑽ゅ枑閻熴儳鈧凹鍣ｉ崺鐐差吋閸涱亝鏂€闂佺粯锚瀵埖寰勯崟顖涚厱閻庯絻鍔屾俊璺ㄧ磼鏉堛劌绗ч柍褜鍓ㄧ紞鍡涘储閻ｅ本鍏滈柛顐ｆ礃閻撶喐銇勯幇鈺佺仼妞ゅ浚鍘介幈銊︾節閸愨斂浠㈤悗瑙勬礃缁捇鐛幘璇茬鐎广儱娲﹂弲濂告⒒閸屾瑧顦﹂柟鑺ョ矋閹便劑鎮界粙璺槷闂佸搫娲㈤崹褰掓偂濠靛牃鍋撻獮鍨姎妞わ富鍨跺畷姗€鍩€椤掆偓椤啴濡堕崱妯烘殫闂佸摜鍠庡锟犵嵁韫囨拋娲敂閸涱亝瀚奸梻浣告啞缁嬫垿鏁冮妷褌鐒婇柟娈垮枟閸犳劙鏌℃径濠勪虎闁哄棛鍋熺槐鎺楀磼濮樻瘷銏ゆ煃鐟欏嫬鐏寸€规洖銈告慨鈧い顐枤鎼村﹪姊绘担钘夊惞濠殿喗娼欑叅闁挎洑闄嶆禍褰掓煕閹伴潧娅橀柡浣稿€归妵鍕箻鐠虹洅娑㈡煕鐎ｎ偅灏柍缁樻崌瀹曞綊顢欓悾灞兼喚婵犵數濮烽弫鎼佸磻濞戙垺鏅濋柕鍫濐樈閺佸鏌ㄥ┑鍡╂▓闁轰礁顑夐弻宥堫檨闁告挻鐟╅幃楣冩煥鐎ｎ剟妾梺鍛婄☉閿曘倖绂嶅鍫熲拺闁告稑锕︾粻鎾绘倵濮樼厧澧扮紒顔碱煼閹瑩鎮滃Ο閿嬪闂備胶顭堥張顒勫礄瑜版帗鍋傛い鎺戝閻撴洟鎮楅敐搴濈盎妞ゅ繆鏅犻幃锟犲Χ婢跺鍘梺鍓插亝缁诲秴螣閸儲鐓涘ù锝囶焾椤忣參鏌＄仦鍓р槈闁宠棄顦靛畷锟犳倷鐎甸晲绨界紓鍌氬€峰ù鍥敋瑜嶈灋婵犻潧顑呴拑鐔兼煟閺冨洦顏犻柣顓熺懇閺岀喓绮欓崹顕呭妷閻庤娲栫壕顓犳閹惧瓨濯撮柛婵嗗閺嬪懐绱撴担鍝勑ｉ柣妤冨Т椤曪絿鎷犲ù瀣潔濠碘槅鍨堕弨閬嶏綖瀹ュ應鏀介柍钘夋閻忥絿绱掗鍛仸妤犵偛鍟埢搴ょ疀婵犲啯鏉搁梻浣虹帛閸旀浜稿▎蹇ヨ€块柟缁㈠枟閻撴洖鈹戦悩鎻掓灓婵炲牊绮嶉〃銉╂倷瀹割喗鈻堝Δ鐘靛仦閿曘垽銆佸▎鎾村殟闁靛濡囪ぐ鍡涙⒒閸屾艾鈧兘鎮為敃鍌ゆ晞婵炲棙鎸搁崹鍌炴煟濡も偓閻楀繘銆呴弻銉︾厽闁逛即娼ф晶缁樼箾閸粎鐭欓柡宀嬬秮楠炲洭顢涘杈嚄濠电偛顕慨浼村垂娴犲钃熸繛鎴欏灩缁秹鏌嶈閸撴瑩鎮惧┑瀣濞达絾鐡曢幗鏇㈡⒑缂佹ɑ鈷掗柛妯犲懐鐭嗛柛鎰靛枟閻撴洘绻涢崱妤佺婵℃彃婀遍埀顒冾潐濞叉牜澹曢鐘愁潟闁规儳顕悷褰掓煕閵夋垵瀚禍顏堟⒒娴ｈ櫣甯涙い銊ユ噽閹广垹顫滈埀顒€鐣峰ú顏勭劦妞ゆ帊闄嶆禍婊堟煙閻戞ê鐓€闁告梻鍠撶槐鎺撳緞鐏炵偓姣堥梺鍝勬湰閻╊垶骞冮埡鍛疀濞达絽婀遍弶褰掓⒒娴ｇ瓔鍤冮柛锝庡灣閹广垹鈹戦崶锔剧畾闂佺粯鍨兼慨銈夊疾濠婂牊鐓曢柟鐐殔閼活垶寮查銏♀拻闁稿本鐟ㄩ崗宀勬煙閾忣偅宕屾い銏″哺椤㈡﹢濮€閻樼绱甸梻浣烘嚀椤曨參宕戦悢鍏煎仭閻熸瑥瀚粻楣冩煙鐎电浠ч柟鍏呰兌缁辨帞鎷犻崣澶樻＆闂佸搫鐬奸崰鎾舵閹烘嚦鐔兼惞鐠団€冲壃闂傚倷鑳剁涵鍫曞疾椤忓棛绀婂〒姘ｅ亾鐎?
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍蓟閸ヮ剚鏅濋柍褜鍓熷畷婵嬪箳閹惧墎顔曢悗鐟板閸犳洜鑺辨繝姘厱闁硅埇鍔屾禍鎯р攽閿涘嫬浜奸柛濞垮€濆畷婊堟晝閸屾氨顦┑顔筋焾娴滎剟鎯屽▎鎾寸厵閺夊牆澧介悾閬嶆煟閹绢垪鍋撻幇浣哄數闁荤姴鎼妶浠嬫晸閻樺啿浜滈梺鐓庮潟閸婃寮堕幖浣光拺闁告繂瀚婵嬫煕鐎ｎ偆鈯曞ǎ鍥э躬閹煎綊宕烽鐙呯闯濠电偠鎻紞鈧繛鍜冪悼閺侇喖鈽夊杈╋紲闁荤姴娲╃亸娆愭櫠閺囥垺鐓熼柨婵嗘搐閸樺瓨顨ラ悙鍙夘棦闁哄苯娲弫鍌炲矗椤愵偄澧ǎ鍥э躬閹瑩顢旈崘顏嗩槬濠电偠鎻徊浠嬪箟閳╁啫绶為柛鏇ㄥ灡閻撳啴姊洪崹顕呭剰闁诲繐鐡ㄩ幈銊︾節閸屻倗鍚嬮悗瑙勬礀閵堟悂骞冮姀鐘垫殝闁绘鐗忔禍鐐电磽閸屾艾鈧兘鎮為敃鍌氳埞闁煎鍊栧畷鍙夌節闂堟侗鍎忛柦鍐枑缁绘盯骞嬮悙鍐╄壘鍗遍柛顐犲劜閻撴洘淇婇妶鍛殭濞寸姵绮庣槐鎾存媴閸欏鐝斿銈庝簻閸熷瓨淇婇崼鏇炲耿婵☆垳鍘у铏繆閻愵亜鈧呯不閹炬剚鐒界憸鏃堟晲閻愬墎鐤€闁哄啫鍊婚惁鍫濃攽椤旀枻渚涢柛鎾寸洴閺佸秴顭ㄩ崘锝嗘杸闂佺粯鍔曞鍫曀夐悙鐑樼厱闁靛鍎虫禒娑樓庨崶褝韬柟绋匡攻瀵板嫰宕卞Ο缁樻緫闂傚倷鑳剁划顖炲礉閺嶎兙浜归柛鎰靛枛缁犳牠鏌熸潏楣冩闁绘挾鍠栭弻锝夊棘閸喚楠囧┑鐐叉噹閿曘儵銆冮妷鈺傚€烽柟缁樺笚濞堫參鏌ｉ幘鍗炩偓鏍Φ閸曨喚鐤€闁规儳纾鎼佹⒑閸愬弶鎯堥柟鍐茬箻閹ょ疀閹垮啰鍞甸梺鑲╊焾缁堕亶顢旈崼婵嗗殤闂侀潧鐗嗛ˇ浼存偂閺囥垺鐓涢柛鎰╁妼閳ь剚鎮傚畷銏⑩偓娑櫳戦崣蹇撯攽閻樻彃鏆為柕鍥ㄧ箘閳ь剝顫夊ú妯煎垝閹捐崵宓侀柟鎹愵嚙缁犲磭鈧箍鍎卞ú銈夊吹椤掑倻纾介柛灞捐壘閳ь剛鍏橀幃鐐烘晝閳ь剟鈥旈崘顔藉仼鐎光偓閳ь剚绋夊澶嬬厸鐎规搩鍠栭懟顖氣枔閵娾晜鈷戠憸鐗堝笚閿涚喓绱掗埀顒佹媴閾忕懓鏆楅悗骞垮劚椤︿即鎮￠弴鐔翠簻闁规壋鏅涢埀顒佹礋瀵憡鎯旈妸锔惧帗閻熸粍绮撳畷婊冣槈閵忕姵鐎繝鐢靛У閼归箖鎷戦悢鍏肩厪濠电偛鐏濋崝妤呮煛閳ь剚绂掔€ｎ偆鍘遍梺鏂ユ櫅閸熲晝娆㈤柆宥嗙厓鐟滄粓宕滃韬测偓鍐川缁厜鍋撻敃鍌涘殑妞ゆ牭绲鹃鍥⒒娴ｈ鍋犻柛鏂跨箲缁傚秹顢楅崟顐ゎ唹闂侀潧绻掓慨顓炍ｉ崼銉︾厪闊洦娲栭埢鍫ユ煕閻愯埖纭鹃柍瑙勫灴閹晝绱掑Ο濠氭暘闂備胶绮〃鍛崲濡櫣鏆︽繝闈涱儛閺佸秹鏌ｉ幇顔克夐柟閿嬫そ濮婃椽宕ㄦ繝鍕暤闁诲孩鍑归崹鍫曟晲閻愮儤鏅濋柛灞剧〒閸橀亶姊洪崫鍕偓钘夆枖閺囥垺鍊块柟闂寸劍閻撴洟鏌嶉悷鎵虎闁告梹绮庨埀顒€鐏氬妯尖偓姘煎幘閹广垹鈽夐姀鐙€娼婇梺闈涚箳婵敻鎮橀崼銉﹀€甸悷娆忓缁€鈧紓鍌氱Т閿曨亪濡存担鍓叉僵閻犲搫鎼粣娑橆渻閵堝棗绗掗柛濠傤煼閺佸秹鎮欓悜妯锋嫽闂佺鏈懝楣冨焵椤掍焦鍊愮€规洘鍔曢悾锟犲箠婵犲倻绉虹€规洖鐖兼俊姝岊槾鐟滀即绠栧娲礈閹绘帊绨肩紓浣筋嚙鐎氭澘顕ｉ幎鑺ユ櫇闁稿本绋撻崢浠嬫⒑闂堟稓绠為柛鈺佸閹偤宕滆绾惧ジ鏌熼柇锕€寮炬繛鍫熺矋椤ㄣ儵鎮欑€电鈪归柤鎸庡姈閵囧嫰骞掗崱妞惧濠电姷顣介埀顒傚仺閸嬨垽鏌″畝瀣М妤犵偞顭囬埀顒勬涧閹芥粓鎮块崟顖涒拺闁告繂瀚悞璺ㄧ磼缂佹绠撻柣锝呭槻椤粓鍩€椤掍椒绻嗘慨婵嗙焾濡茶顪冮妶蹇曠ɑ婵＄偘绮欏濠氭偄绾拌鲸鏅┑鐘诧工閸燁垶鎮橀崼婵冩斀妞ゆ梻銆嬮弨缁樹繆閻愯埖顥夐摶鐐烘煕閹扳晛濡锋俊鎻掔墦瀵爼宕煎☉妯侯棊闂侀潧鐗嗛ˇ浼村煕閹寸姷纾奸悗锝庝簼閸嬨儲銇勬惔銏╂畼闁逞屽墲椤煤閿曞倸绀堟慨妯挎硾閻ら箖鏌曟径鍡樻珕闁抽攱鍨块弻娑樷槈濮楀牆濮涢梺鐟板暱閸燁垶濡甸崟顖涙櫇濞达絽鍢查幆鍫㈢磽娓氬洤鏋ょ紒顕呭灦婵″爼鏁愭径濠勵槰闂佸啿鎼崯顐︾嵁瀹ュ鈷戦悹鍥у级閹癸綁鏌℃担瑙勫€愰柍銉畵瀹曞ジ鎮㈡笟顖涚カ闂備焦瀵уú鏍磹瑜版帒纾归柛顐ｆ礈閸欐捇鏌涢妷鈺婃闁告帗澹嗛埀顒侇問閸ｎ噣宕抽敐澶婅摕闁挎稑瀚▽顏堟偣閸ャ劌绲绘い顐㈡喘閹鈻撻崹顔界亶濠电偛鍚嬮悷銊╂倶閹烘鈷戦柛娑橈功缁犳捇鎮楀鐓庡⒉妞ゃ倕鍊垮濠氬磼濮橆兘鍋撳畡鎳婂綊宕堕澶嬫櫔闂佸搫绋侀崢鑲╃玻濡ゅ懏鐓欓柟瑙勫姦閸ゆ瑧鐥幆褎鍋ユ慨濠傤煼瀹曞ジ鎮㈤幁鎺嗗亾閹烘鐓涢柛鈽嗗弮濡绢喚绱掔紒妯肩畺缂佺粯绻堝畷鎺戭潩闂傚妫繝鐢靛У椤旀牠宕归柆宥呯闁规儼妫勯拑鐔兼煥濠靛棭妲归柛瀣閺岋綁骞樺畷鍥у毈闂佸摜濮撮敃顏勵潖濞差亝鐒婚柣鎰蔼鐎氭澘顭胯閸ㄥ爼寮婚敐澶婄厸濠电姴鍊绘禒鈺侇渻閵堝骸浜滅紒缁樺姉閸欏懎顪冮妶鍛闁瑰嘲顑呰灋闁告劑鍔夐弨浠嬫煟閹邦厽缍戦柣蹇ョ畵閺屻倛銇愰幒鏃傛毇闂侀潧妫楅崐鍝ュ弲濡炪倕绻愰幊澶愬箯閾忓湱纾介柛灞剧懅閸斿秹鏌ㄩ弴妤佹珔闁崇粯鎹囧顕€宕掑鍜冪床缂傚倸鍊烽悞锕傛晪婵犳鍠栧ú銊╁焵?
	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Del("x-goog-api-key")
	req.Header.Set("authorization", "Bearer "+token)

	// OAuth 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯闁靛／灞芥櫏闂備浇顕у锕傦綖婢跺⊕鍝勵潨閳ь剟骞冮敓鐘虫櫢闁绘灏欓悾鍝勨攽鎺抽崐鏇㈠箠韫囨稑鐤鹃柡灞诲劚缁犲湱绱掗鐓庡辅闁稿鎹囬獮鍥ㄦ媴鐟欏嫮銈板┑鐘殿暜缁辨洟宕戦幋锕€纾归柕鍫濐槸绾惧鏌涘☉鍗炲Ц婵炲樊浜濋崑鍕煟閹捐櫕鍞夐柟鑺ユ礀閳规垿鎮欓弶鎴犱户闂佺硶鏅涚€氭澘顕ｉ锕€绀冩い鏃傛櫕閸欏棗鈹戦悩缁樻锭婵☆偅鐟╄棢闁绘鍋ㄦ禍婊堟煥閺傛寧鎯堥柛鏂诲€楃槐鎺撴綇閵婏箑闉嶉梺鐟板槻閹虫﹢鐛幘璇茬鐎广儱鎷嬪Λ?ChatGPT internal API 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鎾煙椤斿吋鍋ユい銏＄懄閹便劑骞囬鍡欐晨闂傚倷绀侀幖顐ょ矙娓氣偓瀹曟垿宕熼鍌ゆ祫濠电姴锕ら幊鎰涢鐐寸厵妞ゆ牕妫楃€氼剚绂掗幘顔解拺闁告繂瀚～锕傛煕婵犲啰绠撻柣锝囧厴閹粎绮电€ｎ偅娅嶉梻浣虹帛閸ㄩ潧螞濞嗗警鎺撳緞閹邦厸鎷绘繛杈剧悼閹虫捇顢氬鍛＜閻庯綆鍋勯悘瀵糕偓瑙勬礃閸旀瑩鐛弽銊﹀闁告縿鍎遍獮瀣⒒娴ｄ警鏀伴柟娲讳簽閳ь剟娼ч惌鍌氼嚕椤愶絿绡€闁搞儯鍔庨崣鍐ㄢ攽閳藉棗鐏熼悹鈧敃鈧嵄闁绘垶菧娴滄粓鏌熺€涙绠撻柍褜鍓濆畷闈浳ｉ幇鏉跨閻庨潧鎽滈幊婵嬫⒑閹肩偛鍔€闁告洦鍘奸拏瀣⒒閸屾瑧鍔嶉柟顔肩埣瀹曟繆绠涢幘顖涚亙濠电偞鍨崹娲磻閳哄啠鍋撻悷鏉款仾濠㈢懓顑夊銊︾鐎ｎ偄鈧敻鏌ㄥ┑鍡欏嚬缂併劏宕甸埀顒冾潐閹哥螞濠靛棭娼栭柧蹇撴贡绾惧吋淇婇姘濞存粠浜滈悾鐑筋敍閻愯尙顔囬柟鍏肩暘閸ㄥ綊宕濋崨顔剧瘈闁汇垽娼у瓭濠电偠顕滅粻鎾诲春濞戙垹绠ｉ柨鏃囆掗幏鍝勵渻閵堝棗濮傞柛濠冩礃缁傛帒顭ㄩ崼鐔哄幘濠电偠灏褔鎮橀敓鐘崇厸闁告侗鍘鹃崺锝嗩殽閻愯揪鑰挎い銏＄懇閹墽浠﹂挊澶岊吋婵犵绱曢崑鎴﹀磹閺嵮屾綎鐟滅増甯掔壕鍧楁煙閹増顥夐柛灞诲姂閹鎮介悽鐐光偓濠囨煕鐎ｎ偅灏柍缁樻崌瀹曞綊顢欓悾灞兼喚濠电姷鏁搁崑娑㈠触鐎ｎ喗鍋￠柍鍝勬噺閺呮煡鏌ｉ幇顓犮偞闁衡偓娴犲鐓曢柕澶堝妼閻撴劗绱撳鍡楃伌婵﹥妞藉畷顐﹀礋椤撳濞€閺屾盯骞橀崘鑼跺煘濡炪値鍘归崝鎴濈暦婵傚憡鍋勯柛娆忣槸椤忕儤绻濋悽闈涗沪闁圭顭峰畷娲礃椤旇偐锛涢梺瑙勫劤椤曨厾绮婚幆顬″綊鏁愰崨顓熸瘣闂佸憡蓱閸庡磭鍙呭銈呯箰閹冲繐鈻嶉崶顒佲拺闂侇偆鍋涢懟顖涙櫠閺屻儲鐓熼煫鍥ㄦ惄閸庢梹鎱ㄦ繝鍐┿仢鐎规洏鍔嶇换婵嬪磼濮樺吋缍傞梻鍌欑閹碱偊骞婅箛娑欏仭鐟滄棃鐛?
	if account.Type == AccountTypeOAuth {
		promptCacheKey := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
		req.Host = "chatgpt.com"
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
		apiKeyID := getAPIKeyIDFromContext(c)
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆洨鎽傛繛鍏煎姇椤潡鎮烽悧鍫！闂佸搫鎳撳▔娑滅亙闂佸憡渚楅崢楣冩晬濞戙垺鐓熼幖鎼灣缁夌敻鏌涢悩鎰佹疁闁诡噯绻濆鎾偄閾忚鍟庨梻浣烘嚀閻°劎鎹㈤崒鐐村殝鐟滅増甯楅悡鐔肩叓閸ャ儱鍔ょ痪鎯ф健閺屽秷顧侀柛鎾寸箞閿濈偞寰勬繛鎺撴そ瀵€燁檨闁搞倖娲熼弻娑㈩敃閿濆棛顦ョ紓浣哄Х閸嬨倝寮婚弴鐔虹闁割煈鍠栨竟鏃堟⒑缂佹ɑ鈷掗柛搴涘€楅埀顒佺濮樸劑鍩€椤掑倹鍤€濠㈢懓锕畷鏉课旈埀顒勫煝閹捐秮娲敂閸涱亝瀚藉┑鐐舵彧缁茶偐鎷冮敃鍌涘€块柣鎰靛厵娴滄粓鏌熺€涙绠栨い銉ｅ灪椤ㄣ儵鎮欓弶鎴濐潚濡ょ姷鍋為悧妤呭箯閸涙潙浼犻柕澶堝劚缂佲晜绻濋悽闈浶為柛銊у帶閳绘柨鈽夊鍐炬澓闂傚倷绀侀幖顐︻敄閸涱垳鐭撻柣銏㈩焾妗呴梺鍛婃处閸ㄦ壆绮荤紒妯圭箚闁靛牆鍊告禍楣冩⒑缁嬪尅宸ユ繛灏栤偓宕囨殾闁瑰瓨鎯婇弮鍫濈劦妞ゆ帒瀚ч埀顒佹瀹曟﹢顢欓崲澹洦鐓曢柟鎵虫櫅婵″灝霉閻樺啿鍔ゆい顏勫暣婵¤埖鎯旈垾宕囶啇闂備胶绮〃鍡涘箰缂佹鈹嶅┑鐘叉处閸嬨劎绱掔€ｎ厽纭堕柣銈傚亾闂傚倷绀侀幖顐λ囬崘娴嬫灃闁哄洢鍩勯弫濠傗攽閻樺弶澶勯柍閿嬪灴閺岋綁鎮㈤悡搴濆枈闂佹悶鍊楅崰搴ㄥ煘閹达富鏁嶉柨婵嗘噹缁秹鎮楃憴鍕８闁稿酣娼ч悾鐑藉Ω閳哄﹥鏅ｅ┑鐐村灦閻燂箓寮堕幖浣光拻濞达絿鎳撻婊呯磼鐎ｎ偄鐏╅柍褜鍓氶崙褰掑礈濮樿京鐭夐柟鐑樺焾閻撱儵鏌涢弴妤佹珒缂併劌顭峰娲偡閹殿喗鎲肩紓浣筋嚙閸熸潙鐣烽搹顐ゎ浄閻庯綆鍋嗛崢鎾绘偡濠婂嫮鐭掔€规洘绮岄埢搴ㄥ箻閺夋垟鍋撻懜鍨弿婵☆垰娼￠崫娲煛閸℃鐭嬪ǎ鍥э躬婵″爼宕ㄩ鐔哥杹闂備胶顭堥鍡涘箲閸ヮ灛娑㈠礃閵娿垺顫嶉梺鍝勮癁閸屾艾鐏婇梻浣筋嚙濮橈箓锝炴径濞掑搫螣閸忓吋娈伴梺璺ㄥ枔婵绮婚弽銊х闁糕剝锚閻忓秹鏌涚€ｎ偅宕岄柟顔挎閳绘挾鎹勯妸銉バ┑掳鍊楁慨鐑藉磻濞戙垺鍋嬪┑鐘插瀹曞弶绻濋棃娑氬闁哄鐗婃穱濠囶敍濞戞瑣浠у┑鐐插悑閸ㄥ灝顫忓ú顏勭闁绘劖褰冮‖澶嬬節閻㈤潧浠滈柨鏇樺€濋幃楣冩倻閽樺娼婇梺鎸庣箓鐎涒晠鍩㈠畝鍕厽閹兼番鍩勯崯蹇涙煕閻樺磭澧垫鐐差樀閺佹捇鎮╅懠顒傛毎闂備礁鎼崯顐︽偋閻愮數绀婇柡宥冨妸娴滄粓鏌熸潏鍓хɑ缁绢厼鐖奸弻锝夊煛娴ｅ壊鍔夋繛锝呮搐閿曨亪骞冨▎鎿冩晜闁告洏鍔屾禍楣冩煛瀹ュ骸骞栫紒鈧径灞惧枑闊洦鎼╅崵妤€鈹戦悩鍙夊闁搞倕绉归弻鏇熷緞濞戞氨鏆犻悗娈垮枛濞硷繝骞冨Δ鍛祦闁割煈鍠栨慨搴♀攽閻愯泛鐨哄┑鐐╁亾閻庤娲橀崹鍧楃嵁濮椻偓閹虫粓妫冨☉娆戔偓顓㈡⒒娴ｅ憡鍟炴繛璇х畵瀹曟粌鈽夐埗鍝勬喘婵℃悂鍩￠崒妤佸闂備礁鎲＄粙鎴︽晝閵夆晛鐓曢柟閭﹀幑娴滄粓鏌曟繛鍨姕闁搞倕娲弻娑㈠煘閸喖濮曢悗鍨緲鐎氫即鐛崶顒夋晢闁逞屽墴閹顢橀悜鍡樺瘜闂侀潧鐗嗗Λ妤佹叏閿曞倹鐓曢悗锝庝簼閸ｆ椽鏌￠崨顐㈠姦婵﹦绮幏鍛瑹椤栨稒鏆炲┑鐘灮閹虫捇鎮ч幘鑽ゅ祦闁圭偓妞块弫鍌炴煕閳╁啰鎳呮い锔诲灦濮婅櫣鍖栭弴鐐测拤闂侀潧娲ら崐鍧楀极閹邦厼绶為悗锝庝簴閸嬫挻绻濆顓犲幘闂佽鍘界敮鎺楀礉濡ゅ懏鐓欏瀣捣鐢稓绱掔紒妯尖姇婵炵厧绻樺畷婊嗩槻闁糕晛鐭傚铏规兜閸滀礁娈濈紓浣虹帛缁诲倿鎮鹃悜绛嬫晬闁绘劖娼欓埀顒傚厴閺岋綁骞嬮悜鍡欏姺闂佸憡锕㈡禍璺侯潖濞差亜浼犻柛鏇ㄥ亝濞堟粓姊虹粙娆惧剱闁规瓕宕甸幑銏犫槈閵忕姷鐓戞繝銏ｆ硾閻ジ鎮￠崒鐐粹拺闁告稑锕ょ粭姘舵偨椤栨せ鍋撻幇浣告濡炪倖娲嶉崑鎾垛偓瑙勬礃鐢帟鐏冮梺閫炲苯澧紒鍌氱Ч閹瑩顢楅崒婊庡晭闂備胶鎳撻顓㈠磻閻旂厧绠犻柟鎵閻撶喖鏌熼悜妯荤鐞氭岸姊?compact 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呮／闁诡垎宀€鍚嬮梺鍝勭焿缂嶄線鐛崶顒夋晩闁兼亽鍎查惁搴ㄦ⒒娴ｈ銇熼柛妯圭矙閹兘鍩￠崨顔间簵濡炪倖鍔х粻鎴︽倷婵犲洦鐓忓┑鐐戔偓閸嬫挻淇婇幓鎺斿ⅱ缂佽鲸甯￠幃娆擃敆閳ь剟宕濋妶鍡欑閻忓繑鐗戦崑鎾诲箛娴ｅ湱绋佹繝鐢靛仜濡﹥绂嶉崼鏇炴瀬闁糕剝绋掗悡鍐喐濠婂牆绀堟繛鎴炶壘閸ㄦ繈鏌￠崘銊у缂佺姷绮妵鍕籍閸パ傛睏闂佹寧绋掔划鎾愁潖濞差亝顥堟繛娣劚閻楁挸顕ｉ幓鎺濈叆闁割偆鍠庢禍妤€鈹戦悙鏉戠仸闁糕晛鍟村畷鎴﹀箻閼姐倕绁﹂梺鍓茬厛閸犳牗鎱ㄦ惔顫箚闁靛牆娲ゆ牎闂佽鍠栭崐鍧楀箖娴兼惌鏁婇柛銏狀槹濡啫鐣烽妸鈺婃晩闁诡垎鍐唶闂傚倸鍊风粈渚€骞夐敍鍕灊闁割偁鍨婚惌鍡椕归敐鍫綈婵炲懐濮撮湁闁绘挸娴烽幗鐘绘煟閹惧啿鏆熼柟鑼焾椤劑宕煎┑鍫Ф婵犵數鍋涘Λ娆撳箰閹间礁纾奸柕濠忓缁♀偓婵犵數濮撮崐缁樻櫠閺囩姷妫柟顖嗗瞼鍚嬮梺鍝勭灱閸犳牕鐣峰鍡╂Ь闁汇埄鍨遍惄顖炲蓟閿濆绠婚悗娑欘焽椤︿即姊洪崫鍕効缂佺粯绻傞悾鐑藉醇閺囩倣銊╂煏婢跺牆鍔撮柣搴や含缁辨捇宕掑▎鎾搭€栭梺鍛婃煥闁帮絽顕ｉ弻銉晝闁靛牆妫鐔兼⒑閸︻厼鍔嬮柟姝屽吹缁辩偤宕堕浣哄帾婵犵數鍋涢悘婵嬪礉濮樿埖鐓欏ù鐘差儐濞懷囨煃鐟欏嫬鐏存い銏＄懅濞戠敻鎮滈悾灞藉冀濠电姷鏁搁崑娑㈠箯閹寸姴绶ら柛褎顨呯粈鍡涙煙閻戞ê鐏嶉柡瀣叄閺岀喓鈧數顭堟禒婊呯棯缂併垹鐏︽慨濠冩そ濡啫霉閵堝棛銆掑ù鐙呯畵閹崇偤濡烽敂鐣屾缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾剧懓顪冪€ｎ亝鎹ｉ柣顓炴閵嗘帒顫濋敐鍛闂佽姤蓱缁诲倿婀侀梺绋跨箰閸氬绱為幋鐘电＜闁绘ê宕畵鍡涙煛瀹€鈧崰鏍х暦濠婂嫭濯撮悷娆忓瑜把囨煟鎼淬値娼愰柟鎼侇棑濞嗐垹顫濈捄楦挎憰濠电偞鍨崹鍦不濞戙垺鐓冮弶鐐村椤︼附銇勯妷銉剶婵﹥妞藉畷顐﹀礋椤愶絾顔勯梻浣虹帛椤ㄥ懎螞濞戞艾鍨濋柛顐犲劚缁€鍐煃閻熻埇浠掔紒銊ヮ煼濡懘顢曢姀鈥愁槱闂佸湱顭堥…鐑藉极瀹ュ绫嶉柛灞剧矌閿涙粓姊洪崨濠庢畼闁稿绋撻惀顏囶槻閼挎劙鏌涢妷鎴濈Х閸氼偊姊虹拠鈥虫灍闁荤啿鏅犻妴渚€寮崼婵堫槹闂侀潧顭堥崕鐗堢珶閺囥垺鈷戦柛婵嗗閳诲鏌涘Ο鍨汗缂侇喚绮€佃偐鈧稒顭囬崢浠嬫⒑闁稑宓嗘繛浣冲嫭娅犳い鏍仦閻撴洘绻涢崱妤冪缂佺姴顭烽弻鈥崇暆鐎ｎ剛袦濡ょ姷鍋涢澶愬极閹版澘骞㈤柍鍝勫亞濡嫰姊婚崒娆戭槮闁圭⒈鍋嗛埀顒佸搸閸旀垵鐣烽弴銏″仺缂佸鍎婚幗鏇㈡⒑閹稿海绠撻柟鍙夛耿閹垽宕楅悡搴㈩吋闂備線娼ч悧鍡涘疮椤愶絿顩烽柨鏃傛櫕缁♀偓缂佸墽澧楅敋濠⒀嗗皺閹叉悂寮堕崹顔芥闂佽鍟崶褏顔掗柣鐘叉穿鐏忣亪骞楅弴銏♀拺缂備焦蓱椤ュ棙绻涢崪鍐偧濠㈣娲樼缓浠嬪川婵犲嫬甯鹃梻濠庡亜濞诧箑煤閺嵮勬瘎婵犵數鍋涢顓熸叏椤撱垹鐤炬繝濠傛噺瀹曞弶绻涢幋鐐茬劰闁稿鎸搁埥澶娾枎濡厧濮洪梻浣规た閸樼晫鏁悙鍨潟闁圭儤顨嗛崑鎰偓瑙勬礀濞层倝鍩呴悷閭︽富闁靛牆楠告禍婊呯磼缂佹ê鐏╅柟骞垮灩閳规垿宕遍埡鍌氬厞婵＄偑鍊栫敮鎺楀磻婵犲嫭顫曢柛娆忣槺缁♀偓濠电偛鐗嗛悘婵嬪几閻斿吋鐓ラ柡鍥殕濞呭﹦鈧娲忛崹浠嬪箖閳╁啯鍎熸俊顖濆吹閳ь剚妞藉娲濞戞艾顣洪梺鐟板暱椤﹂潧顕ｉ妸鈺傚殟闁靛绲肩花濠氭⒑閸濆嫬鏆欓柛濠傜埣閸┾偓妞ゆ巻鍋撴い鎴濇嚇椤㈡瑨绠涘☉妯溿劑鏌ㄩ弮鍥ㄣ€冪紒銊嚙椤啴濡堕崱妯烘殫婵犳鍠楅幐鍐茬暦閵忋倕绠绘い鏃傛櫕閸橀亶姊洪棃娴ュ牓寮插┑瀣仼闁惧繐鍘滈崑鎾斥枔閸喗鐏嗙紓渚囧枟閻熲晠濡存担鍓叉建闁逞屽墴楠炲啫鈻庨幘鎼濠电偞鍨剁敮鎺撶妤ｅ啯鐓熼柟閭﹀墯閳绘洘绻涢幘鎰佺吋闁哄本娲熷畷鐓庘攽閸ヨ埖顥ｉ梻浣侯潒閸愯儻鍚梺鍝勬湰缁嬫捇鍩€椤掑﹦绉甸柛瀣噽娴滄悂鎮介悽鐢碉紲闁哄鐗勯崝灞矫归濮愪簻闁靛繆鍓濈粈瀣攽椤旂懓浜鹃梻浣哥枃濞夋盯鎮橀弴銏犵倞闁靛ě鍐ㄧ闂佽楠搁崢婊堝磻閹剧粯鐓冪憸婊堝礈閻旈鏆﹀ù鍏兼綑閸愨偓濡炪倖鎸绘竟鏇㈠磻閹惧瓨濯寸紒顖涙礃椤秹姊洪棃娑氱畾闁逞屽墯閺嬪ジ宕戦幘缁樺€婚柤鎭掑労濡懎顪冮妶鍡楀Е闁稿瀚伴悰顕€寮介鐔哄幐闁诲函缍嗘禍鐐存櫠閿旇姤鍙忓┑鐘插鐢盯鏌熷畡鐗堝櫧闁瑰弶鎸冲畷鐔碱敃閳ヨ櫕鎲㈤梻鍌氬€烽懗鍫曗€﹂崼銉︽櫇闁靛鏅涢崹鍌炴煕椤垵鏋ら柡鍡畵閹嘲鈻庤箛鎿冧紑濡炪倐鏅濋崗姗€骞冨Δ鍛櫜閹煎瓨绻冮幑锝夋⒑绾懎袚婵炲弶鐗犻崺鐐哄箣閿旇棄鈧兘鏌℃径瀣仼濞寸姍鍥ㄢ拺闁硅偐鍋涙俊鍏肩節閳ь剚娼忛埡浣哥亰婵犵數濮甸懝鍓х不椤曗偓閺屻倝骞侀幒鎴濆Б闂佸憡顭囬弫璇差潖閾忚瀚氶柍銉ョ－閳ь剙顭烽幃姗€鎮欑捄鐩掓挻銇勯弬璺ㄐф慨濠勭帛閹峰懘宕ㄦ繝鍌涙畼闂備胶纭堕弲娑㈡晝閿曞倸绠查柕蹇曞Л閺€浠嬫煕閳锯偓閺呮粎绱炴惔鈾€鏀介柣鎰级閳绘洖霉濠婂嫮鐭婃い鏂跨箰閳规垿宕辫箛鏃€鏉搁梻浣虹帛钃辩憸鏉垮暣閸┾偓妞ゆ帒鍊荤敮娑氱磼閸屾氨效鐎规洖銈稿鎾偄閸欏顏归梻鍌欑閹诧紕绮欓幋锔芥櫇闁靛／鍌滃墾闂佽顔栭崰姘卞婵傚憡鍋ｉ柛銉ｅ妼缁插鏌嶈閸撴瑥顭囬敓鐘参ュù锝呭濞尖晠鎮规潪鎵Э闁靛ň鏅滈悡鍐级閻愭潙顎滈柛蹇撴湰閵囧嫰鏁傞懞銉у嚒濡炪値浜滈崯瀛樹繆閸洖骞㈡繛鍡楁禋濞奸攱绻濆▓鍨灍閼垦囨煕閺冣偓閸ㄧ厧螞閻斿吋鈷戦柛婵嗗濡茶銇勯幋鐐垫噧闁靛棙甯楃换婵嗩潩椤撶姴甯鹃梻浣稿閸嬪懐鎹㈤崘顔㈠鎮欓悜妯衡偓鍫曠叓閸ャ劍绀冮柡鍡╁墴閺岋紕浠﹂崜褎鍒涢梺璇″枓閺呯姴顕ｆ繝姘ㄩ柕澶堝劚瀵兘姊婚崒娆掑厡缁绢厼鐖煎顒佺瑹閳ь剙鐣烽幋锕€绠婚柟棰佺劍鐎靛矂姊洪棃娑氬婵☆偅顨堢划顓㈠箳濡や胶鍘辨繝鐢靛Т閸婂綊宕悙鐢电＜闁稿本绋戝ù顕€鏌＄仦鑺ヮ棞妞ゆ挸銈稿畷鍗炩枎韫囨挾顔囬梻浣筋嚙妤犳悂宕㈠鍫濈；闁瑰墽绮崐鐢告煥濠靛棝顎楀褎澹嗛幃顕€鏁冮埀顒勫煘閹达附鍋愭繛鍫熷濮ｅ矂姊洪崨濠冨鞍缂佽鐗撻悰顕€宕橀鑲╋紲濠电偞鍨堕〃鍫㈢不濮樿埖鈷戠紓浣姑悘杈ㄤ繆椤愩垹顏╅柡渚囧枟娣囧﹪鎮欓鍕ㄥ亾閺嵮屾綎闁煎鍊曢崹婵囥亜閺嶎偄浠滅痪鎯ь煼閺屾稑鈽夊Ο鍏兼喖闂佹娊鏀辩敮锟犲蓟濞戞矮娌柛鎾楀嫬娅樼紓鍌欑劍閸旀牕螞閸愵喖钃熺€广儱鐗滃銊╂⒑閸涘﹥灏伴柣鈺婂灠閻ｇ兘骞嬪┑鍐╊潔闂侀潧绻嗛埀顒冩珪椤撶儤淇婇悙顏勨偓鏍偋濡ゅ啰鐭欓柟鐑樺灍閺嬪秹鏌ｅΟ鑲╁笡闁绘挾鍠愭穱濠囧Χ閸屾矮澹曢梻浣虹帛椤ㄥ懐鈧凹鍘剧划瀣吋閸滀胶鍙嗛梺鍓插亞閸犳捇宕㈤悽鍛婄厽閹艰揪绲鹃弳鈺傘亜椤撶偟澧涚紒鍌涘浮閺佸倿宕滆閿涙粓姊洪柅鐐茶嫰婢т即鏌嶈閸撱劎绱為崱娑樼獥婵°倕鎳忛崑鍌炴煛閸ャ儱鐏柣鎾冲暣閹嘲鈻庤箛鎿冧患闂佸憡鏌ｉ崐鏇㈡箒?
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
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵潙螖閳ь剚绂嶉幆褜娓婚柕鍫濈凹缁ㄥ鏌涢悢鍛婄稇闁伙絿鏌夐妵鎰板箳濠靛洦娅旈梻浣告啞娓氭宕归悧鍫熷弿婵炲樊浜濋埛鎺懨归敐鍕劅婵炲吋甯￠弻娑㈠即閻愬吀绮甸梺浼欑到婢у海妲愰幘瀛樺闁圭粯甯婃竟鏇炩攽閻橆喖鐏遍柛鈺傜墵瀹曟繈寮介鐔蜂簵濠电偛妫欓幐濠氬磹閸偆绠鹃柟瀵稿仧閹虫劙鏌ｉ幒鏇燁棄闁宠鍨块崹楣冩惞椤愩垺鐏庨梻浣告惈閻妲愰弴銏犵劦妞ゆ帒锕︾粔鐢告煕鐎ｎ亝鍤囬柛鈺傜洴楠炴帒螖娴ｅ搫骞嶉梻浣告啞閻熴儵藝闁秴绠洪柛鎰典簽绾惧ジ鏌ｅΟ铏癸紞婵炴彃顕埀顒侇問閸犳骞愰幎钘夌畺婵炲棙鎸婚崵鎴炪亜閹存繄浠㈤柡瀣墦濮婂宕掑▎鎰偘濡炪倖娉﹂崶褏顦╅梺浼欑到閻偐绮堟繝鍥ㄧ厱闁斥晛鍟伴埥澶岀磼閳ь剟宕奸悢绋垮伎濠碘槅鍨甸褔銆傞幎鑺ョ厱閻庯綆鍋呯亸顓熴亜椤愶絿绠炴い銏★耿閹瑩宕ｆ径瀣畼闂傚倸鍊风粈渚€骞栭锕€瀚夋い鎺戝€婚惌娆撴煙鏉堝墽鐣遍柣鎾寸箞楠炴牕菐椤掆偓閳ь剚顨婇幃鍧楀焵椤掑嫭鈷戦柟绋挎捣缁犳捇鏌＄仦鏂よ含闁轰礁鍟村畷鎺戔槈閹烘挸顏归梻鍌氬€风欢锟犲磻閸℃稑纾绘繛鎴欏灪閸ゆ劖銇勯弽銊р姇婵炲懐濮撮湁闁绘挸娴烽幗鐘绘煟閹惧鈽夐棁澶愭煥濠靛棙鍣规い銉ョ箻閺岋綁骞橀悷棰佹睏闂?session 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻樻爠鍥ㄧ厱闁靛鍨哄▍鍥煕濡厧鈻堟慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵＄偑鍊х紓姘跺础閸愯尙鏆﹂梻鍫熶緱濞尖晜銇勯幋鐘蹭沪婵＄偘绮欓妴渚€寮崼婵堫槹濡炪倖鎸鹃崑娑氱不娴煎瓨鈷掗柛灞剧懅椤︼箓鏌熷ù瀣⒉缂佹鍠庤灃闁告侗鍘鹃悰銉╂⒑閸濆嫮鈻夐柛妯垮亹缁牓宕奸妷锔惧帾婵犵數鍋熼崑鎾斥枍閸℃稒鐓冮梺鍨儏閻忔挳鏌″畝瀣М闁诡喓鍨介幃鈩冩償濠靛棙鐎抽梻鍌欑劍閹爼宕愬Δ鍛獥閹兼番鍔岄悡姗€鏌熸潏鎯х槣闁轰礁顑夐弻宥堫檨闁告挾鍠曞Λ銏ゆ⒑鐟欏嫬绀冩繛澶嬬洴瀹曠懓鈹戦崱蹇旀杸闂佺粯锚閻ゅ洦绔熷Ο鑲╃＜妞ゆ劑鍨绘晥闂佸搫鏈惄顖炪€侀弴銏犖ч柛娑卞枤娴滄牠姊绘担鍛婂暈闁荤喆鍎辫灋婵犻潧妫鏍ㄧ箾瀹割喕绨诲ù鑲╁█閺屾盯寮撮妸銉ヮ潻濡炪倧鑵归弲娑㈠煘閹达富鏁婇悷娆愬笚缁挸鐣峰┑鍥ㄥ劅闁靛鍎抽鍡涙偡濠婂懎顣奸悽顖涱殜瀹曟垹鈧綆鍠楅悡鏇熴亜閹邦喖孝闁诲浚鍠楅妵鍕籍閳ь剙煤閻旂厧绠栨俊銈呮噺閸嬶繝鏌℃径瀣靛劌婵☆偄鐭傚娲传閵夈儛锝夋煟濡や緡娈滅€殿喗鐓″畷濂稿即閻愭鍟嬫俊鐐€栧濠氬疾椤愶箑鍌ㄩ柟闂寸劍閸婄敻鏌涜箛鎿冩Ц濞存粓绠栭弻锝嗘償椤栨粎校闂佸憡鎸荤粙鏍焻閻㈠憡鈷掗柛灞剧懆閸忓瞼鐥鐐靛煟鐎殿喗鐓￠、鏃堝幢濡搫绨ユ繝鐢靛█濞佳兾涘☉銏犳辈闁挎洖鍊归悡娆撴煟閹寸伝顏堟倿妤ｅ啯鐓熼柟鎯у船閸旓箓鏌＄仦鍓ф创妤犵偛娲畷婊勬媴缁嬭法鍘掗梻鍌欐祰椤曟牠宕伴弽顓炵疇閹兼番鍔岄悡姗€鏌熸潏楣冩闁哄拋鍓熼弻娑㈠即閵娿儰绨甸梺鍝勵樈閸欏啫顫忛搹鍦煓闁告牑鍓濋弫楣冩⒑缂佹﹩娈樺┑鐐╁亾闂佺粯渚楅崳锝夌嵁閸ヮ剙绾у璺烘憸閻愬﹪姊绘笟鈧褔鏁嶈箛娑樻そ濞达絽澧ｉ敂鐣岀瘈闁汇垽娼у暩闂佽桨绀侀幉锟犲箞閵娾晛绾ч柟鐐藉妽濡炶姤淇婇幖浣肝ㄧ憸搴♀枔閸忚偐绠鹃柟鐐綑閻掑綊鏌涚€ｎ偅宕岄柡宀€鍠栭、娆戠驳鐎ｎ偆鏉归梻浣芥〃閻掞箓宕濋弽顓炵畾闁哄啫鐗嗛悘鎶芥煠閼圭増纭鹃柣鎺曨嚙閳规垿鎮╁▓鎸庢瘜闂佸憡鎸荤换鍡涘Φ閹版澘绀冩い蹇撴椤︻垱绻濋悽闈浶㈡繛灞傚妼閻ｅ灚绗熼埀顒勫蓟閳ユ剚鍚嬮幖绮光偓宕囶啇缂傚倷鑳舵慨楣冿綖婢舵劕桅闁告洦鍨扮粻濠氭偣妤︽寧銆冩繛宀婁邯濮婅櫣鎷犻懠顒傤唶濠碘槅鍋呯粙鎾澄ｉ幇鏉跨閻庢稒锚椤庢挻绻涚€电孝妞ゆ垵鎳橀獮妤呮偨閸涘ň鎷洪梺闈╁瘜閸樹粙宕甸埀顒€鈹戦悙鑼勾闁稿﹥绻堥弫鎰版倷閸撲胶鏉搁梺鍝勬川婵兘鏁嶅鍐ｆ斀闁宠棄妫楅悘銉╂煕鐎ｎ偄濮嶆鐐诧工閳规垹鈧綆鍋€閹锋椽姊虹涵鍛汗闁稿鐩畷婵嬪焵椤掑嫭鈷戠紒瀣濠€鎵磼椤斿ジ顎楅柍缁樻⒒閹瑰嫰濡歌閿涙繈姊虹粙鎸庢拱闁煎綊绠栭崺鈧い鎺嗗亾闁搞垺鐓″﹢渚€姊洪幖鐐插姶闁告搫绠撳顐も偓锝庡墯閸犳劙鏌￠崘銊у闁逞屽厸缁舵艾顕ｆ禒瀣垫晣闁绘劙娼ч埀顒傚仜椤啴濡舵惔鈥斥拻闂佸憡鎸堕崝搴ｆ閻愬鐟归柍褜鍓熼幃浼搭敋閳ь剙鐣烽崼鏇ㄦ晢闁逞屽墰閻ヮ亣顦归柡灞剧椤﹁櫕銇勯妸銉﹀殗鐎殿噮鍋夐妵鎰板箳閹绢垱瀚藉┑鐐舵彧缁蹭粙骞夐敓鐘茬畾闁割偆鍠撶粻楣冩倶閻愭彃鈧憡鎱ㄩ崒婧惧亾鐟欏嫭绀冮柛銊ユ健楠炲啴濮€閵堝懐顦ч梺绋跨箰閸氬鎮甸锝囩瘈闁汇垽娼ф禒婊勪繆椤愶絿鎳囩€规洖缍婂畷鐑筋敇閻戝棙顥￠梻浣稿悑缁佹挳寮插┑鍫濐棜闁芥ê顥㈣ぐ鎺撴櫜濠㈣泛顑嗛柨顓㈡⒑鐠囪尙绠氶柡鍛箞閸┾偓妞ゆ帊绶￠崯蹇涙煕閿濆骸娅嶇€规洘婢樿灃闁告侗鍘欓敃鍌涚厱闁哄洢鍔岄悘鐘诲船椤栫偞鈷戦柟绋垮椤ュ棙淇婇銏狀伃鐎殿喖鍟胯灃闁告劦浜為敍婵囩箾閹剧澹樻繛灞傚€濆绋库槈閵忥紕鍘藉┑掳鍊愰崑鎾绘煥閺囨ê鈧繈銆佸璺何ㄩ柍杞拌兌椤︽澘顪冮妶鍡欏婵ǜ鍔戦幃鍧楀焵椤掍椒绻嗛柣鎰典簻閳ь剚鐗犻獮鎰版嚒閵堝洨鐓撻梺鍦劋閺岋繝宕戦幘鎰佹僵妞ゆ劑鍊楅悡鎾斥攽椤旂》鍔熺紒顕呭灦楠炲繘宕ㄩ弶鎴濈獩婵犵數濮撮崐鐟扳枔濮椻偓濮婄粯鎷呴崨濠傛殘濠殿喖锕ょ紞濠傜暦瑜版帒鍨傛い鏃傚亾濞堟儳鈹戦悩缁樻锭婵☆偅鐩鎶藉幢濞戞瑧鍘撻悷婊勭矒瀹曟粓鎮㈤悡搴ｇ暰闂侀€炲苯澧柕鍥у缁犳盯骞樼€垫悶鍋愭繝纰樺墲瑜板啴濡剁粙娆炬綎婵炲樊浜濋ˉ鍫熺箾閹寸偟鎳勯柟顔界懇閺岋絾鎯旈姀銏㈠彎闂佸憡顨嗘繛濠囧箖娴兼惌鏁嬮柍褜鍓熼悰顕€骞掗幊铏閸┾偓妞ゆ帒鍊绘稉宥呪攽閻樺磭顣叉い銉ワ攻閵囧嫰骞掑鍫濆帯闂侀潧鐗婇幐鎶藉蓟濞戞埃鍋撻敐搴′簼鐎规洖鏈〃銉╂倷閺夋垹鐟ㄩ柧缁樼墪闇夐柨婵嗘噺閹牊顨ラ悙鑼ф慨濠勭帛閹峰懘鎸婃径濠冨劒闂備礁鎽滄慨鐢稿礉閺団懇鈧箓宕归鍛缓闂侀€炲苯澧撮柕鍡曠窔瀵噣宕奸锝嗘珝闂備胶绮崝蹇涘疾濠婂牆妫橀柍褜鍓熷缁樻媴鐟欏嫮浼囬梺鍝勬噺閻╊垰鐣烽娑橆嚤闁哄鍨归鎴︽⒑閸涘﹤濮€闁哄倸鍊圭粋宥咁煥閸喓鍙嗛梺鍝勫暙濞诧箓藟婢舵劖鐓冪紓浣股戠粈瀣煛鐏炵硶鍋撳畷鍥ㄦ畷闁诲函缍嗛崜娑㈡儊閸儲鍊甸悷娆忓缁€鈧悗瑙勬处閸撴繈鎮橀崘顔解拺闁告稑锕ゆ慨锕€霉濠婂啰鍩ｇ€规洦鍨抽埀顒婄秵閸犳鍩涢幒鎳ㄥ綊鏁愰崨顔兼殘闂佽鍨伴悧鎾诲蓟閿濆鏁嗛柍褜鍓熸俊鍓佺矙鐠恒劍娈炬繛鏉戝悑濞兼瑧绮绘繝姘€甸梻鍫熺⊕閹插摜鎲告导瀵哥暫婵﹥妞藉畷顐﹀礋椤掆偓缁愭稒绻濆▓鍨珮闁告瑥鍟悾鐑藉箣閿曗偓缁犺崵绱撴担鑲℃垵鈻嶅鍫熺厵闁兼祴鏅炶棢闂侀€炲苯澧憸鏉垮暣瀹曠敻鏌嗗鍡忔嫽婵炶揪绲介幉锟犲疮閻愮儤鐓犵憸鐗堝笧閻ｈ櫣鈧娲橀崝娆撳箠閿熺姴围闁糕檧鏅滅紞妤呮⒒娴ｇ瓔娼愰柛搴㈠▕閹椽濡搁敃鈧崹鏃堟煛婢跺娈憸鐗堝笚閸嬫劗鈧懓澹婇崰鏍礈娴煎瓨鈷戠痪顓炴噺椤ュ鏌ｈ箛鏃€鐨戦柟骞垮灩閳规垹鈧綆浜跺濠囨⒑闂堟稓绠氶柍褜鍓濆▍鏇㈡偩椤掍胶绡€缁剧増蓱椤﹪鏌涚€ｎ亝鍣介柟骞垮灲瀹曞ジ濡烽敃鈧禒鎺戭渻閵堝棙顥嗘繛澶嬫礋閹潡鍩€椤掆偓閳规垿鎮欓弶鎴犱桓闂佽崵鍠嗛崕鑼矉瀹ュ鍋￠柟浣冩珪閺傗偓闂備胶绮崝鏇㈡偤閵娧呯闁瑰鍋戞禍婊堟煙鐎涙绠栨い銉ｅ灮閳ь剚顔栭崳顕€宕滈悢鐓庢槬闁逞屽墯閵囧嫰骞掗幋婵愪痪闂佹娊鏀辩敮鎺楁箒闂佹寧绻傚ù鍌炴倿閻愵兙浜滈柕澶堝劤缁犲鏌＄仦鍓с€掗柍褜鍓ㄧ紞鍡涘磻閸曨個褰掝敊闁款垰浜鹃悷娆忓缁€鍐煕閵娿儳浠㈤柣锝囧厴婵℃悂鍩℃繝鍐╂珨闂備礁鎲℃笟妤呭闯椤曗偓瀹曨垰顓兼径瀣ф嫽婵炶揪绲介幉锟犲疮閻愬绠鹃悹鍥囧懐鏆ら悗瑙勬礀缂嶅﹪寮婚崱妤婂悑闁告侗鍨伴獮鍫ユ⒒娴ｄ警鏀伴柟娲讳邯濮婁粙宕熼娑樹簵闂佺粯鏌ㄩ崥瀣偂閻斿吋鐓涢柛灞剧☉椤曟粓鏌涢悢鍝勪户缂佽鲸甯￠幊婊堟偨闂堟稑缁╅梻浣告惈閼活垳绮旇ぐ鎺戣摕闁靛ě鈧崑鎾绘晲鎼粹€茶埅濠碘槅鍋勯崯鏉戭潖濞差亜浼犻柛鏇ㄥ墯閹疯京绱撴担鍓插剱闁圭懓娲畷娲焵椤掍降浜滈柟鐑樺煀閸旂喓绱掓径灞炬毈闁哄本绋撻埀顒婄秵娴滄粓顢旈銏＄厵妞ゆ梹鏋婚懓鎸庮殽閻愯揪鑰挎い銏＄懇閹墽浠﹂挊澶岀杽闂傚倸鍊风粈渚€鎮块崶顒婄稏濠㈣埖鍔曠壕鍧楁煣韫囷絽浜炴い?
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯闁靛／灞芥櫏闂備浇顕у锕傦綖婢跺⊕鍝勵潨閳ь剟骞冮敓鐘虫櫢闁绘灏欓悾鍝勨攽鎺抽崐鏇㈠箠韫囨稑鐤鹃柡灞诲劚缁犲湱绱掗鐓庡辅闁稿鎹囬獮鍥ㄦ媴鐟欏嫮銈板┑鐘殿暜缁辨洟宕戦幋锕€纾归柕鍫濐槸绾惧鏌涘☉鍗炲Ц婵炲樊浜濋崑鍕煟閹捐櫕鎹ｉ柣锝嗘そ閺岋綁鎮㈤崫銉﹀櫑闁诲孩鍑归崢浠嬪箞閵娿儺鐓ラ柛顐ゅ枔閸橀潧鈹戦悙鑼闁诲繑绻堝鎼佸Χ婢跺﹦顢呴棅顐㈡处缁嬫帡鍩涢幒鎳ㄥ綊鏁愰崼鐕佷哗闁汇埄鍨辩粙鎺楀箞閵婏妇绡€闁稿被鍊楅崥瀣倵鐟欏嫭纾搁柛銊ャ偢钘濈憸鐗堝笚閻撴瑦銇勯弮鍌滃彄妞ゅ繐鐗婇崑鈺呮煟閹达絾顥夐柣鎰躬閺屻劌鈹戦崱妯烘婵犮垼顫夊ú鐔煎蓟閿濆鍋勯柛婵勫劜閸Ｑ囨煟鎼淬垹鍤柛妯哄⒔閸掓帡宕奸妷銉у姦濡炪倖宸婚崑鎾绘煃鐟欏嫬鐏撮柟顔界懇閹崇娀顢楁担绋跨憥闂傚倷绀侀幉鈥愁潖瑜版帒鍨傞柟绋跨凹缁诲棝鎮楀☉娆樼劷闁荤喎缍婇弻宥堫檨闁告挻鐟╅幃楣冩倻閽樺）鈺呮煃閸濆嫸鏀婚柡鍛櫊濮婃椽宕ㄦ繝鍐ㄧ樂闁诲繒鍋犳慨銈呪枍濡ゅ懏鈷掗柛灞捐壘閳ь剛鍏橀幊妤呭醇閺囩偟鐤囬梺鍦亾濡炲潡寮€ｎ偁浜滈柟鎯у船閻忊晝鐥幆褜鐓奸柡宀嬬秮楠炲洭顢楁担鐟板壍闂備胶绮幐璇裁洪悢鐓庤摕闁绘柨鍚嬮崐缁樻叏濡も偓濡瑩鎮鹃悜鑺モ拺缂備焦蓱鐏忕増绻涢懠顒€鏋涚€殿喖顭烽弫鎰板川閸屾粌鏋涚€规洦鍋婂畷鐔碱敆娴ｇ儤袠濠电姷鏁告慨鐢割敊閺嶎厼绐楅柡宥庡亞閻捇鏌涢锝嗙闁绘帒鐏氶妵鍕箳瀹ュ洩绐楅梺鍝ュ枎缁绘﹢寮诲☉銏℃櫜閹肩补鍓濋悵婵嬫倵鐟欏嫭绀冮柛銊ュ閹广垹鈹戠€ｎ亞鍊為梺闈涱槶閸ㄨ绂嶉幆顬″綊鏁愰崨顓ф濠电偛鐨烽弲鐘诲箖鐟欏嫨鍋婇柟绋垮瘨娴犫晠姊洪崨濠冾棖缂佺姵鍨块垾鏃堝礃椤斿槈褔鏌涢埄鍐炬畼闁荤喆鍔戦弻锝嗘償閵忕姴姣堥梺鍝ュУ閻楃娀骞冩ィ鍐ㄥ瀭妞ゆ劑鍊楅幊婵嬫⒑闁偛鑻晶瀵糕偓娈垮枛椤兘骞冮姀銈呯闁兼祴鏅涙慨娲⒒娴ｇ懓顕滄繛鎻掔Ч瀹曟垿骞橀幖顓燁啍闂佺粯鍔栭幆灞解枔閺冨牊鐓冮悷娆忓閻忓瓨銇勯姀锛勨槈闁宠棄顦埢搴ょ疀閹惧瓨鏆橀梻鍌氬€风粈渚€骞栭锔藉剹濠㈣泛鐬兼稉宥夋煙鏉堝墽鎮煎ù婊嗘閳规垿鎮欑€涙ê闉嶉梺鍛婂灥缂嶅﹤鐣疯ぐ鎺撶劶鐎广儱鎳愰崢娲⒑閸︻厼浜鹃柡瀣偢瀵劍绂掔€ｎ偆鍘撻梺瀹犳〃缁€渚€寮抽悢鍏肩厱婵炲棗鑻禍鐐節閻㈤潧浠滈柣掳鍔庨崚鎺曠疀閹绢垱鐏冨┑鐐村焾濠胶绱為弽顓犲彄闁搞儯鍔嶆刊鍏肩箾瀹割喕鎲鹃柡浣稿€块弻娑樷槈閸楃偛绠抽梺姹囧灩椤嘲顫忔繝姘＜婵炲棙鍨肩粣妤呮⒑閸涘﹦鎳冪紒缁樺笧閸掓帗绻濆顒傤吅濠电娀娼ч崯顖炴倶娓氣偓濮婃椽宕滈幓鎺嶇凹濠电偛寮堕敃銏ゅ春濞戙垹绠ｉ柨鏃傛櫕閸樺崬鈹戦悩缁樻锭婵☆偅顨婇、鏃堫敆閸曨剙鈧灚鎱ㄥΟ鍝勮埞妞わ絽銈搁弻鏇㈠炊瑜嶉顓㈡煛娴ｇ鏆ｇ€规洘甯掗埥澶娾枎閹惧瓨娈梻鍌氬€峰鎺旀椤旀儳绶ら柛褎顨呯粈鍌涗繆椤栨瑧绉挎繛鎴欏灩缁€鍐煏婵炲灝鐏い顐㈢Ч閺屸剝寰勬繝鍕暤闂佸搫鎳忕粙鎴︻敋閿濆绠柤鎭掑劗閹风粯绻涙潏鍓у埌闁硅绻濆畷顖炴倷鐏炲倵鍋撻幒鎴僵妞ゆ挆鍕殾缂傚倷绀侀崐鍝ョ矓閹绢喒鈧箓濡搁埡浣侯槹濡炪倖鎸嗛崟顖涙殔濠电姴鐥夐弶搴撳亾瑜忓濠冪鐎ｎ亞鏌堝銈嗙墱閸嬫稓澹曟繝姘厓闁宠桨绀侀弳鐔虹磼閻樿崵鐣洪柡灞剧洴閸╁嫰宕橀悙顒傛毉缂傚倷鑳舵慨閿嬫櫠濡ゅ懎桅闁告洦鍠氶悿鈧梺鎸庣箓閹冲秹宕版繝鍥ㄢ拺闁告繂瀚悞璺ㄧ磽瀹ュ嫮绐旈柣娑卞枛閳诲酣骞樺畷鍥舵О闂備線娼ц噹闁稿本鑹鹃褏绱?User-Agent 婵?ForceCodexCLI 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆洤鍔嬮柛銈傚亾婵＄偑鍊栧ú锕傚储娴犲绠為柕濞炬櫅缁犵粯銇勯弮鍌氬付闁诲寒鍘奸埞鎴︽偐缂佹ɑ閿梺缁橆殔閸熸潙鐣烽幋锕€绠婚悹鍥皺椤ρ勭節閵忥絾纭鹃柨鏇稻缁旂喖寮撮姀鈾€鎷绘繛鎾村焹閸嬫捇鏌嶈閸撴盯宕戝☉銏″殣妞ゆ牗绋掑▍鐘绘煙缂併垹鏋熼柣鎾寸懅閳ь剝顫夊ú鏍洪妶鍡欘浄闂侇剙绉甸悡娑㈡倶閻愯泛袚妞ゃ儲绮嶉妵鍕箻閻愯棄浠悗瑙勬磸閸旀垿銆佸☉銏℃櫜闁告侗鍠楀▓浠嬫⒒閸屾艾鈧悂宕愬畡鎳婂綊宕堕妸锕€寮块梺闈涚墕椤︿即寮查鍫熲拻?
	customUA := account.GetOpenAIUserAgent()
	if customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}
	// OAuth 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟版晥濠电姭鍋撳〒姘ｅ亾婵﹨娅ｇ槐鎺懳熼搹閫涚礃婵犵妲呴崑鍕偓姘煎枤閸掓帗绻濆顓炰汗缂傚倷鐒﹂…鍥储閻㈠憡鈷戦悹鍥ｂ偓宕団偓濠氭煕閹般劍娅撻柛鐔风墦濮婄粯鎷呯憴鍕哗闂佺瀛╁钘夌暦濠婂牊鏅濋柛灞剧閻庮剟鎮楅獮鍨姎妞わ缚鍗抽幃锟犳偄閸忚偐鍘甸梻渚囧弿缁犳垿鎮橀悩缁樼厽闁靛闄勯妵婵嬫煛瀹€瀣М妤犵偞锕㈠鍫曞箣閻樿京绀夊┑鐘殿暯濡插懘宕戦崟顖氱疇婵せ鍋撻柕鍡曠窔瀵挳濮€閻欌偓濞煎﹪姊洪悙钘夊姕闁告挻鑹惧嵄閺夊牃鏅濈壕浠嬫煕鐏炲墽鎳呮い锔奸檮閵囧嫰骞嬪┑鍥舵￥闂佽桨绶￠崳锝夊极閹剧粯鍋愰柤纰卞墰閺嗩偊姊绘担铏瑰笡闁告梹娲熼、姘额敇閻樻彃袣闂侀€炲苯澧柍瑙勫灴椤㈡瑩鎯岄顐＄盎闁伙絿鍏橀、妤呭礃閿旀寧鐫忛梻浣告啞濞诧箓宕归柆宥呯厱闁硅揪闄勯悡娆撴煠濞村娅呭ù鐘崇矒閺屾盯寮崫鍕闂傚倸鍊风粈渚€骞夐垾瓒佹椽鏁冮崒姘鳖槶濠电偛妫欓幐濠氬煕閹烘挶浜滈柡宥庡亜娴狅箓鏌嶉柨瀣瑨闂囧鏌ㄥ┑鍡欏妞ゅ繒濞€閹粙顢涘☉姘垱闂佸搫鏈惄顖氼嚕椤曗偓閸┾偓妞ゆ巻鍋撻悡銈夋煟閺冨倸甯堕柡瀣╃窔閺屾盯骞囬棃娑欑彯闂佽桨绀侀崐鍧楀蓟濞戙埄鏁冮柨婵嗘川閻ｉ箖姊虹涵鍛撴繛鑼枎椤繐煤椤忓拋妫冨┑鐐村灱娴滎剟宕濋幖浣光拺缂佸鐏濋惁銊╂煕閻樺磭澧€规挸瀚板娲礈閹绘帊绨介梺鍝ュУ閹稿墽鍒掔紒妯稿亝闁告劏鏅濋崢閬嶆⒑閺傘儲娅呴柛姗€绠栭、鏃堝幢濞嗘垹鏆ラ梻浣告贡閸庛倝骞愰懜鐢殿洸婵犲﹤鐗婇悡蹇擃熆閼哥數鈽夋い鈺婂墰缁辨帡鎮╅搹顐㈤瀺闂侀潧娲ょ€氫即銆侀弮鍫濈妞ゆ劧绲鹃鎺戔攽閻樻鏆柍褜鍓欑壕顓㈠春閿濆洠鍋撶憴鍕鐎规洦鍓濋悘鎺撶箾閹炬潙鍤柛銊ㄧ簿閳悂姊婚崒姘偓鎼佸磹妞嬪孩顐芥慨姗嗗墻閻掍粙鏌ゆ慨鎰偓妤冪矆婵犲洦鐓曢柕澶堝灪濞呭懐绱掗煬鎻掆偓婵嬪蓟濞戞瑧绡€闁告洦鍋呴悘浣圭箾?Codex UA 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鏁愭径濠勵吅闂佹寧绻傞幉娑㈠箻缂佹鍘遍梺闈涚墕閹冲酣顢旈銏＄厸閻忕偛澧藉ú瀛樸亜閵忊剝绀嬮柡浣瑰姍瀹曞崬鈻庡Ο鎭嶆氨绱撻崒姘偓鐑芥倿閿曞倸绀夐柡宥庡幑閳ь剙鍟村畷濂稿Ψ閵壯冨Е婵＄偑鍊栧濠氬磻閹惧墎纾奸柣妯垮皺鏁堥悗瑙勬礃濞茬喎鐣烽敓鐘冲€剁紓浣股戦妵婵嗏攽閳╁啯鍊愰柛鈹惧墲閹峰懏绗熼娑欐殢婵犵數濮甸鏍窗濡ゅ應鈧箓宕奸妷銉︽К闂佸搫绋侀崢浠嬪磻閸屾稓绠鹃柛鈩兠悘鈺冪棯閹规劖顥夐棁澶愭煥濠靛棭妲规繛鎳峰洦鐓熼煫鍥э攻濞呭﹪鏌″畝瀣？闁逞屽墾缂嶅棝宕戦崱娑樜ラ柟鐑樻⒒绾惧ジ寮堕崼娑樺婵炴惌鍣ｉ弻娑㈠煘閹傚濠碉紕鍋戦崐鏍暜閹烘柡鍋撳鐓庡濠㈣娲熷畷妤呮嚃閳哄喚鍟庨梻浣烘嚀椤曨參宕戦悙鐑樺剭闁硅揪闄勯悡鐔兼煥閺囨浜鹃梺缁橆殔閿曪箓骞戦姀鐘斀閻庯綆鍋勬禒娲⒒閸屾氨澧涢柛鎺嗗亾闂侀潧绻堥崐鏍偂閺囩偐鏀介柣妯诲絻閺嗙偤鏌涙繝鍐ㄥ闁逞屽墯椤旀牠宕板Δ浣虹濠电姴娲ょ粻鏍煏韫囧鈧牠藟閸儲鐓熼柟閭﹀幘閸欌偓闂佸憡鐟ョ€氼噣鍩€椤掑喚娼愭繛鍙夌墵閹儵宕楅梻瀵哥畾闂佸綊妫块悞锕傚疾濠婂牆绾ч柣鎰綑椤庢粍淇婇悙顒佸€愭慨濠傤煼瀹曟帒鈻庨幋顓熜滈梻浣告贡閳峰牓宕戦崱娑樼畺妞ゆ洍鍋撴い銏℃礋閺佸啴鍩€椤掑倻鐭嗗璺哄閸嬫捇宕楁径濠佸闂備礁鎲″ú锕傚磻閸℃稑鐭楅柍褜鍓欓埞鎴︽偐椤旇偐浼囧┑鐐差槹缁嬫牠鎯傚畝鍕拺缂備焦蓱閻撱儵鏌涘顒夊剶鐎规洜鏁诲畷姗€顢欓悾灞藉笚闂佸搫顦遍崑鐐寸珶閸℃蛋鍥晜閻愵剙浠忛梺鍐叉惈閸婅崵寮ч埀顒勬⒑濮瑰洤鐏叉繛浣冲啯姣勫┑锛勫亼閸娧呯礊閸℃稑绐楁慨妯挎硾缁犳煡鏌曡箛鏇炐涢柡鈧禒瀣厱妞ゆ劦鍋呯欢姘辩磼濡も偓椤﹂潧顫忓ú顏勪紶闁告洦鍘搁弸鍡涙⒑閸涘鐒鹃柛瀣姉濡叉劙骞掑Δ鈧悡娑㈡煕濞戝崬鏋撻柟宄版惈椤啴濡堕崱妤€娼戦梺绋款儐閹稿墽妲愰幒鎾崇窞濠电姴楠稿▓鍓佺磽娓氬洤鏋涙い顓犲厴閵嗕礁鈽夐姀鈥斥偓鐑芥倵閻㈡鐒炬鐐茬墛缁绘繈鎮介棃娑楁勃闂佹悶鍔戝褔鎮鹃悜绛嬫晢闁告洦鍋勯崵鎴︽⒑缂佹ɑ鐓ラ柛姘儔閹繝寮撮姀鐘殿啇闁哄鐗嗘晶浠嬪礆娴煎瓨鐓涢悗锝庡亞濞叉挳鏌″畝瀣？濞寸媴绠撻幃娆擃敆閸屻倖效闂佽姘﹂～澶娒哄鍫濆偍鐟滄棃宕洪悙鍝勭闁挎洍鍋撴鐐灲閺屽秵娼幍顕呮М闁哥儐鍨跺濠氬磼濮橆兘鍋撳畡鎳婂綊宕堕妸锕€寮块梺闈涚墕椤︻垶宕归崒鐐寸厽闁靛繆鍓濈紞鍕煕閵夘喖澧紒鐘劜閵囧嫰寮崶顭戞闁瑰吋娼欓敃銈夊煘閹达附鍊烽柡澶嬪灩娴犙囨⒑閹肩偛濡兼繝鈧潏鈺佸灊濠电姵纰嶉弲鎻掝熆鐠轰警鍎戦柛妯圭矙濮婇缚銇愰幒鎴滃枈闂佸憡顭囬弲顐ゆ閻愬搫绠ｉ柣妯虹仛閿涘繘姊洪崫鍕垫Ъ婵炲娲橀弲銉︾節濞堝灝鏋涢柨鏇樺劚椤啯绂掔€ｎ剙绁﹂梺褰掑亰閸樹粙宕曢悢鍏肩厪闊浄绲剧欢娑氱磼妲屾牕寮慨濠勭帛閹峰懏绗熼婊冨Ъ婵犵數鍋涢顓㈠礂濡警鍤曟い鎰剁悼缁♀偓濠殿喗锚瀹曨剛绮婇敃鍌涒拺闁告稑锕ゆ慨锕傛煕閻樺磭澧柡渚囧櫍濮婄粯鎷呯粙鎸庢瘣闂佸湱鈷堥崑濠傜暦閹邦喚鐭欓幖瀛樻尰閻忎礁鈹戦悩缁樻锭婵☆偄鐭傚銊︾鐎ｎ偆鍘介梺褰掑亰閸ㄤ即鎯冮崫鍕电唵鐟滃酣鎯勯鐐茬畺婵°倕鍟扮粻鏃€绻涢幋鐏活亪顢旈鍛闁告侗鍘剧粻缁樻叏婵犲偆鐓肩€规洘甯掗埢搴ㄥ箳閹存繂鑵愰梻鍌欒兌椤㈠﹤鈻嶉弴銏犵闁搞儜鍛闂侀潧锛忛埀顒勫磻閹剧粯鏅查幖绮瑰墲閻忓秹姊虹粙娆惧剬闁哄懏绮撴俊鐢稿礋椤斿墽鏉搁梺瑙勫礃濞夋盯銆傚ú顏呪拺闁硅偐鍋涙俊濂告煕婵犲倹鍋ユ鐐插暙閻ｏ繝骞嶉搹顐も偓濠氭椤愩垺澶勯柟灏栨櫆鐎靛ジ鍩€椤掍椒绻嗛柣鎰典簻閳ь剚鍔欏鏌ユ偐鐠囪尙鍝楅梻渚囧墮缁夊绮婚悙鐑樼厪濠电偛鐏濋崝瀛樼箾閹炬剚鐓奸柟顔肩秺瀹曞爼顢旈崟顓烆槱濠电姰鍨奸～澶娒洪悢濂夋綎缂備焦蓱婵挳鏌涘☉姗堟敾闁稿孩鎹囧鍝劽虹拠鎻掝潻闂侀潧鐗忛…鍫ユ偩闁垮顕遍柡澶嬪灥閸炪劑鎮峰鍐鐎殿喗濞婇弫鍌炴偩瀹€鈧鏇㈡⒑缁洖澧查拑閬嶆倶韫囷絽寮€规洖澧庨埀顒佺⊕椤洨寮ч埀顒勬⒑闂堟盯鐛滅紒鎻掑⒔濞戠敻鎮欓鍙ョ盎闂佺懓顕崑娑欐叏瀹ュ鐓曢柍瑙勫劤娴滅偓淇婇悙顏勨偓鏍ь啅婵犳艾纾婚柟鎯у绾剧厧顭跨捄鐑樻拱闁搞倐鍋撴俊鐐€ら崑鍕崲濮椻偓楠炴牞銇愰幒鎾充画闂佸搫顦伴娆撳吹濞嗘挻鈷掑ù锝呮啞閹牊绻涚仦鍌氬鐎规洑鍗抽獮鍥偋閸繀鎴烽梻浣告惈閸燁偊鎮ф繝鍥х厱闁硅揪闄勯悡鐔兼煟閺傛寧鍟炵紒鑸电叀閺岋繝宕卞☉妤佸枤濠殿喖锕ㄥ▍锝夊极椤曗偓椤㈡瑩宕滈‖顔界矒濮婄儤娼幍顔煎闂佸憡姊归悷鈺呯嵁閸愩劉鏋庨柟鎯х－閻撴垿姊虹粙鎸庢拱缂佸甯¤棢闁割偀鎳囬崑鎾舵喆閸曨剛锛橀梺鍛婃⒐閸ㄥ潡濡存担鍓叉建闁逞屽墴楠炲啴顢旈崼婵嗙獩濡炪倖鐗楀銊╁传鎼粹檧鏀介柣妯虹仛閺嗏晛鈹戦鎯у幋鐎殿噮鍋婂畷銊︾節閸愩劌浼庨梻浣告贡閾忓酣宕板Δ鍛柧婵犻潧顑嗛悡蹇涙煕椤愶絿绠ユ俊鍙夋倐閺岋綁濡舵惔鈩冪彎闂佸搫鐫欓崱娆戞澑闂佹寧绻傞幊宥囪姳婵傚憡鈷戦柛娑橈功閹冲啰绱掔紒姗堣€跨€殿喖顭烽弫鎰緞婵犲孩缍傞梻浣虹帛閿氶柛鐔锋健瀵娊宕奸妷锔规嫼?
	if account.Type == AccountTypeOAuth && !openai.IsCodexCLIRequest(req.Header.Get("user-agent")) {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}

	// 婵犵數鍋炲娆戞崲濡ゅ拑缍栫€广儱顦梻顖炴煏婵炲灝鍔ら柍?UA 闂備胶顭堢换鎴犲垝瀹€鈧懞閬嶅箮閼恒儲娅栧┑顔斤供閸嬪棝鎯?OAuth闂備焦瀵х粙鎴濓耿閹辩嘲tGPT 闂備礁鎲￠崝鏇㈠箠濮椻偓瀹曟洟骞橀鑲╊唺闂侀潧顦崕杈╃礊閹剧粯鐓ユ繛鎴烇供濡茬厧顭跨捄鍝勵伃鐎规洩绲鹃ˇ鐗堟償閵忊剝顏熼梻浣芥〃缁€渚€鎮ч悙鍨潟婵犻潧顑嗛崵鎰版煏婢舵盯妾ù鐘愁焽缁?user-agent 濠电偛顕慨瀵哥矓瀹曞洦瀚婚柣鏃囶問绾懐鐤€婵ê鍚嬬紞宀勬⒑?	// 闂備焦瀵х粙鎴濓耿閹辩磧ome/Firefox/Safari/Edge 缂傚倷鐒︾粙鎴βㄩ埀顒傜磼鏉堛劌鍝洪柡浣哥Ф娴狅箓鎮欓鍌ゆ捶闂備胶顢婇鏍ь熆閳ь剟鎮归幇顔兼灈鐎规洏鍎查幆鏃堝灳閹惰棄褰欓梻鍌欑劍濠㈡绮旈崼鏇熷仾闁糕剝绋掗崕?Codex UA闂備焦瀵х粙鎴︽儗閸屾稑顕遍柍鍝勬噹缁€?Cloudflare 闂佽崵鍠愰悷杈╃不閹达絻浜?JS 闂佽崵濮甸崝鎴﹀礉韫囨侗鏁嬫俊銈呮噹杩?	s.overrideBrowserUserAgent(ctx, account, req)

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
	reqModel, _, _ := extractOpenAIRequestMetaFromBody(requestBody)
	_ = s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body, reqModel)
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
	// 闂傚倷绶￠崑鍕崲鐎ｎ剚顫曢悹杞扮秿閻旂厧鐏崇€规洖娲ㄩ、鍛箾閿濆懏澶勭紒璇插瀵娊鍩€椤掑嫭鐓曟俊顖滃帶閺嬪孩銇勯锝呭缂佸顦扮换婵嬪礃椤忓嫮鍘梻鍌欑劍鐎笛囨偡閵娾晩鏁嬮柕鍫濐槸娴肩姷鈧箍鍎遍幊鎰偓鐟邦樀閺屻劌鈽夊Ο渚缂備浇灏欓崑鎾愁焽婵犳艾绠涙い鏍电稻閺傗偓闂備浇顕栭崗娑樏归崶銊ョ窞闁告洦鍨伴惌妤€顪冪€ｎ亪顎楃粭鎴︽⒑鐠団€冲幐婵＄偞甯為崚鎺戔槈閵忊槅姊块梺閫炲苯澧寸€殿喖顕埀顒佺⊕钃遍柣鎾亾闂?	// 闂傚倷绶￠崜娆撴倶濠靛鍌ㄩ柕鍫濇搐閸ㄦ繃淇婇婊冨姦闁稿鎹侀ˇ鍫曟煟椤撱垻鐣洪柟顔垮Г濞煎繘鍩￠崒姘卞幈缂傚倸鍊风欢锟犲储婵傛潌鍥蓟閵夛箑浠洪梺闈涱焾閸庨亶鎮橀悩缁樺仩婵鍘ф禍鐗堜繆閹勬儓闂囧霉閿濆洤鍔嬮柣锝変憾閹綊骞囬崜浣烘殸閻熸粍婢樺鈩冧繆?	reqModel, _,
	reqModel, _, _ := extractOpenAIRequestMetaFromBody(requestBody)
	_ = s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body, reqModel)
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
			"[OpenAI passthrough] 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ゆい顓犲厴瀵鏁愭径濠勭潉闂佺鏈懝鐐繆濞差亝鍊甸悷娆忓缁€鈧悗娈垮枛婢у酣骞戦姀鐘斀閻庯綆鍋掑Λ鍐ㄢ攽閻愭潙鐏﹂柣鐔濆嫮顩峰┑鍌氭啞閸嬧剝绻濇繝鍌涘櫣妞わ絽銈搁幃浠嬵敍濞戞ɑ璇為梺璇″枟閻燂妇鎹㈠┑瀣闁挎棁娉曢惄搴ㄦ⒒娴ｅ憡璐￠柛搴涘€濆畷娲醇濠㈩亷缍€閵囨劙骞掗幘璺哄箰闂備礁鎲＄划鍫濐渻閹烘棁濮抽柤娴嬫櫇绾捐偐绱撴担璇＄劷婵炴彃顕埀顒侇問閸犳牠鎮ユ總鍝ュ祦閻庯綆浜栭弨浠嬫煕閵夋垟鍋撻柛瀣崌椤㈡稑顫濋敐鍡樻澑闂備胶绮崝鏍亹閸愵喖绠栭柟杈鹃檮閻撶喖鏌ｉ弬鍨骇婵炲懎锕ラ〃銉╂倷閸欏妫ゆ繛瀵稿闂勫嫬顭囬鍫濈閹兼番鍩勫Λ鍐倵鐟欏嫭纾搁柛鏃€鍨块妴浣糕枎閹惧磭鐣鹃悷婊冪Ф缁粯瀵肩€涙ǚ鎷洪柣鐘叉搐瀵爼宕径瀣ㄤ簻妞ゆ劑鍩勫Ο鈧Δ鐘靛仦閻楃娀宕洪敓鐘插窛妞ゆ挾濮峰畷鍫曟⒒娴ｅ憡鎯堢紒瀣╃窔瀹曘垽鎳栭埡鍐х瑝闂佺粯顭囩划顖炴偂濞嗘挻鐓欐繛鍫濈仢閺嬫稒銇勯敐鍛仮闁哄矉绻濆畷顏呮媴缁嬫娼界紓鍌欒兌缁垳鎹㈤崼婢盯宕橀妸銏☆潔闂佸憡顨堥崑娑欑珶婢舵劖鈷掑ù锝堟鐢盯鏌熼幖浣虹暫鐎规洑鍗冲浠嬵敇閻愯埖鎲伴梻浣告惈濞村嫮妲愰弴銏″仾闁逞屽墯缁绘繈鎮介棃娴躲垽鏌涢悤浣哥仸闁诡喚鏁诲顕€宕奸悢鍙夊闂佽崵濮村ú锕傛偂閿熺姵鍋樻繝濠傛噽绾惧ジ鏌ｅΟ鎸庣彧鐎规洖鐬奸埀顒侇問閸犳牠鈥﹂悜钘夋瀬闁瑰墽绮崑鎰版煙缂佹ê绗ч柣娑掓櫆娣囧﹪鎮欓鍕ㄥ亾閺嵮屾綎鐟滅増甯掔壕鍧楁煕閹邦剚鈻曢柛銈嗘礋閺屾洘绻涢悙顒佺彆闂佹娊鏀遍崹褰掓儉椤忓牜鏁囬柣鎰綑濞呫倝姊虹紒妯肩濞存粠浜濠氭晲婢跺浜滈梺鍛婄懃椤︿即鎮靛鑸碘拺缂備焦眉缁堕亶鏌涢悩鍐插摵闁糕斁鍋撳銈嗗笒閿曪妇绮旈悽鍛婄厱闁绘ɑ鍓氬▓婊呪偓娈垮枟瑜板啴鍩為幋鐘亾閿濆骸浜為柛妯绘崌閹嘲顭ㄩ崟顓犵厜閻庤娲樼划宀勫煝鎼淬劌绠抽柡鍌氬⒔缂? account=%d request_id=%s err=%v",
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
		).Info("上游流在未收到 [DONE] 时结束，疑似断流")
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
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆洤鍔嬮柛銈傚亾婵＄偑鍊栧ú锕傚储娴犲绠為柕濞炬櫅缁犵粯銇勯弮鍌氬付闁诲寒鍘奸埞鎴︽偐缂佹ɑ閿梺缁橆殔閸熸潙鐣烽幋锕€绠婚悹鍥皺椤ρ勭節閵忥絾纭鹃柨鏇稻缁旂喖寮撮姀鈾€鎷绘繛鎾村焹閸嬫捇鏌嶈閸撴盯宕戝☉銏″殣妞ゆ牗绋掑▍鐘绘煙缂併垹鏋熼柣鎾寸懅閳ь剝顫夊ú鏍洪妶鍡欘浄闂侇剙绉甸悡娑㈡倶閻愯泛袚妞ゃ儲绮嶉妵鍕箻閻愯棄浠悗瑙勬磸閸旀垿銆佸☉銏℃櫜闁告侗鍠楀▓浠嬫⒒閸屾艾鈧悂宕愭搴ｇ焼濞撴埃鍋撶€规洏鍎抽埀顒婄秵閸嬪棝銆呴幓鎹楀綊鎮╁顔煎壈缂佺偓鍎冲锟犲蓟閻旇櫣纾兼俊顖氭惈濞兼垿姊洪崫鍕靛剰缂佸缍婂濠氬灳瀹曞洦娈煎銈嗘⒒閹虫挻绂嶆ィ鍐┾拺閻炴稈鈧厖澹曢梻浣稿悑娴滀粙宕曢娑氼洸婵犲﹤鐗婇悡娑㈡煕閹板墎鍒板ù婊堢畺濮婃椽宕崟闈涘壋缂備緡鍣崹宕囧垝椤撱垺鍋勯悘蹇庣劍椤秹姊洪棃娑㈢崪缂佽鲸娲熷畷銏ゆ焼瀹ュ棌鎷洪柣鐔哥懃鐎氼剟宕濋妶鍡愪簻闁哄洢鍔屽顕€鏌涢埞鍨姕鐎垫澘瀚禒锕傚箚瑜岀划鈩冪節閻㈤潧浠滄俊顐ｇ懇楠炴劖绻濆顓炰簵闂侀潧顦弲婊堟偂濞戙垺鐓曢柕澶堝灪濞呭懎霉閻樺磭鐭掗柡灞界Х椤т線鏌涢幘瀵告噮缂佽京鍋炵换婵嬪磼濠婂嫭顔曢梻浣侯攰閹活亪姊介崟顖氱柧闁圭粯甯╅悢鍡涙偣鏉炴媽顒熼柛搴㈠灴閺屾稑顫滈埀顒佺鐠轰警娼栨繛宸簼椤ュ牊绻涢幋锝夊摵妞ゅ骸妫涚槐鎾存媴閽樺鏁挎繝娈垮枔閸婃牜绮氭潏銊х瘈闁搞儯鍔屾禒濂告倵閸忓浜鹃梺閫炲苯澧摶鐐淬亜閺嶎偄浠﹂柣鎾寸懇閹嘲鈻庤箛鎿冧患婵犳鍠栧ú顓㈠蓟濞戞﹩娼╂い鎺嶆祰绾偓缂?SSE 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鏉款熆鐟欏嫭绀嬬€规洜鍏橀、姗€鎮╃喊澶屽簥闂備浇顕ч崙鐣岀礊閸℃稑纾婚柟鎹愬煐椤洟鏌嶉崫鍕偓鑽ょ不閸撗€鍋撻悷鏉款棌闁哥姵娲滈懞杈ㄧ附閸涘﹦鍘搁梺鍛婁緱閸犳氨绮婚悙鐑樼厸閻忕偟鏅暩濡炪伇鍌滅獢闁哄本鐩獮妯兼崉閻戞浜柣搴㈩問閸ｎ噣宕戞繝鍌滄殾婵せ鍋撴い銏＄懅閸犲﹥娼忛妸褏袩闂傚倸鍊风欢姘焽瑜旈幃褔宕卞銏＄☉铻栧ù锝勮濞叉悂姊绘笟鍥у缂佸鏁婚崺娑㈠箣閿旂晫鍘遍梺鍦亾濞兼瑧寰婄拠瑁佺懓顭ㄩ崟顐㈩潚闂佸搫鐬奸崰鏍х暦濮椻偓瀹曪絾寰勬繝鍐悍闂佽瀛╅鏍窗濡ゅ懎绠栭柛灞惧嚬閸ゆ洟鏌涙繝鍕珡婵′勘鍔岄埞鎴︽倷鐎涙ê纰嶅銈庡幘閸忔ê顕ｆ繝姘嵆闁靛繒濞€閸炶泛鈹戦悩缁樻锭婵炴潙鍊歌灋闁跨喓濮甸悡鐔肩叓閸ャ劍灏电紒鐘靛仱閺屾盯寮埀顒勫垂閸ф宓侀柛鎰╁壆閺冨牆鐒垫い鎺戝閻撴繈鏌熼幑鎰靛殭闁搞劌鍊归妵鍕棘鐠恒劍鐧侀梺鐟邦殠閸嬪懏绌辨繝鍥ч柛銉仢閵夆晜鐓曢悗锝庡墮娴犙囨煛娓氬洤娅嶆鐐村浮瀵挳鎮╂潏鈺勫厭闂傚倸鍊搁崐椋庢濮樿泛鐒垫い鎺戝€告禒婊堟煠濞茶鐏￠柡鍛閳ь剛鏁哥涵鍫曞磻?usage
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
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆洤鍔嬮柛銈傚亾婵＄偑鍊栧ú锕傚储娴犲绠為柕濞炬櫅缁犵粯銇勯弮鍌氬付闁诲寒鍘奸埞鎴︽偐缂佹ɑ閿梺缁橆殔閸熸潙鐣烽幋锕€绠婚悹鍥皺椤ρ勭節閵忥絾纭鹃柨鏇稻缁旂喖寮撮姀鈾€鎷绘繛鎾村焹閸嬫捇鏌嶈閸撴盯宕戝☉銏″殣妞ゆ牗绋掑▍鐘绘煙缂併垹鏋熼柣鎾寸懅閳ь剝顫夊ú鏍洪妶鍡欘浄闂侇剙绉甸悡娑㈡倶閻愯泛袚妞ゃ儲绮嶉妵鍕箻閻愯棄浠悗瑙勬磸閸旀垿銆佸☉銏℃櫜闁告侗鍠楀▓浠嬫⒒閸屾艾鈧悂宕愭搴ｇ焼濞撴埃鍋撶€规洏鍎抽埀顒婄秵閸嬪棝銆呴幓鎹楀綊鎮╁顔煎壈缂佺偓鍎冲锟犲蓟閻旇櫣纾兼俊顖氭惈濞兼垿姊洪崫鍕靛剰缂佸缍婂濠氬灳瀹曞洦娈煎銈嗘⒒閹虫挻绂嶆ィ鍐┾拺閻炴稈鈧厖澹曢梻浣稿悑娴滀粙宕曢娑氼洸婵犲﹤鐗婇悡娑㈡煕閹板墎鍒板ù婊堢畺濮婃椽宕崟闈涘壋缂備緡鍣崹宕囧垝椤撱垺鍋勯悘蹇庣劍椤秹姊洪棃娑㈢崪缂佽鲸娲熷畷銏ゆ焼瀹ュ棌鎷洪梺闈╁瘜閸樺吋绂嶉悙顒傜闁稿繗鍋愰幊鍥┾偓瑙勬处閸嬪﹪骞栬ぐ鎺戠濠㈣泛锕ｆ竟鏇炩攽閻愯尙澧曢柣蹇旂箞瀵鈽夊锝呬壕婵炲牆鐏濋弸娑橆熆瑜嶅ù椋庢閹捐绠婚悗闈涘濞村嫰鏌ｆ惔顖滅У濞存粍绮庣划锝呪槈濞嗘垹鐦堥梺姹囧灲濞佳勭瑜旈弻娑氣偓锝庡亝鐏忕敻鏌嶈閸撴岸鎳濇ィ鍐ㄎх紒瀣儥濞兼牜绱撴担鑲℃垶鍒婇幘顔界厱婵炴垶锕Λ姘辩棯椤撴稑浜剧紓鍌氬€搁崐椋庢閿熺姴闂い鏇楀亾鐎规洖缍婇獮搴ㄦ寠婢跺鈧剙顪冮妶鍡樼５闁稿鎹囬弻鐔碱敊閻ｅ本鍣伴梺璇″枙缁瑥鐣峰Δ鍛紶闁靛／鍕灀闂傚倸鍊搁崐鐑芥倿閿旈敮鍋撶粭娑樻噺瀹曟煡鏌涚仦缁㈠殧婵炲樊浜滄儫閻熸粌绉硅棢婵鍩栭悡鐘崇箾閺夋埈鍎愭繛鍛川閻ヮ亪骞戦幇顓ф闂傚洤顦甸弻銊モ攽閸℃ê顎涘┑鐐跺亹婵敻濡甸崟顔剧杸闁规崘娉涢。鐑樼箾绾惧浜瑰┑顔炬暩閹广垹鈽夊鍡楁櫊濡炪倖妫佸畷鐢稿礄瑜版帗鈷戠紓浣股戠亸浼存煟閻曞倻鐣靛┑锛勬暬瀹曠喖顢涘杈╂綁闂備焦鎮堕崕娲偂閸惊娑㈠礋椤栨稈鎷洪梺鍛婄☉閿曘儵鎮￠悢鍏肩厱濠电姴鍟扮粻鐐碘偓娈垮枦椤曆囶敇婵傜鐐婇柕濞垮劤瑜版挳姊绘担鍛婃儓閻炴凹鍋婂畷鏇㈠础閻戝棗娈ㄩ梺鍝勵槼椤曟娊寮崼鐔蜂汗闂傚倸鐗婄粙鎰垝鐠鸿　鏀介柣鎰级閳绘洟鏌涘▎蹇撴殻濠碘€崇摠閹峰懘鎳栧┑鍫濇灁闁诡喕鍗抽崺鍕礃閳哄倹绶梻鍌氬€风粈浣圭珶婵犲洤纾婚柛娑卞灡瀹曟煡鏌涢弴銊ョ伇闁轰礁鍟埞鎴﹀磼濮橆厼鏆堥梺鎶芥敱鐢繝寮诲☉姘勃闁硅鍔曢ˉ婵嬫⒑闁偛鑻晶浼存煕韫囨棑鑰挎鐐诧工铻栭柛娑卞幘椤︻厽绻涙潏鍓ф偧闁硅櫕鎹囧畷銏ｃ亹閹烘挴鎷洪梺鍛婃尰瑜板啯绂嶅┑鍥╃闁告瑥顦辨晶鐢告煙椤旇棄顕滈柕鍫秮瀹曟﹢鍩￠崘銊ョ闂傚倷绀佺紞濠囧磻婵犲洤鍌ㄥΔ锝呭暙鍥撮梺鍦檸閸犳鎮″☉銏″€甸柨婵嗙凹缁ㄤ粙寮崼銉︹拺闁告繂瀚～锕傛煕閺傝法鐒搁柛鈺冨仱楠炲鏁冮埀顒勭嵁閵忋倖鐓冮柛婵嗗閳ь剚鍔欐俊鑸靛緞鐎Ｑ勫闂傚倷绶￠崑鍡涘磻濞戙垺鍤愭い鏍仜閸屻劑鏌涘┑鍕姢缁炬儳銈搁弻銈夋⒐閹邦喚銈锋繝銏ｎ潐閿曘垽寮婚敐澶嬫櫜闁搞儜鍐ㄧ濠电姷顣介崜婵嬫偤閺団懞鍥敋閳ь剟寮诲☉姘ｅ亾閿濆骸浜濋悘蹇ｅ弮閺屽秹鎸婃径妯恍﹂梺瀹狀嚙缁夊綊銆佸Ο娆炬Ш闂備緡鍙庨崹鎶藉焵?content-type
		if v := strings.TrimSpace(src.Get("Content-Type")); v != "" {
			dst.Set("Content-Type", v)
		}
	}
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯闁靛／灞芥櫏闂備浇顕у锕傦綖婢跺⊕鍝勵潨閳ь剟骞冮敓鐘虫櫢闁绘灏欓悾鍝勨攽鎺抽崐鏇㈠箠韫囨稑鐤鹃柡灞诲劚缁犲湱绱掗鐓庡辅闁稿鎹囬獮鍥ㄦ媴鐟欏嫮銈板┑鐘殿暜缁辨洟宕戦幋锕€纾归柕鍫濐槸绾惧鏌涘☉鍗炲Ц婵炲樊浜濋崑鍕煟閹捐櫕鎹ｉ柣锝嗘そ閺岋綁鎮㈤崫銉﹀櫑闁诲孩鍑归崢浠嬪箞閵娿儺鐓ラ柛顐ゅ枔閸橀潧鈹戦悙鑼闁诲繑绻堝鎼佸Χ婢跺﹦顢呴棅顐㈡处缁嬫帡鍩涢幒鎳ㄥ綊鏁愰崼鐕佷哗闁汇埄鍨辩粙鎺楀箞閵婏妇绡€闁稿被鍊楅崥瀣倵鐟欏嫭纾搁柛銊ャ偢钘濈憸鐗堝笚閻撴瑦銇勯弮鍌滃彄妞ゅ繐鐗婇崑鈺呮煟閹达絾顥夐柣鎰躬閺屻劌鈹戦崱妯烘婵犮垼顫夊ú鐔煎蓟閿濆鍋勯柛婵勫劜閸Ｑ囨煟鎼淬垹鍤柛妯哄⒔閸掓帡宕奸妷銉у姦濡炪倖宸婚崑鎾绘煃鐟欏嫬鐏撮柟顔界懇閹崇娀顢楁担绋跨憥闂傚倷绀侀幉鈥愁潖瑜版帒鍨傞柟绋跨凹缁诲棝鎮楀☉娆樼劷闁荤喎缍婇弻宥堫檨闁告挻鐟╅幃楣冩倻閽樺）鈺冩喐婢舵劕纾婚柟鐐灱濡插牊绻涢崪浣稿季濞存粠浜畷娲焵椤掍降浜滈柟鍝勬娴滈箖姊虹拠鑼婵ǜ鍔戦獮鎴﹀閻橆偅顫嶉梺闈涚箳婵炩偓闁哥偠娉涢埞鎴︽偐缂佹ɑ閿梺缁橆殔閹虫﹢骞栫憴鍕劅闁靛鑵归幏娲⒑閸涘﹦绠撻悗姘卞厴瀹曟洟骞囬悧鍫㈠幗闂佸啿鎼敃銈夋倶閿旈敮鍋撳▓鍨灈妞ゎ厾鍏橀獮鍐閵堝懍绱堕梺鍛婃处閸撴盯鍩€椤掍礁鈻曟慨濠冩そ瀹曘劍绻濇惔銏㈡毉闂備胶顭堥鍡涙晝閵忕姷鏆﹂柣妤€鐗忕弧鈧梺鎼炲劘閸斿矂鍩€椤掑倹鏆柟顔煎槻閳诲氦绠涢幙鍐х棯缂傚倷璁查崑鎾绘煃瑜滈崜鐔奉潖濞差亜鎹舵い鎾寸⊕鐎氭盯姊虹粙娆惧剰妞わ妇鏁婚獮鍐潨閳ь剙鐣峰鈧、娆戞喆閿濆棗顏归梻鍌欑閹诧紕绮欓幋锔芥櫇闁挎繂娲﹂浠嬫⒑椤掆偓缁夌敻鎮￠弴鐔稿弿婵＄偠顕ф禍鍓х磽娴ｅ搫校闁绘娲熼幃楣冩倻缁涘鏅濋梺铏劶婵倕螞閸愩劎鏆﹂柕濞炬櫓閺佸﹪鎮峰▎蹇擃仼妞ゅ繐婀辩槐鎾诲磼濞嗘帒鍘￠梺鎯х箰闁帮絽鐣烽姀锛勯檮缂佸娉曢鍡涙煟鎼搭垳绉靛ù婊冪埣閹偞绂掔€ｎ偆鍘甸梺瑙勵問閸犳牠銆傛總鍛婄厽闁规儳鐡ㄧ粈瀣煛瀹€瀣瘈鐎规洘锕㈤崺锟犲磼濮橆厽顔傞梻鍌欐缁鳖喚寰婇崸妤€鏋侀柟闂寸閻?x-codex-* 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鍝勬娴滈箖鏌熼崗鍏肩稇闁挎洦浜滈～蹇涙惞閸︻厾鐓撻梺鍦焾鐎涒晠骞忔潏鈺冪＝濞撴艾娲ら弸娑㈡⒑鐢喚绉鐐叉閻ｆ繈宕熼銈庡晪缂傚倸鍊烽悞锕傛晪闂侀€炲苯澧伴柡浣割煼瀵鈽夐埗鈹惧亾閿曞倸绠ｆ繝闈涙噽閺佹牠姊绘担渚敯婵☆偄瀚板畷顖烆敃閿曗偓閻撴﹢鏌熸潏鎯х槣闁轰礁锕弻锕€螣娓氼垱孝闂佸搫顑嗛悷鈺侇潖濞差亜浼犻柛鏇ㄥ墯閹疯京绱撴担鍓插剱闁搞劌娼″畷娲閳╁啫鍔呴梺闈涚墕濞层劑寮堕幖浣光拺闁圭娴风粻鎾绘煛鐏炴枻韬柡浣稿€垮畷婊嗩槾闁挎稒绻冪换娑欐綇閸撗勫仹濡炪値鍘奸悧鎾诲Υ閸岀偛绠柤鎭掑劤閸橀亶鏌ｆ惔顖滅シ闁告柨鐭傞崺鈧い鎺嶇劍缁€瀣亜閵忊€冲摵闁轰焦鍔栧鍕節閸パ勬毆闂傚倷绀侀幖顐⒚洪妸鈺佺；闁靛牆顦懜褰掓偣妤﹁￥鈧偓闁衡偓娴犲鐓熸俊顖濆亹鐢稒绻涢幊宄板缁诲棙銇勯幇鈺佺伄闁绘帒鎲￠〃銉╂倷閺夋垵顫掗悗瑙勬礃閿曘垽銆侀弮鍫濆耿婵炲棙鐟ч梻顖涚節閻㈤潧浠╅柟娲讳簽瀵板﹪骞戦幇鈺€姹楅梺鍛婂姀閺呮繈銆呴幓鎹ㄦ棃鏁愰崨顓熸闂佹娊鏀遍崹鍧楀蓟濞戞ǚ妲堟慨妤€鐗婇弫鐐節閵忥絾纭鹃悗姘嵆瀵鈽夐姀鐘栤晠鏌嶇悰鈥充壕闂佽鐓＄粻鏍蓟閻旂⒈鏁嬮柛鈩冪懅閻﹀牓姊虹拠鈥虫灀闁哄懐濞€楠炴牞銇愰幒婵囨櫇闂侀潧绻掓慨宕囨閻斿吋鈷掗柛灞剧懄缁佹壆鈧娲滈弫璇茬暦娴兼潙绠婚悗娑櫳戦悵宄扳攽鎺抽崐鏇㈠箠鎼达絿涓嶇憸鐗堝笚閻撴瑩鏌熷▓鍨灈妞わ絽銈搁弻锝夊箻鐎涙顦伴梺鍝勭焿缁绘繂鐣峰鈧弫鎰板川椤掆偓椤ユ岸姊婚崒娆愮グ鐎规洜鏁诲畷顖氣枎閹惧啿绐涘銈嗘礀閹冲繒绱炲Δ鍛拻濞达絽鎲＄拹锟犳煕鎼存稑鈧繂鐣烽悽鍛婃櫇闁稿本姘ㄩ惈鍕⒑閸撴彃浜濈紒璇插瀹曟繈鏁冮埀顒勨€旈崘顔嘉ч柛鈩冾殘閻熴劑鏌ｆ惔銏犲毈闁告挻绋掔粩鐔煎即鎺虫禍褰掓煙閻戞ɑ灏甸柛妯兼暬濮婂宕掑顑藉亾閹间緡鏁嬫い鎾嚍閸ヮ剚鏅滈柤鎭掑劙濮橈箑鈹戦悩缁樻锭妞ゆ垵妫濋崺娑㈠箣閿旂晫鍘遍梺鎸庢椤曆囩嵁閺嶎厽鐓曢柡鍌涱儥閸庢棃鏌＄仦鍓ф创闁糕斁鍓濋幏鍛村传閸曞灚姣囧┑锛勫亼閸娿倝宕㈤悡骞熸椽顢橀姀銏犵ウ閻庡箍鍎遍ˇ浠嬪极婵犲洦鐓熼柣鏃傗拡閺嗘帒顭跨憴鍕婵﹨娅ｇ划娆戞崉閵娧屽敹婵＄偑鍊х€靛矂宕规潏鈺冪焿鐎广儱妫庨崑鍛存煕閹般劍鏉归柟閿嬫そ濮婄粯绗熼崶褌绨介梺绋款儐閻╊垶骞婇悢纰辨晬婵炴垶鐟﹂悵鐑芥⒑鐟欏嫷鍟忛柛鐘虫皑婢规洟宕楃粭杞扮盎闂佸搫鍟崐鐢稿箯閿熺姵鐓熼柟鍓ф嚀娴滃綊鏌嶈閸撴岸顢欓弽顓炵獥闁哄稁鍘搁埀顒婄畵閺屻劎鈧絺鏅濈粻姘舵⒑缂佹ê鐏卞┑顔哄€濆鎻掝煥閸喓鍘靛銈嗙墪濡宕导瀛樺仾闁告洦鍨遍埛鎺楁煕鐏炴崘澹橀柍褜鍓熼ˉ鎾斥枎閵忕媭娼╂い鎺戭槺閸旂兘鎮峰鍕棃鐎殿噮鍋勯濂稿醇閵忋垺婢戞繝鐢靛仦閸ㄥ爼鈥﹂崶顑﹀洭濡搁埡鍌楁嫼闂傚倸鐗婃笟妤呮倿妤ｅ啯鐓曢幖绮瑰墲閹牓鎽堕悙鍝勭閺夊牆澧介幃鑲╃棯閸欍儳鐭欓柡灞剧〒娴狅箓骞戦幇顒夋闂備線鈧偛鑻晶顕€鏌涢姀锛勫弨婵犫偓娓氣偓濮婃椽骞栭悙鎻掑闂佸憡鏌ㄩ敃銉х矉閹烘閱囬柕蹇嬪灮閿涙粓鏌ｆ惔顖滅У闁告鏅☉鐢稿醇閺囩喓鍘搁梺绯曞墲椤ㄥ棝藟閵忋倖鐓涢悘鐐插⒔閳藉鏌嶉挊澶樻Ц妞ゎ偅绮撳畷濂告偆娴ｈ鐦旂紓?	// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃缂侇噮鍨抽幑銏犫槈閵忕姷顓洪梺鍝勫暊閸嬫捇鏌涢妶鍛ч柡灞剧洴婵＄兘顢欓悡搴樻嫽闂備浇妗ㄧ粈浣该洪銏犺摕闁哄浄绱曢悿鈧梺鍝勬川閸婎偊濡烽敂杞扮盎闂佹寧妫侀褍鈻嶅澶嬬厵妞ゆ梻鐡斿▓婊堟煟濞戝崬娅嶇€规洖缍婇、娆撴偂鎼搭喗缍撴繝纰夌磿閸嬫垿宕愯閳ь剟娼ч惌鍌氱暦閻熸壆鏆﹂柛銉戝啰浜伴梻浣稿閸嬩線宕曢柆宥嗙厑闁搞儯鍔庣弧鈧梺闈涢獜缁辨洜绮婚幘鍓佺＝鐎广儱鎷戦煬顒侇殽閻愭彃鏆ｉ柛鈺佸瀹曟﹢鍩℃担绋课ら梻鍌欑劍鐎笛呮崲閸屾娑樷枎閹惧磭鐛ラ梺鍝勭▉閸樹粙鍩涢幒鎳ㄥ綊鏁愰崨顔兼殘闂佽鍨伴悧鎾诲蓟閻旈鏆嬮梺顓ㄧ畱閸撳爼鎮楃憴鍕缂侇喖鐭傞敐鐐测攽閸喎纾梺鎯х箰濠€閬嶅级娴犲鈷掑〒姘ｅ亾婵炰匠鍥ｂ偓锕傚醇閵夈儳锛熷┑鐐叉閹稿宕戦崟顖涚厽闁圭偓濞婇崣鍕煛娴ｅ湱鐭婇柍瑙勫灦楠炲﹪鏌涙繝鍐ㄥ鐎规洘鍨块獮妯肩磼濡厧骞嶉梻浣风串缁蹭粙寮甸鍕辈妞ゆ帊鐒﹂崣蹇撯攽閻樺弶鍣烘い蹇曞█閺屾盯寮介妸褍鈷岄悗娈垮枟閹告娊骞冮姀銈呬紶闁靛鍎抽崚浼存⒒閸屾艾鈧兘鎳楅崼鏇椻偓锕傚醇閵夛附娅囬梺闈涚墕閹峰鎮炴禒瀣叆闁绘柨鎼暩閻?http.Response.Header 闂?key 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁惧繐婀遍敍婊堟⒑闂堟稓绠冲┑顔炬暬閹﹢宕奸姀銏紲闂佺粯鍔﹂崜娆撳礉閵堝棎浜滄い鎾跺Т閸樺鈧鍠栭…鐑藉箖閵忋倖鍋傞幖杈剧秵濡插爼鏌ｉ悢鍝ョ煀缂佺粯甯″顐︻敊鐏忔牗顫嶉梺闈涢獜缁辨洟宕㈡禒瀣拺閻熸瑥瀚粈鍐╃箾婢跺娲撮柛鈹垮灲瀹曞ジ濡烽敂鎯у箰闂佽绻掗崑娑欐櫠閽樺铏光偓娑欙供濞堜粙鏌ｉ幇顖涘涧闁兼媽娉曢埀顒侇問閸犳牠鎮ユ總鍝ュ祦閻庯綆鍠楅崑鎰版煟閹邦喗鍤€濞寸媭鍙冨濠氬磼濮橆兘鍋撴搴ｇ焼濞撴埃鍋撴鐐差樀閺佹捇鎮╅搹顐ｇ彣闂傚倸鍊烽懗鍫曘€佹繝鍐╁弿闁靛牆娲ら崹婵嬪箹缁懓鐏抽柨婵嗩槸缁犳盯鏌℃径搴㈢《妞ゆ柨锕铏规喆閸曨偄濮㈠銈嗘处閸樺墽鍒掗弮鍫濋唶闁哄洨鍟块幏娲⒒閸屾氨澧涘〒姘殜閹偞銈ｉ崘鈺冨幈闁硅偐琛ラ崜婵嬪箚閸儲鎳氶柣鎰劋閻撴洟鏌嶉埡浣告殶闁愁垱娲熼弻?canonicalize闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ礂閻撳簶鍋撶紒妯圭箚妞ゆ牗绮岄崝锕傛煙瀹勯偊鐓兼慨濠傛惈鐓ら悹鍥ㄥ絻缁犳椽姊洪懡銈呮毐闁哄懐濮撮锝嗙節濮橆厼浜滈梺绋跨箺閸嬫劙宕濋悜鑺モ拺闁圭瀛╃壕鐢告煕鐎ｎ偅宕岄柡宀嬬磿閳ь剨缍嗛崑鍡涘Υ閹烘梻纾奸弶鍫涘妼濞搭噣鏌ｉ幙鍐ㄥ⒋妤犵偞鎹囬獮鎺楀幢濡炴儳顥氭繝鐢靛█濞佳囶敄閸涘瓨瀚呴柣鏂垮悑閻撱儲绻濋棃娑欙紞婵℃彃鎽滅槐鎺楁偐瀹曞洦鍒涘┑顔硷工椤嘲鐣锋總鍛婂亜閻炴稈鍓濋～宥嗕繆閻愵亜鈧倝宕戦幒妤€纾诲┑鐘插亞濞兼牠鏌涘┑鍡楊伀闁哄棴绠撻弻娑㈠Ψ閵忊剝鐝旂紓浣广仜閳ь剚鏋奸弨浠嬫煃閽樺顥滃ù婊勭矒閺屾盯鎮ゆ担闀愬枈濡炪們鍨哄畝鎼佸极閹邦厼绶炲┑鐘叉搐閺佸綊姊绘担鍛婃儓婵炲眰鍔戝畷鐗堟償閵娿儳鍔﹀銈嗗笒閸婃悂宕㈤幘顔界厸閻忕偟顭堟晶鑼偓鍨緲鐎氼噣鍩€椤掑﹦绉靛ù婊呭仱瀹曟繄绮欐惔鎾存杸闂佸疇妫勫Λ妤佺濠婂嫪绻嗘い鎰剁悼缁犳挻銇勯銏㈢閻撱倖銇勮箛鎾愁仹缂佸崬鐖煎娲濞戣鲸肖闂佺瀛╅悡锟犲箖濡や胶绡€婵﹩鍘搁幏娲⒑閸︻収鐒鹃悗娑掓櫆瀵板嫰宕熼鈧悷閭︾叆閹煎瓨绻勯惄搴☆渻閵堝棙绌跨紒鎻掓健楠炲繘鎮╃憗浣告贡閳ь剨缍嗛崑鎺戭焽閺冣偓缁绘繈鎮介棃娴躲垽鏌ㄩ弴妯衡偓婵嬬嵁婵犲洤绠涢柡澶庢硶妤犲洭姊鸿ぐ鎺擄紵闁绘帪绠撻崺娑㈠箣閿旂晫鍘卞┑鐘绘涧濡顢旈鍡忓亾閻熺増鍟炵紒璇茬墦瀵鈽夊锝呬壕闁挎繂楠告禍鐐烘煃椤栨稒绀堢紒杈ㄥ浮瀹曟帒鈽夊Ο鑲╂噯闂佸彞绱徊浠嬪Υ閹烘埈娼╅柣鎾虫捣娴狀參鏌ｆ惔銊︽锭闁活厼鍊搁～蹇撁洪鍜佹濠电偞鍨堕懝楣冦€傞崫鍕ㄦ斀闁宠棄妫楁禍婵嬫煟閻斿弶娅婄€规洖鎼埥澶愬閻樼數鏉搁梻浣虹帛椤ㄥ懘鎮ц箛娑樺偍闂侇剙绉甸埛鎴︽煕濠靛棗顏╅柡鍡楋躬閺屾稓鈧綆鍋呯亸鎵磼缂佹娲撮柟宕囧█椤㈡鍩€椤掑嫬鍑犻幖娣妽閻撱儵鏌￠崶顭嬵亪鎮橀鍡欑＜婵°倕鍟弸娑㈡煕閳规儳浜炬俊鐐€栫敮鎺斺偓姘煎墴閹锋垿鎮㈤崗鑲╁弳濠电娀娼уΛ娆撍夐悙鐑樼厱閹艰揪绲介弸鎴︽煏閸パ冾伃鐎殿喕绮欐俊姝岊槾闁绘挸鍊块弻鈥愁吋韫囨洜鐦堝┑顔硷功缁垶骞忛崨鏉戝窛濠电姴鍊瑰▓姗€姊洪挊澶婃殻濞存粌鐖煎璇测槈濞嗘劕鍔呴梺鐐藉劥濞呮洖袙閹扮増鍊?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆樻К闂備胶顭堢悮顐﹀礉鎼淬劌绠熼柟闂寸缁秹鏌涢锝嗗剷闁靛鏅滈悡鏇㈡煙閼割剙濡芥繛鍛閺屾稒鎯旈敐鍛亪闂佸搫鏈ú婵堢不濞戞埃鍋撻敐搴濈敖闁告梹娼欓埞鎴︽倷鐠鸿櫣姣㈠銈庡幖閻楁捇宕洪悙鍝勭闁挎棁妫勬禍褰掓煟閻樺弶鍘傞柛鎰屽懐鐓戦梻鍌氬€烽懗鍫曘€佹繝鍌楁瀺闁哄洢鍨圭粈澶嬩繆閵堝嫯鍏岄柣婵嗙埣閺岋繝宕堕埡浣圭亖濡炪倐鏅濋崗姗€骞冨Δ鍛櫜閹肩补鈧尙鏁栭梻浣告啞椤洭寮拠宸綎闁惧繗顫夐崗婊堟煕濠娾偓閻掞妇绱為崼銉︹拺闁告稑锕ラˉ鏍磼閻樿櫕灏柣锝囧厴濡啫鈽夐幒鎾垛偓濠氭倵閻㈤潧顣肩紒璇插€荤槐鐐寸節閸パ呯枃闂佸搫绋侀崢濂告偂濞戞◤褰掓晲閸涱喖鍋嶉梺绋挎捣閺佽顕ｇ拠宸悑闁割偒鍋呴鍥⒒娴ｅ憡鍟為柟绋款煼閹柉顦撮柛鎺撳笧閹风姴顔忛鍏煎€┑鐘灱濞夋盯鏁冮敃鍌涙櫖闁绘柨鍚嬮埛鎺戙€掑顒佹悙濠⒀屽枤缁辨帗寰勬繝搴℃缂備緡鍠栭…鐑藉箖閳哄啰纾兼俊顖滃帶楠炲秹姊婚崒娆戣窗闁告瑥绻掔划濠氬箣閿曗偓閸戠娀鏌熼崜褏甯涢柣鎾存礋閹鏁愭惔鈥茬凹閻庤娲栭惌鍌炲蓟閻旂⒈鏁婇柟顖嗗啫绠ｉ梻浣告惈閼活垰煤椤撱垹鏋侀柛灞剧矋閸犲棝鏌ㄥ┑鍡橆棤闁逞屽厸閸楀啿顫?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾剧懓顪冪€ｎ亝鎹ｉ柣顓炴閵嗘帒顫濋敐鍛婵°倗濮烽崑鐐烘偋閻樻眹鈧線寮村杈┬㈤梻浣规偠閸庢椽宕滈敃鍌氭瀬闁告劦鍠楅悡銉╂煛閸ヮ煈娈斿ù婊堢畺濮婂搫效閸パ€鍋撳Δ鍛；闁规崘鍩栧畷鍙夌節闂堟稒宸濈紒鈾€鍋撻梻浣侯焾閺堫剛鍒掑畝鍔肩兘鍩€椤掑嫭鈷掑ù锝勮閻掔偓銇勯幋鐐茬仼闁瑰箍鍨归埞鎴犫偓锝庝海閹芥洟鎮楅獮鍨姎妞わ富鍨辩€靛ジ鎮╃紒妯煎幈闂佸搫娲㈤崝灞炬櫠椤旂晫绠鹃柛婊冨暟缁夘喗鎱ㄦ繝鍌ょ吋鐎规洘甯掗～婵嬵敄鐠恒劍鏅奸梻鍌欑劍閹爼宕濆畝鍕亯闁绘挸瀵掗崵鏇炩攽閻樺磭顣查柡鍛絻椤法鎹勯悮鏉戝濡炪倖鎸诲钘夘潖濞差亜绠伴幖杈剧悼閻ｇ敻姊洪悷鏉跨骇闁烩晩鍨堕悰顔嘉熼懖鈺冿紲濠碘槅鍨抽崢褔鐛?EqualFold 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欑拠淇卞妽閵囧嫰寮崶顬挻绻涢崨顓犲ⅵ婵﹦绮幏鍛存惞楠炲簱鍋撴繝鍥ㄧ厱闁规儳顕粻妯肩磼椤旂晫鎳囨鐐村笒铻栭柍褜鍓涙竟鏇㈠箰鎼存繄绠氶梺闈涚墕鐎氭澘顬婂畡鎳婄懓顭ㄩ崟顓犵厜闂佸搫鐭夌换婵嗙暦閹烘埈娼╅柛娆愵焾濡炬悂姊绘担鐟扳枙闁衡偓閸楃倣娑㈠礃椤旇壈鎽曞┑鐐村灦椤倿寮埀顒勫箯閸涱垳鐭欓柟绋垮閸ゅ牓姊婚崒娆戭槮闁圭⒈鍋呭鍕炊椤掆偓缁€鍫熺節闂堟稒锛嶉柛銈嗩殜閺屾盯寮撮妸銉ョ闂佺顑嗛幑鍥极閹邦厽鍎熸繝闈涚墛閺呯厧鈹戦悙鑼憼缂侇喖鐭傞幃銉╂偂鎼搭喗缍庡┑鐐叉▕娴滄粍瀵奸悩缁樼厱闁哄洢鍨哄☉褔鏌涢幘鎻掑祮婵﹦绮幏鍛矙鐠恒劎娉跨紓鍌氬€哥粔宕囩矆娓氣偓椤㈡岸鏁愰崶銊ョ彴閻庣懓澹婇崰鏍箖閹寸偟绡€闁靛骏绲剧涵鍓ф嫬閳哄懏鐓忓┑鐐戝啫顏╅柡瀣灴閹鐛崹顔煎濠碘槅鍋呴惄顖炲Υ閸涘瓨鍊婚柤鎭掑劤閸樺崬鈹戦悩缁樻锭婵☆偅顨婇崺銏ゅ即閵忥紕鍘介梺缁樻⒒閸樠勭娴煎瓨鐓熼柟鎯у船閸旓箓鏌″畝鈧崰鏍蓟閸ヮ剚鏅濋柍褜鍓熷鎼佸Χ閸ャ劌浠忛梺鎸庢礀閸婂綊鍩涢幒鎳ㄥ綊鏁愰崶鈺傛啒闂佹悶鍊曢崯鎾蓟濞戙垺鍋愰柟棰佺劍閻や線姊洪悙钘夊姷缂佺姵鎸搁悾閿嬬附缁嬫娼婂銈庡亽閸橀箖鎮峰┑鍥╃瘈闁汇垽娼ф禒锕傛煕椤垵鐏︾€规洖缍婂畷鐑筋敇閻樿京鐟濆┑鐘垫暩婵數鍠婂澶嬪亗闁哄洢鍨洪悡娆撴煙鐟欏嫬濮囬柣銊︽そ閹綊骞囬鍕ギ闂佸搫鏈惄顖炲箖濞嗘挻鍤戞い鎺嗗亾闁绘挾鍠庨埞鎴︻敊绾嘲濮涚紓渚囧櫘閸ㄥ爼鐛箛娑樺窛闁哄鍨电粣娑欑節閻㈤潧孝闁哥噥鍋婂鎼佸箣閿旇В鎷绘繛杈剧到閹诧繝骞夌粙搴撴斀妞ゆ梻鍋撻弳顒傗偓瑙勬礃缁诲倿鍩㈡惔銊ョ閻犻缚娅ｉ妶锕傛⒒娴ｇ瓔娼愬鐟版閺呰泛螖閸涱厼鎯為梺鍦劋椤ㄥ棝鎮¤箛娑欑厱妞ゆ劧绲跨粻鏍偓鐟版啞缁诲牓寮婚敐澶婄閻庢稒锚闂夊秶绱撴担铏瑰笡缂佽鐗婇幈銊╁焵椤掑嫭鐓ユ繝闈涙椤ョ娀鏌曢崱妯哄妞ゎ亜鍟存俊鑸垫償閳ュ啿顒滈梻浣告啞閹搁箖宕版惔銊﹀仼闁绘垼妫勫敮閻熸粌顦靛畷鎴﹀箻缂佹ɑ娅滈柟鑲╄ˉ閸撴繈鎮樺澶嬧拺缂備焦蓱鐏忕増绻涢懠顒€鏋涚€殿喛顕ч鍏煎緞婵犲嫬骞愬┑鐐舵彧缁插潡鈥﹂崼銏犵筏鐟滅増甯楅埛鎴犵棯椤撶偞鍣虹憸鎶婂洦鈷掗柛鏇ㄥ亜椤忣厾鈧娲橀崹鍧楀箖閵忋倕绀傞悘蹇旂墬鐎氬吋淇婇悙顏勨偓鏍ь啅婵犳艾纾婚柟鎯у绾剧厧顭跨捄鐑樻拱闁哄棭鍓熼弻娑㈠煛閸屾粍鍒涘Δ鐘靛仜椤戝寮崒鐐村癄濠㈣泛顦遍弫鎯р攽閻樺灚鏆╅柛瀣☉铻炴俊銈呭暞瀹曟煡鏌熸导鏉戜喊闁轰礁鍊块弻宥夊传閸曨偀鍋撻懜鐢殿洸婵犲﹤鐗婇悡蹇擃熆鐠鸿櫣澧曢柛鏃€鎸抽弻娑欐償閵忊槅妫冮梺鍝勭焿缁辨洘绂掗敃鍌氱鐟滃酣宕氬☉銏♀拺閻犲洠鈧櫕鐏€闂佸搫鎳愭繛鈧鐐寸墳閵囨劙骞掑┑鍥ㄦ珖闂備線娼х换鍫ュ春閸曨垰姹查柨鐔哄У閳锋帡鏌涚仦鍓ф噮妞わ讣濡囬惀顏嗙磼閵忕姴绠婚梺鍛婂笚鐢繝銆佸☉銏″€烽柤纰卞墻閸炲綊姊绘担瑙勫仩闁稿孩妞藉畷婊冣槈濡吋娈兼繛鎾村焹閸嬫捇鏌熼绛嬫疁闁轰焦鍔栭幆鏃堝焺閸愵亙鎲鹃梻鍌欑閹碱偊骞婅箛鏇熷床闁割偁鍎遍拑鐔衡偓骞垮劚閻楁粌顬婇妸鈺傗拺闁告稑锕ョ亸鐢告煕閻樺磭澧甸柣娑卞枦缁犳稑鈽夊Ο纰卞悈闂備焦瀵уΛ浣肝涢崟顒傤洸婵犲﹤鐗婇埛鎴犵磼鐎ｎ厽纭剁紒鐘虫そ閺屾稓鈧綆鍋勬慨宥団偓瑙勬磸閸ㄤ粙銆侀弮鍫濆窛妞ゆ牗绮庡Σ鍥⒒娓氣偓濞佳勵殽韫囨洖绶ゅù鐘差儐閸嬪倿鏌熼柇锕€鏋ょ痪鎯с偢閹鏁愭惔鈥茬凹閻庤娲栭惌鍌炲蓟閻旂⒈鏁婇柟顖嗗倸顥氭繝鐢靛仧閸樠呮崲濡绻嗛柟闂寸劍閺呮煡鏌涢弴銊ュ箻婵炲娼″濠氬磼濞嗘垹鐛㈠┑鐐板尃閸よ翰鍔戦崺鈧い鎺嗗亾闁宠鍨块崹楣冩惞椤愩垻鐛╃紓鍌欑贰閸犳牠鈥﹂悜钘夌畺婵炲棙鎸婚崐缁樹繆椤栨粌甯舵?
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
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))

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
		// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ゆい顓犲厴瀵鏁愭径濠勭杸濡炪倖甯婇悞锕傚磿閹惧墎纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼淬劍娑у鐟帮工鍗辨い鏂垮⒔绾捐棄霉閿濆懏鎯堥崯鍛婄節濞堝灝鏋涢柣蹇旀皑缁碍娼忛妸褏鐦堥梺鎼炲劥閸╂牠寮查鈧埞鎴︽偐缂佹ɑ閿柣搴㈢殰閸パ咃紱闂佽宕橀褔鎮為崹顐犱簻闁圭儤鍨甸顏堟煟閹惧娲撮柟顔筋殜閺佹劖鎯斿┑鍫熸櫦闂佽桨绀侀悧鍡氱亙闂佺粯锕㈠褎绂掗敃鍌涚厽婵°倓绶″▓鏃€銇勯鐐寸┛缂佺姵绋戦埥澶娢熺喊杈ㄐら梺鑽ゅ枑缁瞼绮旇ぐ鎺戠畾閻忕偠袙閺€浠嬫倵閿濆骸浜愰柟鐤缁辨挻鎷呴崜鎻掑壉濡炪倖鍨堕悷鈺佺暦閺囥垹绠涢柣妤€鐗忛崢鐢告⒑閼姐倕鏋斿褎顨婂畷鏉款潨閳ь剟寮婚悢纰辨晩閻犲洦褰冪粊顔碱渻閵堝啫鐏€光偓閹间礁鏄ラ柍褜鍓氶妵鍕箳閸℃ぞ澹曢梻浣风串缁插潡寮绘径鎰﹂柟鐗堟緲缁犺櫕绻濋棃娑欏婵炲懏鎹囧缁樻媴閸涘﹤鏆堥梺鑽ゅ櫐缂嶄礁鐣烽弴銏犵闁告瑥鍊归惄顖炪€佸☉銏″€风紒顔款潐鐎氫粙姊绘担渚劸闁哄牜鍓熼幃鐑藉Ω閳轰胶顦ч悗鍏夊亾闁逞屽墴閹偓妞ゅ繐鐗滈弫鍥煟閹扮増娑ч柣鎾跺枛閹鈻撻崹顔界亶閻熸粍婢橀崯鎾灳閿曞倸惟闁宠桨绀佺粣娑橆渻閵堝棛澧紒顔奸叄瀹曘垽顢涢悙绮规嫽婵炶揪绲块悺鏂款焽閹邦喒鍋撶憴鍕闁挎岸鏌嶇紒妯荤叆闁宠棄顦垫慨鈧柣妯垮皺閳ь剦鍘奸埞鎴︽偐缂佹ɑ閿銈嗗灥濡繈骞冮敓鐘茬妞ゆ梻鏅崢鍗炩攽閻樼粯娑ф俊顐ｎ殜椤㈡棃顢旈崟銊︽杸濡炪倖鐗楅崫搴ㄥ磻閵忋倖鐓涢悘鐐殿焾婢ф煡鏌熷畡鐗堝殗鐎规洦鍋婃俊鐑藉箛鐏炶姤澶勯柣鎾寸〒閳ь剙鍘滈崑鎾绘倵閿濆骸澧扮悮锔戒繆閵堝洤啸闁稿绋戠叅妞ゆ挾鍠嶇换鍡涙煙闂傚鍔嶉柡鍛矒閺岋綁鏁愰崨鏉款伃缂備浇灏慨銈囨崲濠靛鍋ㄩ梻鍫熺◥缁爼姊虹€癸附婢樻俊鍧楁煕閹烘挸娴鐐达耿瀹曟粍鎷呮搴ｆ喒闂傚倷绀侀幖顐ょ矙娓氣偓瀹曘垺绂掔€ｎ亞锛涢梺绋挎湰缁嬪繑绂嶅鍫熺厵闁哄鐏濋。鎶芥煟閿濆鎲鹃柡宀嬬秮楠炲洭顢楁繝鍌氼潬闂備浇顫夎ぐ鍐箟閿熺姴绠氶柡鍐ㄧ墕鎯熼梺闈涳紡鐏炶姤鏆梻鍌氬€烽悞锔锯偓绗涘懏宕查柛宀€鍊涢崶銊ヮ嚤閻庢稒锚娴滄姊虹紒妯荤叆闁告艾顑夐幃鈥斥枎閹剧补鎷哄銈嗘尪閸斿酣鎮鹃崡鐑嗙唵鐟滄梻绮婚弽顬盯宕ㄩ幖顓熸櫌婵犮垼娉涢敃銈夊煕鐎ｎ喗鈷?session 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘劙婀侀梺绋跨箰閸氬绱為幋锔界厱闁靛鍎遍埀顒€婀遍幑銏犫槈濮楀棗鏅犲銈嗘瀹曠敻鎯勬惔锝囩＜闁绘劦鍓欓崝銈夋煟韫囨梻绠為柛鈺冨仱楠炲鏁冮埀顒傚閸忚偐绡€濠电姴鍊归幑锝夋煕閺傛寧鎲哥紒杈ㄦ尰閹峰懘宕崟鎴欏灲閺岋綀绠涢弬鍨懙婵犵绱曢弫璇茬暦閻旂⒈鏁婇柛鎾楀嫭绶梻鍌欑濠€閬嶅磻閹捐绠氶悘鐐跺▏濞戙垹鐏抽柟棰佺劍鐎靛矂姊洪棃娑氬濡ょ姴鎲＄粋宥呪攽閸モ晝顔曢梺鍛婄懃椤︿粙鎮炴禒瀣厵妤犵偛鐏濋悘鈺呮煃鐟欏嫬鐏╅柍褜鍓ㄧ紞鍡涘磻閹烘嚦娑㈠礃閵娿垺鏂€闂佺粯鍔栧娆撴倶閵壯€鍋撶憴鍕闁告挾鍠庨悾宄邦煥閸曨偒鍤ら柣搴㈢⊕椤洭宕㈡禒瀣拺缂備焦蓱閻撱儵鏌熼懞銉х煉妤犵偛锕鎾偐閻㈢绱查梻浣虹帛椤牆鈻嶉弴鐐垫殾鐎光偓閸曨剛鍘藉┑鐐村灥瀹曨剙鈻嶅鍥ｅ亾濞堝灝鏋欑紒顔界懄娣囧﹪骞栨担瑙勬珕闂佸憡鎼╅悘婵嬪Χ婢跺鎷绘繛杈剧到閹诧繝宕悙鐑樼厵缂佸瀵чˉ銏⑩偓瑙勬礈閸犳牠銆侀弴銏☆€愮紓浣虹帛濞茬喎顫忔繝姘＜婵﹩鍏橀崑鎾崇暋閹冲﹤缍婂畷鎯邦檨婵炲瓨鐗犻弻鏇熺箾瑜嶇€氱兘宕戦妸銉㈡斀妞ゆ梻鐡旈悞鎯瑰搴″缂佸顦鍏煎緞鐎ｎ剙骞愰柣搴＄畭閸庤鲸顨ラ幖浣哄祦闁哄稁鍘介悡銏ゆ煕閹板吀绨婚柡瀣洴閺岋紕浠︾拠鎻掝瀳闂佸疇顫夐崹鍨暦闁秴鐓涢柛灞诲€愰崑鎾诲幢濞嗘垹锛濋梺绋挎湰濮樸劍鐗庨梻浣虹帛椤ㄥ棝銆冩繝鍐х箚闁汇垻顭堢粈瀣亜閺嶃劍鐨戞い鏂匡躬濮婅櫣鍖栭弴鐐测拤闂侀潧娲ㄩ崑鐐电博閻旂厧鍗抽柣鎰ㄦ櫆閺傗偓闂備胶绮摫闁告挻宀稿畷顖濈疀濞戞瑧鍘遍梺缁樏壕顓熸櫠閻㈢鍋撶憴鍕闁告梹鐟ラ悾鐑芥倻缁涘鏅ｉ梺缁樻煥閸㈣尙鑺遍妷锔剧瘈闁汇垽娼ф禒锕傛煕椤垵鐏︾€规洜顢婇妵鎰板箳閹捐泛甯掗梻浣规偠閸庢椽鎮橀幇顖樹汗闁圭儤鎸诲▍婊堟⒑閸涘﹣绶遍柛鐘崇墵瀵娊宕奸妷锔规嫼闂傚倸鐗冮弲婵堚偓浣冨亹缁辨帡顢欓悾灞惧櫚閻庢鍠栭…鐑藉极閹邦厼绶炴俊顖滅帛濞呭秹姊绘担铏瑰笡闁搞劑娼ц灋婵炲棙鎸搁悡婵嬫煕椤愮姴鍔滈柍閿嬪灴閹綊宕堕敐鍌氫壕闁惧浚鍋嗘禍顏嗙磽閸屾艾鈧摜绮旈幘顔芥櫇闁靛牆顦伴崑鈺呮煟閹达絾顥夌紒鐘冲哺濮婃椽顢橀妸鈺傤€嶅ù婊勭〒缁辨捇宕掑顑藉亾閻戣姤鍤勯柛鎾茶閸嬫挸顫濋鍌滎啋閻庤娲栭悥鍏间繆濮濆矈妲诲Δ鐘靛仜閻楀棝鍩為幋锔藉亹闁割煈鍋呭В鍕節濞堝灝鏋涙い鏇ㄥ弮閸┾偓妞ゆ帊绀侀崵顒勬煕濞嗗繐鏆欐い鏇秮楠炲海绮电€ｎ偅娅岄梻浣告啞濞诧箓宕滃☉銏犲偍闁归棿鐒﹂崐鐢告煕韫囨搩妲稿ù婊堢畺閺岋絾鎯旈婊呅ｉ梺鍛婃尰缁嬫牠鍩呴崹顐ょ瘈闁汇垽娼у暩濡炪倧绲肩划娆忕暦濠婂啠鏀介悗锝庝簽閻ｆ椽姊虹粙鎸庢拱缂佸鍔楅幑銏ゅ幢濞戞瑧鍘介梺纭呮彧缁查箖藟婢舵劕鑸规い鏍ㄥ嚬濞撳鏌曢崼婵囶棡闁抽攱甯炵槐鎺楊敋閸涱厾浠搁悗瑙勬礃閸ㄥ灝鐣烽妸褉鍋撳☉娅辨岸骞忔繝姘拺缂佸瀵у﹢浼存煟閻旀繂鎳愰悳濠氭煛閸愩劎澧涢柣鎾冲暟閹茬顭ㄩ崼婵堫槶濠电偞鍨崹鍦缂佹绠鹃柟瀛樼懃閻忣亪鏌涢妶鍡楀闁靛洤瀚板浠嬪Ω瑜夋慨鍥р攽閻愬弶鍣藉┑顔炬暬婵＄敻宕熼姘祮濠德板€愰崑鎾趁瑰鍫㈢暫婵﹥妞藉畷顐﹀礋椤掍焦瀚虫繝鐢靛仩鐏忔瑩宕版惔銊﹀仼闂佸灝顑囬梽鍕煕濞戞﹫宸ュù婊勭矌缁辨挻鎷呴崜鎻掑壈缂備降鍔戞禍璺虹暦閹达箑绠婚悹鍥皺閻も偓婵＄偑鍊栧濠氬箠閹惧顩插Δ锝呭暞閸嬧剝绻涢崱妤冪妞ゅ浚浜炵槐鎺楀焵椤掑嫬绀冮柕濞垮灪閺傗偓闂備胶绮崝姗€顢氬鍫㈠彆妞ゆ帒瀚悡鐔哥箾閹存繂鑸规繛鍛Ф閳ь剚顔栭崰鏇犲垝濞嗘劒绻嗛柟闂寸劍閺呮繈鏌嶈閸撶喖銆侀幘璇茬缂備焦菤閹锋椽姊洪崨濠勨槈闁挎洩绠撻崺銉﹀緞鐎ｃ劋绨诲銈嗘寙閸愵煈娼婚梺鎹愬吹閸嬨倝寮婚敐澶婃闁割煈鍠楅崐顖炴⒑閹惰姤鏁遍悽顖涘浮濠€渚€姊洪幐搴ｇ畵婵炲眰鍊濆畷婵堚偓锝庡枟閻撴洟鏌曢崼婵嗏偓鍛婄妤ｅ啯鈷掗柛灞剧懅閸斿秹鏌熼鑲╁煟鐎规洜鎳撶叅妞ゅ繐鎳庢禍妤呮⒑鐠恒劌鏋斿┑顔碱嚟缁顫濋幍浣镐壕闁稿繐顦禍楣冩⒑閸涘﹥澶勯柛瀣躬瀹曨剟宕奸弴鐔叉嫼闂傚倸鐗婃笟妤€危閸洘鐓曢幖娣灩閳绘洜鈧鍠栭…宄扮暦婵傜唯闁挎梹鍎抽獮宥夋⒒娴ｅ搫甯舵繝鈧潏銊︽珷婵°倕瀚ㄦ禍鍦喐韫囨洘顫曢柟鐑橆殕閸ゆ垿鏌ら崫銉︽毄闁靛棗锕ョ换婵嬪閿濆孩缍堝┑鐐插级椤洨鍒掔€ｎ喖绠抽柡鍌氭惈娴滈箖鏌ㄥ┑鍡涱€楀ù婊呭仱閺屾稑顫滈埀顒佺鐠轰警娼栧Δ锕侊骏娴滃綊鏌熼悜妯虹仯闁哥姴锕娲川婵炴帩浜俊鍓佺矙鐠恒劍娈惧┑鐘绘涧椤戝懎娲块梻浣虹《閸撴繈鎮疯椤㈡瑩寮撮姀鈾€鎷洪梺鍛婄☉閿曪絿娆㈤柆宥嗙厱闁绘ê鍟块崫鐑橆殽閻愯韬€规洏鍔戦、娑橆煥閸滃啰搴婇梻浣告惈椤︻垶鎮ч崟顖氱鐎光偓閳ь剛鍒掗鐑嗘僵妞ゆ挾濮烽鏇㈡⒑缂佹ɑ灏繛鎾棑缁崵鎷犲ù瀣杸闂佹枼鏅涢崯浼村箺閸愨斂浜滈柨鏃囧Г鐏忥箓鏌″畝瀣М濠碘€崇埣瀹曘劑顢樿閹綁姊绘担鍝ユ瀮妞ゎ偄顦靛畷褰掑锤濡炲皷鍋撴担绯曟瀻闁瑰鍋涢崢褰掓倵閻熸澘顏い锝忓濡叉劙鏁愭径瀣ф嫽婵炴挻鍩冮崑鎾寸箾娴ｅ啿娲﹂崑瀣叓閸ャ劎鈻撻柍褜鍓ㄧ粻鎾崇暦婵傜唯闁挎洍鍋撳ù鐘层偢濮婅櫣绱掑Ο铏逛紘婵犳鍠撻崐婵嗩嚕閺屻儱钃熼柕澶涘閸橀亶姊洪弬銉︽珔闁哥姵鑹鹃埢鎾澄熼懖鈺冿紲闂佸綊鍋婇崰鏍ㄧ濠婂牊鐓冮悷娆忓閻忔挳鏌涢埞鍨姦鐎规洖宕灃闁告剬鍐╂啟闂傚倸鍊风粈浣革耿闁秴鐓曢柛顐犲劚绾捐鈹戦悩鍙夊櫤闁挎稑鍊圭换婵嬫偨闂堟稈鏋呭┑鐐板尃閸ヨ埖鏅為梺绯曞墲缁嬫垿鎮￠弴銏＄厵閺夊牓绠栧顕€鏌ｉ幘璺盒㈤柍瑙勫灴椤㈡瑩鎸婃径濠庢綋闂備線鈧偛鑻晶顖涚箾閸欏鑰块柟顔ㄥ嫭鍎熼柍閿亾闁衡偓娴犲鐓熼柟閭﹀灠閻ㄦ椽鏌熼悾灞叫ョ紒杈ㄥ浮椤㈡瑩鎳栭埡浣插彙闂備礁鎲＄敮妤冩暜閹烘鐓濋幖娣€楅悿鈧梺鍝勬川閸犳劙顢欓弴鐔虹瘈闁汇垽娼ч埢鍫熺箾娴ｅ啿鍘惧ú顏呮櫇闁稿本姘ㄩ敍鐔兼⒑缂佹ɑ鐓ラ柛姘儔閹ょ疀閹绢垱鏂€闂佺粯鍔樼亸娆愮閵忋倖鐓曢柡鍐ｅ亾缂侇喗鎹囧濠氭晲閸℃ê鍔呭銈嗘⒒閸樠囧煕鐎ｎ亖鏀芥い鏃傘€嬮弨缁樹繆閻愯埖顥夐摶鐐烘煕閹扳晛濡锋俊鎻掔墛閹便劌顫滈崱妤€鈷掗梺鍝勬－閸嬪懏绌辨繝鍥ㄥ€锋い蹇撳閸嬫捁顦撮柍褜鍓熷褔鎯岄崒姘辨殾闁荤喐澹嗛弳锕傛煕閵夋垵鍟版禍鏉库攽閻樺灚鏆╁┑顔炬暩閸犲﹤顓兼径濠勬煣闂佸湱绮敮鈺呮偄閸℃稒鍋ｉ弶鐐村椤掔喖鏌涙惔銏犵伌闁哄本绋戦埢搴ょ疀閹垮啩鎮ｉ梻浣哥枃椤宕归崸妞尖偓浣糕枎閹捐櫕顥濋梺闈涚墕鐎氼喗娼婚弮鍫熲拻濞达絽鎲￠崯鐐烘煙缁嬫鐓奸柟顔惧厴閸╋繝宕ㄩ鐔感氶梻渚€鈧偛鑻晶顖炴煏閸パ冾伃妤犵偞甯￠獮瀣攽閸ヮ煈鍞查梻鍌欑閹诧繝鎳濋幆褝鑰块弶鍫氭櫆椤洟鏌熼悜姗嗘闁轰礁顑夐弻娑㈠焺閸愵亝鍣ч梺绯曟櫅閻倿寮昏缁辨帒螣閻撳簶鎷伴梻浣告惈閺堫剛绮欓幋锕€鐓″鑸靛姇绾偓闂佺粯鍔曢顓熺椤忓嫧鏀介柣鎰▕閸ょ喎鈹戦锝呭籍鐎规洖婀遍幑鍕瑹椤栨稓绋佹繝鐢靛仜濡﹥绂嶅鍛笉鐟滅増甯楅悡鐔兼煙闁箑骞栫紒鑼额嚙閳规垿顢欓懖鈺冨姱濠殿喖锕ュ钘夌暦閵婏妇绡€闁告劦浜滃铏節閻㈤潧浠﹂柟鍛婂▕钘熼柟鎹愵嚙缁€鍡涙煙閻戞ɑ鈷愰悗姘哺閺屾稑鈻庤箛鎾亾婵犳艾绐楅幖娣灪濞呭繘姊绘担瑙勫仩闁稿海绮穱濠囧炊椤掍礁鍓柟鍏兼儗濠⑩偓缂佽妫濋弻锝夊箛閸忓摜鐩庨梺閫炲苯澧柣妤冨Т閻ｅ嘲鈹戠€ｎ偅娅囬梺绋挎湰濮樸劑鏁嶅☉銏♀拺闁告稑锕ユ径鍕煕閹惧娲撮柕鍡楁嚇瀵濡烽敂鎯у箺婵犲痉鏉库偓鎰板磻閹剧粯鐓熸俊銈勮兌閻﹪鏌嶇紒妯诲鞍缂佸倹甯為埀顒婄秵閸嬪棝宕㈡禒瀣拺鐟滅増甯掓禍浼存煕閵娿劎绨块柕鍥ㄥ姌椤﹀綊鏌＄仦璇测偓婵嗩嚕閸洖绠伴幖绮光偓鍙夋▕闂傚倷绀侀幖顐︽儔婵傜绐楅柡宥庡幑閳ь兛绀侀埢搴ㄥ箻閺夋垳鎮ｉ梺璇茬箳閸嬬偤宕曢崘娴嬫瀺婵﹩鍘虹换鍡涙煟閹板吀绨婚柍褜鍓氶崹鍨暦瑜版帒浼犻柛鏇炵仛缂嶅酣鎮峰鍛暭閻㈩垱甯炵划濠氼敍閻愬鍘卞┑鐐村灦椤洭骞楅悩缁樼厓闂佸灝顑呮慨宥夋煛瀹€鈧崰鏍蓟閸ヮ剚鏅濋柍褜鍓氶弲鍓佺磼濡湱绠?
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氶梺璇叉唉椤煤韫囨稑纾块柟鎯版閻掑灚銇勯幒鎴姛缂佸鏁婚弻娑㈡偐閹颁焦鐤侀悗瑙勬礃濞茬喖鐛€ｎ喗鍋愰柛鎰级閻庨箖姊绘担铏瑰笡妞ゃ劌鎳橀幃褔宕卞▎鎴犵暥闂佸湱鍎ゅ璇参涢鐐寸厵妞ゆ牕妫楃€氼剚绂掗幘顔藉€垫繛鍫濈仢閺嬫盯鏌ｉ弽顐㈠付闁伙絿鍏橀幃鈩冩償椤旇棄鍏婃俊鐐€栭幐楣冨窗閹惧箍浜归柛鈩兠肩换鍡涙煟閹板吀绨婚柍褜鍏欓崐婵嗙暦閹达箑宸濇い鎺戝悑濡炰粙骞冮姀銈嗗亗閹兼番鍊栭幉浼存⒒娴ｈ櫣甯涢柡灞诲姂楠炲鏁嶉崟顓狀槸闂佹悶鍎洪崜姘舵偂閺囥垺鐓欓柣鎰靛墻濞堟棃鏌熼崘鑼闁崇粯鎹囬獮鏍ㄦ媴閸忓瀚?ForceCodexCLI闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴︽煙閼测晛浠滈柍褜鍓氶悧鐘茬暦濠靛鍐€妞ゆ挾鍊ｉ敃鍌涚厱闁哄洢鍔岄悘鐘绘煕閹般劌浜惧┑锛勫亼閸婃牠宕濋敃鈧…鍧楀焵椤掑倻纾兼い鏃囧亹閸╋絾鎱ㄦ繝鍐┿仢妤犵偞鐗犻幃娆撳箵閹烘繄鈧娊姊绘担渚劸闁挎洏鍊楃槐鐐寸節閸愌呯畾闂佹眹鍨婚…鍫㈢矆鐎ｎ偁浜滈柡宥冨姀婢规﹢鏌涢悙顏勫婵﹥妞藉Λ鍐ㄢ槈鏉堚晛濮奸梻浣侯焾妤犵鐣烽鍐炬毎闂備礁澹婇崑渚€宕曢弻銉ョ厱闁硅揪闄勯悡娆撴煠濞村娅呭ù鐘崇矒閺屽秷顧侀柛鎾村哺閹囧即閻樻彃鐤鹃梻鍌欒兌缁垰顫忔繝姘偍鐟滄棃骞忛幋锔藉亜闁告縿鍎抽鏇㈡⒑閻熼偊鍤熼柛濠冨灥椤繄鎹勯搹鐟版憢濠电偛顕慨鎾敄閸℃稒鍋傞柡鍥ュ灪閻撳啴鏌嶆潪鎵槮闁哄鍊栫换娑㈠醇濞戞浠奸梻鍥ь樀閺岋絽螣閾忚鍕鹃梺绋款儍閸ㄦ椽骞堥妸锔剧瘈闁告侗鍣禒鈺冪磽娴ｄ粙鍝洪悽顖涘笩閻忔帡鏌ｉ悩鍙夊闁绘搫绻濋幃妤€煤椤忓應鎷绘繛杈剧秬椤宕戦悩缁樼厱闁哄倽鍎荤€氱増銇勯鐐村枠闁轰焦鎹囬幃鈺呮濞戞哎鍋婇梻鍌欑閹诧繝宕濋幋锕€绀堟い鏇楀亾闁诡喓鍨介幃婊兾熼搹鐧哥础闂傚倷绶氬褑鍣归梺鍛婃处閸撴稓鏁崱娑欌拻闁稿本鑹鹃埀顒勵棑缁牊鎷呴棃鈺勨偓鍧楁⒑椤掆偓缁夊澹曟繝姘厵闁硅鍔﹂崵娆戠磼閻樺啿鍝洪柟顔斤耿閹瑧鎹勭悰鈩冪€兼繝鐢靛仦閸ㄦ儼褰滃┑鈩冨絻閻楀﹪骞堥妸銉庣喐寰勭粙鎸庡創闂備焦瀵х粙鎺旀崲閸愵亝宕叉繛鎴炵懄缂嶅洭鏌涢幘妤€鎲涘顒夋富闁靛牆妫欑粈鍐煟濡や焦绀嬮柛鈹垮灲楠炴鎷犻懠顒傛綁闂備礁澹婇崑鍡涘窗閹捐鍌ㄩ弶鍫涘妿缁♀偓闂佹眹鍨藉褍鐡梻浣告憸閸ｃ儵宕圭捄铏规殾闁圭増婢樼粻娑欍亜閹达絽袚闁哄倵鍋?User-Agent 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闂佸憡顨堥崑鐔哥婵傚憡鈷戠紓浣股戠亸鐗堢箾閼碱剙鏋涚€殿喛顕ч埥澶愬閳哄倹娅囬梻渚€娼ч悧鍡涘箠韫囨稒鍊跺ù锝呭濞撳鏌曢崼婵嗏偓鐟扳枍閸℃稒鐓熼柟鍨缁♀偓閻庢鍠栭…閿嬩繆閹间礁唯闁靛繆鍓濋弶鎼佹⒒娴ｈ櫣銆婃俊鎻掓嚇瀹曘垽宕滆閻棝鏌涚仦鍓с€掔紒鈾€鍋撻梻渚€娼ф蹇曞緤閸撗勫厹闁绘劦鍏欐禍婊堟煙鐎涙绠栫€瑰憡绻勯埀顒侇問閸燁偊宕惰閸旓箑顪冮妶鍡楃瑨閻庢凹鍙冮崺娑㈠箣閿旂晫鍘卞┑鐐村灦閿曨偄顔忛妷銉㈡斀妞ゆ洖妫涢悾閬嶆婢跺绡€濠电姴鍊搁顐㈩熆瑜忛弫鎼佸焵?Codex CLI闂?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵潙螖閳ь剚绂嶉幆褜娓婚柕鍫濈凹缁ㄥ鏌涢悢鍛婄稇闁伙絿鏌夐妵鎰板箳濠靛洦娅旈梻浣告啞娓氭宕归悧鍫熷弿婵炲樊浜濋埛鎺懨归敐鍕劅婵炲吋甯￠弻娑㈠即閻愬吀绮甸梺浼欑到婢у海妲愰幘瀛樺闁圭粯甯婃竟鏇炩攽閻橆喖鐏遍柛鈺傜墵瀹曟繈寮介鐐碉紮闂佹眹鍨归幉锟犳偂閵夛妇绡€闂傚牊绋掗ˉ銏°亜鎼淬埄娈旀い顓″劵椤﹀磭鎲搁弶鍨殻闁糕晝鍋ら獮瀣晜缂佹ɑ娅撻梻浣告贡椤牆霉閻ゎ垬浜圭憸鏂款潖濞差亝顥堟繛鎴炵懄閹瑩鏌ｆ惔銏㈢叝闁告濞婇幃浼搭敋閳ь剟鐛幒鎳虫棃鍩€椤掑倻涓嶉柨婵嗘缁♀偓闂傚倸鐗婄粙鎴﹀汲濞嗗緷鐟邦煥閸垻鏆梺鍝勮閸婃牠骞堥妸鈺佺疀妞ゆ垼妫勬禍楣冩煛閸ワ絾鍤嶅璺侯煬濞尖晜銇勯幋鏃€娅嗘俊顐㈠暣瀵偊骞囬弶鍨獩濡炪倖鎸鹃崰鎰邦敊婢舵劖鈷掑ù锝堟鐢盯鏌熺粙娆剧吋闁诡喚鍏橀崺锟犲川椤撶姷鏆繝寰锋澘鈧劙宕戦幘缁樼厓闁芥ê顦藉Σ鎼佹煃鐠囨煡顎楅摶锝夋煟閹炬娊顎楀Δ鏃傜磽閸屾艾鈧悂宕愰悜鑺ュ殑闁肩鐏氶崣蹇涙煟閵忋埄鐒剧紒鐘虫緲铻栭柨婵嗘噹閺嗘瑧绱掗悩闈涙灈闁哄瞼鍠栧鑽も偓闈涘濡差喚绱撴担鍝勑為柛搴㈢叀婵＄敻宕熼姘棟闁荤姵浜介崝宥夋儌娓氣偓濮婃椽宕崟闈涘壈闂佺娅曢敋妞ゎ偄绻愮叅妞ゅ繐瀚槐鍫曟⒑閸涘﹥澶勯柛姗€绠栧畷婵嬪捶椤撶姷锛濇繛杈剧秬濞咃絿鏁☉姘辩＜閻犲洦褰冮埀顒€娼￠獮鍐┿偅閸愨晝鍙嗛柣搴秵閸撶喎顬婇鈧娲川婵犲啫纰嶉悗娈垮枛閻栫厧顕ｉ幓鎺嗘斀閻庯綆鍋€閹锋椽鏌ｉ悩鍏呰埅闁告柨鐭傚鎼佸棘鎼存挻鏂€濡炪倖妫侀崑鎰板箺閻樼數纾奸柛灞剧☉濞搭喗顨ラ悙鍙夊枠闁诡啫鍥ч唶婵﹩鍘奸鐑樼節閻㈤潧浠﹂柟绋款煼瀹曟椽宕橀鑲╋紱闂佺懓澧界划顖炴偂閸愵亝鍠愭繝濠傜墕缁€鍫熺箾閹存瑥鐏╃痪鍓х帛缁绘盯骞嬪▎蹇曚痪闂佹悶鍊栭崝鏍Φ閸曨垰鍐€闁靛ě鈧慨鍥р攽閻愬弶鍣藉┑顔炬暬婵＄敻宕熼姘棟闂佸壊鐓堥崰鎺楀箰閸愵亞纾奸柣鎰靛墮閸斻倝鏌曢崼鐔稿€愭鐐插暢閵囨劙骞掗幋鐘测偓鐐烘偡濠婂啴鍙勯柛鈹垮灲瀵挳濮€閿涘嫬骞堥梺鐟板悑閻ｎ亪宕硅ぐ鎺撳€堕柨鏇楀亾閼?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳婀遍埀顒傛嚀鐎氼參宕崇壕瀣ㄤ汗闁圭儤鍨归崐鐐差渻閵堝棗鍧婇柛瀣尰濞艰鈹戠€ｎ偀鎷洪梻渚囧亞閸嬫盯鎳熼娑欐珷闁圭虎鍠楅悡娑㈡倶閻愭彃鈷旈柕鍡樺浮閺屽秷顧侀柛鎾卞妿缁辩偤宕卞☉妯硷紱闂佸憡渚楅崢楣冨汲閿旈敮鍋撻崗澶婁壕闂佸憡娲﹂崜娑㈠储閹间焦鍊甸柛蹇擃槸娴滈箖鏌ｆ惔顖滅У闁告挻绋栭埅鐢告⒒閸屾瑦绁版い鏇熺墵瀹曚即骞樼拠鎻掔€梺鑺ッˇ閬嶅汲閿曞倹鐓忓┑鐘茬箰椤╊剛绱掗埦鈧崑鎾绘⒒娴ｅ憡鍟炴繛璇х畵瀹曟垿宕熼鐔哥亖闂侀潧顦弲婊堝煕?User-Agent 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鎾煙椤斿吋鍋ユい銏″哺閸┾偓妞ゆ帒瀚拑鐔兼煟閺冨倸甯剁紒鈧崼婢濆綊鏁愰崶褍濡洪梺鐟板级閿曘垹顫忛悜妯诲闁规鍣Σ顕€姊洪幐搴㈠濞存粠浜滈锝嗙節濮橆厼浜滈梺绋跨箺閸嬫劙宕濋悜鑺モ拺闁圭瀛╃壕鐢告煕鐎ｎ偅宕岄柡宀嬬磿閳ь剨缍嗛崜娆撳几閹达附鐓忛柛銉戝喚浼冨銈冨灪濞茬喖寮幇鏉垮耿婵炲棙蓱琚ㄧ紓鍌氬€搁崐鐑芥嚄閸撲礁鍨濇い鏍ㄧ矋瀹曟煡鏌涢锝嗗剷婵炴垯鍨圭粈鍐煏婵炲灝鐏い顐㈢Т閳规垿鎮欓崣澶樻缂備胶绮敮锟犲箖娴兼潙鐓涢柛灞剧懅缁犳岸姊虹紒妯哄Е濞存粍绮撻崺鈧い鎺嶇婵秶鈧鍠楅悡锟犲箖閳哄懎绀勯柡澶嬵儥濡偓濡ょ姷鍋為…鍥箯閻樼粯鍤戞い鎺嗗亾濠殿噯闄勬穱濠囨倷椤忓嫧鍋撻弽褜鍟呭┑鐘宠壘绾惧鏌熼幆褍顣崇痪鎯с偢閺岋絽螣閸喚姣㈤柡浣哥墦閹鎲撮崟顒傤槰濠电偠灏欓崰鏍ь嚕婵犳艾骞㈡俊銈咃工閹垿姊虹化鏇炲⒉闁靛洨鍋熺槐鏃堝即閵忊檧鎷绘繛杈剧秬濞咃綁濡存繝鍥ㄧ厱闁规儳顕粻妯肩磼椤旂晫鎳囨鐐差儔閺佸啴鍩€椤掑倸顥氬┑鍌氭啞閻撳繐鈹戦悙闈涗壕婵炲懎妫濋弻锝夊箻鐎靛憡鍣紓浣介哺閹稿骞忛崨瀛橆棃婵炴垶鐭幃锝夋⒒娴ｅ憡鎯堟い鎴濇喘瀹曚即寮介鐐舵憰闂佹寧绻傞ˇ顖滅不婵犳碍鍊垫繛鎴炵懐閻掕姤銇勯敂鍝勫缂佽鲸鎸婚幏鍛存惞閻熸壆顐奸梻浣哄劦閺呪晠宕归崼鏇熷仒妞ゆ棃鏁崑鎾绘晲鎼粹剝鐏嶉梺绋匡功閸忔﹢寮诲☉鈶┾偓锕傚箣濠靛懐鐩庢繝鐢靛仜閹冲繘宕归崹顕呮綎缂備焦蓱婵绱掑☉姗嗗剰婵炲牜鍘剧槐鎾存媴閾忕懓绗￠梺鍛婃⒐閻熲晛顕ｉ锕€绀冩い鏇炴噺閺呮粓姊洪崜鎻掍簼缂佽妫濋獮?Codex 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾剁磽娴ｅ搫校婵犮垺锕㈤崺鐐哄箣閿旇棄浜归悗瑙勬礀濞村倿寮抽敓鐘斥拺缂佸娼￠妤冪磼婢跺本鏆╅柟骞垮灩閳规垿宕堕埡鍐闂備胶顭堥張顒傚垝瀹€鍕┾偓鍌炲传閸曞孩妫冮幃鈺呮濞戞婢€闂備焦鎮堕崝蹇撯枍閿濆绠查柕蹇曞Л濡插牓鏌曡箛鏇炐㈤柤鏉跨仢閳规垿鍩ラ崱妤冧哗闂佸湱鈷堥崑澶惵烽崒姘ｆ斀闁绘ê鐏氶弳鈺呮煕鐎ｎ剙鏋涙い銏＄墵瀵挳濮€閻樼绱甸梻鍌氬€搁悧濠冪瑹濡も偓椤洭寮介銈囷紳婵炶揪缍€閸嬪倿骞嬮悙鎻掔亖闂佸湱铏庨崰妤呮偂閺囩喓绠鹃柟瀵稿剳閸忣剛鈧鎮堕崕鐢稿蓟閿熺姴骞㈡俊銈呭暙閳敻姊洪崫鍕効缂傚秳鐒﹂幈銊╁焵椤掑嫭鐓冮弶鐐村閸忓苯霉閻樺啿鍝烘慨濠冩そ瀹曨偊宕熼鈧▍褎绻濆▓鍨灍闁瑰憡濞婇獮鍐┿偅閸愨晛鈧鏌﹀Ο渚Ш妞ゆ挻妞藉铏圭磼濡搫顫庨梺杞扮劍閹倿骞嗘笟鈧畷濂稿Ψ閿旇瀚奸梻浣侯攰閸嬫劙宕戝☉銏犵闁逞屽墴濮婃椽宕ㄦ繝浣虹箒闂佸憡顭堟竟鍫ヂ烽崒姘ｆ斀闁绘ɑ顔栭弳婊呯磼鏉堛劍绀嬬€规洘鍨块獮瀣晝閳ь剛澹曡ぐ鎺撶厸鐎广儱楠告禍婵嬫煛閸℃鐭婇柍瑙勫灴閹晠宕ｆ径濠庢П闂備胶顭堥敃銈咁焽閿熺姴钃熼柕濞炬櫅閸楁娊鏌ｉ幇顖氱处闁哥姴锕︾槐鎾诲磼濮樻瘷锝夋煟濡や緡娈曠紒宀冮哺缁绘繈宕堕‖顑洦鐓曟繛鎴濆船楠炴绻涢崼婵嗩暢缂佽鲸甯￠獮鍡氼槻闁抽攱妫冮弻娑樜熼悩鍙夘棤濞存粓浜跺缁樻媴閾忕懓绗″銈冨妼閹虫﹢骞冭缁犳盯寮撮悩杈╃憹闂備浇顫夊畷妯肩矓椤曗偓閹兘鏌囬敂鎯у汲闂備礁鎲″ú锕傚礈濞戙垹绀勯柨鐔哄У閳?
	if s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		req.Header.Set("user-agent", codexCLIUserAgent)
	}

	// 婵犵數鍋炲娆戞崲濡ゅ拑缍栫€广儱顦梻顖炴煏婵炲灝鍔ら柍?UA 闂備胶顭堢换鎴犲垝瀹€鈧懞閬嶅箮閼恒儲娅栧┑顔斤供閸嬪棝鎯?OAuth闂備焦瀵х粙鎴濓耿閹辩嘲tGPT 闂備礁鎲￠崝鏇㈠箠濮椻偓瀹曟洟骞橀鑲╊唺闂侀潧顦崕杈╃礊閹剧粯鐓ユ繛鎴烇供濡茬厧顭跨捄鍝勵伃鐎规洩绲鹃ˇ鐗堟償閵忊剝顏熼梻浣芥〃缁€渚€鎮ч悙鍨潟婵犻潧顑嗛崵鎰版煏婢舵盯妾ù鐘愁焽缁?user-agent 濠电偛顕慨瀵哥矓瀹曞洦瀚婚柣鏃囶問绾懐鐤€婵ê鍚嬬紞宀勬⒑?	// 闂備焦瀵х粙鎴濓耿閹辩磧ome/Firefox/Safari/Edge 缂傚倷鐒︾粙鎴βㄩ埀顒傜磼鏉堛劌鍝洪柡浣哥Ф娴狅箓鎮欓鍌ゆ捶闂備胶顢婇鏍ь熆閳ь剟鎮归幇顔兼灈鐎规洏鍎查幆鏃堝灳閹惰棄褰欓梻鍌欑劍濠㈡绮旈崼鏇熷仾闁糕剝绋掗崕?Codex UA闂備焦瀵х粙鎴︽儗閸屾稑顕遍柍鍝勬噹缁€?Cloudflare 闂佽崵鍠愰悷杈╃不閹达絻浜?JS 闂佽崵濮甸崝鎴﹀礉韫囨侗鏁嬫俊銈呮噹杩?	s.overrideBrowserUserAgent(ctx, account, req)

	// Ensure required headers exist
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}

	return req, nil
}

// overrideBrowserUserAgent 婵犵妲呴崑鈧柛瀣崌閺岋紕浠︾拠鎻掑Г濡炪倖娲﹂崜娆愭櫏闂佺鏈划宥夋偩闁秵鐓涢柛顐ｈ壘娴滃墽绱?user-agent闂備焦瀵х粙鎴︽儗娴ｈ　鍋撳銉ョ仾缂佸顦甸幃婊兾熺粭鍝勪桓闂佽崵鍠愰悷銉╂偋閸℃せ鏋?UA 闂備礁鎲＄敮妤呮偡閿曗偓闇夋繛鎴欏灩缁犲弶銇勯顐㈠绩缂佲偓鐎ｎ喗鐓曟慨姗嗗墴椤庢鎲搁悧鍫熷唉闁哄苯锕ら濂稿炊閳哄倻鈧參姊?Codex UA闂?// 闂備焦妞垮鍧楀礉鐎ｎ剝濮虫い鎺戝€归崰鍡涙煕閺囥劌鐏熺紓?Cloudflare 闂佽绨肩徊濠氾綖婢跺瞼鐭堥柤濮愬€栭崰鍡涙煕椤愶絿绠栭柣婵勫€濋弻?UA 闂?ChatGPT 闂備礁鎲￠崝鏇㈠箠濮椻偓瀹曟洟骞橀鑲╊唺闂侀潧顦崕杈╃礊閹炬枼妲堥柟鍓ь劜閸旂喐绻涢崼鐔风仼闁瑰嘲顑夊畷绋课旀笟濠勬そ闂佽崵濮甸崝鎴﹀礉韫囨侗鏁嬫俊銈呮噹杩?// 闁荤喐绮嶅妯虹暦椤掑嫬绠归柣鎴ｅГ閸ゆ垿鏌涢幇鈺佸缂佺虎鍨伴埥澶愬箻閸楃偛濮曢梺缁橆殕閹倸顫忔禒瀣疀妞ゆ帒鍊瑰В搴ㄦ⒑閹稿海鈯曢柣顓у枤閸?OAuth闂備焦瀵х粙鎴濓耿閹插儰ex/ChatGPT 闂備礁鎲￠崝鏇㈠箠濮椻偓瀹曟洟骞橀鑲╊唺闂侀潧顦崕杈╃礊閹剧粯鐓ユ繛鎴烇供濡茬厧顭跨捄鍝勵伃鐎规洩绲鹃ˇ鐗堟償閵忊剝顏熼梻浣芥〃缁€渚€鎮ч悙鍨潟闁烩晛妲 Key 缂傚倷鐒︾粙鎴λ囬鐐茬煑闁绘劦鍓涢々鐑芥煛瀹ュ骸澧い锔规櫊閺屾稑鈻庤箛鎾葱╅柣銏╁灱閸嬪﹤顕ｆ禒瀣€婚悷娆欑到娴滈箖鐓崶褎鎹ｉ柣鎰躬閺?// 濠电偛顕慨鎾箠鎼粹槄鑰挎い蹇撴鐎氭岸鏌涢埄鍐炬當闁绘帞鍘ч埥澶愬箻鐠哄搫濡虹紓浣锋祰瀹曠敻骞夐幘顔奸唶婵犻潧娲ㄨぐ婊堟⒑閹稿海鈽夋繝鈧·绁昳lla/...闂備焦瀵х粙鎴λ囬弶娆炬富闁稿瞼鍋涚紒鈺呮煟閻斿憡绶查柡鍛倐閺屻劌鈽夊Ο鍨伃闂佸憡鐟ч崑鎾剁矉?CLI/闁诲氦顫夐幃鍫曞磿闁秴鐭?UA 濠电偞鍨堕幐鍝ョ矓鐎涙ɑ鍙忛柍鍝勬噹杩?
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
	requestedModel ...string,
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
	var reqModel string
	if len(requestedModel) > 0 {
		reqModel = strings.TrimSpace(requestedModel[0])
	}
	if reqModel == "" {
		reqModel, _, _ = extractOpenAIRequestMetaFromBody(requestBody)
	}
	shouldDisable := s.handleOpenAIAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, body, reqModel)
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
			RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
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
	requestedModel ...string,
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
	var modelForCooldown string
	if len(requestedModel) > 0 {
		modelForCooldown = requestedModel[0]
	}
	shouldDisable := s.handleOpenAIAccountUpstreamError(
		c.Request.Context(), account, resp.StatusCode, resp.Header, body, modelForCooldown,
	)
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
			RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
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
	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€跨€殿喖顭烽幃銏ゆ惞閸︻叏绱查梻渚€娼х换鎺撴叏閻㈠憡鍊堕柛顐犲灮绾捐棄霉閿濆懏鎯堥崯鍛婄節閻㈤潧浜归柛瀣崌濮婃椽宕崟顓犱紘闂佸摜濮甸悧鐘差嚕婵犳艾鐐婃い鎺嶇劍濞呭洭姊洪柅鐐茶嫰婢у鈧娲樼换鍫ョ嵁閹烘绠婚柛鎾茬贰閸熷酣鏌ｆ惔锝嗗殌濠㈢懓锕畷浼村即閻樺搫小闂佺厧顫曢崐妤冨婵傚憡鐓冪憸婊堝礈閻旈鏆︽慨妞诲亾闁糕晝鍋炲鍕传閵壯勬櫒闂傚倸鍊风粈渚€骞栭锕€鐤い鎰ㄦ寣濞差亜围闁糕剝鐟ú鎼佹煟閻斿摜鎳冮悗姘ュ妽缁傚秴顭ㄩ崼鐔哄幐闂佺鏈懝楣冩儍閿熺姵顥婃い鎺戭槸婢ф挳鏌″畝鈧崰鏍€佸▎鎾村亗閹肩补鎳ｉ埡渚囨富闁靛牆楠搁獮鎴︽煕閺冣偓閸ㄧ敻鎮惧畡閭︽建闁逞屽墴閵嗕線寮崼婵嗚€块梺鍝勬川婵兘藝閵娧呯＝闁稿本鑹鹃埀顒佹倐瀹曟劖顦版惔銏╁仺闂佽法鍠撴慨鎾⒒椤栨稏浜滈柡鍥殔娴滈箖鎮楃憴鍕┛缂傚秳绀侀悾宄邦潨閳ь剚淇婂宀婃Ъ濠电偛鐗撶粻鏍ь潖濞差亜鎹舵い鎾跺仜婵℃椽姊虹化鏇熸珔闁绘妫濋垾锕傚炊椤掆偓閻撴盯鏌涘☉鍗炴灓闁告﹩浜濈换婵嬪閿濆棛銆愰梺缁橆殔濡繈骞冨鈧崺鈩冨鏉堚晜鍤€妞ゎ厹鍔戝畷濂稿閵忊剝鐦掗梻鍌欑閹碱偊鎯夋總绋跨獥闁哄稁鍘肩粻鏍ㄧ節婵犲倸鏋ゆ繛绗哄姂閺屾盯鍩勯崘鍓у姺闂佸搫妫庨崕鐢稿蓟閿濆棙鍎熼柨婵嗘濞堝矂姊虹涵鍜佸殝缂佽鲸娲熼獮鎴﹀閵堝懐锛滈梺缁樺姌鐏忔瑩宕㈤幘顔解拺缁绢厼鎳忚ぐ褔姊婚崟顐㈩伃鐎规洘鍨块獮鍥偋閸垹骞嶇紓鍌氬€烽悞锕傛晪缂備焦顨嗙喊宥囨崲濞戙垹閱囬柣鏃堫棑缁佸嘲顪冮妶搴濈盎闁哥喎鐡ㄦ穱濠囧醇閺囩偛鑰垮┑鈽嗗灣閸樠冾嚕瀹曞洨纾介柛灞剧懅椤︼附銇勯敃鍌欐喚妤犵偛锕獮妯诲濞嗘垹鈼ら梻濠庡亜濞诧妇绮欓幒妤€绠氶柣鎰劋閸嬧剝绻涢崱妤冪妞ゅ繆鏅犻弻娑樜熼搹鍦ㄥ┑顔硷攻濡炶棄鐣烽妸锔剧瘈闁告洦鍘鹃崝鍓佺磽閸屾瑨顔夐柛瀣尭闇夐柨婵嗘川閵嗗﹪鏌＄€ｎ偅顥堥柡宀€鍠栭獮宥夘敊绾拌鲸姣夐梻浣瑰▕閺€閬嶅垂閸噮娼栧┑鐘宠壘闁卞洦鎱ㄥ鍡楀箺闁绘繃鐗滅槐鎾存媴閸撳弶笑婵犫拃鍐╂崳婵″弶鍔欓獮鎺懳旈埀顒傜不閻㈠憡鐓欓柛婵嗗椤ユ粌霉濠婂牏鐣洪柡宀嬬秮瀵噣宕堕妸锝傚亾閸愵亞纾介柛鎰ㄦ櫆缁€瀣叏婵犲偆鐓肩€规洘甯掗埢搴ㄥ箳閹存繂鑵愬┑锛勫亼閸娿倝宕戦崟顖ｆ晞濠㈣泛锕﹂弳锕傛煟閺冨倸甯堕柛銊ュ€归妵鍕箛閳轰讲鍋撻弽褉鏋斿Δ锝呭暞閳锋帡鏌涚仦鍓ф噮闁告柨绉撮埞鎴︽倷鐠囇嗗惈閻庢鍠撻崝鎴濈暦閿熺姵鍊剁紓浣股戦妵婵嬫煙椤斿搫鐏查柟顔瑰墲閹棃濡堕崱妯烘毇闂傚倸鍊峰ù鍥р枖閺囥垹绐楅柟鎯х摠閸欏繘鎮楅棃娑欐喐闁活厼妫濋弻娑㈠箛闂堟稒鐏嶉梺缁樻尭缁绘劙鍩為幋锔藉亹闁肩⒈鍓涢濠囨⒑娴兼瑧绉ù婊庡墰濡叉劙骞掑Δ鈧粻娑欍亜閹捐泛校闁告帗鐩铏规嫚閳ヨ櫕鐏嶅銈冨妼閹虫﹢宕洪妷锕€绶炲┑鐐靛亾閻庡姊洪悷閭﹀殶濠殿喚鍏樺鍫曟嚍閵壯呯槇濠电偛鐗嗛悘婵嬪几濞戙垺鐓ラ柡鍥崝姘亜椤忓嫬鏆ｉ柟绋匡攻瀵板嫮浠﹂悙顒夊晭濠碉紕鍋戦崐鏍礉瑜忕划濠氬箣閻樼數鐒奸柣搴秵閸嬩焦绂嶅鍫熺厵闁哄鐏濋。宕囩磼鐎ｎ剛甯涢柕鍥у瀵噣鍩€椤掑嫬鍨傞柤鍝ユ暩閳瑰秴鈹戦悩鍙夊闁稿﹪鏀遍妵鍕疀閹炬剚浠遍梺绋款儐閹瑰洤顕ｉ崐鐕佹濠电偛鍚嬮悧婊堝箟閹间焦鍋嬮柛顐ｇ箘閻熴劎绱撴担鎻掍壕婵犮垼娉涢鍕崲閸℃稒鐓忛柛顐ｇ箓閳ь剙鎲＄粋宥嗐偅閸愨晝鍘卞┑掳鍊曢幊宥夊箟妤ｅ啯鐓涚€光偓閳ь剟宕伴弽顓炶摕闁搞儺鍓氶弲婵嬫煃瑜滈崜鐔风暦閵忋倕绠绘い鏃傛櫕閸樻悂姊洪柅鐐茶嫰婢ь垳绱掗崒娑樼闁逞屽墾缂嶅棝宕戦崱娑樺偍闂侇剙绉甸埛鎴︽⒒閸喍绶辨俊鎻掔埣閺屾盯濡搁敂濮愪虎閻庢鍠楅崕濂稿Φ閹版澘绠抽柟鎹愭硾楠炴劙姊虹拠鎻掑毐缂傚秴妫濆畷鎴﹀幢濡粯鐝烽梺姹囧灩閹诧繝鎮″▎鎰╀簻闁哄啫鍊婚幗鍌涚箾閸喓鐭掗柡宀嬬到閳藉宕￠悙瀵稿綆闁诲氦顫夊ú鏍礊婵犲洢鈧礁鈻庨幘鏉戜簵闁瑰吋鎯岄崰妤冪礊閹剧粯鈷掗柛灞捐壘閳ь剛鍏橀幊妤呭礈娴ｇ鐏婂銈嗙墱閸嬫垿鍩€椤掆偓閸熸潙鐣烽崡鐐╂瀻闊洦鎸炬禍浼存煟鎼粹€冲辅闁稿鎹囬弻娑㈠即濡搫唯闂佺顑嗛幑鍥极閸愵喖纾兼慨妯哄船閳ь剛鍋ゅ娲礃閸欏鍎撻梺绋匡工閹芥粍绔熼弴銏╂晬闁绘劕顕崢浠嬫⒑缂佹ê濮囨い鏇ㄥ弮閸┿垽骞樼紒妯煎幈闂佽鍎抽顓犵不閻愮儤鐓ユ繝闈涚墕娴犳粍銇勯幘鍐叉倯鐎垫澘瀚埀顒婃€ラ崟顐紪闂傚倸鍊风欢姘焽閼姐倐鍋撻棃娑氱劯鐎规洏鍨藉畷妤呮嚃閳哄﹥閿ゅ┑掳鍊х徊浠嬪疮椤栫偞鍋傞柡鍥ュ灪閻撴瑩鏌ｉ幇闈涘缂傚秵鍨块弻娑樷攽婵犲喚浠╅梺閫炲苯澧叉い顐㈩槸鐓ゆ俊顖氥偨濞差亝鍋勯柤娴嬫櫅缁侊箓姊洪幖鐐插姶闁告挻宀稿畷鎴犫偓锝庡枟閻撴洘銇勯幇顔夹㈤柣蹇婃櫆椤ㄣ儵鎮欓幖顓熺杹濠殿喖锕︾划顖炲箯閸涙潙宸濆┑鐘插暙閸撶敻姊洪懡銈呅㈡繛灞傚姂瀹曟垿濡堕崨顖涙闂侀潧艌閺呪晠寮崶顒佺厽婵☆垵顕х徊鑽ょ磼閻樿櫕銇濇慨濠勭帛閹峰懐绮电€ｎ亝顔勬繝娈垮枟閿曨偆绮婚幘缁樻櫜闁绘劖娼欑欢鐐测攽閻愭潙绗掗柟纰卞亰閿濈偛顭ㄩ崼婵堝姦濡炪倖甯掔€氼剟寮告笟鈧幃妤呮晲鎼粹剝鐏嶉梺鎶芥敱閸ㄥ潡骞冨Δ鍛嵍妞ゆ挾鍎愰埀顒€鐭傞弻娑㈠Χ閸℃瑦鍣梺閫涚┒閸斿矂锝炲鍫濋唶婵犲灚鍔栭崰妯肩磽閸屾瑧璐伴柛鐘崇洴椤㈡俺顦归柛鈹垮劜瀵板嫰骞囬澶嬬秱闂備胶绮摫妞ゆ梹鐗犲畷顖溾偓锝庡枟閳锋帒霉閿濆洦鍤€闁告柣鍊栫换娑氣偓娑櫭崫铏光偓瑙勬礃閸ㄨ泛顕ラ崟顓濇勃缂備降鍨规禒娲⒒娓氣偓濞佳呮崲閸儱纾归柡宥庡幖閻撴洟鏌熼悜姗嗘畷闁绘挻娲熼幃姗€鎮欓弶鎴狀槰婵犮垼顫夐敃銏ゅ蓟瀹ュ牜妾ㄩ梺鍛婃尰瀹€鎼佺嵁閸愵喖纾兼慨妯块哺濞堥箖姊洪崷顓烆暭婵犮垺顭囩划濠氭偡閹冲﹤缍婇弫鎰板川椤旇棄鏋戦梻浣告惈閼活垶鎮ч幘鎰佹綎婵炲樊浜濋崑锟犳煛婢跺鍎ラ柡鍡愬灲濮婃椽鎮滈埡浣糕拤濠碘槅鍋勯崯顐︻敋閿濆鏁冮柕鍫濆€告禍楣冩煥濠靛棝顎楀褜鍨堕弻娑㈡偐閸愭彃顫掗梺鍝勭焿缁绘繂鐣峰鈧俊姝岊槻妞わ絾妞藉鍫曞煛閸屾粎鐣虹紓浣虹帛閻╊垶鐛鈧鍫曞箣閻樻彃袪闂傚倷鑳堕…鍫ヮ敄閸℃稑绠查柛銉墮閽冪喖鏌嶉埡浣告殲濠殿垱鎸抽弻锝夋偄閸涘﹦鍑″銈庡亝鐢繝骞冨Δ鍐╁枂闁告洦鍓涢ˇ顓㈡⒑閸涘鐒奸柛銉戝懎寮ㄥ┑鐘灱濞夋稖鐧岄梺鍝勫暊閸嬫捇鏌熷畡鐗堝殗闁诡喗绮岃灒闁兼祴鏂侀弸鍡涙⒒閸屾艾鈧悂宕愰幖浣哥９闁归棿绀佺壕鐟邦渻鐎ｎ亝鎹ｉ柣顓炴閵嗘帒顫濋敐鍛婵°倗濮烽崑娑⑺囬悽绋挎瀬闁瑰墽绮崑鎰亜閺冨倹鍤€濞存粓绠栭弻娑㈠箛闂堟稒鐏堥梻浣稿船濞差參寮婚弴鐔风窞闁割偅绻傛慨銏ゆ⒑閹稿海鈯曢柣鈺婂灠椤繘鎼归崷顓狅紲濠碘槅鍨伴幖顐︼綖瀹€鍕拺闁革富鍘搁幏锟犳煕鐎ｎ亝顥犻柛?
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

	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒姗堣€跨€殿喖顭烽幃銏ゆ惞閸︻叏绱查梻渚€娼х换鎺撴叏閻㈠憡鍊堕柛顐犲灮绾捐棄霉閿濆懏鎯堥崯鍛婄節閻㈤潧浜归柛瀣尭铻栭柣姗€娼ф禒锕傛煟濡や焦绀夌憸棰佺椤啴濡堕崱妤€娼戦梺绋款儐閹瑰洭寮诲鍫闂佸憡鎸诲畝绋跨暦瑜版帒绀堝ù锝堟閻掑潡鎮楅獮鍨姎闁绘瀚粋宥堛亹閹烘挾鍘甸梺缁樺灦钃遍悘蹇曟暬閺屾稑顫滈埀顒佺閸洖绠栨俊銈呮噹缁€鍌氼熆鐠虹尨姊楀瑙勬礋濮婄粯绗熼埀顒勫焵椤掑倸浠滈柤娲诲灡閺呭墎鈧數纭堕崑鎾舵喆閸曨剛顦ㄩ梺鎼炲妼濞硷繝鐛崘銊㈡瀻闁圭偓娼欓埀顒傜帛娣囧﹪顢涘┑鍕ㄥ亾閳ь剟鏌涚€ｎ偅宕岀€规洜顭堣灃濞达綁鏅查崠鏍⒒娴ｈ鍋犻柛搴ｅ劋缁傚秶鎹勬笟顖涚稁闁荤姵浜介崝宥夊窗閹扮増鐓熸俊銈傚亾闁绘妫涚划顓烆潩閼哥數鍘搁梺鍛婁緱閸犳牜寰婄紒妯镐簻妞ゆ挾鍋為崑銉╂煙閾忣偆鐭掗柟顖氼樀椤㈡棃宕ㄩ鐘辩礉闂佽姤顭囬崰鏍蓟閳ュ磭鏆ゆい鏂垮悑濠€浼存倵闂堟稓鐒告慨濠冩そ濡啫鈽夊顒夋毇闂備胶顢婂▍鏇㈠礉濡ゅ啫鍨濆┑鐘宠壘娴肩娀鏌涢弴鐐典粵閻庨潧鐭傚娲传閸曞灚效闂佹悶鍔岄悥濂哥嵁婢舵劕绠瑰ù锝呮贡閸橀潧螖閻橀潧浠滈柣蹇旂箞閹﹢鍩￠崨顔惧帗闂佽姤锚椤﹁棄螣閳ь剚绻濆▓鍨灀闁稿鎹囧铏圭磼濮楀棛鍔搁柣蹇撴禋娴滎亪宕洪埀顒併亜閹哄棗浜鹃梺璇茬箲缁诲牓鍨鹃弮鍫濈妞ゆ柨妲堣閺屾盯鍩勯崗鐙€浜鍐参旀担铏圭槇濠电偛鐗嗛悘婵嬪几閵堝鐓曢煫鍥ㄦ閼版寧顨ラ悙鍙夘棡缂佹鍠栭崺鈧い鎺嗗亾闁伙綁鏀辩缓鐣岀矙鐠囦勘鍔戦弻銊╁籍閸ヨ埖缍堥柣搴㈣壘閵堢顫忓ú顏勪紶闁告洖鐏氭闂備胶顭堥鍥磻閻愮數鍗氶柣鏃傚帶楠炪垺绻涚€涙ɑ绶叉繛宸弮瀵偊宕橀鑲╋紲濠电偞鍨堕懝鎯ь嚕閹惰姤鈷掑ù锝呮啞閹牊绻涢弶鎴濃偓鍦矉瀹ュ鍊烽柣鎴灻禒濂告⒑閸撹尙鍘涢柛瀣笒閳绘挻绂掔€ｎ偆鍘介梺褰掑亰閸ㄤ即鎯冮搹鍦＜闁绘ê妯婇悡濂告煛瀹€鈧崰鏍€佸▎鎴炲枂闁告洦鍓涜ぐ鍛存⒑缂佹ɑ灏柛濠冾殘濡叉劙骞樼€涙ê顎撻梺鍏肩ゴ閸撴繈宕归悽绋跨厺鐎广儱顦崘鈧梺闈浤涢崨顖涱潓闂傚倷鐒﹂惇褰掑垂閽樺鐒界憸搴ｇ矉閹烘埈鐓ラ柛鏇ㄤ簽缁犳艾顪冮妶鍡楀Ё缂佹彃娼￠獮濠囧炊閳规儳浜鹃悷娆忓缁€鈧紓鍌氱Т閿曨亪鐛崘顔藉€婚柦妯侯槺椤撳ジ姊洪幆褎绂嬮柛瀣缁傚秹寮介鐔叉嫽婵炴挻鍩冮崑鎾寸箾娴ｅ啿娲﹂崑瀣煕閳╁啨浜楁繛鎴炃氬Σ鍫熸叏濡搫缍佺紒妤€顦版穱濠囧Χ韫囨洖鍩岄梺鍝ュ櫏閸ㄨ泛鐣烽幇鐗堝仺闁告稑锕﹂崢浠嬫⒑闂堟稓绠氭俊鎻掓嚇瀹曨垵绠涘☉娆戝幗闂佸湱鍎ら〃鍫ユ偩閻戞ü绻嗘い鎰╁灩椤忣偊鏌嶈閸撱劎绱為崱妯碱洸閻犵儤浜介埀顒佸笚缁绘繂顫濋鐘插妇闂備礁澹婇崑鍛崲閸愵啟澶愭倷閻戞鍘撻梻浣哥仢椤戝懏鎱ㄥ澶嬬厸閻忕偛澧藉ú瀵糕偓娈垮櫘閸ｏ綁宕洪埀顒併亜閹烘垵顏ラ柍褜鍓欓崯鏉戠暦閵娾晩鏁囬柣鎰仯閸嬫帡姊婚崒姘偓椋庣矆娴ｈ櫣绀婂┑鐘叉搐绾捐鈹戦悩鍙夋悙缂佲偓閸屾稒鍙忔俊鐐额嚙娴滈箖鎮楀▓鍨灁闁告柨绉剁划瀣箳閺傚搫浜鹃柨婵嗛娴滄粓鏌嶈閸撴盯寮繝姘摕闁靛鍎弨浠嬫煕閳╁啩绶遍柍褜鍓氶〃鍛存箒濠电姴锕ら幊搴㈢闁秵鐓涢悘鐐插⒔濞插鈧鍣崜娑㈠箲閸曨垰惟闁靛／鍐ㄧ婵犵數濮甸鏍窗閺嶎厽鏅濋柕澶堝労濞撳鏌﹀Ο渚Ш闁哄棴闄勭换婵囩節閸屾粌顤€闂佹悶鍊栭崹鍫曞Φ閸曨垰绠抽柛鈩冦仦婢规洟姊哄Ч鍥х労闁割煈鍓熷畷鏇㈡濞戣鲸缍庡┑鐐叉▕娴滄粍瀵奸悩缁樼厪濠㈣泛鐗嗛崜楣冩煥濠靛棭妲归柣鎾跺枑娣囧﹪濡堕崟顔煎帯闂佹椿鍘界敮鐐哄焵椤掍緡鍟忛柛锝庡櫍瀹曟垶绻濋崶銉㈠亾娴ｅ壊娼╅悹楦挎閸旓箑顪冮妶鍡楀潑闁稿鎹囬弻锝夋晲閸涱厽些闁句紮绲介妴鎺戭潩閻撳海浠╅梺鍝勬噽婵挳鈥旈崘顔嘉ч柛鈩兦氶幏褰掓⒑缁嬪灝顒㈠┑鐐诧工椤曪綁顢曢敃鈧粻鐟懊归敐鍛辅闁归绮换娑欐綇閸撗冨煂闂佸湱鈷堥崑鍡涘极椤曗偓瀹曞ジ寮撮悢鍝勫箥闂備礁鍚嬫禍浠嬪磿閹惰姤鍎楅柟鐑樻煛閸嬫挸鈻撻崹顔界彯闂佺顑呴敃顏堟偘椤曗偓瀵粙濡搁敃鈧鎾剁磼閸撗冾暭闁挎艾霉濠婂牏鐣洪柡灞剧☉閻ｆ繈鍩€椤掑嫬纾绘繛鎴欏灪閸庢淇婇妶鍌氫壕濡炪値鍙€濞夋洟骞戦崟顖涙優闁荤喖鍋婇弳顏嗙磽閸屾瑩妾烽柛鏂跨焸閳ワ箑鐣￠柇锔界稁婵°倧绠掗敓銉︾瑜版帗鐓欓柣鎴灻悘锝夋煕閻樿韬慨濠呮缁瑧鎹勯妸褜鍞归梻浣藉吹閸熷潡寮插☉銏″仼闁绘垼妫勭粻锝夋煥閺囨浜惧銈庡亜閹虫﹢寮诲澶婂瀭婵炴垶鐟﹂悵鏃€绻涚€涙鐭掔紒鐘崇墵楠炲啫鐣￠幍铏€婚棅顐㈡处閹尖晜绂掗崜褏纾藉ù锝嗗絻娴滈箖姊洪崨濠傚Е闁哥姵顨婇幃锟犲Ψ閿旇棄寮垮┑鈽嗗灠閻忔繈鎮￠幇鐗堢厽闁规儳鍟块弳锝夋煙椤旇偐绉虹€规洘鍎奸ˇ鑼磼閻欐瑥娲﹂悡鏇㈡煟閹邦剚鈻曟俊鑼舵缁辨帗娼忛妸銉﹁癁闂佽鍠掗弲娑㈡偩閻戣棄鐐婇柕濞垮劚閻忊€斥攽鎺抽崐妤佹叏閻戣棄纾婚柣鎰劋閺呮繈鏌曡箛瀣偓鏇犵不閸︻厽鍠愰柣妤€鐗嗘穱顖炴煛娴ｇ鏆ｉ柡灞炬礃缁旂喖顢涘顒佹缂備礁寮跺钘夘潖婵犳艾纾兼慨姗嗗厴閸嬫捇骞栨担鍝ワ紮閻庣懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻濠氭煙瀹曞洤浠遍柡灞芥椤撳ジ宕卞Δ浣烘殶闂傚倷绀侀悿鍥ь浖閵娾晜鍤勯柛鎾茶兌妞规娊鐓崶銊︾缁惧彞绮欓弻娑氫沪閸撗勫櫘闂佸憡鏌ㄧ粔鐢稿Φ閸曨垼鏁囬柣鎰綑閺嬬娀姊虹拠鈥虫灓闁轰浇顕ч悾閿嬬附缁嬪灝宓嗛梺缁樻煥閹碱偊鐛崼鐔剁箚闁绘劦浜滈埀顒佺墵瀹曟繂螖娴ｇ懓寮块梺鍦檸閸犳牜绮堟径瀣弿婵☆垰鐏濋悡鎰版煟閹惧瓨绀冮柟渚垮妼椤粓宕卞Δ鈧～顏堟煟閻樺啿濮х紒缁樼箞瀵鎮㈤悡搴ｇ厬婵犮垼娉涢鍛村焵椤掍礁濮夐柍褜鍓氶鏍闯椤曗偓瀹曟粌鈽夐姀鐘殿唹闂佸憡娲﹂崹鐗堝劔闂備焦瀵у濠氭偤閺冨牆鐤幖娣妽閳锋帡鏌涚仦鎹愬闁逞屽墮閹芥粓鍩€椤掍礁鍤柛鎾寸懅閸欏懘姊洪棃娴ゆ盯宕橀妸褌鍠婇梻浣藉吹婵潙煤閿斿墽鐭堥柡澶嬪殾閿濆閱囬柣鏃傤焾瀵灝鈹戞幊閸婃劙宕戦幘缈犵箚妞ゆ劧绲块幊鍥┾偓瑙勬礃閹瑰洭骞冩禒瀣窛濠电偟鍋撶€氳偐绱撻崒姘偓鐑芥倿閿曚焦鎳岄梺璇茬箰濞存岸宕ｉ崘顔肩畺婵°倐鍋撻柍钘夘樀楠炴帡骞樼€电缍冮梺璇插椤旀牠宕板Δ浣虹濠电姴鍟╃换鍡涙煟閹达絽袚闁哄懏绮撻弻娑㈠箻濡も偓閹冲繘锝炲畝鍕拻闁稿本鐟ч崝宥夋煟椤忓嫮绉虹€规洖缍婇幐濠冨緞濡厧濮洪柣鐔哥矋閺屻劑鎮惧畡閭︾叆闁告劦浜濆▓楣冩⒑閹肩偛鍔楅柡鍛箞閺佸秴鈹戠€ｎ偄鈧灚顨ラ悙鑼虎闁告梹鐟х槐鎺楀焵椤掍焦濯撮柛锔诲弾濞叉悂姊虹紒妯哄Е闁告挻绋撳褔鍩€椤掑嫭鈷戞慨鐟版搐閻忓弶绻涙担鍐插椤╃兘鏌ㄩ弴鐐测偓褰掓偂濞嗘挻鐓曟繝闈涘閸旀挳鏌ｉ幒妤冪暫闁哄被鍔戝鏉懳熼搹閫涚礄闂備礁鎼懟顖毼涘Δ鍜佹晣濠靛倻顭堝婵囥亜閺嶇數鍒伴柡浣规倐濮婃椽鎳￠妶鍛€剧紓渚囧枛鐎涒晠寮茬捄浣曟棃宕橀埡鍌涱唶婵犲痉鏉库偓鏇㈠箠鎼淬劍鍋Δ锝呭暞閻撴瑩姊婚崒姘煎殶闁告柨绉归弻锝夊箻鐠虹儤鐎炬繛锝呮搐閿曨亪銆佸☉姗嗘僵妞ゆ挾鍠庨崜鍨節?	// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閹冣挃缂侇噮鍨抽幑銏犫槈閵忕姷顓洪梺鍝勫暊閸嬫捇鏌涢妶鍛ч柡灞剧洴婵＄兘顢欓悡搴樻嫽闂備浇妗ㄧ粈浣该洪銏犺摕闁哄浄绱曢悿鈧梺鍝勬川閸婎偊濡烽敂杞扮盎闂佹寧妫侀褍鈻嶅澶嬬厵妞ゆ梻鐡斿▓婊堟煟濞戝崬娅嶇€规洖缍婇、娆撴偂鎼搭喗缍撴繝纰夌磿閸嬫垿宕愯閳ь剟娼ч惌鍌氱暦閻熸壆鏆﹂柛銉戝啰浜伴梻浣稿閸嬩線宕曢柆宥嗙厑闁搞儯鍔庣弧鈧梺闈涢獜缁辨洜绮婚幘鍓佺＝鐎广儱鎷戦煬顒侇殽閻愭彃鏆ｉ柛鈺佸瀹曟﹢鍩℃担绋课ら梻鍌欑劍鐎笛呮崲閸屾娑樷枎閹惧磭鐛ラ梺鍝勭▉閸樹粙鍩涢幒鎳ㄥ綊鏁愰崨顔兼殘闂佽鍨伴悧鎾诲蓟閻旈鏆嬮梺顓ㄧ畱閸撳爼鎮楃憴鍕缂侇喖鐭傞敐鐐测攽閸喎纾梺鎯х箰濠€閬嶅级娴犲鈷掑〒姘ｅ亾婵炰匠鍥ｂ偓锕傚醇閵夈儳锛熷銈嗘磵閸嬫挾鈧鍠撻崝宥囩矉閹烘柡鍋撻敐搴′簽闁告﹢浜跺娲棘閵夛附鐝旈梺鍝ュУ閼归箖鍩㈤幘鎰佹建闁逞屽墴瀵鈽夊Ο閿嬬€婚棅顐㈡处閹稿憡鎱ㄥ鎵怚 `/v1/responses` streaming 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掓禒楣冩⒑缁嬫鍎嶉柛濠冪箞瀵寮撮悢铏诡啎閻熸粌绉瑰畷顖烆敃閿旇棄鈧泛鈹戦悩鍙夊闁稿﹦鏁婚弻娑滅疀閹垮啯笑婵炲瓨绮撶粻鏍ь潖濞差亜宸濆┑鐘插暟椤︺儵姊虹拠鑼鐎光偓缁嬫鍤曢柡灞诲劜閸婄兘鏌ｉ幋鐐冩岸骞忓ú顏呪拺闁告稑锕﹂埥澶愭煥閺囨ê鍔滅€垫澘瀚板畷鐔碱敍濞戞艾骞堥梺璇插嚱閹儵宕樿椤ユ岸姊婚崒姘偓椋庣矆娓氣偓楠炴牠顢曢敂缁樻櫈闂佸憡绋戦悺銊╂偂閳ь剟姊洪幐搴ｇ畵妞わ富鍨堕幏鎴︽偄閸忚偐鍘搁梺鎼炲劦椤ユ挾澹曢崸妤佺厽闁靛牆鍊告禍楣冩⒒閸屾瑧绐旀繛浣冲洦鏅煫鍥ㄧ☉缁€瀣亜閹偣鍊楃粻鐑芥⒒閸屾艾鈧悂宕愰悜鑺ュ€块柨鏇炲€归崕鎴犳喐閻楀牆绗掔痪鎯х秺閺岀喖鎮ч崼鐔哄嚒缂佺偓鍎抽…鐑藉蓟閻旂厧绀堢憸蹇曟暜濞戙垺鐓冮梺鍨儏缁楁帡妫佹径鎰叆婵犻潧妫欓崳娲煕閻斿搫浠遍柡灞剧洴閹垽宕崟顏呭煕缂傚倷娴囨ご鍝ユ暜閻愬搫鐒垫い鎺戯功缁夌敻鏌涚€ｎ亜顏╅棁澶嬨亜閺囨浜鹃梺鍝勭灱閸犳牠銆佸璺虹劦妞ゆ帒鍊绘稉宥夋煛瀹ュ骸骞戦柍褜鍏涚欢姘嚕椤曗偓瀹曠厧鈹戠€ｎ亝鏆┑鐘垫暩婵炩偓婵炰匠鍥ㄥ亱闁糕剝蓱閸欏繘鏌ㄩ弮鈧崹婵堟崲閸℃稒鐓熼柟鏉垮悁缁ㄥ鏌嶈閸撴岸鎮ч弴銏╂晩闊洦姊荤弧鈧┑顔斤供閸撴盯鏁嶅☉銏♀拺閻熸瑥瀚崝銈咁熆瑜嶅ù閿嬬珶閺囥垺鍤冮柍鍝勫暟閿涙繃绻涙潏鍓у埌妞ゎ偅鐗犻、鏃堝幢濞嗘垹鏆ラ柣鐔哥矋濡啫顕ｇ拠娴嬫闁靛繒濮烽濠囨⒑鐟欏嫬绀冩繛鍛礈閹风娀鎮欏顔藉瘜闂侀潧鐗嗛崯顐︽倶椤忓牊鐓ラ柡鍥悘顏堟煙娓氬灝濮傞柛鈹惧亾濡炪倖甯掔€氼參鎮￠崘顔界厓閺夌偞澹嗛ˇ锕傛煛閸℃瑥浠遍柡宀嬬到閳规垿宕堕埡浣叉嫲婵犳鍠栭敃锔惧垝椤栨粎绠旈柣鏃傚帶閻掑灚銇勯幒鎴濐仼闁绘帒鐏氶妵鍕箳閹存績鍋撻崨濠勵浄婵犲﹤鐗婇悡鐘崇箾閼奸鍤欓柣蹇ョ節閺岋繝宕ㄩ鐑嗘殺闂侀€炲苯澧紒瀣浮閺佸啴鍩℃担鍙夌亖?OpenAI Responses schema闂?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锔剧磼閵娧呭笡濞ｅ洤锕幃娆擃敂閸曘劌浜鹃柡宥庡亝閺嗘粓鏌熼悜姗嗘當缁炬儳婀辩槐鎾存媴鐠囷紕鍔烽梺宕囩帛濮婂鍩€椤掆偓缁犲秹宕曢柆宓ュ洦瀵肩€涙ê浜楀┑鐐叉閹稿摜鐥閺屾盯顢曢敐鍥╃暤闂佹娊鏀卞Λ鍐蓟閿濆鏅插璺侯槹閸犳岸姊洪崫鍕拱闁烩晩鍨辨穱濠囧箹娴ｈ倽銊╂煏婢跺牆鐏╁ù婊冨⒔閹叉悂鎮ч崼婵堢懆婵℃鎳樺娲川婵犲啫顦╅梺鎼炲妺閸楀啿鐣峰鈧崺鈧い鎺戝閳锋垿姊婚崼鐔惰吂婵炴垶鐟︽刊鎾煙缂佹ê绗傚瑙勬礋濮婃椽宕烽鐐插濡炪們鍔屽Λ婵嬪箖閿熺姴绀冩い蹇撴閿涙繃绻涙潏鍓хК闁稿鍊块獮瀣╃秴濠㈣埖鍔曢柋鍥煏婢舵稑顩柛姗€娼ч—鍐Χ閸℃ǚ鎷婚梺鐑╁墲閺屻劑鍩㈤幘鎰佺叆闁割偆鍠撻崢鍛婄箾鏉堝墽鎮奸柟铏尰閹便劑宕奸妷锔惧幍濡炪倖鐗楃粙鎺椝夐崼銉︾厱闁靛ň鏅濋悾娲煙瀹曞洤鈻堟い銏☆殕閹峰懐鎲撮崟鍏稿寲闂?SDK闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ偒鍎ラ柛搴＄箲閵囧嫰骞嬪┑鎰枅閻庢鍠涢褔鍩ユ径濞㈢喖鏌ㄧ€ｅ灚缍岄梻鍌欑閹诧繝銆冮崼銉ョ；闁绘柨鎽滈々閿嬨亜閹捐泛鍓辨繛鎾愁煼閺屾洟宕煎┑鍥舵婵犳鍠栭崯鎵閹烘鏁婇柣锝呮湰閸ｄ即鎮楀▓鍨灈妞ゎ厼鍢查锝夊箻椤旇棄浜滄俊鐐差儏濞寸兘鎮伴鐣岀瘈闁汇垽娼ф禒锕傛煕閵娿劍纭炬い顐ｇ箞婵℃瓕顦撮柣婊呯帛娣囧﹪濡堕崨顓熸缂?OpenCode闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊婚崟顐ｅ枠妞ゃ垺淇洪ˇ鏌ユ偂閵堝棎浜滈柟鍨暞婵炲洭鏌嶈閸忔稓绮堟笟鈧崺銏℃償閵娿儳顔掗梺鍝勵槹閸ㄧ喖宕戦幘缁樼叆閻庯絻鍔嬬花濠氭⒑閸︻厼鍔嬮柛鈺佺墕鍗遍柛锔诲幘绾剧晫鈧箍鍎卞Λ顓炩枔閺冨牊鐓冮悷娆忓閻忔挳鏌熼鐣屾噮闁逞屽墯缁嬫帡鈥﹂崶鈺冪煓闁搞儺鍓氶埛鎴︽煕濞戞﹫榫氬瑙勆戦妵鍕箣濠靛洤娈楅梺鐐藉劵缁犳挸鐣烽幆閭︽Х闂佸搫顑呴柊锝夊蓟閻斿吋鍊锋い鎺嶈兌缁嬪洭姊洪柅鐐茶嫰婢ь噣鏌涢姀锛勫弨闁糕晝鍋ら獮瀣晜閽樺姹楅梻浣告啞閻熴儵藝椤栫倛鍥煛閸涱喒鎷虹紓鍌欑劍钃遍悘蹇曟暬閺屾盯鎮╅崘鎻掝潔缂備礁鐭佹ご鍝ユ崲濠靛鐐婇柤绋跨仛濞呮牗绻濋悽闈涒枅婵炰匠鍏犳椽濡搁埡浣勩儱鈹戦悩鎻掝伀缁炬崘妫勯湁闁挎繂顦板☉褍鈹戦鐟颁壕濠电姷鏁搁崑娑⑺囬弶妫靛搫顭ㄩ崗鎾呯秮楠炲洭寮剁捄顭戝敽婵犲痉鏉库偓鎰板磻閹剧粯鐓熼柟鐑樺灩娴犳盯鏌曢崶褍顏鐐村浮楠炲鈹戦崘銊ゅ闂佹眹鍨婚…鍫㈠婵犳碍鐓欓柛鎾楀懎绗￠梺鍝勬噺閻擄繝鐛弽顐㈠灊闁稿繐顦禍楣冩煙妫颁胶鍔嶅Δ鐘插缁辨捇宕掑▎鎺戝帯闂佺顑嗛幑鍥х暦閺囥垹绠柦妯侯槼琚濋梺璇插嚱缂嶅棝宕伴弽顭戞敯闂傚倷绀侀幉鈥趁洪敃鍌氬瀭闁规鍠氭稉宥嗙箾閹存瑥鐏柣鎾冲暣瀵爼鎮欓弶鎴濡炪倧璁ｇ粻鎴﹀煘閹达富鏁婇柤娴嬫櫅閻撶喎鈹戦纭锋敾婵＄偘绮欓悰顕€寮介鐔封偓閿嬬箾閺夋埈鍎忓ù婊堢畺閺屸€愁吋鎼粹€崇闂佺顑戠换婵嬪蓟瀹ュ浼犻柛鏇ㄥ墮濞咃綁姊洪挊澶婃殶闁哥姵鐗犲濠氬Ω閵夈垺鏂€闂佺硶鍓濋敃鈺佄涢敓鐘斥拺缂佸瀵ч幑锝夋煕閻樺磭澧电€殿喖顭锋俊鎼佸Ψ閵忊剝鏉搁梻浣虹《閸撴繈鏁嬪┑鐐叉嫅缁插€熺亙闂佺粯锕㈠褎绂掑鍕╀簻闁瑰瓨绻嶅Ο鈧Δ鐘靛仜閸燁偉鐏冮梺閫炲苯澧撮柛鈹垮灲楠炴鎷犻幓鎺斺偓顓烆渻閵堝棙鈷掗柡鍜佸亝缁傚秹鎮欓悽鐢碉紳闂佺鏈悷褏鎷规导瀛樼厱闁绘ɑ鍓氬▓婊堟煙椤旀儳鍘撮柛鈹惧墲缁楃喖宕惰椤撴寧淇婇悙顏勨偓鏍礉瑜忕划濠氬箣閻樺吀绗夐梺鐓庮潟閸婃澹曢挊澹濆綊鏁愰崪浣圭稑濡炪倕绻愰幊鎰般€呴懠顒佸枑闁绘鐗嗙粭姘舵煃闁垮娴柡灞剧〒娴狅箓鎮欓鍌涱吇濠电姭鎷冮崟顓溾偓鎺旂磼鏉堛劌绗ч柍褜鍓ㄧ紞鍡涘磻閸涱厾鏆︾€光偓閸曨剛鍘靛銈嗘礀濡稓寮ч埀顒€螖閻橀潧浠滄い鎴濇嚇閸╃偤骞嬮敃鈧粈鍐煃閸濆嫬鈧懓顭块幋锔解拻闁稿本鐟ㄩ崗宀€绱掗鍛仸闁轰礁鍟撮崺锟犲川椤撶姷宕堕梻浣告惈缁嬩線宕㈡禒瀣９闁煎摜鍋ｆ禍婊堢叓閸ャ劍灏靛褎鐩弻锝夊箻鐎涙顦板Δ鐘靛仦閻楁洝褰佸銈嗗坊閸嬫捇鏌嶈閸撴艾煤濡偐鐭夌€广儱鐗勬禍褰掓煙閻戞ɑ灏甸柛姗€浜跺娲捶椤撶偛濡洪梺鐟版憸椤牓婀佸銈嗘閺侇噣宕戦幘鑸靛枂闁告洦鍓涢ˇ銊х磽娓氬洤鏋涚紒缁橆焾濡喖姊虹憴鍕姸濠殿喓鍊濋幃锟犲即閵忥紕鍘藉┑鈽嗗灠閹碱偆鐥閵囧嫰濡烽敐鍛紙闂佸搫鐭夌槐鏇熺閿旂偓瀚氶柟缁樺俯濞煎酣鏌ｆ惔銏╁晱闁哥姵顨婇獮鎰板箮閽樺鐣哄┑鐐叉缁剁兘宕烽娑樹壕闁挎繂绨肩花宄邦熆鐟欏嫭绀嬮柟顔煎槻椤劑宕ㄩ褎姣夐梺姹囧焺閸ㄨ京鏁垾宕囨殾闁靛鍔婃禍褰掓煙閻戞ɑ灏甸柛妯绘崌閹嘲顭ㄩ崟顓犵厜閻庤娲樼划鎾诲箖閵忋倖鍋傞幖娣€栭幉浼存⒒娴ｇ懓顕滅紒璇插€块幃褑绠涘☉娆忎痪闂侀€炲苯澧存慨濠冩そ瀹曨偊宕熼鈧粣娑㈡⒑缁嬫鍎愰柨姘舵煃鐠佸磭鐭欐い銏℃瀹曞ジ鎮㈤崣澶婎伖缂傚倸鍊风粈渚€顢栭崨顓х劷闁跨喓濯弫浣逛繆閵堝懏鍣洪柣鎾跺枛閺屾洟宕煎┑鍥ф闂佸摜濮甸敃銏ゅ蓟閺囥垹骞㈡俊銈傚亾闁逞屽墮閻忔繈鎮炬搴ｇ煓閻犲洨鍋撳褰掑箯閸涙潙鐭楀璺侯煬娴兼粓姊婚崒姘偓鐑芥嚄閸洍鈧箓宕奸妷銉ョ彉濡炪倖甯掔€氼參宕戦敍鍕枑闁哄倽娉曢弳锔界節闂堟稒锛嶉柛姘儏椤法鎹勬笟顖氬壈闂侀€炲苯澧伴柡浣告憸濡叉劙骞掗幊铏⒐閹峰懐鎮伴埄鍐炬綌濠电姷鏁搁崑鐘活敋濠婂懐涓嶉柟鎯х－閺嗭箓鏌熸潏楣冩闁哄懏鎮傞弻锝呂熼搹鐧哥礊闂佸憡鐟ュΛ妤呭煘閹达附鍋愮紓浣股戦柨顓炩攽閳藉棗浜介柛妯煎帶瀹撳嫰姊鸿ぐ鎺擄紵闁绘帪绠撻幃?
	errorEventSent := false
	clientDisconnected := false // 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟撮崣鍕煏閸℃鏆ｅ┑锛勫厴閸┾剝鎷呮搴ｅ€為梻鍌欑窔濞佳囨晬韫囨稑纾兼繝濠傛噺閸犳帡姊绘担绛嬪殭濡ょ姴鎽滅划璇差吋婢跺﹦锛熼梻渚囧墮閸楁洟宕堕澶嬫櫖闂佺粯鍔栬ぐ鍐倵椤撱垺鈷戠紒瀣濠€鎵磼鐎ｎ偅灏电紒顔碱煼瀹曟粏顦柛瀣崌閹兘寮跺▎鐐棏闂備礁鎽滄慨闈浢哄鍫熷殟閺夊牄鍔庣弧鈧┑顔斤供閸橀箖宕㈤幖浣光拺闁告稑锕ョ壕鐢告煛閸涱喚娲寸€规洘绻傞…銊╁川椤栨粣绱茬紓鍌氬€烽悞锕傗€﹂崶顬¤櫣鈧稒锕╁▓浠嬫煟閹邦剚鈻曢柛搴㈡閺岀喖顢欓悾灞惧櫘闂侀€炲苯澧存繛浣冲洤绠熼柨鐔哄閺佸洤鈹戦悩宕囶暡闁抽攱甯掗湁闁挎繂鎳忛崯鐐烘煕閻斿搫浠﹂柕鍥у婵℃悂濡烽敂缁橈骏闂備礁鎽滄慨鎾晝閵堝鏁囬柛蹇曞帶缁剁偛鈹戦悜鍡樼窙缂佺姵鎹囧璇测槈閵忕姴宓嗛梺闈涱焾閸庤櫕绂掗埡鍛拺闁告稑锕ゆ慨鈧梺绋款儐閹瑰洤顫忕紒妯肩懝闁逞屽墮椤洩顦堕柛锝呯秺濮婃椽鏌呴悙鑼跺濠⒀屽灦閺岀喐绗熼崹顔碱潎閻庤娲栭悥濂搞€佸Δ浣瑰闁告瑥顦鍦磽閸屾艾鈧悂宕愰悜鑺ュ€块柨鏇炲€归弲顏嗙磽閸屾瑧鍔嶉柕鍥ф瀹曞爼濡歌楠炴劙姊绘担鍛婂暈缂侇喖鐗忕划鍫熸媴閸︻収娲告繛瀵稿帶閻°劑鎮￠弴銏＄厓閻熸瑥瀚崝銈吤瑰鍛壕濞ｅ洤锕幃娆擃敂閸曘劌浜鹃柡宥庡亝閺嗘粌鈹戦悩鎻掝仾妞ゆ劒绮欓弻銊╂偄閸濆嫅銏ゆ煟閵堝骸娅嶉柣鎿冨亰瀹曞爼濡歌婵洭姊虹紒妯诲鞍婵炶尙鍠栧濠氭晲閸涘倻鍠栧畷顐﹀礋椤掍胶妲┑鐘垫暩閸嬬偤骞嗗畝鍕獥婵娉涢悞鍨亜閹烘垵鏋ゆ繛鍏煎姈缁绘盯宕ｆ径娑溾偓璺ㄢ偓瑙勬礀缂嶅﹤鐣风粙璇炬棃宕橀妸褋鍋婂┑鐘殿暯濡插懘宕归悽绋跨；闁归偊鍠栭ˉ姘舵煕韫囨稒锛熺紒璇叉閵囧嫰寮介妸褏鐓€闁汇埄鍨甸崺鏍€冮妷鈺傚€烽柤纰卞墮椤も偓缂傚倷鑳剁划顖滄崲閸岀偛鐓濋柟鎹愵嚙閸ㄥ倹銇勯弮鍌涙珪濞存粌鐖煎缁樻媴閻熼偊鍤嬪┑鐐村絻缁夌懓顕ｉ弻銉﹀亹闁肩⒈鍓氬▓鎯р攽閻樿宸ラ柛鐘宠壘椤洭鍩￠崨顔间化婵°倧闄勭€笛囶敂閻樼偨浜滈柡鍌涱儥濡偓濠殿喖锕ュ浠嬬嵁閹邦厽鍎熼柨婵嗘川閺嗐倖淇婇悙顏勨偓褏绱撳璺虹闁规儼妫勭粻鏍煃閵夛附鐏遍柡瀣叄閺岀喖骞嗚閸ょ喖鏌涘Ο鍏兼毄缂佽鲸鎹囧畷鎺戔枎閹炬惌鈧牠姊洪幖鐐插闁硅櫕鎹囬崺銏狀吋閸涘倻鍠撻幏瀣暦閸モ晝鏆板┑锛勫亼閸婃牠宕濊瀵板﹥绂掔€ｎ偄鈧爼鏌嶉崫鍕殶缁?drain 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼达絾鏆╂い顓炵墛娣囧﹥绂掔€ｎ偀鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜閻犲洩灏欐晶锔锯偓娈垮枛椤嘲顕ｉ幘顔藉亜濡炲娴烽悰顕€姊绘担铏广€婇柛鎾寸箚閹筋偊姊虹紒妯肩畺婵炶尙鍠庨～蹇涙惞閸︻厾鐓撳┑鐐叉閸庢娊宕滈弶娆炬富闁靛牆绻愰々顒勬煛娴ｇ瓔鍤欐い鏇稻缁绘繂顫濋鈹喚鐔嗛悹铏瑰皑濮婃绱掗崡鐐靛煟婵﹥妞藉Λ鍐ㄢ槈鏉堫煈鈧棝姊婚崒姘仼閻庢凹鍓濋。楣冩⒑閸涘﹥澶勯柛瀣笒閵嗘帗绻濆顓犲帾闂佸壊鍋呯换鍌炲焵椤掑倹鍤€闁宠绉瑰畷鍫曨敆娴ｅ搫骞堥梻濠庡亜濞诧箑螞閹达附鍤€閻犳亽鍔夐崑鎾斥枔閸喗鐏堝銈庡幖閸㈡煡顢氶敐澶婄妞ゆ棁妫勬禍婊勪繆閻愬樊鍎忔繛瀵稿厴閹即顢欓悾宀€鐦堥梺姹囧灲濞佳勭閿曞倹鐓曟い顓熷灥閻忥妇鈧娲滈幊鎾跺弲濡炪倕绻愮粔鐢稿疾濞戙垺鍊垫鐐茬仢閸旀碍銇勯敂鍨祮闁糕晜鐩獮瀣晜閻ｅ苯骞堟繝鐢靛█濞佳兾涘Δ鍜佹晜妞ゅ繐鐗婇悡銉︾箾閹寸儐鐒界€涙繈姊婚崶褜妲圭紒缁樼箖缁绘繈宕掑鍐ㄦ瘓缂傚倷闄嶉崝搴ｅ垝椤栫偛桅?usage
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
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟撮崣鍕煏閸℃鏆ｅ┑锛勫厴閸┾剝鎷呮搴ｅ€為梻鍌欑窔濞佳囨晬韫囨稑纾兼繝濠傛噺閸犳帡姊绘担绛嬪殭濡ょ姴鎽滅划璇差吋婢跺﹦锛熼梻渚囧墮閸楁洟宕堕澶嬫櫖闂佺粯鍔栬ぐ鍐倵椤撱垺鈷戠紒瀣濠€鎵磼鐎ｎ偅灏电紒顔碱煼瀹曟粏顦柛瀣崌閹兘寮跺▎鐐棏闂備礁鎽滄慨闈浢哄鍫熷殟閺夊牄鍔庣弧鈧┑顔斤供閸橀箖宕㈤幖浣光拺闁告稑锕ョ壕鐢告煛閸涱喚娲寸€规洘绻傞…銊╁川椤栨粣绱茬紓鍌氬€烽悞锕傗€﹂崶顬¤櫣鈧稒锕╁▓浠嬫煟閹邦剚鈻曢柛搴㈡閺岀喖顢欓悾灞惧櫘闂侀€炲苯澧存繛浣冲洤绠熼柨鐔哄閺佸洤鈹戦悩宕囶暡闁抽攱甯掗湁闁挎繂鎳忛崯鐐烘煕閻斿搫浠﹂柕鍥у婵℃悂濡烽敂缁橈骏闂備礁鎽滄慨鎾晝閵堝鏁囬柛蹇曞帶缁剁偛鈹戦悜鍡樼窙缂佺姵鎹囧璇测槈閵忕姴宓嗛梺闈涱焾閸庤櫕绂掗埡鍛拺闁告稑锕ゆ慨鈧梺绋款儐閹瑰洤顫忕紒妯肩懝闁逞屽墮椤洩顦堕柛锝呯秺濮婃椽鏌呴悙鑼跺濠⒀屽灦閺岀喐绗熼崹顔碱潎閻庤娲栭悥濂搞€佸Δ浣瑰闁告瑥顦鍦磽閸屾艾鈧悂宕愰悜鑺ュ€块柨鏇炲€归弲顏嗙磽閸屾瑧鍔嶉柕鍥ф瀹曞爼濡歌楠炴劙姊绘担鍛婂暈缂侇喖鐗忕划鍫熸媴閸︻収娲告繛瀵稿帶閻°劑鎮￠弴銏＄厓閻熸瑥瀚崝銈吤瑰鍛壕濞ｅ洤锕幃娆擃敂閸曘劌浜鹃柡宥庡亝閺嗘粌鈹戦悩鎻掝仾妞ゆ劒绮欓弻銊╂偄閸濆嫅銏ゆ煟閵堝骸娅嶉柣鎿冨亰瀹曞爼濡歌婵洭姊虹紒妯诲鞍婵炶尙鍠栧濠氭晲閸涘倻鍠栧畷顐﹀礋椤掍胶妲┑鐘垫暩閸嬬偤骞嗗畝鍕獥婵娉涢悞?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锕傛煟閵忕媭鐓兼慨濠勭帛閹峰懘鎮烽柇锕€娈濇繝鐢靛仜瀵爼鎮ч悩鑼殾闁归偊鍨禍褰掓煙閻戞ê鐏╅柨娑欑洴閺岋絾鎯旈婊呅ｉ梺绋款儏閸婂骞戦姀銈呭耿婵炴垶鐟ч崢顏堟⒑閸撴彃浜濈紒璇插暣瀹曞疇銇愰幒鎾跺幐闂佺硶鍓濋悷銉╁几濞戙垺鐓涚€光偓鐎ｎ剛袦濡炪們鍨洪敃銏ゅ箖閵忋垻鐭欓悹渚厛閺嗏€斥攽閿涘嫬浜奸柛濞垮€栫粋宥咁煥閸繄鏌堟繛鏉戝悑濞兼瑩鎮為崹顐犱簻闁圭儤鍩婇崝鐔虹磼婢舵劖娑ч棁澶嬬節婵犲倸顏柣顓熷笚閵囧嫰濮€閳藉棙鐤佹繝娈垮枓閸嬫捇姊洪棃娑氬婵☆偅顨婇崺鈧い鎺嗗亾闁诲繑宀搁獮鍫ュΩ閵夘喗寤洪梺绯曞墲椤ㄥ懐绮昏ぐ鎺撯拺闁告稑锕︾粻姗€鏌涙惔娑樷偓婵嬬嵁閸儱惟闁挎柨澧介惁鍫ユ⒑閸涘﹤濮€闁哄倸鍊圭粋鎺楀閵堝棗鈧敻鎮峰▎蹇擃仾缂佲偓閳ь剛绱撻崒姘毙㈤柨鏇ㄤ邯婵″瓨鎷呴崜鍙夊兊闁荤姴鎼幖顐ょ玻濞戞﹩娓婚柕鍫濇鐏忕敻鏌涚€ｎ偆娲撮挊鐔兼煙闁箑鍘撮柡鈧禒瀣厽婵☆垵娅ｆ禒娑㈡煛閸″繑娅呴柍瑙勫灴椤㈡岸宕ㄩ鐐电潉闁诲氦顫夊ú姗€宕归崸妤冨祦婵☆垵鍋愮壕鍏间繆椤栨粌甯舵鐐茬Ф缁辨捇宕掑顑藉亾閻戣姤鈷旂€广儱顦崹鍌滄喐閻楀牆绗掗柛鎴犲█閺岋綁寮崹顔藉€梺缁樻尵閸犳牠寮婚悢鐓庣畾鐟滄粓宕甸悢鍝ョ闁告瑥顦扮亸锕傛煛瀹€瀣？濞寸媴绠撳畷婊嗩槼闁告帗绋戣灃闁绘﹢娼ф禒婊勭箾瀹割喖骞栭摶鐐烘煕閹扳晛濡锋俊鎻掔墛娣囧﹪顢涘▎鎺濆妳闂佸憡鏌￠崑鎾绘⒒閸屾艾鈧悂宕愭搴ｇ焼濞撴埃鍋撴鐐搭殔椤劑宕煎┑鍫㈠炊闂佺鍋愮悰銉╁春濡も偓鐓ゆい蹇撳珋閳哄啯鍠愰幖鎼厜缂嶆牠鏌￠崶銉ョ仾闁绘挻鐟╅弻锝夊箣閻愬棙鍨规禍鎼佸箥椤斿墽锛濋悗骞垮劚閹冲繘宕板鈧弻锛勪沪閻愵剛顦ㄧ紓浣虹帛缁嬫牠藝閺屻儲鐓欓柧蹇ｅ亜婵秶鈧鍠楁繛濠囥€侀弴銏犖ч柛銉㈡櫔缁辨娊姊绘担渚劸闁哄牜鍓熼幃鐤樄閽樻繈鏌ㄩ弬娆炬綗濞存粍绮撻弻鐔衡偓娑欘焽缁犳牠鏌涢妶鍛枠闁圭锕ら埞鎴犫偓锝庡亞閸橀潧鈹戦悙鑼闁诲繑绻堝鎼佹偄鐞涒€充壕閻熸瑥瀚粈鈧悗娈垮枛婢у酣骞戦姀鐘斀閻庯綆鍋掑Λ鍐ㄢ攽閻愭潙鐏ョ€规洦鍓熷畷婊堝箥椤斿墽锛濇繛杈剧稻瑜板啯绂嶆ィ鍐┾拺闁告稑锕ゆ慨鈧梺鍝勫€搁崐鍦矉瀹ュ拋鐓ラ柛顐犲灩瑜板嫰姊洪幖鐐插姌闁告柨绉舵禍鎼侇敇閻旂繝绨婚棅顐㈡祫缁茶姤绂嶆导瀛樼厸閻忕偛澧介埥澶嬨亜椤愶絿鐭掔€规洖宕灃闁逞屽墴閸╂稓鈧綆浜栭弨鑺ャ亜閺冨倶鈧寮ㄧ紒妯圭箚闁绘劖澹嗛惌娆愵殽閻愯韬柡浣规崌閹晠宕ｉ崒鍐ㄦ处閻撴洘绻涢幋鐐嗘垿宕抽悜鑺ョ厱闁靛鍔嶉鐘绘煙楠炲灝鐏╅柍钘夘槸铻ｉ煫鍥ㄦ尰鐠愶繝鏌嶈閸撱劎绱為崱娑樼；闁告稑鐡ㄩ崑鐔告叏濮楀棗澧绘繛鎾愁煼閺屾洟宕煎┑鍡樻闂佽娴氭禍婊嗗絹闂佹悶鍎崝宥夊Φ濠靛瀵犳繝闈涱儐閻撴洘銇勯幇鍓佺ɑ缂佲偓閸愵喗鐓熼柟鎯х－缁犲鏌＄仦鍓ф创濠碉紕鍏橀弫鎰板川椤撗呪偓鍐测攽閻樻剚鍟忛柛鐘崇墱缁棃宕奸弴鐐靛姦濡炪倖甯掗崰姘焽閹邦厾绠鹃柛娆忣檧閼拌法鈧娲樺ú妯肩紦娴犲搴婇幖瀛樼☉婵＄晫绱掑Δ鍐ㄦ灈闁糕斁鍋撳銈嗗笒鐎氼剟鎷戦悢鍝ョ闁瑰瓨鐟ラ悘鈺呭炊閹绢喗鈷戠紒顖涙礃椤庡棝鏌￠崨顔剧煉闁诡喗瀵х粭鐔煎焵椤掆偓椤繐煤椤忓嫪绱堕梺鍛婃处閸犳牠顢旈鐘电＝濞达絽澹婂Σ娲煕韫囨棏鐒介柣锔诲墯缁绘盯骞橀弶鎴犲姲闂佺顑嗛幑鍥蓟閿涘嫪娌柛鎾楀嫬鍨辨俊銈囧Х閸嬬偤鈥﹂崶顒€鐒垫い鎺戝濞懷囨煙鐠囇呯瘈鐎规洘濞婂璺衡枎閻愵剙鐦滈梻渚€娼ч悧鍡涘疮椤愶附鍊舵い蹇撴閸嬫捇宕楁径濠佸闂備礁鎲＄粙鎴︽晝閿濆洦宕查柛鈩冪⊕閻撶喖鏌熸潏鍓хɑ妞ゃ儱顦甸弻锝夊箻鐎电硶妲堥梻鍥ь樀閺屻劌鈹戦崱妯烘闂佽鍨伴悧鍡涘煘閹达富鏁嬮柛鈩冪⊕椤庡秵绻涢敐鍛悙闁挎洦浜獮鍐ㄢ枎閹垮啯鏅滈梺鍛婃磸閸斿本绂嶆ィ鍐╃厸鐎广儱楠告禍婵嬫煛閸℃鐭婇柍瑙勫灴閹晠骞撻幒婵呯磻闂備焦妞块崢鐓幬涘Δ鍛ラ柟鐑樺焾濞尖晠鏌ㄥ┑鍡樺櫢濠㈣娲樻穱濠囨倷椤忓嫧鍋撻幋锕€鍨傞柧蹇曟嚀閺嬪牏鈧箍鍎遍幏瀣倿娴犲鍙撻柛銉ｅ妿閳藉骞嗛悢鍏尖拺闁圭瀛╃壕鐢告煕鐎ｎ偅灏甸柍褜鍓氶鏍窗閺嶎厽鍊舵繝闈涱儏閻撴﹢鏌熸潏鍓х暠闁绘搫绻濋弻娑㈠焺閸愮偓鐣兼繛瀵稿У濞兼瑩鍩為幋锔藉€烽柛娆忣槸閻濇梻绱撻崒姘毙＄紒鑸佃壘椤曪綁顢曢姀鈺佹倯闂佸憡绮堢粈渚€骞冮幋锔解拺闁硅偐鍋涢崝妤呮煛閸涱喚绠為柡浣哥Т椤劑宕奸悢鍙夊缂傚倸鍊烽悞锕傚煟閵堝鏁傞柛鈩冾殢濡粌顪冮妶鍡樺蔼闁搞劌缍婇幆灞轿旈崨顔惧帗閻熸粍绮撳畷婊堟偄婵傚缍?context canceled闂?		// /v1/responses 闂?SSE 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掓禒楣冩⒑缁嬫鍎嶉柛濠冪箞瀵寮撮悢铏诡啎閻熸粌绉瑰畷顖烆敃閿旇棄鈧泛鈹戦悩鍙夊闁稿﹦鏁婚弻娑滅疀閹垮啯笑婵炲瓨绮撶粻鏍ь潖濞差亜宸濆┑鐘插暟椤︺儵姊虹拠鑼鐎光偓缁嬫鍤曢柡灞诲劜閸婄兘鏌ｉ幋鐐冩岸骞忓ú顏呪拺闁告稑锕﹂埥澶愭煥閺囨ê鍔滅€垫澘瀚板畷鐔碱敍濞戞艾骞堥梺璇插嚱閹儵宕樿椤ユ岸姊婚崒姘偓椋庣矆娓氣偓楠炴牠顢曢敂缁樻櫈闂佸憡绋戦悺銊╂偂閳ь剟姊洪幐搴ｇ畵妞わ富鍨堕幏鎴︽偄閸忚偐鍘搁梺鎼炲劦椤ユ挾澹曢崸妤佺厽闁靛牆鍊告禍楣冩⒒閸屾瑧绐旀繛浣冲洦鏅煫鍥ㄧ☉缁€瀣亜閹偣鍊楃粻鐑芥⒒閸屾艾鈧悂宕愰悜鑺ュ€块柨鏇炲€归崕鎴犳喐閻楀牆绗掔痪鎯х秺閺岀喖鎮ч崼鐔哄嚒缂佺偓鍎抽…鐑藉蓟閻旂厧绀堢憸蹇曟暜濞戙垺鐓冮梺鍨儏缁楁帡妫佹径鎰叆婵犻潧妫欓崳娲煕閻斿搫浠遍柡灞剧洴閹垽宕崟顏呭煕缂傚倷娴囨ご鍝ユ暜閻愬搫鐒垫い鎺戯功缁夌敻鏌涚€ｎ亜顏╅棁澶嬨亜閺囨浜鹃梺鍝勭灱閸犳牠銆佸璺虹劦妞ゆ帒鍊绘稉宥夋煛瀹ュ骸骞戦柍褜鍏涚欢姘嚕椤曗偓瀹曠厧鈹戠€ｎ亝鏆┑鐘垫暩婵炩偓婵炰匠鍥ㄥ亱闁糕剝蓱閸欏繘鏌ㄩ弮鈧崹婵堟崲閸℃稒鐓熼柟鏉垮悁缁ㄥ鏌嶈閸撴岸鎮ч弴銏╂晩闊洦姊荤弧鈧┑顔斤供閸撴盯鏁嶅☉銏♀拺閻熸瑥瀚崝銈咁熆瑜嶅ù閿嬬珶閺囥垺鍤冮柍鍝勫暟閿涙繃绻涙潏鍓у埌妞ゎ偅鐗犻、鏃堝幢濞嗘垹鏆ラ柣鐔哥矋濡啫顕ｇ拠娴嬫闁靛繒濮烽濠囨⒑鐟欏嫬绀冩繛鍛礈閹风娀鎮欏顔藉瘜闂侀潧鐗嗛崯顐︽倶椤忓牊鐓ラ柡鍥悘顏堟煙娓氬灝濮傞柛鈹惧亾濡炪倖甯掔€氼參鎮￠崘顔界厓閺夌偞澹嗛ˇ锕傛煛閸℃瑥浠遍柡宀嬬到閳规垿宕堕埡浣叉嫲婵犳鍠栭敃锔惧垝椤栨粎绠旈柣鏃傚帶閻掑灚銇勯幒鎴濐仼闁绘帒鐏氶妵鍕箳閹存績鍋撻崨濠勵浄婵犲﹤鐗婇悡鐘崇箾閼奸鍤欓柣蹇ョ節閺岋繝宕ㄩ鐑嗘殺闂侀€炲苯澧紒瀣浮閺佸啴鍩℃担鍙夌亖?OpenAI 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇顓滀簻闁哄啫鍊归崵鈧繛瀛樼矒缁犳牠寮诲☉銏犵疀闂傚牊绋掗悘鍫ユ倵閻熺増鍟炵紒璇插暣婵＄敻宕熼姘敤闂侀潧臎閳ь剙危閸儲鐓欓柛蹇撳悑缂嶆垿鏌ㄩ弴妯衡偓婵嬪春閻愬搫绠ｉ柣鎰皺閻ゅ洭姊绘担渚劸闁挎洏鍎辩叅闁绘棁鍋愬畵渚€鐓崶銊︾５闁稿鎹囬弫鎰償閳╁啰浜梻浣告惈椤戝嫮绮堟笟鈧崺鐐哄箣閿旇棄浜归梺鍛婄懃椤︻垶藝閳哄懏鈷戠紓浣股戠亸鐗堢箾閸欏鐒介柟骞垮灩閳藉濮€閻樻鍚呴梻浣虹帛閸ㄩ潧螞濞戙垹绀夐柣鎴ｅГ閳锋垿鏌涢敂璇插箺婵炲懏娲栭埞鎴︽倷閳轰椒澹曢梻鍌欑閹碱偊鎯屾径灞惧床婵犻潧妫涢弳锔姐亜韫囨挻鍣哄┑顖氼嚟缁辨帞鈧綆鍋掗崕銉︾箾閼测晜娅曠紒杈ㄦ崌瀹曟帒鈻庨幒鎴濆腐缂傚倷绶￠崳顕€宕归崼鏇炵畾闁告洦鍨奸弫宥夋煟閹扮増娑х紒渚婄畵閺岋絾鎯旈婊呅ｉ梺鍝ュУ椤ㄥ﹤鐣烽幇鐗堝€婚柤鎭掑劤閸樹粙姊洪棃娑氱疄闁搞劍妞藉鎶藉煛閸涱喚鍘卞┑鈽嗗灥椤曆呯箔瑜忕槐鎺撴綇閵婏箑纾抽悗瑙勬礃鐢帡鍩㈡惔銊ョ闁瑰瓨绻傞懙鎰節閻㈤潧校妞ゆ梹鐗犲畷浼村冀椤撗勬櫔濡炪倖鎸炬慨鐢革綖閺囥垺鐓熼柟閭﹀墻閸ょ喓绱掗埦鈧崑鎾绘⒒娴ｄ警鐒鹃悶姘煎亰瀹曟劙寮介銈囶槸闁硅偐琛ュ褔寮ㄦ禒瀣厽闁归偊鍓欑痪褔鏌嶇紒妯荤闂囧绻濇繝鍌氼伀闁活厼鐬肩槐鎺楊敊绾拌京鍚嬮梺璇″枙缁瑥鐣烽妸锔剧瘈闁告洦鍋勭粻銉╂⒒閸屾瑨鍏岀紒顕呭灦瀹曞綊宕奸弴鐔告珨闂傚倷鑳剁划顖滄暜閻愬搫纾婚柟鎹愬煐瀹曞弶绻涢幋娆忕仼闂佸崬娲﹂妵鍕箛閳轰胶浠肩紓浣哄Ь瀹曠數妲愰幘瀛樺濞寸姴顑呴幗鐢告⒑閸︻収鐒炬い顓犲厴閻涱喛绠涘☉妯虹獩濡炪倖鐗楃粙鎴︽偟娴煎瓨鈷戦柡鍌樺劜濞呭懘鏌涢悢璺哄祮鐎殿喗濞婃俊鐑芥晜鐠恒劎鐣鹃梻浣虹帛閸旓附绂嶅鍫濈劦妞ゆ埈鍓欓幊澶愬磻閹剧粯顥堟繛鎴炴皑閸旑垶鎮楃憴鍕缂傚秴锕ら悾閿嬬附缁嬪灝宓嗛梺缁樺姇閻°劏鈪查梻鍌氬€风欢姘焽瑜忛幑銏ゅ醇閵夈儳锛欓梺鍝勬川婵浜搁悽纰樺亾楠炲灝鍔氭い锔垮嵆閹繝寮撮悢缈犵盎闂佽婢樻晶搴ｇ矙婵犳碍鐓曢幖娣灪缁€瀣煛鐏炵晫效鐎规洦鍋婂畷鐔碱敆閳ь剙鈻嶉妶澶嬧拺缂備焦蓱鐏忣厾绱掔紒妯哄闁炽儻濡囬幑鍕Ω閿曗偓绾绢垶姊洪幆褏绠抽柟铏崌閵嗗倸煤椤忓應鎷洪梺鍛婄☉閿曘儵鍩涢幇顔炬／闁哄娉曟晥濡ょ姷鍋涢悧鎾翠繆濮濆矈妲奸梺鍝ュТ濡繈寮诲☉銏犲嵆闁靛鍎辩粻褰掓倵濞堝灝鏋旈柛鏂跨焸閸╃偤骞嬮敂缁樻櫓缂佺虎鍙冮弨閬嶅极閹€鏀介柨娑樺娴犳帞绱掗鐣屾噰闁绘侗鍣ｅ畷姗€顢欓懖鈺佸Ф闂備礁鎲￠崝蹇涘疾濞戙垹绀夐柛娑卞弾濞撳鏌曢崼婵囶棡闁抽攱甯￠弻锟犲椽娴ｉ晲鍠婇悗娈垮枟閹倸鐣峰鈧俊鎼佸閳╁啯婢戦梻鍌欒兌缁垶宕濆Ο闂寸剨婵炲棙鎸哥粻娲煟濡吋鏆╃痪鎯у悑閵囧嫰寮崶褌姹楅柤鍙夌墵濮婃椽宕ㄦ繝鍌滅懖闁汇埄鍨界换婵嬫偘椤斿槈鐔煎礂閻撳海褰撮梻浣藉亹閳峰牓宕滃☉銏犳辈妞ゆ劏鎳囬弨浠嬪箳閹惰棄纾圭憸蹇擃嚗婵犲啰顩烽悗锝庝簽閺屽牆顪冮妶鍡欏⒈闁稿绋撴竟鏇°亹閹烘挴鎷洪梺鍓茬厛閸ｎ噣宕曞鍚ょ懓顭ㄩ崟顓犵厜闂佸搫鐬奸崰鏍箖濠婂吘鐔兼惞闁稒妯婇梻鍌欑窔閳ь剛鍋涢懟顖涙櫠娴煎瓨鐓涘ù锝囩摂閸ゆ瑦銇?error event闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灪閵囧嫰濡搁妷顖濆惈閻庢鍠涢褔鍩ユ径濠庢僵妞ゆ劧绲芥刊浼存⒒娴ｅ憡鍟為柟绋挎閸┾偓妞ゆ巻鍋撻崡閬嶆煕椤愶絿绠ユ繛鎾愁煼閺屾洟宕煎┑鍥舵！缂備讲鍋撻悗锝庘偓銏㈡嚀椤劑宕橀鍕幗闁诲孩顔栭崰娑㈩敋瑜旈、姗€宕楅悡搴ｇ獮婵犵數濮抽懗鍫曟倷婵犲洦鈷掑ù锝呮啞閸熺偞绻涚拠褏鐣电€规洘绮岄埢搴ょ疀婵犲啰鈧椽姊洪幐搴ｇ畵婵炲眰鍔庢竟鏇㈡寠婢规繂缍婇弫鎰板幢濡ゅ啰銈峰┑鐐茬摠缁挾绮婚弽褜娼栭柧蹇撴贡閻瑩鏌熺粙鍨劉闁圭柉椴哥换婵嬪閿濆孩缍堝┑鐐跺皺閸犲酣鎮鹃悜钘夌闁挎洍鍋撶紒鐘差煼閺屻倖鎱ㄩ幇顑藉亾閺囩姷鐭堥柨鏇炲€归埛鎴︽煕閹剧懓鐨洪柛妯荤洴閺屾盯鎮╁畷鍥ь潷闂侀€涚┒閸斿秹骞嗛弮鍫澪╅柨鏃€鍎抽獮妤呮⒑閻熸澘鎮戦柣锝庝邯瀹曠銇愰幒鎴濇優濡炪倖甯掔€氼參鎮￠弴銏＄叆婵犻潧妫涚粻宕囩磼婢舵ê鏋熸い銊ｅ劦閹瑩骞撻幒鎾搭啋闂備浇顕栭崰妤冨垝閹捐绠板┑鐘插暙缁剁偟鈧厜鍋撳┑鐘插€瑰▍鏃堟⒒閸屾瑦绁版い鏇嗗洦鐓€闁挎繂鎷嬮悞鐣屾喐閺冨牏宓佹俊銈傚亾妞ゎ厹鍔戝畷濂割敃閿濆棙鐝?SDK 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呯＜闁归偊鍙庡▓婊堟煛瀹€鈧崰鏍嵁瀹ュ鏁婄痪鎷岄哺濮ｅ姊绘担渚劸妞ゆ垶鍨归幑銏犫攽閸♀晛娈ㄩ梺鍓插亝濞叉牠鏌嬮崶銊﹀弿婵妫楅獮妤呮煟濠靛洦鈷掔紒杈ㄦ尰閹峰懘鎮剧仦鐣屽闂備胶顭堥敃銉ッ哄┑瀣€堕柛鎰靛枟閳锋垿鏌熺粙鎸庢崳缂佺姵鎹囬弻鐔煎礃閺屻儱寮伴悗娈垮枟婵炲﹪骞冨▎鎾村€绘俊顖滃帶楠炲牆鈹戦悩鍨毄濠殿喖顕埀顒佸嚬閸欏啫顕ｉ幎绛嬫晢闁告洦鍓涢崢鎼佹煟韫囨洖浠╂い鏇嗗洤鐒垫い鎺嶈兌缁犵偤鏌ｅ☉鍗炴灍缂佹鍠栭崺鈧い鎺戝瀹撲線鏌熼悜姗嗘當缂佺媴绲剧换婵嬫濞戞瑱绱炲┑鐐插悑缁嬫挾妲愰幘璇茬＜婵﹩鍏橀崑鎾搭槹鎼达絿鐒兼繛杈剧秬椤鈻嶉悩璇茬婵烇綆鍓欐俊鑲╃棯閹岀吋闁哄本娲熷畷鐓庘攽閸パ屸偓娑㈡⒑閹肩偛鈧牕煤閻旂厧钃熸繛鎴欏焺閺佸啴鏌ㄥ┑鍡橆棤妞わ负鍎靛缁樻媴缁嬭儻鍩炲┑鐐额嚋缁犳挻淇婇悽绋跨妞ゆ牗姘ㄩ悿鈧梻浣稿閻撳牓宕戦崱娆戠幓闁哄啫鐗婇埛鎴︽煕濞戞﹫宸ラ柣蹇ョ悼缁辨帡顢欓悾灞惧櫚濡ょ姷鍋涢崯浼村箲閸曨剛鐟规い鏍ㄧ☉閸忓﹥淇婇悙顏勨偓鏍箰閻愵剚鍙忛柣鎴ｅГ閸嬵亪鏌涢弴銊ヤ簮闁衡偓?
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
		// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟撮崣鍕煏閸℃鏆ｅ┑锛勫厴閸┾剝鎷呮搴ｅ€為梻鍌欑窔濞佳囨晬韫囨稑纾兼繝濠傛噺閸犳帡姊绘担绛嬪殭濡ょ姴鎽滅划璇差吋婢跺﹦锛熼梻渚囧墮閸楁洟宕堕澶嬫櫖闂佺粯鍔栬ぐ鍐倵椤撱垺鈷戠紒瀣濠€鎵磼鐎ｎ偅灏电紒顔碱煼瀹曟粏顦柛瀣崌閹兘寮跺▎鐐棏闂備礁鎽滄慨闈浢哄鍫熷殟閺夊牄鍔庣弧鈧┑顔斤供閸橀箖宕㈤幖浣光拺闁告稑锕ョ壕鐢告煛閸涱喚娲寸€规洘绻傞…銊╁川椤栨粣绱茬紓鍌氬€烽悞锕傗€﹂崶顬¤櫣鈧稒锕╁▓浠嬫煟閹邦剚鈻曢柛搴㈡閺岀喖顢欓悾灞惧櫘闂侀€炲苯澧存繛浣冲洤绠熼柨鐔哄閺佸洤鈹戦悩宕囶暡闁抽攱甯掗湁闁挎繂鎳忛崯鐐烘煕閻斿搫浠﹂柕鍥у婵℃悂濡烽敂缁橈骏闂備礁鎽滄慨鎾晝閵堝鏁囬柛蹇曞帶缁剁偛鈹戦悜鍡樼窙缂佺姵鎹囧璇测槈閵忕姴宓嗛梺闈涱焾閸庤櫕绂掗埡鍛拺闁告稑锕ゆ慨鈧梺绋款儐閹瑰洤顫忕紒妯肩懝闁逞屽墮椤洩顦归柍銉畵瀹曞ジ濡烽妷褜妲烽柣搴″帨閸嬫捇鏌涢弴鐑囧伐妞わ富鍨堕崺鐐哄箣閻橆偄浜鹃柨婵嗛娴滄劙鏌熼柨瀣仢婵﹨娅ｅ☉鐢稿川椤撴繃鐫忛梻浣侯焾椤戝棝鎯勯鐐叉槬闁逞屽墯閵囧嫰骞掑鍫濆帯婵犫拃鍥︽喚闁哄备鍓濈粭鐔煎炊瑜庨悵婵嬫⒑鏉炴壆鍔嶉柟鐟版喘楠炲啴鍩￠崨顔兼闂佽偐顭堥悘姘跺疮濞差亝鈷掗柛灞剧懄缁佺増绻涙径瀣鐎规洑鍗抽獮妯尖偓娑櫭鍧楁煟鎼达絾鏆╂い顓炵墦钘熸慨姗嗗厴閺€浠嬫煕鐏炲墽顣查柛鐔哄仱閺岋綁骞樺畷鍥╊啋闂佸搫鏈惄顖炪€侀弴銏℃櫜闁搞儮鏅濋弳顐︽⒒娓氣偓濞艰崵寰婇挊澶涜€块弶鍫氭櫆椤洟鏌熼悜姗嗘畷闁稿鍔欓弻娑樷枎韫囷絾歇缂佺虎鍘搁崑鎾剁磽閸屾艾鈧嘲霉閸ャ劊浠堥柟闂寸缁犳澘鈹戦悩瀹犲缂佺媭鍨堕弻娑樷槈閸楃偞鐏撳銈傛櫇閸忔﹢骞冨Δ鍛櫜閹煎瓨绻勯崙褰掓⒑绾懏鐝柣鐔叉櫅椤繒绱掑Ο璇差€撻柣鐔哥懃鐎氼剚绂掗埡鍛拺闁告稑锕ラ悡銉╂煟椤撶偛鈧潡鐛崘銊庣喓鎮伴埄鍐╂澑闂佽鍑界徊浠嬫倶濮樿泛鐤炬い蹇撶墛閳锋垿鏌熺粙鎸庢崳缂佺姵鎸婚妵鍕晜鐠恒劎鐓撻悗娈垮櫘閸嬪懐鎹㈠┑瀣倞闁靛ě鍐ㄧ闂傚倷绶氶埀顒佺☉瀹撳棙绻涙担鍐插濞呯姵銇勯弬娆炬綗濞存粍绮撻獮鏍庨鈧悘顔界箾閹绘帞鎽犻柟渚垮妽缁绘繈宕橀埞澶歌檸婵犵妲呴崑鍕疮閹绢喖鏄ラ柍鈺佸暞婵挳鏌ｉ幋鐑嗙労闁圭儤顨嗛埛鎺懨归敐鍛暈闁哥喓鍋ら弻銊╁即濡櫣浼勭紓渚囧枛椤嘲顕ｆ禒瀣垫晢濞撴艾娲﹂敍鍫熺節绾板纾块柛瀣灴瀹曟劙寮借濞兼牕鈹戦悩瀹犲闁汇倗鍋撻妵鍕箛閳轰讲鍋撳Δ浣衡枖鐎广儱鎲橀弮鍫熸櫜闁告侗鍘藉▓鏌ユ⒑缂佹ɑ灏伴柣鐔叉櫊楠炲啫螖閸涱喖浠哄┑鐐茬墣濞夋洟宕濋崨濠勭閻庣數顭堥鎾剁磼閻樿櫕宕屾鐐插暙铻ｉ悹鎭掑妿閺夋悂姊洪幐搴ｇ畵闁瑰啿娴烽悮鎯ь吋婢跺鎷洪梺鍛婄箓鐎氼垶锝為敃鍌涚厱闁哄啠鍋撻柛銊ф暬椤㈡岸鏁愭径濠勯獓闂佺懓鍟跨壕顓㈠窗閺嵮呮殾妞ゆ劧绠戠粈瀣亜閹邦剛校婵炶尙鍠庨～蹇曠磼濡顎撶紓浣圭☉椤戝懎鈻撻幇鐗堚拺闁告縿鍎辨牎闂佸湱鎳撳ú銈夋偩閻戣棄鍗抽柕蹇婃濡啫鈹戦悙鏉戠仸缁炬澘绉撮埢鎾绘倷椤掑倻鐦堥梻鍌氱墛缁嬫垿鍩€椤掆偓椤兘鍨鹃敃鍌氶唶闁靛鍎抽、鍛存⒑鐟欏嫬鍔跺┑顔哄€濋幃锟犳偄閸忚偐鍘甸梻渚囧弿缁犳垵顕ｆィ鍐╃厱闁绘棃顥撶粻鎻捛庨崶褝韬柟宕囧█瀹曞ジ寮撮悙瀛樺殘缂傚倸鍊烽懗鍓佸垝椤栨粍宕查柛顐犲劚缁犳牕霉閻樺樊鍎愭い銉ョ墛缁绘盯骞嬮悜鍡樼暭闂佺顫夊ú妯兼崲濠靛牆鏋堟俊顖涙た濞兼垿姊洪幖鐐插闁告濞婂顐﹀礃椤斿槈銊ф喐韫囨稒鍎楀璺哄閸嬫捇鐛崹顔煎闂佺懓鍟块ˇ闈涱嚕閵婏妇顩烽悗锝庡亜閳ь剛鏁婚弻銊モ攽閸℃瑥鍤紓浣靛妺缁瑩寮婚敍鍕勃闁绘棃鏀辨径鍕煃闁垮鐏撮柟顔筋殜瀹曠兘顢橀悙鎰剁秮閺屾洟宕卞Δ鈧弳锝夋煛瀹€鈧崰鏍€侀弽顓炵畾鐟滃秵鎱ㄩ妶澶嬧拺闂傚牊绋掓径鍕煕閺冣偓閻熲晠鎮伴鈧獮鎺懳旈埀顒傜不婵犳碍鐓曢煫鍥ㄦ尰閸ｆ椽鏌ｉ銏㈢婵﹤顭峰畷鎺戔枎閹搭厽袦闂備胶顢婇婊呮崲濠靛绠栭柣鎴ｆ鍞梺鍐差嚃閸愨晛鏋犲銈冨灪缁嬫垿锝炲┑瀣櫜闁糕剝鐟㈤崑鎾趁洪鍛嫼闂佸湱顭堝ù椋庣不閹剧繝绻嗘い鎰剁悼閵嗘帞鈧鍟崶褏鍔﹀銈嗗笒鐎氼參鍩涢幒妤佺厱閻忕偟鍋撻惃鎴濐熆瑜庣粙鎾舵閹烘柡鍋撻敐搴′簻闁诲繑鎸抽弻娑㈠煘閹傚濠碉紕鍋戦崐鏍ь啅婵犳艾纾婚柟鎯у绾剧晫鈧箍鍎遍幏鎴︾叕椤掍緡娈介柣鎰絻閺嗐垺銇勯敃鈧紞濠囧蓟閻旂⒈鏁婇悷娆忓閻濇岸姊洪悷鎵紞濠电偐鍋撳銈冨灪椤ㄥ棝骞忛崨顖涘珰闁炽儴娅曞▍妤€鈹戦悩娈挎殰缂佽鲸娲熷畷鎴﹀箣閿曗偓绾惧綊鏌″搴″箹缂佲偓婢舵劖鐓欓弶鍫濆⒔閻ｉ亶鏌￠崟鈺佸姦闁哄本鐩鎾Ω閵壯傚摋闂佽崵鍠愰悷杈╃不閹捐绠栨俊銈呮噺閺呮煡骞栫划鐟板⒉闁诲繐绉瑰铏圭磼濡闉嶅┑鐐跺皺閸犳牕顕ｆ繝姘櫢闁绘灏欓崐鐐烘⒑闂堟侗妲堕柛搴ㄤ憾閳ユ牠宕煎顏呮閹晠妫冨☉妤佸媰闂備焦瀵х喊宥夊Φ閸曨垼鏁冮柣妯垮皺娴煎牓鎮楃憴鍕闁稿骸銈歌棟妞ゆ洍鍋撻柡宀嬬節閸┾偓妞ゆ帊鑳堕々鐑芥倵閿濆簼绨芥い鏂匡躬濮婅櫣鎲撮崟顐㈠Ц濠碘槅鍋勭€氼喗绔熼弴鐘冲枂闁告洦浜炵粻姘舵⒑闂堟稓澧曢悗姘煎櫍瀹曟繂顓兼径瀣幈濠电偛妫楃换鎺旂不婵犳碍鐓涚€光偓閳ь剟宕伴幘鑸殿潟闁圭儤顨呴～鍛存煟濡櫣锛嶅ù婊冪埣濮婄粯鎷呮笟顖滃姼闁诲孩绋堥弲鐘诲Υ閸愵喖宸濈紒顔煎帨閸嬫捇鏁傞悙顒€纾梺闈涱煭缁犳垹澹曢鐐粹拺缂備焦锚閻忥箓鎷戦柆宥嗙厵鐎瑰嫭婢橀懜瑙勩亜椤忓嫬鏆ｅ┑陇鍩栭幆鏃堝灳閼碱剙鍤┑鐘垫暩閸嬫稑顕ｉ崼鏇椻偓锕傚炊椤掆偓缁犳煡鏌曡箛鏇炐涢柡鈧禒瀣€甸柨婵嗛娴滆姤淇婇銏犳殭闁宠鍨块幃娆撳级閹寸姳妗撻梻浣藉吹閸犲棝宕曞畷鍥у灊闁挎繂鎳夊Σ鍫ユ煏韫囧﹥顎嗛柟閿嬫そ閹鎲撮崟顒傤槬濠电偛鐪伴崝搴ㄥ极椤斿皷妲堟繛鍡樺姇瀵寧绻濋悽闈浶㈤柛瀣閹剝绺介崨濠勫幈闂佸疇顫夐崕铏閻愵兛绻嗛柣鎰典簻閳ь剚鐗曢蹇旂節濮橆剛锛涢梺瑙勫劤婢у海澹曟總鍛婄厽婵☆垰鐏濋惃娲极閸儲鍊甸悷娆忓缁€鍐╃箾閼碱剙鏋涚€殿喖顭峰畷鍗炍旀繝鍌涘€梻浣告啞娓氭宕归幎钘夊嚑闁告劦鍠楅埛鎴︽煟閻斿憡绶查柟顔藉灴閺岋綁鏁愭惔婵嬪仐閻庤娲樼换鍐崲濠靛鐐婄憸蹇涘礉閿曗偓椤啴濡堕崱妤€娼戦梺绋款儐閹瑰洭寮诲☉銏″亞濞达綁鏅查崰濠囨⒑缁洘鏉洪柛搴㈠絻椤曘儵宕熼姘辩杸濡炪倖甯掗ˇ鎵礊閸岀偞鈷掑ù锝囨嚀椤曟粎绱掔€ｎ偄鐏╅柍褜鍓氶崙褰掑礈閻斿吋鍋樻い鏇楀亾鐎规洖銈搁幃銏ゆ惞閸︻厽顫岄梻鍌欑劍閻綊宕归挊澶樼劷鐟滄棃宕洪姀銈呴唶闁绘棁娅ｉ鏇熺箾鏉堝墽鍒伴柛鐔锋健瀹曘儱螣鐞涒剝顫嶅┑鈽嗗灦閺€閬嶏綖瀹ュ應鏀介柍钘夋閻忥絿绱掗鍛仸闁诡垯鐒﹂幆鏃堝閵忋垻妲囩紓浣哄亾濠㈡﹢藝鏉堚晛顥氶柛褎顨嗛悡鏇㈡倵閿濆骸浜滈柣蹇擃嚟閳ь剝顫夊ú姗€宕濆▎蹇ｅ殨濞寸姴顑愰弫鍥煟閹邦収鍟忛柛鐐茬埣濮婄粯鎷呴崫銉ㄥ┑鈽嗗亯濞夋洜鍒掗崼鐔稿闁惧繘鈧稓鐟濋梻浣虹帛閸ㄥ爼鈥﹂崶鈺佸К闁逞屽墯缁绘繈鎮介棃娴躲垽鏌涙繝鍕笡闁哄懎鐖煎鎾偐閻㈢绱查梻浣虹帛閻熴垽宕戦幘缁樼厱闁靛绠戦埢鏇燁殽閻愬樊妯€妤犵偞鐗楅幏鍛存偡妫颁胶缍嶉梻鍌欑婢瑰﹪宕戦崨顖涘床闁告洦鍨遍崑锟犳煛鐏炶鍔滈柍閿嬪灩缁辨帞鈧綆浜濋崑銉︺亜鎼淬埄娈滈柡宀嬬秮椤㈡﹢鎮欏ù瀣壕闁割煈鍣崵鏇㈡偣閸ャ劎銈存俊鎻掔墛娣囧﹪顢涢悙瀛樻殸闂佸搫鍊甸崑鎾绘⒒閸屾瑧顦﹂柟娴嬧偓瓒佹椽鏁冮崒姘憋紱闂佸憡娲﹂崜姘跺矗韫囨稑绠规繛锝庡墮婵′粙鏌嶉柨瀣伌闁哄本绋栫粻娑㈠箻閾忣偄鈧垳绱撴担鑺ョ【鐎殿喖澧庨幑銏犫槈濮橆厼鍘规俊鐐差儏濞寸兘鎯侀崼銉﹀€垫繛鍫濈仢閺嬫稒銇勯鐘插幋鐎规洘妞藉畷鐔碱敍濮橀硸妲伴梻渚€娼ц噹闁告侗鍨扮粊锕傛⒒閸屾艾鈧兘鎳楅崜浣瑰厹闁割偅娲栫粈鍫熺節闂堟冻鏀婚柟鍐茬焸濮婄粯鎷呴搹鐟扮缂備浇顕ч崯顐ゅ弲闂佺粯鏌ㄩ〃搴☆焽閺嶃劎绠剧€瑰壊鍠曠花璇裁归懖鈺佲枅闁哄本鐩鎾Ω閵夛妇褰繝鐢靛仧閸樠囨偉閻撳寒娼栨繛宸簻瀹告繂鈹戦悩鎻掓殶闁告瑥妫濆鍝劽虹拠鎻掝潻闂侀潧鐗忛…鍫ユ偩閻戣姤鏅插璺猴工缁愭盯鏌ｆ惔銏⑩姇閼裤倝鏌熼柨瀣仢婵﹥妞藉畷銊︾節閸曨厾鏆ら梺璇插閸戝綊宕归崼鏇炵畾闁稿本绋撻悷褰掓煃瑜滈崜鐔肩嵁閸愵煈娼ㄩ柍褜鍓熼獮鍐ㄢ枎閹寸偛纾梺鐑╂櫆鐢骞愰懡銈嗗床婵炴垯鍨瑰婵囥亜閹惧崬濮€闁告凹鍋勯埞鎴︻敊绾嘲濮涚紓渚囧櫘閸ㄥ爼鐛箛娑樺窛闁哄鍨电粣娑欑節閻㈤潧孝闁稿﹦鎳撻埢鎾愁潨閳ь剙顫忕紒妯诲闁芥ê顦幆鐐烘⒑閸濄儱孝婵☆偅绻傞悾鐑藉即閻旂顎撻梺鍛婂姀閺呪晛螞閵婏妇绡€闁汇垽娼ф禒锔界箾閸忚偐鎳囩€规洏鍎抽埀顒婄秵娴滃爼鎮為懞銉х闁糕剝顨夐鐔烘喐閺傝法鏆﹂柨婵嗘缁剁偤鏌涢锝囩煂闁烩晛閰ｅ缁樼節鎼粹€茬盎濠电偠顕滅粻鎾荤嵁閹扮増鍊锋い鎺嶇劍閻濆嘲鈹戦悙鏉戠仧闁搞劍妞介幃?usage
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

			// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厾缁炬澘宕晶缁樹繆閼碱剙鍘存慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍缂傚倷绀侀鍕濮橆剛鏆﹂柕蹇ョ磿椤╃兘鎮楅敐搴濈敖闁挎稒鐩铏规崉閵娿儲鐏佹繝娈垮枤閺佸骞冮檱缁犳盯骞欓崘顏勬暩闂佽崵濮惧▍锝夊磿閵堝＆澶愭倷鐎靛摜顔曢梺鍛婁緱閸樹粙宕甸崶顒佺厸鐎光偓鐎ｎ剛袦濡炪們鍨洪敃銏ゅ箖閵堝棙濯撮柛锔诲幗濠㈡帡姊婚崒娆戭槮闁圭⒈鍋婂鐢割敆閸曨剙鍓銈嗙墱閸嬫盯寮伴妷鈺傜厱闁哄洢鍔岄悘鐘电磼閳ь剟宕橀埞澶哥盎闂婎偄娲﹂幐濠氬闯瑜版帗鐓涢悗锝呭閻ｅ灚鎱ㄦ繝鍕笡闁瑰嘲鎳忕粭鐔碱敍濠婂啫歇濠碉紕鍋戦崐鏍垂闂堟党娑樷攽鐎ｎ剙绁﹂梺纭呮彧缁犳垿鎮欐繝鍐︿簻闁瑰搫绉堕ˇ锕€霉閻樺啿绗掓い顏勫暣婵″爼宕卞Ο纭风穿闂備胶顭堥鍡涘箰妤ｅ啫鐒垫い鎺戝枤濞兼劖绻涢崣澶屽⒈缂佽京鍋炵换婵嬪炊閵夈垹浜惧ù锝囩《閺嬪酣鏌熼幆褏锛嶉柣鎾村灴濮婃椽妫冨☉宕囩闂佸憡娲栨晶搴ｆ兜閳ь剟姊婚崒姘偓鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌ｉ幋锝呅撻柛銈呭閺屾盯骞橀懠顒€濡介梺绋跨箲缁捇寮诲☉銏╂晝闁挎繂妫涢ˇ銉╂⒑濮瑰洤鈧宕戦幘鑸靛床婵犻潧娲ㄧ弧鈧梺绋挎湰缁嬫垵鈻嶈濮婂搫煤鐠囨彃绠洪梺鑽ゅ暱閺呯姴顕ｆ繝姘亜闁绘挸娴烽澶愭⒑瑜版帒浜伴柛姗€绠栨俊鎾箛閻楀牃鎷洪梺纭呭亹閸嬫稒淇婇悾宀€纾奸悹鍥皺婢э妇鈧鍠涢褔顢橀崗鐓庣窞濠电姳鑳堕悙濠囨⒒娴ｅ憡鍟炴繛璇х畵瀹曟粌鈻庨幙鍐╂櫅濠电偞鍨崹娲煕閹达箑绾ч柣鎰綑椤ュ銇勯敂鐓庮洭缂佽鲸甯楀鍕節閸曨厜銊╂倵濞堝灝鏋涙い顓炲槻椤曪綁骞橀纰辨綂闂佺粯顭堥褔寮憴鍕瘈闁汇垽娼ч埢鍫熺箾娴ｅ啿鍘惧ú顏勎ч柛銉㈡櫇楠炴挸鈹戦悙鏉戠仸闁挎洍鏅涢妴鎺撶節濮橆厾鍘梺鍓插亝缁诲嫮绮诲ú顏呯厱婵﹩鍓欓埀顒€娼″濠氭晸閻樻彃绐涘銈嗙墬閸╁啴寮搁崨瀛樺€垫繛鍫濈仢閺嬫稒銇勯鐘插幋鐎殿噮鍋婇獮鍥级閸喛鈧灝鈹戦埥鍡楃仩闁圭⒈鍋呮穱濠囧箮閼恒儮鎷绘繛杈剧悼閹虫捇顢氬鍛＜閻庯綆鍋勯悘鈺呮煃瑜滈崜姘跺礄瑜版帒闂柨婵嗘媼閸ゆ洟鏌涢幇顓犮偞婵℃彃鐗撻弻鏇＄疀婵犲喚鈧棝鏌熼柨瀣仢婵﹨娅ｅ☉鐢稿川椤撴繃鐫忛梻浣侯焾椤戝棛绮欓幒鎾垛攳濠电姴娲﹂崵宥夋煏婢跺牆鈧绮诲鑸碘拺缂備焦锚婵矂鏌涢埡鍌滃⒌閽樼喎顭块懜闈涘闁绘挾鍠栭弻銊モ攽閸℃ɑ鎮欓梺鍛娒顓㈠焵椤掑喚娼愭繛璇х畵瀹曟垶绻濋崒婊勬闂佺粯姊婚崢褎鍎梻浣瑰濮婅崵鍒掗婊呯閻忕偠袙閺€浠嬫煟濡澧柛鐔风箻閺屾盯鎮╅幇浣圭杹闂佽桨绀侀澶愬箖濡ゅ啯鍠嗛柛鏇ㄥ墮椤︹晠姊洪崨濠冨鞍闁荤啿鏅犻獮鍐┿偅閸愨晛鈧鏌﹀Ο鐚寸礆闁靛ň鏅滈悡蹇擃熆閼稿緱顏堝几閻斿吋鍊甸梻鍫熺〒閻掑憡鎱ㄦ繝鍐┿仢婵☆偄鍟埥澶婎潩椤掑姣囧┑鐘殿暯濡插懘宕戦崨顖滅煓闁圭儤顨呴悿楣冩煕椤愶絾澶勯柡浣告閺屾稓浠﹂崜褏鐓傞梺鎸庣⊕缁捇寮婚敐鍡樺劅妞ゆ牗绮庢牎闂備胶顭堥鍛偓姘煎墴濠€浣割渻閵堝棙纾甸柛瀣崌閺屸€崇暆閳ь剟宕伴弽顓溾偓浣糕枎閹惧磭鐣鹃悷婊冭嫰鍗遍柟鎵閳锋垿鎮归崶锝傚亾閾忣偆浜┑鐘媰閸℃ぅ鎾剁磼閸屾稑娴柡浣稿€垮畷婊嗩槾闁绘挻鎸荤换婵嬪閻樺樊鏆㈠銈庡幖閻楁捁妫㈠┑顔斤供閸樺墽寮ч埀顒勬⒑濮瑰洤鐏叉繛浣冲嫮顩锋繝濠傜墛閻撶喐淇婇妶鍌氫壕濠碘槅鍋呯换鍫ョ嵁閸愩剮鐔烘偘閳╁啯鏉搁梺璇插嚱缂嶅棙绂嶅鍫濆惞闁硅揪闄勯埛鎴︽煟閻斿憡绶查柍閿嬫⒒缁辨帡顢氶崨顓犱桓闂佽鍣换婵嗩嚕閹绢喖顫呴柍銉︽灱閸嬫捇鎮介崨濠勫弳濠电娀娼уΛ婵嬵敁濡も偓闇夋繝濠傚缁犳﹢鏌嶈閸撴繈锝炴径濞掓椽寮介鐔峰壒闂佺鐬奸崑娑㈡嫅閻斿吋鐓ユ繛鎴灻褎绻涘畝濠侀偗闁哄本鐩獮妯侯渻鐠囪弓澹曟繝纰樺墲瑜板啴鎮ч崱娑掆偓鏃堝礃椤斿槈褔骞栫划鍏夊亾閼碱剙鍤紓鍌氬€峰ù鍥敋瑜旈幃褔骞樼拠鍙傘儱霉閿濆牆鈧粙寮埀顒勫箯閸涘瓨鍋￠梺顓ㄨ吂閸嬫捇骞樼紒妯锋嫼闂佺厧顫曢崐鏇㈠几瀹ュ鐓曟俊銈傚亾闁哥喕鍎婚悘瀣⒑缂佹ê鐏卞┑顔哄€濋崺娑㈠箳濡や胶鍘遍柣蹇曞仜婢т粙骞婇崟顓犵＜濞达絽鎼。濂告煏閸パ冾伃妞ゃ垺娲熸慨鈧柍鍝勫€甸弸鍛繆閵堝洤啸闁稿鍋ゅ畷婵嗏枎閹捐泛绁﹂柟鍏肩暘閸斿瞼绮堥崼銉︾厱妞ゎ厽鍨靛▍妯讳繆椤栨浜惧┑掳鍊楁慨鐑藉磻閻愮儤鏅濋柕蹇嬪€曠粻鐘崇節婵犲倸鏆婇柡瀣閺岀喓绮欓幐搴㈠闯缂?drain 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｆ繝姘亜闁稿繐鐨烽幏濠氭煟鎼达絾鏆╂い顓炵墛娣囧﹥绂掔€ｎ偀鎷洪梺鍛婄箓鐎氼參宕抽搹鍦＜閻犲洩灏欐晶锔锯偓娈垮枛椤嘲顕ｉ幘顔藉亜濡炲娴烽悰顕€姊绘担铏广€婇柛鎾寸箚閹筋偊姊虹紒妯肩畺婵炶尙鍠庨～蹇涙惞閸︻厾鐓撳┑鐐叉閸庢娊宕滈弶娆炬富闁靛牆绻愰々顒勬煛娴ｇ瓔鍤欐い鏇悼閹风姴霉鐎ｎ偒娼旈梻渚€娼х换鍡涘疾濠婂牆鐤炬繝闈涱儐閳锋垿鏌熺粙鎸庢崳缂佺姵鎸绘穱濠囶敃閿濆洦鍒涢柦妯荤箞閺屾洘绻涢悙顒佺彆闂?
			if !clientDisconnected {
				shouldFlush := queueDrained && (clientOutputStarted || startsClientOutput)
				if firstTokenMs == nil && startsClientOutput {
					// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤濞硷繝寮婚悢琛″亾閻㈡鐒剧€涙繄绱撴担鐣屽牚闁稿﹥绻堝濠氭晝閳ь剝鐏掓繛鎾村嚬閸ㄩ亶鏁嶉崱妞绘斀闁绘劕寮堕埢鏇灻瑰鍕疄鐎殿喗褰冮…銊╁醇閻斿搫骞楅梻渚€鈧稑宓嗘繛浣冲嫭娅犳い鏇楀亾妤犵偞鐗犲鍫曞箣椤栨繂鎯堟繝娈垮枛閿曘劌鈻嶉敐澶婄闁哄稁鍘奸崡鎶芥煟閹邦厾銈撮柟鏋€楃槐鎾诲磼濞嗘埈妲銈嗗灥濡盯寮鈧崹鎯х暦閸ャ劍顔曢梻浣告惈濞层劑宕伴幘璺哄К闁逞屽墴濮婅櫣绮欓懗顖ｆ蕉闂佺锕ラ〃濠傜暦椤掑嫭鍋ㄩ柛娑橈功閸樹粙姊虹紒姗嗙劷闁稿簺鍊濇俊鎾礃椤旇棄浠梺閫炲苯澧撮柡浣稿暣瀹曟帒顫濋幉瀣惞濠电姷鏁告慨鎾晝閵堝绠犻柟閭﹀枛椤ユ艾鈹戦崒婊庣劸缂佺嫏鍥ㄧ厱妞ゆ劧绲块埥澶娾攽闄囨慨銈夊Φ閸曨垰唯闁靛／鍐ｅ徍闂備礁鎼惌澶屾閺囩喓顩烽柨鏃€鍎崇€垫煡鏌￠崶鈺佷粧濠㈣娲熷楦裤亹閹烘繃娈ョ紓浣插亾濞撴埃鍋撴鐐插暣楠炲鏁傜粵瀣棅婵＄偑鍊栭崝鎴﹀垂閻戞ê绶為柛鏇ㄥ墰缁犻箖鏌涘▎蹇ｆШ闁活厽甯為埀顒侇問閸犳绻涙繝鍐х箚闁兼悂娼х欢鐐烘倵閿涘崬瀚?token 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掓禒楣冩⒑缁嬫鍎嶉柛濠冪箞瀵寮撮悢铏诡啎閻熸粌绉瑰畷顖烆敃閿旇棄鈧泛鈹戦悩鍙夊闁稿﹦鏁婚弻娑滅疀閹垮啯笑婵炲瓨绮撶粻鏍ь潖濞差亜宸濆┑鐘插暟椤︺儵姊虹拠鑼鐎光偓缁嬫鍤曢柡灞诲劜閸婄兘鏌ｉ幋鐐冩岸骞忓ú顏呪拺闁告稑锕﹂埥澶愭煥閺囨ê鍔滅€垫澘瀚板畷鐔碱敍濞戞艾骞堥梺璇插嚱閹儵宕樿椤ユ岸姊婚崒姘偓椋庣矆娓氣偓楠炴牠顢曢敂缁樻櫈闂佸憡绋戦悺銊╂偂閳ь剟姊洪幐搴ｇ畵妞わ富鍨堕幏鎴︽偄閸忚偐鍘搁梺鎼炲劦椤ユ挾澹曢崸妤佺厽闁靛牆鍊告禍楣冩⒒閸屾瑧鍔嶉悗绗涘懏宕查柛灞绢嚤濞戞ǚ妲堟慨妯夸含楠炴捇姊洪崫鍕殭闁绘锕崺娑㈠箣閻樼數锛濇繛杈剧到椤牠顢旈崱娆戠暥闂佸湱鍎ら〃鍡涙偂閵夆晜鐓涢柛銉ｅ劚婵℃椽姊洪褍鐏ｉ柍褜鍓氶鏍窗閺嶎厸鈧箓鎮滈挊澶嬬€梺鍛婂姦閸犳牠鎮為崹顐犱簻闁圭儤鍨甸弳鐔访瑰鍡橆棄闁宠鍨块幃娆戔偓娑櫭棄宥夋⒑閸涘﹤鐏ョ紓宥咃工閻ｉ鎲撮崟顒傜Ф闂侀潧顭梽鍕偟閿熺姵鈷掗柛灞剧懄缁佹壆鈧娲滈弫璇茬暦娴兼潙绠涙い鏃傚帶閻忓﹪妫呴銏″缂佸鍨垮浼村Ψ閳哄倻鍘遍梺闈涱樈閸犳牗鏅堕鍓ф／闁诡垎鍕淮闂佸搫鐭夌紞浣规叏閳ь剟鏌曡箛濠冩珖闁告梹鎮傚娲寠婢跺﹥娈堕梺鍝ュУ閸旀骞戦姀鐘闁靛繒濮烽鍝勨攽閻愬弶顥滅紒缁樺笚缁傛帡鎳栭埡鍐紳婵炶揪绲介幖顐︻敁閹惧墎纾界€广儱瀚粣鏃傗偓娈垮枛椤攱淇婇幖浣肝ㄩ柕蹇婂墲閺夋悂姊绘担铏广€婇柛鎾寸箞閺佸啴顢曢妶鍡╂綗闂佺粯鍔曢顓犵不妤ｅ啯鍊甸柣銏☆問閻掑墽鎮妷鈺傗拺闁告繂瀚崳钘夆攽椤旇姤灏﹂柨婵堝仩缁犳稑鈽夊▎鎰姃闂備線娼荤€靛矂宕㈡ィ鍐╂櫖婵犲﹤鍟犻弨浠嬫煃閽樺顥滈柣蹇嬪劦閺屾稓鈧綆鍋勬慨宥嗐亜閵忊埄鎴犵紦娴犲宸濆┑鐐靛亾鐎氬ジ姊绘担渚敯闁稿鍔欏畷鎴濃槈濞嗗海绠氭繝闈涘€搁幉锟犳偂閻斿吋鐓涚€广儱楠告禍婊堟偨椤栨繂鐓愰柕鍥у瀵挳顢旈崱娅烘粌顪冮妶鍐ㄧ仾婵炶尙鍠愰幈銊╁焵椤掑嫭鐓冮弶鐐村閸斿秹鏌涙繝搴＄仸婵﹦绮幏鍛村川婵犲倹娈樺┑鐐存尰绾板秹銆冩繝鍌滄殾鐟滅増甯掔涵鈧梺缁樺姀閺呮粓鎮樻笟鈧娲传閸曨剙鍋嶉梺鎼炲妽濡炰粙宕哄☉銏犵闁挎梻鏅崢鍗炩攽閻愭潙鐏﹂柨鏇ㄥ亰瀵劎鎷犲顔惧數闁荤姴鎼幖顐︻敂閳哄懏顥嗗璺侯儑缁♀偓婵犵數濮撮崐褰掑闯閻熸噴褰掓偐濞嗗繐顏х紒璇叉閺屾稑鈻庤箛锝喰︽繝娈垮枛濞诧箓濡甸崟顖ｆ晝闁挎繂娲ㄩ悿鍕⒑绾懏鐝紒顔奸叄瀹曠増绻濋崑鐣屽枔閹风姴顔忛鑽ゆそ缂傚倸鍊搁崐鎼佸磹閹间礁纾瑰瀣捣閻棗銆掑锝呬壕濡炪們鍨洪悧鐘茬暦閻撳簶鏀介柛鎰ㄦ櫇娴滄瑩姊绘担鍛婃儓婵炲眰鍔嶉幈銊╂煥鐎ｎ兘鏋栭悗骞垮劚椤︿即鎮″▎鎰闁割偅绻勬禒銏ゆ煛鐎ｃ劌鈧鎯€椤忓牆绀堢憸宥囨暜閼哥偣浜滄い鎾墲绾爼鏌熼悷鏉款伃闁诡噮鍣ｅ鍫曞箣濠垫劖娴嗛梻鍌氬€风粈浣革耿闁秴鐤炬い鎰剁稻閸欏繐顪冪€ｎ亜顒㈡い顐ｆ礋閺岀喖鎮滃Ο璇查瀺婵犳鍠楃划鎾诲蓟閻斿皝鏋旈柛顭戝枟閻忔捇姊虹紒妯诲鞍闁圭懓娲獮鍐ㄎ旈埀顒勫煝閹捐鍨傛い鏃傛櫕娴滅偟绱撻崒娆戣窗闁哥姵顨婇幃鐑藉煛閸涱厙銉ッ归敐鍛棌婵炵鍔戦弻宥堫檨闁告挾鍠栭悰顕€宕橀妸銏＄€婚梺瑙勫劤绾绢參顢橀悡搴富闁靛牆妫欓埛鎰版煕鎼淬垹鈻曢柛鈹惧亾?TTFT闂?
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鎾煙椤斿吋鍋ユい銏″哺閸┾偓妞ゆ帒瀚拑鐔兼煥濠靛棙鍟掗柡鍐ㄧ墕閻掑灚銇勯幒鎴濐仾闁稿顑呴埞鎴︽偐閸欏鎮欓梺娲诲幗椤ㄥ﹪鎮￠锕€鐐婇柕濞р偓婵洭姊虹紒妯诲鞍婵炶尙鍠栧濠氭偄閸忕厧鈧攱銇勯幒鎴Ч闁伙絽鐖煎铏规兜閸涱喚褰ч梺鍛婃⒐閻熲晠鐛崘顔芥櫖闁告洏鍔屾禍楣冩煥濠靛棝顎楅柡瀣枛閺屽秹鏌ㄧ€ｎ亞浼岄梺鍝勬湰閻╊垶寮崒婊勫珰闁圭粯甯為崫妤佺節绾板纾块柛瀣崌楠炲啴宕掗悙鑼舵憰濠电偞鍨崹娲磻閹邦喒鍋撶憴鍕婵炲眰鍊濋崺娑㈠醇閵夛腹鎷?闂?keepalive 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵挳鎮欏ù瀣壕鐟滅増甯掔壕鍧楁煙鐎电校闁哥姵鍔欓弻锝呂旈埀顒勬偋閸℃瑧绠旈柟鐑樻⒒绾惧ジ鏌嶈閸撴艾顕ラ崟顓濇勃缂佸銇樻竟鏇㈡⒑缁嬭法绠版い锔诲灡缁傚秵銈ｉ崘鈹炬嫼闂備緡鍋嗛崑娑㈡嚐椤栨稒娅犻柟缁㈠枟閻撴洟鏌嶇憴鍕姢濞存粎鍋撴穱濠囨倷椤忓嫧鍋撻弽顐ｆ殰濠电姴瀚惌鍡椼€掑锝呬壕閻庤娲滈弫濠氥€佸Δ鍛妞ゆ帒鍊搁獮鍫ユ⒒娴ｇ瓔娼愮€规洘锚閳绘柨鈽夐姀鐘插殤濠电偞鍨堕悷顖氣枔娴犲鐓熼柟閭﹀幖缁插鏌￠崱顓犲埌闁宠鍨块弫宥夊礋椤愩垹绠ｉ梻浣虹《閺備線宕滃┑瀣闁告侗鍠氶悷瑙勩亜閺嶃劌鍨侀柕鍫濇缁♀偓闂佹眹鍨藉褎绂掗埡鍛厵婵炶尪顔婄花鐣屸偓鍨緲閿曨亜鐣风粙璇炬棃鍩€椤掑倻涓嶉柣妯肩帛閻撴瑩鏌熺憴鍕Е闁搞倖鐟╅弻娑㈠箣閻樻祴鏋呴梺鍝勬湰缁嬫挻绂掗敃鍌氱鐟滃鍩€椤掍礁绗х紒杈ㄥ浮閹晠鎳￠妶鍥ㄦ瘒闁诲孩顔栭崰鏍偉閻撳海鏆﹂柣鎾崇岸閺€浠嬫煕閳ュ磭绠查悗鍨墵濮婄粯鎷呯粵瀣異闂佸摜濮甸悧鐘充繆閸洖绠瑰ù锝嗙摃閹芥洟姊虹捄銊ユ灁濠殿喗娼欓—鍐寠婢规繂缍婇弫鎰緞鐎ｎ偆褰搁梻浣告憸婵挳鏁冮鍕垫綎闁惧繗顫夌€氭岸鏌嶉妷銊︾彧闁诲繋绶氬娲箹閻愭彃顫呭┑鈽嗗亜鐎氫即骞冩ィ鍐╁€婚柦妯猴級閳哄懏鐓冮柛婵嗗閺€濠氭煛閸滀礁澧柍瑙勫灦楠炲﹪鏌涙繝鍐ㄥ鐎规洘鍨块獮妯兼惥娴ｇ儤鍤€妞ゎ厹鍔戝畷濂稿閵忊剝鐦掗梻鍌欑閹碱偊寮甸鍕剮妞ゆ牜鍋涢悘鎶芥煥閺囩偛鈧摜绮婚敐鍡欑瘈闂傚牊绋掗崳浠嬫煕閹存繃璐＄紒杈ㄦ崌瀹曟帒鈻庨幇顔哄仒闂備胶纭堕弲婊堟偋閻樿绠栭柨鐔哄У閺呮悂鏌ｅΟ鍨毢闁伙絽鎼埞鎴炲箠闁稿﹥鍔欏畷鎴﹀箻濞ｎ兛绨诲銈嗗姧缁茶法绮婚悙鐑樼厵妞ゆ梻鎳撴晶鏌ユ煙椤栨稒顥堥柡浣瑰姍瀹曠喖顢楁径瀣珝闂傚倸鍊搁崐鎼佸磹閻戣姤鍤勯柛顐ｆ磸閳ь兛鐒︾换婵嬪磻閼恒儳娲寸€规洜鍠栭、娑橆潩閹插涓叉繝鐢靛Х閺佸憡鎱ㄩ悽鍛婂殞濡わ絽鍟崐鍧楁煕椤垵浜栧ù婊勭矒閺岀喖鎮滃Ο铏瑰帎闂佸憡姊婚崰鎾舵閹烘挸绶為悘鐐靛亾濮ｅ牓姊洪崫鍕拱闁烩晩鍨堕獮鍐煛娴ｇ儤娈鹃梺鎼炲劀閳ь剟骞忛敓鐘斥拻濞达絿鍎ら崵鈧悗娈垮枛閻栧ジ鐛幇鏉跨闁芥ê顦伴悗顒勬⒑閸涘﹤濮﹂柛鐘崇墱婢规洟宕楅懖鈺冾啎闂佺懓顕崐鎴濐潩鐠鸿櫣锛涢梺瑙勫礃椤曆囨煁閸ヮ剚鐓涢柛銉㈡櫅閺嬫棃鏌涘Ο鍦煓婵﹪缂氶妵鎰板箳閹存粌鏋堝┑鐐茬摠缁酣宕戦悢鍝勫灊濠电姴娲﹂悡銉╂倵閿濆倹娅囩紒鐘冲哺濮婅櫣绱掑Ο鍝勑曢梺鍛婃尰瀹€绋跨暦閹版澘绠涢柣妤€鐗忛崢鎼佹煟韫囨洖浠滃褑妫勭叅閻庣數纭堕崑鎾斥枔閸喗鐝梺鍛婃尵閸犲酣鎮鹃悿顖樹汗闁圭儤绻冮弲顏堟⒑閸涘﹣绶遍柛鎾寸〒婢规洘銈ｉ崘鈹炬嫽闂佺鏈悷褔藝閿曞倹鐓欑痪鏉垮船娴滀即鏌熼姘兼Ч缂佽櫣鏅划娆戞崉閵娧冪倞闂備胶鎳撻崥瀣偩椤忓牆绀夌€光偓閸曨剙浜楀┑鐘绘涧椤戝棝鎮″▎鎾寸厵閻熸瑥瀚慨锕傛煕閵堝棛鎳囬柡灞剧洴婵℃悂濡烽妷銏犱壕鐟滅増甯掓闂佸憡娲﹂崹鎵尵瀹ュ鐓曢柕澶嬪灥閸燁垶宕曢鍫熲拻濞达絽鎲￠崯鐐烘煕閵娧勬毄缂佽京鍋炵换婵嬪炊瑜戦幗鏇㈡⒑閹稿海绠撴俊顐ｇ懇瀹曟洟鎮㈤崗鑲╁帾闂婎偄娲ら敃銉モ枍閸涘瓨鐓涢柛娑氱節閹茬偓鎱ㄦ繝鍐┿仢鐎规洦鍋婃俊鐑藉Ψ閹板墎绉柡宀嬬到铻栧ù锝呮贡椤︿即姊洪悙钘夊姷缂佺姵鎸搁悾鐑藉醇閺囥劍鏅㈡繛杈剧秬椤顢旈悢鍏尖拻濞达綀娅ｇ敮娑㈡煥濮樻墎鍋撳▓鍨灈闁绘牕銈搁獮鍐ㄎ旈崪浣规櫌闂佸憡娲﹂崜娆撳焵椤掆偓閻忔岸銆冮妷鈺傚€烽柤纰卞厸閾忓酣姊洪崨濠冣拹缁炬澘绉堕幑銏犫槈濮橆厼鐝伴悷婊冪箻瀹曟碍瀵肩€涙鍘撻柣鐘辩绾绢參鎯屽▎鎰╀簻闁靛繆鍓濋ˉ鍫⑩偓瑙勬礀閵堟悂骞冮姀銈呬紶闁告洦鍋嗛惈鍕節閻㈤潧啸妞わ綀妫勫嵄闁告稒娼欓崹鍌涖亜閺嶃劎鐭嬪┑顖欏嵆閺屻劑寮村Δ鈧禍楣冩煕濡ゅ懍鎲鹃柟顔煎槻閳诲氦绠涢幙鍐ф樊濠电偛鐡ㄧ划宥囧垝閹剧粯鍋傛い鎰剁畱閻愬﹪鏌曟径鍫濆姎濠殿喓鍨荤槐鎾存媴閸欏鈧棝鏌涚€ｎ偅宕屾慨?goroutine 婵?channel 闂傚倸鍊搁崐鎼佸磹閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞妞ゆ帒顦伴弲顏堟偡濠婂啰绠婚柛鈹惧亾濡炪倖甯婇懗鍫曞煝閹剧粯鐓涢柛娑卞灠閳诲牓鏌曢崱鏇狀槮闁宠閰ｉ獮姗€宕橀幓鎺撴殢濠碉紕鍋戦崐鏍箰妤ｅ啫纾婚柣鏂垮悑閸嬫﹢鏌曟径鍡樻珕闁抽攱鍨块弻娑㈡晜鐠囨彃绠归梺鍛婃煥椤戝棝濡甸崟顖毼╅柕澶涚畱濞呇勭節閵忥綆娼愭繛鍙夘焽閹广垹鈹戦崱鈺傚兊濡炪倖鎸炬慨鎾嵁瀹ュ鈷掑ù锝呮啞閸熺偤鏌熺粙鎸庮棦鐎规洏鍔戦、姗€鎮埀顒勫磻瑜斿濠氬磼濞嗘垵濡芥繝鐢靛仜閿曨亜顕ｉ妸鈺傜劶鐎广儱鎳庨悗顓㈡⒑缁夊棗瀚峰▓妯尖偓瑙勬尫缁舵岸寮婚垾鎰佸悑閹肩补鈧尙鐖遍梻浣侯焾椤戝棝骞愰幖浣哥叀濠㈣泛艌閺嬪秹鏌ц箛锝呬簻闁诲繆鏅涢湁婵犲﹤瀚崐鎰亜閵忊槄鑰块柟顔斤耿閹筹繝濡堕崱妤佺暚婵＄偑鍊ら崢褰掑礉閹存繄鏆﹀┑鍌滎焾椤懘鏌曡箛濠傚⒉闁伙附甯掗埞?
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
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴楠炴﹢鎳犻澶嬓滈梻浣规偠閸斿秶鎹㈤崘顔嘉﹂柛鏇ㄥ灠閸愨偓濡炪倖鍔﹀鈧繛宀婁邯濮婅櫣绮欏▎鎯у壃闂佸憡鎸荤换鍫熶繆閻㈢绀嬫い鏍ㄨ壘閸炪劌顪冮妶鍡楀Е婵犫懇鍋撶紓渚囧枛婢у海妲愰幘瀛樺闁圭粯甯婃竟鏇㈡⒒娴ｇ顥忛柛瀣瀹曚即骞橀崜浣风瑝婵°倧绲介崯顖炴偂濞嗘挻鍊垫繛鎴炵懐閻掍粙鏌熼崘鎻掓殻闁哄矉缍€缁犳盯鏁愰崟顖氫粣闂?goroutine 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌＄仦绋垮⒉鐎垫澘瀚埀顒婄秵娴滄繈顢欓崨顓涙斀闁绘劕寮堕埢鏇灻瑰鍕煉闁诡喒鈧枼妲堥柕蹇ョ磿閸樺崬鈹戦悩鎵嶅牓宕戦幘缁樼厽闁硅櫣鍋熼悾鐢碘偓娈垮枟閻擄繝銆侀弮鍫濋唶闁绘棁娓归悽缁樼節閻㈤潧孝闁挎洏鍊楅埀顒佸嚬閸撶喖骞冨鈧鍫曞箣閺冣偓閺傗偓闂備焦瀵х粙鎴犫偓姘煎弮瀵娊顢楁笟鍥啍闂佺粯鍔曞鍫曞窗濡皷鍋撳▓鍨灈濠⒀冮叄楠炴垿宕熼姣尖晠鏌曟径鍫濈仼濞存粓绠栭弻锝夊箛椤旂厧濡洪梺鎶芥敱閸ㄥ灝顫忔繝姘唶闁绘梹鍎奸崥鍌氣攽閻愭潙鐏熼柛銊ョ秺閹偤宕归鐘辩盎闂佸湱鍎ら崹鐢割敂閳哄懏鐓曢柟瀵稿Т閳诲牓鏌＄仦绯曞亾閹颁礁鎮戦梺鍛婂姂閸斿矂鈥栫€ｎ剛纾藉ù锝呭级椤庡棝鏌涚€ｎ偅宕屾慨濠勭帛閹峰懘宕烽鐔诲即闂備焦鎮堕崝蹇撐涢崟顖椻偓锕傚炊椤忓秵些闂備胶鍎靛Σ鍛村矗閸愵煈娼栫紓浣股戞刊鎾煕濞戞﹫鏀婚柛鐘冲姍濮婃椽宕妷銉愩垻绱掓径灞惧殌闁伙絿鍏橀弫鎾绘偐閼碱剦妲伴梻浣藉亹閳峰牓宕滃顑芥灁濡わ絽鍟埛鎴︽煕濞戞﹫鍔熺紒鐘虫崌閹顫濋敐鍛闂傚倷绀侀幖顐﹀疮椤愶附鍋嬮柛鈩冾殢閸熷懏绻濋棃娑欏偍濞存粍绮撻弻锟犲礃閿濆懍澹曢梻浣侯焾妤犳悂宕崸妤婃晪闁挎繂顦伴幆鐐淬亜閹扳晛鈧鎯侀崼銉︹拺婵懓娲ら悘鍙夌箾娴ｅ啿瀚々鐑芥煥閺囩偛鈧綊鎮￠弴銏＄厪濠电偛鐏濋崝銈夋煕閳哄绉柡灞界Ч婵＄兘顢涘鍛闁诲氦顫夊ú鏍偉婵傛悶鈧礁螖娴ｇ懓顎撶紓渚囧灡濞叉牗绂嶆ィ鍐╁仯闁诡厽甯掓俊璺ㄧ棯閹佸仮闁诡喗顨婇弫鎰償閳ユ彃顥氱紓鍌欑劍婢у酣寮查悩璇茶摕婵炴垯鍨洪弲婊堟偣閸ャ劌绲荤紒鐘冲▕濮婅櫣绱掑Ο铏圭懆闂佹寧娲︽禍顏勵嚕婵犳艾鐏抽柟棰佺閹垿姊洪崨濠傚闁告柨鐭傞垾鏍ㄥ緞閹邦厸鎷虹紓鍌欑劍椤洨绮婚弽顐熷亾閸忓浜鹃梺褰掓？缁€浣虹不閺嶃劋绻嗛柕鍫濇噹閺嗙偤鏌涢悩鍙夘棦闁哄本鐩鎾Ω閵夈儺娼炬俊鐐€х拹鐔煎礉瀹€鍕ㄢ偓鏃堝礃椤斿槈褔鏌涢幇鈺佸妞ゎ剙鐗撳铏规兜閸涱喚褰ч梺鎸庢磸閸ㄨ姤淇婄€涙ɑ濯撮柤鍙夌箘閸犳牠骞婇敓鐘参у璺侯儛閳ь剙绉剁槐鎾诲磼濞嗘埈妲銈嗗灥閹冲酣鈥﹂崶顒佹櫢闁绘灏崺鐐烘煟閻樼儤銆冮悹鈧敃鈧妴鎺撶節濮橆厾鍘梺鍓插亝缁诲啴藟濠婂啠鏀芥い鏃傚帶閳ь剙娼″璇测槈閵忕姈鈺呮煥閺傛娼熷ù鐘灩椤啴濡堕崘銊т痪闂佽崵鍟块弲鐘绘偘椤旈敮鍋撻敐搴℃灍闁哄懏绮撻幃宄扳枎濞嗘垹蓱闁诲孩淇哄Λ鍕煘閹达附鍊烽柤鎼佹涧濞懷呯磽娴ｈ棄绱︾紒顔界懇閻涱喗寰勯幇顓熸闂佺粯顭堢亸娆撳蓟閸儲鈷戠紓浣姑慨澶愭煕鎼存稑鈧繈骞冮敓鐘冲亜闁稿繗鍋愰崢浠嬫⒑閸濆嫭宸濋柛瀣洴閸┾偓妞ゆ巻鍋撴い顓犲厴楠炲啯銈ｉ崘鈺佷缓闂佸憡绋戦敃锕傚储闂堟侗娓婚柕鍫濇閻撶喖鏌涢弬鑳閻撱倝鏌ㄩ弴鐐测偓鍝ュ婵犳碍鐓欓柟娈垮枛椤ｅジ鏌ｉ幘瀛樼缂佺粯鐩畷鍗炍旈崘顏嶅敹闂備線鈧偛鑻晶顔姐亜椤撶偛妲婚柣锝夋敱鐎靛ジ寮堕幋婵嗘暏闁荤喐绮岀换妯侯嚕閹惰姤鏅濋柛灞剧〒閸樺崬鈹戞幊閸婃洟宕锕€鐒垫い鎺戯功缁夘噣鏌熼鍝勭伈鐎规洘顨婇幃鈩冩償閵忋垺娈堕梻鍌氬€风粈渚€宕崸妤€绠规い鎰剁畱閻ゎ喗銇勯弽顭戝毀闁圭儤鍨归弳鍡涙煕閺囥劌鍘撮柟宄邦煼濮婅櫣绮欓幐搴㈡嫳闂佽崵鍠嗛崝鎴濈暦濡も偓閻ｆ繈鍩€椤掑倹顫曢柟鐑橆殢閺佸鏌涘☉鍗炲箻濞寸姵鎮傞幃妤冩喆閸曨剛顦ㄧ紓浣筋嚙閻楀棝顢氶敐鍡欘浄閻庯絽鐏氶弲鐐烘⒑閸涘﹦鎳€闁稿孩鐓￠幆鍫ｇ疀濞戞瑢鎷绘繛杈剧到閹诧繝宕悙鐑樼厱闁哄啯鎸鹃悾鐢碘偓瑙勬磻閸楁娊鐛Ο鍏煎珰闁肩⒈鍓欐慨锔戒繆閻愵亜鈧牕顔忔繝姘；闁瑰墽绮悡鏇㈡煟閺囨氨顦﹀ù婊€鍗抽弻娑㈠煘閸喖濮曢悗鍨緲鐎氫即鐛崶顒夋晢濠㈣泛顑囩粔閬嶆⒒閸屾瑧鍔嶉悗绗涘懐鐭欓柟瀵稿Л閸嬫挸顫濋悡搴♀拫闂?keepalive/闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌ｉ幋锝呅撻柛銈呭閺屾盯顢曢敐鍡欘槬缂佺偓鍎冲锟犲蓟閿濆顫呴柕蹇婃櫇閸斿摜绱撴担鎻掍壕闂佸壊鍋嗛崰鎾跺姬閳ь剟姊婚崒姘卞缂佸鎸婚弲璺衡槈閵忥紕鍘搁梺绯曞墲椤洭鎯岄幒妤佺厸鐎光偓閳ь剟宕伴弽顓炵畺婵犲﹤鍠氬銊╂煕閳╁喚鐒介柡鍡愬劤缁辨捇宕掑▎鎺戝帯婵犳鍣ｉ弨杈╂崲濞戙垹鐒垫い鎺戝閻撶喖鏌熼崹顔兼殭濞存粍澹嗛埀顒冾潐濞叉牠鎮ラ崗闂寸箚闁归棿鐒﹂弲婊堟煢濡警妲稿鐟版閳规垿鎮╅崹顐ｆ瘎闂佺顑囬崑銈夌嵁閹版澘绠瑰ù锝嗙ゴ閸嬫捇鏁冮崒娑樷偓濠氭煢濡警妲奸柟鑺ユ礀閳规垿鎮欓弶鎴犱桓闂佹椿鍙庨崰鏍煝娴犲鏁傞柛顐ゅ枔閸橀亶姊洪崷顓炰壕闁靛洦鐩畷鎴﹀箻缂堢姷绠氶梺鍦帛鐢偞鏅堕姀銏㈢＜閺夊牄鍔嶇亸浼存煙瀹勭増鍣界紒顔界懃閳诲酣鎮欓渚囧晥闂傚倸鍊烽懗鍫曗€﹂崼銉晞闁告劦鍠栫壕瑙勭節闂堟稒锛嶉柛銈嗘礋閹綊宕堕妸褋鍋炲┑鈩冨絻閻楁捇寮婚弴锛勭杸闁哄洨鍊姀鈶╁亾鐟欏嫭灏俊顐ｇ箓椤繐煤椤忓拋妫冨┑鐐村灱娴滎剟宕濋幖浣光拺缂佸瀵ч崬澶娒瑰搴″闁告帗甯為幏鐘垫啑娴ｈ銇濇い銏℃瀹曘劑顢旈崨顓т紪?
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
			// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘劙婀侀梺绋跨箰閸氬绱為幋锔界厱闁靛鍎遍埀顒€娼″濠氭晲婢跺﹦顔掗悗瑙勬礀濞层倝宕ú顏呪拺闁告繂瀚烽崕鎰版煟濡や緡娈橀柟骞垮灩閳藉濮€閻樻鍚呴梻浣虹帛閸旀浜稿▎鎾崇濞寸厧鐡ㄩ埛鎴犵磼鐎ｎ偒鍎ラ柛搴㈠姍閺岀喖宕ㄦ繝鍐ㄢ偓鎰版煥濠靛牆浠滈柍瑙勫灩閳ь剨缍嗛崑鍕濡ゅ懏鐓欓柛蹇氬亹閺嗘﹢鏌涢妸锔筋潡闁靛洦鍔栭幆鏃堬綖椤撶姷鐣鹃梻渚€娼ч悧鍡椕洪妶鍛瘎闂傚倷鑳堕…鍫ヮ敄閸℃稒鍎庢い鏍ㄦ皑閺嗭附銇勯弽顐㈠壉闁轰礁绉电换婵囩節閸屾凹浠奸柟鍏兼綑閿曘倝鍩為幋锔藉亹缂備焦蓱闁款厼顪冮妶鍡楃仴闁硅櫕鍔栫粩鐔煎即閻樼數锛滃┑鈽嗗灦閺呰尙鑺辨繝姘拺闁告繂瀚ⅹ闂佸憡鏌ㄥù椋庡垝婵犳艾唯闁挎柨澧介鏇㈡⒑缁嬭法绠抽柛妯犲嫭鍙忕€广儱顦伴悡娑氣偓鍏夊亾闁逞屽墴瀹曠増鎯旈妸锕€浠奸梺璺ㄥ枔婵敻宕戠€ｎ喗鐓曟い鎰剁悼缁犳牠鏌ｉ敐鍜佺吋婵﹦绮幏鍛驳鐎ｎ亝鐣伴梻浣告憸婵敻鎮уΔ鍛柧闁割偅娲橀崵宥夋煏婢诡垰鏈粊顐︽⒒娴ｇ懓顕滄繛鎻掔Ч瀹曟垿骞樼紒妯煎帗闁荤喐鐟ョ€氼剟鎮樼€电硶鍋撶憴鍕闁哥姵鐗犻妴渚€寮撮姀鐘栄囨煕閺囥劌骞樻い锔规櫊濮婂宕掑顑藉亾妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈡晜閽樺澹掑┑鐘灱濞夋盯寮甸鈧悾鐑藉矗婢跺瞼鐦堥梻鍌氱墛缁嬫帡藟閻樼粯鐓涢柛鈽嗗弮濡绢噣鏌熸笟鍨妞ゎ亜鍟伴幏鐘荤叓椤撶儐妫滄繝纰夌磿閸嬫垿宕愰幋锕€鍨傛繛宸簴閳ь剨绠撳畷濂稿Ψ鐎ｎ亝鍠樼€殿喖顭锋俊鐑芥晜閹冪闂傚倷绀侀幉锛勫垝瀹€鍕柈濞村吋娼欑粻鏍煠濞村娅囩痪鎯с偢瀵爼宕煎☉妯侯瀴闁诲繐绻嬮崡鎶藉蓟閻旈鏆嬮柣妤€鐗嗗▍銈囩磽娴ｇ鈧湱鏁悙鍝勭劦妞ゆ帒锕︾粔闈浢瑰鍕疄妤犵偛锕ら埢搴ㄥ箣閻樼绱抽梻浣侯焾閺堫剟鎮烽敂鍓х焾闁绘鐗勬禍婊堟煛閸パ勵棞婵炶绠撳畷鎴犫偓锝庡枟閻撴洟鏌嶉埡浣告殶闁愁垱娲熼弻娑氣偓锛卞嫭鐝氶梺鍝勭焿缂嶄線鐛€ｎ喗鏅查柛娑樻噺閹瑰洭寮婚敓鐘插窛妞ゆ棁鍋愮粊宄邦渻閵堝骸骞栨繛纭风節楠炲﹤顭ㄩ崼鐔峰壎濡炪倕绻愰幊搴ㄦ倶鐎电硶鍋撳▓鍨灍闁绘搫绻濋妴浣肝旈崨顓犲姦濡炪倖甯掔€氼剟鎮″鈧弻鐔告綇妤ｅ啯顎嶉梺绋款儐閸旀妲愰幘瀛樺闁兼祴鍓濋崹鍧楀蓟閵娾晩鏁囬柣鎰ㄦ櫆閺傗偓闂備礁鐤囧Λ鍕涘Δ浣侯洸婵犻潧娲㈡禍婊堟煏婵炲灝鍔甸棅顒夊墴閺岀喖顢欑拠鎻掔ギ濡炪們鍨洪悷鈺呭蓟閵娾晩鏁嶆慨姗嗗亜椤ユ岸姊绘担鐟邦嚋缂佽鍊归〃銉╁箹娴ｇ鍤戦梺缁樻煥閸氬鎮￠崘顔解拺闁割煈鍣崕蹇涙煟韫囨梹灏﹂柡宀嬬磿娴狅箓宕滆濡插牓姊虹€圭媭娼愰柛銊ョ仢閻ｇ兘宕￠悙宥嗘⒐缁绘繃鎷呴搹鐧哥幢闂傚倷娴囬褏鈧稈鏅濈划娆撳箳閺冣偓瀹曟煡鏌涘畝鈧崑娑㈡偂濮椻偓閺岀喐娼忔ィ鍐╊€嶉梺绋匡功閸忔﹢骞冨Δ鍛瀭妞ゆ劧绲芥禒鈺呮⒑缁嬫鍎戦柛瀣ㄥ€曢～蹇撁洪鍕炊闂佸憡娲﹂崜娆忊枍瑜庣换婵嬫偨闂堟稐姹楅梺绋款儐閹瑰洤顫忓ú顏咁棃婵炴垶鐟Λ娑氱磽娴ｆ彃浜炬繛鎾村焹閸嬫捇鏌涢埞鍨仾闁诡垱妫冩俊鎼佸Ψ瑜滈崯瀣煟鎼淬値娼愭繛鍙夌墪鐓ら柕鍫濐槸閻掑灚銇勯幒鎴濃偓鍛婄鏉堛劍鍙忓┑鐘插暞閵囨繄鈧娲橀敃銏犵暦閿濆棗绶為悗锝庝簻婢瑰秹姊婚崒娆掑厡妞ゎ厼鐗撻弫鍐Ψ閳轰胶鐣洪梺绋跨灱閸嬫垿鍩€椤戣法绐旂€殿噮鍣ｅ畷鐓庘攽閸偄鏅ｆ繝鐢靛仩閹活亞寰婃禒瀣偍婵犲﹤鐗嗙粻顖涚箾瀹割喕绨奸柍閿嬪灴閺屾盯鏁傜拠鎻掑闂佺粯甯＄粻鏍箖濡も偓閳藉骞掗幘瀵稿綃闂佽姤顭囬崰鏍蓟濞戙垹鐒洪柛蹇曞Ь閸嬫劙骞戦姀銈呭耿婵☆垶鏅茬花濠氭⒑鐟欏嫬顥愰柡鍛洴閹﹢鍩￠崒妯圭盎闂婎偄娴勭徊钘夘嚕椤曗偓閺屸€崇暆閳ь剟宕伴弽顓溾偓浣割潩鐠哄搫鑰垮┑鐐叉閸ㄧ懓危閸ヮ剚鐓熼幖娣焺閸熷繘鏌涢悩鎰佹疁妤犵偞鍔欓獮搴ㄦ寠婢跺瞼鏆繝鐢靛仜濡瑩骞愰幖浣瑰亗婵炴垯鍨洪悡鏇㈡倶閻愰鍤欓柍褜鍓氶悧婊呮閻愯尙鏆嗛柛鏇ㄥ厴閹峰姊虹粙鎸庢拱闁荤噦濡囩划濠囨偋閸稐绨诲銈嗗姂閸╁嫬危閹间焦鐓熼柨婵嗘噹濡茬粯銇勯锝囩煉闁糕斁鍋撳銈嗗笒鐎氼參寮查弻銉︾厽闁归偊鍘界紞鎴犵磼閸撲礁浠遍柡宀€鍠栭弻鍥晝閳ь剟宕濋敃鍌涚厪濠㈣泛妫欏▍鍡涙煟閹惧瓨绀嬮柡宀€鍠栭獮宥夘敊閼恒儲鐦庨梻浣告啞钃遍柣鈺婂灦瀵鏁愭径濠冾棟闂佸湱顭堢€垫帒螞閿曞倹鈷戦悹鍥ｂ偓铏仌濡炪値鍋勯ˇ鍨繆閸洘鏅濋柛灞炬皑椤斿洭鏌熼崗鍏煎剹闁哥姵娲熷畷顐⑽旈崨顔规嫼闁哄鍋炴竟鍡浰囬敃鍌涚厽婵°倓鐒﹀畷灞绢殽閻愭彃鏆欓摶鏍煃瑜滈崜鐔煎春閵忋倕绠婚悹鍥ㄥ絻閻庮厼顪冮妶鍡樷拻闁哄拋鍋夐妵鎰邦敍閻愮补鎷绘繛鎾村焹閸嬫挻绻涢懝鏉垮惞缂佽京鍋ゅ畷鍫曞煛娴ｈ櫣鐡樺┑鐐差嚟婵挳顢栭幇鏉挎瀬閻庯綆鍠楅悡銉︾節闂堟稒锛嶆俊鎻掔秺閺屾稓鈧絻鍔屾慨鍌炴煛鐏炲墽鈽夐柍钘夘槸椤粓宕煎┑鍛煑婵犵數鍋涢顓熸叏闂堟侗娓婚柦妯侯樈濞兼牠鏌涘┑鍡楃彅鐟滄柨鐣烽幆閭︽Ь婵炲濮撮妶绋款潖閾忚瀚氶柤纰卞墰椤斿鎮楅崗澶婁壕缂備礁顑堝▔鏇㈠汲閿曗偓闇夐柛蹇撳悑缂嶆垹绱掗悩鐑樼彧濞ｅ洤锕俊鍫曞礋椤曞懎鏁奸梻浣哥秺椤ユ挾鍒掗婊勫床婵炴垶鐟︾紞鍥煕閹炬鍟悡鍌炴⒒娴ｄ警鏀版い鏇熺矌閹广垹鈹戦崱娆愭濡炪倖鐗滈崑鐐哄磹閻戣姤鐓熼柟瀵稿剱閻掍粙鏌涘Ο鍦煓婵﹨娅ｇ槐鎺懳熼懡銈忕幢缂傚倷璁查崑鎾炽€掑锝呬壕闂佽鍠楅敃銏ゅ极閸岀偞鐓ユい鏍ㄧ煯婢规洟鏌ｉ悢鍝ユ噧閻庢凹鍘剧划鍫ュ焵椤掑倻纾介柛灞炬皑瀛濋梺鎸庢处娴滎亪鎮伴鈧畷姗€鍩￠崘鐐カ闂佽鍑界紞鍡樼濠婂懐鐜绘繛鎴炵懅缁♀偓闂佹眹鍨藉褍鐡梻浣侯焾閿曘倝骞夐敍鍕崥闁绘梻鍘х粈瀣亜閺嶃劎鈻撻柟椋庣帛缁绘稒娼忛崜褏袣濠电偛鎷戠徊鍧楀极椤斿皷妲堥柕蹇ョ磿閸樻悂鏌ｈ箛鏇炰哗妞ゆ泦鍕弿闁稿本渚楀▓浠嬫煟閹邦剚鈻曟俊顖楀亾闂備浇妗ㄩ悞锕傚箲閸ヮ剙绠栭柍鍝勬噺閸ゆ垶銇勯幒鎴姛闁告艾缍婂濠氬磼濞嗘帒鍘＄紓渚囧櫘閸ㄨ泛顕ｆ繝姘╅柍杞版祰琚濋梺璇插嚱缂嶅棝宕板Δ鍛；闁稿瞼鍋為悡蹇擃熆鐠虹儤顥炴繛鍛閺岋綁骞樼€靛憡鍒涢梺鍝勬湰閻╊垶鐛崶顒€鐓涘ù锝呭閻庢潙鈹戦悩顔肩伇婵炲绋撻埀顒佸嚬閸欏啯淇婇悽绋跨妞ゆ牗顕遍妸鈺傜厪濠㈣埖锚閺嬬喖鏌嶇拠鑼ⅵ婵﹥妞藉畷顐﹀礋椤撶姴濮界紓鍌氬€哥粔顕€宕戦幘鍓佺＝濞达絽鎼牎闂佺儵鏅╅崹浼搭敋閿濆鏁嬮柍褜鍓熼悰顔锯偓锝庡枟閸婄兘鏌涢…鎴濅簼缂侀鍓氱换婵嬫偨闂堟刀锝嗐亜閺冣偓閻楃姴鐣锋导鏉戝唨妞ゆ挾鍠愬▍鍥⒑闂堟侗妲堕柛搴℃惈閵嗘帞鎷犵憗浣哥秺閹晛顔忛鐓庡闁诲氦顫夐幐椋庢濮樿泛钃熼柣鏂垮悑閻掍粙鏌ㄩ弴妤€浜惧Δ鐘靛仦椤ㄥ﹪寮诲☉娆愬劅闁挎繂鎳忛悘鍫ユ⒑閸濆嫯顫﹂柛鏂跨焸閸╃偤骞嬮敃鈧獮銏℃叏濡も偓濡瑩藟閿濆棛绡€闁汇垽娼ф禒婊堟煙閸愯尙绠婚柟顔惧厴閺佸倿鎮剧仦鍛婃暤濠电姷鏁告慨鏉懨洪敃鍌氱９闂佸灝顑冩禍婊堢叓閸ャ劍灏い蹇嬪€濋弻娑氣偓锝庡亝瀹曞瞼鈧鍠曠划娆撳灳閺嶎厽鏅搁柣妯兼暩缁愭绱撴担鍝勑ｇ紒瀣灴閸┿儲寰勬繛銏㈠枔濞戝灚顦版惔婵婎洬濠电姷鏁告慨鐑藉极閸涘﹦绠鹃柍褜鍓氱换娑欐媴閸愬弶绁╅柡浣稿閺屾稑鈽夐崡鐐典化婵炴垶鎸哥粔褰掑蓟閳ユ剚鍚嬮幖绮光偓宕囶啈闂備胶绮幐鎼佸疮閹绢喖钃熸繛鎴炃氶弨浠嬫煕閳╁喚娈㈠ù鐘荤畺濮婃椽妫冮埡鍐戝┑锛勫仒缁瑥顕ｇ拠娴嬫闁靛繒濮堣閺屾稑顭ㄩ埀顒傜矆娓氣偓瀹曟粓骞庨懞銉㈡嫼濠殿喚鎳撳ú銈夊礉閺夋嚚鐟邦煥鎼存繈鍋楀Δ鐘靛仜閸熸娊藝閸︻厸鍋撶憴鍕婵＄偘绮欓妴渚€寮撮姀鈺傛櫇闂佹寧绋戠€氼剟宕甸鍌滅＝?
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

// extractOpenAISSEDataLine 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐鍛傜喎鈻庨幆褎顔勭紓鍌欒兌婵挳鎮樺璺何﹂柛鏇ㄥ枤閻も偓闂佸湱鍋撻幆灞轿涢悙鐢电＝濞达絿鏅崼顏堟煕婵犲啰绠炵€殿喖顭锋俊鑸靛緞婵犲嫮鏆㈤梻浣告贡閸庛倝宕归崹顐ｅ弿閻犻缚銆€閺€浠嬫煟閹邦垰鐨哄褎姊荤槐鎺楊敊閻ｅ本鍣伴悗娈垮枦椤曆囧煡婢跺á鐔兼煥鐎ｎ偅婢戦梻鍌欑閹测€趁洪敃鍌氱；闁告洦鍓氶崣蹇涙煛閸モ晛鍓辨繛鎾愁煼閺屾洟宕煎┑瀣碘偓妤€霉濠婂嫮绠栫紒缁樼洴閹崇娀顢楅埀顒勫几濞戞埃鍋撳▓鍨灈妞ゎ厾鍏橀獮鍐閵堝懎绐涙繝鐢靛Т鐎氼喛鍊村┑鐘茬棄閺夊簱鍋撹瀵板﹥绂掔€ｎ亞鏌堝銈嗙墱閸庢劙寮担琛″亾楠炲灝鍔氭い锔垮嵆閸╂盯骞嬮悩鐢碉紳婵炶揪缍€閸嬪倿骞嬮悩杈╁墾濡炪倕绻愰悧濠囨偂閺囥垹绠归弶鍫濆⒔缁嬪鏌￠崱鈺佸闁逞屽墯椤旀牠宕板☉銏╂晪鐟滄棃銆佸Ο鑽ら檮缂佸娼￠崬璺侯渻閵堝棗濮х紓宥呮閸┾偓妞ゆ巻鍋撴繛宸幖椤繒绱掑Ο璇差€撻梺鍛婄☉閿曘倝寮抽崼銉︹拺閻熸瑥瀚崐鎰磼閻樺磭澧柣锝夋敱鐎靛ジ寮堕幋婵嗘暏婵＄偑鍊栭幐楣冨窗鎼淬垻顩烽柟鐑樺焾濞撳鏌曢崼婵囶棡缁惧墽鏁婚弻娑氣偓锝庡亝鐏忕増銇勯妸锝呭姦闁诡喗鐟╅獮鏍敇閻樻彃楔闂傚倷绀佺紞濠囁夐幘瀵哥闁逞屽墴閺岀喎鐣烽崶褉鏋呭銈冨灪椤ㄥ﹤鐣烽幒鎴旀婵炲棙鍨电粩鍏肩節閻㈤潧啸妞わ綆鍠氬Σ鎰板即閵忕姷锛涢梺纭呮彧缁犳垹澹曡ぐ鎺撶厸鐎规搩鍠栭懟顖炲疾閻樺磭绡€闁汇垽娼ф牎闂佽偐鎳撴晶鑺ョ珶閺囩喓绡€婵﹩鍘鹃崢鐢告⒑鐠団€崇€婚柛鎰亾濞堣櫣绱?SSE `data:` 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呮／闁诡垎宀€鍚嬮梺鍝勭焿缂嶄線鐛崶顒夋晩闁兼亽鍎查惁搴ㄦ⒒娴ｈ銇熼柛妯圭矙閹兘鍩￠崨顓℃憰闂佺粯鏌ㄩ幉锛勭礊閸ヮ灛鏃堟晲閸涱厽娈插銈呴缁夌懓顫忓ú顏勭闁绘劖绁撮崑鎾诲冀椤撶偟顦梺鍝勭▉閸樺吋顢婇梻浣告啞濞诧箓宕规导鏉戠闁逞屽墮椤啴濡堕崱妤€袝濠电姭鎳囬崑鎾剁磽娓氬洤鏋涢柣鏍帶椤繐煤椤忓懎娈愰梺鍐叉惈閿曪箓宕濋崫銉х＝闁稿本姘ㄥ瓭闂佹寧娲︽禍顏堟偘椤曗偓瀹曟﹢濡告惔銏☆棃鐎规洏鍔戦、娆撴⒒鐎靛摜澶勯梻鍌氬€烽悞锕傛儑瑜版帒鍨傚┑鐘宠壘绾惧鏌ㄥ┑鍡╂Ц闁绘帒鐏氶妵鍕箳閸℃ぞ澹曢梻浣瑰濞诧附绂嶉鍕靛殨閻犲洦绁村Σ鍫熸叏濮楀牏鍒板ù婊堢畺閺屻劌鈹戦崱娑扁偓妤€霉濠婂嫮绠橀柍褜鍓濋～澶娒洪弽顓熷剹闁稿瞼鍋涢拑鐔兼煟閺冨倵鎷￠柡浣革躬閺岀喖顢涢崱妤€鏆欑紒鐙呯稻娣囧﹪濡堕崶顬儵鏌涚€ｎ偆銆掔紒顔芥閵囨劙骞掔€Ｑ冧壕濞达絿纭堕弸搴ㄦ煙閻愵剚缍戦柍褜鍓熼弨閬嶅Φ閸曨垰绠崇€广儱娲╃粣妤呮⒑閸濄儱浠ч柡浣筋嚙椤?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欏▎鐐秱闂備胶鍋ㄩ崕閬嶅疮鐠恒劏濮抽柕澶嗘櫆閳锋帒霉閿濆懏鍟為悹鎰剁節閺屾稒鎯斿☉妯峰亾濠靛违闁告劦鍠栧婵嬫煛婢跺鐏╅柨娑欑矒濮婃椽妫冨ù銈嗙洴瀹曠喖顢曢敐鍥ｅ亾妤ｅ啯鈷掑ù锝呮啞閹牓鏌涢悤浣镐喊闁诡啫鍥х妞ゆ牗姘ㄩ崝?`data: xxx` 婵?`data:xxx` 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｇ拠娴嬫闁靛繆鏅滈弲婵嬫⒑閹稿海绠撴俊顐ｇ洴钘濋柕澹懐锛滈梺缁樺姦閸撴瑧绮堥崘鈹夸簻妞ゆ劑鍩勫Σ鍦磼瀹€鍐摵缂佺粯绻堝畷鍫曗€栭顒€娲﹂悡鏇熺箾閹存繂鑸归柡瀣洴閺岋絽鈹戦崶銊㈡嫽缂備浇椴搁幑鍥х暦閹烘埈娼╂慨锝嗙懕缂嶄線寮婚敐鍛傛棃宕橀妸銏＄€伴柣搴㈩問閸犳牠鈥﹂悜钘夌畺闁靛繈鍊曠粈鍌炴煟閹惧磭宀搁柛瀣尵缁辨帒螣鐠囨彃浼庢繝寰锋澘鈧劙宕戦幘缈犵箚妞ゆ劧绲块幊鍥煙椤曗偓缁犳牕鐣锋總绋垮嵆闁绘柨寮剁€氬ジ姊绘担鍛婂暈缂佸鍨块弫鍐Χ閸℃ê寮块梺閫炲苯澧存慨濠呮缁瑥鈻庨幆褍澹夐梻浣告贡閹虫挸煤閵堝缍栭煫鍥ㄦ礈绾惧吋淇婇婵愬殭妞ゅ孩鎹囧娲箚瑜忕粻鎶芥煙閾忣偅灏电紒顕呭弮閺佹捇鎮╅弬銉﹀闂備浇宕甸崰鎾存櫠濡ゅ懎绠熼柛娑橆焾娴滄粍銇勯幘璺轰户濠⒀嶉檮缁绘繈鍩€椤掍焦濯撮梺顐ｇ〒缁犳岸姊虹紒妯哄婵☆垰锕畷鏇＄疀濞戞瑥鈧灚绻涢幋鐐茬瑲婵炲懎锕﹂埀顒冾潐濞叉﹢銆冮崨绮光偓锕傚Ω閳轰線鍞堕梺缁樻煥閹碱偆鏁Δ鍛厽闁绘柨鎽滈惌瀣磼椤旇姤灏柣锝呭槻椤劑宕奸悢铚傜盎濠电娀娼ч崐鎼佀囬幍顔剧?
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

// correctToolCallsInResponseBody 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤濞硷繝寮婚悢琛″亾閻㈡鐒剧€涙繄绱撴担鐣屽牚闁稿﹥绻堝濠氭晝閳ь剝鐏掓繛鎾村嚬閸ㄩ亶鏁嶉崱娑欏仭婵犲﹤鎳庨。濂告偨椤栨稑绗у瑙勬礃缁轰粙宕ㄦ繝鍕箺闂備礁缍婇崑濠囧垂娴煎瓨鍎嶆繛宸簼閻撶喐绻涢幋鐐靛ⅹ闁告梹宀搁弻宥囨喆閸曨偆浠稿Δ鐘靛仜閿曨亪寮诲☉娆戠瘈闁告劏鏅濋ˇ鏉课旈悩闈涗粶缂佺粯甯￠崺鈧い鎺戯功缁夌敻鏌涢悩宕囧⒌鐎殿喖鎲＄粭鐔煎焵椤掑嫬钃熸繛鎴欏灩閻掓椽鏌涢幇顒€鈷旈柛鏃堫棑缁辨挻鎷呴崣澶樷偓鍡涙煕鐎ｎ偅灏柍瑙勫灴閹晠宕归锝嗙槑濠电姵顔栭崰妤呭箰閸愯尙鏆﹂柟閭﹀枤绾惧吋淇婇婊呭笡闁绘繄鍏樺娲传閸曨剙绐涙繛鏉戝悑缁诲棝濡甸幇鏉跨闁规崘娅曢悾浼存⒒娴ｅ摜鏋冩俊妞煎妿瀵板﹪宕稿Δ鈧敮闂佸啿鎼崐濠氬储閹剧粯鍋℃繝濠傚閻帞鈧娲樼划宀勫煝鎼淬劌绠ｉ柣妯夸含閻熸繈鏌ｆ惔锛勭暛闁稿酣浜惰棟妞ゆ牗绮岄ˉ姘舵煕瑜庨〃鍡涘煕閹寸偑浜滈柟鍝勬娴滃墽绱撴担鍓叉Ц妞ゆ洦鍘鹃崚鎺撶節濮橆厼浜圭紓鍌欑劍椤洭宕㈤悽鍛娾拻濞撴艾娲ゅ璺ㄧ磼閻樺啿鐏寸€殿喗鎮傞獮妯肩磼濡厧骞堥梺璇插嚱缂嶅棝宕戦崨顓犳殾鐎光偓閸曨剛鍘靛銈嗘⒐閸庢娊宕㈤幘顔界厸閻忕偠濮ら崰姗€鏌℃担绋库偓鍨暦閿濆鏁冩い鎺戝€婚弳銈呪攽椤旂》宸ユい顓炲槻閻ｇ兘骞嗛柇锔筋€囨繝纰樻閸亪宕戦幘缁樷拻濞达絽鎲￠幆鍫ユ偠濮樼厧浜扮€规洘绻傞悾婵嬪礋椤愩倕骞嬮梻浣侯攰閹活亞绮婚幋锔惧祦闁靛繆鍓濋崣蹇斾繆閵堝倸浜惧┑鈽嗗亝閻熲晠寮鍡欑懝闁逞屽墮椤繘鎼圭憴鍕彴闂佽偐鈷堥崜娑㈩敊閸パ€鏀介柣鎰皺閹界姷鈧厜鍋撶紒瀣硶閺嗭箓鏌熺€电浠х紒鈾€鍋撻梻浣规偠閸庢粓宕卞Ο璇叉毇闂傚倸鍊风欢姘焽瑜旈幃褔宕卞ù鏉挎喘椤㈡盯鎮欓幓鎺斺偓顓㈡⒑缁嬫寧婀扮紒瀣箲缁傚秴顭ㄩ崼鐔哄幍闂侀€涚祷濞呮洖鈻嶉崨瀛樼厽闊浄绲介弸娑㈡婢舵劖鐓熼柟鎹愭珪閹癸綁鏌ｉ幒鎴炲暈闁逛究鍔嶇换婵嬪川椤曞懍鍝楅梻浣哥秺椤ユ挻绻涢埀顒勬煕閵婏箑鍔ら柣锝囧厴瀹曞爼鏁冮埀顒勫箺鐎ｎ喗鈷掗柛灞剧懅椤︼箓鏌熼懞銉х煉鐎规洑鍗抽獮鍥敆婢跺苯濮烘繝鐢靛仜濡瑩骞愭繝姘；闁冲搫鎳忛悡鐔兼煙鏉堝墽鍒扮悮姘舵⒑缁嬫鍎忛悗姘煎櫍閹偓妞ゅ繐鐗嗙粻顖溾偓鍏夊亾闁告洦鍋呴幊娆撴⒒娴ｅ憡鎯堥柟鍐茬箳閹广垽宕煎┑鎰稁濠电偛妯婃禍婊呯不娴兼潙绠归弶鍫濆⒔瀹€娑欎繆閹绘帞澧﹂柡宀嬬稻閹棃濮€閻樺弶鐦撻梻渚€娼荤紞鍥╃礊娓氣偓閻涱噣宕橀鑺ユ闂佺粯顭堢亸娆擃敇濞差亝鈷戦柟绋垮椤ュ棛鎮▎鎾寸厵濡炲娴风敮娑氱磼缂佹鈯曟繛鐓庣箻瀹曟粏顦叉い鏇憾閺岋綁鎮╅顫闂備焦瀵уú鏍磹瑜版帒纾归柛褎顨嗛悡鏇㈡煙閻愵剦娈旈柟铏姍閺佸秴顭ㄩ崨顖滐紳闂佺鏈悷褔宕濆澶嬬叆婵﹩鍘剧粻楣冩煙绾板崬骞栭柡瀣ㄥ€濋弻宥堫檨闁告挻宀搁、娆撳冀椤撶偟鐛ュ┑顔矫畷顒勫垂濠靛洢浜滈柡宥冨妼閸ゎ剟鏌涢妶鍡╂疁闁哄本鐩鎾Ω閵壯傚摋缂傚倷鑳舵慨鐢电矙閹烘桅闁告洦鍨扮粻濠氭偣妤︽寧銆冪紒顔垮煐缁绘繈濮€閵忊€虫畬闁诲孩纰嶅姗€锝炶箛鏇犵＜婵☆垵顕ч鎾绘⒑閹呯闁硅櫕鎸剧划顓㈡晸閻樻枼鎷洪梺鍛婄箓鐎氼垶锝為敃鍌涚厱闁哄啠鍋撻柛銊ф暬閹箖鎮滈挊澹┿劎鎲告径鎰；闁瑰墽绮弲鏌ュ箹缁懓澧查柣蹇撶墢缁?
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
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鎯у⒔閹虫捇鈥旈崘顏佸亾閿濆簼绨绘い鎺嬪灪閵囧嫰骞囬鍡欑厯闂佸搫琚崝鎴﹀箖閵忋倕浼犻柛鏇熷灟閸ㄥ鎯€椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔濞插鈧鍠楅幐铏繆閹间礁唯鐟滃宕戦幘娲绘晢闁告洦鍓涢崢閬嶆⒑闂堟侗妯堥柛鐘崇墬閺呭爼顢欓崜褏锛滈梺缁橈供閸犳牠宕濆鍫熺厪闁搞儜鍐句純閻庢鍣崳锝呯暦閹烘垟妲堟俊顖濆吹閺嗘碍绻濋悽闈涗粶闁宦板妿閸掓帗鎯旈妸銉э紱闂佽宕橀褏绮婚悙鐑樼厪濠电偛鐏濋崜濠氭煟閺冨洦顏犻柣顓熺懇閺屾盯鈥﹂幋婵囩亪濡炪値鍋呴幐鎼佸煘閹达附鍋愰柟棰佺閺呴亶姊洪崫銉バｉ柣妤冨Т閻ｇ兘骞囬妯规睏闂佺懓鎼鍌炲磻閹捐绀冩い鏂挎閵娾晜鐓冮柛婵嗗閳ь剚鎮傞崺鈧い鎺戝€搁崢鎾煛鐏炵晫校婵炵⒈浜幃褔宕奸锝傛瀳婵犵數濮伴崹濂革綖婢跺⊕娲偄婵傚缍庡┑鐐叉▕娴滄繈宕甸幒妤佺厪闁割偅绻冮崳娲煟鎼粹€斥挃缂佽鲸鎸荤粭鐔煎炊瑜庨悘鍫㈢磽娴ｅ壊妲洪柟铏崌閹偓顦版惔銏犳瀭闂佸憡娲﹂崣鍐瀶椤曗偓濮婅櫣娑甸崨顓濇睏闂佺顑嗛崝娆忕暦閹达箑绠婚悹鍥皺閿涙粌鈹戦悙鏉戠仸闁绘娲樼粋宥呪攽鐎ｎ偀鎷洪梺鍛婄☉閿曘倖鎱ㄩ埀顒勬⒑閸濆嫬鈧娆㈠鍓佸祦闊洦鎷嬪ú顏嶆晜闁告侗浜濈€氬ジ姊绘担鍛婂暈缂佸鍨块弫鍐晜閸撗傜瑝闂佺懓澧界划顖炲磹閸偅鍙忔俊顖氭惈閼稿綊鏌涘Ο渚殶闁逞屽墲椤煤濡ソ娲偄閼测晛绁﹂梺鎼炲労閸撴岸宕戠€ｎ喗鐓曟い鎰Т閻忊晜顨ラ悙鑼ⅵ婵﹦绮幏鍛村川婵犲啫鍓甸梻浣侯焾閿曘倝鈥﹀畡鎵殾闁瑰瓨绺惧Σ鍫ユ煏韫囨洖啸闁汇倕娲Λ鍛搭敃閵忊€愁槱闂佸憡鍔樺畷鐢垫閻愬搫鍐€妞ゆ挾鍠撻崢鎼佹⒑閹肩偛鍔橀柛搴ㄤ憾閹﹢顢旈崼鐔哄幗闂佽宕樺▔娑㈠几閹存劲搴ㄥ炊瑜濋煬顒€鈹戦垾宕囧煟鐎规洘甯掗～婵嬵敃閵忊晜顥￠梻鍌氬€搁崐鐑芥嚄閸撲礁鍨濇い鏍亼閳ь剙鍟村畷銊╊敍濞戞ê绨ユ繝娈垮枟閵囨盯宕戦幘鏂ユ斀闁斥晛鍟亸锔锯偓瑙勬磸閸斿酣鍩€椤掍胶鈯曢柨姘舵煃瑜滈崜娆撯€﹂悜钘夎摕闁靛牆顦粻鎺楁煙閻戞ê鐏ュ┑顔哄灲濮婃椽宕ㄦ繛鎺濅簻閻ｇ兘顢楅崟顐ゅ幋闂佺鎻梽鍕磹閻戣姤鐓涘璺侯儏閻忊晛霉閻樿櫕灏﹂柟顔筋殘閹叉挳宕熼鈧ˇ鈺呮⒑閸涘﹥灏甸柛鐘崇墵楠炲啯瀵奸幖顓熸櫓闂佹悶鍨归ˇ鏉课涢崘銊ф殾闁靛ň鏅╅弫瀣煕濞戝崬娅樻俊顐㈡噹閳规垿鏁嶉崟顐℃澀闂佺锕ラ悧婊堝极椤曗偓楠炴帒顪冨┑鍡樺枠妞ゃ垺绋戦埥澶娾枎閹邦喖濞囬梻鍌欑濠€閬嶆惞鎼淬劌绐楁俊銈呮噺閸嬪倹绻涢幋娆忕仾闁稿﹤鐖奸弻锝夊箛椤撗冩櫛闂佺粯甯楅崕鎶解€﹂懗顖ｆЪ闂佺粯鎼换婵嗩嚕婵犳碍鍋勫瀣濞堟繈姊虹紒姗嗘當闁绘妫楅埢鎾澄旈崨顔尖偓鐢告偡濞嗗繐顏璺哄閺屾稓鈧綆浜峰銉╂煟閿濆洤鍘撮柟顔哄灲閹剝鎯旈敍鍕ㄥ亾椤撶儐娓婚柕鍫濇婵呯磼閹绘帞浠㈤崡杈ㄦ叏濡炶浜惧┑顔硷功缁垶骞忛崨顖滈┏閻庯綆浜濋鍕節濞堝灝鏋熼柨鏇ㄥ亞缁骞樼紒妯绘珳闂佺粯鍔栧娆戞閻愮儤鐓欓梺顓ㄧ畱婢х増銇勯弬鎸庡枠婵﹨娅ｇ槐鎺懳熼搹閫涚礃濠电姵顔栭崰鎾诲磹濠靛棛鏆﹂柟鐗堟緲闁裤倖淇婇妶鍕厡闁告瑥妫濆娲传閸曨偀鍋撻幖浣瑰€舵繝闈涱儏閻撴洟鏌￠崘銊у闁绘挾鍠愰妵鍕箻鐠虹儤鐏侀梺鐟板暱閸燁垶濡甸崟顖ｆ晣闁绘劖娼欐禒鎾⒑绾懏鐝紒顔芥尭铻為柛鎰╁妷濡插牊淇婇婊冧沪婵炶尙鍠庨～蹇涙倻濡顫￠梺瑙勵問閸ｎ喖危椤斿皷鏀介柣鎰级閳绘洟鏌涘▎蹇撴殭妞ゆ洩绲剧换婵嗩潩椤撶喐鐝冲┑鐘灱濞夋盯鏁冮敃鍌涘仾闁逞屽墯娣囧﹪鎮欓鍕ㄥ亾閺嶎厽鍋嬫俊銈傚亾妞ゎ偅绻堟俊鎼佸Ψ閹邦厼娴┑顔瑰亾闂佺粯锕╅崑鍛村棘閳ь剟姊绘担鍝ユ瀮婵☆偄瀚灋婵°倕鎳忛崐鍫曟煟閺冨洦顏犵痪鎯с偢閺屻倝姊归幇顔俱偡婵犮垼顫夎摫闁靛洤瀚伴、鏇㈠閵忋埄鍞堕梺缁樻尪閸婃牠濡甸崟顔剧杸闁圭偓鍓氭禒鑲╃磽娴ｇ懓濮堢紒瀣笧閹广垹鈹戠€ｎ偄浠洪梻鍌氱墛閸掆偓闁靛繈鍊栭悡鏇炩攽閻樻彃顏撮柣鎺戝⒔閹喖鈻庨幘瀵稿幈濡炪倖鍔楁慨鎾礉濠婂牊鐓冮梺鍨儏閻忊晝绱掓潏銊ョ闁逞屽墾缂嶅棝宕戦幒妤€纾块柕澶嗘櫆閻撴洟骞栨潏鍓х？缂佺姵褰冭彁闁搞儜宥堝惈濡炪們鍨哄ú鐔煎极閹版澘鐐婇柕濞垮劚閻忥繝姊虹拠鍙夊攭妞ゎ偄顦叅婵☆垵娅ｉ弳鍡涙煙闂傜鍏岀€规挷绶氶弻鐔兼倻濡儤顔曢梺鍝勫暙閻楀棝鎮為崹顐犱簻闁圭儤鍨甸埀顒傜帛缁嬪顓奸崨顏呮杸闂佺粯锚绾绢參銆傞弻銉︾厽闁规儳顕幊鍐懚閻愬弬娑欙紣娴ｅ湱銈伴梺绋跨箰閻倿寮诲☉銏犳閻犳亽鍓遍敐鍚冲酣宕惰瀹搞儲銇勯鍕殻闁诡喒鍓濋幆鏃堝煡閸℃瑢鍋撻搹顐＄箚闁绘劦浜滈埀顑惧€栫粋宥咁煥閸繄鏌堥柣搴㈢⊕鐪夌紒璇叉閺屻倗绮欑捄銊ょ驳闂佺娴烽崰鏍蓟瀹ュ唯闁靛鍎撮弫鍧楁⒑缁洘娅呴柟铏～蹇曠磼濡顎撻梺鍛婄☉閿曘倝寮抽崼銉︹拺闁圭瀛╃壕鎼佹煕閵娿劌鍚归柛鎺撳笚缁绘繂顫濋鈧崬銊ヮ渻閵堝棙灏甸柛瀣枛閹偓銈ｉ崘鈹炬嫼闂佸憡绋戦敃銈嗘叏閿曞倹鐓曢柣妯虹－婢ц京绱掗纰辩吋闁轰焦鍔欏畷鍗炩枎閹寸姵顫岄梻鍌欑窔濞佳勵殽韫囨洘顫曢柡鍥╁Х娑撳秹鏌熼幆褍顣崇痪鍓ф櫕閳ь剙绠嶉崕閬嶅箯閹达妇鍙曟い鎺嶇贰濞堜粙鏌ｉ幇顓炵祷闁哄棴缍侀弻娑㈠煛鐎ｎ剛鐦堥悗瑙勬磸閸旀垿銆佸Ο琛℃斀闁割偁鍨归埀顒傚仱濮婂宕掑▎鎴М闂佽绁撮埀顒冪М濞差亝鏅濋柛灞捐壘閸嬪秹姊洪崗鑲┿偞闁哄懏绮撳鍐差煥閸喓鍘遍悷婊冮叄閵嗗啴宕卞☉妯煎幈闂佸湱鍎ら崵姘炽亹閹烘挻娅滈梺鍓插亞閸ｃ儲绂掗埡鍛厽闁靛繆鏅涢悘锟犳煟閳哄﹤鐏︾€殿喖顭烽弫鎰板川閸屾稒鈷愮紒缁樼箞瀹曟帒顫濇鏍ф櫔闂備浇顕ф鎼佹倶濮橆剦鐔嗘慨妞诲亾闁轰礁鍟撮、鏃堝礋椤撶喐顔曠紓鍌欑椤戝牓顢氶幎钘夌睄闁割偅绻勯崝锕€顪冮妶鍡楃瑨闁稿﹦绮粙澶婎吋閸℃劒绨婚梺鍝勭▉閸嬪嫭绂掗敃鍌涚厽闁规崘娉涢弸鎴澢庨崶褝韬い銏＄☉閳藉宕￠悙鎻掝瀳闂傚倷鐒﹂幃鍫曞磹閺嶎厽鍋￠柨鏇炲€堕埀顒€鍟存俊鐑藉煛閸屾埃鍋撻柨瀣ㄤ簻闊洦鎸搁顐⒚归悩闈涗壕缂佺粯鐩弫鎰板川椤旇姤鍊烽梻浣瑰濞诧附绂嶉鍫澪ュù锝囩《濡插牊淇婇鐐存暠妞ゎ偄绉撮埞鎴︽倷閸欏妫炵紓浣割槹婵炲﹪鐛€ｎ喗鐓ラ悗锝庡墰閻﹀牏绱掗崜褍顣奸懣銈夋煕鐎ｎ偅宕岄柟鐓庣秺椤㈡洟鎮㈤崫銉ょ敖闂傚倸鍊烽悞锕傚箖閸洖绀夐煫鍥ㄦ尨閺嬫牗绻涢幋顓熺窙缂傚秵鐗犻悡顐﹀炊閵婏妇鍘介梺绋款儏閿曨亪骞冮柨瀣闂傚牊绋堥崑鎾诲箻閹靛灕鍏炬棃宕ㄩ瑙勫闂備線娼荤€靛矂宕㈡總绋跨閻庯綆鍠楅悡娆戠磼鐎ｎ偒鍎ラ柣鎾炽偢閺岋紕浠︾拠鎻掝潎濡炪們鍨洪敃銏ゅ箖閳哄懏鍋ㄧ痪鏉款槺闂傤垱绻濋悽闈涗哗闁规椿浜炲濠囨嚍閵壯呭骄婵犵數濮存导锝夋偄閻撳海鍔﹀銈嗗坊閸嬫挻銇勯鍕殻濠碘€崇埣瀹曟﹢濡搁妷銉渐闂傚倷绀佸﹢閬嶅箠閹捐秮娲敇閵忊€充粧濡炪倖娲嶉崑鎾搭殽閻愬樊鍎旈柡浣稿暣閸┾偓妞ゆ帒瀚哥紞鏍煕濞戞鎽犻柣鎾跺枑缁绘盯宕卞Δ鍕伃婵炲瓨绮嶉崕鎶藉煘閹达附鍊锋繛鍫濈仢閸斿棝姊洪崨濠冣拹闁搞劌鐏濋锝囨嫚瀹割喖鎮戦梺鍓插亽閸嬪懘顢撻崶顒佲拻濞达絽鎲￠崯鐐烘煙缁嬫鐓奸柟顔惧厴閸╋繝宕ㄩ鐘垫毇闂備浇顕栭崹搴ㄥ川椤旂晫鏉洪梻鍌欑缂嶅﹪寮ㄩ崡鐑嗘綎闁荤喐瀚堝☉銏犵闁靛ě鍕啎闂備線娼ц墝闁哄應鏅犲顐﹀炊椤掍胶鍘遍梺闈涱檧缁蹭粙宕濆顑芥斀妞ゆ梻鍘ч埀顒€娼″濠氭晬閸曨亝鍕冮梺缁樺姦閸撴盯藝閵娧呯＝濞达綀顫夐埛鎺楁煕閻樺磭澧甸柕鍡曠閳藉顫滈崱妯哄厞婵＄偑鍊栫敮鎺椝囬鈧幆鈧い蹇撶墛閳锋帒霉閿濆懏鍟為柟鑼焾閳规垿顢氶埀顒€顭囪閸欏懎鈹戦埥鍡楃仩闁告艾顑呴悾鐑藉矗婢跺瞼鐦堥梻鍌氱墛娓氭宕曢幇鐗堢厱閻庯絻鍔屾慨鍌涙叏婵犲偆鐓肩€规洖銈搁幃銏㈡偘閳╁喛绱氱紓鍌氬€风欢锟犲窗閺嶎厽鍋夐柣鎾冲瘨濞兼牠鏌ц箛鎾磋础缁炬儳鍚嬮幈銊ノ旈埀顒€螞濞嗘挻鍋╅悹楦裤€€閺€浠嬫煃閽樺顥滃ù婊勭箘缁辨帞鎷犻懠顒€鈪甸梺缁樹緱閸ｏ絽鐣峰鈧俊鍛婃償閿濆懏鐎剧紓浣诡殘閸犳牠銆佸☉妯锋婵☆垰鎼敮顏堟⒒閸屾艾鈧兘鎳楅崜浣稿灊妞ゆ牜鍋涚粈澶愭煛瀹ュ骸骞栭柛銊ュ€圭换婵嬫濞戞瑧鍘愰悷婊冪Х閻忓啯绻涙潏鍓хК婵炲拑缍侀、娆撳即閵忊檧鎷洪梺闈╁瘜閸樺吋绂嶉悙顒傜闁稿繗鍋愰幊鍥煙椤旀儳浠╅柕鍥ㄥ姍楠炴帡骞樼捄鍝勭疄闂備浇顕ч崙鐣岀礊閸℃稑纾婚柛娑卞灟閻掑﹪鏌ｉ姀鐘冲暈闁绘挸鍟伴幉绋款煥閸繄顦┑鐘绘涧椤戞垹绱為弽褜鐔嗛悹楦裤€€婵洦绻涘顔荤盎闁绘挻鐩弻娑㈠箛閸忓摜鏁栧銈忓瘜閸ㄨ京鎹㈠☉娆愮秶闁告挆鍐ㄧ厒濠电姰鍨婚幊鎾绘晝椤忓嫮鏆﹂柣銏㈩焾閸楁娊鏌ｈ箛姘煎殭缂傚秴锕獮鍐煛娓氬洤鏅犲銈嗘⒒閸樠勭珶閸儲鈷戦柛婵勫劚閸撻亶鏌涢妸銊︾【妞ゎ偄绻愮叅妞ゅ繐鎷嬪Λ鍐ㄢ攽閻愭潙鐏︽慨妯稿姂椤㈡瑦绻濋崘顏嗙槇濠电偛鐗嗛悘婵嬪几閿旂晫绠鹃柛娑卞枟缁€鍫㈢磼?
	if len(data) < 72 {
		return
	}
	eventType := gjson.GetBytes(data, "type").String()
	if eventType != "response.completed" && eventType != "response.done" && eventType != "response.failed" &&
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
	bodyLooksLikeSSE := looksLikeOpenAINonStreamingSSEBody(body)
	if isEventStreamResponse(resp.Header) || bodyLooksLikeSSE {
		return s.handleSSEToJSON(resp, c, body, originalModel, mappedModel)
	}

	// For OAuth accounts, also fall back to a body-content heuristic because
	// the upstream may omit the Content-Type header while still sending SSE.
	// This heuristic is NOT applied to API-key accounts to avoid false
	// positives on JSON responses that coincidentally contain "data:" or
	// "event:" in their text content.
	if account.Type == AccountTypeOAuth && (bodyLooksLikeSSE || bytes.Contains(body, []byte("data:")) || bytes.Contains(body, []byte("event:"))) {
		return s.handleSSEToJSON(resp, c, body, originalModel, mappedModel)
	}

	usageValue, usageOK := extractOpenAIUsageFromJSONBytes(body)
	if !usageOK {
		if bodyLooksLikeSSE {
			return s.handleSSEToJSON(resp, c, body, originalModel, mappedModel)
		}
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
	UserAgent          string // 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌＄仦绋垮⒉鐎垫澘瀚埀顒婄秵娴滄繈顢欓崨顓涙斀闁绘劕寮堕埢鏇灻瑰鍐煟鐎殿噮鍋婂畷鍫曨敆娴ｅ搫甯鹃梻濠庡亜濞诧箑煤閺嵮勬瘎闂傚倷绀侀幉锛勬崲閸愵喓鈧啯绻濋崒銈嗙稁缂傚倷鐒﹂…鍥偡瑜版帗鐓曢柕澶嬪灥閸犳艾顭囬懡銈囩＝闁稿本鐟чˇ锔姐亜閿曞倷鎲剧€殿噮鍋嗛幏鐘绘嚑椤掍焦顔曢梻浣告惈濞层垽宕归崷顓犱笉闁挎繂妫涚弧鈧梺闈涢獜缁辨洜绮婚幘鍓佺＝鐎广儱鎷戦煬顒侇殽閻愭彃鏆ｉ柛鈺佸瀹曟﹢鍩℃担绋课ら梻鍌欑劍鐎笛呮崲閸屾娑樷枎閹惧磭鐛ラ梺鍝勭▉閸樹粙鍩?User-Agent
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

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犳娊鏌嶈閸撴繈锝炴径濞掓椽寮介‖鈩冩そ閺佸啴宕掗妶鍡樻珖濠电偛顕慨鎾敄閸℃稒鍋傞煫鍥ㄧ⊕閻撴洘銇勯幇鍓佹偧缂佺姵鐗曢…璺ㄦ崉閸濆嫷浠鹃梺闈涙搐鐎氫即銆侀弴銏℃櫜闁糕剝鐟Σ褰掓⒒娴ｅ憡鎯堥柣顓烆槺閹广垹鈹戦崱娆愭闂佸壊鍋呭ú鏍ф暜闂備線娼ч敍蹇涘磼濠婂嫸绱￠梻鍌氬€搁崐鐑芥嚄閸洍鈧箓宕奸妷顔芥櫈闂佺鐬奸崑娑㈡偪妤ｅ啯鐓熸俊顖涱儥閸ゆ瑩鏌﹂崘顏勬灈闁哄本娲熷畷鐓庘攽閸ヨ埖锛侀梻浣告啞閻熴儳鎹㈠鈧濠氭偄閸忚偐鍔烽梺鎸庢磵閸嬫捇鏌涘Ο缁樺唉闁哄本绋撻埀顒婄秵閸嬪懎鐣风仦缁㈡闁绘劕寮堕崰妯尖偓娈垮枛婢у酣骞戦崟顖椻偓锕傚箳閺傝儻鈧潡姊婚崒姘偓宄懊归崶顒婄稏濠㈣泛锕﹂弳锔芥叏濡炶浜惧銈冨灪娣囨椽藝鐟欏嫷娈介柣鎰綑閻忔挳鏌熼搹顐ょ煉闁诡喗鐟╅弻鍛槈濮樿鲸鏅ㄩ梻鍌氬€峰ù鍥綖婢跺﹦鏆︽俊顖濄€€閺嬫牗绻涢幋鐐垫噮妞も晝鍏橀弻鐔兼倻濡鐭紓浣筋嚙濡繈寮诲☉妯锋瀻闊浄绲剧瑧濠电姵顔栭崰鏍洪銏犺摕鐎广儱娲﹂崰鍡涙煕閺囥劌浜炲ù鐓庣焸濮婄儤娼幍顕呮М缂備礁顦遍幊鎾绘偩閻戣姤鍋勭痪鎷岄哺閺呪晝绱撴担鍦槈妞ゆ垵鍊垮畷鎴﹀箻濠㈠嫭妫冨畷銊╊敊鐟欏嫬顏归梻鍌欒兌椤㈠﹪宕戦幇鏉跨；婵炴垶鐟ょ换鍡椻攽閸屾碍鍟為柣鎾跺枛閺屸€崇暤椤旇崵顦﹀ù鐙€鍣ｅ铏圭磼濡湱绻侀梺闈╃秶缁蹭粙鎮鹃悜钘夌闁绘劏鏅滈～宥呪攽閳藉棗鐏ｉ柍宄扮墕鍗遍柣妤€鐗呯换鍡涙煟閹板吀绨婚柍褜鍓氶悧鏇＄亱婵炴挻鍩冮崑鎾垛偓瑙勬礃閸ㄥ潡鐛崶顒€绀傛い鎰╁灪閸犳鈧鍠涘▔娑㈠煝鎼淬劍鍊锋い鎺嗗亾闁伙附绮撳缁樼瑹閳ь剙顭囪閹囧幢濡炪垺绋戣灃闁告劦浜為悾鍫曟⒑缁嬭法鐏遍柛瀣仱閹繝鎮╃紒妯煎幈闂佸綊鍋婇崹浼存儍濞差亝鐓曢柍鍝勵儑閹ジ鏌嶈閸撴繈锝炴径濞掗缚绠涘☉妯虹€繛瀵稿Т椤戝洤鐣垫笟鈧弻娑㈠Ψ椤旂厧顫梺绋款儏椤戝寮诲☉銏╂晝闁绘灏欐禒鐓幬旈悩闈涗沪闁告梹娲熼崺鐐哄箣閿旇棄浜瑰銈嗘閸嬫劖瀵奸崶褉鏀介柣鎰▕濡插綊鏌ｉ埡濠傜仸鐎殿噮鍋婂畷姗€顢欓懞銉︾彆闂備礁鍚嬫禍浠嬪磿閺屻儲鍋熺€瑰嫭澹嬮弨浠嬫煟閹邦厽缍戦柣蹇曞枛閺屾盯濡搁敂濮愪虎闂佽鍠掗埀顒佹灱閺嬪酣鏌熼幆褜鍤熼柛妯哄船閳规垿鎮欓弶鎴犱桓闂佹寧娲﹂崑濠傤嚕閵婏妇绡€婵﹩鍘鹃崢杈ㄧ節閻㈤潧孝闁哥噥鍋呴幈銊﹀緞閹邦厾鍘撻悷婊勭矒瀹曟粌鈹戠€ｅ墎绋忔繝銏ｆ硾閳洖煤椤忓嫮鍘搁梺鍛婂姦娴滅偤藝椤栨稓绠鹃弶鍫濆⒔閸掍即鏌熼懞銉х煉鐎殿喗濞婇弫鍌涙叏閹邦亞鐩庨梻浣烘嚀閹碱偄螞濡や胶顩查柟顖嗗本瀵岄梺闈涚墕妤犲憡绂嶅鍛＜缂備焦锕懓鎸庮殽閻愭彃鏆ｅ┑锛勫厴閸┾剝绗熼埀顒€螞閻戣姤鈷戦梺顐ゅ仜閹冲繘宕曢鈧濠氬磼濮橆兘鍋撻幖浣哥９闁归棿绀佺壕褰掓煙闂傚顦︾痪鎯х秺閺岋綁骞嬮敐鍛呮捇鏌涙繝鍌涘仴闁哄被鍔戝顕€宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ偒鍎ラ柛搴＄箲閵囧嫰骞嬪┑鎰枅閻庢鍠涢褔鍩ユ径鎰潊闁绘﹢娼ф慨鍫曟⒒娴ｅ憡鍟為柛鏃撻檮缁傚秹寮介‖顒婄秮瀹曞ジ濡烽敂瑙勫闂備胶顭堥張顒勬嚌閻愵剛顩查柣鎰劋閻撴洟鏌曟繛鍨闁告凹鍋婇弻鐔肩嵁閸喚浠奸梺瀹犳椤︻垶锝炲鍫濆耿婵°倓鐒﹂悵姘攽閿涘嫬浜奸柛濠冪墪铻炲ù锝呮憸閺嗭箓鏌ｉ姀鐘冲暈闁稿骸閰ｉ獮鏍庨鈧俊浠嬫煃闁垮鐏╃紒杈ㄥ笧閳ь剨缍嗛崢濂稿礉閸偁浜滄い鎰靛墰閻ｇ敻鏌＄仦鍓ф创鐎殿噮鍣ｉ崺鈧い鎺戝閸ㄥ倿鏌涢…鎴濇珮闁搞倖娲栭埞鎴︽偐鐎圭姴顥濋柛銉︽尦濮婅櫣鍖栭弴鐐测拤闂佹寧姘ㄩ埀顒侇問閸犳牕顭囬垾鎰佹綎濡わ箒锟ユ禍褰掓煙閻戞ê鐏ｉ柛鐘诧躬濮婅櫣绮欑捄銊ь唹闂佹寧娲忛崹褰掓偩閻戣棄绠ｉ柨鏇楀亾缂佺姷绮换婵嬪閻樺樊浠惧┑顔硷工缂嶅﹪鐛箛娑樺窛闁哄鍨崇槐鍫曟⒑闂堟冻绱￠柛娑卞幘閵堬箓姊婚崒姘偓鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偛顦甸崹鍓ф惥娴ｈ銇濋柡浣稿暣瀹曟帒顫濋鍌楀亾椤撱垺鈷掑〒姘搐婢ь喚绱掓径濠庡殶濠㈣娲濋妵鎰板箳閹捐泛骞堟俊鐐€ら崢浠嬪垂閸偆顩查柣鎰暩绾惧ジ鏌ｅΟ鍨惞闁伙絾妞介弻鈥崇暆閳ь剟宕伴幘璇茬劦妞ゆ帒鍊归弳鈺傘亜椤撶偟澧涚紒鍌涘笩椤﹀綊鏌″畝鈧崰鏍箹瑜版帩鏁冮柕蹇曞濞兼岸姊绘担鍛婃儓闁活厼顦辩槐鐐寸瑹閳ь剟鎮伴鈧獮鎺懳旈埀顒€螞濡警鐔嗛柛鎾茬贰閻掔偓銇勯妷锕€鍝烘慨濠勭帛缁楃喖鍩€椤掆偓椤洩顦查悡銈夋煏閸繃绀岄柛瀣尭椤繈顢橀悢鍝勫殥婵＄偑鍊栧ú鈺冪礊娴ｅ壊鍤曢柛顐ｆ礀缁狅絾銇勯幘璺烘瀻闁稿繐锕ユ穱濠囨倷椤忓嫧鍋撻弽褜鍟呭┑鐘宠壘绾惧鏌熼幆褍顣崇痪鍓у帶閵嗘帒顫濋浣规倷濠电偛鎳庣粔褰掑蓟閻旂⒈鏁婄紒娑橆儐閻ｅジ姊洪悷鏉跨骇闁圭鍟块悾鐑芥偄绾拌鲸鏅㈤梺绋挎湰椤ㄥ棝寮查鍌滅＝闁稿本鑹鹃埀顒佹倐瀹曟劙宕妷褏鐓嬮悗瑙勬礀濞层劑鎯岄崱妞尖偓鎺戭潩閿濆懍澹曟俊鐐€戦崹鍝勭暆閹间礁鏋侀柟鐗堟緲楠炪垺绻涢崱妯曟垹绮婇锔解拺閻犲洦褰冮崵杈╃磼闊厾鐭欓柟顔矫～婵堟崉閾忚鐓ｆ繝鐢靛█濞佳囶敄閸℃稑鐓曢柟杈鹃檮閻撴瑧绱掔€ｎ偄顕滈柟鐧哥悼缁辨帡鎮╅搹顐㈤瀺缂備胶绮粙鎾诲箯閻樹警妲鹃梺浼欑秮娴滃爼寮诲鍥╃＜婵☆垵顕ч弳妤冪磽娓氬洤鏋撻柡鍛█閸ㄩ箖寮崒婊呯劸婵烇絽娴傚▍鐐烘⒒?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵挳鎮欏ù瀣壕闁告縿鍎虫稉宥夋煛瀹ュ骸骞楅柣鎾存礃閵囧嫰骞囬崜浣荷戠紓浣插亾闁逞屽墰缁辨帡鎮欓鈧崝銈嗙箾绾绡€鐎殿喗妲掗ˇ鍓佺磼閻樺磭娲撮柡浣瑰姍瀹曘劑顢楅崒姘瘓闂傚倸鍊风粈渚€骞夐敓鐘冲仭闁靛／鍕簥闂佸湱鍎ら〃鍛存倿閸偁浜滈柟鐑樺灥閺嬨倖绻涢崗鐓庡缂佺粯鐩畷锝嗗緞鐏炶В鍚傞梺缁樻尪閸婃繈寮婚妸鈺佺睄闁割偆鍠愬▓浼存⒑?input_tokens 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锔剧磽閸愨晜绀嬫慨濠勭帛閹峰懘宕ㄦ繝鍐ㄥ壍婵犵數鍋涢惇浼村垂閽樺鏆︾憸鐗堝笒閹硅埖銇勯幘璺盒＄紒妤€顦靛铏圭磼濮楀棛鍔稿銈嗘肠閸涱垯绗夐柣鐔哥懃鐎氼喚绮绘ィ鍐╃叆婵犻潧妫濋妤€顭胯閸犳牠鍩為幋锔筋€愰梺绋款儐閸旀危閹版澘绠虫俊銈勭娴滃綊姊洪崨濠傚鐟滄澘鍟撮獮妤呮偨閸涘ň鎷洪梺鍛婄箓鐎氼厼锕㈡导瀛樼厽闁冲搫锕ら悘锕傛煏閸℃洜绐旂€殿喗鎸虫慨鈧柣妯荤垹閸ャ劎鍘卞┑鐐村灥瀹曨剟寮搁妶鍚ょ懓顭ㄩ崟顓烆潷婵?cache_read_tokens闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灪閵囧嫰濡搁妷锕€娈楁繝纰樺墲閹告娊鐛幒妤€绠ｆい鎾跺枎閸忓﹪姊绘担鐟邦嚋缂佽鍊块獮濠傤吋閸℃瑧褰鹃梺鍝勬储閸ㄦ椽鍩涢幒鎳ㄥ綊鏁愰崼鐕佷哗闂侀潧妫楅敃顏堝蓟閿熺姴纾兼繛鍡樺灩閻涖垽鎮楃憴鍕闁搞劌娼￠悰顔碱潨閳ь剟鐛崶銊﹀闁荤喐澹嗗Σ锝夋⒒閸屾瑧绐旀繛浣冲洦鍋嬮柛鈩冦亗濞戞鏃堝川椤撱垺鏆呭┑鐘垫暩婵潙煤閿曞倹鍋傞柡鍥ュ灪閸婄敻鏌ㄥ┑鍡涱€楀ù婊勭箘閳ь剝顫夊ú鏍儗閸岀偛钃熼柍銉ョ－閺嗗棝鎮楅敐搴″鐞氾附淇婇悙顏勨偓鎴﹀垂濞差亗鈧啯绻濋崒婊勬濠殿喗銇涢崑鎾斥攽閳╁啯鍊愬┑鈥崇埣瀹曞崬鈻庨幇顔锯枆婵犵數濮烽弫鍛婃叏閻戝鈧倹绂掔€ｎ亞鍔﹀銈嗗坊閸嬫捇鏌涢悢缁樼《闁汇儺浜ｉˇ褰掓煛鐏炲墽娲寸€殿噮鍣ｉ崺鈧い鎺戝閸嬶繝鏌嶆潪鎵窗闁搞倖娲橀妵鍕箛閸撲胶鏆犵紒鐐劤閵堟悂寮婚敐澶婄疀妞ゆ帒顦▓鍫曟⒑閸涘﹦鎳冩俊顐ｎ殜閳ユ棃宕橀鍢壯囨煕閳╁啰鎳冩い顐庡洦鈷戞繛鑼额嚙瀵箖鏌涢悩鎰佹畼缂侇喒鏅犻幃銏ゆ偂鎼粹€崇ギ闂備胶绮崝蹇涘疾濠婂牆妫橀柍褜鍓熷缁樻媴閾忕懓绗￠梺鍛婃⒐閻楁洟鈥﹂崶褉鏋庨柟鐐綑閳ь剙鐖奸弻銊╁即閻愭祴鍋撹ぐ鎺撳亗闁绘柨鍚嬮悡鐔兼煛閸ャ劍鐨戞い锔肩畵閺岋綀绠涢妷褏鏆ら梺鍝勭灱閸犳牠銆佸▎鎾崇畾鐟滃苯危閹扮増鈷戦悹鍥ｂ偓铏亶闂佽崵鍟块弲鐘诲Υ娓氣偓瀵粙顢橀悙鐢靛幀闂備線娼ч敍蹇涘川椤旇姤鐝ら梻鍌氬€搁崐鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞鐗犻、鏇㈠煕濮橆厽銇濋柡浣稿暣瀹曟帒鈽夊Ο鑽ゆ澖闂傚倷娴囬鏍垂鎼淬劌宸濇い鏃傜摂閸熷矂姊婚崒娆掑厡缁绢厼鐖煎畷婊冣攽鐎ｎ偄浠梺闈涚箞閸婃洜澹曟繝姘厵闁硅鍔曢悡鎰版煕閻樺弶顥㈤柡灞剧洴瀵挳濡搁妷銉骄闂備礁鐤囬～澶岀矙閹捐埖顫曢柟鎹愵嚙绾惧吋鎱ㄥΟ鑽ゆ▊闁挎稑瀚壕濂告煃瑜滈崜姘辩箔閻旂厧鐒垫い鎺戝閳ь兛绀侀～婵嬫嚋閸偅鐝抽梻浣虹《閸撴繈鎮烽姣綊宕熼娑氬幗闁瑰吋鎯岄崹鎶藉礆娴煎瓨鐓熼煫鍥ㄦ煥閸濊櫣鈧鍠楄ぐ鍐煘閹寸姭鍋撻敐搴′簮闁归攱妞藉娲捶椤撗呭姼濠电偞鎸抽ˉ鎾跺垝婵犳艾绠婚悗闈涙憸椤旀洟鏌ｉ悩鍙夊巶闁告侗鍨卞▓鎼佹⒒娴ｅ摜锛嶇紒顕呭灦楠炴劙宕滆閸ㄦ繈鏌嶉崫鍕櫝闁逞屽墯濮婅崵鍒掓繝鍐攳妞ぱ冪摠缁绘繈鎮介棃娴躲儵鏌℃担鍛婂暈闁逛究鍔戦幃婊堟寠婢跺鈧剟姊鸿ぐ鎺擄紵缂佲偓娴ｅ搫顥氬┑鐘崇閻撶喖鏌熼柇锕€澧柟顖氱墦閺屾盯濡搁妷銉㈠亾閸ф钃熸繛鎴欏焺閺佸啴鏌ㄥ┑鍡橆棤妞わ负鍎崇槐鎾存媴閸濆嫅锟犳煕濡や礁鈻曠€殿喛顕ч埥澶娢熼柨瀣澑闂備礁鎲″ú锕傚磻閸曨剚鍙忛柕蹇嬪€栭埛鎴炵箾閼奸鍤欐鐐搭殜閺岀喖鎮烽悧鍫濇灎濡ょ姷鍋涢崯顐ョ亙闂佸憡渚楅崢濂告倵椤撶儐娓婚柕鍫濇婵倿鏌涙繝鍐⒌妤犵偛顦埢搴ㄥ箛椤斿墽妲囨繝鐢靛仜閻楀棝鎮樺┑瀣嚑闁绘柨鐨濋弨浠嬫煟閹邦垼鍤嬮棅顒夊墰閳ь剚顔栭崰鏍€﹂悜钘夋瀬鐎广儱顦粈瀣亜韫囨挻鍣瑰┑顖欏嵆濮婃椽鎳￠妶鍛呫垺绻涚拠褏鐣抽柕鍥ㄥ姍瀹曟﹢鍩￠崒姘紟闂佺澹堥幓顏嗗緤鐠恒劌顥氶柦妯侯棦瑜版帗鏅查柛娑卞枟閸庢捇鏌″浣插亾閺傘儲鏂€闂佺粯锕╅崰鏍倶椤忓牊鐓ラ柡鍥悘鏌ユ煕閵娾晝鐣洪柡浣稿€块幃娆擃敆閳ь剛澹曢鐐粹拺闂傚牊渚楅悡顓犵磼閻樺啿鐏╁瑙勬礋椤㈡盯鎮欑划瑙勫缂傚倸鍊烽悞锕傛晪闂佽楠忕粻鎾诲箯閹寸偞缍囬柍瑙勫劤娴滈箖鎮峰▎蹇擃仾缂佲偓閳ь剛绱掗崜褑妾搁柛妯诲劤鍗遍柟鐗堟緲缁犲鎮归崶顏勭毢闁挎稒绮岄埞鎴︻敊閻偒浜滈悾鐑筋敆閸曗斁鍋撻崒鐐蹭紶闁告洖鐏氱€靛矂姊洪棃娑氬濡ょ姵鎮傞悰顔碱潨閳ь剟寮诲☉婊呯杸闁哄倸妫禍顏堛€佸鑸垫櫜闁搞儯鍔岄悵鏉库攽閻愬瓨缍戞い鎴濇閿濈偛顓奸崨顏呮杸闂佺粯顭堥婊冾啅閵夆晜鐓欓柧蹇ｅ亜婵秶鈧鍠栭悥濂哥嵁閺嶃劍濯撮柛锔诲幖楠炴姊洪悷鏉挎倯闁诡垰鐭傚畷鐟扮暦閸モ晝鐓嬮梻鍌氱墛閼颁粙鎮烽柇锔惧弳闂佸憡娲﹂崢楣冩偂娓氣偓濮婃椽鏌呭☉姘ｆ晙闂佸憡鏌ㄩ惌鍌炪€佸鈧畷妤呮偂鎼达絿鐛梺璇插嚱閹儵宕熼顐＄椽缂傚倸鍊搁崐鎼佸磹閹间礁纾归柣鎴ｅГ閸婂潡鏌ㄩ弴鐐测偓鐢稿焵椤掑﹦鐣电€规洖鐖奸、妤佹媴缁嬪灝顥楅梻鍌欑窔濞佳囨偋閸℃蛋鍥ㄥ鐎涙ê浜楅梺鍝勬储閸ㄦ椽鎮¤箛鎾斀闁绘劘灏欐禒銏犫攽閿涘嫭娅曢柣銉邯椤㈡﹢鎮╅煫顓烆棜闂備線娼уú銈団偓姘緲椤曪綁骞橀鍛櫆闂佺硶鍓濋浼村箣閻樼數锛濇繛杈剧悼椤牓鍩€椤掆偓濠€閬嶅极椤曗偓閹瑩宕崟顓炲Е婵＄偑鍊栫敮鎺楁晝閿斿墽鐭撻柣銏犳啞閻撴洟鎮楅敐搴濈盎妞ゅ繆鏅犻弻宥囩磼濡纾抽梺璇″枟閻熲晠骞冨鍏剧喖鏌ㄧ€ｎ亶浼滃┑鐘垫暩閸嬬偤骞愭繝姘殞闁诡垼鐏愬ú顏勎ч柛婊€绀佸▓銊╂煟閻樺厖鑸柛鏂挎捣婢规洘绺介崨濠勫幗濠碘槅鍨伴幖顐﹀汲闁秵鐓熼煫鍥ㄦ尵閹界姵銇勯妸锝呭姦闁诡喗鐟╅獮鎾诲箳閹炬惌鍞圭紓鍌氬€烽悞锕傘€冮崼銉ョ獥闁哄稁鍘奸弰銉╂煃瑜滈崜姘跺Φ閸曨垰绠抽柟瀛樼箥娴犲ジ姊?
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
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犳娊鏌嶈閸撴繈锝炴径濞掓椽寮介‖鈩冩そ婵℃悂鍩￠崒姘闂備胶纭堕崜婵嬎囨ウ瑁や汗闁圭儤鎸诲▍婊堟⒑閸撴彃浜介柛瀣噽缁辩偤宕煎┑鍐╂杸闂佺粯鍔曞鍫曞闯閽樺鏀芥い鏃囧Г鐏忥附绻濋埀顒佺瑹閳ь剙顫忛搹鍦＜婵☆垰娴氭禍婊嗙亽婵犵數濮村ú銈囧閸ф鐓欓柟娈垮枛椤ｅジ鏌ｉ幘瀛樼闁诡喗顨婇弫鎰板川椤撶喎鈧酣姊绘笟鍥у季闁搞劌鐖煎濠氭晬閸曨亝鍕冮梺鍛婃寙閸曨偄鐏￠梻鍌欒兌绾爼寮插☉姘ｅ亾濮橆厽绶查柣锝呭槻椤粓鍩€椤掑嫬鏋侀柛灞剧矋瀹曞鏌曟繛鍨偓妤呮偡濠靛鈷掗柛灞捐壘閳ь剟顥撳▎銏狀潩鐠鸿櫣鍔﹀銈嗗笒閸婂憡绂掑鍫熺厾婵炶尪顕ч悘锟犳煛閸涱厾鍩ｆい銏＄洴閹瑩寮堕幋婊呯处濠碉紕鍋戦崐鏍偋閹捐纾规俊銈呭暙婵剟鏌嶈閸撴瑨鐏冮梺缁橈耿濞佳勭濠婂牊鐓曢柣鏂挎啞鐏忥妇鈧娲樺浠嬪极閹剧粯鍋愰柤纰卞墻濡蹭即姊绘担鍝ユ瀮婵℃ぜ鍔庡▎銏ゆ晸閻樿尙鍔﹀銈嗗笂閼宠泛煤鐎涙ɑ鍙忓┑鐘插暞閵囨繄鈧娲忛崝宥囨崲濠靛纾兼慨妯哄悑缂嶅鈹戦悩娈挎毌婵℃彃鎳樺畷瑙勫鐎涙ɑ娅囬梺闈涚墕閻楀繑绂嶆潏銊х瘈闁汇垽娼у瓭闂佹寧娲忛崐婵嗙暦椤栫偞鍋愰悹鍥皺閸濇绻涚€电孝妞ゆ垵妫濋幃锟犳偄閸忚偐鍘甸柣搴㈢⊕椤洨绮婚弽顐熷亾閻熺増鍟炵紒璇茬墦瀵鈽夐姀鐘殿唺闂佸搫鍊归娆撳汲椤撶姷纾藉ù锝堟鐢盯鏌ｉ埡濠傜仸妤犵偛锕幃鈺伱虹紒妯绘珜闂備線鈧偛鑻晶鎾煕閳瑰灝鐏茬€规洖銈搁崺妤呭煛娴ｅ嘲顥氶梺鑽ゅ枑閻熴儳鈧凹鍘剧划鍫ュ礃椤忓棛锛滃銈嗘閸嬫劙鎮為幖浣圭厱闁冲搫鍟禒杈殽閻愬樊鍎旈柡浣稿暣閸┾偓妞ゆ帒瀚崐宄扳攽閻樻彃顏柛鐘冲姈缁绘繃绻濋崒婊冾暫闂?	usageLog.ChannelID = optionalInt64Ptr(input.ChannelID)
	usageLog.ModelMappingChain = optionalTrimmedStringPtr(input.ModelMappingChain)
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犳娊鏌嶈閸撴繈锝炴径濞掓椽寮介‖鈩冩そ婵℃悂鍩￠崒姘闂備胶纭堕崜婵嬎囨ウ瑁や汗闁圭儤鎸诲▍婊堟⒑閸撴彃浜介柛瀣噽缁辩偤宕煎┑鍐╂杸闂佺粯鍔曞鍫曞闯閽樺鏀芥い鏃囧Г鐏忥附绻濋埀顒佺瑹閳ь剙顫忛搹鍦＜婵☆垰娴氭禍婊嗙亽婵犵數濮村ú銈囧閸ф鐓欓柟娈垮枛椤ｅジ鏌ｉ幘瀛樼闁诡喗顨婇弫鎰板川椤撶喎鈧酣姊绘笟鍥у季闁搞劌鐖煎濠氭晬閸曨亝鍕冮梺鍛婃寙閸曨偄鐏￠梻鍌欒兌绾爼寮插☉姘ｅ亾濮橆厽绶查柣锝呭槻椤粓鍩€椤掑嫨鈧線寮崼婵嗙獩濡炪倖鎸鹃崰鎾广亹椤愨懇鏀介柣妯虹仛閺嗏晛鈹戦鐐毈妞ゃ垺鐟╁浠嬵敇閻愯尙鈧厽绻涢弶鎴濇倯婵炲吋鐟ュú璺ㄧ磽閸屾瑧顦︽い鎴濇椤㈡牕鈻庤箛锝囧姺闂佸搫绋侀悘鎰亹閹烘挸浜归悗鐟板閸犳牠宕滄导瀛樷拺缂備焦锚婵洭鏌ㄩ弴妯哄姦濠碉紕鏁诲畷鐔碱敍濮橀硸鍞洪梻浣虹《閸撴繈濡甸悙瀵哥彾闁哄洢鍨洪埛鎺懨归敐鍫綈闁稿濞€閺屾盯寮捄銊у姱閻庤娲橀崝娆忕暦缁嬭鏃堝焵椤掑嫭瀚呴柣鏂挎憸缁犻箖鏌熺€电浠ч柣鎿冨灡缁绘稑鐣濇繝浣烘晼缂備浇椴搁幐濠氬箯閸涘瓨顥堟繛鎴炵煯閹絾绻濋悽闈涗粶闁绘妫濇俊鍓佺矙濞嗙偓缍庨梺鎯х箰濠€杈╁閸忛棿绻嗘い鏍ㄧ矊閸旓箓鏌熷畡閭︾吋婵﹥妞藉畷銊︾節鎼达絽濮搁梻浣规偠閸庮垶宕濆畝鍕劦妞ゆ巻鍋撳褍閰ｉ崺鈧い鎺戝枤濞兼劖绻涢崣澶屽ⅹ閻撱倝鏌ｅΟ娆惧殭缁炬儳顭烽弻鐔煎箲閹邦剛鍘梺鍝ュТ濡繈寮诲☉銏犲嵆闁靛鍎遍～顐㈩渻閵堝繗绀嬮柛搴ㄤ憾閸╃偤骞嬮敂钘夆偓鐑芥煠绾板崬澧柛鏇㈢畺濮婃椽宕ㄦ繝鍐ｆ嫻闂佹悶鍔嶇€笛勭┍?
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
	// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ゆ繛纭风節瀵鎮㈢悰鈥充壕闁汇垺顔栭悞楣冨疮閸濄儳纾藉ù锝嗗絻娴滅偓绻濋悽闈浶㈡繛璇х畵閹繝宕橀鍛瀾濠电姴锕ら悧鍡欑矆閸喐鍙忔俊顖涘绾儳顩奸崨瀛樷拺闁告稑锕ユ径鍕煕閵娿倕宓嗙€规洩绻濋獮搴ㄦ嚍閵壯冨箞闂備焦鏋奸弲娑㈠窗濮橆兘鏋旈柡鍐ｅ亾闁靛洤瀚粻娑㈡晲閸曨垰浠愰梻?UserAgent
	if input.UserAgent != "" {
		usageLog.UserAgent = &input.UserAgent
	}

	// 濠电姷鏁告慨鐑藉极閸涘﹥鍙忛柣鎴ｆ閺嬩線鏌涘☉姗堟敾闁告瑥绻橀弻锝夊箣閿濆棭妫勯梺鍝勵儎缁舵岸寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ゆ繛纭风節瀵鎮㈢悰鈥充壕闁汇垺顔栭悞楣冨疮閸濄儳纾藉ù锝嗗絻娴滅偓绻濋悽闈浶㈡繛璇х畵閹繝宕橀鍛瀾濠电姴锕ら悧鍡欑矆閸喐鍙忔俊顖涘绾儳顩奸崨瀛樷拺闁告稑锕ユ径鍕煕閵娿倕宓嗙€规洩绻濋獮搴ㄦ嚍閵壯冨箞闂備焦鏋奸弲娑㈠窗濮橆兘鏋旈柡鍐ｅ亾闁靛洤瀚粻娑㈡晲閸曨垰浠愰梻?IPAddress
	if input.IPAddress != "" {
		usageLog.IPAddress = &input.IPAddress
	}

	if apiKey.GroupID != nil {
		usageLog.GroupID = apiKey.GroupID
	}
	if subscription != nil {
		usageLog.SubscriptionID = &subscription.ID
	}

	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犳娊鏌嶈閸撴繈锝炴径濞掓椽寮介‖鈩冩そ閺佸啴宕掗妶鍡樻珖濠电偛顕慨鎾敄閸℃稒鍋傞煫鍥ㄧ⊕閻撴洘銇勯幇鍓佹偧缂佺姵鐗曢…璺ㄦ崉閸濆嫷浠鹃梺闈涙搐鐎氫即銆侀弴銏℃櫜闁糕剝鐟Σ褰掓⒒娴ｅ憡鎯堥柣顓烆槺閹广垹鈹戦崱娆愭闂佸壊鍋呭ú鏍ф暜闂備線娼ч敍蹇涘磼濠婂嫸绱￠梻鍌氬€搁崐鐑芥嚄閸洍鈧箓宕奸妷顔芥櫈闂佺鐬奸崑娑㈡偪妤ｅ啯鐓熸俊顖涱儥閸ゆ瑩鏌﹂崘顏勬灈闁哄本娲熷畷鐓庘攽閸ヨ埖锛侀梻浣告啞閻熴儳鎹㈠鈧濠氭偄閸忚偐鍔烽梺鎸庢磵閸嬫捇鏌涘Ο缁樺唉闁哄本绋撻埀顒婄秵閸嬪懎鐣峰畝鈧埀顒侇問閸犳洜鍒掑▎鎾扁偓渚€寮撮姀鈥充簻闂佺偓鑹鹃崐鎼佀夊顓犵瘈婵炲牆鐏濋弸鐔兼煥閺囨娅婄€规洘绮撻弫鍐磼濞戞ü绨靛┑鐘绘涧閸婃悂骞夐敓鐘冲亗婵炴垯鍨洪崐鍫曟煟閹伴偊鏉洪柛銈嗙懃閳规垿顢欓悡搴樺亾婵犳艾鐒垫い鎺嶇贰閸熷繘鏌涢悩鎰佹當妞ゎ厼娲ら埢搴ㄥ箳閺傚墽鑳洪梻鍌氬€风欢姘跺焵椤掑倸浠滈柤娲诲灡閺呭爼顢涘鍛紲闂佺鏈銊︾墡闂備線娼ч悧鐐电礊娴ｅ摜鏆︽慨妞诲亾闁糕斁鍓濋幏鍛槹鎼淬垺姣庡┑鐘垫暩婵兘寮崨濠冨弿闁圭虎鍠楅弲婵嬫煏閸繃绀岄柛瀣尭椤繈鎮℃惔鈾€鎷俊鐐€戦崹娲箰閹间讲鈧箓濡搁埡浣侯槹濡炪倖鎸炬刊顓㈠疮鎼粹檧鏀介柣姗嗗枛閻忚鲸绻涙径瀣创妞ゃ垺鐗犲畷鍗炩枎閹寸姷鍔堕梻浣稿閸嬪棝宕板鍥ㄥ床闁糕剝绋掗悡鏇熺箾閸℃绂嬫俊鑼焾閳规垿顢欓懖鈹絾銇勯妸锝呭姦闁诡喗鐟╁鍫曞箣閻樺灚鍣梻浣稿⒔缁垶鎮ч弴銏＄畳闂備焦瀵х换鍌炈囬弶搴撴瀺婵犲﹤鎳愮壕濂告煏婵炑冩噹缁楋紕绱撴担铏瑰笡缂佽鐗撻獮鍐╃鐎ｎ偒妫冨┑鐐村灦椤ㄥ懘骞楃€ｎ喗鈷戦悹鍥皺缁犵増绻涚€涙顣茬紒鍌氱Ч椤㈡棃宕煎☉鎺戜壕濞达絽澹婂鈺呮煙妫颁胶鍔嶉柍璇叉湰娣囧﹪濡堕崶顬儵鏌涚€ｎ偆娲寸€规洦鍨堕、娑橆煥閸涱垳鏆ラ梻浣虹帛濮婂宕㈣閹偤宕归鐘辩盎闂佸湱鍎ら崹鐢割敂閳哄懏鐓㈤柛鎰典簻閺嬫盯鏌＄仦璇插闁宠鍨垮畷鍗烆潨閸℃﹫绱掗梻鍌欒兌椤牆霉閻戣棄绐楅柡宥庡幖閽冪喖鏌￠崶銉ョ仼闂佸崬娲︾换婵嬫濞戞瑱绱為梺鍛婃煥椤﹁京妲愰幘璇茬＜婵ɑ鐦烽姀銈嗙厽婵°倓鐒︾亸顓熴亜椤愩垻绠荤€规洦鍋婂畷婵嬪磹閻斿壊浠╅梺褰掝棑婵炩偓妤犵偞鎹囬獮鎺楀箣椤撶偛螚闂傚倸鍊搁崐椋庢濮橆剦鐒介柤濮愬€楃粈濠囨煕閵夈垺娅囩紒鈧繝鍌樷偓鎺戭潩閿濆懍澹曢梻浣虹帛娓氭宕抽敐鍛殾婵せ鍋撴い銏＄懇閹虫牠鍩℃繝鍌涙殽缂傚倸鍊搁崐鐑芥倿閿斿墽鐭欓柟杈惧瘜閺佸嫰鏌涢埄鍐槈闁汇倗鍋撻妵鍕箳閸℃ぞ澹曢梻浣哥枃椤宕归崸妤€绠栭柍鍝勫暞鐎氭氨鎲稿鍡楊嚤闁规壆澧楅悡鐔煎箹濞ｎ剙鐏╅柛銈庡墴閺屾盯骞樼捄鐑樼亪闂侀潧妫旂欢姘嚕椤曗偓瀹曠厧鈹戦崼顐㈡櫍婵犵數鍋犻幓顏嗗緤娴犲鍌ㄦ繝濠傜墕缁狀垱绻涘顔荤凹闁抽攱甯￠弻娑氫沪閻愵剛娈ゆ繝鈷€鍕闁哄矉绻濆畷銊╊敊閻愵剙鍨遍梻浣筋嚃閸垳绮堟笟鈧垾锕傚Ω閳轰線鍞堕梺缁樻煥閹碱偊鐛崼鐔虹瘈缁剧増蓱椤﹪鏌涢妸锔界凡闁挎洏鍨介弻鍡楊吋閸℃澹掓繝鐢靛仜濡瑩宕归悢鐓庣；闁圭偓鏋奸弸鏃堟煕椤垵鏋熼柣蹇撶墦濮婂搫效閸パ€鍋撻弴鐏绘椽濡歌閸ㄦ繈鎮归崶銊с偞婵℃彃鐗婃穱濠囶敍閻愬瓨鏆犲銈庡亜缁夋挳鍩為幋锔藉€烽柛娆忣樈濡繝姊洪幖鐐插婵炰匠鍥舵晪闁挎繂顦伴幆鐐淬亜閹扳晛鈧鎯侀崼銉︹拺婵懓娲ら悘鍙夌箾娴ｅ啿瀚々鐑芥煥閺囩偛鈧綊鎮￠弴銏＄厪濠电偛鐏濋崝銈夋煕閳哄绉柡灞界Ч婵＄兘顢涘鍛闁诲氦顫夊ú鏍偉閸忛棿绻嗛柟闂寸劍閺呮粓鏌ｉ敐鍛板濠电偛娲濠氬磼濞嗘埈妲梺鍦拡閸嬪﹤鐣烽幇鏉夸紶闁靛鍔屽皬闂傚倷绶￠崜娆戠矓鐎靛摜纾奸柍杞版€ヨぐ鎺撳亹鐎瑰壊鍠栭崜閬嶆⒑缂佹ɑ灏甸柛鐘崇墵瀵鎮㈤崗鑲╁姺闂佹寧娲嶉崑鎾愁熆瑜滈崰姘辨閹烘鍋戞繛鍡楃箲婢跺嫰鏌ｉ幘瀛樼闁哄矉绲借灒缁炬澘宕慨鏇㈡⒑閸忓吋绶叉繛纭风節瀵鍨惧畷鍥ㄦ畷闂侀€炲苯澧寸€规洑鍗冲浠嬵敇濠ф儳浜惧ù锝囩《閺嬪酣鏌熼悙顒佺稇婵炲牄鍎靛娲濞戞艾顣哄┑鐐茬湴閸旀垵鐣烽姀銈庢晬婵﹫绲鹃弬鈧梻浣虹帛钃辨い鏃€鐗犲鎶筋敍閻愬鍘遍梺瑙勫劤椤曨厾绮婚悙鎼闁绘劖褰冮弳锝団偓瑙勬礃鐢繝骞冨▎鎾崇煑濠㈣泛澶囬崑鎾活敆閸曨兘鎷虹紓浣割儓濞夋洟鎮橀柆宥嗙厱閻庯綆鍋嗛妴鎺旂磼椤旂⒈鐓奸柡浣瑰姈瀵板嫮鈧綆鍓欓獮宥夋⒒娴ｈ櫣甯涢柛銊﹀劶閹筋偊姊虹紒妯诲蔼闁稿骸纾Σ鎰板箳閹宠櫕姊婚埀顒婄秵閸撴盯鏁嶉悢铏圭＝濞达絼绮欓崫娲煙閻熺増鎼愰柣锝囧厴楠炴帡骞嬮鐔峰厞婵＄偑鍊栭崹鐓幬涢崟顒傤洸濡わ絽鍟悡娆撴⒒閸屾凹鍤熸い锔煎閻ヮ亪宕滆閸も偓濡炪値浜滈崯浼村焵椤掑﹦绉靛ù婊勭箞楠炴垿鏁愭径瀣幗濡炪倖鎸鹃崑鐐核夊鍫熺厪闁糕剝锚缁楁帗銇勯锝囩疄闁轰焦鍔欏畷銊╊敆閳ь剟藟濮樿埖鈷掗柛灞剧懄缁佺増銇勯弴鐔哄⒌鐎规洦鍨堕、鏇㈡晝閳ь剟鎮欐繝鍕枑鐎广儱娲ㄩ悳缁樼箾閹寸儑渚涙繛鎾愁煼閺屾洟宕煎┑鍥舵￥婵犫拃灞藉缂佽鲸甯￠、娆愮附缁嬪灝鍙婇梻浣筋嚃閸ｏ絿绮婚弽褏鏆﹀┑鍌滎焾閸楁娊鏌ｉ弬鍨骇閻庢俺鍋愮槐鎾诲磼濞嗘埈妲銈嗗灥濡盯鍩€椤掑倻鎳楅柛鎰劵閳ь剙鐏濋湁闁绘挸娴烽幗鐘绘煟閹惧瓨绀冪紒缁樼箞濡啫鈽夊▎妯伙紗闂備礁鎲￠悷銉р偓姘嵆瀵鈽夐姀鐘靛幋闂佽鍨庨崒姘兼濠电姷顣槐鏇㈠磻閹达箑纾归柡宥庡亝閺嗘粌鈹戦悩鎻掝伀闁活厼妫楅湁闁挎繂鐗滈崵澶屾喐閻楀牆淇柡浣革躬濮婂搫鈻庨幆褏浠╂繛瀛樼矒缁犳牠骞冨畡鎵虫瀻闊洦鎼╂禒鍓х磼閸撗冧壕闁诡喖鍊垮璇测槈閵忕姵顥濋柣鐘充航閸撻亶濡舵径瀣幍濡ょ姷鍋涢悘婵嬫倶閳哄懏鐓冮柦妯侯樈濡偓婵犳鍠掗崑鎾绘⒑缂佹〞鎴﹀礈濮橆儵娑㈡偄閸忓皷鎷绘繛杈剧到閹诧繝宕悙鐑樼厵缂佸瀵чˉ銏⑩偓瑙勬磸閸旀垿銆佸☉姗嗘僵妞ゆ帒鍊搁幖鎼佹⒒閸屾艾鈧嘲霉閸ヮ剦鏁嬬憸宥夛綖濠靛鏅濋柛宀嬪缁嬪繑绻濋姀锝嗙【闁愁垱娲栫叅妞ゅ繐瀚崬銊╂偡濠婂嫮绠為柟铏崌瀹曠螖娴ｅ弶瀚兼俊鐐€栧濠氬磻閹惧墎纾煎璺侯儐鐏忥箓鏌熼鎯т槐鐎规洖缍婇、鏇㈡偐鏉堚晝娉块梻鍌欑濠€閬嶅磿閵堝鏄ュ┑鐘叉搐缁€澶愭煟閺冨洦顏犵痪鍓у帶椤潡鎳滈棃娑橆潔濠电偞鍤崶顭戞⒖婵犮垼娉涢鍥╁姬閳ь剚绻濋悽闈浶㈤柛濠冩倐椤㈡棃顢曢敂鐣屽幗濡炪倖鎸鹃崰搴∶归鈧弻鈥崇暆閳ь剟宕伴弽顓溾偓浣糕枎閹惧厖绱堕梺鍛婃处閸嬪嫯顤傞梻鍌氬€风粈渚€骞栭锔藉殣妞ゆ牗顕卞☉妯滄棃宕担瑙勬珝濠电娀娼ч崐濠氣€﹂崼銉у祦闁靛繈鍊楅崣鎾绘煕閵夛絽濡介柣鎾卞劜閵囧嫯绠涢幘鍓佇ㄥ┑顔硷工椤嘲鐣烽幒鎴旀瀻闁规惌鍘借ⅵ闂傚倷绀佸﹢閬嶅煕閸儱纾诲┑鐘叉噺椤ャ倝姊绘担鍛婂暈闁告梹鐗滅划濠囧箻椤旂厧鍤戦梺鍝勭▉閸樹粙鎮″☉銏＄厱闁靛鍨哄▍鍛归悪鍛存闁逛究鍔戦弫鎰板川椤撗傜磾婵犳鍠栭敃銈夆€﹀畡鎵殾闁圭儤鍨熼弸搴ㄦ煙鐎涙绠撴い顒€顦靛缁樻媴缁涘娈愰梺鍝ュУ閻楃娀骞冮妷鈺傚亗閹艰揪绲惧▓楣冩⒑閸︻厼顣兼繝銏★耿瀹曟﹢鍩€椤掆偓椤啴濡堕崱妤€顫囬梺鎼炲妼缂嶅﹪骞冨Ο鑽ょ瘈闁搞儯鍔庨崢鐢告⒑缁嬭法绠洪柛瀣姍瀵啿顫滈埀顒勫蓟濞戙垹妫橀悹鎭掑妿閸旑垶姊虹拠鈥虫灓闁稿鍊曢悾鐤亹閹烘繃鏅濋梺缁橆焾娴滎剟鎯勬惔銊︹拻濞达絽鎲＄拹锟犳煕鎼存稑鈧繂鐣疯ぐ鎺撳仺缂佸鐏濋崵鎴︽⒑闂堟稓澧曟俊顐ｇ〒缁粯銈ｉ崘鈺冨幈闂婎偄娲﹀ú鏍暜鐠鸿　鏀芥い鏃囧亹鏁堥梺鍝勫閳ь剙纾弳鍡涙倵閿濆骸澧扮悮锔戒繆閻愵亜鈧垿宕瑰ú顏佲偓锕傚醇閵夊娲濠氬Ψ閿旀儳骞愰梺璇插嚱缁插宕濆畝鍕劦妞ゆ帊绶″▓妯讳繆閸欏濮嶆鐐村笒铻栧┑鐘插暞濞呮挾绱撴担鍝勪壕濠殿垵濮ょ粋宥夘敆閸曨偉袝闁诲函缍嗛崰妤呭煕閹达附鍋ｉ柛銉岛閸嬫捇鎼归銈勭按闂傚倷鐒﹂幃鍫曞礉鐎ｎ剙鍨濇繛鍡樻尵瀹撲線鏌涢幇鐢靛帥闁绘挶鍎甸弻娑滅疀閹炬枼鎸冮梺鍝ュУ閻楃娀鐛崘顭戠叆闁稿繐澧介崰鎾寸閹间礁鍐€鐟滃本绔熼弴鐑嗘富闁靛牆妫欑亸顏堟煕閵婏附銇濋柛鈺傜洴楠炴帡骞婇妸銉хШ闁轰焦鍔欏畷銊╊敊閸忓吋鐣奸梻鍌欑閹芥粓宕伴幘璇茬；闁绘劕鐏氬畷鍙夌箾閹寸偟顣查悗姘皑閹叉瓕绠涘☉娆忎患闂佹眹鍨婚。浠嬪磻閹捐埖鍠嗛柛鏇ㄥ墰椤︺劑姊洪幖鐐插闁诲繑绻堥幃楣冩倻閼恒儱浜滅紒鐐妞存悂寮查鈧埞鎴︽倷閺夋垹浠搁柤瑁ゅ€濋弻娑氣偓锝庡亝瀹曞瞼鈧娲栭妶绋款嚕閹绢喗鍋勯柛婵嗗缁犳椽姊婚崒娆戭槮闁圭⒈鍋婂畷鎰板箹娴ｇ懓鈧埖绻濋棃娑卞剰缂佺姵鐗犻弻鐔告綇妤ｅ啯顎嶉梺?
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
			Platform:              PlatformFromAPIKey(apiKey),
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

	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤濞硷繝寮婚悢琛″亾閻㈡鐒剧€涙繄绱撴担鐣屽牚闁稿﹥绻堝濠氭晝閳ь剝鐏掓繛鎾村嚬閸ㄩ亶鏁嶉崱妞绘斀闁绘劕寮堕埢鏇灻瑰鍕疄鐎殿喗褰冮…銊╁醇閻斿搫骞楅梻渚€鈧稑宓嗘繛浣冲嫭娅犳い鏇楀亾妤犵偞鐗犲鍫曞箣椤栨繂鎯堟繝娈垮枛閿曘劌鈻嶉敐鍥у灊婵炲棙鍨跺畷澶愭煏婵炑冨€荤粈鍐⒒閸屾瑨鍏屾い顓炵墦椤㈡牠宕堕鈧壕濠氭煏閸繃鍣介柡鍡畵閺岀喐娼忛崜褏鏆犻柛銉ョ摠缁绘繈濮€閿濆棛銆愬銈嗗灥濞差厼鐣烽姀銈庢晜闁告侗鍨抽惁鍫ユ⒑濮瑰洤鐏叉繛浣冲嫮顩风憸鏃堝蓟濞戞埃鍋撻敐搴′簼閻忓浚鍙冮弻宥囨嫚閼碱儷褏鈧娲忛崝搴ㄥ焵椤掍胶鈯曟い顓炴喘瀹曘垽鏌嗗鍡欏幗闂婎偄娲﹀ú鏍ㄧ閳哄倶浜滈柡鍥ф鐎氼參宕ｈ箛娑欑厱鐎光偓閳ь剟宕戦悙鐑樺亗婵炴垶鍩冮崑鎾诲礂婢跺﹣澹曢梺璇插嚱缂嶅棝宕滃☉婧惧徍闂備浇顕х€涒晠顢欓弽顓為棷妞ゆ洍鍋撶€规洘绮岄埢搴ㄥ箻瀹曞洤骞掗梺鐟板悑閻ｎ亪宕濆澶婄９闂佸灝顑冩禍婊堟煙閻戞ê鐏ラ柍褜鍓欑紞濠囧箖濡ゅ拋鏁婇悘蹇旂墬閺傗偓?primary/secondary 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻濠氭煕閳规儳浜炬俊鐐€栫敮濠囨嚄閸洖鐓濋柟鍓х帛閻撴盯鏌涘☉鍗炴灓缂佺姵锕㈤弻娑㈠箳閹惧磭鐟ㄩ梺瀹狀嚙闁帮綁鐛Ο铏规殾闁搞儴娉涢弫钘夆攽閻樻鏆滅紒杈ㄦ礋瀹曟垵鈽夐姀鈥冲壄闂佺粯鍨煎Λ鍕婵犳碍鐓欓柟瑙勫姦閸ゆ瑧绱掗埀顒勫礃閳瑰じ绨婚梺鍝勫暙閸婂摜鏁崼鏇熺厾闁哄娉曟禒銏ゆ煃鐟欏嫬鐏撮柟顔界懇瀵爼骞嬮悩杈╃婵犵绱曢崑娑㈡偤閵娾晛绠栭柛灞惧嚬閸ゆ洟鏌＄仦璇插姎闁绘挻鐩弻娑樷槈閸楃偞鐏堥梺閫炲苯澧伴柡浣割煼瀵鈽夊鍛澑闂佺懓鐏濋崯顖滅懅婵犵數鍋涢悺銊у垝閹惧墎涓嶉柡宓本缍庡┑鐐叉▕娴滄粌顔忓┑鍡忔斀闁绘ɑ褰冮弳娆愩亜閿旇娅婃慨濠冩そ瀹曘劍绻濋崘銊╃€洪梻浣哄帶缂嶅﹦绮婚弽顓炴槬闁靛繒濯崥瀣熆鐠虹尨宸ラ柛鐐妼椤啴濡堕崱妯烘殫闂佸摜濮甸幑鍥х暦閵忥紕顩烽悗锝庡亽濡懎顪冮妶鍡楀闁搞劎鍎ゅ鍕礋椤掑倻顔曢梺鍛婄懃椤﹁鲸鏅堕鍌滅＜闁稿本绋戝ù顕€鏌涢埡瀣瘈鐎规洏鍔戦、娆撳箚瑜嶇粻鐐烘⒒娴ｇ瓔鍤欓柛鎴犳櫕缁辩偤宕卞☉妯硷紱闂佸憡娲﹂崜姘跺矗韫囨洍鍋撻獮鍨姎妞わ缚鍗抽幃锟犲即閵忥紕鍘撻梺鍛婄箓鐎氼剟寮冲▎寰濆綊鎮╁ú顏勬懙濡炪們鍔婇崕鐢稿箖濞嗘挸绾ч柟瀵稿仩閻т焦淇婇妶鍥ラ柛瀣洴閺佸啴顢旈崨顒傜畾濠殿喗绻傞惌鍫澪ｆ總鍛娾拺闁硅偐鍋涢埀顒佸姍瀹曟垿骞樼紙鐘电畾闂佺粯鍔︽禍婊堝焵椤戞儳鈧繂鐣烽姀锝冧汗闁圭儤鍨归ˇ顕€姊洪悷鎵憼缂佽瀚板鎶芥晜閼恒儱寮垮┑锛勫仩椤曆勭妤ｅ啯鈷戦柣鎰閸旂數绱掗悩铏磳闁绘侗鍠氶埀顒婄秵娴滄牠寮ㄦ禒瀣厱闁绘瑢鍋撻柛鐘虫崌瀹曟繈骞嬪┑鎰稁缂傚倷鐒﹁摫濠殿垱鎸抽弻娑㈡晜鐠囨彃绠洪梺鐓庣仢濞差厼顫忛搹鐟板闁哄洨鍠愰悵鏃堟煟鎼淬垻顣叉繝銏★耿閹箖鎮块妯规睏闂佸湱鍎ら幐楣冨礉閸洘鍊垫鐐茬仢閸旀岸鏌熼搹顐㈠妞ゃ垺鐟﹀鍕箛椤撴稒瀚奸梻浣侯攰閸嬫劙宕戝☉銏犵闁逞屽墴閺岋綀绠涢弴鐐板摋婵犳鍠撻崐婵嬨€佸鈧畷褰掝敊鐟欏嫭鏉搁梻浣虹帛閿氶柣蹇斿哺瀵娊鍩℃担鍙夋杸濡炪倖姊婚崑鎾诲汲椤掑嫭鐓欓柧蹇ｅ亞缁犵偞顨ラ悙鍙夘棥妞わ附濞婇弻锟犲幢濡偐鐓夊┑顔硷工椤嘲鐣烽幒鎴旀瀻闁规惌鍘借ⅵ闂傚倷绀侀幗婊堝窗鎼粹槅鐒介柨鐔哄Х瀹撲線鏌″搴″季闁轰礁鍟撮弻銊╁即濡も偓娴滃墽绱掗悙顒€绀冩い鏇嗗洤鐓橀柟杈鹃檮閸嬫劙鏌熺紒妯轰刊濞寸姵鍎抽—鍐Χ鎼粹€茬凹濠电偛寮堕敃銏′繆閻㈢绀嬫い鏂垮⒔閺夋悂姊洪崷顓炰壕婵炲吋鐟ラ埢宥夊閵忋垻锛?
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

// normalizeOpenAIPassthroughOAuthBody 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻缁橆殽閻愭潙鐏村┑顔瑰亾闂侀潧鐗嗛幊鎰版偪閳ь剚淇婇悙顏勨偓鏍涙担鑲濇盯宕熼浣稿妳婵犵數濮村ú锕傚煕閹寸姵鍠愰柣妤€鐗嗙粭鎺懨瑰鈧崡鎶藉蓟濞戞瑦鍎熼柕濠忛檮闁款參姊虹€圭姵顥夋い锔诲灥閻忔帡姊绘担鍝ヤ虎妞ゆ垵妫濆鍫曞箹娴ｅ厜鎷洪柣鐘叉处瑜板啴宕垫潏鈺冪＝鐎广儱鎳忛ˉ鐐电磼閸屾氨效闁诡喗鐟╁畷顐﹀礋椤撗勑ラ梻鍌欑劍鐎笛兠哄澶婄；闁圭偓鐣禍婊勩亜韫囨挸顏╅柡鍡到閳规垿鍨惧畷鍥х厽閻庤娲栧畷顒冪亙闂侀€炲苯澧い顓炴喘楠炲鏁傜憴锝嗗缂傚倷绀侀鍡涱敄濞嗘挸纾块柟鎵閻撴瑧绱掔€ｎ亞浠㈤柍閿嬫⒐娣囧﹪宕ｆ径濠傤潚濡ょ姷鍋炵敮鈥愁嚕椤曗偓閸┾偓妞ゆ帒鍊甸崑鎾愁潩椤撶喓娈ら梺閫炲苯澧伴柟铏尰缁旂喐绻濋崶褏鐛ラ梺鍝勭▉閸樿偐绮堥崼銉︾厱闁归偊鍘鹃妶鎾煛閳?OAuth 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌＄仦绋垮⒉鐎垫澘瀚埀顒婄秵娴滄繈顢欓崨顓涙斀闁绘劕寮堕埢鏇灻瑰鍐煟鐎殿噮鍋婂畷鍫曨敆娴ｅ搫甯鹃梻濠庡亜濞诧箑煤閺嵮勬瘎闂傚倷绀侀幉锛勬崲閸愵喓鈧啯绻濋崒銈嗙稁缂傚倷鐒﹂…鍥偡瑜版帗鐓曢柕澶嬪灥閸犳艾顭囬懡銈囩＝闁稿本鐟чˇ锔姐亜閿曞倷鎲剧€殿噮鍋嗛幏鐘绘嚑椤掍焦顔曢梻浣告惈濞层垽宕归崷顓犱笉闁挎繂顦伴悡銉╂煛閸愩劌鈧懓鈻嶉弴銏＄厱婵☆垱瀵чˉ澶愭煃鐟欏嫬鐏︽鐐诧躬閺屾稒绻濋崘銊ヮ潚閻庤娲橀崹鍧楃嵁濡偐纾兼俊顖滃帶楠炴绻濆閿嬫緲閳ь剚鍔欏畷鎴﹀箻缂佹鍙嗛梺鍝勬处閿氶柛鏃撳閳ь剝顫夊ú妯煎枈瀹ュ洦宕叉繝闈涱儏閻愬﹦鎲歌箛娑辨晩闁圭儤鍩堝〒濠氭煏閸繃顥為柍閿嬪浮閺岋繝宕担闀愬枈閻庤娲忛崹浠嬪箖娴犲宸濆┑鐐靛亾鐎氬ジ姊洪懡銈呅㈡繛鑼█閸┾偓妞ゆ巻鍋撶痪缁㈠弮閸┾偓妞ゆ巻鍋撴い顓犲厴瀵鈽夊Ο閿嬵潔闂佸憡顨堥崑鐐烘倶瀹ュ鈷戦柛婵勫劚鏍￠梺缁橆殘婵挳锝炶箛鏇犵＜婵☆垵顕ч鎾绘⒑閸涘﹦鈽夐柨鏇樺劦瀹曟洟骞橀弬銉︽杸闂佸疇妫勫Λ妤佺娴犲鐓曟俊顖氬悑濞呮洜绱掗纰卞剰妞ゆ挸鍚嬪鍕熺紒妯荤彆闂傚倷绀佹竟濠囧磻閸涱垱宕查柛鈩冪☉缁犳椽鏌￠崶銉ョ仾闁抽攱鍨块弻锟犲磼濡搫濮曞┑鐐叉噹閹虫﹢寮诲☉銏″亞濞达絽鎽滄禒顓㈡⒑閸涘﹤绗掗柨鏇ㄤ邯閻涱喖顫滈埀顒勩€佸▎鎾村仼閻忕偠妫勭粻鐐烘⒒閸屾瑧鍔嶉悗绗涘吘娑欑瑹閳ь剟銆佸鎰佹▌闂佺粯渚楅崳锝夌嵁閸ヮ剦鏁囨繝闈涚墢閻ｇ偓淇婇悙顏勨偓鏍礉濡ゅ懎绐楅柟閭﹀厴閺嬪秴鈹戦悩鍙夊闁绘挻绋撻埀顒€鍘滈崑鎾绘倵閿濆骸澧扮悮锕傛⒒娴ｇ瓔鍤冮柛鐘冲浮瀵煡鎮╅懠顒佹濠殿喗銇涢崑鎾搭殽閻愬瓨宕屾鐐村笒閳规垿宕堕埡鍐冾參姊婚崒姘偓鎼佸磹妞嬪海鐭嗗〒姘ｅ亾妤犵偞顨呴…銊╁礋椤掆偓瀵潡姊哄Ч鍥х仼闁硅绻濆畷鎴犫偓锝庡枟閻撴盯鏌涢妷锝呭姎闁绘帗妞介弻鈩冩媴閸撴彃鍓堕梺鍝勮閸斿矂鍩為幋锕€骞㈤柍鍝勫€甸弸鏃堟⒒娴ｈ姤銆冪紒鈧笟鈧弫鍐Ψ瑜庡畷鍙夌箾閹存瑥鐏╂鐐灪缁绘盯宕卞Δ鍐吅婵犮垼顫夊ú鐔奉潖閾忓厜鍋撻崷顓烆€岄柛銈嗙懇閺岋綁顢橀悙娴嬪亾閹间緤缍栭煫鍥ㄦ礈绾惧吋淇婇婵愬殭妞ゅ孩鎹囧铏圭磼濡櫣浠搁梺鎸庣缁绘盯宕煎┑鍫㈠姱濠殿喖锕ュ浠嬪蓟閸涘瓨鍊烽柤鑹版硾椤忣厽绻濋埛鈧崒姘ギ闂佸搫鏈惄顖炲箖閳哄懎绠甸柟鐑樼箓婵￠绱撻崒娆戭槮闁稿﹤鎽滅划鏃堟倻閽樺）锕傛煙閻楀牊绶茬紒鐘冲▕閺岀喖骞戦幇顓犮€愬Δ鐘靛仜缁夌懓顫忕紒妯诲闁芥ê顦幆鐐烘⒑閸濄儱校闁告梹娲熼垾锕傚垂椤曞懏寤洪梺閫炲苯澧撮柕鍡曠窔瀵噣宕奸锝嗘珝闂備胶绮鑽ゆ崲濠靛牐濮冲┑鐘崇閳锋垿姊婚崼鐔烘创闁绘稒绮庣槐鎺撴綇閵婏妇顦伴梺绯曟杺閸庢彃顕ラ崟顓涘亾閿濆啫濡虹紒銊嚙椤啴濡堕崱妤€衼缂備浇灏▔鏇犲垝婵犳碍鍊锋繛锝庡厸缁ㄥ姊洪棃娑氱畾婵℃彃鎳庨埢鎾愁煥閸喓鍘遍梺瀹狀潐閸庤櫕绂嶉悙顑跨箚闁绘劦浜滈埀顒佺墱閺侇噣骞掑Δ鈧壕褰掓煠婵劕鈧鎯岄崱妞绘斀闁绘ɑ褰冮弳鐐烘煕濞嗗繒绠插ǎ鍥э躬閹瑦锛愬┑鍡橆唲濠电偛鐡ㄧ划宥夊磿閹惰棄鐓橀柟杈惧瘜閺佸﹦鐥鐐叉Щ濠殿喗绋撶槐鎾寸瑹閸パ勭彯闂佹悶鍔岄悥濂告偘椤旇法鐤€婵炴垶锚缁愭稑顪冮妶鍡欏缂佸鍨块幃鍨鐎涙ǚ鎷洪梺缁樺姌濡嫰宕濆鑸电厱闁绘梻顭堥婊兦庨崶褝鏀荤紒杞扮矙瀹曘劍绻涢悙顒€顏烘繝鐢靛仩閹活亞寰婃禒瀣疅闁跨喓濮寸粻鐔虹磼鐎ｎ亞姘ㄩ柡鈧懞銉ｄ簻闁哄啫鍊告禍楣冩煛閸♀晛澧撮柡宀€鍠栭、娑橆潩椤掑绱旀俊鐐€戦崹娲儎椤栫偛绠栨繛鍡樻尭閻鏌涚仦涔咁亪宕濆鑸电厪闁糕剝锚缁楁帗銇勯锝囩疄闁轰焦鍔欏畷銊╊敆閳ь剟藟濮橆兘鏀介幒鎶藉磹濡や焦鍙忛柣鎴ｆ绾剧粯绻涢幋娆忕労闁轰礁瀚…鍧楁嚋闂堟稑顫岀紓浣哄У閻楁鎹㈠☉銏犲耿婵☆垵娅ｆ禒鎾⒑閸濆嫷鍎忛柣妤€锕ョ粚杈ㄧ節閸ヨ埖鏅┑鐘绘涧濡骞嗛悙瀵哥閻庢稒顭囬惌瀣煟閻斿弶娅呮い顐㈢箲缁绘繂顫濋鍌︾床婵犵數鍋涘Λ娆戞暜閹烘鍌ㄩ柨娑樺绾捐棄銆掑顒佹悙婵炲懏锕㈤弻娑㈠Ω閵壯冪厽濡ょ姷鍋涢敃銈夊煘閹达箑鐐婇柕澶堝劙缁ㄥジ姊绘担鍛婂暈闁告梹顨婂畷鎴﹀箻鐠囪尙鏌у銈呯箰鐎氬嘲銆掓繝姘厪闁割偅绻冮ˉ鐐烘倶韫囨稒娑ч柍瑙勫灴椤㈡瑩骞嗚楠炲姊?
// 1) 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鍝勭Ф閸斿秵銇勯弬鎸庡枠婵﹦绮幏鍛存惞閻熸壆顐奸梻浣告啞濮婄懓煤閻旂鈧礁顫濇０婵囨櫍闂佺粯锚閸氣偓缂佹顦版穱濠囧Χ韫囨洖鍩岄梺鍝ュ櫏閸ㄥ爼骞冮敓鐘茬妞ゅ繐鎳庨弸鎴濃攽閻樿宸ラ柣妤€妫涚划鍫ュ醇閻旂寮垮┑鈽嗗灠濞硷繝宕搹鍏夊亾?ChatGPT internal API 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈灏忛梺鍛婎殕婵炲﹤顕ｇ拠娴嬫婵☆垶鏀遍弬鈧梻浣告啞濞诧箓宕戦崟顒佸弿闁靛繈鍊栭埛鎴炵箾閼奸鍤欐鐐搭殜閺岀喖鎮烽悧鍫濇灎閻庢鍠栭…鐑藉极閹邦厼绶炲┑鐘插缂嶅倿姊绘笟鈧褏鎹㈤崱娑樼柧婵犻潧顑呯粈澶愭煙闂傚鍔嶉柍閿嬪灴閺屾稑鈹戦崱妤婁紝闂佺楠搁柊锝夊蓟閿濆绠抽柣鎰暩閺嗐倖绻濈喊澶岀？闁轰浇顕ч悾鐑芥偄绾拌鲸鏅┑顔斤供閸撴瑧绮婇鐣岀瘈鐎典即鏀卞姗€鍩€椤掍礁濮嶇€规洘鍨块獮妯兼惥娴ｇ儤鍤€妞ゎ厹鍔戝畷鐔碱敃閵忕姌鎴︽⒑閸撗呭笡闁绘濞€瀵鏁愰崪浣瑰缓闂侀€炲苯澧い顓炴穿椤﹀磭绱掗崒娑樻诞闁轰礁鍟村畷鎺戔槈濮橆剙绠為梻鍌欑閹碱偊宕悩璇茬；闁瑰墽绮悡鐔镐繆閵堝懎鏆欓柍璇茬墦閺岋絽鈽夐崡鐐寸仌缂備胶濮电粙鎴﹀煡婢跺á鐔兼惞閸︻厼鏄ラ梻鍌欐祰椤曆呪偓娑掓櫇缁瑩骞掑Δ浣规珨闂傚倷绶氶埀顒傚仜閼活垱鏅堕幍顔剧＜妞ゆ洖鎳庨悘锛勭磼瀹€鍐摵缂佺粯绻堝畷鍫曟嚋閸偅鐝┑鐘愁問閸犳鏁冮埡鍛婵娉涚壕瑙勩亜閺嶃劌鐒归柡鈧禒瀣厽闁归偊鍨伴惃鍝勵熆瑜嬮崹娲Φ閸曨垼鏁囬柣鎰版涧閳敻姊虹化鏇熸珖闁稿鍊濋悰顔锯偓锝庡枟閺呮粓鏌ら幁鎺戝姕闁绘繍鍋婂缁樻媴閸涘﹤鏆堝┑鐐额嚋缁犳挸鐣烽幋锕€纾奸柣鎰皺妤犲洤顪冮妶鍡樺蔼闁搞劍妞介崺娑㈠箣閻樼數锛滈柣搴秵娴滄粓鎯冮敓鐘崇厸闁逞屽墴閹筹繝濡堕崶鈺嬬床闂佽鍑界紞鍡涘磻閸曨垱鍊堕柟缁㈠枟閻撴洟鏌嶉悷鎵虎闁诲繗灏欓埀顒侇問閸ｎ噣宕滃璺虹畾闁哄啫鐗嗙粻濠氭煙绾板崬骞楃紓宥呯焸濮婂宕掑▎鎴犵崲濠电偘鍖犻崨顔煎簥闂佺硶鍓濈粙鎴︽倿閸偁浜滈柟鍝勬娴滈箖姊洪崫銉ユ瀾閻庣瑳鍛崥?Responses 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锕傛煟閵忕媭鐓兼慨濠勭帛缁楃喖鍩€椤掆偓椤洩顦归挊婵囥亜閹板墎鐣遍柣銈囧亾缁绘盯骞嬮悙璺侯棟濡炪們鍎插畝鎼佸蓟濞戙垺鏅滈悹鍥ㄥ絻缁犳椽姊洪崫銉バｉ柛鏃€鐟╁濠氭晲閸涘倹妫冮崺鈧い鎺戝閸嬪鏌涢埄鍐噮缂佺姵妫冮弻鐔兼倻濡闉嶉梺?// 2) store=false 3) 闂?compact 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤濞硷繝寮婚悢琛″亾閻㈡鐒剧€涙繄绱撴担鐣屽牚闁稿﹥绻堝濠氭晝閳ь剝鐏掓繛鎾村嚬閸ㄩ亶鏁嶉崱妞绘斀闁绘劕寮堕埢鏇灻瑰鍕疄鐎殿喗褰冮…銊╁醇閻斿搫骞楅梻渚€鈧稑宓嗘繛浣冲嫭娅犳い鏇楀亾妤犵偞鐗犲鍫曞箣椤栨繂鎯堟繝娈垮枛閿曘儱顪冮挊澶屾殾闁靛濡囩弧鈧梺绋挎湰椤曟挳寮撮悢铏诡啎闁诲孩绋掗…鍥儗鐎ｎ喗鐓熸俊銈呭暙瀛濋梺?stream=true闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ礂閻撳簶鍋撶紒妯诲弿婵°倐鍋撴俊顐ｇ懇閹箖鎮滈懞銉㈡嫼缂備礁顑呴悘婵嬵敆閵徛颁簻闁靛鍎婚煬顒傗偓娈垮枛椤兘寮幇鏉垮耿婵☆垰鎼俊鎶芥⒒娴ｈ姤鐝柛搴″悑閹便劑濡舵径濠勫摋缂備礁寮堕々绱?闂傚倸鍊搁崐鎼佸磹閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞妞ゆ帒顦伴弲顏堟偡濠婂啰绠婚柛鈹惧亾濡炪倖甯婇懗鍫曞煝閹剧粯鐓涢柛娑卞灠閳诲牓鏌曢崱鏇狀槮闁宠閰ｉ獮姗€宕橀幓鎺撴殢濠碉紕鍋戦崐鏍箰妤ｅ啫纾婚柣鏂垮悑閸嬫﹢鏌曟径鍡樻珕闁抽攱鍨块弻娑樷攽閸℃浼€闂佸疇顕чˇ鐢稿蓟濞戞鐔煎垂椤旂粯鐫忕紓鍌欑贰閸犳稑鐣烽悽绋跨疅闁圭虎鍠栫粈瀣煃鐞涒€充壕缂備降鍔岄…宄邦潖閾忚鍠嗛柛鏇ㄥ墰閸戔剝绻濋埛鈧崘顔煎及闂佽鍣ｇ粻鏍蓟閸℃鍚嬮柛鈩冪懃楠?stream=false
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
	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳婀遍埀顒傛嚀鐎氼參宕崇壕瀣ㄤ汗闁圭儤鍨归崐鐐差渻閵堝棗鍧婇柛瀣尰濞艰鈹戠€ｎ偀鎷洪梻渚囧亞閸嬫盯鎳熼娑欐珷闁圭虎鍠楅悡娑㈡倶閻愭彃鈷旈柕鍡樺浮閺屽秷顧侀柛鎾卞妿缁辩偤宕卞☉妯硷紱闂佸憡渚楅崢楣冨汲閿旈敮鍋撻崗澶婁壕闂佸憡娲﹂崜娑㈠储閹间焦鍊甸柛蹇擃槸娴滈箖姊洪柅鐐茶嫰婢у鈧娲戦崡鎶界嵁濡吋瀚氶柤纰卞墰閺嬧偓闂傚倷绀佸﹢閬嶆惞鎼淬劌绐楅柡宥庡亞閻棝鎮楅敐搴′簴濞存粍绮撻弻鐔煎箥閾忣偅鐝旈梺缁樺笧閸庛倗鎹㈠☉銏犵闁稿繗鍋愰ˇ顓犵磽?OpenAI 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟版晥闂佹寧绻勯崑娑㈠煘閹寸姭鍋撻敐搴′簼婵炲懎娲铏圭矙鐠恒劎鍔规繝纰樷偓铏窛缂侇喗鐟ㄧ粻娑㈠籍閸屾粎妲囬梻渚€娼ф蹇曞緤閸撗勫厹濡わ絽鍟崐鍨箾閹寸偛绗氭繛鍛攻椤ㄣ儵鎮欑拠褍浼愰梺浼欑稻缁诲牆鐣烽悢纰辨晞闁兼亽鍎遍鐑樼節閻㈤潧浠╅柟娲讳簽瀵板﹥绂掔€ｎ亞鐤呴梺璺ㄥ枔婵绮诲鑸电厱妞ゆ劗濮撮崝婊堟煟閹捐泛鏋戦柕鍥у楠炴帡宕卞鎯ь棜闂傚倷绶氬褔鎮ц箛娑掆偓锕傚醇閵夛箑浠奸梻浣哥仢椤戝懘顢氶柆宥嗙厸濠㈣泛顑呴悘銉╂煕閻愵亜濮傞柟顔煎槻椤劑宕熼鐘靛帨闁诲氦顫夊ú婊堝极鐠囧樊鍤曢柟缁㈠枟閸嬪嫮绱撻崼銏犘ラ柣鈺侀叄濮婄粯鎷呴崨濠呯闂佺绨洪崐婵嗙暦瑜版帗鍋ㄩ柛鎾冲级閺呮粓姊洪崘鍙夋儓闁瑰啿绻橀幃锟犳偄閸忚偐鍘甸梻渚囧弿缁犳垶鏅舵ィ鍐╊棅妞ゆ帒顦晶鎾煛瀹€瀣？濞寸媴绠撻幃娆擃敆閸屻倖袨缂傚倸鍊烽懗鍓佸垝椤栨粍宕查柛鈩冪懅娑撳秵绻涢幋娆忕仼闁告濞婇弻锝夊箛椤旇姤姣勯梺鍛娚戞繛濠傤潖婵犳艾纾兼繛鍡樺笒閸樷€愁渻閵堝骸骞栭柣妤佹崌閵嗕線寮介鐐茶€垮┑鐐村灦椤洭顢欐径鎰拺闁硅偐鍋涢崝鈧梺鍛婁緱閸犳顢欓弴銏♀拻濞达絽鎲＄拹锟犳煕鎼存稑鈧繂鐣疯ぐ鎺撳亜闁绘挸娴烽崫妤呮⒑閹稿孩绀€闁稿﹤缍婇幃鈥斥枎閹扳晙绨婚梺鍝勮癁閸曞灚顥ｉ梻浣侯焾閿曘儵骞冮崒婵囷紓婵犳鍠楅…鍫ュ春閺嶎厼鐓曢柟杈鹃檮閻撶姴鈹戦钘夊闁逞屽墯濞茬喎顕ｉ幖浣哥闁挎梻鏅崢浠嬫⒑缂佹ɑ鈷愭繛鍏肩懇瀹曟繈濡舵径瀣帗閻熸粍绮撳畷婊冾潩鐠鸿櫣顦╅悷婊呭鐢帡姊婚鐐寸叆婵犻潧妫Σ鍝ョ磼閻樺磭澧甸柡灞诲姂閹垽宕崟纰樺亾瀹€鍕厸濞达絽鎽滄牎缂備胶绮换鍐崲濠靛纾兼繛鎴炆戦銈夋⒒娴ｇ懓顕滅紒瀣笧閸掓帡骞橀幇浣圭稁闂佹儳绻愬﹢杈╁閸忓吋鍙忔俊銈傚亾闁绘鍔欓崺鈧い鎺嶇椤曟粎绱掔紒妯尖姇闁瑰嘲鎳樺畷姗€宕ｆ径瀣€烽梻浣告啞閸ㄨ鎱ㄩ悽鍨床婵炴垯鍨圭粻锝嗙箾閸℃绠冲ù鐘荤畺濮婅櫣绱掑Ο璇查瀺闂佽崵鍣︾粻鎾翠繆閻㈢绀嬫い鏍ㄨ壘閸炪劑姊洪棃娑辩劸闁告柨鐭傝棢婵犻潧顑嗛埛鎴︽煕濠靛棗顏柨娑欐⒒缁辨帡骞撻幒鎾充淮闂佺硶鏂侀崑鎾愁渻閵堝棗绗掗柛鐘宠壘椤洭骞囬鐘殿啎闂佸憡鐟ラˇ閬嶆儗閹烘柡鍋撶憴鍕；闁告鍟块锝嗙鐎ｅ灚鏅ｅ┑鐘欏嫬鍔ゅù婊勫劤闇夐柨婵嗙墕閳ь兛绮欏顕€宕煎┑鍡欑崺婵＄偑鍊栭幐鐐叏鐎涙ɑ鍙忛柨鏃€鍨濈换鍡涙煟閹板吀绨婚柍褜鍓氶悧婊堝极椤曗偓楠炴帡寮崫鍕濠殿喗菧閸庤鲸鎱ㄩ崒鐐寸厱闁宠鍎虫禍鐐繆閻愵亜鈧牕顔忔繝姘；闁瑰墽绮悡鏇炩攽閻樻彃鈧粯绂掗姀掳浜滈柡鍥朵簽缁嬭崵绱掔紒妯肩畵妞ゎ偅绻堥、鏍煘閸喖顫囬梺鍝勮閸旀垵顕ｉ鈧崺鈧い鎺戝绾惧潡鏌熼崜浣规珪鐎规挷绶氶弻鐔煎箥椤旂⒈鏆梺鎶芥敱鐢帡婀侀梺鎸庣箓濞村倿鎮為悙顑句簻闁靛鍎崇粻濠氭煛鐏炲墽銆掗柍褜鍓ㄧ紞鍡樼濠靛闂ù鐘差儐閻撴瑦顨ラ悙鑼虎闁诲繆鏅滈妵鍕箻閻愯棄浠悗瑙勬磸閸旀垵鐣烽幒妤佹櫆闁告搩鐏濆鍏炬棃鎮╅棃娑楁勃闂佺粯顨嗛〃濠囩嵁韫囨稑宸濋柡澶嬪灦瀹撳秴顪冮妶鍡樺暗闁稿绋撶划濠氬Ψ閵夈垺鏂€闂佹寧绋戠€氼剚绂嶆總鍛婄厱濠电姴鍟版晶鐢碘偓娈垮枛椤攱淇婇幖浣哥厸濞达絽瀚囬崶銊у幍闂傚倸鍊搁顓犳嫻娴煎瓨鐓涘ù锝夘棑閸斿秴菐閸パ嶈含濠碘€崇埣瀹曘劑顢欓崗纰变画闂?tier 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆欑穿闂備線鈧偛鑻晶鍓х磼閻樿櫕灏柣锝夋敱缁虹晫绮欑拠淇卞姂閺屻劑寮崶顬捇鏌ｅ┑鍥╁⒌婵﹦绮幏鍛存偡闁箑娈濇繝鐢靛仜瀵爼鎮ч悩鑼殾闁归偊鍓﹀鎵偓鍏夊亾闁逞屽墰缁粯绂掔€ｎ偀鎷绘繛杈剧悼閻℃棃宕靛▎鎾寸厱闁瑰濮靛▍鏇犵磼鏉堚晛浠辩€规洖宕埥澶娢熺喊杈ㄐゅ┑鐘愁問閸犳鈥﹂崼銉晩闁归偊鍠氱粈濠囨煛瀹ュ骸骞楅柣鎾寸洴閺屾盯濡烽敐鍛闂佽娴氭禍顏堝蓟閿濆绠婚悹铏瑰劋閻忓牓姊洪崫鍕拱婵炲弶顭囬幑銏犫槈閵忕姴鐎銈嗘⒒閸樠囷綖閹剧粯鈷掑ù锝呮啞閸熺偞銇勯鐐村窛闁轰緡鍠氱划濠囧闯閹唸y/flex/auto/default/scale闂?	// 闂?Codex 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟撮崣鍕煏閸℃鏆ｅ┑锛勫厴閸┾剝鎷呮搴ｅ€為梻鍌欑窔濞佳囨晬韫囨稑纾兼繝濠傛噺閸犳帡姊绘担绛嬪殭濡ょ姴鎽滅划璇差吋婢跺﹦锛熼梻渚囧墮閸楁洟宕堕澶嬫櫖闂佺粯鍔栬ぐ鍐倵椤撱垺鈷戠紒瀣濠€鎵磼鐎ｎ偅灏电紒顔碱煼瀹曟粏顦柛瀣崌閹兘寮跺▎鐐棏闂備礁鎽滄慨闈浢哄鍫熷殟閺夊牄鍔庣弧鈧┑顔斤供閸橀箖宕㈤幖浣光拺闁告稑锕ョ壕鐢告煛閸涱喚娲寸€规洘绻傞…銊╁川椤栨粣绱茬紓鍌氬€烽悞锕傗€﹂崶顬¤櫣鈧稒锕╁▓浠嬫煟閹邦剚鈻曢柛搴㈡閺岀喖顢欓悾灞惧櫘闂侀€炲苯澧存繛浣冲洤绠熼柨鐔哄閺佸洤鈹戦悩宕囶暡闁抽攱甯掗湁闁挎繂鎳忛崯鐐烘煕閻斿搫浠﹂柕鍥у婵℃悂濡烽敂缁橈骏闂備礁鎽滄慨鎾晝閵堝鏁囬柛蹇曞帶缁剁偛鈹戦悜鍡樼窙缂佺姵鎹囧璇测槈閵忕姴宓嗛梺闈涱焾閸庤櫕绂掗埡鍛拺闁告稑锕ゆ慨鈧梺绋款儐閹哥粯绌辨繝鍥ㄥ€锋い蹇撳閸嬫捇寮介鐐殿槷闂佹寧娲嶉崑鎾淬亜椤撶偞鍋ラ柛鈹惧亾濡炪倖甯婇懗鍓佺不閸撗€鍋撻崗澶婁壕婵犵數濮抽懗鍫曞极閺嶎偆纾藉ù锝勭矙閸濇椽鏌熺粙娆剧吋妤犵偛绻樺畷銊р偓娑櫭禒鎯ь渻閵堝棛澹勭紒韫矙瀵偅绺介崨濞炬嫽婵炶揪绲介幉锟犲箚閸儲鐓欓柛鎰皺缁犳娊鏌熼獮鍨伈鐎规洖銈告俊鐤槻缂佷緤闄勭换婵嬪閿濆懐鍘梺鍛婃⒐濞叉粎鍒掓繝姘仭闁瑰啿锕ょ紞濠囧极閹版澘宸濇い鏃囨閺嬫垵鈹戞幊閸婃鎱ㄩ弶鎳ㄦ椽顢橀悜鍡樼稁濠电偛妯婃禍婊呯矆閸儲鐓犵痪鏉垮船婢ь噣鎮介妯哄姦婵﹥妞藉畷顐﹀Ψ閵夋劧缍侀弻锝夊煛婵犲倻浠╅梺浼欑到閸㈣尙鍙呭銈呯箰閹冲酣鍩€椤掑倸鍘撮柡宀嬬秮楠炲鎮欓崘鈺佸摵妤犵偛绻橀幖褰掑捶椤撶媴绱茬紓鍌氬€烽梽宥夊垂瑜版帞宓侀柡宥庡幗閻撴瑩鏌涢幇顓炵祷妞ゆ帇鍨荤槐鎺楀磼濮樻瘷銉╂煏閸ャ劌濮嶆鐐村浮楠炴鎹勯崫鍕杽闂傚倸鍊搁…顒勫磻閸曨個娲晝閳ь剝鐏嬮梺纭呮彧闂勫嫰宕甸埀顒勬煟閻樺厖鑸柛鏂款樀瀹曟垿骞橀弬銉︾亖闂佸壊鐓堥崰妤呮偟椤曗偓濮婅櫣绮欏▎鎯у壈闁诲孩鐭崡鎶藉Υ娴ｅ壊娼ㄩ柍褜鍓欓悾宄拔旈崨顔间簵闁硅壈鎻徊鍧楁偘椤斿皷鏀介柣妯虹仛閺嗏晛鈹戦鑺ュ唉鐎规洦鍨堕、鏇㈠Χ閸モ晝鍔搁梻浣稿閸嬪懎煤濮椻偓閸╂盯骞掗幊銊ョ秺閺佹劙宕ㄩ鍏兼畼闂備浇顕栭崹浼存偋閹捐绠栨俊銈傚亾妞ゎ偅绻堥幃鈩冩償閵忣澀澹曠紓鍌氬€风欢锟犲窗閺嶎収鏁勯柛銉墮閻掑灚銇勯幒鎴濇灓婵炲吋鍔欓弻?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锕傛煟閵忋垻甯涘ǎ鍥э躬閹瑩顢旈崟銊ヤ壕闁哄稁鍘介崑瀣繆閵堝懎鏆熼柣顓熺懇閺岀喖鎮欓鍌涜弴闂侀€炲苯澧柟铏悾鐑芥晲閸℃绐為悗鍏夊亾闁逞屽墴閹線宕奸悢铏圭槇闂佹眹鍨藉褍鐡梻浣侯焾閿曘倗绱炴繝鍛紓闂備浇顫夊畷妯肩矓?priority 闂?flex闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灪閵囧嫰濡搁妷锕€娈楁繝纰樺墲閹告娊鐛幒妤€绠ｆい鎾跺枎閸忓﹪姊绘担鐟邦嚋缂佽鍊块獮濠傤吋閸℃瑧褰鹃梺鍝勬储閸ㄦ椽鍩?codex-rs/core/src/client.rs闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊洪悙钘夊姎缁剧虎鍘奸敃銏ゅ箻椤旂晫鍘遍梺鍝勫暊閸嬫捇鏌涢妸銉т虎闁伙絽鍢茬叅妞ゅ繐瀚崝锕€顪冮妶鍡楃瑐闂傚嫬绉电粋宥咁煥閸喓鍘甸梺缁樺灦閿氶柣蹇嬪劦閺屽秷顧侀柛鎾寸箓閻ｇ兘宕归銈囧骄?	// 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐鍛傜喎鈻庨幆褎顔勭紓鍌欒兌婵挳鎮樺璺何﹂柛鏇ㄥ枤閻も偓闂佸湱鍋撻幆灞轿涢悙鐢电＝濞达絿鏅崼顏堟煕婵犲啰绠炵€殿喖顭烽弫鍐焵椤掑啰浜介梻浣虹帛閼归箖顢氶鐘电焼闁糕剝銇涢弨浠嬫煟閹邦厽缍戦柣蹇ョ畵閺屻劑寮村Δ浣圭彋闂佽鍣换婵囦繆閻戣棄鐓涢柛灞剧矊楠炲牊淇婇悙顏勨偓鏍礉瑜忓濠勬崉閵娧傜瑝濠碘槅鍨遍惇瑙勭濠婂牊鏅┑鐘宠壘缁€鍐煥濠靛棙顥為柡澶嬫倐濮婄粯鎷呴崫銉︾€┑鈽嗗亜鐎氼剝鐏嬪┑掳鍊曢幊蹇涘磹閸洘鐓熸俊顖濆亹鐢盯鏌ｉ幘瀛樼闁逛究鍔岄～婊堝幢濡も偓閳锋帡姊虹粙鍨劉闁告梹鐟╁璇测槈閵忕姷顔婇梺瑙勬儗閸ㄩ亶寮崜褏纾藉ù锝堝Г閵嗗啰绱掗鐣屾噧闁伙絿鍏樻俊鎼佹晜閸撗呮闂備礁鎲″ú锕傚磻閸曨厾绠旀慨妯垮煐閳锋帡鏌涚仦鍓ф噮妞わ讣绠撻弻鐔烘嫚瑜忕壕鍧楁煙楠炲灝鐏茬€规洘甯掗埞鍐箚瑜屾竟鏇炩攽椤旂瓔鐒介柛妯犲嫭娅犻悗鐢电《閸嬫挸鈻撻崹顔界亶婵犵數鍋涢敃顏堢嵁閸愩劉鏋庨柟鐐綑娴犳椽姊哄Ч鍥х伄妞わ綆鍠氬Σ鎰版嚑椤掑倻锛濋梺绋挎湰閼归箖鍩€椤掑倸鍘存慨濠呮椤撳吋寰勬繝鍌溾偓顓㈡偡濠婂懎顣奸悽顖涱殜瀹曟劙宕奸弴鐔哄弳闂佸搫鍟悧濠囧汲濠婂牊鐓?OpenAI SDK 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵挳鎮欏ù瀣壕鐟滅増甯掔壕鍧楁煙鐎电校闁哥姵鍔欓弻锝呂旈埀顒勬偋閸℃瑧绠旈柟鐑樻⒒绾惧ジ鏌嶈閸撴艾顕ラ崟顓濇勃缂佸銇樻竟鏇㈡⒑缁嬭法绠版い锔诲灡缁傚秵銈ｉ崘鈹炬嫼闂備緡鍋嗛崑娑㈡嚐椤栨稒娅犻柟缁㈠枟閻撴洟鏌嶇憴鍕姢濞存粎鍋撴穱濠囨倷椤忓嫧鍋撻弽顐ｆ殰濠电姴瀚惌鍡椼€掑锝呬壕閻庤娲滈弫濠氥€佸Δ鍛妞ゆ帒鍊搁獮鍫ユ⒒娴ｇ懓顕滅紒璇插€婚幑銏ゅ箳濡も偓缁€鍡涙煟濡も偓閻楀嫭绂嶅鍫熺厸闁搞儲婀圭花鐣岀磼婢舵ê鏋熼柕鍥у婵偓闁斥晛鍟伴ˇ浼存⒑鏉炴壆顦﹂柛鐔告尦瀹曟椽鍩€椤掍降浜滈柟鍝勬娴滄儳鈹戦悙鏉戠祷闁诲繑绻堥崺鐐哄箣閿旇棄浜瑰銈嗘閸嬫劖瀵奸崶褉鏀介柣鎰▕濡插綊鏌ｉ埡濠傜仸鐎殿噮鍋婂畷姗€顢欓懞銉︾彇闂備胶顭堢悮顐﹀磹閺嶎厼鐤鹃柨婵嗘噳閺€浠嬫煟閹邦剙绾ч柍缁樻礀闇夋繝濠傚缁犵偤鏌涢埡鍌滄创妤犵偛顑夐幃鈺呮偨绾板闂繝鐢靛仩閹活亞寰婃禒瀣剁稏闁哄稁鍙庨弫鍌炴煕椤愩倕鏋庨柍褜鍓欓悥鐓庮嚕閸洖閱囨慨姗嗗幗閻濇洟姊虹粙娆惧剱闁绘顨堥幑銏犫槈閵忕姴鑰垮┑鈽嗗灥椤曆囨瀹ュ鈷戠紓浣股戠亸鐗堢箾閼碱剙鏋涙鐐插暙铻栭柛娑卞枟濞呮粓鏌熼懖鈺勊夐柍褜鍓濈亸娆撳礄瑜版帗鐓熼幖娣€ゅ鎰箾閸欏澧遍柍褜鍓氶崙褰掑窗濮橆儵锝夊箛閺夎法顔婂┑掳鍊撶粈浣圭瑜版帗鍊甸悷娆忓缁€鈧┑鐐跺皺婵敻骞堥妸鈺佺倞妞ゆ帊璁查幏娲煟閻樺弶绀岄柍褜鍓欑壕顓熺椤撶偐鏀介柍钘夋娴滄粓鏌涢悢鍛婄稇妞ゆ洩缍侀、鏇㈡晝閳ь剛绮诲☉銏＄厸濠㈣泛瀛╃涵鍫曟煕婵犲嫬浠遍柟顔煎槻椤劑宕橀悙顑芥瀰闁?auto/default/scale 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鎯у⒔缁垳鎹㈠☉銏犵婵炲棗绻掗崝鎾⒑鏉炴壆顦︽い鎴濇婵＄敻宕熼姘鳖啋闁荤姾娅ｉ崕銈夋倵妤ｅ啯鈷戦柛娑橈功閹冲啰绱掔紒妯哄婵犫偓娓氣偓濮婅櫣绮欑捄銊ь唶闂佸憡鑹鹃鍥╂閻愬搫绠ｉ柣妯虹仛閿涘繘姊虹拠鈥崇€婚柛鏇ㄥ亗濞ｎ噣姊绘担鍛婃儓闁活剙銈稿畷鐗堟償閵娿儳鍘洪梺瑙勫礃椤曆囧垂閸屾稏浜滈柡鍐ㄥ€瑰▍鏇灻瑰鍫㈢暫婵﹦绮幏鍛喆閸曗晙鎴烽梻浣告啞椤牆螞閸曨垱鍋╂繝闈涱儏缁犵懓霉閿濆棛鎽冮柟鑺ユ礀閳规垿鎮欓弶鎴犱户闂佺硶鏅涚€氭澘顕ｉ锕€绀冩い鏃傛櫕閸欏棗鈹戦悩缁樻锭婵☆偅鐟╄棢闁绘鍋ㄦ禍婊堟煥閺傛寧鎯堥柛鏂诲€楃槐鎺撴綇閵婏箑闉嶉梺鐟板槻閹虫ê鐣烽锕€绀嬬痪鐗埫禍鍓р偓鍏夊亾闁告洦鍓涢崢閬嶆⒑閹稿海绠撶紒缁樺笧缁棃顢欓柨顖氫壕婵炲牆鐏濋弸娑欍亜椤撶姴鍘存鐐插暣婵偓闁靛牆妫楁禍妤呮煙閼圭増褰х紒鎻掓健瀵櫕瀵肩€涙鍘介梺缁樻煥閹芥粓鎯屾繝鍕＜濠㈣泛鏈崵鈧銈嗘穿缂嶄線鐛惔銊﹀殟闁靛／鍐ㄐ﹂梺璇查缁犲秹宕曢崡鐐嶆稑鈽夐姀鐘插亶?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幈濡炪値鍘介崹鍨濠靛鐓曟繛鍡楃箳缁犲鏌″畝鈧崰鎾舵閹烘顫呴柣妯虹－瑜邦垶姊绘担鑺ャ€冪紒鈧担琛″亾濮橆偄宓嗙€殿喛顕ч埥澶娢熼柨瀣垫綌婵犵數鍋涘Λ娆撳礉濡ゅ啰鐭欓柛銉墯閳锋垶鎱ㄩ悷鐗堟悙闁逞屽厵閸婃繂鐣烽幎鑺ユ櫜濠㈣泛锕ㄩ幗鏇㈡⒑缂佹ɑ鈷掗柛妯犲懐涓嶆慨妯垮煐閻撴稑顭跨捄鐚存敾婵炲懎娲弻鐔煎礈瑜忕敮娑㈡煟閹惧瓨绀嬫鐐寸墱閸掓帞鎲撮崟鍨秷闂備浇銆€閸嬫挸鈹戦悩宕囶暡闁绘挸绻橀弻娑㈠Ψ閹存繂鏋ゅù鐓庡暣閹鈻撻崹顔界亞缂備緡鍠楅悷鈺呭Υ娓氣偓瀵挳锝為鍓р棨婵＄偑鍊栭幐鐐叏闂堟稓鏆﹂柣鏂垮悑閳?	// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵潙螖閳ь剚绂嶉幆褜娓婚柕鍫濈凹缁ㄥ鏌涢悢鍛婂唉闁绘侗鍣ｉ獮鍥敇閻斿嘲濡抽梻浣瑰缁诲倻鈧稈鏅犻弻銊╁Χ婢跺鎷绘繛杈剧秬濞咃絿鏁☉銏＄厱闁哄啠鍋撴繛鑼枛閻涱噣寮介褎鏅濋梺闈涚墕濞层劑鐛幘鏂ユ斀閹烘娊宕愰弴銏犵疇闊洦绋掗崑鍌涚箾閸℃ɑ灏伴柍閿嬪灦閵囧嫰骞橀崡鐐差瀷闂佷紮绲藉畷顒勨€﹂懗顖ｆ▌缂備胶濮甸悧鐘差嚕婵犳艾围濠㈣泛顑呮禍閬嶆⒑閸撴彃浜濈紒璇茬Ч瀹曠數鈧綆鍠楅埛鎴︽⒒閸喓鈯曢柟鍏煎姍閹顫濋銏犵ギ闂佺粯渚楅崰鏍綖濠婂牆鐒垫い鎺嗗亾闁伙絿鍏橀弫鎾绘偐閸愭祴鍋撻悜鑺ョ厱闊洦鎸婚幉鍛婁繆椤愵偄鐏︽慨濠傤煼瀹曟帒鈻庨幒鎴濆腐婵＄偑鍊ら崢濂告偋韫囨稑鐒垫い鎺嶈兌閳绘捇鏌￠崨顖氣枅濠碘剝鎸冲畷姗€鍩￠崘顏嗘闂備焦鎮堕崕婊堝幢濡鏆梻鍌氬€风粈渚€骞栭鈷氭椽濡搁埡浣虹崶闁硅壈鎻徊楣冾敃娴犲鐓熼柟閭﹀枛閸斿鏌ｉ幘瀛樼闁靛洤瀚伴獮鍥礈娴ｇ懓浠圭紓鍌欒兌婵敻骞愰幖浣哥厴闁硅揪瀵岄弫濠勭棯閹峰矂鍝洪柍璇茬墕閳规垿顢欑涵鐑界反濠电偛鎷戠徊鍧楀礆閹烘垟鏋庨煫鍥э攻閻庡姊虹拠鈥崇€婚柛鎰电厛濡﹪姊婚崒姘偓椋庣矆娴ｉ潻鑰块梺顒€绉寸粻鐘绘煙閹规劗袦婵炲樊浜滃洿婵犮垼鍩栫粙鎾剁矙韫囨稒顥婃い鎰╁灪婢跺嫮绱掔€ｎ偄娴€规洜鏁婚獮鎺懳旀担鍝勫妇闂備礁澹婇崑鍛崲閸岀偛鐒垫い鎺嗗亾闁硅绱曠划瀣吋閸℃劕浜濋梺鍛婂姀閺備線骞忔繝姘拺闁告挻褰冩禍鐐烘煕閻樿櫕宕岄柛鈺侊躬瀵挳濮€閿涘嫬骞嶉梻浣告啞閻熴儵藝娴兼潙纾归柟鎵閻撴稓鈧厜鍋撻悗锝庡墰琚︽俊銈囧Х閸嬬偤鏁冮姀銈囧祦闁哄稁鍙庨弫鍥煟閺冨倸鍔嬫繝銏″浮濮婂宕掑▎鎴М闂佸湱鈷堥崑鍛嚗閸曨垰鐐婃い鎺嗗亾缂佲偓閸喓绡€闂傚牊绋掗敍宥夋煕閵堝棭娈滈柡灞剧洴瀵挳濡搁妷褌鍝楁俊銈囧Х閸嬫垿宕归悜妯尖攳濠电姴娴傞弫宥嗙節闂堟稒顥滈柟顖滃仱閹嘲顭ㄩ崟顒傚嚒濠碘槅鍋呯换鍌烇綖韫囨洜纾兼俊顖濐嚙椤庢捇姊洪崨濠勨槈闁挎洏鍎靛畷鏇㈠箻缂佹鍘介梺缁樺姈濞兼瑩宕甸鍕厱閻庯綆浜峰銉╂煟閿濆懎妲婚柍瑙勫灴瀹曞崬鈻庨幇顓犳殾闂傚倷绶氶埀顒傚仜閼活垱鏅剁€电硶鍋?nil闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ亞浠㈡い鎺嬪灲閺岋紕浠︾拠鎻掝潎闂佺硶鏅濋崑銈夌嵁鐎ｎ喗鏅濋柍褜鍓涙竟鏇㈠礂閼测晝鐦堟繝鐢靛Т閸婂綊骞戦敐澶嬬厱闁哄啠鍋撻柟顔煎€搁～?normalizeResponsesBodyServiceTier 婵?body 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈瀛濈紓鍌氱Т閿曨亜顕ｇ拠宸悑濠㈣泛锕ｇ槐鍫曟⒑閸涘﹥澶勯柛鎾寸懃閳诲秹鏁愭径瀣ф嫼缂備礁顑堥崕濠氾綖閿曞倹鐓曢柡鍌濇硶閻掓悂鏌涢埡鍐ㄤ槐妤犵偛顑夐弫鍐焵椤掑倸顥氶柛蹇氬亹缁♀偓婵犵數濮撮崐褰掑箲閿濆鐓曢柡鍐ｅ亾闁诡喖鍊搁～蹇撁洪鍕啇闂佺粯鍔栬ぐ鍐€栭崼銉︹拺缂佸灏呭銉︺亜椤撶姴鍘撮柣娑卞枟瀵板嫰骞囬鍌ゅ晣濠电偠鎻徊鍧楀箠閹惧顩查梺顒€绉甸埛鎴︽煕閹炬潙绲诲ù婊勭箘缁辨帞鎷犻幓鎺濅純闂佽鍠氶崗妯侯嚕婵犳艾唯闁挎棁顫夌€氬ジ姊绘担绛嬫綈闁稿骸鍚嬮幈銊╁Χ婢跺﹥鐎柡澶婄墑閸斿鎹㈤崱妯镐簻闁规壋鏅涢悘鈺呮煛閸☆厾鎮奸柍褜鍓氶鏍窗閺嶎厸鈧箑鐣￠柇锕€娈ㄩ梺鍦檸閸犳牠锝為崨瀛樼厽婵☆垱顑欓崵娆撴煛娴ｇ瓔娼愮紒缁樼箞閹粙妫冨☉鎺撶€版繝鐢靛仒閸栫娀宕楅悙顒傗槈闁宠姘︾粻娑欑節閸愵亙绨村┑鐘垫暩婵挳鏁冮妶澶嬪亱濠电姴娲ら悡?
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

// openAIFastPolicyCtxKey 闂?context 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈瀛濈紓鍌氱Т閿曨亜顕ｇ拠宸悑濠㈣泛锕ｇ槐鍫曟⒑閸涘﹥澶勯柛鎾寸懃閳诲秹鏁愭径瀣ф嫼濠电偠灏褔鐛Δ浣典簻闁靛鍎婚煬顒傗偓娈垮枦椤曆囧煡婢舵劕顫呴柣妯兼暩閺夋悂姊婚崒姘偓鎼佹偋婵犲嫮鐭欓柟鐑橆殕閸婅泛銆掑锝呬壕濠殿喖锕ュ浠嬪蓟閸涘瓨鍊烽柤鑹版硾椤忣厽绻濋埛鈧仦鑺ョ彎闂佸搫鏈粙鎺旀崲濠靛纾兼繛鎴灻煎ǎ顕€鏌ｆ惔銈庢綈婵炲弶锚椤啯绂掔€ｎ亝鐎梺鐟板⒔缁垶宕戦幇鐗堢厵缂備焦锚缁椦囨煟濞戞帗娅嗗ǎ鍥э躬閹瑩顢旈崟銊ヤ壕闁哄稁鍘肩粻浼存煟閹伴潧澧柛娆忕箻閺岋綁骞嬮敐鍡╂闂佺琚崝宥夊Φ閸曨垰绠抽柟瀛樼箥娴犻箖姊洪幎鑺ユ暠閻㈩垱甯″﹢渚€姊洪幐搴ｇ畵闁瑰啿绻樺畷顖炴倷閻戞鍘靛銈嗙墬濮樸劍鏅堕姀銏㈢＜閺夊牄鍔岀粭褔鏌嶈閸撱劎寰婂ú顏勭柧婵犻潧鐗忛悳濠氭煛閸愩劌鈧敻宕戦幘鑸靛枂闁告洦鍓涢ˇ銊╂⒑閹肩偛濡奸柣蹇旂箞閹箖鎮滈挊澹┾晝鎲告径鎰；闁圭偓鏋煎Σ鍫熺箾閸滀礁寮惧ù婊勭箘缁?OpenAIFastPolicySettings 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鎮㈤崗灏栨嫽闁诲酣娼ф竟濠偽ｉ鍓х＜闁绘劦鍓欓崝銈嗙節閳ь剟鏌嗗鍛枀闂佸綊妫块悞锕傚磻鐎ｎ喗鐓曟い鎰剁悼缁犳﹢鏌ｉ悢鏉戝缂佽鲸鎸婚幏鍛村传閸曟埊绻濋弻娑樜旀担绯曟灆閻庢鍠栭…鐑藉箖閵忋倕绀傞悘蹇旂墬鐎氫粙姊虹拠鍙夋崳闁轰焦鎮傞垾锕傚醇閻斿墎绠氭繛瀵稿Т椤戝棝鍩?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚敐澶婄闁挎繂鎲涢幘缁樼厱濠电姴鍊归崑銉╂煛鐏炶濮傜€殿喗鎸抽幃娆撶叓椤撶姵鏅奸梻鍌欐祰濡椼劎绮堟担琛″亾濮樼厧骞橀柟骞垮灩閳藉濮€閻樿尪鈧灝鈹戦埥鍡楃仴妞ゆ泦鍛瀳鐎广儱顦伴埛鎺楁煕鐏炴崘澹樺ù婊呭仱閺屾盯濡搁埡鈧幉鎯р攽閿涘嫭鏆€规洜鍠栭、娑橆潩椤掆偓閺併倝姊绘笟鈧褑鍣归梺鍛婁緱閸ㄦ壆鏁幒鎴旀斀闁挎稑瀚禍濂告煕婵炲灝鈧繂鐣烽幋锕€宸濇い鎺斿帶閹碱偉鐏掗梺绋跨箳閸樠勬償婵犲倵鏀介柣妯肩帛濞懷勩亜閹存繃顥㈤挊鐔兼煕椤愩倕鏋旂紒鐘荤畺閺岀喓鈧數顭堟禒褔鏌熼崘鎻掓殶缂佽鲸甯￠獮鍥敇閻橆偅顫嶆俊鐐€ゆ禍婊堝疮鐎涙ü绻嗛柛顐ｆ礀绾惧吋绻涢幋鐐垫噮鐟滄澘娲缁樻媴娓氼垳鍔哥紓浣虹帛閸ㄥ潡寮崘顔碱潊闁绘瑢鍋撻柛銈嗘礋閹綊宕堕妸褋鍋炲┑鈩冨絻閻楀﹥绌辨繝鍥舵晬闁绘劕鐡ㄩ弳鐘差渻閵堝骸浜滅紒澶屾嚀椤繐煤椤忓嫬绐涙繝鐢靛Т鐎涒晠鎮炬總鍛娾拺缂佸顑欓崕鎴︽煕閻樺啿濮夌紒顔碱儔楠炴帒螖婵犲啯娅嶉梻渚€娼х换鍡楊瀶瑜旈獮蹇撁洪鍛嫼闂佸憡绋戦敃锔剧不閹剧粯鍊垫慨妯煎帶瀵噣鎸婇悢鍝ョ瘈濠电姴鍊归崳鐣岀棯閹规劕浜圭紒杈ㄦ尰閹峰懏鎱ㄩ幋顓濈凹闁逛究鍔岄濂稿幢濞嗘垹妲囬梻?WebSocket 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鏁愭径濠庢綂闂侀潧绻嗛弲婵嬪礉閸濄儳纾奸柣鎰靛墮閸斻倖銇勯鐘插幋鐎殿喖顭烽幃銏ゅ礂閻撳簶鍋撻柨瀣ㄤ簻闁瑰搫妫楁禍鐐節閳封偓鐏炵晫浠搁梺鍝勭焿缂嶄礁顕ｉ鍕閹兼惌鍠楅敍鍡涙⒒娴ｇ瓔鍤冮柛鐘崇墱缁辩偞绻濋崶褏顔愰悷婊呭鐢宕戦敓鐘崇厸濠㈣泛锕︽禒銏°亜椤掆偓濡繂顫忕紒妯诲缂佹稑顑嗙紞鍫ユ偡濠婂嫭绶查柛鐔告尦閻涱喖螖閳ь剟鈥﹂妸鈺佸窛妞ゆ洖鎳忕紞妤呮⒒娴ｅ憡璐￠柛搴涘€濋妴鍐川椤栨粈绗夊┑鐐村灦閻燂絾绂嶅鍫熺厵闁绘劦鍓﹀▓鏃堟倵濮樼厧娅嶇€殿喗鍎奸妵鎰板箳閹绢垱瀚藉┑鐐舵彧缁插潡鎮洪弮鍫濆惞闁告劦鍠楅悡鏇㈡煟濡櫣锛嶅褑浜槐鎺撴綇閳轰椒娌紓浣虹帛缁诲牆鐣烽幒鎴叆闁告洦鍓﹂崯搴ㄦ⒒閸屾瑧顦﹀鐟帮躬瀹曟垿宕ㄩ鍏兼そ瀵粙濡搁妷銉ヮ潑闂傚倷娴囧畷鐢稿窗閹邦喖鍨濈€光偓閳ь剟骞戦姀鐙€娼ㄩ柍褜鍓熼悰顕€宕橀埞鍨簼闂佸憡鍔忛弲娑欑濞差亝鈷戦柛娑橈功閳藉鎳濆畝鍕厵闂佸灝顑嗛妵婵囨叏婵犲啯銇濈€规洏鍔嶇换婵嬪礋椤撶姷鐛ラ梻鍌欒兌椤牆顫濋敂濮愪粓闁告縿鍎抽惌澶愭煙閻戞ê鐒鹃柣鎾卞劦閺屾盯顢曢妶鍛亖闂佸憡眉缁瑩寮婚悢鍏煎€绘俊顖濄€€閺嬫瑩鎮楃憴鍕缂佽鍊块幃鎯р攽鐎ｎ亞顦伴梺鍓茬厛閸嬪懎鈻嶉弮鍫熲拻濞撴埃鍋撴繛浣冲泚鍥敃閿曗偓閻ょ偓绻涢幋鐐寸殤闁活厼妫濋幃妤呮晲鎼粹€茬按婵炲瓨绮嶇划鎾诲蓟閻旂顕遍悗娑櫳戦柨顓熺箾鐎涙鐭嬬紒璇茬墦瀵鈽夊鍡楁倯婵犮垼娉涢鍡涙煥椤撶儐娓婚柕鍫濈凹缁ㄥ鏌涢悢椋庢憼濞ｅ洤锕畷濂稿即閻愯尪鈧灝鈹戞幊閸婃洟宕导鏉戠疅妞ゅ繐妫涚壕浠嬫煕鐏炲墽鎳嗘い鏂款樀閺屾稓鈧綆鍋呯亸顓㈡煏閸℃ê绗掓い顐ｇ箞閺佹劙宕ㄩ鈧ˉ姘舵⒒閸屾瑧鍔嶉悗绗涘懏宕查柛鏇ㄥ灠绾惧鏌曢崼婵愭Ш鐎规挷鐒︽穱濠囶敍濮樿鲸鐧侀梺绋款儐閹瑰洤螞閸愩劉妲堟繛鍡樺姇閺嬫垿姊绘担椋庝覆缂佺姵鍨块幃褔骞橀幇浣圭稁濠电偛妯婃禍婵嬎夐崼鐔虹闁瑰瓨鐟ラ悘顔锯偓瑙勬处娴滅偟妲愰幘瀛樺闁荤喐澹嗙粊閿嬬節閳封偓閸℃﹩妫勯梺瀹犳椤︾敻骞冮悾宀€鐭欓悹渚厛濡插爼鏌ｉ悢鍝ョ煀缂佺粯甯楃粩鐔煎即閵忊€充簵闁瑰吋鎯岄崰姘跺船閻㈠憡鐓熼柣妯煎劋椤忕娀鎮樿箛瀣鐎规洘绻堥幃銏ゅ礂閼测晛骞堥梻浣稿暱閹碱偊顢栭崶顒€绠栧┑鍌氭啞閻撳啰鎲稿鍫濈婵﹢顤傞弫濠囨煛瀹ュ骸骞栭梻鍌ゅ灡閵囧嫰寮村Δ鈧禍楣冩倵鐟欏嫭绀冮柛銊ュ閹广垹鈹戠€ｎ亞顦ㄩ悷婊冪箳缁顫濋懜纰樻嫼闂佺绻愰崥瀣磹閹邦厾绠惧ù锝呭暱閹冲繘顢曟禒瀣厽闁归偊鍘鹃妶鎾煛閳ь剚绂掔€ｎ偆鍘藉┑鈽嗗灠閹碱偆鐥閺屾盯鎮㈤崜鍙夌杹闂佸搫鐭夌换婵嗙暦閸洖唯闁靛／鍌滄／闂傚倷鑳剁划顖炲箰妤ｅ啫绐楅柡宥庡幗閸嬪倹銇勯幇鍓佺暠闁绘劕锕弻鏇熺節韫囨搩娲銈忚礋閸庤尙鎹㈠☉姘ｅ亾閻㈡鐒剧悮銉╂⒑閸濆嫭鍣藉┑鐐╁亾閻庢鍠涢褔鍩ユ径濠庢僵闁挎繂鎳嶆竟鏇㈡⒑閹稿海绠撳Δ鐘叉啞缁傚秴顭ㄩ崘锝嗘杸濡炪倖妫侀崑鎰墡闂備線娼уú銈団偓姘卞閹便劑鍩€椤掑嫭鐓冮柍杞扮閺嬨倖淇婇銏犳殻闁诡喗顨堥幉鎾礋椤掑倵鏁嶇紓鍌欒兌缁垳鎹㈤崼銉у祦闁告劑鍓悢鐓庣闁绘挸娴疯ぐ鎾⒒娴ｈ棄浜归柍宄扮墦瀹曟粌鈽夊▎妯活唲闂傚倸鍊搁崐宄懊归崶顒夋晪闁哄稁鍘肩粣妤呮煙閻戞﹩娈旈柣銈夌畺閺岋絽螣閸喚姣㈤梺鍝勬４婵″洭骞夐幖浣瑰亱闁割偅绻€閸掑﹪姊烘潪鎵槮闁哥喐鎸冲濠氬Ω閵夈垺鏂€闂佺硶鍓濊摫闁诡喗鐟ラ埞鎴︽倷閺夋垶鐦戦梺鎼炲劘閸斿骞忔繝姘拺缂佸瀵у﹢浼存煟閻旀繂娉氶崶顒佹櫆濠殿喗鍔掔花濠氭⒑鐟欏嫬绀冮悘蹇旂懇閹苯鈻庨幋鏂夸壕婵炲牆鐏濋弸娑㈡煟閺嶎偄甯堕柣锝囨焿閵囨劙骞掑┑鍥ㄦ珦濠电姰鍨洪崕濂革綖婢跺⊕娲偄婵傚缍庣紓鍌欑劍钃卞┑顖氼嚟缁辨帒鈽夊鍡楀壈濠电偛鐗婇崝娆忣潖濞差亝顥堥柍鍝勫暟鑲栫紓鍌欐祰鐏忣亜鈻旈弴銏犵闁圭儤鎸剧弧鈧┑顔斤供閸撴瑩宕妸鈺傗拺闂傚牊渚楅悞楣冩煕婵犲喚娈樼紒顔款嚙椤繈鎳滅喊妯诲缂傚倸鍊烽悞锕傛晪闂佽绻楁ご鍛婄┍婵犲浂鏁冮柕蹇曞娴犲ジ姊烘潪鎵窗闁革綇绲介～蹇曠磼濡顎撻梺鑽ゅ枛閸嬪﹪宕电€ｎ亖鏀介柣鎰綑缁茶崵绱掓径灞惧殌闁伙絿鍏橀獮瀣晝閳ь剛绮绘繝姘€甸梻鍫熺⊕閹插憡銇勯弮鈧ú鐔奉潖濞差亜鎹舵い鎾跺仒婢规洖鈹戦悙鎻掓倯闁荤噦绠撻獮鍫ュΩ閵夘喖鎮戦梺绯曞墲閿氱€殿喗瀵х换婵嬫偨闂堟刀銏ゆ煙閸愯尙绠绘い銏℃閹晠鎮介悽纰夌床闂佸搫顦悧鍕礉瀹€鈧划顓㈠箳濡や焦鍤夐梺鎸庣箓椤︿即宕戦敐澶嬬厱闁靛鍠曠花濠氭煟閵娧冨妺濞ｅ洤锕、娑橆潩椤戣棄浜鹃柛褎顨呴拑鐔兼煥濠靛棙顥為柛搴ｅ枛閺屻劌鈽夊Ο渚紑闂佸搫妫欓惄顖氼潖閾忓湱鐭欓柛褎顨忛埀顒侇殘閳ь剝顫夊ú锕傚礈濞戙垹鐭楅柛娑樼摠閳锋垿姊婚崼鐔峰礋闁割偁鍎遍弸渚€鏌ㄩ悢鍝勑㈤柣鎰攻閵囧嫰骞掗幋顓熜ч梺鍝勬媼娴滎亜顫忓ú顏呯劵闁绘劘灏€氭澘顭胯閸犳濡甸崟顖ｆ晝闁靛繈鍨婚濠勭磽娴ｄ粙鍝洪悽顖涘笩閻忔帡姊洪崗鑲┿偞闁哄懏绮撳畷闈涚暆閸曨兘鎷绘繛杈剧秬椤宕戦悩缁樼厱閹兼番鍨婚惌鎺斺偓瑙勬礃閿曘垽骞冨▎鎿冩晞闁圭楠搁弨顓㈡⒒閸屾艾鈧悂宕愰幖浣哥９鐎瑰嫭鍣磋ぐ鎺戠倞妞ゆ帒顦伴弲顏堟偡濠婂啰绠绘鐐村灴婵偓闁靛牆鎳愰娲⒑缂佹◤顏堝疮閹稿孩鍎熷┑鐘叉处閳锋垿鏌涢幘鏉戠祷濞存粍绻勭槐鎺旀嫚閹绘帩浼冮悗娈垮枟婵炲﹪寮崘顔肩＜婵炴垶鑹剧敮楣冩⒒婵犲骸浜滄繛灞傚€濋、鏍川閺夋垹鍔﹀銈嗗笂閻掞妇绮堥埀顒€鈹戦悙鏉戠祷婵炲皷鈧剚娼栫紓浣股戞刊鎾煕濞戞﹫宸ラ柡鍡楃墢缁辨挻鎷呴崜鎻掑壉闂佹悶鍔屽锟犳偘椤旂晫绡€闁告侗鍨抽弶鎼佹⒑閸濆嫭宸濋柛鐘冲姇閳绘挻瀵肩€涙ǚ鎷虹紓鍌欑劍钃辨い銉ユ缁绘盯宕崘顏喩戠紓浣割儏椤︻垶顢樻總绋垮耿婵炲棙鍩堥崯搴ㄦ⒒娴ｇ儤鍤€闁宦板姂閹兘濡烽埡鍌氣偓鍫曟煃閸濆嫬鏆熺痪鎯с偢閺岋絽螣閻戞ǚ鏋欓梺缁樻尭閵堟悂寮婚敍鍕勃缂佸鐏濋埛澶愭倵閸偅绶查悗姘煎櫍閸┾偓妞ゆ帒锕︾粔闈浢瑰鍛槐鐎规洘绻勭划娆愭償閹惧瓨鏉?DB 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎悗骞垮劚椤︻垳绮堢€ｎ偁浜滈柟鎹愭硾鍟搁梺鍛婏供閸ㄨ泛顫忕紒妯诲闁告稑锕ㄧ涵鈧梻浣侯焾缁ㄦ椽宕愬┑瀣ラ柛鎰靛枛瀹告繈鏌℃径瀣仴闁诲寒鍙冨铏圭矙閹稿孩鎷辩紓浣割儐閹告儳顕ｈ閸┾偓妞ゆ帒瀚埛鎺楁煕鐏炵偓鐨戝褎绋戦妴鎺戭潩椤撗勭杹閻庤娲樺ú鐔肩嵁閸ヮ剚鍋嬮柛顐犲灩楠炲牓姊绘笟鈧褔鎮ч崱娑樼疇闁归偊鍘藉▍鐘绘煥閺囩偛鈧綊鎮″☉姘ｅ亾閸忓浜鹃梺閫炲苯澧寸€规洑鍗抽獮鍥敆婢跺苯濮洪梻濠庡亜濞诧妇绮欓幋鐘电焼闁割偁鍨洪崰鎰扮叓閸ャ劎鈽夋慨瑙勭叀閺岋絽螣濞嗘儳娈紓?//
// Trade-off闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ礂閻撳簶鍋撶紒妯圭箚妞ゆ牗绻冮鐘绘煕閺冨偆鐒鹃柍瑙勫灴閹瑩鎳犻浣烘闂備礁鎼幊鎰箾閳ь剛鈧鍠栭…宄邦嚕閹绢喗鍋勯柧蹇氼嚃閸熷酣姊绘担铏瑰笡闁告棑绠撳畷婊冾潩閼搁潧浠ч梺鍝勬储閸ㄦ椽鍩涢幋鐘电＜閻庯綆鍋掗崕銉╂煕鎼达紕效闁哄本绋掔换婵嬪礃椤忓棛鏉介柣搴㈩問閸犳牠鈥﹀畡閭﹀殨闁圭虎鍠楅崑鍕渻鐎ｎ亝鎹ｉ柣娑卞櫍濮婄粯鎷呴搹骞库偓濠囨煕閹惧绠氶柟绛嬪亰濮婅櫣绱掑Ο鑲╃暫濠电偠灏欓崰搴綖韫囨稒鎯為柛锔诲幘閻撴挸鈹戦悙鍙夘棡闁稿孩濞婂畷婊堫敆閸曨兘鎷洪梺鍛婄箓鐎氼參宕抽挊澶嗘斀闁绘劏鏅涙禍楣冩煟鎼淬埄鍟忛柛鐘崇墵閳ワ箑鐣￠柇锕€娈ㄩ梺鍦檸閸犳宕戦崟顐富閻庯綆浜濋幑锝夋煟濞戝崬娅嶆慨濠勭帛閹峰懐绮欏▎鐐棏闂備胶顭堥鍛村箠閹版澘绠查柕蹇嬪€曠壕鍏肩箾閹寸偟鎳冪€殿喖娼￠弻鐔兼嚌閻楀牆娑х紓浣藉蔼濞夋洘绔熼弴鐔侯浄閻庯綆鍋嗛崢閬嶆煟韫囨洖浠ч柛瀣崌閹啴骞嬮悩杈╁墾闂婎偄娲﹀ú婊呭閽樺褰掓晲閸噥妫勯梺缁樻尰閸旀鍩€椤掆偓閸樻粓宕戦幘鍓佺＜閻庯綆鍋掗崕銉╂煕閵堝棙绀€闁宠鍨块幃鈺佺暦閸ヨ埖娈归梻浣告惈閹冲繘骞冮崒鐐茶摕闁挎繂鎲橀悢鐓庡瀭妞ゆ梻鍋撻妤呮⒒娴ｄ警鐒鹃柨鏇畵瀹曟垿濡堕崪浣圭稁濠电偛妯婃禍婊呯矆閸愵喗鐓曟繛鍡楁禋濡插綊鏌涙惔銏″磳婵﹥妞藉畷顐﹀Ψ閵夛妇鈧椽姊洪崨濠庢畷濠电偛锕悰顕€宕卞Ο鑲╂嚌闂侀€炲苯澧柣锝夋敱缁虹晫绮欏▎鐐秱闂備線娼уù姘熆娴ｇ儤顫曢柛鎰ゴ閺€浠嬫煟閹邦垰鐓愮憸鎶婂懐纾界€广儰绀佹禍鐐繆閻愵亜鈧牕煤濮椻偓閹囧即閵忕姷鍘洪梺瑙勫礃椤曆呯尵瀹ュ鐓冪憸婊堝礈閻旈晲绻嗛柣銏㈩焾缁€瀣亜閺嶃劎鈯曟繛鍛濮婃椽鎳為妷鍐句邯钘濋柦妯猴級濞戙垹骞㈡繛鎴炵憿閹锋椽姊绘笟鍥т簽闁稿鐩幊鐔碱敍濞戞瑦鐝锋繛瀵稿帶閻°劑鎮″☉妯忓綊鏁愰崼顐ｇ秷闂佺锕ラ崝妤呭焵椤掑喚娼愭繛鎻掔箻瀹曡绺介弶鍡楁穿閵囨劙骞掗幘璺哄汲婵犵數濞€濞佳囨偋閸℃顩锋繝濠傜墛閻撳啴鎮峰▎蹇擃仼闁诲繆鏅犻弻宥堫檨闁告挶鍔庡濠冪鐎ｎ偄鈧埖鎱ㄥΟ鎸庣【闁汇倝绠栭弻鏇＄疀鐎ｎ亖鍋撻弽顓炵厱闁硅揪闄勯埛鎴炪亜閹扳晛鈧洘绂掑鍫熺厾婵炶尪顕ч悘锟犳煛閸涱厾鍩ｆ鐐达耿椤㈡瑩鎮剧仦钘夊闂傚倷鑳剁涵鍫曞礈濠靛鈧啴宕ㄩ弶鎴炶緢濠电偛妫欓幐濠氭偂閸愵喗鐓㈡俊顖欒濡牊淇婇妤€浜炬繝鐢靛У椤旀牠宕伴幒妤€纾婚柟鍓х帛閳锋垿鏌熺粙鎸庢崳闁宠棄顦甸幃妤€顫濋悡搴♀拫闂佺粯渚楅崳锝呯暦婵傚憡鍋勯柧蹇撴贡濡插洭姊绘担鍦菇闁搞劏妫勯…鍥樄闁糕斂鍨藉畷濂告偄閾忚鍟庡┑鐐舵彧缁蹭粙宕崸妤€瑙︽俊顖濄€€濡插牊淇婇鐐存暠妞ゎ偄绉撮埞鎴﹀煡閸℃浠╅梺鍛婅壘椤戝鐛繝鍥х妞ゅ繐妫涢敍婵嬫⒑缁嬫寧婀伴柤褰掔畺閸┾偓妞ゆ帊鐒﹂崐鎰偓瑙勬礃閸旀瑩鐛弽銊﹀闁告縿鍎荤槐顕€姊绘担鍛婂暈缂佸搫娼″畷鏇㈠箮閼恒儱鍓归梺鐟板⒔缁垶鎮″▎鎴犵＝濞达綁娼ч悘鈺傜箾閸涱噯鑰块柡灞稿墲閹峰懐绮欏▎鍙ユ偅婵＄偑鍊ら崣鈧繛澶嬫礋楠炴垿宕熼娑樹粡闂佸搫顦伴崹鐢革綖閻樼粯鈷?WS session闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏ゅ川婵犲嫮肖濠德板€х徊浠嬪疮椤栫儐鏁佺€广儱顦伴埛鎴犵磼鐎ｎ偒鍎ラ柛搴＄箲閵囧嫰骞嬪┑鎰枅閻庢鍠涢褔鍩ユ径鎰潊闁绘﹢娼ф慨鍫曟⒒娴ｅ憡鍟為柛鏃€鍨垮畷婵嗩吋婢跺﹦鐣惧┑鐐村灦閻熝呯不妤ｅ啯鍊垫繛鎴炵懐閻掍粙鏌ｉ鐑囨敾缂佺粯绻堥崺鈧い鎺戝閻掕偐鈧箍鍎遍幊鎰版偪閸涘瓨鈷戠憸鐗堝笚閿涚喓绱掗埀顒佹媴閾忛€涚瑝闂佹寧绻傞ˇ浼存偂閵夆晜鐓涢柛銉厛濞堟柨霉濠婂懎浜惧ǎ鍥э工铻栭柍褜鍓熼獮濠偽熸笟顖涚稁濠电偛妯婃禍婊堝箲閼哥偣浜滈柟鎹愭硾閸撻亶鏌￠崱鈺佸箹闁宠鍨块幃鈺咁敊閼测晙鐥俊鐐€ら崢鐓幟洪妶澶婄叀濠㈣埖鍔栭崑銊х磼鐎ｎ厽纭堕柛鏃撶畱椤啴濡堕崱妤冪懆闁诲孩鍑归崣鍐春濞戙垹绠ｉ柣妯兼暩閿涙繃绻涙潏鍓у埌妞ゎ偅鐗犻、鏃堝川椤旂厧浼庨梻浣规偠閸庢椽宕滃鑸靛亗闁绘梻鍘х粻鎶芥煙閹増顥夌紒鈧径鎰厵閻庢稒顭囩粻姗€鏌￠崱顓犵暤闁哄矉绻濆畷鍫曞Ψ閵壯傛偅闂佸摜鍎愰崹璺侯潖濞差亜绀冮柛娆忣槹閸庢捇姊虹粙鍖″姛闁轰礁顭烽悰?session闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏焊娴ｅ湱鈧姊婚崟顐ｅ枠妞ゃ垺淇洪ˇ鏌ユ偂閵堝棎浜滈柟鍨暞婵炲洭鏌嶈閸忔稓绮堟笟鈧敐鐐差煥閸繄鍔﹀銈嗗笂閻掞箓宕ｈ箛娑欑厓鐟滄粓宕滈悢鐓庤摕闁挎繂鎷嬪銊╂煃瑜滈崜娆撯€﹂崶顏嶆Ъ闂侀€涚┒閸旀垿寮崒鐐茬闁圭儤姊婚悾楣冩⒒娴ｈ櫣甯涢柛鏃撻檮缁傚秴顭ㄩ崼婵堝姦濡炪倖甯婇懗鑸垫櫠椤忓懌浜滈柡鍌濇硶缁犺尙绱掔紒妯肩畵妞ゎ偅绻堥、妤呭磼閿旀儳鑰块梻鍌氬€烽懗鍫曞箠閹惧瓨娅犻柣锝呰嫰閸ㄦ繃銇勯弽顐沪闁稿鍊块弻锟犲炊閳轰椒姹楅梺琛″亾濞寸姴顑嗛悡鐔镐繆椤栨繍鍤欑紒鑼帛閵囧嫯绠涢敐鍕嚬缂備胶绮惄顖氱暦閵娾晩鏁婇柣锝呯焾濡插綊姊绘担鍛婃儓闁活剙銈稿畷浼村冀椤撶喎浠掑銈嗘磵閸嬫挾鈧娲栫紞濠囥€佸▎鎾崇煑闁靛／鍜佹П闂傚倷娴囬褏鈧稈鏅犻、娆撳冀椤撶偟鐛ラ梺鍝勬川婵挳宕瑰┑鍥╃闁糕剝蓱鐏忣亪鏌嶈閸忔稓绮堟笟鈧敐鐐差煥閸繄鍔﹀銈嗗笒鐎氼剛绮婚弽銊х闁糕剝蓱鐏忣厾绱掗悩鍨毈闁哄瞼鍠栭幃婊兾熺拠鑼暡闂?// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝曢梻浣藉Г閿氭い锔诲枤缁辨棃寮撮姀鈾€鎷绘繛杈剧秬濞咃綁濡存繝鍥ㄧ厱闁规儳顕粻鐐烘煙閽樺鈧鍩€椤掑﹦绉甸柛鐘崇墵閹瑦绻濋崶銊у帾婵犵數鍋涢悘婵嬪礉閹绢喗鐓曢柕鍫濆暙閳ь剚绻傞～蹇撁洪鍕炊闂佸憡娲栭悘姘櫏闂傚倷鐒﹂幃鍫曞礉瀹€鈧槐鐐寸節閸涱噮妫ㄥ┑锛勫亼閸婃牠骞愰崼鏇炲瀭婵炲樊浜滈悡鏇㈡煙鐎电啸缁炬崘妫勯湁闁挎繂娲﹂鐔枫€掑顓犫姇缂佺粯鐩畷濂告偄閸撳弶顥堥梻浣告惈閻寰婇崐鐔轰簷闂備線鈧偛鑻晶浼存煙楠炲灝鐏╂い顐ｇ矒閸┾偓妞ゆ巻鍋撻柣锝呭槻鐓ゆい蹇撳濞煎﹪姊洪棃娑氬闁硅櫕鍔欓幆鍫ュ礋椤栨稈鎷洪梺鍛婃尰瑜板啯绂嶅┑鍫㈢＜閻犲洦褰冮顏呫亜閵婏絽鍔︾€规洖鐖奸、妤佹媴閸欏顏圭紓鍌氬€搁崐鐑芥倿閿曞倹鏅┑鐘愁問閸犳牠鎮ф繝鍕床婵炴垯鍨圭粈鍌炴煠濞村娅嗛柛锝勫嵆閺岋箑螣閻撳孩鐏堥梺璇″枟椤ㄥ﹪寮幇顓熷劅闁炽儴灏欓幐澶娾攽?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕閵堝懎顏柡灞剧洴椤㈡洟鏁愰崱娆樻К缂備胶鍋撻崕鎶解€﹂悜钘夎摕闁哄洨鍠撶粻楣冩煟閹伴潧澧柣婵囨⒒缁辨帡鎮欓鈧崝銈嗙箾绾绡€鐎殿喖顭烽弫宥夊礋椤忓懎濯伴梺鑽ゅТ濞诧箒銇愰崘鈺傚弿閹兼番鍔嶉埛鎴︽煢濡警妲搁柡鍡欏枛閺屾盯鎮╁畷鍥р拰閻庢鍠栭…閿嬩繆閹间礁鐓涢柛灞剧煯缁ㄥ姊绘担鍛婂暈缂佽鍊婚埀顒佽壘閹虫ɑ鎱ㄩ埀顒勬煏閸繃顥犻柛?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤瀚ˉ鍫⑩偓娈垮枤缁垳鎹㈠┑瀣倞闁靛闄勯悵鎶芥⒒娴ｅ憡鍟炵紒顔肩墦閹虫宕奸弴鐐殿槷闂佺粯鍨兼慨銈夊磻閳╁啰绡€濠电姴鍊搁弳鐔虹磼閻樼儤鐝ǎ鍥э躬瀹曪絾寰勬繝鍌ゆ綒闂備浇顕栭崰妤呮偡瑜旈崺鈧い鎺戝濞懷囨煙閼恒儳鐭掗柟顖欑窔瀹曞ジ濡烽敂瑙勫闂備胶顢婇崑鎰板磻濞戙垹绀夐柍褜鍓熷娲川婵犲啫顦╅梺鍝ュУ瀹€绋跨暦閿濆绠ｉ柨鏃囆掗幏缁樼箾鏉堝墽鍒伴柟璇х節楠炲棝宕奸悢缈犵盎闂佹寧妫侀褔鎮橀敃鍌涚厵妞ゆ柣鍔屽ú銈囩矆鐎ｎ偁浜滈柡宥冨妽閻ㄦ垶銇勯敂璇叉珝闁诡喗顨婂畷妤佸緞婵犱礁顥氶梻鍌欑閹测€趁洪弽顓熷€舵慨妯挎硾閻ょ偓銇勮箛鎾跺闁绘挻绋戦…璺ㄦ崉閻氭潙濮涙繝鈷€鍕伌闁哄本鐩顒傛崉閵婃剬鍛亾鐟欏嫭绌跨紓宥佸亾缂備胶濮甸惄顖氼嚕椤掑嫬绀堢憸蹇涙偩閻愵兛绻嗛柣鎰典簻閳ь剚鐗犻幃褍螖閸愨晛搴婇梺鍛婂姦閸ｎ噣寮崒鐐寸厽婵☆垵鍋愮敮娑㈡煟閹惧崬鍔滅紒缁樼洴楠炲鎮滈崱娆忓Ш闂佸搫绉崜婵堟崲濞戙垺鏅查柛娑卞枟閹瑩姊洪幐搴㈠濞存粠浜俊瀛樻媴缁洘顫嶅┑顔筋殔濡瑩宕撻悽鍛娾拺闁告稑顭▓姗€鏌涚€ｎ偄濮嶇€规洩绲界叅妞ゅ繐鎳夐幏缁樼箾鏉堝墽鍒伴柟璇х節楠炲棝宕煎┑鎰數闁荤姾娅ｉ崕銈夊窗濡皷鍋撶憴鍕缂侇喖鐭傞崺鐐哄箣閿曗偓楠炪垽鏌嶆潪鎵妽妞ゆ柨鍊垮缁樻媴閸涘﹤鏆堝┑鐐额嚋缁犳挸鐣风憴鍕嚤閻庢稒顭囬崢鎴︽⒑閹肩偛鍔橀柛鏂挎捣瀵囧焵椤掑嫭鈷掗柛灞炬皑婢ф稓绱掔€ｎ偄娴挊鐔哥節婵犲倹鍣洪柛瀣墬閹便劌顫滈崱妤€骞嬮梺琛″亾濞寸姴顑嗛悡銉︾箾閹寸伝顏堫敂閳轰急鐟邦煥閸愶箑浠梺鍝勬湰缁嬫帞鎹㈠┑瀣妞ゅ繐瀚ч崥鍌炴⒑缂佹ɑ灏柛濠冾殜閸┾偓妞ゆ巻鍋撶紒鐘茬Ч瀹曟洟鏌嗗畵銉ユ喘椤㈡盯鎮欓弶鎴滅棯濠德板€х徊浠嬪疮椤栫偞鍋傞柣鏃堟櫜缁诲棙銇勯弽銊ょ繁闁稿簺鍎甸弻娑㈠Χ閸℃瑦鍣伴梺?缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚敐澶婄闁挎繂鎲涢幘缁樼厱濠电姴鍊归崑銉╂煛鐏炶濮傜€殿噮鍓熷畷褰掝敊鐟欏嫬鐦遍梻鍌欑劍濡炲潡宕㈡禒瀣仭闁冲搫鎳庨拑鐔兼煟閺傚灝鎮戦柛濠勭帛閹便劌螖閳ь剙螞濞嗘挻鍋╅柣鎴ｅГ閳锋垹绱掔€ｎ偒鍎ラ柛搴＄箲閵囧嫰骞嬪┑鍥舵￥闂佺懓绠嶉崹褰掑煘閹达箑鐐婇柍鍝勫暟濡插洭姊绘担渚劸闁哄牜鍓涢崚鎺戠暆閸曗斁鍋撻崒鐐蹭紶闁告洏鍔嶉弬鈧俊鐐€栧褰掝敄濞嗗浚鐒介柟鎵閻撶喖鏌熼崹顔碱伀缂佲檧鍋撻柣搴㈩問閸犳牠鈥﹂悜钘夌畺闁靛繈鍊曠粈鍌炴煠濞村娅呭┑顔芥そ濮婄粯鎷呴悷閭﹀殝濠电偞褰冪粔鐟扮暦閹达箑绠婚柟纰卞幗椤旀棃姊虹紒妯哄闁规椿浜幊鎾诲锤濡や胶鍘搁柣蹇曞仧閺咁偉鍊寸紓鍌欐祰妞村摜鏁幒鏇犱航婵犵數鍋犵亸娆戝垝椤栫倛澶愬醇閵夛腹鎷绘繛杈剧到閹诧繝骞嗛崼銉︾厾婵炶尪顕ч悘锝囩磼椤旇姤顥堥柟顔界矒閺屟囨嚋椤掆偓婵＄晫绱掑Δ鍐ㄦ灈闁糕斁鍋撳銈嗗笒鐎氼剟鎷戦悢鍝ョ闁瑰瓨鐟ラ悘鈺冪磼閻樺樊鐓奸柟顔肩秺閹煎綊鎮烽弶鍨瀱闂備浇顕х换鎰版偋閸℃氨浜欓梻浣虹帛閿曘垹顭囪缁傛帒顭ㄩ崼鐔哄弳闂佸搫娲﹂敋闁逞屽墯缁诲牆鐣峰ú顏勎ㄩ柨鏇楀亾缂佸墎鍋涢埞鎴︽偐閹绘帩鍔夐梺绋跨Ф閺佽顫忓ú顏勭閹艰揪绲块悾鐢告⒑閻熸澘鏆辩紒澶屾暩缁晠鎮㈤悡搴″祮闂佺粯鍔忛弲婊堝棘閳ь剟姊绘担铏瑰笡闁告梹鐗曢…鍥р枎閹炬潙鈧爼鏌ㄩ弬娆炬綗濞存粍绮撻悡顐﹀炊妞嬪骸鍩岄梺鍝勵儐缁嬫帡濡甸崟顖ｆ晝闁挎繂娲ㄩ悾濂告⒑閸︻厽鍤€閻庢矮鍗抽獮鏍亹閹烘挸鍓梺鍛婄缚閸庨亶鎳?濠?缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚敐澶婄闁挎繂鎲涢幘缁樼厱濠电姴鍊归崑銉╂煛鐏炶濮傜€殿喗鎸抽幃娆徝圭€ｎ亙澹曢梺褰掓？閻掞妇鈧艾鎳橀弻锝夊棘閸喗鍊梺缁樻尭缁绘劙鍩為幋锔藉€烽梻鍫熺◥婢规洖鈹戦悙鍙夊櫤闁圭懓娲濠氬焺閸愨晛顎撶紓浣割儐椤戞瑦瀵奸崘顔解拺闁告繂瀚﹢鎵磼鐎ｎ偄鐏撮柛鈹垮劜瀵板嫭绻涢悙顒傗偓濠氭⒑瑜版帒浜伴柛鐘冲哺閸┿垼绠涘☉娆屾嫽婵炶揪绲介幉锟犲箚閸儲鍊垫慨妯稿劚婵倻鈧娲樺ú鐔煎箖閻ｅ瞼鐭欓柤鎰佸灡閹蹭即姊绘担鍛婃儓婵炴潙瀚Σ鎰板即閵忊€充痪闂侀€炲苯澧撮柡宀嬬稻閹棃顢涘鍛咃絾绻涚€涙鐭嬮柣妤冨Т閻ｇ兘骞嬮敃鈧粻濠氭偣閸ャ劌绲婚柛鐐妼椤啴濡堕崱妯烘殫闂佺顑囬崰鏍极瀹ュ應妲堥柕蹇婃閹锋椽姊婚崒姘卞闁告娲熷畷濂稿Ψ閵壯勭叄婵犵數濮撮敃銈夋偋婵犲洤纾婚柛宀€鍋為悡鐔兼煛閸屾氨浠㈤柟顔藉灴閺岋綁骞樼€靛憡鍒涢梺鍝勬湰缁嬫挻绂掗敃鍌氱鐟滃危閸繍娓婚柕鍫濇缁€鍐磼椤旇姤宕岀€殿喛顕ч埥澶愬閳ュ厖绨婚梻浣告啞缁诲倻鈧凹浜滈埢浠嬵敂閸℃瑧锛?闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閻橀潧骞堟繝娈垮枟閿曗晠宕㈡禒瀣︽繝闈涱儐閻撴稑霉閿濆浂鐒鹃柡鍡悼閳ь剝顫夊ú妯兼暜閹烘缍栨繝闈涙祩濞兼瑩鏌″畵顔煎濮ｅ棝姊婚崒姘偓鎼佸磹瀹勬噴褰掑炊椤掑﹦绋忔繝銏ｅ煐閸旀洜澹曢崸妤佺厽闁规儳鍟块鍌涚箾閹存瑥鐏╃紒鈧€ｎ偁浜滈柡宥冨妿閳绘捇鏌熼悿顖ｅ殭妞ゎ亜鍟存俊鍫曞幢濞嗗繆鎷￠梻浣侯焾椤戝洭宕戦妶澶婄畺闁跨喓濮撮崡鎶芥煟閺冨洦顏犻柣锕€鐗撳濠氬磼濮樺崬顤€缂備礁顑嗛崹鍧椼€侀弮鍫濈厸闁告侗鍠氶崢閬嶆⒑閸濆嫬鏆婇柛瀣尵缁辨帡顢欓懞銉ョ３閻庢鍠涢褔鍩ユ径鎰潊闁炽儱鍘栫花濠氭⒒閸屾瑧顦﹂柣蹇旂箞椤㈡牠宕卞☉妯硷紵闂佺懓澧界划顖炲煕閹达附鈷掗柛顐ゅ枔閳洘銇勯妷褍浠遍柟顔荤矙椤㈡稑鈽夊Ο鏄忕檨闂備焦瀵х换鍕磻濞戙垹鐓橀柟瀵稿Л閸嬫捇鏁愰崨顖欑驳闂佸搫鎳岄崹铏规崲濠靛鍋ㄩ梻鍫熷垁閵忋倖鐓曞┑鐘插暞閸婃劙鏌ㄥ┑鍫濅槐闁诡喒鏅犻幃浠嬫偨绾板闂梻鍌欒兌椤牓寮甸鍕仭闁靛ň鏅╅弫濠傤熆閼搁潧濮堥柣鎾存礋閹鏁愭惔鈥茬凹闁诲繐娴氶崢浠嬪Φ閸曨垼鏁囬柍鈺佸暟閵嗗﹪姊洪崷顓熷殌閻庢碍婢橀悾鐑芥晲閸涱垱娈濋梺瑙勵問閸犳牠銆傚ú顏呯厱?Claude
// BetaPolicy 闂?gin.Context 缂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒鐎靛壊妲紒鐐劤缂嶅﹪寮婚悢鍏尖拻閻庨潧澹婂Σ顔剧磼閻愵剙鍔ょ紓宥咃躬瀵鎮㈤崗灏栨嫽闁诲酣娼ф竟濠偽ｉ鍓х＜闁绘劦鍓欓崝銈嗙節閳ь剟鏌嗗鍛枀闂佸綊妫块悞锕傚磻鐎ｎ喗鐓曟い鎰剁悼缁犳﹢鏌ｉ悢鏉戝缂佽鲸鎸婚幏鍛村传閸曟埊绻濋弻娑樜旀担绯曟灆閻庢鍠栭…鐑藉箖閵忋倕绀傞悘蹇旂墬鐎氫粙姊虹拠鍙夋崳闁轰焦鎮傞垾锕傚醇閻斿墎绠氭繛瀵稿Т椤戝棝鍩涢幋鐘电＜閻庯綆鍋掗崕銉╂煕鎼达紕效闁哄本绋掔换婵嬪礃椤忓棛鏉介柣搴㈩問閸犳牠鈥﹀畡閭﹀殨闁圭虎鍠楅崑鍕煣韫囨挻璐℃俊鎻掓喘濮婄粯绻濇惔鈥茬盎濠电偠顕滅粻鎾荤嵁閺嶎厼鐓涘ù鑹伴哺濡炰粙銆佸鈧慨鈧柍閿亾闁归攱妞介幃妤冩喆閸曨剛顦ュ┑鐐茬湴閸斿酣寮灏栨婵炲棙鍨归鏇㈡⒑閼测斁鎷￠柛鎾寸懇瀵鈽夊锝呬壕婵炲牆鐏濋弸娑欍亜椤撶姴鍘寸€殿噮鍋婇獮鍥敇閻愮數鐛┑鐘垫暩婵挳宕板顑╂盯鎮㈤崗灏栨嫽婵炶揪绲介幉锛勬嫻閳╁啨浜滄い鎾跺仧婢ф稒銇勯妸锝呭姦妤犵偞鐗楅幏鍛村川婵犲懐鍙戦梻鍌欑婢瑰﹪宕戞笟鈧畷褰掓嚒閵堝拋妫滈梺绋跨箻濡法鎹㈤崱娑欑厪闁割偅绻傞顐ｃ亜閿旇偐鐣甸柡宀嬬秬缁犳盯骞橀懜鍨枛闂備線鈧稓鈹掗柛鏂跨Ф閹广垹鈹戠€ｎ亞顦ㄩ梺宕囨嚀閵囨﹢鎼规惔銊︾厽閹兼番鍊ゅ鎰箾閹绘帞绠荤€规洘绻傞濂稿炊閵娿儱绨ユ繝鐢靛█濞佳囶敄閸涱収鍟呮繝闈涙储娴滄粓鏌熼悙顒夋當閻庢艾鍢茶灋闁靛ň鏅滈埛鎺楁煕鐏炴崘澹橀柍褜鍓熼ˉ鎾跺垝閸喓鐟归柍褜鍓熼悰顕€寮介褎鏅濋梺鎸庢濡嫭绂嶉崡鐐╂斀闁绘顕滃銉╂煟濡も偓濡稓鍒掗崼銉ラ唶闁靛濡囬崢浠嬫⒑閸濆嫬鏆欓柛濠傛憸閺侇噣宕滄担铏癸紲闂佺粯锚绾绢厽鏅堕柆宥嗙厵闁告瑥顦伴崐鎰偓娈垮櫘閸ｏ絽鐣烽悡搴樻斀濠电姴鍟崯娲⒒閸屾瑨鍏岀痪顓炵埣瀹曟粌鈹戠€ｃ劉鍋撻崘顓犵杸婵炴垶顭堣闂佽鍑界紞鍡涘窗濡ゅ懏鍋傞柡鍥╁枔缁犻箖鏌熺€涙绠撻柤绋跨秺閺岋綀绠涢妷褏袦闂佸搫鏈粙鎴︼綖濠婂牆骞㈡俊銈傚亾闁冲嘲顑呴埞鎴︻敊绾嘲濮涚紓渚囧櫘閸ㄥ爼鐛箛娑樺窛闁哄鍨电粣娑欑節閻㈤潧孝闁哥噥鍋婂畷婊堟偨閻㈢數锛濋梺绋挎湰閻熝囁囬敂濮愪簻闁瑰瓨绻傞顐ｇ箾閸℃劕鐏╂い顐ｇ箞椤㈡宕掑☉娆樺晭濠电姷鏁告慨鎾晝閵堝鐤柣妯款嚙閸戠娀骞栧ǎ顒€濡介柣鎾寸懇閺岋綁骞囬棃娑橆潽缂傚倸绉甸崹鍧楀蓟瀹ュ牜妾ㄩ梺鍛婃尰閻熲晠骞冮妷锔鹃檮缂佸鍎婚幗鏇炩攽閻愭潙鐏熼柛銊ョ秺瀹曪繝骞庨懞銉у帾婵犵數鍋涢悘婵嬪礉閵堝悿褰掓偑閳ь剟宕ｉ崘顭戞綎闁惧繐婀辩壕鍏间繆椤栨繍鍤欑痪鏉跨Ф缁辨挻鎷呴搹鐟扮闂佹寧娲忛崹褰掝敋閿濆洦瀚氭繛鏉戭儐椤秹姊洪棃娑氱濠殿喚鍏樺畷婵嬪箻椤旇В鎷洪梺鍛婄☉閿曘倖鎱ㄦ径鎰厱閻庯綆鍋呭畷宀勬煙椤曗偓缁犳牕鐣锋總绋垮嵆闁绘柨寮剁€氬ジ姊婚崒娆戣窗闁稿妫濆畷鎴濃槈閵忊€虫濡炪倖鐗楃粙鎺戔枍閻樼偨浜滈柡鍥殔娴滈箖姊洪崫鍕効缂傚秮鍋撶紓浣哄У閻╊垰顕ｉ鈧畷鎺戭煥閸涱喗绶板┑鐘垫暩婵兘銆傛禒瀣婵犻潧顑呯粻浼存煙閸撲焦娅曠€规挷绶氶弻鐔兼倻濡儤顔曢梺鍝勫暙閻楀棝鎮為崹顐犱簻闁圭儤鍨甸埀顒€顭烽幆宀勫幢濞戞瑧鍘卞銈嗗姧缁茶法鎷归埡鍐╁枑闁绘鐗嗙粭姘舵煛閸涱喚鍙€闁哄本鐩俊鐑芥晲閸涱収鐎虫繝鐢靛仜閻楀棝鎯岄崒姘兼綎?hot-reload 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻閻愮儤鍋嬮柣妯荤湽閳ь兛绶氬鎾閳╁啯鐝栭梻渚€鈧偛鑻晶鎾煙椤斿吋鍋ユい銏＄懄閹便劑骞囬鍡欐晨闂傚倷娴囬～澶婎熆濡粯娅犻幖杈剧悼閻瑧绱撴担璇＄劷缂佺娀绠栭弻娑樷槈閸楃偟浠╅梺瀹狀嚙閻楀﹪銆冮妷鈺傚€烽悗鐢殿焾閳峰苯螖閻橀潧浠﹂柣妤冨█閵嗕礁顫滈埀顒勫箖閳哄懎绠甸柟鐑樺灩閼稿綊姊婚崒姘偓宄懊归崶褜娴栭柕濞у懐鐒兼繛杈剧秬椤曟艾煤椤忓秵鏅ｉ梺闈涚箳婵箖骞楅弴銏♀拺缂備焦蓱閻撱儵鏌涘顒夊剶闁糕晜鐩獮鎺懳旀担绯曞亾閻㈠憡鐓熼柕蹇婃閸熷繘鏌ｉ幒鎾冲姢闁宠鍨块崹楣冩嚑椤掑倻鍘繝娈垮枛閿曘劌鈻嶉敐鍥у灊婵炲棗绻嗛弸搴ㄦ煙椤栧棗鍟ˉ鎰節閻㈤潧啸闁轰焦鎮傚畷鎴﹀箛閺夎法锛涢梺鐟板⒔缁垶鍩涢幋锔界厱婵犻潧妫楅顏呯節閳ь剚瀵肩€涙鍘介梺鍐叉惈閿曘倝鎮橀敂閿亾鐟欏嫭绀冩俊鐐舵閻ｇ兘鎮㈢喊杈ㄦ櫖濠殿喗锕㈢涵绋课ｆ导瀛樷拻濞达綀娅ｇ敮娑㈡煕閺冩挾鐣电€规洘绮岄埥澶愬閻樺疇绶㈤梻浣瑰濞叉牠宕愯ぐ鎺戠厱?
// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁惧墽鎳撻—鍐偓锝庝簼閹癸綁鏌ｉ鐐搭棞闁靛棙甯掗～婵嬫晲閸涱剙顥氬┑掳鍊楁慨鐑藉磻濞戔懞鍥偨缁嬫寧鐎梺鐟板⒔缁垶宕戦幇鐗堢厱闁归偊鍨扮槐锕傛煟閵忋垻甯涘ǎ鍥э躬閹瑩顢旈崟銊ヤ壕闁哄稁鍘介崑瀣節婵犲倻澧曠痪鍓х帛缁绘盯骞嬪▎蹇曚患闂佹悶鍔岄崐鍧楀蓟濞戞矮娌柛鎾椻偓濡劍绻涚€涙鐭嬬紒顔芥崌瀵鍨鹃幇浣告倯闂佸憡鍔戦崝宀勨€栫€ｎ喗鈷戦梻鍫熺⊕椤ョ偤鎮介娑辨疁濠碉紕鏁诲畷鐔碱敍濮樺崬骞愰梻浣侯焾閺堫剛鍒掔仦绛嬪殨闁靛ň鏅滈埛鎴︽煕濠靛棗顏柣蹇涗憾閺屾盯鎮╁畷鍥р拰濡ょ姷鍋涢崯顐︹€﹂妸鈺佺闁绘劦鍓欑紓鎾绘⒒娴ｈ櫣銆婇柛鎾寸箞閳ワ箓宕堕鈧Ч鏌ユ煥閺囩偛鈧綊鍩涢幒鎳ㄥ綊鏁愰崼鐕佷哗闁汇埄鍨遍惄顖炲箖娴犲鏁嶆繝濠傚暕閹寸兘姊洪崨濠傜瑲閻㈩垽绻濋妴浣糕槈閵忊€斥偓鐑芥煠绾板崬澧伴柡澶夌矙濮婄粯绗熼埀顒€顭囪閳ワ箓宕奸妷銉э紵濠电偛妫欓幐濠氬磻閸曨垱鐓曢煫鍥ㄦ尭閹垿鏌涙惔顔婚偗婵﹨娅ｉ幏顐ｆ償閳ヨ櫕娈查梺瑙勫絻閹芥粎妲愰幒妤€鐒垫い鎺嶈兌缁♀偓闂佸憡娲﹂崢楣冩晬濠靛鈷戠紒瀣濠€浼存煟閻旀潙濮傜€规洘顨堟禒锔炬喆閿濆棙鏉告俊鐐€栧濠氬磻閹捐姹叉い鎺戝閻撴瑦銇勯弬璇插婵炶绠撳畷鎴﹀箛閻楀牏鍘卞┑鐐叉缁绘劙鍩€椤掆偓缂嶅﹪骞嗛崟顖ｆ晬婵椴稿▓楣冩⒑闂堟单鍫ュ疾濞戞氨涓嶉悷娆忓娴滄粓鏌熼幑鎰【闁哄鍨剁换娑㈠川椤撶噥妫﹀┑顔硷功缁垶骞忛崨顔剧懝妞ゆ牗绮屾慨濂告⒒娴ｅ憡鎯堥柣妤佺矒瀹曟粌鈻庨幙鍕◤濠电娀娼ч鍛劔闂備焦瀵у濠氭惞鎼粹埗褰掑礋椤戣法绠氶梺闈涚墕缁绘帡宕氭导瀛樼厱闁绘洑绀侀悘锕傛煕?session 闂傚倸鍊搁崐鎼佸磹閹间礁纾圭€瑰嫭鍣磋ぐ鎺戠倞妞ゆ帒顦伴弲顏堟偡濠婂啰绠婚柛鈹惧亾濡炪倖甯婇懗鍫曞煝閹剧粯鐓涢柛娑卞灠閳诲牓鏌曢崱鏇狀槮闁宠閰ｉ獮姗€宕橀幓鎺撴殢濠碉紕鍋戦崐鏍箰妤ｅ啫纾婚柣鏂垮悑閸嬫﹢鏌曟径鍡樻珕闁抽攱鍨块弻娑樷攽閸℃浼€闂佸疇顕чˇ鐢稿蓟濞戞鐔煎垂椤旂粯鐫忕紓鍌欑贰閸犳稑鐣烽悽绋跨疅闁圭虎鍠栫粈瀣煃鐞涒€充壕缂備降鍔岄…宄邦潖閾忚鍠嗛柛鏇ㄥ墰閸戔剝绻濋埛鈧崘顔煎及闂佽鍣ｇ粻鏍蓟閸℃鍚嬮柛鈩冪懃楠炴劙姊绘担鍛婂暈婵炴彃绉瑰鎻掆槈閵忕姷顦┑鐘诧工閸欏洦鎯旈妸銉у€為悷婊勭箞閻擃剟顢楅埀顒勫煘閹达箑鐏崇€规洖娲ら悡鐔兼倵鐟欏嫭纾搁柛鏃€鍨块妴浣糕枎閹惧磭顦х紒鐐緲瑜板宕Δ鍐＝闁稿本鑹鹃埀顒佹倐瀹曟劖顦版惔銏狀€涢梺鍝勮閸庤京澹曟繝姘厵闁告挆鍛闂佺粯鎸诲ú鐔煎蓟瀹ュ鐓涘ù锝呮啞濞堝爼姊洪幇浣哥仭婵炶尙鍠栧璇差吋婢跺鍙嗛柣搴秵娴滅偞瀵煎畝鍕拺闁告繂瀚﹢浼存煟閳哄﹤鐏﹂柛銊╃畺閺佸啴宕掑槌栧悈婵犵數濞€濞佳兠洪妸銊ｄ汗妞ゆ牜鍋為埛鎺懨归敐鍕劅闁绘帡绠栭幃妤€鈽夐幒鎾寸彋閻庤娲樼换鍌炲煝鎼淬劌绠婚悗鍦У閹蹭即姊绘担鐟邦嚋缂佽鍊胯棟妞ゆ牗绮庢稉宥囩磽娴ｉ婊勭濠婂牊鐓涚€广儱鍟俊鍧楁煟閹垮啫鐏犳い顓℃硶閹叉挳宕熼鍌ゆЧ婵?
type openAIFastPolicyCtxKeyType struct{}

var openAIFastPolicyCtxKey = openAIFastPolicyCtxKeyType{}

// withOpenAIFastPolicyContext 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊炲銈嗗笒椤︿即寮查鍫熷仭婵犲﹤鍟扮粻缁橆殽閻愭潙鐏村┑顔瑰亾闂侀潧鐗嗛幊鎰版偪閳ь剚淇婇悙顏勨偓鏍涙担鑲濇盯宕熼浣稿妳婵犵數濮村ú锕傚煕閹寸姵鍠愰柣妤€鐗嗙粭鎺懨瑰鈧崡鎶藉蓟濞戞瑦鍎熼柕濠忛檮闁款參姊虹€圭姵顥夋い锕€鐏氶幈銊╁焵椤掑嫭鐓熸俊顖氬悑閺嗏晜绻涢悡搴█婵﹦绮幏鍛瑹椤栨粌濮奸梻浣告惈閻楁粓宕滃☉銏犵闁圭儤鏌у▽顏堟煟閿濆懏婀版繛鍫ョ畺濮婃椽妫冨☉杈ㄐ㈤梺鍝勬噺缁捇骞冮敓鐘冲亜闁绘挸娴烽鎰攽閻戝洨绉甸柛鎾寸懇閹﹢顢氶埀顒勫蓟瀹ュ鏁嶆繛鎴炵懅椤︺劑姊洪棃娑欐悙閻庢矮鍗抽妴浣割潨閳ь剟骞冨▎鎾崇骇闁瑰鍋熸禍鐑芥⒒?settings 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻鐔兼⒒绾惧鍞归梺閫炲苯澧剧紒鐘虫崌閵嗕礁螖閸涱厾顦板銈嗗姂閸娧呮闁秵鈷掑ù锝呮啞閸熺偞绻涚拠褏鐣电€规洘鍔楅埀顒婄秵閸樿绂嶈ぐ鎺撶厵闁绘垶锚閻忊晠鏌ㄥ☉娆戠煉婵﹨娅ｇ槐鎺懳熼崫鍕垫綑闂佽绻愮换鎴︽偡閳哄懎鏋佺€广儱鎳夊Σ鍫ユ煏韫囧﹥顎嗛柟鐤缁辨挻绗熼崶褏浠┑鐐插级閻楁寮查崼鏇熷殤妞ゆ帒鍊婚敍婊堟煟鎼搭垳绉甸柛瀣閹﹢骞橀鐣屽幍濡炪倖鐗楁穱铏光偓姘嵆閹藉爼寮介鐔哄幈濠电偞鍨靛畷顒勭嵁濡眹浜滈柡鍐ｅ亾闁告梹鐟╁濠氬焺閸愩劎绐炴繝鐢靛Т鐎氼剟銆傞悜妯肩瘈闁冲皝鍋撻柛灞剧矌閻撴挸螖閻橀潧浠滈柛鐔告尦楠炲﹪寮介鐐靛幐闂佸憡鍔忛弲顏嗘閿熺姵鈷掑ù锝囩摂閸ゅ啴鏌涢妸銉ヮ劉闁瑰嘲鎳樺褰掑箛椤掆偓閻忓﹪姊婚崒姘偓椋庣矆娓氣偓楠炴牠顢曢敂钘夊壎婵犻潧鍊婚…鍫㈢玻濡ゅ懏鐓欓柟瑙勫姦閸ゆ瑩鏌ｉ幒鎴犱粵闁靛洤瀚伴獮鎺戭吋閸繂甯梻浣瑰墯閸ㄥ崬煤濮椻偓婵＄敻宕熼锝嗘櫍闂佺粯鍔栧娆戠玻濞戞氨纾藉〒姘搐閺嬶附銇勯弴鍡楁祫缂嶆牗绻濇繝鍌滃闁绘帒鐏氶妵鍕箳閹存繍浠奸梺钘夊暟閸犳牠寮婚妸鈺傚亜闁告繂瀚呴姀鈽嗘闁绘劦浜滈悘鏌ユ煛瀹€瀣瘈鐎规洖鐖奸崺鈩冩媴妞嬪孩宕熺紓鍌氬€峰鎺旀閿熺姴闂柨婵嗘媼濞?context闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵噣宕奸悢鍛婎唶闂備胶顭堥鍡涘箰閸撗冨灊妞ゆ挾鍋愬Σ鍫熶繆椤栨繍鍤欐繛鍛囧洦鈷戞繛鑼额嚙楠炴鏌ｉ悢鍙夋珚鐎殿喖顭烽幃銏㈡偘閳ュ厖澹曢梺姹囧灪椤旀牠鎮為幆顬″綊鎮╁▎蹇斿闁抽攱甯￠弻娑㈠即閵娿儰绨诲銈呮禋閸欏啴寮婚垾宕囨殕閻庯綆鍓涢惁鍫ユ倵鐟欏嫭绀冮柨鏇樺灪娣囧﹪骞栨担鑲濄劑鏌曡箛鏇炐″瑙勬礋閹嘲顭ㄩ崨顓ф毉闁汇埄鍨弲鐘差嚕椤愶箑绀冩い顓烆儏缂嶅﹪骞冮埡渚囧晠妞ゆ梻鍘ф竟澶愭⒒娴ｈ櫣甯涢柟鎼佺畺瀹曚即寮介鐐舵憰闂佸搫娴勭槐鏇㈡偪閳ь剟姊洪崫鍕窛闁稿鍋婃俊鐑藉Ψ閸愩劎绉洪柟顔规櫅閻ｇ兘宕堕妸銉ョ仭闂?ctx
// 闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧湱鈧懓瀚崳纾嬨亹閹烘垹鍊為悷婊冪箻瀵娊鏁冮崒娑氬幗闂侀潧绻堥崺鍕倿閸撗呮／闁诡垎宀€鍚嬮梺鍝勭焿缂嶄線鐛崶顒夋晩闁兼亽鍎查惁搴ㄦ⒒娴ｈ銇熼柛妯圭矙閹兘鍩￠崨顓℃憰闂佺粯妫侀妴鈧柛瀣崌閹棄鈻撶捄銊ュЪ闂佸摜鍠愰幃鍌氼潖閸濆娊铏规嫚閹绘帞顔戦梻浣告贡閹虫挸煤椤撶儐鍤曢悹鍥ㄧゴ濡插牓鏌曡箛濠冩珕闁哄懏绻堝娲箰鎼达絿鐣靛┑鈽嗗亝閻╊垵妫熼梺闈涱槴閺呮粓鍩?goroutine 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻鐔兼⒒鐎靛壊妲紒鐐劤椤兘寮婚敐澶婄疀妞ゆ帊鐒﹂崕鎾绘⒑閹肩偛濡奸柛濠傛健瀵鈽夐姀鈺傛櫇闂佹寧绻傚Λ娑⑺囬妷褏纾藉ù锝呮惈瀛濈紓鍌氱Т閿曨亜顕ｇ拠宸悑濠㈣泛锕ｇ槐鍫曟⒑閸涘﹥澶勯柛鎾寸懃閳诲秹鏁愭径瀣ф嫼缂備礁顑堥崕濠氾綖閿曞倹鐓曢柡鍐ｅ亾闁搞劌鐏濋锝嗙節濮橆厼浜滅紒鐐妞存悂寮查鍕拺缂侇垱娲嶉崑鎾崇暦閸モ晩鍞跺┑鐐茬摠缁挾绮婚弽褜娼?evaluateOpenAIFastPolicy 婵犵數濮烽弫鍛婃叏閻戣棄鏋侀柛娑橈攻閸欏繘鏌ｉ幋锝嗩棄闁哄绶氶弻娑樷槈濮楀牊鏁鹃梺鍛婄懃缁绘劙婀侀梺绋跨箰閸氬绱為幋锔界厱闁靛鍎遍埀顒€娼″濠氭晲婢跺﹦顔掗悗瑙勬礀濞层倝宕ú顏呪拺闁告繂瀚烽崕鎰版煟濡や緡娈橀柟骞垮灩閳藉濮€閻樻鍚呴梻浣虹帛閸旀洟顢氶鐐╂灁闁靛牆顦伴埛鎴炵箾閼奸鍤欐鐐搭殜閺岀喖鎮烽悧鍫濇灎閻庢鍠涢褔鍩ユ径鎰潊闁冲搫鍊瑰▍鍥⒒娴ｇ懓顕滅紒璇插€歌灋婵炴垟鎳為崶顒€唯闁冲搫鍊甸幏娲⒑閸涘﹦绠撻悗姘卞厴瀹曟洘鎯旈敐鍥╋紲闂佸吋鎮傚褔宕搹鍏夊亾濞堝灝鏋涙い顓炲槻椤曪綁骞橀纰辨綂闂佺粯顭堥褔寮憴鍕瘈闁汇垽娼у瓭闂佽绻戝畝绋跨暦濠靛鍐€妞ゆ劑鍊栭崳顕€姊?
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
	MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"type":    "permission_error",
			"message": err.Message,
		},
	})
}

// applyOpenAIFastPolicyToWSResponseCreate evaluates the OpenAI fast policy
// against a single client闂傚倸鍊搁崐鎼佸磹閹间礁纾归柟闂寸绾惧綊鏌熼梻瀵割槮缁炬儳缍婇弻锝夊箣閿濆憛鎾绘煕婵犲倹鍋ラ柡灞诲姂瀵挳鎮欏ù瀣壕闁告縿鍎抽惌鍡涙煕椤愩倕鏋戦柛娆忕箲娣囧﹪顢涘☉姗嗗殝缂佺虎鍘兼晶浠嬫儉椤忓牆绠氱憸婊堟偂婵傚憡鐓涢悘鐐插⒔閳藉鏌嶉挊澶樻█鐎规洩绻濋幃娆戔偓鐢告櫜閹查箖姊婚崒娆戭槮闁硅绻濋幃鐑藉Ψ瑜庡畷鏌ユ煙閻戞﹩娈曢柛搴☆槹閵囧嫯绠涢幘璺侯杸闂佺锕弨閬嶅Φ閸曨垰绠抽柛鈩冦仦婢规洟鏌ｉ悙瀵稿暡缂佺姷鐣瀍am WebSocket frame whose top-level
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
	// every client event, so an empty type is malformed input; let the
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
