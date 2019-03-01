package compliance

import (
	"database/sql"
	"time"
)

type Features []*Feature

type Feature struct {
	Name              string
	Timestamp         time.Time
	CppVersion        int            `db:"cpp_version"`
	PaperName         sql.NullString `db:"paper_name"`
	PaperLink         sql.NullString `db:"paper_link"`
	GccSupport        bool           `db:"gcc_support"`
	GccDisplayText    sql.NullString `db:"gcc_display_text"`
	GccExtraText      sql.NullString `db:"gcc_extra_text"`
	ClangSupport      bool           `db:"clang_support"`
	ClangDisplayText  sql.NullString `db:"clang_display_text"`
	ClangExtraText    sql.NullString `db:"clang_extra_text"`
	MsvcSupport       bool           `db:"msvc_support"`
	MsvcDisplayText   sql.NullString `db:"msvc_display_text"`
	MsvcExtraText     sql.NullString `db:"msvc_extra_text"`
	ReportedToTwitter bool           `db:"reported_to_twitter"`
	ReportedBroken    bool           `db:"reported_broken"`
}
