package excelize

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// NewSheet provice function to greate a new sheet by given index, when
// creating a new XLSX file, the default sheet will be create, when you
// create a new file, you need to ensure that the index is continuous.
func (f *File) NewSheet(index int, name string) {
	// Update docProps/app.xml
	f.setAppXML()
	// Update [Content_Types].xml
	f.setContentTypes(index)
	// Create new sheet /xl/worksheets/sheet%d.xml
	f.setSheet(index)
	// Update xl/_rels/workbook.xml.rels
	rid := f.addXlsxWorkbookRels(index)
	// Update xl/workbook.xml
	f.setWorkbook(name, rid)
}

// Read and update property of contents type of XLSX.
func (f *File) setContentTypes(index int) {
	var content xlsxTypes
	xml.Unmarshal([]byte(f.readXML(`[Content_Types].xml`)), &content)
	content.Overrides = append(content.Overrides, xlsxOverride{
		PartName:    `/xl/worksheets/sheet` + strconv.Itoa(index) + `.xml`,
		ContentType: `application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml`,
	})
	output, err := xml.Marshal(content)
	if err != nil {
		fmt.Println(err)
	}
	f.saveFileList(`[Content_Types].xml`, string(output))
}

// Update sheet property by given index.
func (f *File) setSheet(index int) {
	var xlsx xlsxWorksheet
	xlsx.Dimension.Ref = `A1`
	xlsx.SheetViews.SheetView = append(xlsx.SheetViews.SheetView, xlsxSheetView{
		WorkbookViewID: 0,
	})
	output, err := xml.Marshal(xlsx)
	if err != nil {
		fmt.Println(err)
	}
	path := `xl/worksheets/sheet` + strconv.Itoa(index) + `.xml`
	f.saveFileList(path, replaceWorkSheetsRelationshipsNameSpace(string(output)))
}

// Update workbook property of XLSX. Maximum 31 characters allowed in sheet title.
func (f *File) setWorkbook(name string, rid int) {
	var content xlsxWorkbook
	if len(name) > 31 {
		name = name[0:31]
	}
	xml.Unmarshal([]byte(f.readXML(`xl/workbook.xml`)), &content)
	content.Sheets.Sheet = append(content.Sheets.Sheet, xlsxSheet{
		Name:    name,
		SheetID: strconv.Itoa(rid),
		ID:      `rId` + strconv.Itoa(rid),
	})
	output, err := xml.Marshal(content)
	if err != nil {
		fmt.Println(err)
	}
	f.saveFileList(`xl/workbook.xml`, replaceRelationshipsNameSpace(string(output)))
}

// Read and unmarshal workbook relationships of XLSX.
func (f *File) readXlsxWorkbookRels() xlsxWorkbookRels {
	var content xlsxWorkbookRels
	xml.Unmarshal([]byte(f.readXML(`xl/_rels/workbook.xml.rels`)), &content)
	return content
}

// Update workbook relationships property of XLSX.
func (f *File) addXlsxWorkbookRels(sheet int) int {
	content := f.readXlsxWorkbookRels()
	rID := len(content.Relationships) + 1
	ID := bytes.Buffer{}
	ID.WriteString(`rId`)
	ID.WriteString(strconv.Itoa(rID))
	target := bytes.Buffer{}
	target.WriteString(`worksheets/sheet`)
	target.WriteString(strconv.Itoa(sheet))
	target.WriteString(`.xml`)
	content.Relationships = append(content.Relationships, xlsxWorkbookRelation{
		ID:     ID.String(),
		Target: target.String(),
		Type:   `http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet`,
	})
	output, err := xml.Marshal(content)
	if err != nil {
		fmt.Println(err)
	}
	f.saveFileList(`xl/_rels/workbook.xml.rels`, string(output))
	return rID
}

// Update docProps/app.xml file of XML.
func (f *File) setAppXML() {
	f.saveFileList(`docProps/app.xml`, templateDocpropsApp)
}

// Some tools that read XLSX files have very strict requirements about
// the structure of the input XML.  In particular both Numbers on the Mac
// and SAS dislike inline XML namespace declarations, or namespace
// prefixes that don't match the ones that Excel itself uses.  This is a
// problem because the Go XML library doesn't multiple namespace
// declarations in a single element of a document.  This function is a
// horrible hack to fix that after the XML marshalling is completed.
func replaceRelationshipsNameSpace(workbookMarshal string) string {
	oldXmlns := `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`
	newXmlns := `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" mc:Ignorable="x15" xmlns:x15="http://schemas.microsoft.com/office/spreadsheetml/2010/11/main">`
	return strings.Replace(workbookMarshal, oldXmlns, newXmlns, -1)
}

