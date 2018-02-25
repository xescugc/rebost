package file

import "path"

// File represents the structure of a stored File with the Signature (SHA1 of the content of the File)
// and the key which is the name of the file
type File struct {
	Keys      []string
	Signature string
	Replicas  []string
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
