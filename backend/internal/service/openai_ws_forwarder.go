package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

const (
	openAIWSBetaV1Value = "responses_websockets=2026-02-04"
	openAIWSBetaV2Value = "responses_websockets=2026-02-06"

	openAIWSTurnStateHeader    = "x-codex-turn-state"
	openAIWSTurnMetadataHeader = "x-codex-turn-metadata"

	openAIWSLogValueMaxLen      = 160
	openAIWSHeaderValueMaxLen   = 120
	openAIWSIDValueMaxLen       = 64
	openAIWSEventLogHeadLimit   = 20
	openAIWSEventLogEveryN      = 50
	openAIWSBufferLogHeadLimit  = 8
	openAIWSBufferLogEveryN     = 20
	openAIWSPrewarmEventLogHead = 10
	openAIWSPayloadKeySizeTopN  = 6

	openAIWSPayloadSizeEstimateDepth    = 3
	openAIWSPayloadSizeEstimateMaxBytes = 64 * 1024
	openAIWSPayloadSizeEstimateMaxItems = 16

	openAIWSEventFlushBatchSizeDefault    = 4
	openAIWSEventFlushIntervalDefault     = 25 * time.Millisecond
	openAIWSPayloadLogSampleDefault       = 0.2
	openAIWSPassthroughIdleTimeoutDefault = time.Hour

	openAIWSStoreDisabledConnModeStrict   = "strict"
	openAIWSStoreDisabledConnModeAdaptive = "adaptive"
	openAIWSStoreDisabledConnModeOff      = "off"

	openAIWSIngressStagePreviousResponseNotFound = "previous_response_not_found"
	openAIWSMaxPrevResponseIDDeletePasses        = 8
)

var openAIWSLogValueReplacer = strings.NewReplacer(
	"error", "err",
	"fallback", "fb",
	"warning", "warnx",
	"failed", "fail",
)

var openAIWSIngressPreflightPingIdle = 20 * time.Second

// openAIWSFallbackError 閻炴稏鍔庨妵姘跺矗椤栨氨鏆旈柛蹇嬪妼濞叉牠鏌呴埀顒勫礆?HTTP 闁?WS 闂佹寧鐟ㄩ銈夋晬閸繄姣ｉ柡鍫簻閸熸挻绋夌€ｎ偆鍩楅柨娑橆槶閳?
type openAIWSFallbackError struct {
	Reason string
	Err    error
}

func (e *openAIWSFallbackError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("openai ws fallback: %s", strings.TrimSpace(e.Reason))
	}
	return fmt.Sprintf("openai ws fallback: %s: %v", strings.TrimSpace(e.Reason), e.Err)
}

func (e *openAIWSFallbackError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapOpenAIWSFallback(reason string, err error) error {
	return &openAIWSFallbackError{Reason: strings.TrimSpace(reason), Err: err}
}

// OpenAIWSClientCloseError 閻炴稏鍔庨妵姘償閺傚彞绨伴柟绋挎搐閻?WebSocket close code 濞戞捁顕ф慨鈺呭礂閹惰姤锛旈悗骞垮灪閸╂稓绮╅婵堢闁规亽鍎冲▓鎴︽煥濞嗘帩鍤栭柕?
type OpenAIWSClientCloseError struct {
	statusCode coderws.StatusCode
	reason     string
	err        error
}

type openAIWSIngressTurnError struct {
	stage           string
	cause           error
	wroteDownstream bool
}

func (e *openAIWSIngressTurnError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause == nil {
		return strings.TrimSpace(e.stage)
	}
	return e.cause.Error()
}

func (e *openAIWSIngressTurnError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func wrapOpenAIWSIngressTurnError(stage string, cause error, wroteDownstream bool) error {
	if cause == nil {
		return nil
	}
	return &openAIWSIngressTurnError{
		stage:           strings.TrimSpace(stage),
		cause:           cause,
		wroteDownstream: wroteDownstream,
	}
}

func isOpenAIWSIngressTurnRetryable(err error) bool {
	var turnErr *openAIWSIngressTurnError
	if !errors.As(err, &turnErr) || turnErr == nil {
		return false
	}
	if errors.Is(turnErr.cause, context.Canceled) || errors.Is(turnErr.cause, context.DeadlineExceeded) {
		return false
	}
	if turnErr.wroteDownstream {
		return false
	}
	switch turnErr.stage {
	case "write_upstream", "read_upstream":
		return true
	default:
		return false
	}
}

func openAIWSIngressTurnRetryReason(err error) string {
	var turnErr *openAIWSIngressTurnError
	if !errors.As(err, &turnErr) || turnErr == nil {
		return "unknown"
	}
	if turnErr.stage == "" {
		return "unknown"
	}
	return turnErr.stage
}

func isOpenAIWSIngressPreviousResponseNotFound(err error) bool {
	var turnErr *openAIWSIngressTurnError
	if !errors.As(err, &turnErr) || turnErr == nil {
		return false
	}
	if strings.TrimSpace(turnErr.stage) != openAIWSIngressStagePreviousResponseNotFound {
		return false
	}
	return !turnErr.wroteDownstream
}

// NewOpenAIWSClientCloseError 闁告帗绋戠紓鎾寸▔閳ь剚绋夐鍕靛悅闁规挳顥撻?WS 闁稿繑濞婂Λ鎾煥濞嗘帩鍤栭柕?
func NewOpenAIWSClientCloseError(statusCode coderws.StatusCode, reason string, err error) error {
	return &OpenAIWSClientCloseError{
		statusCode: statusCode,
		reason:     strings.TrimSpace(reason),
		err:        err,
	}
}

func (e *OpenAIWSClientCloseError) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return fmt.Sprintf("openai ws client close: %d %s", int(e.statusCode), strings.TrimSpace(e.reason))
	}
	return fmt.Sprintf("openai ws client close: %d %s: %v", int(e.statusCode), strings.TrimSpace(e.reason), e.err)
}

func (e *OpenAIWSClientCloseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *OpenAIWSClientCloseError) StatusCode() coderws.StatusCode {
	if e == nil {
		return coderws.StatusInternalError
	}
	return e.statusCode
}

func (e *OpenAIWSClientCloseError) Reason() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.reason)
}

// OpenAIWSIngressHooks 閻庤鐭粻鐔煎礂閵壯呭讲 WS 婵絽绻嬮柌?turn 闁汇劌瀚弫鎾诲川閽樺鍣柡鍫㈠枎濞叉牜鎷崘锔瑰亾?
type OpenAIWSIngressHooks struct {
	// InitialRequestModel 闁哄嫷鍨堕々鑽ゆ暜瑜庣粭顓㈡焼閹惧瓨衼閻忓繐瀚晶鐘绘儍閸曨噮鍤炴慨鐟板€硅啯闁搞劌顑戠槐婵嬪矗椤忓棙鏆忓ù?usage metadata
	// 闁?reasoning effort 闁告艾娴风槐鎴﹀箳閵娿儺鍤ら柨娑樼灱椤╋箑顫㈤姀銏℃殢濞存粌绨肩粭鍌氥€掑宀婂殲婵懓鍊归崹銊ф媼闄囬崹鍌毼熼垾宕団偓鐑藉Υ?	InitialRequestModel string
	InitialRequestModel string
	BeforeTurn          func(turn int) error
	BeforeRequest       func(turn int, payload []byte, originalModel string) error
	AfterTurn           func(turn int, result *OpenAIForwardResult, turnErr error)
}

func normalizeOpenAIWSLogValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return openAIWSLogValueReplacer.Replace(trimmed)
}

func truncateOpenAIWSLogValue(value string, maxLen int) string {
	normalized := normalizeOpenAIWSLogValue(value)
	if normalized == "-" || maxLen <= 0 {
		return normalized
	}
	if len(normalized) <= maxLen {
		return normalized
	}
	return normalized[:maxLen] + "..."
}

func openAIWSHeaderValueForLog(headers http.Header, key string) string {
	if headers == nil {
		return "-"
	}
	return truncateOpenAIWSLogValue(headers.Get(key), openAIWSHeaderValueMaxLen)
}

func hasOpenAIWSHeader(headers http.Header, key string) bool {
	if headers == nil {
		return false
	}
	return strings.TrimSpace(headers.Get(key)) != ""
}

type openAIWSSessionHeaderResolution struct {
	SessionID          string
	ConversationID     string
	SessionSource      string
	ConversationSource string
}

func resolveOpenAIWSSessionHeaders(c *gin.Context, promptCacheKey string) openAIWSSessionHeaderResolution {
	resolution := openAIWSSessionHeaderResolution{
		SessionSource:      "none",
		ConversationSource: "none",
	}
	if c != nil && c.Request != nil {
		if sessionID := strings.TrimSpace(c.Request.Header.Get("session_id")); sessionID != "" {
			resolution.SessionID = sessionID
			resolution.SessionSource = "header_session_id"
		}
		if conversationID := strings.TrimSpace(c.Request.Header.Get("conversation_id")); conversationID != "" {
			resolution.ConversationID = conversationID
			resolution.ConversationSource = "header_conversation_id"
			if resolution.SessionID == "" {
				resolution.SessionID = conversationID
				resolution.SessionSource = "header_conversation_id"
			}
		}
	}

	cacheKey := strings.TrimSpace(promptCacheKey)
	if cacheKey != "" {
		if resolution.SessionID == "" {
			resolution.SessionID = cacheKey
			resolution.SessionSource = "prompt_cache_key"
		}
	}
	return resolution
}

func shouldLogOpenAIWSEvent(idx int, eventType string) bool {
	if idx <= openAIWSEventLogHeadLimit {
		return true
	}
	if openAIWSEventLogEveryN > 0 && idx%openAIWSEventLogEveryN == 0 {
		return true
	}
	if eventType == "error" || isOpenAIWSTerminalEvent(eventType) {
		return true
	}
	return false
}

func shouldLogOpenAIWSBufferedEvent(idx int) bool {
	if idx <= openAIWSBufferLogHeadLimit {
		return true
	}
	if openAIWSBufferLogEveryN > 0 && idx%openAIWSBufferLogEveryN == 0 {
		return true
	}
	return false
}

func openAIWSEventMayContainModel(eventType string) bool {
	switch eventType {
	case "response.created",
		"response.in_progress",
		"response.completed",
		"response.done",
		"response.failed",
		"response.incomplete",
		"response.cancelled",
		"response.canceled":
		return true
	default:
		trimmed := strings.TrimSpace(eventType)
		if trimmed == eventType {
			return false
		}
		switch trimmed {
		case "response.created",
			"response.in_progress",
			"response.completed",
			"response.done",
			"response.failed",
			"response.incomplete",
			"response.cancelled",
			"response.canceled":
			return true
		default:
			return false
		}
	}
}

func openAIWSEventMayContainToolCalls(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return false
	}
	if strings.Contains(eventType, "function_call") || strings.Contains(eventType, "tool_call") {
		return true
	}
	switch eventType {
	case "response.output_item.added", "response.output_item.done", "response.completed", "response.done":
		return true
	default:
		return false
	}
}

func openAIWSEventShouldParseUsage(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		return true
	default:
		return false
	}
}

func parseOpenAIWSEventEnvelope(message []byte) (eventType string, responseID string, response gjson.Result) {
	if len(message) == 0 {
		return "", "", gjson.Result{}
	}
	values := gjson.GetManyBytes(message, "type", "response.id", "id", "response")
	eventType = strings.TrimSpace(values[0].String())
	if id := strings.TrimSpace(values[1].String()); id != "" {
		responseID = id
	} else {
		responseID = strings.TrimSpace(values[2].String())
	}
	return eventType, responseID, values[3]
}

func openAIWSMessageLikelyContainsToolCalls(message []byte) bool {
	if len(message) == 0 {
		return false
	}
	return bytes.Contains(message, []byte(`"tool_calls"`)) ||
		bytes.Contains(message, []byte(`"tool_call"`)) ||
		bytes.Contains(message, []byte(`"function_call"`))
}

func parseOpenAIWSResponseUsageFromCompletedEvent(message []byte, usage *OpenAIUsage) {
	if usage == nil || len(message) == 0 {
		return
	}
	if parsedUsage, ok := extractOpenAIUsageFromJSONBytes(message); ok {
		*usage = parsedUsage
	}
}

func parseOpenAIWSErrorEventFields(message []byte) (code string, errType string, errMessage string) {
	if len(message) == 0 {
		return "", "", ""
	}
	values := gjson.GetManyBytes(message, "error.code", "error.type", "error.message")
	return strings.TrimSpace(values[0].String()), strings.TrimSpace(values[1].String()), strings.TrimSpace(values[2].String())
}

func summarizeOpenAIWSErrorEventFieldsFromRaw(codeRaw, errTypeRaw, errMessageRaw string) (code string, errType string, errMessage string) {
	code = truncateOpenAIWSLogValue(codeRaw, openAIWSLogValueMaxLen)
	errType = truncateOpenAIWSLogValue(errTypeRaw, openAIWSLogValueMaxLen)
	errMessage = truncateOpenAIWSLogValue(errMessageRaw, openAIWSLogValueMaxLen)
	return code, errType, errMessage
}

func summarizeOpenAIWSErrorEventFields(message []byte) (code string, errType string, errMessage string) {
	if len(message) == 0 {
		return "-", "-", "-"
	}
	return summarizeOpenAIWSErrorEventFieldsFromRaw(parseOpenAIWSErrorEventFields(message))
}

func summarizeOpenAIWSPayloadKeySizes(payload map[string]any, topN int) string {
	if len(payload) == 0 {
		return "-"
	}
	type keySize struct {
		Key  string
		Size int
	}
	sizes := make([]keySize, 0, len(payload))
	for key, value := range payload {
		size := estimateOpenAIWSPayloadValueSize(value, openAIWSPayloadSizeEstimateDepth)
		sizes = append(sizes, keySize{Key: key, Size: size})
	}
	sort.Slice(sizes, func(i, j int) bool {
		if sizes[i].Size == sizes[j].Size {
			return sizes[i].Key < sizes[j].Key
		}
		return sizes[i].Size > sizes[j].Size
	})

	if topN <= 0 || topN > len(sizes) {
		topN = len(sizes)
	}
	parts := make([]string, 0, topN)
	for idx := 0; idx < topN; idx++ {
		item := sizes[idx]
		parts = append(parts, fmt.Sprintf("%s:%d", item.Key, item.Size))
	}
	return strings.Join(parts, ",")
}

func estimateOpenAIWSPayloadValueSize(value any, depth int) int {
	if depth <= 0 {
		return -1
	}
	switch v := value.(type) {
	case nil:
		return 0
	case string:
		return len(v)
	case []byte:
		return len(v)
	case bool:
		return 1
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return 8
	case float32, float64:
		return 8
	case map[string]any:
		if len(v) == 0 {
			return 2
		}
		total := 2
		count := 0
		for key, item := range v {
			count++
			if count > openAIWSPayloadSizeEstimateMaxItems {
				return -1
			}
			itemSize := estimateOpenAIWSPayloadValueSize(item, depth-1)
			if itemSize < 0 {
				return -1
			}
			total += len(key) + itemSize + 3
			if total > openAIWSPayloadSizeEstimateMaxBytes {
				return -1
			}
		}
		return total
	case []any:
		if len(v) == 0 {
			return 2
		}
		total := 2
		limit := len(v)
		if limit > openAIWSPayloadSizeEstimateMaxItems {
			return -1
		}
		for i := 0; i < limit; i++ {
			itemSize := estimateOpenAIWSPayloadValueSize(v[i], depth-1)
			if itemSize < 0 {
				return -1
			}
			total += itemSize + 1
			if total > openAIWSPayloadSizeEstimateMaxBytes {
				return -1
			}
		}
		return total
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return -1
		}
		if len(raw) > openAIWSPayloadSizeEstimateMaxBytes {
			return -1
		}
		return len(raw)
	}
}

func openAIWSPayloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return ""
	}
}

func openAIWSPayloadStringFromRaw(payload []byte, key string) string {
	if len(payload) == 0 || strings.TrimSpace(key) == "" {
		return ""
	}
	return strings.TrimSpace(gjson.GetBytes(payload, key).String())
}

func openAIWSPayloadBoolFromRaw(payload []byte, key string, defaultValue bool) bool {
	if len(payload) == 0 || strings.TrimSpace(key) == "" {
		return defaultValue
	}
	value := gjson.GetBytes(payload, key)
	if !value.Exists() {
		return defaultValue
	}
	if value.Type != gjson.True && value.Type != gjson.False {
		return defaultValue
	}
	return value.Bool()
}

func openAIWSSessionHashesFromID(sessionID string) (string, string) {
	return deriveOpenAISessionHashes(sessionID)
}

func extractOpenAIWSImageURL(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if raw, ok := v["url"].(string); ok {
			return strings.TrimSpace(raw)
		}
	}
	return ""
}

func summarizeOpenAIWSInput(input any) string {
	items, ok := input.([]any)
	if !ok || len(items) == 0 {
		return "-"
	}

	itemCount := len(items)
	textChars := 0
	imageDataURLs := 0
	imageDataURLChars := 0
	imageRemoteURLs := 0

	handleContentItem := func(contentItem map[string]any) {
		contentType, _ := contentItem["type"].(string)
		switch strings.TrimSpace(contentType) {
		case "input_text", "output_text", "text":
			if text, ok := contentItem["text"].(string); ok {
				textChars += len(text)
			}
		case "input_image":
			imageURL := extractOpenAIWSImageURL(contentItem["image_url"])
			if imageURL == "" {
				return
			}
			if strings.HasPrefix(strings.ToLower(imageURL), "data:image/") {
				imageDataURLs++
				imageDataURLChars += len(imageURL)
				return
			}
			imageRemoteURLs++
		}
	}

	handleInputItem := func(inputItem map[string]any) {
		if content, ok := inputItem["content"].([]any); ok {
			for _, rawContent := range content {
				contentItem, ok := rawContent.(map[string]any)
				if !ok {
					continue
				}
				handleContentItem(contentItem)
			}
			return
		}

		itemType, _ := inputItem["type"].(string)
		switch strings.TrimSpace(itemType) {
		case "input_text", "output_text", "text":
			if text, ok := inputItem["text"].(string); ok {
				textChars += len(text)
			}
		case "input_image":
			imageURL := extractOpenAIWSImageURL(inputItem["image_url"])
			if imageURL == "" {
				return
			}
			if strings.HasPrefix(strings.ToLower(imageURL), "data:image/") {
				imageDataURLs++
				imageDataURLChars += len(imageURL)
				return
			}
			imageRemoteURLs++
		}
	}

	for _, rawItem := range items {
		inputItem, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		handleInputItem(inputItem)
	}

	return fmt.Sprintf(
		"items=%d,text_chars=%d,image_data_urls=%d,image_data_url_chars=%d,image_remote_urls=%d",
		itemCount,
		textChars,
		imageDataURLs,
		imageDataURLChars,
		imageRemoteURLs,
	)
}

func dropOpenAIWSPayloadKey(payload map[string]any, key string, removed *[]string) {
	if len(payload) == 0 || strings.TrimSpace(key) == "" {
		return
	}
	if _, exists := payload[key]; !exists {
		return
	}
	delete(payload, key)
	*removed = append(*removed, key)
}

// applyOpenAIWSRetryPayloadStrategy 闁?WS 閺夆晝鍋熼悽缁樺緞鏉堫偉袝闁哄啯婀圭划搴ｇ矓婵犳碍鐝熼柡鍐Ь椤曘垺绋婃径濠勬憻婵炲牏顣槐?// 闂侇剙鐏濋崢銈夋煂瀹ュ牏妲搁柟瀛樺姇婵盯宕＄€涙ɑ鏆柛娆惷敮顐ｆ叏鐎ｎ収鍤炴慨鐟板€介銏＄▕婢跺牃鍋?// 婵炲鍔嶉崜浼存晬濮濈湕ompt_cache_key 濞戞挸绉寸花鏌ュ捶閵娾晛娅㈤悹鍥ㄦ磻閼垫垹绮旀繝姘彑闁挎稒绋戦悾鐘垫暜閸濄儲鏆忓ù婊冪凹缁辨壆鎷犲┑鍩辨梻鈧纰嶉悥锝囨嫚閸☆厾绀剆ession_id 闁稿繑绮岀花鎶芥晬婢跺牃鍋?
func applyOpenAIWSRetryPayloadStrategy(payload map[string]any, attempt int) (strategy string, removedKeys []string) {
	if len(payload) == 0 {
		return "empty", nil
	}
	if attempt <= 1 {
		return "full", nil
	}

	removed := make([]string, 0, 2)
	if attempt >= 2 {
		dropOpenAIWSPayloadKey(payload, "include", &removed)
	}

	if len(removed) == 0 {
		return "full", nil
	}
	sort.Strings(removed)
	return "trim_optional_fields", removed
}

func logOpenAIWSModeInfo(format string, args ...any) {
	logger.LegacyPrintf("service.openai_gateway", "[OpenAI WS Mode][openai_ws_mode=true] "+format, args...)
}

func isOpenAIWSModeDebugEnabled() bool {
	return logger.L().Core().Enabled(zap.DebugLevel)
}

