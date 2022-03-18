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
	"errors"

	"github.com/matrix-org/dendrite/roomserver/api"
	"github.com/matrix-org/dendrite/roomserver/storage"
	"github.com/matrix-org/dendrite/setup/jetstream"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Forgetter struct {
	DB                storage.Database
	JetStream         nats.JetStreamContext
	Subject           string
	PurgeOnLastMember bool
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

	// Send forget to userapi, to remove unnecessary account data
	msg := &nats.Msg{
		Subject: f.Subject,
	}
	msg.Header = make(nats.Header)
	msg.Header.Set(jetstream.RoomID, request.RoomID)
	msg.Header.Set(jetstream.UserID, request.UserID)
	_, err = f.JetStream.PublishMsg(msg)
	if err != nil {
		return err
	}

	if f.PurgeOnLastMember {
		err := f.PurgeRoom(ctx, request.RoomID)
		if err != nil {
			if errors.Is(err, ErrLocalMembersExist) {
				logrus.WithField("roomID", request.RoomID).Warn(err)
				return nil
			}
			return err
		}
	}

	return nil
}

// ErrLocalMembersExist is an error returned from PurgeRoom.
var ErrLocalMembersExist = errors.New("There are still local members in this room.")

// PurgeRoom purges a room from the database, if all local members forgot about it.
func (f *Forgetter) PurgeRoom(ctx context.Context, roomID string) error {
	// Check if this server is still in the room, if not, purge it from the database
	info, err := f.DB.RoomInfo(ctx, roomID)
	if err != nil {
		return err
	}
	forgotten, err := f.DB.GetRoomForgotten(ctx, info.RoomNID)
	if err != nil {
		return err
	}
	if forgotten {
		logrus.WithField("roomID", roomID).Debugf("Sending purge room message")
		msg := &nats.Msg{
			Subject: f.Subject,
		}
		msg.Header = make(nats.Header)
		msg.Header.Set(jetstream.RoomID, roomID)
		_, err := f.JetStream.PublishMsg(msg)
		if err != nil {
			return err
		}
		// run in a go routine and with context.Background to avoid blocking/canceling the request
		go func() {
			if err := f.DB.PurgeRoom(context.Background(), roomID, info); err != nil {
				logrus.WithField("roomID", roomID).WithError(err).Error("roomserver: failed to purge room")
				return
			}
			logrus.WithField("roomID", roomID).Debugf("roomserver: Successfully purged room")
		}()
	} else {
		return ErrLocalMembersExist
	}
	return nil
}
