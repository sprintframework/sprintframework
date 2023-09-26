/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcore

var DefaultCoreServices = []interface{} {
	ZapLogFactory(),
	HCLogFactory(),
	NodeService(),
	ConfigRepository(10000),
	JobService(),
	StorageService(),
	MailService(),
}
