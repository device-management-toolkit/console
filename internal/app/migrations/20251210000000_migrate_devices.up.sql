/*********************************************************************
* Copyright (c) Intel Corporation 2023
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

ALTER TABLE devices ADD COLUMN IF NOT EXISTS mebxpassword TEXT;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS mpspassword TEXT;

ALTER TABLE ciraconfigs ADD COLUMN IF NOT EXISTS generate_random_password TEXT;