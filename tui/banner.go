package tui

import "strings"

// Banner is the ZETACHAT ASCII logo.
// ASCII-only so it renders consistently across Windows terminals.
func banner() string {
	const art = `
███████╗███████╗████████╗ █████╗  ██████╗██╗  ██╗ █████╗ ████████╗
╚══███╔╝██╔════╝╚══██╔══╝██╔══██╗██╔════╝██║  ██║██╔══██╗╚══██╔══╝
  ███╔╝ █████╗     ██║   ███████║██║     ███████║███████║   ██║
 ███╔╝  ██╔══╝     ██║   ██╔══██║██║     ██╔══██║██╔══██║   ██║
███████╗███████╗   ██║   ██║  ██║╚██████╗██║  ██║██║  ██║   ██║
╚══════╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝
`

	return strings.TrimSpace(art)
}