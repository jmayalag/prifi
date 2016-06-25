package auth

const (
	AUTH_METHOD_BASIC = iota // Basic public key authentication
	AUTH_METHOD_DAGA         // Denial Anonymous Group Authentication (DAGA)
	AUTH_METHOD_TOFU         // Trust-On-First-Use (TOFU) authentication
)
