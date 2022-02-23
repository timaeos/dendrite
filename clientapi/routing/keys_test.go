package routing

import (
	"encoding/json"
	"testing"

	"github.com/tidwall/sjson"
)

var validKeys = []byte(`{"device_keys":{"algorithms":["m.olm.v1.curve25519-aes-sha2","m.megolm.v1.aes-sha2"],"device_id":"JLAFKJWSCS","keys":{"curve25519:JLAFKJWSCS":"3C5BFWi2Y8MaVvjM8M22DBmh24PmgR0nPvJOIArzgyI","ed25519:JLAFKJWSCS":"lEuiRJBit0IG6nUf5pUzWTUEsRVVe/HJkoKuEww9ULI"},"signatures":{"@alice:example.com":{"ed25519:JLAFKJWSCS":"dSO80A01XiigH3uBiDVx/EjzaoycHcjq9lfQX0uWsqxl2giMIiSPR8a4d291W1ihKJL/a+myXS367WT6NAIcBA"}},"user_id":"@alice:example.com"}}`)

// required fields are missing
var (
	missingAlgorithms, _ = sjson.DeleteBytes(validKeys, "device_keys.algorithms")
	missingDeviceID, _   = sjson.DeleteBytes(missingAlgorithms, "device_keys.device_id")
	missingKeys, _       = sjson.DeleteBytes(missingDeviceID, "device_keys.keys")
	missingSignatures, _ = sjson.DeleteBytes(missingKeys, "device_keys.signatures")
	missingEverything, _ = sjson.DeleteBytes(missingKeys, "device_keys.user_id")
)

func Test_uploadKeysRequest_valid(t *testing.T) {
	type fields struct {
		DeviceKeys  json.RawMessage
		OneTimeKeys map[string]json.RawMessage
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "valid device keys",
			fields: fields{DeviceKeys: validKeys},
			want:   true,
		},
		{
			name:   "missing everything",
			fields: fields{DeviceKeys: missingEverything},
		},
		{
			name:   "missing signtures",
			fields: fields{DeviceKeys: missingSignatures},
		},
		{
			name:   "missing keys",
			fields: fields{DeviceKeys: missingKeys},
		},
		{
			name:   "missing device_id",
			fields: fields{DeviceKeys: missingDeviceID},
		},
		{
			name:   "missing algos",
			fields: fields{DeviceKeys: missingAlgorithms},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := uploadKeysRequest{
				DeviceKeys:  tt.fields.DeviceKeys,
				OneTimeKeys: tt.fields.OneTimeKeys,
			}
			if got := k.valid(); got != tt.want {
				t.Errorf("valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
