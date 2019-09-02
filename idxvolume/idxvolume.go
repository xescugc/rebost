package idxvolume

// IDXVolume represents the index of where copies are stored of the files
// in other volumes, this also stores replicas stored on this volume
type IDXVolume struct {
	VolumeID   string
	Signatures []string
}

// New returns a new IDXVolume with te Key and Value provided
func New(k string, v []string) *IDXVolume {
	return &IDXVolume{
		VolumeID:   k,
		Signatures: v,
	}
}

// AddSignature adds a new sig to the list of Signatures
// if the sig is already present it'll do nothing
func (idxv *IDXVolume) AddSignature(sig string) {
	for _, s := range idxv.Signatures {
		// If the Signature is already on the list we just skip
		if s == sig {
			return
		}
	}
	idxv.Signatures = append(idxv.Signatures, sig)
}
