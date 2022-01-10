package kdb

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// 開講時期
	_               = iota
	TermSpringACode // 春A: 1
	TermSpringBCode
	TermSpringCCode
	TermFallACode
	TermFallBCode
	TermFallCCode
	TermSummerVacationCode
	TermSpringVacationCode
	TermAllCode
	TermSpringCode
	TermFallCode
)

const (
	// 科目履修生申請可否
	// ×
	CreditedAuditorsCross = iota
	// △
	CreditedAuditorsTriangle
	// 空
	CreditedAuditorsEmpty
)

// 開講時期をパースする
func TermParser(termString string) []string {
	res := []string{}
	if termString == "" {
		return []string{}
	}
	var re *regexp.Regexp
	re = regexp.MustCompile(`(春A|春AA|春AA|春AB|春BA|春AC|春CA|春ABC)`)
	if re.MatchString(termString) {
		res = append(res, "春A")
	}
	re = regexp.MustCompile(`(春B|春BA|春AB|春BB|春BB|春BC|春CB|春ABC)`)
	if re.MatchString(termString) {
		res = append(res, "春B")
	}
	re = regexp.MustCompile(`(春C|春CA|春AC|春CB|春BC|春CC|春CC|春ABC)`)
	if re.MatchString(termString) {
		res = append(res, "春C")
	}
	re = regexp.MustCompile(`(秋A|秋AA|秋AA|秋AB|秋BA|秋AC|秋CA|秋ABC)`)
	if re.MatchString(termString) {
		res = append(res, "秋A")
	}
	re = regexp.MustCompile(`(秋B|秋BA|秋AB|秋BB|秋BB|秋BC|秋CB|秋ABC)`)
	if re.MatchString(termString) {
		res = append(res, "秋B")
	}
	re = regexp.MustCompile(`(秋C|秋CA|秋AC|秋CB|秋BC|秋CC|秋CC|秋ABC)`)
	if re.MatchString(termString) {
		res = append(res, "秋C")
	}
	re = regexp.MustCompile(`(夏季休業中)`)
	if re.MatchString(termString) {
		res = append(res, "夏季休業中")
	}
	re = regexp.MustCompile(`(春季休業中)`)
	if re.MatchString(termString) {
		res = append(res, "春季休業中")
	}
	re = regexp.MustCompile(`(通年)`)
	if re.MatchString(termString) {
		res = append(res, "通年")
	}
	re = regexp.MustCompile(`(春学期)`)
	if re.MatchString(termString) {
		res = append(res, "春学期")
	}
	re = regexp.MustCompile(`(秋学期)`)
	if re.MatchString(termString) {
		res = append(res, "秋学期")
	}
	return res
}

// 担当教員をパースする
func InstructorParser(instructors string) ([]string, error) {
	res := strings.Split(instructors, ",")
	return res, nil
}

// 科目等履修生をパースする
func CreditedAuditorsParser(CreditedAuditors string) (int, error) {
	if CreditedAuditors == "×" {
		return CreditedAuditorsCross, nil
	} else if CreditedAuditors == "△" {
		return CreditedAuditorsTriangle, nil
	} else if CreditedAuditors == "" {
		return CreditedAuditorsEmpty, nil
	} else {
		return -1, errors.New("invalid input:CreditedAuditors input")
	}
}

