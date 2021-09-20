module examples

go 1.17

replace github.com/charmbracelet/charm => ../../charm

replace github.com/charmbracelet/bubbletea => ../../bubbletea

replace github.com/charmbracelet/wish => ../

require (
	github.com/charmbracelet/bubbletea v0.15.0
	github.com/charmbracelet/wish v0.0.0-20210823165316-d78c851f07fe
	github.com/gliderlabs/ssh v0.3.3
)

require (
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/charmbracelet/keygen v0.1.2 // indirect
	github.com/containerd/console v1.0.2 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.14-0.20210829144114-504425e14f74 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mikesmitty/edkey v0.0.0-20170222072505-3356ea4e686a // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.9.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/term v0.0.0-20210422114643-f5beecf764ed // indirect
)