func logOpenAIWSModeDebug(format string, args ...any) {
	if !isOpenAIWSModeDebugEnabled() {
		return
	}
	logger.LegacyPrintf("service.openai_gateway", "[debug] [OpenAI WS Mode][openai_ws_mode=true] "+format, args...)
}

func logOpenAIWSBindResponseAccountWarn(groupID, accountID int64, responseID string, err error) {
	if err == nil {
		return
	}
	logger.L().Warn(
		"openai.ws_bind_response_account_failed",
		zap.Int64("group_id", groupID),
		zap.Int64("account_id", accountID),
		zap.String("response_id", truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen)),
		zap.Error(err),
	)
}

func summarizeOpenAIWSReadCloseError(err error) (status string, reason string) {
	if err == nil {
		return "-", "-"
	}
	statusCode := coderws.CloseStatus(err)
	if statusCode == -1 {
		return "-", "-"
	}
	closeStatus := fmt.Sprintf("%d(%s)", int(statusCode), statusCode.String())
	closeReason := "-"
	var closeErr coderws.CloseError
	if errors.As(err, &closeErr) {
		reasonText := strings.TrimSpace(closeErr.Reason)
		if reasonText != "" {
			closeReason = normalizeOpenAIWSLogValue(reasonText)
		}
	}
	return normalizeOpenAIWSLogValue(closeStatus), closeReason
}

func unwrapOpenAIWSDialBaseError(err error) error {
	if err == nil {
		return nil
	}
	var dialErr *openAIWSDialError
	if errors.As(err, &dialErr) && dialErr != nil && dialErr.Err != nil {
		return dialErr.Err
	}
	return err
}

func openAIWSDialRespHeaderForLog(err error, key string) string {
	var dialErr *openAIWSDialError
	if !errors.As(err, &dialErr) || dialErr == nil || dialErr.ResponseHeaders == nil {
		return "-"
	}
	return truncateOpenAIWSLogValue(dialErr.ResponseHeaders.Get(key), openAIWSHeaderValueMaxLen)
}

func classifyOpenAIWSDialError(err error) string {
	if err == nil {
		return "-"
	}
	baseErr := unwrapOpenAIWSDialBaseError(err)
	if baseErr == nil {
		return "-"
	}
	if errors.Is(baseErr, context.DeadlineExceeded) {
		return "ctx_deadline_exceeded"
	}
	if errors.Is(baseErr, context.Canceled) {
		return "ctx_canceled"
	}
	var netErr net.Error
	if errors.As(baseErr, &netErr) && netErr.Timeout() {
		return "net_timeout"
	}
	if status := coderws.CloseStatus(baseErr); status != -1 {
		return normalizeOpenAIWSLogValue(fmt.Sprintf("ws_close_%d", int(status)))
	}
	message := strings.ToLower(strings.TrimSpace(baseErr.Error()))
	switch {
	case strings.Contains(message, "handshake not finished"):
		return "handshake_not_finished"
	case strings.Contains(message, "bad handshake"):
		return "bad_handshake"
	case strings.Contains(message, "connection refused"):
		return "connection_refused"
	case strings.Contains(message, "no such host"):
		return "dns_not_found"
	case strings.Contains(message, "tls"):
		return "tls_error"
	case strings.Contains(message, "i/o timeout"):
		return "io_timeout"
	case strings.Contains(message, "context deadline exceeded"):
		return "ctx_deadline_exceeded"
	default:
		return "dial_error"
	}
}

func summarizeOpenAIWSDialError(err error) (
	statusCode int,
	dialClass string,
	closeStatus string,
	closeReason string,
	respServer string,
	respVia string,
	respCFRay string,
	respRequestID string,
) {
	dialClass = "-"
	closeStatus = "-"
	closeReason = "-"
	respServer = "-"
	respVia = "-"
	respCFRay = "-"
	respRequestID = "-"
	if err == nil {
		return
	}
	var dialErr *openAIWSDialError
	if errors.As(err, &dialErr) && dialErr != nil {
		statusCode = dialErr.StatusCode
		respServer = openAIWSDialRespHeaderForLog(err, "server")
		respVia = openAIWSDialRespHeaderForLog(err, "via")
		respCFRay = openAIWSDialRespHeaderForLog(err, "cf-ray")
		respRequestID = openAIWSDialRespHeaderForLog(err, "x-request-id")
	}
	dialClass = normalizeOpenAIWSLogValue(classifyOpenAIWSDialError(err))
	closeStatus, closeReason = summarizeOpenAIWSReadCloseError(unwrapOpenAIWSDialBaseError(err))
	return
}

func isOpenAIWSClientDisconnectError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, context.Canceled) {
		return true
	}
	switch coderws.CloseStatus(err) {
	case coderws.StatusNormalClosure, coderws.StatusGoingAway, coderws.StatusNoStatusRcvd, coderws.StatusAbnormalClosure:
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}
	return strings.Contains(message, "failed to read frame header: eof") ||
		strings.Contains(message, "unexpected eof") ||
		strings.Contains(message, "use of closed network connection") ||
		strings.Contains(message, "connection reset by peer") ||
		strings.Contains(message, "broken pipe") ||
		strings.Contains(message, "an established connection was aborted")
}

func classifyOpenAIWSReadFallbackReason(err error) string {
	if err == nil {
		return "read_event"
	}
	switch coderws.CloseStatus(err) {
	case coderws.StatusPolicyViolation:
		return "policy_violation"
	case coderws.StatusMessageTooBig:
		return "message_too_big"
	default:
		return "read_event"
	}
}

func sortedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *OpenAIGatewayService) getOpenAIWSConnPool() *openAIWSConnPool {
	if s == nil {
		return nil
	}
	s.openaiWSPoolOnce.Do(func() {
		if s.openaiWSPool == nil {
			s.openaiWSPool = newOpenAIWSConnPool(s.cfg)
		}
	})
	return s.openaiWSPool
}

func (s *OpenAIGatewayService) getOpenAIWSPassthroughDialer() openAIWSClientDialer {
	if s == nil {
		return nil
	}
	s.openaiWSPassthroughDialerOnce.Do(func() {
		if s.openaiWSPassthroughDialer == nil {
			s.openaiWSPassthroughDialer = newDefaultOpenAIWSClientDialer()
		}
	})
	return s.openaiWSPassthroughDialer
}

func (s *OpenAIGatewayService) SnapshotOpenAIWSPoolMetrics() OpenAIWSPoolMetricsSnapshot {
	pool := s.getOpenAIWSConnPool()
	if pool == nil {
		return OpenAIWSPoolMetricsSnapshot{}
	}
	return pool.SnapshotMetrics()
}

type OpenAIWSPerformanceMetricsSnapshot struct {
	Pool      OpenAIWSPoolMetricsSnapshot      `json:"pool"`
	Retry     OpenAIWSRetryMetricsSnapshot     `json:"retry"`
	Transport OpenAIWSTransportMetricsSnapshot `json:"transport"`
}

func (s *OpenAIGatewayService) SnapshotOpenAIWSPerformanceMetrics() OpenAIWSPerformanceMetricsSnapshot {
	pool := s.getOpenAIWSConnPool()
	snapshot := OpenAIWSPerformanceMetricsSnapshot{
		Retry: s.SnapshotOpenAIWSRetryMetrics(),
	}
	if pool == nil {
		return snapshot
	}
	snapshot.Pool = pool.SnapshotMetrics()
	snapshot.Transport = pool.SnapshotTransportMetrics()
	return snapshot
}

func (s *OpenAIGatewayService) getOpenAIWSStateStore() OpenAIWSStateStore {
	if s == nil {
		return nil
	}
	s.openaiWSStateStoreOnce.Do(func() {
		if s.openaiWSStateStore == nil {
			s.openaiWSStateStore = NewOpenAIWSStateStore(s.cache)
		}
	})
	return s.openaiWSStateStore
}

func (s *OpenAIGatewayService) openAIWSResponseStickyTTL() time.Duration {
	if s != nil && s.cfg != nil {
		seconds := s.cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds
		if seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return time.Hour
}

func (s *OpenAIGatewayService) openAIWSIngressPreviousResponseRecoveryEnabled() bool {
	if s != nil && s.cfg != nil {
		return s.cfg.Gateway.OpenAIWS.IngressPreviousResponseRecoveryEnabled
	}
	return true
}

func (s *OpenAIGatewayService) openAIWSReadTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.ReadTimeoutSeconds > 0 {
		return time.Duration(s.cfg.Gateway.OpenAIWS.ReadTimeoutSeconds) * time.Second
	}
	return 15 * time.Minute
}

func (s *OpenAIGatewayService) openAIWSPassthroughIdleTimeout() time.Duration {
	if timeout := s.openAIWSReadTimeout(); timeout > 0 {
		return timeout
	}
	return openAIWSPassthroughIdleTimeoutDefault
}

func (s *OpenAIGatewayService) openAIWSWriteTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.WriteTimeoutSeconds > 0 {
		return time.Duration(s.cfg.Gateway.OpenAIWS.WriteTimeoutSeconds) * time.Second
	}
	return 2 * time.Minute
}

func (s *OpenAIGatewayService) openAIWSEventFlushBatchSize() int {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.EventFlushBatchSize > 0 {
		return s.cfg.Gateway.OpenAIWS.EventFlushBatchSize
	}
	return openAIWSEventFlushBatchSizeDefault
}

func (s *OpenAIGatewayService) openAIWSEventFlushInterval() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.EventFlushIntervalMS >= 0 {
		if s.cfg.Gateway.OpenAIWS.EventFlushIntervalMS == 0 {
			return 0
		}
		return time.Duration(s.cfg.Gateway.OpenAIWS.EventFlushIntervalMS) * time.Millisecond
	}
	return openAIWSEventFlushIntervalDefault
}

func (s *OpenAIGatewayService) openAIWSPayloadLogSampleRate() float64 {
	if s != nil && s.cfg != nil {
		rate := s.cfg.Gateway.OpenAIWS.PayloadLogSampleRate
		if rate < 0 {
			return 0
		}
		if rate > 1 {
			return 1
		}
		return rate
	}
	return openAIWSPayloadLogSampleDefault
}

func (s *OpenAIGatewayService) shouldLogOpenAIWSPayloadSchema(attempt int) bool {
	// 濡絾鐗楅鑲╀焊濠靛﹦妲稿ǎ鍥ㄧ箘閺嗏偓濞戞挴鍋撻柡澶嗏偓宕囨殮闁?payload_schema 濞撴艾銇樼花顒勫箳閹烘顔囬柕?
	if attempt <= 1 {
		return true
	}
	rate := s.openAIWSPayloadLogSampleRate()
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	return rand.Float64() < rate
}

func (s *OpenAIGatewayService) shouldEmitOpenAIWSPayloadSchema(attempt int) bool {
	if !s.shouldLogOpenAIWSPayloadSchema(attempt) {
		return false
	}
	return logger.L().Core().Enabled(zap.DebugLevel)
}

func (s *OpenAIGatewayService) openAIWSDialTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.DialTimeoutSeconds > 0 {
		return time.Duration(s.cfg.Gateway.OpenAIWS.DialTimeoutSeconds) * time.Second
	}
	return 10 * time.Second
}

func (s *OpenAIGatewayService) openAIWSAcquireTimeout() time.Duration {
	// Acquire 閻熸洖妫涘ú濠囧灳濠婂棛绠鹃柟鎭掑劚椤︽煡鎮介妸銉﹀殥濞?闁圭儤甯″Σ?闁哄倹婢樼紓鎾存交閻愭潙澶嶉柍銉︾箑缁椾焦绋夐鍫熲枆婵炲牏鍋ｉ埀?	// 閺夆晜鐟╅崳閿嬬▔瀹ュ懎鏅欓柛娆戝Т婵?write_timeout闁挎稑鐭傛导鈺呭礂瀹ュ褰鐐舵硾瑜板倿骞掗幒妤佇曢柡鍐煐婵?TTFT 闂傗偓閸喚鍟查柟宄邦槸閸╁矂宕氶崱娑欏缂佺嫏浣插亾?
	dial := s.openAIWSDialTimeout()
	if dial <= 0 {
		dial = 10 * time.Second
	}
	return dial + 2*time.Second
}

func (s *OpenAIGatewayService) buildOpenAIResponsesWSURL(account *Account) (string, error) {
	if account == nil {
		return "", errors.New("account is nil")
	}
	var targetURL string
	switch account.Type {
	case AccountTypeOAuth:
		targetURL = chatgptCodexURL
	case AccountTypeAPIKey:
		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			targetURL = openaiPlatformAPIURL
		} else {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return "", err
			}
			targetURL = buildOpenAIResponsesURL(validatedURL)
		}
	default:
		targetURL = openaiPlatformAPIURL
	}

	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return "", fmt.Errorf("invalid target url: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	case "wss", "ws":
		// 濞ｅ洦绻冪€垫梹绋夊鍛秮
	default:
		return "", fmt.Errorf("unsupported scheme for ws: %s", parsed.Scheme)
	}
	return parsed.String(), nil
}

func (s *OpenAIGatewayService) buildOpenAIWSHeaders(
	c *gin.Context,
	account *Account,
	token string,
	decision OpenAIWSProtocolDecision,
	isCodexCLI bool,
	turnState string,
	turnMetadata string,
	promptCacheKey string,
) (http.Header, openAIWSSessionHeaderResolution) {
	headers := make(http.Header)
	headers.Set("authorization", "Bearer "+token)

	sessionResolution := resolveOpenAIWSSessionHeaders(c, promptCacheKey)
	if c != nil && c.Request != nil {
		if v := strings.TrimSpace(c.Request.Header.Get("accept-language")); v != "" {
			headers.Set("accept-language", v)
		}
	}
	// OAuth 閻犳劧绠戣ぐ鍧楁晬濮橆剛娈?apiKeyID 婵烇絽鍢查崣?session 闁哄秴娲╅惁鎴犵箔閿旇偐绀夐梻鍐ㄥ级椤掓稓鎹勯妸褎鏆忛柟纾嬫腹缁辨壆鎷犲┑鍫骄闁瑰墽鍋戦埀?
	if account != nil && account.Type == AccountTypeOAuth {
		apiKeyID := getAPIKeyIDFromContext(c)
		if sessionResolution.SessionID != "" {
			headers.Set("session_id", isolateOpenAISessionID(apiKeyID, sessionResolution.SessionID))
		}
		if sessionResolution.ConversationID != "" {
			headers.Set("conversation_id", isolateOpenAISessionID(apiKeyID, sessionResolution.ConversationID))
		}
	} else {
		if sessionResolution.SessionID != "" {
			headers.Set("session_id", sessionResolution.SessionID)
		}
		if sessionResolution.ConversationID != "" {
			headers.Set("conversation_id", sessionResolution.ConversationID)
		}
	}
	if state := strings.TrimSpace(turnState); state != "" {
		headers.Set(openAIWSTurnStateHeader, state)
	}
	if metadata := strings.TrimSpace(turnMetadata); metadata != "" {
		headers.Set(openAIWSTurnMetadataHeader, metadata)
	}

	if account != nil && account.Type == AccountTypeOAuth {
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			headers.Set("chatgpt-account-id", chatgptAccountID)
		}
		headers.Set("originator", resolveOpenAIUpstreamOriginator(c, isCodexCLI))
	}

	betaValue := openAIWSBetaV2Value
	if decision.Transport == OpenAIUpstreamTransportResponsesWebsocket {
		betaValue = openAIWSBetaV1Value
	}
	headers.Set("OpenAI-Beta", betaValue)

	customUA := ""
	if account != nil {
		customUA = account.GetOpenAIUserAgent()
	}
	if strings.TrimSpace(customUA) != "" {
		headers.Set("user-agent", customUA)
	} else if c != nil {
		if ua := strings.TrimSpace(c.GetHeader("User-Agent")); ua != "" {
			headers.Set("user-agent", ua)
		}
	}
	if s != nil && s.cfg != nil && s.cfg.Gateway.ForceCodexCLI {
		headers.Set("user-agent", codexCLIUserAgent)
	}
	if account != nil && account.Type == AccountTypeOAuth && !openai.IsCodexCLIRequest(headers.Get("user-agent")) {
		headers.Set("user-agent", codexCLIUserAgent)
	}

	return headers, sessionResolution
}

func (s *OpenAIGatewayService) buildOpenAIWSCreatePayload(reqBody map[string]any, account *Account) map[string]any {
	// OpenAI WS Mode 闁告绻楅鍛存晬濮濈牣sponse.create 閻庢稒顨嗛灞剧▔?HTTP /responses 闁糕晝鍎ゅ﹢鐗堢▔閳ь剟鎳涘ǎ顑藉亾?	// 濞ｅ洦绻勯弳鈧?stream 閻庢稒顨嗛宀勬晬閸粎鐟?Codex CLI 濞戞挴鍋撻柤鐤彧缁辨岸鏁嶇仦鑲╃煂缂佸顭峰▍?background闁?
	payload := make(map[string]any, len(reqBody)+1)
	for k, v := range reqBody {
		payload[k] = v
	}

	delete(payload, "background")
	if _, exists := payload["stream"]; !exists {
		payload["stream"] = true
	}
	payload["type"] = "response.create"

	// OAuth 濮掓稒顭堥缁樼┍濠靛洤鐦?store=false闁挎稑鐭傛导鈺呭礂瀹ュ牜鍤栧〒姘箚缁傚棝寮靛鍛潳缂佹棏鍨板濠氬矗鐏炲倵鍋?
	if account != nil && account.Type == AccountTypeOAuth && !s.isOpenAIWSStoreRecoveryAllowed(account) {
		payload["store"] = false
	}
	return payload
}

func setOpenAIWSTurnMetadata(payload map[string]any, turnMetadata string) {
	if len(payload) == 0 {
		return
	}
	metadata := strings.TrimSpace(turnMetadata)
	if metadata == "" {
		return
	}

	switch existing := payload["client_metadata"].(type) {
	case map[string]any:
		existing[openAIWSTurnMetadataHeader] = metadata
		payload["client_metadata"] = existing
	case map[string]string:
		next := make(map[string]any, len(existing)+1)
		for k, v := range existing {
			next[k] = v
		}
		next[openAIWSTurnMetadataHeader] = metadata
		payload["client_metadata"] = next
	default:
		payload["client_metadata"] = map[string]any{
			openAIWSTurnMetadataHeader: metadata,
		}
	}
}

func (s *OpenAIGatewayService) isOpenAIWSStoreRecoveryAllowed(account *Account) bool {
	if account != nil && account.IsOpenAIWSAllowStoreRecoveryEnabled() {
		return true
	}
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.AllowStoreRecovery {
		return true
	}
	return false
}

func (s *OpenAIGatewayService) isOpenAIWSStoreDisabledInRequest(reqBody map[string]any, account *Account) bool {
	if account != nil && account.Type == AccountTypeOAuth && !s.isOpenAIWSStoreRecoveryAllowed(account) {
		return true
	}
	if len(reqBody) == 0 {
		return false
	}
	rawStore, ok := reqBody["store"]
	if !ok {
		return false
	}
	storeEnabled, ok := rawStore.(bool)
	if !ok {
		return false
	}
	return !storeEnabled
}

func (s *OpenAIGatewayService) isOpenAIWSStoreDisabledInRequestRaw(reqBody []byte, account *Account) bool {
	if account != nil && account.Type == AccountTypeOAuth && !s.isOpenAIWSStoreRecoveryAllowed(account) {
		return true
	}
	if len(reqBody) == 0 {
		return false
	}
	storeValue := gjson.GetBytes(reqBody, "store")
	if !storeValue.Exists() {
		return false
	}
	if storeValue.Type != gjson.True && storeValue.Type != gjson.False {
		return false
	}
	return !storeValue.Bool()
}

func (s *OpenAIGatewayService) openAIWSStoreDisabledConnMode() string {
	if s == nil || s.cfg == nil {
		return openAIWSStoreDisabledConnModeStrict
	}
	mode := strings.ToLower(strings.TrimSpace(s.cfg.Gateway.OpenAIWS.StoreDisabledConnMode))
	switch mode {
	case openAIWSStoreDisabledConnModeStrict, openAIWSStoreDisabledConnModeAdaptive, openAIWSStoreDisabledConnModeOff:
		return mode
	case "":
		// 闁稿繒鍘ч鎰板籍瑜旈崢銈囩磾椤曞棛绐楀ù鐘叉嚇閸樸倗绱旈鑽ゅ晩閻㈩垰鍟惃闈涱嚕閳ь剟宕楅搹顐ｎ槯闁圭顦板Λ顐ゆ嫚椤撴繄鐤呴柟鎭掑妼椤曢亶濡?
		if s.cfg.Gateway.OpenAIWS.StoreDisabledForceNewConn {
			return openAIWSStoreDisabledConnModeStrict
		}
		return openAIWSStoreDisabledConnModeOff
	default:
		return openAIWSStoreDisabledConnModeStrict
	}
}

func shouldForceNewConnOnStoreDisabled(mode, lastFailureReason string) bool {
	switch mode {
	case openAIWSStoreDisabledConnModeOff:
		return false
	case openAIWSStoreDisabledConnModeAdaptive:
		reason := strings.TrimPrefix(strings.TrimSpace(lastFailureReason), "prewarm_")
		switch reason {
		case "policy_violation", "message_too_big", "auth_failed", "write_request", "write":
			return true
		default:
			return false
		}
	default:
		return true
	}
}

