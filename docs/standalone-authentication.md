# Standalone administrator password management

When standalone authentication is enabled, an authenticated administrator can
open `/account/password` in the Console browser to change their password. The
form requires the current password and a matching new-password confirmation.
Passwords must be at least eight characters long and contain at least four
distinct characters.

The form submits to `POST /api/v1/authorize/password`. The endpoint requires an
authenticated session and independently verifies the current password. In
`keyring` mode, the new password is stored as a PBKDF2-HMAC-SHA256 hash in the
`dmt-console` keyring service. In `config` mode, it is saved to the configured
Console config file.

If `AUTH_ADMIN_PASSWORD` is set, the endpoint rejects changes with HTTP 409.
Update the password in the deployment environment and restart Console instead;
an API change to the config file would otherwise be overridden on restart.

## Forgotten keyring password

Console does not provide an unauthenticated password-reset link or a reset
command. To recover access:

1. Stop Console.
2. Using the operating system's keyring/credential manager, locate the
   `dmt-console` service and delete **both** `admin-username` and
   `admin-password-hash` entries. Do not delete only one entry.
3. Restart Console with `AUTH_SECRET_BACKEND=keyring`.
4. When both entries are absent, Console prompts for a new administrator
   username, password, and password confirmation.

The exact keyring manager and interface vary by operating system. On Linux,
use the Secret Service-compatible keyring manager; on macOS, Keychain Access;
on Windows, Credential Manager. Ensure the entries belong to the Console
`dmt-console` service before removing them.
