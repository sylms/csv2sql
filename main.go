package main

import (
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

	"bytes"

	"github.com/gobuffalo/packr/v2"
	"github.com/gocarina/gocsv"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sylms/csv2sql/kdb"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var (
	db         *sqlx.DB
	migrations = &migrate.PackrMigrationSource{
		Box: packr.New("migrations", "./migrations"),
	}
	now time.Time
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

	tx, err := db.Beginx()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	err = insert(tx, courses)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			log.Fatalf("rollback error: %+v", err)
		}
		log.Fatalf("%+v", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	log.Println("done")
	os.Exit(0)
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

		term := kdb.TermParser(row.Term)

		termInt, err := kdb.TermStrToInt(term)
		if err != nil {
			return nil, err
		}

		creditedAuditors, err := kdb.CreditedAuditorsParser(row.CreditedAuditors)
		if err != nil {
			return nil, err
		}

		csvUpdatedAt, err := kdb.DateParser(row.UpdatedAt)
		if err != nil {
			return nil, err
		}

		standardRegistrationYearParser, err := kdb.StandardRegistrationYearParser(row.StandardRegistrationYear)
		if err != nil {
			return nil, err
		}

		period, err := kdb.PeriodParser(row.Period)
		if err != nil {
			return nil, err
		}

		instructor, err := kdb.InstructorParser(row.Instructor)
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
			Classroom:                row.Classroom,
			Instructor:               instructor,
			CourseOverview:           row.CourseOverview,
			Remarks:                  row.Remarks,
			CreditedAuditors:         creditedAuditors,
			ApplicationConditions:    row.ApplicationConditions,
			AltCourseName:            row.AltCourseName,
			CourseCode:               row.CourseCode,
			CourseCodeName:           row.CourseCodeName,
			CSVUpdatedAt:             csvUpdatedAt,
			Year:                     year,
			CreatedAt:                now,
			UpdatedAt:                now,
		}
		courses = append(courses, s)
	}
	return courses, nil
}

func execMigrate() error {
	appliedCount, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		return errors.WithStack(err)
	}
	log.Printf("Applied %v migrations", appliedCount)
	return nil
}

func getDateTimeNow() time.Time {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(jst)
	return now
}

func insert(tx *sqlx.Tx, courses []Courses) error {
	// TODO: レコードが重複して存在することが可能であるので、それを防ぐ
	// insert しようとしているレコードが既にテーブルに存在しているかは確認する必要があるかもしれない
	// UNIQUE 指定すれば、確認しなくてもよいらしい（？）

	type insertPrepare struct {
		CourseNumber             string      `db:"course_number"`
		CourseName               string      `db:"course_name"`
		InstructionalType        int         `db:"instructional_type"`
		Credits                  string      `db:"credits"`
		StandardRegistrationYear interface{} `db:"standard_registration_year"`
		Term                     interface{} `db:"term"`
		Period                   interface{} `db:"period_"`
		Classroom                string      `db:"classroom"`
		Instructor               interface{} `db:"instructor"`
		CourseOverview           string      `db:"course_overview"`
		Remarks                  string      `db:"remarks"`
		CreditedAuditors         int         `db:"credited_auditors"`
		ApplicationConditions    string      `db:"application_conditions"`
		AltCourseName            string      `db:"alt_course_name"`
		CourseCode               string      `db:"course_code"`
		CourseCodeName           string      `db:"course_code_name"`
		CSVUpdatedAt             time.Time   `db:"csv_updated_at"`
		Year                     int         `db:"year"`
		CreatedAt                time.Time   `db:"created_at"`
		UpdatedAt                time.Time   `db:"updated_at"`
	}

	// 全て（約 19,000 件）を一気に insert しようとしたら制限に引っかかった
	// pq: got 395920 parameters but PostgreSQL only supports 65535 parameters
	// およそ 20 カラムあるので、20 * 3000 = 60000 より 3000 レコード区切りで insert していく
	const bulkInsertLimit = 3000
	// 3000 レコードごとに分割したときの個数（make で確保するときのために +1）
	bulkInsertCount := (len(courses) / bulkInsertLimit) + 1
	pre := make([][]insertPrepare, bulkInsertCount)

	bulkInsertCountNow := -1

	for count, c := range courses {
		if count%bulkInsertLimit == 0 {
			bulkInsertCountNow++
		}
		temp := insertPrepare{
			CourseNumber:             c.CourseNumber,
			CourseName:               c.CourseName,
			InstructionalType:        c.InstructionalType,
			Credits:                  c.Credits,
			StandardRegistrationYear: pq.Array(c.StandardRegistrationYear),
			Term:                     pq.Array(c.Term),
			Period:                   pq.Array(c.Period),
			Classroom:                c.Classroom,
			Instructor:               pq.Array(c.Instructor),
			CourseOverview:           c.CourseOverview,
			Remarks:                  c.Remarks,
			CreditedAuditors:         c.CreditedAuditors,
			ApplicationConditions:    c.ApplicationConditions,
			AltCourseName:            c.AltCourseName,
			CourseCode:               c.CourseCode,
			CourseCodeName:           c.CourseCodeName,
			CSVUpdatedAt:             c.CSVUpdatedAt,
			Year:                     c.Year,
			CreatedAt:                c.CreatedAt,
			UpdatedAt:                c.UpdatedAt,
		}
		pre[bulkInsertCountNow] = append(pre[bulkInsertCountNow], temp)
	}

	for _, p := range pre {
		_, err := tx.NamedExec(`insert into courses (
			course_number, course_name, instructional_type, credits, standard_registration_year, term, period_, classroom, instructor, course_overview, remarks, credited_auditors, application_conditions, alt_course_name, course_code, course_code_name, csv_updated_at, year, created_at, updated_at
		) values (
			:course_number, :course_name, :instructional_type, :credits, :standard_registration_year, :term, :period_, :classroom, :instructor, :course_overview, :remarks, :credited_auditors, :application_conditions, :alt_course_name, :course_code, :course_code_name, :csv_updated_at, :year, :created_at, :updated_at
		)`, p)

		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
