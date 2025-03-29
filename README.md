# Writ 

## A Simple Multi-Document, Console-Based Word Processor

`writ` is a word processor that runs in a console window.  

<!--p align="center">
<img src="gv.PNG" width=60%>
</p-->


## Getting Started

Either build `writ` from source (see below) or download a pre-built binary release.  `writ` is a single binary executable and can run anywhere.

```bash
$ ./writ
```


## Compiling writ

writ requires golang 1.24 or higher.  It has very few dependencies by design and uses the excellent [gdamore/tcell](https://github.com/gdamore/tcell) and [rivo/tview](https://github.com/rivo/tview) libraries to handle the screen management.

```bash
$ go mod tidy
$ go build -o writ *.go
$ ./writ
```

## Testing writ

The heart of writ is a [PieceTable](https://www.cs.unm.edu/~crowley/papers/sds/node15.html#SECTION00064000000000000000) implementation the provides an efficient data structure to edit large amounts of text. There are extensive unit tests for the PieceTable that can be run as follows:

```bash
$ cd writ/internal/util
$ go test
```

## Debugging writ with VSCode/Delve

Set up the golang extension, Delve, etc...

Launching directly from VSCode doesn't work, you need to run the app outside of VSCode and attach to the process.

First make sure you're built in a way that doesn't interfere with debugging

```bash
$ go build -gcflags=all="-N -l" -o writ *.go
$ ./writ
```

Make sure you have a launch configuration set (in `.vscode/launch.json`)
```json
    {
        "name": "Launch Golang",
        "type": "go",
        "request": "attach",
        "mode": "local",
        "processId": 0
    }
```
Now when you use `F5` to start a debug session (from the `cmd/app.go` file), VSCode will prompt you to find the process you want to attach to- simply pick `writ` from the list and you should be good to go. Set some breakpoints and debug away.


## Getting Help

Found a bug?  Have an idea for an enhancement?  Feel free to create an issue (or better yet submit a pull request).  Please note that `writ` was built for my own personal use.  While I hope someone else finds it useful I can't promise I'll fix your bug or implement your idea.  This software is offered AS-IS with no warranty whatsoever. 


