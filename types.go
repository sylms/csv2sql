package main

import (
	"time"
)

// KdB から csv でエクスポートしたもの
type KdbExportCSV struct {
	CourseNumber      string `csv:"科目番号"`
	CourseName        string `csv:"科目名"`
	InstructionalType int    `csv:"授業方法"`
	// '?' があるため
	Credits                  string `csv:"単位数"`
	StandardRegistrationYear string `csv:"標準履修年次"`
	Term                     string `csv:"実施学期"`
	// Meeting Days,Period etc.
	Period                string `csv:"曜時限"`
	Classroom             string `csv:"教室"`
	Instructor            string `csv:"担当教員"`
	CourseOverview        string `csv:"授業概要"`
	Remarks               string `csv:"備考"`
	CreditedAuditors      string `csv:"科目等履修生申請可否"`
	ApplicationConditions string `csv:"申請条件"`
	// Japanese (English) Course Name
	AltCourseName  string `csv:"英語(日本語)科目名"`
	CourseCode     string `csv:"科目コード"`
	CourseCodeName string `csv:"要件科目名"`
	// Data update date
	UpdatedAt string `csv:"データ更新日"`
}

type Courses struct {
	ID           int    `db:"id"`
	CourseNumber string `db:"course_number"`
	CourseName   string `db:"course_name"`
	// 対応付けを別に持つ
	InstructionalType        int      `db:"instructional_type"`
	Credits                  string   `db:"credits"`
	StandardRegistrationYear []string `db:"standard_registration_year"`
	// 対応付けを別に持つ
	Term []int `db:"term"`
	// 例：月1, 月2
	Period         []string `db:"period_"`
	Classroom      string   `db:"classroom"`
	Instructor     []string `db:"instructor"`
	CourseOverview string   `db:"course_overview"`
	Remarks        string   `db:"remarks"`
	// 0 = 'x', 1 = 三角, 2 = ''
	CreditedAuditors      int    `db:"credited_auditors"`
	ApplicationConditions string `db:"application_conditions"`
	AltCourseName         string `db:"alt_course_name"`
	CourseCode            string `db:"course_code"`
	CourseCodeName        string `db:"course_code_name"`
	// CSV 上にある「データ更新日」
	CSVUpdatedAt time.Time `db:"csv_updated_at"`
	Year         int       `db:"year"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
