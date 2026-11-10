#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const environments = {
  local: {
    // MySQL Database Configuration
    MYSQL_HOST: 'localhost',
    MYSQL_PORT: '3306',
    MYSQL_USER: 'root',
    MYSQL_PASSWORD: '',
    MYSQL_DATABASE: 'iqgncnzy_skripsi',
    
    // Alternative variable names (fallback)
    DB_HOST: 'localhost',
    DB_PORT: '3306',
    DB_USER: 'root',
    DB_PASSWORD: '',
    DB_NAME: 'iqgncnzy_skripsi',
    
    // Mikrotik Configuration - Local (connecting to production MikroTik for testing)
    // Note: If testing production MikroTik from localhost, use production config
    MIKROTIK_HOST: '103.148.18.52',
    MIKROTIK_PORT: '8252',
    MIKROTIK_API_PORT: '8783',
    MIKROTIK_USERNAME: 'polije',
    MIKROTIK_PASSWORD: '2025',
    
    // JWT Secret
    JWT_SECRET: 'your-local-secret-key-here',
    JWT_SECRET_KEY: 'your-local-secret-key-here',
    
    // Server Configuration
    PORT: '3001'
  },
  production: {
    // MySQL Database Configuration - Production (VPS)
    MYSQL_HOST: 'localhost',
    MYSQL_PORT: '3306',
    MYSQL_USER: 'crm_user',
    MYSQL_PASSWORD: 'YOUR_DB_PASSWORD_HERE',
    MYSQL_DATABASE: 'iqgncnzy_skripsi',
    
    // Alternative variable names (fallback)
    DB_HOST: 'localhost',
    DB_PORT: '3306',
    DB_USER: 'crm_user',
    DB_PASSWORD: 'YOUR_DB_PASSWORD_HERE',
    DB_NAME: 'iqgncnzy_skripsi',
    
    // Mikrotik Configuration - Production VPS
    MIKROTIK_HOST: '103.148.18.52',
    MIKROTIK_PORT: '8252',
    MIKROTIK_API_PORT: '8783',
    MIKROTIK_USERNAME: 'polije',
    MIKROTIK_PASSWORD: '2025',
    
    // JWT Secret
    JWT_SECRET: 'your-production-secret-key-change-this',
    JWT_SECRET_KEY: 'your-production-secret-key-change-this',
    
    // Server Configuration
    PORT: '3001',
    NODE_ENV: 'production'
  }
};

const env = process.argv[2];

if (!env || !environments[env]) {
  console.log('Usage: node switch-env.js <local|production>');
  console.log('Available environments:', Object.keys(environments).join(', '));
  process.exit(1);
}

const envContent = Object.entries(environments[env])
  .map(([key, value]) => `${key}=${value}`)
  .join('\n') + '\n';

fs.writeFileSync('.env', envContent);
console.log(`✅ Switched to ${env} environment`);
console.log(`📝 Created .env file for backend`);
console.log('\nConfiguration:');
console.log(`  - Database: ${environments[env].MYSQL_HOST}:${environments[env].MYSQL_PORT}/${environments[env].MYSQL_DATABASE}`);
console.log(`  - MikroTik RouterOS API: ${environments[env].MIKROTIK_HOST}:${environments[env].MIKROTIK_API_PORT}`);
console.log(`  - Server Port: ${environments[env].PORT}`);
