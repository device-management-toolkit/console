#!/bin/bash
# Script to add test device data to the database for Redfish API tests

DB_PATH="${DB_PATH:-$HOME/.config/device-management-toolkit/console.db}"

echo "Adding test device to database: $DB_PATH"

sqlite3 "$DB_PATH" <<EOF
-- Create test device if it doesn't exist
INSERT OR IGNORE INTO devices (
    guid, 
    hostname, 
    tags, 
    mpsinstance, 
    connectionstatus, 
    mpsusername, 
    tenantid, 
    friendlyname, 
    dnssuffix, 
    deviceinfo
) VALUES (
    'test-system-1',
    'test-device-1.local',
    '[]',
    '',
    true,
    'admin',
    '',
    'Test System 1',
    'local',
    '{"BuildNumber":"3000","Certificates":null,"FWBuild":"1234","FWNonce":"ABC123","FWVersion":"16.0.0","FQDN":"test-device-1.local","IPConfiguration":{"AddressOrigin":"DHCP","DHCPEnabled":true,"IPAddress":"192.168.1.100","MACAddress":"00:11:22:33:44:55","PrimaryDNS":"8.8.8.8","SecondaryDNS":"8.8.4.4","SubnetMask":"255.255.255.0"},"Mode":"","OS":"Windows 10","RASRemoteStatus":"Not Connected","SKU":"vPro","UUID":"12345678-1234-1234-1234-123456789012","features":{"AMT":true,"KVM":true,"SOL":true,"IDER":true,"Redirection":true}}'
);

-- Verify the insert
SELECT 'Test device added/verified:' as status, guid, hostname, friendlyname FROM devices WHERE guid = 'test-system-1';
EOF

echo ""
echo "Test data setup complete!"
echo "System ID: test-system-1"
