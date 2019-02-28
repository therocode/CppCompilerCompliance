package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type CppVersion int

const (
	Cpp11 CppVersion = iota
	Cpp14
	Cpp17
	Cpp20
)

func parseCppVersion(text string) (CppVersion, error) {
	if strings.Contains(text, "11") {
		return Cpp11, nil
	} else if strings.Contains(text, "14") {
		return Cpp14, nil
	} else if strings.Contains(text, "17") {
		return Cpp17, nil
	} else if strings.Contains(text, "2a") || strings.Contains(text, "20") {
		return Cpp20, nil
	}

	return Cpp11, fmt.Errorf("could not parse CPP version from '%s'", text)
}

type CompilerSupport struct {
	HasSupport    bool
	DisplayString string
	ExtraString   string
}

type CppFeature struct {
	Name      string
	PaperName string
	PaperLink string

	GccSupport   CompilerSupport
	ClangSupport CompilerSupport
	MsvcSupport  CompilerSupport
}

type CppVersionSupport struct {
	VersionId CppVersion
	Features  []CppFeature
}

type CppSupport struct {
	Versions []CppVersionSupport
}

func ScrapeCppSupport() (result CppSupport) {
	// Make HTTP request
	response, err := http.Get("https://en.cppreference.com/w/cpp/compiler_support")
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	// Create a goquery document from the HTTP response
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}

	document.Find(".mw-headline").Each(func(index int, element *goquery.Selection) {
		titleText := element.Text()

		if !strings.Contains(titleText, "features") {
			return
		}

		fmt.Printf("C++ version: %v\n", titleText)

		cppVersion, err := parseCppVersion(titleText)
		if err != nil {
			log.Print(err)
			return
		}

		versionData := CppVersionSupport{}
		versionData.VersionId = cppVersion

		table := element.Parent()

		hasTable := table.Has("tr")

		for hasTable.Length() == 0 {
			table = table.Next()

			if table.Length() == 0 {
				break
			}

			hasTable = table.Has("tr")
		}

		if table.Length() == 0 {
			println("had no table...")
		}

		table.Find("tr").Each(func(rowIndex int, rowElement *goquery.Selection) {
			isHeading := rowElement.Has("th").Length() > 0

			if isHeading {
				return
			}

			featureData := CppFeature{}

			titleDataElement := rowElement.Children().First()
			featureTitle := titleDataElement.Text()
			featureTitle = strings.TrimSpace(featureTitle)

			featureData.Name = featureTitle

			paperDataElement := titleDataElement.Next()
			hrefElement := paperDataElement.First().Children().First()
			featurePaperTitle := hrefElement.Text()
			featurePaperTitle = strings.TrimSpace(featurePaperTitle)
			featurePaperLink := hrefElement.AttrOr("href", "NO LINK")
			featurePaperLink = strings.TrimSpace(featurePaperLink)

			featureData.PaperName = featurePaperTitle
			featureData.PaperLink = featurePaperLink

			//paperDataElement.Next() //version data element

			gccDataElement := paperDataElement.Next().Next()
			gccSupports := gccDataElement.HasClass("table-yes")
			gccSupportsString := gccDataElement.Text()
			gccSupportsString = strings.TrimSpace(gccSupportsString)
			gccSupportsStringExtra := gccDataElement.Children().First().AttrOr("title", "")
			gccSupportsStringExtra = strings.TrimSpace(gccSupportsStringExtra)

			featureData.GccSupport.HasSupport = gccSupports
			featureData.GccSupport.DisplayString = gccSupportsString
			featureData.GccSupport.ExtraString = gccSupportsStringExtra

			clangDataElement := gccDataElement.Next().Next()
			clangSupports := clangDataElement.HasClass("table-yes")
			clangSupportsString := clangDataElement.Text()
			clangSupportsString = strings.TrimSpace(clangSupportsString)
			clangSupportsStringExtra := clangDataElement.Children().First().AttrOr("title", "")
			clangSupportsStringExtra = strings.TrimSpace(clangSupportsStringExtra)

			featureData.ClangSupport.HasSupport = clangSupports
			featureData.ClangSupport.DisplayString = clangSupportsString
			featureData.ClangSupport.ExtraString = clangSupportsStringExtra

			msvcDataElement := clangDataElement.Next().Next()
			msvcSupports := msvcDataElement.HasClass("table-yes")
			msvcSupportsString := msvcDataElement.Text()
			msvcSupportsString = strings.TrimSpace(msvcSupportsString)
			msvcSupportsStringExtra := msvcDataElement.Children().First().AttrOr("title", "")
			msvcSupportsStringExtra = strings.TrimSpace(msvcSupportsStringExtra)

			featureData.MsvcSupport.HasSupport = msvcSupports
			featureData.MsvcSupport.DisplayString = msvcSupportsString
			featureData.MsvcSupport.ExtraString = msvcSupportsStringExtra

			//fmt.Printf("href elem:%v\n", goquery.NodeName(hrefElement))
			fmt.Printf("title: %v, paper: %v, link: %v\n", featureTitle, featurePaperTitle, featurePaperLink)
			fmt.Printf("  gcc support: %v - %v (%v)\n", gccSupports, gccSupportsString, gccSupportsStringExtra)
			fmt.Printf("  clang support: %v - %v (%v)\n", clangSupports, clangSupportsString, clangSupportsStringExtra)
			fmt.Printf("  msvc support: %v - %v (%v)\n", msvcSupports, msvcSupportsString, msvcSupportsStringExtra)

			versionData.Features = append(versionData.Features, featureData)
		})

		result.Versions = append(result.Versions, versionData)
	})

	return result
}
