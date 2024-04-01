package main

import (
	"fmt"
	"github.com/caio-ishikawa/netscout/app"
)

const logo = `
=======================================================================
 ███▄    █ ▓█████▄▄▄█████▓  ██████  ▄████▄   ▒█████   █    ██ ▄▄▄█████▓
 ██ ▀█   █ ▓█   ▀▓  ██▒ ▓▒▒██    ▒ ▒██▀ ▀█  ▒██▒  ██▒ ██  ▓██▒▓  ██▒ ▓▒
▓██  ▀█ ██▒▒███  ▒ ▓██░ ▒░░ ▓██▄   ▒▓█    ▄ ▒██░  ██▒▓██  ▒██░▒ ▓██░ ▒░
▓██▒  ▐▌██▒▒▓█  ▄░ ▓██▓ ░   ▒   ██▒▒▓▓▄ ▄██▒▒██   ██░▓▓█  ░██░░ ▓██▓ ░ 
▒██░   ▓██░░▒████▒ ▒██▒ ░ ▒██████▒▒▒ ▓███▀ ░░ ████▓▒░▒▒█████▓   ▒██▒ ░ 
░ ▒░   ▒ ▒ ░░ ▒░ ░ ▒ ░░   ▒ ▒▓▒ ▒ ░░ ░▒ ▒  ░░ ▒░▒░▒░ ░▒▓▒ ▒ ▒   ▒ ░░   
░ ░░   ░ ▒░ ░ ░  ░   ░    ░ ░▒  ░ ░  ░  ▒     ░ ▒ ▒░ ░░▒░ ░ ░     ░    
   ░   ░ ░    ░    ░      ░  ░  ░  ░        ░ ░ ░ ▒   ░░░ ░ ░   ░      
         ░    ░  ░              ░  ░ ░          ░ ░     ░              
=======================================================================
`

func main() {
	fmt.Printf("%s", logo)

	settings, err := app.ParseFlags()
	if err != nil {
		fmt.Println("Error parsing flags. Run netscout with -h to see available flags.")
		return
	}

	app, err := app.NewApp(settings)
	if err != nil {
		fmt.Printf("Error initializing netscout. Failed with error: %s\n", err.Error())
		return
	}

	app.Scan()
}
