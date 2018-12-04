package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	//"github.com/gdamore/tcell"
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

	raw, err := exec.Command("go", "test", "-list=.*", "-json", dir).Output()
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

		if testLine.Action == "output" && strings.HasPrefix(testLine.Output, "Test") {
			node := tview.NewTreeNode(testLine.Output)
			root.AddChild(node)
		}
	}

	if err := tview.NewApplication().SetRoot(tree, true).Run(); err != nil {
		panic(err)
	}
}
