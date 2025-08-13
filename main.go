package main

import (
	"os"

	"github.com/jpillora/overseer"
	"github.com/benedict-erwin/insight-collector/cmd"

	_ "github.com/benedict-erwin/insight-collector/http/route"
)

// main initializes and starts the application with overseer for zero-downtime deployment
func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "serve":
			// HTTP server with overseer (:3000)
			overseer.Run(overseer.Config{
				Program: func(state overseer.State) {
					cmd.Execute()
				},
				Address:          ":3000",
				RestartSignal:    overseer.SIGUSR2,
				TerminateTimeout: 30,
			})
		case "dev":
			// Development mode without overseer (for air hot reload)
			cmd.Execute()
		case "worker":
			if len(os.Args) >= 3 && os.Args[2] == "start" {
				// Worker with overseer (:3001)
				overseer.Run(overseer.Config{
					Program: func(state overseer.State) {
						cmd.Execute()
					},
					Address:          ":3001",
					RestartSignal:    overseer.SIGUSR2,
					TerminateTimeout: 30,
				})
			} else {
				// Worker CLI commands without overseer
				cmd.Execute()
			}
		default:
			// Other commands without overseer
			cmd.Execute()
		}
	} else {
		cmd.Execute()
	}
}
