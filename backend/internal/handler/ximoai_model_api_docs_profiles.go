package handler

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	modelDocsCategoryConversation = "conversation"
	modelDocsCategoryImage        = "image"
	modelDocsCategoryVideo        = "video"
	modelDocsCategoryTTS          = "tts"
	modelDocsCategoryASR          = "asr"
)

type ModelAPIDocsWorkflowStep struct {
	ID              string                  `json:"id"`
	Title           string                  `json:"title"`
	Method          string                  `json:"method"`
	Path            string                  `json:"path"`
	RoutePath       string                  `json:"-"`
	ContentType     string                  `json:"content_type,omitempty"`
	RequestExample  string                  `json:"request_example,omitempty"`
	ResponseExample string                  `json:"response_example,omitempty"`
	Parameters      []ModelAPIDocsParameter `json:"parameters"`
}

type ModelAPIDocsParameter struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ModelAPIDocsEndpointVariant struct {
	ID          string                     `json:"id"`
	Label       string                     `json:"label"`
	Mode        string                     `json:"mode"`
	Transport   string                     `json:"transport"`
	Delivery    string                     `json:"delivery"`
	Termination string                     `json:"termination"`
	Steps       []ModelAPIDocsWorkflowStep `json:"steps"`
}

type ModelAPIDocsEndpointProfile struct {
	ID          string                        `json:"id"`
	Category    string                        `json:"category"`
	Protocol    string                        `json:"protocol"`
	Title       string                        `json:"title"`
	Description string                        `json:"description"`
	Variants    []ModelAPIDocsEndpointVariant `json:"variants"`
	platforms   []string
	kinds       []string
	protocols   []string
	capability  string
	nativeSlugs []string
}

type modelAPIDocsResolutionInput struct {
	Platform     string
	Kind         string
	Protocol     string
	Model        string
	Capabilities []string
	BillingModes []string
}