func dropPreviousResponseIDFromRawPayload(payload []byte) ([]byte, bool, error) {
	return dropPreviousResponseIDFromRawPayloadWithDeleteFn(payload, sjson.DeleteBytes)
}

func dropPreviousResponseIDFromRawPayloadWithDeleteFn(
	payload []byte,
	deleteFn func([]byte, string) ([]byte, error),
) ([]byte, bool, error) {
	if len(payload) == 0 {
		return payload, false, nil
	}
	if !gjson.GetBytes(payload, "previous_response_id").Exists() {
		return payload, false, nil
	}
	if deleteFn == nil {
		deleteFn = sjson.DeleteBytes
	}

	updated := payload
	for i := 0; i < openAIWSMaxPrevResponseIDDeletePasses &&
		gjson.GetBytes(updated, "previous_response_id").Exists(); i++ {
		next, err := deleteFn(updated, "previous_response_id")
		if err != nil {
			return payload, false, err
		}
		updated = next
	}
	return updated, !gjson.GetBytes(updated, "previous_response_id").Exists(), nil
}

func setPreviousResponseIDToRawPayload(payload []byte, previousResponseID string) ([]byte, error) {
	normalizedPrevID := strings.TrimSpace(previousResponseID)
	if len(payload) == 0 || normalizedPrevID == "" {
		return payload, nil
	}
	updated, err := sjson.SetBytes(payload, "previous_response_id", normalizedPrevID)
	if err == nil {
		return updated, nil
	}

	var reqBody map[string]any
	if unmarshalErr := json.Unmarshal(payload, &reqBody); unmarshalErr != nil {
		return nil, err
	}
	reqBody["previous_response_id"] = normalizedPrevID
	rebuilt, marshalErr := json.Marshal(reqBody)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return rebuilt, nil
}

func shouldInferIngressFunctionCallOutputPreviousResponseID(
	storeDisabled bool,
	turn int,
	signals ToolContinuationSignals,
	currentPreviousResponseID string,
	expectedPreviousResponseID string,
) bool {
	if !storeDisabled || turn <= 1 || !signals.HasFunctionCallOutput {
		return false
	}
	if strings.TrimSpace(currentPreviousResponseID) != "" {
		return false
	}
	if signals.HasFunctionCallOutputMissingCallID {
		return false
	}
	// If the client already sent the actual tool-call context, treat this as
	// a full replay / self-contained continuation payload rather than
	// downgrading it into an inferred delta continuation. item_reference alone
	// is not enough on the store=false WS path: it still needs a valid prior
	// response anchor so upstream can resolve the referenced function_call.
	if signals.HasToolCallContext {
		return false
	}
	return strings.TrimSpace(expectedPreviousResponseID) != ""
}

func alignStoreDisabledPreviousResponseID(
	payload []byte,
	expectedPreviousResponseID string,
) ([]byte, bool, error) {
	if len(payload) == 0 {
		return payload, false, nil
	}
	expected := strings.TrimSpace(expectedPreviousResponseID)
	if expected == "" {
		return payload, false, nil
	}
	current := openAIWSPayloadStringFromRaw(payload, "previous_response_id")
	if current == "" || current == expected {
		return payload, false, nil
	}

	withoutPrev, removed, dropErr := dropPreviousResponseIDFromRawPayload(payload)
	if dropErr != nil {
		return payload, false, dropErr
	}
	if !removed {
		return payload, false, nil
	}
	updated, setErr := setPreviousResponseIDToRawPayload(withoutPrev, expected)
	if setErr != nil {
		return payload, false, setErr
	}
	return updated, true, nil
}

func cloneOpenAIWSPayloadBytes(payload []byte) []byte {
	if len(payload) == 0 {
		return nil
	}
	cloned := make([]byte, len(payload))
	copy(cloned, payload)
	return cloned
}

func cloneOpenAIWSRawMessages(items []json.RawMessage) []json.RawMessage {
	if items == nil {
		return nil
	}
	cloned := make([]json.RawMessage, 0, len(items))
	for idx := range items {
		cloned = append(cloned, json.RawMessage(cloneOpenAIWSPayloadBytes(items[idx])))
	}
	return cloned
}

func normalizeOpenAIWSJSONForCompare(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, errors.New("json is empty")
	}
	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}

func normalizeOpenAIWSJSONForCompareOrRaw(raw []byte) []byte {
	normalized, err := normalizeOpenAIWSJSONForCompare(raw)
	if err != nil {
		return bytes.TrimSpace(raw)
	}
	return normalized
}

func normalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is empty")
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	delete(decoded, "input")
	delete(decoded, "previous_response_id")
	return json.Marshal(decoded)
}

func openAIWSExtractNormalizedInputSequence(payload []byte) ([]json.RawMessage, bool, error) {
	if len(payload) == 0 {
		return nil, false, nil
	}
	inputValue := gjson.GetBytes(payload, "input")
	if !inputValue.Exists() {
		return nil, false, nil
	}
	if inputValue.Type == gjson.JSON {
		raw := strings.TrimSpace(inputValue.Raw)
		if strings.HasPrefix(raw, "[") {
			var items []json.RawMessage
			if err := json.Unmarshal([]byte(raw), &items); err != nil {
				return nil, true, err
			}
			return items, true, nil
		}
		return []json.RawMessage{json.RawMessage(raw)}, true, nil
	}
	if inputValue.Type == gjson.String {
		encoded, _ := json.Marshal(inputValue.String())
		return []json.RawMessage{encoded}, true, nil
	}
	return []json.RawMessage{json.RawMessage(inputValue.Raw)}, true, nil
}

func openAIWSInputIsPrefixExtended(previousPayload, currentPayload []byte) (bool, error) {
	previousItems, previousExists, prevErr := openAIWSExtractNormalizedInputSequence(previousPayload)
	if prevErr != nil {
		return false, prevErr
	}
	currentItems, currentExists, currentErr := openAIWSExtractNormalizedInputSequence(currentPayload)
	if currentErr != nil {
		return false, currentErr
	}
	if !previousExists && !currentExists {
		return true, nil
	}
	if !previousExists {
		return len(currentItems) == 0, nil
	}
	if !currentExists {
		return len(previousItems) == 0, nil
	}
	if len(currentItems) < len(previousItems) {
		return false, nil
	}

	for idx := range previousItems {
		previousNormalized := normalizeOpenAIWSJSONForCompareOrRaw(previousItems[idx])
		currentNormalized := normalizeOpenAIWSJSONForCompareOrRaw(currentItems[idx])
		if !bytes.Equal(previousNormalized, currentNormalized) {
			return false, nil
		}
	}
	return true, nil
}

func openAIWSRawItemsHasPrefix(items []json.RawMessage, prefix []json.RawMessage) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(items) < len(prefix) {
		return false
	}
	for idx := range prefix {
		previousNormalized := normalizeOpenAIWSJSONForCompareOrRaw(prefix[idx])
		currentNormalized := normalizeOpenAIWSJSONForCompareOrRaw(items[idx])
		if !bytes.Equal(previousNormalized, currentNormalized) {
			return false
		}
	}
	return true
}

func openAIWSRawItemsHasFunctionCallOutput(items []json.RawMessage) bool {
	for _, item := range items {
		if isCodexToolCallOutputItemType(gjson.GetBytes(item, "type").String()) {
			return true
		}
	}
	return false
}

func openAIWSRawPayloadHasToolCallOutput(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	input := gjson.GetBytes(payload, "input")
	if !input.Exists() {
		return false
	}
	if input.IsArray() {
		for _, item := range input.Array() {
			if isCodexToolCallOutputItemType(item.Get("type").String()) {
				return true
			}
		}
		return false
	}
	if input.Type == gjson.JSON {
		return isCodexToolCallOutputItemType(input.Get("type").String())
	}
	return false
}

func buildOpenAIWSReplayInputSequence(
	previousFullInput []json.RawMessage,
	previousFullInputExists bool,
	currentPayload []byte,
	hasPreviousResponseID bool,
) ([]json.RawMessage, bool, error) {
	currentItems, currentExists, currentErr := openAIWSExtractNormalizedInputSequence(currentPayload)
	if currentErr != nil {
		return nil, false, currentErr
	}
	if !hasPreviousResponseID {
		return cloneOpenAIWSRawMessages(currentItems), currentExists, nil
	}
	if !previousFullInputExists {
		return cloneOpenAIWSRawMessages(currentItems), currentExists, nil
	}
	if !currentExists || len(currentItems) == 0 {
		return cloneOpenAIWSRawMessages(previousFullInput), true, nil
	}
	if openAIWSRawItemsHasPrefix(currentItems, previousFullInput) {
		return cloneOpenAIWSRawMessages(currentItems), true, nil
	}
	merged := make([]json.RawMessage, 0, len(previousFullInput)+len(currentItems))
	merged = append(merged, cloneOpenAIWSRawMessages(previousFullInput)...)
	merged = append(merged, cloneOpenAIWSRawMessages(currentItems)...)
	return merged, true, nil
}

func setOpenAIWSPayloadInputSequence(
	payload []byte,
	fullInput []json.RawMessage,
	fullInputExists bool,
) ([]byte, error) {
	if !fullInputExists {
		return payload, nil
	}
	// Preserve [] vs null semantics when input exists but is empty.
	inputForMarshal := fullInput
	if inputForMarshal == nil {
		inputForMarshal = []json.RawMessage{}
	}
	inputRaw, marshalErr := json.Marshal(inputForMarshal)
	if marshalErr != nil {
		return nil, marshalErr
	}
	return sjson.SetRawBytes(payload, "input", inputRaw)
}

func shouldKeepIngressPreviousResponseID(
	previousPayload []byte,
	currentPayload []byte,
	lastTurnResponseID string,
	hasFunctionCallOutput bool,
) (bool, string, error) {
	if hasFunctionCallOutput {
		return true, "has_function_call_output", nil
	}
	currentPreviousResponseID := strings.TrimSpace(openAIWSPayloadStringFromRaw(currentPayload, "previous_response_id"))
	if currentPreviousResponseID == "" {
		return false, "missing_previous_response_id", nil
	}
	expectedPreviousResponseID := strings.TrimSpace(lastTurnResponseID)
	if expectedPreviousResponseID == "" {
		return false, "missing_last_turn_response_id", nil
	}
	if currentPreviousResponseID != expectedPreviousResponseID {
		return false, "previous_response_id_mismatch", nil
	}
	if len(previousPayload) == 0 {
		return false, "missing_previous_turn_payload", nil
	}

	previousComparable, previousComparableErr := normalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(previousPayload)
	if previousComparableErr != nil {
		return false, "non_input_compare_error", previousComparableErr
	}
	currentComparable, currentComparableErr := normalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(currentPayload)
	if currentComparableErr != nil {
		return false, "non_input_compare_error", currentComparableErr
	}
	if !bytes.Equal(previousComparable, currentComparable) {
		return false, "non_input_changed", nil
	}
	return true, "strict_incremental_ok", nil
}

type openAIWSIngressPreviousTurnStrictState struct {
	nonInputComparable []byte
}

func buildOpenAIWSIngressPreviousTurnStrictState(payload []byte) (*openAIWSIngressPreviousTurnStrictState, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	nonInputComparable, nonInputErr := normalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(payload)
	if nonInputErr != nil {
		return nil, nonInputErr
	}
	return &openAIWSIngressPreviousTurnStrictState{
		nonInputComparable: nonInputComparable,
	}, nil
}

func shouldKeepIngressPreviousResponseIDWithStrictState(
	previousState *openAIWSIngressPreviousTurnStrictState,
	currentPayload []byte,
	lastTurnResponseID string,
	hasFunctionCallOutput bool,
) (bool, string, error) {
	if hasFunctionCallOutput {
		return true, "has_function_call_output", nil
	}
	currentPreviousResponseID := strings.TrimSpace(openAIWSPayloadStringFromRaw(currentPayload, "previous_response_id"))
	if currentPreviousResponseID == "" {
		return false, "missing_previous_response_id", nil
	}
	expectedPreviousResponseID := strings.TrimSpace(lastTurnResponseID)
	if expectedPreviousResponseID == "" {
		return false, "missing_last_turn_response_id", nil
	}
	if currentPreviousResponseID != expectedPreviousResponseID {
		return false, "previous_response_id_mismatch", nil
	}
	if previousState == nil {
		return false, "missing_previous_turn_payload", nil
	}

	currentComparable, currentComparableErr := normalizeOpenAIWSPayloadWithoutInputAndPreviousResponseID(currentPayload)
	if currentComparableErr != nil {
		return false, "non_input_compare_error", currentComparableErr
	}
	if !bytes.Equal(previousState.nonInputComparable, currentComparable) {
		return false, "non_input_changed", nil
	}
	return true, "strict_incremental_ok", nil
}

