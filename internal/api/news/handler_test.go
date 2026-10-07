package news

import "testing"

func TestParsePublishedValue(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    bool
		wantErr bool
	}{
		{name: "boolean true", value: true, want: true},
		{name: "boolean false", value: false, want: false},
		{name: "published label", value: "published", want: true},
		{name: "visible label", value: "visible", want: true},
		{name: "hidden label", value: "hidden", want: false},
		{name: "Indonesian hidden label", value: "Sembunyikan", want: false},
		{name: "unknown status", value: "pending", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePublishedValue(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("parsePublishedValue() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Errorf("parsePublishedValue() = %v, want %v", got, test.want)
			}
		})
	}
}
