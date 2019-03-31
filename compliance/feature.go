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
		return "[no]"
	} else if support == 1 {
		return "[yes] " + compilerVersionTextString(displayText, extraText)
	} else {
		return "[partial] " + compilerVersionTextString(displayText, extraText)
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

func compilerSupportListing(feature *Feature, listGcc bool, listClang bool, listMsvc bool) (result string) {
	gccBit := "GCC - " + compilerSupportString(feature.GccSupport, fromNullString(feature.GccDisplayText), fromNullString(feature.GccExtraText))
	clangBit := "Clang - " + compilerSupportString(feature.ClangSupport, fromNullString(feature.ClangDisplayText), fromNullString(feature.ClangExtraText))
	msvcBit := "MSVC - " + compilerSupportString(feature.MsvcSupport, fromNullString(feature.MsvcDisplayText), fromNullString(feature.MsvcExtraText))

	first := true

	if listGcc {
		result += gccBit
		first = false
	}
	if listClang {
		if !first {
			result += "\n"
		}
		result += clangBit

		first = false
	}
	if listMsvc {
		if !first {
			result += "\n"
		}
		result += msvcBit

		first = false
	}

	return
}

func FeatureToTwitterReport(previous *Feature, next *Feature) (string, error) {
	if isReportTypeNewFeatureAdded(previous, next) {
		supportListing := compilerSupportListing(next, true, true, true)

		reportText := fmt.Sprintf("[New Listing] C++%v - \"%v\".\n\nSupport:\n%v", next.CppVersion, next.Name, supportListing)
		reportText = twitterTrimmed(reportText)

		return reportText, nil

	} else if isReportTypeSupportLevelChanged(previous, next) {

		listGcc := previous.GccSupport != next.GccSupport
		listClang := previous.ClangSupport != next.ClangSupport
		listMsvc := previous.MsvcSupport != next.MsvcSupport

		previousSupportListing := compilerSupportListing(previous, listGcc, listClang, listMsvc)
		nextSupportListing := compilerSupportListing(next, listGcc, listClang, listMsvc)

		reportText := fmt.Sprintf("[Support Update] C++%v - \"%v\".\n\nFrom:\n%v\n\nto:\n%v", next.CppVersion, next.Name, previousSupportListing, nextSupportListing)
		reportText = twitterTrimmed(reportText)

		return reportText, nil
	} else if isReportTypeTextChanged(previous, next) {
		listGcc := previous.GccDisplayText != next.GccDisplayText || previous.GccExtraText != next.GccExtraText
		listClang := previous.ClangDisplayText != next.ClangDisplayText || previous.ClangExtraText != next.ClangExtraText
		listMsvc := previous.MsvcDisplayText != next.MsvcDisplayText || previous.MsvcExtraText != next.MsvcExtraText

		previousSupportListing := compilerSupportListing(previous, listGcc, listClang, listMsvc)
		nextSupportListing := compilerSupportListing(next, listGcc, listClang, listMsvc)

		reportText := fmt.Sprintf("[Text Update] C++%v - \"%v\".\n\nFrom:\n%v\n\nto:\n%v", next.CppVersion, next.Name, previousSupportListing, nextSupportListing)
		reportText = twitterTrimmed(reportText)

		return reportText, nil
	} else {
		return "", errors.Errorf("cannot handle")
	}
}
