package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"strconv"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const tmp_file = "TierMaker.tmp"
const titles_file = "titles.txt"
const results_file = "TierMakerResults.csv"

type StateHistory struct {
	StateList []State
}

func getHistory() StateHistory {
	file, err := os.ReadFile(tmp_file)
	var history = StateHistory{}
	if err != nil {
		return history
	}

	json.Unmarshal(file, &history)
	return history
}

func (sh *StateHistory) flushHistory() {
	err_message := "couldn't save state history"
	file, err := os.Create(tmp_file)
	if err != nil {
		log.Fatal(err_message)
		return
	}

	defer file.Close()

	json, err := json.Marshal(sh)
	if err != nil {
		log.Fatal(err_message)
	}

	file.Write(json)
}

func (sh *StateHistory) addState(state State) {
	sh.StateList = append(sh.StateList, state)
	sh.flushHistory()
}

func (sh *StateHistory) popState() {
	if len(sh.StateList) < 2 {
		return
	}
	sh.StateList = sh.StateList[:len(sh.StateList)-1]
	sh.flushHistory()
}

type State struct {
	SortedNames                  []string
	Start, End, NamesIdx, ReqLen int
}

func clearState() {
	os.Remove(tmp_file)
}

func (s State) validate(names []string) bool {

	if s.Start > s.End {
		return false
	}

	if s.NamesIdx >= len(names) {
		return false
	}

	if len(s.SortedNames) >= s.ReqLen {
		return false
	}

	if len(names) != s.ReqLen {
		return false
	}

	return true
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

func (s SplitVisual) splitLayoutDyn(gtx layout.Context, layouts *[]layout.Widget, proportions *[]int) layout.Dimensions {
	var sm = 0

	for _, val := range *proportions {
		sm += val
	}

	single_width := gtx.Constraints.Min.X / sm
	var cur_sum = 0

	for i, val := range *layouts {
		gtx := gtx
		cur_width := (*proportions)[i] * single_width
		if i == len(*proportions)-1 {
			cur_width = gtx.Constraints.Min.X - single_width*(sm-(*proportions)[i])
		}
		gtx.Constraints = layout.Exact(image.Pt(cur_width, (gtx.Constraints.Max.Y)))
		trans := op.Offset(image.Pt(cur_sum, 0)).Push(gtx.Ops)
		val(gtx)
		trans.Pop()
		cur_sum += cur_width
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
func (s SplitVisual) splitLayout(gtx layout.Context, left, middle layout.Widget, right layout.Widget) layout.Dimensions {
	leftsize := gtx.Constraints.Min.X / 3
	rightsize := gtx.Constraints.Min.X - 2*leftsize

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(leftsize, gtx.Constraints.Max.Y))
		left(gtx)
	}

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(leftsize, gtx.Constraints.Max.Y))
		trans := op.Offset(image.Pt(leftsize, 0)).Push(gtx.Ops)
		middle(gtx)
		trans.Pop()
	}

	{
		gtx := gtx
		gtx.Constraints = layout.Exact(image.Pt(rightsize, gtx.Constraints.Max.Y))
		trans := op.Offset(image.Pt(2*leftsize, 0)).Push(gtx.Ops)
		right(gtx)
		trans.Pop()
	}

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func listView(gtx layout.Context, th *material.Theme, sortedListView *widget.List, items *[]string, goBackButton *widget.Clickable) layout.Dimensions {
	var margins = layout.Inset{
		Left:   unit.Dp(5),
		Right:  unit.Dp(5),
		Top:    unit.Dp(15),
		Bottom: unit.Dp(15),
	}

	var maxLen = 0
	for i := range len(*items) {
		var ln = len((*items)[i])
		if ln > maxLen {
			maxLen = ln
		}
	}

	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(
		gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return margins.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				ln_str := strconv.Itoa(len(*items)/10 + 1)
				return sortedListView.Layout(gtx, len(*items), func(gtx layout.Context, index int) layout.Dimensions {
					theme := *th
					theme.Face = font.Typeface("monospace")

					var margins = layout.Inset{
						Top:    unit.Dp(6),
						Bottom: unit.Dp(6),
					}
					lb := material.H5(&theme, fmt.Sprintf("%"+ln_str+"d: %s", index+1, (*items)[index]))
					lb.Alignment = text.Start
					lb.MaxLines = 1
					return margins.Layout(gtx, lb.Layout)
				})
			})
		}),
		layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				buttonMargins := layout.Inset{
					Top:    unit.Dp(25),
					Bottom: unit.Dp(25),
					Right:  unit.Dp(35),
					Left:   unit.Dp(35),
				}
				return buttonMargins.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, goBackButton, "Go back ("+"B"+")")
						return btn.Layout(gtx)
					})
			},
		),
	)

}