func (s *OpenAIGatewayService) forwardOpenAIWSV2(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	reqBody map[string]any,
	token string,
	decision OpenAIWSProtocolDecision,
	isCodexCLI bool,
	reqStream bool,
	originalModel string,
	mappedModel string,
	startTime time.Time,
	attempt int,
	lastFailureReason string,
) (*OpenAIForwardResult, error) {
	if s == nil || account == nil {
		return nil, wrapOpenAIWSFallback("invalid_state", errors.New("service or account is nil"))
	}

	wsURL, err := s.buildOpenAIResponsesWSURL(account)
	if err != nil {
		return nil, wrapOpenAIWSFallback("build_ws_url", err)
	}
	wsHost := "-"
	wsPath := "-"
	if parsed, parseErr := url.Parse(wsURL); parseErr == nil && parsed != nil {
		if h := strings.TrimSpace(parsed.Host); h != "" {
			wsHost = normalizeOpenAIWSLogValue(h)
		}
		if p := strings.TrimSpace(parsed.Path); p != "" {
			wsPath = normalizeOpenAIWSLogValue(p)
		}
	}
	logOpenAIWSModeDebug(
		"dial_target account_id=%d account_type=%s ws_host=%s ws_path=%s",
		account.ID,
		account.Type,
		wsHost,
		wsPath,
	)

	payload := s.buildOpenAIWSCreatePayload(reqBody, account)
	payloadStrategy, removedKeys := applyOpenAIWSRetryPayloadStrategy(payload, attempt)
	previousResponseID := openAIWSPayloadString(payload, "previous_response_id")
	previousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
	promptCacheKey := openAIWSPayloadString(payload, "prompt_cache_key")
	_, hasTools := payload["tools"]
	debugEnabled := isOpenAIWSModeDebugEnabled()
	payloadBytes := -1
	resolvePayloadBytes := func() int {
		if payloadBytes >= 0 {
			return payloadBytes
		}
		payloadBytes = len(payloadAsJSONBytes(payload))
		return payloadBytes
	}
	streamValue := "-"
	if raw, ok := payload["stream"]; ok {
		streamValue = normalizeOpenAIWSLogValue(strings.TrimSpace(fmt.Sprintf("%v", raw)))
	}
	turnState := ""
	turnMetadata := ""
	if c != nil && c.Request != nil {
		turnState = strings.TrimSpace(c.GetHeader(openAIWSTurnStateHeader))
		turnMetadata = strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader))
	}
	setOpenAIWSTurnMetadata(payload, turnMetadata)
	payloadEventType := openAIWSPayloadString(payload, "type")
	if payloadEventType == "" {
		payloadEventType = "response.create"
	}
	if s.shouldEmitOpenAIWSPayloadSchema(attempt) {
		logOpenAIWSModeInfo(
			"[debug] payload_schema account_id=%d attempt=%d event=%s payload_keys=%s payload_bytes=%d payload_key_sizes=%s input_summary=%s stream=%s payload_strategy=%s removed_keys=%s has_previous_response_id=%v has_prompt_cache_key=%v has_tools=%v",
			account.ID,
			attempt,
			payloadEventType,
			normalizeOpenAIWSLogValue(strings.Join(sortedKeys(payload), ",")),
			resolvePayloadBytes(),
			normalizeOpenAIWSLogValue(summarizeOpenAIWSPayloadKeySizes(payload, openAIWSPayloadKeySizeTopN)),
			normalizeOpenAIWSLogValue(summarizeOpenAIWSInput(payload["input"])),
			streamValue,
			normalizeOpenAIWSLogValue(payloadStrategy),
			normalizeOpenAIWSLogValue(strings.Join(removedKeys, ",")),
			previousResponseID != "",
			promptCacheKey != "",
			hasTools,
		)
	}

	stateStore := s.getOpenAIWSStateStore()
	groupID := getOpenAIGroupIDFromContext(c)
	sessionHash := s.GenerateSessionHash(c, nil)
	if sessionHash == "" {
		var legacySessionHash string
		sessionHash, legacySessionHash = openAIWSSessionHashesFromID(promptCacheKey)
		attachOpenAILegacySessionHashToGin(c, legacySessionHash)
	}
	if turnState == "" && stateStore != nil && sessionHash != "" {
		if savedTurnState, ok := stateStore.GetSessionTurnState(groupID, sessionHash); ok {
			turnState = savedTurnState
		}
	}
	preferredConnID := ""
	if stateStore != nil && previousResponseID != "" {
		if connID, ok := stateStore.GetResponseConn(previousResponseID); ok {
			preferredConnID = connID
		}
	}
	storeDisabled := s.isOpenAIWSStoreDisabledInRequest(reqBody, account)
	if stateStore != nil && storeDisabled && previousResponseID == "" && sessionHash != "" {
		if connID, ok := stateStore.GetSessionConn(groupID, sessionHash); ok {
			preferredConnID = connID
		}
	}
	storeDisabledConnMode := s.openAIWSStoreDisabledConnMode()
	forceNewConnByPolicy := shouldForceNewConnOnStoreDisabled(storeDisabledConnMode, lastFailureReason)
	forceNewConn := forceNewConnByPolicy && storeDisabled && previousResponseID == "" && sessionHash != "" && preferredConnID == ""
	wsHeaders, sessionResolution := s.buildOpenAIWSHeaders(c, account, token, decision, isCodexCLI, turnState, turnMetadata, promptCacheKey)
	logOpenAIWSModeDebug(
		"acquire_start account_id=%d account_type=%s transport=%s preferred_conn_id=%s has_previous_response_id=%v session_hash=%s has_turn_state=%v turn_state_len=%d has_turn_metadata=%v turn_metadata_len=%d store_disabled=%v store_disabled_conn_mode=%s retry_last_reason=%s force_new_conn=%v header_user_agent=%s header_openai_beta=%s header_originator=%s header_accept_language=%s header_session_id=%s header_conversation_id=%s session_id_source=%s conversation_id_source=%s has_prompt_cache_key=%v has_chatgpt_account_id=%v has_authorization=%v has_session_id=%v has_conversation_id=%v proxy_enabled=%v",
		account.ID,
		account.Type,
		normalizeOpenAIWSLogValue(string(decision.Transport)),
		truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen),
		previousResponseID != "",
		truncateOpenAIWSLogValue(sessionHash, 12),
		turnState != "",
		len(turnState),
		turnMetadata != "",
		len(turnMetadata),
		storeDisabled,
		normalizeOpenAIWSLogValue(storeDisabledConnMode),
		truncateOpenAIWSLogValue(lastFailureReason, openAIWSLogValueMaxLen),
		forceNewConn,
		openAIWSHeaderValueForLog(wsHeaders, "user-agent"),
		openAIWSHeaderValueForLog(wsHeaders, "openai-beta"),
		openAIWSHeaderValueForLog(wsHeaders, "originator"),
		openAIWSHeaderValueForLog(wsHeaders, "accept-language"),
		openAIWSHeaderValueForLog(wsHeaders, "session_id"),
		openAIWSHeaderValueForLog(wsHeaders, "conversation_id"),
		normalizeOpenAIWSLogValue(sessionResolution.SessionSource),
		normalizeOpenAIWSLogValue(sessionResolution.ConversationSource),
		promptCacheKey != "",
		hasOpenAIWSHeader(wsHeaders, "chatgpt-account-id"),
		hasOpenAIWSHeader(wsHeaders, "authorization"),
		hasOpenAIWSHeader(wsHeaders, "session_id"),
		hasOpenAIWSHeader(wsHeaders, "conversation_id"),
		account.ProxyID != nil && account.Proxy != nil,
	)

	acquireCtx, acquireCancel := context.WithTimeout(ctx, s.openAIWSAcquireTimeout())
	defer acquireCancel()

	lease, err := s.getOpenAIWSConnPool().Acquire(acquireCtx, openAIWSAcquireRequest{
		Account:         account,
		WSURL:           wsURL,
		Headers:         wsHeaders,
		PreferredConnID: preferredConnID,
		ForceNewConn:    forceNewConn,
		ProxyURL: func() string {
			if account.ProxyID != nil && account.Proxy != nil {
				return account.Proxy.URL()
			}
			return ""
		}(),
	})
	if err != nil {
		dialStatus, dialClass, dialCloseStatus, dialCloseReason, dialRespServer, dialRespVia, dialRespCFRay, dialRespReqID := summarizeOpenAIWSDialError(err)
		logOpenAIWSModeInfo(
			"acquire_fail account_id=%d account_type=%s transport=%s reason=%s dial_status=%d dial_class=%s dial_close_status=%s dial_close_reason=%s dial_resp_server=%s dial_resp_via=%s dial_resp_cf_ray=%s dial_resp_x_request_id=%s cause=%s preferred_conn_id=%s force_new_conn=%v ws_host=%s ws_path=%s proxy_enabled=%v",
			account.ID,
			account.Type,
			normalizeOpenAIWSLogValue(string(decision.Transport)),
			normalizeOpenAIWSLogValue(classifyOpenAIWSAcquireError(err)),
			dialStatus,
			dialClass,
			dialCloseStatus,
			truncateOpenAIWSLogValue(dialCloseReason, openAIWSHeaderValueMaxLen),
			dialRespServer,
			dialRespVia,
			dialRespCFRay,
			dialRespReqID,
			truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen),
			truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen),
			forceNewConn,
			wsHost,
			wsPath,
			account.ProxyID != nil && account.Proxy != nil,
		)
		var dialErr *openAIWSDialError
		if errors.As(err, &dialErr) && dialErr != nil && dialErr.StatusCode == http.StatusTooManyRequests {
			s.persistOpenAIWSRateLimitSignal(ctx, account, dialErr.ResponseHeaders, nil, "rate_limit_exceeded", "rate_limit_error", strings.TrimSpace(err.Error()))
		}
		return nil, wrapOpenAIWSFallback(classifyOpenAIWSAcquireError(err), err)
	}
	// cleanExit 闁哄秴娲╅鍥ь潰閿濆懐鍩楃紓浣哥墢椤忣剚绂嶇€ｂ晜顐介梺顐熷亾闁告垹灏ㄧ槐婵嗩潰閵堝棙顦у☉鎾筹攻閻栬埖绋夊鍕獥闁告劕绉磋ぐ鍌炴焻娴ｅ憡濮庨柨娑樼焷缁绘盯骞掗妷銉ヨ閻庣懓顦崣蹇氥亹閹烘洜绠峰璺虹Ф閺併倝濡?	// 闁圭鍋撻柡鍫濐槸缁辨挾鏁宀€鐔呯€垫澘瀚哥槐娆戞嫚鐠囨彃鏅搁梺鎸庣懆椤曘倝濡存稊鐚簉or 濞存粌顑勫▎銏㈢驳婢舵稓绀嗙€瑰憡褰冨﹢顏堝触閸曨喖娈伴柛鎺戞閺侇喗绋夐鐔风倒闁告挸绉烽惃鐔兼偨?MarkBroken闁?	// 闁搞儳濮甸?defer 濞戞搩鍘艰ぐ褔妫侀埀顒佸緞閸曨厽鍊炴慨婵撶到閻栧爼鏌呴埀顒勫礄閻戞ɑ顦у☉?MarkBroken 闁告鍟胯ぐ鏌ュΥ?
	cleanExit := false
	defer func() {
		if !cleanExit {
			lease.MarkBroken()
		}
		lease.Release()
	}()
	connID := strings.TrimSpace(lease.ConnID())
	logOpenAIWSModeDebug(
		"connected account_id=%d account_type=%s transport=%s conn_id=%s conn_reused=%v conn_pick_ms=%d queue_wait_ms=%d has_previous_response_id=%v",
		account.ID,
		account.Type,
		normalizeOpenAIWSLogValue(string(decision.Transport)),
		connID,
		lease.Reused(),
		lease.ConnPickDuration().Milliseconds(),
		lease.QueueWaitDuration().Milliseconds(),
		previousResponseID != "",
	)
	if previousResponseID != "" {
		logOpenAIWSModeInfo(
			"continuation_probe account_id=%d account_type=%s conn_id=%s previous_response_id=%s previous_response_id_kind=%s preferred_conn_id=%s conn_reused=%v store_disabled=%v session_hash=%s header_session_id=%s header_conversation_id=%s session_id_source=%s conversation_id_source=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v",
			account.ID,
			account.Type,
			truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
			truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen),
			normalizeOpenAIWSLogValue(previousResponseIDKind),
			truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen),
			lease.Reused(),
			storeDisabled,
			truncateOpenAIWSLogValue(sessionHash, 12),
			openAIWSHeaderValueForLog(wsHeaders, "session_id"),
			openAIWSHeaderValueForLog(wsHeaders, "conversation_id"),
			normalizeOpenAIWSLogValue(sessionResolution.SessionSource),
			normalizeOpenAIWSLogValue(sessionResolution.ConversationSource),
			turnState != "",
			len(turnState),
			promptCacheKey != "",
		)
	}
	if c != nil {
		SetOpsLatencyMs(c, OpsOpenAIWSConnPickMsKey, lease.ConnPickDuration().Milliseconds())
		SetOpsLatencyMs(c, OpsOpenAIWSQueueWaitMsKey, lease.QueueWaitDuration().Milliseconds())
		c.Set(OpsOpenAIWSConnReusedKey, lease.Reused())
		if connID != "" {
			c.Set(OpsOpenAIWSConnIDKey, connID)
		}
	}

	handshakeTurnState := strings.TrimSpace(lease.HandshakeHeader(openAIWSTurnStateHeader))
	logOpenAIWSModeDebug(
		"handshake account_id=%d conn_id=%s has_turn_state=%v turn_state_len=%d",
		account.ID,
		connID,
		handshakeTurnState != "",
		len(handshakeTurnState),
	)
	if handshakeTurnState != "" {
		if stateStore != nil && sessionHash != "" {
			stateStore.BindSessionTurnState(groupID, sessionHash, handshakeTurnState, s.openAIWSSessionStickyTTL())
		}
		if c != nil {
			c.Header(http.CanonicalHeaderKey(openAIWSTurnStateHeader), handshakeTurnState)
		}
	}

	if err := s.performOpenAIWSGeneratePrewarm(
		ctx,
		lease,
		decision,
		payload,
		previousResponseID,
		reqBody,
		account,
		stateStore,
		groupID,
	); err != nil {
		return nil, err
	}

	if err := lease.WriteJSONWithContextTimeout(ctx, payload, s.openAIWSWriteTimeout()); err != nil {
		lease.MarkBroken()
		logOpenAIWSModeInfo(
			"write_request_fail account_id=%d conn_id=%s cause=%s payload_bytes=%d",
			account.ID,
			connID,
			truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen),
			resolvePayloadBytes(),
		)
		return nil, wrapOpenAIWSFallback("write_request", err)
	}
	if debugEnabled {
		logOpenAIWSModeDebug(
			"write_request_sent account_id=%d conn_id=%s stream=%v payload_bytes=%d previous_response_id=%s",
			account.ID,
			connID,
			reqStream,
			resolvePayloadBytes(),
			truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen),
		)
	}

	usage := &OpenAIUsage{}
	imageCounter := newOpenAIImageOutputCounter()
	var firstTokenMs *int
	responseID := ""
	var finalResponse []byte
	wroteDownstream := false
	needModelReplace := originalModel != mappedModel
	var mappedModelBytes []byte
	if needModelReplace && mappedModel != "" {
		mappedModelBytes = []byte(mappedModel)
	}
	bufferedStreamEvents := make([][]byte, 0, 4)
	eventCount := 0
	tokenEventCount := 0
	terminalEventCount := 0
	bufferedEventCount := 0
	flushedBufferedEventCount := 0
	firstEventType := ""
	lastEventType := ""

	var flusher http.Flusher
	if reqStream {
		if s.responseHeaderFilter != nil {
			responseheaders.WriteFilteredHeaders(c.Writer.Header(), http.Header{}, s.responseHeaderFilter)
		}
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		f, ok := c.Writer.(http.Flusher)
		if !ok {
			lease.MarkBroken()
			return nil, wrapOpenAIWSFallback("streaming_not_supported", errors.New("streaming not supported"))
		}
		flusher = f
	}

	clientDisconnected := false
	flushBatchSize := s.openAIWSEventFlushBatchSize()
	flushInterval := s.openAIWSEventFlushInterval()
	pendingFlushEvents := 0
	lastFlushAt := time.Now()
	flushStreamWriter := func(force bool) {
		if clientDisconnected || flusher == nil || pendingFlushEvents <= 0 {
			return
		}
		if !force && flushBatchSize > 1 && pendingFlushEvents < flushBatchSize {
			if flushInterval <= 0 || time.Since(lastFlushAt) < flushInterval {
				return
			}
		}
		flusher.Flush()
		pendingFlushEvents = 0
		lastFlushAt = time.Now()
	}
	emitStreamMessage := func(message []byte, forceFlush bool) {
		if clientDisconnected {
			return
		}
		frame := make([]byte, 0, len(message)+8)
		frame = append(frame, "data: "...)
		frame = append(frame, message...)
		frame = append(frame, '\n', '\n')
		_, wErr := c.Writer.Write(frame)
		if wErr == nil {
			wroteDownstream = true
			pendingFlushEvents++
			flushStreamWriter(forceFlush)
			return
		}
		clientDisconnected = true
		logger.LegacyPrintf("service.openai_gateway", "[OpenAI WS Mode] client disconnected, continue draining upstream: account=%d", account.ID)
	}
	flushBufferedStreamEvents := func(reason string) {
		if len(bufferedStreamEvents) == 0 {
			return
		}
		flushed := len(bufferedStreamEvents)
		for _, buffered := range bufferedStreamEvents {
			emitStreamMessage(buffered, false)
		}
		bufferedStreamEvents = bufferedStreamEvents[:0]
		flushStreamWriter(true)
		flushedBufferedEventCount += flushed
		if debugEnabled {
			logOpenAIWSModeDebug(
				"buffer_flush account_id=%d conn_id=%s reason=%s flushed=%d total_flushed=%d client_disconnected=%v",
				account.ID,
				connID,
				truncateOpenAIWSLogValue(reason, openAIWSLogValueMaxLen),
				flushed,
				flushedBufferedEventCount,
				clientDisconnected,
			)
		}
	}

	readTimeout := s.openAIWSReadTimeout()

	for {
		message, readErr := lease.ReadMessageWithContextTimeout(ctx, readTimeout)
		if readErr != nil {
			lease.MarkBroken()
			closeStatus, closeReason := summarizeOpenAIWSReadCloseError(readErr)
			logOpenAIWSModeInfo(
				"read_fail account_id=%d conn_id=%s wrote_downstream=%v close_status=%s close_reason=%s cause=%s events=%d token_events=%d terminal_events=%d buffered_pending=%d buffered_flushed=%d first_event=%s last_event=%s",
				account.ID,
				connID,
				wroteDownstream,
				closeStatus,
				closeReason,
				truncateOpenAIWSLogValue(readErr.Error(), openAIWSLogValueMaxLen),
				eventCount,
				tokenEventCount,
				terminalEventCount,
				len(bufferedStreamEvents),
				flushedBufferedEventCount,
				truncateOpenAIWSLogValue(firstEventType, openAIWSLogValueMaxLen),
				truncateOpenAIWSLogValue(lastEventType, openAIWSLogValueMaxLen),
			)
			if !wroteDownstream {
				return nil, wrapOpenAIWSFallback(classifyOpenAIWSReadFallbackReason(readErr), readErr)
			}
			if clientDisconnected {
				break
			}
			setOpsUpstreamError(c, 0, sanitizeUpstreamErrorMessage(readErr.Error()), "")
			return nil, fmt.Errorf("openai ws read event: %w", readErr)
		}

		eventType, eventResponseID, responseField := parseOpenAIWSEventEnvelope(message)
		if eventType == "" {
			continue
		}
		eventCount++
		if firstEventType == "" {
			firstEventType = eventType
		}
		lastEventType = eventType

		if responseID == "" && eventResponseID != "" {
			responseID = eventResponseID
		}

		isTokenEvent := isOpenAIWSTokenEvent(eventType)
		if isTokenEvent {
			tokenEventCount++
		}
		isTerminalEvent := isOpenAIWSTerminalEvent(eventType)
		if isTerminalEvent {
			terminalEventCount++
		}
		if firstTokenMs == nil && isTokenEvent {
			ms := int(time.Since(startTime).Milliseconds())
			firstTokenMs = &ms
		}
		if debugEnabled && shouldLogOpenAIWSEvent(eventCount, eventType) {
			logOpenAIWSModeDebug(
				"event_received account_id=%d conn_id=%s idx=%d type=%s bytes=%d token=%v terminal=%v buffered_pending=%d",
				account.ID,
				connID,
				eventCount,
				truncateOpenAIWSLogValue(eventType, openAIWSLogValueMaxLen),
				len(message),
				isTokenEvent,
				isTerminalEvent,
				len(bufferedStreamEvents),
			)
		}

		if !clientDisconnected {
			if needModelReplace && len(mappedModelBytes) > 0 && openAIWSEventMayContainModel(eventType) && bytes.Contains(message, mappedModelBytes) {
				message = replaceOpenAIWSMessageModel(message, mappedModel, originalModel)
			}
			if openAIWSEventMayContainToolCalls(eventType) && openAIWSMessageLikelyContainsToolCalls(message) {
				if corrected, changed := s.toolCorrector.CorrectToolCallsInSSEBytes(message); changed {
					message = corrected
				}
			}
		}
		if openAIWSEventShouldParseUsage(eventType) {
			parseOpenAIWSResponseUsageFromCompletedEvent(message, usage)
		}
		imageCounter.AddSSEData(message)

		if eventType == "error" {
			errCodeRaw, errTypeRaw, errMsgRaw := parseOpenAIWSErrorEventFields(message)
			s.persistOpenAIWSRateLimitSignal(ctx, account, lease.HandshakeHeaders(), message, errCodeRaw, errTypeRaw, errMsgRaw)
			errMsg := strings.TrimSpace(errMsgRaw)
			if errMsg == "" {
				errMsg = "Upstream websocket error"
			}
			fallbackReason, canFallback := classifyOpenAIWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			errCode, errType, errMessage := summarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			logOpenAIWSModeInfo(
				"error_event account_id=%d conn_id=%s idx=%d fallback_reason=%s can_fallback=%v err_code=%s err_type=%s err_message=%s",
				account.ID,
				connID,
				eventCount,
				truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen),
				canFallback,
				errCode,
				errType,
				errMessage,
			)
			if fallbackReason == "previous_response_not_found" {
				logOpenAIWSModeInfo(
					"previous_response_not_found_diag account_id=%d account_type=%s conn_id=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s event_idx=%d req_stream=%v store_disabled=%v conn_reused=%v session_hash=%s header_session_id=%s header_conversation_id=%s session_id_source=%s conversation_id_source=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v err_code=%s err_type=%s err_message=%s",
					account.ID,
					account.Type,
					connID,
					truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen),
					normalizeOpenAIWSLogValue(previousResponseIDKind),
					truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen),
					eventCount,
					reqStream,
					storeDisabled,
					lease.Reused(),
					truncateOpenAIWSLogValue(sessionHash, 12),
					openAIWSHeaderValueForLog(wsHeaders, "session_id"),
					openAIWSHeaderValueForLog(wsHeaders, "conversation_id"),
					normalizeOpenAIWSLogValue(sessionResolution.SessionSource),
					normalizeOpenAIWSLogValue(sessionResolution.ConversationSource),
					turnState != "",
					len(turnState),
					promptCacheKey != "",
					errCode,
					errType,
					errMessage,
				)
			}
			// error 濞存粌顑勫▎銏ゅ触鎼淬倗绠鹃柟鎭掑劙缁楀宕樺鍛濠㈣泛绉堕弫銈夋晬瀹€鍕級闁稿繐绉村ú鏍ч悩鍙夊€垫慨鍏夊墲閻撳绋夌€ｂ晝顏遍悹鍥敱閻即濡?			lease.MarkBroken()
			if !wroteDownstream && canFallback {
				return nil, wrapOpenAIWSFallback(fallbackReason, errors.New(errMsg))
			}
			statusCode := openAIWSErrorHTTPStatusFromRaw(errCodeRaw, errTypeRaw)
			setOpsUpstreamError(c, statusCode, errMsg, "")
			if reqStream && !clientDisconnected {
				flushBufferedStreamEvents("error_event")
				emitStreamMessage(message, true)
			}
			if !reqStream {
				c.JSON(statusCode, gin.H{
					"error": gin.H{
						"type":    "upstream_error",
						"message": errMsg,
					},
				})
			}
			return nil, fmt.Errorf("openai ws error event: %s", errMsg)
		}

		if reqStream {
			// 闁革负鍔戦々缁樼▔?token 闁告挸绉撮崢娑氱磽閹惧啿鏆卞ù婊冾儎濞嗐垽鏁嶉崼婵愭搐 response.created闁挎稑顧€缁?			// 濞寸姰鍎扮粚鑸电▔婵犲啰鍩楅柡鍐ｆ櫆濠€锟犲棘椤擄紕绠鹃柡鍐╂构缁盯宕ｉ姘辨殧闁稿繈鍔屽ú鏍焻閳ь剟宕?HTTP闁挎稑濂旂粭澶岀磼濞嗗海鐟撴繛鎾额焾瑜板倿鏌呮担绋跨９闁规惌浜濈粊锕傚Υ?
			shouldBuffer := firstTokenMs == nil && !isTokenEvent && !isTerminalEvent
			if shouldBuffer {
				buffered := make([]byte, len(message))
				copy(buffered, message)
				bufferedStreamEvents = append(bufferedStreamEvents, buffered)
				bufferedEventCount++
				if debugEnabled && shouldLogOpenAIWSBufferedEvent(bufferedEventCount) {
					logOpenAIWSModeDebug(
						"buffer_enqueue account_id=%d conn_id=%s idx=%d event_idx=%d event_type=%s buffer_size=%d",
						account.ID,
						connID,
						bufferedEventCount,
						eventCount,
						truncateOpenAIWSLogValue(eventType, openAIWSLogValueMaxLen),
						len(bufferedStreamEvents),
					)
				}
			} else {
				flushBufferedStreamEvents(eventType)
				emitStreamMessage(message, isTerminalEvent)
			}
		} else {
			if responseField.Exists() && responseField.Type == gjson.JSON {
				finalResponse = []byte(responseField.Raw)
			}
		}

		if isTerminalEvent {
			cleanExit = true
			break
		}
	}

	if !reqStream {
		if len(finalResponse) == 0 {
			logOpenAIWSModeInfo(
				"missing_final_response account_id=%d conn_id=%s events=%d token_events=%d terminal_events=%d wrote_downstream=%v",
				account.ID,
				connID,
				eventCount,
				tokenEventCount,
				terminalEventCount,
				wroteDownstream,
			)
			if !wroteDownstream {
				return nil, wrapOpenAIWSFallback("missing_final_response", errors.New("no terminal response payload"))
			}
			return nil, errors.New("ws finished without final response")
		}

		if needModelReplace {
			finalResponse = s.replaceModelInResponseBody(finalResponse, mappedModel, originalModel)
		}
		finalResponse = s.correctToolCallsInResponseBody(finalResponse)
		populateOpenAIUsageFromResponseJSON(finalResponse, usage)
		if responseID == "" {
			responseID = strings.TrimSpace(gjson.GetBytes(finalResponse, "id").String())
		}

		c.Data(http.StatusOK, "application/json", finalResponse)
	} else {
		flushStreamWriter(true)
	}

	if responseID != "" && stateStore != nil {
		ttl := s.openAIWSResponseStickyTTL()
		logOpenAIWSBindResponseAccountWarn(groupID, account.ID, responseID, stateStore.BindResponseAccount(ctx, groupID, responseID, account.ID, ttl))
		stateStore.BindResponseConn(responseID, lease.ConnID(), ttl)
	}
	if stateStore != nil && storeDisabled && sessionHash != "" {
		stateStore.BindSessionConn(groupID, sessionHash, lease.ConnID(), s.openAIWSSessionStickyTTL())
	}
	firstTokenMsValue := -1
	if firstTokenMs != nil {
		firstTokenMsValue = *firstTokenMs
	}
	logOpenAIWSModeDebug(
		"completed account_id=%d conn_id=%s response_id=%s stream=%v duration_ms=%d events=%d token_events=%d terminal_events=%d buffered_events=%d buffered_flushed=%d first_event=%s last_event=%s first_token_ms=%d wrote_downstream=%v client_disconnected=%v",
		account.ID,
		connID,
		truncateOpenAIWSLogValue(strings.TrimSpace(responseID), openAIWSIDValueMaxLen),
		reqStream,
		time.Since(startTime).Milliseconds(),
		eventCount,
		tokenEventCount,
		terminalEventCount,
		bufferedEventCount,
		flushedBufferedEventCount,
		truncateOpenAIWSLogValue(firstEventType, openAIWSLogValueMaxLen),
		truncateOpenAIWSLogValue(lastEventType, openAIWSLogValueMaxLen),
		firstTokenMsValue,
		wroteDownstream,
		clientDisconnected,
	)

	return &OpenAIForwardResult{
		RequestID:        responseID,
		Usage:            *usage,
		Model:            originalModel,
		UpstreamModel:    mappedModel,
		ImageCount:       imageCounter.Count(),
		ImageOutputSizes: imageCounter.Sizes(),
		ServiceTier:      extractOpenAIServiceTier(reqBody),
		ReasoningEffort:  extractOpenAIReasoningEffort(reqBody, originalModel),
		Stream:           reqStream,
		OpenAIWSMode:     true,
		ResponseHeaders:  lease.HandshakeHeaders(),
		Duration:         time.Since(startTime),
		FirstTokenMs:     firstTokenMs,
	}, nil
}

