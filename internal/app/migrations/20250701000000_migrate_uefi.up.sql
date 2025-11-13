/*********************************************************************
* Copyright (c) Intel Corporation 2023
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

ALTER TABLE profiles ADD COLUMN uefi_wifi_sync_enabled BOOLEAN NOT NULL DEFAULT FALSE;