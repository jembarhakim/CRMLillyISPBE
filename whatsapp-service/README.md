# WhatsApp Service using whatsapp-web.js

This service uses [whatsapp-web.js](https://wwebjs.dev/) to send WhatsApp messages using your own WhatsApp number.

## Setup

1. **Install dependencies:**
   ```bash
   cd whatsapp-service
   npm install
   ```

2. **Start the service:**
   ```bash
   npm start
   ```

3. **Scan QR Code:**
   - When you first run the service, it will display a QR code in the terminal
   - Open WhatsApp on your phone
   - Go to Settings → Linked Devices
   - Tap "Link a Device"
   - Scan the QR code displayed in the terminal
   - The service will automatically connect and be ready to send messages

4. **Environment Variables (optional):**
   ```bash
   WHATSAPP_SERVICE_PORT=3002  # Default: 3002
   WHATSAPP_STATUS_WEBHOOK_URL=http://localhost:3001/api/webhook/wa/status
   WHATSAPP_STATUS_WEBHOOK_TOKEN=the_same_secret_as_the_backend
   ```

## API Endpoints

### Health Check
```
GET /health
```
Returns the service status and whether WhatsApp is ready.

### Send Message
```
POST /send-message
Content-Type: application/json

{
  "to": "08970833338",
  "message": "Your message here"
}
```

Response:
```json
{
  "status": "success",
  "message": "Message sent successfully",
  "data": {
    "to": "628970833338",
    "original_to": "08970833338",
    "message": "Your message here",
    "message_id": "...",
    "sent_at": "2025-01-04T10:30:00.000Z"
  }
}
```

### Get Status
```
GET /status
```
Returns whether the WhatsApp client is ready to send messages.

## Running as a Service

### Using PM2 (recommended)
```bash
pm2 start index.js --name whatsapp-service
pm2 save
```

### Using systemd (Linux)
Create `/etc/systemd/system/whatsapp-service.service`:
```ini
[Unit]
Description=WhatsApp Service
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/whatsapp-service
ExecStart=/usr/bin/node index.js
Restart=always

[Install]
WantedBy=multi-user.target
```

Then:
```bash
sudo systemctl enable whatsapp-service
sudo systemctl start whatsapp-service
```

## Notes

- The service stores authentication data in `./whatsapp-session` directory
- After the first scan, you don't need to scan again unless you log out
- Phone numbers are automatically normalized (08xxx → 628xxx)
- The service runs on port 3002 by default
- When both status callback variables are set, sent/delivered acknowledgements are posted to the backend notification queue.

## Troubleshooting

1. **QR Code not showing**: Make sure you have `qrcode-terminal` installed
2. **Connection lost**: Restart the service, it will reconnect automatically
3. **Message not sending**: Check if WhatsApp client is ready via `/health` endpoint





