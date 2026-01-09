/*********************************************************************
* Copyright (c) Intel Corporation 2023
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

ALTER TABLE devices ADD COLUMN mebxpassword TEXT;
ALTER TABLE devices ADD COLUMN mpspassword TEXT;

ALTER TABLE ciraconfigs ADD COLUMN generate_random_password BOOLEAN NOT NULL DEFAULT FALSE;