// Generated by tmpl
// https://github.com/benbjohnson/tmpl
//
// DO NOT EDIT!
// Source: message_types.gen.go.tmpl

/*
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.

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

package message

import (
	"github.com/openGemini/openGemini/engine/executor/spdy/transport"
)

const (
	UnknownMessage uint8 = iota + 1

	PingRequestMessage
	PingResponseMessage

	PeersRequestMessage
	PeersResponseMessage

	CreateNodeRequestMessage
	CreateNodeResponseMessage

	SnapshotRequestMessage
	SnapshotResponseMessage

	ExecuteRequestMessage
	ExecuteResponseMessage

	UpdateRequestMessage
	UpdateResponseMessage

	ReportRequestMessage
	ReportResponseMessage

	GetShardInfoRequestMessage
	GetShardInfoResponseMessage

	GetDownSampleInfoRequestMessage
	GetDownSampleInfoResponseMessage

	GetRpMstInfosRequestMessage
	GetRpMstInfosResponseMessage

	GetUserInfoRequestMessage
	GetUserInfoResponseMessage

	GetStreamInfoRequestMessage
	GetStreamInfoResponseMessage

	GetMeasurementInfoRequestMessage
	GetMeasurementInfoResponseMessage

	Sql2MetaHeartbeatRequestMessage
	Sql2MetaHeartbeatResponseMessage

	GetContinuousQueryLeaseRequestMessage
	GetContinuousQueryLeaseResponseMessage
)

func NewMessage(typ uint8) transport.Codec {
	switch typ {
	case PingRequestMessage:
		return &PingRequest{}
	case PingResponseMessage:
		return &PingResponse{}
	case PeersRequestMessage:
		return &PeersRequest{}
	case PeersResponseMessage:
		return &PeersResponse{}
	case CreateNodeRequestMessage:
		return &CreateNodeRequest{}
	case CreateNodeResponseMessage:
		return &CreateNodeResponse{}
	case SnapshotRequestMessage:
		return &SnapshotRequest{}
	case SnapshotResponseMessage:
		return &SnapshotResponse{}
	case ExecuteRequestMessage:
		return &ExecuteRequest{}
	case ExecuteResponseMessage:
		return &ExecuteResponse{}
	case UpdateRequestMessage:
		return &UpdateRequest{}
	case UpdateResponseMessage:
		return &UpdateResponse{}
	case ReportRequestMessage:
		return &ReportRequest{}
	case ReportResponseMessage:
		return &ReportResponse{}
	case GetShardInfoRequestMessage:
		return &GetShardInfoRequest{}
	case GetShardInfoResponseMessage:
		return &GetShardInfoResponse{}
	case GetDownSampleInfoRequestMessage:
		return &GetDownSampleInfoRequest{}
	case GetDownSampleInfoResponseMessage:
		return &GetDownSampleInfoResponse{}
	case GetRpMstInfosRequestMessage:
		return &GetRpMstInfosRequest{}
	case GetRpMstInfosResponseMessage:
		return &GetRpMstInfosResponse{}
	case GetUserInfoRequestMessage:
		return &GetUserInfoRequest{}
	case GetUserInfoResponseMessage:
		return &GetUserInfoResponse{}
	case GetStreamInfoRequestMessage:
		return &GetStreamInfoRequest{}
	case GetStreamInfoResponseMessage:
		return &GetStreamInfoResponse{}
	case GetMeasurementInfoRequestMessage:
		return &GetMeasurementInfoRequest{}
	case GetMeasurementInfoResponseMessage:
		return &GetMeasurementInfoResponse{}
	case Sql2MetaHeartbeatRequestMessage:
		return &Sql2MetaHeartbeatRequest{}
	case Sql2MetaHeartbeatResponseMessage:
		return &Sql2MetaHeartbeatResponse{}
	case GetContinuousQueryLeaseRequestMessage:
		return &GetContinuousQueryLeaseRequest{}
	case GetContinuousQueryLeaseResponseMessage:
		return &GetContinuousQueryLeaseResponse{}
	default:
		return nil
	}
}

func GetResponseMessageType(typ uint8) uint8 {
	switch typ {
	case PingRequestMessage:
		return PingResponseMessage
	case PeersRequestMessage:
		return PeersResponseMessage
	case CreateNodeRequestMessage:
		return CreateNodeResponseMessage
	case SnapshotRequestMessage:
		return SnapshotResponseMessage
	case ExecuteRequestMessage:
		return ExecuteResponseMessage
	case UpdateRequestMessage:
		return UpdateResponseMessage
	case ReportRequestMessage:
		return ReportResponseMessage
	case GetShardInfoRequestMessage:
		return GetShardInfoResponseMessage
	case GetDownSampleInfoRequestMessage:
		return GetDownSampleInfoResponseMessage
	case GetRpMstInfosRequestMessage:
		return GetRpMstInfosResponseMessage
	case GetUserInfoRequestMessage:
		return GetUserInfoResponseMessage
	case GetStreamInfoRequestMessage:
		return GetStreamInfoResponseMessage
	case GetMeasurementInfoRequestMessage:
		return GetMeasurementInfoResponseMessage
	case Sql2MetaHeartbeatRequestMessage:
		return Sql2MetaHeartbeatResponseMessage
	case GetContinuousQueryLeaseRequestMessage:
		return GetContinuousQueryLeaseResponseMessage
	default:
		return UnknownMessage
	}
}
