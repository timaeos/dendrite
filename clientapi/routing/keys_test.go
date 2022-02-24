package routing

import (
	"encoding/json"
	"testing"

	"github.com/tidwall/sjson"
)

var validKeys = []byte(`{"algorithms":["m.olm.v1.curve25519-aes-sha2","m.megolm.v1.aes-sha2"],"device_id":"JLAFKJWSCS","keys":{"curve25519:JLAFKJWSCS":"3C5BFWi2Y8MaVvjM8M22DBmh24PmgR0nPvJOIArzgyI","ed25519:JLAFKJWSCS":"lEuiRJBit0IG6nUf5pUzWTUEsRVVe/HJkoKuEww9ULI"},"signatures":{"@alice:example.com":{"ed25519:JLAFKJWSCS":"dSO80A01XiigH3uBiDVx/EjzaoycHcjq9lfQX0uWsqxl2giMIiSPR8a4d291W1ihKJL/a+myXS367WT6NAIcBA"}},"user_id":"@alice:example.com"}`)

// required fields are missing
var (
	missingAlgorithms, _ = sjson.DeleteBytes(validKeys, "algorithms")
	missingDeviceID, _   = sjson.DeleteBytes(missingAlgorithms, "device_id")
	missingKeys, _       = sjson.DeleteBytes(missingDeviceID, "keys")
	missingSignatures, _ = sjson.DeleteBytes(missingKeys, "signatures")
	missingEverything, _ = sjson.DeleteBytes(missingKeys, "user_id")
)

func Test_uploadKeysRequest_valid(t *testing.T) {
	type fields struct {
		DeviceKeys  json.RawMessage
		OneTimeKeys map[string]json.RawMessage
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "valid device keys",
			fields:  fields{DeviceKeys: validKeys},
			wantErr: false,
		},
		{
			name:   "no device keys specified",
			fields: fields{DeviceKeys: []byte("")},
		},
		{
			name:   "no device keys specified 2",
			fields: fields{DeviceKeys: nil},
		},
		{
			name:    "missing everything",
			fields:  fields{DeviceKeys: missingEverything},
			wantErr: true,
		},
		{
			name:    "missing signtures",
			fields:  fields{DeviceKeys: missingSignatures},
			wantErr: true,
		},
		{
			name:    "missing keys",
			fields:  fields{DeviceKeys: missingKeys},
			wantErr: true,
		},
		{
			name:    "missing device_id",
			fields:  fields{DeviceKeys: missingDeviceID},
			wantErr: true,
		},
		{
			name:    "missing algos",
			fields:  fields{DeviceKeys: missingAlgorithms},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := uploadKeysRequest{
				DeviceKeys:  tt.fields.DeviceKeys,
				OneTimeKeys: tt.fields.OneTimeKeys,
			}
			if err := k.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}