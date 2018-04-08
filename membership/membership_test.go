package membership_test

import (
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xescugc/rebost/config"
	"github.com/xescugc/rebost/membership"
	"github.com/xescugc/rebost/mock"
	"github.com/xescugc/rebost/storing"
	"github.com/xescugc/rebost/volume"
)

func TestVolumes(t *testing.T) {
	t.Run("WithoutRemote", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		v := mock.NewVolume(ctrl)

		m, err := membership.New(&config.Config{MemberlistBindPort: 4000}, []volume.Volume{v}, "")
		require.NoError(t, err)
		assert.Len(t, m.RemoteVolumes(), 0)
		assert.Equal(t, []volume.Volume{v}, m.LocalVolumes())
	})
	t.Run("WithRemote", func(t *testing.T) {
		t.Run("Add", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolume(ctrl)

			v2 := mock.NewVolume(ctrl)
			m2, err := membership.New(&config.Config{MemberlistName: "am2", MemberlistBindPort: 4001}, []volume.Volume{v2}, "")
			require.NoError(t, err)
			s := storing.New(m2)
			server := httptest.NewServer(storing.MakeHandler(s))
			defer server.Close()

			m, err := membership.New(&config.Config{MemberlistName: "am", MemberlistBindPort: 4002}, []volume.Volume{v}, "0.0.0.0:4001")
			require.NoError(t, err)
			assert.Len(t, m.RemoteVolumes(), 1)
			assert.Equal(t, []volume.Volume{v}, m.LocalVolumes())
		})
		t.Run("Remove", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolume(ctrl)

			v2 := mock.NewVolume(ctrl)
			m2, err := membership.New(&config.Config{MemberlistName: "rm2", MemberlistBindPort: 4003}, []volume.Volume{v2}, "")
			require.NoError(t, err)
			s := storing.New(m2)
			server := httptest.NewServer(storing.MakeHandler(s))
			defer server.Close()

			m, err := membership.New(&config.Config{MemberlistName: "rm", MemberlistBindPort: 4004}, []volume.Volume{v}, "0.0.0.0:4003")
			require.NoError(t, err)
			assert.Len(t, m.RemoteVolumes(), 1)
			assert.Equal(t, []volume.Volume{v}, m.LocalVolumes())

			m2.Leave()
			assert.Len(t, m.RemoteVolumes(), 0)
			assert.Equal(t, []volume.Volume{v}, m.LocalVolumes())
		})
	})
}