func chooseLayout(gtx layout.Context, th *material.Theme, state *State, sortedListView *widget.List, allNames *[]string, name1Button *widget.Clickable, name2Button *widget.Clickable, goBackButton *widget.Clickable) {
	layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, material.H5(th, fmt.Sprintf("%d left", state.ReqLen-len(state.SortedNames))).Layout)
		}),
		layout.Flexed(0.1, func(gtx layout.Context) layout.Dimensions {

			prs := []int{2, 1}

			lys := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
				lb := material.H4(th, "Which one is better?")
				lb.MaxLines = 1
				return layout.Center.Layout(gtx, lb.Layout)
			}, func(gtx layout.Context) layout.Dimensions {
				lb := material.H4(th, "Got so far")
				lb.MaxLines = 1
				return layout.Center.Layout(gtx, lb.Layout)
			}}
			return SplitVisual{}.splitLayoutDyn(gtx, &lys, &prs)
			// return layout.Center.Layout(gtx, material.H3(th, "Which one is better?").Layout)
		}),
		layout.Flexed(0.9, func(gtx layout.Context) layout.Dimensions {

			var lys = []layout.Widget{func(gtx layout.Context) layout.Dimensions {
				return FillWithLabel(gtx, th, state.SortedNames[(state.End+state.Start)/2], name1Button, "This one (Z)")
			},
				func(gtx layout.Context) layout.Dimensions {
					return FillWithLabel(gtx, th, (*allNames)[state.NamesIdx], name2Button, "This one (X)")
				},
				func(gtx layout.Context) layout.Dimensions {
					return listView(gtx, th, sortedListView, &state.SortedNames, goBackButton)
				}}

			var props = []int{1, 1, 1}
			return SplitVisual{}.splitLayoutDyn(gtx, &lys, &props)
		}),
	)
}

func FillWithLabel(gtx layout.Context, th *material.Theme, text string, button *widget.Clickable, buttonText string) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(
		gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, material.H4(th, text).Layout)
		}),
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
						btn := material.Button(th, button, buttonText)
						return btn.Layout(gtx)
					})
			},
		),
	)
}

func getTitles() []string {
	file, err := os.Open(titles_file)
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

	if len(lines) == 0 {
		return nil
	}

	return lines
}

func writeResults(arr []string) {
	file, err := os.OpenFile(results_file, os.O_CREATE|os.O_WRONLY, 0666)
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

func sortedView(theme *material.Theme, gtx layout.Context) {
	label := material.H4(theme, "Everything is sorted\n\n Results have been stored in "+results_file)
	label.Alignment = text.Middle
	layout.Center.Layout(gtx, label.Layout)
}

func run(window *app.Window) error {
	theme := material.NewTheme()
	titles := getTitles()
	haveDate := true
	history := getHistory()

	var curState State

	if titles == nil {
		haveDate = false
	} else {
		curState.ReqLen = len(titles)
	}

	if len(history.StateList) > 0 {
		state := history.StateList[len(history.StateList)-1]
		if !state.validate(titles) {
			history = StateHistory{}
		} else {
			curState = state
		}
	}

	sorted := false
	fmt.Println(titles)

	var name1Button widget.Clickable
	var name2Button widget.Clickable
	var createFileButton widget.Clickable
	var goBackButton widget.Clickable
	var ops op.Ops
	sortedListView := widget.List{
		List: layout.List{
			Axis: layout.Vertical,
		},
	}
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			name1KeyPress := false
			name2KeyPress := false
			createFileKeyPress := false
			goBackKeyPress := false

			if sorted {
				println("done")
				sortedView(theme, gtx)
				e.Frame(gtx.Ops)
				continue
			}

			for {
				ev, ok := gtx.Event(
					key.Filter{Name: "X"},
					key.Filter{Name: "Z"},
					key.Filter{Name: "C"},
					key.Filter{Name: "B"},
				)
				if !ok {
					break
				}
				if ev.(key.Event).State == key.Press {
					name := ev.(key.Event).Name
					switch name {
					case "Z":
						{
							name1KeyPress = true
						}
					case "X":
						{
							name2KeyPress = true
						}
					case "C":
						{
							createFileKeyPress = true
						}
					case "B":
						{
							goBackKeyPress = true
						}
					}
				}
			}

			if !haveDate {
				if createFileButton.Clicked(gtx) || createFileKeyPress {
					file, _ := os.Create(titles_file)
					if file != nil {
						file.Close()
					}
					exec.Command("cmd", "/c", "start", titles_file).Run()
					os.Exit(0)
				}

				FillWithLabel(gtx, theme, "Please fill in titles.txt", &createFileButton, "Open titles.txt (C)")
				e.Frame(gtx.Ops)
				break
			}

			if goBackButton.Clicked(gtx) || goBackKeyPress {
				history.popState()
				curState = history.StateList[len(history.StateList)-1]
			}

			mid := (curState.Start + curState.End) / 2

			if name1Button.Clicked(gtx) || name1KeyPress {
				curState.Start = mid + 1
				if curState.Start > curState.End {
					curState.Start = curState.End
				}
				history.addState(curState)
			}

			if name2Button.Clicked(gtx) || name2KeyPress {
				curState.End = mid
				if curState.End < curState.Start {
					curState.End = curState.Start
				}
				history.addState(curState)
			}

			if curState.Start == curState.End {
				curState.SortedNames = insert(curState.SortedNames, curState.Start, titles[curState.NamesIdx])
				curState.Start, curState.End = 0, len(curState.SortedNames)
				curState.NamesIdx++
				history.popState()
				history.addState(curState)

				if len(curState.SortedNames) == curState.ReqLen {
					sorted = true
					writeResults(curState.SortedNames)
					clearState()
					sortedView(theme, gtx)
					e.Frame(gtx.Ops)
					continue
				}
				fmt.Println(curState.SortedNames)
			}

			println(len(history.StateList))

			chooseLayout(gtx, theme, &curState, &sortedListView, &titles, &name1Button, &name2Button, &goBackButton)

			e.Frame(gtx.Ops)
		}
	}
}
