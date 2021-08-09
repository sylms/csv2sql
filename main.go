package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"bytes"
	"github.com/gocarina/gocsv"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	migrate "github.com/rubenv/sql-migrate"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var (
	db         *sqlx.DB
	migrations = &migrate.FileMigrationSource{
		Dir: "migrations",
	}
	now time.Time
)

const (
	// 科目履修生申請可否
	// ×
	creditedAuditorsCross = iota
	// △
	creditedAuditorsTriangle
	// 空
	creditedAuditorsEmpty
)

const (
	// 開講時期
	_               = iota
	termSpringACode // 春A: 1
	termSpringBCode
	termSpringCCode
	termFallACode
	termFallBCode
	termFallCCode
	termSummerVacationCode
	termSpringVacationCode
	termAllCode
	termSpringCode
	termFallCode
)

const (
	envSylmsPostgresDBKey       = "SYLMS_POSTGRES_DB"
	envSylmsPostgresUserKey     = "SYLMS_POSTGRES_USER"
	envSylmsPostgresPasswordKey = "SYLMS_POSTGRES_PASSWORD"
	envSylmsPostgresHostKey     = "SYLMS_POSTGRES_HOST"
	envSylmsPostgresPortKey     = "SYLMS_POSTGRES_PORT"
	envSylmsCsvYear             = "SYLMS_CSV_YEAR"
)

