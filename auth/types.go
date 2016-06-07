package auth

const (
	AUTH_METHOD_BASIC = iota     // Authenticate using the (long-term) public key
	AUTH_METHOD_DAGA             // Authenticate using deniable anonymous group authentication (DAGA)
	AUTH_METHOD_TOFU             // Trust-On-First-Use (TOFU) authentication
)