// KdB からエクスポートした CSV に含まれている更新日時カラムのものを time.Time に変換する
func DateParser(date string) (time.Time, error) {
	const layout = "2006-01-02 15:04:05"
	jst, _ := time.LoadLocation("Asia/Tokyo")
	t, err := time.ParseInLocation(layout, date, jst)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// 開講時期を数値に変換
// 別テーブルなどで管理するのが適切（？）
func TermStrToInt(term string) (int, error) {
	switch term {
	case "春A":
		return TermSpringACode, nil
	case "春B":
		return TermSpringBCode, nil
	case "春C":
		return TermSpringCCode, nil
	case "秋A":
		return TermFallACode, nil
	case "秋B":
		return TermFallBCode, nil
	case "秋C":
		return TermFallCCode, nil
	case "夏季休業中":
		return TermSummerVacationCode, nil
	case "春季休業中":
		return TermSpringVacationCode, nil
	case "通年":
		return TermAllCode, nil
	case "春学期":
		return TermSpringCode, nil
	case "秋学期":
		return TermFallCode, nil
	default:
		return -1, fmt.Errorf("invalid term string: %s", term)
	}
}

// 標準履修年次をパースする
// 中黒をハイフンとして解釈すると"?" or "int" or "int-int" になるので，"?", "int" はそのまま
// それ以外は間全てをいれる
func StandardRegistrationYearParser(yearString string) ([]string, error) {
	yearString = strings.Replace(yearString, " ", "", -1)

	// 部分文字列を正確に取り出すため
	yearRune := []rune(yearString)

	if len(yearRune) == 1 {
		return []string{yearString}, nil
	}

	minYear, err := strconv.Atoi(string(yearRune[0]))
	if err != nil {
		return []string{}, err
	}

	maxYear, err := strconv.Atoi(string(yearRune[2]))
	if err != nil {
		return []string{}, err
	}

	year := []string{}
	for i := minYear; i <= maxYear; i++ {
		year = append(year, strconv.Itoa(i))
	}

	return year, nil
}

// 時間割をパースする
func PeriodParser(periodString string) ([]string, error) {
	period := []string{}
	periodString = strings.Replace(periodString, " ", "", -1)
	periodString = strings.Replace(periodString, "　", "", -1)
	periodString = strings.Replace(periodString, "ー", "-", -1)
	periodString = strings.Replace(periodString, "・", "", -1)
	periodString = strings.Replace(periodString, ",", "", -1)
	periodString = strings.Replace(periodString, "集中", "集0", -1)
	periodString = strings.Replace(periodString, "応談", "応0", -1)
	periodString = strings.Replace(periodString, "随時", "随0", -1)

	for i := 1; i <= 8; i++ {
		listPeriod := strconv.Itoa(i)
		for j := i + 1; j <= 8; j++ {
			listPeriod = listPeriod + strconv.Itoa(j)
			spanPeriod := strconv.Itoa(i) + "-" + strconv.Itoa(j)
			periodString = strings.Replace(periodString, spanPeriod, listPeriod, -1)
		}
	}

	for i := 0; i <= 8; i++ {
		for _, dayOfWeek := range []string{"月", "火", "水", "木", "金", "土", "日", "応", "随", "集"} {
			beforeStr1 := strconv.Itoa(i) + dayOfWeek
			beforeStr2 := dayOfWeek + strconv.Itoa(i)
			afterStr1 := strconv.Itoa(i) + "," + dayOfWeek
			afterStr2 := dayOfWeek + ":" + strconv.Itoa(i)
			periodString = strings.Replace(periodString, beforeStr1, afterStr1, -1)
			periodString = strings.Replace(periodString, beforeStr2, afterStr2, -1)
		}
	}
	if len(periodString) == 0 {
		return period, nil
	}
	strList := strings.Split(periodString, ",")

	for _, str := range strList {
		strList2 := strings.Split(str, ":")
		if len(strList2) != 2 {
			fmt.Println("-" + periodString + "-")
			return nil, errors.New("unexpected period input : " + str)
		} else {
			dayOfWeek := strList2[0]
			timeTimetable := strList2[1]
			for i := 0; i < len([]rune(dayOfWeek)); i++ {
				for j := 0; j < len([]rune(timeTimetable)); j++ {
					inputStr := string([]rune(dayOfWeek)[i]) + string([]rune(timeTimetable)[j])
					inputStr = strings.Replace(inputStr, "集0", "集", -1)
					inputStr = strings.Replace(inputStr, "集", "集中", -1)
					inputStr = strings.Replace(inputStr, "随0", "随", -1)
					inputStr = strings.Replace(inputStr, "随", "随時", -1)
					inputStr = strings.Replace(inputStr, "応0", "応", -1)
					inputStr = strings.Replace(inputStr, "応", "応談", -1)
					period = append(period, inputStr)
				}
			}
		}
	}

	return period, nil
}
