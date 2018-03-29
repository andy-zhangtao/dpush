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
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"time"

	tm "github.com/buger/goterm"
	"github.com/fsouza/go-dockerclient"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

var name string
var debug bool
var user string
var passwd string
var needPasswd bool
// defaultNameSpace 默认命名空间,当上传没有命名空间镜像时会自动添加到这个命名空间中
var defaultNameSpace string

var DPUSHCONF = os.Getenv("HOME") + "/.dpush.toml"

const (
	ModuleName          = "dpush"
	AliDockerRepository = "registry.cn-beijing.aliyuncs.com"
)

type Process struct {
	Status   string `json:"status"`
	Progress string `json:"progress"`
	Id       string `json:"id"`
}

type Repository struct {
	Repositorys map[string]Info
	Namespace   string
}

type Info struct {
	User   string `toml:"user"`
	Passwd string `toml:"passwd"`
}

func main() {
	app := cli.NewApp()
	app.Name = "dpush"
	app.Usage = "Push your docker image to ali docker repositry"
	app.Version = "v0.3.1"
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
		cli.StringFlag{
			Name:        "default,d",
			Usage:       "The Default Namespace",
			Destination: &defaultNameSpace,
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

	if name == "" {
		cli.ShowAppHelp(c)
		return nil
	}

	cli, err := checkDocker()
	if err != nil {
		logrus.WithFields(logrus.Fields{"Docker Check Error": err}).Error(ModuleName)
	}

	v, _ := cli.Version()
	logrus.WithFields(logrus.Fields{"Docker Version": v.Get("Version"), "Go Version": v.Get("GoVersion")}).Debug(ModuleName)

	if name == "" {
		return errors.New("Image Name Empty!")
	}

	if user == "" {
		info, err := getRepositoryInfo()
		if err != nil {
			return err
		}

		repoName, _ := reverRepositoryName(AliDockerRepository, true)
		if info.Repositorys[repoName].User == "" || info.Repositorys[repoName].Passwd == "" {
			return errors.New(fmt.Sprintf("This Repostiry [%s] Does Not Save. Please Type User & Password", AliDockerRepository))
		}

		user = info.Repositorys[repoName].User
		passwd = info.Repositorys[repoName].Passwd
		needPasswd = false
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

	logrus.WithFields(logrus.Fields{"Ready To Push Docker Image": name, "user": user, "passwd": fmt.Sprintf("%s*****%s", passwd[:1], passwd[len(passwd)-1:]), "default namespace": defaultNameSpace}).Debug(ModuleName)

	repository := strings.Split(name, ":")
	if len(repository) < 2 {
		repository = append(repository, "latest")
	}

	// 如果镜像中不包含/,例如 mysql:latest,此时将此镜像添加到默认命名空间中
	if !strings.Contains(repository[0], "/") {
		repository[0] = defaultNameSpace + "/" + repository[0]
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

	_, newName := reverRepositoryName(AliDockerRepository, true)
	info := Repository{
		Namespace: defaultNameSpace,
		Repositorys: map[string]Info{newName: Info{
			User:   user,
			Passwd: passwd,
		}},
	}

	err = saveRepositoryInfo(info)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	noStop := true
	// pushInfo 保存当前的上传进度信息
	pushInfo := make(map[string]string)
	// idxOut 构建输出序列
	var idxOut []string
	// buildIdx 是否需要构建序列
	buildIdx := true

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
								pushInfo[pr.Id] = fmt.Sprintf("%s %s", tm.Color(pr.Status, tm.BLUE), tm.Color(pr.Progress, tm.RED))
							}
						}

						if buildIdx {
							// header不需要保存
							for key, _ := range pushInfo {
								if key != "" {
									idxOut = append(idxOut, key)
								}
							}
							buildIdx = false
						}

						fmt.Println(tm.Color(pushInfo[""], tm.BLACK))
						delete(pushInfo, "")

						// for key, value := range pushInfo {
						// 	fmt.Printf("[%s] %s\n", tm.Color(key, tm.GREEN), value)
						// }
						for _, key := range idxOut {
							fmt.Printf("[%s] %s\n", tm.Color(key, tm.GREEN), pushInfo[key])
						}
					}
				}

				// Header占了一行,此时游标也占据了一行,因此需要在map元素个数的基础之上在上移2行
				tm.MoveCursorUp(len(pushInfo) + 2)
				time.Sleep(500 * time.Millisecond)
				tm.Flush()
			} else {
				fmt.Println(buf.String())
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
	if strings.Contains(buf.String(), "error") {
		type E struct {
			Error string `json:"error"`
		}
		ts := strings.Split(buf.String(), "\n")
		for _, t := range ts {
			if strings.Contains(t, "error") {
				var e E
				json.Unmarshal([]byte(t), &e)
				fmt.Println(e.Error)
			}
		}
	} else {
		fmt.Printf("O! [%s] Push Succ.\n", aliName)
	}

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

// saveRepositoryInfo 保存仓库用户名和口令信息
func saveRepositoryInfo(info Repository) error {
	_, err := os.Open(DPUSHCONF)
	if err != nil {
		_, err := os.Create(DPUSHCONF)
		if err != nil {
			return err
		}
	}

	data, err := toml.Marshal(info)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(DPUSHCONF, data, 0755)
}

// getRepositoryInfo 读取保存的仓库用户信息
func getRepositoryInfo() (*Repository, error) {

	var info Repository
	data, err := ioutil.ReadFile(DPUSHCONF)
	if err != nil {
		if os.IsNotExist(err){
			return nil, errors.New("Maybe is you first time use Dpush~ Please type userID and password! Use -h for help")
		}
		return nil, err
	}

	err = toml.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}

	if defaultNameSpace == "" {
		defaultNameSpace = info.Namespace
	}
	return &info, nil
}

// reverRepositoryName 仓库名称反转
// 如果forword 为true,则将registry.cn-beijing.aliyuncs.com反转成registry#cn-beijing#aliyuncs#com
// 反之则还原为registry.cn-beijing.aliyuncs.com
// 为了保持在Toml文件中的结构,在字段名前后添加了""
func reverRepositoryName(name string, forword bool) (string, string) {
	newName := ""
	if forword {
		newName = strings.Replace(name, ".", "#", -1)
	} else {
		newName = strings.Replace(name, "#", ".", -1)
	}

	return newName, fmt.Sprintf("\"%s\"", newName)
}
