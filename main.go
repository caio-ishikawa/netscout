package main

import (
	"fmt"
	"github.com/caio-ishikawa/netscout/app"
)

const logo = `
==========================================
 ▐ ▄ ▄▄▄ .▄▄▄▄▄.▄▄ ·  ▄▄·       ▄• ▄▌▄▄▄▄▄
•█▌▐█▀▄.▀·•██  ▐█ ▀. ▐█ ▌▪▪     █▪██▌•██  
▐█▐▐▌▐▀▀▪▄ ▐█.▪▄▀▀▀█▄██ ▄▄ ▄█▀▄ █▌▐█▌ ▐█.▪
██▐█▌▐█▄▄▌ ▐█▌·▐█▄▪▐█▐███▌▐█▌.▐▌▐█▄█▌ ▐█▌·
▀▀ █▪ ▀▀▀  ▀▀▀  ▀▀▀▀ ·▀▀▀  ▀█▄▀▪ ▀▀▀  ▀▀▀
@caio-ishikawa - github.com/caio-ishikawa
==========================================`

func main() {
	fmt.Printf("%s\n\n", logo)

	settings, err := app.ParseFlags()
	if err != nil {
		panic(err)
	}

	app, err := app.NewApp(settings)
	if err != nil {
		panic(err)
	}

	app.Start()
}
