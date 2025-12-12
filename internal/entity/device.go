/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package entity

import "time"

type Device struct {
	ConnectionStatus bool
	MPSInstance      string
	Hostname         string
	GUID             string
	MPSUsername      string
	Tags             string
	TenantID         string
	FriendlyName     string
	DNSSuffix        string
	LastConnected    *time.Time
	LastSeen         *time.Time
	LastDisconnected *time.Time
	DeviceInfo       string
	Username         string
	Password         string
	UseTLS           bool
	AllowSelfSigned  bool
	CertHash         *string
}

type Explorer struct {
	XMLInput  string
	XMLOutput string
}
