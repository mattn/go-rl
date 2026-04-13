# Simple readline for multi-platform

## Supported OSs

* Linux
* Windows
* FreeBSD
* Mac OS X

## TODO

* ~~wrap line~~
* redraw characters that have modified 
* hide overflowed characters
* history
* ~~completion~~
* key binding
* ~~password inputs~~

## Ctrl-D

Set `EOFOnCtrlD` to make `^D` return `io.EOF` even when the current line is not empty.

```go
r := rl.NewRl()
r.EOFOnCtrlD = true
```
