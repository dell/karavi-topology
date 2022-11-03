/*
 Copyright (c) 2020-2022 Dell Inc. or its subsidiaries. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package entrypoint_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dell/karavi-topology/internal/entrypoint"
	"github.com/dell/karavi-topology/internal/entrypoint/mocks"

	"github.com/golang/mock/gomock"
)

func Test_Run(t *testing.T) {
	tests := map[string]func(t *testing.T) (expectError bool, svc entrypoint.ServiceRunner, ctrl *gomock.Controller){

		"success": func(*testing.T) (bool, entrypoint.ServiceRunner, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			svc := mocks.NewMockServiceRunner(ctrl)
			svc.EXPECT().Run().Times(1).Return(nil)
			return false, svc, ctrl
		},
		"error calling Run": func(*testing.T) (bool, entrypoint.ServiceRunner, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			svc := mocks.NewMockServiceRunner(ctrl)
			svc.EXPECT().Run().Times(1).Return(errors.New("error"))
			return true, svc, ctrl
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			expectError, svc, ctrl := test(t)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()
			err := entrypoint.Run(ctx, svc)
			if err != nil && !expectError {
				t.Errorf("nil error expected but it was %v", err)
			}
			if err == nil && expectError {
				t.Errorf("error was expected but received nil error")
			}
			ctrl.Finish()
		})
	}
}
