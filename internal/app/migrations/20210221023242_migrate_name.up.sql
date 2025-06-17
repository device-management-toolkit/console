/*********************************************************************
* Copyright (c) Intel Corporation 2023
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/
CREATE TABLE IF NOT EXISTS devices
(
    guid TEXT NOT NULL,
    tags TEXT,
    hostname TEXT,
    mpsinstance TEXT,
    connectionstatus BOOLEAN NOT NULL,
    mpsusername TEXT,
    tenantid TEXT NOT NULL,
    friendlyname TEXT,
    dnssuffix TEXT,
    lastconnected TEXT,
    lastseen TEXT,
    lastdisconnected TEXT,
    deviceinfo TEXT,
    username TEXT,
    password TEXT,
    usetls BOOLEAN NOT NULL,
    allowselfsigned BOOLEAN NOT NULL,
    certhash TEXT,
    PRIMARY KEY (guid, tenantid),
    UNIQUE (guid)
);
CREATE TABLE IF NOT EXISTS ciraconfigs(
  cira_config_name TEXT NOT NULL,
  mps_server_address TEXT,
  mps_port INTEGER,
  user_name TEXT,
  password TEXT,
  common_name TEXT,
  server_address_format INTEGER,
  auth_method INTEGER,
  mps_root_certificate TEXT,
  proxydetails TEXT,
  tenant_id TEXT NOT NULL,
  PRIMARY KEY (cira_config_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS ieee8021xconfigs(
  profile_name TEXT,
  auth_protocol INTEGER,
  pxe_timeout INTEGER,
  wired_interface BOOLEAN NOT NULL,
  tenant_id TEXT NOT NULL,
  PRIMARY KEY (profile_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS wirelessconfigs(
  wireless_profile_name TEXT NOT NULL,
  authentication_method INTEGER,
  encryption_method INTEGER,
  ssid TEXT,
  psk_value INTEGER,
  psk_passphrase TEXT,
  link_policy TEXT,
  creation_date TEXT, -- TIMESTAMP is usually represented as TEXT in SQLite
  created_by TEXT,
  tenant_id TEXT NOT NULL,
  ieee8021x_profile_name TEXT,
  FOREIGN KEY (ieee8021x_profile_name, tenant_id) REFERENCES ieee8021xconfigs(profile_name, tenant_id),
  PRIMARY KEY (wireless_profile_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS profiles(
  profile_name TEXT NOT NULL,
  activation TEXT NOT NULL,
  amt_password TEXT,
  generate_random_password BOOLEAN NOT NULL,
  cira_config_name TEXT,
  creation_date TEXT, -- TIMESTAMP as TEXT
  created_by TEXT,
  mebx_password TEXT,
  generate_random_mebx_password BOOLEAN NOT NULL, 
  tags TEXT,
  dhcp_enabled BOOLEAN NOT NULL, 
  tenant_id TEXT NOT NULL,
  tls_mode INTEGER,
  user_consent TEXT,
  ider_enabled BOOLEAN NOT NULL, 
  kvm_enabled BOOLEAN NOT NULL, 
  sol_enabled BOOLEAN NOT NULL, 
  tls_signing_authority TEXT,
  ip_sync_enabled BOOLEAN NOT NULL, 
  local_wifi_sync_enabled BOOLEAN NOT NULL, 
  ieee8021x_profile_name TEXT,
  uefi_wifi_sync_enabled BOOLEAN NOT NULL,
  FOREIGN KEY (ieee8021x_profile_name, tenant_id) REFERENCES ieee8021xconfigs(profile_name, tenant_id),
  FOREIGN KEY (cira_config_name, tenant_id) REFERENCES ciraconfigs(cira_config_name, tenant_id),
  PRIMARY KEY (profile_name, tenant_id)
);

CREATE TABLE IF NOT EXISTS profiles_wirelessconfigs(
  wireless_profile_name TEXT,
  profile_name TEXT,
  priority INTEGER,
  creation_date TEXT, -- TIMESTAMP as TEXT
  created_by TEXT,
  tenant_id TEXT NOT NULL,
  FOREIGN KEY (wireless_profile_name, tenant_id) REFERENCES wirelessconfigs(wireless_profile_name, tenant_id),
  FOREIGN KEY (profile_name, tenant_id) REFERENCES profiles(profile_name, tenant_id),
  PRIMARY KEY (wireless_profile_name, profile_name, priority, tenant_id)
);

CREATE TABLE IF NOT EXISTS domains(
  name TEXT NOT NULL,
  domain_suffix TEXT NOT NULL,
  provisioning_cert TEXT,
  provisioning_cert_storage_format TEXT,
  provisioning_cert_key TEXT,
  expiration_date TEXT,
  creation_date TEXT, -- TIMESTAMP as TEXT
  created_by TEXT,
  tenant_id TEXT NOT NULL,
  CONSTRAINT domainsuffix UNIQUE (domain_suffix, tenant_id),
  PRIMARY KEY (name, tenant_id)
);

CREATE UNIQUE INDEX lower_name_suffix_idx ON domains (LOWER(name), LOWER(domain_suffix));
