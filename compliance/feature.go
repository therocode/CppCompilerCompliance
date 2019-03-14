package compliance

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

const (
	SupportNo      = 0
	SupportYes     = 1
	SupportPartial = 2
)

type Features []*Feature

type Feature struct {
	Name              string
	Timestamp         time.Time
	CppVersion        int            `db:"cpp_version"`
	PaperName         sql.NullString `db:"paper_name"`
	PaperLink         sql.NullString `db:"paper_link"`
	GccSupport        int            `db:"gcc_support"`
	GccDisplayText    sql.NullString `db:"gcc_display_text"`
	GccExtraText      sql.NullString `db:"gcc_extra_text"`
	ClangSupport      int            `db:"clang_support"`
	ClangDisplayText  sql.NullString `db:"clang_display_text"`
	ClangExtraText    sql.NullString `db:"clang_extra_text"`
	MsvcSupport       int            `db:"msvc_support"`
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

func countTrue(bools ...bool) (result int) {
	for _, b := range bools {
		if b {
			result++
		}
	}

	return
}

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

func compilerVersionTextString(displayText string, extraText string) string {
	result := displayText
	extra := extraText
	if len(extra) > 0 {
		result += "(" + extraText + ")"
	}
	return result
}

func compilerSupportString(support int, displayText string, extraText string) string {
	if support == 0 {
		return "no"
	} else if support == 1 {
		return "yes - " + compilerVersionTextString(displayText, extraText)
	} else {
		return "partial - " + compilerVersionTextString(displayText, extraText)
	}
}

func compilerSupportStringOrNothing(support int, displayText string, extraText string) string {
	if support != 0 {
		return compilerSupportString(support, displayText, extraText)
	} else {
		return ""
	}
}

func compilerTextChangeStringOrNothing(previousDisplayText string, previousExtraText string, nextDisplayText string, nextExtraText string) string {
	previousText := compilerVersionTextString(previousDisplayText, previousExtraText)
	nextText := compilerVersionTextString(nextDisplayText, nextExtraText)

	bothEmpty := previousText == "" && nextText == ""

	if !bothEmpty {
		return fmt.Sprintf("\"%v\" -> \"%v\"", previousText, nextText)
	} else {
		return ""
	}
}

func isReportTypeNewFeatureAdded(previous *Feature, next *Feature) bool {
	return previous == nil && next != nil
}

func isReportTypeSupportLevelChanged(previous *Feature, next *Feature) bool {
	return (previous.GccSupport != next.GccSupport) ||
		(previous.ClangSupport != next.ClangSupport) ||
		(previous.MsvcSupport != next.MsvcSupport)
}

func isReportTypeTextChanged(previous *Feature, next *Feature) bool {
	return (previous.GccDisplayText != next.GccDisplayText) ||
		(previous.GccExtraText != next.GccExtraText) ||
		(previous.ClangDisplayText != next.ClangDisplayText) ||
		(previous.ClangExtraText != next.ClangExtraText) ||
		(previous.MsvcDisplayText != next.MsvcDisplayText) ||
		(previous.MsvcExtraText != next.MsvcExtraText)
}

func FeatureToTwitterReport(previous *Feature, next *Feature) (string, error) {
	if isReportTypeNewFeatureAdded(previous, next) {
		//feature not currently there has been added
		//"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
		//"A new feature \"Initializer list constructors in class template argument deduction\" has been added for C++20 at cppreferencelink. Current compiler support: GCC: yes from version 4* (not fully supported), Clang: no, MSVC: yes from version 5"
		gccBit := compilerSupportString(next.GccSupport, fromNullString(next.GccDisplayText), fromNullString(next.GccExtraText))
		clangBit := compilerSupportString(next.ClangSupport, fromNullString(next.ClangDisplayText), fromNullString(next.ClangExtraText))
		msvcBit := compilerSupportString(next.MsvcSupport, fromNullString(next.MsvcDisplayText), fromNullString(next.MsvcExtraText))

		reportText := fmt.Sprintf("A new C++%v feature \"%v\" has been added at https://en.cppreference.com/w/cpp/compiler_support. Current compiler support: GCC: %v, Clang: %v, MSVC: %v", next.CppVersion, next.Name, gccBit, clangBit, msvcBit)
		reportText = twitterTrimmed(reportText)

		return reportText, nil

	} else if isReportTypeSupportLevelChanged(previous, next) {

		gccBit := compilerSupportString(next.GccSupport, fromNullString(next.GccDisplayText), fromNullString(next.GccExtraText))
		clangBit := compilerSupportString(next.ClangSupport, fromNullString(next.ClangDisplayText), fromNullString(next.ClangExtraText))
		msvcBit := compilerSupportString(next.MsvcSupport, fromNullString(next.MsvcDisplayText), fromNullString(next.MsvcExtraText))

		reportText := fmt.Sprintf("Support for the C++%v feature \"%v\" has been updated at https://en.cppreference.com/w/cpp/compiler_support. Support after change: ", next.CppVersion, next.Name)

		reportText += "GCC: " + gccBit
		reportText += ", "
		reportText += "Clang: " + clangBit

		reportText += ", "
		reportText += "MSVC: " + msvcBit

		reportText = twitterTrimmed(reportText)

		return reportText, nil
	} else if isReportTypeTextChanged(previous, next) {
		gccTextChanged := previous.GccDisplayText != next.GccDisplayText || previous.GccExtraText != next.GccExtraText
		clangTextChanged := previous.ClangDisplayText != next.ClangDisplayText || previous.ClangExtraText != next.ClangExtraText
		msvcTextChanged := previous.MsvcDisplayText != next.MsvcDisplayText || previous.MsvcExtraText != next.MsvcExtraText

		gccBit := compilerTextChangeStringOrNothing(fromNullString(previous.GccDisplayText), fromNullString(previous.GccExtraText), fromNullString(next.GccDisplayText), fromNullString(next.GccExtraText))
		clangBit := compilerTextChangeStringOrNothing(fromNullString(previous.ClangDisplayText), fromNullString(previous.ClangExtraText), fromNullString(next.ClangDisplayText), fromNullString(next.ClangExtraText))
		msvcBit := compilerTextChangeStringOrNothing(fromNullString(previous.MsvcDisplayText), fromNullString(previous.MsvcExtraText), fromNullString(next.MsvcDisplayText), fromNullString(next.MsvcExtraText))

		var pluralSuffix string

		if countTrue(gccTextChanged, clangTextChanged, msvcTextChanged) > 1 {
			pluralSuffix += "s"
		}

		reportText := fmt.Sprintf("Support text%v changed for the C++%v feature \"%v\". Change%v: ", pluralSuffix, next.CppVersion, next.Name, pluralSuffix)

		if gccTextChanged {
			reportText += "GCC: " + gccBit
		}

		if clangTextChanged {
			if gccTextChanged {
				reportText += ", "
			}

			reportText += "Clang: " + clangBit
		}

		if msvcTextChanged {
			if gccTextChanged || clangTextChanged {
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
