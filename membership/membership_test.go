package membership_test

import (
	"net/http/httptest"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/util"
	"github.com/xescugc/rebost/volume"
)

func TestVolumes(t *testing.T) {
	t.Run("WithoutNodes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		v := mock.NewVolumeLocal(ctrl)
		v.EXPECT().ID().Return("id")

		p, err := util.FreePort()
		require.NoError(t, err)

		m, err := membership.New(&config.Config{MemberlistBindPort: p}, []volume.Local{v}, "", kitlog.NewNopLogger())
		require.NoError(t, err)
		assert.Len(t, m.Nodes(), 0)
		assert.Equal(t, []volume.Local{v}, m.LocalVolumes())
		assert.Equal(t, []string{}, m.RemovedVolumeIDs())
	})
	t.Run("WithNodes", func(t *testing.T) {
		t.Run("Add", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolumeLocal(ctrl)
			v.EXPECT().ID().Return("id")

			v2 := mock.NewVolumeLocal(ctrl)
			v2.EXPECT().ID().Return("id")
			p2, err := util.FreePort()
			require.NoError(t, err)
			cfg2 := &config.Config{MemberlistName: "am2", Replica: -1, MemberlistBindPort: p2}
			m2, err := membership.New(cfg2, []volume.Local{v2}, "", kitlog.NewNopLogger())
			require.NoError(t, err)

			s := storing.New(cfg2, m2)
			server := httptest.NewServer(storing.MakeHandler(s))
			defer server.Close()

			p3, err := util.FreePort()
			require.NoError(t, err)
			cfg := &config.Config{MemberlistName: "am", MemberlistBindPort: p3}
			m, err := membership.New(cfg, []volume.Local{v}, server.URL, kitlog.NewNopLogger())
			require.NoError(t, err)
			assert.Len(t, m.Nodes(), 1)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())
			assert.Equal(t, []string{}, m.RemovedVolumeIDs())
		})
		t.Run("Remove", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolumeLocal(ctrl)
			v.EXPECT().ID().Return("id")

			v2 := mock.NewVolumeLocal(ctrl)
			v2.EXPECT().ID().Return("id2")
			p2, err := util.FreePort()
			require.NoError(t, err)
			cfg2 := &config.Config{MemberlistName: "rm2", Replica: -1, MemberlistBindPort: p2}
			m2, err := membership.New(cfg2, []volume.Local{v2}, "", kitlog.NewNopLogger())
			require.NoError(t, err)
			s := storing.New(cfg2, m2)
			server := httptest.NewServer(storing.MakeHandler(s))
			defer server.Close()

			p3, err := util.FreePort()
			require.NoError(t, err)
			cfg := &config.Config{MemberlistName: "rm", MemberlistBindPort: p3}
			m, err := membership.New(cfg, []volume.Local{v}, server.URL, kitlog.NewNopLogger())
			require.NoError(t, err)
			assert.Len(t, m.Nodes(), 1)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())

			m2.Leave()
			assert.Len(t, m.Nodes(), 0)
			assert.Equal(t, []volume.Local{v}, m.LocalVolumes())
			assert.Equal(t, []string{"id2"}, m.RemovedVolumeIDs())
			assert.Equal(t, []string{}, m.RemovedVolumeIDs())
		})
	})
}