// ProxyResponsesWebSocketFromClient 濠㈣泛瀚幃濠勨偓骞垮灪閸╂稓绮╅姘汲缂?WebSocket闁挎稑婀enAI Responses WS Mode闁挎稑顦懟鐔告姜椤掆偓瑜板倿宕氭０浣虹憪婵炴挻鎮堕埀?// 鐟滅増鎸告晶鐘碘偓鍦仧楠炲洭骞愭径鍡忓亾濠婂啫绀嬮悹鍥敱閻?-> 缂備礁鐗婇娑欑鐎ｂ晜顐?-> 濞戞挸顑勭粩瀵告嫚闁垮婀撮柍銉︾箘濞堟垶銇勯崫鍕濞寸媴绲块幃濠囨晬瀹€鍕ㄥ亾閸岀偛甯?Codex CLI 闁?turn 婵☆垪鈧磭纭€闁?
func (s *OpenAIGatewayService) ProxyResponsesWebSocketFromClient(
	ctx context.Context,
	c *gin.Context,
	clientConn *coderws.Conn,
	account *Account,
	token string,
	firstClientMessage []byte,
	hooks *OpenAIWSIngressHooks,
) error {
	if s == nil {
		return errors.New("service is nil")
	}
	if c == nil {
		return errors.New("gin context is nil")
	}
	if clientConn == nil {
		return errors.New("client websocket is nil")
	}
	if account == nil {
		return errors.New("account is nil")
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("token is empty")
	}

	// 濡澘瀚ぐ鍥ㄧ▔閳ь剙鈻?OpenAI Fast Policy settings闁挎稑鐬肩划锔锯偓瑙勮壘閸?ctx闁挎稑鐭侀鈧悹?WS session
	// 闁告劕鎳忔晶宥夊嫉婢跺﹥濮庨柣?evaluateOpenAIFastPolicy 閻犲鍟伴弫銈嗗緞瀹ュ洦鏆忛柛姘缁斿瓨绂掗挊澶嬪渐闁绘挆宥囩闂侇剙鐏濋崢銈呅掕箛鎾村
	// 閺夆晜绋戦崣?DB / settingRepo闁靛棔绻俽ade-off 閻?withOpenAIFastPolicyContext 婵炲鍔戦崳鎾Υ?
	if s.settingService != nil {
		if settings, err := s.settingService.GetOpenAIFastPolicySettings(ctx); err == nil && settings != nil {
			ctx = withOpenAIFastPolicyContext(ctx, settings)
		}
	}

	wsDecision := s.getOpenAIWSProtocolResolver().Resolve(account)
	modeRouterV2Enabled := s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.ModeRouterV2Enabled
	ingressMode := OpenAIWSIngressModeCtxPool
	if modeRouterV2Enabled {
		ingressMode = account.ResolveOpenAIResponsesWebSocketV2Mode(s.cfg.Gateway.OpenAIWS.IngressModeDefault)
		if ingressMode == OpenAIWSIngressModeOff {
			return NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"websocket mode is disabled for this account",
				nil,
			)
		}
		switch ingressMode {
		case OpenAIWSIngressModePassthrough:
			if wsDecision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
				return fmt.Errorf("websocket ingress requires ws_v2 transport, got=%s", wsDecision.Transport)
			}
			return s.proxyResponsesWebSocketV2Passthrough(
				ctx,
				c,
				clientConn,
				account,
				token,
				firstClientMessage,
				hooks,
				wsDecision,
			)
		case OpenAIWSIngressModeCtxPool, OpenAIWSIngressModeShared, OpenAIWSIngressModeDedicated:
			// continue
		default:
			return NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"websocket mode only supports ctx_pool/passthrough",
				nil,
			)
		}
	}
	if wsDecision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
		return fmt.Errorf("websocket ingress requires ws_v2 transport, got=%s", wsDecision.Transport)
	}
	dedicatedMode := modeRouterV2Enabled && ingressMode == OpenAIWSIngressModeDedicated

	wsURL, err := s.buildOpenAIResponsesWSURL(account)
	if err != nil {
		return fmt.Errorf("build ws url: %w", err)
	}
	wsHost := "-"
	wsPath := "-"
	if parsedURL, parseErr := url.Parse(wsURL); parseErr == nil && parsedURL != nil {
		wsHost = normalizeOpenAIWSLogValue(parsedURL.Host)
		wsPath = normalizeOpenAIWSLogValue(parsedURL.Path)
	}
	debugEnabled := isOpenAIWSModeDebugEnabled()

	type openAIWSClientPayload struct {
		payloadRaw         []byte
		rawForHash         []byte
		promptCacheKey     string
		previousResponseID string
		originalModel      string
		imageBillingModel  string
		imageSizeTier      string
		imageInputSize     string
		payloadBytes       int
	}
	ingressSessionOriginalModel := ""

	applyPayloadMutation := func(current []byte, path string, value any) ([]byte, error) {
		next, err := sjson.SetBytes(current, path, value)
		if err == nil {
			return next, nil
		}

		// 濞寸姴鎳庡﹢顏嗘兜椤旇偐鏉介梻鍥ｅ亾閻熸洑妞掗幈銊╁绩?payload 濞?sjson 濠㈡儼绮剧憴锕傚籍鐠佸湱绀夐梺顐熷亾闁?map 閻犱警鍨扮欢鐐垫兜椤旇崵绠介柛蹇曞帶椤旀劙骞€瑜嬮埀?
		payload := make(map[string]any)
		if unmarshalErr := json.Unmarshal(current, &payload); unmarshalErr != nil {
			return nil, err
		}
		switch path {
		case "type", "model":
			payload[path] = value
		case "client_metadata." + openAIWSTurnMetadataHeader:
			setOpenAIWSTurnMetadata(payload, fmt.Sprintf("%v", value))
		default:
			return nil, err
		}
		rebuilt, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return rebuilt, nil
	}

	parseClientPayload := func(raw []byte) (openAIWSClientPayload, error) {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "empty websocket request payload", nil)
		}
		if !gjson.ValidBytes(trimmed) {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", errors.New("invalid json"))
		}

		values := gjson.GetManyBytes(trimmed, "type", "model", "prompt_cache_key", "previous_response_id")
		eventType := strings.TrimSpace(values[0].String())
		normalized := trimmed
		switch eventType {
		case "":
			eventType = "response.create"
			next, setErr := applyPayloadMutation(normalized, "type", eventType)
			if setErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", setErr)
			}
			normalized = next
		case "response.create":
		case "response.append":
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"response.append is not supported in ws v2; use response.create with previous_response_id",
				nil,
			)
		default:
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				fmt.Sprintf("unsupported websocket request type: %s", eventType),
				nil,
			)
		}

		originalModel := strings.TrimSpace(values[1].String())
		modelMissing := originalModel == ""
		if originalModel == "" {
			// 闁稿繈鍎抽悵?WS 闂傗偓婢跺摜绐楅悹鍥ㄧ箞閸ｇ兘鏁嶅畝鍕姤闁告帒妫楅褰掑箣妞嬪寒浼傞柛娆樹簻濠€顏嗙箔椤戣法顏遍弶?response.create 濞戞挸锕ら敍鎰板及?			// model闁挎稑鑻幃妤冪磼?turn 濠㈣泛绉堕弫銈夊触鐏炶偐顏?session-level model闁靛棗鍊风拹鐔兼焼閸喖甯抽柛銉уХ濞撶兘鎮?			// model 闁烩晛鐡ㄧ敮鎾棘椤撶偟纾婚柣顫妽閸╂稒娼婚悙鏉戝闁挎稑鐭佺换鏍煂鐏炶姤绀€闁解偓閽樺鐓傚☉鎾筹梗缁斿瓨娼鐓庡殥闂侇偅淇虹换鍐冀閿熺姷宕ｉ柣銊ュ椤撳綊骞嬫搴紓婵☆垪鈧磭鈧兘鏁?			// 妤犵偠娉涘﹢顏呯▔鐎ｎ偅鐓欓柛鎰懃濞叉牗绋夋繝鍐煑 payload闁挎稑濂旂换姘辨嫚娴ｈ棄顦╅柛娆撴敱鑶╅柛銊ヮ儐濡惭呬焊?fast policy/闁搞儱澧芥晶鏍级閸愵喗顎?
			// 濞寸姴绉电€垫粓宕ョ仦鑲╊伇婵☆垪鈧磭鈧兘骞嶈椤㈡垿濡?			originalModel = ingressSessionOriginalModel
			originalModel = ingressSessionOriginalModel
			if originalModel == "" {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(
					coderws.StatusPolicyViolation,
					"model is required in response.create payload",
					nil,
				)
			}
		}
		promptCacheKey := strings.TrimSpace(values[2].String())
		previousResponseID := strings.TrimSpace(values[3].String())
		previousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
		if previousResponseID != "" && previousResponseIDKind == OpenAIPreviousResponseIDKindMessageID {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"previous_response_id must be a response.id (resp_*), not a message id",
				nil,
			)
		}
		if turnMetadata := strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)); turnMetadata != "" {
			next, setErr := applyPayloadMutation(normalized, "client_metadata."+openAIWSTurnMetadataHeader, turnMetadata)
			if setErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", setErr)
			}
			normalized = next
		}
		upstreamModel := normalizeOpenAIModelForUpstream(account, account.GetMappedModel(originalModel))
		if modelMissing || upstreamModel != originalModel {
			next, setErr := applyPayloadMutation(normalized, "model", upstreamModel)
			if setErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", setErr)
			}
			normalized = next
		}
		imageIntent := IsImageGenerationIntent(openAIResponsesEndpoint, originalModel, normalized)
		if imageIntent && !GroupAllowsImageGeneration(apiKeyGroup(getAPIKeyFromContext(c))) {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, ImageGenerationPermissionMessage(), nil)
		}
		imageBillingModel := ""
		imageSizeTier := ""
		imageInputSize := ""
		if imageIntent {
			var imageCfgErr error
			imageCfg, imageCfgErr := resolveOpenAIResponsesImageBillingConfigDetailedFromBody(normalized, originalModel)
			if imageCfgErr != nil {
				return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, imageCfgErr.Error(), imageCfgErr)
			}
			imageBillingModel = imageCfg.Model
			imageSizeTier = imageCfg.SizeTier
			imageInputSize = imageCfg.InputSize
		}

		// Apply OpenAI Fast Policy on the response.create frame using the same
		// evaluator/normalize/scope rules as the HTTP entrypoints. This is the
		// single integration point for all WS ingress turns (first + follow-up
		// frames flow through here).
		//
		// Model fallback: first turn still requires model at the handler layer闁?		// follow-up response.create frames may omit it and then reuse
		// ingressSessionOriginalModel. We always write a concrete upstream model
		// before evaluating policy, so whitelist / filter behavior remains stable.
		policyApplied, blocked, policyErr := s.applyOpenAIFastPolicyToWSResponseCreate(ctx, account, upstreamModel, normalized)
		if policyErr != nil {
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "invalid websocket request payload", policyErr)
		}
		if blocked != nil {
			MarkOpsClientBusinessLimited(c, OpsClientBusinessLimitedReasonLocalPolicyDenied)
			// Send a Realtime-style error event to the client first, then
			// signal the handler to close the connection with PolicyViolation.
			// We intentionally do NOT forward this frame upstream.
			//
			// coder/websocket@v1.8.14 Conn.Write is synchronous and flushes
			// the underlying bufio writer before returning (write.go:42 闁?			// 307-311), and the subsequent close handshake re-acquires the
			// same writeFrameMu, so the error event is guaranteed to reach
			// the kernel send buffer before any close frame is queued.
			eventBytes := buildOpenAIFastPolicyBlockedWSEvent(blocked)
			if eventBytes != nil {
				writeCtx, cancel := context.WithTimeout(ctx, s.openAIWSWriteTimeout())
				_ = clientConn.Write(writeCtx, coderws.MessageText, eventBytes)
				cancel()
			}
			return openAIWSClientPayload{}, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				blocked.Message,
				blocked,
			)
		}
		normalized = policyApplied
		ingressSessionOriginalModel = originalModel

		return openAIWSClientPayload{
			payloadRaw:         normalized,
			rawForHash:         trimmed,
			promptCacheKey:     promptCacheKey,
			previousResponseID: previousResponseID,
			originalModel:      originalModel,
			imageBillingModel:  imageBillingModel,
			imageSizeTier:      imageSizeTier,
			imageInputSize:     imageInputSize,
			payloadBytes:       len(normalized),
		}, nil
	}

	firstPayload, err := parseClientPayload(firstClientMessage)
	if err != nil {
		return err
	}

	turnState := strings.TrimSpace(c.GetHeader(openAIWSTurnStateHeader))
	stateStore := s.getOpenAIWSStateStore()
	groupID := getOpenAIGroupIDFromContext(c)
	sessionHash := s.GenerateSessionHash(c, firstPayload.rawForHash)
	if turnState == "" && stateStore != nil && sessionHash != "" {
		if savedTurnState, ok := stateStore.GetSessionTurnState(groupID, sessionHash); ok {
			turnState = savedTurnState
		}
	}

	preferredConnID := ""
	if stateStore != nil && firstPayload.previousResponseID != "" {
		if connID, ok := stateStore.GetResponseConn(firstPayload.previousResponseID); ok {
			preferredConnID = connID
		}
	}

	storeDisabled := s.isOpenAIWSStoreDisabledInRequestRaw(firstPayload.payloadRaw, account)
	storeDisabledConnMode := s.openAIWSStoreDisabledConnMode()
	if stateStore != nil && storeDisabled && firstPayload.previousResponseID == "" && sessionHash != "" {
		if connID, ok := stateStore.GetSessionConn(groupID, sessionHash); ok {
			preferredConnID = connID
		}
	}

	isCodexCLI := openai.IsCodexOfficialClientByHeaders(c.GetHeader("User-Agent"), c.GetHeader("originator")) || (s.cfg != nil && s.cfg.Gateway.ForceCodexCLI)
	wsHeaders, _ := s.buildOpenAIWSHeaders(c, account, token, wsDecision, isCodexCLI, turnState, strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)), firstPayload.promptCacheKey)
	baseAcquireReq := openAIWSAcquireRequest{
		Account: account,
		WSURL:   wsURL,
		Headers: wsHeaders,
		ProxyURL: func() string {
			if account.ProxyID != nil && account.Proxy != nil {
				return account.Proxy.URL()
			}
			return ""
		}(),
		ForceNewConn: false,
	}
	pool := s.getOpenAIWSConnPool()
	if pool == nil {
		return errors.New("openai ws conn pool is nil")
	}

	logOpenAIWSModeInfo(
		"ingress_ws_protocol_confirm account_id=%d account_type=%s transport=%s ws_host=%s ws_path=%s ws_mode=%s store_disabled=%v has_session_hash=%v has_previous_response_id=%v",
		account.ID,
		account.Type,
		normalizeOpenAIWSLogValue(string(wsDecision.Transport)),
		wsHost,
		wsPath,
		normalizeOpenAIWSLogValue(ingressMode),
		storeDisabled,
		sessionHash != "",
		firstPayload.previousResponseID != "",
	)

	if debugEnabled {
		logOpenAIWSModeDebug(
			"ingress_ws_start account_id=%d account_type=%s transport=%s ws_host=%s preferred_conn_id=%s has_session_hash=%v has_previous_response_id=%v store_disabled=%v",
			account.ID,
			account.Type,
			normalizeOpenAIWSLogValue(string(wsDecision.Transport)),
			wsHost,
			truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen),
			sessionHash != "",
			firstPayload.previousResponseID != "",
			storeDisabled,
		)
	}
	if firstPayload.previousResponseID != "" {
		firstPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(firstPayload.previousResponseID)
		logOpenAIWSModeInfo(
			"ingress_ws_continuation_probe account_id=%d turn=%d previous_response_id=%s previous_response_id_kind=%s preferred_conn_id=%s session_hash=%s header_session_id=%s header_conversation_id=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v store_disabled=%v",
			account.ID,
			1,
			truncateOpenAIWSLogValue(firstPayload.previousResponseID, openAIWSIDValueMaxLen),
			normalizeOpenAIWSLogValue(firstPreviousResponseIDKind),
			truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen),
			truncateOpenAIWSLogValue(sessionHash, 12),
			openAIWSHeaderValueForLog(baseAcquireReq.Headers, "session_id"),
			openAIWSHeaderValueForLog(baseAcquireReq.Headers, "conversation_id"),
			turnState != "",
			len(turnState),
			firstPayload.promptCacheKey != "",
			storeDisabled,
		)
	}

	acquireTimeout := s.openAIWSAcquireTimeout()
	if acquireTimeout <= 0 {
		acquireTimeout = 30 * time.Second
	}

	acquireTurnLease := func(turn int, preferred string, forcePreferredConn bool) (*openAIWSConnLease, error) {
		req := cloneOpenAIWSAcquireRequest(baseAcquireReq)
		req.PreferredConnID = strings.TrimSpace(preferred)
		req.ForcePreferredConn = forcePreferredConn
		// dedicated 婵☆垪鈧磭纭€濞戞挸顑嗛惁鈥斥枎闄囬獮蹇涘矗閺嵮勭秵闁哄倹婢樼紓鎾存交閻愭潙澶嶉柨娑樼焸娴尖晠宕楀鍫熺《濞村吋淇洪惁鑺ュ緞瀹ュ洦鏆忔繛鍫濐儑閺嗏偓濞戞挸锕ｇ粭鍛村棘閸ャ儮鍋?		req.ForceNewConn = dedicatedMode
		req.ForceNewConn = dedicatedMode
		acquireCtx, acquireCancel := context.WithTimeout(ctx, acquireTimeout)
		lease, acquireErr := pool.Acquire(acquireCtx, req)
		acquireCancel()
		if acquireErr != nil {
			dialStatus, dialClass, dialCloseStatus, dialCloseReason, dialRespServer, dialRespVia, dialRespCFRay, dialRespReqID := summarizeOpenAIWSDialError(acquireErr)
			logOpenAIWSModeInfo(
				"ingress_ws_upstream_acquire_fail account_id=%d turn=%d reason=%s dial_status=%d dial_class=%s dial_close_status=%s dial_close_reason=%s dial_resp_server=%s dial_resp_via=%s dial_resp_cf_ray=%s dial_resp_x_request_id=%s cause=%s preferred_conn_id=%s force_preferred_conn=%v ws_host=%s ws_path=%s proxy_enabled=%v",
				account.ID,
				turn,
				normalizeOpenAIWSLogValue(classifyOpenAIWSAcquireError(acquireErr)),
				dialStatus,
				dialClass,
				dialCloseStatus,
				truncateOpenAIWSLogValue(dialCloseReason, openAIWSHeaderValueMaxLen),
				dialRespServer,
				dialRespVia,
				dialRespCFRay,
				dialRespReqID,
				truncateOpenAIWSLogValue(acquireErr.Error(), openAIWSLogValueMaxLen),
				truncateOpenAIWSLogValue(preferred, openAIWSIDValueMaxLen),
				forcePreferredConn,
				wsHost,
				wsPath,
				account.ProxyID != nil && account.Proxy != nil,
			)
			var dialErr *openAIWSDialError
			if errors.As(acquireErr, &dialErr) && dialErr != nil && dialErr.StatusCode == http.StatusTooManyRequests {
				s.persistOpenAIWSRateLimitSignal(ctx, account, dialErr.ResponseHeaders, nil, "rate_limit_exceeded", "rate_limit_error", strings.TrimSpace(acquireErr.Error()))
				return nil, &UpstreamFailoverError{
					StatusCode:      http.StatusTooManyRequests,
					ResponseHeaders: cloneHeader(dialErr.ResponseHeaders),
				}
			}
			if errors.Is(acquireErr, errOpenAIWSPreferredConnUnavailable) {
				return nil, NewOpenAIWSClientCloseError(
					coderws.StatusPolicyViolation,
					"upstream continuation connection is unavailable; please restart the conversation",
					acquireErr,
				)
			}
			if errors.Is(acquireErr, context.DeadlineExceeded) || errors.Is(acquireErr, errOpenAIWSConnQueueFull) {
				return nil, NewOpenAIWSClientCloseError(
					coderws.StatusTryAgainLater,
					"upstream websocket is busy, please retry later",
					acquireErr,
				)
			}
			return nil, acquireErr
		}
		connID := strings.TrimSpace(lease.ConnID())
		if handshakeTurnState := strings.TrimSpace(lease.HandshakeHeader(openAIWSTurnStateHeader)); handshakeTurnState != "" {
			turnState = handshakeTurnState
			if stateStore != nil && sessionHash != "" {
				stateStore.BindSessionTurnState(groupID, sessionHash, handshakeTurnState, s.openAIWSSessionStickyTTL())
			}
			updatedHeaders := cloneHeader(baseAcquireReq.Headers)
			if updatedHeaders == nil {
				updatedHeaders = make(http.Header)
			}
			updatedHeaders.Set(openAIWSTurnStateHeader, handshakeTurnState)
			baseAcquireReq.Headers = updatedHeaders
		}
		logOpenAIWSModeInfo(
			"ingress_ws_upstream_connected account_id=%d turn=%d conn_id=%s conn_reused=%v conn_pick_ms=%d queue_wait_ms=%d preferred_conn_id=%s",
			account.ID,
			turn,
			truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
			lease.Reused(),
			lease.ConnPickDuration().Milliseconds(),
			lease.QueueWaitDuration().Milliseconds(),
			truncateOpenAIWSLogValue(preferred, openAIWSIDValueMaxLen),
		)
		return lease, nil
	}

	writeClientMessage := func(message []byte) error {
		writeCtx, cancel := context.WithTimeout(ctx, s.openAIWSWriteTimeout())
		defer cancel()
		return clientConn.Write(writeCtx, coderws.MessageText, message)
	}

	readClientMessage := func() ([]byte, error) {
		msgType, payload, readErr := clientConn.Read(ctx)
		if readErr != nil {
			return nil, readErr
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			return nil, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				fmt.Sprintf("unsupported websocket client message type: %s", msgType.String()),
				nil,
			)
		}
		return payload, nil
	}

	sendAndRelay := func(turn int, lease *openAIWSConnLease, payload []byte, payloadBytes int, originalModel string, imageBillingModel string, imageSizeTier string, imageInputSize string) (*OpenAIForwardResult, error) {
		if lease == nil {
			return nil, errors.New("upstream websocket lease is nil")
		}
		turnStart := time.Now()
		wroteDownstream := false
		if err := lease.WriteJSONWithContextTimeout(ctx, json.RawMessage(payload), s.openAIWSWriteTimeout()); err != nil {
			return nil, wrapOpenAIWSIngressTurnError(
				"write_upstream",
				fmt.Errorf("write upstream websocket request: %w", err),
				false,
			)
		}
		if debugEnabled {
			logOpenAIWSModeDebug(
				"ingress_ws_turn_request_sent account_id=%d turn=%d conn_id=%s payload_bytes=%d",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen),
				payloadBytes,
			)
		}

		responseID := ""
		usage := OpenAIUsage{}
		imageCounter := newOpenAIImageOutputCounter()
		var firstTokenMs *int
		reqStream := openAIWSPayloadBoolFromRaw(payload, "stream", true)
		turnPreviousResponseID := openAIWSPayloadStringFromRaw(payload, "previous_response_id")
		turnPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(turnPreviousResponseID)
		turnPromptCacheKey := openAIWSPayloadStringFromRaw(payload, "prompt_cache_key")
		turnStoreDisabled := s.isOpenAIWSStoreDisabledInRequestRaw(payload, account)
		turnHasFunctionCallOutput := openAIWSRawPayloadHasToolCallOutput(payload)
		eventCount := 0
		tokenEventCount := 0
		terminalEventCount := 0
		firstEventType := ""
		lastEventType := ""
		needModelReplace := false
		clientDisconnected := false
		mappedModel := ""
		var mappedModelBytes []byte
		if originalModel != "" {
			mappedModel = normalizeOpenAIModelForUpstream(account, account.GetMappedModel(originalModel))
			needModelReplace = mappedModel != "" && mappedModel != originalModel
			if needModelReplace {
				mappedModelBytes = []byte(mappedModel)
			}
		}
		for {
			upstreamMessage, readErr := lease.ReadMessageWithContextTimeout(ctx, s.openAIWSReadTimeout())
			if readErr != nil {
				lease.MarkBroken()
				return nil, wrapOpenAIWSIngressTurnError(
					"read_upstream",
					fmt.Errorf("read upstream websocket event: %w", readErr),
					wroteDownstream,
				)
			}

			eventType, eventResponseID, _ := parseOpenAIWSEventEnvelope(upstreamMessage)
			if responseID == "" && eventResponseID != "" {
				responseID = eventResponseID
			}
			if eventType != "" {
				eventCount++
				if firstEventType == "" {
					firstEventType = eventType
				}
				lastEventType = eventType
			}
			if eventType == "error" {
				errCodeRaw, errTypeRaw, errMsgRaw := parseOpenAIWSErrorEventFields(upstreamMessage)
				s.persistOpenAIWSRateLimitSignal(ctx, account, lease.HandshakeHeaders(), upstreamMessage, errCodeRaw, errTypeRaw, errMsgRaw)
				fallbackReason, _ := classifyOpenAIWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
				errCode, errType, errMessage := summarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
				recoverablePrevNotFound := fallbackReason == openAIWSIngressStagePreviousResponseNotFound &&
					turnPreviousResponseID != "" &&
					!turnHasFunctionCallOutput &&
					s.openAIWSIngressPreviousResponseRecoveryEnabled() &&
					!wroteDownstream
				if recoverablePrevNotFound {
					// Log recoverable previous_response_id misses before retrying without writing downstream.
					logOpenAIWSModeInfo(
						"ingress_ws_prev_response_recoverable account_id=%d turn=%d conn_id=%s idx=%d reason=%s code=%s type=%s message=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s store_disabled=%v has_prompt_cache_key=%v",
						account.ID,
						turn,
						truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen),
						eventCount,
						truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen),
						errCode,
						errType,
						errMessage,
						truncateOpenAIWSLogValue(turnPreviousResponseID, openAIWSIDValueMaxLen),
						normalizeOpenAIWSLogValue(turnPreviousResponseIDKind),
						truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen),
						turnStoreDisabled,
						turnPromptCacheKey != "",
					)
				} else {
					logOpenAIWSModeInfo(
						"ingress_ws_error_event account_id=%d turn=%d conn_id=%s idx=%d fallback_reason=%s err_code=%s err_type=%s err_message=%s previous_response_id=%s previous_response_id_kind=%s response_id=%s store_disabled=%v has_prompt_cache_key=%v",
						account.ID,
						turn,
						truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen),
						eventCount,
						truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen),
						errCode,
						errType,
						errMessage,
						truncateOpenAIWSLogValue(turnPreviousResponseID, openAIWSIDValueMaxLen),
						normalizeOpenAIWSLogValue(turnPreviousResponseIDKind),
						truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen),
						turnStoreDisabled,
						turnPromptCacheKey != "",
					)
				}
				// previous_response_not_found 闁?ingress 婵☆垪鈧磭纭€闁衡偓椤栨稑鐦柛妤佹礃椤愬ジ骞侀姀鐙€妲婚梺鎻掔Х閻︻垶鏁?				// 濞戞挸绉垫俊鍝ユ嫚?error 闁烩晛鐡ㄧ敮瀛樼▔鐎ｎ亜绲洪悗骞垮灪閸╂稓绮╅銈囩闁兼澘鏈Σ鎼佹偨閸欐鐟愰悘鐐插€搁獮鎾诲箳?previous_response_id 闁告艾閰ｉ崳鎼佸绩閹呯Ъ闁?turn闁?
				if recoverablePrevNotFound {
					lease.MarkBroken()
					errMsg := strings.TrimSpace(errMsgRaw)
					if errMsg == "" {
						errMsg = "previous response not found"
					}
					return nil, wrapOpenAIWSIngressTurnError(
						openAIWSIngressStagePreviousResponseNotFound,
						errors.New(errMsg),
						false,
					)
				}
				if !wroteDownstream && isOpenAIWSRateLimitError(errCodeRaw, errTypeRaw, errMsgRaw) {
					lease.MarkBroken()
					return nil, &UpstreamFailoverError{
						StatusCode:      http.StatusTooManyRequests,
						ResponseBody:    append([]byte(nil), upstreamMessage...),
						ResponseHeaders: cloneHeader(lease.HandshakeHeaders()),
					}
				}
			}
			isTokenEvent := isOpenAIWSTokenEvent(eventType)
			if isTokenEvent {
				tokenEventCount++
			}
			isTerminalEvent := isOpenAIWSTerminalEvent(eventType)
			if isTerminalEvent {
				terminalEventCount++
			}
			if firstTokenMs == nil && isTokenEvent {
				ms := int(time.Since(turnStart).Milliseconds())
				firstTokenMs = &ms
			}
			if openAIWSEventShouldParseUsage(eventType) {
				parseOpenAIWSResponseUsageFromCompletedEvent(upstreamMessage, &usage)
			}
			imageCounter.AddSSEData(upstreamMessage)

			if !clientDisconnected {
				if needModelReplace && len(mappedModelBytes) > 0 && openAIWSEventMayContainModel(eventType) && bytes.Contains(upstreamMessage, mappedModelBytes) {
					upstreamMessage = replaceOpenAIWSMessageModel(upstreamMessage, mappedModel, originalModel)
				}
				if openAIWSEventMayContainToolCalls(eventType) && openAIWSMessageLikelyContainsToolCalls(upstreamMessage) {
					if corrected, changed := s.toolCorrector.CorrectToolCallsInSSEBytes(upstreamMessage); changed {
						upstreamMessage = corrected
					}
				}
				if err := writeClientMessage(upstreamMessage); err != nil {
					if isOpenAIWSClientDisconnectError(err) {
						clientDisconnected = true
						closeStatus, closeReason := summarizeOpenAIWSReadCloseError(err)
						logOpenAIWSModeInfo(
							"ingress_ws_client_disconnected_drain account_id=%d turn=%d conn_id=%s close_status=%s close_reason=%s",
							account.ID,
							turn,
							truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen),
							closeStatus,
							truncateOpenAIWSLogValue(closeReason, openAIWSHeaderValueMaxLen),
						)
					} else {
						return nil, wrapOpenAIWSIngressTurnError(
							"write_client",
							fmt.Errorf("write client websocket event: %w", err),
							wroteDownstream,
						)
					}
				} else {
					wroteDownstream = true
				}
			}
			if isTerminalEvent {
				// 閻庡箍鍨洪崺娑氱博椤栨艾鍤掗柡鍌ゅ弨缁绘盯寮拋鍦濞戞挸锕ラ悥鑸垫交閻愭潙澶嶉柣?session 闁绘鍩栭埀顑挎缁楀宕ｉ娆庣箚闁挎稑鏈悥锝囨媼?broken 闂侇剙鐏濋崢銈夊炊閻愬娼ㄥ璺虹Ф閺併倝濡?
				if clientDisconnected {
					lease.MarkBroken()
				}
				firstTokenMsValue := -1
				if firstTokenMs != nil {
					firstTokenMsValue = *firstTokenMs
				}
				if debugEnabled {
					logOpenAIWSModeDebug(
						"ingress_ws_turn_completed account_id=%d turn=%d conn_id=%s response_id=%s duration_ms=%d events=%d token_events=%d terminal_events=%d first_event=%s last_event=%s first_token_ms=%d client_disconnected=%v",
						account.ID,
						turn,
						truncateOpenAIWSLogValue(lease.ConnID(), openAIWSIDValueMaxLen),
						truncateOpenAIWSLogValue(responseID, openAIWSIDValueMaxLen),
						time.Since(turnStart).Milliseconds(),
						eventCount,
						tokenEventCount,
						terminalEventCount,
						truncateOpenAIWSLogValue(firstEventType, openAIWSLogValueMaxLen),
						truncateOpenAIWSLogValue(lastEventType, openAIWSLogValueMaxLen),
						firstTokenMsValue,
						clientDisconnected,
					)
				}
				imageCount := imageCounter.Count()
				result := &OpenAIForwardResult{
					RequestID:       responseID,
					Usage:           usage,
					Model:           originalModel,
					UpstreamModel:   mappedModel,
					ServiceTier:     extractOpenAIServiceTierFromBody(payload),
					ReasoningEffort: extractOpenAIReasoningEffortFromBody(payload, originalModel),
					Stream:          reqStream,
					OpenAIWSMode:    true,
					ResponseHeaders: lease.HandshakeHeaders(),
					Duration:        time.Since(turnStart),
					FirstTokenMs:    firstTokenMs,
				}
				if imageCount > 0 {
					result.ImageCount = imageCount
					result.ImageSize = imageSizeTier
					result.ImageInputSize = imageInputSize
					result.ImageOutputSizes = imageCounter.Sizes()
					result.BillingModel = imageBillingModel
				}
				return result, nil
			}
		}
	}

	currentPayload := firstPayload.payloadRaw
	currentOriginalModel := firstPayload.originalModel
	currentImageBillingModel := firstPayload.imageBillingModel
	currentImageSizeTier := firstPayload.imageSizeTier
	currentImageInputSize := firstPayload.imageInputSize
	currentPayloadBytes := firstPayload.payloadBytes
	isStrictAffinityTurn := func(payload []byte) bool {
		if !storeDisabled {
			return false
		}
		return strings.TrimSpace(openAIWSPayloadStringFromRaw(payload, "previous_response_id")) != ""
	}
	var sessionLease *openAIWSConnLease
	sessionConnID := ""
	pinnedSessionConnID := ""
	unpinSessionConn := func(connID string) {
		connID = strings.TrimSpace(connID)
		if connID == "" || pinnedSessionConnID != connID {
			return
		}
		pool.UnpinConn(account.ID, connID)
		pinnedSessionConnID = ""
	}
	pinSessionConn := func(connID string) {
		if !storeDisabled {
			return
		}
		connID = strings.TrimSpace(connID)
		if connID == "" || pinnedSessionConnID == connID {
			return
		}
		if pinnedSessionConnID != "" {
			pool.UnpinConn(account.ID, pinnedSessionConnID)
			pinnedSessionConnID = ""
		}
		if pool.PinConn(account.ID, connID) {
			pinnedSessionConnID = connID
		}
	}
	// lastTurnClean 闁哄秴娲╅鍥嫉閳ь剟宕ユ惔婵堫伇閺?sendAndRelay 闁哄嫷鍨伴幆浣割潰閿濆懐鍩楅悗鐟版湰閸ㄦ岸鏁嶉崼鐔告毆闁告帗澹嗙划鎾剁博椤栨瑧鐨戝ù鐘虫构缁楁牜鈧箍鍨洪崺娑氱博椤栨稒寮撻柡鍌ゅ弨缁绘盯鏁嶆径鍫氬亾?	// 闁圭鍋撻柡鍫濐槸缁辨挾鏁宀€鐔呯€垫澘瀚哥槐娆戞嫚鐠囨彃鏅搁梺鎸庣懆椤曘倝濡存稊鐚簉or 濞存粌顑勫▎銏ゅΥ娴ｅ壊鍚傞柟鎾棑椤忣剟寮锛勭闁挎稑顦崙锟犲捶閵娿儲鍊楅柤濂変簻閸ㄥ酣寮ㄩ娑樼仐濞戞挸锕ら惇浼存晬閸?403闁挎稑顦懙?MarkBroken闁?	// 闁搞儳濮甸?releaseSessionLease 濞戞搩鍘艰ぐ褔妫侀埀顒勫捶閵娾晜濮滄慨婵撶到閻栧墎绱掗幘瀛樺皢闁?MarkBroken闁?
	lastTurnClean := false
	releaseSessionLease := func() {
		if sessionLease == nil {
			return
		}
		if !lastTurnClean {
			sessionLease.MarkBroken()
		}
		unpinSessionConn(sessionConnID)
		sessionLease.Release()
		if debugEnabled {
			logOpenAIWSModeDebug(
				"ingress_ws_upstream_released account_id=%d conn_id=%s",
				account.ID,
				truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
			)
		}
	}
	defer releaseSessionLease()

	turn := 1
	turnRetry := 0
	turnPrevRecoveryTried := false
	lastTurnFinishedAt := time.Time{}
	lastTurnResponseID := ""
	lastTurnPayload := []byte(nil)
	var lastTurnStrictState *openAIWSIngressPreviousTurnStrictState
	lastTurnReplayInput := []json.RawMessage(nil)
	lastTurnReplayInputExists := false
	currentTurnReplayInput := []json.RawMessage(nil)
	currentTurnReplayInputExists := false
	skipBeforeTurn := false
	hasCurrentOrReplayFunctionCallOutput := func(payload []byte) bool {
		if openAIWSRawPayloadHasToolCallOutput(payload) {
			return true
		}
		return currentTurnReplayInputExists && openAIWSRawItemsHasFunctionCallOutput(currentTurnReplayInput)
	}
	resetSessionLease := func(markBroken bool) {
		if sessionLease == nil {
			return
		}
		if markBroken {
			sessionLease.MarkBroken()
		}
		releaseSessionLease()
		sessionLease = nil
		sessionConnID = ""
		preferredConnID = ""
	}
	recoverIngressPrevResponseNotFound := func(relayErr error, turn int, connID string) bool {
		if !isOpenAIWSIngressPreviousResponseNotFound(relayErr) {
			return false
		}
		if turnPrevRecoveryTried || !s.openAIWSIngressPreviousResponseRecoveryEnabled() {
			return false
		}
		// 闁圭厧鎼悽?function_call_output 闁汇劌瀚顒€效閸屾瑧鐟濋柤铏灊濞戭亜顕?previous_response_id闁?		// 濞戞挸锕ラ悥?API 闂傚洠鍋撻悷?response chain 闁哄鍎辩亸顕€鏌?tool_result 濞戞挸绨肩粻锝夊礈瀹ュ洦鐣?tool_use闁?		// 濞戞挶鍨圭槐鏃堝触鎼存繄绐楅悗浣冨閸?"No tool call found for function call output" 400 闂佹寧鐟ㄩ銈夊Υ?
		if hasCurrentOrReplayFunctionCallOutput(currentPayload) {
			return false
		}
		if isStrictAffinityTurn(currentPayload) {
			// Layer 2 intentionally drops previous_response_id once, then retries through scheduling.
			logOpenAIWSModeInfo(
				"ingress_ws_prev_response_recovery_layer2 account_id=%d turn=%d conn_id=%s store_disabled_conn_mode=%s action=drop_previous_response_id_retry",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
				normalizeOpenAIWSLogValue(storeDisabledConnMode),
			)
		}
		turnPrevRecoveryTried = true
		updatedPayload, removed, dropErr := dropPreviousResponseIDFromRawPayload(currentPayload)
		if dropErr != nil || !removed {
			reason := "not_removed"
			if dropErr != nil {
				reason = "drop_error"
			}
			logOpenAIWSModeInfo(
				"ingress_ws_prev_response_recovery_skip account_id=%d turn=%d conn_id=%s reason=%s",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
				normalizeOpenAIWSLogValue(reason),
			)
			return false
		}
		updatedWithInput, setInputErr := setOpenAIWSPayloadInputSequence(
			updatedPayload,
			currentTurnReplayInput,
			currentTurnReplayInputExists,
		)
		if setInputErr != nil {
			logOpenAIWSModeInfo(
				"ingress_ws_prev_response_recovery_skip account_id=%d turn=%d conn_id=%s reason=set_full_input_error cause=%s",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
				truncateOpenAIWSLogValue(setInputErr.Error(), openAIWSLogValueMaxLen),
			)
			return false
		}
		logOpenAIWSModeInfo(
			"ingress_ws_prev_response_recovery account_id=%d turn=%d conn_id=%s action=drop_previous_response_id retry=1",
			account.ID,
			turn,
			truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
		)
		currentPayload = updatedWithInput
		currentPayloadBytes = len(updatedWithInput)
		resetSessionLease(true)
		skipBeforeTurn = true
		return true
	}
	retryIngressTurn := func(relayErr error, turn int, connID string) bool {
		if !isOpenAIWSIngressTurnRetryable(relayErr) || turnRetry >= 1 {
			return false
		}
		if isStrictAffinityTurn(currentPayload) {
			logOpenAIWSModeInfo(
				"ingress_ws_turn_retry_skip account_id=%d turn=%d conn_id=%s reason=strict_affinity",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
			)
			return false
		}
		turnRetry++
		logOpenAIWSModeInfo(
			"ingress_ws_turn_retry account_id=%d turn=%d retry=%d reason=%s conn_id=%s",
			account.ID,
			turn,
			turnRetry,
			truncateOpenAIWSLogValue(openAIWSIngressTurnRetryReason(relayErr), openAIWSLogValueMaxLen),
			truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
		)
		resetSessionLease(true)
		skipBeforeTurn = true
		return true
	}
	for {
		if turn > 1 && !skipBeforeTurn && hooks != nil && hooks.BeforeRequest != nil {
			if err := hooks.BeforeRequest(turn, currentPayload, currentOriginalModel); err != nil {
				return err
			}
		}
		if !skipBeforeTurn && hooks != nil && hooks.BeforeTurn != nil {
			if err := hooks.BeforeTurn(turn); err != nil {
				return err
			}
		}
		skipBeforeTurn = false
		currentPreviousResponseID := openAIWSPayloadStringFromRaw(currentPayload, "previous_response_id")
		expectedPrev := strings.TrimSpace(lastTurnResponseID)
		toolSignals := ToolContinuationSignals{
			HasFunctionCallOutput: openAIWSRawPayloadHasToolCallOutput(currentPayload),
		}
		if toolSignals.HasFunctionCallOutput {
			var currentReqBody map[string]any
			if err := json.Unmarshal(currentPayload, &currentReqBody); err == nil {
				toolSignals = AnalyzeToolContinuationSignals(currentReqBody)
			}
		}
		hasFunctionCallOutput := toolSignals.HasFunctionCallOutput
		// Infer previous_response_id for store=false function_call_output continuations when possible.
		if shouldInferIngressFunctionCallOutputPreviousResponseID(
			storeDisabled,
			turn,
			toolSignals,
			currentPreviousResponseID,
			expectedPrev,
		) {
			updatedPayload, setPrevErr := setPreviousResponseIDToRawPayload(currentPayload, expectedPrev)
			if setPrevErr != nil {
				logOpenAIWSModeInfo(
					"ingress_ws_function_call_output_prev_infer_skip account_id=%d turn=%d conn_id=%s reason=set_previous_response_id_error cause=%s expected_previous_response_id=%s",
					account.ID,
					turn,
					truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
					truncateOpenAIWSLogValue(setPrevErr.Error(), openAIWSLogValueMaxLen),
					truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
				)
			} else {
				currentPayload = updatedPayload
				currentPayloadBytes = len(updatedPayload)
				currentPreviousResponseID = expectedPrev
				logOpenAIWSModeInfo(
					"ingress_ws_function_call_output_prev_infer account_id=%d turn=%d conn_id=%s action=set_previous_response_id previous_response_id=%s",
					account.ID,
					turn,
					truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
					truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
				)
			}
		}
		nextReplayInput, nextReplayInputExists, replayInputErr := buildOpenAIWSReplayInputSequence(
			lastTurnReplayInput,
			lastTurnReplayInputExists,
			currentPayload,
			currentPreviousResponseID != "",
		)
		if replayInputErr != nil {
			logOpenAIWSModeInfo(
				"ingress_ws_replay_input_skip account_id=%d turn=%d conn_id=%s reason=build_error cause=%s",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
				truncateOpenAIWSLogValue(replayInputErr.Error(), openAIWSLogValueMaxLen),
			)
			currentTurnReplayInput = nil
			currentTurnReplayInputExists = false
		} else {
			currentTurnReplayInput = nextReplayInput
			currentTurnReplayInputExists = nextReplayInputExists
		}
		replayHasFunctionCallOutput := currentTurnReplayInputExists &&
			openAIWSRawItemsHasFunctionCallOutput(currentTurnReplayInput)
		hasFunctionCallOutput = hasFunctionCallOutput || replayHasFunctionCallOutput
		if storeDisabled && turn > 1 && currentPreviousResponseID != "" {
			shouldKeepPreviousResponseID := false
			strictReason := ""
			var strictErr error
			if lastTurnStrictState != nil {
				shouldKeepPreviousResponseID, strictReason, strictErr = shouldKeepIngressPreviousResponseIDWithStrictState(
					lastTurnStrictState,
					currentPayload,
					lastTurnResponseID,
					hasFunctionCallOutput,
				)
			} else {
				shouldKeepPreviousResponseID, strictReason, strictErr = shouldKeepIngressPreviousResponseID(
					lastTurnPayload,
					currentPayload,
					lastTurnResponseID,
					hasFunctionCallOutput,
				)
			}
			if strictErr != nil {
				logOpenAIWSModeInfo(
					"ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=keep_previous_response_id reason=%s cause=%s previous_response_id=%s expected_previous_response_id=%s has_function_call_output=%v",
					account.ID,
					turn,
					truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
					normalizeOpenAIWSLogValue(strictReason),
					truncateOpenAIWSLogValue(strictErr.Error(), openAIWSLogValueMaxLen),
					truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
					truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
					hasFunctionCallOutput,
				)
			} else if !shouldKeepPreviousResponseID {
				updatedPayload, removed, dropErr := dropPreviousResponseIDFromRawPayload(currentPayload)
				if dropErr != nil || !removed {
					dropReason := "not_removed"
					if dropErr != nil {
						dropReason = "drop_error"
					}
					logOpenAIWSModeInfo(
						"ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=keep_previous_response_id reason=%s drop_reason=%s previous_response_id=%s expected_previous_response_id=%s has_function_call_output=%v",
						account.ID,
						turn,
						truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
						normalizeOpenAIWSLogValue(strictReason),
						normalizeOpenAIWSLogValue(dropReason),
						truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
						truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
						hasFunctionCallOutput,
					)
				} else {
					updatedWithInput, setInputErr := setOpenAIWSPayloadInputSequence(
						updatedPayload,
						currentTurnReplayInput,
						currentTurnReplayInputExists,
					)
					if setInputErr != nil {
						logOpenAIWSModeInfo(
							"ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=keep_previous_response_id reason=%s drop_reason=set_full_input_error previous_response_id=%s expected_previous_response_id=%s cause=%s has_function_call_output=%v",
							account.ID,
							turn,
							truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
							normalizeOpenAIWSLogValue(strictReason),
							truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
							truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
							truncateOpenAIWSLogValue(setInputErr.Error(), openAIWSLogValueMaxLen),
							hasFunctionCallOutput,
						)
					} else {
						currentPayload = updatedWithInput
						currentPayloadBytes = len(updatedWithInput)
						logOpenAIWSModeInfo(
							"ingress_ws_prev_response_strict_eval account_id=%d turn=%d conn_id=%s action=drop_previous_response_id_full_create reason=%s previous_response_id=%s expected_previous_response_id=%s has_function_call_output=%v",
							account.ID,
							turn,
							truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
							normalizeOpenAIWSLogValue(strictReason),
							truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
							truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
							hasFunctionCallOutput,
						)
						currentPreviousResponseID = ""
					}
				}
			}
		}
		forcePreferredConn := isStrictAffinityTurn(currentPayload)
		if sessionLease == nil {
			acquiredLease, acquireErr := acquireTurnLease(turn, preferredConnID, forcePreferredConn)
			if acquireErr != nil {
				return fmt.Errorf("acquire upstream websocket: %w", acquireErr)
			}
			sessionLease = acquiredLease
			sessionConnID = strings.TrimSpace(sessionLease.ConnID())
			if storeDisabled {
				pinSessionConn(sessionConnID)
			} else {
				unpinSessionConn(sessionConnID)
			}
		}
		shouldPreflightPing := turn > 1 && sessionLease != nil && turnRetry == 0
		if shouldPreflightPing && openAIWSIngressPreflightPingIdle > 0 && !lastTurnFinishedAt.IsZero() {
			if time.Since(lastTurnFinishedAt) < openAIWSIngressPreflightPingIdle {
				shouldPreflightPing = false
			}
		}
		if shouldPreflightPing {
			if pingErr := sessionLease.PingWithTimeout(openAIWSConnHealthCheckTO); pingErr != nil {
				logOpenAIWSModeInfo(
					"ingress_ws_upstream_preflight_ping_fail account_id=%d turn=%d conn_id=%s cause=%s",
					account.ID,
					turn,
					truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
					truncateOpenAIWSLogValue(pingErr.Error(), openAIWSLogValueMaxLen),
				)
				if forcePreferredConn {
					// 闁圭厧鎼悽?function_call_output 闁汇劌瀚顒€效閸屾瑧鐟濋柤铏灊濞戭亜顕?previous_response_id闁?					// 濞戞挸锕ラ悥?API 闂傚洠鍋撻悷?response chain 闁哄鍎辩亸顕€鏌?tool_result 濞戞挸绨肩粻锝夊礈瀹ュ洦鐣?tool_use闁?					// 濞戞挶鍨圭槐鏃堝触鎼存繄绐楅悗浣冨閸?"No tool call found for function call output" 400 闂佹寧鐟ㄩ銈夊Υ?
					hasFCOutput := hasFunctionCallOutput
					if !turnPrevRecoveryTried && currentPreviousResponseID != "" && !hasFCOutput {
						updatedPayload, removed, dropErr := dropPreviousResponseIDFromRawPayload(currentPayload)
						if dropErr != nil || !removed {
							reason := "not_removed"
							if dropErr != nil {
								reason = "drop_error"
							}
							logOpenAIWSModeInfo(
								"ingress_ws_preflight_ping_recovery_skip account_id=%d turn=%d conn_id=%s reason=%s previous_response_id=%s",
								account.ID,
								turn,
								truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
								normalizeOpenAIWSLogValue(reason),
								truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
							)
						} else {
							updatedWithInput, setInputErr := setOpenAIWSPayloadInputSequence(
								updatedPayload,
								currentTurnReplayInput,
								currentTurnReplayInputExists,
							)
							if setInputErr != nil {
								logOpenAIWSModeInfo(
									"ingress_ws_preflight_ping_recovery_skip account_id=%d turn=%d conn_id=%s reason=set_full_input_error previous_response_id=%s cause=%s",
									account.ID,
									turn,
									truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
									truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
									truncateOpenAIWSLogValue(setInputErr.Error(), openAIWSLogValueMaxLen),
								)
							} else {
								logOpenAIWSModeInfo(
									"ingress_ws_preflight_ping_recovery account_id=%d turn=%d conn_id=%s action=drop_previous_response_id_retry previous_response_id=%s",
									account.ID,
									turn,
									truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
									truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
								)
								turnPrevRecoveryTried = true
								currentPayload = updatedWithInput
								currentPayloadBytes = len(updatedWithInput)
								resetSessionLease(true)
								skipBeforeTurn = true
								continue
							}
						}
					}
					if hasFCOutput && currentPreviousResponseID != "" {
						logOpenAIWSModeInfo(
							"ingress_ws_preflight_ping_recovery_skip account_id=%d turn=%d conn_id=%s reason=function_call_output action=fail_close previous_response_id=%s",
							account.ID,
							turn,
							truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
							truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
						)
					}
					resetSessionLease(true)
					return NewOpenAIWSClientCloseError(
						coderws.StatusPolicyViolation,
						"upstream continuation connection is unavailable; please restart the conversation",
						pingErr,
					)
				}
				resetSessionLease(true)

				acquiredLease, acquireErr := acquireTurnLease(turn, preferredConnID, forcePreferredConn)
				if acquireErr != nil {
					return fmt.Errorf("acquire upstream websocket after preflight ping fail: %w", acquireErr)
				}
				sessionLease = acquiredLease
				sessionConnID = strings.TrimSpace(sessionLease.ConnID())
				if storeDisabled {
					pinSessionConn(sessionConnID)
				}
			}
		}
		connID := sessionConnID
		if currentPreviousResponseID != "" {
			chainedFromLast := expectedPrev != "" && currentPreviousResponseID == expectedPrev
			currentPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(currentPreviousResponseID)
			logOpenAIWSModeInfo(
				"ingress_ws_turn_chain account_id=%d turn=%d conn_id=%s previous_response_id=%s previous_response_id_kind=%s last_turn_response_id=%s chained_from_last=%v preferred_conn_id=%s header_session_id=%s header_conversation_id=%s has_turn_state=%v turn_state_len=%d has_prompt_cache_key=%v store_disabled=%v",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
				truncateOpenAIWSLogValue(currentPreviousResponseID, openAIWSIDValueMaxLen),
				normalizeOpenAIWSLogValue(currentPreviousResponseIDKind),
				truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
				chainedFromLast,
				truncateOpenAIWSLogValue(preferredConnID, openAIWSIDValueMaxLen),
				openAIWSHeaderValueForLog(baseAcquireReq.Headers, "session_id"),
				openAIWSHeaderValueForLog(baseAcquireReq.Headers, "conversation_id"),
				turnState != "",
				len(turnState),
				openAIWSPayloadStringFromRaw(currentPayload, "prompt_cache_key") != "",
				storeDisabled,
			)
		}

		result, relayErr := sendAndRelay(turn, sessionLease, currentPayload, currentPayloadBytes, currentOriginalModel, currentImageBillingModel, currentImageSizeTier, currentImageInputSize)
		if relayErr != nil {
			lastTurnClean = false
			if recoverIngressPrevResponseNotFound(relayErr, turn, connID) {
				continue
			}
			if retryIngressTurn(relayErr, turn, connID) {
				continue
			}
			finalErr := relayErr
			if unwrapped := errors.Unwrap(relayErr); unwrapped != nil {
				finalErr = unwrapped
			}
			if hooks != nil && hooks.AfterTurn != nil {
				hooks.AfterTurn(turn, nil, finalErr)
			}
			sessionLease.MarkBroken()
			return finalErr
		}
		turnRetry = 0
		turnPrevRecoveryTried = false
		lastTurnFinishedAt = time.Now()
		lastTurnClean = true
		if hooks != nil && hooks.AfterTurn != nil {
			hooks.AfterTurn(turn, result, nil)
		}
		if result == nil {
			return errors.New("websocket turn result is nil")
		}
		responseID := strings.TrimSpace(result.RequestID)
		lastTurnResponseID = responseID
		lastTurnPayload = cloneOpenAIWSPayloadBytes(currentPayload)
		lastTurnReplayInput = cloneOpenAIWSRawMessages(currentTurnReplayInput)
		lastTurnReplayInputExists = currentTurnReplayInputExists
		nextStrictState, strictStateErr := buildOpenAIWSIngressPreviousTurnStrictState(currentPayload)
		if strictStateErr != nil {
			lastTurnStrictState = nil
			logOpenAIWSModeInfo(
				"ingress_ws_prev_response_strict_state_skip account_id=%d turn=%d conn_id=%s reason=build_error cause=%s",
				account.ID,
				turn,
				truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
				truncateOpenAIWSLogValue(strictStateErr.Error(), openAIWSLogValueMaxLen),
			)
		} else {
			lastTurnStrictState = nextStrictState
		}

		if responseID != "" && stateStore != nil {
			ttl := s.openAIWSResponseStickyTTL()
			logOpenAIWSBindResponseAccountWarn(groupID, account.ID, responseID, stateStore.BindResponseAccount(ctx, groupID, responseID, account.ID, ttl))
			stateStore.BindResponseConn(responseID, connID, ttl)
		}
		if stateStore != nil && storeDisabled && sessionHash != "" {
			stateStore.BindSessionConn(groupID, sessionHash, connID, s.openAIWSSessionStickyTTL())
		}
		if connID != "" {
			preferredConnID = connID
		}

		nextClientMessage, readErr := readClientMessage()
		if readErr != nil {
			if isOpenAIWSClientDisconnectError(readErr) {
				closeStatus, closeReason := summarizeOpenAIWSReadCloseError(readErr)
				logOpenAIWSModeInfo(
					"ingress_ws_client_closed account_id=%d conn_id=%s close_status=%s close_reason=%s",
					account.ID,
					truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
					closeStatus,
					truncateOpenAIWSLogValue(closeReason, openAIWSHeaderValueMaxLen),
				)
				return nil
			}
			return fmt.Errorf("read client websocket request: %w", readErr)
		}

		nextPayload, parseErr := parseClientPayload(nextClientMessage)
		if parseErr != nil {
			return parseErr
		}
		if nextPayload.promptCacheKey != "" {
			// ingress 濞村吋淇洪惁浠嬪捶閵婏附娈诲☉鎿冧簻椤撳綊骞嬫搴紓 WS 闁汇垻鍠庨幊锟犲川閵婏附鍩傞柛鎰噹椤︽煡鎮介妸銉﹀€卞☉鎾亾濞戞挸锕ラ悥鑸垫交閻愭潙澶嶉柨?			// prompt_cache_key 閻庝絻顫夎ぐ娆撳箥鐎ｎ亗浠堥柣銊ュ濞插潡寮０浣虹煂闁革负鍔嶅﹢顓㈠级閵夆晜浠橀悷鏇氱窔閸ｆ悂寮弶璺ㄧ处閺夆晝鍋炲鍌炴偨閻斿憡娅忛柕?			updatedHeaders,
			updatedHeaders, _ := s.buildOpenAIWSHeaders(c, account, token, wsDecision, isCodexCLI, turnState, strings.TrimSpace(c.GetHeader(openAIWSTurnMetadataHeader)), nextPayload.promptCacheKey)
			baseAcquireReq.Headers = updatedHeaders
		}
		if nextPayload.previousResponseID != "" {
			expectedPrev := strings.TrimSpace(lastTurnResponseID)
			chainedFromLast := expectedPrev != "" && nextPayload.previousResponseID == expectedPrev
			nextPreviousResponseIDKind := ClassifyOpenAIPreviousResponseIDKind(nextPayload.previousResponseID)
			logOpenAIWSModeInfo(
				"ingress_ws_next_turn_chain account_id=%d turn=%d next_turn=%d conn_id=%s previous_response_id=%s previous_response_id_kind=%s last_turn_response_id=%s chained_from_last=%v has_prompt_cache_key=%v store_disabled=%v",
				account.ID,
				turn,
				turn+1,
				truncateOpenAIWSLogValue(connID, openAIWSIDValueMaxLen),
				truncateOpenAIWSLogValue(nextPayload.previousResponseID, openAIWSIDValueMaxLen),
				normalizeOpenAIWSLogValue(nextPreviousResponseIDKind),
				truncateOpenAIWSLogValue(expectedPrev, openAIWSIDValueMaxLen),
				chainedFromLast,
				nextPayload.promptCacheKey != "",
				storeDisabled,
			)
		}
		if stateStore != nil && nextPayload.previousResponseID != "" {
			if stickyConnID, ok := stateStore.GetResponseConn(nextPayload.previousResponseID); ok {
				if sessionConnID != "" && stickyConnID != "" && stickyConnID != sessionConnID {
					logOpenAIWSModeInfo(
						"ingress_ws_keep_session_conn account_id=%d turn=%d conn_id=%s sticky_conn_id=%s previous_response_id=%s",
						account.ID,
						turn,
						truncateOpenAIWSLogValue(sessionConnID, openAIWSIDValueMaxLen),
						truncateOpenAIWSLogValue(stickyConnID, openAIWSIDValueMaxLen),
						truncateOpenAIWSLogValue(nextPayload.previousResponseID, openAIWSIDValueMaxLen),
					)
				} else {
					preferredConnID = stickyConnID
				}
			}
		}
		currentPayload = nextPayload.payloadRaw
		currentOriginalModel = nextPayload.originalModel
		currentImageBillingModel = nextPayload.imageBillingModel
		currentImageSizeTier = nextPayload.imageSizeTier
		currentImageInputSize = nextPayload.imageInputSize
		currentPayloadBytes = nextPayload.payloadBytes
		storeDisabled = s.isOpenAIWSStoreDisabledInRequestRaw(currentPayload, account)
		if !storeDisabled {
			unpinSessionConn(sessionConnID)
		}
		turn++
	}
}

