package ui

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	GlobalApp      *tview.Application
	GlobalTextView *tview.TextView
	GlobalRoomList *tview.List
)

func LaunchChatUI(app *tview.Application, conn net.Conn, username string) {
	GlobalApp = app

	// 1. Sidebar
	GlobalRoomList = tview.NewList().
		ShowSecondaryText(false).
		SetSelectedTextColor(tcell.ColorBlack).
		SetSelectedBackgroundColor(tcell.ColorRed)
	GlobalRoomList.SetBorder(true).SetTitle(" [ ROOMS ] ").SetBorderColor(tcell.ColorDarkRed)

	// 2. TextView (Log)
	GlobalTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetRegions(true)
	GlobalTextView.SetBorder(true).SetTitle(" [ VANISHED_LOG ] ").SetBorderColor(tcell.ColorDarkRed)

	// 3. InputField
	inputField := tview.NewInputField().
		SetLabel(" >_ ").
		SetLabelColor(tcell.ColorRed).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite)
	inputField.SetBorder(true).SetBorderColor(tcell.ColorMaroon)

	// 4. Goroutine de Recepción: Lectura asíncrona total
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			rawMsg := scanner.Text()
			// IMPORTANTE: No procesar aquí, solo enviar a la cola de dibujo
			WriteToTerminal("RECV", "darkred", rawMsg)
		}
		if err := scanner.Err(); err != nil {
			WriteToTerminal("SYSTEM", "yellow", "ERROR DE LECTURA: "+err.Error())
		}
	}()

	// 5. Lógica de Envío: NO BLOQUEANTE
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := strings.TrimSpace(inputField.GetText())
			if text != "" {
				// Ejecutamos el envío en una goroutine separada para que la UI no se congele
				go func(m string) {
					_, err := fmt.Fprintf(conn, "%s\n", m)
					if err != nil {
						WriteToTerminal("ERROR", "red", "No se pudo enviar el mensaje")
						return
					}
					// El echo del mensaje lo dará el servidor o lo pintamos nosotros aquí:
					WriteToTerminal("SEND", "red", m)
				}(text)

				inputField.SetText("")
			}
		}
	})

	// 6. Layout
	mainContent := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(GlobalTextView, 0, 1, false).
		AddItem(inputField, 3, 1, true)

	layout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(GlobalRoomList, 20, 1, false).
		AddItem(mainContent, 0, 4, true)

	layout.SetBackgroundColor(tcell.ColorBlack)
	app.SetRoot(layout, true).SetFocus(inputField)
}

func WriteToTerminal(tag string, color string, message string) {
	if GlobalApp == nil || GlobalTextView == nil {
		return
	}
	// Usamos QueueUpdateDraw para delegar el dibujo al hilo principal de forma segura
	GlobalApp.QueueUpdateDraw(func() {
		ts := time.Now().Format("15:04")
		fmt.Fprintf(GlobalTextView, "[gray]%s [-][%s]%s:[-] [white]%s\n", ts, color, tag, message)
		GlobalTextView.ScrollToEnd()
	})
}

// ... (mismo código anterior)

// UpdateRoomsList limpia y llena el panel lateral con las salas recibidas
func UpdateRoomsList(rooms []string) {
	if GlobalApp == nil || GlobalRoomList == nil {
		return
	}
	GlobalApp.QueueUpdateDraw(func() {
		GlobalRoomList.Clear()
		for _, r := range rooms {
			GlobalRoomList.AddItem(" # "+r, "", 0, nil)
		}
	})
}
