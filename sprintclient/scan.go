/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintclient

var ControlClientBeans = []interface{} {
	GrpcClientFactory("control-grpc-client"),
	ControlClient(),
}

