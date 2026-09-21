//go:build windows

package main

import (
	"context"
	"log"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

const serviceName = "LayaDeploy"

type layaService struct {
	run func(ctx context.Context) error
}

func isWindowsService() bool {
	is, err := svc.IsWindowsService()
	if err != nil {
		return false
	}
	return is
}

func runWindowsService(run func(ctx context.Context) error) error {
	return svc.Run(serviceName, &layaService{run: run})
}

func (s *layaService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepts = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- s.run(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: accepts}
	elog, err := eventlog.Open(serviceName)
	if err == nil {
		elog.Info(1, "laya-deploy service running")
		defer elog.Close()
	}

	for {
		select {
		case err := <-done:
			cancel()
			if err != nil {
				if elog != nil {
					elog.Error(1, err.Error())
				}
				log.Printf("service run error: %v", err)
				return true, 1
			}
			return false, 0
		case cr := <-r:
			switch cr.Cmd {
			case svc.Interrogate:
				changes <- cr.CurrentStatus
			case svc.Stop, svc.Shutdown:
				if elog != nil {
					elog.Info(1, "laya-deploy service stopping")
				}
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case <-done:
				case <-time.After(8 * time.Second):
				}
				return false, 0
			default:
				log.Printf("service unexpected control request %d", cr.Cmd)
			}
		}
	}
}
