package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func openAIChannelRoutingModel(requestModel string, channelMapping service.ChannelMappingResult) string {
	if channelMapping.Mapped && strings.TrimSpace(channelMapping.MappedModel) != "" {
		return strings.TrimSpace(channelMapping.MappedModel)
	}
	return requestModel
}
