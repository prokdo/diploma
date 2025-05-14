package utils

import (
	"fmt"
	"image/color"
	"io"
	"math"
	"reflect"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"graphmis/graph"
)

func RenderLineChart(title string, x []float64, y []float64, w io.Writer) error {
	if len(x) != len(y) {
		return fmt.Errorf("x and y slices must have same length")
	}

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = ""
	p.Y.Label.Text = ""

	pts := make(plotter.XYs, len(x))
	for i := range x {
		pts[i].X = x[i]
		pts[i].Y = y[i]
	}

	line, err := plotter.NewLine(pts)
	if err != nil {
		return fmt.Errorf("creating line: %w", err)
	}
	line.Color = color.RGBA{R: 0, G: 128, B: 255, A: 255}
	line.Width = vg.Points(2)

	p.Add(line)

	canvas := vgimg.New(6*vg.Inch, 4*vg.Inch)
	dc := draw.New(canvas)
	p.Draw(dc)
	png := vgimg.PngCanvas{Canvas: canvas}
	if _, err := png.WriteTo(w); err != nil {
		return fmt.Errorf("writing line chart PNG: %w", err)
	}
	return nil
}

func RenderBarChart(title string, values []float64, labels []string, w io.Writer) error {
	if len(values) != len(labels) {
		return fmt.Errorf("values and labels must have same length")
	}

	p := plot.New()
	p.Title.Text = title
	p.Y.Label.Text = ""

	bars := make(plotter.Values, len(values))
	for i, v := range values {
		bars[i] = v
	}

	barWidth := vg.Points(20)
	bchart, err := plotter.NewBarChart(bars, barWidth)
	if err != nil {
		return fmt.Errorf("creating bar chart: %w", err)
	}
	bchart.LineStyle.Width = vg.Length(0)
	bchart.Color = color.RGBA{R: 255, G: 127, B: 0, A: 255}

	p.Add(bchart)
	p.NominalX(labels...)

	canvas := vgimg.New(6*vg.Inch, 4*vg.Inch)
	dc := draw.New(canvas)
	p.Draw(dc)
	png := vgimg.PngCanvas{Canvas: canvas}
	if _, err := png.WriteTo(w); err != nil {
		return fmt.Errorf("writing bar chart PNG: %w", err)
	}
	return nil
}

func RenderHistogram(title string, data plotter.Values, bins int, w io.Writer) error {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = ""
	p.Y.Label.Text = ""

	hist, err := plotter.NewHist(data, bins)
	if err != nil {
		return fmt.Errorf("creating histogram: %w", err)
	}
	hist.Color = color.RGBA{B: 200, A: 200}

	p.Add(hist)

	canvas := vgimg.New(6*vg.Inch, 4*vg.Inch)
	dc := draw.New(canvas)
	p.Draw(dc)
	png := vgimg.PngCanvas{Canvas: canvas}
	if _, err := png.WriteTo(w); err != nil {
		return fmt.Errorf("writing histogram PNG: %w", err)
	}
	return nil
}

func RenderGraphToFyneContainer[T comparable](g graph.Graph[T], size fyne.Size, verticesToColor ...[]T) fyne.CanvasObject {
	const (
		circleR = 25
	)

	content := container.NewWithoutLayout()

	width := size.Width
	height := size.Height
	radius := min(width, height) / 2.5

	centerX, centerY := float32(width)/2, float32(height)/2

	vertices := g.GetAllVertices()
	n := len(vertices)
	if n == 0 {
		return content
	}

	posMap := make(map[T][2]float32)
	for i, v := range vertices {
		angle := 2 * math.Pi * float64(i) / float64(len(vertices))
		x := centerX + radius*float32(math.Cos(angle))
		y := centerY + radius*float32(math.Sin(angle))
		posMap[v] = [2]float32{x, y}
	}

	colors := map[T]color.Color{}
	if len(verticesToColor) > 0 && len(verticesToColor[0]) > 0 {
		for _, v := range verticesToColor[0] {
			colors[v] = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
		}
	}

	lineColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	seenEdges := make(map[string]bool)

	for _, v := range vertices {
		edges := g.GetEdgesOf(&v)

		fromPos := posMap[v]
		x1, y1 := fromPos[0], fromPos[1]

		for _, e := range edges {
			u := *e.To

			if g.Type() == graph.Undirected {
				key := edgeKey(v, u)
				if seenEdges[key] {
					continue
				}
				seenEdges[key] = true
			}

			toPos, exists := posMap[u]
			if !exists {
				continue
			}
			x2, y2 := toPos[0], toPos[1]

			fromX, fromY := pointOnCircle(x1, y1, x2, y2, circleR)
			toX, toY := pointOnCircle(x2, y2, x1, y1, circleR)

			line := canvas.NewLine(lineColor)
			line.StrokeWidth = 2
			line.Position1 = fyne.NewPos(fromX, fromY)
			line.Position2 = fyne.NewPos(toX, toY)
			content.Add(line)

			if g.Type() == graph.Directed {
				drawArrowhead(content, fromX, fromY, toX, toY, lineColor)
			}
		}
	}

	for _, v := range vertices {
		x, y := posMap[v][0], posMap[v][1]
		col, ok := colors[v]
		if !ok {
			col = color.Transparent
		}

		circle := canvas.NewCircle(col)
		circle.StrokeColor = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		circle.StrokeWidth = 2
		circle.Resize(fyne.NewSize(circleR*2, circleR*2))
		circle.Move(fyne.NewPos(x-circleR, y-circleR))

		text := canvas.NewText(toString(v), color.Black)
		text.TextStyle.Bold = true
		text.Resize(text.MinSize())

		textWidth := text.Size().Width
		textHeight := text.Size().Height

		text.Move(fyne.NewPos(
			x-circleR+circleR-textWidth/2,
			y-circleR+circleR-textHeight/2,
		))

		content.Add(circle)
		content.Add(text)
	}

	content.Resize(fyne.NewSize(width, height))
	return content
}

func drawArrowhead(cont *fyne.Container, x1, y1, x2, y2 float32, c color.Color) {
	headLength := float32(10)
	angle := math.Atan2(float64(y2-y1), float64(x2-x1))

	xLeft := x2 - headLength*float32(math.Cos(angle-math.Pi/6))
	yLeft := y2 - headLength*float32(math.Sin(angle-math.Pi/6))
	xRight := x2 - headLength*float32(math.Cos(angle+math.Pi/6))
	yRight := y2 - headLength*float32(math.Sin(angle+math.Pi/6))

	l1 := canvas.NewLine(c)
	l1.StrokeWidth = 2
	l1.Position1 = fyne.NewPos(x2, y2)
	l1.Position2 = fyne.NewPos(xLeft, yLeft)
	cont.Add(l1)

	l2 := canvas.NewLine(c)
	l2.StrokeWidth = 2
	l2.Position1 = fyne.NewPos(x2, y2)
	l2.Position2 = fyne.NewPos(xRight, yRight)
	cont.Add(l2)
}

func pointOnCircle(x1, y1, x2, y2, r float32) (float32, float32) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	lenVec := math.Sqrt(dx*dx + dy*dy)
	if lenVec == 0 {
		return x1, y1
	}
	ux := dx / lenVec
	uy := dy / lenVec
	return x1 + float32(ux)*r, y1 + float32(uy)*r
}

func toString[T comparable](v T) string {
	return reflect.ValueOf(v).String()
}

func edgeKey[T comparable](a, b T) string {
	strA := fmt.Sprintf("%v", a)
	strB := fmt.Sprintf("%v", b)

	if strA < strB {
		return strA + "-" + strB
	}
	return strB + "-" + strA
}
