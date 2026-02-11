package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colores de la aplicación
var (
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSuccess   = lipgloss.Color("#00D9A3")
	ColorWarning   = lipgloss.Color("#FFB800")
	ColorError     = lipgloss.Color("#FF4757")
	ColorInfo      = lipgloss.Color("#5DA7FF")
	ColorSecondary = lipgloss.Color("#FF6B9D")
	ColorMuted     = lipgloss.Color("#7C7C7C")
)

var (
	ReceivedMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#3C3C3C")).
				Padding(0, 1).
				MarginLeft(1)

	UsernameStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	SystemMessageStyle = lipgloss.NewStyle().
				Foreground(ColorInfo).
				Bold(true).
				Italic(true)

	ErrorMessageStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)

	WarningMessageStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	SuccessMessageStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	PromptStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	EncryptedIndicatorStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)
)

func RenderChatMessage(username, message string) string {
	user := UsernameStyle.Render(username)
	msg := ReceivedMessageStyle.Render(message)
	return user + ": " + msg
}

func RenderSystemMessage(message string) string {
	return SystemMessageStyle.Render("[SYSTEM] " + message)
}

func RenderErrorMessage(message string) string {
	return ErrorMessageStyle.Render("[!] " + message)
}

func RenderWarningMessage(message string) string {
	return WarningMessageStyle.Render("[!] " + message)
}

func RenderSuccessMessage(message string) string {
	return SuccessMessageStyle.Render("[+] " + message)
}

func RenderInfoMessage(message string) string {
	return SystemMessageStyle.Render("[i] " + message)
}

func RenderPrompt() string {
	return PromptStyle.Render("> ")
}

func RenderEncryptedIndicator() string {
	return EncryptedIndicatorStyle.Render("🔒 ")
}

func RenderRoomList(rooms []string) string {
	if len(rooms) == 0 {
		return RenderInfoMessage("No public rooms available.")
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Border(lipgloss.DoubleBorder(), true, false, false, false).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(40)

	roomItemStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		PaddingLeft(2)

	var sb strings.Builder
	sb.WriteString(headerStyle.Render("📋 AVAILABLE ROOMS") + "\n\n")

	for i, room := range rooms {
		icon := "🏠"
		if i == 0 {
			icon = "⭐"
		}
		sb.WriteString(roomItemStyle.Render(fmt.Sprintf("%s %s", icon, room)) + "\n")
	}

	borderStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Width(40)

	sb.WriteString("\n" + borderStyle.Render(strings.Repeat("─", 38)))

	return sb.String()
}

func RenderUserList(users []string, currentUser string) string {
	if len(users) == 0 {
		return RenderInfoMessage("No users in this room.")
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true).
		Border(lipgloss.DoubleBorder(), true, false, false, false).
		BorderForeground(ColorSuccess).
		Padding(0, 1).
		Width(40)

	userItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		PaddingLeft(2)

	currentUserStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		PaddingLeft(2)

	var sb strings.Builder
	sb.WriteString(headerStyle.Render("👥 USERS IN ROOM") + "\n\n")

	for _, user := range users {
		if user == currentUser {
			sb.WriteString(currentUserStyle.Render(fmt.Sprintf("🔹 %s (you)", user)) + "\n")
		} else {
			sb.WriteString(userItemStyle.Render(fmt.Sprintf("• %s", user)) + "\n")
		}
	}

	borderStyle := lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Width(40)

	sb.WriteString("\n" + borderStyle.Render(strings.Repeat("─", 38)))

	return sb.String()
}

func RenderHelpMenu() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(ColorPrimary).
		Bold(true).
		Padding(0, 2).
		Width(58).
		Align(lipgloss.Center)

	sectionStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true).
		Underline(true)

	commandStyle := lipgloss.NewStyle().
		Foreground(ColorInfo).
		PaddingLeft(2)

	descStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		PaddingLeft(6)

	borderStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary)

	var sb strings.Builder

	sb.WriteString(borderStyle.Render("┌"+strings.Repeat("─", 58)+"┐") + "\n")
	sb.WriteString(borderStyle.Render("│") + titleStyle.Render("VANISHED ROOMS - COMMAND MENU") + borderStyle.Render("│") + "\n")
	sb.WriteString(borderStyle.Render("├"+strings.Repeat("─", 58)+"┤") + "\n\n")

	// ROOM CREATION
	sb.WriteString(sectionStyle.Render(" ROOM CREATION") + "\n")
	sb.WriteString(commandStyle.Render("/create -n <name> --public") + "\n")
	sb.WriteString(descStyle.Render("Create a room visible to everyone") + "\n\n")
	sb.WriteString(commandStyle.Render("/create -n <name> -p <pass> --private") + "\n")
	sb.WriteString(descStyle.Render("Secure room (min 8 chars password)") + "\n\n")

	// NAVIGATION
	sb.WriteString(sectionStyle.Render("NAVIGATION") + "\n")
	sb.WriteString(commandStyle.Render("/rooms") + "\n")
	sb.WriteString(descStyle.Render("List all public rooms") + "\n\n")
	sb.WriteString(commandStyle.Render("/join -n <name>") + "\n")
	sb.WriteString(descStyle.Render("Join public room (no -p needed)") + "\n\n")
	sb.WriteString(commandStyle.Render("/join -n <name> -p <pass>") + "\n")
	sb.WriteString(descStyle.Render("Join private room") + "\n\n")
	sb.WriteString(commandStyle.Render("/leave-room") + "\n")
	sb.WriteString(descStyle.Render("Exit current room") + "\n\n")

	// SYSTEM
	sb.WriteString(sectionStyle.Render(" SYSTEM") + "\n")
	sb.WriteString(commandStyle.Render("/users") + "\n")
	sb.WriteString(descStyle.Render("List participants in room") + "\n\n")
	sb.WriteString(commandStyle.Render("/help") + "\n")
	sb.WriteString(descStyle.Render("Show this menu") + "\n\n")
	sb.WriteString(commandStyle.Render("/quit") + "\n")
	sb.WriteString(descStyle.Render("Close connection") + "\n\n")

	sb.WriteString(borderStyle.Render("└" + strings.Repeat("─", 58) + "┘"))

	return sb.String()
}
