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

package consumers

import (
	"context"
	"encoding/json"

	"github.com/matrix-org/dendrite/setup/config"
	"github.com/matrix-org/dendrite/setup/jetstream"
	"github.com/matrix-org/dendrite/setup/process"
	"github.com/matrix-org/dendrite/userapi/storage"
	"github.com/matrix-org/gomatrixserverlib"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type InputRoomForgetConsumer struct {
	ctx       context.Context
	cfg       *config.UserAPI
	jetstream nats.JetStreamContext
	durable   string
	db        storage.Database
	topic     string
}

func NewInputRoomForgetConsumer(
	process *process.ProcessContext,
	cfg *config.UserAPI,
	js nats.JetStreamContext,
	store storage.Database,
) *InputRoomForgetConsumer {
	return &InputRoomForgetConsumer{
		ctx:       process.Context(),
		cfg:       cfg,
		jetstream: js,
		topic:     cfg.Matrix.JetStream.TopicFor(jetstream.InputRoomForget),
		durable:   cfg.Matrix.JetStream.Durable("UserAPIRoomserverForgetRooomConsumer"),
		db:        store,
	}
}

func (s *InputRoomForgetConsumer) Start() error {
	return jetstream.JetStreamConsumer(
		s.ctx, s.jetstream, s.topic, s.durable, s.onMessage,
		nats.DeliverAll(), nats.ManualAck(),
	)
}

func (s *InputRoomForgetConsumer) onMessage(ctx context.Context, msg *nats.Msg) bool {
	userID := msg.Header.Get(jetstream.UserID)
	roomID := msg.Header.Get(jetstream.RoomID)
	if userID != "" {
		return s.forgetUserData(ctx, userID, roomID)
	}
	if roomID == "" {
		return true
	}
	if err := s.db.PurgeRoom(ctx, roomID); err != nil {
		log.WithError(err).Error("userapi: unable to purge room")
		return true
	}
	log.WithField("roomID", roomID).Debug("userapi: Successfully purged room")
	return true
}

func (s *InputRoomForgetConsumer) forgetUserData(ctx context.Context, userID, roomID string) bool {
	localpart, _, err := gomatrixserverlib.SplitID('@', userID)
	if err != nil {
		return true
	}
	logger := log.WithField("userID", userID).WithField("roomID", roomID)
	if err := s.db.DeleteNotificationsForUser(ctx, localpart, roomID); err != nil {
		logger.WithError(err).Error("Unable to delete notifications for user")
		return true
	}
	if err := s.db.DeleteAccountDataForUser(ctx, localpart, roomID); err != nil {
		logger.WithError(err).Error("Unable to delete account data for user")
		return true
	}

	globalData, _, err := s.db.GetAccountData(ctx, localpart)
	if err != nil {
		logger.WithError(err).Error("Unable to get account data for user")
		return true
	}
	override := gjson.GetBytes(globalData["m.push_rules"], "global.override")
	newOverride := []json.RawMessage{}
	for _, v := range override.Array() {
		if v.Get("rule_id").Str != roomID {
			newOverride = append(newOverride, []byte(v.Raw))
		}
	}
	globalData["m.push_rules"], err = sjson.SetBytes(globalData["m.push_rules"], "global.override", newOverride)
	if err != nil {
		logger.WithError(err).Error("Unable to update push_rules json for user")
		return true
	}

	rooms := gjson.GetBytes(globalData["m.push_rules"], "global.room")
	newRooms := []json.RawMessage{}
	for _, v := range rooms.Array() {
		if v.Get("rule_id").Str != roomID {
			newRooms = append(newRooms, []byte(v.Raw))
		}
	}
	globalData["m.push_rules"], err = sjson.SetBytes(globalData["m.push_rules"], "global.room", newRooms)
	if err != nil {
		logger.WithError(err).Error("Unable to update push_rules json for user")
		return true
	}

	if err := s.db.SaveAccountData(ctx, localpart, "", "m.push_rules", globalData["m.push_rules"]); err != nil {
		logger.WithError(err).Error("Unable to save new push rules for user")
		return true
	}

	if data, ok := globalData["m.direct"]; ok {
		mDirect := gjson.ParseBytes(data)
		for userID, rooms := range mDirect.Map() {
			newRooms := []string{}
			var found bool
			for _, room := range rooms.Array() {
				if room.Str != roomID {
					newRooms = append(newRooms, room.Str)
				} else {
					found = true
				}
			}
			if found {
				globalData["m.direct"], err = sjson.SetBytes(globalData["m.direct"], userID, newRooms)
				if err != nil {
					logger.WithError(err).Error("Unable to update m.direct json for user")
					return true
				}
			}
		}
	}

	return true
}
