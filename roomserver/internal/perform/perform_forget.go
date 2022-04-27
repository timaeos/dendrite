// Copyright 2020 The Matrix.org Foundation C.I.C.
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

package perform

import (
	"context"
	"fmt"

	"github.com/matrix-org/dendrite/roomserver/api"
	"github.com/matrix-org/dendrite/roomserver/internal/input"
	"github.com/matrix-org/dendrite/roomserver/storage"
)

type Forgetter struct {
	DB      storage.Database
	Inputer *input.Inputer
}

// PerformForget implements api.RoomServerQueryAPI
func (f *Forgetter) PerformForget(
	ctx context.Context,
	request *api.PerformForgetRequest,
	response *api.PerformForgetResponse,
) error {
	if err := f.DB.ForgetRoom(ctx, request.UserID, request.RoomID, true); err != nil {
		return fmt.Errorf("f.DB.ForgetRoom: %w", err)
	}
	return f.Inputer.WriteOutputEvents(request.RoomID, []api.OutputEvent{
		{
			Type: api.OutputForgetRoomEvent,
			ForgetRoom: &api.OutputForgetRoom{
				RoomID: request.RoomID,
				UserID: request.UserID,
			},
		},
	})
}
