package main

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
)

func Test_termParser(t *testing.T) {
	type args struct {
		term_string string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "spring_A_B",
			args: args{
				term_string: "春AB",
			},
			want: []string{
				"春A",
				"春B",
			},
			wantErr: false,
		},
		{
			name: "invalid_input",
			args: args{
				term_string: "",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := termParser(tt.args.term_string)
			if (err != nil) != tt.wantErr {
				t.Errorf("termParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("termParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_creditedAuditorsParser(t *testing.T) {
	type args struct {
		CreditedAuditors string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "true",
			args: args{
				CreditedAuditors: "",
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "sankaku",
			args: args{
				CreditedAuditors: "△",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "batsu",
			args: args{
				CreditedAuditors: "×",
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				CreditedAuditors: "fdasfd",
			},
			want:    -1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := creditedAuditorsParser(tt.args.CreditedAuditors)
			if (err != nil) != tt.wantErr {
				t.Errorf("creditedAuditorsParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("creditedAuditorsParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getCredits(t *testing.T) {
	type args struct {
		credits string
	}
	tests := []struct {
		name string
		args args
		want sql.NullFloat64
	}{
		{
			name: "normal int",
			args: args{
				credits: "3.0",
			},
			want: sql.NullFloat64{Float64: 3.0, Valid: true},
		},
		{
			name: "normal float",
			args: args{
				credits: "3.5",
			},
			want: sql.NullFloat64{Float64: 3.5, Valid: true},
		},
		{
			name: "invalid input: '-'",
			args: args{
				credits: "-",
			},
			want: sql.NullFloat64{Valid: false},
		},
		{
			name: "invalid input: empty",
			args: args{
				credits: "",
			},
			want: sql.NullFloat64{Valid: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCredits(tt.args.credits); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCredits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_csvStringDateParser(t *testing.T) {
	type args struct {
		date string
	}
	jst, _ := time.LoadLocation("Asia/Tokyo")
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				date: "2021-03-01 14:27:49",
			},
			want: time.Date(2021, 03, 01, 14, 27, 49, 00, jst),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := csvStringDateParser(tt.args.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("csvStringDateParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("csvStringDateParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_termStrToInt(t *testing.T) {
	type args struct {
		term []string
	}
	tests := []struct {
		name    string
		args    args
		want    []int
		wantErr bool
	}{
		{
			name: "normal single",
			args: args{
				term: []string{
					"春A",
				},
			},
			want: []int{
				1,
			},
			wantErr: false,
		},
		{
			name: "normal multiple",
			args: args{
				term: []string{
					"春A",
					"春B",
				},
			},
			want: []int{
				1, 2,
			},
			wantErr: false,
		},
		{
			name: "invalid input",
			args: args{
				term: []string{
					"謎の期間",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := termStrToInt(tt.args.term)
			if (err != nil) != tt.wantErr {
				t.Errorf("termStrToInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("termStrToInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
