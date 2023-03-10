package file

import "path"

// File represents the structure of a stored File with the Signature (SHA1 of the content of the File)
// and the key which is the name of the file
type File struct {
	// Keys has all the keys that point to this file
	Keys []string

	// Signature is the SHA1 of the file
	Signature string

	// Replica number of replicas for that file
	Replica int

	// VolumeIDs it's where this file it's replicated to
	VolumeIDs []string

	// Is the size of the object in bytes
	Size int
}

// Path calculates the storage path for the File with the Signature
func (f File) Path(p string) string {
	return Path(p, f.Signature)
}

// Path calculates the storage path for the File with the Signature
func Path(base, sig string) string {
	currentDir := []byte{}
	for _, b := range []byte(sig) {
		currentDir = append(currentDir, b)
		if len(currentDir) == 2 {
			base = path.Join(base, string(currentDir))
			currentDir = []byte{}
		}
	}
	return base
}

// DeleteVolumeID removes the vid from the f.VolumeIDs
// if it does not exists it'll do nothing
func (f *File) DeleteVolumeID(vid string) {
	// TODO: Make this more optimal, without oprating over
	// all the slice, just until we find the vid
	vids := make([]string, 0, len(f.VolumeIDs)-1)
	for _, v := range f.VolumeIDs {
		if v == vid {
			continue
		}
		vids = append(vids, v)
	}
	f.VolumeIDs = vids
}
