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

	"github.com/matrix-org/dendrite/roomserver/api"
	"github.com/matrix-org/dendrite/roomserver/storage"
	"github.com/matrix-org/dendrite/setup/jetstream"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Forgetter struct {
	DB        storage.Database
	JetStream nats.JetStreamContext
	Subject   string
}

// PerformForget implements api.RoomServerQueryAPI
func (f *Forgetter) PerformForget(
	ctx context.Context,
	request *api.PerformForgetRequest,
	response *api.PerformForgetResponse,
) error {
	err := f.DB.ForgetRoom(ctx, request.UserID, request.RoomID, true)
	if err != nil {
		return err
	}
	// Check if this server is still in the room, if not, purge it from the database
	info, err := f.DB.RoomInfo(ctx, request.RoomID)
	if err != nil {
		return err
	}
	inRoom, err := f.DB.GetLocalServerInRoom(ctx, info.RoomNID)
	if err != nil {
		return err
	}
	if !inRoom {
		logrus.Debugf("Sending purge room message, last local member left")
		msg := &nats.Msg{
			Subject: f.Subject,
		}
		msg.Header = make(nats.Header)
		msg.Header.Set(jetstream.RoomID, request.RoomID)
		_, err := f.JetStream.PublishMsg(msg)
		if err != nil {
			return err
		}
	}
	return nil
}
