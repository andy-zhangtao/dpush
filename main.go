/*
 * Copyright (c) 2018.
 * andy-zhangtao <ztao8607@gmail.com>
 */

package main

import (
	"os"

	"github.com/fsouza/go-dockerclient"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var name string
var debug bool

const (
	ModuleName = "dpush"
)

func main() {
	app := cli.NewApp()
	app.Name = "dpush"
	app.Usage = "Push your docker image to ali docker repositry"
	app.Version = "v0.0.1"
	app.Author = "andy zhang"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "image, i",
			Usage:       "The Docker Image ",
			Destination: &name,
		},
		cli.BoolFlag{
			Name:        "verbose, V",
			Usage:       "Enable Verbose Logging",
			Destination: &debug,
		},
	}
	app.Action = pushAction
	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}

}

func pushAction(c *cli.Context) error {
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	cli, err := checkDocker()
	if err != nil {
		logrus.WithFields(logrus.Fields{"Docker Check Error": err}).Error(ModuleName)
	}

	v, _ := cli.Version()
	logrus.WithFields(logrus.Fields{"Docker Version": v.Get("Version"),"Go Version":v.Get("GoVersion")}).Debug(ModuleName)

	return nil
}

func checkDocker() (client *docker.Client, err error) {
	client, err = docker.NewClientFromEnv()
	if err != nil {
		panic(err)
	}

	err = client.Ping()
	return
}
