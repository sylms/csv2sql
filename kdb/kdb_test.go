package kdb

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func Test_TermParser(t *testing.T) {
	type args struct {
		term_string string
	}
	tests := []struct {
		name string
		args args
		want []string
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
		},
		{
			name: "empty string -> empty []string",
			args: args{
				term_string: "",
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TermParser(tt.args.term_string)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TermParser() = %v, want %v", got, tt.want)
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
			got, err := CreditedAuditorsParser(tt.args.CreditedAuditors)
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

func Test_standardRegistrationYearParser(t *testing.T) {
	type args struct {
		yearString string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "中黒を認識できる",
			args: args{
				yearString: "1・3",
			},
			want:    []string{"1", "2", "3"},
			wantErr: false,
		},
		{
			name: "半角ハイフンを認識できる",
			args: args{
				yearString: "1-3",
			},
			want:    []string{"1", "2", "3"},
			wantErr: false,
		},
		{
			name: "～を認識できる",
			args: args{
				yearString: "1～3",
			},
			want:    []string{"1", "2", "3"},
			wantErr: false,
		},
		{
			name: "年次の数字と数字間を結ぶ記号にスペースがある",
			args: args{
				yearString: "1 - 3",
			},
			want:    []string{"1", "2", "3"},
			wantErr: false,
		},
		{
			name: "年次が一つの場合が認識できる",
			args: args{
				yearString: "1",
			},
			want:    []string{"1"},
			wantErr: false,
		},
		{
			name: "'?'をそのまま保持する",
			args: args{
				yearString: "?",
			},
			// ? はそのまま保持する方針
			want:    []string{"?"},
			wantErr: false,
		},
		{
			name: "年次（数字）ではないものはエラーになる",
			args: args{
				yearString: "invalid",
			},
			want:    []string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StandardRegistrationYearParser(tt.args.yearString)
			if (err != nil) != tt.wantErr {
				t.Errorf("standardRegistrationYearParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("standardRegistrationYearParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_PeriodParser(t *testing.T) {
	type args struct {
		periodString string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		// とてもシンプル
		{
			name: "1 コマ",
			args: args{
				periodString: "月1",
			},
			want:    []string{"月1"},
			wantErr: false,
		},

		// 同じ曜日
		{
			name: "同じ曜日で n コマ連続",
			args: args{
				periodString: "月12",
			},
			want:    []string{"月1", "月2"},
			wantErr: false,
		},
		{
			name: "同じ曜日に x コマ飛ばして y コマ存在",
			args: args{
				periodString: "月1-3,5",
			},
			want:    []string{"月1", "月2", "月3", "月5"},
			wantErr: false,
		},

		// 曜日が異なる
		{
			name: "曜日が異なり n コマ",
			args: args{
				periodString: "月1火1",
			},
			want:    []string{"月1", "火1"},
			wantErr: false,
		},
		{
			name: "曜日が異なり同じ時間帯に 1 コマ",
			args: args{
				periodString: "月・火1",
			},
			want:    []string{"月1", "火1"},
			wantErr: false,
		},
		{
			name: "曜日が異なり同じ時間帯 n コマ",
			args: args{
				periodString: "月・木1-3",
			},
			want:    []string{"月1", "木1", "月2", "木2", "月3", "木3"},
			wantErr: false,
		},
		{
			name: "曜日が異なり同じ時間帯 n コマ（1 日の中で時間が飛ぶ）",
			args: args{
				periodString: "月,火3,5-7",
			},
			want:    []string{"月3", "月5", "月6", "月7", "火3", "火5", "火6", "火7"},
			wantErr: false,
		},

		// 応談
		// 応談と随時、集中はすべて同じ処理をしているので↓が成功すればよい
		{
			name: "完全に応談のみ",
			args: args{
				periodString: "応談",
			},
			want:    []string{"応談"},
			wantErr: false,
		},
		{
			name: "完全に随時のみ",
			args: args{
				periodString: "随時",
			},
			want:    []string{"随時"},
			wantErr: false,
		},
		{
			name: "完全に集中のみ",
			args: args{
				periodString: "集中",
			},
			want:    []string{"集中"},
			wantErr: false,
		},
		{
			name: "応談と集中",
			args: args{
				periodString: "応談 集中",
			},
			want:    []string{"応談", "集中"},
			wantErr: false,
		},
		{
			name: "応談に時限がある（？）",
			args: args{
				periodString: "応談78",
			},
			want:    []string{"応談", "応談7", "応談8"},
			wantErr: false,
		},
		{
			name: "応談と通常コマ",
			args: args{
				periodString: "応談 木1",
			},
			want:    []string{"応談", "木1"},
			wantErr: false,
		},
		{
			name: "応談に時限があるものと通常コマ",
			args: args{
				periodString: "応談78 木1",
			},
			want:    []string{"応談", "応談7", "応談8", "木1"},
			wantErr: false,
		},

		// 区切り文字
		{
			name: "曜日またぎの区切り文字がない",
			args: args{
				periodString: "月1火1",
			},
			want:    []string{"月1", "火1"},
			wantErr: false,
		},
		{
			name: "曜日またぎの区切り文字がスペース",
			args: args{
				periodString: "月1 火1",
			},
			want:    []string{"月1", "火1"},
			wantErr: false,
		},
		{
			name: "曜日またぎの区切り文字が中黒",
			args: args{
				periodString: "月1・火1",
			},
			want:    []string{"月1", "火1"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PeriodParser(tt.args.periodString)

			// slice の順序は重視していないためソートして両方あわせる
			sort.Strings(got)
			sort.Strings(tt.want)

			if (err != nil) != tt.wantErr {
				t.Errorf("PeriodParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PeriodParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_DateParser(t *testing.T) {
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
			got, err := DateParser(tt.args.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("DateParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DateParser() = %v, want %v", got, tt.want)
			}
		})
	}
}
