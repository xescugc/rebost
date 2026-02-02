package volume

import (
	"context"
	"time"

	"github.com/xescugc/rebost/uow"
)

// loopTTL will every second check if there is any file on the TTL table to be deleted
// if so it'll trigger a delete to the file
func (l *local) loopTTL() {
	tk := time.NewTicker(time.Second)
	for {
		select {
		case <-l.ctx.Done():
			goto end
		case <-tk.C:
			err := l.startUnitOfWork(l.ctx, uow.Write, func(ctx context.Context, uw uow.UnitOfWork) error {
				ttls, err := uw.IDXTTLs().Filter(l.ctx, time.Now())
				if err != nil {
					return err
				}
				if len(ttls) == 0 {
					return nil
				}

				for _, ttl := range ttls {
					for _, sig := range ttl.Signatures {
						dbf, err := uw.Files().FindBySignature(ctx, sig)
						if err != nil {
							l.logger.Log("msg", err.Error())
							continue
						}

						for _, k := range dbf.Keys {
							err = l.deleteFile(l.ctx, uw, k)
							if err != nil {
								l.logger.Log("msg", err.Error())
								continue
							}
						}
					}
				}
				return nil
			}, l.idxttls, l.files, l.idxkeys, l.files, l.fs, l.state)
			if err != nil {
				l.logger.Log("msg", err.Error())
			}
		}
	}
end:
	return
}
