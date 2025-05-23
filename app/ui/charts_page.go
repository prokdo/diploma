package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func NewChartsPage(state *AppState) (fyne.CanvasObject, func()) {
	var updateFuncs []func()

	saveImage := func(path string, img image.Image) error {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		return png.Encode(file, img)
	}

	buildPlot := func(title, yLabel string, exactData, approxData plotter.XYs) image.Image {
		p := plot.New()
		p.Title.Text = title
		p.X.Label.Text = "ID запуска"
		p.Y.Label.Text = yLabel
		p.BackgroundColor = color.White

		if len(state.Results) > 2 {
			exact, _ := plotter.NewLine(exactData)
			exact.LineStyle.Dashes = []vg.Length{vg.Points(3), vg.Points(3)}
			exact.Color = color.RGBA{0, 0, 255, 255}
			exact.Width = vg.Points(1)

			approx, _ := plotter.NewLine(approxData)
			approx.LineStyle.Dashes = []vg.Length{vg.Points(3), vg.Points(1)}
			approx.Color = color.RGBA{255, 0, 0, 255}
			approx.Width = vg.Points(1)

			p.Add(exact, approx)
			p.Legend.Add("Метод Магу", exact)
			p.Legend.Add("Жадный поиск", approx)
		} else {
			exact, _ := plotter.NewScatter(exactData)
			exact.Shape = draw.RingGlyph{}
			exact.Color = color.RGBA{0, 0, 255, 255}

			approx, _ := plotter.NewScatter(approxData)
			approx.Shape = draw.CrossGlyph{}
			approx.Color = color.RGBA{255, 0, 0, 255}

			p.Add(exact, approx)
			p.Legend.Add("Метод Магу", exact)
			p.Legend.Add("Жадный поиск", approx)
		}

		buf := new(bytes.Buffer)
		writerTo, _ := p.WriterTo(8*vg.Inch, 6*vg.Inch, "png")
		writerTo.WriteTo(buf)

		img, _ := png.Decode(buf)
		return img
	}

	buildChartTab := func(title, yLabel string, getData func() (plotter.XYs, plotter.XYs), filenamePrefix string) fyne.CanvasObject {
		img := canvas.NewImageFromImage(nil)
		img.FillMode = canvas.ImageFillContain

		saveBtn := widget.NewButton("Сохранить график", func() {
			dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
				if err != nil {
					dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
					return
				}
				if uri == nil {
					return
				}
				fullPath := fmt.Sprintf("%s/%s_%s.png", uri.Path(), time.Now().Format("2006-01-02T15-04-05"), filenamePrefix)
				if err := saveImage(fullPath, img.Image); err != nil {
					dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				}
			}, fyne.CurrentApp().Driver().AllWindows()[0])
		})

		updateFuncs = append(updateFuncs, func() {
			exact, approx := getData()
			img.Image = buildPlot(title, yLabel, exact, approx)
			img.Refresh()
		})

		return container.NewBorder(nil, saveBtn, nil, nil, img)
	}

	getTimeData := func() (plotter.XYs, plotter.XYs) {
		var exact, approx plotter.XYs
		for _, res := range state.Results {
			x := float64(res.RunId)
			y := float64(res.Time)
			switch res.Method {
			case string(MaghoutMethod):
				exact = append(exact, plotter.XY{X: x, Y: y})
			case string(GreedySearchMethod):
				approx = append(approx, plotter.XY{X: x, Y: y})
			}
		}
		return exact, approx
	}

	getF1Data := func() (plotter.XYs, plotter.XYs) {
		var exact, approx plotter.XYs
		for _, res := range state.Results {
			x := float64(res.RunId)
			y := res.F1Factor
			switch res.Method {
			case string(MaghoutMethod):
				exact = append(exact, plotter.XY{X: x, Y: y})
			case string(GreedySearchMethod):
				approx = append(approx, plotter.XY{X: x, Y: y})
			}
		}
		return exact, approx
	}

	getCardinalityData := func() (plotter.XYs, plotter.XYs) {
		var exact, approx plotter.XYs
		for _, res := range state.Results {
			x := float64(res.RunId)
			y := float64(len(res.Result))
			switch res.Method {
			case string(MaghoutMethod):
				exact = append(exact, plotter.XY{X: x, Y: y})
			case string(GreedySearchMethod):
				approx = append(approx, plotter.XY{X: x, Y: y})
			}
		}
		return exact, approx
	}

	now := time.Now().Format("2006-01-02T15-04-05")
	tabs := container.NewAppTabs(
		container.NewTabItem("Время выполнения", buildChartTab("Время выполнения", "Время (нс)", getTimeData, now+"_time")),
		container.NewTabItem("F1-score", buildChartTab("F1-score", "F1", getF1Data, now+"_f1")),
		container.NewTabItem("Мощность решений", buildChartTab("Мощность решений", "Размер множества", getCardinalityData, now+"_cardinality")),
	)

	initFunc := func() {
		for _, f := range updateFuncs {
			f()
		}
		state.NavigationState.NextButton.Disable()
		state.NavigationState.BackButton.Enable()
	}

	return tabs, initFunc
}
