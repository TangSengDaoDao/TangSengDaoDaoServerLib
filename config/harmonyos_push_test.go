package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestHarmonyOSPushConfiguration(t *testing.T) {
	cfg := New()
	require.Equal(t, HarmonyOSPush{}, cfg.Push.HARMONYOS)
	vp := viper.New()
	vp.SetConfigType("yaml")
	require.NoError(t, vp.ReadConfig(strings.NewReader(`push:
  harmonyos:
    bundleID: com.example.app
    serviceAccountFile: /run/secrets/push.json
    category: IM
    testMessage: true
`)))
	cfg.ConfigureWithViper(vp)
	require.Equal(t, HarmonyOSPush{BundleID: "com.example.app", ServiceAccountFile: "/run/secrets/push.json", Category: "IM", TestMessage: true}, cfg.Push.HARMONYOS)
	vp.SetEnvPrefix("TS")
	vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	vp.AutomaticEnv()
	t.Setenv("TS_PUSH_HARMONYOS_BUNDLEID", "com.example.env")
	t.Setenv("TS_PUSH_HARMONYOS_TESTMESSAGE", "false")
	cfg.ConfigureWithViper(vp)
	require.Equal(t, "com.example.env", cfg.Push.HARMONYOS.BundleID)
	require.False(t, cfg.Push.HARMONYOS.TestMessage)
	vp.Set("push.harmonyos.bundleID", "com.example.memory")
	cfg.ConfigureWithViper(vp)
	require.Equal(t, "com.example.memory", cfg.Push.HARMONYOS.BundleID)
}
