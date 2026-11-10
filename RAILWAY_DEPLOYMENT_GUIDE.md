# Railway Deployment Guide for CRM Backend

## Step-by-Step Railway Deployment

### Step 1: Prepare GitHub Repository
1. **Push your code to GitHub** (if not already done)
2. **Make sure all files are committed**
3. **Verify your repository is public or you have Railway access**

### Step 2: Deploy to Railway
1. **Go to [Railway.app](https://railway.app)**
2. **Sign up with GitHub**
3. **Click "New Project"**
4. **Select "Deploy from GitHub repo"**
5. **Choose your repository**
6. **Railway will automatically detect the Dockerfile**

### Step 3: Configure Environment Variables
In Railway dashboard, go to your project → Variables tab and add:

```bash
# Database Configuration
DB_HOST=your-database-host
DB_USER=your-database-user
DB_PASSWORD=your-database-password
DB_NAME=your-database-name
DB_PORT=3306

# MikroTik Configuration
MIKROTIK_HOST=10.10.9.203
MIKROTIK_API_PORT=8783
MIKROTIK_USERNAME=rnd
MIKROTIK_PASSWORD=rnd@123

# Server Configuration
PORT=3001
NODE_ENV=production

# WhatsApp Configuration (Optional)
WHATSAPP_API_URL=https://graph.facebook.com/v18.0
WHATSAPP_API_TOKEN=your_whatsapp_token
WHATSAPP_PHONE_ID=your_phone_id
WHATSAPP_WEBHOOK_VERIFY_TOKEN=your_webhook_token
WHATSAPP_APP_SECRET=your_app_secret
```

### Step 4: Get Railway URL
1. **After deployment, Railway will provide a URL like:**
   `https://your-app-name.railway.app`
2. **Copy this URL for frontend configuration**

### Step 5: Update Frontend
Update your frontend API configuration to point to Railway backend.

## Railway Advantages
- ✅ **Automatic deployments** from GitHub
- ✅ **Free tier** ($5 credit/month)
- ✅ **Easy environment management**
- ✅ **Custom domain support**
- ✅ **Built-in monitoring**
- ✅ **SSL certificates** (automatic)

## Troubleshooting
- **Check Railway logs** for deployment errors
- **Verify environment variables** are set correctly
- **Test database connection** separately if needed
