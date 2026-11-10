# MikroTik and The Dude through RouterOS API

The MikroTik dashboard connects to the RouterOS API and reads The Dude inventory. It also manages local Hotspot users, active Hotspot sessions, profiles, and IP bindings. Hotspot operations use `/ip/hotspot` and do not enable or require RADIUS.

## Router setup

Enable the RouterOS `api` service and use the address of the Dude server as the CRM **Host**. In the WinBox screenshots, the server is reached at `103.148.18.248`, and the `api` service listens on port `8783`. The `Available From` field is a source-IP allowlist, not the address to enter as the host: allow the backend server's source IP there and in the firewall.

Use a dedicated RouterOS account with only the required policies. The Dude commands require `dude`; automatic status checks require `test`; listing Hotspot resources requires `read`; and Hotspot changes require `write`. Do not grant `write` if the account should be read-only. The plain `api` service is unencrypted; use a VPN/private network, or configure `api-ssl` with a certificate trusted by the backend and set `use_tls` to `true`. Do not expose plain API to the public internet. The CRM browser should use HTTPS when submitting RouterOS credentials.

The backend server, not the browser, opens the RouterOS connection. It therefore needs network access to the router API port. At startup, the backend uses `MIKROTIK_HOST`, `MIKROTIK_USERNAME`, and `MIKROTIK_PASSWORD`, and connects to `MIKROTIK_API_PORT` (default `8783`). Set `MIKROTIK_API_USE_TLS=true` only when connecting to the `api-ssl` service. The old `MIKROTIK_PORT` setting is not used for startup auto-connect. The API connection and credentials remain in backend memory and are cleared when disconnected or when the backend restarts. If a RouterOS command times out and its socket is closed, the next request reconnects using the in-memory credentials; the failed command itself is not automatically replayed.

## Connect

```http
POST /api/admin/mikrotik/routeros/connect
Authorization: Bearer <CRM token for SUPERADMIN>
Content-Type: application/json
```

```json
{
  "host": "103.148.18.248",
  "port": 8783,
  "username": "crm-monitor",
  "password": "router-api-password",
  "use_tls": false
}
```

Set `use_tls` to `true` only for the `api-ssl` service. Monitoring routes:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/admin/mikrotik/routeros/status` | API connection status |
| `GET` | `/api/admin/mikrotik/routeros/dude/devices` | Devices monitored by The Dude |
| `GET` | `/api/admin/mikrotik/routeros/dude/services` | Probes/services monitored by The Dude |
| `POST` | `/api/admin/mikrotik/routeros/disconnect` | Disconnect from API (SUPERADMIN) |
| `GET` | `/api/admin/mikrotik/routeros/hotspot/users` | List local Hotspot users (passwords are not returned) |
| `GET` | `/api/admin/mikrotik/routeros/hotspot/profiles` | List local Hotspot user profiles |
| `GET` | `/api/admin/mikrotik/routeros/hotspot/active` | List active Hotspot sessions |
| `POST` | `/api/admin/mikrotik/routeros/hotspot/users` | Create a local Hotspot user (ADMIN/FINANCE) |
| `PUT` | `/api/admin/mikrotik/routeros/hotspot/users/:name` | Update a user, including disable/enable (ADMIN/FINANCE) |
| `DELETE` | `/api/admin/mikrotik/routeros/hotspot/users/:name` | Delete a user (ADMIN) |
| `POST` | `/api/admin/mikrotik/routeros/hotspot/active/:name/kick` | Disconnect active sessions for a user (ADMIN/FINANCE) |

The Dude list requests return all properties exposed by RouterOS for each device and service. The dashboard derives a device's aggregate status from its associated probe statuses when the device record does not include a status property. RouterOS versions or Dude configurations that do not expose probe status properties will show the device as `UNKNOWN`.

The **Perangkat The Dude** tab resolves a device IP against DHCP leases, ARP, and Hotspot IP Bindings. Isolate/restore is enabled only when the IP resolves to one MAC and one existing binding. Isolation sets that binding to RouterOS `type=blocked`; CRM records the previous `regular` or `bypassed` type in the binding comment and restores it on request. Bindings already blocked without CRM's marker cannot be restored by this action. This blocks access through Hotspot, not traffic outside Hotspot or a general firewall rule. The API account needs `read` access to DHCP leases, ARP, and Hotspot bindings, and `write` access for the action.

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/admin/mikrotik/routeros/hotspot/device-bindings/resolve` | Resolve device IPs to unambiguous Hotspot bindings (up to 200 addresses) |
| `POST` | `/api/admin/mikrotik/routeros/hotspot/device-bindings/set-type` | Isolate one device with `type=blocked`, or restore its recorded original type (ADMIN/FINANCE) |

SUPERADMIN can connect/disconnect. SUPERADMIN and CUSTOMER_SERVICE can read monitoring data and Hotspot state. SUPERADMIN, ADMIN, and FINANCE can perform Hotspot actions according to the route roles. Disabling/deleting a Hotspot user, disconnecting sessions, and changing IP Binding are live router actions; verify the selected account/device before confirming.
