/*
 * Copyright (c) 2018.
 * andy-zhangtao <ztao8607@gmail.com>
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	tm "github.com/buger/goterm"
	"github.com/fsouza/go-dockerclient"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

var name string
var debug bool
var user string
var passwd string
var needPasswd bool

const (
	ModuleName          = "dpush"
	AliDockerRepository = "registry.cn-beijing.aliyuncs.com"
)

type Process struct {
	Status   string `json:"status"`
	Progress string `json:"progress"`
	Id       string `json:"id"`
}

func main() {
	app := cli.NewApp()
	app.Name = "dpush"
	app.Usage = "Push your docker image to ali docker repositry"
	app.Version = "v0.1.1"
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
		cli.BoolFlag{
			Name:        "passwd,p",
			Usage:       "Ali Repository Passwd",
			Destination: &needPasswd,
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

	if needPasswd {
		passwd, err = getPasswd()
		if err != nil {
			return err
		}
		if passwd == "" {
			return errors.New("Password Can not Empty!")
		}
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
				if len(buf.Bytes()) > 0 {
					ps := strings.Split(buf.String(), "\r\n")
					if len(ps) > 0 {
						for _, pps := range ps {
							if pps != "" {
								var pr Process
								err := json.Unmarshal([]byte(pps), &pr)
								if err != nil {
									logrus.WithFields(logrus.Fields{"Unmarshal Error": err, "json": pps}).Error(ModuleName)
									return
								}
								fmt.Printf("%s %s %s \n", tm.Color(pr.Id, tm.GREEN), tm.Color(pr.Status, tm.BLUE), tm.Color(pr.Progress, tm.RED))
							}
						}
					}
				}
				time.Sleep(2 * time.Second)
			} else {
				return
			}

		}
	}()
	err = cli.PushImage(docker.PushImageOptions{
		Name: aliName,
		Tag:  repository[1],
		// Registry:          AliDockerRepository,
		RawJSONStream: true,
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

// getPasswd 从Stdin读取口令
func getPasswd() (string, error) {
	fmt.Print("Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	password := string(bytePassword)

	return strings.TrimSpace(password), nil
}
