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

		m, err := membership.New(&config.Config{MemberlistBindPort: 5000}, []volume.Volume{v}, "")
		require.NoError(t, err)
		assert.Equal(t, []volume.Volume{v}, m.Volumes())
	})
	t.Run("WithRemote", func(t *testing.T) {
		t.Run("Add", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolume(ctrl)

			v2 := mock.NewVolume(ctrl)
			m2, err := membership.New(&config.Config{MemberlistName: "am2", MemberlistBindPort: 5001}, []volume.Volume{v2}, "")
			require.NoError(t, err)
			s := storing.New(m2)
			_ = httptest.NewServer(storing.MakeHandler(s))

			m, err := membership.New(&config.Config{MemberlistName: "am", MemberlistBindPort: 5002}, []volume.Volume{v}, "0.0.0.0:5001")
			require.NoError(t, err)
			assert.Len(t, m.Volumes(), 2)
		})
		t.Run("Remove", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			v := mock.NewVolume(ctrl)

			v2 := mock.NewVolume(ctrl)
			m2, err := membership.New(&config.Config{MemberlistName: "rm2", MemberlistBindPort: 5003}, []volume.Volume{v2}, "")
			require.NoError(t, err)
			s := storing.New(m2)
			_ = httptest.NewServer(storing.MakeHandler(s))

			m, err := membership.New(&config.Config{MemberlistName: "rm", MemberlistBindPort: 5004}, []volume.Volume{v}, "0.0.0.0:5003")
			require.NoError(t, err)
			assert.Len(t, m.Volumes(), 2)

			m2.Leave()
			assert.Len(t, m.Volumes(), 1)
		})
	})
}