func (s *OpenAIGatewayService) isOpenAIWSGeneratePrewarmEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.PrewarmGenerateEnabled
}

// performOpenAIWSGeneratePrewarm 闁?WSv2 濞戞挸顑嗘晶鐣屾偘鐏炶棄璁查梺顐㈩槺濞?generate=false 濡澘瀚崕褰掑Υ?// 濡澘瀚崕瑙勵渶濡鍚囬柛蹇斿▕濡挳鏁嶇仦鑲╃煂闁革负鍔戦崢銈囩磾椤旇偐纾婚柛姘煎灠閹鎮介悢鍛婃珡闁挎稒绋戦妵鎴犳嫻閵夛附顦ч柟绋款槸瑜版煡骞侀姀鐙€妲婚梺鎸庣懆椤曘倝宕堕悙琛″亾閳ь剟宕?HTTP闁?
func (s *OpenAIGatewayService) performOpenAIWSGeneratePrewarm(
	ctx context.Context,
	lease *openAIWSConnLease,
	decision OpenAIWSProtocolDecision,
	payload map[string]any,
	previousResponseID string,
	reqBody map[string]any,
	account *Account,
	stateStore OpenAIWSStateStore,
	groupID int64,
) error {
	if s == nil {
		return nil
	}
	if lease == nil || account == nil {
		logOpenAIWSModeInfo("prewarm_skip reason=invalid_state has_lease=%v has_account=%v", lease != nil, account != nil)
		return nil
	}
	connID := strings.TrimSpace(lease.ConnID())
	if !s.isOpenAIWSGeneratePrewarmEnabled() {
		return nil
	}
	if decision.Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
		logOpenAIWSModeInfo(
			"prewarm_skip account_id=%d conn_id=%s reason=transport_not_v2 transport=%s",
			account.ID,
			connID,
			normalizeOpenAIWSLogValue(string(decision.Transport)),
		)
		return nil
	}
	if strings.TrimSpace(previousResponseID) != "" {
		logOpenAIWSModeInfo(
			"prewarm_skip account_id=%d conn_id=%s reason=has_previous_response_id previous_response_id=%s",
			account.ID,
			connID,
			truncateOpenAIWSLogValue(previousResponseID, openAIWSIDValueMaxLen),
		)
		return nil
	}
	if lease.IsPrewarmed() {
		logOpenAIWSModeInfo("prewarm_skip account_id=%d conn_id=%s reason=already_prewarmed", account.ID, connID)
		return nil
	}
	if NeedsToolContinuation(reqBody) {
		logOpenAIWSModeInfo("prewarm_skip account_id=%d conn_id=%s reason=tool_continuation", account.ID, connID)
		return nil
	}
	prewarmStart := time.Now()
	logOpenAIWSModeInfo("prewarm_start account_id=%d conn_id=%s", account.ID, connID)

	prewarmPayload := make(map[string]any, len(payload)+1)
	for k, v := range payload {
		prewarmPayload[k] = v
	}
	prewarmPayload["generate"] = false
	prewarmPayloadJSON := payloadAsJSONBytes(prewarmPayload)

	if err := lease.WriteJSONWithContextTimeout(ctx, prewarmPayload, s.openAIWSWriteTimeout()); err != nil {
		lease.MarkBroken()
		logOpenAIWSModeInfo(
			"prewarm_write_fail account_id=%d conn_id=%s cause=%s",
			account.ID,
			connID,
			truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen),
		)
		return wrapOpenAIWSFallback("prewarm_write", err)
	}
	logOpenAIWSModeInfo("prewarm_write_sent account_id=%d conn_id=%s payload_bytes=%d", account.ID, connID, len(prewarmPayloadJSON))

	prewarmResponseID := ""
	prewarmEventCount := 0
	prewarmTerminalCount := 0
	for {
		message, readErr := lease.ReadMessageWithContextTimeout(ctx, s.openAIWSReadTimeout())
		if readErr != nil {
			lease.MarkBroken()
			closeStatus, closeReason := summarizeOpenAIWSReadCloseError(readErr)
			logOpenAIWSModeInfo(
				"prewarm_read_fail account_id=%d conn_id=%s close_status=%s close_reason=%s cause=%s events=%d",
				account.ID,
				connID,
				closeStatus,
				closeReason,
				truncateOpenAIWSLogValue(readErr.Error(), openAIWSLogValueMaxLen),
				prewarmEventCount,
			)
			return wrapOpenAIWSFallback("prewarm_"+classifyOpenAIWSReadFallbackReason(readErr), readErr)
		}

		eventType, eventResponseID, _ := parseOpenAIWSEventEnvelope(message)
		if eventType == "" {
			continue
		}
		prewarmEventCount++
		if prewarmResponseID == "" && eventResponseID != "" {
			prewarmResponseID = eventResponseID
		}
		if prewarmEventCount <= openAIWSPrewarmEventLogHead || eventType == "error" || isOpenAIWSTerminalEvent(eventType) {
			logOpenAIWSModeInfo(
				"prewarm_event account_id=%d conn_id=%s idx=%d type=%s bytes=%d",
				account.ID,
				connID,
				prewarmEventCount,
				truncateOpenAIWSLogValue(eventType, openAIWSLogValueMaxLen),
				len(message),
			)
		}

		if eventType == "error" {
			errCodeRaw, errTypeRaw, errMsgRaw := parseOpenAIWSErrorEventFields(message)
			s.persistOpenAIWSRateLimitSignal(ctx, account, lease.HandshakeHeaders(), message, errCodeRaw, errTypeRaw, errMsgRaw)
			errMsg := strings.TrimSpace(errMsgRaw)
			if errMsg == "" {
				errMsg = "OpenAI websocket prewarm error"
			}
			fallbackReason, canFallback := classifyOpenAIWSErrorEventFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			errCode, errType, errMessage := summarizeOpenAIWSErrorEventFieldsFromRaw(errCodeRaw, errTypeRaw, errMsgRaw)
			logOpenAIWSModeInfo(
				"prewarm_error_event account_id=%d conn_id=%s idx=%d fallback_reason=%s can_fallback=%v err_code=%s err_type=%s err_message=%s",
				account.ID,
				connID,
				prewarmEventCount,
				truncateOpenAIWSLogValue(fallbackReason, openAIWSLogValueMaxLen),
				canFallback,
				errCode,
				errType,
				errMessage,
			)
			lease.MarkBroken()
			if canFallback {
				return wrapOpenAIWSFallback("prewarm_"+fallbackReason, errors.New(errMsg))
			}
			return wrapOpenAIWSFallback("prewarm_error_event", errors.New(errMsg))
		}

		if isOpenAIWSTerminalEvent(eventType) {
			prewarmTerminalCount++
			break
		}
	}

	lease.MarkPrewarmed()
	if prewarmResponseID != "" && stateStore != nil {
		ttl := s.openAIWSResponseStickyTTL()
		logOpenAIWSBindResponseAccountWarn(groupID, account.ID, prewarmResponseID, stateStore.BindResponseAccount(ctx, groupID, prewarmResponseID, account.ID, ttl))
		stateStore.BindResponseConn(prewarmResponseID, lease.ConnID(), ttl)
	}
	logOpenAIWSModeInfo(
		"prewarm_done account_id=%d conn_id=%s response_id=%s events=%d terminal_events=%d duration_ms=%d",
		account.ID,
		connID,
		truncateOpenAIWSLogValue(prewarmResponseID, openAIWSIDValueMaxLen),
		prewarmEventCount,
		prewarmTerminalCount,
		time.Since(prewarmStart).Milliseconds(),
	)
	return nil
}

