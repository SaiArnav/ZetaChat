package tui

import "strings"

// Banner is the ZETACHAT ASCII logo, drawn with block characters.
// It adapts its color to the terminal theme (light text on dark, dark text on light).
func banner() string {
	const art = `███████╗███████╗████████╗ █████╗  ██████╗██╗  ██╗ █████╗ ████████╗
██╔════╝██╔════╝╚══██╔══╝██╔══██╗██╔════╝██║  ██║██╔══██╗╚══██╔══╝
█████╗  █████╗     ██║   ███████║██║     ███████║███████║   ██║   
██╔══╝  ██╔══╝     ██║   ██╔══██║██║     ██╔══██║██╔══██║   ██║   
███████╗███████╗   ██║   ██║  ██║╚██████╗██║  ██║██║  ██║   ██║   
╚══════╝╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   `

	return strings.TrimSuffix(art, "\n")
}
