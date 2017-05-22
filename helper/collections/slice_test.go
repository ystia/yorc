package collections

import "testing"

func TestContainsString(t *testing.T) {
	type args struct {
		s []string
		e string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"EmptySlice", args{[]string{}, "Test"}, false},
		{"NilSlice", args{nil, "Test"}, false},
		{"NilSliceEmptyString", args{nil, ""}, false},
		{"EmptySliceEmptyString", args{[]string{}, ""}, false},
		{"EmptyString", args{[]string{""}, ""}, true},
		{"EmptyStringAtFirst", args{[]string{"", "Test"}, ""}, true},
		{"EmptyStringAtMiddle", args{[]string{"A", "", "Test"}, ""}, true},
		{"EmptyStringAtLast", args{[]string{"A", "Test", ""}, ""}, true},
		{"SimpleString", args{[]string{"Test"}, "Test"}, true},
		{"SimpleStringAtFirst", args{[]string{"Test", "A"}, "Test"}, true},
		{"SimpleStringAtMiddle", args{[]string{"A", "Test", "B"}, "Test"}, true},
		{"SimpleStringAtLast", args{[]string{"A", "B", "Test"}, "Test"}, true},
		{"TestCase", args{[]string{"A", "B", "Test"}, "b"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsString(tt.args.s, tt.args.e); got != tt.want {
				t.Errorf("ContainsString() = %v, want %v", got, tt.want)
			}
		})
	}
}