func main() {
	var err error

	envKeys := []string{envSylmsPostgresDBKey, envSylmsPostgresUserKey, envSylmsPostgresPasswordKey, envSylmsPostgresHostKey, envSylmsPostgresPortKey, envSylmsCsvYear}
	for _, key := range envKeys {
		val, ok := os.LookupEnv(key)
		if !ok || val == "" {
			log.Fatalf("%s is not set or empty\n", key)
		}
	}

	now = getDateTimeNow()

	postgresDb := os.Getenv(envSylmsPostgresDBKey)
	postgresUser := os.Getenv(envSylmsPostgresUserKey)
	postgresPassword := os.Getenv(envSylmsPostgresPasswordKey)
	postgresHost := os.Getenv(envSylmsPostgresHostKey)
	postgresPort := os.Getenv(envSylmsPostgresPortKey)
	db, err = sqlx.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", postgresHost, postgresPort, postgresUser, postgresPassword, postgresDb))
	if err != nil {
		log.Fatalf("%+v", err)
	}

	err = execMigrate()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	kdbCSV, err := readFromCSV()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(kdbCSV)
	csvStr := buf.String()

	// double quotation escape and separate comma enable
	// ダブルクォーテーションをエスケープする
	// ダブルクォーテーションはエスケープされていないのでまず全て2倍にする
	// その後に区切り文字のカンマまわりのダブルクォーテーションが2重になっていることを解消する
	// また，各行の最初と最後のダブルクォーテーションが2重になっていることも解消する
	escapedDoubleQuotationStr := strings.Replace(csvStr, `"`, `""`, -1)
	unEscapedDCAroundCommaStr := strings.Replace(escapedDoubleQuotationStr, `","`, `,`, -1)

	// 各行の最後と次行の最初のダブルクォーテーションが2重になっているため解消する
	// また，行の最初と最後に空白文字が入っている場合があるため，それへの対策を講じている
	re0 := regexp.MustCompile("\"\\s*\r\n\\s*\"")
	unEscapedDCAroundNLStr := re0.ReplaceAllString(unEscapedDCAroundCommaStr, "\r\n")

	// ファイルの先頭のダブルクォーテーションが2重になっているため解消する
	// 行の最初と最後に空白文字が入っている場合があるため，それへの対策を講じている
	re1 := regexp.MustCompile("^\\s*\"\"")
	unEscapedDCBeginOfLineStr := re1.ReplaceAllString(unEscapedDCAroundNLStr, `"`)

	// ファイルの末尾のダブルクォーテーションが2重になっているため解消する
	// 行の最初と最後に空白文字が入っている場合があるため，それへの対策を講じている
	re2 := regexp.MustCompile("\"\"\\s*$")
	unEscapedDCEndOfLineStr := re2.ReplaceAllString(unEscapedDCBeginOfLineStr, `"`)

	replacedCSVStr := unEscapedDCEndOfLineStr

	// string to io.Reader
	readerReplacedCSV := strings.NewReader(replacedCSVStr)
	readerReplacedCSVCloser := io.NopCloser(readerReplacedCSV)

	courses, err := csvToCoursesStruct(readerReplacedCSVCloser)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	for _, c := range courses {
		err = c.insert(tx)
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Fatalf("rollback error: %+v", err)
			}
			log.Fatalf("%+v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	log.Println("done")
}

// 実行ファイルのカレントからみて ${csvDirName}/${csvFilename} の CSV ファイルを読み込む
func readFromCSV() (io.ReadCloser, error) {
	const (
		csvDirName  = "csv"
		csvFilename = "kdb.csv"
	)

	exePath, err := os.Executable()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	exeCurrentDirPath := filepath.Dir(exePath)
	csvFilePath := filepath.Join(exeCurrentDirPath, csvDirName, csvFilename)

	f, err := os.Open(csvFilePath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return f, nil
}

func csvToCoursesStruct(reader io.ReadCloser) ([]Courses, error) {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		// KdB からダウンロードした CSV は ShiftJIS なため
		r := csv.NewReader(transform.NewReader(in, japanese.ShiftJIS.NewDecoder()))
		// KdB からダウンロードした CSV のダブルクオーテーションはエスケープがされていないため
		r.LazyQuotes = true
		return r
	})

	kdbCsvRows := []*KdbExportCSV{}
	// Unmarchal は CSV の生から定義した構造体に落とし込んでくれている．
	err := gocsv.Unmarshal(reader, &kdbCsvRows)
	if err != nil {
		return []Courses{}, errors.WithStack(err)
	}
	reader.Close()

	yearStr := os.Getenv(envSylmsCsvYear)
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// CSV のもの（KdbExportCSV）から DB 向け（Courses）に構造体を組みなおす
	courses := []Courses{}
	for _, row := range kdbCsvRows {
		// 科目番号がないものは、それは科目ではないとみなしデータベースに投入しないようにする
		if row.CourseNumber == "" {
			continue
		}

		term := termParser(row.Term)

		termInt, err := termStrToInt(term)
		if err != nil {
			return nil, err
		}

		creditedAuditors, err := creditedAuditorsParser(row.CreditedAuditors)
		if err != nil {
			return nil, err
		}

		csvUpdatedAt, err := csvStringDateParser(row.UpdatedAt)
		if err != nil {
			return nil, err
		}

		standardRegistrationYearParser, err := standardRegistrationYearParser(row.StandardRegistrationYear)
		if err != nil {
			return nil, err
		}

		period, err := periodParser(row.Period)
		if err != nil {
			return nil, err
		}

		instructor, err := instructorParser(row.Instructor)
		if err != nil {
			return nil, err
		}

		s := Courses{
			CourseNumber:             row.CourseNumber,
			CourseName:               row.CourseName,
			InstructionalType:        row.InstructionalType,
			Credits:                  strings.TrimSpace(row.Credits),
			StandardRegistrationYear: standardRegistrationYearParser,
			Term:                     termInt,
			Period:                   period,
			Classroom:                newSQLNullString(row.Classroom),
			Instructor:               instructor,
			CourseOverview:           newSQLNullString(row.CourseOverview),
			Remarks:                  newSQLNullString(row.Remarks),
			CreditedAuditors:         creditedAuditors,
			ApplicationConditions:    newSQLNullString(row.ApplicationConditions),
			AltCourseName:            newSQLNullString(row.AltCourseName),
			CourseCode:               newSQLNullString(row.CourseCode),
			CourseCodeName:           newSQLNullString(row.CourseCodeName),
			CSVUpdatedAt:             csvUpdatedAt,
			Year:                     year,
			CreatedAt:                now,
			UpdatedAt:                now,
		}
		courses = append(courses, s)
	}
	return courses, nil
}

func newSQLNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func execMigrate() error {
	appliedCount, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Printf("Applied %v migrations", appliedCount)
	return nil
}

