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

	"github.com/matrix-org/dendrite/setup/config"
	"github.com/matrix-org/dendrite/setup/jetstream"
	"github.com/matrix-org/dendrite/setup/process"
	"github.com/matrix-org/dendrite/userapi/storage"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
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
	roomID := msg.Header.Get(jetstream.RoomID)
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
