/*********************************************************************
* Copyright (c) Intel Corporation 2026
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

-- Device identity & lifecycle columns (issue #843). TEXT cols are NOT NULL
-- DEFAULT '' so the ALTER backfills existing rows (modernc sqlite can't scan
-- NULL into a Go string); the partial index on id skips those empty backfills.
-- lastupdate is refreshed only on the main Update path, never by heartbeats,
-- and is intentionally unindexed. isdeleted/deleteddate: plumbing only (soft-
-- delete lands in a separate PR). producttype: vPro/ISM. connectiontype: CIRA/Direct.
ALTER TABLE devices ADD COLUMN id TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN createddate TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN lastupdate TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN isdeleted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE devices ADD COLUMN deleteddate TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN producttype TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN connectiontype TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_id ON devices (id) WHERE id <> '';
