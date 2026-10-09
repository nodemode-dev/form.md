package transfer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func zipFiles(data []byte) (map[string]*zip.File, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	out := map[string]*zip.File{}
	for _, f := range zr.File {
		out[f.Name] = f
	}
	return out, nil
}

func readZipFile(files map[string]*zip.File, name string) []byte {
	f, ok := files[name]
	if !ok || f.UncompressedSize64 > maxEntryBytes {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()
	data, _ := io.ReadAll(io.LimitReader(rc, maxEntryBytes))
	return data
}

func relsOf(files map[string]*zip.File, name string) map[string]string {
	out := map[string]string{}
	data := readZipFile(files, name)
	if data == nil {
		return out
	}
	var rels struct {
		Rel []struct {
			ID         string `xml:"Id,attr"`
			Target     string `xml:"Target,attr"`
			TargetMode string `xml:"TargetMode,attr"`
		} `xml:"Relationship"`
	}
	_ = xml.Unmarshal(data, &rels)
	for _, r := range rels.Rel {
		out[r.ID] = r.Target
	}
	return out
}

type docxRun struct {
	text   string
	bold   bool
	italic bool
	strike bool
	under  bool
	link   string
	image  string
}

type docxPara struct {
	style string
	level int
	numID string
	runs  []docxRun
	table [][]string
}

func importDOCX(name string, data []byte, b *Bundle) {
	files, err := zipFiles(data)
	if err != nil {
		b.Warn("The Word file " + name + " could not be read")
		return
	}
	doc := readZipFile(files, "word/document.xml")
	if doc == nil {
		b.Warn("The Word file " + name + " has no document part")
		return
	}
	rels := relsOf(files, "word/_rels/document.xml.rels")
	numFmt := docxNumbering(readZipFile(files, "word/numbering.xml"))
	styles := docxHeadingStyles(readZipFile(files, "word/styles.xml"))
	assets := map[string]*Asset{}
	image := func(rid string) string {
		target, ok := rels[rid]
		if !ok {
			return ""
		}
		full := cleanPath(path.Join("word", target))
		a, ok := assets[full]
		if !ok {
			raw := readZipFile(files, full)
			if raw == nil {
				return ""
			}
			a = &Asset{Name: path.Base(full), Data: raw}
			assets[full] = a
		}
		return b.AssetRef(a)
	}
	paras := parseDOCX(doc, rels, image)
	title := stripExt(name)
	var blocks []string
	for i, p := range paras {
		if p.table != nil {
			blocks = append(blocks, mdTable(p.table))
			continue
		}
		text := docxInline(p.runs)
		if strings.TrimSpace(text) == "" {
			continue
		}
		lvl := 0
		st := strings.ToLower(p.style)
		if h, ok := styles[st]; ok {
			lvl = h
		} else if strings.HasPrefix(st, "heading") {
			lvl, _ = strconv.Atoi(strings.TrimPrefix(st, "heading"))
		}
		if st == "title" {
			if i < 3 {
				title = oneLine(text)
				continue
			}
			lvl = 1
		}
		switch {
		case lvl >= 1 && lvl <= 6:
			blocks = append(blocks, strings.Repeat("#", min(lvl, 3))+" "+oneLine(text))
		case p.numID != "" && p.numID != "0":
			marker := "- "
			if numFmt[p.numID+":"+strconv.Itoa(p.level)] {
				marker = "1. "
			}
			blocks = append(blocks, strings.Repeat("  ", p.level)+marker+oneLineKeepBreaks(text))
		case strings.Contains(st, "quote"):
			blocks = append(blocks, "> "+text)
		case strings.Contains(st, "code") || strings.Contains(st, "html"):
			blocks = append(blocks, fence(strings.ReplaceAll(text, "\\\n", "\n"), ""))
		default:
			blocks = append(blocks, escapeLead(text))
		}
	}
	b.Roots = append(b.Roots, &Node{Title: title, Body: joinListAware(blocks)})
}

func joinListAware(blocks []string) string {
	var sb strings.Builder
	prevList := false
	for _, s := range blocks {
		list := isListLine(strings.TrimLeft(s, " "))
		if sb.Len() > 0 {
			if list && prevList {
				sb.WriteString("\n")
			} else {
				sb.WriteString("\n\n")
			}
		}
		sb.WriteString(s)
		prevList = list
	}
	return sb.String()
}

func docxInline(runs []docxRun) string {
	var sb strings.Builder
	for _, r := range runs {
		if r.image != "" {
			sb.WriteString("![](" + r.image + ")")
			continue
		}
		t := escapeText(r.text)
		t = strings.ReplaceAll(t, "\n", "\\\n")
		if r.bold {
			t = wrap(t, "**", "**")
		}
		if r.italic {
			t = wrap(t, "*", "*")
		}
		if r.strike {
			t = wrap(t, "~~", "~~")
		}
		if r.under && r.link == "" {
			t = wrap(t, "<u>", "</u>")
		}
		if r.link != "" && strings.TrimSpace(t) != "" {
			t = "[" + t + "](" + r.link + ")"
		}
		sb.WriteString(t)
	}
	return strings.TrimSpace(sb.String())
}

func mdTable(rows [][]string) string {
	width := 0
	for _, r := range rows {
		width = max(width, len(r))
	}
	if width == 0 {
		return ""
	}
	var sb strings.Builder
	line := func(r []string) {
		sb.WriteString("|")
		for i := 0; i < width; i++ {
			v := ""
			if i < len(r) {
				v = strings.ReplaceAll(oneLine(r[i]), "|", "\\|")
			}
			sb.WriteString(" " + v + " |")
		}
		sb.WriteString("\n")
	}
	line(rows[0])
	sb.WriteString("|" + strings.Repeat(" --- |", width) + "\n")
	for _, r := range rows[1:] {
		line(r)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func xmlAttr(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func parseDOCX(data []byte, rels map[string]string, image func(string) string) []docxPara {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out []docxPara
	var cur *docxPara
	var run docxRun
	inRun := false
	link := ""
	tableDepth := 0
	var table [][]string
	var row []string
	var cell strings.Builder
	inText := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tbl":
				tableDepth++
				if tableDepth == 1 {
					table = nil
				}
			case "tr":
				if tableDepth == 1 {
					row = nil
				}
			case "tc":
				if tableDepth == 1 {
					cell.Reset()
				}
			case "p":
				if tableDepth == 0 {
					out = append(out, docxPara{})
					cur = &out[len(out)-1]
				} else if cell.Len() > 0 {
					cell.WriteString(" ")
				}
			case "pStyle":
				if cur != nil && tableDepth == 0 {
					cur.style = xmlAttr(t, "val")
				}
			case "ilvl":
				if cur != nil && tableDepth == 0 {
					cur.level, _ = strconv.Atoi(xmlAttr(t, "val"))
				}
			case "numId":
				if cur != nil && tableDepth == 0 {
					cur.numID = xmlAttr(t, "val")
				}
			case "hyperlink":
				if id := xmlAttr(t, "id"); id != "" {
					if target, ok := rels[id]; ok && (strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:")) {
						link = target
					}
				}
			case "r":
				inRun = true
				run = docxRun{link: link}
			case "b":
				if inRun && xmlAttr(t, "val") != "0" && xmlAttr(t, "val") != "false" {
					run.bold = true
				}
			case "i":
				if inRun && xmlAttr(t, "val") != "0" && xmlAttr(t, "val") != "false" {
					run.italic = true
				}
			case "strike", "dstrike":
				if inRun && xmlAttr(t, "val") != "0" && xmlAttr(t, "val") != "false" {
					run.strike = true
				}
			case "u":
				if inRun && xmlAttr(t, "val") != "none" {
					run.under = true
				}
			case "t":
				inText = true
			case "tab":
				if tableDepth > 0 {
					cell.WriteString(" ")
				} else if inRun {
					run.text += " "
				}
			case "br", "cr":
				if tableDepth > 0 {
					cell.WriteString(" ")
				} else if inRun {
					run.text += "\n"
				}
			case "blip":
				if rid := xmlAttr(t, "embed"); rid != "" && tableDepth == 0 && cur != nil {
					if dest := image(rid); dest != "" {
						cur.runs = append(cur.runs, docxRun{image: dest})
					}
				}
			}
		case xml.CharData:
			if inText {
				if tableDepth > 0 {
					cell.Write(t)
				} else if inRun {
					run.text += string(t)
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "r":
				if inRun && cur != nil && tableDepth == 0 && run.text != "" {
					cur.runs = append(cur.runs, run)
				}
				inRun = false
			case "hyperlink":
				link = ""
			case "tc":
				if tableDepth == 1 {
					row = append(row, strings.TrimSpace(cell.String()))
				}
			case "tr":
				if tableDepth == 1 {
					table = append(table, row)
				}
			case "tbl":
				tableDepth--
				if tableDepth == 0 && len(table) > 0 {
					out = append(out, docxPara{table: table})
					cur = nil
				}
			}
		}
	}
	return out
}

func docxNumbering(data []byte) map[string]bool {
	out := map[string]bool{}
	if data == nil {
		return out
	}
	var n struct {
		Abstract []struct {
			ID  string `xml:"abstractNumId,attr"`
			Lvl []struct {
				Ilvl   string `xml:"ilvl,attr"`
				NumFmt struct {
					Val string `xml:"val,attr"`
				} `xml:"numFmt"`
			} `xml:"lvl"`
		} `xml:"abstractNum"`
		Num []struct {
			ID       string `xml:"numId,attr"`
			Abstract struct {
				Val string `xml:"val,attr"`
			} `xml:"abstractNumId"`
		} `xml:"num"`
	}
	_ = xml.Unmarshal(data, &n)
	abs := map[string]map[string]bool{}
	for _, a := range n.Abstract {
		m := map[string]bool{}
		for _, l := range a.Lvl {
			m[l.Ilvl] = l.NumFmt.Val != "bullet" && l.NumFmt.Val != "none" && l.NumFmt.Val != ""
		}
		abs[a.ID] = m
	}
	for _, x := range n.Num {
		for lvl, ordered := range abs[x.Abstract.Val] {
			out[x.ID+":"+lvl] = ordered
		}
	}
	return out
}

func docxHeadingStyles(data []byte) map[string]int {
	out := map[string]int{}
	if data == nil {
		return out
	}
	var s struct {
		Style []struct {
			ID   string `xml:"styleId,attr"`
			Name struct {
				Val string `xml:"val,attr"`
			} `xml:"name"`
			Outline struct {
				Val string `xml:"val,attr"`
			} `xml:"pPr>outlineLvl"`
		} `xml:"style"`
	}
	_ = xml.Unmarshal(data, &s)
	for _, st := range s.Style {
		name := strings.ToLower(strings.ReplaceAll(st.Name.Val, " ", ""))
		if lvl, err := strconv.Atoi(strings.TrimPrefix(name, "heading")); err == nil && strings.HasPrefix(name, "heading") {
			out[strings.ToLower(st.ID)] = lvl
			continue
		}
		if st.Outline.Val != "" {
			if lvl, err := strconv.Atoi(st.Outline.Val); err == nil && lvl < 6 {
				out[strings.ToLower(st.ID)] = lvl + 1
			}
		}
	}
	return out
}

type xlsxSheet struct {
	name string
	rows [][]string
}

var cellRefRE = regexp.MustCompile(`^([A-Z]+)(\d+)$`)

func colIndex(ref string) int {
	m := cellRefRE.FindStringSubmatch(ref)
	if m == nil {
		return -1
	}
	n := 0
	for _, r := range m[1] {
		n = n*26 + int(r-'A'+1)
	}
	return n - 1
}

func importXLSX(name string, data []byte, b *Bundle) {
	files, err := zipFiles(data)
	if err != nil {
		b.Warn("The Excel file " + name + " could not be read")
		return
	}
	shared := xlsxShared(readZipFile(files, "xl/sharedStrings.xml"))
	dates := xlsxDateStyles(readZipFile(files, "xl/styles.xml"))
	rels := relsOf(files, "xl/_rels/workbook.xml.rels")
	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	_ = xml.Unmarshal(readZipFile(files, "xl/workbook.xml"), &wb)
	var sheets []xlsxSheet
	for _, s := range wb.Sheets {
		target := rels[s.RID]
		if target == "" {
			continue
		}
		full := cleanPath(path.Join("xl", target))
		if strings.HasPrefix(target, "/") {
			full = cleanPath(target)
		}
		rows := xlsxRows(readZipFile(files, full), shared, dates)
		if len(rows) > 0 {
			sheets = append(sheets, xlsxSheet{name: s.Name, rows: rows})
		}
	}
	if len(sheets) == 0 {
		b.Warn("No sheets with data were found in " + name)
		return
	}
	makeTable := func(title string, rows [][]string) *Node {
		width := 0
		for _, r := range rows {
			width = max(width, len(r))
		}
		headers := make([]string, width)
		copy(headers, rows[0])
		return &Node{Title: title, Table: &Table{Headers: headers, Rows: rows[1:]}}
	}
	if len(sheets) == 1 {
		b.Roots = append(b.Roots, makeTable(stripExt(name), sheets[0].rows))
		return
	}
	wrap := &Node{Title: stripExt(name)}
	for _, s := range sheets {
		wrap.Children = append(wrap.Children, makeTable(s.name, s.rows))
	}
	b.Roots = append(b.Roots, wrap)
}

func xlsxShared(data []byte) []string {
	if data == nil {
		return nil
	}
	var out []string
	dec := xml.NewDecoder(bytes.NewReader(data))
	var cur strings.Builder
	inSI, inT, inRPh := false, false, false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI = true
				cur.Reset()
			case "t":
				inT = true
			case "rPh":
				inRPh = true
			}
		case xml.CharData:
			if inSI && inT && !inRPh {
				cur.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "si":
				out = append(out, cur.String())
				inSI = false
			case "t":
				inT = false
			case "rPh":
				inRPh = false
			}
		}
	}
	return out
}

