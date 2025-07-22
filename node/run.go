package node

import (
	"fmt"
	wg "golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/wgctrl"
	"linkany/internal"
	"linkany/management/vo"
	"linkany/pkg/config"
	"linkany/pkg/log"
	"net"
	"os"
)

// Start start linkany daemon
func Start(flags *LinkFlags) error {
	var err error
	ctx := SetupSignalHandler()

	logger := log.NewLogger(log.Loglevel, "linkany")

	conf, err := config.GetLocalConfig()
	if err != nil {
		return err
	}

	engineCfg := &EngineConfig{
		Logger:        logger,
		Conf:          conf,
		Port:          51820,
		InterfaceName: flags.InterfaceName,
		WgLogger: wg.NewLogger(
			wg.LogLevelError,
			fmt.Sprintf("(%s) ", flags.InterfaceName),
		),
		ForceRelay: flags.ForceRelay,
	}

	if flags.ManagementUrl == "" {
		engineCfg.ManagementUrl = fmt.Sprintf("%s:%d", internal.ManagementDomain, internal.DefaultManagementPort)
	}

	if flags.SignalingUrl == "" {
		engineCfg.SignalingUrl = fmt.Sprintf("%s:%d", internal.SignalingDomain, internal.DefaultSignalingPort)
	}

	if flags.TurnServerUrl == "" {
		engineCfg.TurnServerUrl = fmt.Sprintf("%s:%d", internal.TurnServerDomain, internal.DefaultTurnServerPort)
	}

	if flags.DaemonGround {
		env := os.Environ()
		files := [3]*os.File{}
		//if os.Getenv("LOG_LEVEL") != "" && logLevel != device.LogLevelSilent {
		//	files[0], _ = os.Open(os.DevNull)
		//	files[1] = os.Stdout
		//	files[2] = os.Stderr
		//} else {
		files[0], _ = os.Open(os.DevNull)
		files[1], _ = os.Open(os.DevNull)
		files[2], _ = os.Open(os.DevNull)
		//}
		attr := &os.ProcAttr{
			Files: []*os.File{
				files[0], // stdin
				files[1], // stdout
				files[2], // stderr
				//tdev.File(),
				//fileUAPI,
			},
			Dir: ".",
			Env: env,
		}

		path, err := os.Executable()
		if err != nil {
			logger.Errorf("Failed to determine executable: %v", err)
			os.Exit(1)
		}

		process, err := os.StartProcess(
			path,
			os.Args,
			attr,
		)
		if err != nil {
			logger.Errorf("Failed to daemonize: %v", err)
			os.Exit(1)
		}
		process.Release()
	}

	engine, err := NewEngine(engineCfg)
	if err != nil {
		return err
	}

	engine.GetNetworkMap = func() (*vo.NetworkMap, error) {
		// get network map from list
		conf, err := engine.mgtClient.GetNetMap()
		if err != nil {
			logger.Errorf("Get network map failed: %v", err)
			return nil, err
		}

		logger.Infof("Success get net map")

		return conf, err
	}

	err = engine.Start()

	// open UAPI file
	logger.Infof("Interface name is: [%s]", engine.Name)
	fileUAPI, err := func() (*os.File, error) {
		return ipc.UAPIOpen(engine.Name)
	}()

	uapi, err := ipc.UAPIListen(engine.Name, fileUAPI)
	if err != nil {
		logger.Errorf("Failed to listen on uapi socket: %v", err)
		os.Exit(-1)
	}

	go func() {
		for {
			conn, err := uapi.Accept()
			if err != nil {
				return
			}
			go engine.IpcHandle(conn)
		}
	}()
	logger.Infof("Linkany started")

	<-ctx.Done()
	uapi.Close()

	engine.close()
	logger.Infof("Linkany shutting down")
	return err
}

// Stop stop linkany daemon
func Stop(flags *LinkFlags) error {
	interfaceName := flags.InterfaceName
	if flags.InterfaceName == "" {
		ctr, err := wgctrl.New()
		if err != nil {
			return nil
		}

		devices, err := ctr.Devices()
		if err != nil {
			return err
		}

		if len(devices) == 0 {
			return fmt.Errorf("没有找到任何 Linkany 设备")
		}

		interfaceName = devices[0].Name
	}
	// 如果 UAPI 失败，尝试通过 PID 文件停止进程
	return stopViaPIDFile(interfaceName)

}

// 通过 PID 文件停止进程
func stopViaPIDFile(interfaceName string) error {
	// 获取 socket 文件路径
	socketPath := fmt.Sprintf("/var/run/wireguard/%s.sock", interfaceName)
	// 检查 sock 文件是否存在
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return fmt.Errorf("套接字文件:%s不存在", socketPath)
	}

	// 连接到 Unix 域套接字
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("连接套接字失败: %v", err)
	}
	defer conn.Close()
	// 发送消息到服务器
	_, err = conn.Write([]byte("stop\n"))
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	// 接收响应
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("接收响应失败: %v", err)
	}

	fmt.Printf("linkany stopped: %s\n", string(buffer[:n]))

	return nil
}
