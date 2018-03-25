/*
 * Copyright (c) 2018.
 * andy-zhangtao <ztao8607@gmail.com>
 */

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var name string
var debug bool
var user string
var passwd string

const (
	ModuleName          = "dpush"
	AliDockerRepository = "registry.cn-beijing.aliyuncs.com"
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
		cli.StringFlag{
			Name:        "user,u",
			Usage:       "Ali Repository User",
			Destination: &user,
		},
		cli.StringFlag{
			Name:        "passwd,p",
			Usage:       "Ali Repository Passwd",
			Destination: &passwd,
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
	logrus.WithFields(logrus.Fields{"Docker Version": v.Get("Version"), "Go Version": v.Get("GoVersion")}).Debug(ModuleName)

	if name == "" {
		// logrus.WithFields(logrus.Fields{"Image Name Empty!": ""}).Error(ModuleName)
		return errors.New("Image Name Empty!")
	}

	logrus.WithFields(logrus.Fields{"Ready To Push Docker Image": name}).Debug(ModuleName)

	repository := strings.Split(name, ":")
	if len(repository) < 2 {
		repository = append(repository, "latest")
	}

	aliName := fmt.Sprintf("%s/%s", AliDockerRepository, repository[0])
	err = cli.TagImage(name, docker.TagImageOptions{
		Repo:    aliName,
		Tag:     repository[1],
		Context: context.Background(),
	})
	if err != nil {
		logrus.WithFields(logrus.Fields{"Tag Docker Image Error": err}).Error(ModuleName)
		return err
	}

	logrus.WithFields(logrus.Fields{"Repository": AliDockerRepository, "Pull Image": repository[0], "Tag": repository[1]}).Debug(ModuleName)
	// pr, pw := io.Pipe()
	//
	// go func() {
	// 	for {
	// 		var data []byte
	// 		n, err := pr.Read(data)
	// 		if err != nil {
	// 			logrus.WithFields(logrus.Fields{"Read log": err}).Error(ModuleName)
	// 			return
	// 		}
	// 		if n > 0 {
	// 			fmt.Println(data[:n])
	// 		}
	// 	}
	// }()

	auth, err := cli.AuthCheck(&docker.AuthConfiguration{
		Username:      user,
		Password:      passwd,
		ServerAddress: AliDockerRepository,
	})

	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{"Auth": auth.Status}).Debug(ModuleName)

	var buf bytes.Buffer
	noStop := true
	go func() {
		for {
			if noStop {
				fmt.Println("\033[H\033[2J")
				fmt.Println(buf.String())
			} else {
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()
	err = cli.PushImage(docker.PushImageOptions{
		Name: aliName,
		Tag:  repository[1],
		// Registry:          AliDockerRepository,
		RawJSONStream: false,
		OutputStream:  &buf,
		Context:       context.Background(),
	}, docker.AuthConfiguration{
		Username:      user,
		Password:      passwd,
		ServerAddress: AliDockerRepository,
	})
	if err != nil {
		noStop = false
		return err
	}

	noStop = false
	fmt.Printf("O! [%s] Push Succ.", aliName)
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
