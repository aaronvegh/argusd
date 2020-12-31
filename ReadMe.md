#argusd

_argusd_ is a daemon process for Linux that provides interaction with [Argus](https://argus-app.net). In order to use Argus, you must install this daemon. *You do not need to build this yourself*: I am making this repository available and opening the source for informational purposes only. You can manage the installation of argusd from within Argus.

argusd is written in Go, and is designed for Linux systems that use systemd. 

### Building and Running
You can pull this repo and run it yourself! The `build.sh` file contains the build instructions, but you can quickly get a copy running on a Linux system by running this from the base repo directory:

env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o argusd-linux-amd64

Substitute OS and Architecture declarations as needed.

### Feedback and Pull Requests
Pull requests and issues are welcome. However, please note that this daemon is designed to operate with the Argus application; some changes may not be feasible. If you want to try something, please get in touch first by filing an Issue.

### License
argusd is being offered with an MIT license. 