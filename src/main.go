package main

import (
	"bufio"
	"fmt"
	"image"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		window := new(app.Window)
		window.Option((app.Title("TierMaker")))
		err := run(window)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

type SplitVisual struct{}

func (s SplitVisual) splitLayout(gtx layout.Context, left, right layout.Widget) layout.Dimensions {
	leftsize := gtx.Constraints.Min.X / 2
	rightsize := gtx.Constraints.Min.X - leftsize

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(leftsize, gtx.Constraints.Max.Y))
		left(gtx)
	}

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(rightsize, gtx.Constraints.Max.Y))
		trans := op.Offset(image.Pt(leftsize, 0)).Push(gtx.Ops)
		right(gtx)
		trans.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func chooseLayout(gtx layout.Context, th *material.Theme, name1 string, name2 string, name1Button *widget.Clickable, name2Button *widget.Clickable) layout.Dimensions {
	// layout.Center.Layout(gtx, material.H3(th, "hello").Layout)
	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, material.H3(th, "Which one is better?").Layout)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return SplitVisual{}.splitLayout(gtx, func(gtx layout.Context) layout.Dimensions {
				return FillWithLabel(gtx, th, name1, name1Button, "J")
			}, func(gtx layout.Context) layout.Dimensions {
				return FillWithLabel(gtx, th, name2, name2Button, "K")
			})
		}),
	)
}

func FillWithLabel(gtx layout.Context, th *material.Theme, text string, button *widget.Clickable, shortcut string) layout.Dimensions {
	layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(
		gtx,
		layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				margins := layout.Inset{
					Top:    unit.Dp(25),
					Bottom: unit.Dp(25),
					Right:  unit.Dp(35),
					Left:   unit.Dp(35),
				}
				return margins.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, button, "This one ("+shortcut+")")
						return btn.Layout(gtx)
					})
			},
		),
	)
	return layout.Center.Layout(gtx, material.H4(th, text).Layout)
}

func getNames() []string {
	file, err := os.Open("titles.txt")
	if err != nil {
		log.Fatal("couldn't read the file")
		return nil
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal("error")
		}
	}()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var lines = make([]string, 0)

	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		lines = append(lines, text)
	}

	return lines
}

func writeResults(arr []string) {
	file, err := os.OpenFile("TierMakerResults.csv", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("couldn't save results")
	}
	defer file.Close()

	for _, i := range arr {
		file.WriteString(i + ",\n")
	}
}

func insert(a []string, index int, value string) []string {
	if len(a) == index {
		return append(a, value)
	}
	a = append(a[:index+1], a[index:]...)
	a[index] = value
	return a
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	names := getNames()
	valid := true
	if names == nil {
		valid = false
	}
	sorted := false
	var sortedNames []string
	var start, end, mid = -1, -1, -1
	var namesIdx int

	if len(names) < 2 {
		sortedNames = names
	} else {
		sortedNames = append(sortedNames, names[0])
		start = 0
		end = 1
		mid = 0
		namesIdx = 1
	}

	var name1Button widget.Clickable
	var name2Button widget.Clickable
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if !valid {
				label := material.H4(theme, "Couldn't find file titles.txt")
				label.Alignment = text.Middle
				layout.Center.Layout(gtx, label.Layout)
				e.Frame(gtx.Ops)
				break
			}
			name1KeyPress := false
			name2KeyPress := false

			for {
				ev, ok := gtx.Event(
					key.Filter{Name: "K"},
					key.Filter{Name: "J"},
				)
				if !ok {
					break
				}
				if ev.(key.Event).State == key.Press {
					name := ev.(key.Event).Name
					if name == "J" {
						name1KeyPress = true
					} else {
						name2KeyPress = true
					}
				}
			}

			if name1Button.Clicked(gtx) || name1KeyPress {
				start = mid + 1
				if start > end {
					start = end
				}
			}

			if name2Button.Clicked(gtx) || name2KeyPress {
				end = mid
				if end < start {
					end = start
				}
			}

			if start == end && !sorted {
				if start != -1 {
					sortedNames = insert(sortedNames, start, names[namesIdx])
					start, end = 0, len(sortedNames)
					namesIdx++
				}
				if len(sortedNames) == len(names) {
					sorted = true
					writeResults(sortedNames)
				}
				fmt.Println(sortedNames)
			}
			mid = (start + end) / 2

			if sorted {
				gtx := app.NewContext(&ops, e)
				label := material.H4(theme, "Everything is sorted\n\n Results have been stored in TierMakerResults.csv")
				label.Alignment = text.Middle
				layout.Center.Layout(gtx, label.Layout)
			} else {
				fmt.Println(namesIdx, mid, start, end)
				chooseLayout(gtx, theme, sortedNames[mid], names[namesIdx], &name1Button, &name2Button)
			}
			e.Frame(gtx.Ops)
		}
	}
}
