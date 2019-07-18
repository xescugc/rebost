package internal_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/replica"
	"github.com/xescugc/rebost/uow"
	"github.com/xescugc/rebost/volume"
)

func TestNewReplica(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		var suow uow.StartUnitOfWork

		rp := mock.NewReplicaPendentRepository(ctrl)
		rr := mock.NewReplicaRetryRepository(ctrl)
		r := volume.NewReplica(rp, rr, suow)
		assert.NotNil(t, r)
	})
}

func TestReplicasPendent(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl  = gomock.NewController(t)
			rp    = mock.NewReplicaPendentRepository(ctrl)
			rr    = mock.NewReplicaRetryRepository(ctrl)
			uowFn = func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
				uw := mock.NewUnitOfWork(ctrl)
				uw.EXPECT().ReplicaPendent().Return(rp).AnyTimes()
				return uowFn(ctx, uw)
			}
			erp = replica.Pendent{
				ID:              "1",
				Replica:         1,
				Key:             "key",
				Signature:       "e7e8c72d1167454b76a610074fed244be0935298",
				VolumeID:        "12",
				VolumeReplicaID: []byte("123"),
			}
		)
		defer ctrl.Finish()

		r := volume.NewReplica(rp, rr, uowFn)

		rp.EXPECT().First(gomock.Any()).Return(&erp, nil)
		rp.EXPECT().Delete(gomock.Any(), &erp).Return(nil)

		// Consume the chanel as it's not buffered
		// and it would block the execution on
		// the PopFromReplicaPendent
		go func() {
			rp := <-r.ReplicasPendent()
			assert.Equal(t, erp, rp)
		}()

		err := r.PopFromReplicaPendent(context.Background())
		require.NoError(t, err)
	})
}

func TestReplicasRetry(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl  = gomock.NewController(t)
			rp    = mock.NewReplicaPendentRepository(ctrl)
			rr    = mock.NewReplicaRetryRepository(ctrl)
			uowFn = func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
				uw := mock.NewUnitOfWork(ctrl)
				uw.EXPECT().ReplicaRetry().Return(rr).AnyTimes()
				return uowFn(ctx, uw)
			}
			eRr = replica.Retry{
				NodeName: "Name",
			}
		)
		defer ctrl.Finish()

		r := volume.NewReplica(rp, rr, uowFn)

		rr.EXPECT().First(gomock.Any()).Return(&eRr, nil)
		rr.EXPECT().Delete(gomock.Any(), &eRr).Return(nil)

		// Consume the chanel as it's not buffered
		// and it would block the execution on
		// the PopFromReplicaRetry
		go func() {
			rp := <-r.ReplicasRetry()
			assert.Equal(t, eRr, rp)
		}()

		err := r.PopFromReplicaRetry(context.Background())
		require.NoError(t, err)
	})
}

