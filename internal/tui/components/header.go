package components

import (
	"openllmjudge/internal/styles"
	"os"

	"github.com/charmbracelet/lipgloss"
)

func RenderHeader(width int) string {
	if width <= 0 {
		width = 80
	}

	// Top row: Welcome (Left) and Version (Right)
	availableWidth := width - 4
	if availableWidth < 0 {
		availableWidth = 0
	}

	welcome := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Purple).
		Render("Hi, welcome!")

	welcomeWidth := lipgloss.Width(welcome)
	version := styles.DimStyle.Render("v0.0.1")

	versionBoxWidth := availableWidth - welcomeWidth
	if versionBoxWidth < 0 {
		versionBoxWidth = 0
	}

	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		welcome,
		lipgloss.NewStyle().Width(versionBoxWidth).Align(lipgloss.Right).Render(version),
	)

	asciiArt := ` ██████╗ ██████╗ ███████╗███╗   ██╗               
██╔═══██╗██╔══██╗██╔════╝████╗  ██║               
██║   ██║██████╔╝█████╗  ██╔██╗ ██║               
██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║               
╚██████╔╝██║     ███████╗██║ ╚████║               
 ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝                                                            
     ██╗██╗   ██╗██████╗  ██████╗ ███████╗███████╗
     ██║██║   ██║██╔══██╗██╔════╝ ██╔════╝██╔════╝
     ██║██║   ██║██║  ██║██║  ███╗█████╗  ███████╗
██   ██║██║   ██║██║  ██║██║   ██║██╔══╝  ╚════██║
╚█████╔╝╚██████╔╝██████╔╝╚██████╔╝███████╗███████║
 ╚════╝  ╚═════╝ ╚═════╝  ╚═════╝ ╚══════╝╚══════╝`

	tagline := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Gray).
		Render("OpenLLMJudges uses LLM (Large Language Models) to evaluate AI responses against criteria.")

	cwd, _ := os.Getwd()
	pathLine := styles.DimStyle.Faint(true).Render(cwd)

	return lipgloss.JoinVertical(lipgloss.Left,
		topRow,
		styles.HeaderStyle.MarginBottom(0).Render(asciiArt),
		tagline,
		pathLine,
	)
}
