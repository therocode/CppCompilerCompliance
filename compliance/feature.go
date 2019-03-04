package compliance

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/pkg/errors"
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

const (
	TwitterLimit        = 280
	CppRefLinkSize      = len("https://en.cppreference.com/w/cpp/compiler_support")
	TwitterShortUrlSize = len("https://t.co/iqNEBAK9qG")
	TrimLimit           = TwitterLimit + (CppRefLinkSize - TwitterShortUrlSize)
)

func twitterTrimmed(text string) (result string) {
	if len(text) > TrimLimit {
		result = text[0:TrimLimit-3] + "..."
	} else {
		result = text
	}

	return
}

func fromNullString(text sql.NullString) string {
	if text.Valid {
		return text.String
	} else {
		return ""
	}
}

func compilerSupportString(support bool, displayText string, extraText string) (bit string) {
	if support {
		bit = "yes from version: " + displayText
		extra := extraText
		if len(extra) > 0 {
			bit += "(" + extraText + ")"
		}
	} else {
		bit = "no"
	}

	return
}

func FeatureToTwitterReport(previous *Feature, next *Feature) (string, error) {
	if previous == nil && next != nil {
		//feature not currently there has been added
		//"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
		//"A new feature 'Initializer list constructors in class template argument deduction' has been added for C++20 at cppreferencelink. Current compiler support: GCC: yes from version 4* (not fully supported), Clang: no, MSVC: yes from version 5"
		gccBit := compilerSupportString(next.GccSupport, fromNullString(next.GccDisplayText), fromNullString(next.GccExtraText))
		clangBit := compilerSupportString(next.ClangSupport, fromNullString(next.ClangDisplayText), fromNullString(next.ClangExtraText))
		msvcBit := compilerSupportString(next.MsvcSupport, fromNullString(next.MsvcDisplayText), fromNullString(next.MsvcExtraText))

		var reportText string = "A new feature '" + next.Name + "' has been added for C++" + strconv.Itoa(next.CppVersion) + " at https://en.cppreference.com/w/cpp/compiler_support. Current compiler support: GCC: " + gccBit + ", Clang: " + clangBit + ", MSVC: " + msvcBit

		reportText = twitterTrimmed(reportText)

		return reportText, nil

	} else {
		return "", errors.Errorf("cannot handle")
	}
}
