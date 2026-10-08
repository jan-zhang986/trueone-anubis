package model

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TC-BIT-01
// Contract: MySQL BIT(1) round-trips through the driver's []byte representation.
func TestBitBool_ValueEncoding(t *testing.T) {
	if got, err := BitBool(true).Value(); err != nil || !bytes.Equal(got.([]byte), []byte{1}) {
		t.Errorf("BitBool(true).Value() = %v, %v; want []byte{1}, nil", got, err)
	}
	if got, err := BitBool(false).Value(); err != nil || !bytes.Equal(got.([]byte), []byte{0}) {
		t.Errorf("BitBool(false).Value() = %v, %v; want []byte{0}, nil", got, err)
	}
}

// TC-BIT-02
// Contract: Scan must decode every representation MySQL/database-sql can hand
// back for BIT(1). The user.enable and deleted columns are BIT(1) NOT NULL, so a
// scan failure here breaks login outright.
func TestBitBool_ScanRepresentations(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want bool
	}{
		{"driver bytes true", []byte{1}, true},
		{"driver bytes false", []byte{0}, false},
		{"ascii one", []byte{'1'}, true},
		{"ascii zero", []byte{'0'}, false},
		{"empty byte slice", []byte{}, false},
		{"int64 one", int64(1), true},
		{"int64 zero", int64(0), false},
		{"bool true", true, true},
		{"bool false", false, false},
		{"NULL column", nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b BitBool
			if err := b.Scan(tc.in); err != nil {
				t.Fatalf("Scan(%#v) returned error: %v", tc.in, err)
			}
			if bool(b) != tc.want {
				t.Errorf("Scan(%#v) = %v, want %v", tc.in, bool(b), tc.want)
			}
		})
	}
}

// TC-BIT-03
// Contract: an unsupported source type must surface as an error rather than a
// silently wrong boolean, so a schema drift is loud instead of corrupting state.
func TestBitBool_ScanRejectsUnsupportedType(t *testing.T) {
	var b BitBool
	err := b.Scan("1")
	if err == nil {
		t.Fatalf("Scan(\"1\") returned nil error, want a type error")
	}
}

// TC-BIT-04
// Contract: JSON encoding emits a real boolean, not 0/1, because the frontend
// binds these fields to checkbox/switch components.
func TestBitBool_MarshalJSONIsBoolean(t *testing.T) {
	for _, tc := range []struct {
		in   BitBool
		want string
	}{
		{BitBool(true), "true"},
		{BitBool(false), "false"},
	} {
		got, err := json.Marshal(tc.in)
		if err != nil {
			t.Fatalf("Marshal(%v) error: %v", tc.in, err)
		}
		if string(got) != tc.want {
			t.Errorf("Marshal(%v) = %s, want %s", bool(tc.in), got, tc.want)
		}
	}
}

// TC-BIT-05
// Contract: inbound JSON may carry either a boolean or a 0/1 numeric legacy form.
func TestBitBool_UnmarshalJSONAcceptsBooleanAndNumeric(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`true`, true},
		{`false`, false},
		{`1`, true},
		{`0`, false},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			b := BitBool(false)
			if err := json.Unmarshal([]byte(tc.in), &b); err != nil {
				t.Fatalf("Unmarshal(%s) error: %v", tc.in, err)
			}
			if bool(b) != tc.want {
				t.Errorf("Unmarshal(%s) = %v, want %v", tc.in, bool(b), tc.want)
			}
		})
	}
}

// TC-BIT-06
// Contract risk: malformed input is accepted without error and without mutating
// the target. A caller that ignores the returned error cannot distinguish
// "false" from "unparseable", which masks bad upstream payloads.
func TestBitBool_UnmarshalJSONSilentlyIgnoresMalformedInput(t *testing.T) {
	b := BitBool(true)
	err := json.Unmarshal([]byte(`"not-a-bool"`), &b)

	if err != nil {
		t.Fatalf("Unmarshal(malformed) = %v, want nil (documents silent acceptance)", err)
	}
	if bool(b) != true {
		t.Errorf("value mutated to %v; want the original true retained", bool(b))
	}
}

// TC-BIT-07
// Contract: a full JSON round-trip is lossless, so a value read from the DB and
// echoed back to the client reproduces the same stored state.
func TestBitBool_JSONRoundTrip(t *testing.T) {
	for _, original := range []BitBool{BitBool(true), BitBool(false)} {
		encoded, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var decoded BitBool
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if decoded != original {
			t.Errorf("round trip changed %v into %v", bool(original), bool(decoded))
		}
	}
}
