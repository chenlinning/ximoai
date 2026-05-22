package service

var ximoaiOpenAICompatibleRequestHeaders = []string{
	"idempotency-key",
	"openai-organization",
	"openai-project",
	"openai-version",
	"x-request-id",
	"x-stainless-arch",
	"x-stainless-connect-timeout",
	"x-stainless-lang",
	"x-stainless-os",
	"x-stainless-package-version",
	"x-stainless-read-timeout",
	"x-stainless-retry-count",
	"x-stainless-runtime",
	"x-stainless-runtime-version",
	"x-stainless-timeout",
}

func init() {
	for _, header := range ximoaiOpenAICompatibleRequestHeaders {
		openaiAllowedHeaders[header] = true
		openaiPassthroughAllowedHeaders[header] = true
		openaiCCRawAllowedHeaders[header] = true
	}
}
