// Copyright 2025 The Wireflow Authors, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package management

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"wireflow/internal/log"
	"wireflow/management/server"

	"golang.org/x/sync/errgroup"
)

func Start(listen string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := log.GetLogger("management")
	hs, err := server.NewServer(&server.ServerConfig{
		Listen: listen,
	})

	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	// 2. 启动 Wireflow Controller (大脑)
	g.Go(func() error {
		logger.Info("Starting Wireflow Controller...")

		go func() {
			<-ctx.Done()
			//
			hs.GetManager().GetHTTPClient().CloseIdleConnections()
		}()

		return hs.Start(ctx)
	})

	// 3. 启动 Gin API Server (指挥官)
	g.Go(func() error {
		logger.Info("Starting API Server on :8080...")

		// 优雅关闭 Web 服务
		go func() {
			<-ctx.Done()
		}()

		return hs.Run(":8080")
	})

	// return controller.Run(ctx, client, natsConn)
	if !hs.GetManager().GetCache().WaitForCacheSync(ctx) {
		return fmt.Errorf("failed to wait for cache sync")
	}

	logger.Info("management server started, cache sync successfully...")
	// 等待所有组件运行或退出
	if err := g.Wait(); err != nil {
		logger.Error("Wireflow 服务异常退出: %v", err)
	}

	<-ctx.Done()
	return nil
}
