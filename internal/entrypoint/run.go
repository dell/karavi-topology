package entrypoint

// Copyright (c) 2020 Dell Inc., or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0

import (
	"context"
)

// ServiceRunner is an interface of a service that can be run
//
//go:generate mockgen -destination=mocks/service_runner_mocks.go -package=mocks github.com/dell/karavi-topology/internal/entrypoint ServiceRunner
type ServiceRunner interface {
	Run() error
}

// Run is the entrypoint to starting the service
func Run(ctx context.Context, service ServiceRunner) error {

	errCh := make(chan error, 1)

	go func() {
		errCh <- service.Run()
	}()

	for {
		select {
		case err := <-errCh:
			if err == nil {
				continue
			}
			return err
		case <-ctx.Done():
			return nil
		}
	}
}
