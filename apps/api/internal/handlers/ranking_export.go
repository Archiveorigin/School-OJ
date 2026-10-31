package handlers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// The production image builds this standalone page from the same Vue components
// as the live board. No CDN, API connection or login is needed after download.
func offlineRankingTemplate() ([]byte, error) {
	paths := []string{os.Getenv("OJ_RANKING_TEMPLATE"), "/usr/local/share/school-oj/offline-ranking.html", "../web/dist/offline-ranking.html", "../../../web/dist/offline-ranking.html"}
	var err error
	for _, path := range paths {
		if path == "" {
			continue
		}
		var body []byte
		body, err = os.ReadFile(filepath.Clean(path))
		if err == nil {
			return body, nil
		}
	}
	return nil, err
}
func renderStandingsHTML(data eventStandings) ([]byte, error) {
	source, err := offlineRankingTemplate()
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return []byte(strings.Replace(string(source), "__OJ_RANKING_DATA__", string(payload), 1)), nil
}

func rankingExportName(title, format string) string {
	title = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, title))
	title = strings.TrimRight(title, ". ")
	if title == "" {
		title = "排行榜"
	}
	return title + "." + format
}

func rankingExportDisposition(title, format string) string {
	return fmt.Sprintf("attachment; filename=\"standings.%s\"; filename*=UTF-8''%s", format, url.PathEscape(rankingExportName(title, format)))
}

func renderStandingsXLSX(data eventStandings) ([]byte, error) {
	rows := [][]xlsxCell{{{Value: "名次", Style: 3}, {Value: "参赛者", Style: 3}, {Value: "通过题数", Style: 3}, {Value: "罚时 / 分钟", Style: 3}}}
	for _, row := range data.Rows {
		rank := fmt.Sprint(row.Rank)
		if row.Unranked {
			rank = "*"
		}
		rows = append(rows, []xlsxCell{
			{Value: rank, Style: 4, Number: !row.Unranked},
			{Value: row.Name, Style: 4},
			{Value: fmt.Sprint(row.Solved), Style: 4, Number: true},
			{Value: fmt.Sprint(row.PenaltyMinutes), Style: 4, Number: true},
		})
	}
	sheet := buildSheetXML(rows)
	prefix := `<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews><sheetFormatPr defaultRowHeight="28"/><cols><col min="1" max="1" width="10" customWidth="1"/><col min="2" max="2" width="28" customWidth="1"/><col min="3" max="4" width="18" customWidth="1"/></cols>`
	sheet = strings.Replace(sheet, "<sheetData>", prefix+"<sheetData>", 1)
	sheet = strings.Replace(sheet, "</sheetData>", fmt.Sprintf(`</sheetData><autoFilter ref="A1:D%d"/>`, len(rows)), 1)
	return buildWorkbookXLSX(sheet, standingsXLSXStyles)
}

const standingsXLSXStyles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<fonts count="4"><font><sz val="11"/><name val="Microsoft YaHei"/><color rgb="FF25372A"/></font><font><b/><sz val="22"/><name val="Microsoft YaHei"/><color rgb="FF25372A"/></font><font><b/><sz val="11"/><name val="Microsoft YaHei"/><color rgb="FF3D7441"/></font><font><sz val="10"/><name val="Microsoft YaHei"/><color rgb="FF78817A"/></font></fonts>
<fills count="7"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FFF5F7F3"/></patternFill></fill><fill><patternFill patternType="solid"><fgColor rgb="FFDCEFD9"/></patternFill></fill><fill><patternFill patternType="solid"><fgColor rgb="FFEDF6EB"/></patternFill></fill><fill><patternFill patternType="solid"><fgColor rgb="FFFBF0ED"/></patternFill></fill><fill><patternFill patternType="solid"><fgColor rgb="FFFBF5DF"/></patternFill></fill></fills>
<borders count="2"><border/><border><left/><right/><top/><bottom style="hair"><color rgb="FFE5EAE1"/></bottom></border></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="9">
<xf numFmtId="0" fontId="0" fillId="0" borderId="0"/>
<xf numFmtId="0" fontId="1" fillId="0" borderId="0" applyFont="1"><alignment vertical="center"/></xf>
<xf numFmtId="0" fontId="3" fillId="0" borderId="0" applyFont="1"><alignment vertical="center" wrapText="1"/></xf>
<xf numFmtId="0" fontId="2" fillId="2" borderId="1" applyFont="1" applyFill="1" applyBorder="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>
<xf numFmtId="0" fontId="0" fillId="0" borderId="1" applyBorder="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>
<xf numFmtId="0" fontId="2" fillId="3" borderId="1" applyFont="1" applyFill="1" applyBorder="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>
<xf numFmtId="0" fontId="0" fillId="4" borderId="1" applyFill="1" applyBorder="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>
<xf numFmtId="0" fontId="0" fillId="5" borderId="1" applyFill="1" applyBorder="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>
<xf numFmtId="0" fontId="0" fillId="6" borderId="1" applyFill="1" applyBorder="1"><alignment horizontal="center" vertical="center" wrapText="1"/></xf>
</cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`
