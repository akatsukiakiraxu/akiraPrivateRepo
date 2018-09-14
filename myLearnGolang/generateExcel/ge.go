package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/tealeg/xlsx"
	"io/ioutil"
	"os/exec"
	"sort"
	"strconv"
)

type Condom struct {
	Index   int
	Name    string
	Price   int
	checked bool
}

type CondomModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*Condom
	titles     []string
}

func (m *CondomModel) RowCount() int {
	return len(m.items)
}

func (m *CondomModel) Value(row, col int) interface{} {
	item := m.items[row]

	switch col {
	case 0:
		return item.Index
	case 1:
		return item.Name
	case 2:
		return item.Price
	}
	panic("unexpected col")
}

func (m *CondomModel) Checked(row int) bool {
	return m.items[row].checked
}

func (m *CondomModel) SetChecked(row int, checked bool) error {
	m.items[row].checked = checked
	return nil
}

func (m *CondomModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	sort.Stable(m)

	return m.SorterBase.Sort(col, order)
}

func (m *CondomModel) Len() int {
	return len(m.items)
}

func (m *CondomModel) Less(i, j int) bool {
	a, b := m.items[i], m.items[j]

	c := func(ls bool) bool {
		if m.sortOrder == walk.SortAscending {
			return ls
		}

		return !ls
	}

	switch m.sortColumn {
	case 0:
		return c(a.Index < b.Index)
	case 1:
		return c(a.Name < b.Name)
	case 2:
		return c(a.Price < b.Price)
	}

	panic("unreachable")
}

func (m *CondomModel) Swap(i, j int) {
	m.items[i], m.items[j] = m.items[j], m.items[i]
}

func NewCondomModel() *CondomModel {
	m := new(CondomModel)
	m.items = make([]*Condom, 3)

	m.items[0] = &Condom{
		Index: 0,
		Name:  "Item1",
		Price: 20,
	}

	m.items[1] = &Condom{
		Index: 1,
		Name:  "Item2",
		Price: 18,
	}

	m.items[2] = &Condom{
		Index: 2,
		Name:  "Item3",
		Price: 19,
	}
	m.titles = append(m.titles, "番号")
	m.titles = append(m.titles, "名称")
	m.titles = append(m.titles, "単価")
	return m
}

type CondomMainWindow struct {
	*walk.MainWindow
	model      *CondomModel
	tv         *walk.TableView
	rows       *walk.Label
	totalPrice *walk.Label
}

func main() {
	mw := &CondomMainWindow{model: NewCondomModel()}
	mw.rows.SetText(strconv.Itoa(mw.model.RowCount()))
	mw.totalPrice.SetText(strconv.Itoa(mw.getTotalPrice()))
	MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "Aotech",
		Size:     Size{800, 600},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Add",
						OnClicked: func() {
							mw.model.items = append(mw.model.items, &Condom{
								Index: mw.model.Len() + 1,
								Name:  "new",
								Price: 0,
							})
							mw.model.PublishRowsReset()
							mw.tv.SetSelectedIndexes([]int{})
							mw.rows.Text() = strconv.Itoa(mw.model.RowCount())
							mw.totalPrice.Text() = strconv.Itoa(mw.getTotalPrice())
						},
					},
					PushButton{
						Text: "Delete",
						OnClicked: func() {
							items := []*Condom{}
							remove := mw.tv.SelectedIndexes()
							for i, x := range mw.model.items {
								remove_ok := false
								for _, j := range remove {
									if i == j {
										remove_ok = true
									}
								}
								if !remove_ok {
									items = append(items, x)
								}
							}
							mw.model.items = items
							mw.model.PublishRowsReset()
							mw.tv.SetSelectedIndexes([]int{})
							mw.rows.Text() = strconv.Itoa(mw.model.RowCount())
							mw.totalPrice.Text() = strconv.Itoa(mw.getTotalPrice())
						},
					},
					/*
						PushButton{
							Text: "ExecChecked",
							OnClicked: func() {
								for _, x := range mw.model.items {
									if x.checked {
										fmt.Printf("checked: %v\n", x)
									}
								}
								fmt.Println()
							},
						},
						PushButton{
							Text: "AddPriceChecked",
							OnClicked: func() {
								for i, x := range mw.model.items {
									if x.checked {
										x.Price++
										mw.model.PublishRowChanged(i)
									}
								}
							},
						},
					*/
					PushButton{
						Text:      "Export",
						OnClicked: mw.export2Excel,
					},
				},
			},
			Composite{
				Layout: VBox{},
				ContextMenuItems: []MenuItem{
					Action{
						Text:        "I&nfo",
						OnTriggered: mw.tv_ItemActivated,
					},
					Action{
						Text: "E&xit",
						OnTriggered: func() {
							mw.Close()
						},
					},
				},
				Children: []Widget{
					TableView{
						AssignTo: &mw.tv,
						//CheckBoxes:       true,
						ColumnsOrderable: true,
						MultiSelection:   true,
						Columns: []TableViewColumn{
							{Title: mw.model.titles[0]},
							{Title: mw.model.titles[1]},
							{Title: mw.model.titles[2]},
							//TODO:タイトル追加はここ
						},
						Model: mw.model,
						OnCurrentIndexChanged: func() {
							i := mw.tv.CurrentIndex()
							if 0 <= i {
								fmt.Printf("OnCurrentIndexChanged: %v\n", mw.model.items[i].Name)
							}
						},
						OnItemActivated: mw.tv_ItemActivated,
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "行数",
					},
					Label{
						AssignTo: &mw.rows,
					},
					Label{
						Text: "総額",
					},
					Label{
						AssignTo: &mw.totalPrice,
					},
					HSpacer{},
				},
			},
		},
	}.Run()
}

