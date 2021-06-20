
-- +migrate Up

-- とりあえず筑波大学限定の話ということで……
-- https://www.tsukuba.ac.jp/education/ug-courses-openclass/2021/pdf/1.pdf
create type instructional_type as enum ('0', '1', '2', '3', '4', '5', '6', '7', '8');

-- 科目等履修生は、マル・バツ・空文字列の3択であるため
create type credited_auditors as enum ('0', '1', '2');

create table if not exists syllabus (
		id serial not null,
		course_number varchar(16) not null, -- 科目番号
		course_name varchar(256) not null, -- 科目名
		instructional_type instructional_type not null, -- 授業方法
		credits numeric, -- 単位数
		standard_registration_year int[] not null, -- 標準履修年次
		term int[], -- 開講時期
		period_ varchar(16)[] not null, -- 曜時限
		classroom varchar(256), -- 教室
		instructor varchar(32)[] not null, -- 担当教員
		course_overview text, -- 授業概要
		remarks text, -- 備考
		credited_auditors credited_auditors not null, -- 科目履修生申請可否
		application_conditions varchar(256), -- 申請条件
		alt_course_name varchar(256), -- 英語（日本語）科目名
		course_code varchar(16), -- 科目コード
		course_code_name varchar(256), -- 要件科目名
		csv_updated_at timestamp with time zone not null, -- csvに記載されている更新日時
    year int not null, -- 何年度にエクスポートした CSV であるか
		created_at timestamp with time zone not null, -- 最初のデータ作成日時
		updated_at timestamp with time zone not null, -- 実際の最終更新日時
		primary key (id)
	);

-- +migrate Down
