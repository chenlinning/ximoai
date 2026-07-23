package service

// ximoAICustomPlatformPreviewAccount binds a temporary preview account to the
// server-side platform protocol. Client-supplied preview fields must not choose
// how an API key is authenticated upstream.
func ximoAICustomPlatformPreviewAccount(account *Account, platform *Platform) *Account {
	if account == nil || platform == nil {
		return account
	}

	normalized := *account
	normalized.Credentials = make(map[string]any, len(account.Credentials)+1)
	for key, value := range account.Credentials {
		normalized.Credentials[key] = value
	}
	normalized.Credentials["platform_protocol"] = platform.Protocol
	return &normalized
}