func XimoAIModelAPIDocsProfiles() []ModelAPIDocsEndpointProfile {
	profiles := []ModelAPIDocsEndpointProfile{
		{
			ID: "openai_responses", Category: modelDocsCategoryConversation, Protocol: "openai_responses",
			Title: "OpenAI Responses", Description: "Create a model response with synchronous JSON or streaming SSE delivery.",
			protocols: []string{service.PlatformProtocolOpenAI, service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityResponses,
			Variants: []ModelAPIDocsEndpointVariant{
				httpJSONVariant("sync", "Synchronous", "sync", "POST", EndpointResponses, EndpointResponses,
					`{"model":{{MODEL_JSON}},"input":"Hello"}`,
					`{"id":"resp_...","status":"completed","output":[]}`),
				httpSSEVariant("stream", "Streaming", "POST", EndpointResponses, EndpointResponses,
					`{"model":{{MODEL_JSON}},"input":"Hello","stream":true}`,
					"event: response.output_text.delta\ndata: {...}\n\nevent: response.completed\ndata: {...}"),
			},
		},
		{
			ID: "openai_responses_realtime", Category: modelDocsCategoryConversation, Protocol: "openai_responses",
			Title: "OpenAI Responses Realtime", Description: "Use a full-duplex WebSocket session for realtime response events.",
			protocols: []string{service.PlatformProtocolOpenAI, service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityRealtime,
			Variants: []ModelAPIDocsEndpointVariant{
				webSocketVariant("bidirectional", "Realtime", "bidirectional", EndpointResponses, EndpointResponses,
					"Connect with Authorization: Bearer $XIMOAI_API_KEY, then exchange Responses realtime JSON events.",
					"response.created -> response.output_text.delta -> response.completed"),
			},
		},
		{
			ID: "openai_chat_completions", Category: modelDocsCategoryConversation, Protocol: "openai_chat_completions",
			Title: "OpenAI Chat Completions", Description: "Create a chat completion with synchronous JSON or streaming SSE delivery.",
			protocols: []string{service.PlatformProtocolOpenAI, service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityChatCompletions,
			kinds: []string{service.PlatformKindOpenAIAudio},
			Variants: []ModelAPIDocsEndpointVariant{
				httpJSONVariant("sync", "Synchronous", "sync", "POST", EndpointChatCompletions, EndpointChatCompletions,
					`{"model":{{MODEL_JSON}},"messages":[{"role":"user","content":"Hello"}]}`,
					`{"id":"chatcmpl_...","choices":[{"message":{"role":"assistant","content":"Hello"}}]}`),
				httpSSEVariant("stream", "Streaming", "POST", EndpointChatCompletions, EndpointChatCompletions,
					`{"model":{{MODEL_JSON}},"messages":[{"role":"user","content":"Hello"}],"stream":true}`,
					"data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\ndata: [DONE]"),
			},
		},
		{
			ID: "anthropic_messages", Category: modelDocsCategoryConversation, Protocol: "anthropic_messages",
			Title: "Anthropic Messages", Description: "Create an Anthropic message with synchronous JSON or streaming SSE delivery.",
			protocols: []string{service.PlatformProtocolAnthropic, service.PlatformProtocolNative}, capability: service.PlatformCapabilityMessages,
			nativeSlugs: []string{service.PlatformAnthropic, service.PlatformAntigravity},
			Variants: []ModelAPIDocsEndpointVariant{
				httpJSONVariant("sync", "Synchronous", "sync", "POST", EndpointMessages, EndpointMessages,
					`{"model":{{MODEL_JSON}},"max_tokens":1024,"messages":[{"role":"user","content":"Hello"}]}`,
					`{"id":"msg_...","type":"message","content":[{"type":"text","text":"Hello"}]}`),
				httpSSEVariant("stream", "Streaming", "POST", EndpointMessages, EndpointMessages,
					`{"model":{{MODEL_JSON}},"max_tokens":1024,"messages":[{"role":"user","content":"Hello"}],"stream":true}`,
					"event: content_block_delta\ndata: {\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\nevent: message_stop"),
			},
		},
		{
			ID: "gemini_content", Category: modelDocsCategoryConversation, Protocol: "gemini_generate_content",
			Title: "Gemini Generate Content", Description: "Use the native Gemini content generation contract.",
			protocols: []string{service.PlatformProtocolGemini, service.PlatformProtocolNative}, capability: service.PlatformCapabilityNativeGemini,
			nativeSlugs: []string{service.PlatformGemini, service.PlatformAntigravity},
			Variants: []ModelAPIDocsEndpointVariant{
				httpJSONVariant("sync", "Synchronous", "sync", "POST", "/v1beta/models/{model}:generateContent", "/v1beta/models/*modelAction",
					`{"contents":[{"role":"user","parts":[{"text":"Hello"}]}]}`,
					`{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello"}]}}]}`),
				httpSSEVariant("stream", "Streaming", "POST", "/v1beta/models/{model}:streamGenerateContent?alt=sse", "/v1beta/models/*modelAction",
					`{"contents":[{"role":"user","parts":[{"text":"Hello"}]}]}`,
					"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello\"}]}}]}"),
			},
		},
	}
	profiles = append(profiles, openAIAdditionalProfiles()...)
	profiles = append(profiles, imageProfiles()...)
	profiles = append(profiles, videoProfiles()...)
	profiles = append(profiles, audioProfiles()...)
	profiles = append(profiles, volcengineProfiles()...)
	return completeModelAPIDocsProfileMetadata(profiles)
}

func openAIAdditionalProfiles() []ModelAPIDocsEndpointProfile {
	embeddings := ModelAPIDocsEndpointProfile{
		ID: "openai_embeddings", Category: modelDocsCategoryConversation, Protocol: "openai_embeddings",
		Title: "OpenAI Embeddings", Description: "Create vector embeddings with the OpenAI-compatible embeddings contract.",
		protocols: []string{service.PlatformProtocolOpenAI, service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityEmbeddings,
		Variants: []ModelAPIDocsEndpointVariant{
			httpJSONVariant("sync", "Synchronous", "sync", "POST", EndpointEmbeddings, EndpointEmbeddings,
				`{"model":{{MODEL_JSON}},"input":"Text to embed"}`,
				`{"object":"list","data":[{"object":"embedding","embedding":[0.0]}]}`),
		},
	}
	liveCall := ModelAPIDocsEndpointProfile{
		ID: "openai_live_call", Category: modelDocsCategoryConversation, Protocol: "openai_live",
		Title: "OpenAI Live Call", Description: "Create an official OpenAI Live call and receive the negotiated SDP answer.",
		platforms: []string{service.PlatformOpenAI}, protocols: []string{service.PlatformProtocolOpenAI}, capability: service.PlatformCapabilityRealtime,
		Variants: []ModelAPIDocsEndpointVariant{
			httpBinaryVariant("sync", "Create call", "POST", "/v1/live", "/v1/live",
				`{"sdp":"v=0...","session":{"model":{{MODEL_JSON}},"modalities":["audio","text"]}}`,
				"SDP answer in the response body; the Location header contains /v1/live/{call_id}."),
		},
	}
	liveSideband := ModelAPIDocsEndpointProfile{
		ID: "openai_live_sideband", Category: modelDocsCategoryConversation, Protocol: "openai_live",
		Title: "OpenAI Live Sideband", Description: "Attach a WebSocket sideband session to a previously created Live call.",
		platforms: []string{service.PlatformOpenAI}, protocols: []string{service.PlatformProtocolOpenAI}, capability: service.PlatformCapabilityRealtime,
		Variants: []ModelAPIDocsEndpointVariant{
			webSocketVariant("bidirectional", "Sideband", "bidirectional", "/v1/live/{call_id}", "/v1/live/:call_id",
				"Connect with the call ID returned in the Live Location header, then exchange realtime control events.",
				"Realtime server events and client control frames until either side closes the call."),
		},
	}
	return []ModelAPIDocsEndpointProfile{embeddings, liveCall, liveSideband}
}

func imageProfiles() []ModelAPIDocsEndpointProfile {
	base := func(id, title, path, routePath string) ModelAPIDocsEndpointProfile {
		return ModelAPIDocsEndpointProfile{
			ID: id, Category: modelDocsCategoryImage, Protocol: "openai_images", Title: title,
			Description: "Generate or edit an image with the OpenAI-compatible image contract.",
			protocols:   []string{service.PlatformProtocolOpenAI, service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityImages,
			Variants: []ModelAPIDocsEndpointVariant{httpJSONVariant("sync", "Synchronous", "sync", "POST", path, routePath,
				`{"model":{{MODEL_JSON}},"prompt":"A luminous city at night"}`,
				`{"created":0,"data":[{"url":"https://..."}]}`)},
		}
	}
	generation := base("openai_image_generation", "Image Generation", EndpointImagesGenerations, EndpointImagesGenerations)
	edit := base("openai_image_edit", "Image Edit", EndpointImagesEdits, EndpointImagesEdits)
	edit.Description = "Edit an image with the OpenAI-compatible image contract."
	edit.Variants = []ModelAPIDocsEndpointVariant{httpJSONVariant("sync", "Synchronous", "sync", "POST", EndpointImagesEdits, EndpointImagesEdits,
		`{"model":{{MODEL_JSON}},"prompt":"Refine the lighting","image":{"image_url":"https://example.com/input.png"}}`,
		`{"created":0,"data":[{"url":"https://..."}]}`)}
	asyncGeneration := ModelAPIDocsEndpointProfile{
		ID: "openai_image_generation_async", Category: modelDocsCategoryImage, Protocol: "openai_images", Title: "Asynchronous Image Generation",
		Description: "Submit an image task and poll it until completion.",
		protocols:   []string{service.PlatformProtocolOpenAI, service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityImages,
		Variants: []ModelAPIDocsEndpointVariant{asyncHTTPVariant("async", "Asynchronous", []ModelAPIDocsWorkflowStep{
			step("submit", "Submit task", "POST", EndpointImagesGenerations+"/async", EndpointImagesGenerations+"/async", "application/json", `{"model":{{MODEL_JSON}},"prompt":"A luminous city at night"}`, `{"task_id":"task_...","status":"pending"}`),
			step("status", "Get task", "GET", "/v1/images/tasks/{task_id}", "/v1/images/tasks/:task_id", "", "", `{"task_id":"task_...","status":"completed","data":[]}`),
		})},
	}
	asyncEdit := ModelAPIDocsEndpointProfile{
		ID: "openai_image_edit_async", Category: modelDocsCategoryImage, Protocol: "openai_images", Title: "Asynchronous Image Edit",
		Description: "Submit an image edit task and poll it until completion.",
		protocols:   []string{service.PlatformProtocolOpenAI, service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityImages,
		Variants: []ModelAPIDocsEndpointVariant{asyncHTTPVariant("async", "Asynchronous", []ModelAPIDocsWorkflowStep{
			step("submit", "Submit task", "POST", EndpointImagesEdits+"/async", EndpointImagesEdits+"/async", "application/json", `{"model":{{MODEL_JSON}},"prompt":"Refine the lighting","image":{"image_url":"https://example.com/input.png"}}`, `{"task_id":"task_...","status":"pending"}`),
			step("status", "Get task", "GET", "/v1/images/tasks/{task_id}", "/v1/images/tasks/:task_id", "", "", `{"task_id":"task_...","status":"completed","data":[]}`),
		})},
	}
	return []ModelAPIDocsEndpointProfile{generation, edit, asyncGeneration, asyncEdit}
}

func videoProfiles() []ModelAPIDocsEndpointProfile {
	makeProfile := func(id, title, createPath, createRoute string) ModelAPIDocsEndpointProfile {
		request := `{"model":{{MODEL_JSON}},"prompt":"Ocean waves at sunrise"}`
		if id == "openai_video_edit" || id == "openai_video_extension" {
			request = `{"model":{{MODEL_JSON}},"prompt":"Continue the scene","video":{"url":"https://example.com/input.mp4"}}`
		}
		return ModelAPIDocsEndpointProfile{
			ID: id, Category: modelDocsCategoryVideo, Protocol: "openai_videos", Title: title,
			Description: "Submit a video task, poll its status, and fetch the generated content.",
			platforms:   []string{service.PlatformGrok}, protocols: []string{service.PlatformProtocolOpenAICompatible}, capability: service.PlatformCapabilityVideos,
			Variants: []ModelAPIDocsEndpointVariant{asyncHTTPVariant("async", "Asynchronous", []ModelAPIDocsWorkflowStep{
				step("submit", "Submit task", "POST", createPath, createRoute, "application/json", request, `{"id":"video_...","status":"pending"}`),
				step("status", "Get status", "GET", "/v1/videos/{request_id}", "/v1/videos/:request_id", "", "", `{"id":"video_...","status":"completed"}`),
				step("content", "Get content", "GET", "/v1/videos/{request_id}/content", "/v1/videos/:request_id/content", "", "", "Binary video content"),
			})},
		}
	}
	makeXimoAIGrokVideoProfile := func(id, title, createPath, createRoute string) ModelAPIDocsEndpointProfile {
		request := `{"model":{{MODEL_JSON}},"prompt":"Ocean waves at sunrise","image":{"image_url":"https://example.com/input.png"}}`
		if id == "ximoai_grok_video_extension" {
			request = `{"model":{{MODEL_JSON}},"prompt":"Continue the scene","video":{"id":"video_..."}}`
		}
		return ModelAPIDocsEndpointProfile{
			ID: id, Category: modelDocsCategoryVideo, Protocol: "openai_videos", Title: title,
			Description: "Submit a video task through the XimoAI Grok video adapter and poll it until completion.",
			kinds:       []string{service.PlatformKindGrokVideo}, capability: service.PlatformCapabilityVideos,
			Variants: []ModelAPIDocsEndpointVariant{asyncHTTPVariant("async", "Asynchronous", []ModelAPIDocsWorkflowStep{
				step("submit", "Submit task", "POST", createPath, createRoute, "application/json", request, `{"id":"video_...","request_id":"video_...","status":"pending"}`),
				step("status", "Get status", "GET", "/v1/videos/{request_id}", "/v1/videos/:request_id", "", "", `{"id":"video_...","status":"completed","video_url":"https://..."}`),
			})},
		}
	}
	return []ModelAPIDocsEndpointProfile{
		makeProfile("openai_video_generation", "Video Generation", EndpointVideosGenerations, EndpointVideosGenerations),
		makeProfile("openai_video_edit", "Video Edit", EndpointVideosEdits, EndpointVideosEdits),
		makeProfile("openai_video_extension", "Video Extension", EndpointVideosExtensions, EndpointVideosExtensions),
		makeXimoAIGrokVideoProfile("ximoai_grok_video_generation", "Video Generation", EndpointVideosGenerations, EndpointVideosGenerations),
		makeXimoAIGrokVideoProfile("ximoai_grok_video_extension", "Video Extension", EndpointVideosExtensions, EndpointVideosExtensions),
	}
}

func audioProfiles() []ModelAPIDocsEndpointProfile {
	speech := ModelAPIDocsEndpointProfile{
		ID: "openai_audio_speech", Category: modelDocsCategoryTTS, Protocol: "openai_audio", Title: "Speech Synthesis",
		Description: "Generate speech audio with the OpenAI-compatible audio contract.",
		kinds:       []string{service.PlatformKindOpenAIAudio, service.PlatformKindKlingAudio}, capability: service.PlatformCapabilityAudio,
		Variants: []ModelAPIDocsEndpointVariant{httpBinaryVariant("sync", "Synchronous", "POST", EndpointAudioSpeech, EndpointAudioSpeech,
			`{"model":{{MODEL_JSON}},"input":"Hello from XimoAI","voice":"alloy"}`, "Binary audio content")},
	}
	transcription := ModelAPIDocsEndpointProfile{
		ID: "openai_audio_transcription", Category: modelDocsCategoryASR, Protocol: "openai_audio", Title: "Audio Transcription",
		Description: "Transcribe audio with the OpenAI-compatible audio contract.",
		kinds:       []string{service.PlatformKindOpenAIAudio}, capability: service.PlatformCapabilityAudio,
		Variants: []ModelAPIDocsEndpointVariant{httpMultipartVariant("sync", "Synchronous", "POST", EndpointAudioTranscribe, EndpointAudioTranscribe,
			"multipart/form-data: file=<audio>, model={{MODEL}}", `{"text":"Transcribed text"}`)},
	}
	translation := transcription
	translation.ID = "openai_audio_translation"
	translation.Title = "Audio Translation"
	translation.Description = "Translate audio into English text with the OpenAI-compatible audio contract."
	translation.Variants = []ModelAPIDocsEndpointVariant{httpMultipartVariant("sync", "Synchronous", "POST", EndpointAudioTranslate, EndpointAudioTranslate,
		"multipart/form-data: file=<audio>, model={{MODEL}}", `{"text":"Translated text"}`)}
	return []ModelAPIDocsEndpointProfile{speech, transcription, translation}
}

func volcengineProfiles() []ModelAPIDocsEndpointProfile {
	volc := func(id, category, title string, variant ModelAPIDocsEndpointVariant) ModelAPIDocsEndpointProfile {
		return ModelAPIDocsEndpointProfile{
			ID: id, Category: category, Protocol: "volcengine_agent_plan_native", Title: title,
			Description: "Use the native Volcengine Agent Plan frame or payload contract through the XimoAI gateway.",
			kinds:       []string{service.PlatformKindVolcengineAgentPlan},
			Variants:    []ModelAPIDocsEndpointVariant{variant},
		}
	}
	return []ModelAPIDocsEndpointProfile{
		volc("volcengine_images_generations", modelDocsCategoryImage, "Volcengine Seedream Image Generation",
			httpJSONVariant("sync", "Synchronous", "sync", "POST", "/v1/volcengine/images/generations", "/v1/volcengine/images/generations",
				`{"model":{{MODEL_JSON}},"prompt":"A luminous city at night"}`, `{"model":{{MODEL_JSON}},"data":[]}`)),
		volc("volcengine_tts_unidirectional", modelDocsCategoryTTS, "Volcengine TTS Unidirectional",
			httpBinaryVariant("sync", "Synchronous", "POST", "/v1/volcengine/audio/tts/unidirectional", "/v1/volcengine/audio/tts/unidirectional",
				`{"model":{{MODEL_JSON}},"text":"Hello from XimoAI"}`, "Provider-native audio response")),
		volc("volcengine_tts_unidirectional_stream", modelDocsCategoryTTS, "Volcengine TTS Unidirectional Stream",
			webSocketVariant("stream", "Unidirectional stream", "stream", "/v1/volcengine/audio/tts/unidirectional/stream", "/v1/volcengine/audio/tts/unidirectional/stream",
				"StartConnection -> StartSession -> send provider-native TTS request frames -> FinishSession.",
				"ConnectionStarted -> SessionStarted -> audio frames -> SessionFinished")),
		volc("volcengine_tts_bidirectional", modelDocsCategoryTTS, "Volcengine TTS Bidirectional Stream",
			webSocketVariant("bidirectional", "Bidirectional stream", "bidirectional", "/v1/volcengine/audio/tts/bidirection", "/v1/volcengine/audio/tts/bidirection",
				"StartConnection -> StartSession -> exchange provider-native text/control frames -> FinishSession.",
				"ConnectionStarted -> SessionStarted -> interleaved audio/control frames -> SessionFinished")),
		volc("volcengine_asr_bigmodel_async", modelDocsCategoryASR, "Volcengine Big Model ASR Async",
			webSocketVariant("async", "Asynchronous", "async", "/v1/volcengine/audio/asr/bigmodel_async", "/v1/volcengine/audio/asr/bigmodel_async",
				"StartConnection -> StartSession -> upload provider-native audio frames -> FinishSession.",
				"ConnectionStarted -> SessionStarted -> recognition updates -> final result")),
		volc("volcengine_asr_bigmodel_nostream", modelDocsCategoryASR, "Volcengine Big Model ASR Non-streaming Result",
			webSocketVariant("sync", "Non-streaming result", "sync", "/v1/volcengine/audio/asr/bigmodel_nostream", "/v1/volcengine/audio/asr/bigmodel_nostream",
				"StartConnection -> StartSession -> upload provider-native audio frames -> FinishSession.",
				"ConnectionStarted -> SessionStarted -> final recognition result -> SessionFinished")),
	}
}

func httpJSONVariant(id, label, mode, method, path, routePath, request, response string) ModelAPIDocsEndpointVariant {
	return ModelAPIDocsEndpointVariant{ID: id, Label: label, Mode: mode, Transport: "http", Delivery: "json", Termination: "The request completes after one final JSON response.", Steps: []ModelAPIDocsWorkflowStep{
		step("request", "Request", method, path, routePath, "application/json", request, response),
	}}
}

func httpSSEVariant(id, label, method, path, routePath, request, response string) ModelAPIDocsEndpointVariant {
	return ModelAPIDocsEndpointVariant{ID: id, Label: label, Mode: "stream", Transport: "http", Delivery: "sse", Termination: "Read events through the protocol terminal event or [DONE], then close the HTTP stream.", Steps: []ModelAPIDocsWorkflowStep{
		step("request", "Stream request", method, path, routePath, "application/json", request, response),
	}}
}

func httpBinaryVariant(id, label, method, path, routePath, request, response string) ModelAPIDocsEndpointVariant {
	return ModelAPIDocsEndpointVariant{ID: id, Label: label, Mode: "sync", Transport: "http", Delivery: "binary", Termination: "The response is complete when the HTTP response body reaches EOF.", Steps: []ModelAPIDocsWorkflowStep{
		step("request", "Request", method, path, routePath, "application/json", request, response),
	}}
}

func httpMultipartVariant(id, label, method, path, routePath, request, response string) ModelAPIDocsEndpointVariant {
	return ModelAPIDocsEndpointVariant{ID: id, Label: label, Mode: "sync", Transport: "http", Delivery: "json", Termination: "The request completes after one final JSON response.", Steps: []ModelAPIDocsWorkflowStep{
		step("request", "Request", method, path, routePath, "multipart/form-data", request, response),
	}}
}

func asyncHTTPVariant(id, label string, steps []ModelAPIDocsWorkflowStep) ModelAPIDocsEndpointVariant {
	return ModelAPIDocsEndpointVariant{ID: id, Label: label, Mode: "async", Transport: "http", Delivery: "json", Termination: "Submit once, poll until completed or failed, and retrieve the result when the workflow exposes a content step.", Steps: steps}
}

func webSocketVariant(id, label, mode, path, routePath, request, response string) ModelAPIDocsEndpointVariant {
	return ModelAPIDocsEndpointVariant{ID: id, Label: label, Mode: mode, Transport: "websocket", Delivery: "websocket_frames", Termination: "Keep the WebSocket open through the provider terminal frame, then close the session normally.", Steps: []ModelAPIDocsWorkflowStep{
		step("session", "WebSocket session", "GET", path, routePath, "", request, response),
	}}
}

func step(id, title, method, path, routePath, contentType, request, response string) ModelAPIDocsWorkflowStep {
	return ModelAPIDocsWorkflowStep{
		ID: id, Title: title, Method: method, Path: path, RoutePath: routePath,
		ContentType: contentType, RequestExample: request, ResponseExample: response,
		Parameters: []ModelAPIDocsParameter{{
			Name: "Authorization", Location: "header", Required: true, Type: "string",
			Description: "Bearer API key for the XimoAI gateway.",
		}},
	}
}

func completeModelAPIDocsProfileMetadata(profiles []ModelAPIDocsEndpointProfile) []ModelAPIDocsEndpointProfile {
	for profileIndex := range profiles {
		for variantIndex := range profiles[profileIndex].Variants {
			variant := &profiles[profileIndex].Variants[variantIndex]
			for stepIndex := range variant.Steps {
				workflowStep := &variant.Steps[stepIndex]
				for _, placeholder := range []string{"model", "task_id", "request_id", "call_id"} {
					if strings.Contains(workflowStep.Path, "{"+placeholder+"}") {
						appendModelDocsParameter(workflowStep, ModelAPIDocsParameter{
							Name: placeholder, Location: "path", Required: true, Type: "string",
							Description: modelDocsParameterDescription(placeholder),
						})
					}
				}
				if strings.Contains(workflowStep.Path, "alt=sse") {
					appendModelDocsParameter(workflowStep, ModelAPIDocsParameter{Name: "alt", Location: "query", Required: true, Type: "string", Description: "Set to sse for streaming delivery."})
				}
				if workflowStep.ContentType == "multipart/form-data" {
					appendModelDocsParameter(workflowStep, ModelAPIDocsParameter{Name: "file", Location: "form", Required: true, Type: "file", Description: "Audio or image file uploaded by the client."})
					appendModelDocsParameter(workflowStep, ModelAPIDocsParameter{Name: "model", Location: "form", Required: true, Type: "string", Description: modelDocsParameterDescription("model")})
				} else {
					for _, field := range []string{"model", "input", "messages", "max_tokens", "stream", "prompt", "image", "video", "voice", "text", "contents", "sdp", "session"} {
						if strings.Contains(workflowStep.RequestExample, `"`+field+`"`) {
							appendModelDocsParameter(workflowStep, ModelAPIDocsParameter{
								Name: field, Location: "body", Required: field != "stream", Type: modelDocsParameterType(field),
								Description: modelDocsParameterDescription(field),
							})
						}
					}
				}
				if variant.Transport == "websocket" {
					appendModelDocsParameter(workflowStep, ModelAPIDocsParameter{Name: "frames", Location: "frame", Required: true, Type: "provider-defined", Description: "Protocol frames exchanged after the WebSocket handshake."})
				}
			}
		}
	}
	return profiles
}

func appendModelDocsParameter(workflowStep *ModelAPIDocsWorkflowStep, parameter ModelAPIDocsParameter) {
	for _, current := range workflowStep.Parameters {
		if current.Name == parameter.Name && current.Location == parameter.Location {
			return
		}
	}
	workflowStep.Parameters = append(workflowStep.Parameters, parameter)
}

func modelDocsParameterType(name string) string {
	switch name {
	case "messages", "contents":
		return "array"
	case "stream":
		return "boolean"
	case "max_tokens":
		return "integer"
	case "image", "video", "session":
		return "object"
	default:
		return "string"
	}
}

func modelDocsParameterDescription(name string) string {
	switch name {
	case "model":
		return "Public model name shown in the XimoAI model plaza."
	case "task_id":
		return "Task identifier returned by the asynchronous submit step."
	case "request_id":
		return "Video request identifier returned by the submit step."
	case "call_id":
		return "Live call identifier returned in the Location header."
	case "stream":
		return "Enable streaming delivery when true."
	default:
		return "Protocol field required by this endpoint example."
	}
}

func resolveXimoAIModelAPIDocsAutomaticBinding(input modelAPIDocsResolutionInput) service.ModelAPIDocsBinding {
	input = normalizeModelAPIDocsResolutionInput(input)
	binding := service.ModelAPIDocsBinding{Platform: input.Platform, Protocol: input.Protocol, Model: input.Model, Categories: []service.ModelAPIDocsCategoryBinding{}}

	if input.Kind == service.PlatformKindVolcengineAgentPlan {
		wanted := map[string]struct{}{}
		switch input.Model {
		case service.VolcengineAgentPlanSeedreamModel:
			wanted["volcengine_images_generations"] = struct{}{}
		case service.VolcengineAgentPlanTTSModel:
			wanted["volcengine_tts_unidirectional"] = struct{}{}
			wanted["volcengine_tts_unidirectional_stream"] = struct{}{}
			wanted["volcengine_tts_bidirectional"] = struct{}{}
		case service.VolcengineAgentPlanASRModel:
			wanted["volcengine_asr_bigmodel_async"] = struct{}{}
			wanted["volcengine_asr_bigmodel_nostream"] = struct{}{}
		}
		return bindingFromProfiles(binding, compatibleModelAPIDocsProfiles(input), wanted)
	}

	categories := defaultModelAPIDocsCategories(input)
	wanted := make(map[string]struct{})
	for _, profile := range compatibleModelAPIDocsProfiles(input) {
		if _, ok := categories[profile.Category]; ok && modelAPIDocsProfileIsAutomatic(profile.ID) {
			wanted[profile.ID] = struct{}{}
		}
	}
	return bindingFromProfiles(binding, compatibleModelAPIDocsProfiles(input), wanted)
}

func modelAPIDocsProfileIsAutomatic(profileID string) bool {
	switch profileID {
	case "openai_responses_realtime", "openai_embeddings", "openai_live_call", "openai_live_sideband",
		"openai_image_edit", "openai_image_edit_async", "openai_video_edit", "openai_video_extension":
		return false
	default:
		return true
	}
}

func compatibleModelAPIDocsProfiles(input modelAPIDocsResolutionInput) []ModelAPIDocsEndpointProfile {
	input = normalizeModelAPIDocsResolutionInput(input)
	out := make([]ModelAPIDocsEndpointProfile, 0)
	for _, profile := range XimoAIModelAPIDocsProfiles() {
		if modelAPIDocsProfileCompatible(profile, input) {
			out = append(out, profile)
		}
	}
	return out
}

func modelAPIDocsProfileCompatible(profile ModelAPIDocsEndpointProfile, input modelAPIDocsResolutionInput) bool {
	kindScoped := containsFold(profile.kinds, input.Kind)
	if service.IsXimoAIMediaPlatformKind(input.Kind) && !kindScoped {
		return false
	}
	if kindScoped {
		if profile.capability != "" && !containsFold(input.Capabilities, profile.capability) {
			return false
		}
		return true
	}
	if len(profile.platforms) > 0 && !containsFold(profile.platforms, input.Platform) {
		return false
	}
	if !containsFold(profile.protocols, input.Protocol) {
		return false
	}
	if input.Protocol == service.PlatformProtocolNative && len(profile.nativeSlugs) > 0 && !containsFold(profile.nativeSlugs, input.Platform) {
		return false
	}
	if profile.capability != "" && !containsFold(input.Capabilities, profile.capability) {
		return false
	}
	return true
}

func defaultModelAPIDocsCategories(input modelAPIDocsResolutionInput) map[string]struct{} {
	out := make(map[string]struct{})
	for _, mode := range input.BillingModes {
		switch mode {
		case string(service.BillingModeToken):
			out[modelDocsCategoryConversation] = struct{}{}
		case string(service.BillingModeImage):
			out[modelDocsCategoryImage] = struct{}{}
		case string(service.BillingModeVideo):
			out[modelDocsCategoryVideo] = struct{}{}
		case string(service.BillingModePerRequest):
			if input.Kind == service.PlatformKindOpenAIAudio || input.Kind == service.PlatformKindKlingAudio {
				out[modelDocsCategoryTTS] = struct{}{}
				if input.Kind == service.PlatformKindOpenAIAudio {
					out[modelDocsCategoryASR] = struct{}{}
				}
			}
		}
	}
	if len(out) == 0 {
		if input.Kind == service.PlatformKindOpenAIAudio || input.Kind == service.PlatformKindKlingAudio {
			out[modelDocsCategoryTTS] = struct{}{}
			if input.Kind == service.PlatformKindOpenAIAudio {
				out[modelDocsCategoryASR] = struct{}{}
			}
		} else if len(input.BillingModes) == 0 {
			out[modelDocsCategoryConversation] = struct{}{}
		}
	}
	return out
}

func bindingFromProfiles(binding service.ModelAPIDocsBinding, profiles []ModelAPIDocsEndpointProfile, wanted map[string]struct{}) service.ModelAPIDocsBinding {
	categoryIndex := make(map[string]int)
	for _, profile := range profiles {
		if _, ok := wanted[profile.ID]; !ok {
			continue
		}
		index, ok := categoryIndex[profile.Category]
		if !ok {
			index = len(binding.Categories)
			categoryIndex[profile.Category] = index
			binding.Categories = append(binding.Categories, service.ModelAPIDocsCategoryBinding{Category: profile.Category, Endpoints: []service.ModelAPIDocsEndpointBinding{}})
		}
		variants := make([]string, 0, len(profile.Variants))
		for _, variant := range profile.Variants {
			variants = append(variants, variant.ID)
		}
		binding.Categories[index].Endpoints = append(binding.Categories[index].Endpoints, service.ModelAPIDocsEndpointBinding{Profile: profile.ID, Variants: variants})
	}
	return binding
}

func validateXimoAIModelAPIDocsBinding(binding service.ModelAPIDocsBinding, capabilities []string, kind string) error {
	input := normalizeModelAPIDocsResolutionInput(modelAPIDocsResolutionInput{
		Platform: binding.Platform, Kind: kind, Protocol: binding.Protocol, Model: binding.Model, Capabilities: capabilities,
	})
	if input.Platform == "" || input.Protocol == "" || input.Model == "" {
		return errors.New("platform, protocol, and model are required")
	}
	if len(binding.Categories) == 0 || len(binding.Categories) > 5 {
		return errors.New("one to five categories are required")
	}
	compatible := compatibleModelAPIDocsProfiles(input)
	profileByID := make(map[string]ModelAPIDocsEndpointProfile, len(compatible))
	for _, profile := range compatible {
		profileByID[profile.ID] = profile
	}
	seenCategories := make(map[string]struct{})
	seenProfiles := make(map[string]struct{})
	endpointCount := 0
	for _, category := range binding.Categories {
		categoryName := strings.ToLower(strings.TrimSpace(category.Category))
		if _, exists := seenCategories[categoryName]; exists {
			return fmt.Errorf("duplicate category %s", categoryName)
		}
		seenCategories[categoryName] = struct{}{}
		if len(category.Endpoints) == 0 {
			return fmt.Errorf("category %s has no endpoints", categoryName)
		}
		for _, endpoint := range category.Endpoints {
			endpointCount++
			profileID := strings.ToLower(strings.TrimSpace(endpoint.Profile))
			if _, exists := seenProfiles[profileID]; exists {
				return fmt.Errorf("duplicate profile %s", profileID)
			}
			seenProfiles[profileID] = struct{}{}
			profile, ok := profileByID[profileID]
			if !ok {
				return fmt.Errorf("profile %s is not compatible with the platform", profileID)
			}
			if profile.Category != categoryName {
				return fmt.Errorf("profile %s does not belong to category %s", profileID, categoryName)
			}
			if len(endpoint.Variants) == 0 {
				return fmt.Errorf("profile %s has no variants", profileID)
			}
			allowedVariants := make(map[string]struct{}, len(profile.Variants))
			for _, variant := range profile.Variants {
				allowedVariants[variant.ID] = struct{}{}
			}
			seenVariants := make(map[string]struct{})
			for _, variant := range endpoint.Variants {
				variant = strings.ToLower(strings.TrimSpace(variant))
				if _, ok := allowedVariants[variant]; !ok {
					return fmt.Errorf("unknown variant %s for profile %s", variant, profileID)
				}
				if _, duplicate := seenVariants[variant]; duplicate {
					return fmt.Errorf("duplicate variant %s for profile %s", variant, profileID)
				}
				seenVariants[variant] = struct{}{}
			}
		}
	}
	if endpointCount > 32 {
		return errors.New("too many endpoint profiles")
	}
	return nil
}

func selectedXimoAIModelAPIDocsProfiles(binding service.ModelAPIDocsBinding) []ModelAPIDocsEndpointProfile {
	all := XimoAIModelAPIDocsProfiles()
	profileByID := make(map[string]ModelAPIDocsEndpointProfile, len(all))
	for _, profile := range all {
		profileByID[profile.ID] = profile
	}
	out := make([]ModelAPIDocsEndpointProfile, 0)
	for _, category := range binding.Categories {
		for _, endpoint := range category.Endpoints {
			profile, ok := profileByID[endpoint.Profile]
			if !ok {
				continue
			}
			wanted := make(map[string]struct{}, len(endpoint.Variants))
			for _, variant := range endpoint.Variants {
				wanted[variant] = struct{}{}
			}
			variants := make([]ModelAPIDocsEndpointVariant, 0, len(profile.Variants))
			for _, variant := range profile.Variants {
				if _, ok := wanted[variant.ID]; ok {
					variants = append(variants, variant)
				}
			}
			profile.Variants = variants
			out = append(out, profile)
		}
	}
	return out
}

func modelAPIDocsBindingProfileIDs(binding service.ModelAPIDocsBinding) []string {
	out := make([]string, 0)
	for _, category := range binding.Categories {
		for _, endpoint := range category.Endpoints {
			out = append(out, endpoint.Profile)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeModelAPIDocsResolutionInput(input modelAPIDocsResolutionInput) modelAPIDocsResolutionInput {
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	if input.Kind == "" {
		input.Kind = service.XimoAIPlatformKindFromLegacySlug(input.Platform)
	}
	input.Protocol = strings.ToLower(strings.TrimSpace(input.Protocol))
	input.Model = strings.TrimSpace(input.Model)
	input.Capabilities = normalizeModelDocsStrings(input.Capabilities)
	input.BillingModes = normalizeModelDocsStrings(input.BillingModes)
	return input
}

func normalizeModelDocsStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}
