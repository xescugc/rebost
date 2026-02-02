package idxttl

import "time"

// IDXTTL holds the list of Signatures that need to expire at the given ExpiresAt
type IDXTTL struct {
	ExpiresAt  time.Time
	Signatures []string
}

// New initializes a new IDXTTL to expire all the ss at ea
func New(ea time.Time, ss ...string) *IDXTTL {
	ittl := &IDXTTL{
		ExpiresAt: ea,
	}

	ittl.AddSignatures(ss...)

	return ittl
}

// AddSignatures will add all the s to the list of
// signatures to expire on the ExpiresAt if they do
// not exists already
func (i *IDXTTL) AddSignatures(ss ...string) {
	mapSig := make(map[string]struct{})
	for _, s := range i.Signatures {
		mapSig[s] = struct{}{}
	}
	for _, s := range ss {
		if _, ok := mapSig[s]; !ok {
			i.Signatures = append(i.Signatures, s)
			mapSig[s] = struct{}{}
		}
	}
}