func TestPopFromReplicaPendent(t *testing.T) {
	t.Run("SuccessWithMultipleReplicas", func(t *testing.T) {
		var (
			ctrl  = gomock.NewController(t)
			rp    = mock.NewReplicaPendentRepository(ctrl)
			rr    = mock.NewReplicaRetryRepository(ctrl)
			uowFn = func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
				uw := mock.NewUnitOfWork(ctrl)
				uw.EXPECT().ReplicaPendent().Return(rp).AnyTimes()
				return uowFn(ctx, uw)
			}
			erp = replica.Pendent{
				ID:              "1",
				Replica:         2,
				Key:             "key",
				Signature:       "e7e8c72d1167454b76a610074fed244be0935298",
				VolumeID:        "12",
				VolumeReplicaID: []byte("123"),
			}
		)
		defer ctrl.Finish()

		r := volume.NewReplica(rp, rr, uowFn)

		rp.EXPECT().First(gomock.Any()).Return(&erp, nil)
		rp.EXPECT().Delete(gomock.Any(), &erp).Return(nil)
		rp.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, rp *replica.Pendent) error {
			_, err := uuid.FromString(string(rp.ID))
			require.NoError(t, err, "Validates that it's a UUID")
			assert.NotEqual(t, erp.ID, rp.ID, "A new ID has to be generated for it")
			rp.ID = erp.ID

			assert.NotNil(t, rp.VolumeReplicaID)

			// Comparing with a number because as it's a pointer
			// the erp.Replica is already 1 (for the logic on the PopFromReplicaPendent)
			// so to be more clear just use the number directly
			assert.Equal(t, rp.Replica, 1, "The replica needs to be one less than before")
			rp.Replica = erp.Replica

			assert.NotNil(t, rp.VolumeReplicaID)
			assert.NotEqual(t, erp.VolumeReplicaID, rp.VolumeReplicaID, "The internal volume replica ID has to be different")
			rp.VolumeReplicaID = erp.VolumeReplicaID

			assert.Equal(t, &erp, rp)

			return nil
		})

		// Consume the chanel as it's not buffered
		// and it would block the execution on
		// the PopFromReplicaPendent
		go func() { <-r.ReplicasPendent() }()

		err := r.PopFromReplicaPendent(context.Background())
		require.NoError(t, err)
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			ctrl  = gomock.NewController(t)
			rp    = mock.NewReplicaPendentRepository(ctrl)
			rr    = mock.NewReplicaRetryRepository(ctrl)
			uowFn = func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
				uw := mock.NewUnitOfWork(ctrl)
				uw.EXPECT().ReplicaPendent().Return(rp).AnyTimes()
				return uowFn(ctx, uw)
			}
		)
		defer ctrl.Finish()

		r := volume.NewReplica(rp, rr, uowFn)

		rp.EXPECT().First(gomock.Any()).Return(nil, errors.New("not found"))

		err := r.PopFromReplicaPendent(context.Background())
		require.NoError(t, err)
	})
}

func TestPopFromReplicaRetry(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl  = gomock.NewController(t)
			rp    = mock.NewReplicaPendentRepository(ctrl)
			rr    = mock.NewReplicaRetryRepository(ctrl)
			uowFn = func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
				uw := mock.NewUnitOfWork(ctrl)
				uw.EXPECT().ReplicaRetry().Return(rr).AnyTimes()
				return uowFn(ctx, uw)
			}
			erp = replica.Retry{
				NodeName: "Pepe",
			}
		)
		defer ctrl.Finish()

		r := volume.NewReplica(rp, rr, uowFn)

		rr.EXPECT().First(gomock.Any()).Return(&erp, nil)
		rr.EXPECT().Delete(gomock.Any(), &erp).Return(nil)

		// Consume the chanel as it's not buffered
		// and it would block the execution on
		// the PopFromReplicaPendent
		go func() { <-r.ReplicasRetry() }()

		err := r.PopFromReplicaRetry(context.Background())
		require.NoError(t, err)
	})
	t.Run("NotFound", func(t *testing.T) {
		var (
			ctrl  = gomock.NewController(t)
			rp    = mock.NewReplicaPendentRepository(ctrl)
			rr    = mock.NewReplicaRetryRepository(ctrl)
			uowFn = func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
				uw := mock.NewUnitOfWork(ctrl)
				uw.EXPECT().ReplicaRetry().Return(rr).AnyTimes()
				return uowFn(ctx, uw)
			}
		)
		defer ctrl.Finish()

		r := volume.NewReplica(rp, rr, uowFn)

		rr.EXPECT().First(gomock.Any()).Return(nil, errors.New("not found"))

		err := r.PopFromReplicaRetry(context.Background())
		require.NoError(t, err)
	})
}

func TestCreateReplicaRetry(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			ctrl  = gomock.NewController(t)
			rp    = mock.NewReplicaPendentRepository(ctrl)
			rr    = mock.NewReplicaRetryRepository(ctrl)
			uowFn = func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, repositories ...interface{}) error {
				uw := mock.NewUnitOfWork(ctrl)
				uw.EXPECT().ReplicaRetry().Return(rr).AnyTimes()
				return uowFn(ctx, uw)
			}
			eRr = &replica.Retry{
				NodeName: "pepe",
			}
		)
		defer ctrl.Finish()

		r := volume.NewReplica(rp, rr, uowFn)

		rr.EXPECT().Create(gomock.Any(), eRr).Return(nil)

		err := r.CreateReplicaRetry(context.Background(), eRr)
		require.NoError(t, err)
	})
}
