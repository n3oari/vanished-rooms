package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func StartApp(host string, useTor bool, proxy string) {
	app := tview.NewApplication()

	// Configuración de colores para anular el fondo azul
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.PrimaryTextColor = tcell.ColorRed
	tview.Styles.BorderColor = tcell.ColorDarkRed

	// LLAMADA: Aquí solo llamamos a la función que está en login.go
	LaunchLogin(app, host, useTor, proxy)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
