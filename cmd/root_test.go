package cmd

import (
	"reflect"
	"testing"
)

func TestNormalizeBareDBFlag(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "bare --db as the last argument becomes --db=",
			args: []string{"serve", "--headless", "--db"},
			want: []string{"serve", "--headless", "--db="},
		},
		{
			name: "bare -d as the last argument becomes --db=",
			args: []string{"serve", "--headless", "-d"},
			want: []string{"serve", "--headless", "--db="},
		},
		{
			name: "--db immediately followed by another flag becomes --db=",
			args: []string{"serve", "--db", "--headless"},
			want: []string{"serve", "--db=", "--headless"},
		},
		{
			name: "--db followed by a path is left untouched",
			args: []string{"serve", "--db", "custom.db", "--headless"},
			want: []string{"serve", "--db", "custom.db", "--headless"},
		},
		{
			name: "-d followed by a path is left untouched",
			args: []string{"serve", "-d", "custom.db"},
			want: []string{"serve", "-d", "custom.db"},
		},
		{
			name: "--db=value form is left untouched",
			args: []string{"serve", "--db=custom.db"},
			want: []string{"serve", "--db=custom.db"},
		},
		{
			name: "--db=  (already-empty equals form) is left untouched",
			args: []string{"serve", "--db="},
			want: []string{"serve", "--db="},
		},
		{
			name: "no --db/-d at all is left untouched",
			args: []string{"serve", "--headless"},
			want: []string{"serve", "--headless"},
		},
		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeBareDBFlag(tc.args)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("normalizeBareDBFlag(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
