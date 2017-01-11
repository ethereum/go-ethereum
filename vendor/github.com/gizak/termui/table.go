package termui

import "strings"

/*
	table := termui.NewTable()
	table.Rows = rows
	table.FgColor = termui.ColorWhite
	table.BgColor = termui.ColorDefault
	table.Height = 7
	table.Width = 62
	table.Y = 0
	table.X = 0
	table.Border = true
*/

type Table struct {
	Block
	Rows      [][]string
	CellWidth []int
	FgColor   Attribute
	BgColor   Attribute
	FgColors  []Attribute
	BgColors  []Attribute
	Seperator bool
	TextAlign Align
}

func NewTable() *Table {
	table := &Table{Block: *NewBlock()}
	table.FgColor = ColorWhite
	table.BgColor = ColorDefault
	table.Seperator = true
	return table
}

func (table *Table) Analysis() {
	length := len(table.Rows)
	if length < 1 {
		return
	}

	if len(table.FgColors) == 0 {
		table.FgColors = make([]Attribute, len(table.Rows))
	}
	if len(table.BgColors) == 0 {
		table.BgColors = make([]Attribute, len(table.Rows))
	}

	row_width := len(table.Rows[0])
	cellWidthes := make([]int, row_width)

	for index, row := range table.Rows {
		for i, str := range row {
			if cellWidthes[i] < len(str) {
				cellWidthes[i] = len(str)
			}
		}

		if table.FgColors[index] == 0 {
			table.FgColors[index] = table.FgColor
		}

		if table.BgColors[index] == 0 {
			table.BgColors[index] = table.BgColor
		}
	}

	table.CellWidth = cellWidthes

	//width_sum := 2
	//for i, width := range cellWidthes {
	//	width_sum += (width + 2)
	//	for u, row := range table.Rows {
	//		switch table.TextAlign {
	//		case "right":
	//			row[i] = fmt.Sprintf(" %*s ", width, table.Rows[u][i])
	//		case "center":
	//			word_width := len(table.Rows[u][i])
	//			offset := (width - word_width) / 2
	//			row[i] = fmt.Sprintf(" %*s ", width, fmt.Sprintf("%-*s", offset+word_width, table.Rows[u][i]))
	//		default: // left
	//			row[i] = fmt.Sprintf(" %-*s ", width, table.Rows[u][i])
	//		}
	//	}
	//}

	//if table.Width == 0 {
	//	table.Width = width_sum
	//}
}

func (table *Table) SetSize() {
	length := len(table.Rows)
	if table.Seperator {
		table.Height = length*2 + 1
	} else {
		table.Height = length + 2
	}
	table.Width = 2
	if length != 0 {
		for _, cell_width := range table.CellWidth {
			table.Width += cell_width + 3
		}
	}
}

func (table *Table) CalculatePosition(x int, y int, x_coordinate *int, y_coordibate *int, cell_beginning *int) {
	if table.Seperator {
		*y_coordibate = table.innerArea.Min.Y + y*2
	} else {
		*y_coordibate = table.innerArea.Min.Y + y
	}
	if x == 0 {
		*cell_beginning = table.innerArea.Min.X
	} else {
		*cell_beginning += table.CellWidth[x-1] + 3
	}

	switch table.TextAlign {
	case AlignRight:
		*x_coordinate = *cell_beginning + (table.CellWidth[x] - len(table.Rows[y][x])) + 2
	case AlignCenter:
		*x_coordinate = *cell_beginning + (table.CellWidth[x]-len(table.Rows[y][x]))/2 + 2
	default:
		*x_coordinate = *cell_beginning + 2
	}
}

func (table *Table) Buffer() Buffer {
	buffer := table.Block.Buffer()
	table.Analysis()

	pointer_x := table.innerArea.Min.X + 2
	pointer_y := table.innerArea.Min.Y
	border_pointer_x := table.innerArea.Min.X
	for y, row := range table.Rows {
		for x, cell := range row {
			table.CalculatePosition(x, y, &pointer_x, &pointer_y, &border_pointer_x)
			backgraound := DefaultTxBuilder.Build(strings.Repeat(" ", table.CellWidth[x]+3), table.BgColors[y], table.BgColors[y])
			cells := DefaultTxBuilder.Build(cell, table.FgColors[y], table.BgColors[y])

			for i, back := range backgraound {
				buffer.Set(border_pointer_x+i, pointer_y, back)
			}

			coordinate_x := pointer_x
			for _, printer := range cells {
				buffer.Set(coordinate_x, pointer_y, printer)
				coordinate_x += printer.Width()
			}

			if x != 0 {
				devidors := DefaultTxBuilder.Build("|", table.FgColors[y], table.BgColors[y])
				for _, devidor := range devidors {
					buffer.Set(border_pointer_x, pointer_y, devidor)
				}
			}
		}

		if table.Seperator {
			border := DefaultTxBuilder.Build(strings.Repeat("─", table.Width-2), table.FgColor, table.BgColor)
			for i, cell := range border {
				buffer.Set(i+1, pointer_y+1, cell)
			}
		}
	}

	return buffer
}
