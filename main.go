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

func main() {
	dir := ""
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	root := tview.NewTreeNode("")
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	app := tview.NewApplication().SetRoot(tree, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			app.Stop()
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

			node := tview.NewTreeNode(testLine.Output).SetReference(testLine.Output).SetSelectable(true)
			packageNode.AddChild(node)
		}
	}

	log.Print("Ready")

	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		if len(node.GetChildren()) != 0 {
			node.SetExpanded(!node.IsExpanded())
			return
		}

		reference := node.GetReference()
		if reference == nil {
			return
		}

		testName := reference.(string)

		node.SetText(fmt.Sprintf("R %s", testName))

		go func() {
			raw, err := exec.Command("go", "test", "-run", testName, "-json", dir).Output()
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
					app.QueueUpdateDraw(func() { node.SetText(fmt.Sprintf("P %s", testName)) })
				} else if testLine.Action == "fail" {
					app.QueueUpdateDraw(func() { node.SetText(fmt.Sprintf("F %s", testName)) })
				}
			}
		}()
	})

	if err := app.Run(); err != nil {
		panic(err)
	}

}
