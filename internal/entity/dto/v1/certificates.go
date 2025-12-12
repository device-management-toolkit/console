/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package dto

type CertInfo struct {
	Cert      string `json:"cert" binding:"required" example:"-----BEGIN CERTIFICATE-----\n..."`
	IsTrusted bool   `json:"isTrusted" example:"true"`
}

type DeleteCertificateRequest struct {
	InstanceID string `json:"instanceID" binding:"required" example:"cert-instance-123"`
}