// SetActiveSheet provide function to set default active sheet of XLSX by given index.
func (f *File) SetActiveSheet(index int) {
	var content xlsxWorkbook
	if index < 1 {
		index = 1
	}
	index--
	xml.Unmarshal([]byte(f.readXML(`xl/workbook.xml`)), &content)
	if len(content.BookViews.WorkBookView) > 0 {
		content.BookViews.WorkBookView[0].ActiveTab = index
	} else {
		content.BookViews.WorkBookView = append(content.BookViews.WorkBookView, xlsxWorkBookView{
			ActiveTab: index,
		})
	}
	sheets := len(content.Sheets.Sheet)
	output, err := xml.Marshal(content)
	if err != nil {
		fmt.Println(err)
	}
	f.saveFileList(`xl/workbook.xml`, replaceRelationshipsNameSpace(string(output)))
	index++
	buffer := bytes.Buffer{}
	for i := 0; i < sheets; i++ {
		xlsx := xlsxWorksheet{}
		sheetIndex := i + 1
		buffer.WriteString(`xl/worksheets/sheet`)
		buffer.WriteString(strconv.Itoa(sheetIndex))
		buffer.WriteString(`.xml`)
		xml.Unmarshal([]byte(f.readXML(buffer.String())), &xlsx)
		if index == sheetIndex {
			if len(xlsx.SheetViews.SheetView) > 0 {
				xlsx.SheetViews.SheetView[0].TabSelected = true
			} else {
				xlsx.SheetViews.SheetView = append(xlsx.SheetViews.SheetView, xlsxSheetView{
					TabSelected: true,
				})
			}
		} else {
			if len(xlsx.SheetViews.SheetView) > 0 {
				xlsx.SheetViews.SheetView[0].TabSelected = false
			}
		}
		sheet, err := xml.Marshal(xlsx)
		if err != nil {
			fmt.Println(err)
		}
		f.saveFileList(buffer.String(), replaceWorkSheetsRelationshipsNameSpace(string(sheet)))
		buffer.Reset()
	}
	return
}

// GetActiveSheetIndex provide function to get active sheet of XLSX. If not found
// the active sheet will be return integer 0.
func (f *File) GetActiveSheetIndex() int {
	content := xlsxWorkbook{}
	buffer := bytes.Buffer{}
	xml.Unmarshal([]byte(f.readXML(`xl/workbook.xml`)), &content)
	for _, v := range content.Sheets.Sheet {
		xlsx := xlsxWorksheet{}
		buffer.WriteString(`xl/worksheets/sheet`)
		buffer.WriteString(strings.TrimPrefix(v.ID, `rId`))
		buffer.WriteString(`.xml`)
		xml.Unmarshal([]byte(f.readXML(buffer.String())), &xlsx)
		for _, sheetView := range xlsx.SheetViews.SheetView {
			if sheetView.TabSelected {
				id, _ := strconv.Atoi(strings.TrimPrefix(v.ID, `rId`))
				return id
			}
		}
		buffer.Reset()
	}
	return 0
}

// GetSheetName provide function to get sheet name of XLSX by given sheet index.
// If given sheet index is invalid, will return an empty string.
func (f *File) GetSheetName(index int) string {
	content := xlsxWorkbook{}
	xml.Unmarshal([]byte(f.readXML(`xl/workbook.xml`)), &content)
	for _, v := range content.Sheets.Sheet {
		if v.ID == `rId`+strconv.Itoa(index) {
			return v.Name
		}
	}
	return ``
}

// GetSheetMap provide function to get sheet map of XLSX. For example:
//
//    xlsx, err := excelize.OpenFile("/tmp/Workbook.xlsx")
//    if err != nil {
//        fmt.Println(err)
//        os.Exit(1)
//    }
//    for k, v := range xlsx.GetSheetMap()
//        fmt.Println(k, v)
//    }
//
func (f *File) GetSheetMap() map[int]string {
	content := xlsxWorkbook{}
	sheetMap := map[int]string{}
	xml.Unmarshal([]byte(f.readXML(`xl/workbook.xml`)), &content)
	for _, v := range content.Sheets.Sheet {
		id, _ := strconv.Atoi(strings.TrimPrefix(v.ID, `rId`))
		sheetMap[id] = v.Name
	}
	return sheetMap
}
