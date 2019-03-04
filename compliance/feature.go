package compliance

import (
	"database/sql"
	"fmt"
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

func compilerSupportString(support bool, displayText string, extraText string) string {
	if support {
		result := "yes from version: " + displayText
		extra := extraText
		if len(extra) > 0 {
			result += "(" + extraText + ")"
		}
		return result
	} else {
		return "no"
	}
}

func compilerSupportStringOrNothing(support bool, displayText string, extraText string) string {
	if support {
		return compilerSupportString(true, displayText, extraText)
	} else {
		return ""
	}
}
func isReportTypeNewFeatureAdded(previous *Feature, next *Feature) bool {
	return previous == nil && next != nil
}

func isReportTypeSupportAdded(previous *Feature, next *Feature) bool {
	return (!previous.GccSupport && next.GccSupport) ||
		(!previous.ClangSupport && next.ClangSupport) ||
		(!previous.MsvcSupport && next.MsvcSupport)
}

func FeatureToTwitterReport(previous *Feature, next *Feature) (string, error) {
	if isReportTypeNewFeatureAdded(previous, next) {
		//feature not currently there has been added
		//"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
		//"A new feature 'Initializer list constructors in class template argument deduction' has been added for C++20 at cppreferencelink. Current compiler support: GCC: yes from version 4* (not fully supported), Clang: no, MSVC: yes from version 5"
		gccBit := compilerSupportString(next.GccSupport, fromNullString(next.GccDisplayText), fromNullString(next.GccExtraText))
		clangBit := compilerSupportString(next.ClangSupport, fromNullString(next.ClangDisplayText), fromNullString(next.ClangExtraText))
		msvcBit := compilerSupportString(next.MsvcSupport, fromNullString(next.MsvcDisplayText), fromNullString(next.MsvcExtraText))

		reportText := fmt.Sprintf("A new C++%v feature '%v' has been added at https://en.cppreference.com/w/cpp/compiler_support. Current compiler support: GCC: %v, Clang: %v, MSVC: %v", next.CppVersion, next.Name, gccBit, clangBit, msvcBit)
		reportText = twitterTrimmed(reportText)

		return reportText, nil

	} else if isReportTypeSupportAdded(previous, next) {
		gccBit := compilerSupportStringOrNothing(!previous.GccSupport && next.GccSupport, fromNullString(next.GccDisplayText), fromNullString(next.GccExtraText))
		clangBit := compilerSupportStringOrNothing(!previous.ClangSupport && next.ClangSupport, fromNullString(next.ClangDisplayText), fromNullString(next.ClangExtraText))
		msvcBit := compilerSupportStringOrNothing(!previous.MsvcSupport && next.MsvcSupport, fromNullString(next.MsvcDisplayText), fromNullString(next.MsvcExtraText))

		reportText := fmt.Sprintf("Support for the C++%v feature '%v' has been updated at https://en.cppreference.com/w/cpp/compiler_support. Support added: ", next.CppVersion, next.Name)

		if len(gccBit) > 0 {
			reportText += "GCC: " + gccBit
		}

		if len(clangBit) > 0 {

			if len(gccBit) > 0 {
				reportText += ", "
			}

			reportText += "Clang: " + clangBit
		}

		if len(msvcBit) > 0 {

			if len(clangBit) > 0 || len(gccBit) > 0 {
				reportText += ", "
			}

			reportText += "MSVC: " + msvcBit
		}

		reportText = twitterTrimmed(reportText)

		return reportText, nil
	} else {
		return "", errors.Errorf("cannot handle")
	}
}