func payloadAsJSON(payload map[string]any) string {
	return string(payloadAsJSONBytes(payload))
}

func payloadAsJSONBytes(payload map[string]any) []byte {
	if len(payload) == 0 {
		return []byte("{}")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return []byte("{}")
	}
	return body
}

func isOpenAIWSTerminalEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		return true
	default:
		return false
	}
}

func isOpenAIWSTokenEvent(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return false
	}
	switch eventType {
	case "response.created", "response.in_progress", "response.output_item.added", "response.output_item.done":
		return false
	}
	if strings.Contains(eventType, ".delta") {
		return true
	}
	if strings.HasPrefix(eventType, "response.output_text") {
		return true
	}
	if strings.HasPrefix(eventType, "response.output") {
		return true
	}
	// 缂備礁鐗婇娑欑鐎ｂ晜顐介柨娑樻緛esponse.completed/done/failed/...闁挎稑顦遍弫?isOpenAIWSTerminalEvent 闁告娲滅€氼厽寰勯崟顓熷€為柕?	// 濞戞挸绉烽崗姗€骞庢繝鍌滄殜濞寸媭鍓欑紞瀣媴?token event闁挎稑鑻幆渚€宕氬▎蹇曠Ъ濞戞挸锕ラ悥璺衡柦閳╁啯绠掗柛娆樺灥閻︽垿宕氶銈嗙暠 delta 闁哄啳顔愮槐?	// firstTokenMs 濞村吋淇洪～锔界箙椤愩垹鐓傜紓浣哥墛椤掓盯寮捄鍝勭厺闁挎稑鐬奸悺鎴炵鎼淬垹惟"闁诡剚妲掗埀顒侇殕濡?閻犲浂鍨辨慨銈嗙▔?濡?token 鐎点倖鍎肩换?闁?
	return false
}