func termParser(termString string) []string {
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

// TODO: やる
func instructorParser(instructors string) ([]string, error) {
	res := []string{}
	// 一時的に配列の要素1つにそのままデータをいれるようにする（分割しない）
	if utf8.RuneCountInString(instructors) > 10 {
		res = append(res, string([]rune(instructors[:10])))
	} else {
		res = append(res, instructors)
	}
	return res, nil
}

func creditedAuditorsParser(CreditedAuditors string) (int, error) {
	if CreditedAuditors == "×" {
		return creditedAuditorsCross, nil
	} else if CreditedAuditors == "△" {
		return creditedAuditorsTriangle, nil
	} else if CreditedAuditors == "" {
		return creditedAuditorsEmpty, nil
	} else {
		return -1, errors.New("invalid input:CreditedAuditors input")
	}
}

// KdB からエクスポートした CSV に含まれている更新日時カラムのものを time.Time に変換する
func csvStringDateParser(date string) (time.Time, error) {
	const layout = "2006-01-02 15:04:05"
	jst, _ := time.LoadLocation("Asia/Tokyo")
	t, err := time.ParseInLocation(layout, date, jst)
	if err != nil {
		return time.Time{}, errors.WithStack(err)
	}
	return t, nil
}

func getDateTimeNow() time.Time {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	return now
}

// 開講時期を数値に変換
// 別テーブルなどで管理するのが適切（？）
func termStrToInt(term []string) ([]int, error) {
	res := []int{}
	for _, t := range term {
		switch t {
		case "春A":
			res = append(res, termSpringACode)
		case "春B":
			res = append(res, termSpringBCode)
		case "春C":
			res = append(res, termSpringCCode)
		case "秋A":
			res = append(res, termFallACode)
		case "秋B":
			res = append(res, termFallBCode)
		case "秋C":
			res = append(res, termFallCCode)
		case "夏季休業中":
			res = append(res, termSummerVacationCode)
		case "春季休業中":
			res = append(res, termSpringVacationCode)
		case "通年":
			res = append(res, termAllCode)
		case "春学期":
			res = append(res, termSpringCode)
		case "秋学期":
			res = append(res, termFallCode)
		default:
			return nil, fmt.Errorf("invalid term string: %s", t)
		}
	}
	return res, nil
}

func (c *Courses) insert(tx *sql.Tx) error {
	// TODO: レコードが重複して存在することが可能であるので、それを防ぐ
	// insert しようとしているレコードが既にテーブルに存在しているかは確認する必要があるかもしれない
	// UNIQUE 指定すれば、確認しなくてもよいらしい（？）

	// NamedExec では Array の扱いがうまくいかなかったのでとりあえず Exec でやる
	_, err := tx.Exec(`insert into courses (
		course_number, course_name, instructional_type, credits, standard_registration_year, term, period_, classroom, instructor, course_overview, remarks, credited_auditors, application_conditions, alt_course_name, course_code, course_code_name, csv_updated_at, year, created_at, updated_at
		) values (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)`,
		c.CourseNumber,
		c.CourseName,
		c.InstructionalType,
		c.Credits,
		pq.Array(c.StandardRegistrationYear),
		pq.Array(c.Term),
		pq.Array(c.Period),
		c.Classroom,
		pq.Array(c.Instructor),
		c.CourseOverview,
		c.Remarks,
		c.CreditedAuditors,
		c.ApplicationConditions,
		c.AltCourseName,
		c.CourseCode,
		c.CourseCodeName,
		c.CSVUpdatedAt,
		c.Year,
		c.CreatedAt,
		c.UpdatedAt,
	)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// TODO: やる
func standardRegistrationYearParser(yearString string) ([]string, error) {
	year := []string{}
	/*
		yearString = strings.Replace(yearString, "ー", "-", -1)
		yearString = strings.Replace(yearString, "・", "-", -1)
		yearString = strings.Replace(yearString, "～", "-", -1)
		yearString = strings.Replace(yearString, "~", "-", -1)
		yearString = strings.Replace(yearString, "、", ",", -1)
		yearString = strings.Replace(yearString, " ", "", -1)
		moji.Convert(yearString, moji.ZE, moji.HE)
		moji.Convert(yearString, moji.ZS, moji.HS)
		slice := strings.Split(yearString, ",")
		for _, str := range slice {
			if isInt
		}
	*/

	// とりあえずで、"1" を投入する
	year = append(year, "1")
	return year, nil
}

// TODO: やる
func periodParser(periodString string) ([]string, error) {
	// 一時的に配列の要素1つにそのままデータをいれるようにする（分割しない）
	period := []string{}
	if utf8.RuneCountInString(periodString) > 10 {
		period = append(period, string([]rune(periodString[:10])))
	} else {
		period = append(period, periodString)
	}
	return period, nil
}
