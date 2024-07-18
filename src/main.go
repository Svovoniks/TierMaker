package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"slices"
	"strings"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const tmp_file = "TierMaker.tmp"

type State struct {
	SortedNames                       []string
	Start, End, Mid, NamesIdx, ReqLen int
}

func (s State) flushSate() {
	js, err := json.Marshal(s)
	if err != nil {
		fmt.Println(err)
		return
	}

	file, err := os.OpenFile(tmp_file, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err)
	}

	defer file.Close()
	file.Write(js)
}

func clearState() {
	os.Remove(tmp_file)
}

func (s State) tmpIsInvalid() {
	os.Rename(tmp_file, "invalid_tmp_file.tmp")
}

func (s State) validate(names []string) bool {
	if s.Start > s.Mid {
		return false
	}

	if s.End < s.Mid {
		return false
	}

	if s.NamesIdx >= len(names) {
		return false
	}

	if len(s.SortedNames) >= s.ReqLen {
		return false
	}

	return true
}

func loadState() *State {
	file, err := os.ReadFile(tmp_file)
	if err != nil {
		return nil
	}

	var state State
	err = json.Unmarshal(file, &state)
	if err != nil {
		state.tmpIsInvalid()
		return nil
	}
	return &state
}

func main() {
	go func() {
		window := new(app.Window)
		window.Option((app.Title("TierMaker")))
		err := run(window)
		if err != nil {
			log.Print(err)
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

func chooseLayout(gtx layout.Context, th *material.Theme, name1 string, name2 string, left int, name1Button *widget.Clickable, name2Button *widget.Clickable) layout.Dimensions {
	// layout.Center.Layout(gtx, material.H3(th, "hello").Layout)
	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, material.H5(th, fmt.Sprintf("%d left", left)).Layout)
		}),
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
		log.Print("couldn't read the file")
		return nil
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Print("error")
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
		log.Print("couldn't save results")
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

func getState(names *[]string) *State {
	var state = loadState()

	if state == nil {
		state = &State{}
		if len(*names) < 2 {
			state.SortedNames = *names
		} else {
			state.SortedNames = append(state.SortedNames, (*names)[0])
			state.Start = 0
			state.End = 1
			state.Mid = 0
			state.ReqLen = len(*names)
			state.NamesIdx = 1
		}
		return state
	}

	if state.Mid != -1 {
		if state.validate(*names) {
			return state
		}
		state.tmpIsInvalid()
		log.Fatal("tmp file found but invalid")
	}

	var newNames []string
	slices.SortFunc(*names, func(i, j string) int {
		return strings.Compare(i, j)
	})

	var srtCopy = make([]string, len(state.SortedNames))
	copy(srtCopy, state.SortedNames)

	slices.SortFunc(srtCopy, func(i, j string) int {
		return strings.Compare(i, j)
	})
	var i, j = 0, 0

	for i < len(srtCopy) && j < len(*names) {
		if (*names)[j] == srtCopy[i] {
			j++
			i++
		} else if (*names)[j] > srtCopy[i] {
			i++
		} else {
			newNames = append(newNames, (*names)[j])
			j++
		}
	}
	if j < len(*names) {
		newNames = append(newNames, (*names)[j:]...)
	}
	state.ReqLen = len(*names)
	*names = newNames
	state.Start = 0
	state.End = len(srtCopy)
	state.Mid = len(srtCopy) / 2
	state.NamesIdx = 0
	fmt.Println(state.SortedNames, names, state.Mid, state.Start, state.End)

	return state
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	names := getNames()
	valid := true
	var state *State
	if names == nil {
		valid = false
	} else {
		state = getState(&names)
	}
	sorted := false
	fmt.Println(names)

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
				state.Start = state.Mid + 1
				if state.Start > state.End {
					state.Start = state.End
				}
				state.flushSate()
			}

			if name2Button.Clicked(gtx) || name2KeyPress {
				state.End = state.Mid
				if state.End < state.Start {
					state.End = state.Start
				}
				state.flushSate()
			}

			if state.Start == state.End && !sorted {
				if state.Start != -1 {
					state.SortedNames = insert(state.SortedNames, state.Start, names[state.NamesIdx])
					state.Start, state.End = 0, len(state.SortedNames)
					state.NamesIdx++
					state.flushSate()
				}
				if len(state.SortedNames) == state.ReqLen {
					sorted = true
					writeResults(state.SortedNames)
					clearState()
				}
				fmt.Println(state.SortedNames)
			}
			state.Mid = (state.Start + state.End) / 2

			if sorted {
				gtx := app.NewContext(&ops, e)
				label := material.H4(theme, "Everything is sorted\n\n Results have been stored in TierMakerResults.csv")
				label.Alignment = text.Middle
				layout.Center.Layout(gtx, label.Layout)
			} else {
				fmt.Println(state.NamesIdx, state.Mid, state.Start, state.End)
				chooseLayout(gtx, theme, state.SortedNames[state.Mid], names[state.NamesIdx], state.ReqLen-len(state.SortedNames), &name1Button, &name2Button)
			}
			e.Frame(gtx.Ops)
		}
	}
}
