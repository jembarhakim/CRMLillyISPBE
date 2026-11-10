# 🔄 Backend Environment Switching Guide

This guide shows you how to easily switch between local development and production environments for the backend.

## 🚀 Quick Commands

### Switch to Local Development
```bash
# Using Node.js
node switch-env.js local

# Using batch file (Windows)
switch-local.bat
```

### Switch to Production
```bash
# Using Node.js
node switch-env.js production

# Using batch file (Windows)
switch-production.bat
```

## 📝 What Happens When You Switch

When you run a switch command, it creates/updates a `.env` file with the appropriate configuration:

### **Local Environment:**
```env
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=
MYSQL_DATABASE=iqgncnzy_skripsi

MIKROTIK_HOST=10.10.9.203
MIKROTIK_API_PORT=8783
MIKROTIK_USERNAME=rnd
MIKROTIK_PASSWORD=rnd@123

JWT_SECRET=your-local-secret-key-here
PORT=3001
```

### **Production Environment:**
```env
MYSQL_HOST=your-production-host
MYSQL_PORT=3306
MYSQL_USER=your-production-user
MYSQL_PASSWORD=your-production-password
MYSQL_DATABASE=iqgncnzy_skripsi

MIKROTIK_HOST=103.148.18.52
MIKROTIK_API_PORT=8783
MIKROTIK_USERNAME=your-production-mikrotik-user
MIKROTIK_PASSWORD=your-production-mikrotik-password

JWT_SECRET=your-production-secret-key-here
PORT=3001
```

## 🛠️ Development Workflow

### For Local Development:
1. **Switch to local**: `node switch-env.js local`
2. **Start the server**: `./start-server.bat` or `go run cmd/myapp/main.go`
3. **Backend will run on**: `http://localhost:3001`

### For Production Deployment:
1. **Switch to production**: `node switch-env.js production`
2. **Build**: `go build -o backend-app cmd/myapp/main.go`
3. **Deploy** to your production server (Railway/cPanel)

## 🔧 Configuration Details

### Database Configuration
- **Local**: Connects to local MySQL on `localhost:3306`
- **Production**: Connects to your production MySQL server

### Mikrotik Configuration
- **Local**: Uses internal network Mikrotik (`10.10.9.203`)
- **Production**: Uses public Mikrotik server (`103.148.18.52`)

## ⚠️ Important Notes

- **Always switch environments before starting the server**
- **Local development** requires:
  - MySQL running on `localhost:3306`
  - Database `iqgncnzy_skripsi` created
  - Access to local Mikrotik server (`10.10.9.203`)
- **Production** requires:
  - Valid production database credentials
  - Access to production Mikrotik server
- The `.env` file is automatically generated and should **NOT** be committed to git (it's in `.gitignore`)

## 🎯 Current Status

✅ **Environment switching system configured**
✅ **Local development ready**
✅ **Production deployment ready**
✅ **Mikrotik credentials managed per environment**
✅ **Easy one-command switching**

## 🔐 Security Best Practices

1. **Never commit `.env` to git** - It contains sensitive credentials
2. **Use different passwords** for local and production
3. **Update production credentials** in `switch-env.js` before deploying
4. **Keep JWT secrets strong** and different per environment

---

## 📋 Quick Reference

| Environment | Database | Mikrotik | Command |
|------------|----------|----------|---------|
| Local | localhost:3306 | 10.10.9.203:22 | `node switch-env.js local` |
| Production | Production DB | 103.148.18.52:22 | `node switch-env.js production` |

**Need help?** Just run `node switch-env.js` without arguments to see available options.
