package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type TestLine struct {
	Time    string
	Action  string
	Package string
	Output  string
}

type TestItem struct {
	Node       *tview.TreeNode
	Name       string
	Package    string
	lastOutput *strings.Builder
}

var dir string
var app *tview.Application

func main() {
	f, err := os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)

	dir = ""
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	helpText := tview.NewTextView().SetText("space = Select Highlighted Test r = Run Selected Tests")

	selectedTests := make(map[*TestItem]bool)
	root := tview.NewTreeNode("")
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)

	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(tree, 0, 1, true)
	layout.AddItem(helpText, 1, 0, false)
	app = tview.NewApplication().SetRoot(layout, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			if event.Rune() == 'q' {
				app.Stop()
			} else if event.Rune() == 'r' {
				go runSelectedTests(selectedTests)
			}
		}

		return event
	})

	raw, err := exec.Command("go", "test", "-list=.*", "-json", dir).Output()
	if err != nil {
		panic(err)
	}

	packageNodes := map[string]*tview.TreeNode{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		testLine := TestLine{}
		err := json.Unmarshal(scanner.Bytes(), &testLine)
		if err != nil {
			panic(err)
		}

		if testLine.Action == "output" && strings.HasPrefix(testLine.Output, "Test") {
			packageNode, ok := packageNodes[testLine.Package]
			if !ok {
				packageNode = tview.NewTreeNode(testLine.Package).SetColor(tcell.ColorGreen).SetSelectable(true)
				packageNodes[testLine.Package] = packageNode
				root.AddChild(packageNode)
			}

			node := tview.NewTreeNode(fmt.Sprintf("  ( ) %s", testLine.Output)).SetSelectable(true)
			testItem := &TestItem{
				Node:    node,
				Name:    testLine.Output,
				Package: testLine.Package,
			}
			packageNode.AddChild(node.SetReference(testItem))
		}
	}

	log.Print("Ready")

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		if len(node.GetChildren()) != 0 {
			node.SetExpanded(!node.IsExpanded())
			return
		}

		testItem := node.GetReference().(*TestItem)

		_, ok := selectedTests[testItem]
		if ok {
			delete(selectedTests, testItem)
			app.QueueUpdateDraw(func() { node.SetText(replaceAtIndex(node.GetText(), ' ', 3)) })
		} else {
			selectedTests[testItem] = true
			app.QueueUpdateDraw(func() { node.SetText(replaceAtIndex(node.GetText(), 'x', 3)) })
		}
	})

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func replaceAtIndex(in string, r rune, i int) string {
	out := []rune(in)
	out[i] = r
	return string(out)
}

func runSelectedTests(selectedTests map[*TestItem]bool) {
	for testItem, _ := range selectedTests {
		thisTestNode := testItem.Node
		app.QueueUpdateDraw(func() { thisTestNode.SetText(replaceAtIndex(thisTestNode.GetText(), 'R', 0)) })
	}

	for testItem, _ := range selectedTests {
		runTest(testItem.Node)
	}
}

func runTest(node *tview.TreeNode) {
	reference := node.GetReference()
	if reference == nil {
		return
	}

	testItem := reference.(*TestItem)

	raw, err := exec.Command("go", "test", "-run", testItem.Name, "-json", dir).Output()
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		testLine := TestLine{}
		err := json.Unmarshal(scanner.Bytes(), &testLine)
		if err != nil {
			panic(err)
		}

		if testLine.Action == "pass" {
			app.QueueUpdateDraw(func() { node.SetText(replaceAtIndex(node.GetText(), 'P', 0)) })
		} else if testLine.Action == "fail" {
			app.QueueUpdateDraw(func() { node.SetText(replaceAtIndex(node.GetText(), 'F', 0)) })
		}
	}
}