func (mw *CondomMainWindow) getTotalPrice() int {
	totalPrice := 0
	for _, item := range mw.model.items {
		totalPrice += item.Price
	}
	return totalPrice
}

func (mw *CondomMainWindow) tv_ItemActivated() {
	dlgdata := new(MyDialogData)
	for _, i := range mw.tv.SelectedIndexes() {
		dlgdata.name = mw.model.items[i].Name
		dlgdata.price = mw.model.items[i].Price
		cmd, err := RunMyDialog(mw, dlgdata)
		if err != nil {
			fmt.Println(err)
		} else if cmd == walk.DlgCmdCancel { // Canceボタンクリック
			return
		} else if cmd == walk.DlgCmdNone { // 右上xクリック
			return
		}
		mw.model.items[i].Name = dlgdata.name
		mw.model.items[i].Price = dlgdata.price
		break
	}
	//mw.edit.AppendText(fmt.Sprintf("Dialog String: %s\r\n", dlgdata.msg))
}

func (mw *CondomMainWindow) export2Excel() {
	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	var err error

	xlsx.SetDefaultFont(11, "ＭＳ Ｐゴシック")
	file = xlsx.NewFile()
	// create temp file. just generate filename here
	tempFile, err := ioutil.TempFile("", "exported.*.xlsx")
	filename := tempFile.Name()
	tempFile.Close()
	// add sheet1
	sheet, err = file.AddSheet("S1")
	if err != nil {
		fmt.Printf(err.Error())
	}
	// title row
	row = sheet.AddRow()
	//set title style
	titleStyle := xlsx.NewStyle()
	fill := *xlsx.NewFill("solid", "008EA9DB", "008EA9DB")
	border := *xlsx.NewBorder("thin", "thin", "thin", "thin")
	titleStyle.Alignment.Horizontal = "center"
	titleStyle.Alignment.Vertical = "center"
	titleStyle.Fill = fill
	titleStyle.Border = border
	// fill title row
	for _, title := range mw.model.titles {
		cell = row.AddCell()
		cell.SetStyle(titleStyle)
		cell.Value = title
	}
	// add items
	for _, line := range mw.model.items {
		row = sheet.AddRow()
		//set cell style
		cellStyle := xlsx.NewStyle()
		cellStyle.Alignment.Horizontal = "left"
		cellStyle.Alignment.Vertical = "left"
		cellStyle.Border = border
		// add index
		cell = row.AddCell()
		cell.SetStyle(cellStyle)
		cell.SetInt(line.Index)
		// add name
		cell = row.AddCell()
		cell.SetStyle(cellStyle)
		cell.SetString(line.Name)
		// add price
		cell = row.AddCell()
		cell.SetStyle(cellStyle)
		cell.SetInt(line.Price)
	}
	err = file.Save(filename)
	if err != nil {
		fmt.Printf(err.Error())
	}
	err = exec.Command("C:\\Program Files (x86)\\Microsoft Office\\root\\Office16\\Excel.exe", filename).Start()
}

/*********************************************
******************MyDialogData****************
**********************************************/

// MainWindowとDialog間でデータを渡す構造体
type MyDialogData struct {
	name  string
	price int
}

// Dialogで使用するウィジェットの実体
type MyDialogWindow struct {
	dlg      *walk.Dialog
	name     *walk.LineEdit
	price    *walk.LineEdit
	acceptPB *walk.PushButton
	cancelPB *walk.PushButton
}

func RunMyDialog(owner walk.Form, dlgdata *MyDialogData) (int, error) {

	mydlg := new(MyDialogWindow)
	MYDLG := Dialog{
		AssignTo:      &mydlg.dlg,
		Title:         "Dialog",
		DefaultButton: &mydlg.acceptPB,
		CancelButton:  &mydlg.cancelPB,
		MinSize:       Size{300, 100},
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "名称:",
					},
					LineEdit{
						Text:     dlgdata.name,
						AssignTo: &mydlg.name,
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "単価:",
					},
					LineEdit{
						Text:     strconv.Itoa(dlgdata.price),
						AssignTo: &mydlg.price,
					},
				},
			},

			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						AssignTo: &mydlg.acceptPB,
						Text:     "OK",
						OnClicked: func() {
							mydlg.acceptClicked(dlgdata)
						},
					},
					PushButton{
						AssignTo:  &mydlg.cancelPB,
						Text:      "Cancel",
						OnClicked: func() { mydlg.dlg.Cancel() },
					},
				},
			},
		},
	}

	return MYDLG.Run(owner)
}

// Dialog OKクリック時の処理
func (mydlg *MyDialogWindow) acceptClicked(dlgdata *MyDialogData) {
	dlgdata.name = mydlg.name.Text()
	dlgdata.price, _ = strconv.Atoi(mydlg.price.Text())
	mydlg.dlg.Accept()
}
