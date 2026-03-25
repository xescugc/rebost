package volume

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"time"

	"github.com/xescugc/rebost/file"
	"github.com/xescugc/rebost/scrub"
	"github.com/xescugc/rebost/uow"
)

// loopScrub ticks every scrubInterval and checksums all local files,
// enqueuing a repair job for any that no longer match their stored signature.
func (l *local) loopScrub() {
	if l.scrubInterval <= 0 {
		return
	}
	tk := time.NewTicker(l.scrubInterval)
	defer tk.Stop()
	for {
		select {
		case <-l.ctx.Done():
			return
		case <-tk.C:
			l.runScrub()
		}
	}
}

func (l *local) runScrub() {
	// Phase 1: snapshot all file records in a single read transaction.
	var files []*file.File
	if err := l.startUnitOfWork(l.ctx, uow.Read, func(ctx context.Context, uw uow.UnitOfWork) error {
		var err error
		files, err = uw.Files().All(ctx)
		return err
	}); err != nil {
		l.logger.Error("scrub: failed to list files", "err", err)
		return
	}

	// Phase 2: checksum each file outside of a transaction.
	for _, f := range files {
		l.scrubFile(f)
	}
}

func (l *local) scrubFile(f *file.File) {
	fh, err := l.fs.Open(file.Path(l.fileDir, f.Signature))
	if err != nil {
		l.logger.Error("scrub: cannot open file", "signature", f.Signature, "err", err)
		return
	}
	defer fh.Close()

	h := sha1.New()
	if _, err := io.Copy(h, fh); err != nil {
		l.logger.Error("scrub: cannot read file", "signature", f.Signature, "err", err)
		return
	}
	computed := fmt.Sprintf("%x", h.Sum(nil))

	if computed == f.Signature {
		return // healthy
	}

	l.logger.Error("scrub: checksum mismatch detected", "key", f.Keys[0], "signature", f.Signature, "computed", computed)

	// Filter out the local volume ID from the repair candidates.
	remoteVIDs := make([]string, 0, len(f.VolumeIDs))
	for _, vid := range f.VolumeIDs {
		if vid != l.id {
			remoteVIDs = append(remoteVIDs, vid)
		}
	}

	if err := l.startUnitOfWork(l.ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
		return uw.Scrubs().Create(ctx, scrub.Scrub{
			Key:       f.Keys[0],
			Signature: f.Signature,
			VolumeIDs: remoteVIDs,
		})
	}); err != nil {
		l.logger.Error("scrub: failed to enqueue repair job", "key", f.Keys[0], "err", err)
	}
}
