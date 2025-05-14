package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func NewResultsPage(state *AppState) (fyne.CanvasObject, func()) {
	exactResults := binding.NewUntypedList()
	approxResults := binding.NewUntypedList()

	saveBtn := widget.NewButton("Сохранить в CSV", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}
			if uri == nil {
				return
			}

			dir := uri.Path()

			save := func(filename string, results binding.UntypedList) error {
				path := filepath.Join(dir, filename)
				file, err := os.Create(path)
				if err != nil {
					return err
				}
				defer file.Close()

				writer := csv.NewWriter(file)
				defer writer.Flush()

				writer.Write([]string{"ID", "Время (нс)", "F1-фактор", "Мощность"})

				length := results.Length()
				for i := range length {
					item, _ := results.GetValue(i)
					res := item.(*Result)

					record := []string{
						strconv.Itoa(res.RunId),
						strconv.FormatInt(res.Time, 10),
						fmt.Sprintf("%.2f", res.F1Factor),
						strconv.Itoa(len(res.Result)),
					}
					writer.Write(record)
				}
				return nil
			}

			if err := save(fmt.Sprintf("%s_exact_results.csv", time.Now().Format("2006-01-02T15-04-05")), exactResults); err != nil {
				dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}
			if err := save(fmt.Sprintf("%s_approx_results.csv", time.Now().Format("2006-01-02T15-04-05")), approxResults); err != nil {
				dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				return
			}
		}, fyne.CurrentApp().Driver().AllWindows()[0])
	})

	exactHeader := widget.NewLabel("Метод Магу")
	exactHeader.Alignment = fyne.TextAlignCenter
	exactHeader.TextStyle = fyne.TextStyle{Bold: true}

	approxHeader := widget.NewLabel("Жадный поиск")
	approxHeader.Alignment = fyne.TextAlignCenter
	approxHeader.TextStyle = fyne.TextStyle{Bold: true}

	left := container.NewBorder(
		exactHeader,
		nil, nil, nil,
		buildVirtualResultsList(exactResults),
	)

	right := container.NewBorder(
		approxHeader,
		nil, nil, nil,
		buildVirtualResultsList(approxResults),
	)

	split := container.NewHSplit(left, right)
	split.Offset = 0.5

	initFunc := func() {
		state.NavigationState.NextButton.Enable()
		state.NavigationState.BackButton.Enable()

		exactResults.Set([]any{})
		approxResults.Set([]any{})

		for _, res := range state.Results {
			method := strings.TrimSpace(res.Method)
			switch method {
			case string(MaghoutMethod):
				exactResults.Append(res)
			case string(GreedySearchMethod):
				approxResults.Append(res)
			}
		}
	}

	return container.NewBorder(nil, saveBtn, nil, nil, split), initFunc
}

func buildVirtualResultsList(results binding.UntypedList) fyne.CanvasObject {
	headers := container.NewGridWithColumns(5,
		container.NewCenter(widget.NewLabel("ID")),
		container.NewCenter(widget.NewLabel("Время (нс)")),
		container.NewCenter(widget.NewLabel("F1-фактор")),
		container.NewCenter(widget.NewLabel("Мощность")),
	)

	list := widget.NewList(
		func() int {
			return results.Length()
		},
		func() fyne.CanvasObject {
			return container.NewGridWithColumns(5,
				container.NewCenter(widget.NewLabel("")),
				container.NewCenter(widget.NewLabel("")),
				container.NewCenter(widget.NewLabel("")),
				container.NewCenter(widget.NewLabel("")),
				container.NewCenter(widget.NewButton("Сохранить", nil)),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			val, _ := results.GetValue(id)
			res := val.(*Result)

			row, _ := item.(*fyne.Container)
			if row == nil || len(row.Objects) < 5 {
				return
			}

			row.Objects[0].(*fyne.Container).Objects[0] = widget.NewLabel(strconv.Itoa(res.RunId))
			row.Objects[1].(*fyne.Container).Objects[0] = widget.NewLabel(strconv.FormatInt(res.Time, 10))
			row.Objects[2].(*fyne.Container).Objects[0] = widget.NewLabel(fmt.Sprintf("%.2f", res.F1Factor))
			row.Objects[3].(*fyne.Container).Objects[0] = widget.NewLabel(strconv.Itoa(len(res.Result)))

			btn := row.Objects[4].(*fyne.Container).Objects[0].(*widget.Button)
			btn.OnTapped = func(r *Result) func() {
				return func() {
					dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
						if err != nil || uri == nil {
							if err != nil {
								dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
							}
							return
						}
						dir := uri.Path()

						var accuracyPrefix string
						if res.Method == "Метод Магу" {
							accuracyPrefix = "exact"
						} else {
							accuracyPrefix = "approx"
						}
						filenameBase := fmt.Sprintf("%s_result_run%d", accuracyPrefix, r.RunId)

						dotPath := filepath.Join(dir, filenameBase+".dot")
						// pngPath := filepath.Join(dir, filenameBase+".png")

						dotContent := r.Graph.Dot(r.Result)
						os.WriteFile(dotPath, []byte(dotContent), 0644)

						// utils.SaveCanvasToFile(graphCanvas, pngPath)

						dialog.ShowInformation("Успех", "Файлы успешно сохранены!", fyne.CurrentApp().Driver().AllWindows()[0])
					}, fyne.CurrentApp().Driver().AllWindows()[0])
				}
			}(res)

			for _, obj := range row.Objects {
				obj.Refresh()
			}
		},
	)

	results.AddListener(binding.NewDataListener(func() {
		list.Refresh()
	}))

	return container.NewBorder(headers, nil, nil, nil, list)
}
