package util

import (
	"testing"
)

func TestSession(t *testing.T) {
	type args struct {
		tp   int32
		from int64
		to   int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "single chat",
			args: args{
				tp:   0,
				from: 1,
				to:   2,
			},
			want: "000|0000000000000000000100000000000000000002",
		},
		{
			name: "group chat",
			args: args{
				tp:   1,
				from: 1,
				to:   2,
			},
			want: "001|0000000000000000000200000000000000000000",
		},
		{
			name: "broadcast",
			args: args{
				tp:   2,
				from: 1,
				to:   2,
			},
			want: "002|0000000000000000000000000000000000000000",
		},
		{
			name: "channel",
			args: args{
				tp:   3,
				from: 1,
				to:   2,
			},
			want: "003|0000000000000000000100000000000000000002",
		},
		{
			name: "invalid type -1",
			args: args{
				tp:   -1,
				from: 1,
				to:   2,
			},
			want: "",
		},
		{
			name: "invalid type 256",
			args: args{
				tp:   256,
				from: 1,
				to:   2,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Session(tt.args.tp, tt.args.from, tt.args.to); got != tt.want {
				t.Errorf("Session() = %v, want %v", got, tt.want)
			}
		})
	}
}
