/*********************************************************************
* Copyright (c) Intel Corporation 2026
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

-- Device identity & lifecycle columns (issues #843, #1382). TEXT cols are NOT NULL
-- DEFAULT '' so the ALTER backfills existing rows (modernc sqlite can't scan
-- NULL into a Go string); the partial index on id skips those empty backfills.
-- lastupdated is refreshed only on the main Update path, never by heartbeats,
-- and is intentionally unindexed. producttype: vPro/ISM. connectiontype is CIRA
-- or Direct; the use case derives it from the MPS credentials when a client
-- does not supply it.
ALTER TABLE devices ADD COLUMN id TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN createddate TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN lastupdated TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN producttype TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN connectiontype TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_id ON devices (id) WHERE id <> '';
