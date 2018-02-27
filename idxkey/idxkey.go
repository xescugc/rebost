package idxkey

// IDXKey represents a KV of Key => File.Key and Value => File.Signature
type IDXKey struct {
	Key   string
	Value string
}

// New returns anew IDXKey with te Key and Value provided
func New(k, v string) *IDXKey {
	return &IDXKey{
		Key:   k,
		Value: v,
	}
}
