package ui

import (
	"net"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func LaunchLogin(app *tview.Application, host string, useTor bool, proxy string) {
	// Vista del Banner
	bannerView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(GetRandomBanner())

	// Formulario de Acceso
	form := tview.NewForm().
		AddInputField(" [ IDENTITY ] ", "", 25, nil, nil).
		AddPasswordField(" [ PASSCODE ] ", "", 25, '*', nil).
		AddInputField(" [ KEY_PATH ] ", "./resources/privada.pem", 40, nil, nil)

	form.AddButton(" [ CONNECT ] ", func() {
		username := form.GetFormItem(0).(*tview.InputField).GetText()

		go func() {
			// Intento de conexión con timeout
			conn, err := net.DialTimeout("tcp", host, 5*time.Second)
			if err != nil {
				app.QueueUpdateDraw(func() {
					form.SetTitle(" [ CONNECTION_ERROR ] ").SetTitleColor(tcell.ColorYellow)
				})
				return
			}

			app.QueueUpdateDraw(func() {
				LaunchChatUI(app, conn, username)
			})
		}()
	})

	// Estética Deep Web
	form.SetBorder(true).
		SetTitle(" --- SECURE GATEWAY --- ").
		SetTitleColor(tcell.ColorRed).
		SetBorderColor(tcell.ColorDarkRed)

	form.SetLabelColor(tcell.ColorRed).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite).
		SetButtonBackgroundColor(tcell.ColorDarkRed).
		SetButtonTextColor(tcell.ColorBlack)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(bannerView, 0, 2, false).
		AddItem(form, 0, 1, true)

	flex.SetBackgroundColor(tcell.ColorBlack)

	app.SetRoot(flex, true).SetFocus(form)
}
