const { Client, LocalAuth } = require('whatsapp-web.js');
const qrcode = require('qrcode-terminal');
const express = require('express');
const cors = require('cors');

const app = express();
const PORT = process.env.WHATSAPP_SERVICE_PORT || 3002;

// Middleware
app.use(cors());
app.use(express.json());

// Initialize WhatsApp client with proper session handling
const client = new Client({
    authStrategy: new LocalAuth({
        dataPath: './whatsapp-session',
        clientId: 'whatsapp-client'
    }),
    puppeteer: {
        headless: true,
        args: [
            '--no-sandbox',
            '--disable-setuid-sandbox',
            '--disable-dev-shm-usage',
            '--disable-accelerated-2d-canvas',
            '--no-first-run',
            '--no-zygote',
            '--disable-gpu'
        ]
    }
});

let isReady = false;
let clientReady = false;
let qrCode = null;

// Loading previous session
client.on('loading_screen', (percent, message) => {
    console.log(`[WA] Loading: ${percent}% - ${message}`);
});

// QR Code generation
client.on('qr', (qr) => {
    console.log('\n═══════════════════════════════════════════════════════');
    console.log('[WA] QR CODE RECEIVED - SCAN WITH WHATSAPP APP');
    console.log('═══════════════════════════════════════════════════════');
    console.log('\n📱 INSTRUCTIONS:');
    console.log('1. Open WhatsApp on your PHONE (not camera app)');
    console.log('2. Go to: Settings → Linked Devices');
    console.log('3. Tap: "Link a Device"');
    console.log('4. Point your phone camera at the QR code below');
    console.log('5. Wait for "Client is ready!" message\n');
    console.log('═══════════════════════════════════════════════════════\n');
    qrcode.generate(qr, { small: true });
    console.log('\n═══════════════════════════════════════════════════════\n');
    qrCode = qr;
    isReady = false;
    clientReady = false;
});

// Authentication successful
client.on('ready', () => {
    console.log('\n✅ [WA] Client is ready! WhatsApp connected successfully!');
    console.log('[WA] You can now send messages.\n');
    isReady = true;
    clientReady = true;
    qrCode = null;
});

// Authentication failed
client.on('auth_failure', (msg) => {
    console.error('❌ [WA] Authentication failure:', msg);
    console.log('[WA] You may need to delete the whatsapp-session folder and scan QR again.');
    isReady = false;
    clientReady = false;
    qrCode = null;
});

// Disconnected
client.on('disconnected', (reason) => {
    console.log('⚠️ [WA] Client disconnected:', reason);
    isReady = false;
    clientReady = false;
    qrCode = null;
    console.log('[WA] The client disconnected. Restart the service after the browser has closed to reconnect safely.');
});

client.on('message_ack', async (message, ack) => {
    const webhookUrl = process.env.WHATSAPP_STATUS_WEBHOOK_URL;
    const webhookToken = process.env.WHATSAPP_STATUS_WEBHOOK_TOKEN;
    if (!webhookUrl || !webhookToken || !message?.id?._serialized) return;

    const status = ack >= 2 ? 'delivered' : ack === 1 ? 'sent' : '';
    if (!status) return;

    for (const delay of [0, 1000, 3000]) {
        if (delay) await new Promise(resolve => setTimeout(resolve, delay));
        try {
            const response = await fetch(webhookUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-WhatsApp-Webhook-Token': webhookToken
                },
                body: JSON.stringify({
                    message_id: message.id._serialized,
                    status,
                    details: `whatsapp-web.js acknowledgement ${ack}`
                })
            });
            if (response.ok) return;
            console.error(`[WA] Delivery status callback failed: HTTP ${response.status}`);
        } catch (error) {
            console.error('[WA] Delivery status callback failed:', error.message);
        }
    }
});

// Initialize client
console.log('[WA] Initializing WhatsApp client...');
console.log('[WA] Checking for existing session...');
client.initialize().catch((error) => {
    isReady = false;
    clientReady = false;
    console.error('[WA] WhatsApp client initialization failed:', error);
    console.error('[WA] Service is running, but sending is disabled until the browser initializes successfully.');
});

// Helper function to normalize phone number
function normalizePhoneNumber(phone) {
    // Remove all whitespace and non-digit characters except +
    phone = phone.trim().replace(/[\s\-]/g, '');
    
    // Remove + if present
    if (phone.startsWith('+')) {
        phone = phone.substring(1);
    }
    
    // If starts with 0, replace with 62 (Indonesia country code)
    if (phone.startsWith('0')) {
        phone = '62' + phone.substring(1);
    }
    
    // If doesn't start with 62, add it (assuming Indonesian numbers)
    if (!phone.startsWith('62') && phone.length >= 9) {
        phone = '62' + phone;
    }
    
    return phone;
}

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'ok',
        ready: isReady,
        clientReady: clientReady
    });
});

// Send message endpoint
app.post('/send-message', async (req, res) => {
    try {
        const { to, message } = req.body;

        // Validate input
        if (!to || !message) {
            return res.status(400).json({
                status: 'error',
                message: 'to and message are required'
            });
        }

        // Check if client is ready
        if (!isReady || !clientReady) {
            return res.status(503).json({
                status: 'error',
                message: 'WhatsApp client is not ready. Please scan QR code first.'
            });
        }

        // Normalize phone number
        const normalizedPhone = normalizePhoneNumber(to);
        const chatId = normalizedPhone + '@c.us';

        console.log(`[WA] Sending message to ${to} (normalized: ${normalizedPhone})`);

        // Send message
        const result = await client.sendMessage(chatId, message);
        const messageId = result?.id?._serialized || null;

        if (messageId) {
            console.log(`[WA] Message sent successfully to ${normalizedPhone}. Message ID: ${messageId}`);
        } else {
            console.warn(`[WA] Message send completed for ${normalizedPhone}, but WhatsApp returned no message ID.`);
        }

        res.json({
            status: 'success',
            message: messageId ? 'Message sent successfully' : 'Message send completed without a message ID',
            data: {
                to: normalizedPhone,
                original_to: to,
                message: message,
                message_id: messageId,
                sent_at: new Date().toISOString()
            }
        });

    } catch (error) {
        console.error('[WA] Error sending message:', error);
        res.status(500).json({
            status: 'error',
            message: error.message || 'Failed to send message'
        });
    }
});

// Get client status
app.get('/status', (req, res) => {
    res.json({
        status: isReady ? 'ready' : 'not_ready',
        clientReady: clientReady,
        qr_needed: !isReady && !clientReady,
        message: isReady ? 'WhatsApp is ready to send messages' : 
                 qrCode ? 'Please scan the QR code shown in the terminal' : 
                 'Initializing WhatsApp client...'
    });
});

// Get QR code (for debugging)
app.get('/qr', (req, res) => {
    if (qrCode) {
        res.json({
            qr: qrCode,
            message: 'Scan this QR code with WhatsApp app (Settings → Linked Devices)'
        });
    } else if (isReady) {
        res.json({
            message: 'WhatsApp is already connected. No QR code needed.'
        });
    } else {
        res.json({
            message: 'QR code not available yet. Please wait...'
        });
    }
});

// Start server
app.listen(PORT, () => {
    console.log(`[WA Service] WhatsApp service running on port ${PORT}`);
    console.log(`[WA Service] Health check: http://localhost:${PORT}/health`);
    console.log(`[WA Service] Send message: POST http://localhost:${PORT}/send-message`);
});

// Graceful shutdown
process.on('SIGINT', async () => {
    console.log('\n[WA Service] Shutting down...');
    await client.destroy();
    process.exit(0);
});
