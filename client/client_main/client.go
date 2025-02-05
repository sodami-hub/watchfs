// 클라이언트 테스트를 위한 main 함수

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	api "github.com/sodami-hub/watchfs/api/v1"
	"github.com/sodami-hub/watchfs/client/garage"
	"google.golang.org/protobuf/proto"
)

var childProcess *os.Process

func main() {
	args := os.Args[1:]
	userInfo := &api.UserInfo{}
	var hasUserInfo bool
	f, err := os.OpenFile(".garage/.user", os.O_RDWR, 0644)
	defer func() {
		_ = f.Close()
	}()
	if err != nil {
		if err == os.ErrNotExist {
			hasUserInfo = false
		}
	} else {
		hasUserInfo = true
		b := make([]byte, 1024)
		n, err := f.Read(b)
		if err != nil {
			fmt.Println(err)
			return
		}
		proto.Unmarshal(b[:n], userInfo)
	}

	switch args[0] {
	case "conn":
		if hasUserInfo {
			if len(args) != 1 {
				return
			}
			file, err := os.OpenFile(".garage/.user", os.O_RDWR|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = StartWatch(file, userInfo, false)
			if err != nil {
				fmt.Println(err)
				return
			}
			_ = file.Close()
		} else {
			if len(args) != 3 {
				return
			}
			err = garage.GarageConn(args[1], args[2])
			if err != nil {
				fmt.Println(err)
				return
			}

		}
	case "init":
		err = garage.GarageInit(args[1])
		if err != nil {
			fmt.Println(err)
			return
		}
	case "start":
		err = garage.GarageWatch(userInfo)
		if err != nil {
			fmt.Println(err)
			return
		}
	case "stop":
		err := StopProc(int(userInfo.ChildProcessPid))
		if err != nil {
			fmt.Println(err)
			return
		}
	case "changes":
		err = garage.ChangeFile()
		if err != nil {
			fmt.Println(err)
			return
		}
	case "all":
		err = garage.All()
		if err != nil {
			fmt.Println(err)
			return
		}
	case "save": // 로컬의 변경사항을 리모트에 저장하기 위해서 변경 내용을 저장(commit)

		err = garage.Save(args[1])
		if err != nil {
			fmt.Println(err)
			return
		}

		err = StopProc(int(userInfo.ChildProcessPid))
		if err != nil {
			fmt.Println(err)
			return
		}

		err = os.Remove(".garage/clientFS")
		if err != nil {
			fmt.Println(err)
			return
		}

		file, err := os.OpenFile(".garage/.user", os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = StartWatch(file, userInfo, true)
		if err != nil {
			fmt.Println(err)
			return
		}
		_ = file.Close()
	case "history":
		err = garage.ShowHistory()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func StopProc(pgid int) error {
	if pgid != 0 {
		pgid := -pgid // 생성된 프로세스를 음수로 바꿔서 그룹 전체에 시그널을 보냄
		err := syscall.Kill(pgid, syscall.SIGTERM)
		if err != nil {
			fmt.Println("Failed to stop child process: ", err)
			return err
		} else {
			fmt.Println("Child process stopped")
		}
	} else {
		fmt.Println("No child process to stop")
	}
	return nil
}

func StartWatch(file *os.File, userInfo *api.UserInfo, flag bool) error {

	// 설정파일이 있고 garage start 명령을 입력하면 자식 쉘에서 감시를 시작한다.
	cmd := exec.Command("go", "run", "client.go", "start")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // 새로운 프로세스 그룹 생성
	err := cmd.Start()
	if err != nil {
		return err
	}
	childProcess = cmd.Process
	fmt.Printf("Started child process with PID %d\n", childProcess.Pid)
	userInfo.ChildProcessPid = int32(childProcess.Pid)
	b, err := proto.Marshal(userInfo)
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