func replaceOpenAIWSMessageModel(message []byte, fromModel, toModel string) []byte {
	if len(message) == 0 {
		return message
	}
	if strings.TrimSpace(fromModel) == "" || strings.TrimSpace(toModel) == "" || fromModel == toModel {
		return message
	}
	if !bytes.Contains(message, []byte(`"model"`)) || !bytes.Contains(message, []byte(fromModel)) {
		return message
	}
	modelValues := gjson.GetManyBytes(message, "model", "response.model")
	replaceModel := modelValues[0].Exists() && modelValues[0].Str == fromModel
	replaceResponseModel := modelValues[1].Exists() && modelValues[1].Str == fromModel
	if !replaceModel && !replaceResponseModel {
		return message
	}
	updated := message
	if replaceModel {
		if next, err := sjson.SetBytes(updated, "model", toModel); err == nil {
			updated = next
		}
	}
	if replaceResponseModel {
		if next, err := sjson.SetBytes(updated, "response.model", toModel); err == nil {
			updated = next
		}
	}
	return updated
}

func populateOpenAIUsageFromResponseJSON(body []byte, usage *OpenAIUsage) {
	if usage == nil || len(body) == 0 {
		return
	}
	values := gjson.GetManyBytes(
		body,
		"usage.input_tokens",
		"usage.output_tokens",
		"usage.input_tokens_details.cached_tokens",
	)
	usage.InputTokens = int(values[0].Int())
	usage.OutputTokens = int(values[1].Int())
	usage.CacheReadInputTokens = int(values[2].Int())
}

func getOpenAIGroupIDFromContext(c *gin.Context) int64 {
	if c == nil {
		return 0
	}
	value, exists := c.Get("api_key")
	if !exists {
		return 0
	}
	apiKey, ok := value.(*APIKey)
	if !ok || apiKey == nil || apiKey.GroupID == nil {
		return 0
	}
	return *apiKey.GroupID
}

// SelectAccountByPreviousResponseID 闁?previous_response_id 闁告稒鍨濋懙鎴犳嫻閿曗偓瑜拌法鍒掑Ο璺ㄧ闁?// 闁哄牜浜滈幊鈩冪▔椤撶喎鐏楅悹鎰剁畱瑜版寧绋夊鍛闁活潿鍔嶅鍌涙交閺傛寧绀€ (nil, nil)闁挎稑鐬奸弫杈╂嫬閸愵亝鏆忛柡鍌氭贡閹撮绱掗陇娉查悽顖涚摃椤鎷崘銊ヮ唺闁?
func (s *OpenAIGatewayService) SelectAccountByPreviousResponseID(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requireCompact bool,
	platform string,
) (*AccountSelectionResult, error) {
	return s.selectAccountByPreviousResponseIDForCapability(ctx, groupID, previousResponseID, requestedModel, excludedIDs, "", requireCompact, platform)
}

func (s *OpenAIGatewayService) selectAccountByPreviousResponseIDForCapability(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredCapability OpenAIEndpointCapability,
	requireCompact bool,
	platform string,
) (*AccountSelectionResult, error) {
	if s == nil {
		return nil, nil
	}
	responseID := strings.TrimSpace(previousResponseID)
	if responseID == "" {
		return nil, nil
	}
	store := s.getOpenAIWSStateStore()
	if store == nil {
		return nil, nil
	}

	accountID, err := store.GetResponseAccount(ctx, derefGroupID(groupID), responseID)
	if err != nil || accountID <= 0 {
		return nil, nil
	}
	if excludedIDs != nil {
		if _, excluded := excludedIDs[accountID]; excluded {
			return nil, nil
		}
	}

	account, err := s.getSchedulableAccount(ctx, accountID)
	if err != nil || account == nil {
		_ = store.DeleteResponseAccount(ctx, derefGroupID(groupID), responseID)
		return nil, nil
	}
	// 闂?WSv2 闁革妇鍎ゅ▍娆撴晬閸繍娲?force_http/闁稿繈鍔岄惇顒勫礂閹惰姤锛旈柨娑橆槷缁楀鎯旈弬鎯р枏闁?previous_response_id 缂侇喗顭堢换娑㈡晬?	// 濞寸姰鍎扮换姘跺箰娴ｆ祴鍋撳鍐╃婵犲﹥鑹鹃崺?HTTP闁炽儲绻傞幃妤呮儍閸曨偄鍧婇柛娆掑蔼椤㈡垶绋夋潪鎵伇闁煎嘲鐡ㄩ埀顑讲鍋?
	if s.getOpenAIWSProtocolResolver().Resolve(account).Transport != OpenAIUpstreamTransportResponsesWebsocketV2 {
		return nil, nil
	}
	if shouldClearStickySession(account, requestedModel) || !accountMatchesPlatform(account, platform) || !account.IsSchedulable() {
		_ = store.DeleteResponseAccount(ctx, derefGroupID(groupID), responseID)
		return nil, nil
	}
	if requestedModel != "" && !account.IsModelSupported(requestedModel) {
		return nil, nil
	}
	if !account.SupportsOpenAIEndpointCapability(requiredCapability) {
		return nil, nil
	}
	// Quota auto-pause must also gate the previous_response_id sticky path; otherwise an
	// account over its 5h/7d threshold keeps serving the same response chain even though
	// normal scheduling skips it. Pause is transient, so fall through to normal scheduling
	// without deleting the binding (the window may reset before the next turn).
	if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
		return nil, nil
	}
	if s.schedulerSnapshot != nil && s.accountRepo != nil {
		latest, latestErr := s.accountRepo.GetByID(ctx, account.ID)
		if latestErr != nil || latest == nil {
			_ = store.DeleteResponseAccount(ctx, derefGroupID(groupID), responseID)
			return nil, nil
		}
		if shouldClearStickySession(latest, requestedModel) || !latest.IsOpenAI() || !latest.IsSchedulable() {
			_ = store.DeleteResponseAccount(ctx, derefGroupID(groupID), responseID)
			return nil, nil
		}
		if requestedModel != "" && !latest.IsModelSupported(requestedModel) {
			return nil, nil
		}
		if !latest.SupportsOpenAIEndpointCapability(requiredCapability) {
			return nil, nil
		}
		if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, latest); paused {
			return nil, nil
		}
		if s.isOpenAIAccountRuntimeBlocked(latest) {
			_ = store.DeleteResponseAccount(ctx, derefGroupID(groupID), responseID)
			return nil, nil
		}
		account = latest
	}
	if requireCompact && openAICompactSupportTier(account) == 0 {
		_ = store.DeleteResponseAccount(ctx, derefGroupID(groupID), responseID)
		return nil, nil
	}

	result, acquireErr := s.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
	if acquireErr == nil && result.Acquired {
		logOpenAIWSBindResponseAccountWarn(
			derefGroupID(groupID),
			accountID,
			responseID,
			store.BindResponseAccount(ctx, derefGroupID(groupID), responseID, accountID, s.openAIWSResponseStickyTTL()),
		)
		return &AccountSelectionResult{
			Account:     account,
			Acquired:    true,
			ReleaseFunc: result.ReleaseFunc,
		}, nil
	}

	cfg := s.schedulingConfig()
	if s.concurrencyService != nil {
		return &AccountSelectionResult{
			Account: account,
			WaitPlan: &AccountWaitPlan{
				AccountID:      accountID,
				MaxConcurrency: account.Concurrency,
				Timeout:        cfg.StickySessionWaitTimeout,
				MaxWaiting:     cfg.StickySessionMaxWaiting,
			},
		}, nil
	}
	return nil, nil
}

func classifyOpenAIWSAcquireError(err error) string {
	if err == nil {
		return "acquire_conn"
	}
	var dialErr *openAIWSDialError
	if errors.As(err, &dialErr) {
		switch dialErr.StatusCode {
		case 426:
			return "upgrade_required"
		case 401, 403:
			return "auth_failed"
		case 429:
			return "upstream_rate_limited"
		}
		if dialErr.StatusCode >= 500 {
			return "upstream_5xx"
		}
		return "dial_failed"
	}
	if errors.Is(err, errOpenAIWSConnQueueFull) {
		return "conn_queue_full"
	}
	if errors.Is(err, errOpenAIWSPreferredConnUnavailable) {
		return "preferred_conn_unavailable"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "acquire_timeout"
	}
	return "acquire_conn"
}

func isOpenAIWSRateLimitError(codeRaw, errTypeRaw, msgRaw string) bool {
	code := strings.ToLower(strings.TrimSpace(codeRaw))
	errType := strings.ToLower(strings.TrimSpace(errTypeRaw))
	msg := strings.ToLower(strings.TrimSpace(msgRaw))

	if strings.Contains(errType, "rate_limit") || strings.Contains(errType, "usage_limit") {
		return true
	}
	if strings.Contains(code, "rate_limit") || strings.Contains(code, "usage_limit") || strings.Contains(code, "insufficient_quota") {
		return true
	}
	if strings.Contains(msg, "usage limit") && strings.Contains(msg, "reached") {
		return true
	}
	if strings.Contains(msg, "rate limit") && (strings.Contains(msg, "reached") || strings.Contains(msg, "exceeded")) {
		return true
	}
	return false
}

func (s *OpenAIGatewayService) persistOpenAIWSRateLimitSignal(ctx context.Context, account *Account, headers http.Header, responseBody []byte, codeRaw, errTypeRaw, msgRaw string) {
	if s == nil || s.rateLimitService == nil || account == nil || account.Platform != PlatformOpenAI {
		return
	}
	if !isOpenAIWSRateLimitError(codeRaw, errTypeRaw, msgRaw) {
		return
	}
	s.handleOpenAIAccountUpstreamError(ctx, account, http.StatusTooManyRequests, headers, responseBody)
}

func classifyOpenAIWSErrorEventFromRaw(codeRaw, errTypeRaw, msgRaw string) (string, bool) {
	code := strings.ToLower(strings.TrimSpace(codeRaw))
	errType := strings.ToLower(strings.TrimSpace(errTypeRaw))
	msg := strings.ToLower(strings.TrimSpace(msgRaw))

	switch code {
	case "upgrade_required":
		return "upgrade_required", true
	case "websocket_not_supported", "websocket_unsupported":
		return "ws_unsupported", true
	case "websocket_connection_limit_reached":
		return "ws_connection_limit_reached", true
	case "invalid_encrypted_content":
		return "invalid_encrypted_content", true
	case "previous_response_not_found":
		return "previous_response_not_found", true
	}
	if isOpenAIWSRateLimitError(codeRaw, errTypeRaw, msgRaw) {
		return "upstream_rate_limited", false
	}
	if strings.Contains(msg, "upgrade required") || strings.Contains(msg, "status 426") {
		return "upgrade_required", true
	}
	if strings.Contains(errType, "upgrade") {
		return "upgrade_required", true
	}
	if strings.Contains(msg, "websocket") && strings.Contains(msg, "unsupported") {
		return "ws_unsupported", true
	}
	if strings.Contains(msg, "connection limit") && strings.Contains(msg, "websocket") {
		return "ws_connection_limit_reached", true
	}
	if strings.Contains(msg, "invalid_encrypted_content") ||
		(strings.Contains(msg, "encrypted content") && strings.Contains(msg, "could not be verified")) {
		return "invalid_encrypted_content", true
	}
	if strings.Contains(msg, "previous_response_not_found") ||
		(strings.Contains(msg, "previous response") && strings.Contains(msg, "not found")) {
		return "previous_response_not_found", true
	}
	if strings.Contains(errType, "server_error") || strings.Contains(code, "server_error") {
		return "upstream_error_event", true
	}
	return "event_error", false
}

func classifyOpenAIWSErrorEvent(message []byte) (string, bool) {
	if len(message) == 0 {
		return "event_error", false
	}
	return classifyOpenAIWSErrorEventFromRaw(parseOpenAIWSErrorEventFields(message))
}

func openAIWSErrorHTTPStatusFromRaw(codeRaw, errTypeRaw string) int {
	code := strings.ToLower(strings.TrimSpace(codeRaw))
	errType := strings.ToLower(strings.TrimSpace(errTypeRaw))
	switch {
	case strings.Contains(errType, "invalid_request"),
		strings.Contains(code, "invalid_request"),
		strings.Contains(code, "bad_request"),
		code == "invalid_encrypted_content",
		code == "previous_response_not_found":
		return http.StatusBadRequest
	case strings.Contains(errType, "authentication"),
		strings.Contains(code, "invalid_api_key"),
		strings.Contains(code, "unauthorized"):
		return http.StatusUnauthorized
	case strings.Contains(errType, "permission"),
		strings.Contains(code, "forbidden"):
		return http.StatusForbidden
	case isOpenAIWSRateLimitError(codeRaw, errTypeRaw, ""):
		return http.StatusTooManyRequests
	default:
		return http.StatusBadGateway
	}
}

func openAIWSErrorHTTPStatus(message []byte) int {
	if len(message) == 0 {
		return http.StatusBadGateway
	}
	codeRaw, errTypeRaw, _ := parseOpenAIWSErrorEventFields(message)
	return openAIWSErrorHTTPStatusFromRaw(codeRaw, errTypeRaw)
}

func (s *OpenAIGatewayService) openAIWSFallbackCooldown() time.Duration {
	if s == nil || s.cfg == nil {
		return 30 * time.Second
	}
	seconds := s.cfg.Gateway.OpenAIWS.FallbackCooldownSeconds
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func (s *OpenAIGatewayService) isOpenAIWSFallbackCooling(accountID int64) bool {
	if s == nil || accountID <= 0 {
		return false
	}
	cooldown := s.openAIWSFallbackCooldown()
	if cooldown <= 0 {
		return false
	}
	rawUntil, ok := s.openaiWSFallbackUntil.Load(accountID)
	if !ok || rawUntil == nil {
		return false
	}
	until, ok := rawUntil.(time.Time)
	if !ok || until.IsZero() {
		s.openaiWSFallbackUntil.Delete(accountID)
		return false
	}
	if time.Now().Before(until) {
		return true
	}
	s.openaiWSFallbackUntil.Delete(accountID)
	return false
}

func (s *OpenAIGatewayService) markOpenAIWSFallbackCooling(accountID int64, _ string) {
	if s == nil || accountID <= 0 {
		return
	}
	cooldown := s.openAIWSFallbackCooldown()
	if cooldown <= 0 {
		return
	}
	s.openaiWSFallbackUntil.Store(accountID, time.Now().Add(cooldown))
}

func (s *OpenAIGatewayService) clearOpenAIWSFallbackCooling(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiWSFallbackUntil.Delete(accountID)
}
