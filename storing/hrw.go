package storing

import (
	"hash/fnv"
	"sort"
)

// rankVolumeIDs returns vids sorted descending by HRW score for key.
// The first element is the natural owner of the key.
func rankVolumeIDs(key string, vids []string) []string {
	type scored struct {
		vid   string
		score uint64
	}
	scores := make([]scored, len(vids))
	for i, vid := range vids {
		h := fnv.New64a()
		h.Write([]byte(key))
		h.Write([]byte(vid))
		scores[i] = scored{vid, h.Sum64()}
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	ranked := make([]string, len(vids))
	for i, s := range scores {
		ranked[i] = s.vid
	}
	return ranked
}
