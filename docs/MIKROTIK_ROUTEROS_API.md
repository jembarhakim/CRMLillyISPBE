# The Dude monitoring through RouterOS API

The MikroTik dashboard connects to the RouterOS API on the machine running The Dude and reads the device and service inventory from The Dude. It does not manage PPPoE, Hotspot, or voucher accounts.

## Router setup

Enable the RouterOS `api` service and use the address of the Dude server as the CRM **Host**. In the WinBox screenshots, the server is reached at `103.148.18.248`, and the `api` service listens on port `8783`. The `Available From` field is a source-IP allowlist, not the address to enter as the host: allow the backend server's source IP there and in the firewall.

Use a dedicated RouterOS account with only `api`, `read`, and `dude` policies. The Dude commands require the `dude` policy. The plain `api` service is unencrypted; use a VPN/private network, or configure `api-ssl` with a certificate trusted by the backend and set `use_tls` to `true`. Do not expose plain API to the public internet. The CRM browser should use HTTPS when submitting RouterOS credentials.

The backend server, not the browser, opens the RouterOS connection. It therefore needs network access to the router API port. The API connection and credentials remain in backend memory and are cleared when disconnected or when the backend restarts.

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

The Dude list requests return all properties exposed by RouterOS for each device and service. The dashboard derives a device's aggregate status from its associated probe statuses when the device record does not include a status property. RouterOS versions or Dude configurations that do not expose probe status properties will show the device as `UNKNOWN`.

SUPERADMIN can connect/disconnect. SUPERADMIN and CUSTOMER_SERVICE can read monitoring data.
