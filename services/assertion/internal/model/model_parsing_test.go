package model_test

import (
	"testing"

	"github.com/idlab-discover/kebeng/services/assertion/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

// same helper for SlotMap
func assertSlotMapEqual(t *testing.T, expected, actual model.SlotMap) {
	assert.Equal(t, len(expected), len(actual), "map length mismatch")
	for iface, exp := range expected {
		act, ok := actual[iface]
		require.True(t, ok, "missing interface %q", iface)

		if exp.AllowInstallation != nil {
			require.NotNil(t, act.AllowInstallation)
			assert.Equal(t, *exp.AllowInstallation, *act.AllowInstallation)
		} else {
			assert.Nil(t, act.AllowInstallation)
		}
		if exp.DenyInstallation != nil {
			require.NotNil(t, act.DenyInstallation)
			assert.Equal(t, *exp.DenyInstallation, *act.DenyInstallation)
		} else {
			assert.Nil(t, act.DenyInstallation)
		}
		if exp.AllowConnection != nil {
			require.NotNil(t, act.AllowConnection)
			assert.Equal(t, *exp.AllowConnection, *act.AllowConnection)
		} else {
			assert.Nil(t, act.AllowConnection)
		}
		if exp.DenyConnection != nil {
			require.NotNil(t, act.DenyConnection)
			assert.Equal(t, *exp.DenyConnection, *act.DenyConnection)
		} else {
			assert.Nil(t, act.DenyConnection)
		}
		if exp.AllowAutoConnection != nil {
			require.NotNil(t, act.AllowAutoConnection)
			assert.Equal(t, *exp.AllowAutoConnection, *act.AllowAutoConnection)
		} else {
			assert.Nil(t, act.AllowAutoConnection)
		}
		if exp.DenyAutoConnection != nil {
			require.NotNil(t, act.DenyAutoConnection)
			assert.Equal(t, *exp.DenyAutoConnection, *act.DenyAutoConnection)
		} else {
			assert.Nil(t, act.DenyAutoConnection)
		}
	}
}

func TestSlotMap_ScanAndValue(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		expect    model.SlotMap
	}{
		{
			name:      "single boolean field",
			jsonInput: `{"dbus":{"allow-connection":true}}`,
			expect: model.SlotMap{
				"dbus": {AllowConnection: boolPtr(true)},
			},
		},
		{
			name:      "multiple fields",
			jsonInput: `{"foo":{"deny-installation":true,"deny-auto-connection":false}}`,
			expect: model.SlotMap{
				"foo": {
					DenyInstallation:   boolPtr(true),
					DenyAutoConnection: boolPtr(false),
				},
			},
		},
		{
			name:      "empty map",
			jsonInput: `{}`,
			expect:    model.SlotMap{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name+"/Scan", func(t *testing.T) {
			var sm model.SlotMap
			err := sm.Scan([]byte(tc.jsonInput))
			require.NoError(t, err)
			assertSlotMapEqual(t, tc.expect, sm)
		})

		t.Run(tc.name+"/Value+Scan", func(t *testing.T) {
			val, err := tc.expect.Value()
			require.NoError(t, err)
			bs, ok := val.([]byte)
			require.True(t, ok)

			var sm2 model.SlotMap
			err = sm2.Scan(bs)
			require.NoError(t, err)
			assertSlotMapEqual(t, tc.expect, sm2)
		})
	}
}
