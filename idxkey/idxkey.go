package idxkey

type IDXKey struct {
	Key   string
	Value string
}

func New(k, v string) *IDXKey {
	return &IDXKey{
		Key:   k,
		Value: v,
	}
}
