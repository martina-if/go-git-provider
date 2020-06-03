package providers

type Provider interface {
	authorizeSSHKey(key []byte) error
	deleteSSHKey(keyId string) error
}

// GetProvider returns the appropriate provider for the URL
func GetProvider(repoURL string) Provider {
	return nil
}
