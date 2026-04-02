/*********************************************************************
* Copyright (c) Intel Corporation 2023
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

ALTER TABLE devices DROP COLUMN mebxpassword;
ALTER TABLE devices DROP COLUMN mpspassword;

ALTER TABLE ciraconfigs DROP COLUMN generate_random_password;