package safesearch

const (
	// StatusSupported indicates the provider's SafeSearch DNS CNAME rewrite mechanism is verified and supported.
	StatusSupported = "supported"
	// StatusUnsupported indicates the provider cannot be enforced via standard CNAME rewrites.
	StatusUnsupported = "unsupported"
)

// Provider represents a search engine or video platform's SafeSearch DNS policy definition.
type Provider struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Domains []string `json:"domains"`
	Target  string   `json:"target"`
	DocURL  string   `json:"docUrl"`
	Status  string   `json:"status"`
	Warning string   `json:"warning,omitempty"`
}
