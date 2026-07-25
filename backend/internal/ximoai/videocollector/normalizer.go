package videocollector

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

var ErrNoDownloadableFormats = errors.New("no downloadable media formats found")

type rawMediaInfo struct {
	ID           string           `json:"id"`
	WebpageURL   string           `json:"webpage_url"`
	OriginalURL  string           `json:"original_url"`
	Title        string           `json:"title"`
	Uploader     string           `json:"uploader"`
	UploaderID   string           `json:"uploader_id"`
	Thumbnail    string           `json:"thumbnail"`
	Duration     float64          `json:"duration"`
	Extractor    string           `json:"extractor"`
	ExtractorKey string           `json:"extractor_key"`
	Formats      []rawMediaFormat `json:"formats"`
}

type rawMediaFormat struct {
	ID             string  `json:"format_id"`
	Extension      string  `json:"ext"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	VideoCodec     string  `json:"vcodec"`
	AudioCodec     string  `json:"acodec"`
	FileSize       float64 `json:"filesize"`
	FileSizeApprox float64 `json:"filesize_approx"`
	TotalBitrate   float64 `json:"tbr"`
	AudioBitrate   float64 `json:"abr"`
}

func NormalizeMediaInfo(payload json.RawMessage, sourceURL string) (*MediaInfo, error) {
	var raw rawMediaInfo
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, errors.New("media extractor returned invalid metadata")
	}
	formats := make([]MediaFormat, 0, len(raw.Formats))
	for _, candidate := range raw.Formats {
		videoCodec := normalizedCodec(candidate.VideoCodec)
		audioCodec := normalizedCodec(candidate.AudioCodec)
		if videoCodec == "" && audioCodec == "" {
			continue
		}
		size := candidate.FileSize
		if size <= 0 {
			size = candidate.FileSizeApprox
		}
		bitrate := candidate.TotalBitrate
		if bitrate <= 0 {
			bitrate = candidate.AudioBitrate
		}
		formats = append(formats, MediaFormat{
			ID:               fallback(strings.TrimSpace(candidate.ID), "unknown"),
			Extension:        fallback(strings.TrimSpace(candidate.Extension), defaultExtension(videoCodec != "")),
			Width:            candidate.Width,
			Height:           candidate.Height,
			VideoCodec:       videoCodec,
			AudioCodec:       audioCodec,
			ApproximateBytes: int64(size),
			BitrateKbps:      int64(bitrate),
			HasVideo:         videoCodec != "",
			HasAudio:         audioCodec != "",
		})
	}
	if len(formats) == 0 {
		return nil, ErrNoDownloadableFormats
	}
	sort.SliceStable(formats, func(i, j int) bool {
		left, right := formats[i], formats[j]
		if left.HasVideo != right.HasVideo {
			return left.HasVideo
		}
		if left.Height != right.Height {
			return left.Height > right.Height
		}
		if left.HasAudio != right.HasAudio {
			return left.HasAudio
		}
		return left.BitrateKbps > right.BitrateKbps
	})

	resolvedSource := firstNonEmpty(raw.WebpageURL, raw.OriginalURL, sourceURL)
	return &MediaInfo{
		ID:        fallback(strings.TrimSpace(raw.ID), "unknown"),
		SourceURL: resolvedSource,
		Title:     fallback(strings.TrimSpace(raw.Title), "Untitled video"),
		Uploader:  fallback(firstNonEmpty(raw.Uploader, raw.UploaderID), "Unknown creator"),
		Thumbnail: strings.TrimSpace(raw.Thumbnail),
		Duration:  raw.Duration,
		Extractor: fallback(firstNonEmpty(raw.Extractor, raw.ExtractorKey), "Unknown platform"),
		Formats:   formats,
	}, nil
}

func normalizedCodec(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "none") {
		return ""
	}
	return value
}

func defaultExtension(hasVideo bool) string {
	if hasVideo {
		return "mp4"
	}
	return "m4a"
}

func fallback(value, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
