# Staging Branch Workflow

This document outlines the staging branch workflow for the Auto-Focus Cloud API.

## Branch Structure

- **`main`** - Production-ready code, deploys to production automatically
- **`staging`** - Testing and staging code, deploys to staging automatically  
- **`feature/*`** - Feature branches, merge to staging for testing

## Staging Deployment Process

### 1. Automatic Staging Deployment

When you push to the `staging` branch:
```bash
git push origin staging
```

This automatically triggers:
- ✅ Build and test the application
- ✅ Deploy to staging environment (port 8081)  
- ✅ Available at https://staging.auto-focus.app/api/
- ✅ Uses separate `.env.staging` configuration
- ✅ Uses separate staging database

### 2. Manual Staging Deployment

You can also trigger staging deployment manually:
- Go to GitHub Actions → "Deploy to Staging" 
- Click "Run workflow" → "Run workflow"

### 3. Local Staging Testing

Test staging locally before pushing:
```bash
# Ensure you have .env.staging configured
cp .env.example .env.staging
# Edit .env.staging with staging/test values

# Run staging locally on port 8081
./deploy-staging.sh
```

This will show helpful test information including Stripe test cards.

## Workflow Examples

### Feature Development
```bash
# Create feature branch
git checkout -b feature/new-license-validation

# Make changes and test locally
# ... development work ...

# Push to staging for testing
git checkout staging
git merge feature/new-license-validation
git push origin staging

# Staging automatically deploys and is available at:
# https://staging.auto-focus.app/api/

# Test staging with:
# https://auto-focus.app/?staging=1
```

### Production Release
```bash
# After staging testing is complete
git checkout main
git merge staging
git push origin main

# Production automatically deploys to:
# https://auto-focus.app/api/
```

## Environment Differences

| Environment | Branch | Port | Database | Stripe | URL |
|-------------|--------|------|----------|--------|-----|
| Production  | `main` | 8080 | `production.db` | Live keys | `auto-focus.app/api/` |
| Staging     | `staging` | 8081 | `staging.db` | Test keys | `staging.auto-focus.app/api/` |

## Stripe Testing

Staging uses Stripe test mode with these test cards:
- **Success**: `4242424242424242`
- **Declined**: `4000000000000002`  
- **Insufficient funds**: `4000000000009995`

Test staging checkout at: https://auto-focus.app/?staging=1

## Monitoring

### Check Staging Status
```bash
# Via GitHub Actions (recommended)
# Go to Actions tab and check latest "Deploy to Staging" run

# Or via server SSH
systemctl status auto-focus-cloud-staging
journalctl -u auto-focus-cloud-staging -f
```

### Staging Health Check
```bash
curl https://staging.auto-focus.app/api/v1/licenses/validate \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"license_key":"test","app_version":"1.0"}'
```

## Troubleshooting

### Staging Deploy Failed
1. Check GitHub Actions logs
2. Verify `.env.staging` exists on server
3. Check port 8081 availability
4. Review staging service logs

### Database Issues
```bash
# Staging uses separate database file
# Check DATABASE_URL in .env.staging
ls -la /home/autofocus/staging/storage/data/
```

### Port Conflicts
```bash
# Check what's running on port 8081
lsof -i :8081

# Kill conflicting processes if needed
lsof -ti :8081 | xargs kill -9
```