func xlsxDateStyles(data []byte) map[int]bool {
	out := map[int]bool{}
	if data == nil {
		return out
	}
	var st struct {
		NumFmts []struct {
			ID   int    `xml:"numFmtId,attr"`
			Code string `xml:"formatCode,attr"`
		} `xml:"numFmts>numFmt"`
		Xfs []struct {
			NumFmt int `xml:"numFmtId,attr"`
		} `xml:"cellXfs>xf"`
	}
	_ = xml.Unmarshal(data, &st)
	custom := map[int]bool{}
	for _, f := range st.NumFmts {
		code := strings.ToLower(regexp.MustCompile(`"[^"]*"|\[[^\]]*\]`).ReplaceAllString(f.Code, ""))
		if strings.ContainsAny(code, "dy") || (strings.Contains(code, "m") && strings.ContainsAny(code, "dy")) {
			custom[f.ID] = true
		}
	}
	for i, x := range st.Xfs {
		id := x.NumFmt
		if (id >= 14 && id <= 22) || (id >= 45 && id <= 47) || custom[id] {
			out[i] = true
		}
	}
	return out
}

func xlsxRows(data []byte, shared []string, dates map[int]bool) [][]string {
	if data == nil {
		return nil
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	var rows [][]string
	var row []string
	var cellType, ref string
	var style int
	var val strings.Builder
	inV, inT := false, false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				row = nil
			case "c":
				cellType, ref = xmlAttr(t, "t"), xmlAttr(t, "r")
				style, _ = strconv.Atoi(xmlAttr(t, "s"))
				val.Reset()
			case "v":
				inV = true
			case "t":
				inT = true
			}
		case xml.CharData:
			if inV || inT {
				val.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inV = false
			case "t":
				inT = false
			case "c":
				v := val.String()
				switch cellType {
				case "s":
					if i, err := strconv.Atoi(v); err == nil && i >= 0 && i < len(shared) {
						v = shared[i]
					}
				case "b":
					if v == "1" {
						v = "Yes"
					} else {
						v = "No"
					}
				case "str", "inlineStr", "e":
				default:
					if dates[style] {
						if f, err := strconv.ParseFloat(v, 64); err == nil {
							v = excelDate(f)
						}
					}
				}
				idx := colIndex(ref)
				if idx < 0 {
					idx = len(row)
				}
				if idx > 500 {
					continue
				}
				for len(row) < idx {
					row = append(row, "")
				}
				row = append(row, v)
			case "row":
				empty := true
				for _, c := range row {
					if strings.TrimSpace(c) != "" {
						empty = false
						break
					}
				}
				if !empty {
					rows = append(rows, row)
				}
				if len(rows) > 100000 {
					return rows
				}
			}
		}
	}
	return rows
}

func excelDate(serial float64) string {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	days := math.Floor(serial)
	frac := serial - days
	t := base.AddDate(0, 0, int(days)).Add(time.Duration(math.Round(frac*86400)) * time.Second)
	if frac < 1e-9 {
		return t.Format("2006-01-02")
	}
	return t.Format("2006-01-02 15:04")
}
