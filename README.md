# dpush
Dpush can push your image to ali cloud docker repository

## What's Dpush?
> Dpush 是一个方便用户上传镜像到阿里云镜像仓库的工具

国内的网络环境在访问docker repostiory的时候，会因为各种问题产生失败。因此无论是docker push 还是 docker pull都会很困难。
目前国内阿里云提供了`免费`的docker镜像仓库。 但这个仓库使用的是registry.cn-xxxxx.aliyuncs.com的Repository地址，每次拼写都会很麻烦。

dpush可以方便将用户指定的镜像push到阿里云仓库。

## How to use?

只有一个前提条件: `注册`一个阿里云仓库用户。然后就可以使用dpush了。

## Example!

假设目前需要上传`vikings/alpine`
```
dpush -i vikings/alpine
```

dpush会自动将上面的镜像`vikings/alpine`添加为`registry.cn-beijing.aliyuncs.com/vikings/alpine`。如果tag为空，则默认添加为latest。

上传成功后，用户可以在阿里云仓库华北2空间中查看到刚刚上传的镜像

## Note！

注意，在上传之前需要创建空间。例如上面案例中，在上传之前需要首先创建`vikings`空间。然后才能正常使用。

第一次上传时，需要通过-u <用户名> -p 提供阿里云用户名和口令。 Dpush登录成功后，会将登录信息保存在本地，以后只要登录信息不发生变化，就不在需要提供用户名和口令了。

## Usage

```
NAME:
   dpush - Push your docker image to ali docker repositry

USAGE:
   dpush [global options] command [command options] [arguments...]

VERSION:
   v0.2.0

AUTHOR:
   andy zhang

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --image value, -i value  The Docker Image
   --verbose, -V            Enable Verbose Logging
   --user value, -u value   Ali Repository User
   --passwd, -p             Ali Repository Passwd
   --help, -h               show help
   --version, -v            print the version
```

## ScreenShots

![](https://github.com/andy-zhangtao/blogpic/blob/master/dpush.gif?raw=true)