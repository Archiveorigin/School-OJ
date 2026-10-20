package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

type standingsDisplayCell struct {
	Text  string
	Class string
}
type standingsDisplay struct {
	Title, Meta, Status string
	Headers             []string
	Rows                [][]standingsDisplayCell
}

func standingsPresentation(data eventStandings) standingsDisplay {
	status := "待审核"
	if data.Lifecycle == "completed" {
		status = "已确认"
	}
	view := standingsDisplay{
		Title: data.Title, Status: status,
		Meta:    fmt.Sprintf("%s · %d 位参赛者", rankingTime(data.EndsAt), len(data.Rows)),
		Headers: []string{"名次", "参赛者", "编号"},
	}
	if data.Kind == "exam" {
		view.Headers = append(view.Headers, "学号")
	}
	if data.ScoringRule == "acm" {
		view.Headers = append(view.Headers, "通过", "罚时 / 分钟")
	} else {
		view.Headers = append(view.Headers, "总分", "满分题")
	}
	for _, problem := range data.Problems {
		view.Headers = append(view.Headers, problem.Label+" · "+problem.Title)
	}
	for _, row := range data.Rows {
		cells := []standingsDisplayCell{{fmt.Sprint(row.Rank), "rank"}, {row.Name, "name"}, {fmt.Sprint(row.UserID), ""}}
		if data.Kind == "exam" {
			cells = append(cells, standingsDisplayCell{row.studentNo, ""})
		}
		if data.ScoringRule == "acm" {
			cells = append(cells, standingsDisplayCell{fmt.Sprint(row.Solved), "total"}, standingsDisplayCell{fmt.Sprint(row.PenaltyMinutes), "total"})
		} else {
			cells = append(cells, standingsDisplayCell{fmt.Sprintf("%d / %d", row.TotalScore, row.MaxScore), "total"}, standingsDisplayCell{fmt.Sprint(row.Solved), "total"})
		}
		for _, problem := range row.Problems {
			cell := standingsDisplayCell{"—", "none"}
			attempted := problem.Attempts > 0 || problem.Status != ""
			accepted := problem.Status == "accepted"
			if data.ScoringRule != "acm" {
				accepted = problem.MaxScore > 0 && problem.BestScore >= problem.MaxScore
			}
			if attempted {
				switch {
				case accepted:
					cell.Class = "accepted"
				case problem.Pending:
					cell.Class = "pending"
				default:
					cell.Class = "wrong"
				}
				if data.ScoringRule == "acm" {
					if accepted {
						cell.Text = fmt.Sprintf("+%d\n%d′", problem.WrongAttempts, problem.ElapsedMinutes)
					} else if problem.Pending {
						cell.Text = fmt.Sprintf("?%d", problem.Attempts)
					} else {
						cell.Text = fmt.Sprintf("−%d", problem.Attempts)
					}
				} else {
					cell.Text = fmt.Sprintf("%d / %d", problem.BestScore, problem.MaxScore)
					if problem.Pending {
						cell.Text += "\n待评测"
					}
				}
				if accepted && problem.Fastest {
					cell.Class = "first"
					cell.Text = "★ " + cell.Text
				}
			}
			cells = append(cells, cell)
		}
		view.Rows = append(view.Rows, cells)
	}
	return view
}

func rankingTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "—"
	}
	return value.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006年01月02日 15:04")
}

