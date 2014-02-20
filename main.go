package main

import (
	"github.com/shirro/gin/lib"
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/codegangsta/envy/lib"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	startTime  = time.Now()
	logger     = log.New(os.Stdout, "[gin] ", 0)
	buildError error
)

func main() {
	app := cli.NewApp()
	app.Name = "gin"
	app.Usage = "A live reload utility for Go web applications."
	app.Action = MainAction
	app.Flags = []cli.Flag{
		cli.StringFlag{"host,l", "127.0.0.1", "listen address for the proxy server"},
		cli.IntFlag{"port,p", 3000, "port for the proxy server"},
		cli.IntFlag{"appPort,a", 3001, "port for the Go web server"},
		cli.StringFlag{"bin,b", "gin-bin", "name of generated binary file"},
	}
	app.Commands = []cli.Command{
		{
			Name:      "run",
			ShortName: "r",
			Usage:     "Run the gin proxy in the current working directory",
			Action:    MainAction,
		},
		{
			Name:      "env",
			ShortName: "e",
			Usage:     "Display environment variables set by the .env file",
			Action:    EnvAction,
		},
	}

	app.Run(os.Args)
}

func MainAction(c *cli.Context) {
	port := c.GlobalInt("port")
	appPort := strconv.Itoa(c.GlobalInt("appPort"))
	host := c.GlobalString("host")

	// Bootstrap the environment
	envy.Bootstrap()

	// Set the PORT env
	os.Setenv("PORT", appPort)

	wd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}

	builder := gin.NewBuilder(".", c.GlobalString("bin"))
	runner := gin.NewRunner(filepath.Join(wd, builder.Binary()))
	runner.SetWriter(os.Stdout)
	proxy := gin.NewProxy(builder, runner)

	config := &gin.Config{
		Host:    host,
		Port:    port,
		ProxyTo: "http://localhost:" + appPort,
	}

	err = proxy.Run(config)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Printf("listening on port %d\n", port)

	// build right now
	build(builder, logger)

	// scan for changes
	scanChanges(func(path string) {
		runner.Kill()
		build(builder, logger)
	})
}

func EnvAction(c *cli.Context) {
	// Bootstrap the environment
	env, err := envy.Bootstrap()
	if err != nil {
		logger.Fatalln(err)
	}

	for k, v := range env {
		fmt.Printf("%s: %s\n", k, v)
	}

}

func build(builder gin.Builder, logger *log.Logger) {
	err := builder.Build()
	if err != nil {
		buildError = err
		logger.Println("ERROR! Build failed.")
		fmt.Println(builder.Errors())
	} else {
		// print success only if there were errors before
		if buildError != nil {
			logger.Println("Build Successful")
		}
		buildError = nil
	}

	time.Sleep(100 * time.Millisecond)
}

type scanCallback func(path string)

func scanChanges(cb scanCallback) {
	for {
		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if path == ".git" {
				return filepath.SkipDir
			}

			if filepath.Ext(path) == ".go" && info.ModTime().After(startTime) {
				cb(path)
				startTime = time.Now()
				return errors.New("done")
			}

			return nil
		})
		time.Sleep(500 * time.Millisecond)
	}
}