// Portable, printable standings: no scripts, remote assets, or authenticated
// URLs in downloaded files. html/template escapes all user-controlled content.
func renderStandingsHTML(data eventStandings) ([]byte, error) {
	const source = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title>
<style>
:root{font-family:"Segoe UI","Microsoft YaHei",sans-serif;color:#192323;background:#f5f6f3;font-variant-numeric:tabular-nums}
*{box-sizing:border-box}body{margin:0;padding:56px 40px}main{max-width:1440px;margin:auto;background:#fff;padding:40px 44px;border:1px solid #e6e9e4;border-radius:12px}
header{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;margin-bottom:36px}h1{font-size:28px;line-height:1.35;letter-spacing:.02em;margin:0 0 12px;font-weight:650}.meta{margin:0;font-size:13px;color:#78817a}.status{flex:none;font-size:12px;color:#546b5d;padding:6px 12px;border-radius:20px;background:#f0f4ef;border:1px solid #dfe8de}
.board{overflow:auto}table{width:100%;border-collapse:separate;border-spacing:0;font-size:14px}th{padding:14px 12px;background:#f5f7f3;color:#6b766e;font-size:12px;font-weight:500;max-width:190px;border-top:1px solid #e5eae1;border-bottom:1px solid #e5eae1;line-height:1.6}th:first-child{border-radius:7px 0 0 7px}th:last-child{border-radius:0 7px 7px 0}td{padding:10px 12px;min-width:68px;border-bottom:1px solid #edf0e9;text-align:center;height:78px}th:nth-child(2),td.name{text-align:left;min-width:150px}td.name{font-size:15px;font-weight:600;color:#25372a}td.rank{font-size:21px;font-weight:600;color:#3e5043;width:65px}td.total{font-size:17px;font-weight:650;color:#25372a}td:nth-child(3){color:#99a198;font-size:12px}.result{min-height:48px;display:flex;align-items:center;justify-content:center;white-space:pre-line;line-height:1.45;border-radius:6px;padding:6px 12px;min-width:66px}
.name .result{justify-content:flex-start;padding-left:0}.result.accepted{background:#edf6eb;color:#3d7441}.result.first{background:#dcefd9;color:#286135;box-shadow:inset 0 0 0 1px #c0dbbb;font-weight:650}.result.wrong{background:#fbf0ed;color:#b06f5b}.result.pending{background:#fbf5df;color:#9a7d2b}.result.none{color:#c4cbc0}.legend{display:flex;justify-content:flex-end;gap:20px;flex-wrap:wrap;margin-top:22px;color:#8a948b;font-size:11px}.legend span{display:flex;align-items:center;gap:6px}.dot{display:inline-block;width:7px;height:7px;border-radius:50%;background:#d0d7cb}.dot.first{background:#75a56c}.dot.accepted{background:#bdd6b6}.dot.wrong{background:#e3b6a8}
@media(max-width:600px){body{padding:14px}main{padding:22px 18px}header{gap:12px;margin-bottom:24px}h1{font-size:22px}.meta{font-size:11px}.status{font-size:11px;padding:5px 9px}td{height:68px}.legend{justify-content:flex-start;gap:14px}}
@media print{@page{size:landscape;margin:12mm}body{padding:0;background:#fff}main{padding:0;border:0;max-width:none}.board{overflow:visible}header{margin-bottom:22px}h1{font-size:24px}table{font-size:11px}th{font-size:10px}th,td{padding:6px;min-width:0}td{height:56px}td.name{font-size:12px;min-width:105px}td.rank,td.total{font-size:15px}.result{min-width:0;min-height:36px;padding:4px}thead{display:table-header-group}tr{break-inside:avoid}*{-webkit-print-color-adjust:exact;print-color-adjust:exact}}
</style></head><body><main><header><div><h1>{{.Title}}</h1><p class="meta">{{.Meta}}</p></div><span class="status">{{.Status}}</span></header>
<div class="board"><table><thead><tr>{{range .Headers}}<th scope="col">{{.}}</th>{{end}}</tr></thead><tbody>{{range .Rows}}<tr>{{range .}}<td class="{{.Class}}"><div class="result {{.Class}}">{{.Text}}</div></td>{{end}}</tr>{{else}}<tr><td colspan="{{len .Headers}}">暂无参赛者</td></tr>{{end}}</tbody></table></div>
<div class="legend"><span><i class="dot first"></i>首次通过</span><span><i class="dot accepted"></i>通过</span><span><i class="dot wrong"></i>未通过</span><span><i class="dot"></i>未提交</span></div></main></body></html>`
	t, err := template.New("standings").Parse(source)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	err = t.Execute(&b, standingsPresentation(data))
	return b.Bytes(), err
}

func renderStandingsXLSX(data eventStandings) ([]byte, error) {
	view := standingsPresentation(data)
	rows := [][]xlsxCell{
		{{Value: view.Title, Style: 1}},
		{{Value: view.Status + " · " + view.Meta, Style: 2}},
		{},
	}
	header := []xlsxCell{}
	for _, title := range view.Headers {
		header = append(header, xlsxCell{Value: title, Style: 3})
	}
	rows = append(rows, header)
	styles := map[string]int{"first": 5, "accepted": 6, "wrong": 7, "pending": 8}
	for _, row := range view.Rows {
		cells := []xlsxCell{}
		for index, cell := range row {
			style := styles[cell.Class]
			if style == 0 {
				style = 4
			}
			// Inline strings keep IDs/leading zeros and names starting with = safe.
			number := index == 0 || index == 2 || view.Headers[index] == "通过" || view.Headers[index] == "满分题" || view.Headers[index] == "罚时 / 分钟"
			cells = append(cells, xlsxCell{Value: cell.Text, Style: style, Number: number})
		}
		rows = append(rows, cells)
	}
	sheet := buildSheetXML(rows)
	prefix := `<sheetViews><sheetView workbookViewId="0"><pane xSplit="2" ySplit="4" topLeftCell="C5" activePane="bottomRight" state="frozen"/></sheetView></sheetViews><sheetFormatPr defaultRowHeight="38"/><cols><col min="1" max="1" width="8" customWidth="1"/><col min="2" max="2" width="24" customWidth="1"/><col min="3" max="` + fmt.Sprint(len(view.Headers)) + `" width="17" customWidth="1"/></cols>`
	sheet = strings.Replace(sheet, "<sheetData>", prefix+"<sheetData>", 1)
	sheet = strings.Replace(sheet, `<row r="1">`, `<row r="1" ht="44" customHeight="1">`, 1)
	sheet = strings.Replace(sheet, `<row r="2">`, `<row r="2" ht="24" customHeight="1">`, 1)
	sheet = strings.Replace(sheet, `<row r="3">`, `<row r="3" ht="12" customHeight="1">`, 1)
	lastColumn := xlsxColumnName(len(view.Headers))
	merges := `<mergeCells count="2">`
	for i := 1; i <= 2; i++ {
		merges += fmt.Sprintf(`<mergeCell ref="A%d:%s%d"/>`, i, lastColumn, i)
	}
	merges += `</mergeCells>`
	sheet = strings.Replace(sheet, "</sheetData>", fmt.Sprintf(`</sheetData><autoFilter ref="A4:%s%d"/>%s<pageMargins left="0.3" right="0.3" top="0.4" bottom="0.4" header="0.2" footer="0.2"/><pageSetup orientation="landscape" paperSize="9" fitToWidth="1" fitToHeight="0"/>`, lastColumn, len(rows), merges), 1)
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
